package ui

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	rules "scenario-auditor/rules"
)

/*
Rule: Proxy-Compatible UI Base
Description: Ensure browser bundles resolve API/WebSocket endpoints through App Monitor proxy metadata (window.__APP_MONITOR_PROXY_INFO__) before falling back—via guarded branches, nullish coalescing, or ternary fallbacks—to localhost for standalone use.
Reason: Localhost URLs bypass the secure tunnel and leave scenarios blank when viewed through App Monitor unless they are clearly marked as fallbacks when proxy metadata is unavailable.
Category: ui
Severity: high
Standard: tunnel-security-v1
Targets: ui

<test-case id="hardcoded-loopback-js" should-fail="true">
  <description>Direct loopback fetch in UI bundle</description>
  <input language="javascript">
const API_URL = "http://localhost:3250/api/v1";
fetch(`${API_URL}/health`).then(r => r.json());
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>localhost endpoint</expected-message>
</test-case>

<test-case id="hostname-switch-without-proxy" should-fail="true">
  <description>Hostname switch still hardcodes localhost port</description>
  <input language="javascript">
const apiUrl = window.location.hostname === 'localhost'
  ? 'http://localhost:4455'
  : '/api';
export const API_BASE = `${apiUrl}/v1`;
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>localhost endpoint</expected-message>
</test-case>

<test-case id="hardcoded-html-endpoint" should-fail="true">
  <description>Static HTML bootstrap fetches 127.0.0.1</description>
  <input language="html">
<script>
  async function load() {
    const res = await fetch('http://127.0.0.1:3333/status');
    console.log(await res.json());
  }
  load();
</script>
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>localhost endpoint</expected-message>
</test-case>

<test-case id="proxy-aware-ui-fallback" should-fail="false">
  <description>UI reads App Monitor proxy info before localhost fallback</description>
  <input language="javascript">
const proxyInfo = window.__APP_MONITOR_PROXY_INFO__;
let apiUrl;
if (proxyInfo?.primary?.path) {
  apiUrl = `${window.location.origin}${proxyInfo.primary.path}`;
} else {
  apiUrl = `http://localhost:${window.location.port.replace('3', '1')}`;
}
  </input>
</test-case>

<test-case id="proxy-nullish-fallback" should-fail="false">
  <description>Nullish coalescing fallback after proxy metadata</description>
  <input language="javascript">
const proxyInfo = window.__APP_MONITOR_PROXY_INFO__;
const apiUrl = proxyInfo?.primary?.path ?? `http://localhost:${window.location.port}`;
  </input>
</test-case>

<test-case id="proxy-return-guard" should-fail="false">
  <description>Proxy branch returns before hitting localhost fallback</description>
  <input language="javascript">
const proxyInfo = window.__APP_MONITOR_PROXY_INFO__;
if (proxyInfo) {
  const target = `${window.location.origin}${proxyInfo.primary.path}`;
  window.open(target, '_blank');
  return;
}

window.open('http://localhost:3250', '_blank');
  </input>
</test-case>

*/

var loopbackEndpointPattern = regexp.MustCompile(`(?i)(https?:\s*//\s*(?:127\.0\.0\.1|localhost)(?::\d+)?|wss?:\s*//\s*(?:127\.0\.0\.1|localhost)(?::\d+)?|(?:^|[^a-z0-9_])(localhost|127\.0\.0\.1):\d+)`)

// CheckProxyCompatibleUIAssets flags UI assets that still depend on loopback endpoints instead of the shared proxy metadata.
func CheckProxyCompatibleUIAssets(content []byte, filePath string) []rules.Violation {
	if !isUiAssetCandidate(filePath) {
		return nil
	}

	src := string(content)
	if !strings.Contains(src, "localhost") && !strings.Contains(src, "127.0.0.1") {
		return nil
	}

	proxyAware := hasProxyIntegration(strings.ToLower(src))

	lines := strings.Split(src, "\n")
	var violations []rules.Violation
	inBlockComment := false
	inHTMLComment := false

	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)

		if inHTMLComment {
			if strings.Contains(trimmed, "-->") {
				inHTMLComment = false
			}
			continue
		}
		if strings.HasPrefix(trimmed, "<!--") {
			if !strings.Contains(trimmed, "-->") {
				inHTMLComment = true
			}
			continue
		}

		if inBlockComment {
			if strings.Contains(trimmed, "*/") {
				inBlockComment = false
			}
			continue
		}
		if strings.HasPrefix(trimmed, "/*") {
			if !strings.Contains(trimmed, "*/") {
				inBlockComment = true
			}
			continue
		}
		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		if !loopbackEndpointPattern.MatchString(line) {
			continue
		}

		if proxyAware && isGuardedProxyFallback(lines, idx) {
			continue
		}

		violations = append(violations, newProxyLoopbackViolation(filePath, idx+1, line))
	}

	return dedupeProxyLoopbackViolations(violations)
}

func hasProxyIntegration(lower string) bool {
	if strings.Contains(lower, "__app_monitor_proxy_info__") || strings.Contains(lower, "__app_monitor_proxy_index__") {
		return true
	}
	if strings.Contains(lower, "@vrooli/iframe-bridge") {
		return true
	}
	if strings.Contains(lower, "window.parent.postmessage") {
		return true
	}
	return false
}

