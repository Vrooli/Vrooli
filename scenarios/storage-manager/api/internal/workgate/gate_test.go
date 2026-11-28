package workgate

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestQuiesceWaitsAndFencesUntilResume(t *testing.T) {
	var g Gate
	started, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() { defer close(finished); _ = g.Run(func() error { close(started); <-release; return nil }) }()
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := g.Quiesce(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("quiesce: %v", err)
	}
	called := false
	_ = g.Run(func() error { called = true; return nil })
	if called {
		t.Fatal("admitted while quiesced")
	}
	close(release)
	<-finished
	if err := g.Quiesce(context.Background()); err != nil {
		t.Fatal(err)
	}
	g.Resume()
	_ = g.Run(func() error { called = true; return nil })
	if !called {
		t.Fatal("did not resume")
	}
}

func TestConcurrentAdmissionsDrain(t *testing.T) {
	var g Gate
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _ = g.Run(func() error { return nil }) }()
	}
	if err := g.Quiesce(context.Background()); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	if n := g.Active(); n != 0 {
		t.Fatalf("active = %d", n)
	}
}

func TestOneShotWaitsForResumeAndHonorsCancellation(t *testing.T) {
	var g Gate
	if err := g.Quiesce(context.Background()); err != nil {
		t.Fatal(err)
	}
	ran := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- g.RunWhenOpen(context.Background(), func() error { close(ran); return nil }) }()
	select {
	case <-ran:
		t.Fatal("one-shot ran while fenced")
	default:
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := g.RunWhenOpen(ctx, func() error { t.Fatal("cancelled task ran"); return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled task: %v", err)
	}
	g.Resume()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("one-shot did not resume")
	}
	select {
	case <-ran:
	default:
		t.Fatal("one-shot was lost")
	}
}

func TestResumeWakesWaitingOneShot(t *testing.T) {
	var g Gate
	if err := g.Quiesce(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- g.RunWhenOpen(ctx, func() error { return nil }) }()
	deadline := time.Now().Add(time.Second)
	for {
		g.mu.Lock()
		waiting := g.changed != nil
		g.mu.Unlock()
		if waiting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("one-shot did not enter wait")
		}
		runtime.Gosched()
	}
	g.Resume()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("resume did not wake waiting one-shot")
	}
}
