package shared

import "strings"

type severityInfo struct {
	findingLabel string
	weight       int
}

var severityAliases = map[string]severityInfo{
	"blocker":       {findingLabel: "blocker", weight: 5},
	"critical":      {findingLabel: "error", weight: 5},
	"error":         {findingLabel: "error", weight: 4},
	"failure":       {findingLabel: "error", weight: 4},
	"high":          {findingLabel: "error", weight: 4},
	"warn":          {findingLabel: "warning", weight: 3},
	"warning":       {findingLabel: "warning", weight: 3},
	"medium":        {findingLabel: "warning", weight: 3},
	"low":           {findingLabel: "info", weight: 2},
	"info":          {findingLabel: "info", weight: 1},
	"informational": {findingLabel: "info", weight: 1},
	"notice":        {findingLabel: "info", weight: 1},
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

// SeverityWeight returns a comparable severity rank shared by routing
// eligibility and finding normalization.
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
