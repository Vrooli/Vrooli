package browserless

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/browserless/events"
	"github.com/vrooli/browser-automation-studio/browserless/runtime"
	"github.com/vrooli/browser-automation-studio/database"
)

type mockRepository struct {
	folders            map[string]*database.WorkflowFolder
	executionLogs      []*database.ExecutionLog
	screenshots        []*database.Screenshot
	updatedExecutions  []*database.Execution
	extractedData      []*database.ExtractedData
	executionSteps     []*database.ExecutionStep
	executionArtifacts []*database.ExecutionArtifact
}

type fakeEmitter struct {
	mu     sync.Mutex
	events []events.Event
}

func (f *fakeEmitter) Emit(event events.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, event)
}

func (f *fakeEmitter) Events() []events.Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	copyEvents := make([]events.Event, len(f.events))
	copy(copyEvents, f.events)
	return copyEvents
}

func newMockRepository() *mockRepository {
	return &mockRepository{folders: make(map[string]*database.WorkflowFolder)}
}

// Project operations
type noopProject struct{}

func (m *mockRepository) CreateProject(ctx context.Context, project *database.Project) error {
	return nil
}
func (m *mockRepository) GetProject(ctx context.Context, id uuid.UUID) (*database.Project, error) {
	return nil, nil
}
func (m *mockRepository) GetProjectByName(ctx context.Context, name string) (*database.Project, error) {
	return nil, nil
}
func (m *mockRepository) GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.Project, error) {
	return nil, nil
}
func (m *mockRepository) UpdateProject(ctx context.Context, project *database.Project) error {
	return nil
}
func (m *mockRepository) DeleteProject(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockRepository) ListProjects(ctx context.Context, limit, offset int) ([]*database.Project, error) {
	return nil, nil
}
func (m *mockRepository) GetProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]any, error) {
	return nil, nil
}

// Workflow operations
func (m *mockRepository) CreateWorkflow(ctx context.Context, workflow *database.Workflow) error {
	return nil
}
func (m *mockRepository) GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
	return nil, nil
}
func (m *mockRepository) GetWorkflowByName(ctx context.Context, name, folderPath string) (*database.Workflow, error) {
	return nil, nil
}
func (m *mockRepository) UpdateWorkflow(ctx context.Context, workflow *database.Workflow) error {
	return nil
}
func (m *mockRepository) DeleteWorkflow(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockRepository) DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error {
	return nil
}
func (m *mockRepository) ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}
func (m *mockRepository) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}
func (m *mockRepository) CreateWorkflowVersion(ctx context.Context, version *database.WorkflowVersion) error {
	return nil
}
func (m *mockRepository) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*database.WorkflowVersion, error) {
	return nil, nil
}
func (m *mockRepository) ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*database.WorkflowVersion, error) {
	return nil, nil
}

// Execution operations
func (m *mockRepository) CreateExecution(ctx context.Context, execution *database.Execution) error {
	return nil
}
func (m *mockRepository) GetExecution(ctx context.Context, id uuid.UUID) (*database.Execution, error) {
	return nil, nil
}
func (m *mockRepository) UpdateExecution(ctx context.Context, execution *database.Execution) error {
	copyExec := *execution
	m.updatedExecutions = append(m.updatedExecutions, &copyExec)
	return nil
}
func (m *mockRepository) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error) {
	return nil, nil
}

func (m *mockRepository) CreateExecutionStep(ctx context.Context, step *database.ExecutionStep) error {
	copyStep := *step
	if copyStep.ID == uuid.Nil {
		copyStep.ID = uuid.New()
	}
	m.executionSteps = append(m.executionSteps, &copyStep)
	step.ID = copyStep.ID
	return nil
}

func (m *mockRepository) UpdateExecutionStep(ctx context.Context, step *database.ExecutionStep) error {
	for i, existing := range m.executionSteps {
		if existing.ID == step.ID {
			copyStep := *step
			m.executionSteps[i] = &copyStep
			return nil
		}
	}
	return nil
}

func (m *mockRepository) ListExecutionSteps(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionStep, error) {
	var steps []*database.ExecutionStep
	for _, step := range m.executionSteps {
		if step.ExecutionID == executionID {
			copyStep := *step
			steps = append(steps, &copyStep)
		}
	}
	return steps, nil
}

func (m *mockRepository) CreateExecutionArtifact(ctx context.Context, artifact *database.ExecutionArtifact) error {
	copyArtifact := *artifact
	if copyArtifact.ID == uuid.Nil {
		copyArtifact.ID = uuid.New()
	}
	m.executionArtifacts = append(m.executionArtifacts, &copyArtifact)
	artifact.ID = copyArtifact.ID
	return nil
}

func toIntTest(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i)
		}
	case string:
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return 0
}

func (m *mockRepository) ListExecutionArtifacts(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionArtifact, error) {
	var artifacts []*database.ExecutionArtifact
	for _, artifact := range m.executionArtifacts {
		if artifact.ExecutionID == executionID {
			copyArtifact := *artifact
			artifacts = append(artifacts, &copyArtifact)
		}
	}
	return artifacts, nil
}

// Artifact operations
func (m *mockRepository) CreateScreenshot(ctx context.Context, screenshot *database.Screenshot) error {
	copyShot := *screenshot
	m.screenshots = append(m.screenshots, &copyShot)
	return nil
}
func (m *mockRepository) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error) {
	return nil, nil
}

func (m *mockRepository) CreateExecutionLog(ctx context.Context, logEntry *database.ExecutionLog) error {
	copyLog := *logEntry
	m.executionLogs = append(m.executionLogs, &copyLog)
	return nil
}
func (m *mockRepository) GetExecutionLogs(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionLog, error) {
	return nil, nil
}

func (m *mockRepository) CreateExtractedData(ctx context.Context, data *database.ExtractedData) error {
	copyData := *data
	m.extractedData = append(m.extractedData, &copyData)
	return nil
}
func (m *mockRepository) GetExecutionExtractedData(ctx context.Context, executionID uuid.UUID) ([]*database.ExtractedData, error) {
	return nil, nil
}

