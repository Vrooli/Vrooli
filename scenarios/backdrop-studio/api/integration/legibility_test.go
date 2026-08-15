//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"backdrop-studio/integration"
	"backdrop-studio/internal/legibility"
)

// Whether the catalog's declared copy is actually legible, measured through a
// really running image-tools.
//
// This lane exists because the unit suite cannot answer the question. Its fake
// executor applies one fixed duotone whatever chain it is handed, so a
// measurement of a treated style there measures the fake — a scrim, a halftone
// and a bloom all come back as the same picture. The first version of this
// measurement was taken that way and produced a confident catalog-wide number
// that was not about the catalog. The rule this repo already records applies to
// measurements as much as to renders: a style's exact bytes have to make a round
// trip through a running image-tools before anything is claimed about them.
func TestReservedCopyIsLegibleThroughTheRealEngine(t *testing.T) {
	env, _ := newEnvironment(t)
	ctx := context.Background()

	styles, err := env.Styles(ctx)
	require.NoError(t, err)
	surfaces, err := env.Surfaces(ctx)
	require.NoError(t, err)
	sort.Slice(styles, func(i, j int) bool { return styles[i].ID < styles[j].ID })

	type result struct {
		id     string
		ratio  float64
		passes bool
	}
	var measured []result
	var unrenderable []string

	for _, style := range styles {
		regions := overlayRegions(style)
		if len(regions) == 0 {
			continue
		}
		permitted := integration.PermittedSurfaces(style, surfaces)
		if len(permitted) == 0 {
			continue
		}
		surface := permitted[len(permitted)-1]
		for _, candidate := range permitted {
			if candidate.ID == "web.hero" {
				surface = candidate
				break
			}
		}
		job, submitErr := env.Submit(ctx, integration.SubmitOptions{
			StyleID: style.ID, Seed: 7, SurfaceID: surface.ID, Placement: style.Placements[0],
		})
		if submitErr != nil {
			// A style this host cannot render is noted, never failed: a missing
			// model is a dependency fact, not a legibility verdict.
			unrenderable = append(unrenderable, style.ID)
			continue
		}
		threshold := style.ContrastThreshold
		if threshold <= 0 {
			threshold = 4.5
		}
		verdict, measureErr := legibility.Measure(job.Candidates[0].ImagePNG, regions, threshold, "")
		require.NoErrorf(t, measureErr, "measure %q", style.ID)
		measured = append(measured, result{id: style.ID, ratio: verdict.MinimumRatio, passes: verdict.Passes})
	}

	require.NotEmpty(t, measured, "no style with an overlay region rendered; this lane would prove nothing")

	passing := 0
	var report strings.Builder
	report.WriteString("# Reserved-copy legibility\n\n")
	report.WriteString("Worst-pixel contrast inside each style's own declared overlay region, against\n")
	report.WriteString("its own declared text colour and threshold, measured on the picture a really\n")
	report.WriteString("running `image-tools` produced.\n\n")
	report.WriteString("**Reproduce:** `make integration-evidence` from `scenarios/backdrop-studio`.\n\n")
	report.WriteString("| Style | Worst ratio | Threshold | Verdict |\n|---|---|---|---|\n")
	for _, r := range measured {
		verdict := "**fails**"
		if r.passes {
			passing++
			verdict = "passes"
		}
		report.WriteString(fmt.Sprintf("| `%s` | %.2f | 4.50 | %s |\n", r.id, r.ratio, verdict))
	}
	report.WriteString(fmt.Sprintf("\n**%d of %d pass.**\n", passing, len(measured)))
	if len(unrenderable) > 0 {
		report.WriteString("\nNot measured — did not render on this host: `" + strings.Join(unrenderable, "`, `") + "`.\n")
	}
	t.Logf("\n%s", report.String())

	if os.Getenv("BACKDROP_STUDIO_WRITE_EVIDENCE") != "" {
		dir := filepath.Join("..", "..", "docs", "evidence", "legibility")
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "reserved-copy.md"), []byte(report.String()), 0o644))
	}
}

func overlayRegions(style integration.Style) []legibility.Region {
	out := make([]legibility.Region, 0, len(style.Regions))
	for _, region := range style.Regions {
		if region.Kind != "overlay" || strings.TrimSpace(region.TextColor) == "" {
			continue
		}
		out = append(out, legibility.Region{
			X: region.X, Y: region.Y, Width: region.Width, Height: region.Height,
			Kind: region.Kind, TextColor: region.TextColor,
		})
	}
	return out
}
