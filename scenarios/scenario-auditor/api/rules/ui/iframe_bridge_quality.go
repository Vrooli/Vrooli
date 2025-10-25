package ui

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

/*
Rule: Scenario UI Bridge Quality
Description: Validates that UI packages depend on the shared iframe bridge and entry files bootstrap it safely
Reason: The iframe bridge enables orchestration, so scenarios must import the shared implementation and initialize it correctly
Category: ui
Severity: high
Standard: ui-bridge-v1
Targets: ui

<test-case id="entry-react-valid" should-fail="false" path="testdata/entry-react-valid/ui/src/main.tsx">
  <description>React entry imports shared bridge and initializes it safely</description>
  <input language="typescript"><![CDATA[
import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

if (typeof window !== 'undefined' && window.parent !== window) {
  initIframeBridgeChild({ appId: 'demo' })
}

ReactDOM.createRoot(document.getElementById('root')!).render(<App />)
]]></input>
</test-case>

<test-case id="entry-missing-import" should-fail="true" path="testdata/entry-missing-import/ui/src/main.tsx">
  <description>Entry lacks bridge import</description>
  <input language="typescript"><![CDATA[
if (typeof window !== 'undefined' && window.parent !== window) {
  initIframeBridgeChild({ appId: 'demo' })
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing shared bridge import</expected-message>
</test-case>

<test-case id="entry-missing-guard" should-fail="true" path="testdata/entry-missing-guard/ui/src/main.tsx">
  <description>Entry calls bridge without iframe guard</description>
  <input language="typescript"><![CDATA[
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

initIframeBridgeChild({ appId: 'demo' })
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing iframe guard</expected-message>
</test-case>

<test-case id="entry-missing-call" should-fail="true" path="testdata/entry-missing-call/ui/src/main.tsx">
  <description>Entry imports bridge but never invokes it</description>
  <input language="typescript"><![CDATA[
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

if (typeof window !== 'undefined' && window.parent !== window) {
  console.log('preview loaded')
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing bridge invocation</expected-message>
</test-case>

<test-case id="entry-log-disabled" should-fail="true" path="testdata/entry-log-disabled/ui/src/main.tsx">
  <description>Entry disables iframe log capture</description>
  <input language="typescript"><![CDATA[
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

if (typeof window !== 'undefined' && window.parent !== window) {
  initIframeBridgeChild({ appId: 'demo', captureLogs: false })
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>log capture</expected-message>
</test-case>

<test-case id="entry-static-missing-guard" should-fail="true" path="testdata/entry-static-missing-guard/ui/app.js">
  <description>Static bootstrap without iframe guard must violate</description>
  <input language="javascript"><![CDATA[
if (typeof window !== 'undefined') {
  window.initIframeBridgeChild({ appId: 'demo' })
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>iframe guard</expected-message>
</test-case>

<test-case id="entry-html-container" should-fail="false" path="testdata/entry-html-container/ui/index.html">
  <description>HTML shell that defers bridge work to bundled entry should pass</description>
  <input language="html"><![CDATA[
<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <title>Bridge Shell</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
]]></input>
</test-case>

<test-case id="entry-alt-guard" should-fail="false" path="testdata/entry-alt-guard/ui/src/main.tsx">
  <description>Alternate guard ordering is accepted</description>
  <input language="typescript"><![CDATA[
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

if (typeof window !== 'undefined' && window !== window.parent) {
  initIframeBridgeChild({ appId: 'demo', captureLogs: { enabled: true } })
}
]]></input>
</test-case>

<test-case id="entry-electron-main" should-fail="false" path="testdata/entry-electron/ui/electron/main.js">
  <description>Electron main process should not be treated as an iframe entry</description>
  <input language="javascript"><![CDATA[
const { app, BrowserWindow } = require('electron');

function createWindow() {
  const win = new BrowserWindow({ width: 800, height: 600 });
  win.loadFile('index.html');
}

app.whenReady().then(createWindow);
]]></input>
</test-case>

<test-case id="entry-network-disabled" should-fail="true" path="testdata/entry-network-disabled/ui/src/main.tsx">
  <description>Entry disables iframe network capture</description>
  <input language="typescript"><![CDATA[
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

if (typeof window !== 'undefined' && window.parent !== window) {
  initIframeBridgeChild({ appId: 'demo', captureNetwork: { enabled: false } })
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>network capture</expected-message>
</test-case>

<test-case id="package-shared-valid" should-fail="false" path="testdata/package-shared-valid/ui/package.json">
  <description>UI package declares the shared iframe bridge dependency</description>
  <input language="json">
{
  "name": "demo-ui",
  "version": "1.0.0",
  "dependencies": {
    "@vrooli/iframe-bridge": "file:../../../packages/iframe-bridge"
  }
}
  </input>
</test-case>

<test-case id="package-missing-bridge" should-fail="true" path="testdata/package-missing-bridge/ui/package.json">
  <description>Package json missing shared dependency triggers violation</description>
  <input language="json">
{
  "name": "demo-ui",
  "version": "1.0.0"
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>@vrooli/iframe-bridge</expected-message>
</test-case>

<test-case id="package-missing-install" should-fail="true" path="testdata/package-missing-install/ui/package.json">
  <description>Node modules missing iframe bridge bundle triggers violation</description>
  <input language="json">
{
  "name": "demo-ui",
  "version": "1.0.0",
  "dependencies": {
    "@vrooli/iframe-bridge": "file:../../../packages/iframe-bridge"
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>not found in node_modules</expected-message>
</test-case>

<test-case id="package-incomplete-install" should-fail="true" path="testdata/package-incomplete-install/ui/package.json">
  <description>Iframe bridge dependency installed without dist bundle triggers violation</description>
  <input language="json">
{
  "name": "demo-ui",
  "version": "1.0.0",
  "dependencies": {
    "@vrooli/iframe-bridge": "file:../../../packages/iframe-bridge"
  }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>installation is incomplete</expected-message>
</test-case>
*/

