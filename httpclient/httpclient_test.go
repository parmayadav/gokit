package http

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const shortTimeout = 50 * time.Millisecond

func clearClientEnv(t *testing.T) {
	t.Helper()
	t.Setenv("CA_CERT_PATH", "")
	t.Setenv("INSECURE_SKIP_VERIFY", "")
	t.Setenv("HTTP_TIMEOUT", "")
}

func writeServerCA(t *testing.T, server *httptest.Server) string {
	t.Helper()
	block := &pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}
	path := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatalf("failed to write CA file: %v", err)
	}
	return path
}

func transportOf(t *testing.T, c *HttpClient) *http.Transport {
	t.Helper()
	transport, ok := c.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", c.client.Transport)
	}
	return transport
}

func TestCheckStatusCode(t *testing.T) {
	client := &HttpClient{}
	tests := []struct {
		name   string
		status int
		want   bool
	}{
		{"below 2xx", 199, false},
		{"lower bound 200", 200, true},
		{"created", 201, true},
		{"upper bound 299", 299, true},
		{"redirect", 300, false},
		{"client error", 404, false},
		{"server error", 500, false},
		{"zero", 0, false},
		{"negative", -1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := client.CheckStatusCode(tt.status); got != tt.want {
				t.Errorf("CheckStatusCode(%d) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestInitHttpClient_Defaults(t *testing.T) {
	clearClientEnv(t)

	client := InitHttpClient()

	if client == nil || client.client == nil {
		t.Fatal("InitHttpClient returned nil client")
	}
	if want := 30 * time.Second; client.client.Timeout != want {
		t.Errorf("timeout = %v, want %v", client.client.Timeout, want)
	}
	if transportOf(t, client).TLSClientConfig != nil {
		t.Error("expected no TLS config when CA_CERT_PATH is unset")
	}
}

func TestInitHttpClient_Timeout(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{"custom", "5", 5 * time.Second},
		{"invalid falls back to default", "abc", 30 * time.Second},
		{"empty falls back to default", "", 30 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearClientEnv(t)
			t.Setenv("HTTP_TIMEOUT", tt.value)

			if got := InitHttpClient().client.Timeout; got != tt.want {
				t.Errorf("timeout = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInitHttpClient_MissingCAFile(t *testing.T) {
	clearClientEnv(t)
	t.Setenv("CA_CERT_PATH", filepath.Join(t.TempDir(), "missing.pem"))

	client := InitHttpClient()

	if transportOf(t, client).TLSClientConfig != nil {
		t.Error("expected no TLS config when CA file cannot be read")
	}
}

func TestInitHttpClient_WithCAFile(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()
	caPath := writeServerCA(t, server)

	tests := []struct {
		name         string
		insecureEnv  string
		wantInsecure bool
	}{
		{"verification enabled by default", "", false},
		{"verification explicitly enabled", "false", false},
		{"verification skipped", "true", true},
		{"invalid value keeps verification", "notabool", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearClientEnv(t)
			t.Setenv("CA_CERT_PATH", caPath)
			t.Setenv("INSECURE_SKIP_VERIFY", tt.insecureEnv)

			cfg := transportOf(t, InitHttpClient()).TLSClientConfig
			if cfg == nil {
				t.Fatal("expected TLS config to be set")
			}
			if cfg.RootCAs == nil {
				t.Error("expected RootCAs to be populated")
			}
			if cfg.InsecureSkipVerify != tt.wantInsecure {
				t.Errorf("InsecureSkipVerify = %v, want %v", cfg.InsecureSkipVerify, tt.wantInsecure)
			}
		})
	}
}

func TestGet_TLSWithCustomCA(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("secure"))
	}))
	defer server.Close()

	t.Run("trusted CA succeeds", func(t *testing.T) {
		clearClientEnv(t)
		t.Setenv("CA_CERT_PATH", writeServerCA(t, server))

		resp, err := InitHttpClient().Get(context.Background(), "svc", server.URL, nil, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(resp.Body) != "secure" {
			t.Errorf("body = %q, want %q", resp.Body, "secure")
		}
	})

	t.Run("untrusted certificate fails", func(t *testing.T) {
		clearClientEnv(t)

		_, err := InitHttpClient().Get(context.Background(), "svc", server.URL, nil, nil, nil)
		if err == nil {
			t.Fatal("expected certificate verification error")
		}
	})
}

func TestGet(t *testing.T) {
	clearClientEnv(t)

	var gotMethod, gotQuery, gotHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotQuery = r.URL.RawQuery
		gotHeader = r.Header.Get("X-Api-Key")
		w.Header().Set("X-Reply", "pong")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	params := map[string]string{"page": "2", "limit": "10"}
	resp, err := InitHttpClient().Get(
		context.Background(),
		"svc",
		server.URL+"?existing=1",
		map[string]string{"X-Api-Key": "secret"},
		&params,
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Errorf("method = %s, want GET", gotMethod)
	}
	if gotHeader != "secret" {
		t.Errorf("X-Api-Key = %q, want %q", gotHeader, "secret")
	}
	for _, want := range []string{"existing=1", "page=2", "limit=10"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
	if resp.Status != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.Status)
	}
	if string(resp.Body) != `{"ok":true}` {
		t.Errorf("body = %q", resp.Body)
	}
	if resp.Header.Get("X-Reply") != "pong" {
		t.Errorf("X-Reply header = %q, want pong", resp.Header.Get("X-Reply"))
	}
}

func TestGet_NilParamsAndHeaders(t *testing.T) {
	clearClientEnv(t)

	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
	}))
	defer server.Close()

	resp, err := InitHttpClient().Get(context.Background(), "svc", server.URL, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty", gotQuery)
	}
	if resp.Status != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.Status)
	}
}

