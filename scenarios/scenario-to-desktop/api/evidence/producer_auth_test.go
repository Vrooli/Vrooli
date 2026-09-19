package evidence

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type authRoundTripper func(*http.Request) (*http.Response, error)

func (f authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestDeploymentManagerServiceAuthTransportAddsConfiguredToken(t *testing.T) {
	t.Setenv("DEPLOYMENT_MANAGER_SERVICE_TOKEN", "owner-token")
	var got string
	transport := deploymentManagerServiceAuthTransport{base: authRoundTripper(func(req *http.Request) (*http.Response, error) {
		got = req.Header.Get("Authorization")
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header), Request: req}, nil
	})}
	if _, err := transport.RoundTrip(httptestRequest()); err != nil {
		t.Fatal(err)
	}
	if got != "Bearer owner-token" {
		t.Fatalf("authorization = %q, want bearer token", got)
	}
}

func TestDeploymentManagerServiceAuthTransportDoesNotInventToken(t *testing.T) {
	t.Setenv("DEPLOYMENT_MANAGER_SERVICE_TOKEN", "")
	var got string
	transport := deploymentManagerServiceAuthTransport{base: authRoundTripper(func(req *http.Request) (*http.Response, error) {
		got = req.Header.Get("Authorization")
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header), Request: req}, nil
	})}
	if _, err := transport.RoundTrip(httptestRequest()); err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("authorization = %q, want empty", got)
	}
}

func httptestRequest() *http.Request {
	return &http.Request{Method: http.MethodPost, URL: mustURL("http://example.test"), Header: make(http.Header)}
}

func mustURL(raw string) *url.URL {
	value, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return value
}
