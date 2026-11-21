package pipeline

import "testing"

func TestPipelineStatusTransitions(t *testing.T) {
	tests := []struct {
		name string
		from PipelineStatus
		next PipelineStatus
		want PipelineStatus
		ok   bool
	}{
		{name: "idle starts", from: StatusIdle, next: StatusPending, want: StatusPending, ok: true},
		{name: "pending executes", from: StatusPending, next: StatusRunning, want: StatusRunning, ok: true},
		{name: "running completes", from: StatusRunning, next: StatusCompleted, want: StatusCompleted, ok: true},
		{name: "running fails", from: StatusRunning, next: StatusFailed, want: StatusFailed, ok: true},
		{name: "pending recovery fails", from: StatusPending, next: StatusFailed, want: StatusFailed, ok: true},
		{name: "running cancels", from: StatusRunning, next: StatusCancelled, want: StatusCancelled, ok: true},
		{name: "stale recovery resolves cancelled worker as failed", from: StatusCancelled, next: StatusFailed, want: StatusFailed, ok: true},
		{name: "same-state checkpoint is allowed", from: StatusRunning, next: StatusRunning, want: StatusRunning, ok: true},
		{name: "cannot complete before execution", from: StatusPending, next: StatusCompleted, want: StatusPending, ok: false},
		{name: "terminal state cannot reopen", from: StatusCompleted, next: StatusRunning, want: StatusCompleted, ok: false},
		{name: "unknown persisted state cannot advance", from: "unknown", next: StatusCompleted, want: "unknown", ok: false},
		{name: "pipeline cannot use stage skipped state", from: StatusPending, next: StatusSkipped, want: StatusPending, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := &Status{Status: tt.from}
			err := status.TransitionStatus(tt.next)
			if (err == nil) != tt.ok {
				t.Fatalf("TransitionStatus(%q) error = %v, want success=%v", tt.next, err, tt.ok)
			}
			if status.Status != tt.want {
				t.Fatalf("status = %q, want %q", status.Status, tt.want)
			}
		})
	}
}

func TestPipelineStatusTransitionPersistsAndRejectsTerminalReversal(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	store.Save(&Status{PipelineID: "pipeline-1", Status: StatusRunning})
	if !store.Update("pipeline-1", func(status *Status) {
		if err := status.TransitionStatus(StatusCompleted); err != nil {
			t.Errorf("complete pipeline: %v", err)
		}
	}) {
		t.Fatal("expected pipeline update")
	}

	reloaded, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	status, ok := reloaded.Get("pipeline-1")
	if !ok || status.Status != StatusCompleted {
		t.Fatalf("reloaded status = %q, found=%v; want completed", status.Status, ok)
	}

	var transitionErr error
	reloaded.Update("pipeline-1", func(status *Status) {
		transitionErr = status.TransitionStatus(StatusRunning)
	})
	if transitionErr == nil {
		t.Fatal("expected terminal reversal to be rejected")
	}
	status, _ = reloaded.Get("pipeline-1")
	if status.Status != StatusCompleted {
		t.Fatalf("invalid transition changed status to %q", status.Status)
	}
}

func TestRecoverStalePipelineUsesStatusTransitionOwner(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	store.Save(&Status{
		PipelineID: "interrupted",
		Status:     StatusRunning,
		Stages: map[string]*StageResult{
			StageBuild: {Stage: StageBuild, Status: StatusRunning},
		},
	})
	manager := NewManager(WithManagerOrchestrator(NewOrchestrator(WithStore(store))))
	if recovered := manager.RecoverStalePipelines(); recovered != 1 {
		t.Fatalf("recovered count = %d, want 1", recovered)
	}

	reloaded, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	status, ok := reloaded.Get("interrupted")
	if !ok {
		t.Fatal("recovered pipeline was not persisted")
	}
	if status.Status != StatusFailed || status.Stages[StageBuild].Status != StatusFailed {
		t.Fatalf("recovery status = %q, build stage = %q; want failed/failed", status.Status, status.Stages[StageBuild].Status)
	}
}
