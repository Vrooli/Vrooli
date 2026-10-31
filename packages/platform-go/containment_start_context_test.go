package platform

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestContainmentContextChildHelper(t *testing.T) {
	if os.Getenv("GO_WANT_CONTAINMENT_CONTEXT_CHILD") != "1" {
		return
	}
	time.Sleep(30 * time.Second)
	os.Exit(0)
}

func containmentContextChild() *exec.Cmd {
	cmd := exec.Command(os.Args[0], "-test.run=^TestContainmentContextChildHelper$")
	cmd.Env = append(os.Environ(), "GO_WANT_CONTAINMENT_CONTEXT_CHILD=1")
	return cmd
}

func TestContainedStartContextCancelledBeforeSpawn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := &Contained{Cmd: containmentContextChild()}
	if err := c.StartContext(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if c.Cmd.Process != nil {
		t.Fatal("cancelled acquisition spawned a process")
	}
}

func TestContainedStartContextReapsRootAndReportsUnknownDescendants(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	c := &Contained{Cmd: containmentContextChild(), afterContext: func(ctx context.Context, _ *Contained) error { <-ctx.Done(); return ctx.Err() }}
	started := time.Now()
	err := c.StartContext(ctx)
	if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, ErrContainmentCleanupUnverified) {
		t.Fatalf("post-spawn acquisition failure lost cleanup uncertainty: %v", err)
	}
	if c.Cmd.ProcessState == nil {
		t.Fatal("known root was not joined")
	}
	if time.Since(started) > 2*time.Second {
		t.Fatal("failed acquisition cleanup exceeded bounded test budget")
	}
}
