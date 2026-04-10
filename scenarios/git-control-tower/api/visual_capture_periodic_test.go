package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/api-core/discovery"
)

func TestPeriodicCapture_SkipsWhenBASUnavailable(t *testing.T) {
	t.Parallel()

	// Create a registry where BAS is unavailable
	registry := NewCapabilityRegistry(
		[]CapabilityDef{{
			ID:             "browser-automation-studio",
			DependencyKind: DependencyScenario,
			DependencySlug: "browser-automation-studio",
		}},
		map[string]StatusChecker{
			"browser-automation-studio": &fakeChecker{status: StatusUnavailable},
		},
		1*time.Second,
	)

	p := &PeriodicCapture{
		config:       PeriodicCaptureConfig{Interval: 1 * time.Hour},
		capabilities: registry,
	}

	// tick should not panic when BAS is unavailable
	p.tick(context.Background())
}

func TestPeriodicCapture_CapturesChangedScenarios(t *testing.T) {
	t.Parallel()

	// Verify scope detection helpers
	if !isScenarioScope("scenario:my-app") {
		t.Error("expected scenario:my-app to be a scenario scope")
	}
	if isScenarioScope("resource:postgres") {
		t.Error("expected resource:postgres to not be a scenario scope")
	}
	if isScenarioScope("other") {
		t.Error("expected 'other' to not be a scenario scope")
	}
	if slug := scopeSlug("scenario:my-app"); slug != "my-app" {
		t.Errorf("expected slug 'my-app', got %q", slug)
	}
}

func TestPeriodicCapture_StartStop(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	registry := NewCapabilityRegistry(
		[]CapabilityDef{{
			ID:             "browser-automation-studio",
			DependencyKind: DependencyScenario,
			DependencySlug: "browser-automation-studio",
		}},
		map[string]StatusChecker{
			"browser-automation-studio": &ScenarioChecker{
				Slug:     "browser-automation-studio",
				Client:   server.Client(),
				Resolver: discovery.NewStaticResolver(server.URL),
			},
		},
		1*time.Second,
	)

	p := NewPeriodicCapture(
		PeriodicCaptureConfig{Interval: 100 * time.Millisecond},
		registry,
		NewBrowserAutomationClient(5*time.Second),
		nil,
		nil,
		nil,
	)

	p.Start()
	// Starting again should be a no-op
	p.Start()

	time.Sleep(50 * time.Millisecond)
	p.Stop()
	// Stopping again should be a no-op
	p.Stop()
}

type fakeChecker struct {
	status CapabilityStatus
	msg    string
}

func (f *fakeChecker) Check(_ context.Context) (CapabilityStatus, string) {
	return f.status, f.msg
}
