//go:build integration

package integration_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"backdrop-studio/integration"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// Rasterize must honor the Environment HTTP client just like the JSON calls.
// Callers inject it for deterministic routing, authentication, and transport
// policy; silently using DefaultClient bypasses all three.
func TestRasterizeUsesEnvironmentHTTPClient(t *testing.T) {
	called := false
	want := []byte("rendered-png")
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		called = true
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "/api/v1/ops/convert", req.URL.Path)
		require.Contains(t, req.Header.Get("Content-Type"), "multipart/form-data")
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), "document.svg")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(string(want))),
			Request:    req,
		}, nil
	})}

	env := integration.Environment{
		ImageToolsURL: "http://image-tools.invalid",
		HTTP:          client,
	}
	got, err := env.Rasterize(context.Background(), []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`))
	require.NoError(t, err)
	require.True(t, called, "Rasterize bypassed Environment.HTTP")
	require.Equal(t, want, got)
}

func TestCompositeUsesEnvironmentHTTPClient(t *testing.T) {
	called := false
	want := []byte("composited-png")
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		called = true
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "/api/v1/ops/composite", req.URL.Path)
		require.Contains(t, req.Header.Get("Content-Type"), "multipart/form-data")
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), "plate0.png")
		require.Contains(t, string(body), `"width":1`)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(string(want))),
			Request:    req,
		}, nil
	})}

	env := integration.Environment{
		ImageToolsURL: "http://image-tools.invalid",
		HTTP:          client,
	}
	got, err := env.Composite(context.Background(), []integration.CompositePlate{{
		Name: "base", Blend: "normal", Opacity: 1, PNG: []byte("plate pixels"),
	}}, 1, 1)
	require.NoError(t, err)
	require.True(t, called, "Composite bypassed Environment.HTTP")
	require.Equal(t, want, got)
}
