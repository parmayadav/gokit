package examples

import (
	"context"
	"fmt"
	"log"

	gktHttpClient "github.com/parmayadav/gokit/httpclient"
)

func main() {
	// Initialize HTTP client with environment variables
	// Set these before running:
	// export APP_ENV="development"
	// export CA_CERT_PATH="/path/to/cert.pem"
	// export INSECURE_SKIP_VERIFY=false
	// export HTTP_TIMEOUT=60

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
