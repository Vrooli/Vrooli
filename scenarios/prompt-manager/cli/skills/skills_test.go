package skills

import (
	"strings"
	"testing"
)

func TestCommandsRegistersSkillCommand(t *testing.T) {
	groups := Commands(nil)
	if len(groups) != 1 || groups[0].Title != "Skills" {
		t.Fatalf("unexpected command groups: %+v", groups)
	}
	if len(groups[0].Commands) != 1 || groups[0].Commands[0].Name != "skill" {
		t.Fatalf("unexpected skill command: %+v", groups[0].Commands)
	}
}

func TestUsageTextDocumentsReadAndVariants(t *testing.T) {
	text := usageText()
	for _, want := range []string{"read", "variants"} {
		if !strings.Contains(text, want) {
			t.Fatalf("usage text missing %q: %s", want, text)
		}
	}
}
