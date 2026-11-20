package capacitysync

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type execRecorder struct {
	calls []string
	out   string
	err   error
}

func (r *execRecorder) exec(_ context.Context, name string, args ...string) ([]byte, error) {
	call := name + " " + strings.Join(args, " ")
	r.calls = append(r.calls, call)
	if r.err != nil {
		return nil, r.err
	}
	return []byte(r.out), nil
}

func handlers(rec *execRecorder) *Handlers {
	return &Handlers{
		Exec:   rec.exec,
		GetEnv: func(string) string { return "" },
	}
}

// Observation must never mutate the ledger. Re-admitting from here created a
// claim and returned zero, which the loop then released — once per poll.
func TestObserveWithoutAClaimDoesNotStartTheResource(t *testing.T) {
	rec := &execRecorder{out: `{"claims":[]}`}
	footprint, err := handlers(rec).Observe(context.Background())
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if footprint.Bytes != 0 || footprint.Parked {
		t.Fatalf("footprint = %+v, want an empty unparked result", footprint)
	}
	for _, call := range rec.calls {
		if strings.Contains(call, "resource start") {
			t.Fatalf("observation mutated the ledger: %v", rec.calls)
		}
	}
}

func TestObserveReportsADegradedCPUClaimAsParked(t *testing.T) {
	rec := &execRecorder{out: `{"claims":[{"owner_id":"ollaya","amount_bytes":0,"observed_bytes":0,"status":"degraded"}]}`}
	footprint, err := handlers(rec).Observe(context.Background())
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if !footprint.Parked {
		t.Fatalf("footprint = %+v, want a parked zero-memory claim", footprint)
	}
	if footprint.Bytes != 0 {
		t.Fatalf("bytes = %d, want 0", footprint.Bytes)
	}
}

func TestObserveKeepsTheClaimAmountWhenNoMeasurementExists(t *testing.T) {
	rec := &execRecorder{out: `{"claims":[{"owner_id":"ollaya","amount_bytes":2147483648,"observed_bytes":0,"status":"granted"}]}`}
	footprint, err := handlers(rec).Observe(context.Background())
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if footprint.Parked || footprint.Bytes != 2147483648 {
		t.Fatalf("footprint = %+v, want the granted amount and not parked", footprint)
	}
}

func TestObservePrefersTheMeasuredFootprint(t *testing.T) {
	rec := &execRecorder{out: `{"claims":[{"owner_id":"ollaya","amount_bytes":2147483648,"observed_bytes":1073741824,"status":"granted"}]}`}
	footprint, err := handlers(rec).Observe(context.Background())
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if footprint.Bytes != 1073741824 {
		t.Fatalf("bytes = %d, want the observed figure", footprint.Bytes)
	}
}

func TestObserveFailsClosedWhenTheLedgerIsUnavailable(t *testing.T) {
	rec := &execRecorder{err: errors.New("exit status 1")}
	if _, err := handlers(rec).Observe(context.Background()); err == nil {
		t.Fatal("Observe swallowed a ledger failure")
	}
}
