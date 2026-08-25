package main

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type internalCouplingContract struct {
	Ceiling int      `json:"ceiling"`
	Imports []string `json:"imports"`
}

func scenarioInternalImports(root string) ([]string, error) {
	seen := map[string]bool{}
	set := token.NewFileSet()
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil { return err }
		if info.IsDir() || !strings.HasSuffix(path, ".go") { return nil }
		file, err := parser.ParseFile(set, path, nil, parser.ImportsOnly)
		if err != nil { return err }
		for _, spec := range file.Imports {
			value := strings.Trim(spec.Path.Value, `"`)
			if strings.HasPrefix(value, "github.com/vrooli/vrooli/internal/") { seen[value] = true }
		}
		return nil
	})
	if err != nil { return nil, err }
	imports := make([]string, 0, len(seen))
	for value := range seen { imports = append(imports, value) }
	sort.Strings(imports)
	return imports, nil
}

func validateScenarioInternalImports(imports []string, contract internalCouplingContract) []string {
	allowed := map[string]bool{}
	for _, value := range contract.Imports { allowed[value] = true }
	var findings []string
	for _, value := range imports {
		if !allowed[value] { findings = append(findings, value) }
	}
	return findings
}

func TestScenarioInternalImportsStayWithinAllowlist(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "docs", "internal-coupling-allowlist.json"))
	if err != nil { t.Fatal(err) }
	var contract internalCouplingContract
	if err := json.Unmarshal(data, &contract); err != nil { t.Fatal(err) }
	imports, err := scenarioInternalImports(".")
	if err != nil { t.Fatal(err) }
	if findings := validateScenarioInternalImports(imports, contract); len(findings) != 0 { t.Fatalf("unallowlisted internal imports: %v", findings) }
	if len(imports) > contract.Ceiling { t.Fatalf("internal import count = %d, ceiling = %d", len(imports), contract.Ceiling) }
}

func TestScenarioInternalImportGateRejectsSyntheticGrowth(t *testing.T) {
	contract := internalCouplingContract{Ceiling: 1, Imports: []string{"github.com/vrooli/vrooli/internal/config"}}
	if findings := validateScenarioInternalImports([]string{"github.com/vrooli/vrooli/internal/config", "github.com/vrooli/vrooli/internal/newpackage"}, contract); len(findings) != 1 { t.Fatalf("synthetic growth findings = %v", findings) }
}
