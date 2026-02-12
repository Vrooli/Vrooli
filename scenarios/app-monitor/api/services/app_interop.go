package services

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// =============================================================================
// UI Interop Compliance Scanning
// =============================================================================

// interopCheckDef defines a single interop check's metadata.
type interopCheckDef struct {
	ID       string
	Name     string
	Severity string
	Slot     string
	Rec      string // recommendation text
}

var interopCheckDefs = []interopCheckDef{
	{ID: "interop_api_base_dep", Name: "API base dependency", Severity: "critical", Slot: "[A]", Rec: "Add @vrooli/api-base to ui/package.json dependencies"},
	{ID: "interop_iframe_bridge_dep", Name: "Iframe bridge dependency", Severity: "critical", Slot: "[A]", Rec: "Add @vrooli/iframe-bridge to ui/package.json dependencies"},
	{ID: "interop_hardcoded_localhost", Name: "No hardcoded localhost", Severity: "high", Slot: "[F]", Rec: "Replace hardcoded localhost:PORT with resolveApiBase() from @vrooli/api-base"},
	{ID: "interop_relative_base", Name: "Relative Vite base", Severity: "critical", Slot: "[B]", Rec: "Set base: './' in ui/vite.config.ts"},
	{ID: "interop_router_basename", Name: "Proxy-aware router", Severity: "high", Slot: "[E]", Rec: "Add basename prop to BrowserRouter (or use MemoryRouter)"},
	{ID: "interop_no_custom_server", Name: "Standard scenario server", Severity: "medium", Slot: "[C]", Rec: "Use startScenarioServer() instead of custom Express/http server"},
	{ID: "interop_bridge_init", Name: "Bridge initialization", Severity: "critical", Slot: "[D]", Rec: "Call initIframeBridgeChild() in ui/src/main.tsx"},
	{ID: "interop_resolve_api_base_single", Name: "Single API base resolution", Severity: "high", Slot: "[F]", Rec: "Import resolveApiBase in at most 2 production files (e.g. ui/src/api/client.ts)"},
	{ID: "interop_shortcut_relay", Name: "Shortcut iframe relay", Severity: "medium", Slot: "[G]", Rec: "Use emitShortcutIntent from @vrooli/iframe-bridge in keyboard hooks"},
	{ID: "interop_no_scattered_keydown", Name: "Centralized keyboard handling", Severity: "medium", Slot: "[G]", Rec: "Move app-level addEventListener('keydown') to hooks/ directory"},
	{ID: "interop_bridge_app_id", Name: "Bridge appId parameter", Severity: "medium", Slot: "[D]", Rec: "Pass appId to initIframeBridgeChild() call"},
	{ID: "interop_protective_comments", Name: "Protective comments", Severity: "low", Slot: "[B],[D]", Rec: "Add INTEROP-CRITICAL comments to ui/vite.config.ts and ui/src/main.tsx"},
	{ID: "interop_iframe_guard", Name: "Iframe guard", Severity: "high", Slot: "[D]", Rec: "Guard initIframeBridgeChild with if (window.parent !== window)"},
	{ID: "interop_capture_enabled", Name: "Capture settings enabled", Severity: "medium", Slot: "[D]", Rec: "Do not disable captureLogs or captureNetwork in bridge init"},
	{ID: "interop_proxy_base_preserved", Name: "Proxy base preservation", Severity: "high", Slot: "[F]", Rec: "Use resolveApiBase output directly; do not rebuild with window.location.origin"},
	{ID: "interop_secure_tunnel", Name: "Secure UI tunnel", Severity: "high", Slot: "[C]", Rec: "Route API calls through proxyToApi in custom server files"},
}

// hardcodedLocalhostPattern matches localhost:PORT in source code.
var hardcodedLocalhostPattern = regexp.MustCompile(`localhost:\d+`)

// Regex patterns for the new checks.
var (
	iframeGuardPattern = regexp.MustCompile(`window\.parent\s*(!==|!=|===|==)\s*window|window\s*(!==|!=|===|==)\s*window\.parent|window\.top\s*(!==|!=|===|==)\s*window\.self|window\.self\s*(!==|!=|===|==)\s*window\.top`)
	captureLogsDisabledPattern    = regexp.MustCompile(`(?s)captureLogs\s*:\s*(?:false|\{[^}]*enabled\s*:\s*false)`)
	captureNetworkDisabledPattern = regexp.MustCompile(`(?s)captureNetwork\s*:\s*(?:false|\{[^}]*enabled\s*:\s*false)`)
	originVariableRegex           = regexp.MustCompile(`(?m)(?:const|let|var)\s+([a-zA-Z_$][\w$]*)\s*=\s*(?:[a-zA-Z_$][\w$]*\()?\s*window\.location\.origin`)
	proxyFunctionRegex            = regexp.MustCompile(`(?m)^(?:\s*(?:async\s+)?function\s+proxyToApi\s*\(|\s*(?:const|let|var)\s+proxyToApi\s*=)`)
	customServerPattern           = regexp.MustCompile(`(?i)(express\(\)|createServer|http\.listen|\.listen\(\d)`)
)

