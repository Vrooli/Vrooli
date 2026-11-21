package isolation

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func privateSQLitePlan(t *testing.T) Allocation {
	t.Helper()
	parent := filepath.Join(t.TempDir(), "backing")
	if err := os.Mkdir(parent, 0700); err != nil {
		t.Fatal(err)
	}
	if err := ensureSQLiteStagingNamespace(parent); err != nil {
		t.Fatal(err)
	}
	runID := "31c832dd-4761-4f80-aa9c-d43da34173df"
	return Allocation{Kind: "sqlite", RunID: runID, Root: filepath.Join(parent, "test-genie-playbooks-"+runID), StagingRoot: filepath.Join(parent, ".staging", runID)}
}

func TestSQLitePreparationCrashHelper(t *testing.T) {
	if os.Getenv("TG_SQLITE_PREP_HELPER") != "1" {
		return
	}
	raw, err := os.ReadFile(os.Getenv("TG_SQLITE_PLAN"))
	if err != nil {
		t.Fatal(err)
	}
	var plan Allocation
	if err := json.Unmarshal(raw, &plan); err != nil {
		t.Fatal(err)
	}
	err = prepareSQLiteDirectory(plan, func(step string) error {
		if step == os.Getenv("TG_SQLITE_CRASH_CUT") {
			if err := os.WriteFile(os.Getenv("TG_SQLITE_WITNESS"), []byte(step), 0600); err != nil {
				return err
			}
			select {}
		}
		return nil
	})
	t.Fatal("preparation did not reach requested crash cut", err)
}

func TestSQLiteReplacementRetiresActualKilledPreparationAtEveryPublicationCut(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("no-replace backing publication is currently supported on Linux")
	}
	for _, cut := range []string{"staging-created", "marker-synced", "published"} {
		t.Run(cut, func(t *testing.T) {
			plan := privateSQLitePlan(t)
			planPath := filepath.Join(t.TempDir(), "allocation.json")
			raw, _ := json.Marshal(plan)
			if err := os.WriteFile(planPath, raw, 0600); err != nil {
				t.Fatal(err)
			}
			witness := filepath.Join(t.TempDir(), "crash-cut")
			child := exec.Command(os.Args[0], "-test.run=^TestSQLitePreparationCrashHelper$", "-test.timeout=30s")
			child.Env = append(os.Environ(), "TG_SQLITE_PREP_HELPER=1", "TG_SQLITE_PLAN="+planPath, "TG_SQLITE_WITNESS="+witness, "TG_SQLITE_CRASH_CUT="+cut)
			if err := child.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = child.Process.Kill(); _ = child.Wait() }()
			deadline := time.Now().Add(5 * time.Second)
			for {
				if _, err := os.Stat(witness); err == nil {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("allocator did not reach exact crash cut")
				}
				time.Sleep(5 * time.Millisecond)
			}
			if err := child.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			if err := child.Wait(); err == nil {
				t.Fatal("preparing owner exited successfully instead of being killed")
			}
			// Replacement uses only the plan committed before the child started.
			if err := RetireAllocation(t.Context(), plan); err != nil {
				t.Fatal("replacement retained interrupted private preparation", err)
			}
			for _, path := range []string{plan.Root, plan.StagingRoot} {
				if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("replacement left private backing or staging", err)
				}
			}
			if err := RetireAllocation(t.Context(), plan); err != nil {
				t.Fatal("replacement retirement was not idempotent", err)
			}
		})
	}
}

func TestSQLitePublicationNeverReplacesForeignFinalDirectory(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("requires Linux no-replace publication")
	}
	plan := privateSQLitePlan(t)
	err := prepareSQLiteDirectory(plan, func(step string) error {
		if step == "marker-synced" {
			return os.Mkdir(plan.Root, 0700)
		}
		return nil
	})
	if err == nil {
		t.Fatal("publication replaced a foreign final directory")
	}
	if err := RetireAllocation(t.Context(), plan); err == nil {
		t.Fatal("replacement deleted an unmarked foreign final directory")
	}
	if _, err := os.Stat(plan.Root); err != nil {
		t.Fatal("foreign final directory disappeared", err)
	}
	if _, err := os.Stat(plan.StagingRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("known preparation was not retired independently of foreign final root", err)
	}
}

func TestSQLiteMarkerWriteFailureRetiresOnlyOwnedPreparation(t *testing.T) {
	plan := privateSQLitePlan(t)
	err := prepareSQLiteDirectory(plan, func(step string) error {
		if step == "staging-created" {
			return os.Chmod(plan.StagingRoot, 0500)
		}
		return nil
	})
	if err == nil {
		t.Fatal("fixture did not cause a real owner-marker write failure")
	}
	if err := RetireAllocation(t.Context(), plan); err != nil {
		t.Fatal("marker write failure stranded exact private preparation", err)
	}
}

func TestSQLiteStagingRetirementRefusesForeignContentAndAliases(t *testing.T) {
	for _, kind := range []string{"content", "marker", "alias"} {
		t.Run(kind, func(t *testing.T) {
			plan := privateSQLitePlan(t)
			if err := os.Mkdir(plan.StagingRoot, 0700); err != nil {
				t.Fatal(err)
			}
			var err error
			switch kind {
			case "content":
				err = os.WriteFile(filepath.Join(plan.StagingRoot, "keep"), []byte("foreign"), 0600)
			case "marker":
				err = os.WriteFile(filepath.Join(plan.StagingRoot, ".seed-owner"), []byte("different-owner"), 0600)
			case "alias":
				err = os.Symlink("/dev/null", filepath.Join(plan.StagingRoot, ".seed-owner"))
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := RetireAllocation(t.Context(), plan); err == nil {
				t.Fatal("foreign staging content was deleted")
			}
			if entries, err := os.ReadDir(plan.StagingRoot); err != nil || len(entries) != 1 {
				t.Fatal("foreign content changed", err)
			}
		})
	}
}
