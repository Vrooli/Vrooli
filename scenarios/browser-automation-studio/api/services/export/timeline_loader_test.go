package export

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/storage"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type timelineLoaderTestRepository struct {
	execution *database.ExecutionIndex
	workflow  *database.WorkflowIndex
}

func TestPresentationFramePreservesRetryAssertionAndDuration(t *testing.T) {
	stepIndex, duration, total := int32(3), int32(240), int32(710)
	entry := &bastimeline.TimelineEntry{
		StepIndex: &stepIndex, DurationMs: &duration, TotalDurationMs: &total,
		Context: &basbase.EventContext{
			Success: proto.Bool(false), Error: proto.String("retry exhausted"),
			RetryStatus: &basbase.RetryStatus{
				CurrentAttempt: 2, MaxAttempts: 3, DelayMs: 125, BackoffFactor: 1.5, Configured: true,
				History: []*basbase.RetryAttempt{{Attempt: 1, DurationMs: 180, Error: proto.String("timeout")}},
			},
			Assertion: &basbase.AssertionResult{Selector: "#ready", Success: false, Negated: true, Message: proto.String("not ready")},
		},
	}
	frame := presentationFrameForTimelineEntry(entry)
	assert.Equal(t, 3, frame.StepIndex)
	assert.Equal(t, 240, frame.DurationMs)
	assert.Equal(t, 710, frame.TotalDurationMs)
	assert.Equal(t, 2, frame.RetryAttempt)
	assert.Equal(t, 3, frame.RetryMaxAttempts)
	assert.Equal(t, 2, frame.RetryConfigured)
	assert.Equal(t, 125, frame.RetryDelayMs)
	assert.Equal(t, 1.5, frame.RetryBackoffFactor)
	require.Len(t, frame.RetryHistory, 1)
	assert.Equal(t, "timeout", frame.RetryHistory[0].Error)
	require.NotNil(t, frame.Assertion)
	assert.Equal(t, "#ready", frame.Assertion.Selector)
	assert.True(t, frame.Assertion.Negated)
	assert.Equal(t, "retry exhausted", frame.Error)
	assert.False(t, frame.Success)
}

func TestBuildExecutionTimelinePresentationKeepsProgressAndLogs(t *testing.T) {
	execution := &database.ExecutionIndex{ID: uuid.New(), WorkflowID: uuid.New(), Status: "running"}
	started := time.Date(2026, 10, 1, 7, 0, 0, 0, time.UTC)
	presentation := BuildExecutionTimelinePresentation(execution, &bastimeline.ExecutionTimeline{
		Progress: 65,
		Logs:     []*bastimeline.TimelineLog{{Id: "log-1", Level: basbase.LogLevel_LOG_LEVEL_INFO, Message: "step started", Timestamp: timestamppb.New(started)}},
	})
	assert.Equal(t, 65, presentation.Progress)
	require.Len(t, presentation.Logs, 1)
	assert.Equal(t, "log-1", presentation.Logs[0].ID)
	assert.Equal(t, "info", presentation.Logs[0].Level)
	assert.Equal(t, started, presentation.Logs[0].Timestamp)
}

func TestBuildExecutionTimelinePresentationLoadsViewportAndDeviceScale(t *testing.T) {
	resultPath := filepath.Join(t.TempDir(), "result.json")
	raw := []byte(`{"parameters":{"viewportWidth":1440,"viewportHeight":900,"browserProfile":{"fingerprint":{"deviceScaleFactor":2}}}}`)
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(resultPath), "execution.proto.json"), raw, 0o600))
	execution := &database.ExecutionIndex{ID: uuid.New(), WorkflowID: uuid.New(), Status: "completed", ResultPath: resultPath}
	presentation := BuildExecutionTimelinePresentation(execution, &bastimeline.ExecutionTimeline{})
	assert.Equal(t, 1440, presentation.ViewportWidth)
	assert.Equal(t, 900, presentation.ViewportHeight)
	assert.Equal(t, 2.0, presentation.DeviceScaleFactor)
}

