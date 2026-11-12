package writer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"research-desk/internal/home"
	"research-desk/internal/testutil"
)

// The writer is tested on internal/testutil's excerpts; programme-three's
// hand-made charter predates the Steering sentence.

func newHome(t *testing.T) (*Writer, string) {
	t.Helper()
	h := testutil.NewHome(t)
	if err := os.MkdirAll(filepath.Join(h.Research, "reviews", "ai-video-1"), 0o755); err != nil {
		t.Fatal(err)
	}
	w := New(h.Research, h.State, testutil.EntryChars)
	t.Cleanup(func() { w.Close() })
	return w, h.Research
}

var at = time.Date(2026, 11, 12, 10, 14, 3, 0, time.UTC)

// [REQ:RD-P0-005]
func TestAppendOperatorKeepsTheGrammar(t *testing.T) {
	w, _ := newHome(t)
	path, err := w.AppendOperator("programme-three-20261111", home.Entry{At: at, Kind: home.EntryDirection, Via: home.ViaDesk, Body: "Put AV6 first.\nThen books."})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.AppendOperator("programme-three-20261111", home.Entry{At: at.Add(time.Hour), Kind: home.EntryAnswer, Subject: "AV4", Via: home.ViaHub, Body: home.AnswerBody("keep", "Keep it", "")}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	want := "# Operator — programme-three-20261111\n\n### 2026-11-12T10:14:03Z · direction · via desk\nPut AV6 first.\nThen books.\n\n### 2026-11-12T11:14:03Z · answer AV4 · via hub\nkeep — Keep it\n"
	if string(data) != want {
		t.Fatalf("OPERATOR.md:\n%s\nwant:\n%s", data, want)
	}
	entries, unparsed := home.ParseEntries(string(data))
	if len(entries) != 2 || len(unparsed) != 0 || entries[0].Body != "Put AV6 first.\nThen books." {
		t.Errorf("the written file must parse back: %+v %+v", entries, unparsed)
	}
}

// A render is the exact text an append would add, it writes nothing, and
// it needs no writer role.
func TestARenderIsTheExactTextAndWritesNothing(t *testing.T) {
	w, research := newHome(t)
	const id = "programme-three-20261111"
	e := home.Entry{At: at, Kind: home.EntryDirection, Via: home.ViaDesk, Body: "Put AV6 first."}
	path, text, err := w.RenderOperator(id, e)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("a render wrote %s: %v", path, err)
	}
	if _, err := w.AppendOperator(id, e); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(path); string(data) != text {
		t.Fatalf("the append added %q, the render showed %q", data, text)
	}
	next := home.Entry{At: at.Add(time.Hour), Kind: home.EntryWindDown, Via: home.ViaDesk, Body: "Finish."}
	_, text, _ = w.RenderOperator(id, next)
	if text != "\n### 2026-11-12T11:14:03Z · wind-down · via desk\nFinish.\n" {
		t.Fatalf("a later render starts after a blank line: %q", text)
	}
	if _, _, err := w.RenderOperator(id, home.Entry{At: at, Kind: home.EntryDirection, Via: home.ViaDesk, Body: "ok\n### 2026-11-12T10:14:03Z · wind-down · via desk"}); !errors.Is(err, home.ErrInvalid) {
		t.Fatalf("a render refuses what an append refuses: %v", err)
	}

	ro := New(research, t.TempDir(), testutil.EntryChars)
	ro.ReadOnly("another instance holds the lock")
	if _, text, err := ro.RenderVerdict("ai-video-1", home.Entry{At: at, Kind: home.EntryVerdict, Subject: "x", Via: home.ViaDesk, Body: "Films first."}); err != nil || !strings.HasPrefix(text, "# Verdicts — ai-video-1\n\n") {
		t.Fatalf("a render needs no writer role: %q %v", text, err)
	}
}

