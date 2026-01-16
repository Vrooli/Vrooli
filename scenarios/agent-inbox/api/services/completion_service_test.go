package services

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"agent-inbox/domain"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// =============================================================================
// Mock Repository
// =============================================================================

// mockCompletionRepository implements CompletionRepository for testing.
type mockCompletionRepository struct {
	mu sync.Mutex

	// Data stores
	chatSettings      map[string]*repoMockChatSettings
	messages          map[string][]*domain.Message // chatID -> messages
	toolCallRecords   map[string]*domain.ToolCallRecord
	attachments       map[string]*domain.Attachment
	messageAttachments map[string][]string // messageID -> attachment IDs
	activeLeafs       map[string]string    // chatID -> messageID

	// Call tracking
	saveAssistantMessageCalls       []saveAssistantMessageCall
	saveToolResponseMessageCalls    []saveToolResponseMessageCall
	saveToolCallRecordCalls         []saveToolCallRecordCall
	updateToolCallStatusCalls       []updateToolCallStatusCall
	setActiveLeafCalls              []setActiveLeafCall

	// Error injection
	getMessagesError            error
	saveAssistantMessageError   error
	saveToolResponseMessageError error
	saveToolCallRecordError     error
	getToolCallByIDError        error
	updateToolCallStatusError   error
	getPendingApprovalsError    error

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

func (m *mockCompletionRepository) SaveAssistantMessage(ctx context.Context, chatID, model, content string, tokenCount int, parentMessageID string) (*domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.saveAssistantMessageError != nil {
		return nil, m.saveAssistantMessageError
	}

	m.saveAssistantMessageCalls = append(m.saveAssistantMessageCalls, saveAssistantMessageCall{
		ChatID:  chatID,
		Model:   model,
		Content: content,
	})

	msg := &domain.Message{
		ID:              m.generateMessageID(),
		ChatID:          chatID,
		Role:            "assistant",
		Content:         content,
		Model:           model,
		ParentMessageID: parentMessageID,
	}
	m.messages[chatID] = append(m.messages[chatID], msg)
	return msg, nil
}

func (m *mockCompletionRepository) SaveAssistantMessageWithToolCalls(ctx context.Context, chatID, model, content string, toolCalls []domain.ToolCall, responseID, finishReason string, tokenCount int, parentMessageID string) (*domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.saveAssistantMessageError != nil {
		return nil, m.saveAssistantMessageError
	}

	m.saveAssistantMessageCalls = append(m.saveAssistantMessageCalls, saveAssistantMessageCall{
		ChatID:    chatID,
		Model:     model,
		Content:   content,
		ToolCalls: toolCalls,
	})

	msg := &domain.Message{
		ID:              m.generateMessageID(),
		ChatID:          chatID,
		Role:            "assistant",
		Content:         content,
		Model:           model,
		ParentMessageID: parentMessageID,
		ToolCalls:       toolCalls,
		FinishReason:    finishReason,
	}
	m.messages[chatID] = append(m.messages[chatID], msg)
	return msg, nil
}

func (m *mockCompletionRepository) SaveToolResponseMessage(ctx context.Context, chatID, toolCallID, result string, parentMessageID string) (*domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.saveToolResponseMessageError != nil {
		return nil, m.saveToolResponseMessageError
	}

	m.saveToolResponseMessageCalls = append(m.saveToolResponseMessageCalls, saveToolResponseMessageCall{
		ChatID:     chatID,
		ToolCallID: toolCallID,
		Result:     result,
		ParentID:   parentMessageID,
	})

	msg := &domain.Message{
		ID:              m.generateMessageID(),
		ChatID:          chatID,
		Role:            "tool",
		Content:         result,
		ToolCallID:      toolCallID,
		ParentMessageID: parentMessageID,
	}
	m.messages[chatID] = append(m.messages[chatID], msg)
	return msg, nil
}

func (m *mockCompletionRepository) SetActiveLeaf(ctx context.Context, chatID, messageID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.setActiveLeafCalls = append(m.setActiveLeafCalls, setActiveLeafCall{
		ChatID:    chatID,
		MessageID: messageID,
	})
	m.activeLeafs[chatID] = messageID
	return nil
}

func (m *mockCompletionRepository) GetAttachmentsForMessages(ctx context.Context, messageIDs []string) (map[string][]domain.Attachment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make(map[string][]domain.Attachment)
	for _, msgID := range messageIDs {
		attIDs := m.messageAttachments[msgID]
		for _, attID := range attIDs {
			if att := m.attachments[attID]; att != nil {
				result[msgID] = append(result[msgID], *att)
			}
		}
	}
	return result, nil
}

func (m *mockCompletionRepository) CreateAttachment(ctx context.Context, att *domain.Attachment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attachments[att.ID] = att
	return nil
}

func (m *mockCompletionRepository) AttachToMessage(ctx context.Context, attachmentID, messageID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messageAttachments[messageID] = append(m.messageAttachments[messageID], attachmentID)
	return nil
}

func (m *mockCompletionRepository) SaveToolCallRecord(ctx context.Context, messageID string, record *domain.ToolCallRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.saveToolCallRecordError != nil {
		return m.saveToolCallRecordError
	}

	m.saveToolCallRecordCalls = append(m.saveToolCallRecordCalls, saveToolCallRecordCall{
		MessageID: messageID,
		Record:    record,
	})
	m.toolCallRecords[record.ID] = record
	return nil
}

func (m *mockCompletionRepository) GetToolCallByID(ctx context.Context, toolCallID string) (*domain.ToolCallRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.getToolCallByIDError != nil {
		return nil, m.getToolCallByIDError
	}
	return m.toolCallRecords[toolCallID], nil
}

func (m *mockCompletionRepository) UpdateToolCallStatus(ctx context.Context, id, status, errorMessage string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.updateToolCallStatusError != nil {
		return m.updateToolCallStatusError
	}

	m.updateToolCallStatusCalls = append(m.updateToolCallStatusCalls, updateToolCallStatusCall{
		ID:           id,
		Status:       status,
		ErrorMessage: errorMessage,
	})

	if record := m.toolCallRecords[id]; record != nil {
		record.Status = status
		record.ErrorMessage = errorMessage
	}
	return nil
}

func (m *mockCompletionRepository) GetPendingApprovals(ctx context.Context, chatID string) ([]*domain.ToolCallRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.getPendingApprovalsError != nil {
		return nil, m.getPendingApprovalsError
	}

	var pending []*domain.ToolCallRecord
	for _, record := range m.toolCallRecords {
		if record.ChatID == chatID && record.Status == domain.StatusPendingApproval {
			pending = append(pending, record)
		}
	}
	return pending, nil
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

// =============================================================================
// Mock Tool Executor
// =============================================================================

// mockToolExecutor implements ToolExecutorInterface for testing.
type mockToolExecutor struct {
	mu sync.Mutex

	// Execution behavior
	executeFunc   func(ctx context.Context, chatID, toolCallID, toolName, args string) (*domain.ToolCallRecord, error)
	executeResult *domain.ToolCallRecord
	executeError  error

	// Call tracking
	executeCalls []executeToolCall
}

type executeToolCall struct {
	ChatID     string
	ToolCallID string
	ToolName   string
	Arguments  string
}

func newMockToolExecutor() *mockToolExecutor {
	return &mockToolExecutor{}
}

func (m *mockToolExecutor) ExecuteTool(ctx context.Context, chatID, toolCallID, toolName, arguments string) (*domain.ToolCallRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.executeCalls = append(m.executeCalls, executeToolCall{
		ChatID:     chatID,
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Arguments:  arguments,
	})

	if m.executeFunc != nil {
		return m.executeFunc(ctx, chatID, toolCallID, toolName, arguments)
	}

	if m.executeError != nil {
		return &domain.ToolCallRecord{
			ID:           toolCallID,
			ChatID:       chatID,
			ToolName:     toolName,
			Arguments:    arguments,
			Status:       domain.StatusFailed,
			ErrorMessage: m.executeError.Error(),
			StartedAt:    time.Now(),
			CompletedAt:  time.Now(),
		}, m.executeError
	}

	if m.executeResult != nil {
		result := *m.executeResult
		result.ID = toolCallID
		result.ChatID = chatID
		result.ToolName = toolName
		result.Arguments = arguments
		return &result, nil
	}

	// Default success
	return &domain.ToolCallRecord{
		ID:          toolCallID,
		ChatID:      chatID,
		ToolName:    toolName,
		Arguments:   arguments,
		Status:      domain.StatusCompleted,
		Result:      `{"success": true}`,
		StartedAt:   time.Now(),
		CompletedAt: time.Now(),
	}, nil
}

// SetExecuteFunc sets a custom function for execution.
func (m *mockToolExecutor) SetExecuteFunc(fn func(ctx context.Context, chatID, toolCallID, toolName, args string) (*domain.ToolCallRecord, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executeFunc = fn
}

// GetExecuteCalls returns all execute calls for verification.
func (m *mockToolExecutor) GetExecuteCalls() []executeToolCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]executeToolCall, len(m.executeCalls))
	copy(calls, m.executeCalls)
	return calls
}

