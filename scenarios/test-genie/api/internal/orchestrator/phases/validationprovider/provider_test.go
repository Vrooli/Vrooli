package validationprovider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"test-genie/internal/orchestrator/browsergrant"
	"test-genie/internal/shared"

	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	assessmentpkg "github.com/vrooli/maturity-go/assessment"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/audit"
	cartosharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/shared"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	unitv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type fakeClient struct {
	resp *scenariovalidationv1.ValidateScenarioResponse
	err  error
}

func TestTranslatePreservesNativeEvidenceWithoutInferringExecution(t *testing.T) {
	native := &unitv1.ValidateScenarioResponse{
		Scenario:       "demo",
		EvidenceStages: &unitv1.EvidenceStages{Configured: "observed", Analyzed: "partial", Executed: "not_requested", Reviewed: "not_supplied", SourceRunId: "native-run"},
		TestQuality: &unitv1.TestQualityReport{SchemaVersion: "test-quality/v1", Results: []*unitv1.QualityCheckResult{{
			Target:      &unitv1.QualityTestTarget{File: "src/demo.test.ts", TestId: "suite > skipped test"},
			Status:      unitv1.QualityCheckStatus_QUALITY_CHECK_STATUS_UNKNOWN,
			Reason:      unitv1.QualityReason_QUALITY_REASON_SKIPPED,
			Severity:    unitv1.QualitySeverity_QUALITY_SEVERITY_ERROR,
			Enforcement: unitv1.QualityEnforcement_QUALITY_ENFORCEMENT_ADVISORY,
		}}},
	}
	detail, err := anypb.New(native)
	if err != nil {
		t.Fatal(err)
	}
	response := &scenariovalidationv1.ValidateScenarioResponse{Scenario: "demo", Status: scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED, Assessment: testAssessment(""), NativeDetail: detail}
	result := translate(testProvider(false), "demo", response)
	if !result.Success {
		t.Fatalf("advisory evidence became execution failure: %v", result.Error)
	}
	got := &unitv1.ValidateScenarioResponse{}
	if result.NativeDetail == nil {
		t.Fatal("native evidence dropped")
	}
	if err := result.NativeDetail.UnmarshalTo(got); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(got, native) {
		t.Fatalf("native evidence changed: %v", got)
	}
	response.NativeDetail = nil
	if old := translate(testProvider(false), "demo", response); old.NativeDetail != nil {
		t.Fatal("historical absence fabricated evidence")
	}
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
	resp    *scenariovalidationv1.ValidateScenarioResponse
	got     *scenariovalidationv1.ValidateScenarioRequest
	headers http.Header
}

type capturingTargetClient struct {
	resp *scenariovalidationv1.ValidateTargetResponse
	got  *scenariovalidationv1.ValidateTargetRequest
}

func (c *capturingTargetClient) ValidateTarget(_ context.Context, req *connect.Request[scenariovalidationv1.ValidateTargetRequest]) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error) {
	c.got = req.Msg
	return connect.NewResponse(c.resp), nil
}

func (*capturingTargetClient) DescribeProvider(context.Context, *connect.Request[scenariovalidationv1.DescribeProviderRequest]) (*connect.Response[scenariovalidationv1.DescribeProviderResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.DescribeProviderResponse{Capabilities: &scenariovalidationv1.ProviderCapabilities{SupportsRetainedEvidence: true}}), nil
}

type capturingDurableClient struct {
	start   *scenariovalidationv1.StartValidationRunRequest
	wait    *scenariovalidationv1.WaitValidationRunRequest
	abort   *scenariovalidationv1.AbortValidationRunRequest
	run     *scenariovalidationv1.ValidationRun
	waitErr error
}

func (c *capturingDurableClient) StartValidationRun(_ context.Context, req *connect.Request[scenariovalidationv1.StartValidationRunRequest]) (*connect.Response[scenariovalidationv1.StartValidationRunResponse], error) {
	c.start = req.Msg
	return connect.NewResponse(&scenariovalidationv1.StartValidationRunResponse{Run: c.run}), nil
}

func (c *capturingDurableClient) WaitValidationRun(_ context.Context, req *connect.Request[scenariovalidationv1.WaitValidationRunRequest]) (*connect.Response[scenariovalidationv1.WaitValidationRunResponse], error) {
	c.wait = req.Msg
	if c.waitErr != nil {
		return nil, c.waitErr
	}
	return connect.NewResponse(&scenariovalidationv1.WaitValidationRunResponse{Run: c.run}), nil
}

func (c *capturingDurableClient) AbortValidationRun(_ context.Context, req *connect.Request[scenariovalidationv1.AbortValidationRunRequest]) (*connect.Response[scenariovalidationv1.AbortValidationRunResponse], error) {
	c.abort = req.Msg
	return connect.NewResponse(&scenariovalidationv1.AbortValidationRunResponse{Run: c.run}), nil
}

func (c *capturingClient) ValidateScenario(_ context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	c.got = req.Msg
	c.headers = req.Header().Clone()
	return connect.NewResponse(c.resp), nil
}

