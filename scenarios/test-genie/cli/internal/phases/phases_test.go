package phases

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNormalizeSelectionNormalizesAliasesAndDedupes(t *testing.T) {
	got, err := NormalizeSelection([]string{"Unit", "e2e", "unit", "playbooks"})
	if err != nil {
		t.Fatalf("NormalizeSelection returned error: %v", err)
	}

	want := []string{"unit", "playbooks"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected normalized phases %v, got %v", want, got)
	}
}

func TestNormalizeSelectionTreatsAllAsServerDefault(t *testing.T) {
	got, err := NormalizeSelection([]string{" default "})
	if err != nil {
		t.Fatalf("NormalizeSelection returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil selection for default server behavior, got %v", got)
	}
}

func TestNormalizeSelectionRejectsUnknownPhase(t *testing.T) {
	_, err := NormalizeSelection([]string{"chaos"})
	if err == nil {
		t.Fatal("expected unknown phase to fail")
	}
	if !strings.Contains(err.Error(), "unknown phase") {
		t.Fatalf("expected unknown phase error, got %v", err)
	}
}

func TestApplySkipNormalizesAliases(t *testing.T) {
	got := ApplySkip([]string{"unit", "e2e", "playbooks", "quality"}, []string{" PLAYBOOKS ", "quality"})
	want := []string{"unit"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected skipped phases %v, got %v", want, got)
	}
}

func TestMakeDescriptorMapsDoesNotInventFallbackTimeouts(t *testing.T) {
	descMap, targets := MakeDescriptorMaps([]Descriptor{{Name: "Quality"}})
	if _, ok := descMap["quality"]; !ok {
		t.Fatalf("expected quality descriptor map entry, got %v", descMap)
	}
	if len(targets) != 0 {
		t.Fatalf("expected no synthetic timeouts, got %v", targets)
	}
}

func TestMakeDescriptorMapsUsesExplicitTimeouts(t *testing.T) {
	_, targets := MakeDescriptorMaps([]Descriptor{{Name: "Structure", DefaultTimeoutSeconds: 42}})
	if got := targets["structure"]; got != 42*time.Second {
		t.Fatalf("expected explicit timeout, got %v", got)
	}
}
