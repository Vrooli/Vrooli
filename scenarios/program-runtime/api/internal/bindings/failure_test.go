package bindings

import (
	"errors"
	"fmt"
	"testing"
)

func TestClassifyFailureCoversTheClosedVocabulary(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status string
		class  string
		http   int
	}{
		{"unreachable probe", errors.New("binding device-control/device/list is unreachable: scenario API is unavailable"), "unavailable", "scenario_unreachable", 0},
		{"dial refused", fmt.Errorf("invoke x: %w", errors.New("dial tcp 127.0.0.1:1: connect: connection refused")), "unavailable", "scenario_unreachable", 0},
		{"gateway answer", errors.New("invoke x: remote status 503 Service Unavailable: draining"), "unavailable", "scenario_unreachable", 503},
		{"remote 500", errors.New("invoke x: remote status 500 Internal Server Error: boom"), "failed", "remote_error", 500},
		{"remote 404", errors.New("invoke x: remote status 404 Not Found: no such flow"), "failed", "remote_error", 404},
		{"no grant", errors.New(`destructive binding "x" requires an explicit grant`), "refused", "no_grant", 0},
		{"confirmation", errors.New(`binding "x" requires explicit confirmation`), "refused", "no_grant", 0},
		{"inference spend", errors.New("inference_spend_exceeded: ceiling=1 micros accumulated=2 micros"), "refused", "inference_spend_exceeded", 0},
		{"delegated spend", errors.New("delegated_run_spend_exceeded: ceiling=1 micros accumulated=2 micros"), "refused", "delegated_run_spend_exceeded", 0},
		{"ambiguous rows", errors.New("binding x has no determinable primary response field; candidate repeated fields: a, b"), "failed", "ambiguous_response", 0},
		{"rows override", errors.New("binding x rows must name one of its repeated response fields"), "failed", "ambiguous_response", 0},
		{"list kwarg", errors.New("decode binding arguments: proto: syntax error (line 1:10): unexpected token"), "failed", "invalid_input", 0},
		{"client-side flag", errors.New(`argument "facets-json" is client-side only and cannot be used in a program binding`), "failed", "invalid_input", 0},
		{"ungoverned", fmt.Errorf("%w: search-hub/query/missing", errNoBinding), "failed", "no_governed_binding", 0},
		{"deadline", errors.New("context deadline exceeded"), "failed", "deadline_exceeded", 0},
		{"wall budget", errors.New("wall budget exhausted: ceiling=1s consumed=2s"), "failed", "deadline_exceeded", 0},
		{"anything else", errors.New("read x response: unexpected EOF"), "failed", "binding_error", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyFailure(tc.err)
			if got.Status != tc.status || got.Class != tc.class || got.HTTPStatus != tc.http {
				t.Fatalf("ClassifyFailure(%q) = %+v, want status=%s class=%s http=%d", tc.err, got, tc.status, tc.class, tc.http)
			}
		})
	}
}

func TestClassifyFailureNilIsEmpty(t *testing.T) {
	if got := ClassifyFailure(nil); got != (Failure{}) {
		t.Fatalf("nil error classified as %+v", got)
	}
}

// An unreachable scenario that also mentions a deadline is unreachable: the
// rule order is part of the contract, because a program treats `unavailable`
// as "retry later" and `deadline_exceeded` as "my budget was wrong".
func TestClassifyFailureUnreachableOutranksDeadline(t *testing.T) {
	got := ClassifyFailure(errors.New("binding x is unreachable: live reachability unavailable: timed out"))
	if got.Class != "scenario_unreachable" {
		t.Fatalf("got %+v", got)
	}
}
