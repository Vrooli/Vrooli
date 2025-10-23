package structure

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	rules "scenario-auditor/rules"
)

var (
	ruleDir = discoverRuleDir()
)

type uiStructurePayload struct {
	Scenario string   `json:"scenario"`
	Files    []string `json:"files"`
}

/*
Rule: Scenario UI Structure
Description: Validates that scenarios supplying a UI ship a usable shell with required entry points
Reason: Monitoring and orchestration flows depend on a consistent UI entry point when present
Category: structure
Severity: high
Targets: structure

<test-case id="missing-ui-directory" should-fail="false">
  <description>Scenario missing ui directory is allowed</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "api/main.go"
  ]
}
  </input>
</test-case>

<test-case id="missing-shell" should-fail="true" path="testdata/missing-shell">
  <description>ui directory exists but no HTML shell or server entrypoint</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "ui/app.js",
    "ui/styles.css"
  ]
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing required UI shell under demo/ui (expected an index.html or other HTML entrypoint)</expected-message>
</test-case>

<test-case id="missing-entry-script" should-fail="true" path="testdata/missing-entry">
  <description>HTML shell exists but no inline script, referenced script, or known entrypoint</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "ui/index.html",
    "ui/assets/style.css"
  ]
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing required UI script entrypoint (inline <script> or local JS/TS module)</expected-message>
</test-case>

<test-case id="remote-script-only" should-fail="true" path="testdata/remote-script-only">
  <description>HTML shell references only remote CDN script without local fallback</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "ui/index.html"
  ]
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing required UI script entrypoint (inline <script> or local JS/TS module)</expected-message>
</test-case>

<test-case id="remote-unreferenced-script" should-fail="true" path="testdata/remote-unreferenced">
  <description>HTML shell loads CDN script while local assets exist but are never referenced</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "ui/index.html",
    "ui/scripts/app-utils.js"
  ]
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing required UI script entrypoint (inline <script> or local JS/TS module)</expected-message>
</test-case>

<test-case id="missing-local-script" should-fail="true" path="testdata/missing-local-script">
  <description>HTML shell references local script that does not exist</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "ui/index.html"
  ]
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing required UI script entrypoint (inline <script> or local JS/TS module)</expected-message>
</test-case>

<test-case id="vite-public-success" should-fail="false" path="testdata/vite-public-success">
  <description>Vite/React UI served from ui/public with module script</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "ui/public/index.html",
    "ui/src/main.tsx"
  ]
}
  </input>
</test-case>

<test-case id="scripts-dir-success" should-fail="false" path="testdata/scripts-dir-success">
  <description>Static HTML shell that mounts scripts from ui/scripts</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "ui/index.html",
    "ui/scripts/main.js"
  ]
}
  </input>
</test-case>

<test-case id="cra-fallback-success" should-fail="false" path="testdata/cra-fallback-success">
  <description>React create-react-app style UI relies on src/index.tsx fallback entrypoint</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "ui/public/index.html",
    "ui/package.json",
    "ui/src/index.tsx",
    "ui/src/App.tsx"
  ]
}
  </input>
</test-case>

<test-case id="unreferenced-src-main" should-fail="true" path="testdata/unreferenced-src-main">
  <description>Local React entry file exists but shell never references it</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "ui/index.html",
    "ui/src/main.tsx"
  ]
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing required UI script entrypoint (inline <script> or local JS/TS module)</expected-message>
</test-case>

<test-case id="script-only-entry" should-fail="true" path="testdata/script-only-entry">
  <description>Script-only UI entrypoint without HTML shell</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "ui/scripts/main.js"
  ]
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing required UI shell</expected-message>
</test-case>

<test-case id="inline-script-success" should-fail="false" path="testdata/inline-script-success">
  <description>HTML shell with inline script logic</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "ui/index.html"
  ]
}
  </input>
</test-case>

<test-case id="server-render-success" should-fail="false" path="testdata/server-render-success">
  <description>Node server renders the UI shell dynamically</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "ui/server.js"
  ]
}
  </input>
</test-case>

<test-case id="electron-ui-success" should-fail="false" path="testdata/electron-ui-success">
  <description>Electron style UI with nested HTML and renderer scripts</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "ui/electron/overlay.html",
    "ui/electron/renderer.js"
  ]
}
  </input>
</test-case>

<test-case id="app-monitor-example" should-fail="false" scenario="app-monitor" path="testdata/app-monitor-example">
  <description>App monitor scenario illustrates optional UI handling</description>
  <input language="json">
{
  "scenario": "app-monitor",
  "files": [
    "ui/index.html",
    "ui/src/main.tsx"
  ]
}
  </input>
</test-case>
*/

