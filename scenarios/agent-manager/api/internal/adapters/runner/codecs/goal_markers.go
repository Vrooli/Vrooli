package codecs

import (
	"encoding/json"
	"strings"

	"agent-manager/internal/adapters/runner"
)

// codexGoalStatus parses the event_msg/payload shape emitted by Codex TUI.
func codexGoalStatus(line string) (runner.GoalMarker, bool) {
	var envelope struct {
		Type    string `json:"type"`
		Payload struct {
			Type string `json:"type"`
			Goal struct {
				Objective string            `json:"objective"`
				Status    runner.GoalStatus `json:"status"`
			} `json:"goal"`
		} `json:"payload"`
	}
	if json.Unmarshal([]byte(line), &envelope) != nil || envelope.Type != "event_msg" || envelope.Payload.Type != "thread_goal_updated" {
		return runner.GoalMarker{}, false
	}
	marker := runner.GoalMarker{Objective: strings.TrimSpace(envelope.Payload.Goal.Objective), Status: envelope.Payload.Goal.Status}
	return marker, marker.Objective != "" && marker.Status.Valid()
}
