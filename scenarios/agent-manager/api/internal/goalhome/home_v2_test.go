package goalhome

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// v2Home returns the v2-bas goal home with files replaced by edits (an edit on
// a missing name creates it; returning "" deletes it).
func v2Home(t *testing.T, edits map[string]func(string) string) *Home {
	t.Helper()
	files := fstest.MapFS{}
	err := fs.WalkDir(os.DirFS("testdata/v2-bas"), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := os.ReadFile(filepath.Join("testdata", "v2-bas", name))
		files[name] = &fstest.MapFile{Data: data}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	for name, edit := range edits {
		text := ""
		if file, ok := files[name]; ok {
			text = string(file.Data)
		}
		out := edit(text)
		if out == "" {
			delete(files, name)
			continue
		}
		files[name] = &fstest.MapFile{Data: []byte(out)}
	}
	home, err := LoadV2(files)
	if err != nil {
		t.Fatalf("LoadV2: %v", err)
	}
	return home
}

// activeEpoch returns the single non-accepted epoch the board would select, or
// nil. It mirrors the board's newest-last, non-accepted scan.
func activeEpoch(home *Home) *Epoch {
	for i := len(home.Epochs) - 1; i >= 0; i-- {
		if !home.Epochs[i].Epoch.IsAccepted() {
			return home.Epochs[i].Epoch
		}
	}
	return nil
}

func acceptedEpochs(home *Home) []*Epoch {
	var out []*Epoch
	for _, e := range home.Epochs {
		if e.Epoch.IsAccepted() {
			out = append(out, e.Epoch)
		}
	}
	return out
}

func TestLoadV2Destination(t *testing.T) {
	home := v2Home(t, nil)
	dest := home.Goal.Destination
	if dest == nil || dest.Metric != "runtime lines" || dest.Op != "<=" || dest.Target != 205000 {
		t.Fatalf("destination = %+v", dest)
	}
	if home.Goal.BudgetSeconds != 72*3600 {
		t.Fatalf("budget seconds = %d, want %d", home.Goal.BudgetSeconds, 72*3600)
	}
	if home.Goal.Current == nil || home.Goal.Current.Value != 253452 || home.Goal.Current.At.IsZero() {
		t.Fatalf("current = %+v", home.Goal.Current)
	}
}

func TestLoadV2CharterBudgetFormats(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want int64
	}{
		{"72h", 72 * 3600},
		{"90m", 90 * 60},
		{"3 days", 3 * 86400},
		{"2 weeks", 2 * 604800},
		{"", 0},
		{"soon", 0},
	} {
		got := parseBudgetSeconds(tc.raw)
		if got != tc.want {
			t.Fatalf("parseBudgetSeconds(%q) = %d, want %d", tc.raw, got, tc.want)
		}
	}
}

func TestLoadV2ActiveUnitAndWorkers(t *testing.T) {
	home := v2Home(t, nil)
	if got := home.Epochs[len(home.Epochs)-1].Number; got != 26 {
		t.Fatalf("active unit identity = E%d, want E26", got)
	}
	active := activeEpoch(home)
	if active == nil {
		t.Fatal("no active epoch resolved from STATUS ## Active")
	}
	if got := active.Fields["status"]; got != "acceptance review (BAS-FB-064 gate amendment)" {
		t.Fatalf("active status = %q", got)
	}
	workers := WorkerRunIDs(active.Fields["workers"])
	if len(workers) != 1 || workers[0] != "e8c322ca-72e7-434c-9d37-1d2cf853961f" {
		t.Fatalf("active_workers = %v", workers)
	}
}

func TestLoadV2KeepsIndependentActiveUnits(t *testing.T) {
	home := v2Home(t, map[string]func(string) string{StatusFile: func(s string) string {
		return strings.Replace(s, "## Active\n", "## Active\n\n- unit: updates  status: in-progress  workers: 11111111-1111-4111-8111-111111111111\n", 1)
	}})
	var ids []string
	for _, u := range home.Epochs {
		if !u.Epoch.IsAccepted() {
			ids = append(ids, u.ID())
		}
	}
	if strings.Join(ids, ",") != "updates,E26" {
		t.Fatalf("independent active units = %v", ids)
	}
}

func TestLoadV2AcceptedUnits(t *testing.T) {
	home := v2Home(t, nil)
	accepted := acceptedEpochs(home)
	if len(accepted) != 3 {
		t.Fatalf("accepted units = %d, want 3", len(accepted))
	}
	// Newest accepted unit is E25 (last ## Accepted units line before Active).
	last := accepted[len(accepted)-1]
	if got := home.Epochs[len(accepted)-1].Number; got != 25 {
		t.Fatalf("last accepted identity = E%d, want E25", got)
	}
	acc := last.LastAcceptance()
	if acc == nil || acc.At.IsZero() {
		t.Fatalf("E25 acceptance = %+v", acc)
	}
	if acc.Yield == nil || acc.Yield.Value != -332 || acc.Yield.Unit != "runtime lines" {
		t.Fatalf("E25 yield = %+v", acc.Yield)
	}
	if len(acc.Gates) != 3 || acc.Gates[0] != "G1" || acc.Gates[2] != "G3" {
		t.Fatalf("E25 gates = %v", acc.Gates)
	}
	// E23 carries two worker UUIDs with parenthesized notes; both must survive
	// into Fields["workers"] so the board's per-unit spend reads the same UUIDs.
	e23 := accepted[0]
	ids := WorkerRunIDs(e23.Fields["workers"])
	if len(ids) != 2 || ids[0] != "c33912b5-1b51-49b3-8ddb-aad93600b001" || ids[1] != "6980e693-3ffd-42fe-b383-c2f0689354b1" {
		t.Fatalf("E23 workers = %v", ids)
	}
	if y := e23.LastAcceptance().Yield; y == nil || y.Value != -328 {
		t.Fatalf("E23 yield = %+v", y)
	}
}