// CheckUIStructure validates the UI structure entrypoint expected by the scenario lifecycle.
func CheckUIStructure(content []byte, scenarioPath string, scenario string) ([]rules.Violation, error) {
	return CheckUICore(content, scenarioPath, scenario)
}

// CheckUICore validates the UI shell for required entry assets.
func CheckUICore(content []byte, scenarioPath string, scenario string) ([]rules.Violation, error) {
	var payload uiStructurePayload
	if err := json.Unmarshal(content, &payload); err != nil {
		return []rules.Violation{newUIViolation("ui", fmt.Sprintf("UI structure payload is invalid JSON: %v", err), "high")}, nil
	}

	scenarioName := strings.TrimSpace(payload.Scenario)
	if scenarioName == "" {
		scenarioName = strings.TrimSpace(scenario)
	}

	scenarioPath = resolveScenarioRoot(scenarioPath, payload.Scenario)
	if scenarioPath == "" {
		scenarioPath = resolveScenarioRoot(payload.Scenario, "")
	}
	if scenarioName == "" && strings.TrimSpace(scenarioPath) != "" {
		scenarioName = filepath.Base(filepath.Clean(scenarioPath))
	}

	filesSet := make(map[string]struct{}, len(payload.Files))
	for _, f := range payload.Files {
		filesSet[filepath.ToSlash(strings.TrimSpace(f))] = struct{}{}
	}

	var violations []rules.Violation

	if !uiDirectoryExists(scenarioPath, "ui", filesSet) {
		// UI is optional. If a scenario has no UI assets we consider it compliant.
		return nil, nil
	}

	serverEntrypoint, _ := serverEntrypointExists(scenarioPath, filesSet)
	shells := collectUIShells(scenarioPath, filesSet)
	hasShells := len(shells) > 0

	hasEntry := serverEntrypoint
	if !hasEntry && hasShells {
		for _, shell := range shells {
			if shell.ProvidesEntrypoint(scenarioPath, filesSet) {
				hasEntry = true
				break
			}
		}
	}

	if !hasEntry && hasShells && hasFallbackEntrypoint(scenarioPath, filesSet) {
		hasEntry = true
	}

	if !hasShells && !serverEntrypoint {
		relative := "ui"
		if scenarioName != "" {
			relative = filepath.ToSlash(filepath.Join(scenarioName, "ui"))
		}
		message := fmt.Sprintf("Missing required UI shell under %s (expected an index.html or other HTML entrypoint)", relative)
		violations = append(violations, newUIViolation("ui", message, "high"))
		return violations, nil
	}

	if hasShells && !hasEntry {
		violations = append(violations, newUIViolation("ui", "Missing required UI script entrypoint (inline <script> or local JS/TS module)", "high"))
	}

	return violations, nil
}

func newUIViolation(path, message string, severity string) rules.Violation {
	severity = strings.TrimSpace(severity)
	if severity == "" {
		severity = "high"
	}
	recommendation := fmt.Sprintf("Add the required resource at %s", path)
	return rules.Violation{
		Severity:       severity,
		Message:        message,
		FilePath:       filepath.ToSlash(path),
		Recommendation: recommendation,
	}
}

func uiFileExists(root, rel string, known map[string]struct{}) bool {
	rel = filepath.ToSlash(rel)
	if _, ok := known[rel]; ok {
		return true
	}
	_, err := os.Stat(filepath.Join(root, rel))
	return err == nil
}

