package mocks

import (
	"testing"

	"agent-manager/internal/domain"
)

func TestFakeBroadcasterCopiesRunStatus(t *testing.T) {
	b := NewFakeBroadcaster()
	run := &domain.Run{Status: domain.RunStatusRunning}

	b.BroadcastRunStatus(run)
	run.Status = domain.RunStatusComplete

	got := b.StatusBroadcasts()
	if len(got) != 1 {
		t.Fatalf("expected 1 status broadcast, got %d", len(got))
	}
	if got[0].Status != domain.RunStatusRunning {
		t.Fatalf("expected status snapshot %q, got %q", domain.RunStatusRunning, got[0].Status)
	}
}
