package volumeremediation

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestHeldOperationLeaseBlocksSupportedUnmountAndCancellationReleasesWaiter(t *testing.T) {
	state := unmountedDirtyState()
	state.Device.Mountpoint = "/media/test/volume"
	state.Mounted = true
	observe := func(context.Context, string) (State, error) { return state, nil }
	release, err := AcquireOperationLease(context.Background(), state.Device, observe)
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	runner := &fakeRunner{}
	svc := linuxService(t, &fakeObserver{state: state}, runner)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := svc.Execute(ctx, Request{Action: ActionUnmount, Device: state.Device}); done <- err }()
	time.Sleep(40 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Execute error = %v, want canceled lease wait", err)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled lease waiter did not return")
	}
	if len(runner.argv) != 0 {
		t.Fatalf("unmount ran while lease was held: %v", runner.argv)
	}
}

func TestReleasingOperationLeaseUnblocksSupportedUnmount(t *testing.T) {
	state := unmountedDirtyState()
	state.Device.Mountpoint = "/media/test/volume"
	state.Mounted = true
	observe := func(context.Context, string) (State, error) { return state, nil }
	release, err := AcquireOperationLease(context.Background(), state.Device, observe)
	if err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{}
	svc := linuxService(t, &fakeObserver{state: state}, runner)
	done := make(chan error, 1)
	go func() {
		_, err := svc.Execute(context.Background(), Request{Action: ActionUnmount, Device: state.Device})
		done <- err
	}()
	time.Sleep(40 * time.Millisecond)
	if len(runner.argv) != 0 {
		t.Fatal("unmount ran before lease release")
	}
	release()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute after release: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("lease release did not unblock unmount")
	}
	if len(runner.argv) != 1 {
		t.Fatalf("unmount calls = %d, want 1", len(runner.argv))
	}
}

func TestOperationLeaseRejectsChangedIdentityBeforeGrant(t *testing.T) {
	expected := elementsDevice()
	other := expected
	other.UUID = "different-uuid"
	granted := false
	_, err := AcquireOperationLease(context.Background(), expected, func(context.Context, string) (State, error) {
		return State{Device: other}, nil
	})
	if err == nil {
		granted = true
	}
	if granted {
		t.Fatal("lease granted after observed identity changed")
	}
	if !errors.Is(err, ErrDeviceChanged) {
		t.Fatalf("error = %v, want ErrDeviceChanged", err)
	}
}

func TestOperationLeaseRequiresStableIdentity(t *testing.T) {
	_, err := AcquireOperationLease(context.Background(), Device{Path: "/dev/sda1"}, func(context.Context, string) (State, error) { return State{}, nil })
	if err == nil {
		t.Fatal("lease granted using only an unstable device path")
	}
}
