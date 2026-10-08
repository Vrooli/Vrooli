// Responsibility: retain claude declarations within their original package.
package codecs

import (
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/config"
	"agent-manager/internal/domain"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// parseClaudeStreamEvents parses one line from Claude's stream-json
// output. Returns multiple events to preserve tool calls/results emitted
// inside one message. State (text buffer, tool-use accumulator, captured
// session_id, /compact tracking) is mutated in place.
func parseClaudeStreamEvents(state *claudeState, runID uuid.UUID, line string) ([]*domain.RunEvent, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}

	// Quick check: valid JSON objects start with '{', arrays with '['.
	// Skip non-JSON banner lines like "Initializing...", "[Info] ...".
	firstChar := line[0]
	if firstChar != '{' && firstChar != '[' {
		return nil, nil
	}
	if firstChar == '[' && len(line) > 1 {
		secondChar := line[1]
		if (secondChar >= 'A' && secondChar <= 'Z') || (secondChar >= 'a' && secondChar <= 'z') {
			return nil, nil
		}
	}

	var streamEvent ClaudeStreamEvent
	if err := json.Unmarshal([]byte(line), &streamEvent); err != nil {
		// Silently skip malformed JSON from startup/debug output. Real
		// streaming events from Claude Code are always well-formed.
		return nil, nil
	}

	if state.sessionID == "" {
		if streamEvent.SessionID != "" {
			state.sessionID = streamEvent.SessionID
		} else if streamEvent.SessionIDAlt != "" {
			state.sessionID = streamEvent.SessionIDAlt
		}
	}

	switch streamEvent.Type {
	case "message":
		return parseMessageEvent(state, runID, &streamEvent), nil

	case "assistant":
		return parseAssistantEvent(state, runID, &streamEvent), nil

	case "user":
		return parseUserEvent(state, runID, &streamEvent), nil

	case "tool_use":
		if streamEvent.ToolUse != nil {
			var input map[string]interface{}
			if streamEvent.ToolUse.Input != nil {
				_ = json.Unmarshal(streamEvent.ToolUse.Input, &input)
			}
			return []*domain.RunEvent{domain.NewToolCallEvent(
				runID,
				streamEvent.ToolUse.Name,
				streamEvent.ToolUse.ID,
				input,
			)}, nil
		}

	case "tool_result":
		var resultStr string
		if streamEvent.Result != nil {
			_ = json.Unmarshal(streamEvent.Result, &resultStr)
		}
		resultStr = runner.StripANSI(resultStr)
		return []*domain.RunEvent{domain.NewToolResultEvent(
			runID, "", streamEvent.ParentToolUseID, resultStr, nil,
		)}, nil

	case "error":
		if streamEvent.Error != nil {
			return []*domain.RunEvent{domain.NewErrorEvent(
				runID,
				streamEvent.Error.Code,
				streamEvent.Error.Message,
				false,
			)}, nil
		}

	case "usage":
		if streamEvent.Usage != nil {
			return []*domain.RunEvent{domain.NewMetricEvent(
				runID,
				"tokens",
				float64(streamEvent.Usage.InputTokens+streamEvent.Usage.OutputTokens),
				"tokens",
			)}, nil
		}

	case "result":
		state.gotResult = true
		state.resultIsError = streamEvent.IsError

		var events []*domain.RunEvent
		var resultStr string
		if streamEvent.Result != nil {
			_ = json.Unmarshal(streamEvent.Result, &resultStr)
		}
		if !streamEvent.IsError {
			if resultStr == "" {
				resultStr = state.lastAssistant
			}
			if resultStr != "" {
				state.lastAssistant = resultStr
				if state.lastMessageEvent != nil && messageEventContent(state.lastMessageEvent) == resultStr {
					markProviderMessageTerminal(state.lastMessageEvent, streamEvent.Subtype, "result", "stdout:result")
					events = append(events, newProviderTerminalEvidence(runID, state.lastMessageEvent, streamEvent.Subtype, "result", "stdout:result"))
				} else {
					terminalEvent := domain.NewProviderMessageEvent(runID, "assistant", resultStr, domain.MessageEventData{
						ConversationID:    streamEvent.SessionID,
						ProviderOrigin:    "claude",
						CompletionReason:  streamEvent.Subtype,
						Terminal:          true,
						ProviderEventType: "result",
						RawEvidenceRef:    "stdout:result",
					})
					state.lastMessageEvent = terminalEvent
					events = append(events, terminalEvent)
				}
			}
		}
		resultEvents, err := parseClaudeResultEvent(runID, &streamEvent, state.model, state.billing)
		if err != nil || len(resultEvents) == 0 {
			return nil, err
		}
		events = append(events, resultEvents...)
		return events, nil

	case "system":
		var sysResult string
		if streamEvent.Result != nil {
			_ = json.Unmarshal(streamEvent.Result, &sysResult)
		}
		if strings.Contains(strings.ToLower(streamEvent.Subtype), "auto-compact") || isAutoCompactMarker(sysResult) {
			return []*domain.RunEvent{newClaudeCompactionEvent(state, runID, strings.TrimSpace(sysResult), "auto", "", "")}, nil
		}
		// api_retry is informational — only the final `result` with
		// is_error=true determines run outcome, so we never emit a
		// RateLimitEvent from here.
		if streamEvent.Subtype == "api_retry" {
			return []*domain.RunEvent{domain.NewLogEvent(
				runID,
				"warn",
				fmt.Sprintf("Claude CLI auto-retry: HTTP %d, attempt %d/%d, next in %dms",
					streamEvent.ErrorStatus, streamEvent.Attempt, streamEvent.MaxRetries, streamEvent.RetryDelayMs),
			)}, nil
		}
		return []*domain.RunEvent{domain.NewLogEvent(
			runID, "debug", "System context received",
		)}, nil

	case "content_block_start":
		if streamEvent.ContentBlock != nil && streamEvent.ContentBlock.Type == "tool_use" {
			state.toolUseActive = true
			state.toolUseID = streamEvent.ContentBlock.ID
			state.toolUseName = streamEvent.ContentBlock.Name
			state.toolUsePayload.Reset()
			return nil, nil
		}

	case "content_block_delta":
		if streamEvent.Delta != nil {
			switch streamEvent.Delta.Type {
			case "text_delta":
				if streamEvent.Delta.Text != "" {
					state.textBuffer.WriteString(streamEvent.Delta.Text)
				}
				return nil, nil
			case "input_json_delta":
				if state.toolUseActive && streamEvent.Delta.PartialJSON != "" {
					state.toolUsePayload.WriteString(streamEvent.Delta.PartialJSON)
				}
				return nil, nil
			}
		}
		return nil, nil

	case "content_block_stop":
		if state.toolUseActive {
			toolEvent := toolCallFromState(runID, state)
			resetToolUseState(state)
			if toolEvent != nil {
				return []*domain.RunEvent{toolEvent}, nil
			}
		}
		return nil, nil

	case "message_start":
		return nil, nil

	case "message_delta":
		if streamEvent.Delta != nil && streamEvent.Delta.Text != "" {
			state.textBuffer.WriteString(streamEvent.Delta.Text)
			return nil, nil
		}
		return []*domain.RunEvent{domain.NewLogEvent(
			runID, "debug", "message_delta received without text payload",
		)}, nil

	case "message_stop":
		events := flushStreamMessage(runID, state)
		if state.toolUseActive {
			toolEvent := toolCallFromState(runID, state)
			resetToolUseState(state)
			if toolEvent != nil {
				events = append(events, toolEvent)
			}
		}
		if len(events) > 0 {
			return events, nil
		}
		return nil, nil

	case "ai-title", "summary":
		// Metadata records are consumed by ParseTranscriptLine (ai-title)
		// or intentionally ignored (summary); neither is agent output.
		return nil, nil

	case "init", "start", "ping", "heartbeat":
		return nil, nil

	// Interactive-only record types written to claude's on-disk session
	// transcript (never present in the --print stdout stream). They carry
	// UI/session metadata, not agent output, so they are dropped silently
	// rather than logged as "unhandled" debug noise (design §3). Listing
	// them here is safe for the stdout path — those runs never emit them.
	case "mode", "permission-mode", "last-prompt",
		"attachment", "file-history-snapshot", "queue-operation",
		"frame-link":
		return nil, nil

	case "":
		return nil, nil
	}

	if streamEvent.Type != "" {
		return []*domain.RunEvent{domain.NewLogEvent(
			runID, "debug",
			fmt.Sprintf("Unhandled event type: %s", streamEvent.Type),
		)}, nil
	}
	return nil, nil
}