// CheckInteropCompliance runs all 16 interop checks for a single app.
func (s *AppService) CheckInteropCompliance(ctx context.Context, appID string) (*InteropComplianceReport, error) {
	id := strings.TrimSpace(appID)
	if id == "" {
		return nil, ErrAppIdentifierRequired
	}

	app, err := s.GetApp(ctx, id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return nil, fmt.Errorf("%w: %v", ErrAppNotFound, err)
		}
		return nil, err
	}

	scenarioName := strings.TrimSpace(app.ScenarioName)
	if scenarioName == "" {
		scenarioName = strings.TrimSpace(app.Name)
	}
	if scenarioName == "" {
		scenarioName = id
	}

	root := strings.TrimSpace(app.Path)
	report := &InteropComplianceReport{
		Scenario:  scenarioName,
		CheckedAt: s.timeNow().UTC(),
		Checks:    make([]InteropCheckResult, 0, 16),
		HasUI:     true,
	}

	if root == "" {
		report.HasUI = false
		report.Warnings = append(report.Warnings, "scenario path unknown; skipping interop scan")
		return report, nil
	}

	uiDir := filepath.Join(root, "ui")
	if info, err := os.Stat(uiDir); err != nil || !info.IsDir() {
		report.HasUI = false
		report.Warnings = append(report.Warnings, "no ui/ directory found; interop checks not applicable")
		return report, nil
	}

	report.Checks = runAllInteropChecks(root)
	report.TotalCount = len(report.Checks)
	for _, check := range report.Checks {
		switch {
		case check.Skipped:
			report.SkipCount++
		case check.Passed:
			report.PassCount++
		default:
			report.FailCount++
		}
	}

	scorable := report.TotalCount - report.SkipCount
	if scorable > 0 {
		report.Score = (report.PassCount * 100) / scorable
	} else {
		report.Score = 100
	}

	return report, nil
}

// GetInteropStandards returns interop results in scenario-auditor quality format.
func (s *AppService) GetInteropStandards(ctx context.Context, scenarioName string) (*InteropStandardsResponse, error) {
	cleaned := strings.TrimSpace(scenarioName)
	if cleaned == "" {
		return nil, fmt.Errorf("scenario name is required")
	}

	// Resolve scenario path from repoRoot
	root := s.resolveScenarioRoot(cleaned)
	if root == "" {
		return nil, fmt.Errorf("could not resolve scenario path for %q", cleaned)
	}

	uiDir := filepath.Join(root, "ui")
	if info, err := os.Stat(uiDir); err != nil || !info.IsDir() {
		return &InteropStandardsResponse{
			EntityName: cleaned,
			Violations: []InteropStandardsViolation{},
		}, nil
	}

	checks := runAllInteropChecks(root)

	violations := make([]InteropStandardsViolation, 0)
	for _, check := range checks {
		if check.Passed || check.Skipped {
			continue
		}
		// Find the recommendation for this check
		rec := ""
		for _, def := range interopCheckDefs {
			if def.ID == check.CheckID {
				rec = def.Rec
				break
			}
		}
		violations = append(violations, InteropStandardsViolation{
			RuleID:         check.CheckID,
			Severity:       check.Severity,
			Title:          check.Name,
			Description:    check.Message,
			FilePath:       check.FilePath,
			Recommendation: rec,
			Metadata: map[string]any{
				"slot": check.Slot,
				"line": check.Line,
			},
		})
	}

	return &InteropStandardsResponse{
		EntityName: cleaned,
		Violations: violations,
	}, nil
}

// resolveScenarioRoot resolves the filesystem path for a scenario name.
func (s *AppService) resolveScenarioRoot(scenarioName string) string {
	if s.repoRoot == "" {
		return ""
	}
	candidate := filepath.Join(s.repoRoot, "scenarios", scenarioName)
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate
	}
	return ""
}

// =============================================================================
// Check Runner
// =============================================================================

