package workflow

import (
	"errors"
	"sync"
	"testing"
)

func TestExecutionAdmissionConcurrentBurstNeverExceedsProcessLimit(t *testing.T) {
	const limit, callers = 3, 48
	service := &WorkflowService{maxConcurrentExecutions: limit}
	start := make(chan struct{})
	release := make(chan struct{})
	var wg sync.WaitGroup
	var arrived sync.WaitGroup
	var mu sync.Mutex
	var admitted int
	var admissionReleases []func()
	var slotReleases []func()
	var badErrors []error

	for i := 0; i < callers; i++ {
		wg.Add(1)
		arrived.Add(1)
		go func() {
			defer wg.Done()
			<-start
			closeAdmission, closeSlot, err := service.beginExecutionAdmission()
			if err != nil {
				arrived.Done()
				if !errors.Is(err, ErrWorkflowExecutionCapacity) {
					mu.Lock()
					badErrors = append(badErrors, err)
					mu.Unlock()
				}
				return
			}
			mu.Lock()
			admitted++
			admissionReleases = append(admissionReleases, closeAdmission)
			slotReleases = append(slotReleases, closeSlot)
			mu.Unlock()
			arrived.Done()
			<-release
		}()
	}
	close(start)
	arrived.Wait()
	if len(badErrors) != 0 {
		t.Fatalf("unexpected admission errors: %v", badErrors)
	}
	if admitted != limit {
		t.Fatalf("admitted=%d during a %d-caller burst; want exactly configured process limit %d", admitted, callers, limit)
	}
	service.executionLifecycleMu.Lock()
	used := service.executionSlotsInUse
	service.executionLifecycleMu.Unlock()
	if used != limit {
		t.Fatalf("in-use slots while accepted callers hold permits=%d; want %d", used, limit)
	}
	close(release)
	for i := range admissionReleases {
		admissionReleases[i]()
		slotReleases[i]()
		// Release callbacks are idempotent; stale duplicate completion must not
		// free another execution's permit or drive the count negative.
		slotReleases[i]()
	}
	wg.Wait()
	service.executionLifecycleMu.Lock()
	used = service.executionSlotsInUse
	service.executionLifecycleMu.Unlock()
	if used != 0 {
		t.Fatalf("in-use slots after caller cleanup=%d; want 0", used)
	}
	_, releaseSlot, err := service.beginExecutionAdmission()
	if err != nil {
		t.Fatalf("fresh admission after cleanup: %v", err)
	}
	releaseSlot()
}