var (
	bridgeCallPattern             = regexp.MustCompile(`initIframeBridgeChild\s*\(`)
	windowBridgeCallPattern       = regexp.MustCompile(`window\.(?:parent\.)?initIframeBridgeChild\s*\(`)
	bridgeGuardPattern            = regexp.MustCompile(`(?:(?:window\.parent\s*(?:!==|!=|===|==)\s*window)|(?:window\s*(?:!==|!=|===|==)\s*window\.parent))`)
	appIDPropertyPattern          = regexp.MustCompile(`(?m)(?:['"])?appId(?:['"])?\s*:`)
	bridgeImportPattern           = regexp.MustCompile(`(?m)(import\s+[^;]*@vrooli/iframe-bridge/(?:child|dist/iframeBridgeChild)[^\n]*|require\([^\n]*@vrooli/iframe-bridge/(?:child|dist/iframeBridgeChild)[^\n]*\))`)
	captureLogsDisabledPattern    = regexp.MustCompile(`(?s)captureLogs\s*:\s*(?:false|\{[^}]*enabled\s*:\s*false)`)
	captureNetworkDisabledPattern = regexp.MustCompile(`(?s)captureNetwork\s*:\s*(?:false|\{[^}]*enabled\s*:\s*false)`)
	iframeRuleDir                 string
)

const sharedBridgePackageName = "@vrooli/iframe-bridge"

func init() {
	if _, file, _, ok := runtime.Caller(0); ok {
		iframeRuleDir = filepath.Dir(file)
	}
}

// CheckIframeBridgeQuality validates canonical UI bridge implementation and bootstrap usage.
func CheckIframeBridgeQuality(content []byte, filePath string) []rules.Violation {
	path := filepath.ToSlash(strings.TrimSpace(filePath))
	if path == "" {
		return nil
	}

	base := strings.ToLower(filepath.Base(path))
	source := string(content)

	switch {
	case base == "package.json" && strings.Contains(path, "/ui/"):
		return validateBridgePackage(content, path)
	case isEntryCandidate(path, base, source):
		return validateEntry(content, path)
	default:
		return nil
	}
}

func isEntryCandidate(path, base, source string) bool {
	if bridgeCallPattern.MatchString(source) || windowBridgeCallPattern.MatchString(source) {
		return true
	}

	isHTML := strings.HasSuffix(strings.ToLower(base), ".html") || strings.HasSuffix(strings.ToLower(base), ".htm")
	if isHTML {
		return likelyStaticBootstrapSource(source)
	}

	if bridgeImportPattern.MatchString(source) {
		return true
	}

	return isLikelyEntryPath(path, base)
}

func isEntryFileName(base string) bool {
	switch {
	case strings.HasPrefix(base, "main."),
		strings.HasPrefix(base, "app."),
		strings.HasPrefix(base, "entry."),
		strings.HasPrefix(base, "bootstrap."),
		strings.HasPrefix(base, "index."),
		strings.HasPrefix(base, "script."),
		strings.HasPrefix(base, "server."),
		strings.HasPrefix(base, "ui."):
		ext := strings.ToLower(filepath.Ext(base))
		switch ext {
		case ".ts", ".tsx", ".js", ".jsx":
			return true
		}
	}
	return false
}