// Folder operations
func (m *mockRepository) CreateFolder(ctx context.Context, folder *database.WorkflowFolder) error {
	m.folders[folder.Path] = folder
	return nil
}
func (m *mockRepository) GetFolder(ctx context.Context, path string) (*database.WorkflowFolder, error) {
	if folder, ok := m.folders[path]; ok {
		return folder, nil
	}
	return nil, database.ErrNotFound
}
func (m *mockRepository) ListFolders(ctx context.Context) ([]*database.WorkflowFolder, error) {
	result := make([]*database.WorkflowFolder, 0, len(m.folders))
	for _, folder := range m.folders {
		result = append(result, folder)
	}
	return result, nil
}

func newTestClient() (*Client, *mockRepository) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	repo := newMockRepository()
	client := &Client{
		log:               log,
		repo:              repo,
		browserless:       "http://localhost:0",
		httpClient:        &http.Client{Timeout: 2 * time.Second},
		heartbeatInterval: defaultHeartbeatInterval,
	}
	client.recordingsRoot = resolveRecordingsRoot(log)
	return client, repo
}

func TestBuildInstructions(t *testing.T) {
	client, _ := newTestClient()

	workflow := &database.Workflow{
		ID: uuid.New(),
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{
					"id":   "node-1",
					"type": "navigate",
					"data": map[string]any{
						"url":       "https://example.com",
						"waitUntil": "domcontentloaded",
						"timeoutMs": 10000,
					},
				},
				map[string]any{
					"id":   "node-2",
					"type": "wait",
					"data": map[string]any{
						"type":     "time",
						"duration": 1500,
					},
				},
				map[string]any{
					"id":   "node-3",
					"type": "click",
					"data": map[string]any{
						"selector":   "#login",
						"button":     "left",
						"clickCount": 1,
					},
				},
				map[string]any{
					"id":   "node-4",
					"type": "type",
					"data": map[string]any{
						"selector": "input[name=email]",
						"text":     "user@example.com",
						"delayMs":  25,
					},
				},
				map[string]any{
					"id":   "node-5",
					"type": "extract",
					"data": map[string]any{
						"selector":    "#headline",
						"extractType": "text",
					},
				},
				map[string]any{
					"id":   "node-6",
					"type": "screenshot",
					"data": map[string]any{
						"name":                  "after",
						"viewportWidth":         1280,
						"viewportHeight":        720,
						"waitForMs":             500,
						"focusSelector":         "#hero",
						"highlightSelectors":    []any{"#hero", ".cta"},
						"highlightColor":        "#ff00ff",
						"highlightPadding":      12,
						"highlightBorderRadius": 18,
						"maskSelectors":         []any{".mask"},
						"maskOpacity":           0.55,
						"background":            "#111111",
						"zoomFactor":            1.25,
					},
				},
			},
			"edges": []any{
				map[string]any{"id": "edge-1", "source": "node-1", "target": "node-2"},
				map[string]any{"id": "edge-2", "source": "node-2", "target": "node-3"},
				map[string]any{"id": "edge-3", "source": "node-3", "target": "node-4"},
				map[string]any{"id": "edge-4", "source": "node-4", "target": "node-5"},
				map[string]any{"id": "edge-5", "source": "node-5", "target": "node-6"},
			},
		},
	}

	instr, err := client.buildInstructions(workflow)
	if err != nil {
		t.Fatalf("buildInstructions failed: %v", err)
	}

	if len(instr) != 6 {
		t.Fatalf("expected 6 instructions, got %d", len(instr))
	}

	if instr[0].Params.URL != "https://example.com" {
		t.Errorf("navigate url not parsed: %+v", instr[0].Params)
	}

	if instr[1].Params.DurationMs != 1500 {
		t.Errorf("wait duration mismatch: %+v", instr[1].Params)
	}

	if instr[2].Params.Selector != "#login" {
		t.Errorf("click selector not parsed: %+v", instr[2].Params)
	}

	if instr[3].Params.Selector != "input[name=email]" || instr[3].Params.Text != "user@example.com" {
		t.Errorf("type params mismatch: %+v", instr[3].Params)
	}

	if instr[4].Params.Selector != "#headline" || instr[4].Params.ExtractType != "text" {
		t.Errorf("extract params mismatch: %+v", instr[4].Params)
	}

	if instr[5].Params.Name != "after" || instr[5].Params.ViewportWidth != 1280 || instr[5].Params.WaitForMs != 500 {
		t.Errorf("screenshot params mismatch: %+v", instr[5].Params)
	}

	if instr[5].Params.FocusSelector != "#hero" {
		t.Errorf("expected focus selector parsed")
	}
	if len(instr[5].Params.HighlightSelectors) != 2 || instr[5].Params.HighlightSelectors[0] != "#hero" {
		t.Errorf("highlight selectors not normalized: %+v", instr[5].Params.HighlightSelectors)
	}
	if instr[5].Params.HighlightColor != "#ff00ff" {
		t.Errorf("highlight color not parsed: %s", instr[5].Params.HighlightColor)
	}
	if instr[5].Params.HighlightPadding != 12 {
		t.Errorf("highlight padding mismatch: %d", instr[5].Params.HighlightPadding)
	}
	if instr[5].Params.HighlightBorderRadius != 18 {
		t.Errorf("highlight border radius mismatch: %d", instr[5].Params.HighlightBorderRadius)
	}
	if len(instr[5].Params.MaskSelectors) != 1 || instr[5].Params.MaskSelectors[0] != ".mask" {
		t.Errorf("mask selectors not parsed: %+v", instr[5].Params.MaskSelectors)
	}
	if math.Abs(instr[5].Params.MaskOpacity-0.55) > 0.0001 {
		t.Errorf("mask opacity not parsed: %f", instr[5].Params.MaskOpacity)
	}
	if instr[5].Params.Background != "#111111" {
		t.Errorf("background not parsed: %s", instr[5].Params.Background)
	}
	if math.Abs(instr[5].Params.ZoomFactor-1.25) > 0.0001 {
		t.Errorf("zoom factor not parsed: %f", instr[5].Params.ZoomFactor)
	}
}