func parseMessageEvent(state *claudeState, runID uuid.UUID, ev *ClaudeStreamEvent) []*domain.RunEvent {
	if ev.Message == nil {
		return nil
	}
	var events []*domain.RunEvent

	textContent := ev.Message.ExtractTextContent()
	if textContent != "" {
		if ev.Message.Role == "user" {
			if isCompact, focus := parseCompactCommand(textContent); isCompact {
				state.pendingCompact = true
				state.compactCommand = textContent
				state.compactFocus = focus
				return nil
			}
			// User text suppressed — orchestrator already creates message
			// events for the prompt and follow-ups. Subagent Agent-tool
			// echoes go through this path too.
		} else {
			if state.pendingCompact && ev.Message.Role == "assistant" {
				if isCompactionSummary(textContent) {
					state.pendingCompact = false
					summary := extractSummaryContent(textContent)
					return []*domain.RunEvent{newClaudeCompactionEvent(state, runID, summary, "manual", state.compactFocus, state.compactCommand)}
				}
				state.pendingCompact = false
			}
			if ev.Message.Role == "assistant" {
				state.lastAssistant = textContent
			}
			messageEvent := newClaudeMessageEvent(runID, ev, textContent, false)
			if ev.Message.Role == "assistant" {
				state.lastMessageEvent = messageEvent
			}
			events = append(events, messageEvent)
			state.textBuffer.Reset()
		}
	}
	if ev.Message.Role == "assistant" {
		state.messagesSinceCompact++
	}
	if ev.Message.Role == "user" {
		// User prompts are suppressed from the live event stream, but they
		// still occupy a message slot in the context being compacted. The
		// /compact control message itself is the boundary marker, not content
		// being compacted.
		if textContent != "" {
			if isCompact, _ := parseCompactCommand(textContent); !isCompact {
				state.messagesSinceCompact++
			}
		} else if len(ev.Message.ExtractToolResults()) > 0 {
			state.messagesSinceCompact++
		}
	}

	if ev.Message.Role == "user" {
		toolResults := ev.Message.ExtractToolResults()
		for _, r := range toolResults {
			events = append(events, domain.NewToolResultEvent(
				runID, "", r.ToolUseID, r.Content, claudeToolResultError(r),
			))
		}
	}
	toolUses := ev.Message.ExtractToolUses()
	for _, tool := range toolUses {
		var input map[string]interface{}
		if tool.Input != nil {
			_ = json.Unmarshal(tool.Input, &input)
		}
		events = append(events, domain.NewToolCallEvent(runID, tool.Name, tool.ID, input))
	}
	return events
}

