package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFileAndDirExists(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !fileExists(file) {
		t.Error("fileExists(file) should be true")
	}
	if fileExists(dir) {
		t.Error("fileExists(dir) should be false")
	}
	if fileExists(filepath.Join(dir, "missing")) {
		t.Error("fileExists(missing) should be false")
	}
	if !dirExists(dir) {
		t.Error("dirExists(dir) should be true")
	}
	if dirExists(file) {
		t.Error("dirExists(file) should be false")
	}
}

func TestCopyDir(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "out")

	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dst, "sub", "b.txt"))
	if err != nil {
		t.Fatalf("read copied nested file: %v", err)
	}
	if string(got) != "world" {
		t.Errorf("nested content = %q, want world", string(got))
	}
	if a, _ := os.ReadFile(filepath.Join(dst, "a.txt")); string(a) != "hello" {
		t.Errorf("top content = %q, want hello", string(a))
	}
}

func TestReadSpec(t *testing.T) {
	dir := t.TempDir()
	spec := oldSpec{Name: "alpha", Title: "Alpha", Description: "d", Kind: "fix"}
	data, _ := json.Marshal(spec)
	if err := os.WriteFile(filepath.Join(dir, "spec.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readSpec(dir)
	if err != nil {
		t.Fatalf("readSpec: %v", err)
	}
	if got.Name != "alpha" || got.Kind != "fix" {
		t.Errorf("spec = %+v", got)
	}

	// missing file errors.
	if _, err := readSpec(t.TempDir()); err == nil {
		t.Error("expected error for missing spec.json")
	}

	// invalid JSON errors.
	bad := t.TempDir()
	_ = os.WriteFile(filepath.Join(bad, "spec.json"), []byte("{not json"), 0o644)
	if _, err := readSpec(bad); err == nil {
		t.Error("expected parse error for bad spec.json")
	}
}

func TestWriteWorkshopRound(t *testing.T) {
	dir := t.TempDir()
	round := &workshopRound{Round: 1, GeneratedAt: "now", PlanUpdates: "p"}
	if err := writeWorkshopRound(dir, "round-1.json", round); err != nil {
		t.Fatalf("writeWorkshopRound: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "workshop", "round-1.json"))
	if err != nil {
		t.Fatalf("read written round: %v", err)
	}
	var back workshopRound
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Round != 1 || back.PlanUpdates != "p" {
		t.Errorf("round-trip = %+v", back)
	}
}

func TestClarifyToRound(t *testing.T) {
	dir := t.TempDir()
	qf := oldQuestionsFile{
		GeneratedAt: "2024-01-01T00:00:00Z",
		Questions: []oldQuestion{
			{ID: "q1", Question: "Pick one", Options: []string{"sqlite", "postgres"}, Answer: "postgres"},
			{ID: "q2", Question: "Free answer", Options: []string{"a", "b"}, Answer: "custom"},
			{ID: "q3", Question: "Unanswered", Options: []string{"x"}, Answer: nil},
		},
	}
	data, _ := json.Marshal(qf)
	path := filepath.Join(dir, "questions.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	round, err := clarifyToRound(path, 1, true, true)
	if err != nil {
		t.Fatalf("clarifyToRound: %v", err)
	}
	if len(round.Items) != 3 {
		t.Fatalf("items = %d, want 3", len(round.Items))
	}
	// q1 answer matches option index 1 -> selected key "B".
	if round.Items[0].Selected == nil || *round.Items[0].Selected != "B" {
		t.Errorf("q1 selected = %v, want B", round.Items[0].Selected)
	}
	// q2 answer matches no option -> __other__ + freeform.
	if round.Items[1].Selected == nil || *round.Items[1].Selected != "__other__" {
		t.Errorf("q2 selected = %v, want __other__", round.Items[1].Selected)
	}
	if round.Items[1].Freeform == nil || *round.Items[1].Freeform != "custom" {
		t.Errorf("q2 freeform = %v, want custom", round.Items[1].Freeform)
	}
	// q3 unanswered -> no selection.
	if round.Items[2].Selected != nil {
		t.Errorf("q3 selected = %v, want nil", round.Items[2].Selected)
	}
	// 2 of 3 answered -> not all answered -> zero readiness.
	if round.Readiness != (workshopReadiness{}) {
		t.Errorf("partial readiness = %+v, want zero", round.Readiness)
	}
	if round.GeneratedAt != "2024-01-01T00:00:00Z" {
		t.Errorf("generatedAt = %q", round.GeneratedAt)
	}

	// missing file errors.
	if _, err := clarifyToRound(filepath.Join(dir, "nope.json"), 1, false, false); err == nil {
		t.Error("expected error for missing questions file")
	}
}

func TestSuggestToRound(t *testing.T) {
	dir := t.TempDir()
	sf := oldSuggestionsFile{
		GeneratedAt: "2024-02-02T00:00:00Z",
		Suggestions: []oldSuggestion{
			{ID: "s1", Suggestion: "Add cache", Details: "speed", Status: "accepted"},
			{ID: "s2", Suggestion: "Drop feature", Status: "rejected", RejectionReason: "too risky"},
		},
	}
	data, _ := json.Marshal(sf)
	path := filepath.Join(dir, "suggestions.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	round, err := suggestToRound(path, 2, true, true)
	if err != nil {
		t.Fatalf("suggestToRound: %v", err)
	}
	if len(round.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(round.Items))
	}
	if round.Items[0].Selected == nil || *round.Items[0].Selected != "A" {
		t.Errorf("accepted -> A, got %v", round.Items[0].Selected)
	}
	if round.Items[1].Selected == nil || *round.Items[1].Selected != "B" {
		t.Errorf("rejected -> B, got %v", round.Items[1].Selected)
	}
	// rejection reason becomes notes.
	if round.Items[1].Notes == nil || *round.Items[1].Notes != "too risky" {
		t.Errorf("notes = %v, want too risky", round.Items[1].Notes)
	}
	// all decided -> ScopeDefined boosted to >= 2.
	if round.Readiness.ScopeDefined < 2 {
		t.Errorf("ScopeDefined = %d, want >= 2 (all decided)", round.Readiness.ScopeDefined)
	}
	if round.Round != 2 {
		t.Errorf("round = %d, want 2", round.Round)
	}
}
