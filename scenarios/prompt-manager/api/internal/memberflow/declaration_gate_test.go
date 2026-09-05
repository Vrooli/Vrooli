package memberflow

import (
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

// TestLiveTreeHasNoDeclarationErrors is the gate.
//
// `prompt-manager graph topics` could not gate CI because it shared one exit
// code with runtime observation, and a runtime finding cannot be cleared by
// editing the tree. So nothing ever forced the declaration count down, and
// Runtime observations are intentionally excluded from this declaration gate:
// they describe historical writes rather than checked-in declarations.
//
// This runs in the test-genie `unit` phase — a phase that already exists and
// already gates — rather than as a new descriptor-backed phase, because
// testing.json's `phases` block configures registered phases and cannot
// introduce an arbitrary command. The effect the plan asked for is identical:
// a declaration error fails the suite; a runtime finding never does.
func TestLiveTreeHasNoDeclarationErrors(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test filename")
	}
	pkgDir := filepath.Dir(filename)
	storeDir := filepath.Clean(filepath.Join(pkgDir, "..", "..", "..", "store"))
	repoRoot := filepath.Clean(filepath.Join(pkgDir, "..", "..", "..", "..", ".."))

	members, err := LoadAll(storeDir)
	if err != nil {
		t.Fatalf("load member declarations: %v", err)
	}
	skillIDs, _ := LoadSkillIDs(storeDir)
	taxonomies, _ := LoadAllTaxonomies(repoRoot)

	result := Validate(members, ValidationOptions{
		RepoRoot:   repoRoot,
		StoreDir:   storeDir,
		SkillIDs:   skillIDs,
		Taxonomies: taxonomies,
	})

	byRule := map[string]int{}
	for _, f := range result.Findings {
		// Kind is stamped from the catalog. A runtime finding reports what an
		// agent did and no edit to this tree clears it, so gating on one would
		// make this test permanently red — the exact failure being fixed.
		if f.Kind != KindDeclaration || f.Severity != SeverityError {
			continue
		}
		byRule[f.Rule]++
	}
	if len(byRule) == 0 {
		return
	}

	rules := make([]string, 0, len(byRule))
	for rule := range byRule {
		rules = append(rules, rule)
	}
	sort.Strings(rules)
	for _, rule := range rules {
		t.Errorf("declaration error %s: %d finding(s)", rule, byRule[rule])
	}
	t.Errorf("the checked-in tree has declaration errors; run `prompt-manager graph topics` to see them, and `prompt-manager graph runtime` for the observations this gate deliberately ignores")
}