func parseAssistantEvent(state *claudeState, runID uuid.UUID, ev *ClaudeStreamEvent) []*domain.RunEvent {
	if ev.Message == nil {
		return []*domain.RunEvent{domain.NewLogEvent(runID, "debug", "Assistant turn started")}
	}
	var events []*domain.RunEvent
	textContent := ev.Message.ExtractTextContent()
	if textContent != "" {
		if state.pendingCompact && isCompactionSummary(textContent) {
			state.pendingCompact = false
			summary := extractSummaryContent(textContent)
			return []*domain.RunEvent{newClaudeCompactionEvent(state, runID, summary, "manual", state.compactFocus, state.compactCommand)}
		}
		if state.pendingCompact {
			state.pendingCompact = false
		}
		state.lastAssistant = textContent
		messageEvent := newClaudeMessageEvent(runID, ev, textContent, ev.Message.StopReason == "end_turn")
		state.lastMessageEvent = messageEvent
		events = append(events, messageEvent)
		state.textBuffer.Reset()
	}
	state.messagesSinceCompact++
	toolUses := ev.Message.ExtractToolUses()
	for _, tool := range toolUses {
		var input map[string]interface{}
		if tool.Input != nil {
			_ = json.Unmarshal(tool.Input, &input)
		}
		events = append(events, domain.NewToolCallEvent(runID, tool.Name, tool.ID, input))
	}
	if ev.Message.Usage != nil {
		state.turn++
		if state.lastCompaction != nil {
			if data, ok := state.lastCompaction.Data.(*domain.CompactionEventData); ok && data.TokensAfter == 0 {
				data.TokensAfter = int64(usageInputTokens(ev.Message.Usage))
			}
			state.lastCompaction = nil
		}
		state.lastTurnInput = int64(usageInputTokens(ev.Message.Usage))
		events = append(events, newClaudeUsageEvent(runID, ev.Message.Usage, state.turn, state.model))
	}
	if len(events) > 0 {
		return events
	}
	return []*domain.RunEvent{domain.NewLogEvent(runID, "debug", "Assistant turn started")}
}

