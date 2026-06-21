package validationprovider

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"test-genie/internal/shared"

	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/audit"
	cartosharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/shared"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/anypb"
)

type fakeClient struct {
	resp *scenariovalidationv1.ValidateScenarioResponse
	err  error
}

func (f fakeClient) ValidateScenario(context.Context, *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(f.resp), nil
}

// capturingClient records the request it was called with so tests can assert
// what Run put on the wire.
type capturingClient struct {
	resp *scenariovalidationv1.ValidateScenarioResponse
	got  *scenariovalidationv1.ValidateScenarioRequest
}

func (c *capturingClient) ValidateScenario(_ context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	c.got = req.Msg
	return connect.NewResponse(c.resp), nil
}

// TestRunSendsScenarioPath proves Run forwards the physical scenario path as
// ValidateScenarioRequest.path, so providers can validate scenarios that live
// outside the repo scenarios/ registry (deep template validation's temp dir).
func TestRunSendsScenarioPath(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	cap := &capturingClient{resp: &scenariovalidationv1.ValidateScenarioResponse{
		Scenario:   "demo",
		Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
		Assessment: testAssessment(""),
	}}
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client { return cap }
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })

	const wantPath = "/tmp/vrooli-template-deep-123/scenarios/demo"
	Run(context.Background(), testProvider(false), "demo", wantPath)

	if cap.got == nil {
		t.Fatal("ValidateScenario was not called")
	}
	if cap.got.GetScenario() != "demo" {
		t.Fatalf("scenario = %q, want demo", cap.got.GetScenario())
	}
	if cap.got.GetPath() != wantPath {
		t.Fatalf("path = %q, want %q", cap.got.GetPath(), wantPath)
	}
}

func testProvider(optional bool) Provider {
	return Provider{
		Phase:            "proto",
		ProviderScenario: "proto-health",
		FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_PROTO,
		Emoji:            "P",
		Optional:         optional,
		Timeout:          time.Second,
	}
}

func testAssessment(severity string) *commonv1.MaturityAssessment {
	a := &commonv1.MaturityAssessment{
		Scenario: "demo",
		Provider: "proto-health",
		Phase:    "proto",
		Version:  "1",
		Local:    &commonv1.LocalMaturityAssessment{CurrentLevel: "L1", NextLevel: "L2", Clean: true, UnknownCount: 1},
		FindingsBySeverity: map[string]int32{
			severity: 1,
		},
	}
	if severity != "" {
		a.Findings = []*commonv1.AssessmentFinding{{
			Code:        "proto.gen_out_of_sync",
			Severity:    severity,
			Title:       "Generated protos stale",
			Message:     "Regenerate proto artifacts",
			Location:    "packages/proto/gen",
			Remediation: "run make generate",
			Maturity: &commonv1.FindingMaturity{
				Dimension:    "proto-health",
				GlobalImpact: commonv1.GlobalImpact_GLOBAL_IMPACT_FOUNDATION_BLOCKER,
			},
		}}
	}
	return a
}

func testArchitectureProvider() Provider {
	return Provider{
		Phase:            "architecture",
		ProviderScenario: "architecture-cartographer",
		FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE,
		Emoji:            "A",
		Timeout:          time.Second,
		GateEnvVar:       "TEST_GENIE_ARCHITECTURE_GATE",
		DefaultGateMode:  GateModeHighConfidence,
	}
}

func testArchitectureResponse(t *testing.T, severity string, authority auditv1.AuthorityConfidence) *scenariovalidationv1.ValidateScenarioResponse {
	t.Helper()
	return testArchitectureResponseWithClass(t, severity, authority, cartosharedv1.FindingClass_FINDING_CLASS_DETERMINISTIC)
}

func testArchitectureResponseWithClass(t *testing.T, severity string, authority auditv1.AuthorityConfidence, class cartosharedv1.FindingClass) *scenariovalidationv1.ValidateScenarioResponse {
	return testArchitectureResponseWithTypeAndClass(t, "proto.gen_out_of_sync", severity, authority, class)
}

