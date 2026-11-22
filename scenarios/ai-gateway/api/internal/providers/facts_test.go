package providers

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// countingRunner answers every command with its own call number, so a test can
// tell a reused answer from a fresh one. A non-nil gate holds every call until
// it is closed.
type countingRunner struct {
	calls atomic.Int32
	fail  atomic.Bool
	gate  chan struct{}
}

func (r *countingRunner) Run(ctx context.Context, command Command) (Result, error) {
	n := r.calls.Add(1)
	if r.gate != nil {
		<-r.gate
	}
	if r.fail.Load() {
		return Result{}, &CommandError{Code: "exit_error", Command: command.String(), ExitCode: 1, Stderr: "refused"}
	}
	return Result{Stdout: string(rune('0' + n))}, nil
}

type factClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *factClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *factClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// testLifetime refreshes after 15 seconds and serves for two minutes.
var testLifetime = FactLifetime{RefreshAfter: 15 * time.Second, MaxAge: 2 * time.Minute}

func newFactFixture() (*FactCache, *countingRunner, *factClock) {
	runner := &countingRunner{}
	clock := &factClock{now: time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)}
	return NewFactCache(runner, testLifetime, WithFactClock(clock.Now)), runner, clock
}

var rolesFact = Command{Name: "resource-ollama", Args: []string{"policy", "roles", "--json"}, Fact: true}

// waitForCalls waits for background refreshes, which run off the caller's path.
func waitForCalls(t *testing.T, runner *countingRunner, want int32) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for runner.calls.Load() < want && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := runner.calls.Load(); got != want {
		t.Fatalf("resource calls = %d, want %d", got, want)
	}
}

func TestFactCacheReusesFactsWithinTheirLifetime(t *testing.T) {
	cache, runner, clock := newFactFixture()
	first, err := cache.Run(context.Background(), rolesFact)
	if err != nil || first.Stdout != "1" {
		t.Fatalf("first read = %q, %v", first.Stdout, err)
	}
	clock.Advance(10 * time.Second)
	again, err := cache.Run(context.Background(), rolesFact)
	if err != nil || again.Stdout != "1" {
		t.Fatalf("read inside the lifetime = %q, %v; want the reused answer", again.Stdout, err)
	}
	if runner.calls.Load() != 1 {
		t.Fatalf("resource calls = %d, want 1", runner.calls.Load())
	}
	clock.Advance(testLifetime.MaxAge)
	fresh, err := cache.Run(context.Background(), rolesFact)
	if err != nil || fresh.Stdout != "2" {
		t.Fatalf("read past the maximum age = %q, %v; want a fresh answer", fresh.Stdout, err)
	}
}

// Callers that arrive a minute or more apart still plan without waiting: a
// fact past its refresh age but within its maximum age is served at once
// while it is re-read in the background.
func TestFactCacheServesAcrossIdleGapsWithoutWaiting(t *testing.T) {
	cache, runner, clock := newFactFixture()
	if _, err := cache.Run(context.Background(), rolesFact); err != nil {
		t.Fatal(err)
	}
	runner.gate = make(chan struct{})
	clock.Advance(90 * time.Second)
	served, err := cache.Run(context.Background(), rolesFact)
	if err != nil || served.Stdout != "1" {
		t.Fatalf("read after an idle gap = %q, %v; want the held answer at once", served.Stdout, err)
	}
	waitForCalls(t, runner, 2)
	close(runner.gate)
}

func TestFactCacheNeverReusesActions(t *testing.T) {
	cache, runner, _ := newFactFixture()
	action := Command{Name: "resource-ollama", Args: []string{"gateway", "embed", "--role", "embedding.default"}}
	for range 3 {
		if _, err := cache.Run(context.Background(), action); err != nil {
			t.Fatal(err)
		}
	}
	if runner.calls.Load() != 3 {
		t.Fatalf("an action ran %d times for 3 calls; actions must always reach the resource", runner.calls.Load())
	}
}

// Steady traffic should never wait on a fact refresh: past its refresh age a
// fact keeps serving while one background read replaces it.
func TestFactCacheRefreshesAheadOfExpiryWithoutBlocking(t *testing.T) {
	cache, runner, clock := newFactFixture()
	if _, err := cache.Run(context.Background(), rolesFact); err != nil {
		t.Fatal(err)
	}
	clock.Advance(20 * time.Second)
	served, err := cache.Run(context.Background(), rolesFact)
	if err != nil || served.Stdout != "1" {
		t.Fatalf("read past the refresh age = %q, %v; want the current answer served immediately", served.Stdout, err)
	}
	waitForCalls(t, runner, 2)
	clock.Advance(11 * time.Second) // younger than the refresh age again
	refreshed, err := cache.Run(context.Background(), rolesFact)
	if err != nil || refreshed.Stdout != "2" {
		t.Fatalf("read after refresh = %q, %v; want the refreshed answer", refreshed.Stdout, err)
	}
	if runner.calls.Load() != 2 {
		t.Fatalf("resource calls = %d, want 2", runner.calls.Load())
	}
}

func TestFactCacheSharesOneReadAmongConcurrentCallers(t *testing.T) {
	cache, runner, _ := newFactFixture()
	runner.gate = make(chan struct{})
	var wg sync.WaitGroup
	results := make([]string, 8)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := cache.Run(context.Background(), rolesFact)
			if err != nil {
				t.Error(err)
			}
			results[i] = result.Stdout
		}()
	}
	waitForCalls(t, runner, 1)
	close(runner.gate)
	wg.Wait()
	for _, result := range results {
		if result != "1" {
			t.Fatalf("concurrent readers saw %v; want one shared answer", results)
		}
	}
}

