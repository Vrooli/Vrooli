package workshop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// ComputeEffectiveScores
// ---------------------------------------------------------------------------

func allDimensions(score int) map[string]int {
	m := make(map[string]int, len(ReadinessDimensions))
	for _, d := range ReadinessDimensions {
		m[d] = score
	}
	return m
}

func TestComputeEffectiveScores_AllBelowTwo(t *testing.T) {
	raw := allDimensions(1)
	eff := ComputeEffectiveScores(raw, 5, "idea")
	for _, dim := range ReadinessDimensions {
		if eff[dim] != 1 {
			t.Errorf("%s: got %d, want 1 (no boost for raw < 2)", dim, eff[dim])
		}
	}
}

func TestComputeEffectiveScores_RawTwoZeroRounds(t *testing.T) {
	raw := allDimensions(2)
	eff := ComputeEffectiveScores(raw, 0, "idea")
	for _, dim := range ReadinessDimensions {
		if eff[dim] != 2 {
			t.Errorf("%s: got %d, want 2 (no boost when rounds=0)", dim, eff[dim])
		}
	}
}

func TestComputeEffectiveScores_RawTwoWithRounds(t *testing.T) {
	// kind=idea, N=2, rounds=2 → boost = 2/2 = 1 → effective = min(3, 2+1) = 3
	raw := allDimensions(2)
	eff := ComputeEffectiveScores(raw, 2, "idea")
	for _, dim := range ReadinessDimensions {
		if eff[dim] != 3 {
			t.Errorf("%s: got %d, want 3", dim, eff[dim])
		}
	}
}

func TestComputeEffectiveScores_RawThreeCapped(t *testing.T) {
	raw := allDimensions(3)
	eff := ComputeEffectiveScores(raw, 10, "idea")
	for _, dim := range ReadinessDimensions {
		if eff[dim] != 3 {
			t.Errorf("%s: got %d, want 3 (capped)", dim, eff[dim])
		}
	}
}

func TestComputeEffectiveScores_FixBoostsFaster(t *testing.T) {
	raw := allDimensions(2)
	// kind=fix, N=1, rounds=1 → boost = 1/1 = 1 → effective = 3
	effFix := ComputeEffectiveScores(raw, 1, "fix")
	// kind=idea, N=2, rounds=1 → boost = 1/2 = 0 → effective = 2
	effIdea := ComputeEffectiveScores(raw, 1, "idea")

	dim := ReadinessDimensions[0]
	if effFix[dim] != 3 {
		t.Errorf("fix: got %d, want 3", effFix[dim])
	}
	if effIdea[dim] != 2 {
		t.Errorf("idea: got %d, want 2", effIdea[dim])
	}
}

func TestComputeEffectiveScores_UnknownKindDefaultN(t *testing.T) {
	// Unknown kind → N defaults to 2
	raw := allDimensions(2)
	eff := ComputeEffectiveScores(raw, 2, "unknown_kind")
	dim := ReadinessDimensions[0]
	// boost = 2/2 = 1, effective = min(3, 2+1) = 3
	if eff[dim] != 3 {
		t.Errorf("got %d, want 3 (default N=2)", eff[dim])
	}
}

// ---------------------------------------------------------------------------
// IsReady
// ---------------------------------------------------------------------------

func TestIsReady_AllThree(t *testing.T) {
	eff := allDimensions(3)
	if !IsReady(eff) {
		t.Error("expected ready when all dimensions are 3")
	}
}

func TestIsReady_OneDimensionBelowThree(t *testing.T) {
	eff := allDimensions(3)
	eff[ReadinessDimensions[0]] = 2
	if IsReady(eff) {
		t.Error("expected not ready when one dimension is 2")
	}
}

func TestIsReady_AllZero(t *testing.T) {
	eff := allDimensions(0)
	if IsReady(eff) {
		t.Error("expected not ready when all dimensions are 0")
	}
}

func TestIsReady_EmptyMap(t *testing.T) {
	if IsReady(map[string]int{}) {
		t.Error("expected not ready for empty map")
	}
}

// ---------------------------------------------------------------------------
// LoadRounds
// ---------------------------------------------------------------------------

