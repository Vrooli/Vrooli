package handlers

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/minio/minio-go/v7"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	autosession "github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/domain"
	"github.com/vrooli/browser-automation-studio/services/export"
	livecapture "github.com/vrooli/browser-automation-studio/services/live-capture"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	"github.com/vrooli/browser-automation-studio/storage"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	basprojects "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

// ============================================================================
// Mock CatalogService
// ============================================================================

// MockCatalogService is a test mock for workflow.CatalogService
type MockCatalogService struct {
	mu sync.RWMutex

	// Storage
	projects  map[uuid.UUID]*database.ProjectIndex
	workflows map[uuid.UUID]*database.WorkflowIndex

	// Error injection for testing error paths
	CheckHealthError           error
	CheckAutomationHealthError error
	CreateProjectError         error
	GetProjectError            error
	GetProjectByNameError      error
	GetProjectByFolderPathError error
	UpdateProjectError         error
	DeleteProjectError         error
	ListProjectsError          error
	GetProjectStatsError       error
	GetProjectsStatsError      error
	ListWorkflowsByProjectError error
	DeleteProjectWorkflowsError error
	CreateWorkflowError        error
	GetWorkflowAPIError        error
	UpdateWorkflowError        error
	DeleteWorkflowError        error
	ListWorkflowsError         error
	SyncProjectWorkflowsError  error

	// Health check responses
	AutomationHealthy bool

	// Call tracking
	CreateProjectCalled        bool
	DeleteProjectCalled        bool
	SyncProjectWorkflowsCalled bool
}

func NewMockCatalogService() *MockCatalogService {
	return &MockCatalogService{
		projects:          make(map[uuid.UUID]*database.ProjectIndex),
		workflows:         make(map[uuid.UUID]*database.WorkflowIndex),
		AutomationHealthy: true,
	}
}

// Compile-time interface check
var _ workflow.CatalogService = (*MockCatalogService)(nil)

func (m *MockCatalogService) CheckHealth() string {
	return "healthy"
}

func (m *MockCatalogService) CheckAutomationHealth(ctx context.Context) (bool, error) {
	if m.CheckAutomationHealthError != nil {
		return false, m.CheckAutomationHealthError
	}
	return m.AutomationHealthy, nil
}

func (m *MockCatalogService) CreateProject(ctx context.Context, project *database.ProjectIndex, description string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CreateProjectCalled = true

	if m.CreateProjectError != nil {
		return m.CreateProjectError
	}

	if project.ID == uuid.Nil {
		project.ID = uuid.New()
	}
	copy := *project
	m.projects[project.ID] = &copy
	return nil
}

func (m *MockCatalogService) GetProject(ctx context.Context, id uuid.UUID) (*database.ProjectIndex, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetProjectError != nil {
		return nil, m.GetProjectError
	}

	project, ok := m.projects[id]
	if !ok {
		return nil, database.ErrNotFound
	}
	copy := *project
	return &copy, nil
}

func (m *MockCatalogService) GetProjectByName(ctx context.Context, name string) (*database.ProjectIndex, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetProjectByNameError != nil {
		return nil, m.GetProjectByNameError
	}

	for _, project := range m.projects {
		if project.Name == name {
			copy := *project
			return &copy, nil
		}
	}
	return nil, database.ErrNotFound
}

func (m *MockCatalogService) GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.ProjectIndex, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetProjectByFolderPathError != nil {
		return nil, m.GetProjectByFolderPathError
	}

	for _, project := range m.projects {
		if project.FolderPath == folderPath {
			copy := *project
			return &copy, nil
		}
	}
	return nil, database.ErrNotFound
}

func (m *MockCatalogService) UpdateProject(ctx context.Context, project *database.ProjectIndex, description string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.UpdateProjectError != nil {
		return m.UpdateProjectError
	}

	if _, ok := m.projects[project.ID]; !ok {
		return database.ErrNotFound
	}
	copy := *project
	m.projects[project.ID] = &copy
	return nil
}

func (m *MockCatalogService) DeleteProject(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DeleteProjectCalled = true

	if m.DeleteProjectError != nil {
		return m.DeleteProjectError
	}

	delete(m.projects, id)
	return nil
}

func (m *MockCatalogService) ListProjects(ctx context.Context, limit, offset int) ([]*database.ProjectIndex, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.ListProjectsError != nil {
		return nil, m.ListProjectsError
	}

	projects := make([]*database.ProjectIndex, 0, len(m.projects))
	for _, p := range m.projects {
		copy := *p
		projects = append(projects, &copy)
	}
	return projects, nil
}

func (m *MockCatalogService) GetProjectStats(ctx context.Context, projectID uuid.UUID) (*database.ProjectStats, error) {
	if m.GetProjectStatsError != nil {
		return nil, m.GetProjectStatsError
	}
	return &database.ProjectStats{ProjectID: projectID}, nil
}

func (m *MockCatalogService) GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]*database.ProjectStats, error) {
	if m.GetProjectsStatsError != nil {
		return nil, m.GetProjectsStatsError
	}

	result := make(map[uuid.UUID]*database.ProjectStats, len(projectIDs))
	for _, id := range projectIDs {
		result[id] = &database.ProjectStats{ProjectID: id}
	}
	return result, nil
}

func (m *MockCatalogService) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.WorkflowIndex, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.ListWorkflowsByProjectError != nil {
		return nil, m.ListWorkflowsByProjectError
	}

	workflows := make([]*database.WorkflowIndex, 0)
	for _, w := range m.workflows {
		if w.ProjectID != nil && *w.ProjectID == projectID {
			copy := *w
			workflows = append(workflows, &copy)
		}
	}
	return workflows, nil
}

