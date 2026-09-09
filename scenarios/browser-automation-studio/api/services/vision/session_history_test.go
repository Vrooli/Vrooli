package vision

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// ---- NavigationSession primitives -----------------------------------------

func TestNavigationStatus_Terminal(t *testing.T) {
	terminal := []NavigationStatus{StatusCompleted, StatusFailed, StatusAborted, StatusMaxSteps, StatusLoopDetected, StatusAwaitingHuman}
	for _, s := range terminal {
		if !s.Terminal() {
			t.Errorf("%q should be terminal", s)
		}
	}
	for _, s := range []NavigationStatus{StatusIdle, StatusNavigating, ""} {
		if s.Terminal() {
			t.Errorf("%q should not be terminal", s)
		}
	}
}

func TestNavigationSession_RecordStep_BoundedHistory(t *testing.T) {
	s := &NavigationSession{}
	for i := 1; i <= MaxRecordedSteps+25; i++ {
		s.RecordStep(NavigationStepRecord{Index: i, ActionType: "click"})
	}
	if got := len(s.Steps); got != MaxRecordedSteps {
		t.Fatalf("history length = %d, want cap %d", got, MaxRecordedSteps)
	}
	// Oldest entries are dropped: the first surviving step is #26.
	if s.Steps[0].Index != 26 {
		t.Errorf("first surviving step index = %d, want 26", s.Steps[0].Index)
	}
	if last := s.Steps[len(s.Steps)-1].Index; last != MaxRecordedSteps+25 {
		t.Errorf("last step index = %d, want %d", last, MaxRecordedSteps+25)
	}
}

func TestNavigationSession_RecordStep_DefaultsIndexAndTime(t *testing.T) {
	s := &NavigationSession{}
	before := time.Now()
	s.RecordStep(NavigationStepRecord{ActionType: "navigate", URL: "https://example.com"})
	s.RecordStep(NavigationStepRecord{ActionType: "click"})
	if s.Steps[0].Index != 1 || s.Steps[1].Index != 2 {
		t.Errorf("indexes = %d,%d want 1,2", s.Steps[0].Index, s.Steps[1].Index)
	}
	if s.Steps[0].At.Before(before) {
		t.Errorf("At was not defaulted to now")
	}
}

func TestNavigationSession_SetStatus_WakesSnapshotWaiters(t *testing.T) {
	live := &NavigationSession{Status: StatusNavigating}
	snap := live.Snapshot()

	select {
	case <-snap.Changed():
		t.Fatal("Changed() closed before any transition")
	default:
	}

	live.SetStatus(StatusCompleted)

	select {
	case <-snap.Changed():
	case <-time.After(time.Second):
		t.Fatal("snapshot's Changed() channel was not closed on transition")
	}
	if live.Status != StatusCompleted {
		t.Errorf("status = %q, want completed", live.Status)
	}
	// A fresh snapshot after the transition carries a new, open channel.
	select {
	case <-live.Snapshot().Changed():
		t.Fatal("post-transition snapshot channel should be open")
	default:
	}
}

func TestNavigationSession_Snapshot_IsolatesSteps(t *testing.T) {
	live := &NavigationSession{}
	live.RecordStep(NavigationStepRecord{ActionType: "click"})
	snap := live.Snapshot()
	live.RecordStep(NavigationStepRecord{ActionType: "type"})
	live.Steps[0].ActionType = "mutated"
	if len(snap.Steps) != 1 || snap.Steps[0].ActionType != "click" {
		t.Errorf("snapshot leaked live mutations: %+v", snap.Steps)
	}
}

// ---- Playwright navigator records history ---------------------------------

func TestPlaywrightVisionNavigator_RecordsStepHistory(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewPlaywrightVisionNavigator(log)

	session := &NavigationSession{
		NavigationID:  "nav_hist",
		SessionID:     "sess_hist",
		StartedAt:     time.Now(),
		Status:        StatusNavigating,
		NavigatorType: NavigatorPlaywright,
	}
	nav.mu.Lock()
	nav.activeNavigations["nav_hist"] = session
	nav.mu.Unlock()

	steps := []*NavigationStep{
		{NavigationID: "nav_hist", StepNumber: 1, Action: map[string]interface{}{"type": "navigate", "url": "https://example.com/login"}, Reasoning: "open login", CurrentURL: "about:blank"},
		{NavigationID: "nav_hist", StepNumber: 2, Action: map[string]interface{}{"type": "type", "selector": "#user", "text": "alice"}, Reasoning: "fill user", CurrentURL: "https://example.com/login"},
		{NavigationID: "nav_hist", StepNumber: 3, Action: map[string]interface{}{"type": "click", "selector": "button[type=submit]"}, Reasoning: "submit", CurrentURL: "https://example.com/login", Error: "element not visible"},
	}
	for _, st := range steps {
		if err := nav.HandleStepCallback(context.Background(), st); err != nil {
			t.Fatalf("HandleStepCallback: %v", err)
		}
	}

	snap, ok := nav.GetSession("nav_hist")
	if !ok {
		t.Fatal("session missing")
	}
	if len(snap.Steps) != 3 {
		t.Fatalf("recorded %d steps, want 3", len(snap.Steps))
	}
	if s := snap.Steps[0]; s.Index != 1 || s.ActionType != "navigate" || s.URL != "https://example.com/login" || !s.Success {
		t.Errorf("step 1 = %+v", s)
	}
	if s := snap.Steps[1]; s.ActionType != "type" || s.Selector != "#user" || s.Value != "alice" || s.URL != "https://example.com/login" || s.Description != "fill user" {
		t.Errorf("step 2 = %+v", s)
	}
	if s := snap.Steps[2]; s.ActionType != "click" || s.Success || s.Error != "element not visible" {
		t.Errorf("step 3 = %+v", s)
	}
	if snap.StepCount != 3 {
		t.Errorf("StepCount = %d, want 3", snap.StepCount)
	}
}

