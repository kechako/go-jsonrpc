// Package jsonrpc is a JSON-RPC 2.0 client that communicates over HTTP.
package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/google/uuid"
)

// Version is a JSON-RPC version.
const Version = "2.0"

// Client represents a JSPN-RPC 2.0 Client.
type Client struct {
	// HTTPClient is a HTTP client you want to use.
	// Use http.DefaultClient if it is nil.
	HTTPClient *http.Client
}

type request struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      uuid.UUID   `json:"id"`
}

func requestBody(method string, params interface{}) (uuid.UUID, io.Reader, error) {
	r := &request{
		JSONRPC: Version,
		Method:  method,
		Params:  params,
		ID:      uuid.New(),
	}

	b, err := json.Marshal(r)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	return r.ID, bytes.NewReader(b), nil
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *ResponseError  `json:"error"`
	ID      uuid.UUID       `json:"id"`
}

// ErrorCode is a number that indicates the error type that occurred.
type ErrorCode int

const (
	// Invalid JSON was received by the server.
	// An error occurred on the server while parsing the JSON text.
	ParseError ErrorCode = -32700
	// The JSON sent is not a valid Request object.
	InvalidRequest ErrorCode = -32600
	// The method does not exist or is not available.
	MethodNotFound ErrorCode = -32601
	// Invalid method parameter(s).
	InvalidParams ErrorCode = -32602
	// nternal JSON-RPC error.
	InternalError ErrorCode = -32603
)

// ResponseError represents an error responded by the server.
type ResponseError struct {
	Code    ErrorCode   `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func (err *ResponseError) Error() string {
	return fmt.Sprintf("%s (%d)", err.Message, err.Code)
}

// Call calls the method on the url with the params,
// and stores result responded by the server in the result.
func (client *Client) Call(ctx context.Context, url string, method string, params interface{}, result interface{}, opts ...Option) error {
	if method == "" {
		return errors.New("method is empty")
	}

	var callOpts callOptions
	if len(opts) > 0 {
		for _, opt := range opts {
			opt.apply(&callOpts)
		}
	}

	id, body, err := requestBody(method, params)
	if err != nil {
		return err
	}

	req, err := client.newRequest(ctx, url, body, callOpts)
	if err != nil {
		return err
	}

	res, err := client.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("failed to post request: %w", err)
	}
	defer res.Body.Close()
	defer io.Copy(ioutil.Discard, res.Body)

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server does not respond 200 OK: %s", res.Status)
	}

	var rpcRes response

	if err := json.NewDecoder(res.Body).Decode(&rpcRes); err != nil {
		return fmt.Errorf("failed to decode response JSON: %w", err)
	}

	if rpcRes.Error != nil {
		return rpcRes.Error
	}

	if rpcRes.ID != id {
		return errors.New("response ID is not matched to request")
	}

	if err := json.Unmarshal([]byte(rpcRes.Result), result); err != nil {
		return fmt.Errorf("failed to decode result JSON: %w", err)
	}

	return nil
}

func (client *Client) newRequest(ctx context.Context, url string, body io.Reader, opts callOptions) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create new HTTP request: %w", err)
	}

	req.Header.Add("Content-Type", "text/json")

	if opts.Header != nil {
		for key, values := range opts.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}

	return req, nil
}

func (client *Client) httpClient() *http.Client {
	if client.HTTPClient == nil {
		return http.DefaultClient
	}

	return client.HTTPClient
}

type callOptions struct {
	Header http.Header
}

// Option represents an option used to method calling.
type Option interface {
	apply(opts *callOptions)
}

type optionFunc func(opts *callOptions)

func (f optionFunc) apply(opts *callOptions) {
	f(opts)
}

func WithHeader(header http.Header) Option {
	return optionFunc(func(opts *callOptions) {
		opts.Header = header
	})
}
