package validation

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	companionRegistrySchemaVersion = "1.0.0"

	// companionPackagesRoot is the directory holding every shared Go module
	// whose packages may own a test companion.
	companionPackagesRoot = "packages"

	// companionRegistryUpdateEnv regenerates the checked-in registry when the
	// drift test runs with it set to "1".
	companionRegistryUpdateEnv = "UPDATE_TEST_COMPANIONS"
)

// discoverCompanionExports enumerates the canonical test companions in the
// repository and reads their exported surface straight from source.
//
// Discovery follows the published convention rather than a curated list: a
// companion is a package whose name ends in `test` and which serves the sibling
// package named by the remainder. Deriving the registry from the packages is
// what keeps it honest. The hand-maintained version omitted an entire companion
// package and roughly two thirds of the symbols the rest exported, and nothing
// reported that — a registry the rules read but no one verifies is a silent cap
// on what those rules can ever detect.
func discoverCompanionExports(repoRoot string) ([]companionExport, error) {
	packagesDir := filepath.Join(repoRoot, companionPackagesRoot)
	moduleEntries, err := os.ReadDir(packagesDir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", packagesDir, err)
	}

	var exports []companionExport
	for _, moduleEntry := range moduleEntries {
		if !moduleEntry.IsDir() {
			continue
		}
		moduleDir := filepath.Join(packagesDir, moduleEntry.Name())
		modulePath := declaredModulePath(moduleDir)
		if modulePath == "" {
			continue
		}
		for _, dir := range goPackageDirs(moduleDir) {
			pkg := packageNameOf(dir)
			owned, ok := companionOwnerPackage(pkg)
			if !ok {
				continue
			}
			symbols := extractCompanionSymbols(dir)
			if len(symbols) == 0 {
				continue
			}
			importPath := modulePath
			if rel, relErr := filepath.Rel(moduleDir, dir); relErr == nil && rel != "." {
				importPath = modulePath + "/" + filepath.ToSlash(rel)
			}
			exports = append(exports, companionExport{
				Owner:           moduleEntry.Name() + "/" + owned,
				OwnerImportPath: ownerImportPath(moduleDir, modulePath, dir, owned),
				ImportPath:      importPath,
				Symbols:         symbols,
			})
		}
	}
	sort.Slice(exports, func(i, j int) bool { return exports[i].ImportPath < exports[j].ImportPath })
	return exports, nil
}

// companionOwnerPackage reports the package a companion serves. `_test` is an
// external test package for the package under test, not a companion consumers
// import, so it never qualifies.
func companionOwnerPackage(pkg string) (string, bool) {
	if pkg == "" || pkg == "test" || strings.HasSuffix(pkg, "_test") {
		return "", false
	}
	owned := strings.TrimSuffix(pkg, "test")
	if owned == pkg || owned == "" {
		return "", false
	}
	return owned, true
}

// ownerImportPath resolves the import path of the package a companion serves.
// The convention places it beside the companion; a module whose root package is
// the owned one (repo-contract-go/repocontract) puts it at the module root
// instead. Returning "" is honest when neither holds — shape matching then
// falls back to the companion's own import path.
func ownerImportPath(moduleDir, modulePath, companionDir, owned string) string {
	sibling := filepath.Join(filepath.Dir(companionDir), owned)
	if packageNameOf(sibling) == owned {
		if rel, err := filepath.Rel(moduleDir, sibling); err == nil {
			if rel == "." {
				return modulePath
			}
			return modulePath + "/" + filepath.ToSlash(rel)
		}
	}
	if packageNameOf(moduleDir) == owned {
		return modulePath
	}
	return ""
}

