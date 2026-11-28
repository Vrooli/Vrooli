package hyperframes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
)

func TestTaskPressureBeforeFirstSampleCancelsAndVerifiesOwnedCleanup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	exited := make(chan struct{})
	stopped, verified := false, false
	err := waitWithTaskPressure(ctx, func(runCtx context.Context) error {
		return waitOwned(runCtx, func() error { <-exited; return errors.New("child stopped") }, func(context.Context) error {
			stopped = true
			close(exited)
			return nil
		}, func() ([]int, error) { verified = true; return nil, nil })
	}, func() (platform.Occupancy, error) {
		// Current occupancy has fallen; the earlier denial still matters.
		return platform.Occupancy{Tasks: 45, TasksMax: 512, TasksMaxEvents: 3}, nil
	}, time.Millisecond, nil)
	if !errors.Is(err, errTaskPressure) || !stopped || !verified {
		t.Fatalf("pressure not terminated/verified: %v stopped=%t verified=%t", err, stopped, verified)
	}
}

func TestTaskPressureAfterUnknownSampleCancelsWithoutBaselineSubtraction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var samples atomic.Int32
	err := waitWithTaskPressure(ctx, func(runCtx context.Context) error { <-runCtx.Done(); return runCtx.Err() }, func() (platform.Occupancy, error) {
		if samples.Add(1) == 1 {
			return platform.Occupancy{TasksMaxEvents: platform.Unknown}, nil
		}
		return platform.Occupancy{TasksMax: 512, TasksMaxEvents: 1}, nil
	}, time.Millisecond, nil)
	if !errors.Is(err, errTaskPressure) || samples.Load() < 2 {
		t.Fatal(err, samples.Load())
	}
}

func TestTaskPressureUnknownOrCollectedCounterIsNotZeroEvidence(t *testing.T) {
	for _, unreadable := range []bool{false, true} {
		reads := 0
		err := waitWithTaskPressure(context.Background(), func(context.Context) error { return nil }, func() (platform.Occupancy, error) {
			reads++
			if unreadable {
				return platform.Occupancy{}, errors.New("scope already collected")
			}
			return platform.Occupancy{TasksMaxEvents: platform.Unknown}, nil
		}, time.Millisecond, nil)
		// This guard does not certify zero denials from absent telemetry; the
		// normal wait/cleanup remains authoritative and receipt states the limit.
		if err != nil || reads == 0 {
			t.Fatal(err, reads)
		}
	}
}

func TestTaskPressurePreservesCleanupFailureAndParentCancellation(t *testing.T) {
	cleanupErr := errors.New("cleanup unverified")
	err := waitWithTaskPressure(context.Background(), func(context.Context) error { return cleanupErr }, func() (platform.Occupancy, error) {
		return platform.Occupancy{TasksMaxEvents: 1}, nil
	}, time.Millisecond, nil)
	if !errors.Is(err, errTaskPressure) || !errors.Is(err, cleanupErr) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = waitWithTaskPressure(ctx, func(runCtx context.Context) error { return runCtx.Err() }, func() (platform.Occupancy, error) {
		return platform.Occupancy{TasksMaxEvents: platform.Unknown}, nil
	}, time.Millisecond, nil)
	if !errors.Is(err, context.Canceled) || errors.Is(err, errTaskPressure) {
		t.Fatal(err)
	}
}

func TestTaskObservationRetainsKnownPeakAndUnknownReadsInBoundedLog(t *testing.T) {
	var summary taskPressureSummary
	summary.observe(platform.Occupancy{}, errors.New("collected scope"))
	summary.observe(platform.Occupancy{Tasks: platform.Unknown, TasksMax: platform.Unknown, TasksMaxEvents: platform.Unknown}, nil)
	data, _ := json.Marshal(summary)
	if !strings.Contains(string(data), `"max_observed_denials":null`) || !strings.Contains(string(data), `"peak_observed_tasks":null`) || summary.UnknownCounterReads != 2 {
		t.Fatal(string(data))
	}
	summary.observe(platform.Occupancy{Tasks: 487, TasksMax: 512, TasksMaxEvents: 0}, nil)
	summary.observe(platform.Occupancy{Tasks: 45, TasksMax: 512, TasksMaxEvents: 3}, nil)
	var log bytes.Buffer
	ctx := context.WithValue(context.Background(), taskObservationWriterKey{}, &boundedWriter{Writer: &log, remaining: 2048})
	err := writeTaskObservation(ctx, platform.ScopeRef{Kind: platform.ScopeKindCgroup, Name: "owned.scope", Path: "/test/owned.scope"}, "/usr/bin/ffmpeg", summary)
	if err != nil {
		t.Fatal(err)
	}
	var retained struct {
		Scope string `json:"scope"`
		taskPressureSummary
	}
	if err := json.Unmarshal(bytes.TrimPrefix(bytes.TrimSpace(log.Bytes()), []byte("[hyperframes-owner] ")), &retained); err != nil {
		t.Fatal(err)
	}
	if retained.Samples != 4 || retained.KnownCounterReads != 2 || retained.UnknownCounterReads != 2 || retained.MaxObservedDenials == nil || *retained.MaxObservedDenials != 3 || retained.PeakObservedTasks == nil || *retained.PeakObservedTasks != 487 || retained.ObservedTasksMax == nil || *retained.ObservedTasksMax != 512 || retained.Scope != "cgroup:/test/owned.scope" || bytes.Count(log.Bytes(), []byte{'\n'}) != 1 {
		t.Fatal(log.String())
	}
}

func TestTaskObservationReportedAfterWaitAndLogFailureRemainsFailure(t *testing.T) {
	waited, reported := false, false
	logErr := errors.New("diagnostic log budget exceeded")
	err := waitWithTaskPressure(context.Background(), func(context.Context) error { waited = true; return nil }, func() (platform.Occupancy, error) {
		return platform.Occupancy{TasksMaxEvents: platform.Unknown, Tasks: platform.Unknown, TasksMax: platform.Unknown}, nil
	}, time.Millisecond, func(summary taskPressureSummary) error {
		reported = true
		if !waited || summary.Samples == 0 || summary.MaxObservedDenials != nil || summary.KnownCounterReads != 0 {
			t.Fatalf("dishonest or premature observation: %+v", summary)
		}
		return logErr
	})
	if !reported || !errors.Is(err, logErr) {
		t.Fatal(err, reported)
	}
}
