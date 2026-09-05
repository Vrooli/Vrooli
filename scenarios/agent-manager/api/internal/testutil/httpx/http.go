package httpx

import (
	"io"
	"net/http"
	"strings"
)

// DoerFunc adapts a function to the common HTTP Do method shape.
type DoerFunc func(req *http.Request) (*http.Response, error)

func (f DoerFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

// Response returns a minimal HTTP response with a string body.
func Response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
