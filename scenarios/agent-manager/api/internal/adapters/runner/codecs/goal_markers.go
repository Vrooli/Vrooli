package codecs

import (
	"encoding/json"
	"regexp"
	"strings"

	"agent-manager/internal/adapters/runner"

	"github.com/google/uuid"
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
	if json.Unmarshal([]byte(line), &envelope) != nil || envelope.Type != "event_msg" {
		return runner.GoalMarker{}, false
	}
	if envelope.Payload.Type != "thread_goal_updated" {
		return runner.GoalMarker{}, false
	}
	marker := runner.GoalMarker{Objective: strings.TrimSpace(envelope.Payload.Goal.Objective), Status: envelope.Payload.Goal.Status, Iteration: envelope.Payload.Goal.Iteration, LastReason: strings.TrimSpace(envelope.Payload.Goal.LastReason)}
	return marker, marker.Objective != "" && marker.Status.Valid()
}

// Goal correlation belongs to one logical transcript parser, not one line or
// one Consume segment. A new parser must replay the request prefix. Never keep
// this state on the shared codec or infer acceptance from an output-only tail.
type codexGoalContext struct {
	runID     uuid.UUID
	threadID  string
	objective string
	pending   map[string]codexGoalCall
}

type codexGoalCall struct {
	status     runner.GoalStatus
	outputType string
}

type codexAcceptedGoal struct {
	ThreadID   string            `json:"threadId"`
	Objective  string            `json:"objective"`
	Status     runner.GoalStatus `json:"status"`
	Iteration  int               `json:"iteration"`
	LastReason string            `json:"reason"`
}

type codexGoalEnvelope struct {
	Type    string `json:"type"`
	Payload struct {
		Type      string            `json:"type"`
		ID        string            `json:"id"`
		Name      string            `json:"name"`
		CallID    string            `json:"call_id"`
		Input     string            `json:"input"`
		Arguments json.RawMessage   `json:"arguments"`
		Output    json.RawMessage   `json:"output"`
		Goal      codexAcceptedGoal `json:"goal"`
		IsError   bool              `json:"is_error"`
		ErrorFlag bool              `json:"isError"`
		Success   *bool             `json:"success"`
		Error     json.RawMessage   `json:"error"`
	} `json:"payload"`
}

func (s *codexGoalContext) parse(runID uuid.UUID, line string) (runner.GoalMarker, bool) {
	if s.runID != runID {
		*s = codexGoalContext{runID: runID}
	}
	var envelope codexGoalEnvelope
	if json.Unmarshal([]byte(line), &envelope) != nil {
		return runner.GoalMarker{}, false
	}
	if envelope.Type == "session_meta" && envelope.Payload.ID != "" && envelope.Payload.ID != s.threadID {
		*s = codexGoalContext{runID: runID, threadID: envelope.Payload.ID}
	}
	if marker, ok := codexGoalStatus(line); ok {
		threadID := envelope.Payload.Goal.ThreadID
		if s.threadID != "" && threadID != "" && threadID != s.threadID {
			return runner.GoalMarker{}, false
		}
		if threadID != "" {
			s.threadID = threadID
		}
		s.objective, s.pending = marker.Objective, nil
		return marker, true
	}
	return codexGoalToolStatus(s, envelope)
}

// codexGoalToolStatus requires a typed call plus its successful native result.
// The supported exec forms only return update_goal's actual value. Arbitrary
// JavaScript (including conditionals, quoted examples and fabricated text) is
// deliberately unsupported; this is not a JavaScript interpreter.
func codexGoalToolStatus(s *codexGoalContext, envelope codexGoalEnvelope) (runner.GoalMarker, bool) {
	p := envelope.Payload
	if envelope.Type != "response_item" || p.CallID == "" || len(p.CallID) > 256 {
		return runner.GoalMarker{}, false
	}
	if p.Type == "custom_tool_call" || p.Type == "function_call" {
		status, ok := codexGoalRequestedStatus(envelope)
		if !ok {
			delete(s.pending, p.CallID)
			return runner.GoalMarker{}, false
		}
		call := codexGoalCall{status: status, outputType: p.Type + "_output"}
		if prior, exists := s.pending[p.CallID]; exists {
			if prior != call {
				s.pending[p.CallID] = codexGoalCall{} // conflicting ID cannot qualify
			}
			return runner.GoalMarker{}, false
		}
		if len(s.pending) < 128 {
			if s.pending == nil {
				s.pending = make(map[string]codexGoalCall)
			}
			s.pending[p.CallID] = call
		}
		return runner.GoalMarker{}, false
	}
	if p.Type != "custom_tool_call_output" && p.Type != "function_call_output" {
		return runner.GoalMarker{}, false
	}
	call, matched := s.pending[p.CallID]
	delete(s.pending, p.CallID) // rejected/ambiguous results cannot later be retried as success
	if !matched || call.outputType != p.Type || p.IsError || p.ErrorFlag || nonNullJSON(p.Error) || (p.Success != nil && !*p.Success) {
		return runner.GoalMarker{}, false
	}
	goal, ok := codexAcceptedGoalOutput(p.Output)
	if !ok || goal.Status != call.status || (s.threadID != "" && goal.ThreadID != s.threadID) || (s.objective != "" && strings.TrimSpace(goal.Objective) != s.objective) {
		return runner.GoalMarker{}, false
	}
	s.threadID, s.objective = goal.ThreadID, strings.TrimSpace(goal.Objective)
	return runner.GoalMarker{Objective: s.objective, Status: goal.Status, Iteration: goal.Iteration, LastReason: strings.TrimSpace(goal.LastReason)}, true
}

