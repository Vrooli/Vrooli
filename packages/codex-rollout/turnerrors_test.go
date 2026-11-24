package codexrollout

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The fixtures are real rollout records from Web Console Codex homes
// (November 2026), with only the account's credit balance zeroed.

func TestReadLastTurnFindsTheErrorThatEndedTheTurn(t *testing.T) {
	cases := []struct {
		file, turn, at, info, message string
	}{
		// steward-free-host-20261120: the turn's error is the last record.
		{"rollout-transport-disconnect.jsonl", "01a1fa52-1543-7551-8765-6ddf28400d00", "2026-11-20T20:13:09.522Z", "other", "stream disconnected before completion"},
		// steward-test-o8-20261120: sub-agent completions follow the error.
		{"rollout-subagents-after-error.jsonl", "01a1fa71-eaee-7351-9c57-5431a41ee59a", "2026-11-20T20:12:00.780Z", "other", "stream disconnected before completion"},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			got, err := ReadLastTurn(filepath.Join("testdata", tc.file), 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			want, _ := time.Parse(time.RFC3339Nano, tc.at)
			if !got.EndedOnError() || got.TurnID != tc.turn || !got.At.Equal(want) {
				t.Fatalf("outcome %+v", got)
			}
			if got.Error.Info != tc.info || !strings.Contains(got.Error.Message, tc.message) {
				t.Fatalf("error %+v", got.Error)
			}
		})
	}
}

