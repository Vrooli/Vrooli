package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// muteStdout redirects os.Stdout for the duration of fn so migration logging
// doesn't pollute test output.
func muteStdout(t *testing.T, fn func()) {
	t.Helper()
	orig := os.Stdout
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = devnull
	defer func() {
		os.Stdout = orig
		_ = devnull.Close()
	}()
	fn()
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(v)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMigrateItem_NoSpecSkips(t *testing.T) {
	muteStdout(t, func() {
		did, err := migrateItem(t.TempDir(), "fix", false)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if did {
			t.Error("item without spec.json should be skipped")
		}
	})
}

func TestMigrateItem_AlreadyMigratedSkips(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "spec.json"), oldSpec{Title: "T"})
	if err := os.MkdirAll(filepath.Join(dir, "workshop"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plan.md"), []byte("p"), 0o644); err != nil {
		t.Fatal(err)
	}
	muteStdout(t, func() {
		did, err := migrateItem(dir, "fix", false)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if did {
			t.Error("fully-migrated item should be skipped")
		}
	})
}

func TestMigrateItem_NonIdeaStub(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "spec.json"), oldSpec{Title: "My Fix", Description: "desc"})
	muteStdout(t, func() {
		did, err := migrateItem(dir, "fixes", false)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !did {
			t.Error("non-idea item with no data should produce a plan stub")
		}
	})
	plan, err := os.ReadFile(filepath.Join(dir, "plan.md"))
	if err != nil {
		t.Fatalf("plan.md not created: %v", err)
	}
	if !strings.Contains(string(plan), "My Fix") {
		t.Errorf("plan content = %q", string(plan))
	}
}

func TestMigrateItem_IdeaEmptyWorkshop(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "spec.json"), oldSpec{Title: "Idea"})
	muteStdout(t, func() {
		did, err := migrateItem(dir, "ideas", false)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !did {
			t.Error("idea with no data should create empty workshop")
		}
	})
	if !dirExists(filepath.Join(dir, "workshop")) {
		t.Error("empty workshop/ not created")
	}
}

func TestMigrateItem_FullMigrationFromClarifySuggestEnhance(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "spec.json"), oldSpec{Title: "Full"})
	writeJSON(t, filepath.Join(dir, "clarify", "questions.json"), oldQuestionsFile{
		Questions: []oldQuestion{{ID: "q1", Question: "?", Options: []string{"a"}, Answer: "a"}},
	})
	writeJSON(t, filepath.Join(dir, "suggest", "suggestions.json"), oldSuggestionsFile{
		Suggestions: []oldSuggestion{{ID: "s1", Suggestion: "do x", Status: "accepted"}},
	})
	if err := os.MkdirAll(filepath.Join(dir, "enhance"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "enhance", "summary.md"), []byte("# Plan body"), 0o644); err != nil {
		t.Fatal(err)
	}

	muteStdout(t, func() {
		did, err := migrateItem(dir, "ideas", false)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !did {
			t.Error("expected migration to perform work")
		}
	})

	// plan.md copied from enhance/summary.md.
	plan, err := os.ReadFile(filepath.Join(dir, "plan.md"))
	if err != nil || string(plan) != "# Plan body" {
		t.Errorf("plan.md = %q err=%v", string(plan), err)
	}
	// workshop rounds created.
	if !fileExists(filepath.Join(dir, "workshop", "round-001.json")) {
		t.Error("round-001.json (clarify) not created")
	}
	if !fileExists(filepath.Join(dir, "workshop", "round-002.json")) {
		t.Error("round-002.json (suggest) not created")
	}
	// old dirs removed.
	if dirExists(filepath.Join(dir, "clarify")) || dirExists(filepath.Join(dir, "suggest")) || dirExists(filepath.Join(dir, "enhance")) {
		t.Error("old dirs should be removed after migration")
	}
	// backups created.
	if !dirExists(filepath.Join(dir, ".swarm", "pre-workshop-migration", "clarify")) {
		t.Error("clarify backup not created")
	}
}

func TestMigrateItem_DryRunMakesNoChanges(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "spec.json"), oldSpec{Title: "Dry"})
	writeJSON(t, filepath.Join(dir, "clarify", "questions.json"), oldQuestionsFile{
		Questions: []oldQuestion{{ID: "q1", Question: "?", Answer: nil}},
	})
	muteStdout(t, func() {
		did, err := migrateItem(dir, "ideas", true)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !did {
			t.Error("dry-run should still report it would do work")
		}
	})
	// nothing actually changed.
	if fileExists(filepath.Join(dir, "plan.md")) {
		t.Error("dry-run created plan.md")
	}
	if dirExists(filepath.Join(dir, "workshop")) {
		t.Error("dry-run created workshop/")
	}
	if !dirExists(filepath.Join(dir, "clarify")) {
		t.Error("dry-run removed clarify/")
	}
}

func TestRunMigrateWorkshop_MissingRoot(t *testing.T) {
	muteStdout(t, func() {
		if err := RunMigrateWorkshop(MigrateWorkshopOptions{Root: filepath.Join(t.TempDir(), "nope")}); err == nil {
			t.Error("expected error for missing root")
		}
	})
}

func TestRunMigrateWorkshop_MigratesAndSkips(t *testing.T) {
	root := t.TempDir()
	// One idea item with clarify data → migrated.
	idea := filepath.Join(root, "ideas", "alpha")
	writeJSON(t, filepath.Join(idea, "spec.json"), oldSpec{Title: "Alpha"})
	writeJSON(t, filepath.Join(idea, "clarify", "questions.json"), oldQuestionsFile{
		Questions: []oldQuestion{{ID: "q1", Question: "?", Answer: nil}},
	})
	// One fix item already migrated → skipped.
	done := filepath.Join(root, "fix", "beta")
	writeJSON(t, filepath.Join(done, "spec.json"), oldSpec{Title: "Beta"})
	if err := os.MkdirAll(filepath.Join(done, "workshop"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(done, "plan.md"), []byte("p"), 0o644); err != nil {
		t.Fatal(err)
	}

	muteStdout(t, func() {
		if err := RunMigrateWorkshop(MigrateWorkshopOptions{Root: root}); err != nil {
			t.Fatalf("RunMigrateWorkshop: %v", err)
		}
	})

	if !fileExists(filepath.Join(idea, "workshop", "round-001.json")) {
		t.Error("idea item should have been migrated")
	}
}
