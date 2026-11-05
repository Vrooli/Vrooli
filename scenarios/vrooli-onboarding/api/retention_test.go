package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeRetentionFixture(t *testing.T, path string, value any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCleanupOnboardingArtifactsUsesControlledClockAndProtectsActiveRuns(t *testing.T) {
	root := t.TempDir()
	reviews := filepath.Join(root, "apply-reviews")
	runs := filepath.Join(root, "apply-runs")
	admissions := filepath.Join(root, "apply-admissions")
	now := time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC)
	old := now.Add(-8 * 24 * time.Hour).Format(time.RFC3339)
	oldExpiry := now.Add(-48 * time.Hour).Format(time.RFC3339)
	recentExpiry := now.Add(-2 * time.Hour).Format(time.RFC3339)

	writeRetentionFixture(t, filepath.Join(runs, "old.json"), applyRun{ID: "old", Status: "applied", CompletedAt: old})
	writeRetentionFixture(t, filepath.Join(runs, "active.json"), applyRun{ID: "active", Status: "applying", StartedAt: old})
	writeRetentionFixture(t, filepath.Join(runs, "recent.json"), applyRun{ID: "recent", Status: "applied", CompletedAt: now.Add(-time.Hour).Format(time.RFC3339)})
	if err := os.WriteFile(filepath.Join(runs, "old.log"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeRetentionFixture(t, filepath.Join(reviews, "expired.json"), applyReviewRecord{ExpiresAt: oldExpiry})
	writeRetentionFixture(t, filepath.Join(reviews, "recent.json"), applyReviewRecord{ExpiresAt: recentExpiry})
	writeRetentionFixture(t, filepath.Join(reviews, "active.json"), applyReviewRecord{ExpiresAt: oldExpiry, ConsumedBy: "active"})
	if err := os.WriteFile(filepath.Join(reviews, "corrupt.json"), []byte("not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeRetentionFixture(t, filepath.Join(admissions, "old.json"), applyAdmissionRecord{RunID: "old", CreatedAt: old})
	writeRetentionFixture(t, filepath.Join(admissions, "active.json"), applyAdmissionRecord{RunID: "active", CreatedAt: old})

	if err := cleanupApplyReviews(reviews, runs, now); err != nil {
		t.Fatal(err)
	}
	if err := cleanupApplyAdmissions(admissions, runs, now); err != nil {
		t.Fatal(err)
	}
	if err := cleanupApplyRuns(runs, now); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		filepath.Join(reviews, "expired.json"),
		filepath.Join(runs, "old.json"),
		filepath.Join(runs, "old.log"),
		filepath.Join(admissions, "old.json"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("expired artifact %s still exists, err=%v", path, err)
		}
	}
	for _, path := range []string{
		filepath.Join(reviews, "recent.json"),
		filepath.Join(reviews, "active.json"),
		filepath.Join(reviews, "corrupt.json"),
		filepath.Join(runs, "active.json"),
		filepath.Join(runs, "recent.json"),
		filepath.Join(admissions, "active.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("protected artifact %s missing: %v", path, err)
		}
	}
}
