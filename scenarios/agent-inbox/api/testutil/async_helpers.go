// Package testutil provides test utilities for the Agent Inbox API.
package testutil

import (
	"testing"
	"time"

	"agent-inbox/services"
)

// WaitForUpdate waits for an AsyncStatusUpdate on the channel with a timeout.
// Returns the update if received within the timeout, or fails the test.
func WaitForUpdate(t *testing.T, ch <-chan services.AsyncStatusUpdate, timeout time.Duration) services.AsyncStatusUpdate {
	t.Helper()
	select {
	case update, ok := <-ch:
		if !ok {
			t.Fatal("channel closed unexpectedly while waiting for update")
		}
		return update
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for update after %v", timeout)
		return services.AsyncStatusUpdate{}
	}
}

// WaitForCompletion waits for an AsyncCompletionEvent on the channel with a timeout.
// Returns the event if received within the timeout, or fails the test.
func WaitForCompletion(t *testing.T, ch <-chan services.AsyncCompletionEvent, timeout time.Duration) services.AsyncCompletionEvent {
	t.Helper()
	select {
	case event, ok := <-ch:
		if !ok {
			t.Fatal("channel closed unexpectedly while waiting for completion")
		}
		return event
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for completion event after %v", timeout)
		return services.AsyncCompletionEvent{}
	}
}

// ExpectNoUpdate verifies that no update is received within the specified duration.
// Fails the test if an update is received.
func ExpectNoUpdate(t *testing.T, ch <-chan services.AsyncStatusUpdate, wait time.Duration) {
	t.Helper()
	select {
	case update := <-ch:
		t.Fatalf("unexpected update received: %+v", update)
	case <-time.After(wait):
		// Expected behavior - no update received
	}
}

// ExpectNoCompletion verifies that no completion event is received within the specified duration.
// Fails the test if an event is received.
func ExpectNoCompletion(t *testing.T, ch <-chan services.AsyncCompletionEvent, wait time.Duration) {
	t.Helper()
	select {
	case event := <-ch:
		t.Fatalf("unexpected completion event received: %+v", event)
	case <-time.After(wait):
		// Expected behavior - no event received
	}
}

// DrainChannel drains all pending updates from a channel without blocking.
// Useful for cleanup in tests.
func DrainChannel(ch <-chan services.AsyncStatusUpdate) {
	for {
		select {
		case <-ch:
			// Discard
		default:
			return
		}
	}
}

// DrainCompletionChannel drains all pending completion events from a channel without blocking.
func DrainCompletionChannel(ch <-chan services.AsyncCompletionEvent) {
	for {
		select {
		case <-ch:
			// Discard
		default:
			return
		}
	}
}

// CollectUpdates collects all updates received within the timeout period.
// Returns when timeout is reached.
func CollectUpdates(t *testing.T, ch <-chan services.AsyncStatusUpdate, timeout time.Duration) []services.AsyncStatusUpdate {
	t.Helper()
	var updates []services.AsyncStatusUpdate
	deadline := time.After(timeout)
	for {
		select {
		case update, ok := <-ch:
			if !ok {
				return updates
			}
			updates = append(updates, update)
		case <-deadline:
			return updates
		}
	}
}

// WaitForTerminalUpdate waits for an update with IsTerminal=true.
// Returns the terminal update or fails the test if timeout is reached.
func WaitForTerminalUpdate(t *testing.T, ch <-chan services.AsyncStatusUpdate, timeout time.Duration) services.AsyncStatusUpdate {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case update, ok := <-ch:
			if !ok {
				t.Fatal("channel closed before receiving terminal update")
			}
			if update.IsTerminal {
				return update
			}
		case <-deadline:
			t.Fatalf("timed out waiting for terminal update after %v", timeout)
			return services.AsyncStatusUpdate{}
		}
	}
}
