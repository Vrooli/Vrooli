package signals

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Ported UI heuristics: extension list, template signatures, endpoint
// regexes, and routing detection are kept verbatim from the legacy
// collector so scores stay comparable.

var sourceExtensions = map[string]bool{
	".ts":     true,
	".tsx":    true,
	".js":     true,
	".jsx":    true,
	".vue":    true,
	".svelte": true,
}

var skipDirs = map[string]bool{
	"node_modules": true,
	"dist":         true,
	"build":        true,
	"coverage":     true,
}

var templateSignatures = []string{
	"This starter UI is intentionally minimal",
	"Replace it with your scenario-specific",
	"scenario-api-template",
	"TEMPLATE_PLACEHOLDER",
}

// apiPatterns match the ways UI code references API endpoints: direct
// fetch/axios/api calls, quoted /api/ paths, wrapper functions, base-URL
// concatenation, and template literals over a base variable.
var apiPatterns = []*regexp.Regexp{
	regexp.MustCompile("fetch\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]"),
	regexp.MustCompile("axios\\.[a-z]+\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]"),
	regexp.MustCompile("api\\.[a-z]+\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]"),
	regexp.MustCompile("[\"'`](/api/[^\"'`\\s]+)[\"'`]"),
	regexp.MustCompile("[\"'`](/health)[\"'`]"),
	regexp.MustCompile("(?:buildApiUrl|apiClient|getApiUrl|buildUrl)\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]"),
	regexp.MustCompile("(?:config\\.API_URL|API_BASE|API_URL|BASE_URL|getApiBase\\(\\))\\s*(?:\\+\\s*)?[\"'`]([^\"'`]+)[\"'`]"),
	regexp.MustCompile("`\\$\\{(?:config\\.API_URL|API_BASE|API_URL|BASE_URL|getApiBase\\(\\)|this\\.apiBase)\\}([^`]+)`"),
	regexp.MustCompile("this\\.apiBase\\s*\\+\\s*[\"'`]([^\"'`]+)[\"'`]"),
}

var (
	reactRouterImport   = regexp.MustCompile(`from\s+['"]react-router`)
	reactRouteComponent = regexp.MustCompile(`<Route\s+`)
	viewTypePattern     = regexp.MustCompile(`type\s+\w*[Vv]iew\w*\s*=\s*["']`)
	lazyLoadPattern     = regexp.MustCompile(`lazy\s*\(\s*\(\)\s*=>\s*import\s*\(`)
	quotedViewOption    = regexp.MustCompile(`["'][a-z-]+["']`)
)

// uiCollector scans the UI source tree with the legacy heuristics.
type uiCollector struct{}

func (uiCollector) Name() string { return "ui" }

func (uiCollector) Collect(snap *Snapshot) error {
	srcDir := uiSourceDir(snap.Root)
	if srcDir == "" {
		// No UI sources: not collected, not an error.
		return nil
	}

	entry := findAppEntryPoint(snap.Root)
	routing := detectRouting(snap.Root, entry)
	api := extractAPIEndpoints(srcDir)

	snap.UI = UISignals{
		Collected:      true,
		IsTemplate:     detectTemplateUI(entry),
		FileCount:      countSourceFiles(srcDir),
		ComponentCount: countSourceFiles(filepath.Join(srcDir, "components")),
		PageCount: max(
			countSourceFiles(filepath.Join(srcDir, "pages")),
			countSourceFiles(filepath.Join(srcDir, "views")),
		),
		APIEndpoints:    api.total,
		APIBeyondHealth: api.beyondHealth,
		HasRouting:      routing.hasRouting,
		RouteCount:      routing.routeCount,
		TotalLOC:        totalLOC(srcDir),
	}
	return nil
}

// uiSourceDir prefers ui/src/ when it directly contains source files,
// falling back to a flat ui/ directory.
func uiSourceDir(root string) string {
	srcPath := filepath.Join(root, "ui", "src")
	if entries, err := os.ReadDir(srcPath); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && sourceExtensions[filepath.Ext(entry.Name())] {
				return srcPath
			}
		}
	}

	flatPath := filepath.Join(root, "ui")
	if _, err := os.Stat(flatPath); err == nil {
		return flatPath
	}
	return ""
}

