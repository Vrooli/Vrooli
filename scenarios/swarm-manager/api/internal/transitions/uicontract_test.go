package transitions

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The UI decides whether a button shows an "agent run" marker by reading the
// `kind` this package declares for the transition behind it. That mapping
// lives in ui/src/lib/action-semantics.ts.
//
// A kind the UI does not map falls through to a classification that promises
// no agent involvement, so a new transition kind would silently ship buttons
// that dispatch agent work while looking like an ordinary save. This test
// makes that drift a build failure here, where the kind is introduced.
//
// The UI side has the matching assertion (every mapped kind resolves to a
// consequence class) in lib/action-semantics.test.ts.
const uiActionSemanticsPath = "../../../ui/src/lib/action-semantics.ts"

// Matches `case TransitionKind.WORKFLOW:` in the UI's mapping switch.
var uiMappedKind = regexp.MustCompile(`case TransitionKind\.([A-Z_]+):`)

func readUIMappedKinds(t *testing.T) map[string]struct{} {
	t.Helper()
	path := filepath.Clean(uiActionSemanticsPath)
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	matches := uiMappedKind.FindAllSubmatch(source, -1)
	if len(matches) == 0 {
		t.Fatalf("no `case TransitionKind.X:` arms found in %s; the UI kind mapping must stay machine-readable", path)
	}
	mapped := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		mapped[strings.ToLower(string(match[1]))] = struct{}{}
	}
	return mapped
}

// declaredKinds is the vocabulary this package can emit.
func declaredKinds() []string {
	return []string{string(KindSession), string(KindWorkflow), string(KindDeterministic)}
}

func TestUIMapsEveryDeclaredTransitionKind(t *testing.T) {
	mapped := readUIMappedKinds(t)
	var missing []string
	for _, kind := range declaredKinds() {
		if _, ok := mapped[kind]; !ok {
			missing = append(missing, kind)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("transition kinds %v are declared here but unmapped in %s; buttons for those transitions would render without an agent marker", missing, uiActionSemanticsPath)
	}
}

func TestUIDeclaresNoKindTheServerCannotEmit(t *testing.T) {
	declared := make(map[string]struct{}, 3)
	for _, kind := range declaredKinds() {
		declared[kind] = struct{}{}
	}
	// UNSPECIFIED is the proto zero value; the UI maps it deliberately to
	// "unknown" and it is not a kind this package emits.
	delete(declared, "unspecified")

	var unknown []string
	for kind := range readUIMappedKinds(t) {
		if kind == "unspecified" {
			continue
		}
		if _, ok := declared[kind]; !ok {
			unknown = append(unknown, kind)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		t.Fatalf("%s maps transition kinds %v that this package never emits", uiActionSemanticsPath, unknown)
	}
}

// TestRegistryKindsAreAllDeclared guards the other direction: the shipped
// registry file must not carry a kind outside the vocabulary, which would
// reach the UI as an unmapped value.
func TestRegistryKindsAreAllDeclared(t *testing.T) {
	registry, err := LoadDir(filepath.Clean("../../../.vrooli/swarm-transitions"))
	if err != nil {
		t.Skipf("registry not loadable from this working directory: %v", err)
	}
	declared := make(map[string]struct{}, 3)
	for _, kind := range declaredKinds() {
		declared[kind] = struct{}{}
	}
	for _, definition := range registry.Definitions() {
		if _, ok := declared[string(definition.Kind)]; !ok {
			t.Fatalf("transition %q declares kind %q, which is outside the vocabulary the UI can classify", definition.Key, definition.Kind)
		}
	}
}
