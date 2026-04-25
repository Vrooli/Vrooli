package teams

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestCmdDecisionDeferRequiresRevisitAfter(t *testing.T) {
	fc := &fakeContext{t: t}
	err := cmdDecisionDefer(fc, []string{"team-a", "dec-1"})
	if err == nil {
		t.Fatal("expected error for missing --revisit-after, got nil")
	}
	if !strings.Contains(err.Error(), "--revisit-after is required") {
		t.Errorf("error = %v, want '--revisit-after is required'", err)
	}
}

func TestCmdDecisionDeferRejectsMalformedDate(t *testing.T) {
	fc := &fakeContext{t: t}
	err := cmdDecisionDefer(fc, []string{"team-a", "dec-1", "--revisit-after=not-a-date"})
	if err == nil {
		t.Fatal("expected error for malformed date")
	}
	if !strings.Contains(err.Error(), "YYYY-MM-DD") {
		t.Errorf("error = %v, want format hint", err)
	}
}

func TestCmdDecisionDeferRejectsPastDate(t *testing.T) {
	fc := &fakeContext{t: t}
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	err := cmdDecisionDefer(fc, []string{"team-a", "dec-1", "--revisit-after=" + yesterday})
	if err == nil {
		t.Fatal("expected error for past date")
	}
	if !strings.Contains(err.Error(), "today or in the future") {
		t.Errorf("error = %v, want 'today or in the future'", err)
	}
}

func TestCmdDecisionDeferHappyPath(t *testing.T) {
	revisit := time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02")
	resp := DecisionEntry{ID: "dec-1", Status: "deferred"}
	resp.RevisitAfter = &revisit
	fc := &fakeContext{t: t, response: resp}

	err := cmdDecisionDefer(fc, []string{"team-a", "dec-1", "--revisit-after=" + revisit, "--notes=soak"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fc.assertMethodPath(t, "PUT", "/teams/team-a/decisions/dec-1")

	var sent map[string]interface{}
	if err := json.Unmarshal(fc.gotPayload, &sent); err != nil {
		t.Fatalf("payload unmarshal: %v", err)
	}
	if sent["status"] != "deferred" {
		t.Errorf("status = %v, want deferred", sent["status"])
	}
	if sent["revisit_after"] != revisit {
		t.Errorf("revisit_after = %v, want %s", sent["revisit_after"], revisit)
	}
	if sent["notes"] != "soak" {
		t.Errorf("notes = %v, want 'soak'", sent["notes"])
	}
}
