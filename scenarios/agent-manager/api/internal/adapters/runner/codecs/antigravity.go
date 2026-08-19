// Package codecs provides the Agent Manager adapter for Google's Antigravity
// CLI. Antigravity's print mode is intentionally treated as a text stream:
// the adapter records assistant output and lifecycle status without claiming
// tool or usage events that the CLI does not expose in a stable contract.
package codecs

import (
	"encoding/json"
	"strings"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/fallback"

	"github.com/google/uuid"
)

const (
	// AntigravityCLICommand is the host binary installed by resource-antigravity.
	AntigravityCLICommand = "agy"
	antigravityTagEnvKey  = "ANTIGRAVITY_AGENT_TAG"
)

// Antigravity is the conservative print-mode codec for Google's agy CLI.
type Antigravity struct {
	baseCodec
}

type antigravityState struct {
	sessionID string
}

func (s *antigravityState) SessionID() string { return s.sessionID }

func antigravityBase() baseCodec {
	return baseCodec{
		runnerType:     domain.RunnerTypeAntigravity,
		binaryDesc:     "Antigravity CLI",
		installHint:    "Run: vrooli resource install antigravity",
		tagEnvKey:      antigravityTagEnvKey,
		continuePrefix: "antigravity",
		labels: Labels{
			StartMessage:         "Antigravity execution started",
			EndMessage:           "Antigravity execution completed",
			ContinueStartMessage: "Antigravity continuation started",
			ContinueEndMessage:   "Antigravity continuation completed",
		},
	}
}

// NewAntigravity resolves agy on PATH. A missing optional install becomes an
// unavailable runner stub, matching the other external coding-agent codecs.
func NewAntigravity() (*Antigravity, error) {
	c := &Antigravity{baseCodec: resolveBinary(antigravityBase(), AntigravityCLICommand)}
	c.newParser = c.NewTranscriptParser
	return c, nil
}

// NewAntigravityForTest creates a codec without touching the host PATH.
func NewAntigravityForTest() *Antigravity {
	c := &Antigravity{baseCodec: testBase(antigravityBase(), "/fake/agy", "test antigravity codec")}
	c.newParser = c.NewTranscriptParser
	return c
}

// NewAntigravityForTestWithBinary is used only by harmless process-replay
// tests that provide a temporary fake executable.
func NewAntigravityForTestWithBinary(path string) *Antigravity {
	c := NewAntigravityForTest()
	c.binaryPath, c.available = path, true
	return c
}

func (c *Antigravity) ToolCapabilityMap() map[string]string { return map[string]string{} }

func (c *Antigravity) Capabilities() runner.Capabilities {
	return runner.Capabilities{
		SupportsMessages:         true,
		SupportsToolEvents:       false,
		SupportsCostTracking:     false,
		SupportsStreaming:        true,
		SupportsCancellation:     true,
		SupportsContinuation:     true,
		SupportsImageAttachments: false,
		SupportsToolRestriction:  false,
		SupportsEffort:           false,
		MaxTurns:                 0,
		SupportsRunnerDefault:    true,
		SupportedFeatures:        []string{},
		AllowedExtraFlags:        nil,
	}
}

func (c *Antigravity) ControlArgs(_ *domain.RunConfig) ([]string, error) { return nil, nil }

func (c *Antigravity) BuildArgs(_ State, req runner.ExecuteRequest) []string {
	cfg := req.GetConfig()
	args := []string{"--print", req.EffectivePrompt()}
	if cfg.Model != "" && !isAntigravityDefaultModel(cfg.Model) {
		args = append(args, "--model", cfg.Model)
	}
	if cfg.SkipPermissionPrompt {
		args = append(args, "--dangerously-skip-permissions")
	}
	return append(args, cfg.ExtraFlags[domain.RunnerTypeAntigravity]...)
}

func (c *Antigravity) BuildContinueArgs(_ State, req runner.ContinueRequest) []string {
	args := []string{"--conversation", req.SessionID, "--print", req.Prompt}
	if cfg := req.GetConfig(); cfg != nil {
		if cfg.Model != "" && !isAntigravityDefaultModel(cfg.Model) {
			args = append(args, "--model", cfg.Model)
		}
		if cfg.SkipPermissionPrompt {
			args = append(args, "--dangerously-skip-permissions")
		}
		args = append(args, cfg.ExtraFlags[domain.RunnerTypeAntigravity]...)
	}
	return args
}

