package export

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/storage"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/encoding/protojson"
)

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
		return timelineEntryToFrame(retained.Entries[0])
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
