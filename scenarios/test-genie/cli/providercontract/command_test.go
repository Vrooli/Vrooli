package providercontract

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	catalog "test-genie/internal/orchestrator/phases"
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
	if byPhase.Phase != byProvider.Phase || byPhase.Provider != byProvider.Provider {
		t.Fatalf("phase/provider resolved different probes: %#v != %#v", byPhase, byProvider)
	}
}

func TestProbesResolveAgainstCatalog(t *testing.T) {
	probes := Probes()
	if len(probes) == 0 {
		t.Fatal("Probes returned no delegated phases")
	}
	for _, probe := range probes {
		spec, ok := catalog.DefaultCatalog().Lookup(probe.Phase)
		if !ok {
			t.Fatalf("probe phase %q missing from catalog", probe.Phase)
		}
		if spec.Delegated == nil {
			t.Fatalf("probe phase %q is not delegated in catalog", probe.Phase)
		}
		if probe.Provider != spec.Delegated.ProviderScenario {
			t.Fatalf("probe phase %q provider = %q, want %q", probe.Phase, probe.Provider, spec.Delegated.ProviderScenario)
		}
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
	srv := newValidationServer(t, validProviderResponse("demo", "cli-health", "contracts", "2026-06-16", "L3"))
	defer srv.Close()
	restoreURL := stubProviderBaseURL(t, srv.URL)
	defer restoreURL()

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
	srv := newValidationServer(t, &scenariovalidationv1.ValidateScenarioResponse{
		Scenario: "demo",
		Status:   scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
	})
	defer srv.Close()
	restoreURL := stubProviderBaseURL(t, srv.URL)
	defer restoreURL()

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
	srv := newValidationServer(t, validProviderResponse("demo", "old-health", "contracts", "2026-06-16", "L3"))
	defer srv.Close()
	restoreURL := stubProviderBaseURL(t, srv.URL)
	defer restoreURL()

	probe, err := ResolveProbe("contracts")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Check(context.Background(), Args{Target: "demo", Restart: false, Timeout: time.Second}, probe)
	if err == nil || !strings.Contains(err.Error(), `assessment.provider="old-health"`) {
		t.Fatalf("expected provider mismatch error, got %v", err)
	}
}

func TestCheckStandardsValidatesAssessmentFromNonZeroProviderOutput(t *testing.T) {
	restore := stubCommandRunner(t, func(name string, args ...string) ([]byte, error) {
		return []byte(`{"result":{"summary":` + validProviderJSON("demo", "scenario-auditor", "standards", "2026-06-16", "L2") + `}}`), errors.New("exit status 1")
	})
	defer restore()

	probe, err := ResolveProbe("standards")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Check(context.Background(), Args{Target: "demo", Restart: false, Timeout: time.Second}, probe)
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if got.Provider != "scenario-auditor" || got.Assessment.CurrentLevel != "L2" {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestCheckDocsProviderRestartsThenValidatesRPCWrapperAssessment(t *testing.T) {
	restore := stubCommandRunner(t, func(name string, args ...string) ([]byte, error) {
		joined := name + " " + strings.Join(args, " ")
		switch joined {
		case "vrooli scenario restart knowledge-observatory":
			return []byte("ok"), nil
		default:
			t.Fatalf("unexpected command: %s", joined)
			return nil, nil
		}
	})
	defer restore()
	srv := newValidationServer(t, validProviderResponse("demo", "knowledge-observatory", "docs", "2026-06-16", "L2"))
	defer srv.Close()
	restoreURL := stubProviderBaseURL(t, srv.URL)
	defer restoreURL()

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

func TestCheckStandardsProviderAcceptsNestedSummaryAssessment(t *testing.T) {
	restore := stubCommandRunner(t, func(name string, args ...string) ([]byte, error) {
		joined := name + " " + strings.Join(args, " ")
		switch joined {
		case "scenario-auditor standards scan demo --wait --json":
			return []byte(`{"result":{"summary":` + validProviderJSON("demo", "scenario-auditor", "standards", "2026-06-16", "L2") + `}}`), nil
		default:
			t.Fatalf("unexpected command: %s", joined)
			return nil, nil
		}
	})
	defer restore()

	probe, err := ResolveProbe("standards")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Check(context.Background(), Args{Target: "demo", Restart: false, Timeout: time.Second}, probe)
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if got.Provider != "scenario-auditor" || got.Assessment.CurrentLevel != "L2" {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestCheckTidinessProviderUsesLifecycleDiscoveredRPCProbe(t *testing.T) {
	restoreCommands := stubCommandRunner(t, func(name string, args ...string) ([]byte, error) {
		joined := name + " " + strings.Join(args, " ")
		if joined != "vrooli scenario restart tidiness-manager" {
			t.Fatalf("unexpected command: %s", joined)
		}
		return []byte("ok"), nil
	})
	defer restoreCommands()

	srv := newValidationServer(t, validProviderResponse("demo", "tidiness-manager", "tidiness", "2026-06-16", "L1"))
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

type fakeValidationService struct {
	resp *scenariovalidationv1.ValidateScenarioResponse
}

func (f fakeValidationService) ValidateScenario(context.Context, *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	return connect.NewResponse(f.resp), nil
}

func newValidationServer(t *testing.T, resp *scenariovalidationv1.ValidateScenarioResponse) *httptest.Server {
	t.Helper()
	_, handler := scenariovalidationconnect.NewScenarioValidationServiceHandler(fakeValidationService{resp: resp})
	return httptest.NewServer(handler)
}

func validProviderResponse(scenario, provider, phase, version, level string) *scenariovalidationv1.ValidateScenarioResponse {
	return &scenariovalidationv1.ValidateScenarioResponse{
		Scenario: scenario,
		Status:   scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
		Assessment: &commonv1.MaturityAssessment{
			Scenario: scenario,
			Provider: provider,
			Phase:    phase,
			Version:  version,
			Local: &commonv1.LocalMaturityAssessment{
				CurrentLevel: level,
			},
		},
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