func TestBuildInstructionsWorkflowCallNotSupported(t *testing.T) {
	client, _ := newTestClient()

	workflow := &database.Workflow{
		ID: uuid.New(),
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{
					"id":   "node-1",
					"type": "workflowCall",
					"data": map[string]any{
						"workflowId": "wf-123",
					},
				},
			},
		},
	}

	if _, err := client.buildInstructions(workflow); err == nil {
		t.Fatalf("expected error for unsupported workflow call node")
	}
}

func TestBuildInstructionsAssertStep(t *testing.T) {
	client, _ := newTestClient()

	workflow := &database.Workflow{
		ID: uuid.New(),
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{
					"id":   "node-assert",
					"type": "assert",
					"data": map[string]any{
						"assertMode":     "text_equals",
						"selector":       "#status",
						"expectedValue":  "Ready",
						"caseSensitive":  false,
						"negate":         false,
						"timeoutMs":      4500,
						"failureMessage": "Status indicator missing",
					},
				},
			},
			"edges": []any{},
		},
	}

	instructions, err := client.buildInstructions(workflow)
	if err != nil {
		t.Fatalf("buildInstructions returned error: %v", err)
	}

	if len(instructions) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(instructions))
	}

	assertStep := instructions[0]
	if assertStep.Type != "assert" {
		t.Fatalf("expected assert instruction, got %s", assertStep.Type)
	}
	if assertStep.Params.AssertMode != "text_equals" {
		t.Fatalf("unexpected assert mode: %s", assertStep.Params.AssertMode)
	}
	if assertStep.Params.Selector != "#status" {
		t.Fatalf("selector not propagated: %s", assertStep.Params.Selector)
	}
	if assertStep.Params.ExpectedValue != "Ready" {
		t.Fatalf("expected value not propagated: %+v", assertStep.Params.ExpectedValue)
	}
	if assertStep.Params.TimeoutMs != 4500 {
		t.Fatalf("timeout not parsed: %d", assertStep.Params.TimeoutMs)
	}
	if assertStep.Params.FailureMessage != "Status indicator missing" {
		t.Fatalf("failure message not parsed: %s", assertStep.Params.FailureMessage)
	}
}

func TestExecuteWorkflowPersistsTelemetry(t *testing.T) {
	client, repo := newTestClient()

	responses := []string{
		`{"success":true,"steps":[{"index":0,"nodeId":"node-1","type":"navigate","success":true,"durationMs":1200,"finalUrl":"https://example.com/home","consoleLogs":[{"type":"log","text":"navigated","timestamp":25}]}]}`,
		`{"success":true,"steps":[{"index":1,"nodeId":"node-2","type":"extract","success":true,"durationMs":800,"extractedData":"Headline text","elementBoundingBox":{"x":10,"y":20,"width":100,"height":30}}]}`,
	}

	var call int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if call >= len(responses) {
			t.Fatalf("unexpected additional call %d", call)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responses[call]))
		call++
	}))
	defer server.Close()

	client.browserless = server.URL
	client.httpClient = server.Client()

	exec := &database.Execution{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		StartedAt:  time.Now().Add(-time.Minute),
	}

	workflow := &database.Workflow{
		ID: uuid.New(),
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{
					"id":   "node-1",
					"type": "navigate",
					"data": map[string]any{
						"url": "https://example.com",
					},
				},
				map[string]any{
					"id":   "node-2",
					"type": "extract",
					"data": map[string]any{
						"selector":    "#headline",
						"extractType": "text",
					},
				},
			},
			"edges": []any{
				map[string]any{"id": "edge-1", "source": "node-1", "target": "node-2"},
			},
		},
	}

	if err := client.ExecuteWorkflow(context.Background(), exec, workflow, nil); err != nil {
		t.Fatalf("ExecuteWorkflow returned error: %v", err)
	}

	var consoleLogFound bool
	for _, entry := range repo.executionLogs {
		if entry.Message == "navigated" {
			consoleLogFound = true
			break
		}
	}
	if !consoleLogFound {
		t.Fatalf("expected console log to be persisted")
	}

	if len(repo.extractedData) != 1 {
		t.Fatalf("expected extracted data to be stored, got %d", len(repo.extractedData))
	}
	if value, ok := repo.extractedData[0].DataValue["value"].(string); !ok || value != "Headline text" {
		t.Fatalf("unexpected extracted data payload: %+v", repo.extractedData[0].DataValue)
	}

	if exec.Result == nil || exec.Result["success"] != true {
		t.Fatalf("execution result not persisted: %+v", exec.Result)
	}

	if call != len(responses) {
		t.Fatalf("expected %d browserless invocations, got %d", len(responses), call)
	}
}

