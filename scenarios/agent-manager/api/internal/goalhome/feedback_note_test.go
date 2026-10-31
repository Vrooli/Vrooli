package goalhome

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestAddSupervisorFeedbackInsertsOpenEntryBeforeResolvedAndRetriesIdempotently(t *testing.T) {
	original := []byte("# Feedback\n\n## Open items\n\n## Resolved\n\nOlder note.\n")
	at := time.Date(2026, 10, 19, 12, 30, 0, 0, time.UTC)
	updated, changed, err := AddSupervisorFeedback(original, "FB-101", "Use the admitted route", "Keep the exact run association.\nRetry the same entry ID.", at)
	if err != nil || !changed {
		t.Fatalf("insert failed: changed=%t err=%v", changed, err)
	}
	parsed := ParseFeedback(updated)
	if len(parsed.Entries) != 1 || parsed.Entries[0].ID != "FB-101" || !parsed.Entries[0].Open() || !parsed.Entries[0].Steering() {
		t.Fatalf("entry is not visible as open supervisor feedback: %+v", parsed.Entries)
	}
	if strings.Index(string(updated), "### FB-101") > strings.Index(string(updated), "## Resolved") {
		t.Fatal("entry was inserted after the resolved boundary")
	}
	retried, changed, err := AddSupervisorFeedback(updated, "FB-101", "Use the admitted route", "Keep the exact run association.\nRetry the same entry ID.", at.Add(time.Hour))
	if err != nil || changed || string(retried) != string(updated) {
		t.Fatalf("identical retry changed the file: changed=%t err=%v", changed, err)
	}
}

func TestAddSupervisorFeedbackRefusesChangedOrResolvedID(t *testing.T) {
	at := time.Date(2026, 10, 19, 12, 30, 0, 0, time.UTC)
	first, _, err := AddSupervisorFeedback([]byte("# Feedback\n"), "FB-102", "Original", "First content.", at)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := AddSupervisorFeedback(first, "FB-102", "Changed", "First content.", at); !errors.Is(err, ErrFeedbackNoteConflict) {
		t.Fatalf("changed title reused ID: %v", err)
	}
	if _, _, err := AddSupervisorFeedback(first, "FB-102", "Original", "Changed content.", at); !errors.Is(err, ErrFeedbackNoteConflict) {
		t.Fatalf("changed body reused ID: %v", err)
	}
	resolved := append([]byte("# Feedback\n\n## Resolved\n\n"), first[len("# Feedback\n"):]...)
	if _, _, err := AddSupervisorFeedback(resolved, "FB-102", "Original", "First content.", at); !errors.Is(err, ErrFeedbackNoteConflict) {
		t.Fatalf("resolved ID was reopened: %v", err)
	}
}

func TestAddSupervisorFeedbackRefusesDuplicateMatchingIDs(t *testing.T) {
	at := time.Date(2026, 10, 19, 12, 30, 0, 0, time.UTC)
	first, _, err := AddSupervisorFeedback([]byte("# Feedback\n"), "FB-106", "Original", "First content.", at)
	if err != nil {
		t.Fatal(err)
	}
	duplicate := append(append([]byte(nil), first...), []byte("\n### FB-106 — Conflicting duplicate (2026-10-19T12:31:00Z, supervisor)\n\n> Other content.\n\nRecorded by `agent-manager effort feedback add`.\n\nStatus: open\n")...)
	if _, _, err := AddSupervisorFeedback(duplicate, "FB-106", "Original", "First content.", at); !errors.Is(err, ErrFeedbackNoteConflict) {
		t.Fatalf("duplicate matching ID was treated as a successful retry: %v", err)
	}
}

func TestAddSupervisorFeedbackRefusesMalformedOrUnsafeInput(t *testing.T) {
	at := time.Date(2026, 10, 19, 12, 30, 0, 0, time.UTC)
	if _, _, err := AddSupervisorFeedback([]byte("# Feedback\n```md\n"), "FB-103", "Title", "Body", at); !errors.Is(err, ErrFeedbackMalformed) {
		t.Fatalf("unclosed fence accepted: %v", err)
	}
	for _, tc := range []struct{ id, title, detail string }{
		{"bad id\n## Resolved", "Title", "Body"},
		{"FB-104", "Title\n## Resolved", "Body"},
		{"FB-105", "Title", ""},
	} {
		if _, _, err := AddSupervisorFeedback([]byte("# Feedback\n"), tc.id, tc.title, tc.detail, at); !errors.Is(err, ErrFeedbackNoteInvalid) {
			t.Fatalf("unsafe input accepted (%q): %v", tc.id, err)
		}
	}
}