func findAppEntryPoint(root string) string {
	candidates := []string{
		"ui/src/App.tsx",
		"ui/src/App.jsx",
		"ui/src/App.ts",
		"ui/src/App.js",
		"ui/App.tsx",
		"ui/App.jsx",
		"ui/src/main.tsx",
		"ui/src/main.jsx",
	}
	for _, candidate := range candidates {
		path := filepath.Join(root, candidate)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// detectTemplateUI flags entry files that still carry generated-starter
// signatures, or tiny files self-describing as minimal.
func detectTemplateUI(appEntryPoint string) bool {
	if appEntryPoint == "" {
		return false
	}
	content, err := os.ReadFile(appEntryPoint)
	if err != nil {
		return false
	}

	text := string(content)
	for _, sig := range templateSignatures {
		if strings.Contains(text, sig) {
			return true
		}
	}
	lines := strings.Count(text, "\n") + 1
	return lines < 50 && strings.Contains(text, "minimal")
}

type routingInfo struct {
	hasRouting bool
	routeCount int
}

// detectRouting checks the app entry for react-router <Route> usage,
// state-based view unions, and lazy-loaded pages, then falls back to
// dedicated routing files.
func detectRouting(root, appEntryPoint string) routingInfo {
	if appEntryPoint != "" {
		if content, err := os.ReadFile(appEntryPoint); err == nil {
			text := string(content)

			if reactRouterImport.MatchString(text) {
				if n := len(reactRouteComponent.FindAllString(text, -1)); n > 0 {
					return routingInfo{true, n}
				}
			}
			if viewTypePattern.MatchString(text) {
				if n := len(quotedViewOption.FindAllString(text, -1)); n >= 2 {
					return routingInfo{true, n}
				}
			}
			if n := len(lazyLoadPattern.FindAllString(text, -1)); n >= 2 {
				return routingInfo{true, n}
			}
		}
	}

	routingFiles := []string{
		"ui/src/routes.tsx",
		"ui/src/routes.ts",
		"ui/src/router.tsx",
		"ui/src/router.ts",
		"ui/src/App.tsx",
	}
	for _, rf := range routingFiles {
		path := filepath.Join(root, rf)
		if path == appEntryPoint {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		text := string(content)
		if reactRouterImport.MatchString(text) {
			if n := len(reactRouteComponent.FindAllString(text, -1)); n > 0 {
				return routingInfo{true, n}
			}
		}
	}
	return routingInfo{false, 0}
}

type apiInfo struct {
	total        int
	beyondHealth int
}

// extractAPIEndpoints counts unique endpoint strings across UI sources;
// beyondHealth excludes /health and /status references.
func extractAPIEndpoints(srcDir string) apiInfo {
	endpoints := map[string]bool{}
	walkSources(srcDir, func(path string) {
		content, err := os.ReadFile(path)
		if err != nil {
			return
		}
		text := string(content)
		for _, pattern := range apiPatterns {
			for _, match := range pattern.FindAllStringSubmatch(text, -1) {
				if len(match) > 1 {
					endpoints[match[1]] = true
				}
			}
		}
	})

	info := apiInfo{total: len(endpoints)}
	for ep := range endpoints {
		if !strings.Contains(ep, "/health") && !strings.Contains(ep, "/status") {
			info.beyondHealth++
		}
	}
	return info
}

func countSourceFiles(dir string) int {
	count := 0
	walkSources(dir, func(string) { count++ })
	return count
}

func totalLOC(dir string) int {
	total := 0
	walkSources(dir, func(path string) {
		data, err := os.ReadFile(path)
		if err != nil {
			return
		}
		total += countLines(data)
	})
	return total
}

func countLines(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	n := bytes.Count(data, []byte("\n"))
	if data[len(data)-1] != '\n' {
		n++
	}
	return n
}

// walkSources visits every source-extension file under dir, skipping
// vendored/generated trees. Walk errors are ignored: a partial scan is
// better than no signal.
func walkSources(dir string, fn func(path string)) {
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if sourceExtensions[filepath.Ext(d.Name())] {
			fn(path)
		}
		return nil
	})
}
