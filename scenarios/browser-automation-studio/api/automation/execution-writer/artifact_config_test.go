package executionwriter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/storage"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/config"
)

// TestPerExecutionArtifactConfigIsolation proves that two concurrent executions
// configured with different artifact profiles do not leak settings into each
// other through the shared recorder.
func TestPerExecutionArtifactConfigIsolation(t *testing.T) {
	writer := NewFileWriter(noopRepo{}, nil, nil, NewStaticRoot(t.TempDir()))

	execA := uuid.New()
	execB := uuid.New()

	full := config.DefaultArtifactSettingsForProfile(config.ProfileFull)
	none := config.DefaultArtifactSettingsForProfile(config.ProfileNone)

	writer.SetArtifactConfigForExecution(execA, &full)
	writer.SetArtifactConfigForExecution(execB, &none)

	if got := writer.artifactConfigForExecution(execA); !got.CollectDOMSnapshots {
		t.Fatalf("execA should use full profile (DOM snapshots on)")
	}
	if got := writer.artifactConfigForExecution(execB); got.CollectScreenshots {
		t.Fatalf("execB should use none profile (no screenshots)")
	}

	// Hammer both concurrently to surface any shared-state race under -race.
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			if got := writer.artifactConfigForExecution(execA); !got.CollectDOMSnapshots {
				t.Errorf("execA config leaked: DOM snapshots off")
			}
		}()
		go func() {
			defer wg.Done()
			if got := writer.artifactConfigForExecution(execB); got.CollectScreenshots {
				t.Errorf("execB config leaked: screenshots on")
			}
		}()
	}
	wg.Wait()
}

// TestForgetExecutionFallsBackToWriterDefault proves that once an execution is
// forgotten its config no longer applies and the writer-wide default is used.
func TestForgetExecutionFallsBackToWriterDefault(t *testing.T) {
	writer := NewFileWriter(noopRepo{}, nil, nil, NewStaticRoot(t.TempDir()))

	exec := uuid.New()
	none := config.DefaultArtifactSettingsForProfile(config.ProfileNone)
	writer.SetArtifactConfigForExecution(exec, &none)

	if got := writer.artifactConfigForExecution(exec); got.CollectScreenshots {
		t.Fatalf("expected none profile before forget")
	}

	writer.ForgetExecution(exec)

	// After forgetting, falls back to the writer-wide default (full from constructor).
	if got := writer.artifactConfigForExecution(exec); !got.CollectScreenshots {
		t.Fatalf("expected writer default (full) after forget, got screenshots off")
	}
}

// TestSetArtifactConfigForExecutionNilClears proves passing nil clears the
// per-execution override.
func TestSetArtifactConfigForExecutionNilClears(t *testing.T) {
	writer := NewFileWriter(noopRepo{}, nil, nil, NewStaticRoot(t.TempDir()))
	exec := uuid.New()
	minimal := config.DefaultArtifactSettingsForProfile(config.ProfileMinimal)
	writer.SetArtifactConfigForExecution(exec, &minimal)
	writer.SetArtifactConfigForExecution(exec, nil)
	if got := writer.artifactConfigForExecution(exec); !got.CollectDOMSnapshots {
		t.Fatalf("expected writer default (full) after nil clear")
	}
}

func TestForgetPreservesDurableAndOtherActiveExecution(t *testing.T) {
	root := t.TempDir()
	writer := NewFileWriter(nil, nil, nil, NewStaticRoot(root))
	completed := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
	active := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
	for _, plan := range []contracts.ExecutionPlan{completed, active} {
		_, err := writer.RecordStepOutcome(context.Background(), plan, contracts.StepOutcome{ExecutionID: plan.ExecutionID, StepIndex: 0, Attempt: 1, NodeID: "first", StepType: "navigate", Success: true})
		require.NoError(t, err)
	}
	completedPath := filepath.Join(root, completed.ExecutionID.String(), resultFileName)
	before, err := os.ReadFile(completedPath)
	require.NoError(t, err)
	writer.ForgetExecution(completed.ExecutionID)
	writer.ForgetExecution(completed.ExecutionID)
	_, err = writer.RecordStepOutcome(context.Background(), active, contracts.StepOutcome{ExecutionID: active.ExecutionID, StepIndex: 1, Attempt: 1, NodeID: "second", StepType: "click", Success: true})
	require.NoError(t, err)
	after, err := os.ReadFile(completedPath)
	require.NoError(t, err)
	require.Equal(t, before, after)
	activeBytes, err := os.ReadFile(filepath.Join(root, active.ExecutionID.String(), resultFileName))
	require.NoError(t, err)
	var timeline bastimeline.ExecutionTimeline
	require.NoError(t, (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(activeBytes, &timeline))
	require.Len(t, timeline.Entries, 2)
	require.Equal(t, "first", timeline.Entries[0].GetNodeId())
	require.Equal(t, "second", timeline.Entries[1].GetNodeId())
}

// Retained heap is observational evidence, not a timing-sensitive unit gate.
// The store is on disk so stored artifacts do not masquerade as writer retention.
func BenchmarkFinishedExecutionRetainedHeap(b *testing.B) {
	root := b.TempDir()
	store, err := storage.NewFileStorage(filepath.Join(root, "objects"), nil)
	if err != nil {
		b.Fatal(err)
	}
	writer := NewFileWriter(nil, store, nil, NewStaticRoot(root))
	settings := config.DefaultArtifactSettings()
	settings.MaxDOMSnapshotBytes = 2 * 1024 * 1024
	writer.SetArtifactConfig(&settings)
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
		html := strings.Repeat(fmt.Sprintf("<div>capture-%d</div>", i), 32768)
		_, err := writer.RecordStepOutcome(context.Background(), plan, contracts.StepOutcome{ExecutionID: plan.ExecutionID, StepIndex: 0, Attempt: 1, NodeID: "capture", StepType: "navigate", Success: true, DOMSnapshot: &contracts.DOMSnapshot{HTML: html}})
		if err != nil {
			b.Fatal(err)
		}
		writer.ForgetExecution(plan.ExecutionID)
	}
	b.StopTimer()
	runtime.GC()
	runtime.ReadMemStats(&after)
	b.ReportMetric(float64(int64(after.HeapAlloc)-int64(before.HeapAlloc))/float64(b.N), "retained-B/execution")
	runtime.KeepAlive(writer)
}