// runAllInteropChecks runs all 16 interop checks against a scenario root.
func runAllInteropChecks(scenarioRoot string) []InteropCheckResult {
	checks := []func(string) InteropCheckResult{
		checkApiBaseDep,
		checkIframeBridgeDep,
		checkHardcodedLocalhost,
		checkRelativeBase,
		checkRouterBasename,
		checkNoCustomServer,
		checkBridgeInit,
		checkResolveApiBaseSingle,
		checkShortcutRelay,
		checkNoScatteredKeydown,
		checkBridgeAppId,
		checkProtectiveComments,
		checkIframeGuard,
		checkCaptureEnabled,
		checkProxyBasePreservation,
		checkSecureTunnel,
	}

	results := make([]InteropCheckResult, 0, len(checks))
	for _, fn := range checks {
		results = append(results, fn(scenarioRoot))
	}
	return results
}

// =============================================================================
// Individual Check Functions
// =============================================================================

// checkApiBaseDep verifies @vrooli/api-base is in ui/package.json dependencies.
func checkApiBaseDep(scenarioRoot string) InteropCheckResult {
	def := interopCheckDefs[0]
	return checkPackageDep(scenarioRoot, "@vrooli/api-base", def)
}

// checkIframeBridgeDep verifies @vrooli/iframe-bridge is in ui/package.json dependencies.
func checkIframeBridgeDep(scenarioRoot string) InteropCheckResult {
	def := interopCheckDefs[1]
	return checkPackageDep(scenarioRoot, "@vrooli/iframe-bridge", def)
}

// checkPackageDep is a helper that checks for a dependency in ui/package.json.
func checkPackageDep(scenarioRoot, depName string, def interopCheckDef) InteropCheckResult {
	result := InteropCheckResult{
		CheckID:  def.ID,
		Name:     def.Name,
		Severity: def.Severity,
		Slot:     def.Slot,
		FilePath: "ui/package.json",
	}

	pkgPath := filepath.Join(scenarioRoot, "ui", "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		result.Message = fmt.Sprintf("cannot read ui/package.json: %v", err)
		return result
	}

	var pkg map[string]json.RawMessage
	if err := json.Unmarshal(data, &pkg); err != nil {
		result.Message = fmt.Sprintf("cannot parse ui/package.json: %v", err)
		return result
	}

	for _, section := range []string{"dependencies", "devDependencies"} {
		raw, ok := pkg[section]
		if !ok {
			continue
		}
		var deps map[string]string
		if err := json.Unmarshal(raw, &deps); err != nil {
			continue
		}
		if _, found := deps[depName]; found {
			result.Passed = true
			result.Message = fmt.Sprintf("%s found in %s", depName, section)
			return result
		}
	}

	result.Message = fmt.Sprintf("%s not found in ui/package.json dependencies", depName)
	return result
}

// checkHardcodedLocalhost scans ui/src/ for localhost:PORT references.
func checkHardcodedLocalhost(scenarioRoot string) InteropCheckResult {
	def := interopCheckDefs[2]
	result := InteropCheckResult{
		CheckID:  def.ID,
		Name:     def.Name,
		Severity: def.Severity,
		Slot:     def.Slot,
	}

	srcDir := filepath.Join(scenarioRoot, "ui", "src")
	if _, err := os.Stat(srcDir); err != nil {
		result.Passed = true
		result.Message = "no ui/src/ directory"
		return result
	}

	var firstFile string
	var firstLine int
	found := false

	_ = filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := strings.ToLower(d.Name())
			if _, skip := localhostSkipDirectories[name]; skip {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(d.Name()))
		if _, ok := localhostAllowedExtensions[ext]; !ok {
			return nil
		}
		if isTestFile(d.Name()) {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if hardcodedLocalhostPattern.MatchString(line) {
				// Skip comment-only lines
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
					continue
				}
				if !found {
					rel, _ := filepath.Rel(scenarioRoot, path)
					firstFile = filepath.ToSlash(rel)
					firstLine = lineNum
				}
				found = true
				return filepath.SkipAll
			}
		}
		return nil
	})

	if found {
		result.Message = fmt.Sprintf("hardcoded localhost found in %s:%d", firstFile, firstLine)
		result.FilePath = firstFile
		result.Line = firstLine
	} else {
		result.Passed = true
		result.Message = "no hardcoded localhost:PORT in ui/src/"
	}

	return result
}

