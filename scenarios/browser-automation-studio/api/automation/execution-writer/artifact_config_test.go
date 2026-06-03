package executionwriter

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/config"
)

// TestPerExecutionArtifactConfigIsolation proves that two concurrent executions
// configured with different artifact profiles do not leak settings into each
// other through the shared recorder.
func TestPerExecutionArtifactConfigIsolation(t *testing.T) {
	writer := NewFileWriter(noopRepo{}, nil, nil, t.TempDir())

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
	writer := NewFileWriter(noopRepo{}, nil, nil, t.TempDir())

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
	writer := NewFileWriter(noopRepo{}, nil, nil, t.TempDir())
	exec := uuid.New()
	minimal := config.DefaultArtifactSettingsForProfile(config.ProfileMinimal)
	writer.SetArtifactConfigForExecution(exec, &minimal)
	writer.SetArtifactConfigForExecution(exec, nil)
	if got := writer.artifactConfigForExecution(exec); !got.CollectDOMSnapshots {
		t.Fatalf("expected writer default (full) after nil clear")
	}
}
