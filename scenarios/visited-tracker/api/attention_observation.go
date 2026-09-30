package main

import (
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	attentionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/visited-tracker/v1/attention"
)

// Completion order, rather than request start order, defines the failure
// streak. One mutex keeps concurrent snapshots and their counters coherent.
type storageWriteObserver struct {
	mu                             sync.Mutex
	epoch                          string
	completed, failed, consecutive uint64
	lastCompleted                  time.Time
}

var campaignWriteObserver = &storageWriteObserver{epoch: uuid.NewString()}

func (o *storageWriteObserver) record(at time.Time, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.completed++
	o.lastCompleted = at
	if err != nil {
		o.failed++
		o.consecutive++
	} else {
		o.consecutive = 0
	}
}

func (o *storageWriteObserver) snapshot(now time.Time) *attentionv1.StorageWriteObservation {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := &attentionv1.StorageWriteObservation{
		Epoch: o.epoch, ObservedAt: now.UTC().Format(time.RFC3339Nano),
		CompletedTotal: o.completed, FailedTotal: o.failed, ConsecutiveFailures: o.consecutive,
		DurableSyncSupported: runtime.GOOS != "windows",
	}
	if !o.lastCompleted.IsZero() {
		out.LastCompletedAt = o.lastCompleted.UTC().Format(time.RFC3339Nano)
		if age := now.Sub(o.lastCompleted); age > 0 {
			out.LastCompletedAgeSeconds = uint64(age / time.Second)
		}
	}
	return out
}

func observeClaimIntegrity(c *Campaign, now time.Time) *attentionv1.ClaimIntegrity {
	out := &attentionv1.ClaimIntegrity{CheckSchemaVersion: 1, ObservedAt: now.UTC().Format(time.RFC3339Nano)}
	ids, requests := map[string]bool{}, map[string]bool{}
	type subjectRevision struct {
		id       uuid.UUID
		revision string
	}
	subjects := map[subjectRevision]bool{}
	for _, claim := range c.Claims {
		if claimTerminal(claim, now) {
			continue
		}
		out.ExaminedClaims++
		subject := subjectRevision{claim.FileID, claim.Revision}
		invalid := !validIdentity(claim.ID) || !validIdentity(claim.RequestID) || !validIdentity(claim.Worker) ||
			claim.FileID == uuid.Nil || claim.Revision == "" || claim.CreatedAt.IsZero() || !claim.ExpiresAt.After(claim.CreatedAt)
		if invalid || ids[claim.ID] || requests[claim.RequestID] || subjects[subject] {
			out.Violations++
		}
		ids[claim.ID], requests[claim.RequestID], subjects[subject] = true, true, true
	}
	return out
}
