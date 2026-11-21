package rootmetrics

import (
	"testing"
	"time"
)

func TestDeferredDoesNotWaitForUnavailableObservationStore(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	d := beginDeferred("", "bundle_publication", func(string, string) *Attempt { close(entered); <-release; return nil })
	<-entered
	finished := make(chan (<-chan struct{}), 1)
	go func() { d.Bundle("", nil); d.Event("terminal", "failed"); finished <- d.Finish() }()
	var done <-chan struct{}
	select {
	case done = <-finished:
	case <-time.After(time.Second):
		t.Fatal("publisher waited for observation storage")
	}
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("observer did not stop")
	}
}
