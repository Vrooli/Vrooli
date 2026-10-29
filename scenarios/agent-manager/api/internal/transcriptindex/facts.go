package transcriptindex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"agent-manager/internal/domain"
)

// factMarkers gate the structural read of a line: most transcript lines carry
// no non-text work, and a substring test is far cheaper than a second JSON
// decode of a multi-megabyte line.
var factMarkers = [][]byte{
	[]byte(`input_image`), []byte(`"type":"image"`), []byte(`input_audio`),
	[]byte(`web_search`), []byte(`image_generation`), []byte(`McpToolCall`), []byte(`"credits"`),
}

// factAccumulator collects the non-text work between two usage reports. The
// counts are attached to the next report: that request is the one that read
// the image or the search result.
type factAccumulator struct {
	facts TurnFacts
	// Codex older rollouts log both the call item and an *_end event for one
	// web search or image generation; the larger of the two is the count.
	webSearchCalls, webSearchEnds int
	imageGenCalls, imageGenEnds   int
}

// pendingFacts is the durable form of an accumulator (F-W2-2): the work read
// after a file's newest usage report is saved with the file's offset, so a
// restart that resumes at that offset still attaches it to the next report.
type pendingFacts struct {
	Facts          TurnFacts `json:"facts"`
	WebSearchCalls int       `json:"webSearchCalls,omitempty"`
	WebSearchEnds  int       `json:"webSearchEnds,omitempty"`
	ImageGenCalls  int       `json:"imageGenCalls,omitempty"`
	ImageGenEnds   int       `json:"imageGenEnds,omitempty"`
}

func (a *factAccumulator) encode() string {
	encoded, err := json.Marshal(pendingFacts{Facts: a.facts, WebSearchCalls: a.webSearchCalls, WebSearchEnds: a.webSearchEnds, ImageGenCalls: a.imageGenCalls, ImageGenEnds: a.imageGenEnds})
	if err != nil {
		return ""
	}
	return string(encoded)
}

func decodeFacts(encoded string) factAccumulator {
	var pending pendingFacts
	if encoded == "" || json.Unmarshal([]byte(encoded), &pending) != nil {
		return factAccumulator{}
	}
	return factAccumulator{facts: pending.Facts, webSearchCalls: pending.WebSearchCalls, webSearchEnds: pending.WebSearchEnds, imageGenCalls: pending.ImageGenCalls, imageGenEnds: pending.ImageGenEnds}
}

func (a *factAccumulator) take() TurnFacts {
	facts := a.facts
	facts.WebSearches += max(a.webSearchCalls, a.webSearchEnds)
	facts.ImagesOut += max(a.imageGenCalls, a.imageGenEnds)
	credits := facts.CreditsBalance
	*a = factAccumulator{}
	// The credits snapshot is a standing value, not a count: keep it so the
	// next report repeats it until Codex reports a new balance.
	a.facts.CreditsBalance = credits
	return facts
}

func (a *factAccumulator) addTool(kind string) {
	if a.facts.ToolKinds == nil {
		a.facts.ToolKinds = map[string]int{}
	}
	a.facts.ToolCalls++
	a.facts.ToolKinds[kind]++
	switch kind {
	case "web_search":
		a.facts.WebSearches++
	case "web_fetch":
		a.facts.WebFetches++
	}
}

// observeEvents counts tool calls by kind from the parser's own events.
func (a *factAccumulator) observeEvents(events []*domain.RunEvent) {
	for _, event := range events {
		if event == nil || event.EventType != domain.EventTypeToolCall {
			continue
		}
		if call, ok := event.Data.(*domain.ToolCallEventData); ok {
			a.addTool(ToolKind(call.ToolName, call.Input))
		}
	}
}

