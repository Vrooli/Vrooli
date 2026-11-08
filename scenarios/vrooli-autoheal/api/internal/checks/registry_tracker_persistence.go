package checks

import (
	"context"
	"log"
	"time"
)

// Tracker writers hold trackerWriteMu from mutation through persistence. A
// completed healthy observation, resume or reset therefore cannot be undone
// by an older save still queued in a goroutine. The registry state lock is
// released during I/O, so status remains readable on a slow database. A failed
// write is diagnosed; it is not a claim that the in-memory decision is durable.
func persistTrackerSnapshot(store HealTrackerStore, id string, tracker HealTracker) {
	if store == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.SaveHealTracker(ctx, id, &tracker); err != nil {
		log.Printf("vrooli-autoheal: tracker persistence failed for %s: %v", id, err)
	}
}
