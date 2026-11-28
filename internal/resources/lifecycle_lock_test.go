package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/capacity"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestManagedServiceLifecycleLockSerializesConcurrentMutations(t *testing.T) {
	resource := fmt.Sprintf("lifecycle-lock-test-%d", time.Now().UnixNano())
	entered := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- withManagedServiceLifecycleLock(context.Background(), resource, func() error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered

	secondEntered := make(chan struct{})
	secondDone := make(chan error, 1)
	go func() {
		secondDone <- withManagedServiceLifecycleLock(context.Background(), resource, func() error {
			close(secondEntered)
			return nil
		})
	}()
	select {
	case <-secondEntered:
		t.Fatal("concurrent lifecycle mutation entered before the first released")
	case <-time.After(100 * time.Millisecond):
	}

	close(release)
	if err := <-firstDone; err != nil {
		t.Fatalf("first lifecycle mutation: %v", err)
	}
	select {
	case <-secondEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("second lifecycle mutation did not enter after release")
	}
	if err := <-secondDone; err != nil {
		t.Fatalf("second lifecycle mutation: %v", err)
	}
}

func TestLifecycleQueuedCancellationPreservesFirstOwnerAndNextAdmission(t *testing.T) {
	key := fmt.Sprintf("cancel-fixture-%d", time.Now().UnixNano())
	path, err := artifactlock.LockPath("resource-lifecycle:" + key)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
	first, err := acquireManagedServiceLifecycleLock(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	defer first()
	ctx, cancel := context.WithCancel(context.Background())
	var callbacks atomic.Int32
	done := make(chan error, 1)
	enteredQueue := make(chan struct{})
	go func() {
		close(enteredQueue)
		done <- withManagedServiceLifecycleLock(ctx, key, func() error { callbacks.Add(1); return nil })
	}()
	<-enteredQueue
	select {
	case err := <-done:
		first()
		t.Fatalf("waiter escaped held owner before cancellation: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	cancel()
	var queuedErr error
	timely := true
	select {
	case queuedErr = <-done:
	case <-time.After(time.Millisecond * 100):
		timely = false
	}
	// A cancelled waiter cannot release the actual first owner.
	nextEntered := make(chan struct{})
	nextDone := make(chan error, 1)
	go func() {
		nextDone <- withManagedServiceLifecycleLock(context.Background(), key, func() error { close(nextEntered); return nil })
	}()
	select {
	case <-nextEntered:
		t.Error("cancelled waiter released first owner")
	case <-time.After(time.Millisecond * 20):
	}
	first()
	if !timely {
		queuedErr = <-done
	}
	if !timely || !errors.Is(queuedErr, context.Canceled) || callbacks.Load() != 0 {
		t.Errorf("cancelled waiter executed or blocked: timely=%v err=%v callbacks=%d", timely, queuedErr, callbacks.Load())
	}
	select {
	case err := <-nextDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("next owner did not enter after release")
	}
}

func TestCancelledLifecycleReadinessReleasesForNormalNextOwner(t *testing.T) {
	key := fmt.Sprintf("cancel-readiness-fixture-%d", time.Now().UnixNano())
	path, err := artifactlock.LockPath("resource-lifecycle:" + key)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	entered := make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		finished <- withManagedServiceLifecycleLock(parent, key, func() error {
			close(entered)
			<-parent.Done()
			return fmt.Errorf("fixture readiness: %w", parent.Err())
		})
	}()
	<-entered
	nextEntered := make(chan struct{})
	nextDone := make(chan error, 1)
	go func() {
		nextDone <- withManagedServiceLifecycleLock(context.Background(), key, func() error { close(nextEntered); return nil })
	}()
	select {
	case <-nextEntered:
		t.Fatal("normal owner entered before readiness cancellation/release")
	case <-time.After(time.Millisecond * 20):
	}
	cancel()
	if err := <-finished; !errors.Is(err, context.Canceled) {
		t.Fatalf("readiness cancellation cause lost: %v", err)
	}
	select {
	case err := <-nextDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled readiness stranded next normal owner")
	}
}

func TestManagedCapacityAdmissionWaitsForPrecedingVerifiedStop(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	name := fmt.Sprintf("capacity-order-%d", time.Now().UnixNano())
	dir := filepath.Join(root, "resources", name)
	if e := os.MkdirAll(dir, 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(dir, "resource.json"), []byte(`{"acceleration":{"claim":{"resource_kind":"vram","default_preferred_bytes":1073741824,"default_priority":"batch"}}}`), 0600); e != nil {
		t.Fatal(e)
	}
	db := filepath.Join(t.TempDir(), "capacity.db")
	open := func(ctx context.Context) (capacity.AdmitStore, error) {
		return capacity.NewSQLiteStore(ctx, capacity.Config{DBPath: db})
	}
	opts := capacity.AdmitOptions{Root: root, ResourceName: name, OpenStore: open, Source: capacity.StaticSource{}, EnforceEnv: capacity.EnforceAdvisory}
	initial, e := capacity.AdmitResource(ctx, opts)
	if e != nil {
		t.Fatal(e)
	}
	held, e := acquireManagedServiceLifecycleLock(ctx, name)
	if e != nil {
		t.Fatal(e)
	}
	defer held()
	lockPath, _ := artifactlock.LockPath("resource-lifecycle:" + name)
	t.Cleanup(func() { _ = os.Remove(lockPath) })
	entered := make(chan struct{})
	outcome := errors.New("fake verified start entered with live reservation")
	c := NewController(root, t.TempDir())
	c.NoReclaim = true
	c.admitCapacity = func(ctx context.Context, actual capacity.AdmitOptions, _ func(context.Context) error) error {
		close(entered)
		if actual.ResourceName != name || !actual.NoReclaim {
			t.Error("wrong admission owner or lost no-reclaim policy")
		}
		return capacity.WithResourceAdmission(ctx, opts, func(ctx context.Context) error {
			s, e := open(ctx)
			if e != nil {
				return e
			}
			defer s.Close()
			claims, e := s.ListClaims(ctx, capacity.ClaimFilter{OwnerID: name, Statuses: capacity.ActiveClaimStatuses()})
			if e != nil || len(claims) != 1 || claims[0].ClaimID == initial.ClaimID {
				t.Errorf("queued start lacked a fresh live claim after stop: %+v %v", claims, e)
			}
			return outcome
		})
	}
	manifest := managedServiceTestManifest("")
	manifest.Name = name
	done := make(chan error, 1)
	go func() {
		done <- (managedServiceDriver{}).Run(ctx, c, Resource{Name: name}, manifest, "start", nil, io.Discard, io.Discard)
	}()
	select {
	case <-entered:
		t.Fatal("pending start admitted outside held lifecycle lock")
	case <-time.After(20 * time.Millisecond):
	}
	if e := capacity.ReleaseResourceAdmissionAfterStop(ctx, opts, func() error { return nil }); e != nil {
		t.Fatal(e)
	}
	held()
	select {
	case e := <-done:
		if !errors.Is(e, outcome) {
			t.Fatalf("driver did not reach admission after stop: %v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("start remained queued")
	}
}
