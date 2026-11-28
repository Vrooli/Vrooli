package prose

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDocumentWritesRejectConcurrentRequestsAndReleaseOnFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	entered := make(chan struct{})
	fail := make(chan struct{})
	s, gateway := newFlakyService(t, func(_ int, req GatewayRequest) error {
		if strings.Contains(req.Query, "Current section: 2 of 2") {
			select {
			case entered <- struct{}{}:
			case <-ctx.Done():
				return ctx.Err()
			}
			select {
			case <-fail:
				return errors.New("provider disconnected")
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})
	_, err := s.CreateProfile(ctx, Profile{
		Key: "narration", Sampler: Sampler{Kind: "direct", K: 1},
		Budget:      Budget{MaxOutputTokens: 512},
		Composition: CompositionPolicy{GatewayAttempts: 1},
	})
	require.NoError(t, err)
	draft := Document{ID: "same-id", Title: "Fixture", ProfileKey: "narration", Outline: []OutlineSection{
		{Intent: "Opening", Summary: "Set up the problem.", TargetWords: 30},
		{Intent: "Ending", Summary: "Resolve the problem.", TargetWords: 30},
	}}

	// Hold the writer at beat two so the first committed position is visible.
	checkContention := func() {
		t.Helper()
		select {
		case <-entered:
		case <-ctx.Done():
			t.Fatal("writer did not reach the second beat")
		}
		_, err := s.ResumeDocument(ctx, draft.ID)
		require.ErrorIs(t, err, ErrDocumentBusy)
		_, err = s.CreateDocument(ctx, draft, nil)
		require.ErrorIs(t, err, ErrDocumentBusy)
		_, err = s.AssembleDocument(ctx, draft.ID)
		require.ErrorIs(t, err, ErrDocumentBusy)
	}

	done := make(chan error, 1)
	go func() {
		_, err := s.CreateDocument(ctx, draft, nil)
		done <- err
	}()
	checkContention()
	other, err := s.CreateDocument(ctx, Document{ID: "other-id"}, []Section{{Intent: "Independent draft"}})
	require.NoError(t, err)
	require.Equal(t, "other-id", other.ID)
	partial, err := s.GetDocument(ctx, draft.ID)
	require.NoError(t, err)
	require.Len(t, partial.Sections, 1)
	fail <- struct{}{}
	require.ErrorContains(t, <-done, "provider disconnected")

	go func() {
		_, err := s.ResumeDocument(ctx, draft.ID)
		done <- err
	}()
	checkContention()
	fail <- struct{}{}
	require.ErrorContains(t, <-done, "provider disconnected")

	gateway.fail = nil
	finished, err := s.ResumeDocument(ctx, draft.ID)
	require.NoError(t, err)
	require.Equal(t, "assembled", finished.Status)
	require.Len(t, finished.Sections, 2)
	require.Equal(t, partial.Sections[0].CommittedCandidateID, finished.Sections[0].CommittedCandidateID)
	require.Len(t, gateway.inner.calls, 2, "each beat should reach inference only once")
}
