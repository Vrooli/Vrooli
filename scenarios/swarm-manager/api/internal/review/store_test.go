package review

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func validCompletedRound(roundNum int) Round {
	return Round{
		RoundNum:        roundNum,
		GeneratedAt:     "2026-04-02T00:00:00Z",
		Status:          RoundStatusComplete,
		AgentAssessment: "Looks good.",
		Classification:  "ready",
	}
}

func TestLoadRounds_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	rounds, err := LoadRounds(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rounds) != 0 {
		t.Fatalf("expected 0 rounds, got %d", len(rounds))
	}
}

func TestLoadRounds_SingleRound(t *testing.T) {
	dir := t.TempDir()
	reviewDir := filepath.Join(dir, "review")
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		t.Fatal(err)
	}

	round := validCompletedRound(1)
	round.Evidence = []EvidenceItem{
		{ID: "e1", Type: EvidenceTypeScreenshot, Title: "Dashboard", Description: "After changes"},
	}
	data, _ := json.MarshalIndent(round, "", "  ")
	if err := os.WriteFile(filepath.Join(reviewDir, "round-001.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	rounds, err := LoadRounds(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rounds) != 1 {
		t.Fatalf("expected 1 round, got %d", len(rounds))
	}
	if rounds[0].RoundNum != 1 {
		t.Errorf("expected round 1, got %d", rounds[0].RoundNum)
	}
	if len(rounds[0].Evidence) != 1 {
		t.Errorf("expected 1 evidence item, got %d", len(rounds[0].Evidence))
	}
}

