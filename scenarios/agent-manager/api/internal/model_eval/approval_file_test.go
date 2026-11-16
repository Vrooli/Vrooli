package model_eval

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func writeApprovalFixture(t *testing.T, root, inbox string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "work", AdmissionDecisionID), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "DECISIONS.md"), []byte("## Decisions\n- "+AdmissionDecisionID+" — pending\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "work", AdmissionDecisionID, "BRIEF.md"), []byte("Scope: `"+AdmissionScope+"`\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "INBOX.md"), []byte(inbox), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestTopicFileApprovalRequiresExactCurrentRegisteredAnswer(t *testing.T) {
	root := t.TempDir()
	writeApprovalFixture(t, root, "# Inbox\n### 2026-11-12T00:00:00Z · from operator · answer\nLM2-INFERENCE-ADMISSION: approve\n")
	source := TopicFileApprovalSource{TopicDir: root}
	if _, err := source.ReadDecision(context.Background(), AdmissionDecisionID); err == nil {
		t.Fatal("historical answer for another ID authorized the current batch")
	}

	writeApprovalFixture(t, root, "# Inbox\n### 2026-11-16T05:00:00Z · from operator · answer\n"+AdmissionDecisionID+": approve\n")
	approval, err := source.ReadDecision(context.Background(), AdmissionDecisionID)
	if err != nil {
		t.Fatalf("ReadDecision: %v", err)
	}
	if approval.DecisionID != AdmissionDecisionID || approval.Answer != "approve" || approval.Scope != AdmissionScope || approval.RecordedAt.IsZero() {
		t.Fatalf("unexpected approval: %+v", approval)
	}
}

func TestTopicFileApprovalPreservesDeferAndRefusesWrongScopeMaterial(t *testing.T) {
	root := t.TempDir()
	writeApprovalFixture(t, root, "# Inbox\n### 2026-11-16T05:00:00Z · from operator · answer\n"+AdmissionDecisionID+": defer\n")
	approval, err := (TopicFileApprovalSource{TopicDir: root}).ReadDecision(context.Background(), AdmissionDecisionID)
	if err != nil || approval.Answer != "defer" {
		t.Fatalf("defer must remain explicit: approval=%+v err=%v", approval, err)
	}
	if err := os.WriteFile(filepath.Join(root, "work", AdmissionDecisionID, "BRIEF.md"), []byte("Scope: changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (TopicFileApprovalSource{TopicDir: root}).ReadDecision(context.Background(), AdmissionDecisionID); err == nil {
		t.Fatal("changed scope packet authorized inference")
	}
}