// =============================================================================
// Mock Async Tracker
// =============================================================================

// mockAsyncTrackerForCompletion implements AsyncTrackerInterface for testing.
type mockAsyncTrackerForCompletion struct {
	mu sync.Mutex

	operations map[string]*AsyncOperation

	// Call tracking
	startTrackingCalls []startTrackingCall

	// Error injection
	startTrackingError error
}

type startTrackingCall struct {
	ToolCallID string
	ChatID     string
	ToolName   string
	Scenario   string
}

func newMockAsyncTrackerForCompletion() *mockAsyncTrackerForCompletion {
	return &mockAsyncTrackerForCompletion{
		operations: make(map[string]*AsyncOperation),
	}
}

func (m *mockAsyncTrackerForCompletion) GetActiveOperations(chatID string) []*AsyncOperation {
	m.mu.Lock()
	defer m.mu.Unlock()

	var active []*AsyncOperation
	for _, op := range m.operations {
		if op.ChatID == chatID && op.CompletedAt == nil {
			active = append(active, op)
		}
	}
	return active
}

func (m *mockAsyncTrackerForCompletion) GetOperation(toolCallID string) *AsyncOperation {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.operations[toolCallID]
}

func (m *mockAsyncTrackerForCompletion) StartTracking(ctx context.Context, toolCallID, chatID, toolName, scenario string, toolResult interface{}, asyncBehavior *toolspb.AsyncBehavior) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.startTrackingCalls = append(m.startTrackingCalls, startTrackingCall{
		ToolCallID: toolCallID,
		ChatID:     chatID,
		ToolName:   toolName,
		Scenario:   scenario,
	})

	if m.startTrackingError != nil {
		return m.startTrackingError
	}

	m.operations[toolCallID] = &AsyncOperation{
		ToolCallID: toolCallID,
		ChatID:     chatID,
		ToolName:   toolName,
		Scenario:   scenario,
		Status:     "running",
		StartedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	return nil
}

