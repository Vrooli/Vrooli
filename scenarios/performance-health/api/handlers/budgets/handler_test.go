package budgets

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	internalbudgets "performance-health/internal/budgets"

	budgetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/budgets"
)

func newHandler() *Handler {
	return NewHandler(internalbudgets.NewService(internalbudgets.NewStore()), nil)
}

// TestSetThenGetBudgetRoundTrip builds the REAL budgets service over the
// in-memory store, writes a budget via SetBudget, then reads it back via
// GetBudget and asserts every axis maps correctly across the proto boundary.
func TestSetThenGetBudgetRoundTrip(t *testing.T) {
	h := newHandler()
	ctx := context.Background()

	in := &budgetsv1.Budget{
		Scenario:                "demo",
		GoBuildMaxMs:            90000,
		UiBuildMaxMs:            180000,
		BundleMaxBytes:          1048576,
		LcpMaxMs:                2500,
		StartupMaxMs:            5000,
		ComponentCommitAvgMaxMs: 16,
		ComponentCommitMaxMs:    50,
		Ratchet:                 true,
	}
	setResp, err := h.SetBudget(ctx, connect.NewRequest(&budgetsv1.SetBudgetRequest{Budget: in}))
	if err != nil {
		t.Fatalf("SetBudget: %v", err)
	}
	if setResp.Msg.GetDryRun() {
		t.Error("dry_run should be false without the X-Dry-Run header")
	}
	if setResp.Msg.GetBudget().GetGoBuildMaxMs() != 90000 {
		t.Errorf("SetBudget echo wrong: %+v", setResp.Msg.GetBudget())
	}

	getResp, err := h.GetBudget(ctx, connect.NewRequest(&budgetsv1.GetBudgetRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("GetBudget: %v", err)
	}
	if !getResp.Msg.GetDeclared() {
		t.Fatal("budget should be declared after SetBudget")
	}
	got := getResp.Msg.GetBudget()
	switch {
	case got.GetScenario() != "demo":
		t.Errorf("scenario = %q", got.GetScenario())
	case got.GetGoBuildMaxMs() != 90000:
		t.Errorf("go_build_max_ms = %d", got.GetGoBuildMaxMs())
	case got.GetUiBuildMaxMs() != 180000:
		t.Errorf("ui_build_max_ms = %d", got.GetUiBuildMaxMs())
	case got.GetBundleMaxBytes() != 1048576:
		t.Errorf("bundle_max_bytes = %d", got.GetBundleMaxBytes())
	case got.GetLcpMaxMs() != 2500:
		t.Errorf("lcp_max_ms = %d", got.GetLcpMaxMs())
	case got.GetStartupMaxMs() != 5000:
		t.Errorf("startup_max_ms = %d", got.GetStartupMaxMs())
	case got.GetComponentCommitAvgMaxMs() != 16:
		t.Errorf("component_commit_avg_max_ms = %v", got.GetComponentCommitAvgMaxMs())
	case got.GetComponentCommitMaxMs() != 50:
		t.Errorf("component_commit_max_ms = %v", got.GetComponentCommitMaxMs())
	case !got.GetRatchet():
		t.Error("ratchet should round-trip true")
	}
}

// TestSetBudgetDryRunHonorsHeader proves the X-Dry-Run header flips dry_run and
// does NOT persist (a follow-up GetBudget reports not-declared).
func TestSetBudgetDryRunHonorsHeader(t *testing.T) {
	h := newHandler()
	ctx := context.Background()

	req := connect.NewRequest(&budgetsv1.SetBudgetRequest{Budget: &budgetsv1.Budget{Scenario: "demo", GoBuildMaxMs: 1000}})
	req.Header().Set("X-Dry-Run", "true")
	resp, err := h.SetBudget(ctx, req)
	if err != nil {
		t.Fatalf("SetBudget: %v", err)
	}
	if !resp.Msg.GetDryRun() {
		t.Fatal("dry_run should be true when X-Dry-Run: true")
	}
	getResp, err := h.GetBudget(ctx, connect.NewRequest(&budgetsv1.GetBudgetRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("GetBudget: %v", err)
	}
	if getResp.Msg.GetDeclared() {
		t.Error("a dry-run SetBudget must not persist the budget")
	}
}

// TestCheckBudgetUndeclaredPasses asserts CheckBudget on a scenario with no
// declared budget maps to passed=true with no violations.
func TestCheckBudgetUndeclaredPasses(t *testing.T) {
	h := newHandler()
	resp, err := h.CheckBudget(context.Background(), connect.NewRequest(&budgetsv1.CheckBudgetRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("CheckBudget: %v", err)
	}
	if resp.Msg.GetScenario() != "demo" {
		t.Errorf("scenario = %q", resp.Msg.GetScenario())
	}
	if !resp.Msg.GetPassed() {
		t.Error("a scenario with no budget can never breach")
	}
	if len(resp.Msg.GetViolations()) != 0 {
		t.Errorf("expected no violations, got %v", resp.Msg.GetViolations())
	}
}

// TestGetBudgetRequiresScenario asserts the empty-scenario guard maps to
// InvalidArgument.
func TestGetBudgetRequiresScenario(t *testing.T) {
	h := newHandler()
	_, err := h.GetBudget(context.Background(), connect.NewRequest(&budgetsv1.GetBudgetRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want InvalidArgument, got %v (err=%v)", connect.CodeOf(err), err)
	}
}
