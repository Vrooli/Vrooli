package harness

import (
	"os"
	"testing"
)

// TestRenderBrief is a documentation aid: run with -v to read exactly what the
// authoring model is told. It asserts nothing beyond non-emptiness so it cannot
// become a brittle golden test.
func TestRenderBrief(t *testing.T) {
	contract := Load()
	if contract.Instruction() == "" {
		t.Fatal("empty brief")
	}
	if os.Getenv("PRINT_BRIEF") != "" {
		t.Log("\n" + contract.Stamp() + "\n\n" + contract.Instruction())
	}
}