// A caller that gives up must not fail the others sharing its read.
func TestFactCacheCallerCancellationDoesNotFailSharedRead(t *testing.T) {
	cache, runner, _ := newFactFixture()
	runner.gate = make(chan struct{})
	cancelled, cancel := context.WithCancel(context.Background())
	errs := make(chan error, 1)
	go func() {
		_, err := cache.Run(cancelled, rolesFact)
		errs <- err
	}()
	waitForCalls(t, runner, 1)
	cancel()
	if err := <-errs; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled caller error = %v", err)
	}
	done := make(chan Result, 1)
	go func() {
		result, _ := cache.Run(context.Background(), rolesFact)
		done <- result
	}()
	close(runner.gate)
	if result := <-done; result.Stdout != "1" {
		t.Fatalf("waiting caller got %q; want the shared read's answer", result.Stdout)
	}
}

func TestFactCacheRemembersFailuresBriefly(t *testing.T) {
	cache, runner, clock := newFactFixture()
	runner.fail.Store(true)
	for range 3 {
		if _, err := cache.Run(context.Background(), rolesFact); err == nil {
			t.Fatal("a failed fact read must fail")
		}
	}
	if runner.calls.Load() != 1 {
		t.Fatalf("resource calls = %d; a failure inside its window must not be repeated", runner.calls.Load())
	}
	runner.fail.Store(false)
	clock.Advance(factFailureTTL)
	if result, err := cache.Run(context.Background(), rolesFact); err != nil || result.Stdout != "2" {
		t.Fatalf("read after the failure window = %q, %v; want recovery", result.Stdout, err)
	}
}

// A failed background refresh keeps serving the good fact it was replacing.
func TestFactCacheKeepsGoodFactWhenRefreshFails(t *testing.T) {
	cache, runner, clock := newFactFixture()
	if _, err := cache.Run(context.Background(), rolesFact); err != nil {
		t.Fatal(err)
	}
	runner.fail.Store(true)
	clock.Advance(20 * time.Second)
	if _, err := cache.Run(context.Background(), rolesFact); err != nil {
		t.Fatal(err)
	}
	waitForCalls(t, runner, 2)
	if result, err := cache.Run(context.Background(), rolesFact); err != nil || result.Stdout != "1" {
		t.Fatalf("read after a failed refresh = %q, %v; want the good fact", result.Stdout, err)
	}
}

func TestInvalidateFactsDropsOneResourceAndReportsReuse(t *testing.T) {
	cache, runner, _ := newFactFixture()
	ollama := Adapter{Provider: ProviderOllama, CommandName: "resource-ollama", Runner: cache}
	other := Command{Name: "resource-ollaya", Args: []string{"policy", "roles", "--json"}, Fact: true}
	for _, command := range []Command{rolesFact, other} {
		if _, err := cache.Run(context.Background(), command); err != nil {
			t.Fatal(err)
		}
	}
	mark := ollama.FactMark()
	if !ollama.InvalidateFacts(mark) {
		t.Fatal("a fact read before the mark was reused")
	}
	if ollama.InvalidateFacts(AnyFact) {
		t.Fatal("a second invalidation holds nothing")
	}
	if _, err := cache.Run(context.Background(), rolesFact); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Run(context.Background(), other); err != nil {
		t.Fatal(err)
	}
	if runner.calls.Load() != 3 {
		t.Fatalf("resource calls = %d; only the invalidated resource is read again", runner.calls.Load())
	}
	if ollama.InvalidateFacts(mark) {
		t.Fatal("a fact read after the mark is the request's own read, not a reused one")
	}
	if (Adapter{Provider: ProviderOllama, CommandName: "resource-ollama", Runner: runner}).InvalidateFacts(AnyFact) {
		t.Fatal("an adapter without a fact cache holds nothing")
	}
}

// A read that started before an invalidation must not store what it read.
func TestFactReadRacingAnInvalidationIsNotStored(t *testing.T) {
	cache, runner, _ := newFactFixture()
	runner.gate = make(chan struct{})
	done := make(chan struct{})
	go func() {
		_, _ = cache.Run(context.Background(), rolesFact)
		close(done)
	}()
	waitForCalls(t, runner, 1)
	cache.Invalidate("resource-ollama", AnyFact)
	close(runner.gate)
	<-done
	if _, err := cache.Run(context.Background(), rolesFact); err != nil {
		t.Fatal(err)
	}
	if runner.calls.Load() != 2 {
		t.Fatalf("resource calls = %d; the raced read must not have been reused", runner.calls.Load())
	}
}

func TestReportedProviderTimingReadsResourceMeasurements(t *testing.T) {
	timing := reportedProviderTiming(`{"embedding":[0.1],"admission_wait_ns":1200,"provider_duration_ns":91000000}`)
	if timing != (ProviderTiming{AdmissionWaitNS: 1200, RuntimeNS: 91000000}) {
		t.Fatalf("timing = %+v", timing)
	}
	for _, output := range []string{"plain text", `{"content":"no timing"}`, `{"provider_duration_ns":-5}`, `{broken`} {
		if timing := reportedProviderTiming(output); timing != (ProviderTiming{}) {
			t.Fatalf("output %q reported %+v; want zero timing", output, timing)
		}
	}
}
