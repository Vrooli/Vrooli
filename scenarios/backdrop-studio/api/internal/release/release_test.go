package release

import "context"
import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProceduralReleaseDerivesDisclosureAndRequiresAltText(t *testing.T) {
	s := NewStore()
	_, err := s.Release(Request{CandidateID: "c", StyleID: "s", Strategy: "procedural", Width: 10, Height: 10, LegibilityPasses: true})
	require.Error(t, err)
	b, err := s.Release(Request{CandidateID: "c", StyleID: "s", Strategy: "procedural", Width: 10, Height: 10, AltText: "ambient", LegibilityPasses: true})
	require.NoError(t, err)
	require.False(t, b.AIGenerated)
}
func TestReleaseRejectsDirectDisclosureAndGeometry(t *testing.T) {
	s := NewStore()
	_, err := s.Release(Request{CandidateID: "c", StyleID: "s", Strategy: "guided", Width: 9, Height: 10, ExpectedWidth: 10, ExpectedHeight: 10, AIGeneratedSet: true, AltText: "x", LegibilityPasses: true})
	require.Error(t, err)
	require.Contains(t, err.Error(), "ai_generated")
}

type fakePublisher struct{ calls int }

func (p *fakePublisher) Publish(_ context.Context, _ Request, _ bool) (string, error) {
	p.calls++
	return "asset-123", nil
}
func TestModelBackedReleaseHandsOffToAssetStudio(t *testing.T) {
	publisher := &fakePublisher{}
	b, err := NewStoreWithPublisher(publisher).Release(Request{CandidateID: "c", StyleID: "s", Strategy: "guided", Width: 10, Height: 10, AltText: "a restrained field", LegibilityPasses: true, ContrastRatio: 5, ContrastThreshold: 4.5})
	require.NoError(t, err)
	require.Equal(t, 1, publisher.calls)
	require.Equal(t, "asset-123", b.AssetStudioRef)
	require.True(t, b.AIGenerated)
}
