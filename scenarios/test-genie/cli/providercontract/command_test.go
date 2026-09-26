package providercontract

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/maturity-go/assessment"
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

func TestParseScanArgsRestart(t *testing.T) {
	got, err := ParseScanArgs([]string{"scan", "branding", "--restart", "--json", "--target", "brand-manager"})
	if err != nil {
		t.Fatalf("ParseScanArgs returned error: %v", err)
	}
	if !got.Restart || !got.JSON || got.Target != "brand-manager" || got.Subject != "branding" {
		t.Fatalf("unexpected scan args: %#v", got)
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
		if joined != "vrooli scenario restart cli-health" {
			t.Fatalf("unexpected command: %s", joined)
		}
		return []byte("ok"), nil
	})
	defer restore()
	srv := newValidationServer(t, validProviderResponse("demo", "cli-health", "contracts", "2026-06-16", "L3"))
	defer srv.Close()
	restoreURL := stubProviderBaseURL(t, srv.URL)
	defer restoreURL()
	restoreFix := stubFixConformanceProbe(t)
	defer restoreFix()

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
	if got.Presentation == nil || got.Presentation.ContractVersion != assessment.PhasePresentationContractVersion {
		t.Fatalf("check result must surface the provider presentation, got %#v", got.Presentation)
	}
}