func testArchitectureResponseWithTypeAndClass(t *testing.T, findingType, severity string, authority auditv1.AuthorityConfidence, class cartosharedv1.FindingClass) *scenariovalidationv1.ValidateScenarioResponse {
	t.Helper()
	detail, err := anypb.New(&auditv1.AuditRunResponse{
		Scenario:            "demo",
		AuthorityConfidence: authority,
		Categories: []*auditv1.AuditCategory{{
			Key:   "placement_legibility",
			Label: "Placement legibility",
			Score: 0.75,
			TopItems: []*auditv1.CategoryTopItem{{
				Type:         "mislocated_file",
				Severity:     cartosharedv1.Severity_SEVERITY_WARN,
				FindingClass: cartosharedv1.FindingClass_FINDING_CLASS_HEURISTIC,
				Headline:     "mislocated_file @ api/internal/x/x.go",
			}},
		}},
		Findings: []*auditv1.ConflictSummary{{
			Type:         findingType,
			Severity:     cartosharedv1.Severity_SEVERITY_BLOCKER,
			Locations:    []string{"packages/proto/gen"},
			FindingClass: class,
		}},
	})
	if err != nil {
		t.Fatalf("pack native detail: %v", err)
	}
	// The architecture provider's assessment must carry the architecture identity
	// so the consumer's unified RequireIdentity check (provider/phase match) is
	// satisfied; testAssessment defaults to the proto-health identity.
	arch := testAssessment(severity)
	arch.Provider = "architecture-cartographer"
	arch.Phase = "architecture"
	return &scenariovalidationv1.ValidateScenarioResponse{
		Scenario:     "demo",
		Status:       scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED,
		Assessment:   arch,
		NativeDetail: detail,
	}
}

func TestArchitectureGateKeepsHeuristicBlockersAdvisory(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: testArchitectureResponseWithClass(t, "SEVERITY_BLOCKER", auditv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_HIGH, cartosharedv1.FindingClass_FINDING_CLASS_HEURISTIC)}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })
	t.Setenv("TEST_GENIE_ARCHITECTURE_GATE", "")

	got := Run(context.Background(), testArchitectureProvider(), "demo", "")
	if !got.Success {
		t.Fatalf("heuristic blocker should remain advisory: %v", got.Error)
	}
	if got.Summary.GatedBlockers != 0 {
		t.Fatalf("gated blockers = %d, want 0", got.Summary.GatedBlockers)
	}
	if len(got.Findings) != 1 || got.Findings[0].GetFindingClass() != architecturev1.FindingClass_FINDING_CLASS_HEURISTIC {
		t.Fatalf("native class not copied to normalized finding: %+v", got.Findings)
	}
	if hasObservationType(got.Observations, shared.ObservationError) {
		t.Fatalf("heuristic blocker should not render as an error observation: %+v", got.Observations)
	}
	if joined := strings.Join(observationStrings(got.Observations), "\n"); !strings.Contains(joined, "advisory:") {
		t.Fatalf("heuristic blocker should be labeled advisory:\n%s", joined)
	}
}

func TestRunFailedStatusEmitsFindingAndFails(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: &scenariovalidationv1.ValidateScenarioResponse{
			Scenario:   "demo",
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED,
			Assessment: testAssessment("SEVERITY_ERROR"),
		}}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })

	got := Run(context.Background(), testProvider(false), "demo", "")
	if got.Success {
		t.Fatal("expected failed shared status to fail the phase")
	}
	if got.FailureClass != shared.FailureClassTestFailure {
		t.Fatalf("FailureClass = %q, want test_failure", got.FailureClass)
	}
	if got.Remediation != "Run `proto-health validate scenario demo` for details." {
		t.Fatalf("Remediation = %q", got.Remediation)
	}
	if len(got.Findings) != 1 || got.Findings[0].GetSource() != architecturev1.FindingSource_FINDING_SOURCE_PROTO {
		t.Fatalf("expected one PROTO finding, got %+v", got.Findings)
	}
	if got.Summary.LocalCurrentLevel != "L1" || got.Summary.LocalNextLevel != "L2" {
		t.Fatalf("summary local = %q/%q, want L1/L2", got.Summary.LocalCurrentLevel, got.Summary.LocalNextLevel)
	}
	if !got.Summary.LocalClean || got.Summary.LocalUnknownCount != 1 {
		t.Fatalf("summary clean/unknown = %v/%d, want true/1", got.Summary.LocalClean, got.Summary.LocalUnknownCount)
	}
}

