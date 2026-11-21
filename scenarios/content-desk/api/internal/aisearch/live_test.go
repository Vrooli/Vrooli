package aisearch

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	internalartifacts "content-desk/internal/artifacts"
)

func TestLiveSearchServesLexicalWithoutEngine(t *testing.T) {
	source := NewStoreSource(
		sourceDraftsStub{
			drafts:    []internalartifacts.Draft{{ID: "draft-1", Body: "Launch evidence for product team"}},
			revisions: map[string]internalartifacts.CurrentRevision{"draft-1": {DraftID: "draft-1"}},
		},
		sourceLedgerStub{},
	)
	live := NewLexicalSearch(NewService(source))
	require.False(t, live.EngineAvailable())

	response, err := live.Search(context.Background(), "launch", 10)
	require.NoError(t, err)
	require.NotEmpty(t, response.Hits)
	require.NotEmpty(t, response.Generation)
	require.Equal(t, "draft/draft-1", response.Hits[0].FollowUp)
}

func TestApplyIndexTimeUsesReconcileOverMaterialization(t *testing.T) {
	status := StatusReport{LastIndexedAt: time.Date(2026, time.September, 9, 1, 50, 0, 0, time.UTC)}

	got := applyIndexTime(status, "2026-09-12T23:40:00Z")
	require.Equal(t, time.Date(2026, time.September, 12, 23, 40, 0, 0, time.UTC), got.LastIndexedAt,
		"the registered index timestamp is the last successful reconcile, not source age")
}

func TestApplyIndexTimeFallsBackWhenReconcileUnavailable(t *testing.T) {
	materialized := time.Date(2026, time.September, 9, 1, 50, 0, 0, time.UTC)
	status := StatusReport{LastIndexedAt: materialized}

	require.Equal(t, materialized, applyIndexTime(status, "").LastIndexedAt,
		"a boot without a completed reconcile must not fabricate an index time")
	require.Equal(t, materialized, applyIndexTime(status, "not-a-time").LastIndexedAt,
		"an unparsable reconcile time must not fabricate an index time")
}
