package validation

import (
	"fmt"
	"os"
	"path/filepath"
)

// Candidate is a single proposed (dry-run) or applied deterministic edit,
// mirroring the shared scenario-validation FixCandidate shape.
type Candidate struct {
	RuleID      string
	FilePath    string
	Description string
	Before      string
	After       string
	Applied     bool
}

// defaultDesignSystemCSS is a complete, WCAG-AA-passing baseline token set. It is
// only ever WRITTEN when a scenario has no design tokens at all, so applying it
// is non-destructive (create-only) and idempotent (a second run finds the file
// present and proposes nothing).
const defaultDesignSystemCSS = `:root {
  --color-background: #ffffff;
  --color-surface: #f1f5f9;
  --color-foreground: #0f172a;
  --color-primary: #1d4ed8;
  --color-primary-foreground: #ffffff;
  --color-accent: #0891b2;
  --font-sans: Inter, ui-sans-serif, system-ui, sans-serif;
  --font-mono: "JetBrains Mono", ui-monospace, monospace;
  color-scheme: light;
}
`

// deterministicFixers maps a rule id to the fixer that can remediate it without
// human/AI judgment. Rules absent here are reported as non-fixable (the provider
// omits them from candidates per the ScenarioValidationService contract).
var deterministicFixers = map[string]func(root string, apply bool) (Candidate, bool, error){
	"has-color-system": fixColorSystem,
}

// BuildFixCandidates evaluates the deterministic fixers for the requested rules
// (or every supported rule when ruleIDs is empty). When apply is true the edits
// are written to disk and the returned candidates carry Applied=true. It returns
// the candidates plus human-readable messages (e.g. which rules have no fixer).
func BuildFixCandidates(root string, ruleIDs []string, apply bool) ([]Candidate, []string, error) {
	requested := map[string]bool{}
	for _, id := range ruleIDs {
		requested[id] = true
	}
	wantAll := len(requested) == 0

	var candidates []Candidate
	var messages []string

	// Stable order so previews/applies are deterministic.
	for _, rule := range []string{"has-color-system"} {
		if !wantAll && !requested[rule] {
			continue
		}
		fixer := deterministicFixers[rule]
		cand, ok, err := fixer(root, apply)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			candidates = append(candidates, cand)
		}
	}

	// Surface explicitly-requested rules that have no deterministic fixer.
	for id := range requested {
		if _, ok := deterministicFixers[id]; !ok {
			messages = append(messages, fmt.Sprintf("rule %q has no deterministic auto-fix (requires brand assignment + generation)", id))
		}
	}
	if len(candidates) == 0 && len(messages) == 0 {
		messages = append(messages, "no deterministic branding auto-fixes available for this scenario")
	}
	return candidates, messages, nil
}

// fixColorSystem creates a baseline design-token file when the scenario has none.
// It never overwrites an existing token file (that would clobber a real design
// system), so it only fires for scenarios missing tokens entirely.
func fixColorSystem(root string, apply bool) (Candidate, bool, error) {
	rel := designSystemCSSRel
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if _, err := os.Stat(abs); err == nil {
		// File exists — not a create-only fixable case.
		return Candidate{}, false, nil
	}
	cand := Candidate{
		RuleID:      "has-color-system",
		FilePath:    rel,
		Description: "Create a baseline WCAG-AA color + typography token set (no existing design tokens found).",
		Before:      "",
		After:       defaultDesignSystemCSS,
	}
	if apply {
		if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
			return Candidate{}, false, fmt.Errorf("create token dir: %w", err)
		}
		if err := os.WriteFile(abs, []byte(defaultDesignSystemCSS), 0o600); err != nil {
			return Candidate{}, false, fmt.Errorf("write design tokens: %w", err)
		}
		cand.Applied = true
	}
	return cand, true, nil
}