func newClaudeUsageEvent(runID uuid.UUID, usage *ClaudeUsage, turnIndex int, model string) *domain.RunEvent {
	return &domain.RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: domain.EventTypeMetric,
		Timestamp: time.Now(),
		Data: &domain.UsageEventData{
			PayloadKind:         domain.PayloadKindUsage,
			InputTokens:         usageInputTokens(usage),
			OutputTokens:        usageOutputTokens(usage),
			CacheCreationTokens: usageCacheCreation(usage),
			CacheReadTokens:     usageCacheRead(usage),
			TurnIndex:           turnIndex,
			RunnerType:          string(domain.RunnerTypeClaudeCode),
			Model:               model,
		},
	}
}

func newClaudeMessageEvent(runID uuid.UUID, ev *ClaudeStreamEvent, content string, terminal bool) *domain.RunEvent {
	messageID := ""
	role := "assistant"
	stopReason := ""
	if ev.Message != nil {
		messageID = ev.Message.ID
		role = ev.Message.Role
		stopReason = ev.Message.StopReason
	}
	conversationID := ev.SessionID
	if conversationID == "" {
		conversationID = ev.SessionIDAlt
	}
	return domain.NewProviderMessageEvent(runID, role, content, domain.MessageEventData{
		MessageID:      messageID,
		ConversationID: conversationID,
		// Claude's on-disk interactive transcript has no explicit turn_id.
		// The provider message id is its stable per-turn identity and lets
		// final-output selection distinguish an earlier end_turn from the
		// final handoff in the same warm session.
		TurnID:            messageID,
		ProviderOrigin:    "claude",
		CompletionReason:  stopReason,
		Terminal:          terminal,
		ParentMessageID:   ev.ParentToolUseID,
		ProviderEventType: ev.Type,
		RawEvidenceRef:    "claude:" + ev.Type,
	})
}

func parseUserEvent(state *claudeState, runID uuid.UUID, ev *ClaudeStreamEvent) []*domain.RunEvent {
	if ev.Message == nil {
		return []*domain.RunEvent{domain.NewLogEvent(runID, "debug", "User turn marker")}
	}
	var events []*domain.RunEvent
	toolResults := ev.Message.ExtractToolResults()
	for _, r := range toolResults {
		events = append(events, domain.NewToolResultEvent(
			runID, "", r.ToolUseID, r.Content, claudeToolResultError(r),
		))
	}
	textContent := ev.Message.ExtractTextContent()
	if textContent != "" {
		if isCompact, focus := parseCompactCommand(textContent); isCompact {
			state.pendingCompact = true
			state.compactCommand = textContent
			state.compactFocus = focus
		}
		if state.retainUser {
			events = append(events, domain.NewProviderMessageEvent(runID, "user", runner.StripANSI(textContent), domain.MessageEventData{ProviderOrigin: "claude", ProviderEventType: ev.Type, RawEvidenceRef: "claude:" + ev.Type}))
		}
	}
	if len(events) > 0 {
		return events
	}
	return []*domain.RunEvent{domain.NewLogEvent(runID, "debug", "User turn marker")}
}

