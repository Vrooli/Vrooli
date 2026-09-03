package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGateImplementationsStaySmallAndFactsUseOneBatchEntryPoint(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(".", entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Count(string(data), "\n") + 1
		if lines > 400 {
			t.Fatalf("gate package file %s has %d lines; split it by subject", entry.Name(), lines)
		}
	}

	facts, err := os.ReadFile("ast_facts.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(facts)
	if !strings.Contains(source, `"--facts-root"`) {
		t.Fatal("facts reader does not use the batch facts entry point")
	}
	if strings.Contains(source, `"--facts",`) {
		t.Fatal("facts reader retains a per-file analyzer entry point")
	}
}

func TestGatesDoNotOpenTheirOwnWalker(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "filepath.WalkDir") {
			t.Fatalf("gate file %s opens its own walker; use internal/librarywalk", entry.Name())
		}
	}
}

func TestScopedFactsIndexContainsOnlySelectedAsset(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	index, err := readSourceFactsIndex(root, Scope{Root: root, Assets: []string{"controls.button"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(index) == 0 {
		t.Fatal("scoped facts index inspected no Button sources")
	}
	for path := range index {
		if !strings.Contains(filepath.ToSlash(path), "/library/components/Button/") {
			t.Fatalf("scoped facts index included unrelated source %s", path)
		}
	}
}

func TestEveryExecutableGateHasAnOwnedImplementationFile(t *testing.T) {
	for _, definition := range Definitions() {
		if definition.Run == nil {
			continue
		}
		found := false
		name := strings.ReplaceAll(definition.ID, "-", "_") + ".go"
		if _, err := os.Stat(name); err == nil {
			found = true
		}
		if !found {
			t.Fatalf("executable gate %q has no owned implementation file", definition.ID)
		}
	}
}

func TestNoNumberedGateFiles(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		if len(entry.Name()) > 0 && entry.Name()[0] >= '0' && entry.Name()[0] <= '9' {
			t.Fatalf("numbered gate implementation %s bypasses the registry", entry.Name())
		}
	}
}

func TestScopeIsAContract(t *testing.T) {
	seen := map[string]struct{}{}
	for _, definition := range Definitions() {
		if definition.ID == "" {
			t.Fatal("gate definition has no stable ID")
		}
		if _, duplicate := seen[definition.ID]; duplicate {
			t.Fatalf("gate definition %q is registered more than once", definition.ID)
		}
		seen[definition.ID] = struct{}{}
		if definition.Reads > ReadsCorpus {
			t.Fatalf("gate %q declares unknown read scope %d", definition.ID, definition.Reads)
		}
		if definition.Run != nil && len(definition.DeterminismInputs) == 0 {
			t.Fatalf("executable gate %q has no determinism inputs", definition.ID)
		}
	}
}

func TestEveryGateHonoursItsScope(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, definition := range Definitions() {
		if definition.Run == nil {
			continue
		}
		full, fullErr := RunDefinition(definition, Scope{Root: root})
		if fullErr != nil {
			continue
		}
		scoped, scopedErr := RunDefinition(definition, Scope{Root: root, Assets: []string{"controls.button"}})
		if scopedErr != nil {
			continue
		}
		if definition.Reads == ReadsCorpus {
			if scoped.Inspected != full.Inspected {
				t.Errorf("corpus-scoped gate %q changed inspected count from %d to %d", definition.ID, full.Inspected, scoped.Inspected)
			}
			continue
		}
		if scoped.Inspected == full.Inspected && full.Inspected > 0 {
			t.Errorf("asset-scoped gate %q ignored Scope.Assets: inspected %d in both runs", definition.ID, full.Inspected)
		}
	}
}