func TestPlaywrightVisionNavigator_CompleteCallbackWakesWaiter(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewPlaywrightVisionNavigator(log)

	nav.mu.Lock()
	nav.activeNavigations["nav_wait"] = &NavigationSession{NavigationID: "nav_wait", SessionID: "s", Status: StatusNavigating}
	nav.mu.Unlock()

	snap, _ := nav.GetSession("nav_wait")
	if snap.Status.Terminal() {
		t.Fatal("should start non-terminal")
	}

	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		done <- (&playwrightNavigationHandle{navigator: nav, session: snap}).Wait(ctx)
	}()

	time.Sleep(20 * time.Millisecond)
	if err := nav.HandleCompleteCallback(context.Background(), &NavigationResult{NavigationID: "nav_wait", Status: StatusCompleted, TotalSteps: 2}); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Wait returned %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handle.Wait did not return after completion callback")
	}
	select {
	case <-snap.Changed():
	default:
		t.Fatal("snapshot channel not closed by completion")
	}
}

// ---- Claude Code navigator records history --------------------------------

func TestClaudeCodeVisionNavigator_RecordsStepHistory(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewClaudeCodeVisionNavigator(log)

	session := &claudeCodeSession{
		NavigationSession: &NavigationSession{
			NavigationID: "nav_cc",
			SessionID:    "sess_cc",
			Status:       StatusNavigating,
			StartedAt:    time.Now(),
		},
		doneChan: make(chan struct{}),
	}
	nav.mu.Lock()
	nav.activeNavigations["nav_cc"] = session
	nav.mu.Unlock()

	snapBefore, _ := nav.GetSession("nav_cc")

	stream := `{"type":"assistant","content":[{"type":"text","text":"open the site"}]}
{"type":"tool_use","name":"mcp__claude-in-chrome__navigate","input":{"url":"https://example.com"}}
{"type":"assistant","content":[{"type":"text","text":"type the name"}]}
{"type":"tool_use","name":"mcp__claude-in-chrome__form_input","input":{"ref":"ref_12","text":"alice"}}
{"type":"tool_use","name":"mcp__claude-in-chrome__computer","input":{"action":"left_click","coordinate":[10,20]}}
{"type":"result","result":"done"}
`
	nav.parseOutput(session, bytes.NewReader([]byte(stream)))

	snap, ok := nav.GetSession("nav_cc")
	if !ok {
		t.Fatal("session missing")
	}
	if snap.Status != StatusCompleted {
		t.Fatalf("status = %q, want completed", snap.Status)
	}
	if len(snap.Steps) != 3 {
		t.Fatalf("recorded %d steps, want 3: %+v", len(snap.Steps), snap.Steps)
	}
	if s := snap.Steps[0]; s.Index != 1 || s.ActionType != "navigate" || s.URL != "https://example.com" || s.Description != "open the site" {
		t.Errorf("step 1 = %+v", s)
	}
	if s := snap.Steps[1]; s.Index != 2 || s.ActionType != "type" || s.Selector != "ref_12" || s.Value != "alice" || s.Description != "type the name" {
		t.Errorf("step 2 = %+v", s)
	}
	if s := snap.Steps[2]; s.Index != 3 || s.ActionType != "click" || s.Selector != "[10, 20]" {
		t.Errorf("step 3 = %+v", s)
	}
	if snap.StepCount != 3 {
		t.Errorf("StepCount = %d, want 3", snap.StepCount)
	}
	// The "result" event is a terminal transition and must wake waiters.
	select {
	case <-snapBefore.Changed():
	default:
		t.Error("pre-completion snapshot channel was not closed by the result event")
	}
}

// ---- MultiTracker -----------------------------------------------------------

type stubTracker struct {
	sessions map[string]*NavigationSession
	aborted  []string
}

func (s *stubTracker) GetSession(id string) (*NavigationSession, bool) {
	sess, ok := s.sessions[id]
	if !ok {
		return nil, false
	}
	return sess.Snapshot(), true
}

func (s *stubTracker) AbortNavigation(_ context.Context, id string) error {
	s.aborted = append(s.aborted, id)
	return nil
}

func (s *stubTracker) ResumeNavigation(_ context.Context, id string) error {
	return fmt.Errorf("resume not supported")
}

func TestMultiTracker_RoutesToOwner(t *testing.T) {
	a := &stubTracker{sessions: map[string]*NavigationSession{"nav_a": {NavigationID: "nav_a", NavigatorType: NavigatorPlaywright}}}
	b := &stubTracker{sessions: map[string]*NavigationSession{"nav_b": {NavigationID: "nav_b", NavigatorType: NavigatorClaudeCode}}}
	m := MultiTracker{a, b}

	if s, ok := m.GetSession("nav_b"); !ok || s.NavigatorType != NavigatorClaudeCode {
		t.Fatalf("GetSession(nav_b) = %+v, %v", s, ok)
	}
	if _, ok := m.GetSession("nav_zzz"); ok {
		t.Fatal("unknown id should not resolve")
	}
	if err := m.AbortNavigation(context.Background(), "nav_b"); err != nil {
		t.Fatal(err)
	}
	if len(a.aborted) != 0 || len(b.aborted) != 1 {
		t.Errorf("abort routed wrong: a=%v b=%v", a.aborted, b.aborted)
	}
	if err := m.AbortNavigation(context.Background(), "nav_zzz"); err == nil {
		t.Error("abort of unknown id should error")
	}
	if err := m.ResumeNavigation(context.Background(), "nav_zzz"); err == nil {
		t.Error("resume of unknown id should error")
	}
}
