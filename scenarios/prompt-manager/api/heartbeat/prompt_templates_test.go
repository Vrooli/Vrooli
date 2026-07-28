package heartbeat

import "testing"

func TestPromptSectionRegistryContainsEveryEmittedKind(t *testing.T) {
	if len(promptSectionKinds) != 14 {
		t.Fatalf("registered section kinds = %d, want 14", len(promptSectionKinds))
	}
	if err := validatePromptSections([]PromptSection{{Kind: promptSectionKindActiveTaskBrief}}); err != nil {
		t.Fatalf("registered section rejected: %v", err)
	}
	if err := validatePromptSections([]PromptSection{{Kind: "unregistered"}}); err == nil {
		t.Fatal("unregistered section kind was accepted")
	}
}
