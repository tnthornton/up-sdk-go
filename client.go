// Copyright 2021 Upbound Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package up

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"

	uerrors "github.com/upbound/up-sdk-go/errors"
	"github.com/upbound/up-sdk-go/http/headers"
	"github.com/upbound/up-sdk-go/http/request"
)

const (
	defaultBaseURL     = "https://api.upbound.io"
	defaultUserAgent   = "up-sdk-go"
	defaultHTTPTimeout = 10 * time.Second
)

// Client is an HTTP client for communicating with Upbound.
type Client interface {
	NewRequest(ctx context.Context, method, prefix, urlPath string, body interface{}) (*http.Request, error)
	Do(req *http.Request, obj interface{}) error
	With(modifiers ...ClientModifierFn) Client
}

// A ClientModifierFn modifies an HTTP client.
type ClientModifierFn func(*HTTPClient)

// NewClient builds a new default HTTP client for Upbound.
func NewClient(modifiers ...ClientModifierFn) *HTTPClient {
	b, _ := url.Parse(defaultBaseURL)
	c := &HTTPClient{
		BaseURL:      b,
		ErrorHandler: &DefaultErrorHandler{},
		HTTP: &http.Client{
			Timeout:   defaultHTTPTimeout,
			Transport: NewContextTransport(),
		},
		UserAgent: defaultUserAgent,
	}
	for _, m := range modifiers {
		m(c)
	}
	return c
}

// HTTPClient implements the Client interface and allows for overriding of base
// URL, error handling, and user agent.
type HTTPClient struct {
	// BaseURL is the base Upbound API URL.
	BaseURL *url.URL

	// ErrorHandler controls how the client handles errors.
	ErrorHandler ResponseErrorHandler

	// HTTP is the underlying HTTP client.
	HTTP *http.Client

	// User agent for communicating with the Upbound API.
	UserAgent string
}

// A ResponseErrorHandler handles errors in HTTP responses.
type ResponseErrorHandler interface {
	Handle(res *http.Response) error
}

// NewRequest builds an HTTP request.
func (c *HTTPClient) NewRequest(ctx context.Context, method, prefix, urlPath string, body interface{}) (*http.Request, error) {
	u, err := c.BaseURL.Parse(path.Join(prefix, urlPath))
	if err != nil {
		return nil, err
	}
	var buf io.ReadWriter
	if body != nil {
		buf = &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(body); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// Do performs an HTTP request and reads the body into the provided interface.
func (c *HTTPClient) Do(req *http.Request, obj interface{}) error {
	res, err := c.HTTP.Do(req)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to perform request with ID: %s", request.IDFromContext(req.Context())))
	}
	defer res.Body.Close() // nolint:errcheck
	if err := c.handleErrors(res); err != nil {
		return err
	}
	if obj != nil {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		return json.Unmarshal(b, &obj)
	}
	return nil
}

// handleErrors invokes the underlying response error handler.
func (c *HTTPClient) handleErrors(res *http.Response) error {
	return c.ErrorHandler.Handle(res)
}

// With returns a new Client after applying given modifiers.
func (c *HTTPClient) With(modifiers ...ClientModifierFn) Client {
	nc := &HTTPClient{
		BaseURL:      c.BaseURL,
		ErrorHandler: c.ErrorHandler,
		HTTP:         c.HTTP,
		UserAgent:    c.UserAgent,
	}
	for _, m := range modifiers {
		m(nc)
	}
	return nc
}

// DefaultErrorHandler is the default operations for handling errors returned by
// the Upbound API.
type DefaultErrorHandler struct{}

// Handle handles HTTP response errors from the Upbound API. Caller is
// responsible for closing response body.
func (h *DefaultErrorHandler) Handle(res *http.Response) error {
	status := res.StatusCode
	if status >= 200 && status < 300 {
		return nil
	}
	var rErr uerrors.Error
	var details *string

	b, err := io.ReadAll(res.Body)
	// if we can read the body, try to unmarshal it into an error
	// and if that fails, use the body as the details
	if err == nil {
		if err := json.Unmarshal(b, &rErr); err == nil && rErr.Status != 0 {
			return &rErr
		}

		bd := string(b)
		if bd != "" {
			details = &bd
		}
	}
	return &uerrors.Error{
		Status: status,
		Title:  http.StatusText(status),
		Detail: details,
	}
}

// ContextTransport is a http.RoundTripper that enables the caller to propagate
// information within the req.Context to external HTTP targets.
type ContextTransport struct {
	transport http.RoundTripper
}

// ContextTransportOption modifies the underlying ContextTransport.
type ContextTransportOption func(*ContextTransport)

// WithTransport overrides the default http.Roundtripper for the
// ContextTransport.
func WithTransport(t http.RoundTripper) ContextTransportOption {
	return func(ct *ContextTransport) {
		ct.transport = t
	}
}

// NewContextTransport constructs a new ContextTransport.
func NewContextTransport(opts ...ContextTransportOption) *ContextTransport {
	c := &ContextTransport{
		transport: http.DefaultTransport,
	}

	for _, o := range opts {
		o(c)
	}
	return c
}

// RoundTrip adds information that is deemed important to propagate to the
// target. Today we only propagate the request-id, but could expand this in
// the future.
func (c *ContextTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Retrieve x-request-id value from context.
	id := request.IDFromContext(req.Context())
	if id == "" {
		id = request.NewID()
	}
	// Add value to the request.
	req.Header.Add(headers.RequestIDHeader, id)

	return c.transport.RoundTrip(req)
}
