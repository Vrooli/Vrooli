package wizard

import "testing"

func TestWizardUsesTheSharedNineStepSurface(t *testing.T) {
	if got := len([]string{"scenarios", "resources", "credentials", "integrations", "host", "operating-mode", "apply", "validation"}); got != 8 {
		t.Fatalf("post-welcome wizard surface has %d steps", got)
	}
	selection := Selection{ScenarioState: map[string]bool{"demo": true}, Resources: map[string]bool{"postgres": false}}
	patch := selectionPatch(selection)
	if patch["scenarios"] == nil || patch["resources"] == nil {
		t.Fatalf("wizard did not produce the shared state patch: %#v", patch)
	}
}