func claudeToolResultError(result ClaudeContentItem) error {
	if !result.IsError {
		return nil
	}
	msg := strings.TrimSpace(result.Content)
	if msg == "" {
		msg = "tool result reported is_error=true"
	}
	return errors.New(msg)
}

func flushStreamMessage(runID uuid.UUID, state *claudeState) []*domain.RunEvent {
	if state == nil || state.textBuffer.Len() == 0 {
		return nil
	}
	message := runner.StripANSI(state.textBuffer.String())
	state.textBuffer.Reset()
	state.lastAssistant = message
	return []*domain.RunEvent{domain.NewProviderMessageEvent(runID, "assistant", message, domain.MessageEventData{
		ProviderOrigin:    "claude",
		ProviderEventType: "content_block_delta",
		RawEvidenceRef:    "claude:content_block_delta",
	})}
}

func toolCallFromState(runID uuid.UUID, state *claudeState) *domain.RunEvent {
	if state == nil || !state.toolUseActive {
		return nil
	}
	raw := strings.TrimSpace(state.toolUsePayload.String())
	var input map[string]interface{}
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &input); err != nil {
			input = map[string]interface{}{"raw": raw}
		}
	}
	return domain.NewToolCallEvent(runID, state.toolUseName, state.toolUseID, input)
}

func resetToolUseState(state *claudeState) {
	state.toolUseActive = false
	state.toolUseID = ""
	state.toolUseName = ""
	state.toolUsePayload.Reset()
}

// =============================================================================
// Result event handling
// =============================================================================

// parseClaudeResultEvent handles the terminal `result` event. Errors are
// classified via detectClaudeRateLimit; successful results emit a cost
// event when usage data is present.
func parseClaudeResultEvent(runID uuid.UUID, event *ClaudeStreamEvent, model string, billing domain.BillingSnapshot) ([]*domain.RunEvent, error) {
	resultStr := decodeClaudeResultString(event.Result)

	// Rate-limit classification only fires when the CLI itself flagged
	// is_error: true. The result text on a successful run is the agent's
	// final assistant message, which can legitimately mention rate limits
	// (e.g. discussing rate limiting); scanning it would produce false
	// positives. See https://code.claude.com/docs/en/errors.
	if event.IsError {
		if rl := detectClaudeRateLimit(resultStr); rl.Detected {
			return []*domain.RunEvent{domain.NewRateLimitEvent(
				runID, rl.LimitType, rl.Message, rl.ResetTime, rl.RetryAfter,
			)}, nil
		}
		msg := formatErrorMessage(event.Subtype, event.NumTurns, event.DurationMs, resultStr)
		errEvent := domain.NewErrorEvent(runID, "execution_error", msg, false)
		if data, ok := errEvent.Data.(*domain.ErrorEventData); ok {
			data.Details = buildErrorDetails(event.Subtype, event.NumTurns, event.DurationMs, event.SessionID, resultStr, "")
		}
		return []*domain.RunEvent{errEvent}, nil
	}

	// Successful result with usage — emit a cost event.
	if event.Usage != nil || event.TotalCostUSD > 0 {
		usageEvent := &domain.RunEvent{
			ID:        uuid.New(),
			RunID:     runID,
			EventType: domain.EventTypeMetric,
			Timestamp: time.Now(),
			Data: &domain.UsageEventData{
				PayloadKind:             domain.PayloadKindUsage,
				InputTokens:             usageInputTokens(event.Usage),
				OutputTokens:            usageOutputTokens(event.Usage),
				CacheCreationTokens:     usageCacheCreation(event.Usage),
				CacheReadTokens:         usageCacheRead(event.Usage),
				ServiceTier:             event.ServiceTier,
				RunnerType:              string(domain.RunnerTypeClaudeCode),
				Model:                   model,
				ReconciliationAuthority: true,
				TurnIndex:               event.NumTurns,
			},
		}
		chargeEvent := nativeChargeEvent(runID, domain.RunnerTypeClaudeCode, model, billing, event.TotalCostUSD)
		events := []*domain.RunEvent{usageEvent, chargeEvent}
		if event.Usage != nil && event.Usage.ServerToolUse != nil {
			if data, ok := usageEvent.Data.(*domain.UsageEventData); ok {
				data.WebSearchRequests = event.Usage.ServerToolUse.WebSearchRequests
			}
		}
		return events, nil
	}

	return []*domain.RunEvent{domain.NewLogEvent(
		runID, "info",
		fmt.Sprintf("Execution completed in %d turns", event.NumTurns),
	)}, nil
}

