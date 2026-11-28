package process

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestIncarnationIdentifiesCurrentProcess(t *testing.T) {
	got, err := Incarnation(os.Getpid())
	if err != nil {
		t.Fatalf("Incarnation(current PID): %v", err)
	}
	if got == "" {
		t.Fatal("Incarnation(current PID) returned an empty token")
	}
	if _, err := Incarnation(-1); err == nil {
		t.Fatal("Incarnation(invalid PID) succeeded")
	}
}

type incarnationHandle struct {
	alive  bool
	err    error
	closed bool
}

func (h *incarnationHandle) Alive() (bool, error) { return h.alive, h.err }
func (h *incarnationHandle) Signal(bool) error    { return nil }
func (h *incarnationHandle) Close() error         { h.closed = true; return nil }

func TestPinRecordRequiresLiveExactIncarnationAcrossOpen(t *testing.T) {
	for _, tc := range []struct {
		name, before, after string
		alive               bool
		wantOK              bool
	}{
		{"matching", "boot:123", "boot:123", true, true},
		{"stale-record", "boot:999", "boot:999", true, false},
		{"reuse-during-open", "boot:123", "boot:999", true, false},
		{"exit-during-open", "boot:123", "boot:123", false, false},
		{"missing-before", "", "boot:123", true, false},
		{"missing-after", "boot:123", "", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := &incarnationHandle{alive: tc.alive}
			calls, opened := 0, false
			got, err := pinRecord(Record{PID: 42, Generation: "generation", Incarnation: "boot:123"}, func(int) (Handle, error) { opened = true; return h, nil }, func(int) (string, error) {
				calls++
				if calls == 1 {
					return tc.before, nil
				}
				return tc.after, nil
			}, func(int) (string, error) { return "generation", nil })
			if (err == nil) != tc.wantOK {
				t.Fatalf("handle=%v error=%v", got, err)
			}
			if !tc.wantOK && opened && !h.closed {
				t.Fatal("rejected handle leaked")
			}
		})
	}
}

func TestPinRecordRefusesLegacyAndIncompleteLaunchIdentity(t *testing.T) {
	for _, record := range []Record{{PID: 42}, {PID: 42, Generation: "g"}, {PID: 42, Incarnation: "boot:123"}} {
		_, err := pinRecord(record, func(int) (Handle, error) { t.Fatal("opened incomplete identity"); return nil, nil }, func(int) (string, error) { t.Fatal("inspected incomplete identity"); return "", nil }, func(int) (string, error) { t.Fatal("inspected incomplete identity"); return "", nil })
		if err == nil {
			t.Fatal("incomplete record accepted")
		}
	}
}

func TestPinRecordRejectsSameTickPIDReuseByLaunchGeneration(t *testing.T) {
	for _, tc := range []struct{ name, before, after string }{{"before-open", "replacement", "replacement"}, {"during-open", "generation", "replacement"}, {"unobservable-after", "generation", ""}} {
		t.Run(tc.name, func(t *testing.T) {
			h := &incarnationHandle{alive: true}
			calls := 0
			_, err := pinRecord(Record{PID: 42, Generation: "generation", Incarnation: "boot:same-tick"}, func(int) (Handle, error) { return h, nil }, func(int) (string, error) { return "boot:same-tick", nil }, func(int) (string, error) {
				calls++
				if calls == 1 {
					return tc.before, nil
				}
				return tc.after, nil
			})
			if err == nil {
				t.Fatal("replacement with same starttime tick accepted")
			}
		})
	}
}

func TestAwaitExitNeverConvertsTimeoutOrInspectionErrorToExit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := AwaitExit(ctx, &incarnationHandle{alive: true}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel result: %v", err)
	}
	want := errors.New("unknown")
	if err := AwaitExit(context.Background(), &incarnationHandle{err: want}); !errors.Is(err, want) {
		t.Fatalf("inspection result: %v", err)
	}
	if err := AwaitExit(context.Background(), &incarnationHandle{}); err != nil {
		t.Fatalf("pinned exit: %v", err)
	}
}

func TestLiveRecordsNeverProbesVerifiedExitTombstones(t *testing.T) {
	previous := isPIDRunningFn
	defer func() { isPIDRunningFn = previous }()
	probes := 0
	isPIDRunningFn = func(int) bool { probes++; return true }
	if records := LiveRecords([]Record{{PID: 42, Status: "exited_verified", Generation: "old"}}); len(records) != 0 || probes != 0 {
		t.Fatalf("verified old generation became numeric teardown candidate: records=%v probes=%d", records, probes)
	}
}
