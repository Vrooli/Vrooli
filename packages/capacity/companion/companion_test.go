package companion_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/packages/capacity/companion"
)

// Feature: one reporting loop, three observations
//
//	As three resource CLIs
//	I want the claim, resize, release and heartbeat logic in one place
//	So that a fix lands once instead of three times, and a broker outage never
//	stops inference.

// recorder captures the CLI calls the loop makes.
type recorder struct {
	calls   []string
	list    string
	listErr error
	failOn  string
}

func (r *recorder) exec(_ context.Context, name string, args ...string) ([]byte, error) {
	call := name + " " + strings.Join(args, " ")
	r.calls = append(r.calls, call)
	if len(args) >= 2 && args[1] == "list" {
		if r.listErr != nil {
			return nil, r.listErr
		}
		return []byte(r.list), nil
	}
	if r.failOn != "" && strings.Contains(call, r.failOn) {
		return nil, errors.New("broker unavailable")
	}
	return []byte(`{}`), nil
}

func (r *recorder) sawCall(fragment string) bool {
	return slices.ContainsFunc(r.calls, func(call string) bool { return strings.Contains(call, fragment) })
}

func runnerFor(t *testing.T, rec *recorder, footprint companion.Footprint, observeErr error) *companion.Runner {
	t.Helper()
	runner, err := companion.New(companion.Config{
		Resource: "ollama",
		Observer: companion.ObserverFunc(func(context.Context) (companion.Footprint, error) {
			return footprint, observeErr
		}),
		Exec:           rec.exec,
		PreferredBytes: 11 << 30,
		FloorBytes:     3 << 30,
		Priority:       "service",
		YieldWhenIdle:  true,
		IdleGrace:      15 * time.Minute,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return runner
}

const noClaims = `{"claims":[]}`

func oneClaim(amount int64) string {
	return `{"claims":[{"claim_id":"clm-o","owner_id":"ollama","amount_bytes":` + itoa(amount) + `,"generation":3,"status":"granted"}]}`
}

func itoa(v int64) string {
	digits := ""
	if v == 0 {
		return "0"
	}
	for v > 0 {
		digits = string(rune('0'+v%10)) + digits
		v /= 10
	}
	return digits
}

// Scenario: one claim id survives a claim, resize and release cycle.
func TestOneClaimIDSurvivesTheWholeCycle(t *testing.T) {
	// Given a resource that loads a model, grows, then unloads
	steps := []struct {
		scenario  string
		ledger    string
		footprint int64
		wantCall  string
		banned    string
	}{
		{
			scenario:  "Given nothing claimed and a model loaded, Then a claim is created",
			ledger:    noClaims,
			footprint: 4 << 30,
			wantCall:  "capacity claim --owner-kind resource --owner-id ollama",
		},
		{
			scenario:  "Given a claim and a materially larger footprint, Then it is resized in place",
			ledger:    oneClaim(4 << 30),
			footprint: 12 << 30,
			wantCall:  "capacity resize --claim-id clm-o --generation 3",
			banned:    "capacity release",
		},
		{
			scenario:  "Given a claim and everything unloaded, Then it is released",
			ledger:    oneClaim(12 << 30),
			footprint: 0,
			wantCall:  "capacity release --claim-id clm-o",
		},
	}

	for _, step := range steps {
		t.Run(step.scenario, func(t *testing.T) {
			rec := &recorder{list: step.ledger}
			runner := runnerFor(t, rec, companion.Footprint{Bytes: step.footprint}, nil)

			// When the loop syncs once
			runner.SyncOnce(context.Background())

			// Then it takes the expected ledger action
			if !rec.sawCall(step.wantCall) {
				t.Fatalf("calls = %v, want one containing %q", rec.calls, step.wantCall)
			}
			// And never the one that would churn the ledger
			if step.banned != "" && rec.sawCall(step.banned) {
				t.Fatalf("calls = %v, must not contain %q", rec.calls, step.banned)
			}
		})
	}
}

func TestActiveClaimListIsCachedDuringNoopBurst(t *testing.T) {
	rec := &recorder{list: noClaims}
	runner := runnerFor(t, rec, companion.Footprint{Bytes: 0}, nil)
	runner.SyncOnce(context.Background())
	runner.SyncOnce(context.Background())
	listCalls := 0
	for _, call := range rec.calls {
		if strings.Contains(call, "capacity list") {
			listCalls++
		}
	}
	if listCalls != 1 {
		t.Fatalf("capacity list calls = %d, want one during a cached noop burst: %v", listCalls, rec.calls)
	}
}

// Scenario: footprint jitter heartbeats instead of resizing.
func TestSmallFootprintChangeHeartbeatsInsteadOfResizing(t *testing.T) {
	// Given a claim and a footprint that moved less than the threshold
	rec := &recorder{list: oneClaim(4 << 30)}
	runner := runnerFor(t, rec, companion.Footprint{Bytes: 4<<30 + 1<<20}, nil)

	// When the loop syncs
	runner.SyncOnce(context.Background())

	// Then it heartbeats rather than resizing, so jitter does not churn
	if !rec.sawCall("capacity heartbeat --claim-id clm-o") {
		t.Fatalf("calls = %v, want a heartbeat", rec.calls)
	}
	if rec.sawCall("capacity resize") {
		t.Fatalf("calls = %v, jitter must not trigger a resize", rec.calls)
	}
}

// Scenario: an observation that fails leaves the ledger alone.
//
// Treating "could not tell" as "holds nothing" would release a live
// reservation every time a poll timed out.
func TestFailedObservationLeavesTheLedgerUntouched(t *testing.T) {
	// Given an observer that cannot answer
	rec := &recorder{list: oneClaim(4 << 30)}
	runner := runnerFor(t, rec, companion.Footprint{}, errors.New("api unavailable"))

	// When the loop syncs
	runner.SyncOnce(context.Background())

	// Then no ledger mutation happens at all
	for _, call := range rec.calls {
		for _, mutation := range []string{"capacity claim", "capacity release", "capacity resize", "capacity heartbeat"} {
			if strings.Contains(call, mutation) {
				t.Fatalf("an unavailable observation caused %q; unknown is not zero", call)
			}
		}
	}
}

// Scenario: a broker error never stops the loop.
func TestLedgerErrorsFailOpen(t *testing.T) {
	cases := []struct {
		scenario string
		rec      *recorder
	}{
		{scenario: "Given the ledger listing fails, Then the loop claims and continues", rec: &recorder{listErr: errors.New("broker down")}},
		{scenario: "Given the claim call fails, Then the loop continues", rec: &recorder{list: noClaims, failOn: "capacity claim"}},
		{scenario: "Given the resize call fails, Then the loop continues", rec: &recorder{list: oneClaim(1 << 30), failOn: "capacity resize"}},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			// Given a broker that errors
			var log strings.Builder
			runner, err := companion.New(companion.Config{
				Resource: "ollama",
				Observer: companion.ObserverFunc(func(context.Context) (companion.Footprint, error) {
					return companion.Footprint{Bytes: 12 << 30}, nil
				}),
				Exec: tc.rec.exec,
				Log:  &log,
			})
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			// When the loop syncs
			// Then it does not panic and does not stop
			runner.SyncOnce(context.Background())

			// And the failure is reported rather than swallowed silently
			if tc.rec.failOn != "" && !strings.Contains(log.String(), "leaving the ledger unchanged") {
				t.Fatalf("log = %q, want it to say the ledger was left unchanged", log.String())
			}
		})
	}
}

