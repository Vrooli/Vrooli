package agenttranscript

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

// TranscriptEvidenceOptions describes policy hooks and bounds for extracting
// import metadata from JSONL transcripts. The extractor understands generic
// JSON shapes; callers retain ownership of injected-context and goal-marker
// policy through the callbacks.
type TranscriptEvidenceOptions struct {
	ExtractLabels        bool
	MaxLineBytes         int
	GoalMaxLineBytes     int
	TitleSourceBytes     int
	TitleItemBytes       int
	UserPromptBytes      int
	GoalPreambleMinBytes int
	IsInjectedContext    func(string) bool
	ParseGoalStatusLine  func(string) (objective, status string, ok bool)
}

// TranscriptEvidence contains text derived from one native transcript scan.
// GoalMarker fields are opaque to the codec; the caller decides how to map
// them to its own status and identity model.
type TranscriptEvidence struct {
	HarnessTitle        string
	UserPrompt          string
	TitleSource         string
	GoalPreamble        string
	GoalMarkerFound     bool
	GoalMarkerObjective string
	GoalMarkerStatus    string
}

// ExtractTranscriptEvidence scans JSONL once for generic transcript title,
// message, and goal evidence. MaxLineBytes bounds the shared scan. When
// GoalMaxLineBytes is smaller, goal extraction stops at its old scanner limit
// while label extraction can continue with the same line stream.
func ExtractTranscriptEvidence(source io.Reader, options TranscriptEvidenceOptions) (TranscriptEvidence, error) {
	var evidence TranscriptEvidence
	if options.MaxLineBytes <= 0 {
		options.MaxLineBytes = 64 << 20
	}
	if options.TitleSourceBytes < 0 {
		options.TitleSourceBytes = 0
	}
	scanner := bufio.NewScanner(source)
	bufferBytes := min(64*1024, options.MaxLineBytes)
	scanner.Buffer(make([]byte, bufferBytes), options.MaxLineBytes)
	goalEnabled := options.GoalPreambleMinBytes > 0 || options.ParseGoalStatusLine != nil
	goalLineLimit := options.GoalMaxLineBytes
	if goalLineLimit <= 0 || goalLineLimit > options.MaxLineBytes {
		goalLineLimit = options.MaxLineBytes
	}
	var title strings.Builder
	seen := make(map[string]bool)
	for scanner.Scan() {
		line := scanner.Bytes()
		if goalEnabled && len(line) >= goalLineLimit {
			goalEnabled = false
		}
		if goalEnabled && options.ParseGoalStatusLine != nil && !evidence.GoalMarkerFound {
			if objective, status, ok := options.ParseGoalStatusLine(string(line)); ok {
				evidence.GoalMarkerFound = true
				evidence.GoalMarkerObjective = objective
				evidence.GoalMarkerStatus = status
				if !options.ExtractLabels {
					break
				}
			}
		}
		var record any
		if json.Unmarshal(line, &record) != nil {
			continue
		}
		if options.ExtractLabels && evidence.HarnessTitle == "" {
			evidence.HarnessTitle = findJSONLineString(record, func(object map[string]any) (string, bool) {
				kind, _ := object["type"].(string)
				if kind != "ai-title" {
					return "", false
				}
				title, _ := object["aiTitle"].(string)
				title = strings.TrimSpace(title)
				return title, title != ""
			})
		}
		if options.ExtractLabels && evidence.UserPrompt == "" {
			candidate := findJSONLineString(record, jsonLineUserText)
			if candidate != "" && (options.IsInjectedContext == nil || !options.IsInjectedContext(candidate)) {
				evidence.UserPrompt = shortenJSONLineText(candidate, options.UserPromptBytes)
			}
		}
		if options.ExtractLabels && title.Len() < options.TitleSourceBytes {
			item := ""
			if candidate := findJSONLineString(record, jsonLineUserText); candidate != "" &&
				(options.IsInjectedContext == nil || !options.IsInjectedContext(candidate)) {
				item = "User: " + shortenJSONLineText(candidate, options.UserPromptBytes)
			} else if candidate := findJSONLineString(record, jsonLineAgentText); candidate != "" {
				item = "Agent: " + truncateJSONLineUTF8(candidate, options.TitleItemBytes)
			}
			if item != "" && !seen[item] {
				seen[item] = true
				title.WriteString(truncateJSONLineUTF8(item, options.TitleSourceBytes-title.Len()-1))
				title.WriteByte('\n')
			}
		}
		if goalEnabled && !evidence.GoalMarkerFound && evidence.GoalPreamble == "" && options.GoalPreambleMinBytes > 0 {
			evidence.GoalPreamble = findJSONLineGoalPreamble(record, options.GoalPreambleMinBytes)
		}
	}
	evidence.TitleSource = title.String()
	return evidence, scanner.Err()
}

