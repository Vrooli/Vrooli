// Responsibility: retain claude declarations within their original package.
package codecs

import (
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/fallback"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"strings"
)

// ClaudeStreamEvent represents a single event from Claude Code's
// stream-json output.
type ClaudeStreamEvent struct {
	Type      string          `json:"type"`
	Subtype   string          `json:"subtype,omitempty"`
	AiTitle   string          `json:"aiTitle,omitempty"`
	Message   *ClaudeMessage  `json:"message,omitempty"`
	Usage     *ClaudeUsage    `json:"usage,omitempty"`
	ToolUse   *ClaudeToolUse  `json:"tool_use,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *ClaudeError    `json:"error,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	// SessionIDAlt is the camelCase session id claude writes to its
	// on-disk interactive transcript (~/.claude/projects/.../<id>.jsonl).
	// The stdout stream-json dialect uses snake_case session_id; the
	// on-disk dialect uses sessionId. Capturing both lets the transcript
	// parser recover the session id from an interactive run (design §3).
	SessionIDAlt string              `json:"sessionId,omitempty"`
	IsError      bool                `json:"is_error,omitempty"`
	DurationMs   int                 `json:"duration_ms,omitempty"`
	DurationAPI  int                 `json:"duration_api_ms,omitempty"`
	NumTurns     int                 `json:"num_turns,omitempty"`
	TotalCostUSD float64             `json:"total_cost_usd,omitempty"`
	ServiceTier  string              `json:"service_tier,omitempty"`
	ContentBlock *ClaudeContentBlock `json:"content_block,omitempty"`
	Delta        *ClaudeDelta        `json:"delta,omitempty"`

	// system/api_retry payload (HTTP status + retry counters). The
	// "error" field on api_retry is a bare string ("rate_limit"); we let
	// ClaudeError.UnmarshalJSON absorb it.
	ErrorStatus     int    `json:"error_status,omitempty"`
	Attempt         int    `json:"attempt,omitempty"`
	MaxRetries      int    `json:"max_retries,omitempty"`
	RetryDelayMs    int    `json:"retry_delay_ms,omitempty"`
	ParentToolUseID string `json:"parent_tool_use_id,omitempty"`
}

// ClaudeMessage carries the role + content of a stream message. Content
// can be either a JSON string or an array of content blocks.
type ClaudeMessage struct {
	ID      string          `json:"id,omitempty"`
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
	Usage   *ClaudeUsage    `json:"usage,omitempty"`
	// StopReason is present on assistant messages in the on-disk
	// interactive transcript ("end_turn" marks a cleanly finished turn,
	// "tool_use" marks mid-work). The on-disk dialect has no `result`
	// line, so the transcript parser synthesizes the terminal marker from
	// end_turn (design §3 / R2). The stdout stream dialect carries the
	// same field but the transcript parser only acts on it in on-disk mode.
	StopReason string `json:"stop_reason,omitempty"`
}

// ClaudeContentItem is one element in a content array.
type ClaudeContentItem struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

// ExtractTextContent extracts text from a ClaudeMessage, handling both
// the bare-string and content-blocks shapes. ANSI is stripped as defense
// in depth — tool results may carry terminal formatting even when the
// CLI wrapper separates stderr from stdout cleanly.
func (m *ClaudeMessage) ExtractTextContent() string {
	if len(m.Content) == 0 {
		return ""
	}
	var simpleString string
	if err := json.Unmarshal(m.Content, &simpleString); err == nil {
		return runner.StripANSI(simpleString)
	}
	var contentBlocks []ClaudeContentItem
	if err := json.Unmarshal(m.Content, &contentBlocks); err == nil {
		var textParts []string
		for _, block := range contentBlocks {
			if block.Type == "text" && block.Text != "" {
				textParts = append(textParts, runner.StripANSI(block.Text))
			}
		}
		return strings.Join(textParts, "\n")
	}
	return ""
}

// ExtractToolUses extracts tool_use blocks from a content array.
func (m *ClaudeMessage) ExtractToolUses() []ClaudeContentItem {
	if len(m.Content) == 0 {
		return nil
	}
	var contentBlocks []ClaudeContentItem
	if err := json.Unmarshal(m.Content, &contentBlocks); err != nil {
		return nil
	}
	var toolUses []ClaudeContentItem
	for _, block := range contentBlocks {
		if block.Type == "tool_use" {
			toolUses = append(toolUses, block)
		}
	}
	return toolUses
}

// ExtractToolResults extracts tool_result blocks from a user message's
// content array (stripping ANSI from each).
func (m *ClaudeMessage) ExtractToolResults() []ClaudeContentItem {
	if len(m.Content) == 0 {
		return nil
	}
	var contentBlocks []ClaudeContentItem
	if err := json.Unmarshal(m.Content, &contentBlocks); err != nil {
		return nil
	}
	var toolResults []ClaudeContentItem
	for _, block := range contentBlocks {
		if block.Type == "tool_result" {
			block.Content = runner.StripANSI(block.Content)
			toolResults = append(toolResults, block)
		}
	}
	return toolResults
}

