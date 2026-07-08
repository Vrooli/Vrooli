package operatingmode

import (
	"os"
	"testing"
)

// TestMain loads the data-backed operating-mode registry from the on-disk mode
// folder (modesDir) before running the package's tests, so the accessors
// (DefinitionFor, Modes, ValidateRegistry, …) and the registry-mutation test
// helpers observe a populated, validated registry deterministically — the same
// set the server loads at startup from <scenarioRoot>/modes.
func TestMain(m *testing.M) {
	if err := LoadRegistry(modesDir); err != nil {
		panic("load operating-mode registry for tests: " + err.Error())
	}
	os.Exit(m.Run())
}
