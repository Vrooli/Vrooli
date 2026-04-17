// Package testutil provides test utilities for the Agent Inbox API.
package testutil

import (
	"agent-inbox/integrations"
	"context"
	"mime/multipart"
	"sync"
)

// StartAgentChatCall records a call to StartAgentChat.
type StartAgentChatCall struct {
	Message string
	Config  integrations.AgentChatConfig
}

// ContinueChatCall records a call to ContinueChat.
type ContinueChatCall struct {
	RunID         string
	Message       string
	AttachmentIDs []string
}

// GetEventsCall records a call to GetEvents.
type GetEventsCall struct {
	RunID         string
	AfterSequence int64
}

// ListRunsCall records a call to ListRuns.
type ListRunsCall struct {
	Options integrations.ListRunsOptions
}

// MockAgentManagerClient provides a controllable agent-manager client for testing.
type MockAgentManagerClient struct {
	mu sync.Mutex

	// Configurable return values
	StartResult    *integrations.AgentChatSession
	StartError     error
	ContinueErr    error
	EventsResult   []*integrations.TranslatedEvent
	EventsError    error
	StatusResult   *integrations.AgentRunStatus
	StatusError    error
	StopError      error
	ListRunsResult *integrations.ListRunsResult
	ListRunsError  error
	UploadResult   *integrations.UploadResponse
	UploadError    error

	// Call tracking
	StartCalls    []StartAgentChatCall
	ContinueCalls []ContinueChatCall
	EventsCalls   []GetEventsCall
	StatusCalls   []string // runIDs
	StopCalls     []string // runIDs
	ListRunsCalls []ListRunsCall
	UploadCalls   int

	// Custom func hooks (override default behavior when set)
	StartFunc    func(ctx context.Context, message string, cfg integrations.AgentChatConfig) (*integrations.AgentChatSession, error)
	ContinueFunc func(ctx context.Context, runID, message string, attachmentIDs []string) error
	EventsFunc   func(ctx context.Context, runID string, afterSequence int64) ([]*integrations.TranslatedEvent, error)
	StatusFunc   func(ctx context.Context, runID string) (*integrations.AgentRunStatus, error)
	StopFunc     func(ctx context.Context, runID string) error
	ListRunsFunc func(ctx context.Context, opts integrations.ListRunsOptions) (*integrations.ListRunsResult, error)
	UploadFunc   func(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*integrations.UploadResponse, error)
}

// StartAgentChat implements AgentManagerClientInterface.
func (m *MockAgentManagerClient) StartAgentChat(ctx context.Context, message string, cfg integrations.AgentChatConfig) (*integrations.AgentChatSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StartCalls = append(m.StartCalls, StartAgentChatCall{Message: message, Config: cfg})
	if m.StartFunc != nil {
		return m.StartFunc(ctx, message, cfg)
	}
	return m.StartResult, m.StartError
}

// ContinueChat implements AgentManagerClientInterface.
func (m *MockAgentManagerClient) ContinueChat(ctx context.Context, runID, message string, attachmentIDs []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ContinueCalls = append(m.ContinueCalls, ContinueChatCall{RunID: runID, Message: message, AttachmentIDs: attachmentIDs})
	if m.ContinueFunc != nil {
		return m.ContinueFunc(ctx, runID, message, attachmentIDs)
	}
	return m.ContinueErr
}

// GetEvents implements AgentManagerClientInterface.
func (m *MockAgentManagerClient) GetEvents(ctx context.Context, runID string, afterSequence int64) ([]*integrations.TranslatedEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.EventsCalls = append(m.EventsCalls, GetEventsCall{RunID: runID, AfterSequence: afterSequence})
	if m.EventsFunc != nil {
		return m.EventsFunc(ctx, runID, afterSequence)
	}
	return m.EventsResult, m.EventsError
}

// GetRunStatus implements AgentManagerClientInterface.
func (m *MockAgentManagerClient) GetRunStatus(ctx context.Context, runID string) (*integrations.AgentRunStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StatusCalls = append(m.StatusCalls, runID)
	if m.StatusFunc != nil {
		return m.StatusFunc(ctx, runID)
	}
	return m.StatusResult, m.StatusError
}

// StopRun implements AgentManagerClientInterface.
func (m *MockAgentManagerClient) StopRun(ctx context.Context, runID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StopCalls = append(m.StopCalls, runID)
	if m.StopFunc != nil {
		return m.StopFunc(ctx, runID)
	}
	return m.StopError
}

// ListRuns implements AgentManagerClientInterface.
func (m *MockAgentManagerClient) ListRuns(ctx context.Context, opts integrations.ListRunsOptions) (*integrations.ListRunsResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ListRunsCalls = append(m.ListRunsCalls, ListRunsCall{Options: opts})
	if m.ListRunsFunc != nil {
		return m.ListRunsFunc(ctx, opts)
	}
	return m.ListRunsResult, m.ListRunsError
}

// UploadAttachment implements AgentManagerClientInterface.
func (m *MockAgentManagerClient) UploadAttachment(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*integrations.UploadResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UploadCalls++
	if m.UploadFunc != nil {
		return m.UploadFunc(ctx, file, header)
	}
	return m.UploadResult, m.UploadError
}

// Reset clears all recorded calls and return values.
func (m *MockAgentManagerClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StartCalls = nil
	m.ContinueCalls = nil
	m.EventsCalls = nil
	m.StatusCalls = nil
	m.StopCalls = nil
	m.ListRunsCalls = nil
	m.UploadCalls = 0
}