// observeLine counts the non-text work a raw transcript line records that the
// parsers do not surface as events: images in tool results and user messages,
// generated images, web searches, audio, MCP connector calls and the Codex
// credits snapshot.
func (a *factAccumulator) observeLine(line string) {
	raw := []byte(line)
	marked := false
	for _, marker := range factMarkers {
		if bytes.Contains(raw, marker) {
			marked = true
			break
		}
	}
	if !marked {
		return
	}
	var record map[string]any
	if json.Unmarshal(raw, &record) != nil {
		return
	}
	switch record["type"] {
	case "response_item":
		payload, _ := record["payload"].(map[string]any)
		switch payload["type"] {
		case "web_search_call":
			a.webSearchCalls++
		case "image_generation_call":
			a.imageGenCalls++
		default:
			user := payload["type"] == "message" && payload["role"] == "user"
			a.countMedia(payload, user)
		}
	case "event_msg":
		payload, _ := record["payload"].(map[string]any)
		switch payload["type"] {
		case "web_search_end":
			a.webSearchEnds++
		case "image_generation_end":
			a.imageGenEnds++
		case "item_completed":
			if item, ok := payload["item"].(map[string]any); ok && item["type"] == "McpToolCall" {
				a.addTool(mcpKind(fmt.Sprint(item["server"]), fmt.Sprint(item["tool"])))
			}
		case "token_count":
			if limits, ok := payload["rate_limits"].(map[string]any); ok {
				if credits, ok := limits["credits"].(map[string]any); ok && credits["balance"] != nil {
					a.facts.CreditsBalance = fmt.Sprint(credits["balance"])
				}
			}
		}
	case "user":
		// Claude: a user line carries operator-attached images directly and
		// tool-result images nested in tool_result blocks.
		if message, ok := record["message"].(map[string]any); ok {
			if blocks, ok := message["content"].([]any); ok {
				for _, block := range blocks {
					object, _ := block.(map[string]any)
					if object["type"] == "image" {
						a.countImage(object, true)
					} else if object["type"] == "tool_result" {
						a.countMedia(object, false)
					}
				}
			}
		}
	}
}

// countMedia walks one content tree for image and audio parts.
func (a *factAccumulator) countMedia(value any, user bool) {
	switch item := value.(type) {
	case map[string]any:
		switch item["type"] {
		case "input_image", "image":
			a.countImage(item, user)
			return
		case "input_audio":
			a.facts.AudioIn++
			return
		}
		for _, child := range item {
			a.countMedia(child, user)
		}
	case []any:
		for _, child := range item {
			a.countMedia(child, user)
		}
	}
}

func (a *factAccumulator) countImage(image map[string]any, user bool) {
	a.facts.ImagesIn++
	if user {
		a.facts.ImagesInUser++
	}
	if detail, _ := image["detail"].(string); detail == "high" || detail == "original" {
		a.facts.ImagesInHighDetail++
	}
	data, _ := image["image_url"].(string)
	if source, ok := image["source"].(map[string]any); ok {
		data, _ = source["data"].(string)
	}
	if comma := strings.IndexByte(data, ','); comma >= 0 && strings.HasPrefix(data, "data:") {
		data = data[comma+1:]
	}
	a.facts.ImageBytesIn += int64(len(data)) * 3 / 4
}

// scenarioCLIs are Vrooli scenarios reached through the shell whose cost is
// the scenario's own, not a runner meter (R-1).
var scenarioCLIs = []string{"image-tools", "whisper", "tts"}

// ToolKind classifies one tool call for the per-turn tool_kinds count.
func ToolKind(name string, input map[string]any) string {
	switch {
	case strings.HasPrefix(name, "mcp__"):
		parts := strings.SplitN(strings.TrimPrefix(name, "mcp__"), "__", 2)
		if len(parts) == 2 {
			return mcpKind(parts[0], parts[1])
		}
		return "mcp:" + parts[0]
	}
	switch strings.ToLower(name) {
	case "exec", "exec_command", "bash", "shell", "local_shell", "command_execution", "write_stdin":
		command := strings.ToLower(fmt.Sprint(input))
		for _, cli := range scenarioCLIs {
			if strings.Contains(command, cli) {
				return "scenario:" + cli
			}
		}
		return "shell"
	case "websearch", "web_search":
		return "web_search"
	case "webfetch", "web_fetch":
		return "web_fetch"
	case "read", "view_image", "notebookread":
		return "file_read"
	case "edit", "write", "multiedit", "apply_patch", "notebookedit", "file_change":
		return "file_edit"
	case "grep", "glob", "toolsearch":
		return "search"
	case "agent", "task", "spawn_agent", "send_message", "sendmessage", "wait_agent", "followup_task", "list_agents", "interrupt_agent":
		return "agent"
	case "sleep", "wait", "schedulewakeup", "monitor":
		return "wait"
	case "image_gen__imagegen", "imagegen", "image_generation":
		return "image_generation"
	}
	return "other"
}

func mcpKind(server, tool string) string {
	return "mcp:" + strings.TrimSpace(server) + "." + strings.TrimSpace(tool)
}
