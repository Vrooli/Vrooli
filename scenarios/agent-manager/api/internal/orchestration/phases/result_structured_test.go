package phases

import (
	"context"
	"encoding/json"
	"testing"

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
