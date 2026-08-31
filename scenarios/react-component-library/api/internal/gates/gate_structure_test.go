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
		if !strings.Contains(string(data), "func Validate") {
			continue
		}
		lines := strings.Count(string(data), "\n") + 1
		if lines > 400 {
			t.Fatalf("gate implementation %s has %d lines; split shared helpers from the gate", entry.Name(), lines)
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
		for _, name := range []string{definition.ID + ".go", strings.ReplaceAll(definition.ID, "-", "_") + ".go"} {
			if _, err := os.Stat(name); err == nil {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("executable gate %q has no owned implementation file", definition.ID)
		}
	}
}