func TestRunFailedStatusUsesProviderDetailCommand(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: &scenariovalidationv1.ValidateScenarioResponse{
			Scenario:   "demo",
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED,
			Assessment: testAssessment("SEVERITY_ERROR"),
		}}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })

	provider := testProvider(false)
	provider.DetailCommand = "scenario-dependency-analyzer health {{scenario}}"
	got := Run(context.Background(), provider, "demo", "")
	if got.Remediation != "Run `scenario-dependency-analyzer health demo` for details." {
		t.Fatalf("Remediation = %q", got.Remediation)
	}
}

func TestRunSummaryCountsFindingSeverityAliases(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: &scenariovalidationv1.ValidateScenarioResponse{
			Scenario:   "demo",
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
			Assessment: testAssessment("FINDING_SEVERITY_WARNING"),
		}}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })

	got := Run(context.Background(), testProvider(false), "demo", "")
	if got.Summary.Warnings != 1 {
		t.Fatalf("Warnings = %d, want 1", got.Summary.Warnings)
	}
}

func TestRunOptionalProviderUnavailableSkips(t *testing.T) {
	prevResolve := ResolveBaseURL
	ResolveBaseURL = func(context.Context, string) (string, error) { return "", errors.New("not running") }
	t.Cleanup(func() { ResolveBaseURL = prevResolve })

	got := Run(context.Background(), testProvider(true), "demo", "")
	if !got.Success || !got.Summary.Skipped {
		t.Fatalf("optional unavailable provider should skip successfully, got success=%v summary=%+v", got.Success, got.Summary)
	}
	if len(got.Findings) != 0 {
		t.Fatalf("skip should emit no findings, got %+v", got.Findings)
	}
}

func TestRunMissingAssessmentIsMaturityContract(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: &scenariovalidationv1.ValidateScenarioResponse{
			Scenario: "demo",
			Status:   scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
		}}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })

	got := Run(context.Background(), testProvider(false), "demo", "")
	if got.FailureClass != shared.FailureClassMaturityContract {
		t.Fatalf("FailureClass = %q, want maturity_contract", got.FailureClass)
	}
}

func TestArchitectureGateFailsBlockersAtHighConfidence(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: testArchitectureResponse(t, "SEVERITY_BLOCKER", auditv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_HIGH)}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })
	t.Setenv("TEST_GENIE_ARCHITECTURE_GATE", "")

	got := Run(context.Background(), testArchitectureProvider(), "demo", "")
	if got.Success {
		t.Fatal("expected high-confidence blocker to fail the architecture phase")
	}
	if got.FailureClass != shared.FailureClassTestFailure {
		t.Fatalf("FailureClass = %q, want test_failure", got.FailureClass)
	}
	if got.Summary.AuthorityConfidence != "high" || got.Summary.GateMode != "high-confidence" || got.Summary.GatedBlockers != 1 {
		t.Fatalf("summary gate fields = %+v", got.Summary)
	}
	if len(got.Summary.Categories) != 1 || got.Summary.Categories[0].Key != "placement_legibility" {
		t.Fatalf("summary categories = %+v", got.Summary.Categories)
	}
	if joined := strings.Join(observationStrings(got.Observations), "\n"); !strings.Contains(joined, "Architecture Score Matrix") || !strings.Contains(joined, "Placement legibility") {
		t.Fatalf("architecture matrix not rendered in observations:\n%s", joined)
	}
	if !hasObservationType(got.Observations, shared.ObservationError) {
		t.Fatalf("deterministic blocker should render an error observation: %+v", got.Observations)
	}
	if !strings.Contains(got.Error.Error(), "TEST_GENIE_ARCHITECTURE_GATE=high-confidence") {
		t.Fatalf("gate error = %v", got.Error)
	}
}

