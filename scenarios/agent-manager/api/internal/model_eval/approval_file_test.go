package model_eval

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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
	packet := "This packet authorizes at most 20 fixed, source-verified user messages from Agent Manager,\n" +
		"replayed sequentially through AI Gateway. It does not authorize changing the model default,\n" +
		"downloading candidates, promoting or deleting weights, remote inference, paid services, or\n" +
		"displacing another owner's resource claims. Use the already-installed model selected by current\n" +
		"`chat.default` policy; require current rights to evaluate as allowed. One request at a time,\n" +
		"local-only. Produce one metadata-only replay receipt. Scope: `" + AdmissionScope + "`\n"
	if err := os.WriteFile(filepath.Join(root, "work", AdmissionDecisionID, "BRIEF.md"), []byte(packet), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "INBOX.md"), []byte(inbox), 0o600); err != nil {
		t.Fatal(err)
	}
}

func fixtureApprovalSource(t *testing.T, root string) TopicFileApprovalSource {
	t.Helper()
	packet, err := os.ReadFile(filepath.Join(root, "work", AdmissionDecisionID, "BRIEF.md"))
	if err != nil {
		t.Fatal(err)
	}
	return TopicFileApprovalSource{TopicDir: root, expectedPacketSHA256: hashBytes(packet)}
}

func TestTopicFileApprovalRequiresExactCurrentRegisteredAnswer(t *testing.T) {
	root := t.TempDir()
	writeApprovalFixture(t, root, "# Inbox\n### 2026-11-12T00:00:00Z · from operator · answer\nLM2-INFERENCE-ADMISSION: approve\n")
	source := fixtureApprovalSource(t, root)
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
	approval, err := fixtureApprovalSource(t, root).ReadDecision(context.Background(), AdmissionDecisionID)
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

func TestTopicFileApprovalDoesNotReuseAnswerAcrossInboxSections(t *testing.T) {
	root := t.TempDir()
	inbox := "# Inbox\n### 2026-11-16T05:00:00Z · from operator · answer\n" + AdmissionDecisionID + ": approve\n" +
		"### 2026-11-22T05:00:00Z · from operator · answer\nLM20-OTHER-DECISION: approve\n"
	writeApprovalFixture(t, root, inbox)
	if _, err := (TopicFileApprovalSource{TopicDir: root}).ReadDecision(context.Background(), AdmissionDecisionID); err == nil {
		t.Fatal("older LM2 answer must not inherit a newer unrelated operator-answer timestamp")
	}
}

func TestTopicFileApprovalDispatchClaimIsAtomicAndOneShot(t *testing.T) {
	root := t.TempDir()
	writeApprovalFixture(t, root, "# Inbox\n### 2026-11-22T01:09:44Z · from operator · answer\n"+AdmissionDecisionID+": approve\n")
	source := fixtureApprovalSource(t, root)
	approval, err := source.ReadDecision(context.Background(), AdmissionDecisionID)
	if err != nil {
		t.Fatal(err)
	}
	caseSetHash := hashBytes([]byte("fixed cases"))
	if err := source.ClaimDispatch(context.Background(), approval, caseSetHash); err != nil {
		t.Fatalf("first one-time claim: %v", err)
	}
	marker, err := os.ReadFile(filepath.Join(root, "work", AdmissionDecisionID, "dispatch-admission.json"))
	if err != nil || !strings.Contains(string(marker), caseSetHash) || strings.Contains(string(marker), "prompt") {
		t.Fatalf("unexpected one-time marker %q err=%v", marker, err)
	}
	if _, err := source.ReadDecision(context.Background(), AdmissionDecisionID); err == nil {
		t.Fatal("consumed approval remained eligible")
	}
	if err := source.ClaimDispatch(context.Background(), approval, caseSetHash); err == nil {
		t.Fatal("second one-time claim succeeded")
	}
}