func (*capturingClient) DescribeProvider(context.Context, *connect.Request[scenariovalidationv1.DescribeProviderRequest]) (*connect.Response[scenariovalidationv1.DescribeProviderResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.DescribeProviderResponse{Capabilities: &scenariovalidationv1.ProviderCapabilities{SupportsRetainedEvidence: true}}), nil
}

type unsupportedEvidenceClient struct{ capturingClient }

func (*unsupportedEvidenceClient) DescribeProvider(context.Context, *connect.Request[scenariovalidationv1.DescribeProviderRequest]) (*connect.Response[scenariovalidationv1.DescribeProviderResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.DescribeProviderResponse{Capabilities: &scenariovalidationv1.ProviderCapabilities{}}), nil
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
	if cap.got.GetPhase() != "proto" {
		t.Fatalf("phase = %q, want proto", cap.got.GetPhase())
	}
}

func TestRunSendsOnlyExplicitLoopbackBASOverride(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	cap := &capturingClient{resp: &scenariovalidationv1.ValidateScenarioResponse{
		Scenario: "ui-health", Status: scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED, Assessment: testAssessment(""),
	}}
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://ui-health", nil }
	NewClient = func(time.Duration, string) Client { return cap }
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })

	provider := testProvider(false)
	provider.Phase = "experience"
	provider.ProviderScenario = "ui-health"
	provider.BASUnavailableEndpoint = testBASUnavailableEndpoint
	Run(context.Background(), provider, "fixture", "/fixture")
	if got := cap.headers.Get(testBASUnavailableHeader); got != testBASUnavailableEndpoint {
		t.Fatalf("test-only BAS header = %q, want %q", got, testBASUnavailableEndpoint)
	}

	provider.BASUnavailableEndpoint = "http://localhost:17116"
	result := Run(context.Background(), provider, "fixture", "/fixture")
	if result == nil || result.Success {
		t.Fatalf("non-loopback override must fail closed, got %+v", result)
	}
}

func TestAudioToolsSoakForwardsOnlyRequestScopedVerifiedBearerHeaders(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	cap := &capturingClient{resp: &scenariovalidationv1.ValidateScenarioResponse{
		Scenario: "audio-tools", Status: scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED, Assessment: testAssessment(""),
	}}
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://audio-tools", nil }
	NewClient = func(time.Duration, string) Client { return cap }
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })

	var initiator context.Context
	handler := provenance.Middleware(provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: token}}, nil
	}))(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { initiator = r.Context() }))
	req := httptest.NewRequest(http.MethodPost, "/start", nil)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "agent-token-from-request")
	handler.ServeHTTP(httptest.NewRecorder(), req)
	initiator = browsergrant.AttachRequestReceipt(initiator, []string{"browser-grant-from-request"})
	runCtx := provenance.CarryVerifiedIdentity(context.Background(), initiator)
	runCtx = browsergrant.CarryVerifiedReceipt(runCtx, initiator)
	runCtx = browsergrant.AudioToolsSoakContext(runCtx, "lease-receipt-for-this-run")

	provider := testProvider(false)
	provider.ProviderScenario = "audio-tools"
	provider.Phase = "soak"
	Run(runCtx, provider, "audio-tools", "/audio-tools")
	if cap.headers.Get(cliutil.HeaderAgentIdentityToken) != "agent-token-from-request" ||
		cap.headers.Get(browsergrant.HeaderBrowserGrantReceipt) != "browser-grant-from-request" ||
		cap.headers.Get("X-Test-Genie-Lease-Receipt") != "lease-receipt-for-this-run" || len(cap.headers) != 3 {
		t.Fatalf("audio-tools soak bearer headers = %#v", cap.headers)
	}

	cap.headers = nil
	Run(context.Background(), provider, "audio-tools", "/audio-tools")
	if len(cap.headers) != 0 {
		t.Fatalf("unverified request forwarded bearer headers: %#v", cap.headers)
	}
}

func TestRunSendsCapabilitySubset(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	cap := &capturingClient{resp: &scenariovalidationv1.ValidateScenarioResponse{
		Scenario: "demo", Status: scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED, Assessment: testAssessment(""),
	}}
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client { return cap }
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })

	provider := testProvider(false)
	provider.CapabilitySubset = []string{"architecture", "contracts"}
	provider.RetainedEvidenceSets = []*scenariovalidationv1.RetainedEvidenceSet{{ProducerReceiptId: "receipt-1", Producer: "evidence-completeness", Target: "demo", RunId: "run-1", CandidateIdentity: "sha256:candidate", CatalogDigest: "sha256:catalog"}}
	Run(context.Background(), provider, "demo", "/tmp/demo")
	if got := cap.got.GetCapabilitySubset(); len(got) != 2 || got[0] != "architecture" || got[1] != "contracts" {
		t.Fatalf("capability subset = %v", got)
	}
	if len(cap.got.GetRetainedEvidenceSets()) != 1 || cap.got.GetRetainedEvidenceSets()[0].GetProducerReceiptId() != "receipt-1" {
		t.Fatalf("retained evidence was not forwarded: %v", cap.got.GetRetainedEvidenceSets())
	}
}

