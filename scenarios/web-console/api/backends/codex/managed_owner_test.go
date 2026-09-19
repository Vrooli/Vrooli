package codex

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestManagedOwnerForkRetainsOriginalBranch(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"result":{"threadId":"thread-2"}}` + "\n"
	c := NewClient(writeCloser{io.Discard}, strings.NewReader(input))
	o, err := NewManagedOwner(c, "web-console:s1", "thread-1")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := o.ForkThrough(context.Background(), "turn-4"); err != nil || got != "thread-2" {
		t.Fatalf("fork = %q, %v", got, err)
	}
	if o.OriginalThreadID() != "thread-1" || o.ThreadID() != "thread-2" || o.LastVerifiedTurnID() != "turn-4" {
		t.Fatalf("owner lineage not retained: original=%q active=%q turn=%q", o.OriginalThreadID(), o.ThreadID(), o.LastVerifiedTurnID())
	}
}

func TestManagedOwnerRejectsMissingBoundary(t *testing.T) {
	c := NewClient(writeCloser{io.Discard}, strings.NewReader(""))
	o, err := NewManagedOwner(c, "owner", "thread")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := o.ForkThrough(context.Background(), ""); err != ErrMissingTurn {
		t.Fatalf("error = %v", err)
	}
}

func TestManagedOwnerRecordsOnlyCompletedTurnForControl(t *testing.T) {
	o, err := NewManagedOwner(NewClient(writeCloser{io.Discard}, strings.NewReader("")), "owner", "thread")
	if err != nil {
		t.Fatal(err)
	}
	if o.MarkTurnCompleted("other-thread", "turn-1") {
		t.Fatal("foreign thread completion was accepted")
	}
	if !o.MarkTurnCompleted("thread", "turn-2") || o.LastVerifiedTurnID() != "turn-2" {
		t.Fatalf("owner = %+v", o)
	}
}

func TestManagedOwnerRestoresPersistedForkLineage(t *testing.T) {
	o, err := NewManagedOwner(NewClient(writeCloser{io.Discard}, strings.NewReader("")), "owner", "thread-active")
	if err != nil {
		t.Fatal(err)
	}
	if err := o.RestoreLineage("thread-original", "turn-4"); err != nil {
		t.Fatal(err)
	}
	if o.ThreadID() != "thread-active" || o.OriginalThreadID() != "thread-original" || o.LastVerifiedTurnID() != "turn-4" {
		t.Fatalf("restored lineage = active=%q original=%q turn=%q", o.ThreadID(), o.OriginalThreadID(), o.LastVerifiedTurnID())
	}
}