// AddOperation adds an operation for testing.
func (m *mockAsyncTrackerForCompletion) AddOperation(op *AsyncOperation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.operations[op.ToolCallID] = op
}

// =============================================================================
// Test Helpers
// =============================================================================

// makeToolCall creates a ToolCall with the given ID, name, and arguments.
// This helper avoids the verbose anonymous struct syntax for Function.
func makeToolCall(id, name, args string) domain.ToolCall {
	tc := domain.ToolCall{
		ID:   id,
		Type: "function",
	}
	tc.Function.Name = name
	tc.Function.Arguments = args
	return tc
}

// =============================================================================
// Tests for ExecuteToolCalls
// =============================================================================

func TestExecuteToolCalls_SingleTool_Success(t *testing.T) {
	repo := newMockCompletionRepository()
	executor := newMockToolExecutor()
	registry := newMockToolRegistry()

	// Add a tool that doesn't require approval
	registry.addTool("test-scenario", createSimpleTool("test_tool", "A test tool"))
	registry.ApprovalRequirements["test_tool"] = false

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo:     repo,
		Executor: executor,
		Registry: registry,
	})

	toolCalls := []domain.ToolCall{
		makeToolCall("tc-1", "test_tool", `{"input": "value"}`),
	}

	outcome, err := svc.ExecuteToolCalls(context.Background(), "chat-1", "msg-1", toolCalls, "parent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify outcome
	if len(outcome.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(outcome.Results))
	}
	if outcome.Results[0].Status != domain.StatusCompleted {
		t.Errorf("expected status %s, got %s", domain.StatusCompleted, outcome.Results[0].Status)
	}
	if outcome.HasPendingApprovals {
		t.Error("expected no pending approvals")
	}

	// Verify executor was called
	calls := executor.GetExecuteCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 executor call, got %d", len(calls))
	}
	if calls[0].ToolName != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got %q", calls[0].ToolName)
	}

	// Verify tool response message was saved
	if len(repo.saveToolResponseMessageCalls) != 1 {
		t.Fatalf("expected 1 tool response save, got %d", len(repo.saveToolResponseMessageCalls))
	}
}

