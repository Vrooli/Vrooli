package agentharness

import "strings"

// ClassifyToolEvent only classifies intent. It never claims a package manager
// from a language name; adapters provide package-manager evidence separately.
func ClassifyToolEvent(event ToolEvent) RiskClass {
	joined := strings.ToLower(strings.Join(append([]string{event.Tool}, event.Arguments...), " "))
	if strings.TrimSpace(joined) == "" {
		return RiskUnknown
	}
	if event.Shell != "" && strings.ContainsAny(event.Shell, ";|&\n\r`$()") {
		return RiskOpaque
	}
	if hasToken(joined, "publish") || strings.Contains(joined, "npm publish") || strings.Contains(joined, "cargo publish") {
		return RiskPublish
	}
	if strings.Contains(joined, "--frozen") || strings.Contains(joined, "--frozen-lockfile") || strings.Contains(joined, " npm ci") || strings.HasSuffix(joined, " npm ci") || strings.Contains(joined, "cargo fetch") || strings.Contains(joined, "go mod download") {
		return RiskFrozenReproduce
	}
	if hasToken(joined, "postinstall") || hasToken(joined, "preinstall") || hasToken(joined, "prepare") {
		return RiskLifecycle
	}
	if hasToken(joined, "add") || hasToken(joined, "install") {
		return RiskDependencyAdd
	}
	if hasToken(joined, "upgrade") || hasToken(joined, "update") {
		return RiskDependencyUpgrade
	}
	if hasToken(joined, "remove") || hasToken(joined, "uninstall") {
		return RiskDependencyRemove
	}
	if hasToken(joined, "audit") || hasToken(joined, "status") || hasToken(joined, "list") || hasToken(joined, "show") || hasToken(joined, "check") || hasToken(joined, "--help") || hasToken(joined, "--version") {
		return RiskInspection
	}
	if event.Shell != "" || len(event.Arguments) > 1 {
		return RiskOpaque
	}
	return RiskUnknown
}

func hasToken(value, wanted string) bool {
	for _, token := range strings.FieldsFunc(value, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '/' || r == ':' || r == '='
	}) {
		if token == wanted {
			return true
		}
	}
	return false
}
