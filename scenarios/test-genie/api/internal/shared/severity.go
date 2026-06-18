package shared

import "strings"

type severityInfo struct {
	findingLabel string
	auditorLabel string
	weight       int
}

var severityAliases = map[string]severityInfo{
	"blocker":       {findingLabel: "blocker", auditorLabel: "critical", weight: 5},
	"critical":      {findingLabel: "error", auditorLabel: "critical", weight: 5},
	"error":         {findingLabel: "error", auditorLabel: "high", weight: 4},
	"failure":       {findingLabel: "error", auditorLabel: "high", weight: 4},
	"high":          {findingLabel: "error", auditorLabel: "high", weight: 4},
	"warn":          {findingLabel: "warning", auditorLabel: "medium", weight: 3},
	"warning":       {findingLabel: "warning", auditorLabel: "medium", weight: 3},
	"medium":        {findingLabel: "warning", auditorLabel: "medium", weight: 3},
	"low":           {findingLabel: "info", auditorLabel: "low", weight: 2},
	"info":          {findingLabel: "info", auditorLabel: "info", weight: 1},
	"informational": {findingLabel: "info", auditorLabel: "info", weight: 1},
	"notice":        {findingLabel: "info", auditorLabel: "info", weight: 1},
}

// NormalizeFindingSeverityLabel maps provider-specific severity vocabulary to
// the shared ArchitectureFinding severity labels.
func NormalizeFindingSeverityLabel(raw string) string {
	info, ok := severityAliases[severityKey(raw)]
	if !ok {
		return ""
	}
	return info.findingLabel
}

// NormalizeAuditorSeverity maps scenario-auditor severity vocabulary to its
// canonical display/fail-threshold labels.
func NormalizeAuditorSeverity(raw string) string {
	info, ok := severityAliases[severityKey(raw)]
	if !ok {
		return ""
	}
	return info.auditorLabel
}

// SeverityWeight returns a comparable severity rank shared by auditor
// thresholding and routing eligibility.
func SeverityWeight(raw string) int {
	info, ok := severityAliases[severityKey(raw)]
	if !ok {
		return 0
	}
	return info.weight
}

func severityKey(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.TrimPrefix(s, "finding_severity_")
	s = strings.TrimPrefix(s, "severity_")
	return s
}
