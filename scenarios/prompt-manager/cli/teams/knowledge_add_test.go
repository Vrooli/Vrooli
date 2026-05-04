package teams

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCmdKnowledgeAddSendsCallerNote(t *testing.T) {
	fc := &fakeContext{t: t, response: KnowledgeEntry{ID: "knw-x", Topic: "audience-scan/2026-05-04/test"}}
	err := cmdKnowledgeAdd(fc, []string{
		"marketing-crew",
		"--topic=audience-scan/2026-05-04/test",
		"--content=hello world",
		"--caller-note=hand-curated",
	})
	if err != nil {
		t.Fatalf("cmdKnowledgeAdd error: %v", err)
	}
	fc.assertMethodPath(t, "POST", "/teams/marketing-crew/knowledge")

	var sent AddKnowledgeRequest
	if err := json.Unmarshal(fc.gotPayload, &sent); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if sent.Topic != "audience-scan/2026-05-04/test" {
		t.Errorf("Topic = %q", sent.Topic)
	}
	if sent.Content != "hello world" {
		t.Errorf("Content = %q", sent.Content)
	}
	if sent.CallerNote != "hand-curated" {
		t.Errorf("CallerNote = %q, want 'hand-curated'", sent.CallerNote)
	}
}

func TestCmdKnowledgeAddOmitsCallerNoteWhenAbsent(t *testing.T) {
	fc := &fakeContext{t: t, response: KnowledgeEntry{ID: "knw-x"}}
	err := cmdKnowledgeAdd(fc, []string{
		"marketing-crew",
		"--topic=t",
		"--content=c",
	})
	if err != nil {
		t.Fatalf("cmdKnowledgeAdd error: %v", err)
	}
	// caller_note has omitempty; must not appear in the JSON when absent.
	if strings.Contains(string(fc.gotPayload), "caller_note") {
		t.Errorf("caller_note must be omitted when absent, got payload: %s", fc.gotPayload)
	}
	// `by` must not appear in the wire shape — the field is gone from
	// AddKnowledgeRequest entirely (P3.3).
	if strings.Contains(string(fc.gotPayload), `"by"`) {
		t.Errorf("payload must not include legacy 'by' field, got: %s", fc.gotPayload)
	}
}

func TestCmdKnowledgeAddRejectsLegacyByFlag(t *testing.T) {
	fc := &fakeContext{t: t}
	err := cmdKnowledgeAdd(fc, []string{
		"marketing-crew",
		"--by=researcher",
		"--topic=t",
		"--content=c",
	})
	if err == nil {
		t.Fatal("expected error rejecting --by, got nil")
	}
	if !strings.Contains(err.Error(), "--by is removed") {
		t.Errorf("err = %q, want migration message", err)
	}
	if !strings.Contains(err.Error(), "RUNTIME_ATTRIBUTION") {
		t.Errorf("err must point to canon doc, got %q", err)
	}
	if fc.gotMethod != "" {
		t.Errorf("expected no API call when --by is rejected, got %s %s", fc.gotMethod, fc.gotPath)
	}
}

func TestCmdKnowledgeAddRequiresTopic(t *testing.T) {
	fc := &fakeContext{t: t}
	err := cmdKnowledgeAdd(fc, []string{
		"marketing-crew",
		"--content=c",
	})
	if err == nil || !strings.Contains(err.Error(), "topic is required") {
		t.Errorf("expected topic-required error, got %v", err)
	}
}

func TestCmdKnowledgeAddRequiresContent(t *testing.T) {
	fc := &fakeContext{t: t}
	err := cmdKnowledgeAdd(fc, []string{
		"marketing-crew",
		"--topic=t",
	})
	if err == nil || !strings.Contains(err.Error(), "content is required") {
		t.Errorf("expected content-required error, got %v", err)
	}
}

func TestCmdKnowledgeAddRequiresTeamID(t *testing.T) {
	fc := &fakeContext{t: t}
	err := cmdKnowledgeAdd(fc, []string{
		"--topic=t",
		"--content=c",
	})
	if err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Errorf("expected usage error, got %v", err)
	}
}