func isUiAssetCandidate(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}

	lowered := strings.ToLower(path)
	if strings.Contains(lowered, "node_modules") || strings.Contains(lowered, "/dist/") || strings.Contains(lowered, "/build/") {
		return false
	}
	if strings.Contains(lowered, "/test/") || strings.Contains(lowered, "/tests/") || strings.Contains(lowered, "__tests__") || strings.Contains(lowered, "/stories/") {
		return false
	}

	base := strings.ToLower(filepath.Base(path))
	if strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") || strings.Contains(base, ".stories.") {
		return false
	}

	name := strings.TrimSuffix(base, filepath.Ext(base))
	if name == "server" {
		inClientTree := strings.Contains(lowered, "/src/") || strings.Contains(lowered, "/app/") || strings.Contains(lowered, "/components/") || strings.Contains(lowered, "/pages/")
		if !inClientTree {
			return false
		}
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs", ".html", ".svelte", ".vue", ".astro", ".javascript", ".typescript":
		return true
	default:
		return false
	}
}

func newProxyLoopbackViolation(filePath string, line int, snippet string) rules.Violation {
	message := "Hard-coded localhost endpoint detected; derive API/WebSocket URLs from App Monitor proxy metadata (window.__APP_MONITOR_PROXY_INFO__) and only fall back to localhost when the proxy data is unavailable."
	violation := rules.Violation{
		RuleID:         "ui_proxy_loopback",
		Type:           "ui_proxy_loopback",
		Severity:       "high",
		Title:          "Hard-coded localhost endpoint blocks App Monitor proxy",
		Message:        message,
		Description:    message,
		File:           filePath,
		FilePath:       filePath,
		Line:           line,
		LineNumber:     line,
		Recommendation: "Use App Monitor's proxy helpers (window.__APP_MONITOR_PROXY_INFO__ or the iframe bridge) before falling back to localhost.",
	}
	trimmed := strings.TrimSpace(snippet)
	if trimmed != "" {
		violation.CodeSnippet = trimmed
	}
	return violation
}

func dedupeProxyLoopbackViolations(list []rules.Violation) []rules.Violation {
	if len(list) == 0 {
		return list
	}
	seen := make(map[string]bool)
	deduped := make([]rules.Violation, 0, len(list))
	for _, v := range list {
		key := strings.Join([]string{v.Description, v.FilePath, strconv.Itoa(v.LineNumber)}, "|")
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, v)
	}
	return deduped
}

func isGuardedProxyFallback(lines []string, idx int) bool {
	const maxLookback = 40
	line := strings.ToLower(lines[idx])
	if !strings.Contains(line, "localhost") && !strings.Contains(line, "127.0.0.1") {
		return false
	}

	for i := idx; i >= 0 && idx-i <= maxLookback; i-- {
		candidate := strings.ToLower(strings.TrimSpace(lines[i]))
		if !strings.Contains(candidate, "proxyinfo") && !strings.Contains(candidate, "__app_monitor_proxy_info__") {
			continue
		}

		if strings.Contains(candidate, "!proxyinfo") || strings.Contains(candidate, "!window.__app_monitor_proxy_info__") {
			return true
		}

		if guardHasEarlyReturn(lines, i, idx, maxLookback) {
			return true
		}

		if guardHasElseBranch(lines, i, idx, maxLookback) {
			return true
		}

		if guardHasInlineFallback(lines, i, idx, maxLookback) {
			return true
		}
	}

	return false
}

func guardHasElseBranch(lines []string, start, target, maxLookback int) bool {
	for j := start + 1; j <= target && target-j <= maxLookback; j++ {
		mid := strings.ToLower(strings.TrimSpace(lines[j]))
		if strings.HasPrefix(mid, "else if") {
			continue
		}
		if strings.HasPrefix(mid, "else") || strings.Contains(mid, "} else") {
			return true
		}
	}
	return false
}

func guardHasEarlyReturn(lines []string, start, target, maxLookback int) bool {
	depth := 0
	started := false

	for j := start; j <= target && target-j <= maxLookback; j++ {
		line := lines[j]
		lower := strings.ToLower(line)

		if j == start {
			if strings.Contains(lower, "return") {
				return true
			}
			depth += strings.Count(line, "{") - strings.Count(line, "}")
			if depth > 0 {
				started = true
			}
			continue
		}

		if started && strings.Contains(lower, "return") {
			return true
		}

		open := strings.Count(line, "{")
		close := strings.Count(line, "}")
		depth += open - close

		if !started && open > 0 {
			started = true
		}

		if depth <= 0 {
			break
		}
	}

	return false
}

func guardHasInlineFallback(lines []string, start, target, maxLookback int) bool {
	var builder strings.Builder
	for j := start; j <= target && target-j <= maxLookback; j++ {
		trimmed := strings.TrimSpace(lines[j])
		if trimmed == "" {
			continue
		}
		builder.WriteString(strings.ToLower(trimmed))
		builder.WriteString(" ")
	}
	snippet := builder.String()
	if snippet == "" {
		return false
	}
	proxyIdx := strings.Index(snippet, "proxyinfo")
	if proxyIdx == -1 {
		proxyIdx = strings.Index(snippet, "__app_monitor_proxy_info__")
	}
	if proxyIdx == -1 {
		return false
	}
	locIdx := strings.Index(snippet, "localhost")
	if locIdx == -1 {
		locIdx = strings.Index(snippet, "127.0.0.1")
	}
	if locIdx == -1 || locIdx <= proxyIdx {
		return false
	}
	between := snippet[proxyIdx:locIdx]
	if strings.Contains(between, "??") || strings.Contains(between, "||") {
		return true
	}
	if strings.Contains(between, "?") && strings.Contains(between, ":") {
		return true
	}
	return false
}
