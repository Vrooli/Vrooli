package release

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"backdrop-studio/internal/catalog"

	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"

	"github.com/stretchr/testify/require"
)

type fakeCandidateSource map[string]CandidateEvidence

func (f fakeCandidateSource) CandidateEvidence(id string) (CandidateEvidence, bool) {
	candidate, ok := f[id]
	return candidate, ok
}

func solidPNG(t *testing.T, fill color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, fill)
		}
	}
	var out bytes.Buffer
	require.NoError(t, png.Encode(&out, img))
	return out.Bytes()
}

func authoritativeCandidate(t *testing.T) CandidateEvidence {
	return CandidateEvidence{
		ID:                "c",
		JobID:             "job-c",
		ImagePNG:          solidPNG(t, color.Black),
		Width:             8,
		Height:            8,
		StyleID:           "s",
		Strategy:          "procedural",
		SurfaceID:         "web.hero",
		Placement:         "full_bleed",
		Regions:           []catalog.Region{{X: 0, Y: 0, Width: 1, Height: 1, Kind: "overlay", TextColor: "#FFFFFF"}},
		ContrastThreshold: 4.5,
	}
}

func evidenceStore(t *testing.T, candidate CandidateEvidence) *Store {
	t.Helper()
	return durableStore(t,
		&fakePublisher{},
		fakeProvenance{"c": {Strategy: "procedural", ModelBacked: false}},
		fakeCandidateSource{"c": candidate},
	)
}

func durableStore(t *testing.T, publisher AssetPublisher, provenance ProvenanceSource, candidates CandidateSource) *Store {
	t.Helper()
	return NewStoreWithPublisherAndRoots(publisher, provenance, filerouting.New(storage.Paths{DataDir: t.TempDir()}), candidates)
}

func TestReleaseUsesAuthoritativeCandidateBytesAndMeasurement(t *testing.T) {
	candidate := authoritativeCandidate(t)
	store := evidenceStore(t, candidate)

	// Every qualification field below is intentionally forged or omitted. The
	// release owner must use the render record, not the caller's assertion.
	released, err := store.Release(Request{
		CandidateID:       "c",
		StyleID:           "s",
		Strategy:          "procedural",
		SurfaceID:         "web.hero",
		AltText:           "A qualified ambient backdrop",
		LegibilityPasses:  true,
		ContrastRatio:     21,
		ContrastThreshold: 1,
		ImagePNG:          solidPNG(t, color.White),
	})
	require.NoError(t, err)
	require.Equal(t, candidate.ImagePNG, released.ImagePNG)
	require.Equal(t, candidate.JobID, released.JobID)
	require.Equal(t, 8, released.Width)
	require.Equal(t, 8, released.Height)
	require.Equal(t, "image/png", released.MIMEType)
	require.NotEmpty(t, released.ContentHash)
	require.Equal(t, candidate.SurfaceID, released.SurfaceID)
	require.InDelta(t, 21, released.ContrastRatio, 0.001)
	require.InDelta(t, 4.5, released.ContrastThreshold, 0.001)
}

func TestReleaseRefusesCallerOnlyLegibilityEvidence(t *testing.T) {
	candidate := authoritativeCandidate(t)
	candidate.Regions = []catalog.Region{{X: 0, Y: 0, Width: 1, Height: 1, Kind: "overlay", TextColor: "#000000"}}
	store := evidenceStore(t, candidate)

	_, err := store.Release(Request{
		CandidateID:       "c",
		StyleID:           "s",
		Strategy:          "procedural",
		SurfaceID:         "web.hero",
		AltText:           "A backdrop",
		LegibilityPasses:  true,
		ContrastRatio:     21,
		ContrastThreshold: 4.5,
		ImagePNG:          solidPNG(t, color.White),
	})
	require.ErrorContains(t, err, "measured ratio")
}

func TestReleaseRefusesMissingCandidateEvidence(t *testing.T) {
	store := evidenceStore(t, CandidateEvidence{ID: "c", JobID: "job-c"})

	_, err := store.Release(Request{CandidateID: "c", StyleID: "s", Strategy: "procedural", AltText: "A backdrop"})
	require.ErrorContains(t, err, "candidate bytes")
}

func TestReleaseRefusesCandidateMetadataMismatch(t *testing.T) {
	candidate := authoritativeCandidate(t)
	candidate.Width = 7
	store := evidenceStore(t, candidate)

	_, err := store.Release(Request{CandidateID: "c", StyleID: "s", Strategy: "procedural", AltText: "A backdrop"})
	require.ErrorContains(t, err, "candidate dimensions")
}

func TestReleaseRefusesContradictoryProvenance(t *testing.T) {
	candidate := authoritativeCandidate(t)
	store := durableStore(t,
		&fakePublisher{},
		fakeProvenance{"c": {Strategy: "guided", ModelBacked: true}},
		fakeCandidateSource{"c": candidate},
	)

	_, err := store.Release(Request{CandidateID: "c", StyleID: "s", Strategy: "procedural", SurfaceID: "web.hero", AltText: "A backdrop"})
	require.ErrorContains(t, err, "provenance strategy")
}
