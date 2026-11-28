package repocontracttest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// PlatformSkipRecord is the machine-readable evidence emitted before a test
// declares a platform limit. The record deliberately carries the test name,
// host identity, and reason: a bare skipped count cannot distinguish a known
// platform boundary from an accidental early exit.
type PlatformSkipRecord struct {
	Kind       string `json:"kind"`
	Platform   string `json:"platform"`
	Arch       string `json:"arch"`
	Test       string `json:"test"`
	Reason     string `json:"reason"`
	RecordedAt string `json:"recorded_at"`
}

// RecordPlatformSkip appends one JSONL record when VROOLI_SKIP_RECORD_PATH is
// set. Test runners set that variable for a run; leaving it unset keeps local
// unit tests side-effect free while preserving the same structured path when a
// governed run is active.
func RecordPlatformSkip(t *testing.T, reason string) error {
	t.Helper()
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "platform-specific test boundary"
	}
	path := strings.TrimSpace(os.Getenv("VROOLI_SKIP_RECORD_PATH"))
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create skip record directory: %w", err)
	}
	record := PlatformSkipRecord{
		Kind:       "platform_skip",
		Platform:   normalizedPlatform(runtime.GOOS),
		Arch:       runtime.GOARCH,
		Test:       t.Name(),
		Reason:     reason,
		RecordedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal platform skip record: %w", err)
	}
	data = append(data, '\n')
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open platform skip record: %w", err)
	}
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write platform skip record: %w", err)
	}
	return nil
}

// SkipPlatform records a platform limit and then skips the test. Call this
// from the platform guard itself so every platform-gated skip has the same
// evidence contract.
func SkipPlatform(t *testing.T, reason string) {
	t.Helper()
	if err := RecordPlatformSkip(t, reason); err != nil {
		t.Fatalf("record platform skip: %v", err)
	}
	t.Skip(reason)
}

// SkipPlatformf is the formatted form of SkipPlatform for guards whose reason
// includes detected platform details.
func SkipPlatformf(t *testing.T, format string, args ...any) {
	t.Helper()
	reason := fmt.Sprintf(format, args...)
	if err := RecordPlatformSkip(t, reason); err != nil {
		t.Fatalf("record platform skip: %v", err)
	}
	t.Skipf(format, args...)
}

func normalizedPlatform(goos string) string {
	switch strings.ToLower(strings.TrimSpace(goos)) {
	case "darwin":
		return "macos"
	default:
		return goos
	}
}
