package validation

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// This file holds the SVG brand-asset safety rule. Brand SVGs (logos, favicons,
// icons) are frequently served inline (or via <img>/<object>) and can carry
// executable content: a <script> element, on*= event handlers, a javascript:
// URL, or a <foreignObject> (an HTML-injection surface). brand-manager both
// generates and imports SVGs, so it validates the ones a scenario would serve.
//
// It is detect-only: stripping nodes from a hand- or tool-authored SVG can
// silently break its rendering, so remediation (re-export / sanitize) stays a
// human decision and the rule never advertises an autofix.

// brandAssetDirs are the directories a scenario keeps brand imagery in. Shared by
// the SVG-safety and webfont rules. node_modules is never one of these, so build
// dependencies are not scanned.
var brandAssetDirs = []string{"ui/public", "ui/src/assets", "public", "assets"}

var svgDangerPatterns = []struct {
	key string
	re  *regexp.Regexp
}{
	{"script_element", regexp.MustCompile(`(?i)<script\b`)},
	{"event_handler", regexp.MustCompile(`(?i)\son[a-z]+\s*=`)},
	{"javascript_url", regexp.MustCompile(`(?i)javascript:`)},
	{"foreign_object", regexp.MustCompile(`(?i)<foreignObject\b`)},
}

func ruleSVGAssetSafety(c *scanContext) (Finding, bool) {
	issues := svgSafetyIssues(c.root)
	if len(issues) == 0 {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityWarning,
		Title:                  "SVG brand asset contains active content",
		Description:            "A brand SVG carries executable/active content (a script element, event handler, javascript: URL, or foreignObject).",
		FilePath:               "ui/public",
		WhyItMatters:           "An SVG with active content, when rendered inline as a brand mark, is a script-injection (XSS) vector.",
		RecommendedRemediation: "Re-export the SVG without scripts/handlers, or sanitize it before serving.",
		Evidence:               map[string]any{"issues": issues},
	}, true
}

// svgSafetyIssues walks the brand asset directories for *.svg files and returns
// the per-file danger keys detected (stable, scenario-relative paths).
func svgSafetyIssues(root string) map[string]any {
	out := map[string]any{}
	for _, dir := range brandAssetDirs {
		base := filepath.Join(root, filepath.FromSlash(dir))
		_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if skipScanDir(d.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.ToLower(filepath.Ext(path)) != ".svg" {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			var found []string
			for _, p := range svgDangerPatterns {
				if p.re.Match(b) {
					found = append(found, p.key)
				}
			}
			if len(found) > 0 {
				rel, e := filepath.Rel(root, path)
				if e != nil {
					rel = path
				}
				out[filepath.ToSlash(rel)] = found
			}
			return nil
		})
	}
	return out
}

// skipScanDir reports whether a directory should be pruned from a brand-asset
// walk (built dependencies / VCS metadata never hold authored brand assets).
func skipScanDir(name string) bool {
	switch name {
	case "node_modules", "dist", "build", ".git":
		return true
	}
	return false
}
