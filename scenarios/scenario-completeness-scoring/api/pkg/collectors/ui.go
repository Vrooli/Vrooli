package collectors

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"scenario-completeness-scoring/pkg/scoring"
)

// Source file extensions to count
var sourceExtensions = map[string]bool{
	".ts":   true,
	".tsx":  true,
	".js":   true,
	".jsx":  true,
	".vue":  true,
	".svelte": true,
}

// Template detection patterns
var templateSignatures = []string{
	"This starter UI is intentionally minimal",
	"Replace it with your scenario-specific",
	"scenario-api-template",
	"TEMPLATE_PLACEHOLDER",
}

// API endpoint patterns
var apiPatterns = []*regexp.Regexp{
	regexp.MustCompile(`fetch\s*\(\s*["'\x60]([^"'\x60]+)["'\x60]`),
	regexp.MustCompile(`axios\.[a-z]+\s*\(\s*["'\x60]([^"'\x60]+)["'\x60]`),
	regexp.MustCompile(`api\.[a-z]+\s*\(\s*["'\x60]([^"'\x60]+)["'\x60]`),
	regexp.MustCompile(`["'\x60](/api/[^"'\x60\s]+)["'\x60]`),
	regexp.MustCompile(`["'\x60](/health)["'\x60]`),
}

// Routing patterns
var (
	reactRouterImport = regexp.MustCompile(`from\s+['"]react-router`)
	reactRouteComponent = regexp.MustCompile(`<Route\s+`)
	viewTypePattern = regexp.MustCompile(`type\s+\w*[Vv]iew\w*\s*=\s*["']`)
	lazyLoadPattern = regexp.MustCompile(`lazy\s*\(\s*\(\)\s*=>\s*import\s*\(`)
)

// collectUIMetrics gathers UI completeness metrics
// [REQ:SCS-CORE-001D] UI metrics collection
func collectUIMetrics(scenarioRoot string) *scoring.UIMetrics {
	uiSrcDir := getUiSourceDir(scenarioRoot)
	if uiSrcDir == "" {
		return nil
	}

	appEntryPoint := findAppEntryPoint(scenarioRoot)

	isTemplate := detectTemplateUI(scenarioRoot, appEntryPoint)
	fileCount := countFilesRecursive(uiSrcDir)
	componentCount := countFilesRecursive(filepath.Join(uiSrcDir, "components"))
	pageCount := maxInt(
		countFilesRecursive(filepath.Join(uiSrcDir, "pages")),
		countFilesRecursive(filepath.Join(uiSrcDir, "views")),
	)
	totalLOC := getTotalLOC(uiSrcDir)
	routing := detectRouting(scenarioRoot, appEntryPoint)
	apiIntegration := extractAPIEndpoints(uiSrcDir)

	return &scoring.UIMetrics{
		IsTemplate:      isTemplate,
		FileCount:       fileCount,
		ComponentCount:  componentCount,
		PageCount:       pageCount,
		APIEndpoints:    apiIntegration.total,
		APIBeyondHealth: apiIntegration.beyondHealth,
		HasRouting:      routing.hasRouting,
		RouteCount:      routing.routeCount,
		TotalLOC:        totalLOC,
	}
}

// getUiSourceDir determines the UI source directory
func getUiSourceDir(scenarioRoot string) string {
	srcPath := filepath.Join(scenarioRoot, "ui", "src")
	flatPath := filepath.Join(scenarioRoot, "ui")

	// Prefer ui/src/ if it exists and contains source files
	if entries, err := os.ReadDir(srcPath); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				ext := filepath.Ext(entry.Name())
				if sourceExtensions[ext] {
					return srcPath
				}
			}
		}
	}

	// Check if flat ui/ has source files
	if _, err := os.Stat(flatPath); err == nil {
		return flatPath
	}

	return ""
}

// findAppEntryPoint locates the main app entry file
func findAppEntryPoint(scenarioRoot string) string {
	patterns := []string{
		"ui/src/App.tsx",
		"ui/src/App.jsx",
		"ui/src/App.ts",
		"ui/src/App.js",
		"ui/App.tsx",
		"ui/App.jsx",
		"ui/src/main.tsx",
		"ui/src/main.jsx",
	}

	for _, pattern := range patterns {
		fullPath := filepath.Join(scenarioRoot, pattern)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}
	}

	return ""
}