func TestBoundEvidenceRefusesProviderWithoutExplicitCapability(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	cap := &unsupportedEvidenceClient{capturingClient: capturingClient{resp: &scenariovalidationv1.ValidateScenarioResponse{Scenario: "demo", Status: scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED, Assessment: testAssessment("")}}}
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client { return cap }
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })
	provider := testProvider(false)
	provider.RetainedEvidenceSets = []*scenariovalidationv1.RetainedEvidenceSet{{ProducerReceiptId: "receipt", Producer: "owner", Target: "demo", RunId: "run", CandidateIdentity: "candidate", CatalogDigest: "catalog"}}
	result := Run(context.Background(), provider, "demo", "/tmp/demo")
	if result.Success || cap.got != nil {
		t.Fatalf("unsupported provider received bound evidence: result=%+v request=%+v", result, cap.got)
	}
}

func TestRunTargetSendsContractExcludes(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewTargetClient
	cap := &capturingTargetClient{resp: &scenariovalidationv1.ValidateTargetResponse{
		Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
		Assessment: testAssessment(""),
	}}
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewTargetClient = func(time.Duration, string) TargetClient { return cap }
	t.Cleanup(func() { ResolveBaseURL, NewTargetClient = prevResolve, prevClient })

	provider := testProvider(false)
	provider.Exclude = []string{"internal/tools/*", "internal/safeguards/*"}
	provider.RetainedEvidenceSets = []*scenariovalidationv1.RetainedEvidenceSet{{ProducerReceiptId: "receipt-2", Producer: "evidence-completeness", Target: "demo", RunId: "run-2", CandidateIdentity: "sha256:candidate", CatalogDigest: "sha256:catalog"}}
	result := RunTarget(context.Background(), provider, &commonv1.ValidationTarget{
		Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_CONTROL_PLANE,
		Id:   "internal",
		Root: "internal",
	}, "/repo/internal")
	if !result.Success {
		t.Fatalf("RunTarget() failed: %+v", result)
	}
	if got := cap.got.GetExclude(); len(got) != 2 || got[0] != "internal/tools/*" || got[1] != "internal/safeguards/*" {
		t.Fatalf("exclude = %v, want contract excludes", got)
	}
	if len(cap.got.GetRetainedEvidenceSets()) != 1 || cap.got.GetRetainedEvidenceSets()[0].GetProducerReceiptId() != "receipt-2" {
		t.Fatalf("retained evidence was not forwarded: %v", cap.got.GetRetainedEvidenceSets())
	}
}

func TestRunDurableStartsOnceAndWaitsForSharedTerminalResponse(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewDurableClient
	terminal := &scenariovalidationv1.ValidateScenarioResponse{Scenario: "demo", Status: scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED, Assessment: testAssessment("")}
	cap := &capturingDurableClient{run: &scenariovalidationv1.ValidationRun{RunId: "provider-run-1", TerminalResult: terminal}}
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewDurableClient = func(time.Duration, string) DurableClient { return cap }
	t.Cleanup(func() { ResolveBaseURL, NewDurableClient = prevResolve, prevClient })

	provider := testProvider(false)
	provider.CapabilitySubset = []string{"contracts"}
	var startedRef RunReference
	provider.OnStarted = func(ref RunReference) { startedRef = ref }
	result := RunDurable(context.Background(), provider, "demo", "/tmp/demo", "parent-run-1")
	if !result.Success {
		t.Fatalf("durable result = %+v", result)
	}
	if cap.start == nil || cap.wait == nil {
		t.Fatalf("start=%+v wait=%+v, want one each", cap.start, cap.wait)
	}
	if cap.start.GetIdempotencyKey() != "parent-run-1:proto" || cap.start.GetParentRunId() != "parent-run-1" {
		t.Fatalf("start identity = %+v", cap.start)
	}
	if cap.start.GetPhase() != "proto" {
		t.Fatalf("start phase = %q, want proto", cap.start.GetPhase())
	}
	// The generic durable contract carries target and run identity only. Provider
	// safety authorization (including routed isolation) remains provider-owned.
	if cap.start.GetScenario() != "demo" || cap.start.GetPath() != "/tmp/demo" {
		t.Fatalf("start target = %+v", cap.start)
	}
	if got := cap.start.GetCapabilitySubset(); len(got) != 1 || got[0] != "contracts" {
		t.Fatalf("start capability subset = %v", got)
	}
	if cap.wait.GetRunId() != "provider-run-1" {
		t.Fatalf("wait run id = %q", cap.wait.GetRunId())
	}
	if result.Summary.ProviderRunID != "provider-run-1" {
		t.Fatalf("provider run id = %q", result.Summary.ProviderRunID)
	}
	if result.Summary.ProviderParentRunID != "parent-run-1" || result.Summary.ProviderRunState != "unspecified" {
		t.Fatalf("provider terminal reference = %+v", result.Summary)
	}
	if startedRef.RunID != "provider-run-1" || startedRef.ParentRunID != "parent-run-1" {
		t.Fatalf("started child reference = %+v", startedRef)
	}
}