func TestExecuteWorkflowRetriesOnFailure(t *testing.T) {
	client, repo := newTestClient()

	responses := []string{
		`{"success":false,"error":"timeout","steps":[{"index":0,"nodeId":"node-1","type":"navigate","success":false,"durationMs":700,"error":"timeout"}]}`,
		`{"success":true,"steps":[{"index":0,"nodeId":"node-1","type":"navigate","success":true,"durationMs":400,"finalUrl":"https://example.com/home"}]}`,
	}

	var callCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if callCount >= len(responses) {
			t.Fatalf("unexpected additional call %d", callCount)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responses[callCount]))
		callCount++
	}))
	defer server.Close()

	client.browserless = server.URL
	client.httpClient = server.Client()

	exec := &database.Execution{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		StartedAt:  time.Now().Add(-time.Minute),
	}

	workflow := &database.Workflow{
		ID: uuid.New(),
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{
					"id":   "node-1",
					"type": "navigate",
					"data": map[string]any{
						"url":           "https://example.com",
						"retryAttempts": 1,
						"retryDelayMs":  0,
					},
				},
			},
			"edges": []any{},
		},
	}

	if err := client.ExecuteWorkflow(context.Background(), exec, workflow, nil); err != nil {
		t.Fatalf("ExecuteWorkflow returned error: %v", err)
	}

	if callCount != len(responses) {
		t.Fatalf("expected %d browserless invocations, got %d", len(responses), callCount)
	}

	if len(repo.executionSteps) != 1 {
		t.Fatalf("expected 1 execution step, got %d", len(repo.executionSteps))
	}
	stepRecord := repo.executionSteps[0]
	retryMeta, ok := stepRecord.Metadata["retry"].(database.JSONMap)
	if !ok {
		t.Fatalf("expected retry metadata on execution step: %+v", stepRecord.Metadata)
	}
	if attempt := toIntTest(retryMeta["attempt"]); attempt != 2 {
		t.Fatalf("expected retry attempt 2, got %d", attempt)
	}
	if configured := toIntTest(retryMeta["configuredRetries"]); configured != 1 {
		t.Fatalf("expected configured retries 1, got %d", configured)
	}
	if maxAttempts := toIntTest(retryMeta["maxAttempts"]); maxAttempts != 2 {
		t.Fatalf("expected max attempts 2, got %d", maxAttempts)
	}
	var retryHistory []map[string]any
	switch entries := retryMeta["history"].(type) {
	case []map[string]any:
		retryHistory = entries
	case []database.JSONMap:
		for _, item := range entries {
			retryHistory = append(retryHistory, map[string]any(item))
		}
	case []any:
		for _, item := range entries {
			if entryMap, ok := item.(map[string]any); ok {
				retryHistory = append(retryHistory, entryMap)
			}
		}
	}
	if len(retryHistory) != 2 {
		t.Fatalf("expected retry history with 2 entries, got %+v", retryMeta["history"])
	}
	firstAttempt := retryHistory[0]
	if success := firstAttempt["success"]; success != false {
		t.Fatalf("expected first attempt to be marked failure, got %+v", success)
	}
	secondAttempt := retryHistory[1]
	if success := secondAttempt["success"]; success != true {
		t.Fatalf("expected second attempt to be marked success, got %+v", success)
	}

	var timelinePayload database.JSONMap
	for _, artifact := range repo.executionArtifacts {
		if artifact.ArtifactType == "timeline_frame" {
			timelinePayload = artifact.Payload
			break
		}
	}
	if timelinePayload == nil {
		t.Fatalf("expected timeline artifact to be created")
	}
	if attempt := toIntTest(timelinePayload["retryAttempt"]); attempt != 2 {
		t.Fatalf("expected timeline retryAttempt 2, got %d", attempt)
	}
	if max := toIntTest(timelinePayload["retryMaxAttempts"]); max != 2 {
		t.Fatalf("expected timeline retryMaxAttempts 2, got %d", max)
	}
	if total := toIntTest(timelinePayload["totalDurationMs"]); total <= 0 {
		t.Fatalf("expected total duration to be recorded, got %d", total)
	}
}

