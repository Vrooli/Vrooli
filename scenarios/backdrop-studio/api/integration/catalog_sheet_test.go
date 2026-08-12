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

	"backdrop-studio/integration"
	"backdrop-studio/internal/render"

	"github.com/stretchr/testify/require"
)

// The catalog contact sheets.
//
// `starter-catalog.md` states the stakes: "the catalog is the product. A thin or
// incoherent starter set makes the whole scenario look like a toy on first run,
// and first run is where the judgement gets made." Metrics cannot answer that
// question. A perceptual gate proves a treatment did not destroy its subject; it
// says nothing about whether forty styles read as one system or as a pile, and
// whether any given one is worth putting on a landing page.
//
// So the sheets exist to be looked at, by a person, grouped the way the
// judgement is actually made: everything that fails the same way, side by side.
// The written verdicts they produce live in docs/evidence/catalog/verdicts.md.
//
//	make integration-evidence

// sheetFamily groups styles by how their treatment chain fails, which is the
// same grouping the perceptual thresholds use. `untreated` is its own group
// because a `procedural` style is judged on the generator alone.
type sheetFamily struct {
	name  string
	match func(integration.Style) bool
}

var screenOps = map[string]bool{
	"halftone": true, "line_screen": true, "stipple": true, "engraving": true,
	"ascii_mosaic": true, "dither_ordered": true, "dither_diffusion": true,
}

var opticalOps = map[string]bool{
	"bloom": true, "defocus": true, "motion_blur": true, "aberration": true,
	"displacement": true, "pixel_sort": true,
}

var sheetFamilies = []sheetFamily{
	{"untreated", func(s integration.Style) bool { return len(s.Treatments) == 0 }},
	{"model-backed", func(s integration.Style) bool { return s.ModelBacked() }},
	{"screen", func(s integration.Style) bool { return anyOp(s.Treatments, screenOps) }},
	{"optical", func(s integration.Style) bool { return anyOp(s.Treatments, opticalOps) }},
	{"tonal", func(integration.Style) bool { return true }},
}

func anyOp(treatments []string, set map[string]bool) bool {
	for _, t := range treatments {
		if set[t] {
			return true
		}
	}
	return false
}

// familyOf returns the first matching family, so the list above reads as a
// priority order: a chain that both screens and blurs is judged as a screen,
// because that is the failure mode that decides whether it is usable.
func familyOf(s integration.Style) string {
	for _, f := range sheetFamilies {
		if f.match(s) {
			return f.name
		}
	}
	return "tonal"
}

func TestCatalogContactSheetEvidence(t *testing.T) {
	if os.Getenv("BACKDROP_STUDIO_WRITE_EVIDENCE") == "" {
		t.Skip("set BACKDROP_STUDIO_WRITE_EVIDENCE=1 to regenerate docs/evidence/catalog/")
	}
	env, _ := newEnvironment(t)
	ctx := context.Background()

	styles, err := env.Styles(ctx)
	require.NoError(t, err)
	surfaces, err := env.Surfaces(ctx)
	require.NoError(t, err)

	dir := filepath.Join("..", "..", "docs", "evidence", "catalog")
	require.NoError(t, os.MkdirAll(dir, 0o755))

	type cell struct {
		style integration.Style
		raw   []byte
		note  string
	}
	byFamily := map[string][]cell{}
	skipped := map[string]string{}

	for _, style := range styles {
		permitted := integration.PermittedSurfaces(style, surfaces)
		require.NotEmptyf(t, permitted, "style %q can be delivered to no surface", style.ID)
		// The landing-page hero when the style permits it, the largest
		// permitted surface otherwise.
		//
		// Taking the largest unconditionally put every hero style on the sheet
		// in 4:5 portrait, because a social post card is taller than a hero is
		// wide — so a colonnade meant to run across a header was judged
		// cropped to a phone. A sheet exists to show a style doing its job.
		surface := permitted[len(permitted)-1]
		for _, candidate := range permitted {
			if candidate.ID == "web.hero" {
				surface = candidate
				break
			}
		}

		job, submitErr := env.Submit(ctx, integration.SubmitOptions{
			StyleID:   style.ID,
			Seed:      7,
			SurfaceID: surface.ID,
			Placement: style.Placements[0],
		})
		if submitErr != nil {
			// A model-backed style that will not render is noted, never failed,
			// and the reason is printed verbatim.
			//
			// The gate for those lives in the render lane, which knows how to
			// tell a capacity limit from a defect. This test only makes
			// pictures, and it saw the distinction blur: a VAE allocation
			// failure at hero aspect reached it as a bare `EOF` — the
			// connection dropped rather than an error returned — which no
			// capacity matcher recognises and which is indistinguishable here
			// from a crash. Failing the evidence producer on it would report a
			// host limit as a broken catalog.
			if style.ModelBacked() {
				skipped[style.ID] = "not rendered on this host: " + submitErr.Error()
				continue
			}
			t.Errorf("style %q failed to render for the contact sheet: %v", style.ID, submitErr)
			skipped[style.ID] = "FAILED"
			continue
		}
		require.NotEmptyf(t, job.Candidates, "style %q rendered no candidate", style.ID)
		raw := job.Candidates[0].ImagePNG
		_, _, decodeErr := integration.DecodePNG(raw)
		require.NoErrorf(t, decodeErr, "style %q returned bytes that are not a PNG", style.ID)

		chain := strings.Join(style.Treatments, "+")
		if chain == "" {
			chain = "untreated"
		}
		family := familyOf(style)
		byFamily[family] = append(byFamily[family], cell{
			style: style,
			raw:   raw,
			note:  fmt.Sprintf("%s  -  %s  -  %s  -  %dx%d", style.ID, style.Subject, chain, surface.Width, surface.Height),
		})
	}

	families := make([]string, 0, len(byFamily))
	for name := range byFamily {
		families = append(families, name)
	}
	sort.Strings(families)

	for _, family := range families {
		cells := byFamily[family]
		sort.Slice(cells, func(i, j int) bool { return cells[i].style.ID < cells[j].style.ID })
		sheet := make([]render.CatalogCell, 0, len(cells))
		for _, c := range cells {
			sheet = append(sheet, render.CatalogCell{Caption: c.note, PNG: c.raw})
		}
		encoded, sheetErr := render.CatalogSheet(sheet, 3)
		require.NoError(t, sheetErr)
		path := filepath.Join(dir, "sheet-"+family+".png")
		require.NoError(t, os.WriteFile(path, encoded, 0o644))
		t.Logf("wrote %s (%d styles)", path, len(cells))
	}
	for id, why := range skipped {
		t.Logf("not on a sheet: %s — %s", id, why)
	}
}
