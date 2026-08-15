package evidence

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// scenarioRoot is api/ → scenarios/backdrop-studio.
func scenarioRoot() string { return filepath.Join("..", "..", "..") }

// TestEveryEvidenceArtifactHasAProducingCommand is the enforcement half of the
// rule EVIDENCE.md states.
//
// It fails with the list of artifacts nobody can reproduce, and the two ways to
// make it pass are the two the rule allows: write the producing command, or
// delete the file. There is deliberately no third option, because the third
// option is what left fourteen unreproducible PNGs in the tree after a purge
// run for exactly that reason.
func TestEveryEvidenceArtifactHasAProducingCommand(t *testing.T) {
	root, doc, err := Load(scenarioRoot())
	require.NoError(t, err)

	orphans, err := Unreferenced(root, doc)
	require.NoError(t, err)

	require.Emptyf(t, orphans,
		"%d artifact(s) under docs/evidence/ are named by no entry in docs/internal/EVIDENCE.md:\n  %s\n\n"+
			"Every artifact under docs/evidence/ is produced by a command written in EVIDENCE.md. "+
			"Either add the producing command for these files, or delete them. "+
			"An artifact whose command nobody can name looks like proof while being a claim about a build nobody can identify.",
		len(orphans), strings.Join(orphans, "\n  "))
}

// The rule is only worth enforcing if the enforcement can fail. A guard that
// passes against an empty declaration would be theatre.
func TestUnreferencedReportsArtifactsWhenNothingIsDeclared(t *testing.T) {
	root, _, err := Load(scenarioRoot())
	require.NoError(t, err)

	orphans, err := Unreferenced(root, "# Evidence Pack\n\nNo artifact is declared here.\n")
	require.NoError(t, err)
	require.NotEmpty(t, orphans, "the walk found no files at all; the test would pass vacuously")
}

func TestCoverageFormsAdmitTheRightArtifacts(t *testing.T) {
	doc := "" +
		"## Artifacts and their producing commands\n" +
		"| `docs/evidence/render-matrix.md` | cmd | why |\n" +
		"| `docs/evidence/catalog/sheet-*.png` | cmd | why |\n" +
		"| `docs/evidence/perceptual/engraving-repair/` | cmd | why |\n"
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

	require.True(t, admits("docs/evidence/render-matrix.md"), "an exact path admits itself")
	require.True(t, admits("docs/evidence/catalog/sheet-screens.png"), "a glob admits a matching sibling")
	require.True(t, admits("docs/evidence/perceptual/engraving-repair/README.md"), "a directory admits its contents")
	require.True(t, admits("docs/evidence/perceptual/engraving-repair/nested/after.png"), "a directory admits nested contents")

	require.False(t, admits("docs/evidence/catalog/verdicts.md"), "a glob must not admit an undeclared sibling")
	require.False(t, admits("docs/evidence/scenes/sheet-horizon.png"),
		"a glob must not admit a same-named file in another directory")
	require.False(t, admits("docs/evidence/procedural/glaze-mosaic.png"), "an undeclared artifact stays undeclared")
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
		"| `docs/evidence/plates/*.png` and `docs/evidence/plates/README.md` | `make integration-evidence` | Both. |\n" +
		"| `docs/evidence/one.md` | run `docs/evidence/not-declared.md` | Only the first cell declares. |\n"

	patterns := map[string]bool{}
	for _, coverage := range DeclaredCoverage(doc) {
		patterns[coverage.Pattern] = true
	}
	require.True(t, patterns["docs/evidence/plates/*.png"], "the first path in a row must be declared")
	require.True(t, patterns["docs/evidence/plates/README.md"], "the second path in the same cell must be declared too")
	require.True(t, patterns["docs/evidence/one.md"])
	require.False(t, patterns["docs/evidence/not-declared.md"],
		"a path inside a producing command must never declare itself")
}