func TestLoadV2ReopenedOverridesAccepted(t *testing.T) {
	// A ## Reopened entry dated after E25's ACCEPTED ts must reopen it, so the
	// board sees E25 as active (not accepted) exactly as a post-acceptance
	// directive did in v1.
	home := v2Home(t, map[string]func(string) string{
		LogFile: func(s string) string {
			return s[:len(s)-len("(none)\n")] + "- E25 | 2026-10-07T02:00:00Z | coordinator reopened the timeline controller\n"
		},
	})
	var e25 *Epoch
	for _, e := range home.Epochs {
		if acc := e.Epoch.LastAcceptance(); acc != nil && !acc.At.IsZero() && acc.At.Format("15:04") == "01:28" {
			e25 = e.Epoch
		}
	}
	if e25 == nil {
		t.Fatal("E25 not found")
	}
	if e25.IsAccepted() {
		t.Fatal("E25 reopened after acceptance must not count as accepted")
	}
	if len(e25.ReopenedBy()) == 0 {
		t.Fatalf("E25 should be reopened: %+v", e25.ReopenedBy())
	}
	if n := len(acceptedEpochs(home)); n != 2 {
		t.Fatalf("accepted after reopen = %d, want 2", n)
	}
}

func TestLoadV2ReopenBeforeAcceptanceIgnored(t *testing.T) {
	// A ## Reopened entry dated at or before the ACCEPTED ts never reopens.
	home := v2Home(t, map[string]func(string) string{
		LogFile: func(s string) string {
			return s[:len(s)-len("(none)\n")] + "- E25 | 2026-10-07T01:00:00Z | stale note before acceptance\n"
		},
	})
	if n := len(acceptedEpochs(home)); n != 3 {
		t.Fatalf("a pre-acceptance reopen must not reopen: accepted = %d, want 3", n)
	}
}

func TestLoadV2MissingFileIsUnknownNotZero(t *testing.T) {
	// Missing CHARTER.md leaves the destination unknown (nil), never a zero
	// target that the board would read as a real goal.
	home := v2Home(t, map[string]func(string) string{
		CharterFile: func(string) string { return "" },
	})
	if home.Goal.Destination != nil {
		t.Fatalf("missing CHARTER must leave destination unknown, got %+v", home.Goal.Destination)
	}
	if home.Goal.BudgetSeconds != 0 {
		t.Fatalf("missing CHARTER budget = %d, want 0", home.Goal.BudgetSeconds)
	}
	found := false
	for _, m := range home.Missing {
		if m == CharterFile {
			found = true
		}
	}
	if !found {
		t.Fatalf("CHARTER.md should be listed Missing: %v", home.Missing)
	}
	// Missing STATUS.md leaves the active epoch unknown, not a zeroed one.
	home = v2Home(t, map[string]func(string) string{
		StatusFile: func(string) string { return "" },
	})
	if activeEpoch(home) != nil {
		t.Fatalf("missing STATUS must leave active epoch unknown")
	}
}

func TestLoadV2MalformedLogIsPartial(t *testing.T) {
	// A malformed ## Accepted units line yields a partial board projection
	// (EpochsComplete=false), exactly as an unreadable epoch set does in v1, so
	// the board never reads accepted/active facts from it.
	malformed := func(string) string {
		return "# BAS log\n\n## Accepted units\n\n- this line has no ACCEPTED head at all\n"
	}
	// Strict LoadV2 errors.
	files := fstest.MapFS{
		LogFile: {Data: []byte(malformed(""))},
	}
	if _, err := LoadV2(files); err == nil {
		t.Fatal("strict LoadV2 must error on a malformed LOG")
	}
	// Tolerant board projection marks the home partial instead.
	home, err := LoadBoardProjectionV2(files)
	if err != nil {
		t.Fatalf("LoadBoardProjectionV2: %v", err)
	}
	if home.EpochsComplete {
		t.Fatal("a malformed LOG must make the board projection partial")
	}
	found := false
	for _, name := range home.IncompleteEpochFiles {
		if name == LogFile {
			found = true
		}
	}
	if !found {
		t.Fatalf("LOG.md should be named incomplete: %v", home.IncompleteEpochFiles)
	}
}

func TestLoadV2QueueAndWorkaroundsMirrorV1(t *testing.T) {
	home := v2Home(t, nil)
	q := home.Queue
	if len(q.Handoffs) != 1 || q.Handoffs[0].Title != "Handoff" {
		t.Fatalf("handoffs = %+v", q.Handoffs)
	}
	if len(q.NeedsOperator) != 0 || len(q.Next) != 5 || len(q.ReadyNext()) != 0 || len(q.WaitingOnOthers) != 2 {
		t.Fatalf("queue sections misread: %+v", q)
	}
	open := 0
	for _, e := range home.Workarounds.Entries {
		if e.Open {
			open++
		}
	}
	if len(home.Workarounds.Entries) != 4 || open != 3 {
		t.Fatalf("workarounds = %d entries, %d open; want 4 and 3", len(home.Workarounds.Entries), open)
	}
	// FEEDBACK.md is unchanged, so ParseFeedback reads the same four entries.
	var ids []string
	for _, e := range home.Feedback.Entries {
		ids = append(ids, e.ID)
	}
	if len(ids) != 4 || ids[0] != "BAS-FB-064" {
		t.Fatalf("feedback ids = %v", ids)
	}
}
