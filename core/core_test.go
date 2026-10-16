package core

import "testing"

const testKey = "GOKIT_CORE_TEST_KEY"

func TestGetEnvString(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		setEnv       bool
		defaultValue string
		want         string
	}{
		{"returns env value", "hello", true, "default", "hello"},
		{"unset returns default", "", false, "default", "default"},
		{"empty returns default", "", true, "default", "default"},
		{"empty with empty default", "", true, "", ""},
		{"value with spaces preserved", "  spaced  ", true, "default", "  spaced  "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(testKey, tt.value)
			}
			if got := GetEnvString(testKey, tt.defaultValue); got != tt.want {
				t.Errorf("GetEnvString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	const defaultValue = 42
	tests := []struct {
		name   string
		value  string
		setEnv bool
		want   int
	}{
		{"valid positive", "10", true, 10},
		{"valid negative", "-5", true, -5},
		{"zero", "0", true, 0},
		{"unset returns default", "", false, defaultValue},
		{"empty returns default", "", true, defaultValue},
		{"non-numeric returns default", "abc", true, defaultValue},
		{"float returns default", "1.5", true, defaultValue},
		{"surrounding spaces returns default", " 7 ", true, defaultValue},
		{"overflow returns default", "99999999999999999999", true, defaultValue},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(testKey, tt.value)
			}
			if got := GetEnvInt(testKey, defaultValue); got != tt.want {
				t.Errorf("GetEnvInt() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		setEnv       bool
		defaultValue bool
		want         bool
	}{
		{"true", "true", true, false, true},
		{"TRUE", "TRUE", true, false, true},
		{"1", "1", true, false, true},
		{"t", "t", true, false, true},
		{"false", "false", true, true, false},
		{"0", "0", true, true, false},
		{"f", "f", true, true, false},
		{"unset returns default true", "", false, true, true},
		{"unset returns default false", "", false, false, false},
		{"empty returns default", "", true, true, true},
		{"invalid returns default true", "yes", true, true, true},
		{"invalid returns default false", "yes", true, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(testKey, tt.value)
			}
			if got := GetEnvBool(testKey, tt.defaultValue); got != tt.want {
				t.Errorf("GetEnvBool() = %v, want %v", got, tt.want)
			}
		})
	}
}
