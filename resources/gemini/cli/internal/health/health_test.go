package health

import (
	"context"
	"errors"
	"io"
	"net/http"
	"resource-gemini/cli/internal/auth"
	"strings"
	"testing"

	resourceenv "resource-gemini/cli/internal/env"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestProbeAuthenticated(t *testing.T) {
	t.Parallel()

	client := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://generativelanguage.googleapis.com/v1beta/models?key=secret-key" {
			t.Fatalf("Probe() url = %q", req.URL.String())
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}, nil
	})

	result, err := Probe(context.Background(), client, testRuntime(), auth.Credentials{APIKey: "secret-key"})
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if result.Status != "reachable" {
		t.Fatalf("Probe().Status = %q", result.Status)
	}
}

func TestListModels(t *testing.T) {
	t.Parallel()

	client := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"models":[{"name":"models/gemini-pro"},{"name":"models/gemini-1.5-flash"}]}`)),
		}, nil
	})

	models, err := ListModels(context.Background(), client, testRuntime(), auth.Credentials{APIKey: "secret-key"})
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if len(models) != 2 || models[0] != "gemini-pro" || models[1] != "gemini-1.5-flash" {
		t.Fatalf("ListModels() = %v", models)
	}
}

func TestProbeUnreachable(t *testing.T) {
	t.Parallel()

	client := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("dial tcp timeout")
	})

	result, err := Probe(context.Background(), client, testRuntime(), auth.Credentials{})
	if err == nil {
		t.Fatal("Probe() error = nil, want non-nil")
	}
	if result.Status != "unreachable" {
		t.Fatalf("Probe().Status = %q", result.Status)
	}
}

func testRuntime() resourceenv.Runtime {
	return resourceenv.Runtime{
		APIBaseURL: "https://generativelanguage.googleapis.com/v1beta",
	}
}