func TestExecuteToolCalls_MultipleTools_Success(t *testing.T) {
	repo := newMockCompletionRepository()
	executor := newMockToolExecutor()
	registry := newMockToolRegistry()

	registry.addTool("scenario", createSimpleTool("tool_a", "Tool A"))
	registry.addTool("scenario", createSimpleTool("tool_b", "Tool B"))
	registry.addTool("scenario", createSimpleTool("tool_c", "Tool C"))

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo:     repo,
		Executor: executor,
		Registry: registry,
	})

	toolCalls := []domain.ToolCall{
		makeToolCall("tc-1", "tool_a", `{}`),
		makeToolCall("tc-2", "tool_b", `{}`),
		makeToolCall("tc-3", "tool_c", `{}`),
	}

	outcome, err := svc.ExecuteToolCalls(context.Background(), "chat-1", "msg-1", toolCalls, "parent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(outcome.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(outcome.Results))
	}

	// Verify all tools executed
	calls := executor.GetExecuteCalls()
	if len(calls) != 3 {
		t.Fatalf("expected 3 executor calls, got %d", len(calls))
	}

	// Verify order preserved
	expectedNames := []string{"tool_a", "tool_b", "tool_c"}
	for i, call := range calls {
		if call.ToolName != expectedNames[i] {
			t.Errorf("call %d: expected %s, got %s", i, expectedNames[i], call.ToolName)
		}
	}
}

func TestExecuteToolCalls_WithApprovalRequired(t *testing.T) {
	repo := newMockCompletionRepository()
	executor := newMockToolExecutor()
	registry := newMockToolRegistry()

	registry.addTool("scenario", createSimpleTool("dangerous_tool", "A dangerous tool"))
	registry.ApprovalRequirements["dangerous_tool"] = true

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo:     repo,
		Executor: executor,
		Registry: registry,
	})

	toolCalls := []domain.ToolCall{
		makeToolCall("tc-1", "dangerous_tool", `{"target": "important"}`),
	}

	outcome, err := svc.ExecuteToolCalls(context.Background(), "chat-1", "msg-1", toolCalls, "parent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify approval is pending
	if !outcome.HasPendingApprovals {
		t.Error("expected pending approvals")
	}
	if len(outcome.PendingApprovals) != 1 {
		t.Fatalf("expected 1 pending approval, got %d", len(outcome.PendingApprovals))
	}

	// Verify executor was NOT called
	if len(executor.GetExecuteCalls()) != 0 {
		t.Error("executor should not be called for tools requiring approval")
	}

	// Verify result indicates pending
	if outcome.Results[0].Status != domain.StatusPendingApproval {
		t.Errorf("expected status %s, got %s", domain.StatusPendingApproval, outcome.Results[0].Status)
	}
}

