package repocontracttest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRecordPlatformSkipWritesMachineReadableEvidence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "skips.jsonl")
	t.Setenv("VROOLI_SKIP_RECORD_PATH", path)
	if err := RecordPlatformSkip(t, "needs a POSIX process fixture"); err != nil {
		t.Fatalf("RecordPlatformSkip: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	var record PlatformSkipRecord
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("decode record: %v", err)
	}
	if record.Kind != "platform_skip" || record.Test == "" || record.Reason == "" || record.Platform == "" {
		t.Fatalf("incomplete record: %#v", record)
	}
}

func TestRecordPlatformSkipIsNoOpWithoutRunPath(t *testing.T) {
	t.Setenv("VROOLI_SKIP_RECORD_PATH", "")
	if err := RecordPlatformSkip(t, "local test"); err != nil {
		t.Fatalf("RecordPlatformSkip without path: %v", err)
	}
}
