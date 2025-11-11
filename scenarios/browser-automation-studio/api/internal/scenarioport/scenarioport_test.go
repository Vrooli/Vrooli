package scenarioport

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveURLUsesRegistryURL(t *testing.T) {
	t.Setenv("SCENARIO_REGISTRY", `{"browser-automation-studio":{"url":"http://127.0.0.1:6000","ports":{"UI_PORT":6000}}}`)
	resetRegistryCacheForTests()

	called := false
	restore := SetPortLookupFuncForTests(func(ctx context.Context, scenario, port string) (int, error) {
		called = true
		return 0, errors.New("should not be called when registry provides URL")
	})
	defer restore()

	resolved, info, err := ResolveURL(context.Background(), "browser-automation-studio", "/dashboard")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatalf("expected registry resolution to skip external port lookup")
	}
	if resolved != "http://127.0.0.1:6000/dashboard" {
		t.Fatalf("unexpected URL: %s", resolved)
	}
	if info == nil || info.Port != 6000 {
		t.Fatalf("expected port info from registry, got %+v", info)
	}
}

func TestResolvePortFallsBackToRegistryPorts(t *testing.T) {
	t.Setenv("SCENARIO_REGISTRY", `[{"name":"app-monitor","ports":{"API_PORT":7777}}]`)
	resetRegistryCacheForTests()

	restore := SetPortLookupFuncForTests(func(ctx context.Context, scenario, port string) (int, error) {
		return 0, errors.New("should not shell to vrooli when registry provides ports")
	})
	defer restore()

	info, err := ResolvePort(context.Background(), "app-monitor")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil || info.Port != 7777 {
		t.Fatalf("expected port 7777, got %+v", info)
	}
}

func TestResolveURLReadsRegistryFile(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "registry.json")
	if err := os.WriteFile(registryPath, []byte(`{"browser-automation-studio":{"url":"http://localhost:5005"}}`), 0o600); err != nil {
		t.Fatalf("failed to write registry file: %v", err)
	}

	t.Setenv("SCENARIO_REGISTRY", "@"+registryPath)
	resetRegistryCacheForTests()

	restore := SetPortLookupFuncForTests(func(ctx context.Context, scenario, port string) (int, error) {
		return 0, errors.New("should not be called when registry file provides URL")
	})
	defer restore()

	resolved, info, err := ResolveURL(context.Background(), "browser-automation-studio", "?demo=true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != "http://localhost:5005?demo=true" {
		t.Fatalf("unexpected URL: %s", resolved)
	}
	if info == nil || info.Port != 5005 {
		t.Fatalf("expected port 5005 from URL, got %+v", info)
	}
}