func isLikelyEntryPath(path, base string) bool {
	if base == "" {
		return false
	}

	normalized := strings.ToLower(filepath.ToSlash(path))
	if !strings.Contains(normalized, "/ui/") {
		return false
	}

	if strings.HasSuffix(base, ".html") || strings.HasSuffix(base, ".htm") {
		return true
	}

	clientSegments := []string{
		"/ui/src/",
		"/ui/public/",
		"/ui/app",
		"/ui/scripts/",
		"/ui/script",
		"/ui/index",
	}

	for _, segment := range clientSegments {
		if strings.Contains(normalized, segment) {
			return isEntryFileName(base)
		}
	}

	return false
}

func validateEntry(content []byte, path string) []rules.Violation {
	source := string(content)
	lower := strings.ToLower(filepath.ToSlash(path))

	isStaticBootstrap := strings.Contains(lower, "ui/app.") || strings.Contains(lower, "ui/script.") || strings.Contains(lower, "/app.") || strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm") || likelyStaticBootstrapSource(source)
	called := bridgeCallPattern.MatchString(source) || windowBridgeCallPattern.MatchString(source)

	var violations []rules.Violation

	if isStaticBootstrap {
		if !called {
			violations = append(violations, newBridgeViolation(path, "Static UI bootstrap must call window.initIframeBridgeChild"))
		}
	} else {
		if !bridgeImportPattern.MatchString(source) {
			violations = append(violations, newBridgeViolation(path, fmt.Sprintf("Missing shared bridge import; UI entry file must import initIframeBridgeChild from %s", sharedBridgePackageName)))
		}
		if !called {
			violations = append(violations, newBridgeViolation(path, "Missing bridge invocation; UI entry file must invoke initIframeBridgeChild"))
		}
	}

	if !bridgeGuardPattern.MatchString(source) {
		violations = append(violations, newBridgeViolation(path, "Missing iframe guard; UI entry file must guard bridge initialization with an iframe check"))
	}

	if called && !appIDPropertyPattern.MatchString(source) {
		violations = append(violations, newBridgeViolation(path, "UI entry file must pass appId in bridge options"))
	}

	if captureLogsDisabledPattern.MatchString(source) {
		violations = append(violations, newBridgeViolation(path, "UI entry must not disable iframe log capture (captureLogs.enabled=false)"))
	}
	if captureNetworkDisabledPattern.MatchString(source) {
		violations = append(violations, newBridgeViolation(path, "UI entry must not disable iframe network capture (captureNetwork.enabled=false)"))
	}

	return dedupeViolations(violations)
}

func validateBridgePackage(content []byte, path string) []rules.Violation {
	scenario := scenarioFromFilePath(path)
	if strings.EqualFold(scenario, "app-monitor") {
		return nil
	}

	version := extractBridgeDependencyVersion(content)
	if strings.TrimSpace(version) != "" {
		if message := validateBridgeDependencyInstallation(path, version); message != "" {
			return []rules.Violation{newPackageBridgeViolation(relativePackagePath(path), message)}
		}
		return nil
	}
	relative := relativePackagePath(path)
	if relative == "" {
		relative = filepath.ToSlash(filepath.Base(path))
	}
	message := fmt.Sprintf("Missing %s dependency; UI packages must declare the shared bridge", sharedBridgePackageName)
	return []rules.Violation{newPackageBridgeViolation(relative, message)}
}

func validateBridgeDependencyInstallation(manifestPath, version string) string {
	candidates := manifestDirCandidates(manifestPath)

	if strings.HasPrefix(strings.TrimSpace(version), "file:") {
		spec := strings.TrimSpace(strings.TrimPrefix(version, "file:"))
		if spec != "" {
			resolved := resolveFirstExistingPath(candidates, spec)
			if resolved == "" {
				return fmt.Sprintf("Local iframe bridge dependency path %q could not be resolved", spec)
			}
		}
	}

	bridgePath, info := findFirstExistingBridgePath(candidates)
	if bridgePath == "" {
		return "Installed iframe bridge package not found in node_modules; reinstall dependencies"
	}

	resolved := bridgePath
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := filepath.EvalSymlinks(bridgePath)
		if err != nil {
			return fmt.Sprintf("Iframe bridge dependency symlink is broken: %v", err)
		}
		resolved = target
	}

	expectedBundle := filepath.Join(resolved, "dist", "iframeBridgeChild.js")
	if !pathExists(expectedBundle) {
		return "Iframe bridge package installation is incomplete; dist/iframeBridgeChild.js is missing"
	}

	return ""
}

func newBridgeViolation(path, message string) rules.Violation {
	return rules.Violation{
		Severity:       "high",
		Message:        message,
		FilePath:       filepath.ToSlash(path),
		Recommendation: fmt.Sprintf("Import and initialize %s in the UI entry point", sharedBridgePackageName),
	}
}