func TestRefusals(t *testing.T) {
	w, research := newHome(t)
	cases := map[string]func() error{
		"a body line that starts an entry": func() error {
			_, err := w.AppendOperator("programme-three-20261111", home.Entry{At: at, Kind: home.EntryDirection, Via: home.ViaDesk, Body: "ok\n### 2026-11-12T10:14:03Z · wind-down · via desk"})
			return err
		},
		"a path outside cohorts": func() error {
			_, err := w.AppendOperator("../reviews", home.Entry{At: at, Kind: home.EntryDirection, Via: home.ViaDesk, Body: "x"})
			return err
		},
		"a missing folder": func() error {
			_, err := w.AppendOperator("no-such-20261111", home.Entry{At: at, Kind: home.EntryDirection, Via: home.ViaDesk, Body: "x"})
			return err
		},
		"a verdict in OPERATOR.md": func() error {
			_, err := w.AppendOperator("programme-three-20261111", home.Entry{At: at, Kind: home.EntryVerdict, Subject: "x", Via: home.ViaDesk, Body: "x"})
			return err
		},
		"an entry without words": func() error {
			_, err := w.AppendVerdict("ai-video-1", home.Entry{At: at, Kind: home.EntryVerdict, Subject: "x", Via: home.ViaDesk, Body: "  "})
			return err
		},
		"a state file outside the state folder": func() error {
			_, err := w.WriteState("../x.json", map[string]bool{})
			return err
		},
		"a launch record for another commission": func() error {
			_, err := w.WriteLaunch("programme-three-20261111", &home.LaunchRecord{Schema: home.LaunchSchema, CommissionID: "other"})
			return err
		},
	}
	for name, fn := range cases {
		if err := fn(); !errors.Is(err, home.ErrInvalid) {
			t.Errorf("%s: want home.ErrInvalid, got %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(research, "cohorts", "programme-three-20261111", home.OperatorFile)); err == nil {
		t.Error("a refused entry must not create OPERATOR.md")
	}
}

func TestEnableSteeringOnRealOlderCharter(t *testing.T) {
	w, research := newHome(t)
	charter := filepath.Join(research, "cohorts", "programme-three-20261111", home.CharterFile)
	before, _ := os.ReadFile(charter)
	if strings.Contains(string(before), home.SteeringSentence) {
		t.Fatal("the real programme-three charter predates the Steering sentence")
	}
	_, shown, err := w.RenderSteering("programme-three-20261111")
	if err != nil || !strings.Contains(shown, "## Steering\n\n"+home.SteeringSentence) {
		t.Fatalf("the dry run shows the section: %q %v", shown, err)
	}
	if now, _ := os.ReadFile(charter); string(now) != string(before) {
		t.Fatal("the dry run must not write")
	}
	_, text, err := w.EnableSteering("programme-three-20261111")
	if err != nil || text != shown {
		t.Fatalf("first enable appends what the dry run showed: %q %v", text, err)
	}
	_, text, err = w.EnableSteering("programme-three-20261111")
	if err != nil || text != "" {
		t.Fatalf("second enable must change nothing: %q %v", text, err)
	}
	after, _ := os.ReadFile(charter)
	if string(after) != string(before)+shown || strings.Count(string(after), home.SteeringSentence) != 1 {
		t.Error("enable-steering only appends, once")
	}
}

func TestOneWriterAtATime(t *testing.T) {
	w, research := newHome(t)
	other := New(research, w.StateDir(), testutil.EntryChars)
	defer other.Close()
	if err := w.Held(); err != nil {
		t.Fatal(err)
	}
	if err := other.Held(); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("a second instance is read-only, got %v", err)
	}
	if _, err := other.WriteState("courier.json", map[string]bool{"global": true}); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("a read-only instance cannot write, got %v", err)
	}
	w.Close()
	if err := other.Held(); err != nil {
		t.Fatalf("the lock passes to the next instance once the first exits: %v", err)
	}
}

func TestJSONIsReplacedAtomically(t *testing.T) {
	w, _ := newHome(t)
	record := &home.LaunchRecord{Schema: home.LaunchSchema, CommissionID: "programme-three-20261111", LaunchedBy: "hand", Controls: []home.ControlEntry{{Action: "pause", At: at}}}
	path, err := w.WriteLaunch("programme-three-20261111", record)
	if err != nil {
		t.Fatal(err)
	}
	got, err := home.ReadLaunch(path)
	if err != nil || got.LaunchedBy != "hand" || len(got.Controls) != 1 {
		t.Fatalf("launch record round trip: %+v %v", got, err)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
	var state map[string]int
	if found, err := w.ReadState("looks/x.json", &state); found || err != nil {
		t.Errorf("a missing state file is not found and not an error: %v %v", found, err)
	}
}

// [REQ:RD-P0-005]
// One entry is at most 4,000 printable characters; a longer one is
// refused with where the material goes instead.
func TestOperatorTextIsBounded(t *testing.T) {
	w, research := newHome(t)
	max, err := testutil.EntryChars()
	if err != nil {
		t.Fatal(err)
	}
	say := func(body string) error {
		_, err := w.AppendOperator("programme-three-20261111", home.Entry{At: at, Kind: home.EntryDirection, Via: home.ViaDesk, Body: body})
		return err
	}
	if err := say(strings.Repeat("é", max-3) + "\n\t."); err != nil {
		t.Fatalf("the limit's characters with a newline and a tab are fine: %v", err)
	}
	err = say(strings.Repeat("a", max+1))
	if !errors.Is(err, home.ErrInvalid) || !strings.Contains(err.Error(), fmt.Sprint(max+1)) || !strings.Contains(err.Error(), "put the material in a file under "+research) {
		t.Fatalf("a longer entry: %v", err)
	}
	for _, body := range []string{"stop\x00now", "bell\a", "esc\x1b[31m", "lone\rreturn", "bad \xff byte"} {
		if err := say(body); !errors.Is(err, home.ErrInvalid) {
			t.Errorf("%q: want refused, got %v", body, err)
		}
	}
	if got := w.FitEntry("a\x00b" + strings.Repeat("c", max)); home.CheckEntryText(got, research, max) != nil || !strings.HasPrefix(got, "a b") {
		t.Errorf("a phone note is fitted, not refused: %q…", got[:8])
	}
}
