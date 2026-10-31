// Package codecs — codex_rollout.go maps Codex's on-disk rollout dialect
// ($CODEX_HOME/sessions/YYYY/MM/DD/rollout-*.jsonl) to domain events. It is the
// one parser for every Codex transcript Agent Manager ingests: interactive
// tails, the Transcript Indexer's imports and recovery replays.
package codecs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// =============================================================================
// On-disk rollout dialect (interactive runs)
// =============================================================================

// codexTranscriptVersion names the events this codec derives from a rollout.
// Bump it when the same lines start producing different events; the
// Transcript Indexer then re-imports the files an older version read.
//   - 1: item_completed CommandExecution yields a result-only
//     command_execution event per command an exec call ran (P4-W1).
//   - 2: a top-level `compacted` record yields a compaction event.
//   - 3: a forked rollout's copy of its parent's history yields nothing
//     (P4-W5k): the fork's events start at its own first line.
const codexTranscriptVersion = 3

// TranscriptVersion satisfies [runner.TranscriptVersioner].
func (p *codexTranscriptParser) TranscriptVersion() int { return codexTranscriptVersion }

// codexRolloutLine is the outer envelope of a codex on-disk rollout record
// ($CODEX_HOME/sessions/YYYY/MM/DD/rollout-*.jsonl). Every line is one of a
// small set of wrapper types carrying a typed payload.
type codexRolloutLine struct {
	Type    string              `json:"type"`
	Payload codexRolloutPayload `json:"payload"`
}

// codexRolloutPayload is the union of the payload fields this codec maps.
// Unused fields for a given payload type stay zero.
type codexRolloutPayload struct {
	Type string `json:"type"`
	// session_meta. Source is "vscode"/"exec"/... or an object such as
	// {"subagent":{"thread_spawn":{...}}}.
	ID           string          `json:"id"`
	ForkedFromID string          `json:"forked_from_id"`
	ThreadSource string          `json:"thread_source"`
	Source       json.RawMessage `json:"source"`
	// response_item message
	Role     string                `json:"role"`
	Content  json.RawMessage       `json:"content"`
	Metadata *codexMessageMetadata `json:"internal_chat_message_metadata_passthrough"`
	// agent_message / user_message
	Message string `json:"message"`
	// task_started/task_complete/turn_context
	TurnID string `json:"turn_id"`
	// event_msg records of a thread's own activity (thread_settings_applied,
	// item_completed)
	ThreadID string `json:"thread_id"`
	// function_call / custom_tool_call
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // function_call: JSON-encoded args
	Input     string `json:"input"`     // custom_tool_call: raw tool input
	CallID    string `json:"call_id"`
	// function_call_output / custom_tool_call_output
	Output json.RawMessage `json:"output"`
	// event_msg item_completed; decoded only for a CommandExecution, since
	// other items' fields (array outputs) do not fit CodexItem.
	Item json.RawMessage `json:"item"`
	// turn_context
	Model string `json:"model"`
	// turn_aborted
	Reason string               `json:"reason"`
	Info   *codexTokenCountInfo `json:"info"`
	// Native quota metadata is a sibling of info in token_count frames. It is
	// present even when info is null (for example at session start).
	RateLimits codexRateLimits `json:"rate_limits"`
}

// codexRateLimits keeps the window-shaped entries of a token_count frame's
// rate_limits and skips the rest. Newer Codex CLIs add scalar siblings such as
// limit_id, plan_type and credits; a strict map type would fail the whole frame
// and silently drop its usage.
type codexRateLimits map[string]CodexRateLimitWindow

func (m *codexRateLimits) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	out := codexRateLimits{}
	for key, value := range raw {
		trimmed := bytes.TrimSpace(value)
		if len(trimmed) == 0 || trimmed[0] != '{' {
			continue
		}
		var window CodexRateLimitWindow
		if json.Unmarshal(trimmed, &window) == nil && (window.UsedPercent != nil || window.WindowMinutes != 0) {
			out[key] = window
		}
	}
	*m = out
	return nil
}

