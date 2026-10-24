package phases

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

type structuredResolverStub struct{ calls int }

func (s *structuredResolverStub) Resolve(_ context.Context, spec *domain.ResultSpec, result *domain.RunResult) *domain.StructuredResult {
	s.calls++
	if spec == nil || result == nil {
		return nil
	}
	return &domain.StructuredResult{
		Status: domain.StructuredResultSuccess, SpecKind: spec.Kind,
		SchemaDigest: spec.SchemaDigest, Value: json.RawMessage(`{"ok":true}`), Method: "whole_document",
	}
}

func TestHandleResultAttachesStructuredProjectionBeforePersistence(t *testing.T) { // [REQ:REQ-P2-001]
	resolver := &structuredResolverStub{}
	run := &domain.Run{
		RunMode: domain.RunModeInPlace, Status: domain.RunStatusRunning,
		ResolvedConfig: &domain.RunConfig{ResultSpec: &domain.ResultSpec{
			Kind: domain.ResultSpecKindJSONSchema, SchemaDigest: "sha256:test",
		}},
	}
	execution := &runner.ExecuteResult{Success: true, Result: &domain.RunResult{
		Success: true, Selection: domain.FinalOutputSelection{Status: domain.FinalOutputSelectionSelected},
	}}

	HandleResult(context.Background(), HandleResultInput{
		Deps: Deps{StructuredResults: resolver}, Run: run, Result: execution,
	})

	if resolver.calls != 1 || run.Result == nil || run.Result.Structured == nil {
		t.Fatalf("structured projection not attached: calls=%d run=%+v", resolver.calls, run)
	}
	if run.Result.Structured.SchemaDigest != "sha256:test" {
		t.Fatalf("structured result = %+v", run.Result.Structured)
	}
}

func TestHandleResultFinalizesInPlaceSuccessAndFailureWithoutSandboxSideEffects(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	success := &domain.Run{RunMode: domain.RunModeInPlace, Status: domain.RunStatusRunning, IdentityTokenHash: "token"}
	result := &runner.ExecuteResult{Success: true, ExitCode: 0, SessionID: "session", Summary: &domain.RunSummary{Description: "done"}, Result: &domain.RunResult{Success: true}}
	out := HandleResult(context.Background(), HandleResultInput{Deps: Deps{Clock: func() time.Time { return now }}, Run: success, Result: result})
	if out.Outcome != domain.RunOutcomeSuccess || success.Status != domain.RunStatusComplete || success.FinalizationStatus != domain.RunFinalizationStatusSkipped || success.SessionID != "session" || success.EndedAt == nil || success.IdentityTokenRevokedAt == nil {
		t.Fatalf("success result = output=%+v run=%+v", out, success)
	}

	failure := &domain.Run{RunMode: domain.RunModeInPlace, Status: domain.RunStatusRunning}
	out = HandleResult(context.Background(), HandleResultInput{Deps: Deps{Clock: func() time.Time { return now }}, Run: failure, Result: &runner.ExecuteResult{ExitCode: 2, ErrorMessage: "runner failed"}})
	if out.Outcome != domain.RunOutcomeExitError || failure.Status != domain.RunStatusFailed || failure.ErrorMsg != "runner failed" || failure.ExitCode == nil || *failure.ExitCode != 2 || failure.FinalizationStatus != domain.RunFinalizationStatusSkipped {
		t.Fatalf("failure result = output=%+v run=%+v", out, failure)
	}
}

func TestResultErrorClassificationPreservesTypedRecoveryIntent(t *testing.T) {
	if got := ClassifyErrorOutcome(&domain.SandboxError{}); got != domain.RunOutcomeSandboxFail {
		t.Fatalf("sandbox error outcome=%s", got)
	}
	if got := ClassifyErrorOutcome(&domain.ConfigError{Missing: true, Setting: "sandbox"}); got != domain.RunOutcomeSandboxFail {
		t.Fatalf("missing sandbox outcome=%s", got)
	}
	if got := ClassifyErrorOutcome(&domain.ConfigError{Missing: true, Setting: "runner"}); got != domain.RunOutcomeException {
		t.Fatalf("other config outcome=%s", got)
	}
	if got := ClassifyErrorOutcome(&domain.RunnerError{Operation: "timeout"}); got != domain.RunOutcomeTimeout {
		t.Fatalf("timeout runner outcome=%s", got)
	}
	if got := ClassifyErrorOutcome(&domain.RunnerError{Operation: "acquire"}); got != domain.RunOutcomeRunnerFail {
		t.Fatalf("runner outcome=%s", got)
	}
	if got := ClassifyErrorOutcome(errors.New("unexpected")); got != domain.RunOutcomeException {
		t.Fatalf("generic outcome=%s", got)
	}
	if got := classifyOutcome(nil, &runner.ExecuteResult{ExitCode: 0}); got != domain.RunOutcomeSuccess {
		t.Fatalf("success classification=%s", got)
	}
	if got := classifyOutcome(errors.New("execute"), nil); got != domain.RunOutcomeException {
		t.Fatalf("exception classification=%s", got)
	}
}

func TestHandleCancellationMarksInPlaceRunTerminalWithoutSandboxLifecycle(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	run := &domain.Run{RunMode: domain.RunModeInPlace, Status: domain.RunStatusRunning, IdentityTokenHash: "token"}
	HandleCancellation(context.Background(), HandleResultInput{Deps: Deps{Clock: func() time.Time { return now }}, Run: run})
	if run.Status != domain.RunStatusCancelled || run.FinalizationStatus != domain.RunFinalizationStatusSkipped || run.IdentityTokenRevokedAt == nil {
		t.Fatalf("cancelled run=%+v", run)
	}
}
