//go:build linux

package rootmetrics

import "testing"

func TestInitiatorOnlyUsesAllowlistedAncestry(t *testing.T) {
	records := map[int]struct {
		name   string
		parent int
	}{10: {"vrooli-watchdog", 9}, 9: {"private-secret", 8}, 8: {"make", 1}}
	read := func(pid int) (string, int, bool) { r, ok := records[pid]; return r.name, r.parent, ok }
	if got := classifyAncestry(10, read); got != "make" {
		t.Fatalf("got %q", got)
	}
	delete(records, 8)
	if got := classifyAncestry(10, read); got != "unknown" {
		t.Fatalf("leaked or invented identity %q", got)
	}
	records[9] = struct {
		name   string
		parent int
	}{"private-secret", 9}
	if got := classifyAncestry(10, read); got != "unknown" {
		t.Fatalf("cycle: %q", got)
	}
}
