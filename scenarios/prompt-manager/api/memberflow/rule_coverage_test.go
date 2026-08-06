package memberflow

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestEveryCataloguedRuleIsNamedByATest fails when a catalogued rule id appears
// in no test file.
//
// At the start of this plan 16 of 123 rules were named by no test anywhere in
// the module, and the plan-of-record family had 15 of them. A rule with no test
// is a rule nobody has ever seen fire: its silence on a clean tree is
// indistinguishable from its being dead, which is why Phase 3's screens needed
// three questions instead of one. This gate stops the gap from reopening.
func TestEveryCataloguedRuleIsNamedByATest(t *testing.T) {
	catalog, err := DefaultRuleCatalog()
	if err != nil {
		t.Fatalf("build catalog: %v", err)
	}

	named := namedRuleIDs(t)
	var missing, unexpected []string
	for _, id := range catalog.IDs() {
		if named[id] {
			if _, recorded := untestedAtPlanStart[id]; recorded {
				unexpected = append(unexpected, id)
			}
			continue
		}
		if _, allowed := untestedAtPlanStart[id]; allowed {
			continue
		}
		missing = append(missing, id)
	}
	sort.Strings(missing)
	for _, id := range missing {
		t.Errorf("catalogued rule %q is named by no test, and is not in the recorded debt list. A new rule needs a test.", id)
	}
	// The list may only shrink. A rule that gained a test must leave it, or the
	// list stops describing the real debt and quietly re-admits gaps.
	sort.Strings(unexpected)
	for _, id := range unexpected {
		t.Errorf("rule %q now has a test but is still listed in untestedAtPlanStart; remove it from the list", id)
	}
	// A listed id that is no longer catalogued describes no debt. Phase 3
	// deleted two rules whose entries lingered here; without this check the
	// list slowly stops meaning anything.
	for id := range untestedAtPlanStart {
		if _, ok := catalog[id]; !ok {
			t.Errorf("untestedAtPlanStart lists %q, which is not a catalogued rule; remove it", id)
		}
	}
	t.Logf("rule-test debt: %d catalogued rules still named by no test", len(untestedAtPlanStart))
}

// untestedAtPlanStart is the recorded set of catalogued rules that no test
// names. It is a ratchet, not an excuse: a rule may only leave this list, and a
// newly catalogued rule may not join it, so the debt is visible and can only
// shrink.
//
// The plan recorded 16 rules here. The real number, measured reproducibly, is
// larger: the graph family's completeness, docs, and prompt rules are almost
// entirely unnamed by any test. See the plan log entry correcting that reading.
var untestedAtPlanStart = map[string]struct{}{
	// Empty. Every catalogued rule is now named by at least one test. Keep this
	// map rather than deleting it: a new rule arriving without a test fails the
	// gate above, and the only way to land one is to add it here deliberately
	// with a reason, which is a decision someone must make on purpose.
}

// TestEveryEmittedIDIsNamedByATest is the stronger form from step 6: a check
// that declares several ids must have each of them exercised, not just the id
// its registration is keyed on.
func TestEveryEmittedIDIsNamedByATest(t *testing.T) {
	registry, err := DefaultRuleRegistry()
	if err != nil {
		t.Fatalf("build registry: %v", err)
	}
	named := namedRuleIDs(t)
	for _, rule := range registry.Rules() {
		emitted := ruleEmits(rule)
		if len(emitted) < 2 {
			continue
		}
		for _, id := range emitted {
			if named[id] {
				continue
			}
			// Same recorded debt as the catalogue gate. A multi-id check must
			// eventually have each declared id exercised, not just the one its
			// registration is keyed on.
			if _, allowed := untestedAtPlanStart[id]; allowed {
				continue
			}
			t.Errorf("rule %q declares it emits %q, but no test names that id", rule.ID(), id)
		}
	}
}

// namedRuleIDs collects every rule id mentioned in a test file across the api
// module. Naming is a weaker signal than firing, but it is checkable without
// running every rule, and a rule no test even mentions certainly has no
// behavioral coverage.
func namedRuleIDs(t *testing.T) map[string]bool {
	t.Helper()
	apiRoot := filepath.Clean(filepath.Join(memberflowPackageDir(t), ".."))
	named := map[string]bool{}
	err := filepath.WalkDir(apiRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		// This file lists rule ids as recorded debt. Counting its own list as
		// coverage would make every listed rule look tested.
		if entry.Name() == "rule_coverage_test.go" {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(body)
		catalog, err := DefaultRuleCatalog()
		if err != nil {
			return err
		}
		for _, id := range catalog.IDs() {
			if strings.Contains(text, `"`+id+`"`) {
				named[id] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk api tree: %v", err)
	}
	return named
}

func memberflowPackageDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cwd: %v", err)
	}
	return wd
}
