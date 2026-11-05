//go:build integration

package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"backdrop-studio/integration"
	"backdrop-studio/internal/imageengine"

	"github.com/stretchr/testify/require"
)

// TestSeamsDeclaredInSeamsDocHaveARealRun keeps the seam registry honest.
//
// `docs/internal/SEAMS.md` is load-bearing documentation: it is what a future
// agent reads to decide whether a boundary is tested. A seam listed there with
// only a fake is an untested assumption wearing an interface, and both defects
// this scenario has shipped were exactly that. This asserts the document says
// so for every entry.
func TestSeamsDeclaredInSeamsDocHaveARealRun(t *testing.T) {
	root, err := filepath.Abs("../../docs/internal/SEAMS.md")
	require.NoError(t, err)
	raw, err := os.ReadFile(root)
	require.NoError(t, err)
	doc := string(raw)

	// The seams whose real implementation this lane exercises. Each must both
	// appear in the registry and declare where its real run happens.
	for _, seam := range []string{
		"imageengine.Executor",
		"imageengine.Generator",
		"release.Publisher",
	} {
		require.Containsf(t, doc, "### "+seam, "seam %q is exercised by the integration lane but is not in SEAMS.md", seam)
	}

	// Every seam section that names a real run must name where it happens.
	sections := strings.Split(doc, "\n### ")
	realRuns := 0
	for _, section := range sections[1:] {
		if strings.Contains(section, "**Real run**") {
			realRuns++
			require.NotContainsf(t, section, "**Real run** | TBD",
				"a seam declares a real run without saying where: %.60s", section)
		}
	}
	require.GreaterOrEqual(t, realRuns, 3, "the raster seams must declare a real run")
}

// TestImageEngineExecutorAgainstRunningImageTools exercises the production
// Executor implementation — not the fake — against the real service.
//
// This is the seam whose fake accepted parameters image-tools rejects, which is
// how twelve unrenderable styles passed a green suite.
func TestImageEngineExecutorAgainstRunningImageTools(t *testing.T) {
	env, _ := newEnvironment(t)
	ctx := context.Background()

	client := imageengine.NewClient()
	client.Resolve = func(context.Context) (string, error) { return env.ImageToolsURL, nil }

	source := solidPNG(t, 64, 64)

	t.Run("resolved palette round-trips", func(t *testing.T) {
		out, err := client.Apply(ctx, imageengine.ApplyRequest{
			Input:      source,
			Treatments: []string{"duotone", "grain"},
			Params: map[string]string{
				"duotone": `{"dark":"$brand.primary","light":"$brand.background","normalize":true}`,
				"grain":   `{"amount":0.05,"contrast_multiplier":1.02,"seed":3}`,
			},
			Palette: map[string]string{"$brand.primary": "#1B3FBF", "$brand.background": "#F5EFDC"},
		})
		require.NoError(t, err, "the production executor must complete a real chain")
		width, height, decodeErr := integration.DecodePNG(out)
		require.NoError(t, decodeErr)
		require.Equal(t, 64, width)
		require.Equal(t, 64, height)
	})

	t.Run("unresolved slot never reaches the service", func(t *testing.T) {
		_, err := client.Apply(ctx, imageengine.ApplyRequest{
			Input:      source,
			Treatments: []string{"duotone"},
			Params:     map[string]string{"duotone": `{"dark":"$brand.primary","light":"#ffffff"}`},
		})
		var unresolved *imageengine.UnresolvedSlotError
		require.ErrorAs(t, err, &unresolved,
			"an unbindable slot must fail here, not as a 422 from image-tools")
	})

	t.Run("a parameter image-tools rejects is reported, not swallowed", func(t *testing.T) {
		// `screen_angle` is not a field on HalftoneParams. protojson rejects
		// unknown fields, so this must surface as an error naming the operation
		// rather than silently rendering defaults.
		_, err := client.Apply(ctx, imageengine.ApplyRequest{
			Input:      source,
			Treatments: []string{"halftone"},
			Params:     map[string]string{"halftone": `{"lpi":90,"screen_angle":15}`},
			Palette:    map[string]string{"$brand.primary": "#111827", "$brand.background": "#ffffff"},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "halftone")
	})
}