func TestGet_ParamsAreEscaped(t *testing.T) {
	clearClientEnv(t)

	var gotValue string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotValue = r.URL.Query().Get("q")
	}))
	defer server.Close()

	params := map[string]string{"q": "a b&c=d"}
	if _, err := InitHttpClient().Get(context.Background(), "svc", server.URL, nil, &params, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotValue != "a b&c=d" {
		t.Errorf("decoded value = %q, want %q", gotValue, "a b&c=d")
	}
}

func TestGet_InvalidURL(t *testing.T) {
	clearClientEnv(t)

	resp, err := InitHttpClient().Get(context.Background(), "svc", "http://[::1", nil, nil, nil)
	if err == nil {
		t.Fatal("expected URL parse error")
	}
	if resp.Status != 0 || resp.Body != nil {
		t.Errorf("expected zero response, got %+v", resp)
	}
}

func TestGet_NonSuccessStatusIsNotAnError(t *testing.T) {
	clearClientEnv(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := InitHttpClient()
	resp, err := client.Get(context.Background(), "svc", server.URL, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", resp.Status)
	}
	if client.CheckStatusCode(resp.Status) {
		t.Error("CheckStatusCode should report false for 500")
	}
	if !strings.Contains(string(resp.Body), "boom") {
		t.Errorf("body = %q, want it to contain %q", resp.Body, "boom")
	}
}

func TestGet_RequestTimeout(t *testing.T) {
	clearClientEnv(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))
	defer server.Close()

	timeout := shortTimeout
	_, err := InitHttpClient().Get(context.Background(), "svc", server.URL, nil, nil, &timeout)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestGet_ConnectionRefused(t *testing.T) {
	clearClientEnv(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()

	if _, err := InitHttpClient().Get(context.Background(), "svc", url, nil, nil, nil); err == nil {
		t.Fatal("expected connection error")
	}
}

func TestGet_TruncatedBody(t *testing.T) {
	clearClientEnv(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10")
		_, _ = w.Write([]byte("abc"))
	}))
	defer server.Close()

	resp, err := InitHttpClient().Get(context.Background(), "svc", server.URL, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error reading truncated body")
	}
	if resp.Status != 0 || resp.Body != nil {
		t.Errorf("expected zero response on error, got %+v", resp)
	}
}

