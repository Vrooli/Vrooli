package testutil

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"
)

// Request records one CLI command interaction with the API context seam.
type Request struct {
	Method  string
	Path    string
	Query   url.Values
	Payload any
}

// Response configures the fake context response for a method/path pair.
type Response struct {
	Value any
	Err   error
}

// Context is a reusable appctx.Context fake for command tests.
type Context struct {
	t         testing.TB
	responses map[string]Response
	requests  []Request
}

// NewContext creates a fake CLI API context.
func NewContext(t testing.TB) *Context {
	t.Helper()
	return &Context{
		t:         t,
		responses: make(map[string]Response),
	}
}

// Respond configures a successful response for method/path.
func (c *Context) Respond(method, path string, value any) {
	c.t.Helper()
	c.responses[key(method, path)] = Response{Value: value}
}

// Fail configures an error response for method/path.
func (c *Context) Fail(method, path string, err error) {
	c.t.Helper()
	c.responses[key(method, path)] = Response{Err: err}
}

// Requests returns all recorded requests in call order.
func (c *Context) Requests() []Request {
	requests := make([]Request, len(c.requests))
	copy(requests, c.requests)
	return requests
}

// LastRequest returns the most recent request.
func (c *Context) LastRequest() Request {
	c.t.Helper()
	if len(c.requests) == 0 {
		c.t.Fatal("no CLI context requests recorded")
	}
	return c.requests[len(c.requests)-1]
}

// RequireNoRequests fails if the command touched the API seam.
func (c *Context) RequireNoRequests() {
	c.t.Helper()
	if len(c.requests) > 0 {
		c.t.Fatalf("expected no API requests, got %+v", c.requests)
	}
}

// Get performs a fake GET request.
func (c *Context) Get(path string, result interface{}) error {
	return c.GetWithQuery(path, nil, result)
}

// GetWithQuery performs a fake GET request with query parameters.
func (c *Context) GetWithQuery(path string, query url.Values, result interface{}) error {
	return c.record("GET", path, query, nil, result)
}

// Post performs a fake POST request.
func (c *Context) Post(path string, payload interface{}, result interface{}) error {
	return c.record("POST", path, nil, payload, result)
}

// Put performs a fake PUT request.
func (c *Context) Put(path string, payload interface{}, result interface{}) error {
	return c.record("PUT", path, nil, payload, result)
}

// Delete performs a fake DELETE request.
func (c *Context) Delete(path string) error {
	return c.record("DELETE", path, nil, nil, nil)
}

func (c *Context) record(method, path string, query url.Values, payload any, result interface{}) error {
	c.t.Helper()
	c.requests = append(c.requests, Request{
		Method:  strings.ToUpper(method),
		Path:    path,
		Query:   cloneValues(query),
		Payload: payload,
	})

	response := c.responses[key(method, path)]
	if response.Err != nil {
		return response.Err
	}
	if response.Value == nil || result == nil {
		return nil
	}

	raw, err := json.Marshal(response.Value)
	if err != nil {
		return fmt.Errorf("marshal fake response for %s %s: %w", method, path, err)
	}
	if err := json.Unmarshal(raw, result); err != nil {
		return fmt.Errorf("decode fake response for %s %s: %w", method, path, err)
	}
	return nil
}

func key(method, path string) string {
	return strings.ToUpper(method) + " " + path
}

func cloneValues(values url.Values) url.Values {
	if values == nil {
		return nil
	}
	clone := make(url.Values, len(values))
	for k, v := range values {
		clone[k] = append([]string(nil), v...)
	}
	return clone
}
