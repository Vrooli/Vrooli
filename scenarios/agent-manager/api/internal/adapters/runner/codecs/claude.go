// Package codecs — claude.go is the [Codec] implementation for Anthropic
// Claude Code (the `claude` CLI invoked with `--output-format stream-json`).
//
// The [core.Runner] handles process launch, stdout scanning, transcript
// writing, lifecycle, and event emission. This file owns:
//
//   - CLI args + env shape (BuildArgs, BuildContinueArgs, BuildEnv)
//   - The Claude stream-json decoder (DecodeStreamLine, ParseTranscriptLine)
//   - Per-run state: text accumulator, tool-use accumulator, captured
//     session_id, /compact tracking, captured RateLimitEventData
//   - Result classification flip on rate-limit (PostClassify)
//   - Terminal-error classification from stderr (ClassifyTerminalError)
//   - Diagnostic helpers used to enrich `is_error: true` results
package codecs

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/fallback"

	"github.com/google/uuid"
)

// =============================================================================
// Constants
// =============================================================================

// ClaudeCLICommand is the binary name resolved on the host PATH.
const ClaudeCLICommand = "claude"

// ClaudeResourceCommand is the legacy Vrooli wrapper name kept around for
// transition-period process detection in the reconciler/terminator.
const ClaudeResourceCommand = "resource-claude-code"

const claudeTagEnvKey = "CLAUDE_CODE_AGENT_TAG"

// =============================================================================
// Codec
// =============================================================================

// Claude is the [Codec] implementation for the Claude Code CLI.
type Claude struct {
	baseCodec
}

func (c *Claude) ToolCapabilityMap() map[string]string {
	return map[string]string{"Read": "file-read", "Write": "file-edit", "Edit": "file-edit", "Bash": "shell", "Glob": "search", "Grep": "search", "WebSearch": "network", "WebFetch": "network", "Task": "delegate", "TodoWrite": "plan", "wait": "wait"}
}

var claudeToolTranslations = map[domain.CanonicalTool]string{
	domain.CanonicalToolRead: "Read", domain.CanonicalToolWrite: "Write", domain.CanonicalToolEdit: "Edit",
	domain.CanonicalToolGlob: "Glob", domain.CanonicalToolGrep: "Grep", domain.CanonicalToolShell: "Bash",
	domain.CanonicalToolWebSearch: "WebSearch", domain.CanonicalToolWebFetch: "WebFetch",
}

// claudeBase is the identity shared by NewClaude and NewClaudeForTest.
func claudeBase() baseCodec {
	return baseCodec{
		runnerType:     domain.RunnerTypeClaudeCode,
		binaryDesc:     "claude CLI",
		installHint:    "Install: npm install -g @anthropic-ai/claude-code",
		tagEnvKey:      claudeTagEnvKey,
		continuePrefix: "claude",
		goalStatus:     func(line string) (string, bool, bool) { return declaredGoalStatus(line, "goal_status", "goalStatus") },
		labels: Labels{
			StartMessage:         "Claude Code execution started",
			EndMessage:           "Claude Code execution completed",
			ContinueStartMessage: "Claude Code continuation started",
			ContinueEndMessage:   "Claude Code continuation completed",
		},
	}
}

// NewClaude resolves the `claude` binary on PATH and returns a codec ready
// to be wrapped in [core.NewRunner]. Returns a codec with Available=false
// (rather than an error) when the binary is missing, so the runner
// registry can register a stub instead.
func NewClaude() (*Claude, error) {
	c := &Claude{baseCodec: resolveBinary(claudeBase(), ClaudeCLICommand)}
	c.newParser = c.NewTranscriptParser
	return c, nil
}

// NewClaudeForTest returns a Claude codec with a fake binary path and
// Available=false. Used by codec tests that exercise BuildArgs / decode
// paths without launching a real process.
func NewClaudeForTest() *Claude {
	c := &Claude{baseCodec: testBase(claudeBase(), "/fake/path", "test claude codec")}
	c.newParser = c.NewTranscriptParser
	return c
}

// NewClaudeForTestWithBinary is a test-only constructor for process replay.
func NewClaudeForTestWithBinary(path string) *Claude {
	c := NewClaudeForTest()
	c.binaryPath, c.available = path, true
	return c
}

