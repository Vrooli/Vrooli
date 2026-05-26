package eligibility

import (
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"

	internalelig "test-genie/internal/eligibility"
	"test-genie/internal/orchestrator/workspace"

	eligpb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/eligibility"
)

type fakeChecker struct {
	elig    internalelig.Eligibility
	err     error
	gotName string
	gotDir  string
}

func (f *fakeChecker) Check(_ context.Context, scenario string, mapping workspace.Mapping) (internalelig.Eligibility, error) {
	f.gotName = scenario
	f.gotDir = mapping.PhysicalScenarioDir
	return f.elig, f.err
}

func TestCheck_Routed(t *testing.T) {
	checker := &fakeChecker{elig: internalelig.Eligibility{Routed: true}}
	svc := NewService(checker, "/tmp/scenarios")

	resp, err := svc.Check(context.Background(), connect.NewRequest(&eligpb.CheckRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.GetRouted() {
		t.Fatalf("expected routed=true")
	}
	if len(resp.Msg.GetViolations()) != 0 {
		t.Fatalf("expected no violations on routed; got %d", len(resp.Msg.GetViolations()))
	}
	if len(resp.Msg.GetDisqualifyingReasons()) != 0 {
		t.Fatalf("expected no reasons on routed")
	}
	if checker.gotName != "demo" {
		t.Errorf("checker received scenario=%q; want demo", checker.gotName)
	}
	if !strings.HasSuffix(checker.gotDir, "/tmp/scenarios/demo") {
		t.Errorf("checker received mapping dir=%q; want suffix /tmp/scenarios/demo", checker.gotDir)
	}
}

func TestCheck_DisqualifiedByDriver(t *testing.T) {
	checker := &fakeChecker{elig: internalelig.Eligibility{
		Routed: false,
		Violations: []internalelig.ViolationExcerpt{
			{RuleID: internalelig.RuleRoutedDrivers, Severity: "high", FilePath: "db.go", LineNumber: 42, Title: "raw driver"},
		},
	}}
	svc := NewService(checker, "/tmp/scenarios")

	resp, err := svc.Check(context.Background(), connect.NewRequest(&eligpb.CheckRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetRouted() {
		t.Fatalf("expected routed=false")
	}
	if len(resp.Msg.GetViolations()) != 1 {
		t.Fatalf("expected 1 violation; got %d", len(resp.Msg.GetViolations()))
	}
	v := resp.Msg.GetViolations()[0]
	if v.GetRuleId() != internalelig.RuleRoutedDrivers || v.GetLine() != 42 || v.GetFile() != "db.go" {
		t.Errorf("violation mapping wrong: %+v", v)
	}
	reasons := resp.Msg.GetDisqualifyingReasons()
	if len(reasons) != 1 || !strings.Contains(reasons[0], "routed_database_drivers") {
		t.Errorf("expected reason mentioning routed_database_drivers; got %v", reasons)
	}
}

func TestCheck_DisqualifiedByHandleCapture(t *testing.T) {
	checker := &fakeChecker{elig: internalelig.Eligibility{
		Routed: false,
		Violations: []internalelig.ViolationExcerpt{
			{RuleID: internalelig.RuleRoutedHandleCapture, Severity: "medium"},
		},
	}}
	svc := NewService(checker, "/tmp/scenarios")

	resp, err := svc.Check(context.Background(), connect.NewRequest(&eligpb.CheckRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	reasons := resp.Msg.GetDisqualifyingReasons()
	if len(reasons) != 1 || !strings.Contains(reasons[0], "routed_database_handle_capture") {
		t.Errorf("expected reason mentioning routed_database_handle_capture; got %v", reasons)
	}
}

func TestCheck_MissingRule(t *testing.T) {
	checker := &fakeChecker{elig: internalelig.Eligibility{
		Routed: false,
		RuleAssertion: &internalelig.RuleAssertion{
			MissingRules: []string{internalelig.RuleDatabaseBackoff, internalelig.RuleRoutedDrivers},
		},
	}}
	svc := NewService(checker, "/tmp/scenarios")

	resp, err := svc.Check(context.Background(), connect.NewRequest(&eligpb.CheckRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ra := resp.Msg.GetRuleAssertion()
	if ra == nil || len(ra.GetMissingRules()) != 2 {
		t.Fatalf("expected RuleAssertion with 2 missing rules; got %+v", ra)
	}
	if ra.GetMissingRules()[0] != internalelig.RuleDatabaseBackoff {
		t.Errorf("expected sorted missing_rules with %s first; got %v", internalelig.RuleDatabaseBackoff, ra.GetMissingRules())
	}
	reasons := resp.Msg.GetDisqualifyingReasons()
	joined := strings.Join(reasons, " | ")
	if !strings.Contains(joined, "did not register required") {
		t.Errorf("expected 'did not register required' reason; got %v", reasons)
	}
}

func TestCheck_AuditorUnreachable(t *testing.T) {
	checker := &fakeChecker{err: errors.New("scan timeout")}
	svc := NewService(checker, "/tmp/scenarios")

	_, err := svc.Check(context.Background(), connect.NewRequest(&eligpb.CheckRequest{Scenario: "demo"}))
	if err == nil {
		t.Fatal("expected error")
	}
	var ce *connect.Error
	if !errors.As(err, &ce) {
		t.Fatalf("expected connect.Error; got %T", err)
	}
	if ce.Code() != connect.CodeInternal {
		t.Errorf("expected CodeInternal; got %v", ce.Code())
	}
}

func TestCheck_EmptyScenario(t *testing.T) {
	svc := NewService(&fakeChecker{}, "/tmp/scenarios")
	_, err := svc.Check(context.Background(), connect.NewRequest(&eligpb.CheckRequest{Scenario: "  "}))
	if err == nil {
		t.Fatal("expected error")
	}
	var ce *connect.Error
	if !errors.As(err, &ce) || ce.Code() != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument; got %v", err)
	}
}
