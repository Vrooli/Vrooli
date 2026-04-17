package services

import (
	"agent-inbox/domain"
	"context"
	"strconv"
	"sync"
)

// =============================================================================
// Mock Repository
// =============================================================================

// mockCompletionRepository implements CompletionRepository for testing.
type mockCompletionRepository struct {
	mu sync.Mutex

	// Data stores
	chatSettings       map[string]*repoMockChatSettings
	messages           map[string][]*domain.Message // chatID -> messages
	toolCallRecords    map[string]*domain.ToolCallRecord
	attachments        map[string]*domain.Attachment
	messageAttachments map[string][]string // messageID -> attachment IDs
	activeLeafs        map[string]string   // chatID -> messageID

	// Call tracking
	saveAssistantMessageCalls    []saveAssistantMessageCall
	saveToolResponseMessageCalls []saveToolResponseMessageCall
	saveToolCallRecordCalls      []saveToolCallRecordCall
	updateToolCallStatusCalls    []updateToolCallStatusCall
	setActiveLeafCalls           []setActiveLeafCall

	// Error injection
	getMessagesError             error
	saveAssistantMessageError    error
	saveToolResponseMessageError error
	saveToolCallRecordError      error
	getToolCallByIDError         error
	updateToolCallStatusError    error
	getPendingApprovalsError     error

	// Auto-increment message ID
	nextMessageID int
}

type repoMockChatSettings struct {
	Model            string
	ToolsEnabled     bool
	WebSearchEnabled bool
}

type saveAssistantMessageCall struct {
	ChatID    string
	Model     string
	Content   string
	ToolCalls []domain.ToolCall
}

type saveToolResponseMessageCall struct {
	ChatID     string
	ToolCallID string
	Result     string
	ParentID   string
}

type saveToolCallRecordCall struct {
	MessageID string
	Record    *domain.ToolCallRecord
}

type updateToolCallStatusCall struct {
	ID           string
	Status       string
	ErrorMessage string
}

type setActiveLeafCall struct {
	ChatID    string
	MessageID string
}

func newMockCompletionRepository() *mockCompletionRepository {
	return &mockCompletionRepository{
		chatSettings:       make(map[string]*repoMockChatSettings),
		messages:           make(map[string][]*domain.Message),
		toolCallRecords:    make(map[string]*domain.ToolCallRecord),
		attachments:        make(map[string]*domain.Attachment),
		messageAttachments: make(map[string][]string),
		activeLeafs:        make(map[string]string),
		nextMessageID:      1,
	}
}

func (m *mockCompletionRepository) GetChatSettingsWithWebSearch(ctx context.Context, chatID string) (string, bool, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	settings, ok := m.chatSettings[chatID]
	if !ok {
		return "", false, false, nil
	}
	return settings.Model, settings.ToolsEnabled, settings.WebSearchEnabled, nil
}

func (m *mockCompletionRepository) UpdateChatPreview(ctx context.Context, chatID, preview string, markUnread bool) error {
	return nil // No-op for tests
}

func (m *mockCompletionRepository) GetMessagesForCompletion(ctx context.Context, chatID string) ([]domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.getMessagesError != nil {
		return nil, m.getMessagesError
	}

	messages := m.messages[chatID]
	result := make([]domain.Message, len(messages))
	for i, msg := range messages {
		result[i] = *msg
	}
	return result, nil
}

func (m *mockCompletionRepository) generateMessageID() string {
	id := m.nextMessageID
	m.nextMessageID++
	return "msg-" + strconv.Itoa(id)
}

// SetupChat configures a chat in the mock repository.
func (m *mockCompletionRepository) SetupChat(chatID string, settings *repoMockChatSettings) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chatSettings[chatID] = settings
}

// AddMessage adds a message to a chat.
func (m *mockCompletionRepository) AddMessage(chatID string, msg *domain.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages[chatID] = append(m.messages[chatID], msg)
}

// AddToolCallRecord adds a tool call record directly.
func (m *mockCompletionRepository) AddToolCallRecord(record *domain.ToolCallRecord) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.toolCallRecords[record.ID] = record
}