func (m *MockCatalogService) DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error {
	if m.DeleteProjectWorkflowsError != nil {
		return m.DeleteProjectWorkflowsError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, id := range workflowIDs {
		delete(m.workflows, id)
	}
	return nil
}

func (m *MockCatalogService) EnsureSeedProject(ctx context.Context) (*database.ProjectIndex, error) {
	return &database.ProjectIndex{
		ID:         uuid.New(),
		Name:       "Demo Project",
		FolderPath: "/tmp/demo",
	}, nil
}

func (m *MockCatalogService) HydrateProject(ctx context.Context, project *database.ProjectIndex) (*basprojects.Project, error) {
	return &basprojects.Project{
		Id:          project.ID.String(),
		Name:        project.Name,
		Description: "",
		FolderPath:  project.FolderPath,
	}, nil
}

func (m *MockCatalogService) CreateWorkflow(ctx context.Context, req *basapi.CreateWorkflowRequest) (*basapi.CreateWorkflowResponse, error) {
	if m.CreateWorkflowError != nil {
		return nil, m.CreateWorkflowError
	}
	return &basapi.CreateWorkflowResponse{
		Workflow: &basapi.WorkflowSummary{
			Id:   uuid.New().String(),
			Name: req.GetName(),
		},
	}, nil
}

func (m *MockCatalogService) GetWorkflowAPI(ctx context.Context, req *basapi.GetWorkflowRequest) (*basapi.GetWorkflowResponse, error) {
	if m.GetWorkflowAPIError != nil {
		return nil, m.GetWorkflowAPIError
	}
	return &basapi.GetWorkflowResponse{}, nil
}

func (m *MockCatalogService) UpdateWorkflow(ctx context.Context, req *basapi.UpdateWorkflowRequest) (*basapi.UpdateWorkflowResponse, error) {
	if m.UpdateWorkflowError != nil {
		return nil, m.UpdateWorkflowError
	}
	return &basapi.UpdateWorkflowResponse{}, nil
}

func (m *MockCatalogService) DeleteWorkflow(ctx context.Context, req *basapi.DeleteWorkflowRequest) (*basapi.DeleteWorkflowResponse, error) {
	if m.DeleteWorkflowError != nil {
		return nil, m.DeleteWorkflowError
	}
	return &basapi.DeleteWorkflowResponse{}, nil
}

func (m *MockCatalogService) ListWorkflows(ctx context.Context, req *basapi.ListWorkflowsRequest) (*basapi.ListWorkflowsResponse, error) {
	if m.ListWorkflowsError != nil {
		return nil, m.ListWorkflowsError
	}
	return &basapi.ListWorkflowsResponse{
		Workflows: []*basapi.WorkflowSummary{},
	}, nil
}

func (m *MockCatalogService) GetWorkflow(ctx context.Context, workflowID uuid.UUID) (*basapi.WorkflowSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	workflow, ok := m.workflows[workflowID]
	if !ok {
		return nil, database.ErrNotFound
	}
	return &basapi.WorkflowSummary{
		Id:   workflow.ID.String(),
		Name: workflow.Name,
	}, nil
}

func (m *MockCatalogService) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*basapi.WorkflowSummary, error) {
	return m.GetWorkflow(ctx, workflowID)
}

func (m *MockCatalogService) GetWorkflowByProjectPath(ctx context.Context, callingWorkflowID uuid.UUID, workflowPath string, projectRoot string) (*basapi.WorkflowSummary, error) {
	return nil, database.ErrNotFound
}

func (m *MockCatalogService) ListWorkflowVersionsAPI(ctx context.Context, workflowID uuid.UUID) (*basapi.WorkflowVersionList, error) {
	return &basapi.WorkflowVersionList{}, nil
}

func (m *MockCatalogService) GetWorkflowVersionAPI(ctx context.Context, workflowID uuid.UUID, version int32) (*basapi.WorkflowVersion, error) {
	return &basapi.WorkflowVersion{}, nil
}

func (m *MockCatalogService) RestoreWorkflowVersionAPI(ctx context.Context, workflowID uuid.UUID, version int32, changeDescription string) (*basapi.RestoreWorkflowVersionResponse, error) {
	return &basapi.RestoreWorkflowVersionResponse{}, nil
}

func (m *MockCatalogService) SyncProjectWorkflows(ctx context.Context, projectID uuid.UUID) error {
	m.SyncProjectWorkflowsCalled = true
	return m.SyncProjectWorkflowsError
}

func (m *MockCatalogService) ModifyWorkflowAPI(ctx context.Context, workflowID uuid.UUID, prompt string, current *basworkflows.WorkflowDefinitionV2) (*basapi.UpdateWorkflowResponse, error) {
	return &basapi.UpdateWorkflowResponse{}, nil
}

// Test helper methods
func (m *MockCatalogService) AddProject(project *database.ProjectIndex) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if project.ID == uuid.Nil {
		project.ID = uuid.New()
	}
	copy := *project
	m.projects[project.ID] = &copy
}

func (m *MockCatalogService) AddWorkflow(workflow *database.WorkflowIndex) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if workflow.ID == uuid.Nil {
		workflow.ID = uuid.New()
	}
	copy := *workflow
	m.workflows[workflow.ID] = &copy
}

// ============================================================================
// Mock ExecutionService
// ============================================================================