func TestRunDurablePropagatesExplicitParentCancellation(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewDurableClient
	cap := &capturingDurableClient{run: &scenariovalidationv1.ValidationRun{RunId: "provider-run-1"}, waitErr: context.Canceled}
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewDurableClient = func(time.Duration, string) DurableClient { return cap }
	t.Cleanup(func() { ResolveBaseURL, NewDurableClient = prevResolve, prevClient })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = RunDurable(ctx, testProvider(false), "demo", "/tmp/demo", "parent-run-1")
	if cap.abort == nil || cap.abort.GetRunId() != "provider-run-1" {
		t.Fatalf("abort = %+v, want durable child cancellation", cap.abort)
	}
}

func TestRunDurablePreservesChildReferenceWhenWaitFails(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewDurableClient
	cap := &capturingDurableClient{run: &scenariovalidationv1.ValidationRun{RunId: "provider-run-1"}, waitErr: context.DeadlineExceeded}
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewDurableClient = func(time.Duration, string) DurableClient { return cap }
	t.Cleanup(func() { ResolveBaseURL, NewDurableClient = prevResolve, prevClient })

	result := RunDurable(context.Background(), testProvider(false), "demo", "/tmp/demo", "parent-run-1")
	if result.Success {
		t.Fatal("expected wait failure")
	}
	if result.Summary.ProviderRunID != "provider-run-1" || result.Summary.ProviderParentRunID != "parent-run-1" {
		t.Fatalf("durable child reference = %+v", result.Summary)
	}
	if !strings.Contains(result.Remediation, "provider-run-1") {
		t.Fatalf("remediation = %q", result.Remediation)
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
		Scenario:           "demo",
		Provider:           "proto-health",
		Phase:              "proto",
		Version:            "1",
		Local:              &commonv1.LocalMaturityAssessment{CurrentLevel: "L1", NextLevel: "L2", Clean: true, UnknownCount: 1},
		FindingsBySeverity: map[string]int32{},
	}
	if severity != "" {
		a.FindingsBySeverity[severity] = 1
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
	a.Presentation = assessmentpkg.BuildPhasePresentation(a)
	return a
}

func testCapabilityAssessment() *commonv1.MaturityAssessment {
	a := testAssessment("SEVERITY_WARNING")
	a.Capabilities = []*commonv1.CapabilityMaturityAssessment{
		{
			Id:                   "interop",
			Label:                "Interop",
			CurrentLevel:         "L2",
			CurrentSummary:       "Embeds correctly across deployment contexts.",
			Clean:                true,
			PriorityRank:         1,
			BlockingFindingCodes: nil,
			Levels: []*commonv1.LocalMaturityLevel{
				{Id: "L0"},
				{Id: "L1"},
				{Id: "L2"},
			},
		},
		{
			Id:                   "pwa_native_readiness",
			Label:                "PWA Native Readiness",
			CurrentLevel:         "L0",
			NextLevel:            "L1",
			CurrentSummary:       "Install surface is absent.",
			NextUnlock:           "Offline-safe launch and app-shell fallback.",
			PriorityRank:         2,
			PriorityReason:       "lowest current level with required/blocking findings WARNING capability_gap",
			BlockingFindingCodes: []string{"proto.gen_out_of_sync"},
			FindingsBySeverity:   map[string]int32{"SEVERITY_WARNING": 1},
			Levels: []*commonv1.LocalMaturityLevel{
				{Id: "L0"},
				{Id: "L1"},
			},
		},
	}
	a.HighestPriorityCapability = &commonv1.PriorityFocus{
		CapabilityId:    "pwa_native_readiness",
		CapabilityLabel: "PWA Native Readiness",
		CurrentLevel:    "L0",
		NextLevel:       "L1",
		Reason:          "lowest current level with required/blocking findings WARNING capability_gap",
	}
	a.Findings[0].Maturity.CapabilityId = "pwa_native_readiness"
	a.Findings[0].Maturity.LocalLevel = "L1"
	a.Presentation = assessmentpkg.BuildPhasePresentation(a)
	return a
}

func TestMergeCapabilityResultsRebuildsOnePhaseStanding(t *testing.T) {
	full := testCapabilityAssessment()
	full.Findings = append(full.Findings, &commonv1.AssessmentFinding{
		Code:     "proto.interop_warning",
		Severity: "SEVERITY_WARNING",
		Title:    "Interop warning",
		Maturity: &commonv1.FindingMaturity{CapabilityId: "interop"},
	})
	full.Presentation = assessmentpkg.BuildPhasePresentation(full)
	left := FilterCapabilityResult(&Result{RunResult: shared.RunResult[Summary]{Success: true, Summary: Summary{Scenario: "demo"}}, Assessment: full}, "interop")
	right := FilterCapabilityResult(&Result{RunResult: shared.RunResult[Summary]{Success: true, Summary: Summary{Scenario: "demo"}}, Assessment: full}, "pwa_native_readiness")
	merged := MergeCapabilityResults(left, right)
	if merged == nil || merged.Assessment == nil {
		t.Fatal("merged result did not retain assessment")
	}
	if len(merged.Assessment.GetCapabilities()) != 2 || len(merged.Assessment.GetFindings()) != 2 {
		t.Fatalf("merged assessment lost capability projections: capabilities=%d findings=%d", len(merged.Assessment.GetCapabilities()), len(merged.Assessment.GetFindings()))
	}
	if merged.Summary.Status != "passed" || !merged.Success {
		t.Fatalf("merged standing should derive a passing status from warnings: %+v", merged.Summary)
	}
	if merged.Assessment.GetPresentation() == nil || len(merged.Assessment.GetPresentation().GetCapabilities()) != 2 {
		t.Fatalf("merged presentation was not rebuilt: %+v", merged.Assessment.GetPresentation())
	}
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
	arch.Presentation = assessmentpkg.BuildPhasePresentation(arch)
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
	assessment := testAssessment("SEVERITY_ERROR")
	assessment.Findings[0].Evidence = []*commonv1.AssessmentEvidence{{Kind: "command.output", Summary: "--- FAIL: TestAPI", Locator: "go test ./..."}}
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: &scenariovalidationv1.ValidateScenarioResponse{
			Scenario:   "demo",
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED,
			Assessment: assessment,
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
	if evidence := got.Findings[0].GetEvidence(); len(evidence) != 1 || evidence[0].GetSummary() != "--- FAIL: TestAPI" || evidence[0].GetLocator() != "go test ./..." {
		t.Fatalf("provider evidence did not survive normalization: %+v", evidence)
	}
	if got.Summary.LocalCurrentLevel != "L1" || got.Summary.LocalNextLevel != "L2" {
		t.Fatalf("summary local = %q/%q, want L1/L2", got.Summary.LocalCurrentLevel, got.Summary.LocalNextLevel)
	}
	if !got.Summary.LocalClean || got.Summary.LocalUnknownCount != 1 {
		t.Fatalf("summary clean/unknown = %v/%d, want true/1", got.Summary.LocalClean, got.Summary.LocalUnknownCount)
	}
}

func TestUIHealthDirectAndDelegatedPresentationsMatch(t *testing.T) {
	provider := Provider{
		Phase:            "experience",
		ProviderScenario: "ui-health",
		FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_UI,
		Emoji:            "U",
		Timeout:          time.Second,
		IncludeExecution: true,
	}
	// This is the direct ScenarioValidationService response. The assertion below
	// compares its provider-owned presentation with the object surfaced by Test
	// Genie after delegation; transport must not derive or reshape it.
	directAssessment := testCapabilityAssessment()
	directAssessment.Provider = "ui-health"
	directAssessment.Phase = "experience"
	directAssessment.Presentation = assessmentpkg.BuildPhasePresentation(directAssessment)
	providerPresentation := directAssessment.GetPresentation()
	response := &scenariovalidationv1.ValidateScenarioResponse{
		Scenario:   "demo",
		Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
		Assessment: directAssessment,
	}
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client { return fakeClient{resp: response} }
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })

	got := Run(context.Background(), provider, "demo", "")
	if got.Error != nil || got.Presentation == nil {
		t.Fatalf("delegated run did not retain presentation: result=%+v", got)
	}
	if !proto.Equal(got.Presentation, providerPresentation) {
		t.Fatalf("Test Genie changed experience's direct provider presentation:\n got=%+v\nwant=%+v", got.Presentation, providerPresentation)
	}
}

