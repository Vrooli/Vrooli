package hyperframes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	platform "github.com/vrooli/platform-go"
)

var errTaskPressure = errors.New("owned HyperFrames scope exceeded task ceiling")

type taskObservationWriterKey struct{}

type taskPressureSummary struct {
	Samples             int    `json:"samples"`
	KnownCounterReads   int    `json:"known_counter_reads"`
	UnknownCounterReads int    `json:"unknown_counter_reads"`
	MaxObservedDenials  *int64 `json:"max_observed_denials"`
	PeakObservedTasks   *int64 `json:"peak_observed_tasks"`
	ObservedTasksMax    *int64 `json:"observed_tasks_max"`
}

func (s *taskPressureSummary) observe(occupancy platform.Occupancy, err error) {
	s.Samples++
	if err != nil {
		s.UnknownCounterReads++
		return
	}
	maximum := func(target **int64, value int64) {
		if value >= 0 && (*target == nil || value > **target) {
			*target = &value
		}
	}
	if occupancy.TasksMaxEvents < 0 {
		s.UnknownCounterReads++
	} else {
		s.KnownCounterReads++
		maximum(&s.MaxObservedDenials, occupancy.TasksMaxEvents)
	}
	maximum(&s.PeakObservedTasks, occupancy.Tasks)
	maximum(&s.ObservedTasksMax, occupancy.TasksMax)
}

// The diagnostic sink is separate from stdout: version/probe captures must not
// acquire JSON observation records that would change their native protocols.
func writeTaskObservation(ctx context.Context, scope platform.ScopeRef, command string, summary taskPressureSummary) error {
	writer, _ := ctx.Value(taskObservationWriterKey{}).(io.Writer)
	if writer == nil {
		return nil
	}
	data, err := json.Marshal(struct {
		Event   string `json:"event"`
		Scope   string `json:"scope"`
		Name    string `json:"scope_name"`
		Command string `json:"command"`
		taskPressureSummary
	}{"owned_task_observation", scope.String(), scope.Name, filepath.Base(command), summary})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(writer, "[hyperframes-owner] %s\n", data)
	return err
}

// waitWithTaskPressure observes only the caller's newly owned scope. wait owns
// canonical cancellation and cleanup; the observer never selects or kills a
// process. Missing/collected counters remain unknown, not evidence of zero.
func waitWithTaskPressure(ctx context.Context, wait func(context.Context) error, read func() (platform.Occupancy, error), interval time.Duration, report func(taskPressureSummary) error) error {
	runCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	var summary taskPressureSummary
	sample := func() {
		occupancy, err := read()
		summary.observe(occupancy, err)
		if err == nil && occupancy.TasksMaxEvents > 0 {
			cancel(fmt.Errorf("%w: pids.events max=%d, tasks_max=%d", errTaskPressure, occupancy.TasksMaxEvents, occupancy.TasksMax))
		}
	}
	// Pressure may precede the first sample: any nonzero value in a fresh scope
	// is a denial, rather than a baseline to subtract from later values.
	sample()
	watchCtx, stopWatch := context.WithCancel(runCtx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-watchCtx.Done():
				return
			case <-ticker.C:
				sample()
			}
		}
	}()
	err := wait(runCtx)
	stopWatch()
	<-done
	if report != nil {
		err = errors.Join(err, report(summary))
	}
	if cause := context.Cause(runCtx); errors.Is(cause, errTaskPressure) {
		return errors.Join(cause, err)
	}
	return err
}
