package main

import (
	"path/filepath"
	"testing"
	"time"

	runtimeapi "scenario-to-desktop-runtime/api"
	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

type fakeRuntimeClient struct {
	validation *runtimeapi.BundleValidationResult
	secrets    []BundlePreflightSecret
	ready      BundlePreflightReady
	ports      map[string]map[string]int
	telemetry  *BundlePreflightTelemetry
	status     *BundlePreflightRuntime
}

func (f *fakeRuntimeClient) Status() (*BundlePreflightRuntime, error) {
	return f.status, nil
}

func (f *fakeRuntimeClient) Validate() (*runtimeapi.BundleValidationResult, error) {
	return f.validation, nil
}

func (f *fakeRuntimeClient) Secrets() ([]BundlePreflightSecret, error) {
	return f.secrets, nil
}

func (f *fakeRuntimeClient) ApplySecrets(secrets map[string]string) error {
	return nil
}

func (f *fakeRuntimeClient) Ready(request BundlePreflightRequest, timeout time.Duration) (BundlePreflightReady, int, error) {
	return f.ready, 0, nil
}

func (f *fakeRuntimeClient) Ports() (map[string]map[string]int, error) {
	return f.ports, nil
}

func (f *fakeRuntimeClient) Telemetry() (*BundlePreflightTelemetry, error) {
	return f.telemetry, nil
}

func (f *fakeRuntimeClient) LogTails(request BundlePreflightRequest) []BundlePreflightLogTail {
	return nil
}

func TestPreflightServiceDryRun(t *testing.T) {
	service := NewPreflightService(NewInMemorySessionStore(), NewInMemoryJobStore())

	fake := &fakeRuntimeClient{
		validation: &runtimeapi.BundleValidationResult{Valid: true},
		secrets: []BundlePreflightSecret{
			{ID: "API_KEY", Required: true, HasValue: true},
		},
		ready: BundlePreflightReady{Ready: true},
		ports: map[string]map[string]int{
			"api": {"http": 47001},
		},
		telemetry: &BundlePreflightTelemetry{Path: "runtime/telemetry.jsonl"},
		status:    &BundlePreflightRuntime{InstanceID: "fixture"},
	}

	service.newDryRunRuntime = func(manifest *bundlemanifest.Manifest, bundleRoot string, timeout time.Duration) (*preflightRuntimeHandle, error) {
		return &preflightRuntimeHandle{client: fake}, nil
	}

	manifestPath := filepath.Join("..", "runtime", "testdata", "fixture-bundle", "bundle.json")
	resp, err := service.RunBundlePreflight(BundlePreflightRequest{
		BundleManifestPath: manifestPath,
	})
	if err != nil {
		t.Fatalf("RunBundlePreflight returned error: %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("expected status ok, got %q", resp.Status)
	}
	if resp.Validation == nil || !resp.Validation.Valid {
		t.Fatalf("expected validation to pass")
	}
	if resp.Ready == nil || !resp.Ready.Ready {
		t.Fatalf("expected ready response")
	}
	if resp.SessionID != "" {
		t.Fatalf("expected no session id for dry-run, got %q", resp.SessionID)
	}
}
