//go:build liverepo

package memberflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

func realPromptManagerStore(t *testing.T) (string, string) {
	t.Helper()
	contract, repoRoot, err := repocontract.LoadDefaultFromEnvOrCWD()
	if err != nil {
		t.Skipf("repo contract not available in this environment: %v", err)
	}
	scenarioRoot, err := contract.ScenarioRoot(repoRoot, "prompt-manager")
	if err != nil {
		t.Skipf("prompt-manager scenario root not available: %v", err)
	}
	storeDir := filepath.Join(scenarioRoot, "store")
	if _, err := os.Stat(storeDir); err != nil {
		t.Skip("real store not available in this environment")
	}
	return storeDir, repoRoot
}

// TestClassifierPurity_RegisteredClassifiers_NoProseTopicLeak is the
// belt-and-suspenders live-store check that every registered classifier
// or triage skill stays portable. The previous incarnation was a
// substring scan against a hand-curated forbidden list; that list was
// retired in favor of the broader Pillar 2 prose scanner, which catches
// every realistic coupling pattern (proven by
// non_portable_classifier_subsumption_test.go).
//
// What the test now asserts:
//
//   - Discover every -classifier and -triage skill in the live registry.
//   - Run ruleProseTopicLeak against the live store rooted at the repo.
//   - Assert no prose_topic_leak finding is owned by any classifier or
//     triage skill. (Writer-skill SKILL.md targets are governed by their
//     own writes_to[] declaration; classifiers are held to the stricter
//     no-topic-references rule by the kind-conditional join in
//     prose_scan.go.)
//
// Skipped when the real store is not present (CI sandbox / fresh
// checkouts) — the synthetic-fixture coverage in prose_scan_test.go and
// non_portable_classifier_subsumption_test.go covers the contract.
func TestClassifierPurity_RegisteredClassifiers_NoProseTopicLeak(t *testing.T) {
	storeDir, repoRoot := realPromptManagerStore(t)

	// Inventory of registered skill ids that should be portable. The
	// suffix convention is canon: any skill named *-classifier or
	// *-triage is a judgment skill and must not embed team-specific
	// topic strings.
	skillIDs, err := LoadSkillIDs(storeDir)
	if err != nil {
		t.Fatalf("LoadSkillIDs: %v", err)
	}
	classifiers := map[string]bool{}
	for id := range skillIDs {
		if strings.HasSuffix(id, "-classifier") || strings.HasSuffix(id, "-triage") {
			classifiers[id] = true
		}
	}
	if len(classifiers) == 0 {
		t.Fatalf("no -classifier or -triage skills found in registry; expected at least the three migrated by the inbox-flow refactor")
	}

	// Run the prose scanner alone — we don't care what other rules say
	// here; the contract is "no classifier owns a prose_topic_leak."
	members, err := LoadAll(storeDir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	findings := ruleProseTopicLeak(members, ValidationOptions{
		RepoRoot:  repoRoot,
		ScanRoots: []string{repoRoot},
	})

	for _, f := range findings {
		if !strings.HasPrefix(f.OwnerKey, "skill:") {
			continue
		}
		id := strings.TrimPrefix(f.OwnerKey, "skill:")
		if !classifiers[id] {
			continue
		}
		t.Errorf("classifier-or-triage skill %q owns a prose_topic_leak finding (%s, prefix=%q): %s",
			id, f.Severity, f.Prefix, f.Detail)
	}
}
