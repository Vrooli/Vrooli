package chat

import (
	"context"
	"strings"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListChats(ctx context.Context, input SearchInput) ([]Chat, []ChatGroup, error) {
	return s.repo.ListChats(ctx, input)
}

func (s *Service) CreateChat(ctx context.Context, input CreateChatInput) (Chat, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.GroupID = strings.TrimSpace(input.GroupID)
	input.Model = strings.TrimSpace(input.Model)
	input.Mode = NormalizeMode(input.Mode)
	input.AgentHarness = NormalizeAgentHarness(input.AgentHarness)
	return s.repo.CreateChat(ctx, input)
}

func (s *Service) GetChat(ctx context.Context, id string) (Chat, error) {
	return s.repo.GetChat(ctx, strings.TrimSpace(id))
}

func (s *Service) UpdateChat(ctx context.Context, input UpdateChatInput) (Chat, error) {
	input.ID = strings.TrimSpace(input.ID)
	return s.repo.UpdateChat(ctx, input)
}

func (s *Service) DeleteChat(ctx context.Context, id string) (bool, error) {
	return s.repo.DeleteChat(ctx, strings.TrimSpace(id))
}

func (s *Service) ListGroups(ctx context.Context) ([]ChatGroup, error) {
	return s.repo.ListGroups(ctx)
}

func (s *Service) CreateGroup(ctx context.Context, input CreateGroupInput) (ChatGroup, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Color = strings.TrimSpace(input.Color)
	return s.repo.CreateGroup(ctx, input)
}

func (s *Service) UpdateGroup(ctx context.Context, input UpdateGroupInput) (ChatGroup, error) {
	input.ID = strings.TrimSpace(input.ID)
	return s.repo.UpdateGroup(ctx, input)
}

func (s *Service) DeleteGroup(ctx context.Context, id string) (bool, error) {
	return s.repo.DeleteGroup(ctx, strings.TrimSpace(id))
}

func (s *Service) GetTree(ctx context.Context, chatID string) ([]Message, string, error) {
	return s.repo.ListMessages(ctx, strings.TrimSpace(chatID))
}

func (s *Service) SendUserMessage(ctx context.Context, input SendMessageInput) (Message, error) {
	input.ChatID = strings.TrimSpace(input.ChatID)
	input.ParentMessageID = strings.TrimSpace(input.ParentMessageID)
	input.Content = strings.TrimSpace(input.Content)
	input.Model = strings.TrimSpace(input.Model)
	input.Role = RoleUser
	return s.repo.SendMessage(ctx, input)
}

func (s *Service) AppendAssistantMessage(ctx context.Context, input SendMessageInput) (Message, error) {
	input.ChatID = strings.TrimSpace(input.ChatID)
	input.ParentMessageID = strings.TrimSpace(input.ParentMessageID)
	input.Content = strings.TrimSpace(input.Content)
	input.Model = strings.TrimSpace(input.Model)
	input.Role = RoleAssistant
	return s.repo.AppendMessage(ctx, input)
}

func (s *Service) AppendAgentMessage(ctx context.Context, input SendMessageInput) (Message, error) {
	input.ChatID = strings.TrimSpace(input.ChatID)
	input.ParentMessageID = strings.TrimSpace(input.ParentMessageID)
	input.Content = strings.TrimSpace(input.Content)
	input.Model = strings.TrimSpace(input.Model)
	input.Role = RoleAgent
	return s.repo.AppendMessage(ctx, input)
}

func (s *Service) CreateUsageRecord(ctx context.Context, input CreateUsageInput) (UsageRecord, error) {
	input.ChatID = strings.TrimSpace(input.ChatID)
	input.MessageID = strings.TrimSpace(input.MessageID)
	input.Provider = strings.TrimSpace(input.Provider)
	input.Model = strings.TrimSpace(input.Model)
	return s.repo.CreateUsageRecord(ctx, input)
}

func (s *Service) CreateSearchAttachment(ctx context.Context, input CreateSearchAttachmentInput) (SearchAttachment, error) {
	input.ChatID = strings.TrimSpace(input.ChatID)
	input.MessageID = strings.TrimSpace(input.MessageID)
	input.Query = strings.TrimSpace(input.Query)
	return s.repo.CreateSearchAttachment(ctx, input)
}

func (s *Service) ListSearchAttachments(ctx context.Context, chatID string, limit int) ([]SearchAttachment, error) {
	return s.repo.ListSearchAttachments(ctx, strings.TrimSpace(chatID), limit)
}

func (s *Service) EditMessage(ctx context.Context, input BranchMessageInput) (Message, error) {
	input.MessageID = strings.TrimSpace(input.MessageID)
	input.Content = strings.TrimSpace(input.Content)
	input.Model = strings.TrimSpace(input.Model)
	if input.Content == "" {
		return Message{}, ErrInvalidInput
	}
	return s.repo.BranchMessage(ctx, input)
}

func (s *Service) Regenerate(ctx context.Context, input BranchMessageInput) (Message, error) {
	input.MessageID = strings.TrimSpace(input.MessageID)
	input.Content = strings.TrimSpace(input.Content)
	input.Model = strings.TrimSpace(input.Model)
	return s.repo.BranchMessage(ctx, input)
}
