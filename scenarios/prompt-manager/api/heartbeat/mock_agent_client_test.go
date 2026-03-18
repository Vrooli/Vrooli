package heartbeat

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// mockAgentClient is a configurable mock of the AgentClient interface for tests.
type mockAgentClient struct {
	mu sync.Mutex

	// Configurable responses
	healthOK  bool
	healthErr error

	ensureProfileResp *EnsureProfileResponse
	ensureProfileErr  error

	listRunsResp *ListRunsResponse
	listRunsErr  error

	createTaskResp *Task
	createTaskErr  error

	createRunResp *Run
	createRunErr  error

	getRuns   map[string]*Run // keyed by runID
	getRunErr error

	waitRunResp *Run
	waitRunErr  error

	stopRunErr error

	continueRunErr              error
	createInvestigationRunErr   error
	createInvestigationApplyErr error

	getRunEventsData []byte
	getRunEventsErr  error

	// Call tracking
	createTaskCalls    []*Task
	createRunCalls     []*CreateRunRequest
	getRunCalls        []string
	stopRunCalls       []string
	ensureProfileCalls []*EnsureProfileRequest
	listRunsCalls      []ListRunsOptions
}

func newMockAgentClient() *mockAgentClient {
	return &mockAgentClient{
		healthOK: true,
		getRuns:  make(map[string]*Run),
	}
}

func (m *mockAgentClient) WithCreateTaskResponse(task *Task) *mockAgentClient {
	m.createTaskResp = task
	return m
}

func (m *mockAgentClient) WithCreateTaskError(err error) *mockAgentClient {
	m.createTaskErr = err
	return m
}

func (m *mockAgentClient) WithCreateRunResponse(run *Run) *mockAgentClient {
	m.createRunResp = run
	return m
}

func (m *mockAgentClient) WithCreateRunError(err error) *mockAgentClient {
	m.createRunErr = err
	return m
}

func (m *mockAgentClient) WithEnsureProfileResponse(resp *EnsureProfileResponse) *mockAgentClient {
	m.ensureProfileResp = resp
	return m
}

func (m *mockAgentClient) WithEnsureProfileError(err error) *mockAgentClient {
	m.ensureProfileErr = err
	return m
}

func (m *mockAgentClient) WithListRunsResponse(resp *ListRunsResponse) *mockAgentClient {
	m.listRunsResp = resp
	return m
}

func (m *mockAgentClient) WithListRunsError(err error) *mockAgentClient {
	m.listRunsErr = err
	return m
}

func (m *mockAgentClient) WithGetRunResponse(runID string, run *Run) *mockAgentClient {
	m.getRuns[runID] = run
	return m
}

func (m *mockAgentClient) WithWaitRunResponse(run *Run) *mockAgentClient {
	m.waitRunResp = run
	return m
}

func (m *mockAgentClient) WithWaitRunError(err error) *mockAgentClient {
	m.waitRunErr = err
	return m
}

func (m *mockAgentClient) WithStopRunError(err error) *mockAgentClient {
	m.stopRunErr = err
	return m
}

func (m *mockAgentClient) WithContinueRunError(err error) *mockAgentClient {
	m.continueRunErr = err
	return m
}

func (m *mockAgentClient) WithCreateInvestigationRunError(err error) *mockAgentClient {
	m.createInvestigationRunErr = err
	return m
}

func (m *mockAgentClient) WithCreateInvestigationApplyError(err error) *mockAgentClient {
	m.createInvestigationApplyErr = err
	return m
}

func (m *mockAgentClient) WithGetRunEventsData(data []byte) *mockAgentClient {
	m.getRunEventsData = data
	return m
}

func (m *mockAgentClient) WithGetRunEventsError(err error) *mockAgentClient {
	m.getRunEventsErr = err
	return m
}

func (m *mockAgentClient) Health(_ context.Context) (bool, error) {
	return m.healthOK, m.healthErr
}

func (m *mockAgentClient) EnsureProfile(_ context.Context, req *EnsureProfileRequest) (*EnsureProfileResponse, error) {
	m.mu.Lock()
	if req != nil {
		reqCopy := *req
		m.ensureProfileCalls = append(m.ensureProfileCalls, &reqCopy)
	} else {
		m.ensureProfileCalls = append(m.ensureProfileCalls, nil)
	}
	m.mu.Unlock()

	if m.ensureProfileErr != nil {
		return nil, m.ensureProfileErr
	}
	if m.ensureProfileResp != nil {
		return m.ensureProfileResp, nil
	}
	return &EnsureProfileResponse{Created: false}, nil
}

