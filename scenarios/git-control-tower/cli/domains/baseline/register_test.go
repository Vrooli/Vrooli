package baseline

import (
	"context"
	"io"
	"testing"
)

func TestBaselineClientTimeoutIsBounded(t *testing.T) {
	// The diff deadline must be a finite ceiling, never zero (which http.Client
	// treats as "no timeout" — the bare-Background hang this fixes).
	if baselineClientTimeout <= 0 {
		t.Fatalf("baselineClientTimeout must be a positive ceiling, got %v", baselineClientTimeout)
	}
	if snapshotStartCeiling <= 0 {
		t.Fatalf("snapshotStartCeiling must be a positive ceiling, got %v", snapshotStartCeiling)
	}
}

func TestExitCodeForVerdict(t *testing.T) {
	cases := map[string]int{
		"clean":          exitOK,
		"changed":        exitOK, // advisory visual tier — never gates
		"new-failure":    exitOK, // added by the change, not a regression — safe to proceed
		"preexisting":    exitOK, // inherited, not caused by the change
		"regression":     exitRegression,
		"not-comparable": exitNotComparable,
		"":               exitOK,
	}
	for verdict, want := range cases {
		if got := exitCodeForVerdict(verdict); got != want {
			t.Errorf("exitCodeForVerdict(%q) = %d, want %d", verdict, got, want)
		}
	}
}

func TestVerdictMark(t *testing.T) {
	cases := map[string]string{
		"clean":          "✓",
		"regression":     "✗",
		"new-failure":    "✗",
		"preexisting":    "•",
		"not-comparable": "?",
	}
	for v, want := range cases {
		if got := verdictMark(v); got != want {
			t.Errorf("verdictMark(%q) = %q want %q", v, got, want)
		}
	}
}

func TestRegisterExcludesEmptyCreateAndPerSurfaceEdit(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
	group := Register(nil)
	for _, command := range group.Subcommands {
		if command.Name == "create" || command.Name == "edit" {
			t.Fatalf("removed mutable baseline command %q is still registered", command.Name)
		}
	}
}

func TestDurableWaitReconnectsOnceByIDAfterUnexpectedEOF(t *testing.T) { // [REQ:GCT-DURABLE-OPS-P0]
	calls := 0
	blocking := []bool{}
	value, recovered, err := durableReadWithEOFRecovery(context.Background(), true, func(_ context.Context, wait bool) (string, error) {
		calls++
		blocking = append(blocking, wait)
		if calls == 1 {
			return "", io.ErrUnexpectedEOF
		}
		return "durable-state", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !recovered || value != "durable-state" || calls != 2 {
		t.Fatalf("value=%q recovered=%v calls=%d", value, recovered, calls)
	}
	if len(blocking) != 2 || !blocking[0] || blocking[1] {
		t.Fatalf("wait modes = %v, want [true false]", blocking)
	}
}

func TestDurableReadDoesNotRetryNonBlockingOrNonEOF(t *testing.T) { // [REQ:GCT-DURABLE-OPS-P0]
	for _, tc := range []struct {
		name string
		wait bool
		err  error
	}{
		{name: "inspect", wait: false, err: io.ErrUnexpectedEOF},
		{name: "typed failure", wait: true, err: io.ErrClosedPipe},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			_, recovered, err := durableReadWithEOFRecovery(context.Background(), tc.wait, func(context.Context, bool) (string, error) {
				calls++
				return "", tc.err
			})
			if err == nil || recovered || calls != 1 {
				t.Fatalf("err=%v recovered=%v calls=%d", err, recovered, calls)
			}
		})
	}
}

func TestAttachmentDeadlineUsesOneRecoveryRead(t *testing.T) { // [REQ:GCT-DURABLE-OPS-P0]
	calls := 0
	_, recovered, err := durableReadWithEOFRecovery(context.Background(), true, func(context.Context, bool) (string, error) {
		calls++
		if calls == 1 {
			return "", context.DeadlineExceeded
		}
		return "pending", nil
	})
	if err != nil || !recovered || calls != 2 {
		t.Fatalf("err=%v recovered=%v calls=%d", err, recovered, calls)
	}
}