func TestExecuteWorkflowCreatesTimelineArtifactWithHighlightMetadata(t *testing.T) {
	client, repo := newTestClient()

	const pngBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVQYV2P4//8/AwAI/AL+fK5OAAAAAElFTkSuQmCC"
	responses := []string{
		fmt.Sprintf(`{"success":true,"steps":[{"index":0,"nodeId":"node-1","type":"screenshot","success":true,"durationMs":250,"screenshotBase64":"%s","domSnapshot":"<html><body>snapshot</body></html>","focusedElement":{"selector":"#hero","boundingBox":{"x":10,"y":20,"width":200,"height":120}},"highlightRegions":[{"selector":"#hero","boundingBox":{"x":10,"y":20,"width":200,"height":120},"padding":12,"color":"#ff00ff"}],"maskRegions":[{"selector":".mask","boundingBox":{"x":0,"y":0,"width":320,"height":180},"opacity":0.5}],"zoomFactor":1.4,"consoleLogs":[{"type":"log","text":"shot","timestamp":5}]}]}`, pngBase64),
	}

	var call int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if call >= len(responses) {
			t.Fatalf("unexpected additional call %d", call)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responses[call]))
		call++
	}))
	defer server.Close()

	client.browserless = server.URL
	client.httpClient = server.Client()
	client.storage = nil

	if err := os.RemoveAll(filepath.Join(os.TempDir(), "browser-automation-studio")); err != nil {
		t.Fatalf("failed to cleanup temp dir: %v", err)
	}
	t.Setenv("TMPDIR", t.TempDir())

	exec := &database.Execution{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		StartedAt:  time.Now().Add(-time.Minute),
	}

	workflow := &database.Workflow{
		ID: uuid.New(),
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{
					"id":   "node-1",
					"type": "screenshot",
					"data": map[string]any{
						"name":          "marketing",
						"waitForMs":     250,
						"focusSelector": "#hero",
					},
				},
			},
		},
	}

	if err := client.ExecuteWorkflow(context.Background(), exec, workflow, nil); err != nil {
		t.Fatalf("ExecuteWorkflow returned error: %v", err)
	}

	if call != len(responses) {
		t.Fatalf("expected %d browserless invocations, got %d", len(responses), call)
	}

	var timeline *database.ExecutionArtifact
	var screenshotArtifact *database.ExecutionArtifact
	var domArtifact *database.ExecutionArtifact
	for _, artifact := range repo.executionArtifacts {
		switch artifact.ArtifactType {
		case "timeline_frame":
			timeline = artifact
		case "screenshot":
			screenshotArtifact = artifact
		case "dom_snapshot":
			domArtifact = artifact
		}
	}

	if timeline == nil {
		t.Fatalf("expected timeline artifact to be created")
	}
	if screenshotArtifact == nil {
		t.Fatalf("expected screenshot artifact to be created")
	}
	if domArtifact == nil {
		t.Fatalf("expected DOM snapshot artifact to be created")
	}

	if payload := timeline.Payload; payload == nil {
		t.Fatalf("timeline payload missing")
	} else {
		checkHighlight := func(value any) {
			switch regions := value.(type) {
			case []runtime.HighlightRegion:
				if len(regions) != 1 || regions[0].Selector != "#hero" {
					t.Fatalf("unexpected highlight regions payload: %+v", regions)
				}
			case []any:
				if len(regions) != 1 {
					t.Fatalf("unexpected highlight regions payload: %+v", regions)
				}
				region, _ := regions[0].(map[string]any)
				if region == nil || region["selector"] != "#hero" {
					t.Fatalf("unexpected highlight region payload: %+v", region)
				}
			default:
				t.Fatalf("highlight region type mismatch: %T", value)
			}
		}

		checkMask := func(value any) {
			switch regions := value.(type) {
			case []runtime.MaskRegion:
				if len(regions) != 1 || regions[0].Selector != ".mask" {
					t.Fatalf("unexpected mask regions payload: %+v", regions)
				}
			case []any:
				if len(regions) != 1 {
					t.Fatalf("unexpected mask regions payload: %+v", regions)
				}
				region, _ := regions[0].(map[string]any)
				if region == nil || region["selector"] != ".mask" {
					t.Fatalf("unexpected mask region payload: %+v", region)
				}
			default:
				t.Fatalf("mask region type mismatch: %T", value)
			}
		}

		checkHighlight(payload["highlightRegions"])
		checkMask(payload["maskRegions"])
		if zoom, ok := payload["zoomFactor"].(float64); !ok || math.Abs(zoom-1.4) > 0.0001 {
			t.Fatalf("unexpected zoom factor: %+v", payload["zoomFactor"])
		}
		if id, ok := payload["screenshotArtifactId"].(string); !ok || id != screenshotArtifact.ID.String() {
			t.Fatalf("timeline payload missing screenshot reference: %+v", payload["screenshotArtifactId"])
		}
		if preview, ok := payload["domSnapshotPreview"].(string); !ok || !strings.Contains(preview, "snapshot") {
			t.Fatalf("timeline payload missing DOM snapshot preview: %+v", payload["domSnapshotPreview"])
		}
		if id, ok := payload["domSnapshotArtifactId"].(string); !ok || id != domArtifact.ID.String() {
			t.Fatalf("timeline payload missing dom snapshot reference: %+v", payload["domSnapshotArtifactId"])
		}
	}

	if payload := screenshotArtifact.Payload; payload == nil {
		t.Fatalf("screenshot payload missing")
	} else {
		checkHighlight := func(value any) {
			switch regions := value.(type) {
			case []runtime.HighlightRegion:
				if len(regions) == 0 || regions[0].Selector != "#hero" {
					t.Fatalf("unexpected highlight regions payload: %+v", regions)
				}
			case []any:
				if len(regions) == 0 {
					t.Fatalf("unexpected highlight regions payload: %+v", regions)
				}
			default:
				t.Fatalf("highlight region type mismatch: %T", value)
			}
		}
		checkMask := func(value any) {
			switch regions := value.(type) {
			case []runtime.MaskRegion:
				if len(regions) == 0 || regions[0].Selector != ".mask" {
					t.Fatalf("unexpected mask regions payload: %+v", regions)
				}
			case []any:
				if len(regions) == 0 {
					t.Fatalf("unexpected mask regions payload: %+v", regions)
				}
			default:
				t.Fatalf("mask region type mismatch: %T", value)
			}
		}

		switch focus := payload["focusedElement"].(type) {
		case *runtime.ElementFocus:
			if focus == nil || focus.Selector != "#hero" {
				t.Fatalf("expected focused element metadata: %+v", focus)
			}
		case map[string]any:
			if focus["selector"] != "#hero" {
				t.Fatalf("expected focused element selector: %+v", focus)
			}
		default:
			t.Fatalf("unexpected focused element type: %T", payload["focusedElement"])
		}
		checkHighlight(payload["highlightRegions"])
		checkMask(payload["maskRegions"])
		if zoom, ok := payload["zoomFactor"].(float64); !ok || math.Abs(zoom-1.4) > 0.0001 {
			t.Fatalf("unexpected screenshot zoom factor: %+v", payload["zoomFactor"])
		}
	}

	if len(repo.executionSteps) != 1 {
		t.Fatalf("expected one execution step, got %d", len(repo.executionSteps))
	}
	stepMetadata := repo.executionSteps[0].Metadata
	if stepMetadata == nil || stepMetadata["focusedElement"] == nil {
		t.Fatalf("focused element metadata missing from execution step: %+v", stepMetadata)
	}
	artifactIDs, ok := stepMetadata["artifactIds"].([]string)
	if !ok {
		t.Fatalf("expected artifactIds slice in step metadata, got %+v", stepMetadata["artifactIds"])
	}
	if len(artifactIDs) < 3 {
		t.Fatalf("expected artifact ids to include dom snapshot, got %+v", artifactIDs)
	}
	if preview, ok := stepMetadata["domSnapshotPreview"].(string); !ok || !strings.Contains(preview, "snapshot") {
		t.Fatalf("expected dom snapshot preview in step metadata, got %+v", stepMetadata["domSnapshotPreview"])
	}

	if len(repo.executionLogs) == 0 {
		t.Fatalf("expected console log artifact to be created")
	}
}

func TestExecuteWorkflowPersistsAssertionArtifacts(t *testing.T) {
	client, repo := newTestClient()

	responses := []string{
		`{"success":true,"steps":[{"index":0,"nodeId":"node-assert","type":"assert","success":true,"durationMs":180,"assertion":{"mode":"exists","selector":"#status","success":true,"actual":true}}]}`,
	}

	var call int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if call >= len(responses) {
			t.Fatalf("unexpected additional call %d", call)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responses[call]))
		call++
	}))
	defer server.Close()

	client.browserless = server.URL
	client.httpClient = server.Client()

	exec := &database.Execution{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		StartedAt:  time.Now(),
	}

	workflow := &database.Workflow{
		ID: uuid.New(),
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{
					"id":   "node-assert",
					"type": "assert",
					"data": map[string]any{
						"assertMode":     "exists",
						"selector":       "#status",
						"failureMessage": "Status indicator missing",
					},
				},
			},
		},
	}

	if err := client.ExecuteWorkflow(context.Background(), exec, workflow, nil); err != nil {
		t.Fatalf("ExecuteWorkflow returned error: %v", err)
	}

	var assertionArtifact *database.ExecutionArtifact
	var timelineArtifact *database.ExecutionArtifact
	for _, artifact := range repo.executionArtifacts {
		switch artifact.ArtifactType {
		case "assertion":
			assertionArtifact = artifact
		case "timeline_frame":
			timelineArtifact = artifact
		}
	}

	if assertionArtifact == nil {
		t.Fatalf("expected assertion artifact to be created")
	}
	if assertionArtifact.Payload == nil || assertionArtifact.Payload["assertion"] == nil {
		t.Fatalf("assertion payload missing: %+v", assertionArtifact.Payload)
	}

	if timelineArtifact == nil {
		t.Fatalf("expected timeline artifact to be created")
	}
	if timelineArtifact.Payload == nil || timelineArtifact.Payload["assertion"] == nil {
		t.Fatalf("timeline assertion metadata missing: %+v", timelineArtifact.Payload)
	}

	if len(repo.executionSteps) != 1 {
		t.Fatalf("expected one execution step, got %d", len(repo.executionSteps))
	}
	if repo.executionSteps[0].Metadata == nil || repo.executionSteps[0].Metadata["assertion"] == nil {
		t.Fatalf("execution step metadata missing assertion")
	}

	assertLog := false
	for _, entry := range repo.executionLogs {
		if strings.Contains(entry.Message, "assert") {
			assertLog = true
			break
		}
	}
	if !assertLog {
		t.Fatalf("expected assertion log message, got %+v", repo.executionLogs)
	}
}

