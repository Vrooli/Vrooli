package main

import (
	"context"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"web-console/internal/backend"
	"web-console/session"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/eventbus"
)

// This file is Web Console's first out-of-band operator-notification producer.
// It turns a session's activity transitions into generic *.notification.v1
// events on Vrooli Events. notification-hub subscribes to the generic
// **.notification.v1 pattern and turns the neutral notificationFacts payload
// into an operator notification per the operator's catalog + per-session
// preferences (see scenarios/web-console/.vrooli/notifications.json) — no
// per-scenario code in the hub.
//
// Only two transitions notify, both reliable signals the activity detector
// already derives (session/activity.go): a session becoming "waiting" (a
// permission prompt / question is up) and a working-or-waiting session going
// "idle" (a turn ended). Per-message chatter is deliberately NOT a producer.
//
// Two guards keep this from spamming the operator (the detector is noisy — it
// reads a session as "idle" after ~1.5s of quiet, so an agent that pauses
// between tool calls flaps working↔idle):
//
//   - Baseline: the FIRST state we observe for a session never notifies. We
//     only notify on transitions we actually witness, never on the state a
//     session was already in when we began watching it. This kills the
//     startup/reattach burst (every recovered session evaluating its settled
//     screen at once).
//   - Dwell + cancel: a transition does not notify immediately. It schedules
//     the notice after a settle window and cancels it if the session leaves
//     that state first. A flapping working↔idle session, or a prompt the
//     operator answers quickly, never produces a notification. The dwell also
//     acts as soft focus-suppression: if you're watching a session you'll act
//     within the window, cancelling the ping.
const (
	// SessionNotificationEventType is the generic notification event a session
	// transition raises. The `.notification.v1` suffix is the contract
	// notification-hub subscribes to (**.notification.v1).
	SessionNotificationEventType = "web_console.session.notification.v1"

	// notificationSourceScenario is this scenario's slug — the notification
	// source and the self-link target for a session deep link.
	notificationSourceScenario = "web-console"

	// Notification categories. These MUST match the category keys declared in
	// scenarios/web-console/.vrooli/notifications.json so the operator's
	// catalog preferences apply.
	categoryInputNeeded   = "session.input_needed"
	categoryTurnCompleted = "session.turn_completed"

	// Settle windows a transition must persist before it notifies. A prompt the
	// operator clears within inputNeededDwell, or a turn-end that the agent
	// resumes within turnCompletedDwell, never pings.
	inputNeededDwell   = 12 * time.Second
	turnCompletedDwell = 30 * time.Second

	// minTurnActive is the shortest active (non-idle) stretch that counts as a
	// real turn. A parked agent's TUI redraws its status line about once a
	// minute; the detector reads each redraw as a ~1.5s working blip that
	// settles back to idle. Without this gate, every parked session in the
	// fleet fires a phantom "finished its turn" on that cycle — so one real
	// completion arrives buried under N-1 false ones. A genuine turn keeps the
	// agent working for far longer than a redraw, so require the active stretch
	// (idle → working/waiting → idle) to exceed this before notifying.
	minTurnActive = 10 * time.Second
)

// sessionNotice is one notification decided from an activity transition. It
// holds the neutral facts; the deep link (a network resolution) is attached at
// delivery, off the activity path.
type sessionNotice struct {
	SessionID string
	Category  string
	Urgency   string
	Title     string
	Body      string
	DedupeKey string
}

// timerHandle is the subset of *time.Timer the notifier needs; it lets tests
// swap in a controllable scheduler.
type timerHandle interface{ Stop() bool }

// sessionNotifier records per-session activity transitions and, after a settle
// window, publishes the two operator-worthy ones. observe is fast (map +
// timer-arm only) so it is safe to call from the detector's publish path (which
// holds the detector lock); the scheduled callback does the network work off
// that path.
type sessionNotifier struct {
	// publish sends a built event to Vrooli Events. Swapped in tests.
	publish func(ctx context.Context, event eventbus.DomainEvent) error
	// notice resolves a scenario's operator-facing (tunnel-preferring) URL for
	// the session deep link. Swapped in tests.
	notice func(ctx context.Context, scenario string) (string, error)
	// now stamps the event occurrence time. Swapped in tests.
	now func() time.Time
	// schedule arms fn to run after d and returns a handle that can cancel it.
	// Defaults to time.AfterFunc; swapped in tests for deterministic firing.
	schedule func(d time.Duration, fn func()) timerHandle

	mu      sync.Mutex
	last    map[string]session.ActivityState
	active  map[string]time.Time   // per-session time the session last left idle
	pending map[string]timerHandle // per-session not-yet-fired notice
	baseURL string                 // cached vrooli-events base URL
}