func findJSONLineString(value any, match func(map[string]any) (string, bool)) string {
	switch item := value.(type) {
	case map[string]any:
		if result, ok := match(item); ok {
			return result
		}
		for _, child := range item {
			if result := findJSONLineString(child, match); result != "" {
				return result
			}
		}
	case []any:
		for _, child := range item {
			if result := findJSONLineString(child, match); result != "" {
				return result
			}
		}
	}
	return ""
}

func jsonLineUserText(object map[string]any) (string, bool) {
	if role, _ := object["role"].(string); !strings.EqualFold(strings.TrimSpace(role), "user") {
		if kind, _ := object["type"].(string); kind != "user_message" {
			return "", false
		}
	}
	for _, key := range []string{"content", "message", "text", "prompt"} {
		if text := normalizeJSONLineText(object[key], true); text != "" {
			return text, true
		}
	}
	return "", false
}

func jsonLineAgentText(object map[string]any) (string, bool) {
	role, _ := object["role"].(string)
	kind, _ := object["type"].(string)
	if !strings.EqualFold(strings.TrimSpace(role), "assistant") && kind != "agent_message" {
		return "", false
	}
	for _, key := range []string{"content", "message", "text"} {
		if text := normalizeJSONLineText(object[key], true); text != "" {
			return text, true
		}
	}
	return "", false
}

func normalizeJSONLineText(value any, includeMessage bool) string {
	switch item := value.(type) {
	case string:
		return strings.Join(strings.Fields(item), " ")
	case []any:
		parts := make([]string, 0, len(item))
		for _, child := range item {
			if text := normalizeJSONLineText(child, includeMessage); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, " ")
	case map[string]any:
		keys := []string{"text", "content", "value"}
		if includeMessage {
			keys = append(keys, "message")
		}
		for _, key := range keys {
			if text := normalizeJSONLineText(item[key], includeMessage); text != "" {
				return text
			}
		}
	}
	return ""
}

func findJSONLineGoalPreamble(value any, minBytes int) string {
	object, ok := value.(map[string]any)
	if !ok {
		if array, ok := value.([]any); ok {
			for _, child := range array {
				if preamble := findJSONLineGoalPreamble(child, minBytes); preamble != "" {
					return preamble
				}
			}
		}
		return ""
	}
	role, _ := object["role"].(string)
	if strings.EqualFold(strings.TrimSpace(role), "user") {
		for _, key := range []string{"content", "message", "text", "prompt"} {
			if text := normalizeJSONLineText(object[key], false); len(text) >= minBytes {
				return text
			}
		}
	}
	for _, child := range object {
		if preamble := findJSONLineGoalPreamble(child, minBytes); preamble != "" {
			return preamble
		}
	}
	return ""
}

func shortenJSONLineText(text string, maxBytes int) string {
	text = strings.TrimSpace(text)
	if maxBytes > 0 && len(text) > maxBytes {
		return text[:maxBytes-3] + "…"
	}
	return text
}

func truncateJSONLineUTF8(text string, maxBytes int) string {
	if maxBytes <= 0 || len(text) <= maxBytes {
		if maxBytes <= 0 {
			return ""
		}
		return text
	}
	return strings.ToValidUTF8(text[:maxBytes], "")
}
