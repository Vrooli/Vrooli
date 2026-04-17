package metrics

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRecorderWritesValidJSONL(t *testing.T) {
	home := t.TempDir()
	r := New(home, nil)
	if r.Disabled() {
		t.Fatal("recorder should be enabled by default")
	}
	r.Record(Event{
		StartedAt:  time.Unix(1700000000, 0).UTC(),
		Command:    "scenario",
		Args:       []string{"list"},
		Argc:       1,
		DurationMs: 42,
		ExitCode:   0,
		CLIVersion: "9.9.9",
		Hostname:   "testhost",
		PID:        4321,
	})
	r.Record(Event{
		StartedAt:  time.Unix(1700000005, 0).UTC(),
		Command:    "resource",
		Args:       []string{"list"},
		Argc:       1,
		DurationMs: 17,
	})

	path := filepath.Join(home, ".vrooli", "metrics", "timings.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read timings file: %v", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var events []Event
	for scanner.Scan() {
		var e Event
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			t.Fatalf("unmarshal line %q: %v", scanner.Text(), err)
		}
		events = append(events, e)
	}
	if len(events) != 2 {
		t.Fatalf("want 2 events, got %d", len(events))
	}
	if events[0].Command != "scenario" || events[0].DurationMs != 42 || events[0].CLIVersion != "9.9.9" {
		t.Errorf("event[0] fields wrong: %+v", events[0])
	}
	if events[1].Command != "resource" {
		t.Errorf("event[1] command wrong: %q", events[1].Command)
	}

	// README bootstrap
	readme := filepath.Join(home, ".vrooli", "metrics", "README.md")
	if _, err := os.Stat(readme); err != nil {
		t.Errorf("expected README at %s: %v", readme, err)
	}
}

func TestRecorderDisabledByEnv(t *testing.T) {
	t.Setenv(EnvDisable, "0")
	home := t.TempDir()
	r := New(home, nil)
	if !r.Disabled() {
		t.Fatal("recorder should be disabled when VROOLI_METRICS=0")
	}
	r.Record(Event{Command: "scenario"})

	path := filepath.Join(home, ".vrooli", "metrics", "timings.jsonl")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected no timings file, got err=%v", err)
	}
}

func TestRecorderDisableEnvAcceptsCommonValues(t *testing.T) {
	for _, v := range []string{"0", "false", "FALSE", "no", "off", " off "} {
		if !envDisabled(v) {
			t.Errorf("envDisabled(%q) = false, want true", v)
		}
	}
	for _, v := range []string{"", "1", "true", "yes", "on"} {
		if envDisabled(v) {
			t.Errorf("envDisabled(%q) = true, want false", v)
		}
	}
}

func TestRecorderNonFatalOnIOError(t *testing.T) {
	// Point recorder at a path where the parent exists but is not a dir.
	home := t.TempDir()
	blocker := filepath.Join(home, ".vrooli")
	if err := os.WriteFile(blocker, []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("setup blocker: %v", err)
	}
	var captured error
	r := New(home, func(err error) { captured = err })
	r.Record(Event{Command: "scenario"})
	if captured == nil {
		t.Fatal("expected onError to be invoked when mkdir fails")
	}
	if !errors.Is(captured, captured) { // sanity: non-nil
		t.Fatal("captured error should be non-nil")
	}
}

func TestRecorderNilSafe(t *testing.T) {
	var r *Recorder
	r.Record(Event{Command: "scenario"}) // must not panic
	if !r.Disabled() {
		t.Error("nil recorder should report disabled")
	}
	if r.Path() != "" {
		t.Error("nil recorder Path() should be empty")
	}
}

func TestRedactArgsStripsFlagValues(t *testing.T) {
	got := RedactArgs([]string{"--token=abc", "list", "--json", "--", "--hidden=xyz"})
	want := []string{"--token", "list", "--json"}
	if !equalStrings(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestRedactArgsRedactsSecretPositional(t *testing.T) {
	got := RedactArgs([]string{"start", "password=hunter2", "key=hunter2"})
	want := []string{"start", "<redacted>", "<redacted>"}
	if !equalStrings(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestRedactArgsCapsPositional(t *testing.T) {
	in := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	got := RedactArgs(in)
	if len(got) != maxPositionalArgs {
		t.Errorf("len(got) = %d, want %d", len(got), maxPositionalArgs)
	}
}

func TestRedactArgsTruncatesLong(t *testing.T) {
	long := strings.Repeat("x", maxPositionalArgLen+50)
	got := RedactArgs([]string{long})
	if len(got[0]) != maxPositionalArgLen {
		t.Errorf("len(got[0]) = %d, want %d", len(got[0]), maxPositionalArgLen)
	}
}

func TestClassifyError(t *testing.T) {
	if ClassifyError(nil) != "" {
		t.Error("nil error should classify as empty")
	}
	if ClassifyError(errors.New("x")) == "" {
		t.Error("non-nil error should classify as non-empty")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