func usageInputTokens(u *ClaudeUsage) int {
	if u == nil {
		return 0
	}
	return u.InputTokens
}

func usageOutputTokens(u *ClaudeUsage) int {
	if u == nil {
		return 0
	}
	return u.OutputTokens
}

func usageCacheCreation(u *ClaudeUsage) int {
	if u == nil {
		return 0
	}
	return u.CacheCreationInputTokens
}

func usageCacheRead(u *ClaudeUsage) int {
	if u == nil {
		return 0
	}
	return u.CacheReadInputTokens
}

func decodeClaudeResultString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var result string
	if err := json.Unmarshal(raw, &result); err == nil {
		return result
	}
	return strings.TrimSpace(string(raw))
}

// =============================================================================
// Rate-limit detection
// =============================================================================

// rateLimitInfo is the parsed result of detectClaudeRateLimit.
type rateLimitInfo struct {
	Detected   bool
	LimitType  string
	ResetTime  *time.Time
	RetryAfter int
	Message    string
}

// detectClaudeRateLimit parses a Claude `result` payload. Callers MUST
// gate this on the CLI's is_error flag; feeding successful-run output in
// here will produce false positives when an agent's prose mentions rate
// limits.
//
// The size cap (Diagnostics.RateLimitMessageMaxLen) refuses
// classification on payloads much longer than the documented Claude Code
// banners (~100 chars per https://code.claude.com/docs/en/errors). Long
// payloads are almost certainly tool output that happens to contain a
// trigger word.
func detectClaudeRateLimit(resultStr string) rateLimitInfo {
	info := rateLimitInfo{Message: resultStr}

	if len(resultStr) > config.DefaultLevers().Diagnostics.RateLimitMessageMaxLen {
		return info
	}

	lowerMsg := strings.ToLower(resultStr)

	// Anchored phrases per https://code.claude.com/docs/en/errors. Bare
	// "rate limit" is NOT matched because it appears in ordinary prose.
	matched := strings.Contains(lowerMsg, "usage limit reached") ||
		strings.Contains(lowerMsg, "rate limit reached") ||
		strings.Contains(lowerMsg, "rate limit exceeded") ||
		strings.Contains(lowerMsg, "request rejected (429)") ||
		strings.Contains(lowerMsg, "server is temporarily limiting requests") ||
		(strings.Contains(lowerMsg, "hit your") && strings.Contains(lowerMsg, "limit")) ||
		(strings.Contains(lowerMsg, "reached your") && strings.Contains(lowerMsg, "limit"))

	if !matched {
		return info
	}

	info.Detected = true
	info.LimitType = "5_hour" // most common

	// Reset timestamp from "limit reached|1755806400" form.
	parts := strings.Split(resultStr, "|")
	if len(parts) >= 2 {
		if ts, err := strconv.ParseInt(strings.TrimSpace(parts[len(parts)-1]), 10, 64); err == nil {
			resetTime := time.Unix(ts, 0)
			info.ResetTime = &resetTime
			info.RetryAfter = int(time.Until(resetTime).Seconds())
			if info.RetryAfter < 0 {
				info.RetryAfter = 0
			}
		}
	}

	switch {
	case strings.Contains(lowerMsg, "daily"):
		info.LimitType = "daily"
	case strings.Contains(lowerMsg, "weekly"):
		info.LimitType = "weekly"
	case strings.Contains(lowerMsg, "token"):
		info.LimitType = "token"
	}

	return info
}

