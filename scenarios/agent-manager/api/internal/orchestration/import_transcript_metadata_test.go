package orchestration

import (
	"os"
	"path/filepath"
	"testing"
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
	goalID, status := transcriptGoalMetadata(file)
	if goalID == "" || status != "met" {
		t.Fatalf("goal metadata=(%q,%q), want stable id and met", goalID, status)
	}
}
