package checks

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"ui-health/internal/uiinterop"
)

type webAppManifest map[string]any

type manifestRead struct {
	data webAppManifest
	rel  string
	dir  string
}

func readManifestJSON(root string) (manifestRead, bool) {
	rel, ok := locateWebManifest(root)
	if !ok {
		return manifestRead{}, false
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return manifestRead{}, false
	}
	read := manifestRead{rel: rel, dir: filepath.ToSlash(filepath.Dir(rel))}
	var doc map[string]any
	if json.Unmarshal(data, &doc) == nil {
		read.data = doc
	}
	return read, true
}

// viteRootDir is the Vite publicDir: its files are served at the URL root.
const viteRootDir = "ui/public"

var (
	manifestLinkRE = regexp.MustCompile(`(?is)<link\b[^>]*>`)
	attrRE         = regexp.MustCompile(`(?is)\b(rel|href)\s*=\s*("([^"]*)"|'([^']*)')`)
)

// locateWebManifest finds the manifest the page actually links: the href of
// ui/index.html's <link rel="manifest">, resolved from the URL root onto the
// Vite publicDir (so href="public/site.webmanifest" or "/public/..." names
// ui/public/public/site.webmanifest). A page without a resolvable link falls
// back to the conventional locations.
func locateWebManifest(root string) (string, bool) {
	if html, err := os.ReadFile(filepath.Join(root, "ui", "index.html")); err == nil {
		if href, ok := manifestHref(string(html)); ok {
			if rel, ok := servedPathToSource(href); ok {
				if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err == nil && !info.IsDir() {
					return rel, true
				}
			}
		}
	}
	for _, rel := range webmanifestCandidates {
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err == nil && !info.IsDir() {
			return rel, true
		}
	}
	return "", false
}

func manifestHref(html string) (string, bool) {
	for _, tag := range manifestLinkRE.FindAllString(html, -1) {
		var rel, href string
		for _, attr := range attrRE.FindAllStringSubmatch(tag, -1) {
			value := attr[3] + attr[4]
			if strings.EqualFold(attr[1], "rel") {
				rel = value
			} else {
				href = value
			}
		}
		for _, token := range strings.Fields(strings.ToLower(rel)) {
			if token == "manifest" && strings.TrimSpace(href) != "" {
				return strings.TrimSpace(href), true
			}
		}
	}
	return "", false
}

// servedPathToSource maps a page-root URL reference to its Vite publicDir
// source file. External and data: references have no local source.
func servedPathToSource(ref string) (string, bool) {
	if strings.Contains(ref, "://") || strings.HasPrefix(ref, "//") || strings.HasPrefix(ref, "data:") {
		return "", false
	}
	if i := strings.IndexAny(ref, "?#"); i >= 0 {
		ref = ref[:i]
	}
	clean := path.Clean("/" + strings.TrimPrefix(ref, "./"))
	if clean == "/" {
		return "", false
	}
	return viteRootDir + clean, true
}

// manifestScopeCoversAppRoot reports whether a manifest's start_url and scope,
// resolved against the URL the manifest is served from, include the app root.
// A manifest under /public/ with scope "." would install an app scoped to the
// asset folder.
func manifestScopeCoversAppRoot(mf manifestRead, value string) bool {
	if strings.TrimSpace(value) == "" || !strings.HasPrefix(mf.dir+"/", viteRootDir+"/") {
		return true
	}
	u, err := url.Parse(value)
	if err != nil || u.IsAbs() {
		return true // unsafeLaunchURL judges absolute and cross-origin values
	}
	served := "/" + strings.TrimPrefix(strings.TrimPrefix(mf.dir, viteRootDir), "/")
	resolved := u.Path
	if !strings.HasPrefix(resolved, "/") {
		resolved = path.Join(served, resolved)
	}
	return path.Clean(resolved) == "/"
}

func pwaResult(ruleID, passMessage, failMessage string, violations []uiinterop.Violation) uiinterop.RuleResult {
	if len(violations) == 0 {
		return uiinterop.RuleResult{RuleID: ruleID, Passed: true, Message: passMessage}
	}
	return uiinterop.RuleResult{RuleID: ruleID, Passed: false, Message: failMessage, Violations: violations}
}

func skippedPWA(ruleID, reason string) uiinterop.RuleResult {
	return uiinterop.RuleResult{RuleID: ruleID, Skipped: true, SkipReason: reason, Message: reason + "; skipping PWA native-readiness check"}
}

func pwaViolation(ruleID, filePath, title, desc, rec string) uiinterop.Violation {
	return uiinterop.Violation{
		RuleID:         ruleID,
		Severity:       "medium",
		Title:          title,
		Description:    desc,
		FilePath:       filePath,
		Recommendation: rec,
	}
}

func stringField(m map[string]any, field string) string {
	v, _ := m[field].(string)
	return strings.TrimSpace(v)
}

func manifestAssetExists(root, manifestDir, src string) bool {
	u, err := url.Parse(src)
	if err == nil && u.IsAbs() {
		return false
	}
	src = strings.TrimPrefix(src, "/")
	var rel string
	if strings.HasPrefix(src, "ui/") {
		rel = src
	} else {
		rel = path.Clean(path.Join(manifestDir, src))
	}
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil && !info.IsDir()
}

func serviceWorkerSourceExists(root string) bool {
	for _, rel := range []string{
		"ui/public/sw.js",
		"ui/public/service-worker.js",
		"ui/src/sw.ts",
		"ui/src/sw.js",
		"ui/src/service-worker.ts",
		"ui/src/service-worker.js",
	} {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		if err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

func unsafeLaunchURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return true
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return false
	}
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func manifestURLInScope(startURL, scope string) bool {
	startPath := manifestPath(startURL)
	scopePath := manifestPath(scope)
	if scopePath == "." || scopePath == "/" || scopePath == "" {
		return true
	}
	return startPath == scopePath || strings.HasPrefix(strings.TrimPrefix(startPath, "/"), strings.TrimPrefix(scopePath, "/"))
}

func manifestPath(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	p := u.Path
	if p == "" {
		p = raw
	}
	if p == "." {
		return "."
	}
	return path.Clean("/" + strings.TrimPrefix(p, "/"))
}

func allowedDisplayOverride(value string) bool {
	switch value {
	case "fullscreen", "standalone", "minimal-ui", "browser", "window-controls-overlay", "tabbed", "borderless":
		return true
	default:
		return false
	}
}

func malformedOptionalURL(field, value string) string {
	return fmt.Sprintf("%s %q is not deployment-safe", field, value)
}