// MockExecutionService is a test mock for workflow.ExecutionService
type MockExecutionService struct {
	mu sync.RWMutex

	executions map[uuid.UUID]*database.ExecutionIndex

	// Error injection
	ExecuteWorkflowError             error
	ExecuteWorkflowAPIError          error
	ExecuteAdhocWorkflowAPIError     error
	StopExecutionError               error
	ResumeExecutionError             error
	ListExecutionsError              error
	GetExecutionError                error
	UpdateExecutionError             error
	GetExecutionScreenshotsError     error
	GetExecutionTimelineError        error
	GetExecutionTimelineProtoError   error
	DescribeExecutionExportError     error
	ExportToFolderError              error
	HydrateExecutionProtoError       error
	GetExecutionTraceArtifactsError  error
	GetExecutionHarArtifactsError    error
	GetExecutionVideoArtifactsError  error

	// Response overrides
	ExecutionScreenshots      []*basexecution.ExecutionScreenshot
	ExecutionTimeline         *workflow.ExecutionTimeline
	ExecutionTimelineProto    *bastimeline.ExecutionTimeline
	ExecutionTraceArtifacts   []workflow.ExecutionFileArtifact
	ExecutionHarArtifacts     []workflow.ExecutionFileArtifact
	ExecutionVideoArtifacts   []workflow.ExecutionVideoArtifact

	// Call tracking
	StopExecutionCalled   bool
	ResumeExecutionCalled bool
}

func NewMockExecutionService() *MockExecutionService {
	return &MockExecutionService{
		executions: make(map[uuid.UUID]*database.ExecutionIndex),
	}
}

// Compile-time interface check
var _ workflow.ExecutionService = (*MockExecutionService)(nil)

func (m *MockExecutionService) ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.ExecutionIndex, error) {
	if m.ExecuteWorkflowError != nil {
		return nil, m.ExecuteWorkflowError
	}

	execution := &database.ExecutionIndex{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		Status:     "running",
	}

	m.mu.Lock()
	m.executions[execution.ID] = execution
	m.mu.Unlock()

	return execution, nil
}

func (m *MockExecutionService) ExecuteWorkflowAPI(ctx context.Context, req *basapi.ExecuteWorkflowRequest) (*basapi.ExecuteWorkflowResponse, error) {
	return m.ExecuteWorkflowAPIWithOptions(ctx, req, nil)
}

func (m *MockExecutionService) ExecuteWorkflowAPIWithOptions(ctx context.Context, req *basapi.ExecuteWorkflowRequest, opts *workflow.ExecuteOptions) (*basapi.ExecuteWorkflowResponse, error) {
	if m.ExecuteWorkflowAPIError != nil {
		return nil, m.ExecuteWorkflowAPIError
	}

	executionID := uuid.New()
	return &basapi.ExecuteWorkflowResponse{
		ExecutionId: executionID.String(),
		Status:      basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING,
	}, nil
}

func (m *MockExecutionService) ExecuteAdhocWorkflowAPI(ctx context.Context, req *basexecution.ExecuteAdhocRequest) (*basexecution.ExecuteAdhocResponse, error) {
	return m.ExecuteAdhocWorkflowAPIWithOptions(ctx, req, nil)
}

func (m *MockExecutionService) ExecuteAdhocWorkflowAPIWithOptions(ctx context.Context, req *basexecution.ExecuteAdhocRequest, opts *workflow.ExecuteOptions) (*basexecution.ExecuteAdhocResponse, error) {
	if m.ExecuteAdhocWorkflowAPIError != nil {
		return nil, m.ExecuteAdhocWorkflowAPIError
	}

	executionID := uuid.New()
	return &basexecution.ExecuteAdhocResponse{
		ExecutionId: executionID.String(),
		Status:      basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING,
	}, nil
}

