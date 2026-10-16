# gokit

> A lightweight, reusable toolkit for building Go backend applications.

![Version Badge Placeholder](https://img.shields.io/badge/version/1.0.0-007BCE?style=flat)
[![Go Version](https://img.shields.io/badge/go-1.27+-007BCE.svg)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

---

## ✨ Features
- **HTTP Client**: Reusable HTTP client initialization with configurable timeouts and sensible defaults
- **Modular Design**: Keep utilities independent and easy to integrate into existing Go projects
- **Extensible Architecture**: Designed to accommodate additional backend utilities over time

---

## 📦 Project Structure

```
gokit/
├── httpclient/
│   ├── httpclient.go
│   └── httpclinet_test.go
├── examples/
│   └── examples.go
├── go.mod
├── go.sum
└── README.md
```

> **Note:** The package structure may evolve as the project develops.

---

## 🚀 Installation

Install the module using Go:

```bash
go get github.com/<your-username>/gokit
```

Replace `<your-username>` with your GitHub username or organization.

### Import Usage

Import only the packages you need:

```go
import (
    "github.com/<your-username>/gokit/httpclient"
)
```

---

## 🌐 HTTP Client

The `httpclient` package provides reusable HTTP client initialization and request handling.

### Basic Usage

Initialize the HTTP client with environment configuration:

```go
func main() {
	client := gktHttpClient.InitHttpClient()

	// Example 1: Simple GET request
	status, err := client.Get(
		context.TODO(), // Use applicable context
		"api.example.com",
		"https://api.example.com/users",
		map[string]string{},
		nil,
		nil,
	)
	if err != nil {
		log.Printf("GET failed: %v", err)
		return
	}

	fmt.Printf("Status: %d\n", status.Status)
	fmt.Printf("Body: %s\n", string(status.Body))

	// Example 2: GET with parameters and headers
	resp, err := client.Get(
		context.TODO(), // Use applicable context
		"api.example.com",
		"https://api.example.com/search",
		map[string]string{"X-API-Key": "your-api-key"},
		&map[string]string{
			"page":  "1",
			"limit": "10",
		},
		nil,
	)
	if err != nil {
		log.Printf("GET with params failed: %v", err)
		return
	}

	fmt.Printf("Response with params: %s\n", string(resp.Body))

	// Example 3: POST request with JSON body
	jsonBody := map[string]interface{}{
		"name":   "John Doe",
		"email":  "john@example.com",
		"active": true,
	}

	resp2, err := client.Post(
		context.TODO(), // Use applicable context
		"api.example.com",
		"https://api.example.com/users",
		map[string]string{"Content-Type": "application/json"},
		jsonBody,
		nil,
	)
	if err != nil {
		log.Printf("POST failed: %v", err)
		return
	}

	fmt.Printf("Status: %d\n", resp2.Status)
	fmt.Printf("Response: %s\n", string(resp2.Body))

	log.Println("HTTP client examples completed successfully!")
}
```

> 💡 The HTTP client allows you to:
> - Initialize clients with environment configuration (timeout, CA certs)
> - Perform GET and POST requests with headers and parameters
> - Handle response status codes programmatically
> - Support context-aware operations for graceful cancellation
> - Manage request timeouts and connection pooling

> ⚠️ The package APIs shown below are illustrative and will need to match the actual implementation.

---

## 🛠️ Tech Stack

| Category | Technology |
|----------|------------|
| **Language** | Go |
| **HTTP** | Go standard library (net/http) |
| **Testing** | Go testing package |
| **CI/CD** | GitHub Actions |

> 📝 Additional dependencies will be introduced only when they provide clear value.

---

## 🧪 Testing

### Run All Tests

```bash
go test ./...
```

### Test with Race Detector

```bash
go test -race ./...
```

### Run Static Analysis

```bash
go vet ./...
```

---

## 🗺️ Roadmap

- [ ] Implement configuration utilities
- [ ] Implement reusable HTTP client initialization
- [ ] Add unit tests for each package
- [ ] Add practical usage examples
- [ ] Add GitHub Actions CI
- [ ] Add package-level documentation
- [ ] Add structured logging utilities
- [ ] Add validation and masking utilities
- [ ] Publish stable API documentation

---

## 🤝 Contributing

Contributions, suggestions, and bug reports are welcome!

1. Fork the repository
2. Create a feature branch
3. Add implementation and tests
4. Run `go test ./...` and `go vet ./...` before submitting
5. Open a pull request with a clear description of the changes

> 💡 Please keep packages focused, APIs simple, and dependencies minimal.

---

## 📄 License

This project is licensed under the MIT License, unless otherwise specified.

© 2024 - gokit — Practical building blocks for Go backend development