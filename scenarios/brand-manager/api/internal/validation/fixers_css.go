package validation

import (
	"os"
	"path/filepath"
)

// This file holds the deterministic CSS fixers. reduced-motion is the one
// self-contained auto-fix among the asset/accessibility additions: the reduce
// block is identical for every scenario, so it can be appended verbatim.

const reducedMotionCSS = `
/* brand-manager: respect prefers-reduced-motion for users who request it */
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
    scroll-behavior: auto !important;
  }
}
`

// fixReducedMotion appends a prefers-reduced-motion reduce block to the design
// tokens. It only fires when the design-tokens file already exists, so it never
// converts a no-tokens scenario (which has-color-system can create wholesale)
// into a present-but-incomplete one. When the tokens file is absent there is no
// canonical stylesheet to append to deterministically, so it proposes nothing and
// the finding stays honestly guidance-only.
func fixReducedMotion(root string, apply bool) (Candidate, bool, error) {
	if !ruleFiresIsolated(root, "reduced-motion-support") {
		return Candidate{}, false, nil
	}
	existing, ok := readFile(root, designSystemCSSRel)
	if !ok {
		return Candidate{}, false, nil // no canonical stylesheet to append to
	}
	updated := existing + reducedMotionCSS
	cand := Candidate{
		FilePath:    designSystemCSSRel,
		Description: "Add an @media (prefers-reduced-motion: reduce) block so shipped animations are neutralized for users who request reduced motion.",
		Before:      existing,
		After:       updated,
	}
	if apply {
		abs := filepath.Join(root, filepath.FromSlash(designSystemCSSRel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
			return Candidate{}, false, err
		}
		if err := os.WriteFile(abs, []byte(updated), 0o600); err != nil {
			return Candidate{}, false, err
		}
		cand.Applied = true
	}
	return cand, true, nil
}