// newSessionNotifier builds a notifier that publishes to Vrooli Events and
// resolves links through Tunnel Manager.
func newSessionNotifier() *sessionNotifier {
	n := &sessionNotifier{
		notice:   discovery.ResolveExternalURLForNotice,
		now:      func() time.Time { return time.Now().UTC() },
		last:     map[string]session.ActivityState{},
		active:   map[string]time.Time{},
		pending:  map[string]timerHandle{},
		schedule: func(d time.Duration, fn func()) timerHandle { return time.AfterFunc(d, fn) },
	}
	n.publish = n.publishViaEventbus
	return n
}

// observe records the session's new activity state and arms (or cancels) the
// notice its transition warrants. It never does I/O — only a map update and a
// timer arm — so it is safe under the detector lock.
func (n *sessionNotifier) observe(sessionID string, a session.Activity) {
	if n == nil || sessionID == "" {
		return
	}
	state := a.State
	n.mu.Lock()
	defer n.mu.Unlock()

	prev, had := n.last[sessionID]
	n.last[sessionID] = state
	if !had {
		// Baseline: the first state we see for a session is the state it was
		// already in when we began watching — never a witnessed transition, so
		// never a notification. This is what suppresses the startup/reattach
		// burst.
		return
	}
	if state == prev {
		return
	}
	// A real transition supersedes any notice still settling for this session.
	n.cancelLocked(sessionID)

	// Track the active period: remember when the session leaves idle so a
	// return to idle can be measured against it.
	if prev == session.ActivityIdle && state != session.ActivityIdle {
		n.active[sessionID] = n.stamp()
	}

	notice, ok := decideNotice(sessionID, prev, a)
	if !ok {
		return
	}
	if notice.Category == categoryTurnCompleted {
		// Only a turn that kept the agent active for a real stretch counts. A
		// once-a-minute status-line redraw settles back to idle within a
		// second or two; witnessing that as a completed turn is the fleet-wide
		// phantom spam. Require a witnessed active period of at least
		// minTurnActive; if we never saw this session leave idle (e.g. it was
		// mid-turn when we started watching), stay silent rather than guess.
		start, seen := n.active[sessionID]
		delete(n.active, sessionID)
		if !seen || n.stamp().Sub(start) < minTurnActive {
			return
		}
	}
	// Defer delivery until the state settles. If the session leaves this state
	// before the dwell elapses, the next observe cancels it.
	n.pending[sessionID] = n.schedule(noticeDwell(notice.Category), func() {
		n.mu.Lock()
		delete(n.pending, sessionID)
		n.mu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		n.deliver(ctx, notice)
	})
}

// decideNotice is the pure transition→notice decision. prev is the witnessed
// prior state (observe only calls it on a real transition).
func decideNotice(sessionID string, prev session.ActivityState, a session.Activity) (sessionNotice, bool) {
	switch a.State {
	case session.ActivityWaiting:
		dedupe := sessionID + ":input_needed"
		if a.Prompt != nil {
			dedupe += ":" + backend.PromptHash(a.Prompt)
		}
		return sessionNotice{
			SessionID: sessionID,
			Category:  categoryInputNeeded,
			Urgency:   "high",
			Title:     "Session needs your input",
			Body:      inputNeededBody(a),
			DedupeKey: dedupe,
		}, true
	case session.ActivityIdle:
		// A turn actually completed only if the session was working or waiting
		// before. A settle into idle from unknown is not a turn end.
		if prev != session.ActivityWorking && prev != session.ActivityWaiting {
			return sessionNotice{}, false
		}
		return sessionNotice{
			SessionID: sessionID,
			Category:  categoryTurnCompleted,
			Urgency:   "normal",
			Title:     "Session finished its turn",
			Body:      "The agent is idle and waiting for your next instruction.",
			DedupeKey: sessionID + ":turn_completed:" + a.Since.UTC().Format(time.RFC3339Nano),
		}, true
	default:
		return sessionNotice{}, false
	}
}

// noticeDwell is how long a category's state must persist before it notifies.
func noticeDwell(category string) time.Duration {
	if category == categoryTurnCompleted {
		return turnCompletedDwell
	}
	return inputNeededDwell
}

