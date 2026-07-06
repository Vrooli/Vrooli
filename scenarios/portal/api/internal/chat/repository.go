package chat

import "context"

type Repository interface {
	ListChats(ctx context.Context, input SearchInput) ([]Chat, []ChatGroup, error)
	CreateChat(ctx context.Context, input CreateChatInput) (Chat, error)
	GetChat(ctx context.Context, id string) (Chat, error)
	UpdateChat(ctx context.Context, input UpdateChatInput) (Chat, error)
	DeleteChat(ctx context.Context, id string) (bool, error)

	ListGroups(ctx context.Context) ([]ChatGroup, error)
	CreateGroup(ctx context.Context, input CreateGroupInput) (ChatGroup, error)
	UpdateGroup(ctx context.Context, input UpdateGroupInput) (ChatGroup, error)
	DeleteGroup(ctx context.Context, id string) (bool, error)

	ListMessages(ctx context.Context, chatID string) ([]Message, string, error)
	SendMessage(ctx context.Context, input SendMessageInput) (Message, error)
	AppendMessage(ctx context.Context, input SendMessageInput) (Message, error)
	BranchMessage(ctx context.Context, input BranchMessageInput) (Message, error)
	CreateUsageRecord(ctx context.Context, input CreateUsageInput) (UsageRecord, error)
	CreateSearchAttachment(ctx context.Context, input CreateSearchAttachmentInput) (SearchAttachment, error)
	ListSearchAttachments(ctx context.Context, chatID string, limit int) ([]SearchAttachment, error)
}