func TestRunDegradedStatusStaysVisibleWithoutBeingRewrittenAsPassed(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: &scenariovalidationv1.ValidateScenarioResponse{
			Scenario:   "demo",
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED,
			Assessment: testAssessment("SEVERITY_INFO"),
		}}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })

	got := Run(context.Background(), testProvider(false), "demo", "")
	if !got.Success {
		t.Fatalf("degraded provider truth may be non-gating, got %v", got.Error)
	}
	if got.Summary.Status != "degraded" {
		t.Fatalf("summary status = %q, want degraded rather than passed", got.Summary.Status)
	}
}

func TestRunNotEvaluatedNeverPasses(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: &scenariovalidationv1.ValidateScenarioResponse{
			Scenario:   "demo",
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_NOT_EVALUATED,
			Assessment: testAssessment(""),
		}}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })

	got := Run(context.Background(), testProvider(false), "demo", "")
	if got.Success {
		t.Fatal("not-evaluated provider response must never pass")
	}
	if got.Summary.Status != "not_evaluated" || got.FailureClass != shared.FailureClassMissingDependency {
		t.Fatalf("not-evaluated result lost its verdict: status=%q failure=%q", got.Summary.Status, got.FailureClass)
	}
	if got.Error == nil || !strings.Contains(got.Error.Error(), "did not evaluate") {
		t.Fatalf("missing actionable not-evaluated error: %v", got.Error)
	}
}