func TestExecuteToolCalls_MixedApproval(t *testing.T) {
	repo := newMockCompletionRepository()
	executor := newMockToolExecutor()
	registry := newMockToolRegistry()

	registry.addTool("scenario", createSimpleTool("safe_tool", "A safe tool"))
	registry.addTool("scenario", createSimpleTool("dangerous_tool", "A dangerous tool"))
	registry.ApprovalRequirements["safe_tool"] = false
	registry.ApprovalRequirements["dangerous_tool"] = true

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo:     repo,
		Executor: executor,
		Registry: registry,
	})

	toolCalls := []domain.ToolCall{
		makeToolCall("tc-1", "safe_tool", `{}`),
		makeToolCall("tc-2", "dangerous_tool", `{}`),
	}

	outcome, err := svc.ExecuteToolCalls(context.Background(), "chat-1", "msg-1", toolCalls, "parent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify mixed results
	if len(outcome.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(outcome.Results))
	}
	if outcome.Results[0].Status != domain.StatusCompleted {
		t.Errorf("safe tool: expected %s, got %s", domain.StatusCompleted, outcome.Results[0].Status)
	}
	if outcome.Results[1].Status != domain.StatusPendingApproval {
		t.Errorf("dangerous tool: expected %s, got %s", domain.StatusPendingApproval, outcome.Results[1].Status)
	}

	// Verify only safe tool executed
	if len(executor.GetExecuteCalls()) != 1 {
		t.Errorf("expected 1 executor call, got %d", len(executor.GetExecuteCalls()))
	}
}

func TestExecuteToolCalls_ExecutionFailure(t *testing.T) {
	repo := newMockCompletionRepository()
	executor := newMockToolExecutor()
	registry := newMockToolRegistry()

	registry.addTool("scenario", createSimpleTool("failing_tool", "A failing tool"))

	// Make executor return an error
	executor.executeError = errors.New("tool execution failed")

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo:     repo,
		Executor: executor,
		Registry: registry,
	})

	toolCalls := []domain.ToolCall{
		makeToolCall("tc-1", "failing_tool", `{}`),
	}

	outcome, err := svc.ExecuteToolCalls(context.Background(), "chat-1", "msg-1", toolCalls, "parent-1")

	// Should return error but also include the result
	if err == nil {
		t.Error("expected error for failed tool execution")
	}
	if outcome == nil {
		t.Fatal("expected outcome even on error")
	}
	if len(outcome.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(outcome.Results))
	}

	// Result should still be saved
	if len(repo.saveToolResponseMessageCalls) != 1 {
		t.Error("tool response should still be saved on failure")
	}
}

func TestExecuteToolCalls_SkillsInjection(t *testing.T) {
	repo := newMockCompletionRepository()
	executor := newMockToolExecutor()
	registry := newMockToolRegistry()

	registry.addTool("scenario", createSimpleTool("tool_with_skills", "Tool that receives skills"))

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo:     repo,
		Executor: executor,
		Registry: registry,
	})

	// Set skills
	svc.SetSkills([]SkillPayload{
		{
			Key:     "skill1",
			Label:   "Skill 1",
			Content: "Skill content here",
		},
	})

	toolCalls := []domain.ToolCall{
		makeToolCall("tc-1", "tool_with_skills", `{"input": "test"}`),
	}

	_, err := svc.ExecuteToolCalls(context.Background(), "chat-1", "msg-1", toolCalls, "parent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify skills were injected into arguments
	calls := executor.GetExecuteCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(calls[0].Arguments), &args); err != nil {
		t.Fatalf("failed to parse arguments: %v", err)
	}

	if _, ok := args["_context_attachments"]; !ok {
		t.Error("expected _context_attachments in arguments")
	}
}

// =============================================================================
// Tests for SaveCompletionResult
// =============================================================================

func TestSaveCompletionResult_RegularMessage(t *testing.T) {
	repo := newMockCompletionRepository()
	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: repo,
	})

	result := &domain.CompletionResult{
		Content:      "Hello, how can I help you?",
		FinishReason: "stop",
		TokenCount:   10,
	}

	msg, err := svc.SaveCompletionResult(context.Background(), "chat-1", "gpt-4", result, "parent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg == nil {
		t.Fatal("expected message")
	}

	// Verify regular message save was called (not with tool calls)
	if len(repo.saveAssistantMessageCalls) != 1 {
		t.Fatalf("expected 1 save call, got %d", len(repo.saveAssistantMessageCalls))
	}

	call := repo.saveAssistantMessageCalls[0]
	if call.Content != "Hello, how can I help you?" {
		t.Errorf("unexpected content: %s", call.Content)
	}
	if len(call.ToolCalls) != 0 {
		t.Error("expected no tool calls for regular message")
	}
}

