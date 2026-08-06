package memberflow

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoDocsDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test filename")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "docs"))
}

// TestGeneratedRuleTablesMatchTheCatalog is the drift gate. Two hand-written
// tables previously restated the catalog with nothing forcing them to follow
// it; a rule could be added, deleted, or re-severitied and the docs would keep
// describing the old world.
func TestGeneratedRuleTablesMatchTheCatalog(t *testing.T) {
	catalog, err := DefaultRuleCatalog()
	if err != nil {
		t.Fatalf("build catalog: %v", err)
	}
	for _, target := range RuleReferenceTargets(repoDocsDir(t)) {
		raw, err := os.ReadFile(target.Path)
		if err != nil {
			t.Fatalf("read %s: %v", target.Path, err)
		}
		want, ok := ApplyRuleReference(string(raw), RenderRuleReference(catalog, target.Groups))
		if !ok {
			t.Fatalf("%s has no rule-catalog generated block", target.Path)
		}
		if want != string(raw) {
			t.Errorf("%s is out of date with the rule catalog. Regenerate it.", target.Path)
		}
	}
}

// Every catalogued rule must appear in exactly one generated table, or the
// generated reference is not a reference.
func TestEveryCataloguedRuleReachesAGeneratedTable(t *testing.T) {
	catalog, err := DefaultRuleCatalog()
	if err != nil {
		t.Fatalf("build catalog: %v", err)
	}
	targets := RuleReferenceTargets(repoDocsDir(t))
	rendered := make([]string, 0, len(targets))
	for _, target := range targets {
		rendered = append(rendered, RenderRuleReference(catalog, target.Groups))
	}
	// The model, plan-of-record, objective, and contract families have no
	// generated block yet; they are covered by `prompt-manager graph rules`.
	covered := map[RuleGroup]bool{}
	for _, target := range targets {
		for _, group := range target.Groups {
			covered[group] = true
		}
	}
	for _, id := range catalog.IDs() {
		if !covered[catalog[id].Group] {
			continue
		}
		found := 0
		for _, table := range rendered {
			if strings.Contains(table, "`"+id+"`") {
				found++
			}
		}
		if found != 1 {
			t.Errorf("rule %q appears in %d generated tables, want exactly 1", id, found)
		}
	}
}
