package opsrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenericRunnerHasNoModeNameBranches is the static guard for the phase
// invariant: the generic runner must contain NO branch keyed to a named mode,
// phase, or shipped methodology. Selection lives entirely in data (operation
// contracts, bindings, policies) and behind the ModePreparer/ExecutionDriver
// seams. This scans every non-test source file in the package for the shipped
// mode identities and phase-less legacy names; a hit means a caller leaked a
// mode-specific branch into the generic substrate.
func TestGenericRunnerHasNoModeNameBranches(t *testing.T) {
	forbidden := []string{
		"holistic-loop", "phased-plan-drain", "item-level",
		"ModeHolisticLoop", "ModePhasedPlanDrain", "ModeItemLevel",
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(".", name))
		if err != nil {
			t.Fatal(err)
		}
		body := string(raw)
		for _, bad := range forbidden {
			if strings.Contains(body, bad) {
				t.Errorf("%s references mode-specific identifier %q: the generic runner must not branch on a named mode", name, bad)
			}
		}
	}
}
