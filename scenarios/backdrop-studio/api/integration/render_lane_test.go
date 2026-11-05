//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"

	"backdrop-studio/integration"

	"github.com/stretchr/testify/require"
)

// laneSeed is fixed so a failure is reproducible by hand with the same command
// a reader would type.
const laneSeed = 7

// brandEnv names the brand the bound-palette pass uses. It is an environment
// variable rather than a constant because brand ids are per-install; when it is
// unset the bound pass binds tokens directly, which exercises the same code
// path without requiring a seeded brand.
const brandEnv = "BACKDROP_STUDIO_LANE_BRAND"

func newEnvironment(t *testing.T) (integration.Environment, integration.Capabilities) {
	t.Helper()
	ctx := context.Background()

	env, err := integration.Resolve(ctx)
	require.NoError(t, err, "the lane needs the real scenarios running; this is a dependency failure, not a render failure")

	report, err := env.AssertFreshBinary(ctx)
	require.NoError(t, err)
	t.Logf("running backdrop-studio %s (catalog seed v%d applied)", report.Version, report.AppliedSeedVersion)

	caps, err := env.ProbeCapabilities(ctx)
	require.NoError(t, err)
	t.Logf("image-tools: %d operations, %d installed image models %v, generation=%t controlnet=%t",
		len(caps.Operations), len(caps.ImageModels), caps.ImageModels, caps.GenerationOK, caps.ControlNetOK)

	return env, caps
}

// TestEverySeededStyleRendersThroughRunningImageTools is the assertion whose
// absence let twelve broken styles ship past a green suite.
//
// Every seeded style is rendered twice: once with nothing bound, which is what
// a CLI caller has, and once with a palette bound. The unbound pass is the one
// that matters — it is the exact path the old contract test did not cover.
func TestEverySeededStyleRendersThroughRunningImageTools(t *testing.T) {
	env, caps := newEnvironment(t)
	ctx := context.Background()

	styles, err := env.Styles(ctx)
	require.NoError(t, err)
	t.Logf("catalog holds %d styles", len(styles))

	// A brand that is deliberately different from every seeded default, so a
	// style that ignores the palette fails rather than passing by coincidence.
	boundTokens := map[string]string{
		"$brand.primary":    "#7A1F2B",
		"$brand.secondary":  "#3B0D14",
		"$brand.accent":     "#E8B04B",
		"$brand.background": "#F7F1E3",
		"$brand.surface":    "#FFFFFF",
		"$brand.text":       "#1A1A1A",
		"$brand.error":      "#B00020",
	}
	brandID := strings.TrimSpace(os.Getenv(brandEnv))

	passes := []struct {
		name   string
		submit func(integration.Style) integration.SubmitOptions
	}{
		{
			name: "unbound",
			submit: func(s integration.Style) integration.SubmitOptions {
				return integration.SubmitOptions{StyleID: s.ID, Seed: laneSeed}
			},
		},
		{
			name: "bound",
			submit: func(s integration.Style) integration.SubmitOptions {
				opts := integration.SubmitOptions{StyleID: s.ID, Seed: laneSeed}
				if brandID != "" {
					opts.BrandID = brandID
					return opts
				}
				opts.BrandTokens = boundTokens
				return opts
			},
		},
	}

	skipped := 0
	for _, pass := range passes {
		for _, style := range styles {
			style := style
			t.Run(fmt.Sprintf("%s/%s", pass.name, style.ID), func(t *testing.T) {
				if style.ModelBacked() && !caps.GenerationOK {
					// A skip is only honest when it names what is missing and
					// is visible in the output. It must never read as a pass.
					skipped++
					t.Skipf("SKIP(no-image-model): style %q is %s and image-tools reports no enabled, installed model serving text_to_image or image_to_image", style.ID, style.Strategy)
				}

				job, err := env.Submit(ctx, pass.submit(style))
				if integration.IsGPUCapacityFailure(err) {
					skipped++
					t.Skipf("SKIP(gpu-capacity): style %q reached the model but the host could not allocate device memory; other residents hold the card. Re-run when the GPU is free.", style.ID)
				}
				require.NoErrorf(t, err, "style %q failed to render through the running image-tools", style.ID)
				require.Equal(t, "completed", job.Status)
				require.NotEmpty(t, job.Candidates, "style %q produced no candidate", style.ID)

				for _, candidate := range job.Candidates {
					width, height, decodeErr := integration.DecodePNG(candidate.ImagePNG)
					require.NoErrorf(t, decodeErr, "style %q candidate %s", style.ID, candidate.ID)

					// A candidate that reports a size it does not have makes
					// every downstream size decision wrong, and nothing else in
					// the system can notice.
					require.Equalf(t, int(candidate.Width), width,
						"style %q records width %d but its bytes are %dpx wide", style.ID, candidate.Width, width)
					require.Equalf(t, int(candidate.Height), height,
						"style %q records height %d but its bytes are %dpx tall", style.ID, candidate.Height, height)

					// An unresolved slot anywhere in the response means an ink
					// never bound and a literal reached the wire.
					require.NotContainsf(t, candidate.ProvenanceJSON, "$brand.",
						"style %q leaked an unresolved brand slot into provenance", style.ID)
				}
			})
		}
	}
	if skipped > 0 {
		t.Logf("%d model-backed assertions skipped with a named reason", skipped)
	}
}

