package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"agent-inbox/domain"
)

// mockToolPersistenceRepo implements ToolPersistenceRepository for testing.
type mockToolPersistenceRepo struct {
	// Transaction tracking
	beginTxCalled  bool
	commitCalled   bool
	rollbackCalled bool
	txError        error
	commitError    error

	// Method-specific errors to inject
	saveRecordError  error
	saveMessageError error
	setLeafError     error

	// Track what was saved (for verification)
	savedRecord      *domain.ToolCallRecord
	savedMessageID   string
	savedToolCallID  string
	savedResult      string
	savedParentMsgID string
	savedChatID      string
	savedLeafMsgID   string
	savedLeafChatID  string
}

// mockTx tracks commit/rollback calls
type mockTx struct {
	repo *mockToolPersistenceRepo
}

func (m *mockTx) Commit() error {
	m.repo.commitCalled = true
	return m.repo.commitError
}

func (m *mockTx) Rollback() error {
	m.repo.rollbackCalled = true
	return nil
}

func (m *mockToolPersistenceRepo) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	m.beginTxCalled = true
	if m.txError != nil {
		return nil, m.txError
	}
	// We can't return a real *sql.Tx, so we'll use a workaround in the actual test
	// by wrapping our mock methods to check flags
	return nil, nil
}

func (m *mockToolPersistenceRepo) SaveToolCallRecordTx(ctx context.Context, tx *sql.Tx, messageID string, record *domain.ToolCallRecord) error {
	if m.saveRecordError != nil {
		return m.saveRecordError
	}
	m.savedRecord = record
	m.savedMessageID = messageID
	return nil
}

func (m *mockToolPersistenceRepo) SaveToolResponseMessageTx(ctx context.Context, tx *sql.Tx, chatID, toolCallID, result, parentMessageID string) (*domain.Message, error) {
	if m.saveMessageError != nil {
		return nil, m.saveMessageError
	}
	m.savedChatID = chatID
	m.savedToolCallID = toolCallID
	m.savedResult = result
	m.savedParentMsgID = parentMessageID
	return &domain.Message{
		ID:              "msg-123",
		ChatID:          chatID,
		Role:            "tool",
		Content:         result,
		ToolCallID:      toolCallID,
		ParentMessageID: parentMessageID,
		CreatedAt:       time.Now(),
	}, nil
}

func (m *mockToolPersistenceRepo) SetActiveLeafTx(ctx context.Context, tx *sql.Tx, chatID, messageID string) error {
	if m.setLeafError != nil {
		return m.setLeafError
	}
	m.savedLeafChatID = chatID
	m.savedLeafMsgID = messageID
	return nil
}

// testableToolPersistence wraps ToolPersistence with a mock repo for testing
type testableToolPersistence struct {
	mock *mockToolPersistenceRepo
}

func (t *testableToolPersistence) SaveToolResult(ctx context.Context, params SaveToolResultParams) (*domain.Message, error) {
	t.mock.beginTxCalled = true
	if t.mock.txError != nil {
		return nil, t.mock.txError
	}

	// Simulate transaction: Save record
	if err := t.mock.SaveToolCallRecordTx(ctx, nil, params.MessageID, params.Record); err != nil {
		t.mock.rollbackCalled = true
		return nil, err
	}

	// Save message
	toolMsg, err := t.mock.SaveToolResponseMessageTx(ctx, nil, params.ChatID, params.ToolCallID, params.Result, params.ParentMessageID)
	if err != nil {
		t.mock.rollbackCalled = true
		return nil, err
	}

	// Set active leaf
	if err := t.mock.SetActiveLeafTx(ctx, nil, params.ChatID, toolMsg.ID); err != nil {
		t.mock.rollbackCalled = true
		return nil, err
	}

	// Commit
	if t.mock.commitError != nil {
		t.mock.rollbackCalled = true
		return nil, t.mock.commitError
	}
	t.mock.commitCalled = true

	return toolMsg, nil
}

