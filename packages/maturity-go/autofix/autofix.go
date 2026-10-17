// Package autofix provides a shared, domain-neutral auto-fix orchestrator for
// health-scenario providers. Providers register per-rule Fixers; the registry
// drives preview/apply with a dry-run-default contract and a stable idempotency
// guarantee: Apply is Preview-then-write, and re-running over an already-fixed
// tree yields no further candidates (each Fixer re-checks state in Preview and
// CanFix). The package owns no domain rules — every concrete edit lives in a
// provider-supplied Fixer.
package autofix

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/maturity-go/assessment"
)

// FixClass classifies whether a rule's findings can be auto-remediated.
type FixClass string

const (
	// FixClassAutofix marks a rule whose findings a Fixer can remediate.
	FixClassAutofix FixClass = "autofix"
	// FixClassDetectionOnly marks a rule that can only report, not remediate.
	FixClassDetectionOnly FixClass = "detection_only"
)

// Autofixable reports whether the class permits deterministic remediation.
func (c FixClass) Autofixable() bool { return c == FixClassAutofix }

// Candidate is a single proposed (or applied) file edit.
type Candidate struct {
	RuleID      string
	FilePath    string
	Description string
	Before      string
	After       string
	Applied     bool
}

// Fixer is a per-rule remediation registered by a provider. Preview computes the
// edits a rule would make without writing; CanFix reports whether a specific
// finding (located at findingPath, possibly empty) can currently be remediated.
type Fixer struct {
	RuleID  string
	Preview func(root string) ([]Candidate, error)
	CanFix  func(root, findingPath string) bool
}

// Registry holds a provider's fixers and drives preview/apply.
type Registry struct {
	fixers []Fixer
}

// NewRegistry builds a registry from the supplied fixers.
func NewRegistry(fixers ...Fixer) *Registry {
	return &Registry{fixers: append([]Fixer(nil), fixers...)}
}

// Preview returns the candidate edits for the requested rules (all rules when
// ruleIDs is empty), sorted deterministically by file path then rule id.
func (r *Registry) Preview(root string, ruleIDs []string) ([]Candidate, error) {
	var out []Candidate
	for _, f := range r.fixers {
		if f.Preview == nil || !wantsRule(ruleIDs, f.RuleID) {
			continue
		}
		candidates, err := f.Preview(root)
		if err != nil {
			return out, err
		}
		out = append(out, candidates...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].FilePath != out[j].FilePath {
			return out[i].FilePath < out[j].FilePath
		}
		return out[i].RuleID < out[j].RuleID
	})
	return out, nil
}

// Apply previews then writes each candidate's After content, marking it applied.
// It is dry-run by default in the sense that callers must explicitly choose
// Apply over Preview; the write contract is idempotent because Preview re-checks
// state and emits nothing once the tree is already fixed.
//
// Apply writes the previewed snapshots as-is. Providers whose fixers can target
// the *same* file from more than one rule must compose those edits themselves
// (e.g. by re-previewing between writes) rather than relying on this method,
// which would otherwise let a later same-file snapshot clobber an earlier one.
func (r *Registry) Apply(root string, ruleIDs []string) ([]Candidate, error) {
	candidates, err := r.Preview(root, ruleIDs)
	if err != nil {
		return nil, err
	}
	for i := range candidates {
		if err := os.MkdirAll(filepath.Dir(candidates[i].FilePath), 0o755); err != nil {
			return candidates, err
		}
		if err := os.WriteFile(candidates[i].FilePath, []byte(candidates[i].After), 0o644); err != nil {
			return candidates, err
		}
		candidates[i].Applied = true
	}
	return candidates, nil
}

// CanFix reports whether the named rule can currently remediate the finding at
// findingPath (which may be empty for whole-scenario rules).
func (r *Registry) CanFix(root, ruleID, findingPath string) bool {
	for _, f := range r.fixers {
		if f.RuleID != ruleID || f.CanFix == nil {
			continue
		}
		return f.CanFix(root, findingPath)
	}
	return false
}

// AutofixableCount returns how many findings report auto-fix availability. It is
// the standardized numerator for the shared auto-fixable-coverage metric.
func AutofixableCount(findings []assessment.Finding) int {
	n := 0
	for _, f := range findings {
		if f.AutofixAvailable {
			n++
		}
	}
	return n
}

func wantsRule(ruleIDs []string, ruleID string) bool {
	if len(ruleIDs) == 0 {
		return true
	}
	for _, id := range ruleIDs {
		if strings.EqualFold(strings.TrimSpace(id), ruleID) {
			return true
		}
	}
	return false
}
