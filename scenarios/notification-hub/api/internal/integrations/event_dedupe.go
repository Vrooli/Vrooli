package integrations

import (
	"encoding/json"
	"strings"
)

func eventDedupeKey(eventType string, raw json.RawMessage) string {
	var facts struct {
		IncidentID string `json:"incident_id"`
		Status     string `json:"status"`
	}
	_ = json.Unmarshal(raw, &facts)
	if strings.TrimSpace(facts.IncidentID) == "" {
		return eventType
	}
	key := eventType + ":" + facts.IncidentID
	// Alert lifecycle edges are distinct notifications for one stable
	// incident. Including the edge prevents a recent open alert from
	// suppressing its recovery (or vice versa), while repeated opens and
	// repeated resolutions still collapse inside the configured dedupe window.
	status := strings.ToLower(strings.TrimSpace(facts.Status))
	if status == "open" || status == "resolved" {
		key += ":" + status
	}
	return key
}