func TestTranslateVerdictContractAcrossProviderStatuses(t *testing.T) {
	// Keep the complete wire enum visible here. In particular, an enum value
	// introduced by a newer provider must not fall through as a pass.
	tests := []struct {
		name       string
		status     scenariovalidationv1.ValidationStatus
		wantStatus string
		wantPass   bool
		wantClass  shared.FailureClass
	}{
		{"passed", scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED, "passed", true, ""},
		{"failed", scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED, "failed", false, shared.FailureClassTestFailure},
		{"degraded", scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED, "degraded", true, ""},
		{"error", scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR, "error", false, shared.FailureClassSystem},
		{"skipped", scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_SKIPPED, "skipped", true, ""},
		{"not_evaluated", scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_NOT_EVALUATED, "not_evaluated", false, shared.FailureClassMissingDependency},
		{"unspecified", scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_UNSPECIFIED, "error", false, shared.FailureClassMaturityContract},
		{"unknown_future_status", scenariovalidationv1.ValidationStatus(99), "", false, shared.FailureClassMaturityContract},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := translate(testProvider(false), "demo", &scenariovalidationv1.ValidateScenarioResponse{
				Scenario: "demo", Status: tt.status, Assessment: testAssessment(""),
			})
			if got.Success != tt.wantPass {
				t.Fatalf("success=%v, want %v (status=%q error=%v)", got.Success, tt.wantPass, got.Summary.Status, got.Error)
			}
			if tt.wantStatus != "" && got.Summary.Status != tt.wantStatus {
				t.Fatalf("status=%q, want %q", got.Summary.Status, tt.wantStatus)
			}
			if got.FailureClass != tt.wantClass {
				t.Fatalf("failure class=%q, want %q", got.FailureClass, tt.wantClass)
			}
		})
	}
}

func TestMergeCapabilityResultsPreservesNotEvaluated(t *testing.T) {
	assessment := testCapabilityAssessment()
	base := &Result{RunResult: shared.RunResult[Summary]{Success: false, Summary: Summary{Scenario: "demo", Status: "not_evaluated"}}, Assessment: assessment}
	passing := &Result{RunResult: shared.RunResult[Summary]{Success: true, Summary: Summary{Scenario: "demo", Status: "passed"}}, Assessment: assessment}
	projected := FilterCapabilityResult(base, "interop")
	merged := MergeCapabilityResults(projected, passing)
	if merged.Success || merged.Summary.Status != "not_evaluated" {
		t.Fatalf("cached not-evaluated verdict was erased by merge: success=%v status=%q", merged.Success, merged.Summary.Status)
	}
}

func TestMergeCapabilityResultsDoesNotHideFreshNotEvaluatedBehindCachedPass(t *testing.T) {
	assessment := testCapabilityAssessment()
	cached := &Result{RunResult: shared.RunResult[Summary]{Success: true, Summary: Summary{Scenario: "demo", Status: "passed"}}, Assessment: assessment}
	fresh := &Result{RunResult: shared.RunResult[Summary]{Success: false, Summary: Summary{Scenario: "demo", Status: "not_evaluated"}}, Assessment: assessment}
	merged := MergeCapabilityResults(cached, fresh)
	if merged.Success || merged.Summary.Status != "not_evaluated" {
		t.Fatalf("fresh not-evaluated result was hidden by a cached pass: success=%v status=%q", merged.Success, merged.Summary.Status)
	}
}

func TestMergeCapabilityResultsFailsClosedOnUnknownCachedOrFreshLabel(t *testing.T) {
	assessment := &commonv1.MaturityAssessment{Scenario: "demo"}
	for _, tc := range []struct{ cached, fresh string }{
		{"future_status", "passed"}, {"passed", "future_status"},
	} {
		cached := &Result{RunResult: shared.RunResult[Summary]{Success: true, Summary: Summary{Scenario: "demo", Status: tc.cached}}, Assessment: assessment}
		fresh := &Result{RunResult: shared.RunResult[Summary]{Success: true, Summary: Summary{Scenario: "demo", Status: tc.fresh}}, Assessment: assessment}
		got := MergeCapabilityResults(cached, fresh)
		if got.Success || got.Summary.Status != "unspecified" || got.FailureClass != shared.FailureClassMaturityContract {
			t.Fatalf("unknown cached/fresh label did not fail closed: cached=%q fresh=%q result=%+v", tc.cached, tc.fresh, got)
		}
	}
}

func TestFilterCapabilityResultFailsClosedOnUnknownCachedLabel(t *testing.T) {
	assessment := &commonv1.MaturityAssessment{Scenario: "demo", Capabilities: []*commonv1.CapabilityMaturityAssessment{{Id: "ux"}}}
	input := &Result{RunResult: shared.RunResult[Summary]{Success: true, Summary: Summary{Scenario: "demo", Status: "future_status"}}, Assessment: assessment}
	got := FilterCapabilityResult(input, "ux")
	if got.Success || got.FailureClass != shared.FailureClassMaturityContract || got.Error == nil || got.Summary.Status != "unspecified" {
		t.Fatalf("unknown cached label did not fail closed during projection: %+v", got)
	}
}