func newPackageBridgeViolation(path, message string) rules.Violation {
	return rules.Violation{
		Severity:       "high",
		Message:        message,
		FilePath:       filepath.ToSlash(path),
		Recommendation: fmt.Sprintf("Add %s to dependencies", sharedBridgePackageName),
	}
}

func dedupeViolations(violations []rules.Violation) []rules.Violation {
	if len(violations) <= 1 {
		return violations
	}
	seen := map[string]bool{}
	result := make([]rules.Violation, 0, len(violations))
	for _, v := range violations {
		if seen[v.Message] {
			continue
		}
		seen[v.Message] = true
		result = append(result, v)
	}
	return result
}

func likelyStaticBootstrapSource(source string) bool {
	trimmed := strings.TrimSpace(source)
	lower := strings.ToLower(trimmed)

	if strings.Contains(lower, "typeof window.initiframebridgechild") && strings.Contains(lower, "window.parent === window") {
		return true
	}

	if strings.HasPrefix(lower, "(function") || strings.HasPrefix(lower, "!function") || strings.HasPrefix(lower, "(() =>") {
		if strings.Contains(lower, "window.initiframebridgechild") {
			return true
		}
	}

	return false
}

func extractBridgeDependencyVersion(content []byte) string {
	var manifest struct {
		Dependencies    map[string]interface{} `json:"dependencies"`
		DevDependencies map[string]interface{} `json:"devDependencies"`
	}
	if err := json.Unmarshal(content, &manifest); err != nil {
		return ""
	}
	if version := dependencyVersionFromMap(manifest.Dependencies, sharedBridgePackageName); version != "" {
		return version
	}
	return dependencyVersionFromMap(manifest.DevDependencies, sharedBridgePackageName)
}

func dependencyVersionFromMap(deps map[string]interface{}, name string) string {
	if deps == nil {
		return ""
	}
	value, ok := deps[name]
	if !ok {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}

func manifestDirCandidates(manifestPath string) []string {
	clean := strings.TrimSpace(manifestPath)
	dir := filepath.Dir(clean)
	add := func(list *[]string, seen map[string]bool, candidate string) {
		if candidate == "" {
			return
		}
		canonical := filepath.Clean(candidate)
		if seen[canonical] {
			return
		}
		seen[canonical] = true
		*list = append(*list, canonical)
	}

	seen := map[string]bool{}
	var candidates []string

	if filepath.IsAbs(dir) {
		add(&candidates, seen, dir)
		debugLogManifest(manifestPath, candidates)
		return candidates
	}

	if iframeRuleDir != "" {
		add(&candidates, seen, filepath.Join(iframeRuleDir, dir))
		add(&candidates, seen, filepath.Join(filepath.Dir(iframeRuleDir), dir))
	}

	if wd, err := os.Getwd(); err == nil {
		add(&candidates, seen, filepath.Join(wd, dir))
	}

	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		add(&candidates, seen, filepath.Join(root, "scenarios", "scenario-auditor", "api", dir))
		add(&candidates, seen, filepath.Join(root, dir))
	}

	add(&candidates, seen, dir)

	debugLogManifest(manifestPath, candidates)
	return candidates
}

func debugLogManifest(manifestPath string, candidates []string) {
	if os.Getenv("SCENARIO_AUDITOR_DEBUG_BRIDGE") != "1" {
		return
	}
	fmt.Fprintf(os.Stderr, "[iframe_bridge_quality] manifest=%s candidates=%v\n", manifestPath, candidates)
}

func resolveFirstExistingPath(bases []string, relative string) string {
	relative = filepath.Clean(relative)
	for _, base := range bases {
		if base == "" {
			continue
		}
		candidate := filepath.Join(base, relative)
		if pathExists(candidate) {
			return candidate
		}
	}
	return ""
}

func findFirstExistingBridgePath(bases []string) (string, os.FileInfo) {
	for _, base := range bases {
		if base == "" {
			continue
		}
		candidate := filepath.Join(base, "node_modules", "@vrooli", "iframe-bridge")
		info, err := os.Lstat(candidate)
		if err == nil {
			return candidate, info
		}
	}
	return "", nil
}

func pathExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func scenarioFromFilePath(path string) string {
	path = filepath.ToSlash(path)
	marker := "/scenarios/"
	idx := strings.Index(path, marker)
	if idx == -1 {
		return ""
	}
	remainder := path[idx+len(marker):]
	parts := strings.SplitN(remainder, "/", 2)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func relativePackagePath(path string) string {
	path = filepath.ToSlash(path)
	marker := "/scenarios/"
	idx := strings.Index(path, marker)
	if idx == -1 {
		return filepath.ToSlash(path)
	}
	remainder := path[idx+len(marker):]
	parts := strings.SplitN(remainder, "/", 2)
	if len(parts) < 2 {
		return filepath.ToSlash(path)
	}
	return parts[1]
}
