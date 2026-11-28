package policybridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	corestorage "github.com/vrooli/api-core/storage"
)

func TestLeftoverDeclarationsMustBeSpecificOwnedAndExplained(t *testing.T) {
	ok := Leftover{ID: "x", Owner: "o", Root: "$VROOLI_HOME/data", Match: "*_cf2-*", Rationale: "why"}
	if err := ValidateLeftover(ok); err != nil {
		t.Fatal(err)
	}
	bad := map[string]Leftover{
		"no owner":            {ID: "x", Root: ok.Root, Match: ok.Match, Rationale: "why"},
		"no rationale":        {ID: "x", Owner: "o", Root: ok.Root, Match: ok.Match},
		"match is a path":     {ID: "x", Owner: "o", Root: ok.Root, Match: "a/b-*", Rationale: "why"},
		"match is a wildcard": {ID: "x", Owner: "o", Root: ok.Root, Match: "*", Rationale: "why"},
		"match is wildcards":  {ID: "x", Owner: "o", Root: ok.Root, Match: "??*", Rationale: "why"},
		"broken glob":         {ID: "x", Owner: "o", Root: ok.Root, Match: "[x", Rationale: "why"},
		"no root":             {ID: "x", Owner: "o", Match: ok.Match, Rationale: "why"},
	}
	for name, l := range bad {
		if ValidateLeftover(l) == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// The repository's own declaration file is valid, every entry resolves on this
// host, and each becomes an allow rule that matches generated names.
func TestTheRepositorysLeftoverDeclarationsLoadAndBecomeNameMatchingAllowRules(t *testing.T) {
	repo := repoRootForTest(t)
	leftovers, err := LoadLeftovers(repo)
	if err != nil || len(leftovers) == 0 {
		t.Fatalf("leftovers = %v, %v", leftovers, err)
	}
	home := t.TempDir()
	inputs := corestorage.GovernedRootInputs{Home: home, RuntimeHome: filepath.Join(home, ".vrooli"), RepoRoot: filepath.Join(home, "Vrooli")}
	var warnings []string
	rules, err := disposableRules(repo, inputs, &warnings)
	if err != nil || len(warnings) > 0 || len(rules) != len(leftovers) {
		t.Fatalf("rules=%d leftovers=%d warnings=%v err=%v", len(rules), len(leftovers), warnings, err)
	}
	for _, rule := range rules {
		if rule.Action != "allow" || rule.Match == "" || !filepath.IsAbs(rule.Root) || !strings.HasPrefix(rule.Source, "disposable-leftovers/") || rule.Owner == "" {
			t.Errorf("rule %+v", rule)
		}
	}
	// Two declarations on one parent keep both: the merge keys on the name too.
	merged := mergeRules(rules)
	if len(merged) != len(rules) {
		t.Errorf("merge dropped declarations on a shared parent: %d of %d", len(merged), len(rules))
	}
}

func TestMissingOrBrokenLeftoverFileIsHandledSafely(t *testing.T) {
	repo := t.TempDir()
	if l, err := LoadLeftovers(repo); err != nil || l != nil {
		t.Fatalf("a repository without the file declares nothing: %v %v", l, err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(body string) {
		if err := os.WriteFile(filepath.Join(repo, DisposableFile), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for name, body := range map[string]string{
		"not json":     "{",
		"wrong schema": `{"version":2,"leftovers":[]}`,
		"duplicate id": `{"version":1,"leftovers":[{"id":"a","owner":"o","root":"$VROOLI_HOME/x","match":"y-*","rationale":"r"},{"id":"a","owner":"o","root":"$VROOLI_HOME/x","match":"z-*","rationale":"r"}]}`,
		"too broad":    `{"version":1,"leftovers":[{"id":"a","owner":"o","root":"$VROOLI_HOME/x","match":"*","rationale":"r"}]}`,
	} {
		write(body)
		if _, err := LoadLeftovers(repo); err == nil {
			t.Errorf("%s: a broken declaration was accepted, which would silently drop or widen a rule", name)
		}
	}
}

// A policy runner older than the name match ignores fields it does not know. A
// name rule written into path_rules would read there as a rule for the whole
// folder, so it must travel in its own field.
func TestNameRulesNeverShareAFieldWithPlainPathRules(t *testing.T) {
	rules := []PathRule{
		{Root: "/home/u/.vrooli/data/vrooli", Action: "deny", Reason: "durable", Source: "runtime_home/data"},
		{Root: "/home/u/.vrooli/data/vrooli", Match: "*_cf2-*", Only: []string{"*.db"}, Action: "allow", Reason: "leftover", Source: "disposable-leftovers/x"},
	}
	data, err := BuildSnapshot(time.Now(), rules, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	var snapshot struct {
		PathRules []PathRule `json:"path_rules"`
		NameRules []PathRule `json:"name_rules"`
	}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.PathRules) != 1 || snapshot.PathRules[0].Action != "deny" || snapshot.PathRules[0].Match != "" {
		t.Fatalf("path_rules = %+v", snapshot.PathRules)
	}
	if len(snapshot.NameRules) != 1 || snapshot.NameRules[0].Match != "*_cf2-*" || snapshot.NameRules[0].Action != "allow" {
		t.Fatalf("name_rules = %+v", snapshot.NameRules)
	}
}
