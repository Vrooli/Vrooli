package main

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/vrooli/api-core/discovery"
	httpx "github.com/vrooli/api-core/servertest"
)

func TestScenarioChecker_Available(t *testing.T) {
	server := httpx.NewServer(t, map[string]http.HandlerFunc{
		"/health": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	})

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
	server := httpx.NewServer(t, map[string]http.HandlerFunc{
		"/health": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		},
	})

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
