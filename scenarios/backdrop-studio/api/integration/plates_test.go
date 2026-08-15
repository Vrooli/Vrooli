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
	"backdrop-studio/internal/vector"
)

// Every style that declares a plate stack really ships one, through a running
// image-tools compositor.
//
// The unit suite proves the wiring against a fake compositor; this proves the
// picture survives the real one. They are different claims: a compositor that
// returned its first plate unchanged would pass every structural assertion and
// deliver a backdrop missing two layers.
func TestDeclaredPlateStacksShipThroughTheRealCompositor(t *testing.T) {
	env, _ := newEnvironment(t)
	ctx := context.Background()

	styles, err := env.Styles(ctx)
	require.NoError(t, err)
	surfaces, err := env.Surfaces(ctx)
	require.NoError(t, err)

	layered := 0
	for _, style := range styles {
		if len(style.PlateSpec) < 2 {
			continue
		}
		layered++
		t.Run(style.ID, func(t *testing.T) {
			permitted := integration.PermittedSurfaces(style, surfaces)
			require.NotEmpty(t, permitted)
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
			require.NoErrorf(t, submitErr, "style %q declares %d plates and did not render", style.ID, len(style.PlateSpec))
			require.NotEmpty(t, job.Candidates)
			candidate := job.Candidates[0]

			require.Lenf(t, candidate.Plates, len(style.PlateSpec),
				"style %q declares %d plates and shipped %d", style.ID, len(style.PlateSpec), len(candidate.Plates))
			require.NotEmpty(t, candidate.ImagePNG, "the flat composite is never optional")
			require.Equal(t, surface.Width, candidate.Width)
			require.Equal(t, surface.Height, candidate.Height)

			// Depths are distinct and ordered, which is what makes the stack a
			// stack rather than a list.
			seen := map[int32]bool{}
			for i, plate := range candidate.Plates {
				require.NotEmptyf(t, plate.Name, "plate %d has no name", i)
				require.Falsef(t, seen[plate.Depth], "two plates at depth %d", plate.Depth)
				seen[plate.Depth] = true
				if i > 0 {
					require.Greaterf(t, plate.Depth, candidate.Plates[i-1].Depth,
						"plates arrive out of depth order: %v", plateDepths(candidate.Plates))
				}
			}
		})
	}
	require.Positive(t, layered, "no seeded style declares a plate stack; this lane would prove nothing")
}

// The plane evidence sheet: each layered style's plates beside its composite.
//
// A stack nobody can look at is a stack nobody reviews. The sheet is what makes
// "the colonnade separates its canopy from its arcade" a claim a reader can
// check rather than one they take on trust.
func TestPlaneSheetEvidence(t *testing.T) {
	if os.Getenv("BACKDROP_STUDIO_WRITE_EVIDENCE") == "" {
		t.Skip("set BACKDROP_STUDIO_WRITE_EVIDENCE=1 to regenerate docs/evidence/plates/")
	}
	env, _ := newEnvironment(t)
	ctx := context.Background()

	styles, err := env.Styles(ctx)
	require.NoError(t, err)
	surfaces, err := env.Surfaces(ctx)
	require.NoError(t, err)

	dir := filepath.Join("..", "..", "docs", "evidence", "plates")
	require.NoError(t, os.MkdirAll(dir, 0o755))

	var index strings.Builder
	index.WriteString("# Plate stacks\n\n")
	index.WriteString("Every style whose generator separates depth planes: the composite that ships,\n")
	index.WriteString("and the stack it was assembled from.\n\n")
	index.WriteString("The plate images are deliberately NOT here. Plate pixels do not travel on the\n")
	index.WriteString("job record — a three-plate candidate at store geometry is tens of megabytes,\n")
	index.WriteString("and inlining them would make every list call expensive for a field most\n")
	index.WriteString("callers ignore — so this lane can only show what the wire carries. What it\n")
	index.WriteString("proves is that the declared stack really reached a running compositor and came\n")
	index.WriteString("back as one picture at the delivery geometry.\n\n")
	index.WriteString("The alpha behind each plate is exact rather than estimated: a plate is the\n")
	index.WriteString("generator's own layer, and `internal/vector`'s partition test proves every mark\n")
	index.WriteString("lands in exactly one of them.\n\n")
	index.WriteString("**Reproduce:** `make integration-evidence` from `scenarios/backdrop-studio`.\n\n")

	rendered := 0
	for _, style := range styles {
		if len(style.PlateSpec) < 2 {
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
			index.WriteString(fmt.Sprintf("## `%s`\n\nDid not render on this host: %v\n\n", style.ID, submitErr))
			continue
		}
		candidate := job.Candidates[0]
		name := style.ID + "-composite.png"
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), candidate.ImagePNG, 0o644))
		rendered++

		index.WriteString(fmt.Sprintf("## `%s`\n\nAt `%s` (%dx%d), seed 7.\n\n", style.ID, surface.ID, surface.Width, surface.Height))
		index.WriteString(fmt.Sprintf("![composite](%s)\n\n", name))
		index.WriteString("| Plate | Depth | Blend | Opacity | Treatments |\n|---|---|---|---|---|\n")
		for _, plate := range candidate.Plates {
			treatments := "—"
			if len(plate.Treatments) > 0 {
				treatments = "`" + strings.Join(plate.Treatments, "` → `") + "`"
			}
			index.WriteString(fmt.Sprintf("| `%s` | %d | %s | %.2f | %s |\n",
				plate.Name, plate.Depth, plate.Blend, plate.Opacity, treatments))
		}
		index.WriteString("\n")
	}
	require.Positive(t, rendered, "no layered style rendered; the sheet would claim a capability nothing proved")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte(index.String()), 0o644))
}