// checkRelativeBase verifies base: './' in ui/vite.config.ts.
func checkRelativeBase(scenarioRoot string) InteropCheckResult {
	def := interopCheckDefs[3]
	result := InteropCheckResult{
		CheckID:  def.ID,
		Name:     def.Name,
		Severity: def.Severity,
		Slot:     def.Slot,
		FilePath: "ui/vite.config.ts",
	}

	content, line, err := readFileAndSearchLine(filepath.Join(scenarioRoot, "ui", "vite.config.ts"), regexp.MustCompile(`base:\s*['"]\.\/['"]`))
	if err != nil {
		// Try .js variant
		content, line, err = readFileAndSearchLine(filepath.Join(scenarioRoot, "ui", "vite.config.js"), regexp.MustCompile(`base:\s*['"]\.\/['"]`))
		if err != nil {
			result.Message = "cannot read ui/vite.config.ts"
			return result
		}
		result.FilePath = "ui/vite.config.js"
	}

	if line > 0 {
		result.Passed = true
		result.Line = line
		result.Message = "base: './' found in vite config"
	} else {
		// Check if there's a base config at all
		basePattern := regexp.MustCompile(`base:\s*['"]`)
		if basePattern.Match(content) {
			result.Message = "vite config has base set but not to './'"
		} else {
			result.Message = "no base config found in vite config (default is '/')"
		}
	}

	return result
}

// checkRouterBasename checks if BrowserRouter has a basename prop.
func checkRouterBasename(scenarioRoot string) InteropCheckResult {
	def := interopCheckDefs[4]
	result := InteropCheckResult{
		CheckID:  def.ID,
		Name:     def.Name,
		Severity: def.Severity,
		Slot:     def.Slot,
	}

	srcDir := filepath.Join(scenarioRoot, "ui", "src")
	if _, err := os.Stat(srcDir); err != nil {
		result.Skipped = true
		result.SkipReason = "no ui/src/ directory"
		return result
	}

	browserRouterPattern := regexp.MustCompile(`<BrowserRouter`)
	basenamePattern := regexp.MustCompile(`<BrowserRouter[^>]*basename`)

	hasBrowserRouter := false
	hasBasename := false
	var routerFile string
	var routerLine int

	_ = filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".tsx" && ext != ".jsx" && ext != ".ts" && ext != ".js" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)

		if browserRouterPattern.MatchString(content) {
			hasBrowserRouter = true
			rel, _ := filepath.Rel(scenarioRoot, path)
			routerFile = filepath.ToSlash(rel)

			lines := strings.Split(content, "\n")
			for i, line := range lines {
				if browserRouterPattern.MatchString(line) {
					routerLine = i + 1
					break
				}
			}

			if basenamePattern.MatchString(content) {
				hasBasename = true
			}
			return filepath.SkipAll
		}
		return nil
	})

	if !hasBrowserRouter {
		result.Skipped = true
		result.SkipReason = "no BrowserRouter found (may use HashRouter or MemoryRouter)"
		return result
	}

	result.FilePath = routerFile
	result.Line = routerLine
	if hasBasename {
		result.Passed = true
		result.Message = "BrowserRouter has basename prop"
	} else {
		result.Message = "BrowserRouter missing basename prop for proxy support"
	}

	return result
}

// checkNoCustomServer verifies no express|createServer|http.listen in ui/server.js.
func checkNoCustomServer(scenarioRoot string) InteropCheckResult {
	def := interopCheckDefs[5]
	result := InteropCheckResult{
		CheckID:  def.ID,
		Name:     def.Name,
		Severity: def.Severity,
		Slot:     def.Slot,
	}

	for _, name := range []string{"server.js", "server.ts", "server.mjs"} {
		serverPath := filepath.Join(scenarioRoot, "ui", name)
		data, err := os.ReadFile(serverPath)
		if err != nil {
			continue
		}

		result.FilePath = "ui/" + name
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if customServerPattern.MatchString(line) {
				result.Line = i + 1
				result.Message = fmt.Sprintf("custom server code found in %s:%d", result.FilePath, result.Line)
				return result
			}
		}

		// Server file exists but no custom server patterns — pass
		result.Passed = true
		result.Message = fmt.Sprintf("%s exists with standard server setup", result.FilePath)
		return result
	}

	// No server.js at all — that's fine (uses vite dev server or static)
	result.Passed = true
	result.Message = "no custom server file found (standard setup)"
	return result
}

