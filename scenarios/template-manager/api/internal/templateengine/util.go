package templateengine

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

func truncateForIssue(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "... (truncated)"
}

func formatTemplateValidationIssues(issues []templatecontracts.TemplateValidationIssue) string {
	if len(issues) == 0 {
		return ""
	}
	lines := make([]string, 0, len(issues))
	for _, issue := range issues {
		line := issue.Template
		if strings.TrimSpace(issue.Path) != "" {
			line += " [" + issue.Path + "]"
		}
		line += ": " + issue.Message
		lines = append(lines, line)
	}
	return strings.Join(lines, "; ")
}

func randomTemplateToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func copyStringMap(values map[string]string) map[string]string {
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func coalesce(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// validationStatus is the canonical pass/fail verdict string used across the
// template, design, and resource validation reports.
func validationStatus(passed bool) string {
	if passed {
		return "pass"
	}
	return "fail"
}

func FormatTemplateRequiredFlags(manifest templatecontracts.TemplateManifest) string {
	keys := make([]string, 0, len(manifest.RequiredVars))
	for key := range manifest.RequiredVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return " --id <slug>"
	}
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		flag := manifest.RequiredVars[key].Flag
		if flag == "" {
			flag = strings.ToLower(key)
		}
		parts = append(parts, fmt.Sprintf(" --%s <%s>", flag, strings.ToLower(key)))
	}
	return strings.Join(parts, "")
}
