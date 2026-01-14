package agentmanager

import (
	"context"
	"sync"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// MockAgentService is a test double for AgentServiceAPI.
// It allows configuring return values and tracking method calls.
type MockAgentService struct {
	mu sync.Mutex

	// Configuration for return values
	IsAvailableResult bool
	InitializeError   error
	UpdateProfilesErr error
	ResolveURLResult  string
	ResolveURLError   error

	// ExecuteTask configuration
	ExecuteTaskResult *ExecuteResult
	ExecuteTaskError  error

	// ExecuteTaskAsync configuration
	ExecuteTaskAsyncRunID string
	ExecuteTaskAsyncError error

	// ExecuteInsight configuration
	ExecuteInsightResult *ExecuteResult
	ExecuteInsightError  error

	// GetRunStatus configuration
	GetRunStatusResult *domainpb.Run
	GetRunStatusError  error

	// StopRun configuration
	StopRunError error

	// GetRunEvents configuration
	GetRunEventsResult []*domainpb.RunEvent
	GetRunEventsError  error

	// WaitForRun configuration
	WaitForRunResult *domainpb.Run
	WaitForRunError  error

	// Call tracking
	Calls struct {
		IsAvailable      int
		Initialize       int
		UpdateProfiles   int
		ResolveURL       int
		ExecuteTask      int
		ExecuteTaskAsync int
		ExecuteInsight   int
		GetRunStatus     int
		StopRun          int
		GetRunEvents     int
		WaitForRun       int
	}

	// Capture last call arguments
	LastExecuteTaskReq      ExecuteRequest
	LastExecuteTaskAsyncReq ExecuteRequest
	LastInsightReq          InsightRequest
	LastStopRunID           string
	LastGetRunStatusID      string
	LastGetRunEventsID      string
	LastGetRunEventsAfter   int64
	LastWaitForRunID        string
}

// Compile-time assertion that MockAgentService implements AgentServiceAPI.
var _ AgentServiceAPI = (*MockAgentService)(nil)

// NewMockAgentService creates a new mock agent service with sensible defaults.
func NewMockAgentService() *MockAgentService {
	return &MockAgentService{
		IsAvailableResult: true,
		ExecuteTaskResult: &ExecuteResult{
			RunID:   "mock-run-id",
			Success: true,
		},
		ExecuteTaskAsyncRunID: "mock-async-run-id",
		ExecuteInsightResult: &ExecuteResult{
			RunID:   "mock-insight-run-id",
			Success: true,
		},
		GetRunStatusResult: &domainpb.Run{
			Id:     "mock-run-id",
			Status: domainpb.RunStatus_RUN_STATUS_COMPLETE,
		},
		WaitForRunResult: &domainpb.Run{
			Id:     "mock-run-id",
			Status: domainpb.RunStatus_RUN_STATUS_COMPLETE,
		},
	}
}

func (m *MockAgentService) IsAvailable(ctx context.Context) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.IsAvailable++
	return m.IsAvailableResult
}

func (m *MockAgentService) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Initialize++
	return m.InitializeError
}

func (m *MockAgentService) UpdateProfiles(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.UpdateProfiles++
	return m.UpdateProfilesErr
}

func (m *MockAgentService) ResolveURL(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.ResolveURL++
	return m.ResolveURLResult, m.ResolveURLError
}

func (m *MockAgentService) ExecuteTask(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.ExecuteTask++
	m.LastExecuteTaskReq = req
	return m.ExecuteTaskResult, m.ExecuteTaskError
}

func (m *MockAgentService) ExecuteTaskAsync(ctx context.Context, req ExecuteRequest) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.ExecuteTaskAsync++
	m.LastExecuteTaskAsyncReq = req
	return m.ExecuteTaskAsyncRunID, m.ExecuteTaskAsyncError
}

func (m *MockAgentService) ExecuteInsight(ctx context.Context, req InsightRequest) (*ExecuteResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.ExecuteInsight++
	m.LastInsightReq = req
	return m.ExecuteInsightResult, m.ExecuteInsightError
}

func (m *MockAgentService) GetRunStatus(ctx context.Context, runID string) (*domainpb.Run, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.GetRunStatus++
	m.LastGetRunStatusID = runID
	return m.GetRunStatusResult, m.GetRunStatusError
}

func (m *MockAgentService) StopRun(ctx context.Context, runID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.StopRun++
	m.LastStopRunID = runID
	return m.StopRunError
}

func (m *MockAgentService) GetRunEvents(ctx context.Context, runID string, afterSequence int64) ([]*domainpb.RunEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.GetRunEvents++
	m.LastGetRunEventsID = runID
	m.LastGetRunEventsAfter = afterSequence
	return m.GetRunEventsResult, m.GetRunEventsError
}

func (m *MockAgentService) WaitForRun(ctx context.Context, runID string) (*domainpb.Run, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.WaitForRun++
	m.LastWaitForRunID = runID
	return m.WaitForRunResult, m.WaitForRunError
}

// Reset clears all call counts and captured arguments.
func (m *MockAgentService) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = struct {
		IsAvailable      int
		Initialize       int
		UpdateProfiles   int
		ResolveURL       int
		ExecuteTask      int
		ExecuteTaskAsync int
		ExecuteInsight   int
		GetRunStatus     int
		StopRun          int
		GetRunEvents     int
		WaitForRun       int
	}{}
	m.LastExecuteTaskReq = ExecuteRequest{}
	m.LastExecuteTaskAsyncReq = ExecuteRequest{}
	m.LastInsightReq = InsightRequest{}
	m.LastStopRunID = ""
	m.LastGetRunStatusID = ""
	m.LastGetRunEventsID = ""
	m.LastGetRunEventsAfter = 0
	m.LastWaitForRunID = ""
}