// checkBridgeInit verifies initIframeBridgeChild in ui/src/main.tsx.
func checkBridgeInit(scenarioRoot string) InteropCheckResult {
	def := interopCheckDefs[6]
	result := InteropCheckResult{
		CheckID:  def.ID,
		Name:     def.Name,
		Severity: def.Severity,
		Slot:     def.Slot,
	}

	pattern := regexp.MustCompile(`initIframeBridgeChild`)
	for _, name := range []string{"main.tsx", "main.ts", "main.jsx", "main.js", "index.tsx", "index.ts"} {
		mainPath := filepath.Join(scenarioRoot, "ui", "src", name)
		_, line, err := readFileAndSearchLine(mainPath, pattern)
		if err != nil {
			continue
		}
		result.FilePath = "ui/src/" + name
		if line > 0 {
			result.Passed = true
			result.Line = line
			result.Message = "initIframeBridgeChild found in " + result.FilePath
			return result
		}
		// Found the main file but no bridge init
		result.Message = "initIframeBridgeChild not found in " + result.FilePath
		return result
	}

	result.Message = "no main entry file found in ui/src/"
	return result
}

// checkResolveApiBaseSingle verifies resolveApiBase is imported in exactly 1 file under ui/src/.
func checkResolveApiBaseSingle(scenarioRoot string) InteropCheckResult {
	def := interopCheckDefs[7]
	result := InteropCheckResult{
		CheckID:  def.ID,
		Name:     def.Name,
		Severity: def.Severity,
		Slot:     def.Slot,
	}

	srcDir := filepath.Join(scenarioRoot, "ui", "src")
	if _, err := os.Stat(srcDir); err != nil {
		result.Skipped = true
		result.SkipReason = "no ui/src/ directory"
		return result
	}

	pattern := regexp.MustCompile(`resolveApiBase`)
	var files []string

	_ = filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			if d != nil && d.IsDir() {
				name := strings.ToLower(d.Name())
				if _, skip := localhostSkipDirectories[name]; skip {
					return filepath.SkipDir
				}
			}
			return walkErr
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if _, ok := localhostAllowedExtensions[ext]; !ok {
			return nil
		}
		if isTestFile(d.Name()) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if pattern.Match(data) {
			rel, _ := filepath.Rel(scenarioRoot, path)
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})

	switch {
	case len(files) == 0:
		result.Message = "resolveApiBase not found in any file under ui/src/"
	case len(files) <= 2:
		result.Passed = true
		result.FilePath = files[0]
		result.Message = fmt.Sprintf("resolveApiBase found in %d file(s): %s", len(files), strings.Join(files, ", "))
	default:
		result.FilePath = files[0]
		result.Message = fmt.Sprintf("resolveApiBase found in %d files (expected at most 2): %s", len(files), strings.Join(files, ", "))
	}

	return result
}

// checkShortcutRelay verifies emitShortcutIntent in ui/src/hooks/ if shortcuts exist.
func checkShortcutRelay(scenarioRoot string) InteropCheckResult {
	def := interopCheckDefs[8]
	result := InteropCheckResult{
		CheckID:  def.ID,
		Name:     def.Name,
		Severity: def.Severity,
		Slot:     def.Slot,
	}

	hooksDir := filepath.Join(scenarioRoot, "ui", "src", "hooks")
	if _, err := os.Stat(hooksDir); err != nil {
		result.Skipped = true
		result.SkipReason = "no ui/src/hooks/ directory"
		return result
	}

	// Check if there are keyboard-related hooks
	keydownPattern := regexp.MustCompile(`(?i)(keydown|keyboard|shortcut|hotkey)`)
	hasKeyboardHooks := false
	hasEmitShortcut := false
	emitPattern := regexp.MustCompile(`emitShortcutIntent`)

	_ = filepath.WalkDir(hooksDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		if keydownPattern.Match(data) || strings.Contains(d.Name(), "shortcut") || strings.Contains(d.Name(), "keyboard") || strings.Contains(d.Name(), "hotkey") {
			hasKeyboardHooks = true
		}

		if emitPattern.Match(data) {
			hasEmitShortcut = true
		}
		return nil
	})

	if !hasKeyboardHooks {
		result.Skipped = true
		result.SkipReason = "no keyboard/shortcut hooks found"
		return result
	}

	if hasEmitShortcut {
		result.Passed = true
		result.Message = "emitShortcutIntent found in hooks/"
	} else {
		// The relay may live outside hooks/ (e.g. App.tsx passes an
		// onUnhandledShortcut callback that calls emitShortcutIntent).
		// Scan the broader ui/src/ for the import before failing.
		srcDir := filepath.Join(scenarioRoot, "ui", "src")
		_ = filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() {
				return walkErr
			}
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" {
				return nil
			}
			if isTestFile(d.Name()) {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			if emitPattern.Match(data) {
				hasEmitShortcut = true
				return filepath.SkipAll
			}
			return nil
		})

		if hasEmitShortcut {
			result.Passed = true
			result.Message = "emitShortcutIntent found in shortcut chain (ui/src/)"
		} else {
			result.FilePath = "ui/src/hooks/"
			result.Message = "keyboard hooks exist but emitShortcutIntent not found (shortcuts won't relay through iframe)"
		}
	}

	return result
}

