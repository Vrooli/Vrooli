package backlog

import (
	"context"
	"testing"

	"github.com/vrooli/api-core/provenance"
)

// provenanceRecordingLogger captures the context each emit receives so the test
// can assert the request's provenance survived the call chain.
type provenanceRecordingLogger struct {
	EventLogger
	statusCtx context.Context
	calls     int
}

func (l *provenanceRecordingLogger) EmitBacklogStatusChanged(ctx context.Context, _, _, _ string) {
	l.statusCtx = ctx
	l.calls++
}

// The status-change emitter is only useful if the caller's verified provenance
// reaches it. This asserts the threading through logAndEmitUpdate, which is the
// link between the HTTP middleware and the append-only event row.
func TestStatusChangeEmitCarriesRequestProvenance(t *testing.T) {
	logger := &provenanceRecordingLogger{}
	handler := &Handler{eventLogger: logger}

	ctx := provenance.NewContext(context.Background(), provenance.Provenance{
		Actor:              provenance.ActorAgent,
		VerificationStatus: provenance.VerificationVerified,
		RunID:              "run-status-change",
	})

	handler.logAndEmitUpdate(ctx, "fix", "bug-1", "failed", "completed", 1, 1, "M", "M", "m", "m", nil, nil)

	if logger.calls != 1 {
		t.Fatalf("emit calls = %d, want 1", logger.calls)
	}
	got := provenance.FromContext(logger.statusCtx)
	if !got.IsVerifiedAgent() || got.RunID != "run-status-change" {
		t.Fatalf("provenance did not reach the emitter: %+v", got)
	}
}

// A status change that did not actually change status must not emit: a
// no-op update would otherwise look like pushback evidence.
func TestUnchangedStatusDoesNotEmit(t *testing.T) {
	logger := &provenanceRecordingLogger{}
	handler := &Handler{eventLogger: logger}

	handler.logAndEmitUpdate(context.Background(), "fix", "bug-1", "completed", "completed", 1, 1, "M", "M", "m", "m", nil, nil)

	if logger.calls != 0 {
		t.Fatalf("emit calls = %d, want 0", logger.calls)
	}
}