func (r timelineLoaderTestRepository) GetExecution(_ context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	if r.execution.ID != id {
		return nil, database.ErrNotFound
	}
	return r.execution, nil
}

func (r timelineLoaderTestRepository) GetWorkflow(context.Context, uuid.UUID) (*database.WorkflowIndex, error) {
	return r.workflow, nil
}

func TestTimelineLoaderUsesWorkflowCaptureGeometryWhenExecutionHasNoOverride(t *testing.T) {
	dir := t.TempDir()
	workflowPath := filepath.Join(dir, "workflow.json")
	require.NoError(t, os.WriteFile(workflowPath, []byte(`{"settings":{"viewportWidth":1440,"viewportHeight":900,"browserProfile":{"fingerprint":{"deviceScaleFactor":2}}}}`), 0o600))
	execution := &database.ExecutionIndex{ID: uuid.New(), WorkflowID: uuid.New(), Status: "completed", ResultPath: filepath.Join(dir, "result.json")}
	workflow := &database.WorkflowIndex{ID: execution.WorkflowID, FilePath: workflowPath}
	loader := NewTimelineLoader(timelineLoaderTestRepository{execution: execution, workflow: workflow})
	presentation, err := loader.LoadTimeline(context.Background(), execution.ID)
	require.NoError(t, err)
	assert.Equal(t, 1440, presentation.ViewportWidth)
	assert.Equal(t, 900, presentation.ViewportHeight)
	assert.Equal(t, 2.0, presentation.DeviceScaleFactor)
}

func TestTimelineLoaderRedactsSensitiveValuesFromLegacyProtoFiles(t *testing.T) {
	const ordinary = "ordinary preserved input"
	executionID := uuid.New()
	resultPath := filepath.Join(t.TempDir(), "result.json")
	entry := func(id, kind, autocomplete, value string) *bastimeline.TimelineEntry {
		attributes := map[string]string{"type": kind, "value": value}
		if autocomplete != "" {
			attributes["autocomplete"] = autocomplete
		}
		return &bastimeline.TimelineEntry{
			Id: id,
			Action: &basactions.ActionDefinition{
				Type: basactions.ActionType_ACTION_TYPE_INPUT,
				Params: &basactions.ActionDefinition_Input{Input: &basactions.InputParams{
					Selector: "#" + id,
					Value:    value,
				}},
				Metadata: &basactions.ActionMetadata{ElementSnapshot: &basdomain.ElementMeta{
					TagName:    "input",
					InnerText:  value,
					Attributes: attributes,
				}},
			},
		}
	}
	sensitive := []*bastimeline.TimelineEntry{
		entry("password-entry", "password", "", "BAS_SYNTHETIC_PASSWORD_SECRET_017"),
		entry("hidden-entry", "hidden", "", "BAS_SYNTHETIC_HIDDEN_SECRET_017"),
		entry("otp-entry", "text", "one-time-code", "BAS_SYNTHETIC_OTP_SECRET_017"),
		entry("payment-entry", "text", "cc-number", "BAS_SYNTHETIC_PAYMENT_SECRET_017"),
	}
	plain := entry("text-entry", "text", "", ordinary)
	entries := append(sensitive, plain)
	timeline := &bastimeline.ExecutionTimeline{Entries: entries}
	legacyTimeline, err := protojson.Marshal(timeline)
	require.NoError(t, err)
	packageBytes, err := protojson.Marshal(&basevidence.ReplayPackage{Timeline: entries})
	require.NoError(t, err)

	dir := filepath.Dir(resultPath)
	timelinePath := filepath.Join(dir, "timeline.proto.json")
	packagePath := filepath.Join(dir, "evidence.proto.json")
	require.NoError(t, os.WriteFile(resultPath, []byte(`{}`), 0o600))
	require.NoError(t, os.WriteFile(timelinePath, legacyTimeline, 0o600))
	require.NoError(t, os.WriteFile(packagePath, packageBytes, 0o600))

	loader := NewTimelineLoader(timelineLoaderTestRepository{execution: &database.ExecutionIndex{
		ID: executionID, WorkflowID: uuid.New(), ResultPath: resultPath,
	}})
	loadedTimeline, err := loader.LoadTimelineProto(context.Background(), executionID)
	require.NoError(t, err)
	loadedPackage, err := loader.LoadReplayPackage(context.Background(), executionID)
	require.NoError(t, err)

	secrets := []string{"BAS_SYNTHETIC_PASSWORD_SECRET_017", "BAS_SYNTHETIC_HIDDEN_SECRET_017", "BAS_SYNTHETIC_OTP_SECRET_017", "BAS_SYNTHETIC_PAYMENT_SECRET_017"}
	for _, check := range []struct {
		name    string
		message proto.Message
	}{
		{name: "timeline", message: loadedTimeline},
		{name: "replay package", message: loadedPackage},
	} {
		encoded, marshalErr := protojson.Marshal(check.message)
		require.NoError(t, marshalErr)
		assert.Contains(t, string(encoded), ordinary, "%s must preserve ordinary input", check.name)
		for _, secret := range secrets {
			assert.NotContains(t, string(encoded), secret, "%s exposed a saved sensitive value", check.name)
		}
	}
	for i := range secrets {
		assert.Empty(t, loadedTimeline.Entries[i].Action.GetInput().GetValue())
		assert.Empty(t, loadedPackage.Timeline[i].Action.GetInput().GetValue())
	}
	for _, path := range []string{timelinePath, packagePath} {
		stored, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		for _, secret := range secrets {
			assert.Contains(t, string(stored), secret, "legacy bytes on disk must remain unchanged")
		}
	}
}