// HasChargeSource reports that Claude's native result payload is the charge
// source; it does not require the shared pricing lookup.
func (c *Claude) HasChargeSource() bool { return true }

// Capabilities satisfies [Codec].
func (c *Claude) Capabilities() runner.Capabilities {
	return runner.Capabilities{
		SupportsMessages:         true,
		SupportsToolEvents:       true,
		SupportsCostTracking:     true,
		SupportsStreaming:        true,
		SupportsCancellation:     true,
		SupportsContinuation:     true, // Claude Code supports --resume
		SupportsImageAttachments: true,
		SupportsToolRestriction:  true,
		ToolRestrictionMappings:  canonicalToolMappings(claudeToolTranslations),
		SupportsEffort:           true,
		EffortMappings:           map[string]string{"low": "low", "medium": "medium", "high": "high", "xhigh": "xhigh", "max": "max"},
		MaxTurns:                 0, // unlimited
		SupportsRunnerDefault:    true,
		SupportedFeatures:        []string{"EnableBrowser"},
		AllowedExtraFlags:        nil,
	}
}

// BuildEnv satisfies [Codec]. The tag is the value the launcher writes to
// CLAUDE_CODE_AGENT_TAG; codec extras are merged on top. Binary resolution,
// availability, probing, labels, type, tag-key and continuation tags are
// provided by the embedded [baseCodec].
func (c *Claude) BuildEnv(tag string, extras map[string]string) []string {
	return standardBuildEnv(claudeTagEnvKey, tag, extras)
}