func hasObservationType(observations []shared.Observation, want shared.ObservationType) bool {
	for _, observation := range observations {
		if observation.GetType() == want {
			return true
		}
	}
	return false
}

func observationStrings(observations []shared.Observation) []string {
	out := make([]string, 0, len(observations))
	for _, observation := range observations {
		out = append(out, observation.GetMessage())
	}
	return out
}

func TestArchitectureGateKeepsLowConfidenceBlockersAdvisoryByDefault(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: testArchitectureResponse(t, "SEVERITY_BLOCKER", auditv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_LOW)}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })
	t.Setenv("TEST_GENIE_ARCHITECTURE_GATE", "")

	got := Run(context.Background(), testArchitectureProvider(), "demo", "")
	if !got.Success {
		t.Fatalf("low-confidence blocker should remain advisory by default: %v", got.Error)
	}
	if got.Summary.AuthorityConfidence != "low" || got.Summary.GatedBlockers != 1 {
		t.Fatalf("summary gate fields = %+v", got.Summary)
	}
}

func TestArchitectureGateAllFailsLowConfidenceBlockers(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: testArchitectureResponse(t, "SEVERITY_BLOCKER", auditv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_LOW)}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })
	t.Setenv("TEST_GENIE_ARCHITECTURE_GATE", "all")

	got := Run(context.Background(), testArchitectureProvider(), "demo", "")
	if got.Success {
		t.Fatal("TEST_GENIE_ARCHITECTURE_GATE=all should fail low-confidence blockers")
	}
	if got.Summary.GateMode != "all" {
		t.Fatalf("gate mode = %q, want all", got.Summary.GateMode)
	}
}

func TestArchitectureGateOffKeepsHighConfidenceBlockersAdvisory(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: testArchitectureResponse(t, "SEVERITY_BLOCKER", auditv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_HIGH)}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })
	t.Setenv("TEST_GENIE_ARCHITECTURE_GATE", "off")

	got := Run(context.Background(), testArchitectureProvider(), "demo", "")
	if !got.Success {
		t.Fatalf("TEST_GENIE_ARCHITECTURE_GATE=off should keep blockers advisory: %v", got.Error)
	}
	if got.Summary.GateMode != "off" {
		t.Fatalf("gate mode = %q, want off", got.Summary.GateMode)
	}
}

func TestArchitectureGateKeepsIntentFindingsAdvisoryByDefault(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: testArchitectureResponseWithTypeAndClass(t, "intent.req_unowned_domain", "SEVERITY_BLOCKER", auditv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_HIGH, cartosharedv1.FindingClass_FINDING_CLASS_DETERMINISTIC)}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })
	t.Setenv("TEST_GENIE_ARCHITECTURE_GATE", "")
	t.Setenv("INTENT_ALIGNMENT_GATE", "")

	got := Run(context.Background(), testArchitectureProvider(), "demo", "")
	if !got.Success {
		t.Fatalf("default intent gate should keep deterministic intent findings advisory: %v", got.Error)
	}
	if got.Summary.GatedBlockers != 0 {
		t.Fatalf("gated blockers = %d, want 0", got.Summary.GatedBlockers)
	}
}

func TestArchitectureGateStrictIntentFindingsGate(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: testArchitectureResponseWithTypeAndClass(t, "intent.req_unowned_domain", "SEVERITY_BLOCKER", auditv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_HIGH, cartosharedv1.FindingClass_FINDING_CLASS_DETERMINISTIC)}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })
	t.Setenv("TEST_GENIE_ARCHITECTURE_GATE", "")
	t.Setenv("INTENT_ALIGNMENT_GATE", "strict")

	got := Run(context.Background(), testArchitectureProvider(), "demo", "")
	if got.Success {
		t.Fatal("strict intent gate should fail deterministic intent blockers")
	}
	if got.Summary.GatedBlockers != 1 {
		t.Fatalf("gated blockers = %d, want 1", got.Summary.GatedBlockers)
	}
}
