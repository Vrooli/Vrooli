package agentchat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	agentbrief "github.com/vrooli/agentbrief-go"
	internalbrief "portal/internal/brief"
	internalchat "portal/internal/chat"
	"portal/internal/integrations/agentmanager"
)

var (
	ErrNoAgentPrompt          = errors.New("no user message available for agent run")
	ErrAgentImagesUnsupported = errors.New("agent manager does not support image attachments")
)

type ImageResolver interface {
	ResolveMessageImages(context.Context, string, string) ([][]byte, error)
}

type AgentManager interface {
	FindAdmission(context.Context, string) (agentmanager.Session, error)
	Run(context.Context, string) (agentmanager.RunState, error)
	Stop(context.Context, string) (agentmanager.RunState, error)
	Start(ctx context.Context, input agentmanager.StartInput) (agentmanager.Session, error)
	StreamRunEvents(ctx context.Context, runID string, emit func(agentmanager.ActivityEvent) error) error
}

type Service struct {
	chat       *internalchat.Service
	agent      AgentManager
	runs       Repository
	images     ImageResolver
	briefs     BriefProvider
	briefStore BriefStore
}

type BriefProvider interface {
	Build(context.Context, internalbrief.BuildInput) (internalbrief.Record, error)
}

type BriefStore interface {
	Get(context.Context, string) (internalbrief.Record, error)
	RecordUse(context.Context, internalbrief.UseInput) (bool, error)
}

type Config struct {
	Chat         *internalchat.Service
	AgentManager AgentManager
	Runs         Repository
	Images       ImageResolver
	Briefs       BriefProvider
	BriefStore   BriefStore
}

type StreamInput struct {
	ChatID        string
	FromMessageID string
}

type StreamResult struct {
	Message internalchat.Message
	Session agentmanager.Session
	BriefID string
}

func NewService(cfg Config) *Service {
	return &Service{chat: cfg.Chat, agent: cfg.AgentManager, runs: cfg.Runs, images: cfg.Images, briefs: cfg.Briefs, briefStore: cfg.BriefStore}
}

func NewAgentManagerFromEnv() (AgentManager, error) {
	return agentmanager.NewServiceFromEnv()
}

func (s *Service) Stream(ctx context.Context, input StreamInput, emit func(agentmanager.ActivityEvent) error) (StreamResult, error) {
	if s == nil || s.chat == nil || s.agent == nil || s.runs == nil {
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
	briefID := ""
	if s.briefs != nil {
		record, buildErr := s.briefs.Build(ctx, internalbrief.BuildInput{Prompt: prompt, Consumer: agentbrief.ConsumerPortalAgent, ChatID: input.ChatID, MessageID: fromID, Harness: string(chat.AgentHarness), BudgetMS: 2500})
		if buildErr == nil {
			briefID = record.ID
			if strings.TrimSpace(record.Rendered) != "" {
				prompt = record.Rendered + "\n\n" + prompt
			}
		}
	}
	binding, err := s.runs.Reserve(ctx, input.ChatID, fromID)
	if err != nil {
		return StreamResult{}, err
	}
	attachmentIDs, err := s.uploadImages(ctx, input.ChatID, fromID)
	if err != nil {
		return StreamResult{}, err
	}
	session, err := s.agent.Start(ctx, agentmanager.StartInput{
		AdmissionID:   binding.ID,
		ChatID:        input.ChatID,
		Prompt:        prompt,
		AttachmentIDs: attachmentIDs,
	})
	if err != nil {
		return StreamResult{}, err
	}
	// Persist the returned owner identity even if the renderer disconnected during launch.
	bindCtx, cancelBind := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	err = s.runs.Bind(bindCtx, binding.ID, session)
	cancelBind()
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
		Model:           "agent-manager",
		BriefID:         briefID,
	})
	if err != nil {
		return StreamResult{}, err
	}
	s.recordBriefReferences(ctx, briefID, msg.Content)
	return StreamResult{Message: msg, Session: session, BriefID: briefID}, nil
}

func (s *Service) recordBriefReferences(ctx context.Context, briefID, content string) {
	if s.briefStore == nil || strings.TrimSpace(briefID) == "" {
		return
	}
	record, err := s.briefStore.Get(ctx, briefID)
	if err != nil {
		return
	}
	for index, item := range record.Items {
		if strings.TrimSpace(item.Path) == "" && strings.TrimSpace(item.SuggestedCommand) == "" {
			continue
		}
		if (item.Path != "" && strings.Contains(content, item.Path)) || (item.SuggestedCommand != "" && strings.Contains(content, item.SuggestedCommand)) {
			_, _ = s.briefStore.RecordUse(ctx, internalbrief.UseInput{BriefID: briefID, ItemIndex: index, Kind: "REFERENCED"})
		}
	}
}

func (s *Service) uploadImages(ctx context.Context, chatID, messageID string) ([]string, error) {
	if s.images == nil {
		return nil, nil
	}
	images, err := s.images.ResolveMessageImages(ctx, chatID, messageID)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, nil
	}
	uploader, ok := s.agent.(agentmanager.AttachmentUploader)
	if !ok {
		return nil, ErrAgentImagesUnsupported
	}
	ids := make([]string, 0, len(images))
	for index, pixels := range images {
		id, err := uploader.UploadAttachment(ctx, pixels, fmt.Sprintf("portal-context-%d.png", index+1), "image/png")
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// Resolve accepts Portal chat/message identity; callers cannot choose a remote run.
func (s *Service) Resolve(ctx context.Context, input StreamInput) (Binding, error) {
	if s == nil || s.runs == nil || s.agent == nil {
		return Binding{}, agentmanager.ErrUnavailable
	}
	// Durable admission ownership survives deletion of its source chat.
	binding, err := s.runs.Get(ctx, input.ChatID, input.FromMessageID)
	if err != nil {
		return Binding{}, err
	}
	if binding.RunID != "" {
		return binding, nil
	}
	session, err := s.agent.FindAdmission(ctx, binding.ID)
	if err != nil {
		return binding, err
	}
	if err := s.runs.Bind(ctx, binding.ID, session); err != nil {
		return binding, err
	}
	binding.RunID, binding.TaskID = session.RunID, session.TaskID
	return binding, nil
}

func (s *Service) Run(ctx context.Context, input StreamInput) (agentmanager.RunState, error) {
	binding, err := s.Resolve(ctx, input)
	if err != nil {
		return agentmanager.RunState{}, err
	}
	return s.agent.Run(ctx, binding.RunID)
}

func (s *Service) Stop(ctx context.Context, input StreamInput) (agentmanager.RunState, error) {
	binding, err := s.Resolve(ctx, input)
	if err != nil {
		return agentmanager.RunState{}, err
	}
	state, err := s.agent.Run(ctx, binding.RunID)
	if err != nil {
		return state, err
	}
	if state.Terminal {
		return state, nil
	}
	return s.agent.Stop(ctx, binding.RunID)
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

// ListAdmissions returns durable Portal identities only; it never contacts the
// agent owner or interprets a missing run binding as completion.
func (s *Service) ListAdmissions(ctx context.Context, token string, size int) (AdmissionPage, error) {
	if s == nil || s.runs == nil {
		return AdmissionPage{}, agentmanager.ErrUnavailable
	}
	return s.runs.List(ctx, token, size)
}
