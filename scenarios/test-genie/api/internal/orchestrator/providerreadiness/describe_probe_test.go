package providerreadiness

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"

	"test-genie/internal/orchestrator/phasepolicy"
)

// stubProvider serves the shared validation contract so the probe exercises a
// real Connect round-trip rather than an in-process fake.
type stubProvider struct {
	scenariovalidationconnect.UnimplementedScenarioValidationServiceHandler

	describe        *scenariovalidationv1.DescribeProviderResponse
	describeErr     error
	validate        *scenariovalidationv1.ValidateScenarioResponse
	describeCalls   int
	validateCalls   int
	validateLatency time.Duration
}

func (s *stubProvider) DescribeProvider(
	_ context.Context,
	_ *connect.Request[scenariovalidationv1.DescribeProviderRequest],
) (*connect.Response[scenariovalidationv1.DescribeProviderResponse], error) {
	s.describeCalls++
	if s.describeErr != nil {
		return nil, s.describeErr
	}
	return connect.NewResponse(s.describe), nil
}

func (s *stubProvider) ValidateScenario(
	_ context.Context,
	_ *connect.Request[scenariovalidationv1.ValidateScenarioRequest],
) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	s.validateCalls++
	if s.validateLatency > 0 {
		time.Sleep(s.validateLatency)
	}
	if s.validate == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("no validate response configured"))
	}
	return connect.NewResponse(s.validate), nil
}

func serve(t *testing.T, stub *stubProvider) string {
	t.Helper()
	mux := http.NewServeMux()
	path, handler := scenariovalidationconnect.NewScenarioValidationServiceHandler(stub)
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}

func input() Input {
	return Input{
		Phase:            "security",
		ProviderScenario: "security-health",
		TargetScenario:   "browser-automation-studio",
		Policy:           phasepolicy.RequiredProviderPolicy(),
	}
}

func TestDescribeProbeAnswersWithoutTargetAnalysis(t *testing.T) {
	stub := &stubProvider{describe: &scenariovalidationv1.DescribeProviderResponse{
		Provider:    "security-health",
		Phase:       "security",
		SpecVersion: "2.1.0",
		Contract:    "scenario-validation/v1",
	}}
	base := serve(t, stub)

	got, err := describeProbe(context.Background(), base, input())
	if err != nil {
		t.Fatalf("describeProbe: %v", err)
	}
	if !got.Reachable || !got.ContractValid || !got.IdentityMatch {
		t.Fatalf("probe did not accept a healthy provider: %+v", got)
	}
	if got.SpecVersion != "2.1.0" {
		t.Errorf("SpecVersion = %q", got.SpecVersion)
	}
	// The whole point: readiness must never trigger the provider's analysis.
	if stub.validateCalls != 0 {
		t.Errorf("readiness called ValidateScenario %d time(s); it must cost no target work", stub.validateCalls)
	}
}

func TestDefaultProbeFallsBackWhenDescribeUnimplemented(t *testing.T) {
	stub := &stubProvider{
		describeErr: connect.NewError(connect.CodeUnimplemented, errors.New("not adopted")),
		validate: &scenariovalidationv1.ValidateScenarioResponse{
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
			Assessment: newAssessment(t, "security-health", "security", "2.1.0"),
		},
	}
	base := serve(t, stub)

	got, err := legacyValidateProbe(context.Background(), base, input())
	if err != nil {
		t.Fatalf("legacyValidateProbe: %v", err)
	}
	if !got.Reachable || !got.ContractValid || !got.IdentityMatch {
		t.Fatalf("legacy path rejected a healthy provider: %+v", got)
	}
	if got.SpecVersion != "2.1.0" {
		t.Errorf("SpecVersion = %q, want the assessment version", got.SpecVersion)
	}
}

