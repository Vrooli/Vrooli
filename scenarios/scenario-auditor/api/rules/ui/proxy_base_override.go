package ui

import (
	"path/filepath"
	"regexp"
	"strings"

	rules "scenario-auditor/rules"
)

/*
Rule: Proxy Base Preservation
Description: Prevent UI bundles from rebuilding API base URLs using window.location.origin and dropping the /apps/.../proxy prefix required by the secure tunnel.
Reason: Different cloud environments insert the proxy path dynamically; reconstructing the base manually often omits the proxy segment and breaks all API calls.
Category: ui
Severity: high
Standard: tunnel-security-v1
Targets: ui

<test-case id="rebuilds-api-base-with-origin" should-fail="true" path="ui/src/App.tsx">
  <description>resolveApiBase result is rewritten with window.location.origin and loses proxy prefix</description>
  <input language="typescript">
import { resolveApiBase } from '@vrooli/api-base';

const API_BASE_INPUT = resolveApiBase();

function normalizeApiBaseForRuntime(base: string): string {
  const origin = window.location.origin;
  const parsed = new URL(base);
  return `${origin}${parsed.pathname}`;
}

const API_BASE_URL = normalizeApiBaseForRuntime(API_BASE_INPUT);
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>window.location.origin</expected-message>
</test-case>

<test-case id="uses-resolve-api-base-directly" should-fail="false" path="ui/src/App.tsx">
  <description>resolveApiBase output consumed directly without rebuilding the origin</description>
  <input language="typescript">
import { resolveApiBase } from '@vrooli/api-base';

const API_BASE_URL = resolveApiBase({ appendSuffix: true });

fetch(`${API_BASE_URL}/health`);
  </input>
</test-case>

<test-case id="origin-used-for-unrelated-link" should-fail="false" path="ui/src/App.tsx">
  <description>window.location.origin used for non-API navigation while resolveApiBase is untouched</description>
  <input language="typescript">
import { resolveApiBase } from '@vrooli/api-base';

const API_BASE_URL = resolveApiBase();

const homeLink = `${window.location.origin}/apps`;
console.log('home link', homeLink);
  </input>
</test-case>
*/

var (
	originVariableRegex   = regexp.MustCompile(`(?m)(?:const|let|var)\s+([a-zA-Z_$][\w$]*)\s*=\s*(?:[a-zA-Z_$][\w$]*\()?\s*window\.location\.origin`)
	resolveApiBasePattern = regexp.MustCompile(`resolveApiBase\s*\(`)
)

// CheckProxyBasePreservation flags files that rebuild API bases using window.location.origin
// after calling resolveApiBase, which strips the secure tunnel prefix.
func CheckProxyBasePreservation(content []byte, filePath string) []rules.Violation {
	if !isClientRuntimeFile(filePath) {
		return nil
	}

	source := string(content)
	if !resolveApiBasePattern.Match(content) {
		return nil
	}

	matches := originVariableRegex.FindAllStringSubmatch(source, -1)
	if len(matches) == 0 {
		return nil
	}

	var violations []rules.Violation
	for _, match := range matches {
		variable := match[1]
		if variable == "" {
			continue
		}

		if !originVariableReusedForApiBase(source, variable) {
			continue
		}

		line := findLineNumberProxy(source, match[0])
		violations = append(violations, rules.Violation{
			RuleID:         "ui-proxy-base-preservation",
			Type:           "ui_proxy_base_preservation",
			Severity:       "high",
			Title:          "API base rebuilt without proxy prefix",
			Message:        "Do not rebuild API bases with window.location.origin; rely on resolveApiBase/buildApiUrl so the secure tunnel prefix is preserved.",
			Description:    "Do not rebuild API bases with window.location.origin; rely on resolveApiBase/buildApiUrl so the secure tunnel prefix is preserved.",
			File:           filePath,
			FilePath:       filePath,
			Line:           line,
			LineNumber:     line,
			Recommendation: "Use resolveApiBase output directly or call buildApiUrl instead of stitching window.origin and pathname manually.",
			Standard:       "tunnel-security-v1",
		})
	}

	return violations
}

func isClientRuntimeFile(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".js", ".ts", ".tsx", ".jsx", ".mjs", ".cjs":
	default:
		return false
	}

	lower := strings.ToLower(path)
	if strings.Contains(lower, "/node_modules/") {
		return false
	}
	if strings.Contains(lower, "/dist/") || strings.Contains(lower, "/build/") {
		return false
	}

	return strings.Contains(lower, "/src/") || strings.Contains(lower, "/ui/") || strings.Contains(lower, "/client")
}

func originVariableReusedForApiBase(source, variable string) bool {
	needleTemplate := "${" + variable + "}"
	if strings.Contains(source, needleTemplate) {
		return true
	}

	concatenationPatterns := []string{
		variable + " + ",
		"+ " + variable,
		variable + ")",
	}
	for _, pattern := range concatenationPatterns {
		if strings.Contains(source, pattern) {
			return true
		}
	}

	assignmentPatterns := []string{
		"const API_BASE", "let API_BASE", "var API_BASE",
		"const apiBase", "let apiBase", "var apiBase",
	}
	for _, assignment := range assignmentPatterns {
		if !strings.Contains(source, assignment) {
			continue
		}
		segment := extractAssignmentSegment(source, assignment)
		if strings.Contains(segment, variable) {
			return true
		}
	}

	return false
}

func extractAssignmentSegment(source, marker string) string {
	idx := strings.Index(source, marker)
	if idx == -1 {
		return ""
	}

	end := idx + len(marker)
	if end >= len(source) {
		return source[idx:]
	}

	limit := end + 200
	if limit > len(source) {
		limit = len(source)
	}

	return source[idx:limit]
}

func findLineNumberProxy(source, needle string) int {
	if needle == "" {
		return 1
	}
	lines := strings.Split(source, "\n")
	for idx, line := range lines {
		if strings.Contains(line, needle) {
			return idx + 1
		}
	}
	return 1
}