// TestLaneFailsWhenImageToolsIsUnavailable proves the lane reports a missing
// dependency as a missing dependency. A lane that answers "no styles render"
// when image-tools is simply stopped teaches the reader the wrong thing.
func TestLaneFailsWhenImageToolsIsUnavailable(t *testing.T) {
	env, _ := newEnvironment(t)
	ctx := context.Background()

	// Point at a port nothing serves, which is what an unreachable image-tools
	// looks like from inside a render.
	broken := env
	broken.ImageToolsURL = "http://127.0.0.1:1"
	_, err := broken.ProbeCapabilities(ctx)
	require.Error(t, err, "an unreachable image-tools must be an error, never an empty capability set that reads as 'nothing is supported'")
	require.Contains(t, err.Error(), "list image-tools operations")
}

// TestBoundBrandChangesTheRenderedBytes proves the palette is load-bearing
// end to end. The unit-level version of this check asserts on parameter JSON;
// this one asserts on pixels, which is the only claim that matters.
func TestBoundBrandChangesTheRenderedBytes(t *testing.T) {
	env, _ := newEnvironment(t)
	ctx := context.Background()

	const style = "cyanotype-arcade"
	unbound, err := env.Submit(ctx, integration.SubmitOptions{StyleID: style, Seed: laneSeed})
	require.NoError(t, err)
	bound, err := env.Submit(ctx, integration.SubmitOptions{
		StyleID:     style,
		Seed:        laneSeed,
		BrandTokens: map[string]string{"$brand.primary": "#7A1F2B", "$brand.background": "#F7F1E3"},
	})
	require.NoError(t, err)

	require.NotEmpty(t, unbound.Candidates)
	require.NotEmpty(t, bound.Candidates)
	require.NotEqual(t, unbound.Candidates[0].ImagePNG, bound.Candidates[0].ImagePNG,
		"binding a brand must change the rendered pixels; if it does not, the palette never reached image-tools")
}

// TestDeterministicSeedProducesIdenticalBytes pins the reproducibility claim the
// product makes. A seed that does not reproduce makes every committed evidence
// artifact unverifiable.
func TestDeterministicSeedProducesIdenticalBytes(t *testing.T) {
	env, _ := newEnvironment(t)
	ctx := context.Background()

	const style = "stipple-massif"
	first, err := env.Submit(ctx, integration.SubmitOptions{StyleID: style, Seed: laneSeed})
	require.NoError(t, err)
	second, err := env.Submit(ctx, integration.SubmitOptions{StyleID: style, Seed: laneSeed})
	require.NoError(t, err)

	require.NotEmpty(t, first.Candidates)
	require.NotEmpty(t, second.Candidates)
	require.Equal(t, first.Candidates[0].ImagePNG, second.Candidates[0].ImagePNG,
		"the same style at the same seed must produce the same bytes")
}

// solidPNG builds a small gradient source for seam probes. A flat fill would
// pass a tonal treatment trivially; a gradient exercises the ink ramp.
func solidPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			v := uint8(255 * x / max(1, width-1))
			img.SetNRGBA(x, y, color.NRGBA{R: v, G: v, B: v, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// TestEveryStyleRendersAtItsSmallestAndLargestSurface is the two-geometry sweep.
//
// One geometry is not enough, and the reason is specific: a delivery constant
// of 1600x1000 satisfied a single-geometry check perfectly while surfaces
// declared 1440x720, 390x844, 1024x500 and 1290x2796. Every store asset was
// produced at the wrong aspect and cropped, and no test could see it because no
// test ever asked for a second size.
func TestEveryStyleRendersAtItsSmallestAndLargestSurface(t *testing.T) {
	env, caps := newEnvironment(t)
	ctx := context.Background()

	styles, err := env.Styles(ctx)
	require.NoError(t, err)
	surfaces, err := env.Surfaces(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, surfaces)

	for _, style := range styles {
		style := style
		permitted := integration.PermittedSurfaces(style, surfaces)
		require.NotEmptyf(t, permitted, "style %q declares placements no seeded surface permits", style.ID)

		targets := []integration.Surface{permitted[0]}
		if len(permitted) > 1 {
			targets = append(targets, permitted[len(permitted)-1])
		}

		for _, surface := range targets {
			t.Run(fmt.Sprintf("%s/%s", style.ID, surface.ID), func(t *testing.T) {
				if style.ModelBacked() && !caps.GenerationOK {
					t.Skipf("SKIP(no-image-model): style %q is %s and no installed model serves generation", style.ID, style.Strategy)
				}
				job, submitErr := env.Submit(ctx, integration.SubmitOptions{
					StyleID:   style.ID,
					Seed:      laneSeed,
					SurfaceID: surface.ID,
				})
				if integration.IsGPUCapacityFailure(submitErr) {
					t.Skipf("SKIP(gpu-capacity): style %q reached the model but the host could not allocate device memory", style.ID)
				}
				require.NoErrorf(t, submitErr, "style %q at surface %q", style.ID, surface.ID)
				require.Equal(t, surface.ID, job.SurfaceID, "the job must echo the surface it resolved")
				require.NotEmpty(t, job.Candidates)

				width, height, decodeErr := integration.DecodePNG(job.Candidates[0].ImagePNG)
				require.NoError(t, decodeErr)

				// The delivery geometry is the surface's declared geometry.
				// Anything else means a constant is still in the path.
				require.Equalf(t, int(surface.Width), width,
					"style %q at surface %q produced %dpx wide, but the surface declares %d", style.ID, surface.ID, width, surface.Width)
				require.Equalf(t, int(surface.Height), height,
					"style %q at surface %q produced %dpx tall, but the surface declares %d", style.ID, surface.ID, height, surface.Height)
			})
		}
	}
}