func (m *mockAgentClient) CreateTask(_ context.Context, task *Task) (*Task, error) {
	m.mu.Lock()
	m.createTaskCalls = append(m.createTaskCalls, task)
	m.mu.Unlock()

	if m.createTaskErr != nil {
		return nil, m.createTaskErr
	}
	if m.createTaskResp != nil {
		return m.createTaskResp, nil
	}
	// Default: echo back with an ID
	return &Task{
		ID:          "task-" + task.Title,
		Title:       task.Title,
		Description: task.Description,
		ScopePath:   task.ScopePath,
		ProjectRoot: task.ProjectRoot,
	}, nil
}

func (m *mockAgentClient) CreateRun(_ context.Context, req *CreateRunRequest) (*Run, error) {
	m.mu.Lock()
	m.createRunCalls = append(m.createRunCalls, req)
	m.mu.Unlock()

	if m.createRunErr != nil {
		return nil, m.createRunErr
	}
	if m.createRunResp != nil {
		return m.createRunResp, nil
	}
	return &Run{
		ID:     "run-123",
		TaskID: req.TaskID,
		Status: "RUN_STATUS_RUNNING",
	}, nil
}

func (m *mockAgentClient) GetRun(_ context.Context, runID string) (*Run, error) {
	m.mu.Lock()
	m.getRunCalls = append(m.getRunCalls, runID)
	m.mu.Unlock()

	if m.getRunErr != nil {
		return nil, m.getRunErr
	}
	if run, ok := m.getRuns[runID]; ok {
		return run, nil
	}
	return nil, fmt.Errorf("run %s not found", runID)
}

func (m *mockAgentClient) WaitForRun(_ context.Context, runID string, _ time.Duration) (*Run, error) {
	if m.waitRunErr != nil {
		return nil, m.waitRunErr
	}
	if m.waitRunResp != nil {
		return m.waitRunResp, nil
	}
	// Check getRuns as fallback
	if run, ok := m.getRuns[runID]; ok {
		return run, nil
	}
	return &Run{
		ID:     runID,
		Status: "RUN_STATUS_COMPLETE",
	}, nil
}

func (m *mockAgentClient) StopRun(_ context.Context, runID string) error {
	m.mu.Lock()
	m.stopRunCalls = append(m.stopRunCalls, runID)
	m.mu.Unlock()
	return m.stopRunErr
}

func (m *mockAgentClient) GetRunEvents(_ context.Context, _ string, _ int64, _ int) ([]byte, error) {
	if m.getRunEventsErr != nil {
		return nil, m.getRunEventsErr
	}
	if m.getRunEventsData != nil {
		return m.getRunEventsData, nil
	}
	return []byte("[]"), nil
}

func (m *mockAgentClient) ListRuns(_ context.Context, opts ListRunsOptions) (*ListRunsResponse, error) {
	m.mu.Lock()
	m.listRunsCalls = append(m.listRunsCalls, opts)
	m.mu.Unlock()

	if m.listRunsErr != nil {
		return nil, m.listRunsErr
	}
	if m.listRunsResp != nil {
		return m.listRunsResp, nil
	}
	return &ListRunsResponse{Runs: []*Run{}, Total: 0, HasMore: false}, nil
}

func (m *mockAgentClient) ContinueRun(_ context.Context, _ string, _ string) (*Run, error) {
	if m.continueRunErr != nil {
		return nil, m.continueRunErr
	}
	return &Run{ID: "run-continued", Status: "RUN_STATUS_RUNNING"}, nil
}

func (m *mockAgentClient) CreateInvestigationRun(_ context.Context, _ []string, _ string, _ string) (*Run, error) {
	if m.createInvestigationRunErr != nil {
		return nil, m.createInvestigationRunErr
	}
	return &Run{ID: "run-investigate", Status: "RUN_STATUS_RUNNING"}, nil
}

func (m *mockAgentClient) CreateInvestigationApplyRun(_ context.Context, _ string, _ string) (*Run, error) {
	if m.createInvestigationApplyErr != nil {
		return nil, m.createInvestigationApplyErr
	}
	return &Run{ID: "run-apply", Status: "RUN_STATUS_RUNNING"}, nil
}
