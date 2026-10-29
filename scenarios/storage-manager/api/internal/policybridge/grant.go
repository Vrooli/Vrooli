package policybridge

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// A deletion grant is layered on top of the declared rules as the most
// authoritative disposition, so an operator-approved, time-boxed grant turns a
// root that otherwise "needs the owner's confirmation" into one the agent may
// delete unattended.
const priorityDeletionGrant = priorityStorageRoot + 1

// GrantRule is one active operator-approved deletion authority for one root.
// policybridge keeps its own shape so it does not depend on the grant store.
type GrantRule struct {
	Root      string
	GrantID   string
	Approver  string
	ExpiresAt time.Time
}

// GrantSource yields the approved, unexpired deletion grants at a moment.
type GrantSource interface {
	ActiveGrantRoots(ctx context.Context, now time.Time) ([]GrantRule, error)
}

// applyGrants layers approved deletion grants over the declared rules as
// time-boxed "allow" rules and re-merges. A grant may open one exact location
// inside data an owner declares protected (deny) — the operator saw that
// protection in the ask and approved this path — but never the protected root
// itself: changing what an owner protects is a declaration edit, not a grant.
// A grant over a tree that holds a protected root still cannot delete it, since
// the runtime lets a stricter nested declaration win. The agent-policy hard
// floor (the filesystem root, the home and repository roots, .git) is the
// ultimate backstop and cannot be opened by any published rule.
func applyGrants(base []PathRule, grants []GrantRule) ([]PathRule, []string) {
	if len(grants) == 0 {
		return base, nil
	}
	var warnings []string
	out := append([]PathRule(nil), base...)
	for _, grant := range grants {
		root := filepath.Clean(strings.TrimSpace(grant.Root))
		if root == "" || !filepath.IsAbs(root) {
			continue
		}
		reason := fmt.Sprintf("operator deletion grant %s approved by %s until %s", grant.GrantID, grant.Approver, grant.ExpiresAt.UTC().Format(time.RFC3339))
		if deny, protected := denyingRule(base, root); protected {
			if filepath.Clean(deny.Root) == root {
				warnings = append(warnings, fmt.Sprintf("deletion grant %s for %s ignored: it is the root %s declares protected; change the declaration instead", grant.GrantID, root, deny.Source))
				continue
			}
			reason += fmt.Sprintf(", inside %s which %s protects", deny.Root, deny.Source)
		}
		out = append(out, PathRule{
			Root:     root,
			Action:   "allow",
			Reason:   reason,
			Source:   "deletion-grant/" + grant.GrantID,
			priority: priorityDeletionGrant,
		})
	}
	return mergeRules(out), warnings
}

// ProtectingDeclaration names the declaration that protects path, if any, and
// whether path is that protected root itself, which no grant can open.
func ProtectingDeclaration(rules []PathRule, path string) (source string, protectedRoot string, isRoot bool) {
	deny, protected := denyingRule(rules, filepath.Clean(path))
	if !protected {
		return "", "", false
	}
	return deny.Source, deny.Root, filepath.Clean(deny.Root) == filepath.Clean(path)
}

// denyingRule returns the nearest declared rule that denies deletion at or above
// root, so a grant inside protected data is refused rather than silently
// widening it.
func denyingRule(base []PathRule, root string) (PathRule, bool) {
	var best PathRule
	found := false
	for _, rule := range base {
		if rule.Action != "deny" {
			continue
		}
		if pathWithin(root, rule.Root) && (!found || len(rule.Root) > len(best.Root)) {
			best, found = rule, true
		}
	}
	return best, found
}

// pathWithin reports whether path is root or lies under it.
func pathWithin(path, root string) bool {
	if root == "" || path == "" {
		return false
	}
	path, root = filepath.Clean(path), filepath.Clean(root)
	if path == root {
		return true
	}
	if !strings.HasSuffix(root, string(filepath.Separator)) {
		root += string(filepath.Separator)
	}
	return strings.HasPrefix(path, root)
}