func TestFilterCapabilityResultFailsClosedOnUnknownCachedLabelWithoutAssessment(t *testing.T) {
	input := &Result{RunResult: shared.RunResult[Summary]{
		Success: true, Summary: Summary{Scenario: "demo", Status: "future_status"},
	}}
	got := FilterCapabilityResult(input, "ux")
	if got == nil || got.Success {
		t.Fatalf("unknown cached label without assessment passed: %+v", got)
	}
	if got.FailureClass != shared.FailureClassMaturityContract {
		t.Fatalf("FailureClass = %q, want maturity contract", got.FailureClass)
	}
}

func TestMergeCapabilityResultsVerdictContractMatrix(t *testing.T) {
	statuses := []string{"passed", "degraded", "failed", "error", "not_evaluated"}
	for _, cachedStatus := range statuses {
		for _, freshStatus := range statuses {
			t.Run(cachedStatus+"_then_"+freshStatus, func(t *testing.T) {
				cached := &Result{RunResult: shared.RunResult[Summary]{Success: statusIsSuccessful(cachedStatus), Summary: Summary{Scenario: "demo", Status: cachedStatus}}}
				fresh := &Result{RunResult: shared.RunResult[Summary]{Success: statusIsSuccessful(freshStatus), Summary: Summary{Scenario: "demo", Status: freshStatus}}}
				merged := MergeCapabilityResults(cached, fresh)
				if merged.Summary.Status == "not_evaluated" && merged.Success {
					t.Fatalf("merged not_evaluated verdict passed: %+v", merged)
				}
				if (cachedStatus == "failed" || freshStatus == "failed") && merged.Success {
					t.Fatalf("real defect was hidden by merge: %s + %s => %+v", cachedStatus, freshStatus, merged)
				}
				if (cachedStatus == "error" || freshStatus == "error") && merged.Success {
					t.Fatalf("provider error was hidden by merge: %s + %s => %+v", cachedStatus, freshStatus, merged)
				}
				if (cachedStatus == "not_evaluated" || freshStatus == "not_evaluated") && merged.Success {
					t.Fatalf("unevaluated result was hidden by merge: %s + %s => %+v", cachedStatus, freshStatus, merged)
				}
				if merged.Summary.Status == "error" && merged.FailureClass != shared.FailureClassSystem {
					t.Fatalf("provider error was mislabeled as a product defect: %+v", merged)
				}
				if merged.Summary.Status == "failed" && merged.FailureClass != shared.FailureClassTestFailure {
					t.Fatalf("real defect lost its test-failure classification: %+v", merged)
				}
				if merged.Summary.Status == "not_evaluated" && merged.FailureClass != shared.FailureClassMissingDependency {
					t.Fatalf("unevaluated result was mislabeled: %+v", merged)
				}
			})
		}
	}
}

func TestMergeCapabilityResultsObservedAssessmentFailureOutranksNotEvaluated(t *testing.T) {
	assessment := testAssessment("SEVERITY_ERROR")
	unevaluated := &Result{
		RunResult:  shared.RunResult[Summary]{Success: false, Summary: Summary{Scenario: "demo", Status: "not_evaluated"}},
		Assessment: assessment,
	}
	merged := MergeCapabilityResults(unevaluated, unevaluated)
	if merged.Success || merged.Summary.Status != "failed" {
		t.Fatalf("assessment error was hidden by not-evaluated status: success=%v status=%q error=%v", merged.Success, merged.Summary.Status, merged.Error)
	}
}

func TestMergeCapabilityResultsPreservesNotEvaluatedWithoutAssessment(t *testing.T) {
	cached := &Result{RunResult: shared.RunResult[Summary]{Success: false, Summary: Summary{Scenario: "demo", Status: "not_evaluated"}}}
	fresh := &Result{RunResult: shared.RunResult[Summary]{Success: true, Summary: Summary{Scenario: "demo", Status: "passed"}}}
	merged := MergeCapabilityResults(cached, fresh)
	if merged.Success || merged.Summary.Status != "not_evaluated" {
		t.Fatalf("assessment-free merge erased not-evaluated: success=%v status=%q", merged.Success, merged.Summary.Status)
	}
}

func TestRunSummarizesCapabilityMaturity(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: &scenariovalidationv1.ValidateScenarioResponse{
			Scenario:   "demo",
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED,
			Assessment: testCapabilityAssessment(),
		}}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })

	got := Run(context.Background(), testProvider(false), "demo", "")
	if !got.Success {
		t.Fatalf("capability assessment should pass provider transport, got %v", got.Error)
	}
	if len(got.Summary.Capabilities) != 2 {
		t.Fatalf("capabilities = %+v, want two summaries", got.Summary.Capabilities)
	}
	pwa := got.Summary.Capabilities[1]
	if pwa.ID != "pwa_native_readiness" || pwa.NextUnlock != "Offline-safe launch and app-shell fallback." || pwa.BlockingFindingCount != 1 {
		t.Fatalf("pwa summary = %+v", pwa)
	}
	if got.Summary.HighestPriorityCapability == nil || got.Summary.HighestPriorityCapability.CapabilityID != "pwa_native_readiness" {
		t.Fatalf("focus = %+v, want pwa_native_readiness", got.Summary.HighestPriorityCapability)
	}
	if !strings.Contains(got.Summary.String(), "focus=pwa_native_readiness->L1") {
		t.Fatalf("summary string missing focus: %q", got.Summary.String())
	}
	joined := strings.Join(observationStrings(got.Observations), "\n")
	if !strings.Contains(joined, "highest priority capability: PWA Native Readiness to L1") {
		t.Fatalf("observations missing priority focus:\n%s", joined)
	}
	if strings.Contains(joined, "Interop capability: current=L2 maximum maturity reached") {
		t.Fatalf("observations should omit clean non-focus capabilities:\n%s", joined)
	}
}

