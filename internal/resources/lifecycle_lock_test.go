package resources

import (
	"fmt"
	"testing"
	"time"
)

func TestManagedServiceLifecycleLockSerializesConcurrentMutations(t *testing.T) {
	resource := fmt.Sprintf("lifecycle-lock-test-%d", time.Now().UnixNano())
	entered := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- withManagedServiceLifecycleLock(resource, func() error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered

	secondEntered := make(chan struct{})
	secondDone := make(chan error, 1)
	go func() {
		secondDone <- withManagedServiceLifecycleLock(resource, func() error {
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