func TestExecuteWorkflowEmitsHeartbeats(t *testing.T) {
	client, _ := newTestClient()
	client.heartbeatInterval = 10 * time.Millisecond

	responses := []string{
		`{"success":true,"steps":[{"index":0,"nodeId":"node-1","type":"wait","success":true,"durationMs":500}]}`,
	}

	var call int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(35 * time.Millisecond)
		if call >= len(responses) {
			t.Fatalf("unexpected additional call %d", call)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responses[call]))
		call++
	}))
	defer server.Close()

	client.browserless = server.URL
	client.httpClient = server.Client()

	exec := &database.Execution{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		StartedAt:  time.Now(),
	}

	workflow := &database.Workflow{
		ID: uuid.New(),
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{
					"id":   "node-1",
					"type": "wait",
					"data": map[string]any{
						"type":     "time",
						"duration": 1000,
					},
				},
			},
			"edges": []any{},
		},
	}

	emitter := &fakeEmitter{}

	if err := client.ExecuteWorkflow(context.Background(), exec, workflow, emitter); err != nil {
		t.Fatalf("ExecuteWorkflow returned error: %v", err)
	}

	// Allow asynchronous heartbeat goroutines to flush.
	time.Sleep(15 * time.Millisecond)

	emittedEvents := emitter.Events()

	if len(emittedEvents) == 0 {
		t.Fatalf("expected events to be emitted")
	}

	firstHeartbeat := -1
	firstCompletion := -1
	heartbeatCount := 0

	for idx, evt := range emittedEvents {
		switch evt.Type {
		case events.EventStepHeartbeat:
			heartbeatCount++
			if firstHeartbeat == -1 {
				firstHeartbeat = idx
			}
			if evt.Progress == nil {
				t.Fatalf("heartbeat missing progress pointer")
			}
			if *evt.Progress < 0 || *evt.Progress > 100 {
				t.Fatalf("heartbeat progress out of range: %d", *evt.Progress)
			}
			if evt.Payload == nil {
				t.Fatalf("heartbeat missing payload")
			}
			if _, ok := evt.Payload["elapsed_ms"]; !ok {
				t.Fatalf("heartbeat payload missing elapsed_ms: %+v", evt.Payload)
			}
			if evt.Message == "" || !strings.Contains(evt.Message, "wait in progress") {
				t.Fatalf("heartbeat message unexpected: %s", evt.Message)
			}
		case events.EventStepCompleted:
			if firstCompletion == -1 {
				firstCompletion = idx
			}
		}
	}

	if heartbeatCount == 0 {
		t.Fatalf("expected at least one heartbeat event, got %d", heartbeatCount)
	}

	if firstCompletion != -1 && firstHeartbeat != -1 && firstHeartbeat > firstCompletion {
		t.Fatalf("heartbeat emitted after completion: %+v", emittedEvents)
	}
}

func TestExecuteWorkflowRoutesToFailureBranch(t *testing.T) {
	client, repo := newTestClient()

	responses := []string{
		`{"success":true,"steps":[{"index":0,"nodeId":"node-start","type":"navigate","success":true,"durationMs":1200}]}`,
		`{"success":true,"steps":[{"index":1,"nodeId":"node-assert","type":"assert","success":false,"durationMs":400,"error":"assertion failed","assertion":{"mode":"exists","selector":"#status","success":false,"message":"assertion failed"}}]}`,
		`{"success":true,"steps":[{"index":2,"nodeId":"node-failure","type":"screenshot","success":true,"durationMs":600,"screenshotBase64":"ZmFrZQ=="}]}`,
	}

	var call int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if call >= len(responses) {
			t.Fatalf("unexpected additional call %d", call)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responses[call]))
		call++
	}))
	defer server.Close()

	client.browserless = server.URL
	client.httpClient = server.Client()

	exec := &database.Execution{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		StartedAt:  time.Now(),
	}

	workflow := &database.Workflow{
		ID: uuid.New(),
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{
					"id":   "node-start",
					"type": "navigate",
					"data": map[string]any{
						"url": "https://example.com",
					},
				},
				map[string]any{
					"id":   "node-assert",
					"type": "assert",
					"data": map[string]any{
						"assertMode":        "exists",
						"selector":          "#status",
						"continueOnFailure": true,
					},
				},
				map[string]any{
					"id":   "node-success",
					"type": "screenshot",
					"data": map[string]any{
						"name": "success",
					},
				},
				map[string]any{
					"id":   "node-failure",
					"type": "screenshot",
					"data": map[string]any{
						"name": "failure",
					},
				},
			},
			"edges": []any{
				map[string]any{"id": "edge-1", "source": "node-start", "target": "node-assert"},
				map[string]any{"id": "edge-2", "source": "node-assert", "target": "node-success", "data": map[string]any{"condition": "success"}},
				map[string]any{"id": "edge-3", "source": "node-assert", "target": "node-failure", "data": map[string]any{"condition": "failure"}},
			},
		},
	}

	if err := client.ExecuteWorkflow(context.Background(), exec, workflow, nil); err != nil {
		t.Fatalf("ExecuteWorkflow returned error: %v", err)
	}

	if call != len(responses) {
		t.Fatalf("expected %d browserless calls, got %d", len(responses), call)
	}

	if exec.Result == nil {
		t.Fatalf("execution result not persisted")
	}
	if success, ok := exec.Result["success"].(bool); !ok {
		t.Fatalf("execution result missing success flag: %+v", exec.Result)
	} else if success {
		t.Fatalf("expected overall success to be false when failure branch executes")
	}

	if len(repo.executionSteps) != 3 {
		t.Fatalf("expected 3 execution steps, got %d", len(repo.executionSteps))
	}

	var seenFailure, seenSuccess bool
	for _, step := range repo.executionSteps {
		switch step.NodeID {
		case "node-failure":
			seenFailure = true
		case "node-success":
			seenSuccess = true
		}
	}

	if !seenFailure {
		t.Fatalf("expected failure branch to execute, got steps: %+v", repo.executionSteps)
	}
	if seenSuccess {
		t.Fatalf("success branch should be skipped on assertion failure")
	}
}