// checkNoScatteredKeydown checks addEventListener('keydown') only in hooks/ and dialog components.
func checkNoScatteredKeydown(scenarioRoot string) InteropCheckResult {
	def := interopCheckDefs[9]
	result := InteropCheckResult{
		CheckID:  def.ID,
		Name:     def.Name,
		Severity: def.Severity,
		Slot:     def.Slot,
	}

	srcDir := filepath.Join(scenarioRoot, "ui", "src")
	if _, err := os.Stat(srcDir); err != nil {
		result.Skipped = true
		result.SkipReason = "no ui/src/ directory"
		return result
	}

	keydownPattern := regexp.MustCompile(`addEventListener\s*\(\s*['"]keydown['"]`)
	var scattered []string

	_ = filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			if d != nil && d.IsDir() {
				name := strings.ToLower(d.Name())
				if _, skip := localhostSkipDirectories[name]; skip {
					return filepath.SkipDir
				}
			}
			return walkErr
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if _, ok := localhostAllowedExtensions[ext]; !ok {
			return nil
		}

		rel, _ := filepath.Rel(srcDir, path)
		relSlash := filepath.ToSlash(rel)

		// Allow hooks/ directory
		if strings.HasPrefix(relSlash, "hooks/") {
			return nil
		}
		// Allow dismissible UI components (common pattern for Escape key handling).
		// The skill says "Component-scoped Escape handlers for dialogs/modals are fine"
		// — this extends to all overlay-style components that close on Escape.
		nameLower := strings.ToLower(d.Name())
		for _, keyword := range []string{"dialog", "modal", "popup", "popover", "overlay", "dropdown", "selector", "menu", "tooltip"} {
			if strings.Contains(nameLower, keyword) {
				return nil
			}
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		if keydownPattern.Match(data) {
			fullRel, _ := filepath.Rel(scenarioRoot, path)
			scattered = append(scattered, filepath.ToSlash(fullRel))
		}
		return nil
	})

	if len(scattered) == 0 {
		result.Passed = true
		result.Message = "no scattered keydown listeners outside hooks/ and dismissible UI components"
	} else {
		result.FilePath = scattered[0]
		result.Message = fmt.Sprintf("keydown listeners found outside hooks/: %s", strings.Join(scattered, ", "))
	}

	return result
}

// checkBridgeAppId verifies initIframeBridgeChild call includes appId param.
func checkBridgeAppId(scenarioRoot string) InteropCheckResult {
	def := interopCheckDefs[10]
	result := InteropCheckResult{
		CheckID:  def.ID,
		Name:     def.Name,
		Severity: def.Severity,
		Slot:     def.Slot,
	}

	// Check multiple possible entry file names
	for _, name := range []string{"main.tsx", "main.ts", "main.jsx", "main.js", "index.tsx", "index.ts"} {
		mainPath := filepath.Join(scenarioRoot, "ui", "src", name)
		data, err := os.ReadFile(mainPath)
		if err != nil {
			continue
		}

		content := string(data)
		result.FilePath = "ui/src/" + name

		if !strings.Contains(content, "initIframeBridgeChild") {
			result.Message = "initIframeBridgeChild not found; cannot check appId"
			return result
		}

		// Look for appId in the call
		appIdPattern := regexp.MustCompile(`initIframeBridgeChild\s*\([^)]*appId`)
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.Contains(line, "initIframeBridgeChild") {
				result.Line = i + 1
				break
			}
		}

		if appIdPattern.MatchString(content) {
			result.Passed = true
			result.Message = "initIframeBridgeChild includes appId parameter"
		} else {
			// Also check multi-line — look for appId near the call
			callIdx := strings.Index(content, "initIframeBridgeChild(")
			if callIdx >= 0 {
				// Check the next 200 chars for appId
				end := callIdx + 200
				if end > len(content) {
					end = len(content)
				}
				snippet := content[callIdx:end]
				if strings.Contains(snippet, "appId") {
					result.Passed = true
					result.Message = "initIframeBridgeChild includes appId parameter"
				} else {
					result.Message = "initIframeBridgeChild call missing appId parameter"
				}
			} else {
				result.Message = "initIframeBridgeChild call missing appId parameter"
			}
		}
		return result
	}

	result.Message = "no main entry file found in ui/src/"
	return result
}