var (
	codexGoalAssignedCall = regexp.MustCompile(`^\s*(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=\s*await\s+tools\.update_goal\(\s*(\{[^{}]*\})\s*\)\s*;\s*text\(\s*([A-Za-z_$][A-Za-z0-9_$]*)\s*\)\s*;?\s*$`)
	codexGoalReturnedCall = regexp.MustCompile(`^\s*text\(\s*await\s+tools\.update_goal\(\s*(\{[^{}]*\})\s*\)\s*\)\s*;?\s*$`)
	codexGoalStatusArg    = regexp.MustCompile(`^\{\s*(?:status|"status"|'status')\s*:\s*(?:"([a-z_]+)"|'([a-z_]+)')\s*,?\s*\}$`)
)

func codexGoalRequestedStatus(envelope codexGoalEnvelope) (runner.GoalStatus, bool) {
	p := envelope.Payload
	var status runner.GoalStatus
	switch {
	case p.Type == "custom_tool_call" && (p.Name == "exec" || p.Name == "functions.exec") && len(p.Input) <= 4096:
		argument := ""
		if match := codexGoalAssignedCall.FindStringSubmatch(p.Input); match != nil && match[1] == match[3] {
			argument = match[2]
		} else if match := codexGoalReturnedCall.FindStringSubmatch(p.Input); match != nil {
			argument = match[1]
		}
		if match := codexGoalStatusArg.FindStringSubmatch(argument); match != nil {
			status = runner.GoalStatus(match[1] + match[2])
		}
	case p.Type == "function_call" && (p.Name == "update_goal" || p.Name == "tools.update_goal" || p.Name == "functions.update_goal") && len(p.Arguments) <= 4096:
		argument := p.Arguments
		var encoded string
		if json.Unmarshal(argument, &encoded) == nil {
			argument = []byte(encoded)
		}
		var request struct {
			Status runner.GoalStatus `json:"status"`
		}
		if json.Unmarshal(argument, &request) == nil {
			status = request.Status
		}
	}
	return status, status.Valid() && status != runner.GoalStatusActive && status != runner.GoalStatusPaused
}

func codexAcceptedGoalOutput(output json.RawMessage) (codexAcceptedGoal, bool) {
	if len(output) == 0 || len(output) > 64<<10 {
		return codexAcceptedGoal{}, false
	}
	var texts []string
	var text string
	if json.Unmarshal(output, &text) == nil {
		texts = []string{text}
	} else {
		var blocks []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if json.Unmarshal(output, &blocks) != nil || len(blocks) == 0 || len(blocks) > 16 {
			return codexAcceptedGoal{}, false
		}
		for _, block := range blocks {
			if block.Type != "input_text" {
				return codexAcceptedGoal{}, false
			}
			texts = append(texts, block.Text)
		}
	}
	var accepted codexAcceptedGoal
	found := false
	for _, text := range texts {
		// Native exec emits this metadata block before its structured result.
		if strings.HasPrefix(text, "Script completed\nWall time ") && strings.HasSuffix(text, "\nOutput:\n") {
			continue
		}
		var receipt struct {
			Goal    codexAcceptedGoal `json:"goal"`
			Error   json.RawMessage   `json:"error"`
			IsError bool              `json:"isError"`
			Failed  bool              `json:"is_error"`
			Success *bool             `json:"success"`
		}
		if json.Unmarshal([]byte(text), &receipt) != nil || found || receipt.IsError || receipt.Failed || nonNullJSON(receipt.Error) || (receipt.Success != nil && !*receipt.Success) {
			return codexAcceptedGoal{}, false
		}
		goal := receipt.Goal
		if strings.TrimSpace(goal.ThreadID) == "" || strings.TrimSpace(goal.Objective) == "" || !goal.Status.Valid() {
			return codexAcceptedGoal{}, false
		}
		accepted, found = goal, true
	}
	return accepted, found
}

func nonNullJSON(value json.RawMessage) bool {
	return len(value) != 0 && strings.TrimSpace(string(value)) != "null"
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
	if array, ok := value.([]any); ok {
		for _, child := range array {
			if marker, found := findClaudeGoalStatus(child); found {
				return marker, true
			}
		}
		return runner.GoalMarker{}, false
	}
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
