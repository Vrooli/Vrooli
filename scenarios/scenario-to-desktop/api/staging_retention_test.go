package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/retention"
	"scenario-to-desktop-api/build"
	"scenario-to-desktop-api/livedesktop"
	"scenario-to-desktop-api/pipeline"
	"scenario-to-desktop-api/smoketest"
)

func writeStagingEntry(t *testing.T, root, app, id string, age time.Duration) string {
	t.Helper()
	path := filepath.Join(root, app, id)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "bundle"), make([]byte, 10), 0o600); err != nil {
		t.Fatal(err)
	}
	at := time.Now().Add(-age)
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
	return path
}

// stagingOwnerFor wires retention exactly as main does, with the smoke-test
// store loaded from smokePath as a restarted process would load it.
func stagingOwnerFor(t *testing.T, root, smokePath string) (pipeline.StagingRetention, *smoketest.FileStore) {
	t.Helper()
	smokeStore, err := smoketest.NewStore(smokePath)
	if err != nil {
		t.Fatal(err)
	}
	builds := build.NewStore()
	owner := pipeline.StagingRetention{
		Root:       root,
		Status:     func(string) (*pipeline.Status, bool) { return &pipeline.Status{Status: pipeline.StatusCompleted}, true },
		InUse:      stagingInUse(smokeStore, &generationBuildStoreAdapter{store: builds}, builds, func() []*livedesktop.Session { return nil }),
		KeepLatest: 1,
	}
	return owner, smokeStore
}

// [REQ:STD-INTEGRATION-STAGING-RETENTION]
func TestStagingRetentionIgnoresSmokeTestsFromDeadProcess(t *testing.T) {
	root := t.TempDir()
	smokePath := filepath.Join(t.TempDir(), "smoke_tests_v2.json")
	zombie := []*smoketest.Status{{SmokeTestID: "zombie", ScenarioName: "web-console", Status: "running", StartedAt: time.Now().Add(-30 * 24 * time.Hour)}}
	data, err := json.Marshal(zombie)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(smokePath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	oldest := writeStagingEntry(t, root, "web-console", "old", 10*24*time.Hour)
	older := writeStagingEntry(t, root, "web-console", "older", 9*24*time.Hour)
	newest := writeStagingEntry(t, root, "web-console", "new", time.Minute)

	owner, _ := stagingOwnerFor(t, root, smokePath)
	result, err := owner.Prune(context.Background(), retention.Budget{Name: "staging", MaxBytes: 10, MaxAge: 7 * 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if result.Deleted != 2 || result.ProtectedTotal() != 0 {
		t.Fatalf("a dead smoke-test record blocked retention: %+v", result)
	}
	for _, gone := range []string{oldest, older} {
		if _, err := os.Stat(gone); !os.IsNotExist(err) {
			t.Fatalf("%s survived: %v", gone, err)
		}
	}
	if _, err := os.Stat(newest); err != nil {
		t.Fatalf("newest entry removed: %v", err)
	}
}

// [REQ:STD-INTEGRATION-STAGING-RETENTION]
func TestStagingRetentionProtectsLiveSmokeTestAndReportsWhy(t *testing.T) {
	root := t.TempDir()
	smokePath := filepath.Join(t.TempDir(), "smoke_tests_v2.json")
	old := writeStagingEntry(t, root, "web-console", "old", 10*24*time.Hour)
	writeStagingEntry(t, root, "web-console", "new", time.Minute)

	owner, smokeStore := stagingOwnerFor(t, root, smokePath)
	smokeStore.Save(&smoketest.Status{SmokeTestID: "live", ScenarioName: "web-console", Status: "running", StartedAt: time.Now()})
	budget := retention.Budget{Name: "staging", MaxBytes: 10, MaxAge: 7 * 24 * time.Hour}
	result, err := owner.Prune(context.Background(), budget)
	if err != nil {
		t.Fatal(err)
	}
	if result.Deleted != 0 || result.Protected["smoke_test"] == 0 {
		t.Fatalf("live smoke test did not protect staging: %+v", result)
	}
	if _, err := os.Stat(old); err != nil {
		t.Fatalf("staging used by a live smoke test was removed: %v", err)
	}

	receiptErr := stagingReceiptError(result, budget, nil)
	if receiptErr == nil || !strings.Contains(receiptErr.Error(), "smoke_test=") {
		t.Fatalf("over-budget cycle must record why it could not reclaim, got %v", receiptErr)
	}
}

func TestStagingReceiptErrorOnlyWhenBudgetUnmet(t *testing.T) {
	budget := retention.Budget{Name: "staging", MaxBytes: 100}
	within := pipeline.StagingResult{Result: retention.Result{After: retention.Usage{Bytes: 100}}, Protected: map[string]int{"smoke_test": 2}}
	if err := stagingReceiptError(within, budget, nil); err != nil {
		t.Fatalf("within budget recorded as failure: %v", err)
	}
	cycleErr := errors.New("scan failed")
	if err := stagingReceiptError(within, budget, cycleErr); !errors.Is(err, cycleErr) {
		t.Fatalf("cycle error not preserved: %v", err)
	}
	over := pipeline.StagingResult{Result: retention.Result{After: retention.Usage{Bytes: 101}}}
	if err := stagingReceiptError(over, budget, nil); err == nil || !strings.Contains(err.Error(), "protected (none)") {
		t.Fatalf("over budget not recorded: %v", err)
	}
}
