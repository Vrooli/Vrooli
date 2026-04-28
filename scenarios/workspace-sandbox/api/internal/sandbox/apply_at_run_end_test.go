// Validation tests for the apply-at-run-end request shape (Phase 2 of
// execute/agent-manager-sandbox-auto-apply-defaults). The end-to-end Approve
// path is exercised by existing service tests; this file pins the
// auditability-contract-specific request validation that gates incoming
// run-end calls before the shared apply path runs.

package sandbox

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"workspace-sandbox/internal/types"
)

func TestValidateApplyAtRunEndRequest_HappyPath(t *testing.T) {
	req := &types.ApplyAtRunEndRequest{
		SandboxID:         uuid.New(),
		AgentManagerRunID: "run-1",
		Source:            types.SourceAgentManagerAutoApply,
		RunOutcome:        "success",
	}
	if err := validateApplyAtRunEndRequest(req); err != nil {
		t.Fatalf("happy-path request rejected: %v", err)
	}
}

func TestValidateApplyAtRunEndRequest_MissingRunID(t *testing.T) {
	req := &types.ApplyAtRunEndRequest{
		AgentManagerRunID: "   ",
		Source:            types.SourceAgentManagerAutoApply,
	}
	err := validateApplyAtRunEndRequest(req)
	if err == nil {
		t.Fatal("validation accepted empty agentManagerRunId")
	}
	if !strings.Contains(err.Error(), "agentManagerRunId") {
		t.Errorf("expected agentManagerRunId in error; got: %v", err)
	}
}

func TestValidateApplyAtRunEndRequest_RejectsSystemOnlySource(t *testing.T) {
	req := &types.ApplyAtRunEndRequest{
		AgentManagerRunID: "run-1",
		Source:            types.SourceWorkspaceSandboxGC,
	}
	if err := validateApplyAtRunEndRequest(req); err == nil {
		t.Fatal("validation accepted SourceWorkspaceSandboxGC; system-only value must be rejected")
	}
}

func TestValidateApplyAtRunEndRequest_RejectsOperatorSource(t *testing.T) {
	// Operator approval surfaces must use the /approve endpoint, not
	// /apply-at-run-end. apply-at-run-end is exclusively for agent-manager
	// auto-apply.
	cases := []types.ApprovalSource{
		types.SourceGitControlTower,
		types.SourceAgentManagerUI,
		types.SourceWorkspaceSandboxUI,
		types.SourceCLI,
	}
	for _, s := range cases {
		t.Run(string(s), func(t *testing.T) {
			req := &types.ApplyAtRunEndRequest{
				AgentManagerRunID: "run-1",
				Source:            s,
			}
			if err := validateApplyAtRunEndRequest(req); err == nil {
				t.Fatalf("validation accepted source=%q on apply-at-run-end; only SourceAgentManagerAutoApply is valid here", s)
			}
		})
	}
}

func TestValidateApplyAtRunEndRequest_RejectsUnknownSource(t *testing.T) {
	req := &types.ApplyAtRunEndRequest{
		AgentManagerRunID: "run-1",
		Source:            types.ApprovalSource("scrapyard"),
	}
	if err := validateApplyAtRunEndRequest(req); err == nil {
		t.Fatal("validation accepted unknown source")
	}
}

func TestValidateApplyAtRunEndRequest_RejectsUnspecifiedSource(t *testing.T) {
	req := &types.ApplyAtRunEndRequest{
		AgentManagerRunID: "run-1",
		Source:            types.SourceUnspecified,
	}
	if err := validateApplyAtRunEndRequest(req); err == nil {
		t.Fatal("validation accepted unspecified source; explicit attribution required")
	}
}

func TestValidateApplyAtRunEndRequest_AcceptsContractRunOutcomes(t *testing.T) {
	for _, outcome := range []string{"success", "failure", "cancelled", "timeout"} {
		t.Run(outcome, func(t *testing.T) {
			req := &types.ApplyAtRunEndRequest{
				AgentManagerRunID: "run-1",
				Source:            types.SourceAgentManagerAutoApply,
				RunOutcome:        outcome,
			}
			if err := validateApplyAtRunEndRequest(req); err != nil {
				t.Errorf("contract runOutcome %q rejected: %v", outcome, err)
			}
		})
	}
}

func TestValidateApplyAtRunEndRequest_RejectsBogusRunOutcome(t *testing.T) {
	// The agent-manager 7-value RunOutcome (exit_error, exception, etc.)
	// must be mapped to the contract's 4 values BEFORE the wire call;
	// the workspace-sandbox endpoint only accepts the 4 contract values.
	for _, bogus := range []string{"exit_error", "exception", "sandbox_fail", "runner_fail", "running"} {
		t.Run(bogus, func(t *testing.T) {
			req := &types.ApplyAtRunEndRequest{
				AgentManagerRunID: "run-1",
				Source:            types.SourceAgentManagerAutoApply,
				RunOutcome:        bogus,
			}
			if err := validateApplyAtRunEndRequest(req); err == nil {
				t.Errorf("non-contract runOutcome %q accepted; should be rejected", bogus)
			}
		})
	}
}

func TestValidateApplyAtRunEndRequest_AllowsEmptyRunOutcome(t *testing.T) {
	// Empty runOutcome is acceptable (treated as unset / unknown). Some
	// callers may not have a classifiable outcome yet — the contract
	// records "" as a legacy/unset value.
	req := &types.ApplyAtRunEndRequest{
		AgentManagerRunID: "run-1",
		Source:            types.SourceAgentManagerAutoApply,
		RunOutcome:        "",
	}
	if err := validateApplyAtRunEndRequest(req); err != nil {
		t.Errorf("empty runOutcome should be accepted: %v", err)
	}
}