func TestExecuteWorkflowContinuesWithoutFailureEdge(t *testing.T) {
	client, repo := newTestClient()

	responses := []string{
		`{"success":true,"steps":[{"index":0,"nodeId":"node-start","type":"navigate","success":true,"durationMs":800}]}`,
		`{"success":true,"steps":[{"index":1,"nodeId":"node-assert","type":"assert","success":false,"durationMs":350,"error":"not visible","assertion":{"mode":"exists","selector":"#status","success":false,"message":"not visible"}}]}`,
		`{"success":true,"steps":[{"index":2,"nodeId":"node-after","type":"wait","success":true,"durationMs":150}]}`,
	}

	var call int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if call >= len(responses) {
			t.Fatalf("unexpected additional call %d", call)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responses[call]))
		call++
	}))
	defer server.Close()

	client.browserless = server.URL
	client.httpClient = server.Client()

	exec := &database.Execution{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		StartedAt:  time.Now(),
	}

	workflow := &database.Workflow{
		ID: uuid.New(),
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{
					"id":   "node-start",
					"type": "navigate",
					"data": map[string]any{
						"url": "https://example.com",
					},
				},
				map[string]any{
					"id":   "node-assert",
					"type": "assert",
					"data": map[string]any{
						"assertMode":        "exists",
						"selector":          "#status",
						"continueOnFailure": true,
					},
				},
				map[string]any{
					"id":   "node-after",
					"type": "wait",
					"data": map[string]any{
						"type":     "time",
						"duration": 100,
					},
				},
			},
			"edges": []any{
				map[string]any{"id": "edge-1", "source": "node-start", "target": "node-assert"},
				map[string]any{"id": "edge-2", "source": "node-assert", "target": "node-after"},
			},
		},
	}

	if err := client.ExecuteWorkflow(context.Background(), exec, workflow, nil); err != nil {
		t.Fatalf("ExecuteWorkflow returned error: %v", err)
	}

	if call != len(responses) {
		t.Fatalf("expected %d browserless calls, got %d", len(responses), call)
	}
	if len(repo.executionSteps) != 3 {
		t.Fatalf("expected 3 execution steps recorded, got %d", len(repo.executionSteps))
	}

	if exec.Result == nil {
		t.Fatalf("execution result missing")
	}
	if success, ok := exec.Result["success"].(bool); !ok {
		t.Fatalf("execution result missing success key: %+v", exec.Result)
	} else if success {
		t.Fatalf("expected overall success to be false when intermediate assertion fails")
	}

	var hasAfter bool
	for _, step := range repo.executionSteps {
		if step.NodeID == "node-after" {
			hasAfter = true
		}
	}
	if !hasAfter {
		t.Fatalf("expected node-after to execute after assertion failure")
	}
}

func TestExecuteWorkflowStopsOnFatalFailure(t *testing.T) {
	client, repo := newTestClient()

	responses := []string{
		`{"success":true,"steps":[{"index":0,"nodeId":"node-start","type":"navigate","success":true,"durationMs":400}]}`,
		`{"success":false,"error":"element missing","steps":[{"index":1,"nodeId":"node-extract","type":"extract","success":false,"durationMs":120,"error":"element missing"}]}`,
	}

	var call int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if call >= len(responses) {
			t.Fatalf("unexpected additional call %d", call)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responses[call]))
		call++
	}))
	defer server.Close()

	client.browserless = server.URL
	client.httpClient = server.Client()

	exec := &database.Execution{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		StartedAt:  time.Now(),
	}

	workflow := &database.Workflow{
		ID: uuid.New(),
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{
					"id":   "node-start",
					"type": "navigate",
					"data": map[string]any{
						"url": "https://example.com",
					},
				},
				map[string]any{
					"id":   "node-extract",
					"type": "extract",
					"data": map[string]any{
						"selector": "#headline",
					},
				},
			},
			"edges": []any{
				map[string]any{"id": "edge-1", "source": "node-start", "target": "node-extract"},
			},
		},
	}

	if err := client.ExecuteWorkflow(context.Background(), exec, workflow, nil); err == nil {
		t.Fatalf("expected fatal failure to bubble up as error")
	}

	if call != len(responses) {
		t.Fatalf("expected %d browserless calls, got %d", len(responses), call)
	}
	if len(repo.executionSteps) != 2 {
		t.Fatalf("expected 2 execution steps recorded, got %d", len(repo.executionSteps))
	}
	if exec.Result == nil {
		t.Fatalf("execution result missing")
	}
	if success, ok := exec.Result["success"].(bool); ok && success {
		t.Fatalf("expected overall success to be false on fatal failure")
	}
}

func TestNewClientHeartbeatIntervalFromEnv(t *testing.T) {
	t.Setenv("BROWSERLESS_URL", "http://localhost:1234")
	t.Setenv("BROWSERLESS_HEARTBEAT_INTERVAL", "150ms")

	log := logrus.New()
	log.SetOutput(io.Discard)

	repo := newMockRepository()
	client := NewClient(log, repo)

	if client.heartbeatInterval != 150*time.Millisecond {
		t.Fatalf("expected heartbeat interval 150ms, got %s", client.heartbeatInterval)
	}
}

