package paths_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/internal/paths"
)

func TestRecordingsRootProviderRoutesExecutionFilesToLeasedTestRoots(t *testing.T) {
	primary := t.TempDir()
	t.Setenv("BAS_RECORDINGS_ROOT", filepath.Join(primary, "recordings"))

	provider, err := paths.NewRecordingsRootProvider(nil)
	if err != nil {
		t.Fatalf("new recordings root provider: %v", err)
	}
	testData := t.TempDir()
	leaseID := "recordings-routing-test"
	if err := provider.FileRoots().InstallTestRoots(storage.Paths{DataDir: testData}, leaseID, time.Minute); err != nil {
		t.Fatalf("install test roots: %v", err)
	}
	t.Cleanup(func() { _ = provider.FileRoots().ClearTestRoots(leaseID) })

	writer := executionwriter.NewFileWriter(nil, nil, nil, provider)
	plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
	ctx := coredb.WithTestMode(context.Background())
	if err := writer.RecordTelemetry(ctx, plan, contracts.StepTelemetry{StepIndex: 1, Note: "routed artifact"}); err != nil {
		t.Fatalf("record telemetry: %v", err)
	}

	primaryResult := filepath.Join(primary, "recordings", plan.ExecutionID.String(), "result.json")
	if _, err := os.Stat(primaryResult); !os.IsNotExist(err) {
		t.Fatalf("primary result exists during test mode: %v", err)
	}
	testResult := filepath.Join(testData, "recordings", plan.ExecutionID.String(), "result.json")
	if _, err := os.Stat(testResult); err != nil {
		t.Fatalf("routed result missing: %v", err)
	}
	testTimeline := filepath.Join(testData, "recordings", plan.ExecutionID.String(), "timeline.proto.json")
	if _, err := os.Stat(testTimeline); err != nil {
		t.Fatalf("routed timeline missing: %v", err)
	}
	primaryTimeline := filepath.Join(primary, "recordings", plan.ExecutionID.String(), "timeline.proto.json")
	if _, err := os.Stat(primaryTimeline); !os.IsNotExist(err) {
		t.Fatalf("primary timeline exists during test mode: %v", err)
	}
	stats := provider.FileRoots().LeaseStats()
	if stats.TestRootWrites == 0 || stats.PrimaryWritesDuringTestMode != 0 {
		t.Fatalf("unexpected routed write stats: %+v", stats)
	}
}
