package jsonrpc

import (
	"net/http"
	"testing"
)

func TestNewClient(t *testing.T) {
	const testEndpoint = "https://example.com/endpoint"

	client := &Client{}

	if client.httpClient() == nil {
		t.Error("Client.httpClient() must not be nil")
	}

	if client.httpClient() != http.DefaultClient {
		t.Error("Client.httpClient() must be http.DefaultClient")
	}

}
func TestNewClientWithCustomHTTPClient(t *testing.T) {
	var httpClient = &http.Client{}

	client := &Client{
		HTTPClient: httpClient,
	}

	if client.httpClient() == nil {
		t.Error("Client.httpClient() must not be nil")
	}

	if client.httpClient() != httpClient {
		t.Error("Client.httpClient() does not return valid http.Client")
	}
}