func TestNewClientHeartbeatIntervalBounds(t *testing.T) {
	t.Setenv("BROWSERLESS_URL", "http://localhost:1234")

	log := logrus.New()
	log.SetOutput(io.Discard)
	repo := newMockRepository()

	t.Run("clamps to minimum", func(t *testing.T) {
		t.Setenv("BROWSERLESS_HEARTBEAT_INTERVAL", "10ms")
		client := NewClient(log, repo)
		if client.heartbeatInterval != minHeartbeatInterval {
			t.Fatalf("expected heartbeat interval %s, got %s", minHeartbeatInterval, client.heartbeatInterval)
		}
	})

	t.Run("disables when zero", func(t *testing.T) {
		t.Setenv("BROWSERLESS_HEARTBEAT_INTERVAL", "0")
		client := NewClient(log, repo)
		if client.heartbeatInterval != 0 {
			t.Fatalf("expected disabled heartbeat interval, got %s", client.heartbeatInterval)
		}
	})

	t.Run("caps to maximum", func(t *testing.T) {
		t.Setenv("BROWSERLESS_HEARTBEAT_INTERVAL", "24h")
		client := NewClient(log, repo)
		if client.heartbeatInterval != maxHeartbeatInterval {
			t.Fatalf("expected heartbeat interval %s, got %s", maxHeartbeatInterval, client.heartbeatInterval)
		}
	})
}

func TestBuildInstructionsUnsupportedNode(t *testing.T) {
	client, _ := newTestClient()

	workflow := &database.Workflow{
		ID: uuid.New(),
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{
					"id":   "node-1",
					"type": "custom",
					"data": map[string]any{},
				},
			},
		},
	}

	if _, err := client.buildInstructions(workflow); err == nil {
		t.Fatalf("expected error for unsupported node type")
	}
}

func TestBuildStepScript(t *testing.T) {
	instr := runtime.Instruction{Index: 0, NodeID: "node-1", Type: "navigate", Params: runtime.InstructionParam{URL: "https://example.com"}}

	script, err := runtime.BuildStepScript(instr)
	if err != nil {
		t.Fatalf("buildStepScript failed: %v", err)
	}

	if !strings.Contains(script, "https://example.com") {
		t.Errorf("script does not contain instruction payload: %s", script)
	}

	if strings.Contains(script, "__INSTRUCTION__") {
		t.Errorf("placeholder not replaced in script: %s", script)
	}
}

func TestStoreScreenshotFallback(t *testing.T) {
	recordingsRoot := t.TempDir()
	t.Setenv("BAS_RECORDINGS_ROOT", recordingsRoot)

	client, repo := newTestClient()
	client.storage = nil

	exec := &database.Execution{ID: uuid.New()}
	step := runtime.StepResult{Index: 0, Type: "screenshot", StepName: "test"}

	encoded := base64.StdEncoding.EncodeToString([]byte{137, 80, 78, 71, 13, 10, 26, 10})
	step.ScreenshotBase64 = encoded

	rawScreenshot, decodeErr := base64.StdEncoding.DecodeString(encoded)
	if decodeErr != nil {
		t.Fatalf("failed to decode test screenshot payload: %v", decodeErr)
	}
	record, err := client.persistScreenshot(context.Background(), exec, step, rawScreenshot)
	if err != nil {
		t.Fatalf("persistScreenshot failed: %v", err)
	}
	if record == nil {
		t.Fatalf("expected screenshot record")
	}

	if len(repo.screenshots) != 1 {
		t.Fatalf("expected screenshot record to be persisted")
	}

	storageURL := repo.screenshots[0].StorageURL
	expectedPrefix := fmt.Sprintf("/api/v1/recordings/assets/%s/frames/", exec.ID.String())
	if !strings.HasPrefix(storageURL, expectedPrefix) {
		t.Fatalf("unexpected storage URL: %s", storageURL)
	}

	framesDir := filepath.Join(client.recordingsRoot, exec.ID.String(), "frames")
	entries, err := os.ReadDir(framesDir)
	if err != nil {
		t.Fatalf("failed to read frames directory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one frame asset, found %d", len(entries))
	}
	savedPath := filepath.Join(framesDir, entries[0].Name())
	if _, err := os.Stat(savedPath); err != nil {
		t.Fatalf("replay frame not persisted: %v", err)
	}
}

func TestIsLowInformationScreenshot(t *testing.T) {
	client, _ := newTestClient()

	whiteImg := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			whiteImg.Set(x, y, color.White)
		}
	}
	var whiteBuf bytes.Buffer
	if err := png.Encode(&whiteBuf, whiteImg); err != nil {
		t.Fatalf("failed to encode white image: %v", err)
	}

	if !client.IsLowInformationScreenshotForTesting(whiteBuf.Bytes()) {
		t.Fatal("expected white image to be low-information")
	}

	multiImg := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			if (x+y)%2 == 0 {
				multiImg.Set(x, y, color.Black)
			} else {
				multiImg.Set(x, y, color.RGBA{R: 120, G: 200, B: 150, A: 255})
			}
		}
	}
	var multiBuf bytes.Buffer
	if err := png.Encode(&multiBuf, multiImg); err != nil {
		t.Fatalf("failed to encode multi-color image: %v", err)
	}

	if client.IsLowInformationScreenshotForTesting(multiBuf.Bytes()) {
		t.Fatal("expected multi-color image to have sufficient information")
	}
}

func TestCheckBrowserlessHealth(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/pressure" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, _ := newTestClient()
		client.browserless = server.URL
		client.httpClient = server.Client()

		if err := client.CheckBrowserlessHealth(); err != nil {
			t.Fatalf("expected healthy status, got error: %v", err)
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		}))
		defer server.Close()

		client, _ := newTestClient()
		client.browserless = server.URL
		client.httpClient = server.Client()

		if err := client.CheckBrowserlessHealth(); err == nil {
			t.Fatalf("expected error for failing health check")
		}
	})
}

func TestNewClient(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	repo := newMockRepository()
	t.Setenv("BROWSERLESS_URL", "http://localhost:3000")

	client := NewClient(log, repo)
	if client == nil {
		t.Fatalf("expected client instance")
	}

	if client.httpClient == nil {
		t.Fatalf("expected http client to be configured")
	}
}
