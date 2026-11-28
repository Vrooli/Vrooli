package onboard

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCoordinatorRegistrationRemovalIsIsolatedAndIdempotent(t *testing.T) {
	c := newCoordinator()
	firstWaiter, removeFirstWaiter := c.registerWaiter("wait")
	secondWaiter, removeSecondWaiter := c.registerWaiter("wait")
	require.Zero(t, cap(firstWaiter), "waiters remain unbuffered")
	removeFirstWaiter()
	removeFirstWaiter()
	require.Len(t, c.waiters["wait"], 1)
	c.signalTerminal("wait")
	select {
	case <-firstWaiter:
		t.Fatal("removed waiter received the terminal signal")
	default:
	}
	select {
	case _, open := <-secondWaiter:
		require.False(t, open, "remaining waiter is closed by the terminal signal")
	default:
		t.Fatal("remaining waiter did not receive the terminal signal")
	}
	removeSecondWaiter()
	require.NotContains(t, c.waiters, "wait", "terminal delivery and cancellation leave no registry entry")

	firstSubscriber, removeFirstSubscriber := c.subscribe("stream")
	secondSubscriber, removeSecondSubscriber := c.subscribe("stream")
	require.Equal(t, subscriberBuffer, cap(firstSubscriber))
	removeFirstSubscriber()
	removeFirstSubscriber()
	require.Len(t, c.subs["stream"], 1)
	event := StepEvent{OpID: "stream", Sequence: 1, StepID: "setup", Status: StepStatusStarted}
	c.publish(event)
	select {
	case got := <-firstSubscriber:
		t.Fatalf("removed subscriber received an event: %+v", got)
	default:
	}
	require.Equal(t, event, <-secondSubscriber)
	removeSecondSubscriber()
	require.NotContains(t, c.subs, "stream", "removing the final subscriber frees its registry entry")
}

func TestCoordinatorRegistrationRemovalRacesWithDelivery(t *testing.T) {
	c := newCoordinator()
	var wg sync.WaitGroup
	for range 32 {
		waiter, removeWaiter := c.registerWaiter("op")
		subscriber, removeSubscriber := c.subscribe("op")
		wg.Add(2)
		go func() {
			defer wg.Done()
			removeWaiter()
			removeWaiter()
		}()
		go func() {
			defer wg.Done()
			c.signalTerminal("op")
			c.publish(StepEvent{OpID: "op", Sequence: 1})
			removeSubscriber()
			removeSubscriber()
		}()
		_ = waiter
		_ = subscriber
	}
	wg.Wait()
	require.Empty(t, c.waiters, "terminal delivery and cancellation retire all waiter sets")
	require.Empty(t, c.subs, "unsubscribing concurrent listeners retires all subscriber sets")
}