func TestRunOmitsCleanCapabilitySpam(t *testing.T) {
	assessment := testCapabilityAssessment()
	assessment.Findings = nil
	assessment.FindingsBySeverity = map[string]int32{}
	assessment.FindingsByGlobalImpact = map[string]int32{}
	assessment.Capabilities[0].Clean = true
	assessment.Capabilities[1].Clean = true
	assessment.Capabilities[1].FindingsBySeverity = map[string]int32{}
	assessment.Capabilities[1].BlockingFindingCodes = nil
	assessment.HighestPriorityCapability = nil
	assessment.Presentation = assessmentpkg.BuildPhasePresentation(assessment)

	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: &scenariovalidationv1.ValidateScenarioResponse{
			Scenario:   "demo",
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
			Assessment: assessment,
		}}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })

	got := Run(context.Background(), testProvider(false), "demo", "")
	if !got.Success {
		t.Fatalf("clean capability assessment should pass, got %v", got.Error)
	}
	joined := strings.Join(observationStrings(got.Observations), "\n")
	if !strings.Contains(joined, "all 2 capability assessment(s) clean") {
		t.Fatalf("observations missing compact clean capability line:\n%s", joined)
	}
	if strings.Contains(joined, "Interop capability:") || strings.Contains(joined, "PWA Native Readiness capability:") {
		t.Fatalf("observations should not list every clean capability:\n%s", joined)
	}
}

func TestRunMalformedCapabilityAssessmentIsMaturityContract(t *testing.T) {
	bad := testCapabilityAssessment()
	bad.Capabilities[0].CurrentLevel = "L9"
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: &scenariovalidationv1.ValidateScenarioResponse{
			Scenario:   "demo",
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
			Assessment: bad,
		}}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })

	got := Run(context.Background(), testProvider(false), "demo", "")
	if got.FailureClass != shared.FailureClassMaturityContract {
		t.Fatalf("FailureClass = %q, want maturity_contract (err=%v)", got.FailureClass, got.Error)
	}
	if got.Error == nil || !strings.Contains(got.Error.Error(), "current_level") {
		t.Fatalf("expected current_level contract error, got %v", got.Error)
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

func TestRunRequiredProviderUnavailableIsNotEvaluatedAndCannotPass(t *testing.T) {
	prevResolve := ResolveBaseURL
	ResolveBaseURL = func(context.Context, string) (string, error) { return "", errors.New("not running") }
	t.Cleanup(func() { ResolveBaseURL = prevResolve })

	got := Run(context.Background(), testProvider(false), "demo", "")
	if got.Success || got.Summary.Status != "not_evaluated" {
		t.Fatalf("required unreachable provider must be not-evaluated and fail closed: success=%v summary=%+v", got.Success, got.Summary)
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

func TestRunMissingPresentationUsesCompatibilityMode(t *testing.T) {
	assessment := testAssessment("SEVERITY_WARNING")
	assessment.Presentation = nil
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: &scenariovalidationv1.ValidateScenarioResponse{
			Scenario:   "demo",
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
			Assessment: assessment,
		}}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })

	got := Run(context.Background(), testProvider(false), "demo", "")
	if !got.Success || got.Error != nil {
		t.Fatalf("missing presentation must preserve the provider's real result: %+v", got)
	}
	if got.Presentation != nil {
		t.Fatalf("compatibility mode must not synthesize a provider-owned presentation: %+v", got.Presentation)
	}
	if joined := strings.Join(observationStrings(got.Observations), "\n"); !strings.Contains(joined, "presentation compatibility mode") {
		t.Fatalf("missing presentation must be visible to operators:\n%s", joined)
	}
}

func TestRunMalformedPresentationRemainsMaturityContract(t *testing.T) {
	assessment := testAssessment("SEVERITY_WARNING")
	assessment.Presentation.ContractVersion = "v0"
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: &scenariovalidationv1.ValidateScenarioResponse{
			Scenario:   "demo",
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
			Assessment: assessment,
		}}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })

	got := Run(context.Background(), testProvider(false), "demo", "")
	if got.FailureClass != shared.FailureClassMaturityContract || got.Error == nil || !strings.Contains(got.Error.Error(), "presentation") {
		t.Fatalf("malformed presentation must fail as a maturity contract: %+v", got)
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
