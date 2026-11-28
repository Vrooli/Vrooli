package executionartifactlock

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAcquireCoordinatesIndependentServiceInstances(t *testing.T) {
	root := t.TempDir()
	id := uuid.New()
	releaseFirst, err := Acquire(context.Background(), root, id)
	if err != nil {
		t.Fatal(err)
	}
	defer releaseFirst()
	acquired := make(chan func(), 1)
	failed := make(chan error, 1)
	go func() {
		release, err := Acquire(context.Background(), root, id)
		if err != nil {
			failed <- err
			return
		}
		acquired <- release
	}()
	select {
	case release := <-acquired:
		release()
		t.Fatal("second service acquired the same execution lock concurrently")
	case err := <-failed:
		t.Fatal(err)
	case <-time.After(40 * time.Millisecond):
	}
	releaseFirst()
	select {
	case release := <-acquired:
		release()
	case err := <-failed:
		t.Fatal(err)
	case <-time.After(time.Second):
		t.Fatal("second service did not acquire lock after release")
	}
}