func TestSaveCompletionResult_WithToolCalls(t *testing.T) {
	repo := newMockCompletionRepository()
	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: repo,
	})

	result := &domain.CompletionResult{
		Content:      "",
		FinishReason: "tool_calls",
		ToolCalls: []domain.ToolCall{
			makeToolCall("tc-1", "search", `{"query": "test"}`),
		},
	}

	msg, err := svc.SaveCompletionResult(context.Background(), "chat-1", "gpt-4", result, "parent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg == nil {
		t.Fatal("expected message")
	}

	// Verify message with tool calls was saved
	if len(repo.saveAssistantMessageCalls) != 1 {
		t.Fatalf("expected 1 save call, got %d", len(repo.saveAssistantMessageCalls))
	}

	call := repo.saveAssistantMessageCalls[0]
	if len(call.ToolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(call.ToolCalls))
	}
}

func TestSaveCompletionResult_UpdatesActiveLeaf(t *testing.T) {
	repo := newMockCompletionRepository()
	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: repo,
	})

	result := &domain.CompletionResult{
		Content: "Test message",
	}

	msg, _ := svc.SaveCompletionResult(context.Background(), "chat-1", "gpt-4", result, "parent-1")

	// Verify active leaf was updated
	if len(repo.setActiveLeafCalls) != 1 {
		t.Fatalf("expected 1 SetActiveLeaf call, got %d", len(repo.setActiveLeafCalls))
	}

	if repo.setActiveLeafCalls[0].MessageID != msg.ID {
		t.Error("active leaf should be set to new message ID")
	}
}

// =============================================================================
// Tests for ApproveToolCall
// =============================================================================

func TestApproveToolCall_Success(t *testing.T) {
	repo := newMockCompletionRepository()
	executor := newMockToolExecutor()

	// Add a pending approval record
	record := &domain.ToolCallRecord{
		ID:        "tc-pending",
		ChatID:    "chat-1",
		MessageID: "msg-1",
		ToolName:  "dangerous_tool",
		Arguments: `{"target": "important"}`,
		Status:    domain.StatusPendingApproval,
		StartedAt: time.Now(),
	}
	repo.AddToolCallRecord(record)

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo:     repo,
		Executor: executor,
	})

	result, err := svc.ApproveToolCall(context.Background(), "chat-1", "tc-pending")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify tool was executed
	if len(executor.GetExecuteCalls()) != 1 {
		t.Error("expected tool to be executed after approval")
	}

	// Verify status was updated to approved first
	if len(repo.updateToolCallStatusCalls) < 1 {
		t.Fatal("expected status update")
	}
	if repo.updateToolCallStatusCalls[0].Status != domain.StatusApproved {
		t.Errorf("expected status %s, got %s", domain.StatusApproved, repo.updateToolCallStatusCalls[0].Status)
	}

	// Verify result
	if result.ToolResult == nil {
		t.Error("expected tool result")
	}
}

func TestApproveToolCall_NotFound(t *testing.T) {
	repo := newMockCompletionRepository()
	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: repo,
	})

	_, err := svc.ApproveToolCall(context.Background(), "chat-1", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent tool call")
	}
}

func TestApproveToolCall_WrongChat(t *testing.T) {
	repo := newMockCompletionRepository()
	repo.AddToolCallRecord(&domain.ToolCallRecord{
		ID:     "tc-1",
		ChatID: "chat-other", // Different chat
		Status: domain.StatusPendingApproval,
	})

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: repo,
	})

	_, err := svc.ApproveToolCall(context.Background(), "chat-1", "tc-1")
	if err == nil {
		t.Error("expected error for wrong chat")
	}
}

