//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"image"
	_ "image/png"
	"testing"

	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/imageengine"
	"backdrop-studio/internal/perceptual"

	"github.com/stretchr/testify/require"
)

// prePlanEngravingParams are `engraved-colonnade`'s treatment parameters exactly
// as they stood before this plan: an absolute 7px hatch period, no tonal
// normalisation. Rendered through the engraving treatment as it stood before
// the sub-pixel-mark fix, this produced the diagonal moire the plan was written
// around.
const prePlanEngravingParams = `{"spacing":7,"dark":"$brand.primary","light":"$brand.background"}`

// TestTheKnownBadCaseStillFails is the gate's own regression test.
//
// A gate that cannot fail its founding case is not a gate. This renders the
// original defect through the really running image-tools and asserts the
// perceptual metrics refuse it — so a future change that quietly loosens a
// threshold, or reverts the mark-width fix, fails here rather than in a released
// asset.
//
// It scores directly rather than going through `render submit`, because the
// render path takes its parameters from the catalog and the catalog no longer
// contains this defect. That is the point: the defect exists only here now.
func TestTheKnownBadCaseStillFails(t *testing.T) {
	env, _ := newEnvironment(t)
	ctx := context.Background()

	// The source the pre-plan render used: the arcade scene at the geometry the
	// audit recorded.
	source, err := env.Scaffold(ctx, "arcade", 1600, 1000, 7)
	require.NoError(t, err)

	style := catalog.Style{
		ID: "engraved-colonnade", Treatments: []string{"engraving"},
		TreatmentParams: map[string]string{"engraving": prePlanEngravingParams},
		Inks: map[string]string{
			"$brand.primary":    "#1f2937",
			"$brand.background": "#f4eedc",
		},
	}
	treated, err := imageengine.NewClient().Apply(ctx, source, style.Treatments,
		style.TreatmentParams, style.EffectivePalette(nil))
	require.NoError(t, err, "the pre-plan parameters must still be legal on the wire")

	src, err := decodePNG(source)
	require.NoError(t, err)
	out, err := decodePNG(treated)
	require.NoError(t, err)

	bar := style.EffectiveQuality()
	verdict := perceptual.Score(src, out, nil, perceptual.Thresholds{
		MinSubjectSurvival:     bar.MinSubjectSurvival,
		MinTonalOccupancy:      bar.MinTonalOccupancy,
		MinFrequencyModulation: bar.MinFrequencyModulation,
	})

	require.Falsef(t, verdict.Passed,
		"the gate accepted the very image this plan was written to stop shipping: %v", verdict.Metrics)
	t.Logf("known-bad verdict: %s", verdict.Error())

	// Naming which metric caught it keeps the record honest: if a later change
	// makes a *different* metric the one that fires, that is worth noticing.
	failed := make([]string, 0, len(verdict.Metrics))
	for _, m := range verdict.Failures() {
		failed = append(failed, m.Name)
	}
	require.NotEmpty(t, failed)
	t.Logf("caught by: %v", failed)
}

// TestTheRepairedStyleClearsItsBar is the other half. Without it, a gate that
// rejects everything would pass the test above and look healthy.
func TestTheRepairedStyleClearsItsBar(t *testing.T) {
	env, _ := newEnvironment(t)
	ctx := context.Background()

	source, err := env.Scaffold(ctx, "arcade", 1600, 1000, 7)
	require.NoError(t, err)

	style := catalog.Style{
		ID: "engraved-colonnade", Treatments: []string{"engraving"},
		TreatmentParams: map[string]string{
			"engraving": `{"spacing_rel":0.0153,"normalize":true,"dark":"$brand.primary","light":"$brand.background"}`,
		},
		Inks: map[string]string{
			"$brand.primary":    "#1f2937",
			"$brand.background": "#f4eedc",
		},
	}
	treated, err := imageengine.NewClient().Apply(ctx, source, style.Treatments,
		style.TreatmentParams, style.EffectivePalette(nil))
	require.NoError(t, err)

	src, err := decodePNG(source)
	require.NoError(t, err)
	out, err := decodePNG(treated)
	require.NoError(t, err)

	bar := style.EffectiveQuality()
	verdict := perceptual.Score(src, out, nil, perceptual.Thresholds{
		MinSubjectSurvival:     bar.MinSubjectSurvival,
		MinTonalOccupancy:      bar.MinTonalOccupancy,
		MinFrequencyModulation: bar.MinFrequencyModulation,
	})
	require.Truef(t, verdict.Passed, "the repaired style must clear its own bar: %s", verdict.Error())
}

func decodePNG(data []byte) (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}
