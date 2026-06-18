package graph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// Normalize dedupes raw nodes/edges and computes a content hash. The
// result is content-hash-stable: identical RawGraph inputs produce
// identical GraphSnapshot.ContentHash + ID.
func Normalize(scenario string, raw RawGraph) GraphSnapshot {
	files := dedupeFiles(raw.Files)
	packages := dedupePackages(raw.Packages)
	symbols := dedupeSymbols(raw.Symbols)
	imports := dedupeImports(raw.Imports)
	inferFileTestFlags(files)
	inferImportTestFlags(imports, files)

	sort.Slice(files, func(i, j int) bool { return files[i].ID < files[j].ID })
	sort.Slice(packages, func(i, j int) bool { return packages[i].ID < packages[j].ID })
	sort.Slice(symbols, func(i, j int) bool { return symbols[i].ID < symbols[j].ID })
	sort.Slice(imports, func(i, j int) bool {
		if imports[i].From != imports[j].From {
			return imports[i].From < imports[j].From
		}
		return imports[i].ToPackageID < imports[j].ToPackageID
	})

	hash := contentHash(scenario, raw.Languages, files, packages, symbols, imports)

	return GraphSnapshot{
		ID:           "snap:" + scenario + ":" + hash,
		Scenario:     scenario,
		ContentHash:  hash,
		Languages:    uniqueLanguages(raw.Languages),
		ExtractionMS: raw.ExtractionMS,
		Files:        files,
		Packages:     packages,
		Symbols:      symbols,
		Imports:      imports,
	}
}

func dedupeFiles(in []FileNode) []FileNode {
	seen := make(map[string]FileNode, len(in))
	for _, f := range in {
		seen[f.ID] = f
	}
	out := make([]FileNode, 0, len(seen))
	for _, f := range seen {
		out = append(out, f)
	}
	return out
}

func inferFileTestFlags(files []FileNode) {
	for i := range files {
		if files[i].IsTest {
			continue
		}
		files[i].IsTest = isLikelyTestPath(files[i].Path)
	}
}

func inferImportTestFlags(imports []ImportEdge, files []FileNode) {
	testFiles := make(map[string]bool, len(files))
	for _, f := range files {
		if f.IsTest {
			testFiles[f.ID] = true
		}
	}
	for i := range imports {
		if imports[i].TestOnly {
			continue
		}
		imports[i].TestOnly = testFiles[imports[i].From]
	}
}

func isLikelyTestPath(path string) bool {
	switch {
	case hasSuffix(path, "_test.go"),
		hasSuffix(path, ".test.ts"),
		hasSuffix(path, ".test.tsx"),
		hasSuffix(path, ".spec.ts"),
		hasSuffix(path, ".spec.tsx"):
		return true
	default:
		return false
	}
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func dedupePackages(in []PackageNode) []PackageNode {
	seen := make(map[string]PackageNode, len(in))
	for _, p := range in {
		seen[p.ID] = p
	}
	out := make([]PackageNode, 0, len(seen))
	for _, p := range seen {
		out = append(out, p)
	}
	return out
}

func dedupeSymbols(in []SymbolNode) []SymbolNode {
	seen := make(map[string]SymbolNode, len(in))
	for _, s := range in {
		seen[s.ID] = s
	}
	out := make([]SymbolNode, 0, len(seen))
	for _, s := range seen {
		out = append(out, s)
	}
	return out
}

func dedupeImports(in []ImportEdge) []ImportEdge {
	type key struct{ from, to string }
	seen := make(map[key]ImportEdge, len(in))
	for _, e := range in {
		k := key{e.From, e.ToPackageID}
		if existing, ok := seen[k]; ok {
			// Merge symbol ids; preserve test_only=false when at least
			// one edge is non-test.
			existing.SymbolIDs = mergeStrings(existing.SymbolIDs, e.SymbolIDs)
			existing.SymbolKinds = mergeStrings(existing.SymbolKinds, e.SymbolKinds)
			if !e.TestOnly {
				existing.TestOnly = false
			}
			seen[k] = existing
			continue
		}
		// Copy slice to avoid aliasing input.
		ec := e
		if len(e.SymbolIDs) > 0 {
			ec.SymbolIDs = append([]string(nil), e.SymbolIDs...)
			sort.Strings(ec.SymbolIDs)
		}
		if len(e.SymbolKinds) > 0 {
			ec.SymbolKinds = append([]string(nil), e.SymbolKinds...)
			sort.Strings(ec.SymbolKinds)
		}
		seen[k] = ec
	}
	out := make([]ImportEdge, 0, len(seen))
	for _, e := range seen {
		out = append(out, e)
	}
	return out
}

func mergeStrings(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	for _, s := range a {
		seen[s] = struct{}{}
	}
	for _, s := range b {
		seen[s] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func uniqueLanguages(in []Language) []Language {
	seen := make(map[Language]struct{}, len(in))
	out := make([]Language, 0, len(in))
	for _, l := range in {
		if _, ok := seen[l]; ok {
			continue
		}
		seen[l] = struct{}{}
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// contentHash is a canonical content-addressable hash of the snapshot
// payload. Stable across runs as long as the input set is the same.
func contentHash(
	scenario string,
	langs []Language,
	files []FileNode,
	pkgs []PackageNode,
	syms []SymbolNode,
	imps []ImportEdge,
) string {
	payload := struct {
		Scenario  string        `json:"scenario"`
		Languages []Language    `json:"languages"`
		Files     []FileNode    `json:"files"`
		Packages  []PackageNode `json:"packages"`
		Symbols   []SymbolNode  `json:"symbols"`
		Imports   []ImportEdge  `json:"imports"`
	}{
		Scenario:  scenario,
		Languages: uniqueLanguages(langs),
		Files:     files,
		Packages:  pkgs,
		Symbols:   syms,
		Imports:   imps,
	}
	b, _ := json.Marshal(payload)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