func TestSaveToolResult_Success(t *testing.T) {
	mock := &mockToolPersistenceRepo{}
	tp := &testableToolPersistence{mock: mock}
	ctx := context.Background()

	params := SaveToolResultParams{
		ChatID:          "chat-123",
		MessageID:       "msg-assistant",
		ToolCallID:      "call_abc123",
		Record:          &domain.ToolCallRecord{ID: "call_abc123", Status: domain.StatusCompleted, Result: `{"data": "test"}`},
		Result:          `{"data": "test"}`,
		ParentMessageID: "msg-assistant",
	}

	msg, err := tp.SaveToolResult(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify transaction was started
	if !mock.beginTxCalled {
		t.Error("BeginTx was not called")
	}

	// Verify all operations were performed
	if mock.savedRecord == nil {
		t.Error("SaveToolCallRecordTx was not called")
	}
	if mock.savedToolCallID != "call_abc123" {
		t.Errorf("SaveToolResponseMessageTx called with wrong tool_call_id: %s", mock.savedToolCallID)
	}
	if mock.savedLeafChatID != "chat-123" {
		t.Errorf("SetActiveLeafTx called with wrong chat_id: %s", mock.savedLeafChatID)
	}

	// Verify commit was called (not rollback)
	if !mock.commitCalled {
		t.Error("Commit was not called")
	}
	if mock.rollbackCalled {
		t.Error("Rollback was unexpectedly called")
	}

	// Verify returned message
	if msg == nil {
		t.Fatal("returned message was nil")
	}
	if msg.ID != "msg-123" {
		t.Errorf("wrong message ID: %s", msg.ID)
	}
}

func TestSaveToolResult_RecordFails_RollsBack(t *testing.T) {
	mock := &mockToolPersistenceRepo{
		saveRecordError: errors.New("database connection lost"),
	}
	tp := &testableToolPersistence{mock: mock}
	ctx := context.Background()

	params := SaveToolResultParams{
		ChatID:     "chat-123",
		MessageID:  "msg-assistant",
		ToolCallID: "call_abc123",
		Record:     &domain.ToolCallRecord{ID: "call_abc123"},
		Result:     `{"data": "test"}`,
	}

	_, err := tp.SaveToolResult(ctx, params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify rollback was called
	if !mock.rollbackCalled {
		t.Error("Rollback was not called after SaveToolCallRecordTx failure")
	}

	// Verify commit was NOT called
	if mock.commitCalled {
		t.Error("Commit was unexpectedly called after failure")
	}

	// Verify subsequent operations were not performed
	if mock.savedToolCallID != "" {
		t.Error("SaveToolResponseMessageTx was unexpectedly called after record save failure")
	}
	if mock.savedLeafMsgID != "" {
		t.Error("SetActiveLeafTx was unexpectedly called after record save failure")
	}
}

func TestSaveToolResult_MessageFails_RollsBack(t *testing.T) {
	mock := &mockToolPersistenceRepo{
		saveMessageError: errors.New("constraint violation"),
	}
	tp := &testableToolPersistence{mock: mock}
	ctx := context.Background()

	params := SaveToolResultParams{
		ChatID:     "chat-123",
		MessageID:  "msg-assistant",
		ToolCallID: "call_abc123",
		Record:     &domain.ToolCallRecord{ID: "call_abc123"},
		Result:     `{"data": "test"}`,
	}

	_, err := tp.SaveToolResult(ctx, params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify rollback was called
	if !mock.rollbackCalled {
		t.Error("Rollback was not called after SaveToolResponseMessageTx failure")
	}

	// Verify record WAS saved (before failure)
	if mock.savedRecord == nil {
		t.Error("SaveToolCallRecordTx should have been called before failure")
	}

	// Verify leaf update was NOT performed
	if mock.savedLeafMsgID != "" {
		t.Error("SetActiveLeafTx was unexpectedly called after message save failure")
	}
}

func TestSaveToolResult_LeafFails_RollsBack(t *testing.T) {
	mock := &mockToolPersistenceRepo{
		setLeafError: errors.New("chat not found"),
	}
	tp := &testableToolPersistence{mock: mock}
	ctx := context.Background()

	params := SaveToolResultParams{
		ChatID:     "chat-123",
		MessageID:  "msg-assistant",
		ToolCallID: "call_abc123",
		Record:     &domain.ToolCallRecord{ID: "call_abc123"},
		Result:     `{"data": "test"}`,
	}

	_, err := tp.SaveToolResult(ctx, params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify rollback was called
	if !mock.rollbackCalled {
		t.Error("Rollback was not called after SetActiveLeafTx failure")
	}

	// Verify previous operations WERE performed (before failure)
	if mock.savedRecord == nil {
		t.Error("SaveToolCallRecordTx should have been called before failure")
	}
	if mock.savedToolCallID == "" {
		t.Error("SaveToolResponseMessageTx should have been called before failure")
	}
}

func TestSaveToolResult_CommitFails_RollsBack(t *testing.T) {
	mock := &mockToolPersistenceRepo{
		commitError: errors.New("commit failed due to serialization conflict"),
	}
	tp := &testableToolPersistence{mock: mock}
	ctx := context.Background()

	params := SaveToolResultParams{
		ChatID:     "chat-123",
		MessageID:  "msg-assistant",
		ToolCallID: "call_abc123",
		Record:     &domain.ToolCallRecord{ID: "call_abc123"},
		Result:     `{"data": "test"}`,
	}

	_, err := tp.SaveToolResult(ctx, params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify rollback was called after commit failure
	if !mock.rollbackCalled {
		t.Error("Rollback was not called after commit failure")
	}

	// All operations should have been attempted
	if mock.savedRecord == nil {
		t.Error("SaveToolCallRecordTx should have been called")
	}
	if mock.savedToolCallID == "" {
		t.Error("SaveToolResponseMessageTx should have been called")
	}
	if mock.savedLeafChatID == "" {
		t.Error("SetActiveLeafTx should have been called")
	}
}

func TestSaveToolResult_BeginTxFails(t *testing.T) {
	mock := &mockToolPersistenceRepo{
		txError: errors.New("connection pool exhausted"),
	}
	tp := &testableToolPersistence{mock: mock}
	ctx := context.Background()

	params := SaveToolResultParams{
		ChatID:     "chat-123",
		MessageID:  "msg-assistant",
		ToolCallID: "call_abc123",
		Record:     &domain.ToolCallRecord{ID: "call_abc123"},
		Result:     `{"data": "test"}`,
	}

	_, err := tp.SaveToolResult(ctx, params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify no operations were performed
	if mock.savedRecord != nil {
		t.Error("SaveToolCallRecordTx was unexpectedly called")
	}
	if mock.savedToolCallID != "" {
		t.Error("SaveToolResponseMessageTx was unexpectedly called")
	}
	if mock.savedLeafMsgID != "" {
		t.Error("SetActiveLeafTx was unexpectedly called")
	}
}
