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
				Objective  string            `json:"objective"`
				Status     runner.GoalStatus `json:"status"`
				Iteration  int               `json:"iteration,omitempty"`
				LastReason string            `json:"reason,omitempty"`
			} `json:"goal"`
		} `json:"payload"`
	}
	if json.Unmarshal([]byte(line), &envelope) != nil || envelope.Type != "event_msg" || envelope.Payload.Type != "thread_goal_updated" {
		return runner.GoalMarker{}, false
	}
	marker := runner.GoalMarker{Objective: strings.TrimSpace(envelope.Payload.Goal.Objective), Status: envelope.Payload.Goal.Status, Iteration: envelope.Payload.Goal.Iteration, LastReason: strings.TrimSpace(envelope.Payload.Goal.LastReason)}
	return marker, marker.Objective != "" && marker.Status.Valid()
}

// claudeGoalStatus parses the attachment goal_status shape emitted by Claude's
// on-disk interactive transcript. It walks JSON objects only, so assistant
// prose cannot masquerade as an engine-owned marker.
func claudeGoalStatus(line string) (runner.GoalMarker, bool) {
	var value any
	if json.Unmarshal([]byte(strings.TrimSpace(line)), &value) != nil {
		return runner.GoalMarker{}, false
	}
	return findClaudeGoalStatus(value)
}

func findClaudeGoalStatus(value any) (runner.GoalMarker, bool) {
	object, ok := value.(map[string]any)
	if !ok {
		return runner.GoalMarker{}, false
	}
	if object["type"] == "goal_status" {
		condition, _ := object["condition"].(string)
		met, _ := object["met"].(bool)
		iteration, _ := object["iterations"].(float64)
		if iteration == 0 {
			iteration, _ = object["iteration"].(float64)
		}
		reason, _ := object["reason"].(string)
		if strings.TrimSpace(condition) != "" {
			status := runner.GoalStatusActive
			if met {
				status = runner.GoalStatusComplete
			}
			return runner.GoalMarker{Objective: strings.TrimSpace(condition), Status: status, Iteration: int(iteration), LastReason: strings.TrimSpace(reason)}, true
		}
	}
	for _, child := range object {
		if marker, ok := findClaudeGoalStatus(child); ok {
			return marker, true
		}
	}
	return runner.GoalMarker{}, false
}
