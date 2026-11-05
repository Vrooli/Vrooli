package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// childRunsFixture is an in-memory parent/child run set for the children waiter.
type childRunsFixture struct {
	mu     sync.Mutex
	parent *domain.Run
	kids   []*domain.Run
}

func (f *childRunsFixture) GetRun(_ context.Context, id uuid.UUID) (*domain.Run, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.parent != nil && f.parent.ID == id {
		clone := *f.parent
		return &clone, nil
	}
	return nil, domain.NewNotFoundError("Run", id)
}

func (f *childRunsFixture) ListRuns(_ context.Context, opts RunListOptions) ([]*domain.Run, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	rows := []*domain.Run{}
	for _, kid := range f.kids {
		if opts.ParentRunID != nil && kid.ParentRunID != nil && *kid.ParentRunID == *opts.ParentRunID {
			clone := *kid
			rows = append(rows, &clone)
		}
	}
	return rows, nil
}

func (f *childRunsFixture) finish(i int, status domain.RunStatus, at time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.kids[i].Status = status
	f.kids[i].EndedAt = &at
}

// parkedParentWithChildren builds a parent whose previous wake was at lastWake;
// terminal children ended one minute before it, so they were already reported.
func parkedParentWithChildren(lastWake time.Time, kids ...domain.RunStatus) *childRunsFixture {
	parentID := uuid.New()
	started := lastWake.Add(-time.Hour)
	fixture := &childRunsFixture{parent: &domain.Run{
		ID: parentID, Status: domain.RunStatusParked, StartedAt: &started, LastAwaitResolvedAt: &lastWake,
		AwaitHandle: &domain.AwaitHandle{Producer: ProducerChildren, Key: parentID.String(), RegisteredAt: lastWake.Add(time.Minute)},
	}}
	for _, status := range kids {
		kid := &domain.Run{ID: uuid.New(), ParentRunID: &parentID, Status: status}
		if status.IsTerminal() {
			ended := lastWake.Add(-time.Minute)
			kid.EndedAt = &ended
		}
		fixture.kids = append(fixture.kids, kid)
	}
	return fixture
}

func decodeChildrenPayload(t *testing.T, raw string) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("payload is not JSON: %v (%q)", err, raw)
	}
	return payload
}

func TestChildrenWaiterWakesWhenAChildEndsAfterThePreviousWake(t *testing.T) {
	// One child ended before the previous wake, so it was already reported.
	fixture := parkedParentWithChildren(time.Now(), domain.RunStatusComplete, domain.RunStatusRunning)
	waiter := NewChildrenWaiter(fixture, 5*time.Millisecond)

	done := make(chan string, 1)
	go func() {
		result, err := waiter.Wait(context.Background(), fixture.parent.ID.String())
		if err != nil {
			t.Errorf("wait failed: %v", err)
		}
		done <- result
	}()
	select {
	case result := <-done:
		t.Fatalf("parent woke before any child ended after the park: %s", result)
	case <-time.After(40 * time.Millisecond):
	}

	fixture.finish(1, domain.RunStatusFailed, time.Now())
	select {
	case result := <-done:
		payload := decodeChildrenPayload(t, result)
		ended := payload["ended"].([]any)
		if payload["wake_reason"] != "child_run_ended" || len(ended) != 1 || ended[0].(map[string]any)["run_id"] != fixture.kids[1].ID.String() {
			t.Fatalf("unexpected wake payload: %s", result)
		}
		if active := payload["active"].([]any); len(active) != 0 {
			t.Fatalf("active = %v, want none", active)
		}
		if _, listed := payload["children"]; listed {
			t.Fatalf("wake payload must not list children reported at earlier wakes: %s", result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("parent did not wake after its child ended")
	}
}

func TestChildrenWaiterTimerWakeCarriesChildStatus(t *testing.T) {
	fixture := parkedParentWithChildren(time.Now(), domain.RunStatusRunning)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	result, err := NewChildrenWaiter(fixture, 5*time.Millisecond).Wait(ctx, fixture.parent.ID.String())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timer wake error = %v, want deadline exceeded", err)
	}
	payload := decodeChildrenPayload(t, result)
	active := payload["active"].([]any)
	if payload["wake_reason"] != "timer" || len(active) != 1 || active[0].(map[string]any)["run_id"] != fixture.kids[0].ID.String() {
		t.Fatalf("timer payload must report the still-running child: %s", result)
	}
}

func TestChildrenWaiterWakesAtOnceForAChildThatEndedBeforeThePark(t *testing.T) {
	// The child finished while the parent was still working toward its park
	// (and, equally, while agent-manager was down): the parent was never told.
	lastWake := time.Now().Add(-time.Hour)
	fixture := parkedParentWithChildren(lastWake, domain.RunStatusRunning)
	fixture.finish(0, domain.RunStatusComplete, lastWake.Add(time.Minute))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result, err := NewChildrenWaiter(fixture, 5*time.Millisecond).Wait(ctx, fixture.parent.ID.String())
	if err != nil || decodeChildrenPayload(t, result)["wake_reason"] != "child_run_ended" {
		t.Fatalf("parent missed an unreported child: %v %s", err, result)
	}
}

func TestChildrenWaiterFallsBackToTheParentStartBeforeAnyWake(t *testing.T) {
	fixture := parkedParentWithChildren(time.Now(), domain.RunStatusRunning)
	fixture.parent.LastAwaitResolvedAt = nil
	fixture.finish(0, domain.RunStatusFailed, fixture.parent.StartedAt.Add(time.Second))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if result, err := NewChildrenWaiter(fixture, 5*time.Millisecond).Wait(ctx, fixture.parent.ID.String()); err != nil {
		t.Fatalf("a child that ended after the parent started was not reported on the first park: %v %s", err, result)
	}
}

func TestChildrenWaiterRejectsAKeyThatIsNotARunID(t *testing.T) {
	if _, err := NewChildrenWaiter(&childRunsFixture{}, time.Millisecond).Wait(context.Background(), "bas/goal"); err == nil {
		t.Fatal("a non-UUID key must be refused")
	}
}

func TestChildrenWakeMessageStaysSmall(t *testing.T) {
	// Codex keeps every wake message through compaction, so an orchestrator
	// with many past children must not pay for them on every wake.
	fixture := parkedParentWithChildren(time.Now(), domain.RunStatusComplete, domain.RunStatusComplete, domain.RunStatusComplete, domain.RunStatusRunning)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	payload, _ := NewChildrenWaiter(fixture, 5*time.Millisecond).Wait(ctx, fixture.parent.ID.String())
	handle := &domain.AwaitHandle{Producer: ProducerChildren, Key: fixture.parent.ID.String()}

	msg := formatWakeMessage(fixture.parent.ID, handle, payload, true)
	if !strings.HasPrefix(msg, "[wake: timer]") || !strings.Contains(msg, fixture.kids[3].ID.String()) {
		t.Fatalf("timer wake must name the still-active child: %s", msg)
	}
	for _, reported := range fixture.kids[:3] {
		if strings.Contains(msg, reported.ID.String()) {
			t.Fatalf("timer wake repeats a child reported earlier: %s", msg)
		}
	}
	if len(msg) > 300 {
		t.Fatalf("timer wake is %d bytes; keep it under 300: %s", len(msg), msg)
	}
	if got := formatWakeMessage(fixture.parent.ID, handle, "operator note", false); got != "[wake] operator note" {
		t.Fatalf("explicit wake = %q", got)
	}
}
