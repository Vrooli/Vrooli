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
  --color-accent: #0e7490;
  --font-sans: ui-sans-serif, system-ui, sans-serif;
  --font-mono: ui-monospace, "SFMono-Regular", monospace;
  color-scheme: light;
}
`

// BuildFixCandidates evaluates the deterministic fixers for the requested rules
// (or every fixer-backed rule when ruleIDs is empty). When apply is true the
// edits are written to disk and the returned candidates carry Applied=true. It
// returns the candidates plus human-readable messages (e.g. which requested
// rules have no deterministic fixer). The fixer set is derived from the rule
// registry, so a fixer cannot exist without a rule that advertises it.
func BuildFixCandidates(root string, ruleIDs []string, apply bool) ([]Candidate, []string, error) {
	requested := map[string]bool{}
	for _, id := range ruleIDs {
		requested[id] = true
	}
	wantAll := len(requested) == 0

	var candidates []Candidate
	var messages []string

	// Registry order so previews/applies are deterministic.
	for _, ruleID := range fixableRuleIDs {
		if !wantAll && !requested[ruleID] {
			continue
		}
		cand, ok, err := fixerByID[ruleID](root, apply)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			cand.RuleID = ruleID
			candidates = append(candidates, cand)
		}
	}

	// Surface explicitly-requested rules that have no deterministic fixer (or
	// nothing to fix right now).
	for id := range requested {
		if _, ok := fixerByID[id]; !ok {
			messages = append(messages, fmt.Sprintf("rule %q has no deterministic auto-fix (needs a human/design decision or an assigned brand)", id))
		}
	}
	if len(candidates) == 0 && len(messages) == 0 {
		messages = append(messages, "no deterministic branding auto-fixes available for this scenario")
	}
	return candidates, messages, nil
}

// ruleFiresIsolated reports whether the single rule ruleID currently fires for
// the scenario at root, respecting its surface requirements. It evaluates ONLY
// that rule and never computes AutofixAvailable, so fixers can gate on it without
// recursing back into the scan→fixer-preview loop.
func ruleFiresIsolated(root, ruleID string) bool {
	c := newScanContext("", root)
	for _, spec := range specs {
		if spec.id != ruleID {
			continue
		}
		if !c.hasAll(spec.surfaces) {
			return false
		}
		_, fired := spec.eval(c)
		return fired
	}
	return false
}

// loadIndexHTML reads ui/index.html fresh from disk.
func loadIndexHTML(root string) (string, bool) { return readFile(root, indexHTMLRel) }

// writeIndexHTML writes ui/index.html.
func writeIndexHTML(root, content string) error {
	abs := filepath.Join(root, filepath.FromSlash(indexHTMLRel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		return fmt.Errorf("create ui dir: %w", err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write index.html: %w", err)
	}
	return nil
}

// fixColorSystem creates a baseline design-token file when the scenario has none.
// It never overwrites an existing token file (that would clobber a real design
// system), so it only fires for scenarios missing tokens entirely. An existing
// but incomplete token file is therefore NOT auto-fixable (advertised honestly).
func fixColorSystem(root string, apply bool) (Candidate, bool, error) {
	if !ruleFiresIsolated(root, "has-color-system") {
		return Candidate{}, false, nil
	}
	rel := designSystemCSSRel
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if _, err := os.Stat(abs); err == nil {
		// File exists but is incomplete — re-balancing real tokens is not a
		// create-only edit, so leave it to a human/brand-projection path.
		return Candidate{}, false, nil
	}
	cand := Candidate{
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

// fixFavicon derives a favicon from an existing logo asset (self-contained) and
// references it from ui/index.html. When no logo exists there is nothing
// deterministic to derive, so it proposes no candidate (autofix advertised as
// unavailable) and the finding stays guidance-only.
func fixFavicon(root string, apply bool) (Candidate, bool, error) {
	if !ruleFiresIsolated(root, "has-favicon") {
		return Candidate{}, false, nil
	}
	logoRel, ok := anyFileMatches(root,
		[]string{"ui/public", "ui/src/assets", "public", "assets", "ui/public/brand"},
		[]string{"logo.*", "logo-*.*", "*-logo.*"},
	)
	if !ok {
		return Candidate{}, false, nil // no source to derive from — guidance only
	}
	ext := filepath.Ext(logoRel)
	faviconRel := "ui/public/favicon" + ext
	cand := Candidate{
		FilePath:    faviconRel,
		Description: fmt.Sprintf("Derive %s from the existing logo (%s) and reference it from ui/index.html.", faviconRel, logoRel),
		Before:      "",
		After:       fmt.Sprintf("<copied from %s>", logoRel),
	}
	if apply {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(logoRel)))
		if err != nil {
			return Candidate{}, false, fmt.Errorf("read logo: %w", err)
		}
		abs := filepath.Join(root, filepath.FromSlash(faviconRel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
			return Candidate{}, false, fmt.Errorf("create public dir: %w", err)
		}
		if err := os.WriteFile(abs, data, 0o600); err != nil {
			return Candidate{}, false, fmt.Errorf("write favicon: %w", err)
		}
		injectFaviconLink(root, faviconRel, ext)
		cand.Applied = true
	}
	return cand, true, nil
}

// injectFaviconLink adds a <link rel="icon"> for the derived favicon when the
// head has none. Best-effort: a missing index.html is fine (the asset alone
// clears the rule).
func injectFaviconLink(root, faviconRel, ext string) {
	content, ok := loadIndexHTML(root)
	if !ok {
		return
	}
	if headReferencesIcon(parseHead(content, true)) {
		return
	}
	typ := "image/png"
	switch ext {
	case ".svg":
		typ = "image/svg+xml"
	case ".ico":
		typ = "image/x-icon"
	}
	href := "/" + filepath.Base(faviconRel)
	line := `<link rel="icon" type="` + typ + `" href="` + href + `" />`
	updated := injectBeforeHeadClose(content, []string{line})
	if updated != content {
		_ = writeIndexHTML(root, updated)
	}
}