func writeRoundFile(t *testing.T, workshopDir string, filename string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(workshopDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workshopDir, filename), content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func marshalRound(t *testing.T, r Round) []byte {
	t.Helper()
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestLoadRounds_EmptyDirectory(t *testing.T) {
	itemDir := t.TempDir()
	workshopDir := filepath.Join(itemDir, "workshop")
	if err := os.MkdirAll(workshopDir, 0o755); err != nil {
		t.Fatal(err)
	}

	rounds, err := LoadRounds(itemDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rounds) != 0 {
		t.Errorf("expected 0 rounds, got %d", len(rounds))
	}
}

func TestLoadRounds_NonExistentDirectory(t *testing.T) {
	itemDir := filepath.Join(t.TempDir(), "does-not-exist")

	rounds, err := LoadRounds(itemDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rounds != nil {
		t.Errorf("expected nil, got %v", rounds)
	}
}

func TestLoadRounds_SingleValidRound(t *testing.T) {
	itemDir := t.TempDir()
	workshopDir := filepath.Join(itemDir, "workshop")
	r := Round{RoundNum: 1, GeneratedAt: "2026-01-01T00:00:00Z", Readiness: allDimensions(2)}
	writeRoundFile(t, workshopDir, "round-1.json", marshalRound(t, r))

	rounds, err := LoadRounds(itemDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rounds) != 1 {
		t.Fatalf("expected 1 round, got %d", len(rounds))
	}
	if rounds[0].RoundNum != 1 {
		t.Errorf("expected round 1, got %d", rounds[0].RoundNum)
	}
}

func TestLoadRounds_MultipleSortedByRoundNum(t *testing.T) {
	itemDir := t.TempDir()
	workshopDir := filepath.Join(itemDir, "workshop")
	// Write in reverse order to confirm sorting
	for _, num := range []int{3, 1, 2} {
		r := Round{RoundNum: num, GeneratedAt: "2026-01-01T00:00:00Z", Readiness: allDimensions(2)}
		writeRoundFile(t, workshopDir, fmt.Sprintf("round-%d.json", num), marshalRound(t, r))
	}

	rounds, err := LoadRounds(itemDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rounds) != 3 {
		t.Fatalf("expected 3 rounds, got %d", len(rounds))
	}
	for i, r := range rounds {
		if r.RoundNum != i+1 {
			t.Errorf("index %d: expected round %d, got %d", i, i+1, r.RoundNum)
		}
	}
}

func TestLoadRounds_MalformedJSONSkipped(t *testing.T) {
	itemDir := t.TempDir()
	workshopDir := filepath.Join(itemDir, "workshop")
	// Write a valid round
	r := Round{RoundNum: 1, GeneratedAt: "2026-01-01T00:00:00Z", Readiness: allDimensions(2)}
	writeRoundFile(t, workshopDir, "round-1.json", marshalRound(t, r))
	// Write a malformed round (not repairable)
	writeRoundFile(t, workshopDir, "round-2.json", []byte("not valid json at all !!!"))

	rounds, err := LoadRounds(itemDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rounds) != 1 {
		t.Errorf("expected 1 round (malformed skipped), got %d", len(rounds))
	}
}

func TestLoadRounds_NonRoundFilesIgnored(t *testing.T) {
	itemDir := t.TempDir()
	workshopDir := filepath.Join(itemDir, "workshop")
	r := Round{RoundNum: 1, GeneratedAt: "2026-01-01T00:00:00Z", Readiness: allDimensions(2)}
	writeRoundFile(t, workshopDir, "round-1.json", marshalRound(t, r))
	// Non-round files
	writeRoundFile(t, workshopDir, "notes.txt", []byte("some notes"))
	writeRoundFile(t, workshopDir, "summary.json", []byte(`{"key":"value"}`))

	rounds, err := LoadRounds(itemDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rounds) != 1 {
		t.Errorf("expected 1 round, got %d", len(rounds))
	}
}

// ---------------------------------------------------------------------------
// LoadLatestRound
// ---------------------------------------------------------------------------

func TestLoadLatestRound_Empty(t *testing.T) {
	itemDir := t.TempDir()
	round, count, err := LoadLatestRound(itemDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if round != nil {
		t.Error("expected nil round")
	}
	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}
}

func TestLoadLatestRound_Multiple(t *testing.T) {
	itemDir := t.TempDir()
	workshopDir := filepath.Join(itemDir, "workshop")
	for _, num := range []int{1, 2, 3} {
		r := Round{RoundNum: num, GeneratedAt: "2026-01-01T00:00:00Z", Readiness: allDimensions(2)}
		writeRoundFile(t, workshopDir, fmt.Sprintf("round-%d.json", num), marshalRound(t, r))
	}

	round, count, err := LoadLatestRound(itemDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if round == nil {
		t.Fatal("expected non-nil round")
	}
	if round.RoundNum != 3 {
		t.Errorf("expected latest round 3, got %d", round.RoundNum)
	}
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// CountPendingDecisions
// ---------------------------------------------------------------------------

func strPtr(s string) *string { return &s }

func TestCountPendingDecisions_NilRound(t *testing.T) {
	got := CountPendingDecisions(nil)
	if got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestCountPendingDecisions_MixedItems(t *testing.T) {
	round := &Round{
		Items: []Item{
			{Type: "decision", Topic: "Auth method", Options: []Option{{Key: "A", Label: "OAuth", Rationale: "Industry standard"}}, Selected: nil}, // pending
			{Type: "decision", Topic: "Platform", Options: []Option{{Key: "A", Label: "Web", Rationale: "Broad reach"}}, Selected: strPtr("A")},    // answered
			{Type: "decision", Topic: "Cache strategy", Options: []Option{{Key: "A", Label: "Redis", Rationale: "Fast"}}, Selected: strPtr("")},    // empty = pending
			{Type: "decision", Topic: "Deploy target", Options: []Option{{Key: "A", Label: "K8s", Rationale: "Scalable"}}, Selected: strPtr("  ")}, // whitespace = pending
			{Type: "info", Text: "Some background info"}, // ignored
		},
	}
	got := CountPendingDecisions(round)
	if got != 3 {
		t.Errorf("pending decisions: got %d, want 3", got)
	}
}

func TestCountPendingDecisions_AllAnswered(t *testing.T) {
	round := &Round{
		Items: []Item{
			{Type: "decision", Topic: "Auth method", Options: []Option{{Key: "A", Label: "OAuth", Rationale: "Standard"}}, Selected: strPtr("A")},
			{Type: "decision", Topic: "Platform", Options: []Option{{Key: "A", Label: "Web", Rationale: "Broad"}}, Selected: strPtr("A")},
			{Type: "info", Text: "Background info"},
		},
	}
	got := CountPendingDecisions(round)
	if got != 0 {
		t.Errorf("pending decisions: got %d, want 0", got)
	}
}

func TestCountPendingDecisions_OtherKeyVariants(t *testing.T) {
	tests := []struct {
		name     string
		selected *string
		freeform *string
		wantPend int
	}{
		{"other_nil_freeform", strPtr(OtherKey), nil, 1},
		{"other_empty_freeform", strPtr(OtherKey), strPtr(""), 1},
		{"other_whitespace_freeform", strPtr(OtherKey), strPtr("   "), 1},
		{"other_valid_freeform", strPtr(OtherKey), strPtr("my alternative"), 0},
		{"other_freeform_with_whitespace", strPtr(OtherKey), strPtr("  custom idea  "), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			round := &Round{
				Items: []Item{
					{Type: "decision", Topic: "Test", Selected: tt.selected, Freeform: tt.freeform},
				},
			}
			got := CountPendingDecisions(round)
			if got != tt.wantPend {
				t.Errorf("CountPendingDecisions: got %d, want %d", got, tt.wantPend)
			}
		})
	}
}

func TestNeedsSynthesis_ExplicitFlag(t *testing.T) {
	round := &Round{
		PendingSynthesis: true,
		Items:            []Item{{Type: "info", Text: "Already marked"}},
	}
	if !NeedsSynthesis(round) {
		t.Fatal("expected explicit pending_synthesis=true to require synthesis")
	}
}

func TestNeedsSynthesis_LegacyAnsweredDecisions(t *testing.T) {
	round := &Round{
		Items: []Item{
			{Type: "decision", Topic: "Auth method", Options: []Option{{Key: "A", Label: "OAuth", Rationale: "Standard"}}, Selected: strPtr("A")},
			{Type: "decision", Topic: "Platform", Options: []Option{{Key: "A", Label: "Web", Rationale: "Broad"}}, Selected: strPtr("B")},
			{Type: "info", Text: "Background info"},
		},
	}
	if !NeedsSynthesis(round) {
		t.Fatal("expected legacy answered round to require synthesis")
	}
}

func TestNeedsSynthesis_NoDecisionItems(t *testing.T) {
	round := &Round{
		Items: []Item{{Type: "info", Text: "Fully informational round"}},
	}
	if NeedsSynthesis(round) {
		t.Fatal("expected info-only round to not require synthesis")
	}
}

func TestNeedsSynthesis_FinalizeRound(t *testing.T) {
	round := &Round{
		Mode:             "finalize",
		PendingSynthesis: false,
		Items:            []Item{{Type: "decision", Topic: "Legacy", Selected: strPtr("A")}},
	}
	if NeedsSynthesis(round) {
		t.Fatal("expected finalize rounds to not require synthesis")
	}
}

// ---------------------------------------------------------------------------
// LoadPlanContentByName
// ---------------------------------------------------------------------------

func TestLoadPlanContentByName_Exists(t *testing.T) {
	itemDir := t.TempDir()
	content := "# Conclusion\n\nSome details."
	if err := os.WriteFile(filepath.Join(itemDir, "conclusion.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got := LoadPlanContentByName(itemDir, "conclusion.md")
	if got != content {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestLoadPlanContentByName_Missing(t *testing.T) {
	itemDir := t.TempDir()
	got := LoadPlanContentByName(itemDir, "conclusion.md")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// BuildHistory
// ---------------------------------------------------------------------------

func TestBuildHistory_EmptyRounds(t *testing.T) {
	result := BuildHistory(nil)
	if result != "[]" {
		t.Errorf("got %q, want %q", result, "[]")
	}
	result = BuildHistory([]Round{})
	if result != "[]" {
		t.Errorf("got %q, want %q", result, "[]")
	}
}

func TestBuildHistory_ValidRounds(t *testing.T) {
	rounds := []Round{
		{RoundNum: 1, GeneratedAt: "2026-01-01T00:00:00Z", Readiness: allDimensions(2)},
		{RoundNum: 2, GeneratedAt: "2026-01-02T00:00:00Z", Readiness: allDimensions(3)},
	}
	result := BuildHistory(rounds)

	// Verify it's valid JSON
	var parsed []Round
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(parsed) != 2 {
		t.Errorf("expected 2 rounds in output, got %d", len(parsed))
	}
	if parsed[0].RoundNum != 1 || parsed[1].RoundNum != 2 {
		t.Errorf("unexpected round numbers: %d, %d", parsed[0].RoundNum, parsed[1].RoundNum)
	}
}

// ---------------------------------------------------------------------------
// ResetWorkshop
// ---------------------------------------------------------------------------

func TestResetWorkshop_HappyPath(t *testing.T) {
	itemDir := t.TempDir()
	workshopDir := filepath.Join(itemDir, "workshop")
	clarDir := filepath.Join(workshopDir, "clarifications")
	attachDir := filepath.Join(workshopDir, "attachments")

	// Create 3 rounds.
	for i := 1; i <= 3; i++ {
		writeRoundFile(t, workshopDir, RoundFilename(i), marshalRound(t, Round{RoundNum: i}))
	}
	// Create clarification and attachment dirs with files.
	if err := os.MkdirAll(clarDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(clarDir, "round-001-item-d1.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(attachDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(attachDir, "file.png"), []byte("img"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Create deliverable at item root.
	if err := os.WriteFile(filepath.Join(itemDir, "local-deliverable.md"), []byte("# Deliverable"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Create spec.json (should survive).
	if err := os.WriteFile(filepath.Join(itemDir, "spec.json"), []byte(`{"name":"test"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	deleted, err := ResetWorkshop(itemDir, "local-deliverable.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != 3 {
		t.Errorf("expected 3 deleted rounds, got %d", deleted)
	}
	// Workshop dir should be gone.
	if _, err := os.Stat(workshopDir); !os.IsNotExist(err) {
		t.Error("expected workshop dir to be removed")
	}
	// Deliverable should be gone.
	if _, err := os.Stat(filepath.Join(itemDir, "local-deliverable.md")); !os.IsNotExist(err) {
		t.Error("expected local deliverable to be removed")
	}
	// spec.json should still exist.
	if _, err := os.Stat(filepath.Join(itemDir, "spec.json")); err != nil {
		t.Error("expected spec.json to survive reset")
	}
}

func TestResetWorkshop_NoWorkshopDir(t *testing.T) {
	itemDir := t.TempDir()

	deleted, err := ResetWorkshop(itemDir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != 0 {
		t.Errorf("expected 0 deleted rounds, got %d", deleted)
	}
}

func TestResetWorkshop_PartialData(t *testing.T) {
	itemDir := t.TempDir()
	workshopDir := filepath.Join(itemDir, "workshop")

	// Only rounds, no clarifications/attachments/deliverable.
	writeRoundFile(t, workshopDir, RoundFilename(1), marshalRound(t, Round{RoundNum: 1}))

	deleted, err := ResetWorkshop(itemDir, "conclusion.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted round, got %d", deleted)
	}
	if _, err := os.Stat(workshopDir); !os.IsNotExist(err) {
		t.Error("expected workshop dir to be removed")
	}
}

func TestResetWorkshop_EmptyDeliverableFile(t *testing.T) {
	itemDir := t.TempDir()
	workshopDir := filepath.Join(itemDir, "workshop")
	writeRoundFile(t, workshopDir, RoundFilename(1), marshalRound(t, Round{RoundNum: 1}))

	// Empty deliverable name should still work (no file to remove).
	deleted, err := ResetWorkshop(itemDir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted round, got %d", deleted)
	}
}