func (m *MockExecutionService) StopExecution(ctx context.Context, executionID uuid.UUID) error {
	m.StopExecutionCalled = true

	if m.StopExecutionError != nil {
		return m.StopExecutionError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if execution, ok := m.executions[executionID]; ok {
		execution.Status = "stopped"
	}
	return nil
}

func (m *MockExecutionService) ResumeExecution(ctx context.Context, executionID uuid.UUID, parameters map[string]any) (*database.ExecutionIndex, error) {
	m.ResumeExecutionCalled = true

	if m.ResumeExecutionError != nil {
		return nil, m.ResumeExecutionError
	}

	newExecution := &database.ExecutionIndex{
		ID:     uuid.New(),
		Status: "running",
	}

	m.mu.Lock()
	m.executions[newExecution.ID] = newExecution
	m.mu.Unlock()

	return newExecution, nil
}

func (m *MockExecutionService) ListExecutions(ctx context.Context, workflowID *uuid.UUID, projectID *uuid.UUID, limit, offset int) ([]*database.ExecutionIndex, error) {
	if m.ListExecutionsError != nil {
		return nil, m.ListExecutionsError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	executions := make([]*database.ExecutionIndex, 0, len(m.executions))
	for _, e := range m.executions {
		// Note: projectID filtering would require workflow lookup, skip for mock
		if workflowID == nil || e.WorkflowID == *workflowID {
			copy := *e
			executions = append(executions, &copy)
		}
	}
	return executions, nil
}

func (m *MockExecutionService) GetExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	if m.GetExecutionError != nil {
		return nil, m.GetExecutionError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	execution, ok := m.executions[id]
	if !ok {
		return nil, database.ErrNotFound
	}
	copy := *execution
	return &copy, nil
}

func (m *MockExecutionService) UpdateExecution(ctx context.Context, execution *database.ExecutionIndex) error {
	if m.UpdateExecutionError != nil {
		return m.UpdateExecutionError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.executions[execution.ID]; !ok {
		return database.ErrNotFound
	}
	copy := *execution
	m.executions[execution.ID] = &copy
	return nil
}

func (m *MockExecutionService) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*basexecution.ExecutionScreenshot, error) {
	if m.GetExecutionScreenshotsError != nil {
		return nil, m.GetExecutionScreenshotsError
	}

	if m.ExecutionScreenshots != nil {
		return m.ExecutionScreenshots, nil
	}
	return []*basexecution.ExecutionScreenshot{}, nil
}

func (m *MockExecutionService) GetExecutionVideoArtifacts(ctx context.Context, executionID uuid.UUID) ([]workflow.ExecutionVideoArtifact, error) {
	if m.GetExecutionVideoArtifactsError != nil {
		return nil, m.GetExecutionVideoArtifactsError
	}
	if m.ExecutionVideoArtifacts != nil {
		return m.ExecutionVideoArtifacts, nil
	}
	return []workflow.ExecutionVideoArtifact{}, nil
}

func (m *MockExecutionService) GetExecutionTraceArtifacts(ctx context.Context, executionID uuid.UUID) ([]workflow.ExecutionFileArtifact, error) {
	if m.GetExecutionTraceArtifactsError != nil {
		return nil, m.GetExecutionTraceArtifactsError
	}
	if m.ExecutionTraceArtifacts != nil {
		return m.ExecutionTraceArtifacts, nil
	}
	return []workflow.ExecutionFileArtifact{}, nil
}

func (m *MockExecutionService) GetExecutionHarArtifacts(ctx context.Context, executionID uuid.UUID) ([]workflow.ExecutionFileArtifact, error) {
	if m.GetExecutionHarArtifactsError != nil {
		return nil, m.GetExecutionHarArtifactsError
	}
	if m.ExecutionHarArtifacts != nil {
		return m.ExecutionHarArtifacts, nil
	}
	return []workflow.ExecutionFileArtifact{}, nil
}

func (m *MockExecutionService) HydrateExecutionProto(ctx context.Context, execIndex *database.ExecutionIndex) (*basexecution.Execution, error) {
	if m.HydrateExecutionProtoError != nil {
		return nil, m.HydrateExecutionProtoError
	}

	return &basexecution.Execution{
		ExecutionId: execIndex.ID.String(),
		Status:      basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING,
	}, nil
}

func (m *MockExecutionService) GetExecutionTimeline(ctx context.Context, executionID uuid.UUID) (*workflow.ExecutionTimeline, error) {
	if m.GetExecutionTimelineError != nil {
		return nil, m.GetExecutionTimelineError
	}

	if m.ExecutionTimeline != nil {
		return m.ExecutionTimeline, nil
	}
	return &workflow.ExecutionTimeline{}, nil
}

func (m *MockExecutionService) GetExecutionTimelineProto(ctx context.Context, executionID uuid.UUID) (*bastimeline.ExecutionTimeline, error) {
	if m.GetExecutionTimelineProtoError != nil {
		return nil, m.GetExecutionTimelineProtoError
	}
	if m.ExecutionTimelineProto != nil {
		return m.ExecutionTimelineProto, nil
	}
	return &bastimeline.ExecutionTimeline{}, nil
}

func (m *MockExecutionService) DescribeExecutionExport(ctx context.Context, executionID uuid.UUID) (*workflow.ExecutionExportPreview, error) {
	if m.DescribeExecutionExportError != nil {
		return nil, m.DescribeExecutionExportError
	}
	return &workflow.ExecutionExportPreview{
		Package: &export.ReplayMovieSpec{},
	}, nil
}

func (m *MockExecutionService) ExportToFolder(ctx context.Context, executionID uuid.UUID, outputDir string, storageClient storage.StorageInterface) error {
	return m.ExportToFolderError
}

// Test helper methods
func (m *MockExecutionService) AddExecution(execution *database.ExecutionIndex) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if execution.ID == uuid.Nil {
		execution.ID = uuid.New()
	}
	copy := *execution
	m.executions[execution.ID] = &copy
}

// ============================================================================
// Mock WebSocket Hub
// ============================================================================

// MockHub is a test mock for wsHub.HubInterface
type MockHub struct {
	mu sync.RWMutex

	ClientCount             int
	ExecutionFrameSubscribers map[string]bool
	RecordingSubscribers      map[string]bool

	// Call tracking
	BroadcastEnvelopeCalled bool
	LastBroadcastedEvent    any
}

func NewMockHub() *MockHub {
	return &MockHub{
		ExecutionFrameSubscribers: make(map[string]bool),
		RecordingSubscribers:      make(map[string]bool),
	}
}

// Compile-time interface check
var _ wsHub.HubInterface = (*MockHub)(nil)

func (m *MockHub) ServeWS(conn *websocket.Conn, executionID *uuid.UUID) {}

func (m *MockHub) BroadcastEnvelope(event any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.BroadcastEnvelopeCalled = true
	m.LastBroadcastedEvent = event
}

func (m *MockHub) BroadcastRecordingAction(sessionID string, action any) {}

func (m *MockHub) BroadcastRecordingActionWithTimeline(sessionID string, action any, timelineEntry map[string]any) {
}

func (m *MockHub) BroadcastRecordingFrame(sessionID string, frame *wsHub.RecordingFrame) {}

func (m *MockHub) BroadcastBinaryFrame(sessionID string, jpegData []byte) {}

func (m *MockHub) HasRecordingSubscribers(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.RecordingSubscribers[sessionID]
}

func (m *MockHub) BroadcastPerfStats(sessionID string, stats any) {}

func (m *MockHub) BroadcastPageEvent(sessionID string, event any) {}

func (m *MockHub) BroadcastPageSwitch(sessionID, activePageID string) {}

func (m *MockHub) HasExecutionFrameSubscribers(executionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ExecutionFrameSubscribers[executionID]
}

func (m *MockHub) BroadcastExecutionFrame(executionID string, frame *wsHub.ExecutionFrame) {}

func (m *MockHub) GetClientCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ClientCount
}