func TestPost_JSONBody(t *testing.T) {
	clearClientEnv(t)

	var gotMethod, gotContentType string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	}))
	defer server.Close()

	resp, err := InitHttpClient().Post(
		context.Background(),
		"svc",
		server.URL,
		map[string]string{"Content-Type": "application/json"},
		map[string]any{"name": "alice", "active": true},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q", gotContentType)
	}
	if gotBody["name"] != "alice" || gotBody["active"] != true {
		t.Errorf("body = %v", gotBody)
	}
	if resp.Status != http.StatusCreated {
		t.Errorf("status = %d, want 201", resp.Status)
	}
	if string(resp.Body) != "created" {
		t.Errorf("response body = %q", resp.Body)
	}
}

func TestPost_ReaderBodyIsSentRaw(t *testing.T) {
	clearClientEnv(t)

	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
	}))
	defer server.Close()

	const payload = "raw=payload"
	_, err := InitHttpClient().Post(context.Background(), "svc", server.URL, nil, strings.NewReader(payload), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotBody != payload {
		t.Errorf("body = %q, want %q", gotBody, payload)
	}
}

func TestPost_NilBody(t *testing.T) {
	clearClientEnv(t)

	var gotLength int64 = -1
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotLength = int64(len(b))
	}))
	defer server.Close()

	if _, err := InitHttpClient().Post(context.Background(), "svc", server.URL, nil, nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotLength != 0 {
		t.Errorf("body length = %d, want 0", gotLength)
	}
}

func TestPost_UnmarshalableBody(t *testing.T) {
	clearClientEnv(t)

	resp, err := InitHttpClient().Post(context.Background(), "svc", "http://127.0.0.1:0", nil, make(chan int), nil)
	if err == nil {
		t.Fatal("expected JSON marshal error")
	}
	if resp.Status != 0 {
		t.Errorf("expected zero response, got %+v", resp)
	}
}

func TestPost_InvalidURL(t *testing.T) {
	clearClientEnv(t)

	if _, err := InitHttpClient().Post(context.Background(), "svc", "http://[::1", nil, nil, nil); err == nil {
		t.Fatal("expected request creation error")
	}
}

func TestPost_RequestTimeout(t *testing.T) {
	clearClientEnv(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))
	defer server.Close()

	timeout := shortTimeout
	if _, err := InitHttpClient().Post(context.Background(), "svc", server.URL, nil, map[string]string{"a": "b"}, &timeout); err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestPut_JSONBody(t *testing.T) {
	clearClientEnv(t)

	var gotMethod, gotContentType string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("updated"))
	}))
	defer server.Close()

	resp, err := InitHttpClient().Put(
		context.Background(),
		"svc",
		server.URL,
		map[string]string{"Content-Type": "application/json"},
		map[string]any{"name": "bob"},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %s, want PUT", gotMethod)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q", gotContentType)
	}
	if gotBody["name"] != "bob" {
		t.Errorf("body = %v", gotBody)
	}
	if resp.Status != http.StatusOK || string(resp.Body) != "updated" {
		t.Errorf("response = %+v", resp)
	}
}

func TestPut_ReaderBodyIsSentRaw(t *testing.T) {
	clearClientEnv(t)

	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
	}))
	defer server.Close()

	const payload = "raw=payload"
	if _, err := InitHttpClient().Put(context.Background(), "svc", server.URL, nil, strings.NewReader(payload), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotBody != payload {
		t.Errorf("body = %q, want %q", gotBody, payload)
	}
}