// The resource-owned catalog uses "default" as a symbolic model because agy
// does not expose a stable unauthenticated model inventory. Omitting --model
// preserves agy's own runner default while keeping role resolution explicit.
func isAntigravityDefaultModel(model string) bool {
	return strings.EqualFold(strings.TrimSpace(model), "default")
}

func (c *Antigravity) BuildPrompt(_ string, _ []runner.Attachment) string { return "" }

func (c *Antigravity) BuildEnv(tag string, extras map[string]string) []string {
	return standardBuildEnv(antigravityTagEnvKey, tag, extras)
}

func (c *Antigravity) NewState() State { return &antigravityState{} }

// DecodeStreamLine accepts the stable text print surface and also tolerates a
// JSON object when a future agy build emits structured print output. Unknown
// JSON fields are ignored; raw text remains the fallback evidence.
func (c *Antigravity) DecodeStreamLine(state State, runID uuid.UUID, line string) ([]*domain.RunEvent, error) {
	s, ok := state.(*antigravityState)
	if !ok {
		return nil, domain.NewInternalError("antigravity: invalid state type", nil)
	}
	text, sessionID := antigravityText(line)
	if sessionID != "" && s.sessionID == "" {
		s.sessionID = sessionID
	}
	text = runner.StripANSI(strings.TrimSpace(text))
	if text == "" {
		return nil, nil
	}
	return []*domain.RunEvent{domain.NewProviderMessageEvent(runID, "assistant", text, domain.MessageEventData{
		ConversationID:    s.sessionID,
		ProviderOrigin:    "antigravity",
		ProviderEventType: "print",
		RawEvidenceRef:    "antigravity:print",
	})}, nil
}

func antigravityText(line string) (text, sessionID string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
		return trimmed, ""
	}
	var payload struct {
		Text           string `json:"text"`
		Content        string `json:"content"`
		Message        string `json:"message"`
		Output         string `json:"output"`
		SessionID      string `json:"sessionId"`
		SnakeSessionID string `json:"session_id"`
		ConversationID string `json:"conversationId"`
	}
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return trimmed, ""
	}
	for _, candidate := range []string{payload.Text, payload.Content, payload.Message, payload.Output} {
		if strings.TrimSpace(candidate) != "" {
			if payload.SessionID != "" {
				return candidate, payload.SessionID
			}
			if payload.SnakeSessionID != "" {
				return candidate, payload.SnakeSessionID
			}
			return candidate, payload.ConversationID
		}
	}
	return "", firstNonEmpty(payload.SessionID, payload.SnakeSessionID, payload.ConversationID)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (c *Antigravity) NewTranscriptParser() runner.TranscriptParser {
	return &antigravityTranscriptParser{codec: c, state: &antigravityState{}}
}

type antigravityTranscriptParser struct {
	codec *Antigravity
	state *antigravityState
}

func (p *antigravityTranscriptParser) ParseTranscriptLine(runID uuid.UUID, line string) runner.TranscriptParseResult {
	events, err := p.codec.DecodeStreamLine(p.state, runID, line)
	return runner.TranscriptParseResult{Events: events, SessionID: p.state.sessionID, Err: err}
}

func (c *Antigravity) UpdateMetrics(event *domain.RunEvent, metrics *runner.ExecutionMetrics, lastAssistant *string) {
	if event == nil {
		return
	}
	if message, ok := event.Data.(*domain.MessageEventData); ok && message.Role == "assistant" {
		*lastAssistant = message.Content
		metrics.TurnsUsed++
	}
}

func (c *Antigravity) OnEarlyTerminate(_ State, _ string) bool       { return false }
func (c *Antigravity) PostClassify(_ State, _ *runner.ExecuteResult) {}

func (c *Antigravity) ClassifyTerminalError(_ string, _ int) *domain.RunnerError { return nil }

func (c *Antigravity) Classify(stderr string, exitCode int) *fallback.ClassifiedError {
	if strings.TrimSpace(stderr) == "" && exitCode == 0 {
		return nil
	}
	return fallback.NewTextClassifier().Classify(fallback.ClassifyInput{
		RunnerType: string(c.Type()),
		Stderr:     stderr,
		ExitCode:   exitCode,
	})
}

var _ Codec = (*Antigravity)(nil)
