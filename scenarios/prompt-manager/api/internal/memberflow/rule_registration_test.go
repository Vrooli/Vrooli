package memberflow

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// stubRule is a minimal Rule for exercising registration itself.
type stubRule struct {
	id       string
	emits    []string
	group    RuleGroup
	severity Severity
}

func (r stubRule) ID() string                              { return r.id }
func (r stubRule) Group() RuleGroup                        { return r.group }
func (r stubRule) DefaultSeverity() Severity               { return r.severity }
func (stubRule) AppliesTo(RuleContext) bool                { return true }
func (stubRule) Check(RuleContext) []OperatingGraphFinding { return nil }
func (r stubRule) Emits() []string                         { return r.emits }

func testCatalog(t *testing.T, ids ...string) RuleCatalog {
	t.Helper()
	entries := make([]RuleCatalogEntry, 0, len(ids))
	for _, id := range ids {
		entries = append(entries, RuleCatalogEntry{
			ID: id, Group: OperatingRuleGroupEntity, Severity: SeverityError,
			Description: "d", Actuator: "a",
		})
	}
	catalog, err := NewRuleCatalog(entries...)
	if err != nil {
		t.Fatalf("build catalog: %v", err)
	}
	return catalog
}

func TestRegistrationRejectsAnEmittedIDMissingFromTheCatalog(t *testing.T) {
	rule := stubRule{id: "known", emits: []string{"known", "not_catalogued"}, group: OperatingRuleGroupEntity, severity: SeverityError}
	_, err := NewRuleRegistryWithCatalog(testCatalog(t, "known"), rule)
	if err == nil {
		t.Fatal("a rule emitting an uncatalogued id was accepted")
	}
	if !strings.Contains(err.Error(), "not_catalogued") {
		t.Errorf("error does not name the offending id: %v", err)
	}
}

func TestRegistrationRejectsACatalogEntryClaimedByNoRule(t *testing.T) {
	rule := stubRule{id: "known", emits: []string{"known"}, group: OperatingRuleGroupEntity, severity: SeverityError}
	_, err := NewRuleRegistryWithCatalog(testCatalog(t, "known", "orphaned"), rule)
	if err == nil {
		t.Fatal("a catalog entry claimed by no rule was accepted")
	}
	if !strings.Contains(err.Error(), "orphaned") {
		t.Errorf("error does not name the unclaimed entry: %v", err)
	}
}

func TestRegistrationRejectsACatalogEntryClaimedTwice(t *testing.T) {
	a := stubRule{id: "a", emits: []string{"a", "shared"}, group: OperatingRuleGroupEntity, severity: SeverityError}
	b := stubRule{id: "b", emits: []string{"b", "shared"}, group: OperatingRuleGroupEntity, severity: SeverityError}
	_, err := NewRuleRegistryWithCatalog(testCatalog(t, "a", "b", "shared"), a, b)
	if err == nil {
		t.Fatal("one catalog entry claimed by two rules was accepted")
	}
	if !strings.Contains(err.Error(), "shared") {
		t.Errorf("error does not name the doubly-claimed entry: %v", err)
	}
}

// The shipped registry must satisfy both directions at once: every id a rule can
// emit is catalogued, and every catalogued id is claimed by exactly one rule.
func TestDefaultRegistryClaimsEveryCatalogEntryExactlyOnce(t *testing.T) {
	if _, err := DefaultRuleRegistry(); err != nil {
		t.Fatalf("the shipped registry does not satisfy catalog enforcement: %v", err)
	}
}

// TestEveryEmittedRuleIDIsCatalogued walks the package for string literals used
// as rule ids and fails when one is not in the catalog. This is the backstop for
// a check that invents an id inline instead of declaring it in Emits.
func TestEveryEmittedRuleIDIsCatalogued(t *testing.T) {
	catalog, err := DefaultRuleCatalog()
	if err != nil {
		t.Fatalf("build catalog: %v", err)
	}
	// Ids constructed by families this catalog does not own, or synthesized for
	// registry failure itself.
	exempt := map[string]bool{
		"rule_registry_invalid": true,
	}
	pattern := regexp.MustCompile(`(?:Rule:\s*"([a-z][a-z0-9_]*)"|Finding\("([a-z][a-z0-9_]*)")`)
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for _, match := range pattern.FindAllStringSubmatch(string(body), -1) {
			id := match[1]
			if id == "" {
				id = match[2]
			}
			if id == "" || exempt[id] {
				continue
			}
			if _, ok := catalog[id]; !ok {
				t.Errorf("%s emits rule id %q with no catalog entry", name, id)
			}
		}
	}
}