// cancelLocked stops and drops any settling notice for a session. Caller holds n.mu.
func (n *sessionNotifier) cancelLocked(sessionID string) {
	if t := n.pending[sessionID]; t != nil {
		t.Stop()
		delete(n.pending, sessionID)
	}
}

// forget drops a closed session's transition state and any settling notice so
// the maps do not grow without bound.
func (n *sessionNotifier) forget(sessionID string) {
	if n == nil || sessionID == "" {
		return
	}
	n.mu.Lock()
	delete(n.last, sessionID)
	delete(n.active, sessionID)
	n.cancelLocked(sessionID)
	n.mu.Unlock()
}

// inputNeededBody prefers the prompt's own text so the operator sees what is
// being asked; it falls back to a generic line when no prompt text is captured.
func inputNeededBody(a session.Activity) string {
	if a.Prompt != nil {
		if text := strings.TrimSpace(a.Prompt.Text); text != "" {
			const max = 240
			if len(text) > max {
				text = text[:max] + "…"
			}
			return text
		}
	}
	return "An agent session is waiting for your response."
}

// deliver resolves the session deep link and publishes the notification event.
// It is best-effort: a publish failure is logged and dropped. Run it off the
// activity path (it does network I/O).
func (n *sessionNotifier) deliver(ctx context.Context, notice sessionNotice) {
	if n == nil || notice.SessionID == "" {
		return
	}
	payload := map[string]any{
		"title":       notice.Title,
		"body":        notice.Body,
		"category":    notice.Category,
		"urgency":     notice.Urgency,
		"sensitivity": "private",
		"dedupe_key":  notice.DedupeKey,
		// scope_key lets the operator mute/unmute notifications for THIS
		// session alone (notification-hub's per-session preference tier),
		// independent of the global category preference.
		"scope_key":   notice.SessionID,
		"correlation": map[string]any{"session_id": notice.SessionID},
	}
	if link := n.sessionLink(ctx, notice.SessionID); link != "" {
		payload["context_url"] = link
	}
	event := eventbus.DomainEvent{
		Source:    notificationSourceScenario,
		EventType: SessionNotificationEventType,
		Occurred:  n.stamp(),
		Payload:   payload,
	}
	if err := n.publish(ctx, event); err != nil {
		log.Printf("session-notification: publish %s for %s failed: %v", notice.Category, notice.SessionID, err)
	}
}

func (n *sessionNotifier) stamp() time.Time {
	if n.now != nil {
		return n.now()
	}
	return time.Now().UTC()
}

// sessionLink builds the operator-openable deep link to this session. It uses
// the tunnel-preferring notice resolver (the operator opens it on their phone),
// matching the `/?session=<id>` shape agent-manager links to. An unresolvable
// base yields no link rather than a localhost dead link.
func (n *sessionNotifier) sessionLink(ctx context.Context, sessionID string) string {
	if n.notice == nil {
		return ""
	}
	base, err := n.notice(ctx, notificationSourceScenario)
	if err != nil || strings.TrimSpace(base) == "" {
		return ""
	}
	return strings.TrimRight(base, "/") + "/?session=" + url.QueryEscape(sessionID)
}

// publishViaEventbus posts the event to Vrooli Events, resolving the events API
// base lazily (discovery may not be ready at construction) and caching it.
func (n *sessionNotifier) publishViaEventbus(ctx context.Context, event eventbus.DomainEvent) error {
	base := n.eventsBaseURL(ctx)
	if base == "" {
		return nil
	}
	client := eventbus.Client{BaseURL: base}
	return client.PublishDomainEvent(ctx, event)
}

// eventsBaseURL resolves the Vrooli Events API base: an explicit env override
// first, then scenario discovery, caching the discovered value.
func (n *sessionNotifier) eventsBaseURL(ctx context.Context) string {
	if v := strings.TrimSpace(os.Getenv("VROOLI_EVENTS_API_BASE")); v != "" {
		return v
	}
	n.mu.Lock()
	base := n.baseURL
	n.mu.Unlock()
	if base != "" {
		return base
	}
	resolved, _ := discovery.ResolveScenarioURLDefault(ctx, "vrooli-events")
	if strings.TrimSpace(resolved) != "" {
		n.mu.Lock()
		n.baseURL = resolved
		n.mu.Unlock()
	}
	return resolved
}
