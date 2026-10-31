package eligibility

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"

	"test-genie/internal/orchestrator/workspace"
)

func assessmentWith(findings ...*commonv1.AssessmentFinding) *commonv1.MaturityAssessment {
	return &commonv1.MaturityAssessment{Findings: findings}
}

func TestDecideFromAssessment_NoFindings_Routed(t *testing.T) {
	e := decideFromAssessment(assessmentWith())
	if !e.Routed {
		t.Fatalf("expected routed for a clean assessment; got %+v", e)
	}
	if len(e.BlockingFindings) != 0 {
		t.Fatalf("expected no blocking findings; got %+v", e.BlockingFindings)
	}
}

func TestDecideFromAssessment_RoutedSeamsUnwired_Refused(t *testing.T) {
	e := decideFromAssessment(assessmentWith(&commonv1.AssessmentFinding{
		Code:     CodeRoutedSeamsUnwired,
		Severity: "SEVERITY_ERROR",
		Message:  "missing TestModeMiddleware",
		Location: "api/main.go",
	}))
	if e.Routed {
		t.Fatalf("expected refusal when ROUTED_SEAMS_UNWIRED present")
	}
	if e.Unverified {
		t.Fatalf("ROUTED_SEAMS_UNWIRED is a Go scenario; Unverified must stay false")
	}
	if len(e.BlockingFindings) != 1 || e.BlockingFindings[0].Code != CodeRoutedSeamsUnwired {
		t.Fatalf("expected a single ROUTED_SEAMS_UNWIRED blocking finding; got %+v", e.BlockingFindings)
	}
	if e.BlockingFindings[0].Location != "api/main.go" {
		t.Fatalf("expected location carried through; got %q", e.BlockingFindings[0].Location)
	}
}

func TestDecideFromAssessment_Unverified_Refused(t *testing.T) {
	e := decideFromAssessment(assessmentWith(&commonv1.AssessmentFinding{
		Code:     CodeStorageIsolationUnverified,
		Severity: "SEVERITY_WARNING",
		Message:  "non-Go API",
	}))
	if e.Routed {
		t.Fatalf("expected refusal when STORAGE_ISOLATION_UNVERIFIED present")
	}
	if !e.Unverified {
		t.Fatalf("expected Unverified=true for the non-Go fail-safe")
	}
}

func TestDecideFromAssessment_UnrelatedFinding_StaysRouted(t *testing.T) {
	// A non-isolation finding (e.g. a schema-layout issue) must not block the
	// routed path — routing eligibility is narrowly about isolation.
	e := decideFromAssessment(assessmentWith(&commonv1.AssessmentFinding{
		Code:     "SCHEMA_CENTRALIZED",
		Severity: "SEVERITY_ERROR",
	}))
	if !e.Routed {
		t.Fatalf("expected routed (unrelated finding); got %+v", e)
	}
}

// stubValidationClient implements StorageValidationClient for the Check tests.
type stubValidationClient struct {
	resp  *scenariovalidationv1.ValidateScenarioResponse
	err   error
	calls int
}

func (s *stubValidationClient) ValidateScenario(_ context.Context, _ *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.resp), nil
}

func withStubClient(t *testing.T, stub *stubValidationClient) {
	t.Helper()
	origResolve := ResolveStorageManagerURL
	origClient := NewStorageValidationClient
	t.Cleanup(func() {
		ResolveStorageManagerURL = origResolve
		NewStorageValidationClient = origClient
	})
	ResolveStorageManagerURL = func(context.Context) (string, error) { return "http://stub", nil }
	NewStorageValidationClient = func(time.Duration, string) StorageValidationClient { return stub }
}

func TestChecker_QueriesStorageManagerAndCaches(t *testing.T) {
	stub := &stubValidationClient{resp: &scenariovalidationv1.ValidateScenarioResponse{
		Assessment: assessmentWith(&commonv1.AssessmentFinding{Code: CodeRoutedSeamsUnwired, Severity: "SEVERITY_ERROR"}),
	}}
	withStubClient(t, stub)

	c := NewChecker()
	elig, err := c.Check(context.Background(), "demo", workspace.Mapping{PhysicalScenarioDir: "/x"})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if elig.Routed {
		t.Fatalf("expected not routed")
	}
	// Second call is served from cache (no new RPC).
	if _, err := c.Check(context.Background(), "demo", workspace.Mapping{}); err != nil {
		t.Fatalf("Check (cached): %v", err)
	}
	if stub.calls != 1 {
		t.Fatalf("expected 1 RPC (second served from cache); got %d", stub.calls)
	}
	// Invalidate forces a re-fetch.
	c.Invalidate("demo")
	if _, err := c.Check(context.Background(), "demo", workspace.Mapping{}); err != nil {
		t.Fatalf("Check (post-invalidate): %v", err)
	}
	if stub.calls != 2 {
		t.Fatalf("expected re-fetch after Invalidate; got %d calls", stub.calls)
	}
}

func TestChecker_RPCError_ReturnsError(t *testing.T) {
	stub := &stubValidationClient{err: errors.New("boom")}
	withStubClient(t, stub)

	c := NewChecker()
	if _, err := c.Check(context.Background(), "demo", workspace.Mapping{}); err == nil {
		t.Fatalf("expected error when storage-manager RPC fails")
	}
}
