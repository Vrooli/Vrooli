package archtest

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// AuthoredModeIDs returns the shipped operating-mode ids (every modes/
// subdirectory containing a mode.json) plus the reserved member-item-strategy
// sentinel ("item-level"). This is the forbidden-vocabulary source for the
// generic-engine purity guards: a generic agent-operations package must not
// mention ANY of these names — selection lives in data (contracts, bindings,
// policies), never in code. Deriving the list from the authored catalog means
// adding a 16th mode automatically extends the guard.
func AuthoredModeIDs(modesDir string) ([]string, error) {
	entries, err := os.ReadDir(modesDir)
	if err != nil {
		return nil, fmt.Errorf("read modes dir %s: %w", modesDir, err)
	}
	ids := []string{"item-level"}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, statErr := os.Stat(filepath.Join(modesDir, e.Name(), "mode.json")); statErr == nil {
			ids = append(ids, e.Name())
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// ScanPackageDirForTerms scans every non-test Go source file directly in dir
// for the given terms (plain substring match, comments and string literals
// included — a mode name has no business in generic code even as
// documentation) and returns term -> "file:line" hits. It is the single
// detection primitive the purity guards and their red-proof tests share.
func ScanPackageDirForTerms(dir string, terms []string) (map[string][]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read package dir %s: %w", dir, err)
	}
	hits := map[string][]string{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		for i, line := range strings.Split(string(raw), "\n") {
			for _, term := range terms {
				if strings.Contains(line, term) {
					hits[term] = append(hits[term], fmt.Sprintf("%s:%d", name, i+1))
				}
			}
		}
	}
	return hits, nil
}

// RequireNoModeNameBranches fails t when any non-test Go source in pkgDir
// mentions a shipped mode id, the member-item-strategy sentinel, or one of the
// extra forbidden identifiers (e.g. the operatingmode Mode* constants). It is
// the shared purity guard for the generic agent-operations packages
// (opsrunner, opsbridge, opscatalog): the generic substrate must contain no
// branch, comment, or literal keyed to a named mode.
func RequireNoModeNameBranches(t *testing.T, pkgDir, modesDir string, extraTerms ...string) {
	t.Helper()
	terms, err := AuthoredModeIDs(modesDir)
	if err != nil {
		t.Fatalf("derive forbidden mode vocabulary: %v", err)
	}
	terms = append(terms, extraTerms...)
	hits, err := ScanPackageDirForTerms(pkgDir, terms)
	if err != nil {
		t.Fatalf("scan %s: %v", pkgDir, err)
	}
	for _, term := range terms {
		for _, loc := range hits[term] {
			t.Errorf("%s references mode-specific identifier %q: the generic agent-operations substrate must not branch on (or even name) a shipped mode", loc, term)
		}
	}
}
