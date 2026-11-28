package evidence

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEveryEvidenceArtifactHasAProducingCommand(t *testing.T) {
	root := t.TempDir()
	t.Setenv("BACKDROP_STUDIO_EVIDENCE_DIR", root)
	path, err := OutputPath("catalog", "sheet-screens.png")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("capture"), 0o644))
	_, doc, err := Load(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	orphans, err := Unreferenced(root, doc)
	require.NoError(t, err)
	require.Empty(t, orphans)
	require.NoError(t, os.WriteFile(filepath.Join(root, "undeclared.png"), []byte("capture"), 0o644))
	orphans, err = Unreferenced(root, doc)
	require.NoError(t, err)
	require.Equal(t, []string{"evidence/undeclared.png"}, orphans)
}

func TestOutputPathRequiresSafeLogicalNames(t *testing.T) {
	root := t.TempDir()
	t.Setenv("BACKDROP_STUDIO_EVIDENCE_DIR", root)
	got, err := OutputPath("catalog", "sheet.png")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "catalog", "sheet.png"), got)
	for _, part := range []string{"..", "", "/escape", "a/b", `a\b`} {
		_, err := OutputPath(part)
		require.Error(t, err)
	}
	t.Setenv("BACKDROP_STUDIO_EVIDENCE_DIR", "relative")
	_, err = OutputPath("file.png")
	require.Error(t, err)
}

func TestCoverageFormsAdmitTheRightArtifacts(t *testing.T) {
	doc := "" +
		"## Artifacts and their producing commands\n" +
		"| `evidence/render-matrix.md` | cmd | why |\n" +
		"| `evidence/catalog/sheet-*.png` | cmd | why |\n" +
		"| `evidence/perceptual/engraving-repair/` | cmd | why |\n"
	coverage := DeclaredCoverage(doc)
	require.Len(t, coverage, 3)

	admits := func(artifact string) bool {
		for _, c := range coverage {
			if c.covers(artifact) {
				return true
			}
		}
		return false
	}

	require.True(t, admits("evidence/render-matrix.md"), "an exact path admits itself")
	require.True(t, admits("evidence/catalog/sheet-screens.png"), "a glob admits a matching sibling")
	require.True(t, admits("evidence/perceptual/engraving-repair/README.md"), "a directory admits its contents")
	require.True(t, admits("evidence/perceptual/engraving-repair/nested/after.png"), "a directory admits nested contents")

	require.False(t, admits("evidence/catalog/verdicts.md"), "a glob must not admit an undeclared sibling")
	require.False(t, admits("evidence/scenes/sheet-horizon.png"),
		"a glob must not admit a same-named file in another directory")
	require.False(t, admits("evidence/procedural/glaze-mosaic.png"), "an undeclared artifact stays undeclared")
}

// A row may declare more than one artifact, and every one of them counts.
//
// The first version read only the first backticked path per row, so a row
// naming a directory of images and the README that indexes them covered the
// images and silently left the README undeclared — which this rule then
// reported as an unreproducible artifact. The narrowing that closes the
// original defect is the CELL, not the count: a path mentioned inside a
// producing command still cannot declare itself.
func TestARowMayDeclareSeveralArtifacts(t *testing.T) {
	doc := ArtifactTableHeading + "\n\n" +
		"| Artifact | Command | What it proves |\n|---|---|---|\n" +
		"| `evidence/plates/*.png` and `evidence/plates/README.md` | `make integration-evidence` | Both. |\n" +
		"| `evidence/one.md` | run `evidence/not-declared.md` | Only the first cell declares. |\n"

	patterns := map[string]bool{}
	for _, coverage := range DeclaredCoverage(doc) {
		patterns[coverage.Pattern] = true
	}
	require.True(t, patterns["evidence/plates/*.png"], "the first path in a row must be declared")
	require.True(t, patterns["evidence/plates/README.md"], "the second path in the same cell must be declared too")
	require.True(t, patterns["evidence/one.md"])
	require.False(t, patterns["evidence/not-declared.md"],
		"a path inside a producing command must never declare itself")
}
