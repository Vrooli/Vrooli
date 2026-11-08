package reconcile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"experience-manager/internal/spec"
)

// A release that composes a base through a major-version library import is
// proven by the base's treatment at the exact version the release pinned.
func TestComponentSourceGraphFollowsPinnedLibraryImports(t *testing.T) {
	scenario := filepath.Join(t.TempDir(), "react-component-library")
	components := filepath.Join(scenario, "library", "components")
	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(components, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("Action/versions/1.0.0/Action.tsx", `import { Base } from "@vrooli/react-component-library/Base/2";`)
	write("Action/versions/1.0.0/story.json", `{}`)
	write("Action/versions/1.0.0/dependencies.json", `{"dependencies":[{"libraryId":"react-component-library:Base","major":2,"observed":"2.1.0"}]}`)
	write("Base/versions/2.0.0/Base.tsx", `export const Base = "unpinned";`)
	write("Base/versions/2.1.0/Base.tsx", `const css = "[data-control]:hover { filter: brightness(1.06); } .x { background: var(--color-primary); }";`)

	report := spec.Report{TargetPath: scenario, Spec: &spec.ScenarioSpec{ExperienceDir: filepath.Join(scenario, "experience")}}
	component := spec.ComponentDocument{Component: spec.ComponentIdentity{StoryRef: "../../library/components/Action/versions/1.0.0/story.json"}}

	source, err := readComponentSourceGraph(report, component)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source, "brightness(1.06)") || strings.Contains(source, "unpinned") {
		t.Fatalf("source graph did not follow the pinned dependency version:\n%s", source)
	}
	if !sourceClaimPasses(spec.Claim{Type: "state-contrast"}, source) {
		t.Fatal("state-contrast claim did not see the composed base treatment")
	}

	write("Action/versions/1.0.0/dependencies.json", `{"dependencies":[]}`)
	source, err = readComponentSourceGraph(report, component)
	if err != nil {
		t.Fatal(err)
	}
	if sourceClaimPasses(spec.Claim{Type: "state-contrast"}, source) {
		t.Fatal("an unpinned library import must not contribute proof")
	}
}
