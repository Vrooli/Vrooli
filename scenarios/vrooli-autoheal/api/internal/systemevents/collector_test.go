package systemevents

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/journal"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// kernelGrepArgs reconstructs the exact journalctl argv the collector issues
// for a kernel-signal grep, matching journal.buildArgs ordering so the
// MockExecutor can key on it.
func kernelGrepArgs(boot, afterCursor string, showCursor bool, tail int) []string {
	args := []string{"--no-pager", "-o", "json", "-k", "-b", boot}
	if afterCursor != "" {
		args = append(args, "--after-cursor="+afterCursor)
	}
	if showCursor {
		args = append(args, "--show-cursor")
	}
	args = append(args, "-g", kernelGrepPattern)
	if tail > 0 {
		args = append(args, "-n", itoa(tail))
	}
	return args
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func cmdKey(args []string) string {
	key := "journalctl"
	for _, a := range args {
		key += " " + a
	}
	return key
}

// newKernelCollector builds a Linux HostCollector wired to a MockExecutor and
// the given cursor store, with deterministic time.
func newKernelCollector(t *testing.T, cursors CursorStore) (*HostCollector, *checks.MockExecutor) {
	t.Helper()
	mock := checks.NewMockExecutor()
	// journalctl --version (Available) and --list-boots succeed by default
	// where individually keyed; the per-test responses fill in the rest.
	mock.Responses = map[string]checks.MockResponse{}
	reader := journal.NewReader(mock)
	c := NewHostCollectorWithCursors(&platform.Capabilities{Platform: platform.Linux}, mock, reader, cursors)
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	return c, mock
}

func TestCollectKernelSignalsColdStartBoundedRescan(t *testing.T) {
	cursors := NewMemoryCursorStore()
	c, mock := newKernelCollector(t, cursors)
	// Cold start, no cursor: collector issues a bounded -n rescan of boot 0
	// with --show-cursor and persists the newest cursor.
	mock.Responses[cmdKey(kernelGrepArgs("0", "", true, currentBootRescanTail))] = checks.MockResponse{Output: []byte(
		`{"__CURSOR":"s=z;i=10","__REALTIME_TIMESTAMP":"1750766400000000","MESSAGE":"NVRM: Xid error"}` + "\n",
	)}

	events := c.collectCurrentBootKernelSignals(context.Background(), journal.BootRecord{Index: 0, BootID: "cur"})
	if len(events) != 1 || events[0].Category != "driver" {
		t.Fatalf("cold-start events = %#v, want one driver event", events)
	}
	state, _ := cursors.GetJournalCursor(context.Background(), kernelSignalSourceKey)
	if state.Cursor != "s=z;i=10" || state.BootID != "cur" {
		t.Fatalf("cursor not advanced: %+v", state)
	}
	if c.ExecsAvoided() != 0 {
		t.Fatalf("execsAvoided = %d, want 0 on cold start", c.ExecsAvoided())
	}
}

func TestCollectKernelSignalsSecondRunReadsOnlyDelta(t *testing.T) {
	cursors := NewMemoryCursorStore()
	_ = cursors.SetJournalCursor(context.Background(), kernelSignalSourceKey, CursorState{Cursor: "s=z;i=10", BootID: "cur"})
	c, mock := newKernelCollector(t, cursors)

	// With a valid cursor for the current boot, the collector reads ONLY the
	// delta via --after-cursor (no bounded -n rescan). Returning the delta key
	// but NOT the bounded-rescan key proves the incremental path was taken.
	mock.Responses[cmdKey(kernelGrepArgs("0", "s=z;i=10", true, 0))] = checks.MockResponse{Output: []byte(
		`{"__CURSOR":"s=z;i=11","__REALTIME_TIMESTAMP":"1750766460000000","MESSAGE":"amdgpu: ring timeout"}` + "\n",
	)}
	// If the collector wrongly issued a bounded rescan, this would surface a
	// second event and we'd catch it.
	mock.Responses[cmdKey(kernelGrepArgs("0", "", true, currentBootRescanTail))] = checks.MockResponse{Output: []byte(
		`{"__CURSOR":"s=z;i=99","__REALTIME_TIMESTAMP":"1750766400000000","MESSAGE":"NVRM: should-not-appear"}` + "\n",
	)}

	events := c.collectCurrentBootKernelSignals(context.Background(), journal.BootRecord{Index: 0, BootID: "cur"})
	if len(events) != 1 || !strings.Contains(events[0].Summary, "amdgpu") {
		t.Fatalf("delta events = %#v, want only the amdgpu delta entry", events)
	}
	if c.ExecsAvoided() != 1 {
		t.Fatalf("execsAvoided = %d, want 1 (incremental read avoided a full rescan)", c.ExecsAvoided())
	}
	state, _ := cursors.GetJournalCursor(context.Background(), kernelSignalSourceKey)
	if state.Cursor != "s=z;i=11" {
		t.Fatalf("cursor not advanced to delta tip: %+v", state)
	}
}

func TestCollectKernelSignalsCursorInvalidationFallsBack(t *testing.T) {
	cursors := NewMemoryCursorStore()
	_ = cursors.SetJournalCursor(context.Background(), kernelSignalSourceKey, CursorState{Cursor: "s=stale;i=1", BootID: "cur"})
	c, mock := newKernelCollector(t, cursors)

	// The incremental read errors (journalctl: "Failed to seek to cursor") —
	// simulating vacuum/rotation. The collector MUST fall back to a bounded
	// rescan and re-emit the events rather than silently dropping them.
	mock.Responses[cmdKey(kernelGrepArgs("0", "s=stale;i=1", true, 0))] = checks.MockResponse{
		Output: []byte("Failed to seek to cursor"),
		Error:  errors.New("exit status 1"),
	}
	mock.Responses[cmdKey(kernelGrepArgs("0", "", true, currentBootRescanTail))] = checks.MockResponse{Output: []byte(
		`{"__CURSOR":"s=fresh;i=5","__REALTIME_TIMESTAMP":"1750766400000000","MESSAGE":"machine check event"}` + "\n",
	)}

	events := c.collectCurrentBootKernelSignals(context.Background(), journal.BootRecord{Index: 0, BootID: "cur"})
	if len(events) != 1 || events[0].Category != "crash" {
		t.Fatalf("fallback events = %#v, want one crash event from the bounded rescan", events)
	}
	state, _ := cursors.GetJournalCursor(context.Background(), kernelSignalSourceKey)
	if state.Cursor != "s=fresh;i=5" {
		t.Fatalf("cursor should re-anchor to fresh rescan tip, got %+v", state)
	}
}

func TestCollectKernelSignalsFailedRescanLeavesCursorUntouched(t *testing.T) {
	cursors := NewMemoryCursorStore()
	_ = cursors.SetJournalCursor(context.Background(), kernelSignalSourceKey, CursorState{Cursor: "s=keep;i=7", BootID: "cur"})
	c, mock := newKernelCollector(t, cursors)

	// Both the incremental read and the bounded fallback fail. The cursor must
	// NOT advance (no silent event loss): the next ingest retries the window.
	mock.Responses[cmdKey(kernelGrepArgs("0", "s=keep;i=7", true, 0))] = checks.MockResponse{Error: errors.New("exit status 1")}
	mock.Responses[cmdKey(kernelGrepArgs("0", "", true, currentBootRescanTail))] = checks.MockResponse{Error: errors.New("exit status 1")}

	events := c.collectCurrentBootKernelSignals(context.Background(), journal.BootRecord{Index: 0, BootID: "cur"})
	if len(events) != 0 {
		t.Fatalf("events on total failure = %#v, want none", events)
	}
	state, _ := cursors.GetJournalCursor(context.Background(), kernelSignalSourceKey)
	if state.Cursor != "s=keep;i=7" {
		t.Fatalf("cursor must be untouched on failed ingest, got %+v", state)
	}
}

func TestCollectHistoricalBootScannedOnce(t *testing.T) {
	cursors := NewMemoryCursorStore()
	c, mock := newKernelCollector(t, cursors)
	hist := journal.BootRecord{Index: -1, BootID: "old"}

	mock.Responses[cmdKey(kernelGrepArgs("old", "", false, currentBootRescanTail))] = checks.MockResponse{Output: []byte(
		`{"__CURSOR":"s=o;i=1","__REALTIME_TIMESTAMP":"1750000000000000","MESSAGE":"NVRM: Xid 79"}` + "\n",
	)}

	// First scan: emits the event and records the marker.
	first := c.collectHistoricalBootKernelSignals(context.Background(), hist)
	if len(first) != 1 {
		t.Fatalf("first historical scan events = %d, want 1", len(first))
	}
	if scanned, _ := cursors.IsBootScanned(context.Background(), kernelSignalSourceKey, "old"); !scanned {
		t.Fatal("historical boot should be marked scanned after first pass")
	}
	if c.ExecsAvoided() != 0 {
		t.Fatalf("execsAvoided after first scan = %d, want 0", c.ExecsAvoided())
	}

	// Second scan: skipped entirely (no events), exec avoided.
	second := c.collectHistoricalBootKernelSignals(context.Background(), hist)
	if len(second) != 0 {
		t.Fatalf("second historical scan events = %d, want 0 (already scanned)", len(second))
	}
	if c.ExecsAvoided() != 1 {
		t.Fatalf("execsAvoided after second scan = %d, want 1", c.ExecsAvoided())
	}
}

func TestCollectHistoricalBootNotMarkedOnFailure(t *testing.T) {
	cursors := NewMemoryCursorStore()
	c, mock := newKernelCollector(t, cursors)
	hist := journal.BootRecord{Index: -2, BootID: "flaky"}
	// journalctl exits 1 for a successful no-match query. A non-empty output
	// makes this mock represent an actual command failure, which is the case
	// this test is intended to protect.
	mock.Responses[cmdKey(kernelGrepArgs("flaky", "", false, currentBootRescanTail))] = checks.MockResponse{Output: []byte("partial output"), Error: errors.New("permission denied")}

	c.collectHistoricalBootKernelSignals(context.Background(), hist)
	if scanned, _ := cursors.IsBootScanned(context.Background(), kernelSignalSourceKey, "flaky"); scanned {
		t.Fatal("a failed historical scan must NOT mark the boot scanned")
	}
}

func TestParseDPKGLogExtractsDriverAndKernelEvents(t *testing.T) {
	events := ParseDPKGLog([]byte(strings.Join([]string{
		"2026-05-06 06:10:23 install linux-image-6.17.0-23-generic:amd64 <none> 6.17.0-23.23~24.04.1",
		"2026-05-08 12:57:16 upgrade nvidia-driver-580-open:amd64 580.126.09-0ubuntu0.24.04.2 580.142-0ubuntu0.24.04.1",
		"2026-05-08 22:59:39 upgrade linux-firmware 20240318.git3b128b60-0ubuntu2.26 20240318.git3b128b60-0ubuntu2.27",
		"2026-05-08 23:00:00 upgrade unrelated-package 1 2",
	}, "\n")))

	if len(events) != 3 {
		t.Fatalf("event count = %d, want 3", len(events))
	}
	if events[0].Category != "kernel" {
		t.Fatalf("first category = %q, want kernel", events[0].Category)
	}
	if events[1].Category != "driver" {
		t.Fatalf("second category = %q, want driver", events[1].Category)
	}
	if events[2].Category != "firmware" {
		t.Fatalf("third category = %q, want firmware", events[2].Category)
	}
}

func TestParseAPTHistoryGroupsRelevantPackages(t *testing.T) {
	events := ParseAPTHistory([]byte(`Start-Date: 2026-05-08  12:57:16
Commandline: apt-get install linux-modules-nvidia-580-open-6.17.0-23-generic
Install: nvidia-firmware-580-580.142:amd64 (580.142-0ubuntu0.24.04.1), linux-modules-nvidia-580-open-6.17.0-23-generic:amd64 (6.17.0-23.23~24.04.1+1)
Upgrade: nvidia-driver-580-open:amd64 (580.126.09-0ubuntu0.24.04.2, 580.142-0ubuntu0.24.04.1)
End-Date: 2026-05-08  12:57:26
`))

	if len(events) < 2 {
		t.Fatalf("event count = %d, want driver and firmware events", len(events))
	}
	categories := map[string]bool{}
	for _, event := range events {
		categories[event.Category] = true
	}
	if !categories["driver"] || !categories["firmware"] {
		t.Fatalf("categories = %#v, want driver and firmware", categories)
	}
}

func TestBuildCorrelationsUsesConservativeTemporalHints(t *testing.T) {
	base := time.Date(2026, 5, 6, 6, 10, 0, 0, time.UTC)
	events := []Event{
		{ID: 1, OccurredAt: base, Source: "dpkg-log", Category: "kernel", Title: "Package install: linux-image", Summary: "install linux-image"},
		{ID: 2, OccurredAt: base.Add(16 * time.Hour), Source: "journalctl", Category: "crash", Title: "Hardware/reset signal", Summary: "unclean reset"},
		{ID: 3, OccurredAt: base.Add(40 * time.Hour), Source: "dpkg-log", Category: "firmware", Title: "Package upgrade: linux-firmware", Summary: "upgrade linux-firmware"},
	}

	correlations := BuildCorrelations(events)
	if len(correlations) < 2 {
		t.Fatalf("correlation count = %d, want at least kernel-before-crash and firmware-after-crash", len(correlations))
	}
	for _, correlation := range correlations {
		if correlation.Confidence != "temporal" {
			t.Fatalf("confidence = %q, want temporal", correlation.Confidence)
		}
		if strings.Contains(strings.ToLower(correlation.Summary), "root cause") {
			t.Fatalf("correlation overclaims causality: %#v", correlation)
		}
	}
}
