package render

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"

	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/perceptual"

	"github.com/stretchr/testify/require"
)

// moireExecutor is a treatment that destroys whatever it is given: it returns
// uniform diagonal hatching of the right geometry, carrying none of the source.
// This is `engraved-colonnade`'s pre-plan output reduced to its essence — an
// image with excellent contrast, correct dimensions, valid PNG encoding, and no
// picture in it.
type moireExecutor struct{}

func (moireExecutor) Apply(_ context.Context, input []byte, _ []string, _, _ map[string]string) ([]byte, error) {
	src, err := png.Decode(bytes.NewReader(input))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	out := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if (x+y)%3 == 0 {
				out.SetNRGBA(x, y, color.NRGBA{R: 12, G: 14, B: 20, A: 255})
			} else {
				out.SetNRGBA(x, y, color.NRGBA{R: 244, G: 240, B: 228, A: 255})
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// TestARenderThatDestroysItsSubjectIsRefused is the assertion whose absence let
// `engraved-colonnade` ship illegible moire with every suite green.
//
// The refusal happens inside the render path, not in a test helper: a candidate
// that reaches the job record is a candidate an operator can select and
// release, so the gate has to run before the record exists.
func TestARenderThatDestroysItsSubjectIsRefused(t *testing.T) {
	store := NewStore(moireExecutor{})
	style := catalog.Style{
		ID: "engraved-colonnade", Strategy: "procedural-treated", Subject: "statuary_architecture",
		Placements: []string{"full_bleed"}, Treatments: []string{"engraving"},
	}
	_, err := store.SubmitWithContext(context.Background(), Request{
		Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 7, Count: 1,
	})
	require.Error(t, err)

	var rejected *QualityRejectedError
	require.ErrorAs(t, err, &rejected, "the caller must be able to tell a quality refusal from a wire failure")
	require.Equal(t, "engraved-colonnade", rejected.StyleID)
	require.Equal(t, int64(7), rejected.Seed)
	require.Contains(t, rejected.Error(), perceptual.MetricSubjectSurvival,
		"the verdict must name the metric that failed, not merely report a failure")
	require.Contains(t, rejected.Error(), "below")
}

// TestAGoodRenderCarriesItsScores proves the gate is not merely absent on the
// happy path: a passing candidate records what it scored, so "the gate is
// green" is a claim a reader can check rather than take on faith.
func TestAGoodRenderCarriesItsScores(t *testing.T) {
	store := NewStore(&fakeExecutor{})
	style := catalog.Style{
		ID: "stipple-massif", Strategy: "procedural-treated", Subject: "geological",
		Placements: []string{"full_bleed"}, Treatments: []string{"stipple"},
	}
	job, err := store.SubmitWithContext(context.Background(), Request{
		Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 7, Count: 1,
	})
	require.NoError(t, err)
	require.Len(t, job.Candidates, 1)

	var verdict perceptual.Verdict
	require.NoError(t, json.Unmarshal([]byte(job.Candidates[0].QualityJSON), &verdict))
	require.True(t, verdict.Passed)
	require.NotEmpty(t, verdict.Metrics)

	bar := style.EffectiveQuality()
	for _, m := range verdict.Metrics {
		require.True(t, m.Passed, "%s scored %v", m.Name, m.Value)
		if m.Name == perceptual.MetricSubjectSurvival {
			require.Equal(t, bar.MinSubjectSurvival, m.Min,
				"the recorded bar must be the style's effective bar, not a constant")
		}
	}
}

// TestDecorativeRegionsAreNotJudgedForQuiet records a deliberate exclusion. A
// decorative region is not a place text has to be readable, so measuring
// texture inside it would reject candidates for a problem that does not exist.
func TestDecorativeRegionsAreNotJudgedForQuiet(t *testing.T) {
	style := catalog.Style{Regions: []catalog.Region{
		{X: 0.1, Y: 0.1, Width: 0.3, Height: 0.2, Kind: "headline"},
		{X: 0.5, Y: 0.5, Width: 0.3, Height: 0.2, Kind: "decorative"},
		{X: 0.1, Y: 0.8, Width: 0, Height: 0.2, Kind: "headline"},
	}}
	regions := reservedRegions(style)
	require.Len(t, regions, 1, "only the real headline region is a legibility obligation")
	require.InDelta(t, 0.1, regions[0].X, 1e-9)
}

func TestQualityRejectionSurvivesErrorWrapping(t *testing.T) {
	inner := &QualityRejectedError{StyleID: "x", Seed: 1, Verdict: perceptual.Verdict{}}
	var target *QualityRejectedError
	require.True(t, errors.As(errors.Join(errors.New("context"), inner), &target))
}
