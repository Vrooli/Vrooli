package transcriptindex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"
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
	case runsignal.KindWebSearch:
		a.facts.WebSearches++
	case runsignal.KindWebFetch:
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

// ToolKind classifies one tool call for the per-turn tool_kinds count: its
// runsignal tool kind, with an MCP call named by server and tool, and a shell
// call that runs a Vrooli scenario's CLI named by the scenario, whose cost is
// the scenario's own rather than a runner meter (R-1).
func ToolKind(name string, input map[string]any) string {
	kind := runsignal.CapabilityForTool(name)
	switch kind {
	case runsignal.KindMCP:
		parts := strings.SplitN(name[len("mcp__"):], "__", 2)
		if len(parts) == 2 {
			return mcpKind(parts[0], parts[1])
		}
		return "mcp:" + parts[0]
	case runsignal.KindShell:
		if scenario := runsignal.CallScenario(name, input); scenario != "" {
			return "scenario:" + scenario
		}
	}
	return kind
}

func mcpKind(server, tool string) string {
	return "mcp:" + strings.TrimSpace(server) + "." + strings.TrimSpace(tool)
}