// detectTemplateUI checks if UI is still the unmodified template
func detectTemplateUI(scenarioRoot string, appEntryPoint string) bool {
	if appEntryPoint == "" {
		appEntryPoint = findAppEntryPoint(scenarioRoot)
	}

	if appEntryPoint == "" {
		return false // No UI at all
	}

	content, err := os.ReadFile(appEntryPoint)
	if err != nil {
		return false
	}

	contentStr := string(content)
	lines := strings.Count(contentStr, "\n") + 1

	// Check for template signatures
	for _, sig := range templateSignatures {
		if strings.Contains(contentStr, sig) {
			return true
		}
	}

	// Small file with minimal content might be template
	if lines < 50 && strings.Contains(contentStr, "minimal") {
		return true
	}

	return false
}

// routingInfo holds routing detection results
type routingInfo struct {
	hasRouting bool
	routeCount int
}

// detectRouting detects routing usage in the UI
func detectRouting(scenarioRoot string, appEntryPoint string) routingInfo {
	if appEntryPoint == "" {
		appEntryPoint = findAppEntryPoint(scenarioRoot)
	}

	if appEntryPoint == "" {
		return routingInfo{false, 0}
	}

	content, err := os.ReadFile(appEntryPoint)
	if err != nil {
		return routingInfo{false, 0}
	}

	contentStr := string(content)

	// Check for React Router
	if reactRouterImport.MatchString(contentStr) {
		matches := reactRouteComponent.FindAllString(contentStr, -1)
		if len(matches) > 0 {
			return routingInfo{true, len(matches)}
		}
	}

	// Check for state-based views
	if viewTypePattern.MatchString(contentStr) {
		// Count view options
		viewMatches := regexp.MustCompile(`["'][a-z-]+["']`).FindAllString(contentStr, -1)
		if len(viewMatches) >= 2 {
			return routingInfo{true, len(viewMatches)}
		}
	}

	// Check for lazy loaded pages
	lazyMatches := lazyLoadPattern.FindAllString(contentStr, -1)
	if len(lazyMatches) >= 2 {
		return routingInfo{true, len(lazyMatches)}
	}

	// Also check routing files
	routingFiles := []string{
		"ui/src/routes.tsx",
		"ui/src/routes.ts",
		"ui/src/router.tsx",
		"ui/src/router.ts",
		"ui/src/App.tsx",
	}

	for _, rf := range routingFiles {
		fullPath := filepath.Join(scenarioRoot, rf)
		if fullPath == appEntryPoint {
			continue // Already checked
		}
		if content, err := os.ReadFile(fullPath); err == nil {
			contentStr := string(content)
			if reactRouterImport.MatchString(contentStr) {
				matches := reactRouteComponent.FindAllString(contentStr, -1)
				if len(matches) > 0 {
					return routingInfo{true, len(matches)}
				}
			}
		}
	}

	return routingInfo{false, 0}
}

// apiInfo holds API endpoint extraction results
type apiInfo struct {
	total        int
	beyondHealth int
}

// extractAPIEndpoints extracts API endpoints called from UI code
func extractAPIEndpoints(uiSrcDir string) apiInfo {
	if uiSrcDir == "" {
		return apiInfo{0, 0}
	}

	endpoints := make(map[string]bool)

	filepath.WalkDir(uiSrcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Skip directories we don't want to scan
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == "dist" || name == "build" || name == "coverage" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(path)
		if !sourceExtensions[ext] {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		contentStr := string(content)
		for _, pattern := range apiPatterns {
			matches := pattern.FindAllStringSubmatch(contentStr, -1)
			for _, match := range matches {
				if len(match) > 1 {
					endpoints[match[1]] = true
				}
			}
		}

		return nil
	})

	total := len(endpoints)
	beyondHealth := 0
	for ep := range endpoints {
		if !strings.Contains(ep, "/health") && !strings.Contains(ep, "/status") {
			beyondHealth++
		}
	}

	return apiInfo{total, beyondHealth}
}

// countFilesRecursive counts source files in a directory
func countFilesRecursive(dir string) int {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return 0
	}

	count := 0
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		// Skip node_modules, dist, etc.
		if strings.Contains(path, "node_modules") || strings.Contains(path, "/dist/") {
			return nil
		}

		ext := filepath.Ext(d.Name())
		if sourceExtensions[ext] {
			count++
		}
		return nil
	})

	return count
}

// getTotalLOC counts total lines of code in a directory
func getTotalLOC(dir string) int {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return 0
	}

	total := 0
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		// Skip node_modules, dist, etc.
		if strings.Contains(path, "node_modules") || strings.Contains(path, "/dist/") {
			return nil
		}

		ext := filepath.Ext(d.Name())
		if !sourceExtensions[ext] {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			total++
		}
		return nil
	})

	return total
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