func uiDirectoryExists(root, rel string, known map[string]struct{}) bool {
	rel = filepath.ToSlash(rel)
	if _, ok := known[rel]; ok {
		return true
	}
	prefix := rel
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for path := range known {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	info, err := os.Stat(filepath.Join(root, rel))
	if err != nil {
		return false
	}
	return info.IsDir()
}

type uiShellInfo struct {
	Path            string
	HasInlineScript bool
	ScriptSources   []string
}

func (s uiShellInfo) ProvidesEntrypoint(root string, files map[string]struct{}) bool {
	if s.HasInlineScript {
		return true
	}
	for _, src := range s.ScriptSources {
		if scriptFileExists(root, s.Path, src, files) {
			return true
		}
	}
	return false
}

func serverEntrypointExists(root string, files map[string]struct{}) (bool, string) {
	candidates := []string{
		"ui/server.js",
		"ui/server.ts",
		"ui/server.mjs",
		"ui/server.cjs",
		"ui/server/index.js",
		"ui/server/index.ts",
		"ui/server/main.js",
	}

	for _, candidate := range candidates {
		rel := filepath.ToSlash(candidate)
		if _, ok := files[rel]; ok {
			return true, rel
		}
		if uiFileExists(root, rel, files) {
			return true, rel
		}
	}

	return false, ""
}

func collectUIShells(root string, files map[string]struct{}) []uiShellInfo {
	seen := make(map[string]struct{})
	var shells []uiShellInfo

	addShell := func(rel string) {
		rel = filepath.ToSlash(rel)
		if _, ok := seen[rel]; ok {
			return
		}
		info := uiShellInfo{Path: rel}
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if data, err := os.ReadFile(abs); err == nil {
			content := string(data)
			info.HasInlineScript = hasInlineScript(content)
			info.ScriptSources = extractScriptSources(content)
		}
		shells = append(shells, info)
		seen[rel] = struct{}{}
	}

	for path := range files {
		lower := strings.ToLower(path)
		if !strings.HasPrefix(lower, "ui/") {
			continue
		}
		if strings.Contains(lower, "/node_modules/") {
			continue
		}
		if strings.HasSuffix(lower, ".html") {
			addShell(path)
		}
	}

	defaultCandidates := []string{
		"ui/index.html",
		"ui/public/index.html",
		"ui/dashboard.html",
		"ui/app.html",
		"ui/example.html",
		"ui/electron/overlay.html",
	}

	for _, candidate := range defaultCandidates {
		if uiFileExists(root, candidate, files) {
			addShell(candidate)
		}
	}

	return shells
}

var scriptSrcRegex = regexp.MustCompile(`(?i)<script[^>]*src\s*=\s*"([^"]+)"|<script[^>]*src\s*=\s*'([^']+)'`)

func extractScriptSources(content string) []string {
	matches := scriptSrcRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	var sources []string
	for _, match := range matches {
		candidate := ""
		if len(match) > 1 && match[1] != "" {
			candidate = match[1]
		} else if len(match) > 2 && match[2] != "" {
			candidate = match[2]
		}
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		lower := strings.ToLower(candidate)
		if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "//") || strings.HasPrefix(lower, "data:") || strings.HasPrefix(lower, "blob:") {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		sources = append(sources, candidate)
		seen[candidate] = struct{}{}
	}
	return sources
}

var inlineScriptRegex = regexp.MustCompile(`(?is)<script\b([^>]*)>(.*?)</script>`)

func hasInlineScript(content string) bool {
	matches := inlineScriptRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return false
	}
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		attrs := strings.ToLower(match[1])
		body := strings.TrimSpace(match[2])
		if strings.Contains(attrs, "src=") {
			// Attribute source means the script is loaded externally; inline presence is not guaranteed.
			continue
		}
		if body == "" {
			continue
		}
		typeAttr := extractScriptType(attrs)
		if typeAttr != "" {
			typeAttr = strings.ToLower(typeAttr)
			if !(strings.Contains(typeAttr, "javascript") || strings.Contains(typeAttr, "module") || strings.Contains(typeAttr, "jsx") || strings.Contains(typeAttr, "tsx")) {
				continue
			}
		}
		return true
	}
	return false
}

var scriptTypeRegex = regexp.MustCompile(`type\s*=\s*"([^"]+)"|type\s*=\s*'([^']+)'`)

func extractScriptType(attrs string) string {
	matches := scriptTypeRegex.FindStringSubmatch(attrs)
	if len(matches) == 0 {
		return ""
	}
	if len(matches) > 1 && matches[1] != "" {
		return matches[1]
	}
	if len(matches) > 2 {
		return matches[2]
	}
	return ""
}

func scriptFileExists(root, shellPath, src string, files map[string]struct{}) bool {
	normalized := normalizeScriptPath(shellPath, src)
	if normalized == "" {
		return false
	}

	candidates := []string{normalized}
	if strings.HasPrefix(normalized, "ui/") {
		trimmed := strings.TrimPrefix(normalized, "ui/")
		if trimmed != "" {
			publicCandidate := filepath.ToSlash(filepath.Join("ui", "public", trimmed))
			if publicCandidate != normalized {
				candidates = append(candidates, publicCandidate)
			}
		}
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, ok := files[candidate]; ok {
			return true
		}
		if uiFileExists(root, candidate, files) {
			return true
		}
	}

	return false
}

func normalizeScriptPath(shellPath, src string) string {
	src = strings.TrimSpace(src)
	if src == "" {
		return ""
	}
	if idx := strings.IndexAny(src, "?#"); idx != -1 {
		src = src[:idx]
	}
	if src == "" {
		return ""
	}
	baseDir := filepath.ToSlash(filepath.Dir(shellPath))

	var candidate string
	if strings.HasPrefix(src, "/") {
		candidate = filepath.ToSlash(filepath.Join("ui", strings.TrimPrefix(src, "/")))
	} else {
		candidate = filepath.ToSlash(filepath.Clean(filepath.Join(baseDir, src)))
	}

	if !strings.HasPrefix(candidate, "ui/") {
		candidate = filepath.ToSlash(filepath.Join("ui", candidate))
	}

	candidate = filepath.ToSlash(filepath.Clean(candidate))

	if !strings.HasPrefix(candidate, "ui/") {
		return ""
	}

	return candidate
}

