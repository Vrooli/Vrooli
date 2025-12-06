package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	sharedartifacts "test-genie/internal/shared/artifacts"
)

// TraceEventType identifies the type of trace event.
type TraceEventType string

const (
	// TracePhaseStart indicates the playbooks phase has started.
	TracePhaseStart TraceEventType = "phase_start"
	// TracePhaseComplete indicates the playbooks phase completed successfully.
	TracePhaseComplete TraceEventType = "phase_complete"
	// TracePhaseFailed indicates the playbooks phase failed.
	TracePhaseFailed TraceEventType = "phase_failed"

	// TraceWorkflowStart indicates a workflow execution has started.
	TraceWorkflowStart TraceEventType = "workflow_start"
	// TraceWorkflowQueued indicates a workflow was submitted to BAS.
	TraceWorkflowQueued TraceEventType = "workflow_queued"
	// TraceWorkflowProgress indicates progress on a workflow.
	TraceWorkflowProgress TraceEventType = "workflow_progress"
	// TraceWorkflowComplete indicates a workflow completed successfully.
	TraceWorkflowComplete TraceEventType = "workflow_complete"
	// TraceWorkflowFailed indicates a workflow failed.
	TraceWorkflowFailed TraceEventType = "workflow_failed"

	// TraceSeedApply indicates seeds are being applied.
	TraceSeedApply TraceEventType = "seed_apply"
	// TraceSeedCleanup indicates seeds are being cleaned up.
	TraceSeedCleanup TraceEventType = "seed_cleanup"

	// TraceBASHealth indicates a BAS health check.
	TraceBASHealth TraceEventType = "bas_health"
	// TraceBASStart indicates BAS is being started.
	TraceBASStart TraceEventType = "bas_start"
)

// TraceEvent represents a single trace event in the execution log.
type TraceEvent struct {
	Timestamp    time.Time      `json:"ts"`
	Event        TraceEventType `json:"event"`
	WorkflowFile string         `json:"workflow,omitempty"`
	ExecutionID  string         `json:"execution_id,omitempty"`
	StepIndex    int            `json:"step,omitempty"`
	NodeID       string         `json:"node_id,omitempty"`
	Status       string         `json:"status,omitempty"`
	Progress     float64        `json:"progress,omitempty"`
	Duration     string         `json:"duration,omitempty"`
	Error        string         `json:"error,omitempty"`
	Message      string         `json:"message,omitempty"`
	Details      map[string]any `json:"details,omitempty"`
}

// TraceWriter writes execution trace events to a JSONL file.
type TraceWriter interface {
	// Write records a trace event.
	Write(event TraceEvent) error
	// Close flushes and closes the trace file.
	Close() error
	// Path returns the path to the trace file.
	Path() string
}

// FileTraceWriter writes trace events to a JSONL file on disk.
type FileTraceWriter struct {
	mu      sync.Mutex
	file    *os.File
	path    string
	encoder *json.Encoder
	closed  bool
}

// NewTraceWriter creates a new trace writer for a scenario.
// The trace file is written to coverage/automation/<scenario>-<timestamp>.trace.jsonl
func NewTraceWriter(scenarioDir, scenarioName string) (*FileTraceWriter, error) {
	targetDir := filepath.Join(scenarioDir, sharedartifacts.AutomationDir)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create trace directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.trace.jsonl", scenarioName, timestamp)
	path := filepath.Join(targetDir, filename)

	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace file: %w", err)
	}

	return &FileTraceWriter{
		file:    file,
		path:    path,
		encoder: json.NewEncoder(file),
	}, nil
}

// Write records a trace event to the file.
func (w *FileTraceWriter) Write(event TraceEvent) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return fmt.Errorf("trace writer is closed")
	}

	// Set timestamp if not already set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	if err := w.encoder.Encode(event); err != nil {
		return fmt.Errorf("failed to write trace event: %w", err)
	}

	// Flush to ensure events are written immediately
	return w.file.Sync()
}

// Close flushes and closes the trace file.
func (w *FileTraceWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}

	w.closed = true
	return w.file.Close()
}

// Path returns the path to the trace file.
func (w *FileTraceWriter) Path() string {
	return w.path
}

// NullTraceWriter is a no-op trace writer for when tracing is disabled.
type NullTraceWriter struct{}

// Write is a no-op.
func (w *NullTraceWriter) Write(event TraceEvent) error { return nil }

