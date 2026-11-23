package orchestration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner/codecs"
)

func TestTranscriptGoalMetadataFindsNestedArrayAttachment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transcript.jsonl")
	body := `{"type":"turn.completed","attachments":[{"type":"goal_status","condition":"tests pass","met":true}]}` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	// The import path routes goal markers through the codec seam; the Claude
	// codec walks arbitrary object/array shapes for a nested goal_status.
	goalID, status := transcriptGoalMetadata(file, codecs.NewClaudeForTest())
	if goalID == "" || status != "complete" {
		t.Fatalf("goal metadata=(%q,%q), want stable id and complete", goalID, status)
	}
}

func TestTranscriptGoalMetadataReconstructsCohortFromUserPreamble(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transcript.jsonl")
	body := `{"role":"user","content":"Please investigate the agent run evidence and explain which commands caused the repeated failures, then validate the proposed repair."}` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	goalID, status := transcriptGoalMetadata(file)
	if goalID == "" || status != "unmet" {
		t.Fatalf("goal metadata=(%q,%q), want reconstructed cohort and unmet status", goalID, status)
	}
}

func TestTranscriptGoalMarkerOverridesEarlierPreamble(t *testing.T) {
	preamble := `{"role":"user","content":"Investigate the archived run evidence and explain the repeated failure, then verify the repair."}`
	marker := `{"type":"turn.completed","attachments":[{"type":"goal_status","condition":"tests pass","met":true}]}`
	markerReader := strings.NewReader(preamble + "\n" + marker)
	goalID, status := transcriptGoalMetadata(markerReader, codecs.NewClaudeForTest())
	preambleID, preambleStatus := transcriptGoalMetadata(strings.NewReader(preamble))
	if goalID == "" || goalID == preambleID || status != "complete" || preambleStatus != "unmet" {
		t.Fatalf("marker result=(%q,%q), preamble result=(%q,%q)", goalID, status, preambleID, preambleStatus)
	}
}

func TestReadTranscriptHeadExtractsLabelAndGoalInOnePass(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transcript.jsonl")
	body := strings.Join([]string{
		`{"type":"turn.completed","attachments":[{"type":"goal_status","condition":"reader passes","met":true}]}`,
		`{"type":"ai-title","aiTitle":"Saved task title"}`,
		`{"role":"user","content":"# AGENTS.md instructions for /home/operator/Vrooli and other injected context"}`,
		`{"role":"user","content":"Repair the transcript reader and verify its actual resume behavior"}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	head := readTranscriptHead(path, codecs.NewClaudeForTest())
	if head.label.HarnessTitle != "Saved task title" || head.label.UserPrompt != "Repair the transcript reader and verify its actual resume behavior" {
		t.Fatalf("label evidence = %+v", head.label)
	}
	wantGoalID, wantStatus := transcriptGoalMetadata(strings.NewReader(body), codecs.NewClaudeForTest())
	if head.goalID != wantGoalID || wantStatus != "complete" {
		t.Fatalf("combined goal ID=%q, standalone=%q status=%q", head.goalID, wantGoalID, wantStatus)
	}
}