func (m *MockHub) Run() {}

func (m *MockHub) CloseExecution(executionID uuid.UUID) {}

// ============================================================================
// Mock Storage
// ============================================================================

// MockStorage is a test mock for storage.StorageInterface
type MockStorage struct {
	mu sync.RWMutex

	screenshots map[string][]byte
	artifacts   map[string][]byte

	// Error injection
	HealthCheckError       error
	GetScreenshotError     error
	StoreScreenshotError   error
	DeleteScreenshotError  error
	ListScreenshotsError   error
	GetArtifactError       error
	StoreArtifactError     error

	// Config
	BucketName string
	Healthy    bool
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		screenshots: make(map[string][]byte),
		artifacts:   make(map[string][]byte),
		BucketName:  "test-bucket",
		Healthy:     true,
	}
}

// Compile-time interface check
var _ storage.StorageInterface = (*MockStorage)(nil)

func (m *MockStorage) GetScreenshot(ctx context.Context, objectName string) (io.ReadCloser, *minio.ObjectInfo, error) {
	if m.GetScreenshotError != nil {
		return nil, nil, m.GetScreenshotError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	data, ok := m.screenshots[objectName]
	if !ok {
		return nil, nil, database.ErrNotFound
	}

	return io.NopCloser(strings.NewReader(string(data))), &minio.ObjectInfo{
		Key:  objectName,
		Size: int64(len(data)),
	}, nil
}

func (m *MockStorage) StoreScreenshot(ctx context.Context, executionID uuid.UUID, stepName string, data []byte, contentType string) (*storage.ScreenshotInfo, error) {
	if m.StoreScreenshotError != nil {
		return nil, m.StoreScreenshotError
	}

	objectName := executionID.String() + "/" + stepName + ".png"

	m.mu.Lock()
	m.screenshots[objectName] = data
	m.mu.Unlock()

	return &storage.ScreenshotInfo{
		ObjectName: objectName,
		SizeBytes:  int64(len(data)),
	}, nil
}

func (m *MockStorage) StoreScreenshotFromFile(ctx context.Context, executionID uuid.UUID, stepName string, filePath string) (*storage.ScreenshotInfo, error) {
	return m.StoreScreenshot(ctx, executionID, stepName, []byte("mock-file-content"), "image/png")
}

func (m *MockStorage) DeleteScreenshot(ctx context.Context, objectName string) error {
	if m.DeleteScreenshotError != nil {
		return m.DeleteScreenshotError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.screenshots, objectName)
	return nil
}

func (m *MockStorage) ListExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]string, error) {
	if m.ListScreenshotsError != nil {
		return nil, m.ListScreenshotsError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	prefix := executionID.String() + "/"
	var result []string
	for key := range m.screenshots {
		if strings.HasPrefix(key, prefix) {
			result = append(result, key)
		}
	}
	return result, nil
}

func (m *MockStorage) GetArtifact(ctx context.Context, objectName string) (io.ReadCloser, *minio.ObjectInfo, error) {
	if m.GetArtifactError != nil {
		return nil, nil, m.GetArtifactError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	data, ok := m.artifacts[objectName]
	if !ok {
		return nil, nil, database.ErrNotFound
	}

	return io.NopCloser(strings.NewReader(string(data))), &minio.ObjectInfo{
		Key:  objectName,
		Size: int64(len(data)),
	}, nil
}

func (m *MockStorage) StoreArtifactFromFile(ctx context.Context, executionID uuid.UUID, label string, filePath string, contentType string) (*storage.ArtifactInfo, error) {
	return m.StoreArtifact(ctx, executionID.String()+"/"+label, []byte("mock-artifact"), contentType)
}

func (m *MockStorage) StoreArtifact(ctx context.Context, objectName string, data []byte, contentType string) (*storage.ArtifactInfo, error) {
	if m.StoreArtifactError != nil {
		return nil, m.StoreArtifactError
	}

	m.mu.Lock()
	m.artifacts[objectName] = data
	m.mu.Unlock()

	return &storage.ArtifactInfo{
		ObjectName: objectName,
		SizeBytes:  int64(len(data)),
	}, nil
}

func (m *MockStorage) HealthCheck(ctx context.Context) error {
	if m.HealthCheckError != nil {
		return m.HealthCheckError
	}
	if !m.Healthy {
		return database.ErrNotFound
	}
	return nil
}

func (m *MockStorage) GetBucketName() string {
	return m.BucketName
}

// ============================================================================
// Mock DriverClient (implements driver.ClientInterface)
// ============================================================================

// MockDriverClient is a test mock for driver.ClientInterface.
// It provides configurable responses and error injection for testing handlers.
type MockDriverClient struct {
	mu sync.RWMutex

	// Error injection
	StopRecordingError        error
	GetRecordingStatusError   error
	GetRecordedActionsError   error
	NavigateError             error
	ReloadError               error
	GoBackError               error
	GoForwardError            error
	GetNavigationStateError   error
	GetNavigationStackError   error
	UpdateViewportError       error
	UpdateStreamSettingsError error
	ValidateSelectorError     error
	ReplayPreviewError        error
	CaptureScreenshotError    error
	GetFrameError             error
	ForwardInputError         error

	// Response overrides
	StopRecordingResponse   *driver.StopRecordingResponse
	RecordingStatusResponse *driver.RecordingStatusResponse
	RecordedActionsResponse *driver.GetActionsResponse
	NavigateResponse        *driver.NavigateResponse
	ReloadResponse          *driver.ReloadResponse
	GoBackResponse          *driver.GoBackResponse
	GoForwardResponse       *driver.GoForwardResponse
	NavigationStateResponse *driver.NavigationStateResponse
	NavigationStackResponse *driver.NavigationStackResponse
	UpdateViewportResponse  *driver.UpdateViewportResponse
	StreamSettingsResponse  *driver.UpdateStreamSettingsResponse
	ValidateSelectorResponse *driver.ValidateSelectorResponse
	ReplayPreviewResponse   *driver.ReplayPreviewResponse
	ScreenshotResponse      *driver.CaptureScreenshotResponse
	FrameResponse           *driver.GetFrameResponse

	// Call tracking
	StopRecordingCalled      bool
	GetRecordedActionsCalled bool
	ForwardInputCalled       bool
	LastSessionID            string
	LastForwardInputBody     []byte
}

// NewMockDriverClient creates a new MockDriverClient.
func NewMockDriverClient() *MockDriverClient {
	return &MockDriverClient{}
}

func (m *MockDriverClient) StopRecording(ctx context.Context, sessionID string) (*driver.StopRecordingResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StopRecordingCalled = true
	m.LastSessionID = sessionID

	if m.StopRecordingError != nil {
		return nil, m.StopRecordingError
	}
	if m.StopRecordingResponse != nil {
		return m.StopRecordingResponse, nil
	}
	return &driver.StopRecordingResponse{
		SessionID:   sessionID,
		IsRecording: false,
		ActionCount: 5,
		StoppedAt:   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (m *MockDriverClient) GetRecordingStatus(ctx context.Context, sessionID string) (*driver.RecordingStatusResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetRecordingStatusError != nil {
		return nil, m.GetRecordingStatusError
	}
	if m.RecordingStatusResponse != nil {
		return m.RecordingStatusResponse, nil
	}
	return &driver.RecordingStatusResponse{
		SessionID:   sessionID,
		IsRecording: false,
		ActionCount: 0,
		FrameCount:  0,
	}, nil
}

func (m *MockDriverClient) GetRecordedActions(ctx context.Context, sessionID string, clear bool) (*driver.GetActionsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GetRecordedActionsCalled = true
	m.LastSessionID = sessionID

	if m.GetRecordedActionsError != nil {
		return nil, m.GetRecordedActionsError
	}
	if m.RecordedActionsResponse != nil {
		return m.RecordedActionsResponse, nil
	}
	return &driver.GetActionsResponse{
		SessionID:   sessionID,
		IsRecording: false,
		Actions:     []driver.RecordedAction{},
	}, nil
}

func (m *MockDriverClient) Navigate(ctx context.Context, sessionID string, req *driver.NavigateRequest) (*driver.NavigateResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.NavigateError != nil {
		return nil, m.NavigateError
	}
	if m.NavigateResponse != nil {
		return m.NavigateResponse, nil
	}
	return &driver.NavigateResponse{
		URL:        req.URL,
		Title:      "Test Page",
		StatusCode: 200,
	}, nil
}

func (m *MockDriverClient) Reload(ctx context.Context, sessionID string, req *driver.ReloadRequest) (*driver.ReloadResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.ReloadError != nil {
		return nil, m.ReloadError
	}
	if m.ReloadResponse != nil {
		return m.ReloadResponse, nil
	}
	return &driver.ReloadResponse{
		SessionID: sessionID,
		URL:       "https://example.com",
		Title:     "Reloaded Page",
	}, nil
}

func (m *MockDriverClient) GoBack(ctx context.Context, sessionID string, req *driver.GoBackRequest) (*driver.GoBackResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GoBackError != nil {
		return nil, m.GoBackError
	}
	if m.GoBackResponse != nil {
		return m.GoBackResponse, nil
	}
	return &driver.GoBackResponse{
		SessionID: sessionID,
		URL:       "https://example.com/previous",
		Title:     "Previous Page",
	}, nil
}

func (m *MockDriverClient) GoForward(ctx context.Context, sessionID string, req *driver.GoForwardRequest) (*driver.GoForwardResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GoForwardError != nil {
		return nil, m.GoForwardError
	}
	if m.GoForwardResponse != nil {
		return m.GoForwardResponse, nil
	}
	return &driver.GoForwardResponse{
		SessionID: sessionID,
		URL:       "https://example.com/next",
		Title:     "Next Page",
	}, nil
}

func (m *MockDriverClient) GetNavigationState(ctx context.Context, sessionID string) (*driver.NavigationStateResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetNavigationStateError != nil {
		return nil, m.GetNavigationStateError
	}
	if m.NavigationStateResponse != nil {
		return m.NavigationStateResponse, nil
	}
	return &driver.NavigationStateResponse{
		URL:          "https://example.com",
		Title:        "Test Page",
		CanGoBack:    false,
		CanGoForward: false,
	}, nil
}

func (m *MockDriverClient) GetNavigationStack(ctx context.Context, sessionID string) (*driver.NavigationStackResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetNavigationStackError != nil {
		return nil, m.GetNavigationStackError
	}
	if m.NavigationStackResponse != nil {
		return m.NavigationStackResponse, nil
	}
	return &driver.NavigationStackResponse{
		SessionID:    sessionID,
		BackStack:    []driver.NavigationStackEntry{},
		ForwardStack: []driver.NavigationStackEntry{},
	}, nil
}

func (m *MockDriverClient) UpdateViewport(ctx context.Context, sessionID string, req *driver.UpdateViewportRequest) (*driver.UpdateViewportResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.UpdateViewportError != nil {
		return nil, m.UpdateViewportError
	}
	if m.UpdateViewportResponse != nil {
		return m.UpdateViewportResponse, nil
	}
	return &driver.UpdateViewportResponse{
		SessionID: sessionID,
		Width:     req.Width,
		Height:    req.Height,
	}, nil
}

func (m *MockDriverClient) UpdateStreamSettings(ctx context.Context, sessionID string, req *driver.UpdateStreamSettingsRequest) (*driver.UpdateStreamSettingsResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.UpdateStreamSettingsError != nil {
		return nil, m.UpdateStreamSettingsError
	}
	if m.StreamSettingsResponse != nil {
		return m.StreamSettingsResponse, nil
	}

	quality := 55
	if req.Quality != nil {
		quality = *req.Quality
	}
	fps := 6
	if req.FPS != nil {
		fps = *req.FPS
	}

	return &driver.UpdateStreamSettingsResponse{
		SessionID:   sessionID,
		Quality:     quality,
		FPS:         fps,
		CurrentFPS:  fps,
		Scale:       req.Scale,
		IsStreaming: true,
		Updated:     true,
	}, nil
}

func (m *MockDriverClient) ValidateSelector(ctx context.Context, sessionID string, req *driver.ValidateSelectorRequest) (*driver.ValidateSelectorResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.ValidateSelectorError != nil {
		return nil, m.ValidateSelectorError
	}
	if m.ValidateSelectorResponse != nil {
		return m.ValidateSelectorResponse, nil
	}
	return &driver.ValidateSelectorResponse{
		Valid:      true,
		MatchCount: 1,
		Selector:   req.Selector,
	}, nil
}

func (m *MockDriverClient) ReplayPreview(ctx context.Context, sessionID string, req *driver.ReplayPreviewRequest) (*driver.ReplayPreviewResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.ReplayPreviewError != nil {
		return nil, m.ReplayPreviewError
	}
	if m.ReplayPreviewResponse != nil {
		return m.ReplayPreviewResponse, nil
	}
	return &driver.ReplayPreviewResponse{
		Success:         true,
		PassedActions:   len(req.Actions),
		FailedActions:   0,
		Results:         []driver.ActionResult{},
		TotalDurationMs: 100,
	}, nil
}

func (m *MockDriverClient) CaptureScreenshot(ctx context.Context, sessionID string, req *driver.CaptureScreenshotRequest) (*driver.CaptureScreenshotResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.CaptureScreenshotError != nil {
		return nil, m.CaptureScreenshotError
	}
	if m.ScreenshotResponse != nil {
		return m.ScreenshotResponse, nil
	}
	return &driver.CaptureScreenshotResponse{
		Data: "base64-screenshot-data",
	}, nil
}

func (m *MockDriverClient) GetFrame(ctx context.Context, sessionID, queryParams string) (*driver.GetFrameResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetFrameError != nil {
		return nil, m.GetFrameError
	}
	if m.FrameResponse != nil {
		return m.FrameResponse, nil
	}
	return &driver.GetFrameResponse{
		Data:        "base64-frame-data",
		MediaType:   "image/jpeg",
		Width:       1920,
		Height:      1080,
		CapturedAt:  time.Now().UTC().Format(time.RFC3339),
		ContentHash: "abc123",
	}, nil
}

func (m *MockDriverClient) ForwardInput(ctx context.Context, sessionID string, body []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ForwardInputCalled = true
	m.LastSessionID = sessionID
	m.LastForwardInputBody = body

	return m.ForwardInputError
}

// Compile-time interface check
var _ driver.ClientInterface = (*MockDriverClient)(nil)

// ============================================================================
// Mock RecordModeService
// ============================================================================

// MockRecordModeService is a test mock for RecordModeService interface.
// It implements the RecordModeService interface defined in handler.go.
type MockRecordModeService struct {
	mu sync.RWMutex

	// Session tracking
	Sessions map[string]*livecapture.SessionResult

	// Mock driver client for pass-through operations
	mockDriverClient *MockDriverClient

	// Error injection for service-level operations
	CreateSessionError   error
	CloseSessionError    error
	GetStorageStateError error
	StartRecordingError  error
	GenerateWorkflowError error

	// Response overrides
	GeneratedWorkflow *livecapture.GenerateWorkflowResult
	StorageState      json.RawMessage

	// Call tracking
	CreateSessionCalled      bool
	CloseSessionCalled       bool
	StartRecordingCalled     bool
	GenerateWorkflowCalled   bool
	LastSessionID            string
}

func NewMockRecordModeService() *MockRecordModeService {
	return &MockRecordModeService{
		Sessions:         make(map[string]*livecapture.SessionResult),
		mockDriverClient: NewMockDriverClient(),
	}
}

// DriverClient returns the mock driver client for testing.
func (m *MockRecordModeService) DriverClient() driver.ClientInterface {
	return m.mockDriverClient
}

// MockClient provides direct access to the mock driver client for test configuration.
func (m *MockRecordModeService) MockClient() *MockDriverClient {
	return m.mockDriverClient
}

func (m *MockRecordModeService) CreateSession(ctx context.Context, cfg *livecapture.SessionConfig) (*livecapture.SessionResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CreateSessionCalled = true

	if m.CreateSessionError != nil {
		return nil, m.CreateSessionError
	}

	result := &livecapture.SessionResult{
		SessionID: "test-session-" + uuid.NewString()[:8],
		CreatedAt: time.Now().UTC(),
	}
	m.Sessions[result.SessionID] = result
	m.LastSessionID = result.SessionID

	return result, nil
}

func (m *MockRecordModeService) CloseSession(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CloseSessionCalled = true
	m.LastSessionID = sessionID

	if m.CloseSessionError != nil {
		return m.CloseSessionError
	}

	delete(m.Sessions, sessionID)
	return nil
}

func (m *MockRecordModeService) GetStorageState(ctx context.Context, sessionID string) (json.RawMessage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetStorageStateError != nil {
		return nil, m.GetStorageStateError
	}

	if m.StorageState != nil {
		return m.StorageState, nil
	}
	return json.RawMessage(`{"cookies":[],"origins":[]}`), nil
}

func (m *MockRecordModeService) StartRecording(ctx context.Context, sessionID string, cfg *livecapture.RecordingConfig) (*driver.StartRecordingResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StartRecordingCalled = true
	m.LastSessionID = sessionID

	if m.StartRecordingError != nil {
		return nil, m.StartRecordingError
	}

	return &driver.StartRecordingResponse{
		SessionID:   sessionID,
		IsRecording: true,
		StartedAt:   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (m *MockRecordModeService) GenerateWorkflow(ctx context.Context, sessionID string, cfg *livecapture.GenerateWorkflowConfig) (*livecapture.GenerateWorkflowResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GenerateWorkflowCalled = true
	m.LastSessionID = sessionID

	if m.GenerateWorkflowError != nil {
		return nil, m.GenerateWorkflowError
	}

	if m.GeneratedWorkflow != nil {
		return m.GeneratedWorkflow, nil
	}

	return &livecapture.GenerateWorkflowResult{
		FlowDefinition: map[string]interface{}{
			"nodes": []map[string]interface{}{},
			"edges": []map[string]interface{}{},
		},
		NodeCount:   0,
		ActionCount: 0,
	}, nil
}

// Multi-page support methods

func (m *MockRecordModeService) GetSession(sessionID string) (*autosession.Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return nil session for testing - tests should use full mock if needed
	return nil, false
}

func (m *MockRecordModeService) GetPages(sessionID string) (*livecapture.PageListResult, error) {
	return &livecapture.PageListResult{
		Pages:        []*domain.Page{},
		ActivePageID: "",
	}, nil
}

func (m *MockRecordModeService) ActivatePage(ctx context.Context, sessionID string, pageID uuid.UUID) error {
	return nil
}

// Service worker management methods

func (m *MockRecordModeService) GetServiceWorkers(ctx context.Context, sessionID string) (*driver.GetServiceWorkersResponse, error) {
	return &driver.GetServiceWorkersResponse{
		SessionID: sessionID,
		Workers:   []driver.ServiceWorkerInfo{},
	}, nil
}

func (m *MockRecordModeService) UnregisterAllServiceWorkers(ctx context.Context, sessionID string) (*driver.UnregisterServiceWorkersResponse, error) {
	return &driver.UnregisterServiceWorkersResponse{
		SessionID:         sessionID,
		UnregisteredCount: 0,
	}, nil
}

func (m *MockRecordModeService) UnregisterServiceWorker(ctx context.Context, sessionID, scopeURL string) (*driver.UnregisterServiceWorkerResponse, error) {
	return &driver.UnregisterServiceWorkerResponse{
		SessionID: sessionID,
	}, nil
}

// Timeline support methods

func (m *MockRecordModeService) CreatePage(ctx context.Context, sessionID string, url string) (*driver.CreatePageResponse, error) {
	return &driver.CreatePageResponse{
		DriverPageID: "mock-page-id",
		URL:          url,
	}, nil
}

func (m *MockRecordModeService) AddTimelineAction(sessionID string, action *driver.RecordedAction, pageID uuid.UUID) {
	// No-op for mock
}

func (m *MockRecordModeService) AddTimelinePageEvent(sessionID string, event *domain.PageEvent) {
	// No-op for mock
}

func (m *MockRecordModeService) GetTimeline(sessionID string, pageID *uuid.UUID, limit int) (*domain.TimelineResponse, error) {
	return &domain.TimelineResponse{
		Entries:      []domain.TimelineEntry{},
		HasMore:      false,
		TotalEntries: 0,
	}, nil
}

// ============================================================================
// Test Handler Factory
// ============================================================================

// NewTestHandler creates a Handler with all mock dependencies for testing.
// Returns the handler and all mocks for test assertions.
func NewTestHandler() (*Handler, *MockCatalogService, *MockExecutionService, *MockRepository, *MockHub, *MockStorage) {
	repo := NewMockRepository()
	hub := NewMockHub()
	catalogSvc := NewMockCatalogService()
	execSvc := NewMockExecutionService()
	storageMock := NewMockStorage()

	handler := &Handler{
		catalogService:   catalogSvc,
		executionService: execSvc,
		repo:             repo,
		wsHub:            hub,
		storage:          storageMock,
	}

	return handler, catalogSvc, execSvc, repo, hub, storageMock
}