// codexMessageMetadata is the structural evidence Codex records on a message:
// the kind of each content item and, from about 2026-10-29, who provided it.
type codexMessageMetadata struct {
	ContentItemKinds    []string `json:"content_item_kinds"`
	ContentItemMetadata []struct {
		Provenance struct {
			Type string `json:"type"`
		} `json:"provenance"`
	} `json:"content_item_metadata"`
}

// sourceEvidence returns the item kinds and provenance types; both are empty
// when the rollout predates the metadata.
func (m *codexMessageMetadata) sourceEvidence() (kinds, provenance []string) {
	if m == nil {
		return nil, nil
	}
	for _, item := range m.ContentItemMetadata {
		if item.Provenance.Type != "" {
			provenance = append(provenance, item.Provenance.Type)
		}
	}
	return m.ContentItemKinds, provenance
}

// isCodexSubagentThread reports whether session_meta names a sub-agent
// thread: thread_source "subagent", or a source object with a subagent key
// (spawned workers and guardian reviews alike).
func isCodexSubagentThread(threadSource string, source json.RawMessage) bool {
	if strings.EqualFold(strings.TrimSpace(threadSource), "subagent") {
		return true
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(source, &object) != nil {
		return false
	}
	_, ok := object["subagent"]
	return ok
}

type codexTokenCountInfo struct {
	LastTokenUsage  *CodexUsage `json:"last_token_usage"`
	TotalTokenUsage *CodexUsage `json:"total_token_usage"`
}

// CodexRateLimitWindow is the bounded, metadata-only native quota shape. The
// CLI reports a percentage/window/reset rather than an absolute token quota;
// keeping these as optional fields avoids inventing a ceiling.
type CodexRateLimitWindow struct {
	UsedPercent   *float64 `json:"used_percent,omitempty"`
	WindowMinutes int64    `json:"window_minutes,omitempty"`
	ResetAt       int64    `json:"reset_at,omitempty"`
	ResetsAt      int64    `json:"resets_at,omitempty"`
}

const codexProvider = "openai"

// rolloutWrapperTypes are the outer `type` values that mark a line as the
// on-disk rollout dialect rather than the flat exec-json stdout dialect.
var rolloutWrapperTypes = map[string]bool{
	"event_msg":     true,
	"response_item": true,
	"session_meta":  true,
	"compacted":     true,
	"turn_context":  true,
}

// parseRolloutLine maps one on-disk rollout record to the same domain events
// the exec-json decoder emits. ok=false means the line is not a rollout record
// (blank, non-JSON, or a flat exec-json line) and the caller should fall back
// to the stdout decoder. The mapping mirrors parseCodexEvents:
//   - event_msg.agent_message           → assistant MessageEvent
//   - event_msg.user_message            → suppressed (orchestrator owns the prompt)
//   - response_item.function_call       → ToolCallEvent
//   - response_item.custom_tool_call    → ToolCallEvent
//   - response_item.*_output            → ToolResultEvent
//   - event_msg.task_complete/turn_completed → terminal success
//   - event_msg.turn_aborted            → terminal (not success)
//   - event_msg.error                   → error event + terminal failure
//   - session_meta.id                   → SessionID (thread id)
func (p *codexTranscriptParser) parseRolloutLine(runID uuid.UUID, line string) (runner.TranscriptParseResult, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed[0] != '{' {
		return runner.TranscriptParseResult{}, false
	}
	// Recognize the dialect before decoding variable payload content. A known
	// wrapper with unsupported content must never fall into flat stdout parsing.
	var envelope struct {
		Type string `json:"type"`
	}
	if json.Unmarshal([]byte(trimmed), &envelope) != nil || !rolloutWrapperTypes[envelope.Type] {
		return runner.TranscriptParseResult{}, false
	}
	var rl codexRolloutLine
	if err := json.Unmarshal([]byte(trimmed), &rl); err != nil {
		warning := domain.NewLogEvent(runID, "warning", "Unsupported Codex rollout payload: "+err.Error())
		return runner.TranscriptParseResult{Events: []*domain.RunEvent{warning}, Timestamp: transcriptLineTimestamp(line)}, true
	}

	result := runner.TranscriptParseResult{Timestamp: transcriptLineTimestamp(line)}
	pl := rl.Payload
	// A forked rollout opens with a copy of its parent's history (the
	// parent's session_meta, compacted record, turns, calls and terminals),
	// re-stamped at the fork. That history is the parent's run, not this one:
	// it yields nothing until the fork's own first line, the first that names
	// this thread (thread_settings_applied) or a turn begun after the fork
	// (turn and thread ids are time-ordered UUIDv7s).
	if p.state.forkPrefix {
		if pl.ThreadID != p.state.threadID && (pl.TurnID == "" || pl.TurnID <= p.state.threadID) {
			return result, true
		}
		p.state.forkPrefix = false
	}
	if pl.TurnID != "" {
		p.state.rolloutTurnID = pl.TurnID
	}

	switch rl.Type {
	case "session_meta":
		p.state.subagentThread = p.state.subagentThread || isCodexSubagentThread(pl.ThreadSource, pl.Source)
		if pl.ID != "" {
			p.state.observeSession(pl.ID)
			p.state.sessionFromStart = true
			p.state.forkPrefix = pl.ForkedFromID != ""
			result.SessionID = pl.ID
		}
		return result, true

	case "turn_context":
		// The turn's own model is the truth; a configured model may be absent
		// (imported sessions) or overridden in the session.
		if model := strings.TrimSpace(pl.Model); model != "" {
			p.state.runModel = model
		}
		p.state.turn++
		result.ProviderTurnStarted = true
		return result, true

	case "response_item":
		if pl.Type == "message" {
			role := strings.ToLower(strings.TrimSpace(pl.Role))
			result.UserTurnStarted = role == "user" || role == "operator"
		}
		result.Events = parseCodexRolloutItem(runID, pl, p.state)
		return result, true

	case "compacted":
		event := domain.NewCompactionEvent(runID, pl.Message, "auto", "", 0, 0, 0, "")
		if !result.Timestamp.IsZero() {
			event.Timestamp = result.Timestamp
		}
		result.Events = []*domain.RunEvent{event}
		return result, true

	case "event_msg":
		switch pl.Type {
		case "task_started":
			p.state.turnUsageObserved, p.state.turnAmount = false, domain.UsageCounters{}
			if p.state.turn == 0 {
				p.state.turn = 1
			}
		case "token_count":
			if pl.Info != nil && (pl.Info.TotalTokenUsage != nil || pl.Info.LastTokenUsage != nil) {
				if p.state.turn == 0 {
					p.state.turn = 1
				}
				// Quota refreshes repeat the previous frame's counters; only a
				// changed counter is a new usage report.
				if usage, fresh := p.state.usageReport(runID, p.codec.pricingService, pl.Info.TotalTokenUsage, pl.Info.LastTokenUsage); fresh {
					// The report happened at the line's time, not at parse time:
					// an indexer back-fill reads it days later.
					for _, event := range usage {
						if !result.Timestamp.IsZero() {
							event.Timestamp = result.Timestamp
						}
					}
					result.Events = usage
					p.state.turnUsageObserved = true
					p.state.turnAmount = counterSum(p.state.turnAmount, usage[0].Data.(*domain.UsageEventData).Counters())
				}
			}
			result.Events = append(result.Events, codexQuotaEvents(runID, result.Timestamp, pl.RateLimits)...)
		case "agent_message":
			if text := runner.StripANSI(strings.TrimSpace(pl.Message)); text != "" {
				p.state.lastMessage = text
				p.state.lastMessageID = pl.ID
				messageEvent := newCodexMessageEvent(runID, p.state, text, pl.ID, "event_msg.agent_message", false, "")
				p.state.lastMessageEvent = messageEvent
				result.Events = []*domain.RunEvent{messageEvent}
			}
		case "user_message":
			result.UserTurnStarted = true
			if p.state.retainUser && strings.TrimSpace(pl.Message) != "" {
				text := runner.StripANSI(pl.Message)
				source := domain.ClassifyCodexUserMessage(text, p.state.subagentThread, nil, nil)
				result.Events = []*domain.RunEvent{domain.NewProviderMessageEvent(runID, "user", text, domain.MessageEventData{ProviderOrigin: "codex", ProviderEventType: "event_msg.user_message", RawEvidenceRef: "codex:event_msg.user_message", Source: source})}
			}
		case "task_complete", "turn_completed":
			result.Events = append(result.Events, p.rolloutTerminalReceipt(runID, result.Timestamp)...)
			if p.state.lastMessageEvent != nil {
				markProviderMessageTerminal(p.state.lastMessageEvent, pl.Type, "event_msg."+pl.Type, "codex:event_msg."+pl.Type)
				result.Events = append(result.Events, newProviderTerminalEvidence(runID, p.state.lastMessageEvent, pl.Type, "event_msg."+pl.Type, "codex:event_msg."+pl.Type))
			}
			result.Terminal = &runner.TranscriptTerminal{Success: true, ExitCode: 0}
		case "turn_aborted":
			result.Events = append(result.Events, p.rolloutTerminalReceipt(runID, result.Timestamp)...)
			msg := "turn aborted"
			if pl.Reason != "" {
				msg = "turn aborted: " + pl.Reason
			}
			result.Terminal = &runner.TranscriptTerminal{Success: false, ExitCode: 1, ErrorMessage: msg}
		case "error":
			if msg := runner.StripANSI(strings.TrimSpace(pl.Message)); msg != "" {
				result.Events = []*domain.RunEvent{domain.NewErrorEvent(runID, "execution_error", msg, false)}
				result.Terminal = &runner.TranscriptTerminal{Success: false, ExitCode: 1, ErrorMessage: msg}
			}
		case "item_completed":
			// Other completed items repeat response_item records; only a
			// CommandExecution carries what nothing else does: the outcome of
			// one command an exec call ran.
			var item CodexItem
			if bytes.Contains(pl.Item, []byte(`"CommandExecution"`)) && json.Unmarshal(pl.Item, &item) == nil && item.Type == "CommandExecution" {
				result.Events = parseCodexCommandExecution(runID, &item)
				for _, event := range result.Events {
					if !result.Timestamp.IsZero() {
						event.Timestamp = result.Timestamp
					}
				}
			}
		}
		return result, true
	}
	return result, true
}

// codexQuotaEvents converts only the native rate-limit metadata into typed
// observations. A Codex token_count frame has no absolute quota ceiling, so
// this intentionally emits percentage/window/reset metadata only.
func codexQuotaEvents(runID uuid.UUID, observedAt time.Time, rateLimits map[string]CodexRateLimitWindow) []*domain.RunEvent {
	if len(rateLimits) == 0 {
		return nil
	}
	capacity := len(rateLimits)
	if capacity > 64 {
		capacity = 64
	}
	result := make([]*domain.RunEvent, 0, capacity)
	for pool, limit := range rateLimits {
		if len(result) >= 64 {
			break
		}
		pool = strings.TrimSpace(pool)
		if pool == "" || len(pool) > 64 || limit.UsedPercent == nil || *limit.UsedPercent < 0 || *limit.UsedPercent > 100 || limit.WindowMinutes <= 0 {
			continue
		}
		resetEpoch := limit.ResetAt
		if resetEpoch == 0 {
			resetEpoch = limit.ResetsAt
		}
		var reset *time.Time
		if resetEpoch > 0 {
			value := time.Unix(resetEpoch, 0).UTC()
			reset = &value
		}
		event := domain.NewRateLimitEvent(runID, fmt.Sprintf("%dm", limit.WindowMinutes), "Codex provider quota observation", reset, 0)
		event.Timestamp = observedAt
		data, ok := event.Data.(*domain.RateLimitEventData)
		if !ok {
			continue
		}
		percent := *limit.UsedPercent
		data.Provider = codexProvider
		data.Pool = pool
		data.UsedPercent = &percent
		data.WindowMinutes = limit.WindowMinutes
		data.Provenance = "codex:event_msg.token_count.rate_limits"
		result = append(result, event)
	}
	return result
}

// Both explicit successful and aborted turn boundaries close provider usage.
// The receipt carries the turn's summed amount and the session's latest
// cumulative counter, so it stands alone when it is the only retained report
// (transcript recovery) and adds nothing when the turn's reports are retained:
// the accounting credits each counter value once. Missing token evidence
// remains unknown; a terminal marker alone cannot create zero.
func (p *codexTranscriptParser) rolloutTerminalReceipt(runID uuid.UUID, at time.Time) []*domain.RunEvent {
	if !p.state.turnUsageObserved || p.state.lastCumulative == nil {
		return nil
	}
	cumulative := *p.state.lastCumulative
	events := markUsageTurn(buildCostEvents(runID, domain.RunnerTypeCodex, p.codec.pricingService, p.state.runModel, usageTokens(p.state.turnAmount), p.state.billing), p.state.turn)
	usage := events[0].Data.(*domain.UsageEventData)
	usage.ReconciliationAuthority, usage.Turns = true, 1
	usage.SessionID, usage.Cumulative = p.state.threadID, &cumulative
	events[0].Timestamp = at
	return attachCharge(events)
}

// attachCharge moves a receipt's charge onto its usage event, so the charge
// has one representation: an authoritative receipt carries it and no separate
// charge event repeats it (F-W1-2). events[0] is the usage event.
func attachCharge(events []*domain.RunEvent) []*domain.RunEvent {
	usage, ok := events[0].Data.(*domain.UsageEventData)
	if !ok {
		return events
	}
	out := events[:1]
	for _, event := range events[1:] {
		if charge, isCharge := event.Data.(*domain.ChargeEventData); isCharge {
			usage.Charge = charge
			continue
		}
		out = append(out, event)
	}
	return out
}

// closingReport repeats the session's latest counter with no amount, so a
// turn boundary can close the invocation without adding usage.
func (s *codexState) closingReport(runID uuid.UUID, pricing PricingService) []*domain.RunEvent {
	events := markUsageTurn(buildCostEvents(runID, domain.RunnerTypeCodex, pricing, s.runModel, usageTokens{}, s.billing), s.turn)
	if usage, ok := events[0].Data.(*domain.UsageEventData); ok && s.lastCumulative != nil {
		cumulative := *s.lastCumulative
		usage.SessionID, usage.Cumulative = s.threadID, &cumulative
	}
	return events
}

// usageReport turns one provider counter report into usage events carrying
// the raw session-cumulative counter and the provider's explicit amount for
// this report. The amount is last_token_usage when Codex supplies it, the
// difference from this parser's previous counter, the whole counter when the
// parser saw the session start, and otherwise zero (unknown) — never the
// counter of a session observed mid-way. fresh is false when the counter
// repeats the previous report. A report without a cumulative counter (older
// rollouts) carries only its amount.
func (s *codexState) usageReport(runID uuid.UUID, pricing PricingService, total, last *CodexUsage) ([]*domain.RunEvent, bool) {
	var amount domain.UsageCounters
	var cumulative *domain.UsageCounters
	switch {
	case total != nil:
		value := codexCounters(total)
		if s.lastCumulative != nil && *s.lastCumulative == value {
			return nil, false
		}
		switch {
		case last != nil:
			amount = codexCounters(last)
		case s.lastCumulative != nil:
			amount = counterDifference(value, *s.lastCumulative)
		case s.sessionFromStart:
			amount = value
		}
		s.lastCumulative, cumulative = &value, &value
	case last != nil:
		amount = codexCounters(last)
	default:
		return nil, false
	}
	events := markUsageTurn(buildCostEvents(runID, domain.RunnerTypeCodex, pricing, s.runModel, usageTokens(amount), s.billing), s.turn)
	if usage, ok := events[0].Data.(*domain.UsageEventData); ok {
		usage.SessionID, usage.Cumulative = s.threadID, cumulative
	}
	return events, true
}

// codexCounters splits Codex's usage into disjoint components. Codex input
// includes cached reads (and cache writes, when reported), so those subsets
// are removed from input before persistence and consumed exactly once.
func codexCounters(u *CodexUsage) domain.UsageCounters {
	input := u.InputTokens
	if input >= u.CachedInputTokens+u.CacheWriteInputTokens {
		input -= u.CachedInputTokens + u.CacheWriteInputTokens
	}
	return domain.UsageCounters{InputTokens: input, OutputTokens: u.OutputTokens, CacheReadTokens: u.CachedInputTokens, CacheCreationTokens: u.CacheWriteInputTokens}
}

func counterSum(a, b domain.UsageCounters) domain.UsageCounters {
	return domain.UsageCounters{InputTokens: a.InputTokens + b.InputTokens, OutputTokens: a.OutputTokens + b.OutputTokens, CacheReadTokens: a.CacheReadTokens + b.CacheReadTokens, CacheCreationTokens: a.CacheCreationTokens + b.CacheCreationTokens}
}

func counterDifference(current, previous domain.UsageCounters) domain.UsageCounters {
	return domain.UsageCounters{
		InputTokens:         max(0, current.InputTokens-previous.InputTokens),
		OutputTokens:        max(0, current.OutputTokens-previous.OutputTokens),
		CacheReadTokens:     max(0, current.CacheReadTokens-previous.CacheReadTokens),
		CacheCreationTokens: max(0, current.CacheCreationTokens-previous.CacheCreationTokens),
	}
}

// parseCodexRolloutItem maps a response_item payload to tool events. Both the
// generic function_call/output and MCP-style custom_tool_call/output shapes
// carry a call_id, a name (on the call side), and a string output.
func parseCodexRolloutItem(runID uuid.UUID, pl codexRolloutPayload, state *codexState) []*domain.RunEvent {
	if pl.Type == "message" {
		text := codexRolloutMessageText(pl.Content)
		if text == "" {
			return nil
		}
		text = runner.StripANSI(strings.TrimSpace(text))
		switch strings.ToLower(strings.TrimSpace(pl.Role)) {
		case "assistant":
			message := newCodexMessageEvent(runID, state, text, pl.ID, "response_item.message", false, "")
			state.lastMessage = text
			state.lastMessageID = pl.ID
			state.lastMessageEvent = message
			return []*domain.RunEvent{message}
		case "user", "operator":
			if state != nil && state.retainUser {
				kinds, provenance := pl.Metadata.sourceEvidence()
				source := domain.ClassifyCodexUserMessage(text, state.subagentThread, kinds, provenance)
				return []*domain.RunEvent{domain.NewProviderMessageEvent(runID, "user", text, domain.MessageEventData{ProviderOrigin: "codex", ProviderEventType: "response_item.message", Source: source})}
			}
		}
		return nil
	}
	switch pl.Type {
	case "function_call":
		var input map[string]interface{}
		if pl.Arguments != "" {
			_ = json.Unmarshal([]byte(pl.Arguments), &input)
		}
		return []*domain.RunEvent{domain.NewToolCallEvent(runID, pl.Name, pl.CallID, input)}
	case "custom_tool_call":
		input := map[string]interface{}{}
		if pl.Input != "" {
			input["input"] = pl.Input
		}
		return []*domain.RunEvent{domain.NewToolCallEvent(runID, pl.Name, pl.CallID, input)}
	case "function_call_output", "custom_tool_call_output":
		return codexRolloutToolResult(runID, pl)
	}
	// message / reasoning and other response items carry no distinct
	// domain event here (assistant text arrives via event_msg.agent_message).
	return nil
}

// inlineImageData matches the base64 body of an inline image data URI.
var inlineImageData = regexp.MustCompile(`(data:[A-Za-z0-9.+/-]+;base64,)[A-Za-z0-9+/=]+`)

// stripInlineImageData drops inline image bytes from stored tool output: a
// screenshot is megabytes of base64 with no recall value, and the indexer
// counts images from the source transcript, not from stored events (R-1).
func stripInlineImageData(raw string) string {
	if !strings.Contains(raw, ";base64,") {
		return raw
	}
	return inlineImageData.ReplaceAllString(raw, "${1}<omitted>")
}

// codexRolloutToolResult preserves native strings and ordered typed text blocks.
// A returned tool payload establishes evidence, not a successful process exit.
func codexRolloutToolResult(runID uuid.UUID, pl codexRolloutPayload) []*domain.RunEvent {
	data := &domain.ToolResultEventData{ToolCallID: pl.CallID, RawOutputJSON: stripInlineImageData(string(pl.Output))}
	known := false
	data.OutcomeKnown = &known
	unsupported := false
	conflictingMetadata := false
	conflictingDuration := false
	var text string
	if json.Unmarshal(pl.Output, &text) == nil && len(pl.Output) > 0 && string(bytes.TrimSpace(pl.Output)) != "null" {
		data.Output = runner.StripANSI(text)
		data.ExitCode, data.DurationMS, unsupported = codexNestedResultMetadata(text)
	} else {
		var blocks []json.RawMessage
		if json.Unmarshal(pl.Output, &blocks) != nil || string(bytes.TrimSpace(pl.Output)) == "null" {
			unsupported = true
		} else {
			for _, raw := range blocks {
				var block struct {
					Type string  `json:"type"`
					Text *string `json:"text"`
				}
				output := domain.ToolOutputBlock{RawJSON: stripInlineImageData(string(raw))}
				if json.Unmarshal(raw, &block) == nil {
					output.Type = block.Type
					if (block.Type == "input_text" || block.Type == "output_text" || block.Type == "text") && block.Text != nil {
						output.Text = runner.StripANSI(*block.Text)
						// Typed metadata is parsed only from a complete text block, never
						// recursively extracted or synthesized from concatenated fragments.
						exit, duration, invalid := codexNestedResultMetadata(*block.Text)
						unsupported = unsupported || invalid
						if exit != nil {
							if data.ExitCode != nil && *data.ExitCode != *exit {
								conflictingMetadata = true
							}
							data.ExitCode = exit
						}
						if duration != nil {
							if data.DurationMS != nil && *data.DurationMS != *duration {
								conflictingDuration = true
							}
							data.DurationMS = duration
						}
					} else {
						unsupported = true
					}
				} else {
					unsupported = true
				}
				data.OutputBlocks = append(data.OutputBlocks, output)
			}
		}
	}
	if !domain.SupportedToolExitCode(data.ExitCode) {
		conflictingMetadata = true
	}
	if conflictingMetadata {
		data.ExitCode = nil
		unsupported = true
	}
	if conflictingDuration {
		data.DurationMS = nil
		unsupported = true
	}
	if data.ExitCode != nil {
		known = true
		data.Success = *data.ExitCode == 0
		if !data.Success {
			data.Error = fmt.Sprintf("command exited with code %d", *data.ExitCode)
		}
	}
	event := &domain.RunEvent{ID: uuid.New(), RunID: runID, EventType: domain.EventTypeToolResult, Timestamp: time.Now(), Data: data}
	events := []*domain.RunEvent{event}
	if unsupported {
		events = append(events, domain.NewLogEvent(runID, "warning", "Unsupported Codex tool output content retained for call "+pl.CallID))
	}
	return events
}

// codexRolloutMessageText extracts the text-bearing content items used by
// Codex response_item messages. Developer/system messages are filtered by the
// caller; content itself is intentionally reduced to text, never attachments
// or tool metadata.
func codexRolloutMessageText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return ""
	}
	var collect func(any)
	parts := []string{}
	collect = func(item any) {
		switch value := item.(type) {
		case string:
			if strings.TrimSpace(value) != "" {
				parts = append(parts, value)
			}
		case []any:
			for _, child := range value {
				collect(child)
			}
		case map[string]any:
			if text, ok := value["text"].(string); ok && strings.TrimSpace(text) != "" {
				parts = append(parts, text)
				return
			}
		}
	}
	collect(value)
	return strings.Join(parts, "\n")
}