// extractCompanionSymbols reads a companion package's exported surface using
// the same declaration walk the matcher runs against scenario code, so a
// registry entry and a local declaration are rendered identically and compare
// exactly. Symbols from the companion's own tests are not part of its surface.
func extractCompanionSymbols(dir string) []companionSymbol {
	clean := filepath.Clean(dir)
	var symbols []companionSymbol
	for _, declaration := range collectLocalDeclarations(dir) {
		if filepath.Dir(declaration.File) != clean || isGoTestFile(declaration.File) {
			continue
		}
		if !ast.IsExported(declaration.Name) {
			continue
		}
		symbol := companionSymbol{Name: declaration.Name, Kind: declaration.Kind}
		if declaration.Kind == "function" {
			symbol.Signature = declaration.Signature
		} else {
			symbol.Methods = sortedMethodNames(declaration.Methods)
		}
		symbols = append(symbols, symbol)
	}
	sort.Slice(symbols, func(i, j int) bool { return symbols[i].Name < symbols[j].Name })
	return symbols
}

// generateCompanionRegistry builds the registry the companion rules consume.
//
// Seams are carried through rather than discovered: a seam is a production
// interface its consumers inject, so it lives in a production package that no
// naming rule can distinguish from any other. Companions are discoverable
// because the convention gives them a name.
func generateCompanionRegistry(repoRoot string, seams []companionExport) (companionRegistry, error) {
	companions, err := discoverCompanionExports(repoRoot)
	if err != nil {
		return companionRegistry{}, err
	}
	return companionRegistry{
		SchemaVersion: companionRegistrySchemaVersion,
		Companions:    companions,
		Seams:         seams,
	}, nil
}

func marshalCompanionRegistry(registry companionRegistry) ([]byte, error) {
	raw, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal companion registry: %w", err)
	}
	return append(raw, '\n'), nil
}

// diffCompanionRegistries reports how the checked-in registry differs from what
// the packages currently export, as lines an operator can act on directly.
func diffCompanionRegistries(want, got companionRegistry) []string {
	wanted := indexRegistrySymbols(want)
	current := indexRegistrySymbols(got)

	var lines []string
	for _, key := range sortedKeys(wanted) {
		if _, ok := current[key]; !ok {
			lines = append(lines, "missing from registry: "+key)
		}
	}
	for _, key := range sortedKeys(current) {
		if _, ok := wanted[key]; !ok {
			lines = append(lines, "no longer exported: "+key)
		}
	}
	for _, key := range sortedKeys(wanted) {
		other, ok := current[key]
		if !ok || other == wanted[key] {
			continue
		}
		lines = append(lines, fmt.Sprintf("shape changed: %s (registry %s, source %s)", key, other, wanted[key]))
	}
	return lines
}

func indexRegistrySymbols(registry companionRegistry) map[string]string {
	index := map[string]string{}
	for _, export := range registry.Companions {
		for _, symbol := range export.Symbols {
			key := export.ImportPath + "." + symbol.Name
			shape := symbol.Kind
			if symbol.Signature != "" {
				shape += " " + symbol.Signature
			}
			if len(symbol.Methods) > 0 {
				shape += " {" + strings.Join(symbol.Methods, ",") + "}"
			}
			index[key] = shape
		}
	}
	return index
}

func sortedKeys(in map[string]string) []string {
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// goPackageDirs lists every directory under root that holds non-test Go source.
func goPackageDirs(root string) []string {
	seen := map[string]bool{}
	walkSourceFiles(root, func(file string) {
		if isGoSourceFile(file) {
			seen[filepath.Dir(file)] = true
		}
	})
	dirs := make([]string, 0, len(seen))
	for dir := range seen {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	return dirs
}

// packageNameOf reads the package clause of a directory's first non-test Go
// file. Directory names are not authoritative; the package clause is.
func packageNameOf(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if !isGoSourceFile(path) {
			continue
		}
		file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
		if parseErr != nil || file == nil || file.Name == nil {
			continue
		}
		return file.Name.Name
	}
	return ""
}

// declaredModulePath reads the module clause of the go.mod in dir itself.
// Unlike goModulePath it never walks to an ancestor: a directory with no module
// of its own is not a module, and inheriting the repository's module path would
// invent companions under it.
func declaredModulePath(dir string) string {
	for _, line := range strings.Split(readFileString(filepath.Join(dir, "go.mod")), "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "//", 2)[0])
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}
