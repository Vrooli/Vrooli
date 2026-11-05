package lifecycle

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestAwaitPolicyMath(t *testing.T) {
	cases := []struct {
		name   string
		policy AwaitPolicy
		// doneAt makes the condition report done on the Nth evaluation (0 = never).
		doneAt     int
		wantErr    error
		wantEvals  int
		wantSleeps []time.Duration
	}{
		{
			name:       "done on first evaluation sleeps zero times",
			policy:     AwaitPolicy{Timeout: 10 * time.Second, Interval: time.Second},
			doneAt:     1,
			wantEvals:  1,
			wantSleeps: nil,
		},
		{
			name:       "fixed interval until done",
			policy:     AwaitPolicy{Timeout: 10 * time.Second, Interval: time.Second},
			doneAt:     3,
			wantEvals:  3,
			wantSleeps: []time.Duration{time.Second, time.Second},
		},
		{
			name:       "inclusive deadline expires when now reaches deadline",
			policy:     AwaitPolicy{Timeout: 2 * time.Second, Interval: time.Second},
			wantErr:    ErrAwaitExpired,
			wantEvals:  3, // evals at 0s, 1s sleep, 2s: third eval then now>=deadline
			wantSleeps: []time.Duration{time.Second, time.Second},
		},
		{
			name:       "strict-after deadline grants one extra tick",
			policy:     AwaitPolicy{Timeout: 2 * time.Second, Interval: time.Second, ExpireStrictlyAfter: true},
			wantErr:    ErrAwaitExpired,
			wantEvals:  4, // eval at exactly 2s is NOT expired; one more sleep then expire at 3s
			wantSleeps: []time.Duration{time.Second, time.Second, time.Second},
		},
		{
			name:       "max attempts bound",
			policy:     AwaitPolicy{MaxAttempts: 3, Interval: time.Second},
			wantErr:    ErrAwaitExpired,
			wantEvals:  3,
			wantSleeps: []time.Duration{time.Second, time.Second},
		},
		{
			name:       "backoff doubles up to cap",
			policy:     AwaitPolicy{Timeout: time.Minute, Interval: time.Second, MaxInterval: 5 * time.Second},
			doneAt:     6,
			wantEvals:  6,
			wantSleeps: []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 5 * time.Second, 5 * time.Second},
		},
		{
			name:       "max attempts of one evaluates exactly once",
			policy:     AwaitPolicy{MaxAttempts: 1, Interval: time.Second},
			wantErr:    ErrAwaitExpired,
			wantEvals:  1,
			wantSleeps: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clock := newFakeAwaitClock()
			evals := 0
			err := Await(AwaitClock{Now: clock.Now, Sleep: clock.Sleep}, tc.policy, func() (bool, error) {
				evals++
				return tc.doneAt > 0 && evals >= tc.doneAt, nil
			})
			if !errors.Is(err, tc.wantErr) && !(err == nil && tc.wantErr == nil) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if evals != tc.wantEvals {
				t.Fatalf("evaluations = %d, want %d", evals, tc.wantEvals)
			}
			if !durationsEqual(clock.sleeps, tc.wantSleeps) {
				t.Fatalf("sleeps = %v, want %v", clock.sleeps, tc.wantSleeps)
			}
		})
	}
}

func TestAwaitConditionErrorIsFatal(t *testing.T) {
	clock := newFakeAwaitClock()
	boom := fmt.Errorf("boom")
	evals := 0
	err := Await(AwaitClock{Now: clock.Now, Sleep: clock.Sleep},
		AwaitPolicy{Timeout: time.Minute, Interval: time.Second},
		func() (bool, error) {
			evals++
			if evals == 2 {
				return false, boom
			}
			return false, nil
		})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
	if evals != 2 {
		t.Fatalf("evaluations = %d, want 2", evals)
	}
	if len(clock.sleeps) != 1 {
		t.Fatalf("sleeps = %v, want exactly one", clock.sleeps)
	}
}

func TestHealthAwaitPolicyResolution(t *testing.T) {
	cases := []struct {
		name           string
		timeoutMillis  int
		intervalMillis int
		wantTimeout    time.Duration
		wantInterval   time.Duration
	}{
		{"defaults", 0, 0, 30 * time.Second, 500 * time.Millisecond},
		{"manifest overrides", 5000, 250, 5 * time.Second, 250 * time.Millisecond},
		{"interval capped at two seconds", 0, 5000, 30 * time.Second, 2 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policy := healthAwaitPolicy(tc.timeoutMillis, tc.intervalMillis)
			if policy.Timeout != tc.wantTimeout {
				t.Fatalf("timeout = %v, want %v", policy.Timeout, tc.wantTimeout)
			}
			if policy.Interval != tc.wantInterval {
				t.Fatalf("interval = %v, want %v", policy.Interval, tc.wantInterval)
			}
			if !policy.ExpireStrictlyAfter {
				t.Fatal("health policy must keep strict-after expiry (degraded-grace boundary)")
			}
		})
	}
}