// A non-Unimplemented error means the provider is unhealthy, not unmigrated. It
// must surface rather than silently trigger the expensive legacy path.
func TestDescribeProbeSurfacesNonUnimplementedErrors(t *testing.T) {
	stub := &stubProvider{describeErr: connect.NewError(connect.CodeInternal, errors.New("boom"))}
	base := serve(t, stub)

	_, err := describeProbe(context.Background(), base, input())
	if err == nil {
		t.Fatal("expected an error for an unhealthy provider")
	}
	if connect.CodeOf(err) == connect.CodeUnimplemented {
		t.Error("internal error was misreported as Unimplemented; it would trigger a needless fallback")
	}
}

func TestDescribeProbeRejectsIdentityMismatch(t *testing.T) {
	stub := &stubProvider{describe: &scenariovalidationv1.DescribeProviderResponse{
		Provider: "quality-health", // wrong provider for the security phase
		Phase:    "security",
		Contract: "scenario-validation/v1",
	}}
	base := serve(t, stub)

	got, err := describeProbe(context.Background(), base, input())
	if err != nil {
		t.Fatalf("describeProbe: %v", err)
	}
	if got.IdentityMatch {
		t.Error("probe accepted a provider serving another provider's identity")
	}
	if !got.Reachable || !got.ContractValid {
		t.Errorf("identity mismatch should not be reported as unreachable: %+v", got)
	}
}

func TestDescribeProbeRejectsPhaseMismatch(t *testing.T) {
	stub := &stubProvider{describe: &scenariovalidationv1.DescribeProviderResponse{
		Provider: "security-health",
		Phase:    "quality", // right provider, wrong phase
		Contract: "scenario-validation/v1",
	}}
	base := serve(t, stub)

	got, _ := describeProbe(context.Background(), base, input())
	if got.IdentityMatch {
		t.Error("probe accepted a provider backing a different phase")
	}
}

func TestDescribeProbeRejectsMissingContract(t *testing.T) {
	stub := &stubProvider{describe: &scenariovalidationv1.DescribeProviderResponse{
		Provider: "security-health",
		Phase:    "security",
	}}
	base := serve(t, stub)

	got, _ := describeProbe(context.Background(), base, input())
	if got.ContractValid {
		t.Error("probe accepted a provider that named no contract")
	}
}

func TestClassifyFreshBinaryRequiresReportedSpecVersion(t *testing.T) {
	in := input()
	in.Policy.Freshness = phasepolicy.FreshnessRequireFreshBinary
	in.Policy.ProviderReadiness = phasepolicy.ProviderReadinessRequiredWhenApplicable
	in.Policy.Unavailable = phasepolicy.UnavailableFail

	out := classify(in, probeOutcome{result: ProbeResult{
		Reachable: true, ContractValid: true, IdentityMatch: true,
	}}, false, false)

	if out.Ready {
		t.Error("a provider that cannot prove its spec version passed a fresh-binary gate")
	}
	if out.Status != OutcomeStale {
		t.Errorf("status = %q, want %q", out.Status, OutcomeStale)
	}
}

func TestClassifyFreshBinaryDetectsSpecVersionDrift(t *testing.T) {
	in := input()
	in.Policy.Freshness = phasepolicy.FreshnessRequireFreshBinary
	in.Policy.ProviderReadiness = phasepolicy.ProviderReadinessRequiredWhenApplicable
	in.Policy.Unavailable = phasepolicy.UnavailableFail
	in.ExpectedSpecVersion = "3.0.0"

	out := classify(in, probeOutcome{result: ProbeResult{
		Reachable: true, ContractValid: true, IdentityMatch: true, SpecVersion: "2.1.0",
	}}, false, false)

	if out.Ready {
		t.Error("a binary serving an outdated maturity spec was accepted")
	}
	if out.Status != OutcomeStale {
		t.Errorf("status = %q, want %q", out.Status, OutcomeStale)
	}
}