// ClaudeUsage carries detailed token-usage info.
type ClaudeUsage struct {
	InputTokens              int               `json:"input_tokens"`
	OutputTokens             int               `json:"output_tokens"`
	CacheCreationInputTokens int               `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int               `json:"cache_read_input_tokens,omitempty"`
	ServerToolUse            *ClaudeServerTool `json:"server_tool_use,omitempty"`
}

// ClaudeServerTool carries server-side tool counters (web search, etc.).
type ClaudeServerTool struct {
	WebSearchRequests int `json:"web_search_requests,omitempty"`
}

// ClaudeToolUse is the body of a top-level tool_use stream event.
type ClaudeToolUse struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ClaudeError accepts both the {code,message} object form and the bare
// string form used by system/api_retry.
type ClaudeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// UnmarshalJSON accepts either an object or a bare string. A bare string
// is stored in Code and Message is left empty.
func (e *ClaudeError) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		e.Code = s
		return nil
	}
	type alias ClaudeError
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*e = ClaudeError(a)
	return nil
}

// ClaudeContentBlock is the body of content_block_start.
type ClaudeContentBlock struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Text string `json:"text,omitempty"`
}

// ClaudeDelta is the body of content_block_delta / message_delta.
type ClaudeDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
}

// =============================================================================
// Stream decoding
// =============================================================================

// DecodeStreamLine satisfies [Codec]. Returns zero or more events.
// Errors here are surfaced as warn-level log events upstream; lines that
// look like CLI debug noise (non-JSON banners, etc.) are silently skipped.
func (c *Claude) DecodeStreamLine(state State, runID uuid.UUID, line string) ([]*domain.RunEvent, error) {
	s := state.(*claudeState)
	events, err := parseClaudeStreamEvents(s, runID, line)
	if err != nil {
		return events, err
	}
	for _, evt := range events {
		if evt == nil {
			continue
		}
		if data, ok := evt.Data.(*domain.RateLimitEventData); ok {
			s.rateLimit = data
		}
	}
	return events, nil
}

// PostClassify satisfies [Codec]. When the stream produced a terminal
// rate-limit event but the process still exited cleanly, flip the result
// so the orchestrator sees the failure.
func (c *Claude) PostClassify(state State, result *runner.ExecuteResult) {
	s := state.(*claudeState)
	if s.rateLimit == nil {
		return
	}
	if result.Success {
		result.Success = false
		result.ExitCode = 429
		msg := strings.TrimSpace(s.rateLimit.Message)
		if msg == "" {
			msg = "rate limit reached"
		}
		result.ErrorMessage = msg
		// Drop the success summary; PostClassify swept it.
		result.Summary = nil
	}
}

// ClassifyTerminalError satisfies [Codec]. Claude has only one
// recognised typed-failure shape today: stderr mentions "session" +
// "not found" → ErrCodeRunnerSessionExpired. All other failures fall
// through to ErrCodeRunnerExecution.
func (c *Claude) ClassifyTerminalError(stderr string, exitCode int) *domain.RunnerError {
	if strings.Contains(stderr, "session") && strings.Contains(stderr, "not found") {
		return domain.NewRunnerSessionExpiredError(c.Type(), errors.New(strings.TrimSpace(stderr)))
	}
	return nil
}

// Classify satisfies [Codec]. Claude classification order:
//
//  1. Captured rate-limit signal — PostClassify rewrites ErrorMessage
//     when the result event carried `is_error: true` and the body
//     parsed via detectClaudeRateLimit. Detect the rewritten message
//     here so we always return ReasonRateLimit on rate-limited runs.
//  2. Session-not-found stderr — ReasonSessionExpired (matches the
//     [ClassifyTerminalError] mapping).
//  3. Residual TextClassifier — covers unknown/deprecated model,
//     auth, quota, network, context-length, etc.
//
// Returns nil only when stderr is empty and exitCode == 0.
func (c *Claude) Classify(stderr string, exitCode int) *fallback.ClassifiedError {
	if stderr == "" && exitCode == 0 {
		return nil
	}
	if rl := detectClaudeRateLimit(stderr); rl.Detected {
		return fallback.New(fallback.ReasonRateLimit, strings.TrimSpace(stderr), nil)
	}
	if strings.Contains(stderr, "session") && strings.Contains(stderr, "not found") {
		return fallback.New(fallback.ReasonSessionExpired, strings.TrimSpace(stderr), nil)
	}
	return fallback.NewTextClassifier().Classify(fallback.ClassifyInput{
		RunnerType: string(c.Type()),
		Stderr:     stderr,
		ExitCode:   exitCode,
	})
}

// UpdateMetrics satisfies [Codec]. RateLimitEventData is captured into
// state by DecodeStreamLine; here we only update rolling counters.
func (c *Claude) UpdateMetrics(event *domain.RunEvent, metrics *runner.ExecutionMetrics, lastAssistant *string) {
	if event == nil {
		return
	}
	switch data := event.Data.(type) {
	case *domain.MessageEventData:
		if data.Role == "assistant" && !data.EvidenceOnly {
			*lastAssistant = data.Content
			metrics.TurnsUsed++
		}
	case *domain.ToolCallEventData:
		metrics.ToolCallCount++
	case *domain.MetricEventData:
		if data.Name == "tokens" {
			totalTokens := int(data.Value)
			if totalTokens > metrics.TokensInput+metrics.TokensOutput {
				metrics.TokensOutput = totalTokens - metrics.TokensInput
			}
		}
	case *domain.UsageEventData:
		metrics.TokensInput = data.InputTokens
		metrics.TokensOutput = data.OutputTokens
		metrics.CacheReadTokens = data.CacheReadTokens
		metrics.CacheCreationTokens = data.CacheCreationTokens
	case *domain.ChargeEventData:
		if data.AmountMicroUSD != nil {
			metrics.CostEstimateUSD = float64(*data.AmountMicroUSD) / 1_000_000
		}
	}
}