func hasFallbackEntrypoint(root string, files map[string]struct{}) bool {
	if isCreateReactAppEntrypoint(root, files) {
		return true
	}

	return false
}

func isCreateReactAppEntrypoint(root string, files map[string]struct{}) bool {
	// CRA templates rely on the dev server to inject scripts. We only trust this
	// pattern when the scenario clearly depends on react-scripts and ships the
	// expected template under ui/public/index.html.
	if !uiFileExists(root, "ui/public/index.html", files) {
		return false
	}

	packageJSON := filepath.Join(root, "ui", "package.json")
	data, err := os.ReadFile(packageJSON)
	if err != nil {
		return false
	}

	lower := strings.ToLower(string(data))
	if !strings.Contains(lower, "react-scripts") {
		return false
	}

	// Require a known CRA entry module so we know something will boot.
	for _, candidate := range []string{"ui/src/index.tsx", "ui/src/index.ts", "ui/src/index.jsx", "ui/src/index.js"} {
		if uiFileExists(root, candidate, files) {
			return true
		}
	}

	return false
}

func hasSupportedScriptExtension(path string) bool {
	for _, ext := range []string{".js", ".ts", ".tsx", ".jsx", ".mjs", ".cjs"} {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

func resolveScenarioRoot(input string, fallback string) string {
	candidates := []string{strings.TrimSpace(input), strings.TrimSpace(fallback)}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if resolved := resolveCandidate(candidate); resolved != "" {
			return resolved
		}
	}
	return ""
}

func resolveCandidate(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return filepath.Clean(path)
		}
		return ""
	}

	tryPaths := []string{
		path,
		filepath.Join(ruleDir, path),
		filepath.Join(filepath.Dir(ruleDir), path),
		filepath.Join(filepath.Dir(filepath.Dir(ruleDir)), path),
		filepath.Join(ruleDir, "scenarios", path),
		filepath.Join(filepath.Dir(ruleDir), "scenarios", path),
		filepath.Join(filepath.Dir(filepath.Dir(ruleDir)), "scenarios", path),
	}

	for _, envVar := range []string{"VROOLI_ROOT", "APP_ROOT"} {
		if base := strings.TrimSpace(os.Getenv(envVar)); base != "" {
			tryPaths = append(tryPaths,
				filepath.Join(base, path),
				filepath.Join(base, "scenarios", path),
			)
		}
	}

	if wd, err := os.Getwd(); err == nil {
		tryPaths = append(tryPaths,
			filepath.Join(wd, path),
			filepath.Join(wd, "rules", "structure", path),
			filepath.Join(wd, "api", "rules", "structure", path),
			filepath.Join(wd, "scenarios", "scenario-auditor", "api", "rules", "structure", path),
			filepath.Join(wd, "scenarios", path),
		)
	}

	for _, candidate := range tryPaths {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			if abs, err := filepath.Abs(candidate); err == nil {
				return abs
			}
			return candidate
		}
	}

	return ""
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func isNumericString(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func discoverRuleDir() string {
	if _, file, _, ok := runtime.Caller(0); ok {
		if strings.HasSuffix(file, "ui_structure.go") {
			return filepath.Dir(file)
		}
	}

	if wd, err := os.Getwd(); err == nil {
		if dir := searchRuleDirFrom(wd); dir != "" {
			return dir
		}
	}

	for _, envVar := range []string{"VROOLI_ROOT", "APP_ROOT"} {
		if base := strings.TrimSpace(os.Getenv(envVar)); base != "" {
			if dir := searchRuleDirFrom(base); dir != "" {
				return dir
			}
		}
	}

	return "."
}

func searchRuleDirFrom(start string) string {
	start = strings.TrimSpace(start)
	if start == "" {
		return ""
	}
	current := filepath.Clean(start)
	visited := map[string]struct{}{}
	for {
		if _, seen := visited[current]; seen {
			break
		}
		visited[current] = struct{}{}

		candidates := []string{
			current,
			filepath.Join(current, "rules", "structure"),
			filepath.Join(current, "api", "rules", "structure"),
			filepath.Join(current, "scenarios", "scenario-auditor", "api", "rules", "structure"),
		}

		for _, candidate := range candidates {
			file := filepath.Join(candidate, "ui_structure.go")
			if info, err := os.Stat(file); err == nil && !info.IsDir() {
				return filepath.Dir(file)
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return ""
}
