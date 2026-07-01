package checks

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
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
	for _, rel := range webmanifestCandidates {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		var doc map[string]any
		if json.Unmarshal(data, &doc) != nil {
			return manifestRead{rel: rel, dir: filepath.ToSlash(filepath.Dir(rel))}, true
		}
		return manifestRead{data: doc, rel: rel, dir: filepath.ToSlash(filepath.Dir(rel))}, true
	}
	return manifestRead{}, false
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
