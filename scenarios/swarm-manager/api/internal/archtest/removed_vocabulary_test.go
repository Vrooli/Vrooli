package archtest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// removedVocabulary is the closed set of Phase-9-deleted identifiers that must
// never reappear anywhere in api, cli, or the UI sources:
//   - "plan-manager-plan": the retired unmanaged plan-ref TARGET KIND (the
//     current vocabulary is backlog-item | initiative | plan-execution |
//     scenario; the domain plan_ref FIELD is unrelated and stays).
//   - "existing_item_flow" / "RunStrategyExistingItemFlow": the retired
//     item-level RUN STRATEGY enum value and its Go constant.
//   - "uses_item_execution_flow" / "usesItemExecutionFlow": the retired wire
//     capability (proto field 8 is reserved).
//
// NOTE: "item-level" itself is NOT here — it survives deliberately as the
// member-item-strategy sentinel wire value (operatingmode.ModeItemLevel /
// ui/src/lib/member-item-strategy.ts). What must not return is the item-level
// MODE FOLDER, asserted separately below.
var removedVocabulary = []string{
	"plan-manager-plan",
	"existing_item_flow",
	"RunStrategyExistingItemFlow",
	"uses_item_execution_flow",
	"usesItemExecutionFlow",
}

// vocabularyScanRoots are the source trees the guard sweeps, relative to this
// package (internal/archtest). Generated proto output is excluded: it is
// regenerated from the schemas, where the field is reserved by number.
var vocabularyScanRoots = []string{
	"../..",           // api/ (Go)
	"../../../cli",    // cli/ (Go)
	"../../../ui/src", // ui sources incl. ui/src/types (TS/TSX)
}

var vocabularyScanExts = map[string]bool{".go": true, ".ts": true, ".tsx": true}

// scanTreeForTerms walks root and returns term -> "path:line" hits over files
// with the given extensions, skipping dependency/build output directories and
// any file whose basename is in skipFiles. It is the detection primitive the
// removed-vocabulary guard and its red-proof share.
func scanTreeForTerms(root string, exts map[string]bool, terms []string, skipFiles map[string]bool) (map[string][]string, error) {
	hits := map[string][]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", "dist", "coverage", ".git":
				return filepath.SkipDir
			}
			return nil
		}
		if !exts[filepath.Ext(d.Name())] || skipFiles[d.Name()] {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(raw), "\n") {
			for _, term := range terms {
				if strings.Contains(line, term) {
					hits[term] = append(hits[term], fmt.Sprintf("%s:%d", path, i+1))
				}
			}
		}
		return nil
	})
	return hits, err
}

// TestRemovedVocabularyStaysRemoved sweeps api + cli + UI sources (tests
// included — the vocabulary is dead everywhere, not just in production code)
// for the Phase-9-deleted identifiers. Zero hits was proven at cutover
// (P9-A grep sweep); this guard keeps it zero. Red-proof:
// TestRemovedVocabularyScannerFiresOnViolation runs the same primitive
// against a synthetic violation.
func TestRemovedVocabularyStaysRemoved(t *testing.T) {
	self := map[string]bool{"removed_vocabulary_test.go": true}
	for _, root := range vocabularyScanRoots {
		abs, err := filepath.Abs(root)
		if err != nil {
			t.Fatalf("resolve %s: %v", root, err)
		}
		hits, err := scanTreeForTerms(abs, vocabularyScanExts, removedVocabulary, self)
		if err != nil {
			t.Fatalf("scan %s: %v", abs, err)
		}
		for _, term := range removedVocabulary {
			for _, loc := range hits[term] {
				t.Errorf("removed vocabulary %q reappeared at %s — this identifier was deleted in Phase 9 and must not return", term, loc)
			}
		}
	}
}

// TestItemLevelModeFolderStaysDeleted asserts the item-level MODE cannot be
// resurrected as data: modes/item-level/ (and in particular its mode.json)
// must not exist. The loader also rejects a mode.json declaring the reserved
// id (operatingmode's authored-modes static test); this folder check catches
// the resurrection even before a loader runs.
func TestItemLevelModeFolderStaysDeleted(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join(modesRoot, "item-level"))
	if err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
		t.Fatalf("modes/item-level exists (stat err: %v) — \"item-level\" is the reserved member-item-strategy sentinel, never a mode folder", statErr)
	}
}

// TestRemovedVocabularyScannerFiresOnViolation red-proofs the sweep: the same
// primitive run over a synthetic tree containing one removed identifier must
// flag it, and must honor the self-exclusion and directory skips.
func TestRemovedVocabularyScannerFiresOnViolation(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"violating.ts":               "const kind = \"plan-manager-plan\";\n",
		"clean.go":                   "package sample\n",
		"removed_vocabulary_test.go": "var s = \"existing_item_flow\"\n", // self-excluded
		"node_modules/dep.ts":        "export const x = \"RunStrategyExistingItemFlow\";\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	hits, err := scanTreeForTerms(dir, vocabularyScanExts, removedVocabulary, map[string]bool{"removed_vocabulary_test.go": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits["plan-manager-plan"]) != 1 {
		t.Fatalf("scanner must flag the synthetic plan-manager-plan reference, got %v", hits)
	}
	if len(hits["existing_item_flow"]) != 0 || len(hits["RunStrategyExistingItemFlow"]) != 0 {
		t.Fatalf("scanner must honor self-exclusion and node_modules skip, got %v", hits)
	}
}