// BuildPrompt satisfies [Codec]. Claude reads image attachments by file
// path; we prepend the paths to the prompt.
func (c *Claude) BuildPrompt(prompt string, attachments []runner.Attachment) string {
	if len(attachments) == 0 {
		return prompt
	}
	var sb strings.Builder
	for _, att := range attachments {
		sb.WriteString(att.FilePath)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(prompt)
	return sb.String()
}

func (c *Claude) ControlArgs(cfg *domain.RunConfig) ([]string, error) { return claudeControlArgs(cfg) }

// BuildArgs satisfies [Codec]. Claude has no per-run state to stash
// from the request; the state argument is unused.
func (c *Claude) BuildArgs(state State, req runner.ExecuteRequest) []string {
	args := []string{
		"--print",
		"--output-format", "stream-json",
		"--verbose", // required with --print --output-format stream-json
	}

	cfg := req.GetConfig()
	if s, ok := state.(*claudeState); ok {
		s.model = strings.TrimSpace(cfg.Model)
		if s.model == "" {
			s.model = "unknown"
		}
	}
	if cfg.MaxTurns > 0 {
		args = append(args, "--max-turns", strconv.Itoa(cfg.MaxTurns))
	} else {
		args = append(args, "--max-turns", "30")
	}
	// Centralize portable control translation with interactive launches.
	controlArgs, _ := c.ControlArgs(cfg)
	args = append(args, controlArgs...)

	if cfg.SkipPermissionPrompt {
		args = append(args, "--dangerously-skip-permissions")
	}

	if req.SystemPrompt != "" {
		args = append(args, "--append-system-prompt", req.SystemPrompt)
	}

	if cfg.Features.EnableBrowser {
		args = append(args, "--chrome")
	}

	if extras, ok := cfg.ExtraFlags[domain.RunnerTypeClaudeCode]; ok {
		args = append(args, extras...)
	}

	args = append(args, "-") // read prompt from stdin
	return args
}

// BuildContinueArgs satisfies [Codec]. Claude has no per-run state to
// stash; the state argument is unused.
func (c *Claude) BuildContinueArgs(_ State, req runner.ContinueRequest) []string {
	args := []string{
		"--print",
		"--output-format", "stream-json",
		"--verbose",
		"--resume", req.SessionID,
	}
	cfg := req.GetConfig()
	if cfg.MaxTurns > 0 {
		args = append(args, "--max-turns", strconv.Itoa(cfg.MaxTurns))
	} else {
		args = append(args, "--max-turns", "30")
	}
	controlArgs, _ := c.ControlArgs(cfg)
	args = append(args, controlArgs...)
	if cfg.SkipPermissionPrompt {
		args = append(args, "--dangerously-skip-permissions")
	}
	if cfg.Features.EnableBrowser {
		args = append(args, "--chrome")
	}
	if extras, ok := cfg.ExtraFlags[domain.RunnerTypeClaudeCode]; ok {
		args = append(args, extras...)
	}
	return append(args, "-")
}

// =============================================================================
// State
// =============================================================================

// claudeState carries per-run mutable state through the stream decode loop.
// Implements [State].
type claudeState struct {
	textBuffer       strings.Builder
	toolUseActive    bool
	toolUseID        string
	toolUseName      string
	toolUsePayload   strings.Builder
	lastAssistant    string
	lastMessageEvent *domain.RunEvent
	sessionID        string
	gotResult        bool
	resultIsError    bool
	model            string
	retainUser       bool

	// /compact command tracking
	pendingCompact bool
	compactCommand string
	compactFocus   string

	// Captured by DecodeStreamLine when the terminal `result` event is a
	// rate-limit; consumed by PostClassify to flip Success=false / 429.
	rateLimit *domain.RateLimitEventData
}

func (s *claudeState) SessionID() string { return s.sessionID }

func (p *claudeTranscriptParser) SetTranscriptRetention(retain bool) { p.state.retainUser = retain }

// NewState satisfies [Codec].
func (c *Claude) NewState() State { return &claudeState{} }

// =============================================================================
// Stream-event types
// =============================================================================

// ClaudeStreamEvent represents a single event from Claude Code's
// stream-json output.
type ClaudeStreamEvent struct {
	Type      string          `json:"type"`
	Subtype   string          `json:"subtype,omitempty"`
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

// NewTranscriptParser satisfies [Codec]. Single-line parsing is provided by
// the embedded [baseCodec.ParseTranscriptLine], which delegates here.
func (c *Claude) NewTranscriptParser() runner.TranscriptParser {
	return &claudeTranscriptParser{state: &claudeState{}}
}

func (p *claudeTranscriptParser) SetTranscriptModel(model string) {
	p.state.model = strings.TrimSpace(model)
	if p.state.model == "" {
		p.state.model = "unknown"
	}
}

type claudeTranscriptParser struct {
	state *claudeState
	// onDisk latches once a camelCase `sessionId` field is seen, marking
	// this replay as claude's on-disk interactive transcript rather than
	// the --print stdout stream. In on-disk mode the parser synthesizes
	// the terminal marker from assistant stop_reason=end_turn (there is no
	// `result` line on disk, design §3 / R2). The stdout dialect never
	// carries sessionId, so this never trips during pipe-mode replay.
	onDisk bool
}

func (p *claudeTranscriptParser) ParseTranscriptLine(runID uuid.UUID, line string) runner.TranscriptParseResult {
	events, err := parseClaudeStreamEvents(p.state, runID, line)
	result := runner.TranscriptParseResult{
		Events:    events,
		Timestamp: transcriptLineTimestamp(line),
		Err:       err,
	}

	var streamEvent ClaudeStreamEvent
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &streamEvent); err == nil {
		result.SessionID = streamEvent.SessionID
		if streamEvent.SessionIDAlt != "" {
			result.SessionID = streamEvent.SessionIDAlt
			p.onDisk = true
		}
		// On-disk interactive dialect: no `result` line ever arrives, so
		// the terminal marker is the final assistant turn whose
		// stop_reason is end_turn. stop_reason=tool_use means the turn is
		// mid-work and is NOT terminal. Phase 4 layers an idle debounce on
		// top of this marker (interactive sessions stay open awaiting input).
		if p.onDisk && strings.EqualFold(streamEvent.Type, "assistant") &&
			streamEvent.Message != nil && streamEvent.Message.StopReason == "end_turn" {
			result.Terminal = &runner.TranscriptTerminal{Success: true, ExitCode: 0}
		}
		if strings.EqualFold(streamEvent.Type, "result") {
			terminal := &runner.TranscriptTerminal{
				Success:  !streamEvent.IsError,
				ExitCode: 0,
			}
			if streamEvent.IsError {
				terminal.Success = false
				terminal.ExitCode = 1
				terminal.ErrorMessage = strings.TrimSpace(decodeClaudeResultString(streamEvent.Result))
				// Use the same rate-limit detector as the live stream path
				// (DecodeStreamLine → parseClaudeResultEvent →
				// detectClaudeRateLimit). Previously the transcript-replay
				// code had a narrower string check that diverged from the
				// live path; unifying them ensures a recovered run sees
				// the same exit code as the live run.
				if rl := detectClaudeRateLimit(terminal.ErrorMessage); rl.Detected {
					terminal.ExitCode = 429
				}
				if terminal.ErrorMessage == "" {
					terminal.ErrorMessage = "runner reported terminal error"
				}
			}
			result.Terminal = terminal
		}
	}
	return result
}

// =============================================================================
// Stream parser
// =============================================================================

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
		resultEvents, err := parseClaudeResultEvent(runID, &streamEvent, state.model)
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
			return []*domain.RunEvent{domain.NewCompactionEvent(
				runID,
				strings.TrimSpace(sysResult),
				"auto",
				"",
				0, 0, 0,
				"",
			)}, nil
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

	case "init", "start", "ping", "heartbeat":
		return nil, nil

	// Interactive-only record types written to claude's on-disk session
	// transcript (never present in the --print stdout stream). They carry
	// UI/session metadata, not agent output, so they are dropped silently
	// rather than logged as "unhandled" debug noise (design §3). Listing
	// them here is safe for the stdout path — those runs never emit them.
	case "mode", "permission-mode", "ai-title", "last-prompt",
		"attachment", "file-history-snapshot", "queue-operation",
		"frame-link", "summary":
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
					return []*domain.RunEvent{domain.NewCompactionEvent(
						runID, summary, "manual", state.compactFocus,
						0, 0, 0, state.compactCommand,
					)}
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
			return []*domain.RunEvent{domain.NewCompactionEvent(
				runID, summary, "manual", state.compactFocus,
				0, 0, 0, state.compactCommand,
			)}
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
	toolUses := ev.Message.ExtractToolUses()
	for _, tool := range toolUses {
		var input map[string]interface{}
		if tool.Input != nil {
			_ = json.Unmarshal(tool.Input, &input)
		}
		events = append(events, domain.NewToolCallEvent(runID, tool.Name, tool.ID, input))
	}
	if len(events) > 0 {
		return events
	}
	return []*domain.RunEvent{domain.NewLogEvent(runID, "debug", "Assistant turn started")}
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
		MessageID:         messageID,
		ConversationID:    conversationID,
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
func parseClaudeResultEvent(runID uuid.UUID, event *ClaudeStreamEvent, model string) ([]*domain.RunEvent, error) {
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
				PayloadKind:         domain.PayloadKindUsage,
				InputTokens:         usageInputTokens(event.Usage),
				OutputTokens:        usageOutputTokens(event.Usage),
				CacheCreationTokens: usageCacheCreation(event.Usage),
				CacheReadTokens:     usageCacheRead(event.Usage),
				ServiceTier:         event.ServiceTier,
				RunnerType:          string(domain.RunnerTypeClaudeCode),
				Model:               model,
			},
		}
		amount := int64(event.TotalCostUSD*1_000_000 + 0.5)
		chargeEvent := &domain.RunEvent{ID: uuid.New(), RunID: runID, EventType: domain.EventTypeMetric, Timestamp: time.Now(), Data: &domain.ChargeEventData{
			PayloadKind:    domain.PayloadKindCharge,
			Basis:          domain.ChargeBasisMetered,
			AmountMicroUSD: &amount,
			Currency:       "USD",
			RunnerType:     string(domain.RunnerTypeClaudeCode),
			Model:          model,
		}}
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

// secretRedactors strips obvious credential patterns out of diagnostics
// before they're attached to error events. Not full DLP — just the
// patterns historically observed leaking via CLI wrappers.
var secretRedactors = []*regexp.Regexp{
	regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._\-]{8,}`),
	regexp.MustCompile(`sk-[A-Za-z0-9_\-]{8,}`),
	regexp.MustCompile(`(?i)api[_-]?key[=:\s]+[A-Za-z0-9._\-]{8,}`),
}

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

// Compile-time interface checks.
var (
	_ Codec = (*Claude)(nil)
	_ State = (*claudeState)(nil)
)
