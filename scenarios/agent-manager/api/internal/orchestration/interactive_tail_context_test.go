package orchestration

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/interactive"
	"agent-manager/internal/runstate"

	"github.com/google/uuid"
)

func TestLoadInteractiveTailContextUsesRunStateMetadata(t *testing.T) {
	root := t.TempDir()
	runID := uuid.New()
	startedAt := time.Date(2026, 10, 20, 14, 0, 0, 0, time.UTC)
	state, err := runstate.Open(runID, runstate.OpenOptions{
		RootDir:    root,
		RunnerType: domain.RunnerTypeCodex,
		WorkingDir: "/workspace/project",
		StartedAt:  startedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := state.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := loadInteractiveTailContext(context.Background(), runID, func(context.Context) (string, error) {
		return root, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	wantRunDir, err := runstate.RunDir(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	want := interactive.TailContext{RunDir: wantRunDir, WorkingDir: "/workspace/project", LaunchedAt: startedAt}
	if got != want {
		t.Fatalf("tail context = %+v, want %+v", got, want)
	}
}

func TestLoadInteractiveTailContextKeepsScopedDirectoryWhenMetadataIsMissing(t *testing.T) {
	root := t.TempDir()
	runID := uuid.New()
	got, err := loadInteractiveTailContext(context.Background(), runID, func(context.Context) (string, error) {
		return root, nil
	})
	if err == nil {
		t.Fatal("expected missing runstate metadata error")
	}
	wantRunDir, pathErr := runstate.RunDir(root, runID)
	if pathErr != nil {
		t.Fatal(pathErr)
	}
	if got.RunDir != wantRunDir {
		t.Fatalf("RunDir = %q, want safe per-run directory %q", got.RunDir, wantRunDir)
	}
}

func TestLoadInteractiveTailContextDoesNotInventDirectoryWhenRootFails(t *testing.T) {
	rootErr := errors.New("runstate root unavailable")
	got, err := loadInteractiveTailContext(context.Background(), uuid.New(), func(context.Context) (string, error) {
		return "", rootErr
	})
	if !errors.Is(err, rootErr) {
		t.Fatalf("error = %v, want root resolution error", err)
	}
	if got.RunDir != "" {
		t.Fatalf("RunDir = %q, want empty when root could not be resolved", got.RunDir)
	}
}

func TestLoadInteractiveTailContextRejectsEmptyResolvedRoot(t *testing.T) {
	got, err := loadInteractiveTailContext(context.Background(), uuid.New(), func(context.Context) (string, error) {
		return "", nil
	})
	if err == nil {
		t.Fatal("expected empty runstate root error")
	}
	if got.RunDir != "" {
		t.Fatalf("RunDir = %q, want no synthesized relative path", got.RunDir)
	}
}
