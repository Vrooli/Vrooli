package autofix

import (
	"os"
	"path/filepath"
	"strings"

	"ui-health/internal/uiinterop"
)

// Interop rule codes ui-health can auto-remediate. Only the *safe* mechanical
// subset is autofixed; every other interop rule is detection-only (declared
// detection_only/manual in maturity.json). Keep this list in lockstep with the
// maturity.json declarations and FixClassFor below — the ConsistencyWarnings
// check enforces it.
const (
	// RuleInteropHScreen rewrites viewport-relative full-size utilities
	// (h-screen→h-full, w-screen→w-full, 100vh/100vw→100%) to container-relative
	// equivalents. Pure token substitution, format-preserving, idempotent.
	RuleInteropHScreen = "interop_h_screen"
	// RuleInteropProtectiveComments inserts an INTEROP-CRITICAL banner comment at
	// the top of a vite config / main entry that is missing one. Additive,
	// format-preserving, idempotent.
	RuleInteropProtectiveComments = "interop_protective_comments"
)

const interopCriticalBanner = "// INTEROP-CRITICAL: interop-sensitive configuration below — do not remove without checking host-frame embedding."

// viewportReplacements maps each banned viewport utility to its container-relative
// replacement. The h-screen/w-screen substitutions also cover the min-* variants
// (min-h-screen→min-h-full) because they are substrings.
var viewportReplacements = [][2]string{
	{"h-screen", "h-full"},
	{"w-screen", "w-full"},
	{"100vh", "100%"},
	{"100vw", "100%"},
}

// registerInteropFixers adds the safe-subset interop fixers to the registry. It
// is called from New (after the manifest fixers) so the registry is the single
// ui-health auto-fix entrypoint across all check groups.
func (f *Fixer) interopFixers() []interopFixerSpec {
	return []interopFixerSpec{
		{ruleID: RuleInteropHScreen, rewrite: rewriteViewportUnits},
		{ruleID: RuleInteropProtectiveComments, rewrite: insertProtectiveComment},
	}
}

type interopFixerSpec struct {
	ruleID  string
	rewrite func(content string) (string, bool) // returns (newContent, changed)
}

// previewInterop re-derives the offending files from the interop engine (the same
// authority that detects the violations), reads each file, applies the rule's
// rewrite, and emits a whole-file Before/After candidate when the content
// changes. Driving off RunAll keeps detection and remediation on one source of
// truth and makes the fix idempotent (once rewritten, the rule no longer flags
// the file, so the next preview yields nothing).
func (f *Fixer) previewInterop(spec interopFixerSpec) func(root string) ([]Candidate, error) {
	return func(root string) ([]Candidate, error) {
		seen := map[string]bool{}
		var out []Candidate
		for _, abs := range f.interopFiles(root, spec.ruleID) {
			if seen[abs] {
				continue
			}
			seen[abs] = true
			data, err := os.ReadFile(abs)
			if err != nil {
				continue
			}
			before := string(data)
			after, changed := spec.rewrite(before)
			if !changed {
				continue
			}
			out = append(out, Candidate{
				RuleID:      spec.ruleID,
				FilePath:    abs,
				Description: interopFixDescription(spec.ruleID, abs),
				Before:      before,
				After:       after,
			})
		}
		return out, nil
	}
}

// canFixInterop scopes a preview to a single finding path (absolute file path the
// finding points at), so the runtime AutofixAvailable flag never claims a no-op.
func (f *Fixer) canFixInterop(spec interopFixerSpec) func(root, findingPath string) bool {
	preview := f.previewInterop(spec)
	return func(root, findingPath string) bool {
		candidates, err := preview(root)
		if err != nil || len(candidates) == 0 {
			return false
		}
		if findingPath == "" {
			return true
		}
		for _, c := range candidates {
			if c.FilePath == findingPath {
				return true
			}
		}
		return false
	}
}

// interopFiles returns the absolute paths the named interop rule flagged for the
// scenario at root, derived from a live RunAll.
func (f *Fixer) interopFiles(root, ruleID string) []string {
	results := uiinterop.RunAll(root, filepath.Base(root))
	var paths []string
	for _, r := range results {
		if r.RuleID != ruleID || r.Passed || r.Skipped {
			continue
		}
		for _, v := range r.Violations {
			if v.FilePath == "" {
				continue
			}
			abs := v.FilePath
			if !filepath.IsAbs(abs) {
				abs = filepath.Join(root, filepath.FromSlash(v.FilePath))
			}
			paths = append(paths, abs)
		}
	}
	return paths
}

func rewriteViewportUnits(content string) (string, bool) {
	out := content
	for _, r := range viewportReplacements {
		out = strings.ReplaceAll(out, r[0], r[1])
	}
	return out, out != content
}

func insertProtectiveComment(content string) (string, bool) {
	if strings.Contains(content, "INTEROP-CRITICAL") {
		return content, false
	}
	if content == "" {
		return interopCriticalBanner + "\n", true
	}
	return interopCriticalBanner + "\n" + content, true
}

func interopFixDescription(ruleID, abs string) string {
	switch ruleID {
	case RuleInteropHScreen:
		return "Rewrite viewport-relative sizing to container-relative (h-screen→h-full, w-screen→w-full, 100vh/100vw→100%) in " + abs + "."
	case RuleInteropProtectiveComments:
		return "Insert an INTEROP-CRITICAL banner comment at the top of " + abs + "."
	case RuleStandardTSConfigStrict:
		return `Enable TypeScript strict mode (flip "strict": false to true) in ` + abs + "."
	default:
		return "Apply remediation to " + abs + "."
	}
}
