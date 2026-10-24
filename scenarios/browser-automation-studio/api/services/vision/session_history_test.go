package vision

import (
	"context"
	"errors"
	"io"
	"testing"
	"testing/synctest"
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

func TestNavigationSession_Snapshot_IsolatesExtractedData(t *testing.T) {
	live := &NavigationSession{ExtractedData: map[string]interface{}{
		"nested": map[string]interface{}{"value": "original"},
		"items":  []interface{}{map[string]interface{}{"value": "original"}},
	}}
	snapshot := live.Snapshot()
	snapshot.ExtractedData["nested"].(map[string]interface{})["value"] = "changed"
	snapshot.ExtractedData["items"].([]interface{})[0].(map[string]interface{})["value"] = "changed"

	if got := live.ExtractedData["nested"].(map[string]interface{})["value"]; got != "original" {
		t.Errorf("nested extracted data leaked through snapshot: %v", got)
	}
	if got := live.ExtractedData["items"].([]interface{})[0].(map[string]interface{})["value"]; got != "original" {
		t.Errorf("slice extracted data leaked through snapshot: %v", got)
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
	synctest.Test(t, func(t *testing.T) {
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
		canceled, cancel := context.WithCancel(context.Background())
		cancel()
		if err := (&playwrightNavigationHandle{navigator: nav, session: snap}).Wait(canceled); !errors.Is(err, context.Canceled) {
			t.Fatalf("Wait with canceled context = %v, want context.Canceled", err)
		}

		done := make(chan error, 1)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			done <- (&playwrightNavigationHandle{navigator: nav, session: snap}).Wait(ctx)
		}()
		// Ensure Wait has reached its channel wait before completing the session.
		synctest.Wait()

		if err := nav.HandleCompleteCallback(context.Background(), &NavigationResult{NavigationID: "nav_wait", Status: StatusCompleted, TotalSteps: 2}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if err := <-done; err != nil {
			t.Fatalf("Wait returned %v", err)
		}
		select {
		case <-snap.Changed():
		default:
			t.Fatal("snapshot channel not closed by completion")
		}
		// Complete the navigator's scheduled five-minute cleanup timer in virtual time.
		time.Sleep(5 * time.Minute)
		synctest.Wait()
		if err := (&playwrightNavigationHandle{navigator: nav, session: snap}).Wait(context.Background()); err != nil {
			t.Fatalf("Wait after cleanup = %v, want nil", err)
		}
	})
}