func TestReadLastTurnReportsRunningAndInterruptedTurns(t *testing.T) {
	dir := t.TempDir()
	// The disconnect fixture's first record starts the turn; alone, it is a
	// running turn.
	started := firstLine(t, "rollout-transport-disconnect.jsonl")
	path := filepath.Join(dir, "running.jsonl")
	if err := os.WriteFile(path, append(started, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ReadLastTurn(path, 1<<20)
	if err != nil || got.Event != EventTurnStarted || got.EndedOnError() || got.TurnID != "01a1fa52-1543-7551-8765-6ddf28400d00" {
		t.Fatalf("running: %+v %v", got, err)
	}
	got, err = ReadLastTurn(filepath.Join("testdata", "rollout-interrupted.jsonl"), 1<<20)
	if err != nil || got.Event != EventTurnAborted || got.AbortReason != "interrupted" || got.EndedOnError() {
		t.Fatalf("interrupted: %+v %v", got, err)
	}
}

func TestReadLastTurnCrossesChunksAndStopsAtItsLimit(t *testing.T) {
	// A long record after the boundary (a compacted history is megabytes)
	// spans several backward reads; the boundary before it is still found.
	data, err := os.ReadFile(filepath.Join("testdata", "rollout-subagents-after-error.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	filler := `{"timestamp":"2026-11-20T20:30:00Z","type":"response_item","payload":{"type":"message","text":"` + strings.Repeat("x", 3*tailChunk) + `"}}` + "\n"
	data = append(data, filler...)
	path := filepath.Join(t.TempDir(), "long.jsonl")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ReadLastTurn(path, int64(len(data)))
	if err != nil || !got.EndedOnError() {
		t.Fatalf("across chunks: %+v %v", got, err)
	}
	if _, err := ReadLastTurn(path, int64(len(filler))); err != ErrNoTurnBoundary {
		t.Fatalf("a limit inside the filler must leave the outcome unknown, got %v", err)
	}
}

func TestEveryObservedTurnErrorHasItsClass(t *testing.T) {
	table, err := DefaultTurnErrorTable()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct {
		class, rule, action string
	}{
		"unexpected status 503":                   {"transient", "upstream-5xx", ActionResume},
		"Selected model is at capacity":           {"capacity", "server-overloaded", ActionResume},
		"The 'luna' model is not supported":       {"configuration", "model-not-found", ActionEscalate},
		"flagged for possible cybersecurity risk": {"policy", "cyber-policy", ActionEscalate},
		"reason: max_output_tokens":               {"output_limit", "max-output-tokens", ActionResume},
		"stream disconnected before completion":   {"transient", "stream-disconnected", ActionResume},
	}
	file, err := os.Open(filepath.Join("testdata", "turn-errors.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	seen := 0
	for scanner.Scan() {
		outcome, ok, err := boundary(scanner.Bytes())
		if !ok || err != nil || !outcome.EndedOnError() {
			t.Fatalf("not a turn error: %s (%v)", scanner.Text(), err)
		}
		for fragment, w := range want {
			if !strings.Contains(outcome.Error.Message, fragment) {
				continue
			}
			seen++
			got := table.Classify(*outcome.Error)
			if got.Class != w.class || got.Rule != w.rule || got.Recovery.Action != w.action {
				t.Errorf("%q: got %s/%s/%s, want %+v", fragment, got.Class, got.Rule, got.Recovery.Action, w)
			}
		}
	}
	if seen != len(want) {
		t.Fatalf("classified %d of %d observed shapes", seen, len(want))
	}
}

func TestAnUnrecognisedErrorEscalates(t *testing.T) {
	table, err := DefaultTurnErrorTable()
	if err != nil {
		t.Fatal(err)
	}
	got := table.Classify(TurnError{Message: "something new", Info: "brand_new_info"})
	if got.Class != "unrecognised" || got.Rule != "" || got.Plan(time.Now(), time.Now(), 0).Action != ActionEscalate {
		t.Fatalf("got %+v", got)
	}
	// An HTTP status outside every rule's range is not a transient error.
	if got := table.Classify(TurnError{Info: "http_connection_failed", HTTPStatus: 401}); got.Class != "unrecognised" {
		t.Fatalf("401 classified as %s", got.Class)
	}
}

func TestPlanBacksOffCapsAndRefusesStaleErrors(t *testing.T) {
	table, _ := DefaultTurnErrorTable()
	transient := table.Classify(TurnError{Message: "stream disconnected before completion: Transport error", Info: "other"})
	at := time.Date(2026, 11, 20, 20, 13, 9, 0, time.UTC)
	steps := []struct {
		attempts int
		now      time.Time
		action   string
		due      time.Duration
	}{
		{0, at.Add(time.Minute), ActionResume, 2 * time.Minute},
		{1, at.Add(time.Minute), ActionResume, 5 * time.Minute},
		{2, at.Add(time.Minute), ActionResume, 10 * time.Minute},
		{3, at.Add(time.Minute), ActionEscalate, 0},
		// The real dead commissions: nothing acted for hours.
		{0, at.Add(20 * time.Hour), ActionEscalate, 0},
	}
	for _, s := range steps {
		plan := transient.Plan(at, s.now, s.attempts)
		if plan.Action != s.action {
			t.Fatalf("attempts %d at %s: %+v", s.attempts, s.now, plan)
		}
		if s.action == ActionResume && (!plan.Due.Equal(at.Add(s.due)) || plan.Attempt != s.attempts+1) {
			t.Fatalf("attempts %d: due %s attempt %d", s.attempts, plan.Due, plan.Attempt)
		}
		if s.action == ActionEscalate && plan.Escalation == "" {
			t.Fatalf("an escalation must say why: %+v", plan)
		}
	}
}

func TestATableThatWouldRetryBlindIsRefused(t *testing.T) {
	data, _ := os.ReadFile("turn_errors.json")
	for name, edit := range map[string][2]string{
		"fallback resumes":    {`"meaning": "No rule matches this error yet; someone reads it, decides, and adds a rule here.",` + "\n      \"recovery\": {\"action\": \"escalate\"}", `"meaning": "x", "recovery": {"action": "resume", "max_attempts": 1, "backoff": ["1m"], "window": "1h", "stale_after": "1h"}`},
		"rule names no class": {`"class": "policy",`, `"class": "nonesuch",`},
		"resume without cap":  {`"max_attempts": 2,`, `"max_attempts": 0,`},
	} {
		changed := bytes.Replace(data, []byte(edit[0]), []byte(edit[1]), 1)
		if bytes.Equal(changed, data) {
			t.Fatalf("%s: edit did not apply", name)
		}
		if _, err := LoadTurnErrorTable(changed); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func firstLine(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	line, _, _ := bytes.Cut(data, []byte("\n"))
	return line
}