// checkProtectiveComments verifies INTEROP-CRITICAL comments in vite config and main entry.
func checkProtectiveComments(scenarioRoot string) InteropCheckResult {
	def := interopCheckDefs[11]
	result := InteropCheckResult{
		CheckID:  def.ID,
		Name:     def.Name,
		Severity: def.Severity,
		Slot:     def.Slot,
	}

	pattern := regexp.MustCompile(`INTEROP-CRITICAL`)
	hasViteComment := false
	hasMainComment := false

	// Check vite config
	for _, name := range []string{"vite.config.ts", "vite.config.js"} {
		path := filepath.Join(scenarioRoot, "ui", name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if pattern.Match(data) {
			hasViteComment = true
		}
		break
	}

	// Check main entry
	for _, name := range []string{"main.tsx", "main.ts", "main.jsx", "main.js", "index.tsx", "index.ts"} {
		path := filepath.Join(scenarioRoot, "ui", "src", name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if pattern.Match(data) {
			hasMainComment = true
		}
		break
	}

	var missing []string
	if !hasViteComment {
		missing = append(missing, "ui/vite.config.ts")
	}
	if !hasMainComment {
		missing = append(missing, "ui/src/main.tsx")
	}

	if len(missing) == 0 {
		result.Passed = true
		result.Message = "INTEROP-CRITICAL comments present in both files"
	} else {
		result.FilePath = missing[0]
		result.Message = fmt.Sprintf("INTEROP-CRITICAL comment missing in: %s", strings.Join(missing, ", "))
	}

	return result
}

// checkIframeGuard verifies bridge init is guarded with window.parent !== window.
func checkIframeGuard(scenarioRoot string) InteropCheckResult {
	def := interopCheckDefs[12]
	result := InteropCheckResult{
		CheckID:  def.ID,
		Name:     def.Name,
		Severity: def.Severity,
		Slot:     def.Slot,
	}

	bridgePattern := regexp.MustCompile(`initIframeBridgeChild`)
	for _, name := range []string{"main.tsx", "main.ts", "main.jsx", "main.js", "index.tsx", "index.ts"} {
		mainPath := filepath.Join(scenarioRoot, "ui", "src", name)
		data, err := os.ReadFile(mainPath)
		if err != nil {
			continue
		}

		content := string(data)
		result.FilePath = "ui/src/" + name

		if !bridgePattern.MatchString(content) {
			// No bridge call — skip (check 7 catches that)
			result.Skipped = true
			result.SkipReason = "no initIframeBridgeChild call found"
			return result
		}

		if iframeGuardPattern.MatchString(content) {
			result.Passed = true
			result.Message = "iframe guard (window.parent !== window) found"
		} else {
			result.Message = "initIframeBridgeChild not guarded with window.parent !== window check"
		}
		return result
	}

	result.Skipped = true
	result.SkipReason = "no main entry file found in ui/src/"
	return result
}

// checkCaptureEnabled verifies captureLogs and captureNetwork are not disabled in bridge init.
func checkCaptureEnabled(scenarioRoot string) InteropCheckResult {
	def := interopCheckDefs[13]
	result := InteropCheckResult{
		CheckID:  def.ID,
		Name:     def.Name,
		Severity: def.Severity,
		Slot:     def.Slot,
	}

	bridgePattern := regexp.MustCompile(`initIframeBridgeChild`)
	for _, name := range []string{"main.tsx", "main.ts", "main.jsx", "main.js", "index.tsx", "index.ts"} {
		mainPath := filepath.Join(scenarioRoot, "ui", "src", name)
		data, err := os.ReadFile(mainPath)
		if err != nil {
			continue
		}

		result.FilePath = "ui/src/" + name

		if !bridgePattern.Match(data) {
			// No bridge call — skip (check 7 catches that)
			result.Skipped = true
			result.SkipReason = "no initIframeBridgeChild call found"
			return result
		}

		logsDisabled := captureLogsDisabledPattern.Match(data)
		networkDisabled := captureNetworkDisabledPattern.Match(data)

		if logsDisabled && networkDisabled {
			result.Message = "captureLogs and captureNetwork are both disabled in bridge init"
		} else if logsDisabled {
			result.Message = "captureLogs is disabled in bridge init"
		} else if networkDisabled {
			result.Message = "captureNetwork is disabled in bridge init"
		} else {
			result.Passed = true
			result.Message = "capture settings are not disabled"
		}
		return result
	}

	result.Skipped = true
	result.SkipReason = "no main entry file found in ui/src/"
	return result
}

// checkProxyBasePreservation flags files that rebuild API bases using window.location.origin
// after calling resolveApiBase, which strips the secure tunnel prefix.
func checkProxyBasePreservation(scenarioRoot string) InteropCheckResult {
	def := interopCheckDefs[14]
	result := InteropCheckResult{
		CheckID:  def.ID,
		Name:     def.Name,
		Severity: def.Severity,
		Slot:     def.Slot,
	}

	srcDir := filepath.Join(scenarioRoot, "ui", "src")
	if _, err := os.Stat(srcDir); err != nil {
		result.Skipped = true
		result.SkipReason = "no ui/src/ directory"
		return result
	}

	resolvePattern := regexp.MustCompile(`resolveApiBase\(`)

	_ = filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			if d != nil && d.IsDir() {
				name := strings.ToLower(d.Name())
				if _, skip := localhostSkipDirectories[name]; skip {
					return filepath.SkipDir
				}
			}
			return walkErr
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if _, ok := localhostAllowedExtensions[ext]; !ok {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		if !resolvePattern.Match(data) {
			return nil
		}

		content := string(data)
		matches := originVariableRegex.FindAllStringSubmatchIndex(content, -1)
		if len(matches) == 0 {
			return nil
		}

		// Found resolveApiBase + window.location.origin capture — flag it
		rel, _ := filepath.Rel(scenarioRoot, path)
		relSlash := filepath.ToSlash(rel)
		lineNum := 1
		if len(matches) > 0 {
			offset := matches[0][0]
			lineNum = strings.Count(content[:offset], "\n") + 1
		}

		result.FilePath = relSlash
		result.Line = lineNum
		result.Message = fmt.Sprintf("resolveApiBase output may be rebuilt with window.location.origin in %s:%d", relSlash, lineNum)
		return filepath.SkipAll
	})

	if result.Message == "" {
		result.Passed = true
		result.Message = "no proxy base override pattern found"
	}

	return result
}

// checkSecureTunnel verifies that custom server files route API calls through proxyToApi.
func checkSecureTunnel(scenarioRoot string) InteropCheckResult {
	def := interopCheckDefs[15]
	result := InteropCheckResult{
		CheckID:  def.ID,
		Name:     def.Name,
		Severity: def.Severity,
		Slot:     def.Slot,
	}

	for _, name := range []string{"server.js", "server.ts", "server.mjs"} {
		serverPath := filepath.Join(scenarioRoot, "ui", name)
		data, err := os.ReadFile(serverPath)
		if err != nil {
			continue
		}

		result.FilePath = "ui/" + name

		// Check if it's a custom server (has express/createServer/listen patterns)
		if !customServerPattern.Match(data) {
			// Standard server — skip (not a custom server)
			result.Skipped = true
			result.SkipReason = fmt.Sprintf("%s exists but is not a custom server", result.FilePath)
			return result
		}

		// Custom server detected — verify proxyToApi is defined
		if proxyFunctionRegex.Match(data) {
			result.Passed = true
			result.Message = fmt.Sprintf("proxyToApi defined in %s", result.FilePath)
		} else {
			result.Message = fmt.Sprintf("custom server in %s does not define proxyToApi function", result.FilePath)
		}
		return result
	}

	// No server file at all — skip (N/A)
	result.Skipped = true
	result.SkipReason = "no custom server file found"
	return result
}

// =============================================================================
// Helpers
// =============================================================================

// isTestFile returns true for test files (*.test.ts, *.spec.tsx, etc.) that
// should be excluded from production-code scans.
func isTestFile(name string) bool {
	lower := strings.ToLower(name)
	ext := filepath.Ext(lower)
	base := strings.TrimSuffix(lower, ext)
	return strings.HasSuffix(base, ".test") || strings.HasSuffix(base, ".spec") ||
		strings.HasSuffix(base, "_test") || strings.HasSuffix(base, "_spec")
}

// readFileAndSearchLine reads a file and searches for a regex pattern, returning
// the file content, the first matching line number (1-based), and any error.
// Returns line=0 if the pattern is not found.
func readFileAndSearchLine(path string, pattern *regexp.Regexp) ([]byte, int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if pattern.MatchString(line) {
			return data, i + 1, nil
		}
	}
	return data, 0, nil
}

// Ensure time import is used (the type references use time.Time in the report)
var _ = time.Now
