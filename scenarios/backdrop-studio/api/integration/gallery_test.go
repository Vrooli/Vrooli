//go:build integration

package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"backdrop-studio/integration"
	"backdrop-studio/internal/imageengine"

	"github.com/stretchr/testify/require"
)

// galleryTreatments is every treatment the catalog can name, in the order the
// taxonomy groups them. The list is explicit rather than derived from the
// operation registry because the gallery is art direction: it shows what THIS
// scenario's defaults produce, not what image-tools is capable of.
var galleryTreatments = []string{
	// Reprographic and quantization.
	"duotone", "posterize", "halftone", "dither_ordered", "dither_diffusion",
	"line_screen", "stipple", "engraving",
	// Optical and device.
	"grain", "scrim", "aberration", "bloom", "curve", "defocus", "motion_blur",
	// Displacement and symbolic.
	"displacement", "pixel_sort", "ascii_mosaic",
}

// galleryGeometry is `web.hero`. The gallery is art-directed against the
// surface the catalog is art-directed against, and the relative parameters in
// `imageengine.spatialDefaults` quote their pixel equivalents at this size.
const (
	galleryWidth  = 1440
	galleryHeight = 720
	gallerySeed   = 7
)

// TestTreatmentGalleryEvidence is the producer of `docs/evidence/treatments/`.
//
// The gallery was previously made by hand, which meant that when Phase 4 moved
// every spatial default from pixels to a fraction of the short edge, eighteen
// committed images silently became pictures of parameters no longer in the
// code. Evidence that cannot be regenerated cannot be trusted after the first
// change that touches it.
//
// It runs each treatment through the same `imageengine.Client` the render path
// uses, against the really running image-tools, so the parameters shown are the
// scenario's own defaults resolved by the real merge — there is no second copy
// of the defaults here to drift from the first.
//
//	make integration-evidence
func TestTreatmentGalleryEvidence(t *testing.T) {
	env, _ := newEnvironment(t)
	ctx := context.Background()

	source, err := env.Scaffold(ctx, "horizon", galleryWidth, galleryHeight, gallerySeed)
	require.NoError(t, err)
	sw, sh, err := integration.DecodePNG(source)
	require.NoError(t, err)
	require.Equal(t, [2]int{galleryWidth, galleryHeight}, [2]int{sw, sh})

	client := imageengine.NewClient()
	write := os.Getenv(writeEvidenceEnv) != ""
	dir := filepath.Join("..", "..", "docs", "evidence", "treatments")

	for _, op := range galleryTreatments {
		t.Run(op, func(t *testing.T) {
			// A nil palette is the cold-install case: the treatment renders
			// from the scenario's declared default inks, which is what a
			// reader of the gallery should be looking at.
			treated, applyErr := client.Apply(ctx, source, []string{op}, nil, nil)
			require.NoErrorf(t, applyErr, "%s failed through the real wire", op)

			w, h, decodeErr := integration.DecodePNG(treated)
			require.NoErrorf(t, decodeErr, "%s returned bytes that are not a PNG", op)
			require.Equalf(t, [2]int{galleryWidth, galleryHeight}, [2]int{w, h},
				"%s changed the geometry; a treatment must screen the picture, not resize it", op)
			require.NotEqualf(t, source, treated, "%s returned its input unchanged", op)

			if !write {
				return
			}
			require.NoError(t, os.WriteFile(filepath.Join(dir, op+".png"), treated, 0o644))
		})
	}
}
