package gate

import "testing"

// [REQ:BRG-P1-002] The core of the cross-OS deployment gate: per-OS node
// verdicts aggregate into a single pass/fail. A gate passes ONLY when every
// target OS validated green; ANY failing OS — a non-zero run, an aborted run, OR
// a target OS with no eligible node — fails the gate. While targets are still
// running and none has failed, the gate is PENDING.
func TestAggregateVerdict(t *testing.T) {
	res := func(d OSDisposition) OSResult { return OSResult{Disposition: d} }

	cases := []struct {
		name    string
		results []OSResult
		want    GateVerdict
	}{
		{
			name:    "all OSes green → passed",
			results: []OSResult{res(OSDispositionPassed), res(OSDispositionPassed), res(OSDispositionPassed)},
			want:    VerdictPassed,
		},
		{
			name:    "one OS failed (non-zero run) → failed, even with others green",
			results: []OSResult{res(OSDispositionPassed), res(OSDispositionFailed), res(OSDispositionPassed)},
			want:    VerdictFailed,
		},
		{
			name:    "a target OS with no eligible node → failed (cannot validate that OS)",
			results: []OSResult{res(OSDispositionPassed), res(OSDispositionNoNode)},
			want:    VerdictFailed,
		},
		{
			name:    "an aborted run → failed",
			results: []OSResult{res(OSDispositionPassed), res(OSDispositionAborted)},
			want:    VerdictFailed,
		},
		{
			name:    "a dispatch rejection → failed",
			results: []OSResult{res(OSDispositionPassed), res(OSDispositionDispatchFailed)},
			want:    VerdictFailed,
		},
		{
			name:    "still running, none failed yet → pending",
			results: []OSResult{res(OSDispositionPassed), res(OSDispositionPending)},
			want:    VerdictPending,
		},
		{
			name:    "a failure dominates a still-pending target (fail fast)",
			results: []OSResult{res(OSDispositionPending), res(OSDispositionFailed)},
			want:    VerdictFailed,
		},
		{
			name:    "no classifiable results → failed (nothing validated)",
			results: []OSResult{},
			want:    VerdictFailed,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := aggregateVerdict(tc.results); got != tc.want {
				t.Fatalf("aggregateVerdict = %v, want %v", got, tc.want)
			}
		})
	}
}

// [REQ:BRG-P1-002] The summary counts a gate carries must tally the per-OS
// dispositions exactly, so the operator sees passed/failed/pending at a glance
// without walking the ledger.
func TestCounts(t *testing.T) {
	results := []OSResult{
		{Disposition: OSDispositionPassed},
		{Disposition: OSDispositionPassed},
		{Disposition: OSDispositionFailed},
		{Disposition: OSDispositionNoNode}, // counts as failed
		{Disposition: OSDispositionPending},
	}
	passed, failed, pending := counts(results)
	if passed != 2 || failed != 2 || pending != 1 {
		t.Fatalf("counts = (passed=%d failed=%d pending=%d), want (2,2,1)", passed, failed, pending)
	}
}

// normaliseOSes lowercases, trims, dedupes, and preserves first-seen order so a
// gate validates each OS exactly once.
func TestNormaliseOSes(t *testing.T) {
	got := normaliseOSes([]string{" Linux ", "darwin", "LINUX", "", "windows", "Darwin"})
	want := []string{"linux", "darwin", "windows"}
	if len(got) != len(want) {
		t.Fatalf("normaliseOSes len = %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("normaliseOSes[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
