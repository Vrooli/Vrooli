package prose

import (
	"os"
	"strings"
	"testing"
)

func TestSelectionCodeCannotReadVerbalizedHint(t *testing.T) {
	raw, err := os.ReadFile("service.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	start := strings.Index(source, "func choose(")
	end := strings.Index(source[start:], "func mergedLexicon")
	if start < 0 || end < 0 {
		t.Fatal("selection function boundary not found")
	}
	selection := source[start : start+end]
	if strings.Contains(selection, "VerbalizedHint") || strings.Contains(selection, "HintOrdinal") || strings.Contains(selection, "verbalized_hint") {
		t.Fatal("selection policy reads the uncalibrated verbalized hint")
	}
}