func TestPut_Errors(t *testing.T) {
	clearClientEnv(t)
	client := InitHttpClient()

	t.Run("unmarshalable body", func(t *testing.T) {
		resp, err := client.Put(context.Background(), "svc", "http://127.0.0.1:0", nil, make(chan int), nil)
		if err == nil {
			t.Fatal("expected JSON marshal error")
		}
		if resp.Status != 0 {
			t.Errorf("expected zero response, got %+v", resp)
		}
	})

	t.Run("invalid URL", func(t *testing.T) {
		if _, err := client.Put(context.Background(), "svc", "http://[::1", nil, nil, nil); err == nil {
			t.Fatal("expected request creation error")
		}
	})

	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
			case <-time.After(2 * time.Second):
			}
		}))
		defer server.Close()

		timeout := shortTimeout
		if _, err := client.Put(context.Background(), "svc", server.URL, nil, map[string]string{"a": "b"}, &timeout); err == nil {
			t.Fatal("expected timeout error")
		}
	})
}

func TestDelete(t *testing.T) {
	clearClientEnv(t)

	var gotMethod, gotQuery, gotHeader string
	var gotBodyLength int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotQuery = r.URL.RawQuery
		gotHeader = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBodyLength = len(b)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	params := map[string]string{"force": "true"}
	resp, err := InitHttpClient().Delete(
		context.Background(),
		"svc",
		server.URL+"?existing=1",
		map[string]string{"Authorization": "Bearer token"},
		&params,
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", gotMethod)
	}
	if gotHeader != "Bearer token" {
		t.Errorf("Authorization = %q", gotHeader)
	}
	for _, want := range []string{"existing=1", "force=true"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
	if gotBodyLength != 0 {
		t.Errorf("body length = %d, want 0", gotBodyLength)
	}
	if resp.Status != http.StatusNoContent {
		t.Errorf("status = %d, want 204", resp.Status)
	}
}

func TestDelete_NilParamsAndHeaders(t *testing.T) {
	clearClientEnv(t)

	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
	}))
	defer server.Close()

	resp, err := InitHttpClient().Delete(context.Background(), "svc", server.URL, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty", gotQuery)
	}
	if resp.Status != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.Status)
	}
}

func TestDelete_NonSuccessStatusIsNotAnError(t *testing.T) {
	clearClientEnv(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	client := InitHttpClient()
	resp, err := client.Delete(context.Background(), "svc", server.URL, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusNotFound || client.CheckStatusCode(resp.Status) {
		t.Errorf("unexpected response %+v", resp)
	}
}

func TestDelete_Errors(t *testing.T) {
	clearClientEnv(t)
	client := InitHttpClient()

	t.Run("invalid URL", func(t *testing.T) {
		resp, err := client.Delete(context.Background(), "svc", "http://[::1", nil, nil, nil)
		if err == nil {
			t.Fatal("expected URL parse error")
		}
		if resp.Status != 0 || resp.Body != nil {
			t.Errorf("expected zero response, got %+v", resp)
		}
	})

	t.Run("connection refused", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := server.URL
		server.Close()

		if _, err := client.Delete(context.Background(), "svc", url, nil, nil, nil); err == nil {
			t.Fatal("expected connection error")
		}
	})

	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
			case <-time.After(2 * time.Second):
			}
		}))
		defer server.Close()

		timeout := shortTimeout
		if _, err := client.Delete(context.Background(), "svc", server.URL, nil, nil, &timeout); err == nil {
			t.Fatal("expected timeout error")
		}
	})
}

func TestDoRequest_ReturnsHeadersAndStatus(t *testing.T) {
	clearClientEnv(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Trace", "abc")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	body, header, status, err := InitHttpClient().doRequest(http.MethodGet, server.URL, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusAccepted {
		t.Errorf("status = %d, want 202", status)
	}
	if header.Get("X-Trace") != "abc" {
		t.Errorf("X-Trace = %q, want abc", header.Get("X-Trace"))
	}
	if string(body) != "ok" {
		t.Errorf("body = %q, want ok", body)
	}
}

func TestDoRequest_InvalidMethod(t *testing.T) {
	clearClientEnv(t)

	if _, _, _, err := InitHttpClient().doRequest("BAD METHOD", "http://example.com", nil, nil, nil); err == nil {
		t.Fatal("expected request creation error for invalid method")
	}
}
