package domain

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestRunLineageHeaderNamesPlaceAndBriefWithoutTheGoal(t *testing.T) {
	run, task, parent := uuid.New(), uuid.New(), uuid.New()
	for name, tc := range map[string]struct {
		lineage RunLineage
		want    []string
		absent  []string
	}{
		"unchartered root": {lineage: RunLineage{RunID: run, TaskID: task, MaxDepth: 3}},
		"chartered root": {
			lineage: RunLineage{RunID: run, TaskID: task, MaxDepth: 3, CharterPath: "/work/CHARTER.md"},
			want:    []string{"you are run " + run.String() + ".", "Charter: /work/CHARTER.md", "every unit start and after any context compaction", "--parent-run-id " + run.String()},
			absent:  []string{"child of run"},
		},
		"child below the cap": {
			lineage: RunLineage{RunID: run, TaskID: task, ParentRunID: &parent, Depth: 1, MaxDepth: 3},
			want:    []string{"depth 1 of max 3", "a child of run " + parent.String(), "agent-manager task get " + task.String(), "native sub-agents with the same brief"},
		},
		"child at the cap": {
			lineage: RunLineage{RunID: run, TaskID: task, ParentRunID: &parent, Depth: 3, MaxDepth: 3},
			want:    []string{"depth 3 of max 3", "at the maximum depth"},
			absent:  []string{"--parent-run-id"},
		},
		"children disabled": {
			lineage: RunLineage{RunID: run, TaskID: task, MaxDepth: 0, CharterPath: "/work/CHARTER.md"},
			want:    []string{"child runs are disabled"},
			absent:  []string{"--parent-run-id"},
		},
	} {
		header := tc.lineage.Header()
		if len(tc.want) == 0 && header != "" {
			t.Errorf("%s: header = %q, want none", name, header)
		}
		for _, want := range tc.want {
			if !strings.Contains(header, want) {
				t.Errorf("%s: header lacks %q:\n%s", name, want, header)
			}
		}
		for _, absent := range tc.absent {
			if strings.Contains(header, absent) {
				t.Errorf("%s: header must not contain %q:\n%s", name, absent, header)
			}
		}
		if lines := strings.Count(header, "\n"); lines > 2 {
			t.Errorf("%s: header has %d lines; keep it to three", name, lines+1)
		}
	}
	if got := WithLineageHeader("", "do the work"); got != "do the work" {
		t.Errorf("no header must leave instructions unchanged, got %q", got)
	}
	if got := WithLineageHeader("HEADER", "do the work"); got != "HEADER\n\ndo the work" {
		t.Errorf("header must precede instructions, got %q", got)
	}
}

func TestSettledAndLiveStatusesPartitionWorkerActivity(t *testing.T) {
	for status, want := range map[RunStatus][2]bool{
		RunStatusPending: {false, true}, RunStatusStarting: {false, true}, RunStatusRunning: {false, true}, RunStatusParked: {false, true},
		RunStatusNeedsReview: {true, false}, RunStatusComplete: {true, false}, RunStatusFailed: {true, false}, RunStatusCancelled: {true, false}, RunStatusUnknown: {true, false},
	} {
		if status.IsSettled() != want[0] || status.IsLive() != want[1] {
			t.Errorf("%s: settled=%t live=%t, want %v", status, status.IsSettled(), status.IsLive(), want)
		}
	}
}
