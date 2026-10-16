package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/parmayadav/gokit/core"
)

type HttpClient struct {
	client *http.Client
}

type HttpResponse struct {
	Header http.Header
	Status int
	Body   []byte
}

func InitHttpClient() *HttpClient {
	transport := &http.Transport{}
	caCertPath := core.GetEnvString("CA_CERT_PATH", "") // Default to empty string if not set
	caCert, err := os.ReadFile(caCertPath)

	if err != nil {
		log.Printf("Warning: Failed to load unified PEM file: %v", err)
	} else {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		transport.TLSClientConfig = &tls.Config{
			RootCAs:            caCertPool,
			InsecureSkipVerify: core.GetEnvBool("INSECURE_SKIP_VERIFY", false), // Default to false if not set, and it will try to verify the ssl cert
		}
	}

	return &HttpClient{
		client: &http.Client{
			Timeout:   time.Duration(core.GetEnvInt("HTTP_TIMEOUT", 30)) * time.Second,
			Transport: transport,
		},
	}
}

func (h *HttpClient) CheckStatusCode(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}

func (h *HttpClient) Get(
	ctx context.Context,
	serviceName string,
	baseURL string,
	headers map[string]string,
	params *map[string]string,
	timeout *time.Duration,
) (HttpResponse, error) {

	reqURL, err := url.Parse(baseURL)
	if err != nil {
		return HttpResponse{}, err
	}

	q := reqURL.Query()
	if params != nil {
		for k, v := range *params {
			q.Set(k, v)
		}
	}
	reqURL.RawQuery = q.Encode()

	respBytes, respHeader, status, err := h.doRequest(
		http.MethodGet,
		reqURL.String(),
		headers,
		nil,
		timeout,
	)

	if err != nil {
		return HttpResponse{}, err
	}

	result := HttpResponse{
		Header: respHeader,
		Status: status,
		Body:   respBytes,
	}

	return result, nil
}

func (h *HttpClient) Post(
	ctx context.Context,
	serviceName string,
	url string,
	headers map[string]string,
	body interface{},
	timeout *time.Duration,
) (HttpResponse, error) {

	respBytes, respHeader, status, err := h.doRequest(
		http.MethodPost,
		url,
		headers,
		body,
		timeout,
	)
	if err != nil {
		return HttpResponse{}, err
	}

	result := HttpResponse{
		Header: respHeader,
		Status: status,
		Body:   respBytes,
	}

	return result, nil
}

func (h *HttpClient) Put(
	ctx context.Context,
	serviceName string,
	url string,
	headers map[string]string,
	body interface{},
	timeout *time.Duration,
) (HttpResponse, error) {

	respBytes, respHeader, status, err := h.doRequest(
		http.MethodPut,
		url,
		headers,
		body,
		timeout,
	)
	if err != nil {
		return HttpResponse{}, err
	}

	result := HttpResponse{
		Header: respHeader,
		Status: status,
		Body:   respBytes,
	}

	return result, nil
}

func (h *HttpClient) Delete(
	ctx context.Context,
	serviceName string,
	baseURL string,
	headers map[string]string,
	params *map[string]string,
	timeout *time.Duration,
) (HttpResponse, error) {

	reqURL, err := url.Parse(baseURL)
	if err != nil {
		return HttpResponse{}, err
	}

	q := reqURL.Query()
	if params != nil {
		for k, v := range *params {
			q.Set(k, v)
		}
	}
	reqURL.RawQuery = q.Encode()

	respBytes, respHeader, status, err := h.doRequest(
		http.MethodDelete,
		reqURL.String(),
		headers,
		nil,
		timeout,
	)
	if err != nil {
		return HttpResponse{}, err
	}

	result := HttpResponse{
		Header: respHeader,
		Status: status,
		Body:   respBytes,
	}

	return result, nil
}

func (h *HttpClient) doRequest(
	method string,
	url string,
	headers map[string]string,
	body interface{},
	timeout *time.Duration,
) ([]byte, http.Header, int, error) {

	var reqBody io.Reader
	if body != nil {
		if r, ok := body.(io.Reader); ok {
			reqBody = r
		} else {
			b, err := json.Marshal(body)
			if err != nil {
				return nil, nil, 0, err
			}
			reqBody = bytes.NewBuffer(b)
		}
	}

	reqCtx := context.Background()
	if timeout != nil {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(reqCtx, *timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(reqCtx, method, url, reqBody)
	if err != nil {
		return nil, nil, 0, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.Header, resp.StatusCode, err
	}

	return respBody, resp.Header, resp.StatusCode, nil
}
