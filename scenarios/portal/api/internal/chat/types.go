package chat

import (
	"errors"
	"fmt"
	"time"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/shared"
)

const (
	DefaultModel        = "anthropic/claude-3.5-sonnet"
	DefaultAgentHarness = AgentHarnessClaudeCode
)

type ChatMode string

const (
	ChatModeLLM   ChatMode = "llm"
	ChatModeAgent ChatMode = "agent"
)

type AgentHarness string

const (
	AgentHarnessClaudeCode AgentHarness = "claude-code"
	AgentHarnessCodex      AgentHarness = "codex"
	AgentHarnessOpencode   AgentHarness = "opencode"
	AgentHarnessGrok       AgentHarness = "grok"
)

type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleAgent     MessageRole = "agent"
)

type Chat struct {
	ID                  string
	Title               string
	Preview             string
	GroupID             string
	SortOrder           int32
	Model               string
	WebSearchEnabled    bool
	Mode                ChatMode
	AgentHarness        AgentHarness
	ActiveLeafMessageID string
	SystemPrompt        string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type ChatGroup struct {
	ID        string
	Name      string
	Color     string
	Collapsed bool
	SortOrder int32
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Message struct {
	ID                string
	ChatID            string
	ParentMessageID   string
	SiblingIndex      int32
	Role              MessageRole
	Content           string
	Model             string
	TokenCount        int32
	ResponseID        string
	FinishReason      string
	WebSearch         *bool
	SearchAttachments []SearchAttachment
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type SearchHit struct {
	ProviderID  string
	Type        string
	Title       string
	Snippet     string
	Path        string
	Score       float64
	RerankScore float64
	Locations   []string
}

type SearchAttachment struct {
	ID        string
	ChatID    string
	MessageID string
	Query     string
	Hits      []SearchHit
	Degraded  bool
	Reason    string
	LatencyMS int64
	CreatedAt time.Time
}

type UsageRecord struct {
	ID               string
	ChatID           string
	MessageID        string
	Provider         string
	Model            string
	PromptTokens     int32
	CompletionTokens int32
	TotalTokens      int32
	CostUSD          float64
	CreatedAt        time.Time
}

type CreateChatInput struct {
	Title            string
	GroupID          string
	Model            string
	WebSearchEnabled bool
	Mode             ChatMode
	AgentHarness     AgentHarness
}

type UpdateChatInput struct {
	ID                       string
	Title                    *string
	GroupID                  *string
	Model                    *string
	WebSearchEnabled         *bool
	ActiveLeafMessageID      *string
	ClearActiveLeafMessageID bool
}

type CreateGroupInput struct {
	Name  string
	Color string
}

type UpdateGroupInput struct {
	ID        string
	Name      *string
	Color     *string
	Collapsed *bool
	SortOrder *int32
}

type SendMessageInput struct {
	ChatID          string
	ParentMessageID string
	Role            MessageRole
	Content         string
	Model           string
	WebSearch       *bool
}

type CreateUsageInput struct {
	ChatID           string
	MessageID        string
	Provider         string
	Model            string
	PromptTokens     int32
	CompletionTokens int32
	CostUSD          float64
}

type CreateSearchAttachmentInput struct {
	ChatID    string
	MessageID string
	Query     string
	Hits      []SearchHit
	Degraded  bool
	Reason    string
	LatencyMS int64
}

type BranchMessageInput struct {
	MessageID string
	Content   string
	Model     string
}

type SearchInput struct {
	GroupID string
	Query   string
}

type ErrNotFound struct {
	Resource string
	ID       string
}

func (e ErrNotFound) Error() string {
	if e.Resource == "" {
		return fmt.Sprintf("not found: %s", e.ID)
	}
	return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

var ErrInvalidInput = errors.New("invalid chat input")

func NormalizeMode(mode ChatMode) ChatMode {
	if mode == "" {
		return ChatModeLLM
	}
	return mode
}

func NormalizeAgentHarness(h AgentHarness) AgentHarness {
	if h == "" {
		return DefaultAgentHarness
	}
	return h
}

func ChatModeFromProto(mode sharedv1.ChatMode) ChatMode {
	switch mode {
	case sharedv1.ChatMode_CHAT_MODE_AGENT:
		return ChatModeAgent
	default:
		return ChatModeLLM
	}
}

func AgentHarnessFromProto(h sharedv1.AgentHarness) AgentHarness {
	switch h {
	case sharedv1.AgentHarness_AGENT_HARNESS_CODEX:
		return AgentHarnessCodex
	case sharedv1.AgentHarness_AGENT_HARNESS_OPENCODE:
		return AgentHarnessOpencode
	case sharedv1.AgentHarness_AGENT_HARNESS_GROK:
		return AgentHarnessGrok
	default:
		return AgentHarnessClaudeCode
	}
}

func (m ChatMode) Proto() sharedv1.ChatMode {
	switch m {
	case ChatModeAgent:
		return sharedv1.ChatMode_CHAT_MODE_AGENT
	default:
		return sharedv1.ChatMode_CHAT_MODE_LLM
	}
}

func (h AgentHarness) Proto() sharedv1.AgentHarness {
	switch h {
	case AgentHarnessCodex:
		return sharedv1.AgentHarness_AGENT_HARNESS_CODEX
	case AgentHarnessOpencode:
		return sharedv1.AgentHarness_AGENT_HARNESS_OPENCODE
	case AgentHarnessGrok:
		return sharedv1.AgentHarness_AGENT_HARNESS_GROK
	default:
		return sharedv1.AgentHarness_AGENT_HARNESS_CLAUDE_CODE
	}
}
