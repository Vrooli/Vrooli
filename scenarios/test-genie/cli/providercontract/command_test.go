package providercontract

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseArgs(t *testing.T) {
	got, err := ParseArgs([]string{"check", "contracts", "demo", "--no-restart", "--json"})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if got.Subject != "contracts" || got.Target != "demo" || got.Restart || !got.JSON {
		t.Fatalf("unexpected args: %#v", got)
	}
}

func TestResolveProbeAcceptsPhaseAndProvider(t *testing.T) {
	byPhase, err := ResolveProbe("contracts")
	if err != nil {
		t.Fatalf("ResolveProbe(phase): %v", err)
	}
	byProvider, err := ResolveProbe("cli-health")
	if err != nil {
		t.Fatalf("ResolveProbe(provider): %v", err)
	}
	if byPhase.Phase != byProvider.Phase || byPhase.Provider != byProvider.Provider || strings.Join(byPhase.Invocation, " ") != strings.Join(byProvider.Invocation, " ") {
		t.Fatalf("phase/provider resolved different probes: %#v != %#v", byPhase, byProvider)
	}
}

func TestCheckRestartsThenValidatesAssessment(t *testing.T) {
	restore := stubCommandRunner(t, func(name string, args ...string) ([]byte, error) {
		joined := name + " " + strings.Join(args, " ")
		switch joined {
		case "vrooli scenario restart cli-health":
			return []byte("ok"), nil
		case "cli-health validate scenario demo --json":
			return []byte(validProviderJSON("demo", "cli-health", "contracts", "2026-06-16", "L3")), nil
		default:
			t.Fatalf("unexpected command: %s", joined)
			return nil, nil
		}
	})
	defer restore()

	probe, err := ResolveProbe("contracts")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Check(context.Background(), Args{Target: "demo", Restart: true, Timeout: time.Second}, probe)
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if !got.Restarted || got.Status != "ok" || got.Assessment.CurrentLevel != "L3" {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestCheckRejectsMissingAssessment(t *testing.T) {
	restore := stubCommandRunner(t, func(name string, args ...string) ([]byte, error) {
		return []byte(`{"scenario":"demo"}`), nil
	})
	defer restore()

	probe, err := ResolveProbe("contracts")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Check(context.Background(), Args{Target: "demo", Restart: false, Timeout: time.Second}, probe)
	if err == nil || !strings.Contains(err.Error(), "assessment is required") {
		t.Fatalf("expected missing assessment error, got %v", err)
	}
}

func TestCheckRejectsStaleProviderIdentity(t *testing.T) {
	restore := stubCommandRunner(t, func(name string, args ...string) ([]byte, error) {
		return []byte(validProviderJSON("demo", "old-health", "contracts", "2026-06-16", "L3")), nil
	})
	defer restore()

	probe, err := ResolveProbe("contracts")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Check(context.Background(), Args{Target: "demo", Restart: false, Timeout: time.Second}, probe)
	if err == nil || !strings.Contains(err.Error(), `assessment.provider="old-health"`) {
		t.Fatalf("expected provider mismatch error, got %v", err)
	}
}

func TestCheckDocsProviderRestartsThenValidatesRPCWrapperAssessment(t *testing.T) {
	restore := stubCommandRunner(t, func(name string, args ...string) ([]byte, error) {
		joined := name + " " + strings.Join(args, " ")
		switch joined {
		case "vrooli scenario restart knowledge-observatory":
			return []byte("ok"), nil
		case "knowledge-observatory docs health demo --json":
			return []byte(validProviderJSON("demo", "knowledge-observatory", "docs", "2026-06-16", "L2")), nil
		default:
			t.Fatalf("unexpected command: %s", joined)
			return nil, nil
		}
	})
	defer restore()

	probe, err := ResolveProbe("docs")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Check(context.Background(), Args{Target: "demo", Restart: true, Timeout: time.Second}, probe)
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if !got.Restarted || got.Provider != "knowledge-observatory" || got.Assessment.Phase != "docs" {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestCheckTidinessProviderUsesLifecycleDiscoveredHTTPProbe(t *testing.T) {
	restoreCommands := stubCommandRunner(t, func(name string, args ...string) ([]byte, error) {
		joined := name + " " + strings.Join(args, " ")
		if joined != "vrooli scenario restart tidiness-manager" {
			t.Fatalf("unexpected command: %s", joined)
		}
		return []byte("ok"), nil
	})
	defer restoreCommands()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/scan/tidiness" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(validProviderJSON("demo", "tidiness-manager", "tidiness", "2026-06-16", "L1")))
	}))
	defer srv.Close()
	restoreURL := stubProviderBaseURL(t, srv.URL)
	defer restoreURL()

	probe, err := ResolveProbe("tidiness")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Check(context.Background(), Args{Target: "demo", Restart: true, Timeout: time.Second}, probe)
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if !got.Restarted || got.Provider != "tidiness-manager" || got.Assessment.CurrentLevel != "L1" {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestCheckReportsRestartFailure(t *testing.T) {
	restore := stubCommandRunner(t, func(name string, args ...string) ([]byte, error) {
		return nil, errors.New("not running")
	})
	defer restore()

	probe, err := ResolveProbe("contracts")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Check(context.Background(), Args{Target: "demo", Restart: true, Timeout: time.Second}, probe)
	if err == nil || !strings.Contains(err.Error(), "restart provider cli-health via lifecycle") {
		t.Fatalf("expected restart failure, got %v", err)
	}
}

func stubCommandRunner(t *testing.T, fn func(name string, args ...string) ([]byte, error)) func() {
	t.Helper()
	previous := commandRunner
	commandRunner = func(ctx context.Context, timeout time.Duration, dir string, name string, args ...string) ([]byte, error) {
		return fn(name, args...)
	}
	return func() {
		commandRunner = previous
	}
}

func stubProviderBaseURL(t *testing.T, url string) func() {
	t.Helper()
	previous := resolveProviderBaseURL
	resolveProviderBaseURL = func(ctx context.Context, provider string) (string, error) {
		return url, nil
	}
	return func() {
		resolveProviderBaseURL = previous
	}
}

func validProviderJSON(scenario, provider, phase, version, level string) string {
	return `{
  "assessment": {
    "scenario": "` + scenario + `",
    "provider": "` + provider + `",
    "phase": "` + phase + `",
    "version": "` + version + `",
    "local": {
      "currentLevel": "` + level + `"
    }
  }
}`
}