// [REQ:BAS-RH-J24] Capture/export reads the same retained condition evidence.
func TestRetainedConditionExportsWithoutLosingFields(t *testing.T) {
	persistAndExport := func(outcome contracts.StepOutcome) TimelineFrame {
		t.Helper()
		root := t.TempDir()
		plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
		writer := executionwriter.NewFileWriter(nil, storage.NewMemoryStorage(), nil, executionwriter.NewStaticRoot(root))
		outcome.ExecutionID = plan.ExecutionID
		_, err := writer.RecordStepOutcome(context.Background(), plan, outcome)
		require.NoError(t, err)
		writer.ForgetExecution(plan.ExecutionID)
		data, err := os.ReadFile(filepath.Join(root, plan.ExecutionID.String(), "timeline.proto.json"))
		require.NoError(t, err)
		var retained bastimeline.ExecutionTimeline
		require.NoError(t, protojson.Unmarshal(data, &retained))
		require.Len(t, retained.Entries, 1)
		return presentationFrameForTimelineEntry(retained.Entries[0])
	}
	for _, truth := range []bool{false, true} {
		condition := &contracts.ConditionOutcome{
			Type: "expression", Outcome: truth, Negated: !truth,
			Expression: "return true;", Actual: map[string]any{"ready": false}, Expected: true,
			Variable: "answer", Operator: "equals", Selector: "#ready",
		}
		frame := persistAndExport(contracts.StepOutcome{
			StepType: "conditional", Success: true, Condition: condition,
		})
		require.NotNil(t, frame.Condition)
		assert.Equal(t, condition, frame.Condition)
		assert.True(t, frame.Success, "false condition remains a successfully evaluated step")
	}
	frame := persistAndExport(contracts.StepOutcome{
		StepType: "conditional", Failure: &contracts.StepFailure{Message: "evaluation failed"},
	})
	assert.Nil(t, frame.Condition, "an evaluator error cannot acquire a false condition")
	assert.False(t, frame.Success)
	assert.Equal(t, "evaluation failed", frame.Error)
}