func TestCheckIncludesCapabilitySummary(t *testing.T) {
	resp := validProviderResponse("demo", "ui-health", "ui-health", "2.0.0", "L0")
	resp.Assessment.Local.NextLevel = "L1"
	resp.Assessment.Capabilities = []*commonv1.CapabilityMaturityAssessment{
		{
			Id:             "interop",
			Label:          "Interop",
			CurrentLevel:   "L5",
			CurrentSummary: "Iframe and proxy interop are clean.",
			Clean:          true,
			PriorityRank:   2,
			Levels: []*commonv1.LocalMaturityLevel{
				{Id: "L0"},
				{Id: "L5"},
			},
		},
		{
			Id:                   "pwa_native_readiness",
			Label:                "PWA Native Readiness",
			CurrentLevel:         "L0",
			NextLevel:            "L1",
			NextUnlock:           "Install metadata baseline.",
			BlockingFindingCodes: []string{"pwa.manifest.missing"},
			Clean:                false,
			PriorityRank:         1,
			PriorityReason:       "lowest current level with required/blocking findings",
			Levels: []*commonv1.LocalMaturityLevel{
				{Id: "L0"},
				{Id: "L1"},
			},
		},
	}
	resp.Assessment.HighestPriorityCapability = &commonv1.PriorityFocus{
		CapabilityId:    "pwa_native_readiness",
		CapabilityLabel: "PWA Native Readiness",
		CurrentLevel:    "L0",
		NextLevel:       "L1",
		Reason:          "lowest current level with required/blocking findings",
	}
	resp.Assessment.Presentation = assessment.BuildPhasePresentation(resp.Assessment)
	srv := newValidationServer(t, resp)
	defer srv.Close()
	restoreURL := stubProviderBaseURL(t, srv.URL)
	defer restoreURL()
	restoreFix := stubFixConformanceProbe(t)
	defer restoreFix()

	probe, err := ResolveProbe("ui-health")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Check(context.Background(), Args{Target: "demo", Restart: false, Timeout: time.Second}, probe)
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if got.Assessment.HighestPriorityCapability == nil || got.Assessment.HighestPriorityCapability.CapabilityID != "pwa_native_readiness" {
		t.Fatalf("focus = %#v, want pwa_native_readiness", got.Assessment.HighestPriorityCapability)
	}
	if len(got.Assessment.Capabilities) != 2 {
		t.Fatalf("capabilities = %#v, want two", got.Assessment.Capabilities)
	}
	if got.Assessment.Capabilities[1].BlockingFindingCount != 1 || got.Assessment.Capabilities[1].NextUnlock == "" {
		t.Fatalf("capability summary missing blocking count/next unlock: %#v", got.Assessment.Capabilities[1])
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
	restoreFix := stubFixConformanceProbe(t)
	defer restoreFix()

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
	restoreFix := stubFixConformanceProbe(t)
	defer restoreFix()

	probe, err := ResolveProbe("contracts")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Check(context.Background(), Args{Target: "demo", Restart: false, Timeout: time.Second}, probe)
	if err == nil || !strings.Contains(err.Error(), `assessment.provider="old-health"`) {
		t.Fatalf("expected provider mismatch error, got %v", err)
	}
}

func TestCheckRejectsMissingPresentation(t *testing.T) {
	resp := validProviderResponse("demo", "cli-health", "contracts", "2026-06-16", "L3")
	resp.Assessment.Presentation = nil
	srv := newValidationServer(t, resp)
	defer srv.Close()
	restoreURL := stubProviderBaseURL(t, srv.URL)
	defer restoreURL()
	restoreFix := stubFixConformanceProbe(t)
	defer restoreFix()

	probe, err := ResolveProbe("contracts")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Check(context.Background(), Args{Target: "demo", Restart: false, Timeout: time.Second}, probe)
	if err == nil || !strings.Contains(err.Error(), "assessment.presentation is required") {
		t.Fatalf("check must fail exactly like the run gate on a missing presentation, got %v", err)
	}
	found := false
	for _, code := range got.ReasonCodes {
		if code == "presentation_invalid" {
			found = true
		}
	}
	if !found {
		t.Fatalf("reason codes = %v, want presentation_invalid", got.ReasonCodes)
	}
}

func TestCheckRejectsMissingMetrics(t *testing.T) {
	resp := validProviderResponse("demo", "cli-health", "contracts", "2026-06-16", "L3")
	resp.Metrics = nil
	srv := newValidationServer(t, resp)
	defer srv.Close()
	restoreURL := stubProviderBaseURL(t, srv.URL)
	defer restoreURL()
	restoreFix := stubFixConformanceProbe(t)
	defer restoreFix()

	probe, err := ResolveProbe("contracts")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Check(context.Background(), Args{Target: "demo", Restart: false, Timeout: time.Second}, probe)
	if err == nil || !strings.Contains(err.Error(), "metrics_missing") {
		t.Fatalf("check must enforce metrics adoption like the fleet scan, got %v", err)
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
	restoreFix := stubFixConformanceProbe(t)
	defer restoreFix()

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
	restoreFix := stubFixConformanceProbe(t)
	defer restoreFix()

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

func TestRestartScanProvidersRestartsEachProviderOnce(t *testing.T) {
	var calls []string
	restore := stubCommandRunner(t, func(name string, args ...string) ([]byte, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return []byte("ok"), nil
	})
	defer restore()

	restartScanProviders(context.Background(), time.Second, "")

	seen := map[string]int{}
	for _, call := range calls {
		if !strings.HasPrefix(call, "vrooli scenario restart ") {
			t.Fatalf("unexpected restart command: %s", call)
		}
		seen[strings.TrimPrefix(call, "vrooli scenario restart ")]++
	}
	for _, probe := range Probes() {
		if got := seen[probe.Provider]; got != 1 {
			t.Fatalf("provider %s restarted %d times, want once", probe.Provider, got)
		}
	}
}

func TestRestartScanProvidersHonorsSubject(t *testing.T) {
	var calls []string
	restore := stubCommandRunner(t, func(name string, args ...string) ([]byte, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return []byte("ok"), nil
	})
	defer restore()

	restartScanProviders(context.Background(), time.Second, "branding")

	if len(calls) != 1 || calls[0] != "vrooli scenario restart brand-manager" {
		t.Fatalf("restart calls = %#v", calls)
	}
}

// stubFixConformanceProbe replaces the live PreviewFix/ApplyFix transport with
// a minimal conforming pair so Check tests stay hermetic even for providers
// whose descriptors declare implemented auto-fixes.
func stubFixConformanceProbe(t *testing.T) func() {
	t.Helper()
	previous := fixConformanceProbe
	fixConformanceProbe = func(ctx context.Context, provider, target, path string, ruleIDs []string, timeout time.Duration) (*scenariovalidationv1.FixResponse, *scenariovalidationv1.FixResponse, error) {
		return &scenariovalidationv1.FixResponse{Scenario: target}, &scenariovalidationv1.FixResponse{Scenario: target}, nil
	}
	return func() {
		fixConformanceProbe = previous
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
	// Embed the generated unimplemented handler so the fake satisfies the full
	// ScenarioValidationService interface (PreviewFix/ApplyFix) while only the
	// ValidateScenario path these tests exercise is overridden below.
	scenariovalidationconnect.UnimplementedScenarioValidationServiceHandler
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
	a := &commonv1.MaturityAssessment{
		Scenario: scenario,
		Provider: provider,
		Phase:    phase,
		Version:  version,
		Local: &commonv1.LocalMaturityAssessment{
			CurrentLevel: level,
		},
	}
	a.Presentation = assessment.BuildPhasePresentation(a)
	return &scenariovalidationv1.ValidateScenarioResponse{
		Scenario:   scenario,
		Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
		Assessment: a,
		Metrics:    &commonv1.ExecutionMetrics{},
	}
}
