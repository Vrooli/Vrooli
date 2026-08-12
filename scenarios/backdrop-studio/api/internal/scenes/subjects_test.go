package scenes

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// taxonomyPath is the document that defines the subject vocabulary. The test
// reads it rather than restating it, because a vocabulary written down twice
// diverges on the first edit that touches only one copy.
const taxonomyPath = "../../../docs/reference/taxonomy.md"

// problemsPath is where a subject nobody has built for is recorded. A subject
// may be unreachable — the taxonomy is an open list and the procedural lane is
// not meant to cover all of it — but it may not be unreachable *silently*.
const problemsPath = "../../../docs/internal/PROBLEMS.md"

var axis3Row = regexp.MustCompile("(?m)^\\| `([a-z_]+)` \\| ")

// TestEverySubjectIsReachableOrRecorded is the coverage gate.
//
// Every subject the taxonomy defines must be one of three things: drawn by a
// named generator, declared as needing the model lane, or written down in
// PROBLEMS.md as unbuilt. A subject that is none of the three is one a style
// can name and nothing can honour — which, before this phase, meant the lane
// quietly rendered a different picture instead.
func TestEverySubjectIsReachableOrRecorded(t *testing.T) {
	taxonomy := readDoc(t, taxonomyPath)
	problems := readDoc(t, problemsPath)

	subjects := axis3Subjects(t, taxonomy)
	require.GreaterOrEqual(t, len(subjects), 12,
		"the Axis 3 table did not parse; this test would pass vacuously")

	procedural := map[string]bool{}
	for _, s := range ProceduralSubjects() {
		procedural[s] = true
	}

	for _, subject := range subjects {
		if procedural[subject] {
			require.NotEmptyf(t, PresetsForSubject(subject),
				"%s is listed as procedural but no generator claims it", subject)
			continue
		}
		// Not drawable procedurally: the taxonomy must say so, or PROBLEMS.md
		// must record it as unbuilt. Both are honest; silence is not.
		recorded := containsSubject(taxonomy, subject, "model lane") ||
			containsSubject(problems, subject, "")
		require.Truef(t, recorded,
			"subject %q is reachable by no generator and recorded nowhere: a style may name it and the lane cannot honour it. "+
				"Either build a generator, mark it model-lane in taxonomy.md, or record it as unbuilt in PROBLEMS.md.",
			subject)
	}
}

// TestEveryGeneratorDeclaresASubject is the other direction: a generator that
// declares nothing cannot be reached by any style, and would be dead code
// wearing the appearance of a feature.
func TestEveryGeneratorDeclaresASubject(t *testing.T) {
	for _, preset := range Presets {
		subject, ok := SubjectOf(preset)
		require.Truef(t, ok, "generator %q declares no subject, so no style can reach it", preset)
		require.NotEmpty(t, subject)
	}
	// And every subject with generators must have exactly one default, or a
	// style that names none gets an arbitrary answer.
	for _, subject := range ProceduralSubjects() {
		preset, err := ResolvePreset(subject, "")
		require.NoErrorf(t, err, "subject %q has generators but no default", subject)
		depicts, _ := SubjectOf(preset)
		require.Equal(t, subject, depicts)
	}
}

func TestResolvePresetRefusesAMismatch(t *testing.T) {
	_, err := ResolvePreset("geological", "caustics")
	require.Error(t, err)
	require.Contains(t, err.Error(), "depicts")
	require.Contains(t, err.Error(), "terrain", "the message must say what to use instead")

	_, err = ResolvePreset("celestial", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "model-backed")

	_, err = ResolvePreset("non_representational", "not-a-generator")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown generator")
}

func TestSeveralGeneratorsShareTheAbstractSubject(t *testing.T) {
	// The point of the new field generators: four distinct pictures under one
	// subject, selectable by name. Before this, one subject meant one picture.
	presets := PresetsForSubject("non_representational")
	require.GreaterOrEqual(t, len(presets), 4,
		"the abstract half of the catalog needs real breadth, not one generator with four names")
	for _, preset := range presets {
		resolved, err := ResolvePreset("non_representational", preset)
		require.NoError(t, err)
		require.Equal(t, preset, resolved)
	}
}

func readDoc(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Clean(path))
	require.NoErrorf(t, err, "the coverage gate needs %s; if it moved, move this test with it", path)
	return string(raw)
}

func axis3Subjects(t *testing.T, doc string) []string {
	t.Helper()
	start := indexAfter(doc, "## Axis 3")
	require.Positive(t, start, "taxonomy.md has no Axis 3 section")
	end := indexAfter(doc[start:], "## Axis 4")
	if end <= 0 {
		end = len(doc) - start
	}
	out := []string{}
	for _, m := range axis3Row.FindAllStringSubmatch(doc[start:start+end], -1) {
		out = append(out, m[1])
	}
	return out
}

func indexAfter(doc, marker string) int {
	for i := 0; i+len(marker) <= len(doc); i++ {
		if doc[i:i+len(marker)] == marker {
			return i
		}
	}
	return -1
}

func containsSubject(doc, subject, alongside string) bool {
	needle := "`" + subject + "`"
	for i := 0; i+len(needle) <= len(doc); i++ {
		if doc[i:i+len(needle)] != needle {
			continue
		}
		if alongside == "" {
			return true
		}
		// The mention must be on the same line as the qualifier, so an
		// unrelated occurrence elsewhere in the document does not count.
		lineEnd := i
		for lineEnd < len(doc) && doc[lineEnd] != '\n' {
			lineEnd++
		}
		lineStart := i
		for lineStart > 0 && doc[lineStart-1] != '\n' {
			lineStart--
		}
		if indexAfter(doc[lineStart:lineEnd], alongside) >= 0 {
			return true
		}
	}
	return false
}
