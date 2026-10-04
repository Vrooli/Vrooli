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
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"regexp"
	"strconv"
	"strings"
)

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
	// pricingService computes a charge for interactive runs whose on-disk
	// transcript has no cost line. Optional; nil leaves the charge unknown.
	pricingService PricingService
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
		goalStatus:     claudeGoalStatus,
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
func NewClaude(opts ...ClaudeOption) (*Claude, error) {
	c := &Claude{baseCodec: resolveBinary(claudeBase(), ClaudeCLICommand)}
	for _, opt := range opts {
		opt(c)
	}
	c.newParser = c.NewTranscriptParser
	return c, nil
}

// ClaudeOption configures a Claude codec.
type ClaudeOption func(*Claude)

// WithClaudePricingService injects the pricing lookup used to compute a charge
// for interactive runs, whose on-disk transcript carries usage but no cost line.
func WithClaudePricingService(svc PricingService) ClaudeOption {
	return func(c *Claude) { c.pricingService = svc }
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

// HasChargeSource reports whether a charge source is available. Claude's
// codec-pipe result line carries a native dollar cost, but the interactive
// on-disk transcript does not; the injected pricing lookup covers that case.
func (c *Claude) HasChargeSource() bool { return true }

// Capabilities satisfies [Codec].
func (c *Claude) Capabilities() runner.Capabilities {
	return codingAgentCapabilities(runner.Capabilities{
		SpawnCapabilities:        []runner.SpawnCapability{{ExecutionMode: "interactive", SandboxModes: []string{"tracking", "off"}, NativeObjective: true}},
		SupportsToolEvents:       true,
		SupportsCostTracking:     true,
		SupportsImageAttachments: true,
		SupportsToolRestriction:  true,
		ToolRestrictionMappings:  canonicalToolMappings(claudeToolTranslations),
		SupportsEffort:           true,
		EffortMappings:           map[string]string{"low": "low", "medium": "medium", "high": "high", "xhigh": "xhigh", "max": "max"},
		SupportedFeatures:        []string{"EnableBrowser"},
		AllowedExtraFlags:        nil,
	})
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
	textBuffer           strings.Builder
	toolUseActive        bool
	toolUseID            string
	toolUseName          string
	toolUsePayload       strings.Builder
	lastAssistant        string
	lastMessageEvent     *domain.RunEvent
	sessionID            string
	gotResult            bool
	resultIsError        bool
	model                string
	billing              domain.BillingSnapshot
	retainUser           bool
	turn                 int
	lastTurnInput        int64
	messagesSinceCompact int64
	lastCompaction       *domain.RunEvent

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

// secretRedactors strips obvious credential patterns out of diagnostics
// before they're attached to error events. Not full DLP — just the
// patterns historically observed leaking via CLI wrappers.
var secretRedactors = []*regexp.Regexp{
	regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._\-]{8,}`),
	regexp.MustCompile(`sk-[A-Za-z0-9_\-]{8,}`),
	regexp.MustCompile(`(?i)api[_-]?key[=:\s]+[A-Za-z0-9._\-]{8,}`),
}

// Compile-time interface checks.
var (
	_ Codec = (*Claude)(nil)
	_ State = (*claudeState)(nil)
)