func TestApproveToolCall_NotPending(t *testing.T) {
	repo := newMockCompletionRepository()
	repo.AddToolCallRecord(&domain.ToolCallRecord{
		ID:     "tc-1",
		ChatID: "chat-1",
		Status: domain.StatusCompleted, // Already completed
	})

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: repo,
	})

	_, err := svc.ApproveToolCall(context.Background(), "chat-1", "tc-1")
	if err == nil {
		t.Error("expected error for non-pending tool call")
	}
}

// =============================================================================
// Tests for RejectToolCall
// =============================================================================

func TestRejectToolCall_Success(t *testing.T) {
	repo := newMockCompletionRepository()

	record := &domain.ToolCallRecord{
		ID:        "tc-pending",
		ChatID:    "chat-1",
		MessageID: "msg-1",
		ToolName:  "dangerous_tool",
		Status:    domain.StatusPendingApproval,
	}
	repo.AddToolCallRecord(record)

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: repo,
	})

	err := svc.RejectToolCall(context.Background(), "chat-1", "tc-pending", "Too risky")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify status was updated to rejected
	if len(repo.updateToolCallStatusCalls) != 1 {
		t.Fatal("expected status update")
	}
	if repo.updateToolCallStatusCalls[0].Status != domain.StatusRejected {
		t.Errorf("expected status %s, got %s", domain.StatusRejected, repo.updateToolCallStatusCalls[0].Status)
	}

	// Verify rejection result was saved
	if len(repo.saveToolResponseMessageCalls) != 1 {
		t.Error("expected tool response message for rejection")
	}
}

func TestRejectToolCall_DefaultReason(t *testing.T) {
	repo := newMockCompletionRepository()
	repo.AddToolCallRecord(&domain.ToolCallRecord{
		ID:        "tc-1",
		ChatID:    "chat-1",
		MessageID: "msg-1",
		Status:    domain.StatusPendingApproval,
	})

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: repo,
	})

	err := svc.RejectToolCall(context.Background(), "chat-1", "tc-1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify default error message
	if repo.updateToolCallStatusCalls[0].ErrorMessage != "Rejected by user" {
		t.Errorf("expected default rejection message, got %q", repo.updateToolCallStatusCalls[0].ErrorMessage)
	}
}

// =============================================================================
// Tests for Async Operations
// =============================================================================

func TestExecuteToolCalls_StartsAsyncTracking(t *testing.T) {
	repo := newMockCompletionRepository()
	executor := newMockToolExecutor()
	registry := newMockToolRegistry()
	asyncTracker := newMockAsyncTrackerForCompletion()

	// Create an async tool
	asyncTool := createToolWithMetadata("async_tool", "An async tool", &toolspb.ToolMetadata{
		LongRunning: true,
		AsyncBehavior: &toolspb.AsyncBehavior{
			StatusPolling: &toolspb.StatusPolling{
				StatusTool:          "check_status",
				OperationIdField:    "run_id",
				PollIntervalSeconds: 5,
			},
		},
	})
	registry.addTool("scenario", asyncTool)

	// Make executor return a result with run_id
	executor.SetExecuteFunc(func(ctx context.Context, chatID, toolCallID, toolName, args string) (*domain.ToolCallRecord, error) {
		return &domain.ToolCallRecord{
			ID:       toolCallID,
			ChatID:   chatID,
			ToolName: toolName,
			Status:   domain.StatusCompleted,
			Result:   `{"run_id": "run-123", "status": "started"}`,
		}, nil
	})

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo:         repo,
		Executor:     executor,
		Registry:     registry,
		AsyncTracker: asyncTracker,
	})

	toolCalls := []domain.ToolCall{
		makeToolCall("tc-1", "async_tool", `{}`),
	}

	outcome, err := svc.ExecuteToolCalls(context.Background(), "chat-1", "msg-1", toolCalls, "parent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify async tracking was started
	if !outcome.HasAsyncOperations {
		t.Error("expected HasAsyncOperations to be true")
	}
	if len(outcome.AsyncOperations) != 1 {
		t.Fatalf("expected 1 async operation, got %d", len(outcome.AsyncOperations))
	}

	// Verify tracker was called
	if len(asyncTracker.startTrackingCalls) != 1 {
		t.Fatalf("expected 1 StartTracking call, got %d", len(asyncTracker.startTrackingCalls))
	}
}
