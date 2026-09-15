package release

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProceduralReleaseDerivesDisclosureAndRequiresAltText(t *testing.T) {
	s := evidenceStore(t, authoritativeCandidate(t))
	_, err := s.Release(Request{CandidateID: "c", StyleID: "s", Strategy: "procedural", Width: 10, Height: 10, LegibilityPasses: true})
	require.Error(t, err)
	b, err := s.Release(Request{CandidateID: "c", StyleID: "s", Strategy: "procedural", AltText: "ambient", LegibilityPasses: true})
	require.NoError(t, err)
	require.False(t, b.AIGenerated)
}

func TestReleaseRejectsDirectDisclosureAndGeometry(t *testing.T) {
	s := NewStore()
	_, err := s.Release(Request{CandidateID: "c", StyleID: "s", Strategy: "guided", Width: 9, Height: 10, ExpectedWidth: 10, ExpectedHeight: 10, AIGeneratedSet: true, AltText: "x", LegibilityPasses: true})
	require.Error(t, err)
	require.Contains(t, err.Error(), "ai_generated")
}

type fakePublisher struct {
	calls int
	last  Provenance
}

func (p *fakePublisher) Publish(_ context.Context, _ Request, prov Provenance) (string, error) {
	p.calls++
	p.last = prov
	return "asset-123", nil
}

// fakeProvenance stands in for the render store: it is what the release path
// consults instead of trusting its caller's description of how a candidate was
// made.
type fakeProvenance map[string]Provenance

func (f fakeProvenance) CandidateProvenance(id string) (Provenance, bool) {
	p, ok := f[id]
	return p, ok
}

func guidedProvenance() fakeProvenance {
	return fakeProvenance{"c": {
		Strategy: "guided", ModelBacked: true, Model: "sd-1.5/local-gpu",
		Prompt: "sunlit modernist interior", Seed: "7", Conditioner: "edge",
	}}
}

func modelBackedRequest() Request {
	return Request{CandidateID: "c", StyleID: "s", Strategy: "guided", AltText: "a restrained field", LegibilityPasses: true, ContrastRatio: 5, ContrastThreshold: 4.5}
}

func TestModelBackedReleaseHandsOffToAssetStudio(t *testing.T) {
	publisher := &fakePublisher{}
	candidate := authoritativeCandidate(t)
	candidate.Strategy = "guided"
	b, err := NewStoreWithPublisher(publisher, guidedProvenance(), fakeCandidateSource{"c": candidate}).Release(modelBackedRequest())
	require.NoError(t, err)
	require.Equal(t, 1, publisher.calls)
	require.Equal(t, "asset-123", b.AssetStudioRef)
	require.True(t, b.AIGenerated)
	// The facts handed over are the render's, not the caller's.
	require.Equal(t, "sd-1.5/local-gpu", publisher.last.Model)
	require.Equal(t, "sunlit modernist interior", publisher.last.Prompt)
	require.Equal(t, "7", publisher.last.Seed)
}

// TestModelBackedReleaseRefusesWithoutRecordedProvenance is the honest-refusal
// half. A candidate this process did not render has no model or prompt recorded
// anywhere, so its disclosure cannot be written — and inventing one, or falling
// back to releasing it as procedural, would put an undisclosed synthetic image
// into circulation.
func TestModelBackedReleaseRefusesWithoutRecordedProvenance(t *testing.T) {
	publisher := &fakePublisher{}
	candidate := authoritativeCandidate(t)
	candidate.Strategy = "guided"
	_, err := NewStoreWithPublisher(publisher, fakeProvenance{}, fakeCandidateSource{"c": candidate}).Release(modelBackedRequest())
	require.ErrorContains(t, err, "no recorded provenance")
	require.Zero(t, publisher.calls)
}

// TestModelBackedReleaseRefusesWhenAssetStudioIsAbsent preserves the behaviour
// that existed before the ingress landed: a missing capability is named, and
// nothing is published under a fabricated provenance.
func TestModelBackedReleaseRefusesWhenAssetStudioIsAbsent(t *testing.T) {
	candidate := authoritativeCandidate(t)
	candidate.Strategy = "guided"
	_, err := NewStoreWithPublisher(nil, guidedProvenance(), fakeCandidateSource{"c": candidate}).Release(modelBackedRequest())
	require.ErrorContains(t, err, "asset-studio publisher capability")
}

// TestReleaseRefusesAStrategyItsRenderDoesNotAgreeWith catches a caller that
// declares a procedural candidate as model-backed, or the reverse, which would
// otherwise decide the AI-generated label from the caller's word alone.
func TestReleaseRefusesAStrategyItsRenderDoesNotAgreeWith(t *testing.T) {
	publisher := &fakePublisher{}
	source := fakeProvenance{"c": {Strategy: "procedural-treated", ModelBacked: false}}
	candidate := authoritativeCandidate(t)
	candidate.Strategy = "procedural-treated"
	_, err := NewStoreWithPublisher(publisher, source, fakeCandidateSource{"c": candidate}).Release(modelBackedRequest())
	require.ErrorContains(t, err, "does not match candidate evidence")
	require.Zero(t, publisher.calls)
}