func TestLoadRounds_MultipleSortedByNumber(t *testing.T) {
	dir := t.TempDir()
	reviewDir := filepath.Join(dir, "review")
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write rounds out of order on disk.
	for _, num := range []int{3, 1, 2} {
		round := validCompletedRound(num)
		data, _ := json.MarshalIndent(round, "", "  ")
		if err := os.WriteFile(filepath.Join(reviewDir, RoundFilename(num)), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	rounds, err := LoadRounds(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rounds) != 3 {
		t.Fatalf("expected 3 rounds, got %d", len(rounds))
	}
	for i, expected := range []int{1, 2, 3} {
		if rounds[i].RoundNum != expected {
			t.Errorf("round[%d]: expected %d, got %d", i, expected, rounds[i].RoundNum)
		}
	}
}

func TestLoadRounds_SkipsMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	reviewDir := filepath.Join(dir, "review")
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a valid round.
	round := validCompletedRound(1)
	data, _ := json.MarshalIndent(round, "", "  ")
	if err := os.WriteFile(filepath.Join(reviewDir, "round-001.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a malformed file.
	if err := os.WriteFile(filepath.Join(reviewDir, "round-002.json"), []byte("{invalid"), 0o644); err != nil {
		t.Fatal(err)
	}

	rounds, err := LoadRounds(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rounds) != 1 {
		t.Fatalf("expected 1 valid round (skipping malformed), got %d", len(rounds))
	}
}

func TestLoadRounds_IgnoresNonRoundFiles(t *testing.T) {
	dir := t.TempDir()
	reviewDir := filepath.Join(dir, "review")
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a captures/ subdirectory and a non-round file.
	if err := os.MkdirAll(filepath.Join(reviewDir, "captures"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reviewDir, "metadata.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	round := validCompletedRound(1)
	data, _ := json.MarshalIndent(round, "", "  ")
	if err := os.WriteFile(filepath.Join(reviewDir, "round-001.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	rounds, err := LoadRounds(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rounds) != 1 {
		t.Fatalf("expected 1 round, got %d", len(rounds))
	}
}

func TestLoadLatestRound(t *testing.T) {
	dir := t.TempDir()
	reviewDir := filepath.Join(dir, "review")
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, num := range []int{1, 2} {
		round := validCompletedRound(num)
		data, _ := json.MarshalIndent(round, "", "  ")
		if err := os.WriteFile(filepath.Join(reviewDir, RoundFilename(num)), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	latest, count, err := LoadLatestRound(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 total, got %d", count)
	}
	if latest.RoundNum != 2 {
		t.Errorf("expected latest round 2, got %d", latest.RoundNum)
	}
}

func TestLoadLatestRound_Empty(t *testing.T) {
	dir := t.TempDir()
	latest, count, err := LoadLatestRound(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
	if latest != nil {
		t.Error("expected nil round")
	}
}

func TestLoadRound_Specific(t *testing.T) {
	dir := t.TempDir()
	reviewDir := filepath.Join(dir, "review")
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		t.Fatal(err)
	}

	round := Round{RoundNum: 2, GeneratedAt: "2026-04-02T00:00:00Z", Status: RoundStatusGathering}
	data, _ := json.MarshalIndent(round, "", "  ")
	if err := os.WriteFile(filepath.Join(reviewDir, "round-002.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadRound(dir, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected round, got nil")
	}
	if loaded.Status != RoundStatusGathering {
		t.Errorf("expected status gathering, got %s", loaded.Status)
	}
}

func TestLoadRound_NotFound(t *testing.T) {
	dir := t.TempDir()
	loaded, err := LoadRound(dir, 99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil for non-existent round")
	}
}

func TestSaveRound_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	round := Round{
		RoundNum:    1,
		GeneratedAt: "2026-04-02T00:00:00Z",
		Status:      RoundStatusGathering,
		Evidence:    []EvidenceItem{},
	}

	if err := SaveRound(dir, round); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file exists and is valid JSON.
	data, err := os.ReadFile(filepath.Join(dir, "review", "round-001.json"))
	if err != nil {
		t.Fatalf("file not found: %v", err)
	}
	var loaded Round
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if loaded.RoundNum != 1 {
		t.Errorf("expected round 1, got %d", loaded.RoundNum)
	}
}

func TestNextRoundNumber(t *testing.T) {
	dir := t.TempDir()

	// Empty: should return 1.
	n, err := NextRoundNumber(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1, got %d", n)
	}

	// After saving round 1: should return 2.
	if err := SaveRound(dir, validCompletedRound(1)); err != nil {
		t.Fatal(err)
	}
	n, err = NextRoundNumber(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2, got %d", n)
	}
}

func TestSaveCapture_CreatesDirectoryAndRejectsTraversal(t *testing.T) {
	dir := t.TempDir()

	// Valid capture.
	relPath, err := SaveCapture(dir, "dashboard.png", []byte("png-data"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if relPath != filepath.Join("captures", "dashboard.png") {
		t.Errorf("unexpected path: %s", relPath)
	}

	// Verify file exists.
	data, err := os.ReadFile(filepath.Join(dir, "review", "captures", "dashboard.png"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "png-data" {
		t.Errorf("unexpected content: %s", string(data))
	}

	// Path traversal should be rejected.
	_, err = SaveCapture(dir, "../../../etc/passwd", []byte("bad"))
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestLoadCapture_RejectsTraversal(t *testing.T) {
	dir := t.TempDir()

	_, err := LoadCapture(dir, "../../../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal")
	}

	_, err = LoadCapture(dir, "/absolute/path")
	if err == nil {
		t.Error("expected error for absolute path")
	}
}

func TestLoadRounds_NormalizesInvalidCompletedRound(t *testing.T) {
	dir := t.TempDir()
	reviewDir := filepath.Join(dir, "review")
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		t.Fatal(err)
	}

	round := Round{
		RoundNum:    1,
		GeneratedAt: "2026-04-02T00:00:00Z",
		Status:      RoundStatusComplete,
	}
	data, _ := json.MarshalIndent(round, "", "  ")
	if err := os.WriteFile(filepath.Join(reviewDir, "round-001.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	rounds, err := LoadRounds(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rounds) != 1 {
		t.Fatalf("expected 1 round, got %d", len(rounds))
	}
	if rounds[0].Status != RoundStatusFailed {
		t.Fatalf("expected invalid complete round to normalize to failed, got %s", rounds[0].Status)
	}
	if rounds[0].FailureReason == "" {
		t.Fatal("expected normalized round to include failure reason")
	}
}

func TestLoadRound_NormalizesInvalidCompletedRound(t *testing.T) {
	dir := t.TempDir()
	reviewDir := filepath.Join(dir, "review")
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		t.Fatal(err)
	}

	round := Round{
		RoundNum:    2,
		GeneratedAt: "2026-04-02T00:00:00Z",
		Status:      RoundStatusComplete,
	}
	data, _ := json.MarshalIndent(round, "", "  ")
	if err := os.WriteFile(filepath.Join(reviewDir, "round-002.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadRound(dir, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected round, got nil")
	}
	if loaded.Status != RoundStatusFailed {
		t.Fatalf("expected invalid complete round to normalize to failed, got %s", loaded.Status)
	}
}

func TestRoundFilename(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{1, "round-001.json"},
		{10, "round-010.json"},
		{100, "round-100.json"},
	}
	for _, tt := range tests {
		got := RoundFilename(tt.n)
		if got != tt.want {
			t.Errorf("RoundFilename(%d) = %s, want %s", tt.n, got, tt.want)
		}
	}
}
