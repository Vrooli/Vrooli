package httpx

import (
	"io"
	"net/http"
	"strings"
)

// Doer is the narrow HTTP client seam used by tests and local adapters.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// DoerFunc adapts a function to Doer.
type DoerFunc func(*http.Request) (*http.Response, error)

func (f DoerFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

// Response returns a minimal, fully closable HTTP response for tests.
func Response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode:    status,
		Status:        http.StatusText(status),
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
		Header:        make(http.Header),
	}
}
