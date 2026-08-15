package catalog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// evidenceEnv gates the artifact writers, so the normal suite stays read-only.
const evidenceEnv = "BACKDROP_STUDIO_WRITE_EVIDENCE"

func TestResemblanceIsOneForAStyleAgainstItself(t *testing.T) {
	store := NewStore(freshDB(t))
	ctx := context.Background()
	require.NoError(t, store.Seed(ctx))
	styles, err := store.ListStyles(ctx, "", "", "", "", "")
	require.NoError(t, err)
	require.NotEmpty(t, styles)

	for _, style := range styles {
		sig, err := ComputeSignature(style)
		require.NoErrorf(t, err, "signature for %q", style.ID)
		require.InDeltaf(t, 1.0, Resemble(sig, sig), 1e-9,
			"style %q does not resemble itself; the measure is broken, not the catalog", style.ID)
	}
}

// A recolour is not a divergence. This is the rule the operator stated for
// repairing a resemblance cluster, and it has to be a property of the measure
// rather than a note in a document, or the first cluster repaired by swapping
// inks will report as fixed.
func TestRecolouringDoesNotReduceResemblance(t *testing.T) {
	base := Style{
		ID: "base", Name: "Base", Role: "ambient", Subject: "non_representational",
		Lineage: "bauhaus", Strategy: "procedural-treated",
		Scaffold:   &ScaffoldBinding{Preset: "mesh", ParamsJSON: `{"palette":0}`},
		Treatments: []string{"halftone"},
		TreatmentParams: map[string]string{
			"halftone": `{"lpi":120,"dark":"#0f172a","light":"#e0f2fe"}`,
		},
		Placements: []string{"full_bleed"},
	}
	recoloured := base
	recoloured.ID = "recoloured"
	recoloured.TreatmentParams = map[string]string{
		"halftone": `{"lpi":120,"dark":"#7c2d12","light":"#fef3c7"}`,
	}

	a, err := ComputeSignature(base)
	require.NoError(t, err)
	b, err := ComputeSignature(recoloured)
	require.NoError(t, err)
	require.InDelta(t, 1.0, Resemble(a, b), 1e-9,
		"two styles differing only by ink read as one picture; the measure must say so")
}

// The other half: a genuinely different source must separate, or the measure
// reports every style as a cluster and tells a repair nothing.
func TestADifferentSourceSeparates(t *testing.T) {
	mesh := Style{
		ID: "mesh", Name: "Mesh", Role: "ambient", Subject: "non_representational",
		Lineage: "bauhaus", Strategy: "procedural",
		Scaffold: &ScaffoldBinding{Preset: "mesh", ParamsJSON: `{"palette":0}`}, Placements: []string{"full_bleed"},
	}
	terrain := mesh
	terrain.ID, terrain.Subject = "terrain", "geological"
	terrain.Scaffold = &ScaffoldBinding{Preset: "terrain"}

	a, err := ComputeSignature(mesh)
	require.NoError(t, err)
	b, err := ComputeSignature(terrain)
	require.NoError(t, err)
	require.Lessf(t, Resemble(a, b), 0.9,
		"a mesh gradient and a mountain range must not read as one picture (got %.3f)", Resemble(a, b))
}

// A model-backed style and a procedural one are never compared. Their
// descriptors have no common scale, and a number produced by comparing them
// would look like a measurement without being one.
func TestKindsAreNeverCompared(t *testing.T) {
	procedural := Style{
		ID: "procedural", Name: "P", Role: "ambient", Subject: "non_representational",
		Lineage: "bauhaus", Strategy: "procedural",
		Scaffold: &ScaffoldBinding{Preset: "mesh"}, Placements: []string{"full_bleed"},
	}
	modelBacked := Style{
		ID: "model", Name: "M", Role: "ambient", Subject: "interior",
		Lineage: "bauhaus", Strategy: "synthesized", Treatments: []string{"grain"},
		Generation: &GenerationBlock{PromptTemplate: "a sunlit modernist interior"},
		Placements: []string{"full_bleed"},
	}
	a, err := ComputeSignature(procedural)
	require.NoError(t, err)
	b, err := ComputeSignature(modelBacked)
	require.NoError(t, err)
	require.Zero(t, Resemble(a, b))
}

// TestResemblanceReportEvidence writes the nearest-neighbour table.
//
// It is a test rather than a hand-run probe for the reason EVIDENCE.md states:
// an artifact nobody can reproduce is a claim about a build nobody can
// identify. Set BACKDROP_STUDIO_WRITE_EVIDENCE=1 to regenerate.
func TestResemblanceReportEvidence(t *testing.T) {
	store := NewStore(freshDB(t))
	ctx := context.Background()
	require.NoError(t, store.Seed(ctx))
	styles, err := store.ListStyles(ctx, "", "", "", "", "")
	require.NoError(t, err)

	report, err := BuildReport(styles)
	require.NoError(t, err)
	require.NotEmpty(t, report.Pairs)

	clusters, err := Clusters(styles, FamilyResemblanceThreshold)
	require.NoError(t, err)

	var out strings.Builder
	out.WriteString("# Family resemblance\n\n")
	out.WriteString("Each style's nearest neighbour in the settled catalog, measured on the\n")
	out.WriteString("arrangement of its source and the treatment chain over it. Colour is\n")
	out.WriteString("deliberately excluded: a recolour is not a divergence.\n\n")
	out.WriteString("**Reproduce:** `BACKDROP_STUDIO_WRITE_EVIDENCE=1 GOWORK=off go test ./internal/catalog/ -run TestResemblanceReportEvidence` from `scenarios/backdrop-studio/api`.\n\n")
	out.WriteString(fmt.Sprintf("Cluster threshold: **%.2f**. Styles at or above it read as one picture.\n\n", FamilyResemblanceThreshold))

	if len(clusters) == 0 {
		out.WriteString("## Clusters\n\nNone. No two styles in the settled catalog reach the threshold.\n\n")
	} else {
		out.WriteString("## Clusters\n\n| Members | Strongest resemblance |\n|---|---|\n")
		for _, cluster := range clusters {
			out.WriteString(fmt.Sprintf("| `%s` | %.3f |\n", strings.Join(cluster.StyleIDs, "`, `"), cluster.Resemblance))
		}
		out.WriteString("\n")
	}

	out.WriteString("## Nearest neighbour, every style\n\n")
	out.WriteString("| Style | Nearest | Kind | Source | Chain | Resemblance |\n|---|---|---|---|---|---|\n")
	for _, pair := range report.Pairs {
		out.WriteString(fmt.Sprintf("| `%s` | `%s` | %s | %.3f | %.3f | %.3f |\n",
			pair.StyleID, pair.NearestID, pair.Kind, pair.Source, pair.ChainSim, pair.Resemblance))
	}
	if len(report.Unmeasured) > 0 {
		out.WriteString("\n## Unmeasured\n\nNo comparable neighbour of the same kind: ")
		out.WriteString("`" + strings.Join(report.Unmeasured, "`, `") + "`.\n")
	}

	t.Logf("resemblance report:\n%s", out.String())
	if os.Getenv(evidenceEnv) == "" {
		t.Skipf("set %s=1 to write docs/evidence/catalog/resemblance.md", evidenceEnv)
	}
	dir := filepath.Join("..", "..", "..", "docs", "evidence", "catalog")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "resemblance.md"), []byte(out.String()), 0o644))
}