func plateDepths(plates []integration.Plate) []int {
	out := make([]int, 0, len(plates))
	for _, plate := range plates {
		out = append(out, int(plate.Depth))
	}
	sort.Ints(out)
	return out
}

// Flatten-equivalence, measured on pixels.
//
// The unit suite proves the plane documents PARTITION the composite — every
// mark in exactly one layer, nothing duplicated or dropped. That is a claim
// about markup. This is the claim about the picture: rasterizing each plane and
// compositing them normal-over-normal must produce what rasterizing the whole
// document produces.
//
// Both halves matter and neither implies the other. A correct partition could
// still composite wrong if the compositor's alpha arithmetic were off, and a
// correct composite could hide a mark drawn twice in the same place.
func TestFlattenedPlanesMatchTheWholeDocument(t *testing.T) {
	env, _ := newEnvironment(t)
	ctx := context.Background()

	inks := map[string]string{
		vector.InkPaper:  "#efe7d3",
		vector.InkInk:    "#12327a",
		vector.InkAccent: "#c9432f",
	}
	const w, h = 720, 360

	for _, preset := range vector.Presets {
		t.Run(preset, func(t *testing.T) {
			drawn, err := vector.Render(vector.Request{
				Preset: preset, Width: w, Height: h, Seed: 7, Inks: inks,
			})
			require.NoError(t, err)
			require.Greater(t, len(drawn.Planes), 1, "a one-plane generator proves nothing here")

			whole, err := env.Rasterize(ctx, drawn.SVG)
			require.NoError(t, err, "rasterize the whole document")

			plates := make([]integration.CompositePlate, 0, len(drawn.Planes))
			for i, plane := range drawn.Planes {
				pixels, planeErr := env.Rasterize(ctx, drawn.PlaneDocuments[i])
				require.NoErrorf(t, planeErr, "rasterize plane %q", plane)
				plates = append(plates, integration.CompositePlate{
					Name: plane, Depth: i, Blend: "normal", Opacity: 1, PNG: pixels,
				})
			}
			flattened, err := env.Composite(ctx, plates, w, h)
			require.NoError(t, err, "composite the planes")

			// Compared as pixels rather than bytes: the two paths encode
			// through different code — one PNG comes back from `convert`, the
			// other from `composite` — so equal bytes was never the available
			// claim. Equal PIXELS is, and it is the one that matters.
			difference := integration.MeanPixelDifference(t, whole, flattened)
			require.LessOrEqualf(t, difference, 1.0/255.0,
				"%s: flattening its planes differs from the whole document by %.4f mean channel error; the partition or the compositor is wrong",
				preset, difference)
		})
	}
}
