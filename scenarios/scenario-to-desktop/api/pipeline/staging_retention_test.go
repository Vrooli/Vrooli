package pipeline

import (
	"context"
	"github.com/vrooli/api-core/retention"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// [REQ:STD-INTEGRATION-STAGING-RETENTION]
func TestStagingRetentionCapacityProtectsLiveWork(t *testing.T) {
	root := t.TempDir()
	now := time.Now()
	names := []string{"active", "smoke", "marked", "old", "new"}
	for i, name := range names {
		path := filepath.Join(root, "app", name)
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "bundle"), make([]byte, 10), 0600); err != nil {
			t.Fatal(err)
		}
		if name == "marked" {
			if err := os.WriteFile(filepath.Join(path, ".building"), nil, 0600); err != nil {
				t.Fatal(err)
			}
		}
		at := now.Add(time.Duration(i-10) * time.Minute)
		if err := os.Chtimes(path, at, at); err != nil {
			t.Fatal(err)
		}
	}
	owner := StagingRetention{Root: root, Status: func(id string) (*Status, bool) {
		if id == "active" {
			return &Status{Status: "running"}, true
		}
		return &Status{Status: StatusCompleted}, true
	}, InUse: func(app, path string) bool { return filepath.Base(path) == "smoke" }, KeepLatest: 1}
	result, err := owner.Prune(context.Background(), retention.Budget{Name: "staging", MaxBytes: 40, MaxAge: 7 * 24 * time.Hour})
	if err != nil || result.Deleted != 1 || result.After.Bytes != 40 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	for _, name := range names {
		_, err := os.Stat(filepath.Join(root, "app", name))
		if name == "old" {
			if !os.IsNotExist(err) {
				t.Fatalf("old build survived: %v", err)
			}
		} else if err != nil {
			t.Fatalf("protected/new build %s missing: %v", name, err)
		}
	}
}

// [REQ:STD-INTEGRATION-STAGING-RETENTION]
func TestStagingRetentionUnknownGraceAndUnavailableOwner(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "app", "unknown")
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "bundle"), []byte("1234"), 0600); err != nil {
		t.Fatal(err)
	}
	owner := StagingRetention{Root: root, Status: func(string) (*Status, bool) { return nil, false }}
	budget := retention.Budget{Name: "staging", MaxBytes: 1}
	result, err := owner.Prune(context.Background(), budget)
	if err != nil || result.Deleted != 0 || !result.Incomplete {
		t.Fatalf("grace result=%+v err=%v", result, err)
	}
	owner.Status = nil
	if _, err = owner.Prune(context.Background(), budget); err == nil {
		t.Fatal("missing owner must fail closed")
	}
	at := time.Now().Add(-3 * time.Hour)
	if err = os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
	owner.Status = func(string) (*Status, bool) { return nil, false }
	result, err = owner.Prune(context.Background(), budget)
	if err != nil || result.Deleted != 1 {
		t.Fatalf("legacy result=%+v err=%v", result, err)
	}
}

func TestStagingBudgetLoadsManifestAndOverrides(t *testing.T) {
	t.Setenv("DESKTOP_STAGING_RETENTION_MAX_BYTES", "")
	t.Setenv("DESKTOP_STAGING_RETENTION_MAX_AGE", "")
	path := filepath.Join("..", "..", ".vrooli", "service.json")
	budget, err := StagingBudget(path)
	if err != nil || budget.MaxBytes != 20<<30 || budget.MaxAge != 7*24*time.Hour {
		t.Fatalf("budget=%+v err=%v", budget, err)
	}
	t.Setenv("DESKTOP_STAGING_RETENTION_MAX_BYTES", "1GiB")
	budget, err = StagingBudget(path)
	if err != nil || budget.MaxBytes != 1<<30 {
		t.Fatalf("budget=%+v err=%v", budget, err)
	}
	t.Setenv("DESKTOP_STAGING_RETENTION_MAX_AGE", "0")
	if _, err = StagingBudget(path); err == nil {
		t.Fatal("invalid budget accepted")
	}
}
