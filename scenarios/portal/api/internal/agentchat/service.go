package agentchat

import (
	"context"
	"errors"
	"strings"

	internalchat "portal/internal/chat"
	"portal/internal/integrations/agentmanager"
)

var ErrNoAgentPrompt = errors.New("no user message available for agent run")

type AgentManager interface {
	Start(ctx context.Context, input agentmanager.StartInput) (agentmanager.Session, error)
	StreamRunEvents(ctx context.Context, runID string, emit func(agentmanager.ActivityEvent) error) error
}

type Service struct {
	chat  *internalchat.Service
	agent AgentManager
}

type Config struct {
	Chat         *internalchat.Service
	AgentManager AgentManager
}

type StreamInput struct {
	ChatID        string
	FromMessageID string
	Model         string
	Harness       internalchat.AgentHarness
}

type StreamResult struct {
	Message internalchat.Message
	Session agentmanager.Session
}

func NewService(cfg Config) *Service {
	return &Service{chat: cfg.Chat, agent: cfg.AgentManager}
}

func NewAgentManagerFromEnv() (AgentManager, error) {
	return agentmanager.NewServiceFromEnv()
}

func (s *Service) Stream(ctx context.Context, input StreamInput, emit func(agentmanager.ActivityEvent) error) (StreamResult, error) {
	if s == nil || s.chat == nil || s.agent == nil {
		return StreamResult{}, agentmanager.ErrUnavailable
	}
	chat, err := s.chat.GetChat(ctx, input.ChatID)
	if err != nil {
		return StreamResult{}, err
	}
	messages, leafID, err := s.chat.GetTree(ctx, input.ChatID)
	if err != nil {
		return StreamResult{}, err
	}
	fromID := strings.TrimSpace(input.FromMessageID)
	if fromID == "" {
		fromID = leafID
	}
	prompt, err := userPrompt(messages, fromID)
	if err != nil {
		return StreamResult{}, err
	}
	harness := input.Harness
	if harness == "" {
		harness = chat.AgentHarness
	}
	if harness == "" {
		harness = internalchat.DefaultAgentHarness
	}
	model := strings.TrimSpace(input.Model)
	if model == "" {
		model = chat.Model
	}
	session, err := s.agent.Start(ctx, agentmanager.StartInput{
		ChatID:  input.ChatID,
		Prompt:  prompt,
		Harness: harness,
		Model:   model,
	})
	if err != nil {
		return StreamResult{}, err
	}
	var transcript strings.Builder
	if err := emit(agentmanager.ActivityEvent{
		Kind:  agentmanager.EventKindStatus,
		RunID: session.RunID,
		Text:  "Agent run started",
	}); err != nil {
		return StreamResult{}, err
	}
	err = s.agent.StreamRunEvents(ctx, session.RunID, func(ev agentmanager.ActivityEvent) error {
		if strings.TrimSpace(ev.Text) != "" {
			transcript.WriteString(ev.Text)
			transcript.WriteString("\n")
		}
		return emit(ev)
	})
	if err != nil {
		return StreamResult{}, err
	}
	content := strings.TrimSpace(transcript.String())
	if content == "" {
		content = "Agent run completed."
	}
	msg, err := s.chat.AppendAgentMessage(ctx, internalchat.SendMessageInput{
		ChatID:          input.ChatID,
		ParentMessageID: fromID,
		Content:         content,
		Model:           string(harness),
	})
	if err != nil {
		return StreamResult{}, err
	}
	return StreamResult{Message: msg, Session: session}, nil
}

func userPrompt(messages []internalchat.Message, messageID string) (string, error) {
	for _, msg := range messages {
		if msg.ID == messageID {
			if msg.Role != internalchat.RoleUser || strings.TrimSpace(msg.Content) == "" {
				return "", ErrNoAgentPrompt
			}
			return strings.TrimSpace(msg.Content), nil
		}
	}
	return "", internalchat.ErrNotFound{Resource: "message", ID: messageID}
}