// Scenario: an observer that reports activity has it forwarded.
func TestActivityIsForwardedWhenTheObserverReportsIt(t *testing.T) {
	// Given an observer that knows the resource is working
	rec := &recorder{list: oneClaim(4 << 30)}
	runner, err := companion.New(companion.Config{
		Resource: "whisper",
		Observer: companion.ObserverFunc(func(context.Context) (companion.Footprint, error) {
			return companion.Footprint{Bytes: 4 << 30, Activity: companion.ActivityActive}, nil
		}),
		Exec: rec.exec,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	rec.list = `{"claims":[{"claim_id":"clm-w","owner_id":"whisper","amount_bytes":4294967296,"generation":2,"status":"granted"}]}`

	// When the loop syncs
	runner.SyncOnce(context.Background())

	// Then the activity state reaches the broker, which is the idleness truth
	// source and must never be inferred from the footprint alone
	if !rec.sawCall("capacity activity --claim-id clm-w --generation 2 --state active") {
		t.Fatalf("calls = %v, want the activity report", rec.calls)
	}
}

// Scenario: a misconfigured companion is refused at construction.
func TestNewRefusesAnIncompleteConfig(t *testing.T) {
	cases := []struct {
		scenario string
		cfg      companion.Config
		wantErr  string
	}{
		{scenario: "Given no resource name, Then it is refused", cfg: companion.Config{Observer: companion.ObserverFunc(nil), Exec: func(context.Context, string, ...string) ([]byte, error) { return nil, nil }}, wantErr: "resource name is required"},
		{scenario: "Given no observer, Then it is refused", cfg: companion.Config{Resource: "ollama", Exec: func(context.Context, string, ...string) ([]byte, error) { return nil, nil }}, wantErr: "observer is required"},
		{scenario: "Given no exec seam, Then it is refused", cfg: companion.Config{Resource: "ollama", Observer: companion.ObserverFunc(nil)}, wantErr: "exec seam is required"},
	}
	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			_, err := companion.New(tc.cfg)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("New() = %v, want an error containing %q", err, tc.wantErr)
			}
		})
	}
}

// Scenario: cancelling the context stops the loop cleanly.
func TestRunReturnsCleanlyOnCancellation(t *testing.T) {
	// Given a running loop
	rec := &recorder{list: noClaims}
	runner := runnerFor(t, rec, companion.Footprint{}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// When the context is already cancelled
	// Then Run returns nil: being asked to stop is not a failure
	if err := runner.Run(ctx); err != nil {
		t.Fatalf("Run() = %v, want nil on cancellation", err)
	}
}
