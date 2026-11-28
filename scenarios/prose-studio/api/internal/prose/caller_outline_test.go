package prose

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCallerOutlineCompositionAndResume(t *testing.T) {
	s, gateway := newFlakyService(t, func(_ int, req GatewayRequest) error {
		if strings.Contains(req.Query, "Current section: 2 of 2") {
			return errors.New("fixture provider offline")
		}
		return nil
	})
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{
		Key: "narration", Sampler: Sampler{Kind: "direct", K: 1},
		Budget:      Budget{MaxOutputTokens: 512},
		Constraints: Constraints{MaxWords: 200},
		Composition: CompositionPolicy{MaxSections: 2, GatewayAttempts: 1},
	})
	require.NoError(t, err)

	outline := []OutlineSection{
		{Intent: "Opening", Summary: "Set up the missing key.", TargetWords: 80},
		{Intent: "Reveal", Summary: "Reveal where it was left.", TargetWords: 100},
	}
	_, err = s.CreateDocument(ctx, Document{ID: "caller-outline", Title: "The key", ProfileKey: "narration", Outline: outline}, nil)
	require.ErrorContains(t, err, "fixture provider offline")
	partial, err := s.GetDocument(ctx, "caller-outline")
	require.NoError(t, err)
	require.Len(t, partial.SectionIDs, 1)
	require.Equal(t, outline, partial.Outline)

	_, err = s.CreateDocument(ctx, Document{ID: partial.ID, ProfileKey: "narration", Outline: outline}, nil)
	require.ErrorIs(t, err, ErrDocumentExists)

	gateway.fail = nil
	doc, err := s.ResumeDocument(ctx, partial.ID)
	require.NoError(t, err)
	require.Equal(t, partial.SectionIDs[0], doc.SectionIDs[0])
	require.Equal(t, outline, doc.Outline)
	require.Empty(t, doc.OutlineID)
	require.Equal(t, "assembled", doc.Status)
	require.Equal(t, 2, doc.Provenance.SectionCount)
	require.Len(t, gateway.inner.calls, 2)
	require.Contains(t, gateway.inner.calls[0].Query, "Current section: 1 of 2")
	require.Contains(t, gateway.inner.calls[1].Query, "Section summary: Reveal where it was left.")
	require.Contains(t, gateway.inner.calls[1].Query, "Prior section 1 (Opening)")
	require.Contains(t, gateway.inner.calls[1].Query, "follow the supplied beat order")

	resumed, err := s.ResumeDocument(ctx, doc.ID)
	require.NoError(t, err)
	require.Equal(t, doc.SectionIDs, resumed.SectionIDs)
	require.Equal(t, doc.AssembledText, resumed.AssembledText)
	require.Equal(t, outline, resumed.Outline)
	require.Len(t, gateway.inner.calls, 2)
}

func TestInvalidCallerOutlineStopsBeforePersistenceAndInference(t *testing.T) {
	s, gateway := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{
		Key: "narration", Sampler: Sampler{Kind: "direct", K: 1},
		Budget:      Budget{MaxOutputTokens: 512},
		Constraints: Constraints{MaxWords: 100},
		Composition: CompositionPolicy{MaxSections: 1},
	})
	require.NoError(t, err)

	valid := OutlineSection{Intent: "Opening", Summary: "Set up the missing key.", TargetWords: 80}
	for _, outline := range [][]OutlineSection{
		{{Intent: "Opening", TargetWords: 80}},
		{{Intent: "Opening", Summary: "Reveal the secret.", TargetWords: 101}},
		{valid, valid},
	} {
		_, err := s.CreateDocument(ctx, Document{ID: "invalid-outline", ProfileKey: "narration", Outline: outline}, nil)
		require.ErrorIs(t, err, ErrMalformedOutline)
		_, err = s.GetDocument(ctx, "invalid-outline")
		require.ErrorIs(t, err, ErrDocumentNotFound)
	}
	require.Empty(t, gateway.calls)
}