func TestClassifyFreshBinaryAcceptsMatchingSpecVersion(t *testing.T) {
	in := input()
	in.Policy.Freshness = phasepolicy.FreshnessRequireFreshBinary
	in.ExpectedSpecVersion = "2.1.0"

	out := classify(in, probeOutcome{result: ProbeResult{
		Reachable: true, ContractValid: true, IdentityMatch: true, SpecVersion: "2.1.0",
	}}, false, false)

	if !out.Ready {
		t.Errorf("a fresh binary was refused: %+v", out)
	}
}

// The default policy is require_live_contract, not require_fresh_binary, so a
// provider that reports no spec version must still be usable.
func TestClassifyLiveContractIgnoresSpecVersion(t *testing.T) {
	in := input()
	out := classify(in, probeOutcome{result: ProbeResult{
		Reachable: true, ContractValid: true, IdentityMatch: true,
	}}, false, false)
	if !out.Ready {
		t.Errorf("live-contract policy rejected a healthy provider: %+v", out)
	}
}

// newAssessment builds the minimal valid assessment the legacy probe's identity
// check requires.
func newAssessment(t *testing.T, provider, phase, version string) *commonv1.MaturityAssessment {
	t.Helper()
	return &commonv1.MaturityAssessment{
		Scenario: "browser-automation-studio",
		Provider: provider,
		Phase:    phase,
		Version:  version,
		Local: &commonv1.LocalMaturityAssessment{
			CurrentLevel: "L1",
		},
	}
}

func TestProbeAtPrefersDescribeAndSkipsValidateEntirely(t *testing.T) {
	stub := &stubProvider{describe: &scenariovalidationv1.DescribeProviderResponse{
		Provider: "security-health", Phase: "security", SpecVersion: "2.1.0",
		Contract: "scenario-validation/v1",
	}}
	base := serve(t, stub)

	got, err := probeAt(context.Background(), base, input())
	if err != nil {
		t.Fatalf("probeAt: %v", err)
	}
	if !got.IdentityMatch {
		t.Fatalf("healthy provider rejected: %+v", got)
	}
	if stub.describeCalls != 1 {
		t.Errorf("describeCalls = %d, want 1", stub.describeCalls)
	}
	if stub.validateCalls != 0 {
		t.Errorf("validateCalls = %d, want 0 — the duplicate analysis this change removes", stub.validateCalls)
	}
}

func TestProbeAtFallsBackOnlyForUnimplemented(t *testing.T) {
	stub := &stubProvider{
		describeErr: connect.NewError(connect.CodeUnimplemented, errors.New("not adopted")),
		validate: &scenariovalidationv1.ValidateScenarioResponse{
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
			Assessment: newAssessment(t, "security-health", "security", "2.1.0"),
		},
	}
	base := serve(t, stub)

	got, err := probeAt(context.Background(), base, input())
	if err != nil {
		t.Fatalf("probeAt: %v", err)
	}
	if !got.IdentityMatch {
		t.Fatalf("fallback rejected a healthy unmigrated provider: %+v", got)
	}
	if stub.describeCalls != 1 || stub.validateCalls != 1 {
		t.Errorf("describeCalls=%d validateCalls=%d, want 1 and 1", stub.describeCalls, stub.validateCalls)
	}
}

// An unhealthy provider must not be given a second, far more expensive chance.
func TestProbeAtDoesNotFallBackOnUnhealthyProvider(t *testing.T) {
	stub := &stubProvider{
		describeErr: connect.NewError(connect.CodeUnavailable, errors.New("starting up")),
		validate: &scenariovalidationv1.ValidateScenarioResponse{
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
			Assessment: newAssessment(t, "security-health", "security", "2.1.0"),
		},
	}
	base := serve(t, stub)

	if _, err := probeAt(context.Background(), base, input()); err == nil {
		t.Fatal("expected the unhealthy provider's error to surface")
	}
	if stub.validateCalls != 0 {
		t.Errorf("validateCalls = %d; an unhealthy provider must not trigger a full analysis", stub.validateCalls)
	}
}