// =============================================================================
// Compaction helpers
// =============================================================================

func newClaudeCompactionEvent(state *claudeState, runID uuid.UUID, summary, trigger, focus, originalCommand string) *domain.RunEvent {
	event := domain.NewCompactionEvent(
		runID, summary, trigger, focus,
		state.messagesSinceCompact, state.lastTurnInput, 0, originalCommand,
	)
	state.messagesSinceCompact = 0
	state.lastCompaction = event
	return event
}

// parseCompactCommand extracts focus from "/compact focus on auth" → "auth".
func parseCompactCommand(content string) (isCompact bool, focus string) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "/compact") {
		return false, ""
	}
	rest := strings.TrimPrefix(content, "/compact")
	if rest != "" && rest[0] != ' ' && rest[0] != '\t' && rest[0] != '\n' {
		return false, ""
	}
	remainder := strings.TrimSpace(rest)
	if strings.HasPrefix(remainder, "focus on ") {
		focus = strings.TrimPrefix(remainder, "focus on ")
	} else if remainder != "" {
		focus = remainder
	}
	return true, strings.TrimSpace(focus)
}

func isCompactionSummary(content string) bool {
	return strings.Contains(content, "<summary>") ||
		strings.HasPrefix(strings.TrimSpace(content), "Summary of")
}

func extractSummaryContent(content string) string {
	start := strings.Index(content, "<summary>")
	end := strings.Index(content, "</summary>")
	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(content[start+len("<summary>") : end])
	}
	return content
}

// isAutoCompactMarker recognises the log-style strings Claude Code emits
// around an automatic (non-user-triggered) compaction.
func isAutoCompactMarker(content string) bool {
	c := strings.ToLower(strings.TrimSpace(content))
	if c == "" {
		return false
	}
	markers := []string{
		"auto-compacting",
		"auto compacting",
		"conversation history has been compacted",
		"context has been compacted",
		"automatic compaction",
	}
	for _, m := range markers {
		if strings.Contains(c, m) {
			return true
		}
	}
	return false
}

// =============================================================================
// Diagnostic helpers
// =============================================================================

// redactSecrets replaces credential-shaped substrings with "<redacted>".
func redactSecrets(s string) string {
	for _, re := range secretRedactors {
		s = re.ReplaceAllString(s, "<redacted>")
	}
	return s
}

// tailBytesUTF8Safe returns the last max bytes of s, rewound to the
// nearest UTF-8 rune boundary so callers never see half a multi-byte
// character. Returns s unchanged if it already fits.
func tailBytesUTF8Safe(s string, max int) string {
	if len(s) <= max {
		return s
	}
	start := len(s) - max
	for start < len(s) && !utf8.RuneStart(s[start]) {
		start++
	}
	return s[start:]
}

// buildErrorDetails collects the structured context derivable from a
// Claude `result` event when is_error: true.
func buildErrorDetails(subtype string, numTurns int, durationMs int, sessionID, resultText, stderrTail string) map[string]interface{} {
	details := map[string]interface{}{
		"subtype":     subtype,
		"num_turns":   numTurns,
		"duration_ms": durationMs,
		"result_text": resultText,
	}
	if sessionID != "" {
		details["session_id"] = sessionID
	}
	if stderrTail != "" {
		details["stderr_tail"] = stderrTail
	}
	return details
}

// formatErrorMessage builds a human-readable summary for an
// execution_error event. Always produces a non-empty string.
func formatErrorMessage(subtype string, numTurns int, durationMs int, resultText string) string {
	summary := "claude-code terminated with is_error=true"
	parts := []string{}
	if subtype != "" {
		parts = append(parts, "subtype="+subtype)
	}
	parts = append(parts, "turns="+strconv.Itoa(numTurns), "duration_ms="+strconv.Itoa(durationMs))
	summary += " (" + strings.Join(parts, ", ") + ")"
	if strings.TrimSpace(resultText) != "" {
		summary += ": " + strings.TrimSpace(resultText)
	}
	return summary
}
