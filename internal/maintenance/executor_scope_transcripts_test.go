package maintenance

import (
	"encoding/json"
	"strings"
	"testing"
)

// A present executor reports the requested transcripts that it or a descendant
// holds open, and nothing else: unrequested open files never leave the host
// read, an absent executor makes no claim, and the verdicts are unchanged.
func TestExecutorScopeReportsRequestedTranscriptsHeldByPresentExecutors(t *testing.T) {
	const (
		mainThread = "/home/op/.codex/sessions/2026/11/20/rollout-a.jsonl"
		subThread  = "/home/op/.codex/sessions/2026/11/20/rollout-b.jsonl"
		closed     = "/home/op/.codex/sessions/2026/11/20/rollout-closed.jsonl"
		absentRun  = "/home/op/.codex/sessions/2026/11/20/rollout-absent.jsonl"
	)
	scopeFixture(t,
		map[int]processTableEntry{
			10: {PID: 10, PGID: 10},           // launcher wrapper of run-1
			11: {PID: 11, PPID: 10, PGID: 10}, // codex child holds both threads
			30: {PID: 30, PPID: 1, PGID: 30},  // unrelated process holding absentRun
		},
		map[int]scopeIdentity{10: {RunID: "run-1"}})
	old := scopeOpenFiles
	t.Cleanup(func() { scopeOpenFiles = old })
	scopeOpenFiles = func(pid int) ([]string, error) {
		switch pid {
		case 10:
			return []string{"/dev/pts/3"}, nil
		case 11:
			return []string{mainThread, "/home/op/private-notes.txt", subThread, mainThread}, nil
		case 30:
			return []string{absentRun}, nil
		}
		return nil, nil
	}
	report, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{
		Executors: []ExecutorScopeRef{
			{RunID: "run-1", Tag: "current", LegacyTag: "legacy", PID: 10, PGID: 10},
			{RunID: "run-2", Tag: "current-2", LegacyTag: "legacy-2"},
		},
		Transcripts: []string{subThread, closed, mainThread + "/../rollout-a.jsonl", absentRun},
	})
	if err != nil || !report.Complete || len(report.Executors) != 2 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	present, absent := report.Executors[0], report.Executors[1]
	if present.State != "present" || strings.Join(present.OpenTranscripts, ",") != mainThread+","+subThread {
		t.Fatalf("present executor open transcripts = %+v", present)
	}
	if absent.State != "absent" || len(absent.OpenTranscripts) != 0 {
		t.Fatalf("absent executor must make no transcript claim: %+v", absent)
	}
	data, _ := json.Marshal(report)
	if strings.Contains(string(data), "private-notes") || strings.Contains(string(data), "/dev/pts") {
		t.Fatalf("report exposed an unrequested open file: %s", data)
	}
}

func TestExecutorScopeRejectsUnboundedTranscriptReferences(t *testing.T) {
	scopeFixture(t, map[int]processTableEntry{}, nil)
	for _, transcripts := range [][]string{{"relative/rollout.jsonl"}, {"/ok\x00bad"}, make([]string, ExecutorScopeMaxTranscripts+1)} {
		_, err := (&Controller{}).ExecutorScope(t.Context(), ExecutorScopeRequest{
			Executors:   []ExecutorScopeRef{{RunID: "run-1", Tag: "current", LegacyTag: "legacy"}},
			Transcripts: transcripts,
		})
		if err == nil {
			t.Fatalf("transcripts %q accepted", transcripts[:min(len(transcripts), 2)])
		}
	}
}
