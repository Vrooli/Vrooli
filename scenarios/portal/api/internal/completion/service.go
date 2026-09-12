package completion

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	internalbrief "portal/internal/brief"
	internalchat "portal/internal/chat"
	"portal/internal/integrations/openrouter"
)

const (
	defaultSystemPrompt = "You are Vrooli Portal, a concise operator assistant for the Vrooli ecosystem."
	openRouterProvider  = "openrouter"
)

var ErrNoCompletionMessages = errors.New("no messages available for completion")

type SkillResolver interface {
	ResolveSkills(ctx context.Context, ids []string) ([]Skill, error)
}

type Skill struct {
	ID      string
	Content string
}

type OpenRouterStreamer interface {
	StreamCompletion(ctx context.Context, req openrouter.CompletionRequest, emit func(openrouter.StreamEvent) error) error
}

type BriefProvider interface {
	Build(context.Context, internalbrief.BuildInput) (internalbrief.Record, error)
}

type BriefStore interface {
	Get(context.Context, string) (internalbrief.Record, error)
	RecordUse(context.Context, internalbrief.UseInput) (bool, error)
}

// MessageImageResolver owns the authenticated association from a conversation
// message to live rendered artifacts. It must reject inaccessible or expired
// attachments; returning no images is only valid for an unattached message.
type MessageImageResolver interface {
	ResolveMessageImages(context.Context, string, string) ([][]byte, error)
}

type Service struct {
	chat          *internalchat.Service
	openRouter    OpenRouterStreamer
	skillResolver SkillResolver
	briefs        BriefProvider
	briefStore    BriefStore
	images        MessageImageResolver
}

type Config struct {
	Chat          *internalchat.Service
	OpenRouter    OpenRouterStreamer
	SkillResolver SkillResolver
	Briefs        BriefProvider
	BriefStore    BriefStore
	Images        MessageImageResolver
}

type StreamInput struct {
	ChatID           string
	FromMessageID    string
	Model            string
	WebSearchEnabled bool
	SelectedSkillIDs []string
}

type StreamResult struct {
	AssistantMessage internalchat.Message
	Usage            internalchat.UsageRecord
	BriefID          string
}

func NewService(cfg Config) *Service {
	return &Service{
		chat:          cfg.Chat,
		openRouter:    cfg.OpenRouter,
		skillResolver: cfg.SkillResolver,
		briefs:        cfg.Briefs,
		briefStore:    cfg.BriefStore,
		images:        cfg.Images,
	}
}

func NewPromptManagerSkillResolver() SkillResolver {
	return promptManagerSkillResolver{}
}

func NewOpenRouterStreamerFromEnv() (OpenRouterStreamer, error) {
	cfg, err := openrouter.ResolveConfig()
	if err != nil {
		return nil, err
	}
	return openrouter.NewClient(cfg)
}

func (s *Service) BuildOpenRouterRequest(ctx context.Context, input StreamInput) (openrouter.CompletionRequest, string, error) {
	request, parentID, _, err := s.buildOpenRouterRequest(ctx, input)
	return request, parentID, err
}

func (s *Service) buildOpenRouterRequest(ctx context.Context, input StreamInput) (openrouter.CompletionRequest, string, string, error) {
	chat, err := s.chat.GetChat(ctx, input.ChatID)
	if err != nil {
		return openrouter.CompletionRequest{}, "", "", err
	}
	messages, leafID, err := s.chat.GetTree(ctx, input.ChatID)
	if err != nil {
		return openrouter.CompletionRequest{}, "", "", err
	}
	fromID := strings.TrimSpace(input.FromMessageID)
	if fromID == "" {
		fromID = leafID
	}
	path, err := activePath(messages, fromID)
	if err != nil {
		return openrouter.CompletionRequest{}, "", "", err
	}
	model := strings.TrimSpace(input.Model)
	if model == "" {
		model = chat.Model
	}
	if model == "" {
		model = internalchat.DefaultModel
	}

	currentPrompt := ""
	for _, msg := range path {
		if msg.Role == internalchat.RoleUser {
			currentPrompt = msg.Content
		}
	}
	systemPrompt, briefID := s.systemPromptForPrompt(ctx, chat, currentPrompt, input.SelectedSkillIDs)
	orMessages := []openrouter.Message{{Role: "system", Content: systemPrompt}}
	imageCount, imageBytes := 0, 0
	for _, msg := range path {
		role := openRouterRole(msg.Role)
		if role == "" {
			continue
		}
		outgoing := openrouter.Message{Role: role, Content: msg.Content}
		if role == "user" && s.images != nil {
			images, err := s.images.ResolveMessageImages(ctx, chat.ID, msg.ID)
			if err != nil {
				return openrouter.CompletionRequest{}, "", "", err
			}
			for _, pixels := range images {
				imageCount++
				imageBytes += len(pixels)
				if len(pixels) == 0 || imageCount > 4 || imageBytes > 32*1024*1024 {
					return openrouter.CompletionRequest{}, "", "", errors.New("message image context exceeds bound")
				}
				outgoing.Images = append(outgoing.Images, append([]byte(nil), pixels...))
			}
		}
		orMessages = append(orMessages, outgoing)
	}
	if len(orMessages) == 1 {
		return openrouter.CompletionRequest{}, "", "", ErrNoCompletionMessages
	}

	req := openrouter.CompletionRequest{
		Model:    model,
		Messages: orMessages,
		Stream:   true,
	}
	if input.WebSearchEnabled || chat.WebSearchEnabled || pathEnablesWebSearch(path) {
		req.Plugins = []openrouter.Plugin{{ID: "web", MaxResults: 5}}
	}
	return req, fromID, briefID, nil
}

