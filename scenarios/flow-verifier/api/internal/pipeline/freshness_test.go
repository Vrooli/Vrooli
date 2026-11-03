package pipeline

import (
	"context"
	"errors"
	"testing"
	"time"

	"flow-verifier/internal/flows/kinds/temporal/model"
)

type capturingRecorder struct {
	entries []RunEntry
}

func TestRecordPerFlowUsesStructuredFailureIdentity(t *testing.T) {
	rec := &capturingRecorder{}
	flows := []model.Flow{
		{FlowID: "first.flow"},
		{FlowID: "second.flow"},
		{FlowID: "third.flow"},
	}
	started := map[string]time.Time{
		"first.flow":  time.Now(),
		"second.flow": time.Now(),
		"third.flow":  time.Now(),
	}
	runErr := &flowExecutionError{
		flowID: "second.flow",
		err:    errors.New("failure detail mentions third.flow path"),
	}
	recordPerFlow(context.Background(), rec, flows, started, ModeCheck, "/root", "fresh first.flow\n", runErr)

	if len(rec.entries) != 2 {
		t.Fatalf("recorded %d entries, want first success and second failure only: %+v", len(rec.entries), rec.entries)
	}
	if rec.entries[0].FlowID != "first.flow" || rec.entries[0].Status != "passed" {
		t.Fatalf("first entry = %+v, want first.flow passed", rec.entries[0])
	}
	if rec.entries[1].FlowID != "second.flow" || rec.entries[1].Status != "failed" {
		t.Fatalf("second entry = %+v, want second.flow failed", rec.entries[1])
	}
}

func (c *capturingRecorder) Record(_ context.Context, e RunEntry) error {
	c.entries = append(c.entries, e)
	return nil
}

func TestRecordPerFlow_StampsMissingArtifactsReason(t *testing.T) {
	rec := &capturingRecorder{}
	flows := []model.Flow{{FlowID: "demo.flow", ContractPath: "api/demo/flow/flow.json"}}
	started := map[string]time.Time{"demo.flow": time.Now()}
	err := &FreshnessError{
		FlowID:  "demo.flow",
		Kind:    FreshnessMissing,
		Missing: []string{"api/demo/flow/generated/runtime.go", "api/demo/flow/generated/model.qnt"},
	}
	recordPerFlow(context.Background(), rec, flows, started, ModeCheck, "/root", "", err)
	if len(rec.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(rec.entries))
	}
	entry := rec.entries[0]
	if entry.Status != "failed" {
		t.Fatalf("expected status=failed, got %q", entry.Status)
	}
	if entry.FailureReason != FailureReasonMissingArtifacts {
		t.Fatalf("expected FailureReason=missing_artifacts, got %q", entry.FailureReason)
	}
	if len(entry.MissingArtifacts) != 2 {
		t.Fatalf("expected 2 missing paths, got %v", entry.MissingArtifacts)
	}
}

func TestRecordPerFlow_StampsStaleArtifactsReason(t *testing.T) {
	rec := &capturingRecorder{}
	flows := []model.Flow{{FlowID: "demo.flow"}}
	started := map[string]time.Time{"demo.flow": time.Now()}
	err := &FreshnessError{
		FlowID: "demo.flow",
		Kind:   FreshnessStale,
		Stale:  []string{"api/demo/flow/generated/runtime.go"},
	}
	recordPerFlow(context.Background(), rec, flows, started, ModeCheck, "/root", "", err)
	if rec.entries[0].FailureReason != FailureReasonStaleArtifacts {
		t.Fatalf("expected stale_artifacts, got %q", rec.entries[0].FailureReason)
	}
}
