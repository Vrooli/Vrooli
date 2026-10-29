package policybridge

import (
	"strings"
	"testing"
	"time"
)

func ruleFor(rules []PathRule, root string) (PathRule, bool) {
	for _, r := range rules {
		if r.Root == root {
			return r, true
		}
	}
	return PathRule{}, false
}

func TestApplyGrantsUpgradesAskToAllow(t *testing.T) {
	base := []PathRule{
		{Root: "/home/u/.vrooli/cache/staging", Action: "ask", Source: "repo-contract storage.roots/scenario-desktop-staging", priority: priorityStorageRoot},
	}
	exp := time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)
	granted, warns := applyGrants(base, []GrantRule{{Root: "/home/u/.vrooli/cache/staging", GrantID: "g1", Approver: "op", ExpiresAt: exp}})
	if len(warns) != 0 {
		t.Fatalf("unexpected warnings: %v", warns)
	}
	rule, ok := ruleFor(granted, "/home/u/.vrooli/cache/staging")
	if !ok || rule.Action != "allow" || rule.priority != priorityDeletionGrant {
		t.Fatalf("grant must upgrade the staging root to allow, got %+v", rule)
	}
}

func TestApplyGrantsOpensOnlyTheApprovedPathInsideProtectedData(t *testing.T) {
	base := []PathRule{
		{Root: "/home/u/data", Action: "deny", Source: "owner durable data", priority: priorityOwnerEntry},
	}
	granted, warns := applyGrants(base, []GrantRule{{Root: "/home/u/data/orphan", GrantID: "g2", Approver: "op", ExpiresAt: time.Now().Add(time.Hour)}})
	if len(warns) != 0 {
		t.Fatalf("unexpected warnings: %v", warns)
	}
	rule, ok := ruleFor(granted, "/home/u/data/orphan")
	if !ok || rule.Action != "allow" || !strings.Contains(rule.Reason, "owner durable data protects") {
		t.Fatalf("approved path inside protected data = %+v, want an allow naming the protection it overrides", rule)
	}
	if rule, _ := ruleFor(granted, "/home/u/data"); rule.Action != "deny" {
		t.Fatalf("the protected root must stay denied, got %+v", rule)
	}
}

func TestApplyGrantsNeverOpensAProtectedRootItself(t *testing.T) {
	base := []PathRule{
		{Root: "/home/u/data", Action: "deny", Source: "owner durable data", priority: priorityOwnerEntry},
	}
	granted, warns := applyGrants(base, []GrantRule{{Root: "/home/u/data", GrantID: "g2", Approver: "op", ExpiresAt: time.Now().Add(time.Hour)}})
	if len(warns) == 0 {
		t.Fatal("a grant for the protected root itself must warn and be ignored")
	}
	if rule, _ := ruleFor(granted, "/home/u/data"); rule.Action != "deny" {
		t.Fatalf("the deny must survive, got %+v", rule)
	}
}

func TestProtectingDeclarationReportsTheProtectedRoot(t *testing.T) {
	base := []PathRule{{Root: "/home/u/data", Action: "deny", Source: "owner durable data"}, {Root: "/home/u/cache", Action: "allow", Source: "cache"}}
	if source, root, isRoot := ProtectingDeclaration(base, "/home/u/data/x/y"); source != "owner durable data" || root != "/home/u/data" || isRoot {
		t.Fatalf("inside = %q %q %v", source, root, isRoot)
	}
	if _, _, isRoot := ProtectingDeclaration(base, "/home/u/data/"); !isRoot {
		t.Fatal("the protected root itself must report isRoot")
	}
	if source, _, _ := ProtectingDeclaration(base, "/home/u/cache/x"); source != "" {
		t.Fatalf("allowed location reported protected by %q", source)
	}
}

func TestApplyGrantsAddsNewAllowRoot(t *testing.T) {
	exp := time.Now().Add(time.Hour)
	granted, _ := applyGrants(nil, []GrantRule{{Root: "/home/u/vrooli-git-recovery", GrantID: "g3", Approver: "op", ExpiresAt: exp}})
	rule, ok := ruleFor(granted, "/home/u/vrooli-git-recovery")
	if !ok || rule.Action != "allow" {
		t.Fatalf("grant must add an allow rule for an undeclared location, got %+v", rule)
	}
}

func TestApplyGrantsIgnoresRelativeRoot(t *testing.T) {
	granted, _ := applyGrants(nil, []GrantRule{{Root: "relative/x", GrantID: "g4", Approver: "op", ExpiresAt: time.Now().Add(time.Hour)}})
	if len(granted) != 0 {
		t.Fatalf("a non-absolute grant root must be ignored, got %+v", granted)
	}
}
