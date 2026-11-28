package control

import (
	"context"
	"errors"
	"fmt"
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// QuiesceObservers requests cancellation of every in-process state observer
// and waits for each worker's completion receipt. It is called only after the
// lifecycle provider has closed request admission; cancellation alone is not
// treated as completed cleanup.
func (s *Service) QuiesceObservers(ctx context.Context) error {
	s.mu.Lock()
	// The lifecycle fence stops new requests, but already-admitted handlers can
	// still finish inventory and identity work. Close observer creation while
	// holding the same lock used by startObserverLocked so no late worker can
	// appear after this snapshot.
	s.observerAdmissionClosed = true
	jobs := make([]*observerJob, 0, len(s.observerCancels))
	for _, job := range s.observerCancels {
		if job == nil || job.cancel == nil || job.done == nil {
			s.mu.Unlock()
			return errors.New("device observer is missing cancellation or completion ownership")
		}
		job.stopping = true
		jobs = append(jobs, job)
	}
	for _, job := range jobs {
		job.cancel()
	}
	s.mu.Unlock()

	for _, job := range jobs {
		select {
		case <-job.done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// LifecycleInventory snapshots the in-process owners that Device Control can
// observe directly. It reports work as blockers until the owner has a terminal
// completion state; it never treats cancellation request as cleanup completion.
func (s *Service) LifecycleInventory(ctx context.Context) ([]*commonv1.LifecycleWorkItem, []*commonv1.LifecycleBlocker, bool) {
	if err := ctx.Err(); err != nil {
		return nil, []*commonv1.LifecycleBlocker{{Code: "owner_inventory_failed", Message: err.Error(), Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN, Observed: true}}, false
	}

	work := make([]*commonv1.LifecycleWorkItem, 0, 6)
	blockers := make([]*commonv1.LifecycleBlocker, 0)
	add := func(kind, id, status, message string) {
		if id == "" {
			id = "owner/" + kind
		}
		work = append(work, &commonv1.LifecycleWorkItem{Id: id, Kind: kind, Status: status})
		if status != "idle" && status != "completed" {
			blockers = append(blockers, &commonv1.LifecycleBlocker{Code: "owner_work_pending", Message: message, Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_BLOCKING, Observed: true})
		}
	}

	s.sessionMu.Lock()
	active := len(s.activeCancels)
	liveLeases := 0
	now := time.Now()
	for _, session := range s.sessions {
		if session.State == "held" && now.Before(session.ExpiresAt) {
			liveLeases++
		}
	}
	s.sessionMu.Unlock()
	add("flows-and-cleanup", "owner/flows-and-cleanup", "idle", "")
	if active > 0 {
		work[len(work)-1].Status = "active"
		blockers = append(blockers, &commonv1.LifecycleBlocker{Code: "flow_cleanup_pending", Message: fmt.Sprintf("%d leased flow operation(s) still own execution or cleanup", active), Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_BLOCKING, Observed: true})
	}
	if liveLeases > 0 {
		add("durable-leases-and-recovery", "owner/durable-leases-and-recovery", "active", fmt.Sprintf("%d unexpired device lease(s) remain held", liveLeases))
	} else {
		add("durable-leases-and-recovery", "owner/durable-leases-and-recovery", "idle", "")
	}

	s.mu.Lock()
	observers := len(s.observerCancels)
	pairings := len(s.pendingPairings)
	recordings := len(s.externalRecordings)
	for id, recording := range s.externalRecordings {
		status := map[closeState]string{stateOpen: "active", stateClosing: "stopping", stateUncertain: "uncertain"}[recording.State]
		add("recordings", "recording/"+id, status, "recording "+id+" has not reached a verified stop receipt")
	}
	for id, pairing := range s.pendingPairings {
		status := "pending"
		message := "pairing session " + id + " remains open"
		switch pairing.close {
		case stateClosing:
			status = "closing"
			message = "pairing session " + id + " is closing and has not returned a close receipt"
		case stateUncertain:
			status = "uncertain"
			message = "pairing session " + id + " close did not return a verified receipt"
		}
		add("pairing", "pairing/"+id, status, message)
	}
	for id, observer := range s.observerCancels {
		status := "active"
		if observer.stopping {
			status = "stopping"
		}
		add("observers", "observer/"+id, status, "observer "+id+" has not acknowledged cancellation and exited")
	}
	s.mu.Unlock()
	if recordings == 0 {
		add("recordings", "owner/recordings", "idle", "")
	}
	if pairings == 0 {
		add("pairing", "owner/pairing", "idle", "")
	}
	if observers == 0 {
		add("observers", "owner/observers", "idle", "")
	}
	// Agent planning and execution are synchronous under the admitted request;
	// the process-wide admission counter remains held until the handler returns.
	add("agents", "owner/agents", "idle", "")
	return work, blockers, true
}
