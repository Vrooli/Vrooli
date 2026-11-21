package databasetest

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestFakePingerReturnsConfiguredErrorAndCountsCalls(t *testing.T) {
	want := errors.New("database unavailable")
	pinger := &FakePinger{PingErr: want}
	for count := 1; count <= 3; count++ {
		if err := pinger.PingContext(context.Background()); !errors.Is(err, want) {
			t.Fatalf("PingContext error = %v, want %v", err, want)
		}
		if got := pinger.Calls.Load(); got != int64(count) {
			t.Fatalf("Calls = %d after call %d", got, count)
		}
	}
}

func TestFakePingerCountsConcurrentCalls(t *testing.T) {
	const callers = 100
	pinger := &FakePinger{}
	var wait sync.WaitGroup
	wait.Add(callers)
	for i := 0; i < callers; i++ {
		go func() {
			defer wait.Done()
			_ = pinger.PingContext(context.Background())
		}()
	}
	wait.Wait()
	if got := pinger.Calls.Load(); got != callers {
		t.Fatalf("Calls = %d, want %d", got, callers)
	}
}
