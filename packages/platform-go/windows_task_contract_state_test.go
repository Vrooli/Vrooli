package platform

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestWindowsTaskExtinctionRequiresFinitePositiveNativeState(t *testing.T) {
	for _, state := range []int32{-1, 0, 1, 2, 3, 4, 5} {
		for _, populated := range []bool{false, true} {
			snapshot := WindowsTaskSnapshot{State: state}
			if populated {
				snapshot.Instances = []WindowsTaskInstance{{GUID: "exact", State: windowsTaskRunning}}
			}
			quiet, err := windowsTaskQuiescent(snapshot)
			want := !populated && (state == windowsTaskDisabled || state == windowsTaskReady)
			if quiet != want || (state < 0 || state > 4) && !errors.Is(err, ErrOperationIdentity) || state == 0 && !errors.Is(err, ErrOperationOutcomeLost) {
				t.Fatalf("state=%d populated=%t quiet=%t err=%v", state, populated, quiet, err)
			}
		}
	}
	if _, err := windowsTaskQuiescent(WindowsTaskSnapshot{State: windowsTaskReady, Instances: []WindowsTaskInstance{{State: 5}}}); !errors.Is(err, ErrOperationIdentity) {
		t.Fatal("invalid native instance state accepted", err)
	}
	if !windowsTaskHRESULTNotRunning(0x8004130b) || windowsTaskStopAcknowledged(0x8004130b) {
		t.Fatal("no running instance was confused with cancellation acknowledgement")
	}
	for _, hr := range []uint32{0, 1, 0x80070002, 0x80070005, 0x8004130f, 0x41300} {
		if windowsTaskHRESULTNotRunning(hr) {
			t.Fatalf("unrelated HRESULT accepted as already inactive: %#x", hr)
		}
	}
}

func TestWindowsOperatorNormalStopIsIdempotentAndRetainsBootEnablement(t *testing.T) {
	for _, state := range []int32{windowsTaskReady, windowsTaskQueued} {
		snapshot := WindowsTaskSnapshot{State: state, Enabled: true}
		cancels, runs := 0, 0
		effects := windowsOperatorTaskControlEffects{
			stop: func(ctx context.Context, _ WindowsTaskSnapshot) error {
				cancels++ // modeled native registered-task cancellation, not GetInstances
				snapshot.State = windowsTaskReady
				return waitWindowsTaskQuiescent(ctx, func() (WindowsTaskSnapshot, error) { return snapshot, nil })
			},
			run: func() error { runs++; return nil },
		}
		for retry := 0; retry < 2; retry++ {
			if err := controlWindowsOperatorTaskWith(t.Context(), "stop", snapshot, effects); err != nil {
				t.Fatal(err)
			}
		}
		if cancels != 2 || runs != 0 || !snapshot.Enabled {
			t.Fatalf("ordinary Stop changed boot policy or ran again: cancel=%d run=%d enabled=%t", cancels, runs, snapshot.Enabled)
		}
	}
}

func TestWindowsOperatorRestartUsesFreshStateWithoutDuplicateRun(t *testing.T) {
	for _, state := range []int32{-1, 0, 1, 2, 3, 4, 5} {
		events := []string{}
		effects := windowsOperatorTaskControlEffects{
			stop: func(context.Context, WindowsTaskSnapshot) error {
				events = append(events, "cancel-and-prove-quiet")
				return nil
			},
			query: func() (WindowsTaskSnapshot, error) {
				events = append(events, "fresh")
				return WindowsTaskSnapshot{State: state, Enabled: true}, nil
			},
			run:   func() error { events = append(events, "run"); return nil },
			await: func(context.Context) error { events = append(events, "await-exact-action"); return nil },
		}
		err := controlWindowsOperatorTaskWith(t.Context(), "restart", WindowsTaskSnapshot{State: windowsTaskRunning, Enabled: true}, effects)
		want := []string{"cancel-and-prove-quiet", "fresh"}
		if state == windowsTaskReady {
			want = append(want, "run", "await-exact-action")
		} else if state == windowsTaskQueued || state == windowsTaskRunning {
			want = append(want, "await-exact-action")
		}
		if !reflect.DeepEqual(events, want) || (state == 2 || state == 3 || state == 4) != (err == nil) {
			t.Fatalf("fresh state=%d events=%v err=%v want=%v", state, events, err, want)
		}
	}
}

func TestWindowsOperatorUnresolvedCancellationCannotRestartOrDelete(t *testing.T) {
	for _, state := range []int32{windowsTaskUnknown, windowsTaskQueued, windowsTaskRunning} {
		ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
		defer cancel()
		runs := 0
		snapshot := WindowsTaskSnapshot{State: state, Enabled: true}
		err := controlWindowsOperatorTaskWith(ctx, "restart", snapshot, windowsOperatorTaskControlEffects{
			stop: func(ctx context.Context, _ WindowsTaskSnapshot) error {
				return waitWindowsTaskQuiescent(ctx, func() (WindowsTaskSnapshot, error) { return snapshot, nil })
			},
			run: func() error { runs++; return nil },
		})
		if err == nil || runs != 0 {
			t.Fatalf("unresolved state=%d dispatched another action: %v", state, err)
		}
	}
	for _, state := range []int32{-1, 0, 1, 2, 3, 4, 5} {
		deleted, absent := false, false
		err := deleteWindowsQuiescentTask(func() (WindowsTaskSnapshot, error) {
			if absent {
				return WindowsTaskSnapshot{}, ErrOperationNotFound
			}
			return WindowsTaskSnapshot{Name: "exact-owned-task", State: state, Enabled: false}, nil
		}, func(name string) error { deleted, absent = true, true; return nil }, func(err error) bool { return errors.Is(err, ErrOperationNotFound) })
		want := state == windowsTaskDisabled || state == windowsTaskReady
		if deleted != want || (err == nil) != want {
			t.Fatalf("final deletion state=%d deleted=%t err=%v", state, deleted, err)
		}
	}
}

func TestWindowsOperatorStartAwaitsExistingQueuedAction(t *testing.T) {
	for _, state := range []int32{windowsTaskQueued, windowsTaskRunning} {
		runs, awaits := 0, 0
		err := controlWindowsOperatorTaskWith(t.Context(), "start", WindowsTaskSnapshot{State: state, Enabled: true}, windowsOperatorTaskControlEffects{
			run: func() error { runs++; return nil }, await: func(context.Context) error { awaits++; return nil },
		})
		if err != nil || runs != 0 || awaits != 1 {
			t.Fatalf("existing state=%d run=%d await=%d err=%v", state, runs, awaits, err)
		}
	}
}