func (s *Service) Stream(ctx context.Context, input StreamInput, emit func(openrouter.StreamEvent) error) (StreamResult, error) {
	if s.openRouter == nil {
		return StreamResult{}, openrouter.ErrAPIKeyMissing
	}
	req, parentID, briefID, err := s.buildOpenRouterRequest(ctx, input)
	if err != nil {
		return StreamResult{}, err
	}

	var content strings.Builder
	var last openrouter.StreamEvent
	if err := s.openRouter.StreamCompletion(ctx, req, func(ev openrouter.StreamEvent) error {
		last = ev
		if ev.Token != "" {
			content.WriteString(ev.Token)
		}
		return emit(ev)
	}); err != nil {
		return StreamResult{}, err
	}
	if strings.TrimSpace(content.String()) == "" {
		return StreamResult{}, errors.New("openrouter returned an empty completion")
	}

	assistant, err := s.chat.AppendAssistantMessage(ctx, internalchat.SendMessageInput{
		ChatID:          input.ChatID,
		ParentMessageID: parentID,
		Content:         content.String(),
		Model:           req.Model,
		BriefID:         briefID,
	})
	if err != nil {
		return StreamResult{}, err
	}
	s.recordBriefReferences(ctx, briefID, assistant.Content)
	usage, err := s.chat.CreateUsageRecord(ctx, internalchat.CreateUsageInput{
		ChatID:           input.ChatID,
		MessageID:        assistant.ID,
		Provider:         openRouterProvider,
		Model:            req.Model,
		PromptTokens:     last.Usage.PromptTokens,
		CompletionTokens: last.Usage.CompletionTokens,
	})
	if err != nil {
		return StreamResult{}, err
	}
	return StreamResult{AssistantMessage: assistant, Usage: usage, BriefID: briefID}, nil
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

func (s *Service) systemPromptForPrompt(ctx context.Context, chat internalchat.Chat, prompt string, selectedSkillIDs []string) (string, string) {
	parts := []string{defaultString(chat.SystemPrompt, defaultSystemPrompt)}
	briefID := ""
	if s.briefs != nil {
		if record, err := s.briefs.Build(ctx, internalbrief.BuildInput{Prompt: prompt, Consumer: "PORTAL_LLM", ChatID: chat.ID}); err == nil {
			briefID = record.ID
			if strings.TrimSpace(record.Rendered) != "" {
				parts = append(parts, record.Rendered)
			}
		}
	}
	if s.skillResolver == nil || len(selectedSkillIDs) == 0 {
		return strings.Join(parts, "\n\n"), briefID
	}
	skills, err := s.skillResolver.ResolveSkills(ctx, selectedSkillIDs)
	if err != nil || len(skills) == 0 {
		return strings.Join(parts, "\n\n"), briefID
	}
	var b strings.Builder
	b.WriteString("Selected operator skills. Apply this guidance directly in your response; it is not a tool call.\n")
	for _, skill := range skills {
		if strings.TrimSpace(skill.Content) == "" {
			continue
		}
		b.WriteString("\n[Skill: ")
		b.WriteString(skill.ID)
		b.WriteString("]\n")
		b.WriteString(strings.TrimSpace(skill.Content))
		b.WriteString("\n")
	}
	if strings.TrimSpace(b.String()) != "" {
		parts = append(parts, b.String())
	}
	return strings.Join(parts, "\n\n"), briefID
}

func activePath(messages []internalchat.Message, leafID string) ([]internalchat.Message, error) {
	if len(messages) == 0 {
		return nil, ErrNoCompletionMessages
	}
	byID := make(map[string]internalchat.Message, len(messages))
	for _, msg := range messages {
		byID[msg.ID] = msg
	}
	if strings.TrimSpace(leafID) == "" {
		sort.SliceStable(messages, func(i, j int) bool {
			return messages[i].CreatedAt.Before(messages[j].CreatedAt)
		})
		leafID = messages[len(messages)-1].ID
	}
	var reversed []internalchat.Message
	seen := make(map[string]bool)
	for id := leafID; id != ""; {
		if seen[id] {
			return nil, fmt.Errorf("message tree cycle at %s", id)
		}
		seen[id] = true
		msg, ok := byID[id]
		if !ok {
			return nil, internalchat.ErrNotFound{Resource: "message", ID: id}
		}
		reversed = append(reversed, msg)
		id = msg.ParentMessageID
	}
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	return reversed, nil
}

func openRouterRole(role internalchat.MessageRole) string {
	switch role {
	case internalchat.RoleUser:
		return "user"
	case internalchat.RoleAssistant:
		return "assistant"
	case internalchat.RoleSystem:
		return "system"
	default:
		return ""
	}
}

func pathEnablesWebSearch(path []internalchat.Message) bool {
	for _, msg := range path {
		if msg.Role == internalchat.RoleUser && msg.WebSearch != nil && *msg.WebSearch {
			return true
		}
	}
	return false
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

type promptManagerSkillResolver struct{}

func (promptManagerSkillResolver) ResolveSkills(ctx context.Context, ids []string) ([]Skill, error) {
	cleaned := make([]string, 0, len(ids))
	for _, id := range ids {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	if len(cleaned) == 0 {
		return nil, nil
	}
	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	args := append([]string{"skill", "read"}, cleaned...)
	cmd := exec.CommandContext(cmdCtx, "prompt-manager", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return []Skill{{ID: strings.Join(cleaned, ","), Content: string(out)}}, nil
}
