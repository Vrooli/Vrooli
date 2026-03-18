package services

import (
	"context"
	"errors"
	"testing"

	"agent-inbox/domain"
)

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