// Close is a no-op.
func (w *NullTraceWriter) Close() error { return nil }

// Path returns empty string.
func (w *NullTraceWriter) Path() string { return "" }

// Convenience methods for common trace events

// TracePhaseStartEvent creates a phase start event.
func TracePhaseStartEvent(workflowCount int) TraceEvent {
	return TraceEvent{
		Event:   TracePhaseStart,
		Message: fmt.Sprintf("starting playbooks phase with %d workflows", workflowCount),
		Details: map[string]any{"workflow_count": workflowCount},
	}
}

// TracePhaseCompleteEvent creates a phase complete event.
func TracePhaseCompleteEvent(passed, failed int, duration time.Duration) TraceEvent {
	return TraceEvent{
		Event:    TracePhaseComplete,
		Duration: duration.String(),
		Message:  fmt.Sprintf("phase complete: %d passed, %d failed", passed, failed),
		Details:  map[string]any{"passed": passed, "failed": failed},
	}
}

// TracePhaseFailedEvent creates a phase failed event.
func TracePhaseFailedEvent(err error, duration time.Duration) TraceEvent {
	return TraceEvent{
		Event:    TracePhaseFailed,
		Duration: duration.String(),
		Error:    err.Error(),
	}
}

// TraceWorkflowStartEvent creates a workflow start event.
func TraceWorkflowStartEvent(workflowFile string) TraceEvent {
	return TraceEvent{
		Event:        TraceWorkflowStart,
		WorkflowFile: workflowFile,
		Message:      fmt.Sprintf("starting workflow: %s", workflowFile),
	}
}

// TraceWorkflowQueuedEvent creates a workflow queued event.
func TraceWorkflowQueuedEvent(workflowFile, executionID string) TraceEvent {
	return TraceEvent{
		Event:        TraceWorkflowQueued,
		WorkflowFile: workflowFile,
		ExecutionID:  executionID,
		Message:      fmt.Sprintf("workflow queued with execution ID: %s", executionID),
	}
}

// TraceWorkflowProgressEvent creates a workflow progress event.
func TraceWorkflowProgressEvent(workflowFile, executionID string, progress float64, currentStep string) TraceEvent {
	return TraceEvent{
		Event:        TraceWorkflowProgress,
		WorkflowFile: workflowFile,
		ExecutionID:  executionID,
		Progress:     progress,
		Status:       currentStep,
		Message:      fmt.Sprintf("progress: %.0f%% - %s", progress*100, currentStep),
	}
}

// TraceWorkflowCompleteEvent creates a workflow complete event.
func TraceWorkflowCompleteEvent(workflowFile, executionID string, duration time.Duration, stats string) TraceEvent {
	return TraceEvent{
		Event:        TraceWorkflowComplete,
		WorkflowFile: workflowFile,
		ExecutionID:  executionID,
		Duration:     duration.String(),
		Message:      fmt.Sprintf("workflow complete%s", stats),
	}
}

// TraceWorkflowFailedEvent creates a workflow failed event.
func TraceWorkflowFailedEvent(workflowFile, executionID string, err error, duration time.Duration) TraceEvent {
	return TraceEvent{
		Event:        TraceWorkflowFailed,
		WorkflowFile: workflowFile,
		ExecutionID:  executionID,
		Duration:     duration.String(),
		Error:        err.Error(),
	}
}

// TraceBASHealthEvent creates a BAS health check event.
func TraceBASHealthEvent(healthy bool, endpoint string) TraceEvent {
	status := "healthy"
	if !healthy {
		status = "unhealthy"
	}
	return TraceEvent{
		Event:   TraceBASHealth,
		Status:  status,
		Message: fmt.Sprintf("BAS health check: %s at %s", status, endpoint),
		Details: map[string]any{"endpoint": endpoint},
	}
}

// TraceSeedApplyEvent creates a seed apply event.
func TraceSeedApplyEvent(scriptPath string) TraceEvent {
	return TraceEvent{
		Event:   TraceSeedApply,
		Message: fmt.Sprintf("applying seeds: %s", scriptPath),
		Details: map[string]any{"script": scriptPath},
	}
}

// TraceSeedCleanupEvent creates a seed cleanup event.
func TraceSeedCleanupEvent(scriptPath string) TraceEvent {
	return TraceEvent{
		Event:   TraceSeedCleanup,
		Message: fmt.Sprintf("cleaning up seeds: %s", scriptPath),
		Details: map[string]any{"script": scriptPath},
	}
}
