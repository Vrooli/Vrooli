package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/api-core/discovery"
)

func TestScenarioChecker_Available(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	checker := &ScenarioChecker{
		Slug:     "workspace-sandbox",
		Client:   server.Client(),
		Resolver: discovery.NewStaticResolver(server.URL),
	}

	status, msg := checker.Check(context.Background())
	if status != StatusAvailable {
		t.Errorf("expected available, got %s: %s", status, msg)
	}
	if msg == "" {
		t.Error("expected non-empty message")
	}
}

func TestScenarioChecker_HealthFails(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	checker := &ScenarioChecker{
		Slug:     "workspace-sandbox",
		Client:   server.Client(),
		Resolver: discovery.NewStaticResolver(server.URL),
	}

	status, msg := checker.Check(context.Background())
	if status != StatusUnavailable {
		t.Errorf("expected unavailable, got %s: %s", status, msg)
	}
}

func TestScenarioChecker_NotRunning(t *testing.T) {
	notRunningRunner := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("scenario is not running"), errors.New("exit status 1")
	}

	resolver := discovery.NewResolver(discovery.ResolverConfig{
		CommandRunner: notRunningRunner,
	})

	checker := &ScenarioChecker{
		Slug:     "workspace-sandbox",
		Client:   http.DefaultClient,
		Resolver: resolver,
	}

	status, msg := checker.Check(context.Background())
	if status != StatusUnavailable {
		t.Errorf("expected unavailable, got %s: %s", status, msg)
	}
}
