package main

import (
	"testing"
	"time"
)

// drainStateCh non-blockingly collects all currently queued alt-buffer
// transitions from a client's StateCh and returns them in arrival order.
func drainStateCh(ch chan bool) []bool {
	var out []bool
	for {
		select {
		case v := <-ch:
			out = append(out, v)
		default:
			return out
		}
	}
}

// TestSession_AltBufferBroadcastsToSubscriber verifies that a session
// observing a `?1049h` toggle in its PTY output delivers the transition
// to a subscribed client's StateCh.
func TestSession_AltBufferBroadcastsToSubscriber(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

	if sub.InitialAltBuffer {
		t.Fatal("new session should start outside alt-buffer")
	}

	// Inject an alt-buffer entry sequence and drain the output so the
	// broadcast path runs synchronously with the test.
	if _, err := fake.outW.Write([]byte("\x1b[?1049h")); err != nil {
		t.Fatalf("write to fake pty: %v", err)
	}
	// Give readLoop a moment to consume. We wait up to 500ms for the
	// state transition to arrive.
	deadline := time.Now().Add(500 * time.Millisecond)
	var got []bool
	for time.Now().Before(deadline) {
		// Drain output so readLoop doesn't block on a full channel.
		select {
		case <-sub.OutputCh:
		default:
		}
		got = append(got, drainStateCh(sub.StateCh)...)
		if len(got) > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if len(got) != 1 || got[0] != true {
		t.Fatalf("expected one altBuffer=true transition, got %v", got)
	}

	// Now exit alt-buffer; expect another transition.
	if _, err := fake.outW.Write([]byte("\x1b[?1049l")); err != nil {
		t.Fatalf("write: %v", err)
	}
	got = got[:0]
	deadline = time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		select {
		case <-sub.OutputCh:
		default:
		}
		got = append(got, drainStateCh(sub.StateCh)...)
		if len(got) > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if len(got) != 1 || got[0] != false {
		t.Fatalf("expected one altBuffer=false transition, got %v", got)
	}
}

// TestSession_SubscribeReportsInitialAltBufferTrue verifies that a client
// subscribing mid-session (while the TUI is in alt-buffer) sees the
// current state in SubscribeResult.
func TestSession_SubscribeReportsInitialAltBufferTrue(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	// First subscriber: observe the enter.
	sub1 := sess.Subscribe(0)
	if _, err := fake.outW.Write([]byte("\x1b[?1049h")); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Drain sub1 until we see the transition.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		select {
		case <-sub1.OutputCh:
		default:
		}
		if len(drainStateCh(sub1.StateCh)) > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	sess.Unsubscribe(sub1.OutputCh)

	// Second subscriber joins after the transition; Subscribe should
	// report InitialAltBuffer == true.
	sub2 := sess.Subscribe(0)
	defer sess.Unsubscribe(sub2.OutputCh)
	if !sub2.InitialAltBuffer {
		t.Fatal("expected InitialAltBuffer=true for post-entry subscriber")
	}
}

// TestSession_SIGWINCHRecovery_SuppressedInAltBuffer verifies the central
// behavior for the footer-duplication bug: when a coalesced output buffer
// is trimmed while the PTY is in the alt-buffer, the server does NOT
// fire SIGWINCH (which would race the TUI's redraw).
func TestSession_SIGWINCHRecovery_SuppressedInAltBuffer(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	// Drive the session into alt-buffer by default, and make the
	// recovery cooldown non-zero so behavior is deterministic.
	sm.cfg.SIGWINCHCooldownMs = 0 // cooldown not relevant for this test
	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	sess.mu.Lock()
	// Force the tracker into alt-buffer without touching the fake PTY's
	// readLoop (which would require pumping data through it).
	sess.ptyState.Observe([]byte("\x1b[?1049h"))
	sess.mu.Unlock()

	// Record SetSize calls on the fake before the recovery.
	fake.mu.Lock()
	baselineCalls := fake.setSizeCalls
	fake.mu.Unlock()

	// Trigger the recovery path directly by faking a trimmed state.
	sess.mu.Lock()
	sess.maybeSIGWINCHRecovery()
	sess.mu.Unlock()

	fake.mu.Lock()
	afterCalls := fake.setSizeCalls
	fake.mu.Unlock()
	if afterCalls != baselineCalls {
		t.Fatalf("SIGWINCH must not fire while in alt-buffer; setSize calls went from %d to %d", baselineCalls, afterCalls)
	}
}

// TestSession_SIGWINCHRecovery_RateLimited verifies that rapid repeated
// invocations of the recovery path only issue SIGWINCH once per cooldown.
func TestSession_SIGWINCHRecovery_RateLimited(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sm.cfg.SIGWINCHCooldownMs = 50 // short cooldown for the test
	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	fake.mu.Lock()
	baseline := fake.setSizeCalls
	fake.mu.Unlock()

	sess.mu.Lock()
	sess.maybeSIGWINCHRecovery() // 1 — fires
	sess.maybeSIGWINCHRecovery() // 2 — suppressed
	sess.maybeSIGWINCHRecovery() // 3 — suppressed
	sess.mu.Unlock()

	fake.mu.Lock()
	after := fake.setSizeCalls
	fake.mu.Unlock()
	if got := after - baseline; got != 1 {
		t.Fatalf("rate-limit broken: expected 1 SIGWINCH in a burst, got %d", got)
	}

	// After the cooldown elapses, a follow-up should fire again.
	time.Sleep(60 * time.Millisecond)
	sess.mu.Lock()
	sess.maybeSIGWINCHRecovery()
	sess.mu.Unlock()
	fake.mu.Lock()
	final := fake.setSizeCalls
	fake.mu.Unlock()
	if got := final - after; got != 1 {
		t.Fatalf("post-cooldown SIGWINCH not fired: delta=%d", got)
	}
}

// TestSession_SIGWINCHRecovery_FiresWhenNotInAltBuffer verifies the
// non-alt-buffer path still recovers as designed.
func TestSession_SIGWINCHRecovery_FiresWhenNotInAltBuffer(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sm.cfg.SIGWINCHCooldownMs = 0
	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	fake.mu.Lock()
	baseline := fake.setSizeCalls
	fake.mu.Unlock()

	sess.mu.Lock()
	sess.maybeSIGWINCHRecovery()
	sess.mu.Unlock()

	fake.mu.Lock()
	after := fake.setSizeCalls
	fake.mu.Unlock()
	if after-baseline != 1 {
		t.Fatalf("expected SIGWINCH to fire outside alt-buffer, got delta=%d", after-baseline)
	}
}
