package heartbeat

import "testing"

func TestPromptSectionRegistryContainsEveryEmittedKind(t *testing.T) {
	if err := validatePromptSections([]PromptSection{{Kind: promptSectionKindMemberPolicy}}); err != nil {
		t.Fatalf("registered section rejected: %v", err)
	}
	if err := validatePromptSections([]PromptSection{{Kind: "unregistered"}}); err == nil {
		t.Fatal("unregistered section kind was accepted")
	}
}

// Every registry entry must carry both halves of its identity. A blank label or
// heading is how a section reaches an agent's prompt with no name to rank it by.
func TestPromptSectionRegistryEntriesAreComplete(t *testing.T) {
	seen := make(map[string]string, len(promptSectionKinds))
	for kind, entry := range promptSectionKinds {
		if entry.Label == "" {
			t.Errorf("section kind %q has no label", kind)
		}
		if entry.Heading == "" {
			t.Errorf("section kind %q has no heading", kind)
			continue
		}
		if other, dup := seen[entry.Heading]; dup {
			t.Errorf("section kinds %q and %q share heading %q", other, kind, entry.Heading)
		}
		seen[entry.Heading] = kind
	}
}
