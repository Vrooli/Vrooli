package recall

import (
	"context"
	"errors"
	"testing"
	"time"

	journal "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	learningpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/learning"

	"vrooli-memory/internal/learning"

	"vrooli-memory/internal/ledgerclient"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	sourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall"
	sourceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall/recall_v1connect"
	memoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/recall"
	"google.golang.org/protobuf/proto"
)

type unavailableRecall struct{}

func (unavailableRecall) Recall(context.Context, *connect.Request[sourcev1.RecallRequest]) (*connect.Response[sourcev1.RecallResponse], error) {
	return nil, &ledgerclient.UnavailableError{Operation: "recall", Err: errors.New("source-ledger stopped")}
}

func (unavailableRecall) Wake(context.Context, *connect.Request[sourcev1.WakeRequest]) (*connect.Response[sourcev1.WakeResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unused"))
}

func (unavailableRecall) Zoom(context.Context, *connect.Request[sourcev1.ZoomRequest]) (*connect.Response[sourcev1.ZoomResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unused"))
}

func (unavailableRecall) ListSiblingEvents(context.Context, *connect.Request[sourcev1.ListSiblingEventsRequest]) (*connect.Response[sourcev1.ListSiblingEventsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unused"))
}

var _ sourceconnect.RecallServiceClient = unavailableRecall{}

func TestRecallReturnsTypedUnavailableWhenSourceLedgerIsDown(t *testing.T) {
	h := NewConnectHandler(unavailableRecall{}, nil)
	_, err := h.Recall(context.Background(), connect.NewRequest(&memoryv1.RecallRequest{Query: "durable memory", Scope: "agent-memory"}))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

// [REQ:LV-12] Later-run negative feedback changes the evidence on recalled advice.
func TestRecallLearningEvidenceAppliesLaterFeedback(t *testing.T) {
	stamp := time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano)
	attempt := &learningpb.Attempt{AttemptId: "attempt-1", TaskId: "task-1", Operation: "demo.run", ContextKey: "ctx", StartedAt: stamp, FinishedAt: stamp, TaskStartedAt: stamp, AttemptNumber: 1, Outcome: "verified_success", EvidenceRefs: []string{"observed:ok"}, RecallStatus: "matched", Provenance: "agent", Trigger: "task", Approach: "test", Advice: []*learningpb.AdviceUse{{EntryId: "note-1", Decision: "applied", DecisionChange: "selected:one", Verdict: "supported", EvidenceRefs: []string{"observed:ok"}}}}
	body, err := learning.Encode(attempt)
	if err != nil {
		t.Fatal(err)
	}
	observation, err := learning.EncodeObservation(&learningpb.Observation{ObservationId: "feedback-1", AttemptId: "attempt-1", Disposition: "contradicted", EvidenceRefs: []string{"later:failed"}, MethodRevision: "learn.feedback", Provenance: "agent", ObservedAt: stamp})
	if err != nil {
		t.Fatal(err)
	}
	hits := []*memoryv1.RecallHit{{EntryId: "note-1"}}
	applyLearningEvidence(hits, []*journal.Entry{{Body: body}, {Body: body}, {Body: observation}}, true)
	if hits[0].Supported != 0 || hits[0].Contradicted != 1 || !hits[0].GetEvidenceReliable() {
		t.Fatalf("wrong evidence: %+v", hits[0])
	}
	applyLearningEvidence(hits, []*journal.Entry{{Body: body}}, false)
	if hits[0].GetEvidenceReliable() {
		t.Fatal("partial evidence must remain unreliable")
	}
}

func TestRecallNoteInheritsOriginEvidenceAndRootContext(t *testing.T) {
	stamp := time.Now().UTC().Format(time.RFC3339Nano)
	root := &learningpb.Attempt{AttemptId: "root", TaskId: "task", Operation: "demo.run", ContextKey: "stable", StartedAt: stamp, FinishedAt: stamp, TaskStartedAt: stamp, AttemptNumber: 1, Outcome: "verified_success", EvidenceRefs: []string{"checked:root"}, RecallStatus: "no_match", Provenance: "agent", Trigger: "task", Approach: "test"}
	child := *root
	child.AttemptId, child.ParentAttemptId, child.ContextKey, child.StepName = "child", proto.String("root"), "step-context", proto.String("choice")
	child.Outcome, child.FailureFingerprint = "failed", "fixture:failed"
	rootBody, err := learning.Encode(root)
	require.NoError(t, err)
	childBody, err := learning.Encode(&child)
	require.NoError(t, err)
	noteBody, err := learning.EncodeObservation(&learningpb.Observation{ObservationId: "note", AttemptId: "child", Disposition: "unknown", Correction: `preference/v1 {"option_id":"bad"}`, EvidenceRefs: []string{"observed:note"}, MethodRevision: "learn.preference", Provenance: "agent", ObservedAt: stamp})
	require.NoError(t, err)
	hits := []*memoryv1.RecallHit{{EntryId: "note"}}
	applyLearningEvidence(hits, []*journal.Entry{{Body: rootBody}, {Body: childBody}, {Id: "note", Body: noteBody}}, true)
	require.Equal(t, int32(1), hits[0].Contradicted)
	require.Zero(t, hits[0].Supported)
	require.Equal(t, "stable", hits[0].ContextKey)
	require.Equal(t, "demo.run", hits[0].Operation)
	require.True(t, hits[0].GetEvidenceReliable())
}
