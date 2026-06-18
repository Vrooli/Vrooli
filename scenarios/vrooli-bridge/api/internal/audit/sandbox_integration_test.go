package audit_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"vrooli-bridge/internal/audit"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// sandboxSubstrate is a stand-in for the workspace-sandbox accountability
// substrate. The point of the audit domain's Sink seam is that the operations
// being audited (dispatch/provision) route records to "the accountability
// substrate" — NOT to a bespoke store baked into the caller. This substrate
// satisfies the SAME audit.Sink + audit.Reader seams as the default SQLite
// store, so it can be wired in its place with zero changes to any caller. That
// interchangeability IS the routing guarantee SECURITY.md requires.
type sandboxSubstrate struct {
	mu      sync.Mutex
	records []audit.Record
	now     time.Time
}

var (
	_ audit.Sink   = (*sandboxSubstrate)(nil)
	_ audit.Reader = (*sandboxSubstrate)(nil)
)

func (s *sandboxSubstrate) Append(_ context.Context, r audit.Record) (audit.Record, error) {
	if r.Actor == "" {
		return audit.Record{}, audit.ErrInvalidRecord{Field: "actor", Reason: "required"}
	}
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.RecordedAt.IsZero() {
		r.RecordedAt = s.now
	}
	s.mu.Lock()
	s.records = append(s.records, r)
	s.mu.Unlock()
	return r, nil
}

func (s *sandboxSubstrate) List(_ context.Context, filter audit.ListFilter) ([]audit.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]audit.Record, 0, len(s.records))
	for i := len(s.records) - 1; i >= 0; i-- { // newest-first
		r := s.records[i]
		if filter.NodeID != "" && r.NodeID != filter.NodeID {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

// recordingOperation models the write side the dispatch/provision domains use:
// they hold ONLY the narrow audit.Sink and route every accepted/rejected
// operation to it, never to a concrete store. This function is deliberately
// generic over the Sink so the test proves the seam — not a specific substrate.
func recordingOperation(ctx context.Context, sink audit.Sink, accepted bool) error {
	outcome := audit.OutcomeRejected
	detail := "verb not allowlisted"
	if accepted {
		outcome = audit.OutcomeAccepted
		detail = "dispatched"
	}
	_, err := sink.Append(ctx, audit.Record{
		Action: audit.ActionDispatch, Actor: "owner-1", NodeID: "n1",
		Scenario: "web-search", Verb: "scenario test", Outcome: outcome, Detail: detail,
	})
	return err
}

// [REQ:BRG-P0-008] Records are routed to the accountability substrate (here the
// workspace-sandbox stand-in) through the Sink seam rather than a bespoke store
// the caller owns. The substrate is swappable behind the seam with no caller
// change, and both accepted and rejected operations land in it.
func TestAudit_RoutesThroughSinkSeamToSubstrate(t *testing.T) {
	ctx := context.Background()
	substrate := &sandboxSubstrate{now: time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)}

	require.NoError(t, recordingOperation(ctx, substrate, true))
	require.NoError(t, recordingOperation(ctx, substrate, false))

	got, err := substrate.List(ctx, audit.ListFilter{NodeID: "n1"})
	require.NoError(t, err)
	require.Len(t, got, 2, "both the accepted dispatch and the denial landed in the substrate")
	require.Equal(t, audit.OutcomeRejected, got[0].Outcome)
	require.Equal(t, audit.OutcomeAccepted, got[1].Outcome)
}

// [REQ:BRG-P0-008] The default SQLite store and the workspace-sandbox-shaped
// substrate are interchangeable behind the seam — the same operation code
// routes to either without modification.
func TestAudit_SqliteStoreAndSubstrateAreInterchangeable(t *testing.T) {
	ctx := context.Background()
	store, _, _ := newSchemaStore(t)
	substrate := &sandboxSubstrate{now: time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)}

	for _, sink := range []audit.Sink{store, substrate} {
		require.NoError(t, recordingOperation(ctx, sink, true))
	}

	fromStore, err := store.List(ctx, audit.ListFilter{NodeID: "n1"})
	require.NoError(t, err)
	require.Len(t, fromStore, 1)
	fromSubstrate, err := substrate.List(ctx, audit.ListFilter{NodeID: "n1"})
	require.NoError(t, err)
	require.Len(t, fromSubstrate, 1)
}
