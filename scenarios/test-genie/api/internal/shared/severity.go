package shared

import "strings"

type severityInfo struct {
	weight int
}

var severityAliases = map[string]severityInfo{
	"blocker":       {weight: 5},
	"critical":      {weight: 5},
	"error":         {weight: 4},
	"failure":       {weight: 4},
	"high":          {weight: 4},
	"warn":          {weight: 3},
	"warning":       {weight: 3},
	"medium":        {weight: 3},
	"low":           {weight: 2},
	"info":          {weight: 1},
	"informational": {weight: 1},
	"notice":        {weight: 1},
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
