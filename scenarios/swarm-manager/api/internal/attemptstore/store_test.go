package attemptstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testRound struct {
	Number int    `json:"round"`
	Value  string `json:"value"`
	Proofs []struct {
		ID string `json:"id"`
	} `json:"proofs,omitempty"`
}

func (r testRound) RoundNumber() int { return r.Number }
func decodeTestRound(data []byte) (testRound, error) {
	var r testRound
	return r, json.Unmarshal(data, &r)
}

func TestRoundStoreOrdersAndRedacts(t *testing.T) {
	root := t.TempDir()
	if err := SaveRound(root, "review", testRound{Number: 2, Value: os.Getenv("HOME")}); err != nil {
		t.Fatal(err)
	}
	if err := SaveRound(root, "review", testRound{Number: 1, Value: "first"}); err != nil {
		t.Fatal(err)
	}
	rounds, err := LoadRounds(root, "review", decodeTestRound)
	if err != nil {
		t.Fatal(err)
	}
	if len(rounds) != 2 || rounds[0].Number != 1 || rounds[1].Number != 2 {
		t.Fatalf("round order = %#v", rounds)
	}
	data, err := os.ReadFile(filepath.Join(root, "review", RoundFilename(2)))
	if err != nil {
		t.Fatal(err)
	}
	if home := os.Getenv("HOME"); home != "" && strings.Contains(string(data), home) {
		t.Fatalf("persisted attempt leaked operator path: %s", data)
	}
}

func TestCaptureRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if _, err := SaveCapture(root, "review", "../escape", []byte("x")); err == nil {
		t.Fatal("expected traversal rejection")
	}
	path, err := SaveCapture(root, "review", "nested/proof.txt", []byte("ok"))
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join("captures", "nested", "proof.txt") {
		t.Fatalf("path = %q", path)
	}
}

func TestLoadRoundRepairsTruncatedPayload(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "review")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	// The first proof is complete while the second was interrupted. This is the
	// real failure shape from a partially persisted agent result.
	broken := `{"round":1,"value":"kept","proofs":[{"id":"e1"},{"id":"e2"`
	if err := os.WriteFile(filepath.Join(dir, RoundFilename(1)), []byte(broken), 0o600); err != nil {
		t.Fatal(err)
	}
	round, err := LoadRound(root, "review", 1, decodeTestRound)
	if err != nil {
		t.Fatalf("LoadRound: %v", err)
	}
	if round == nil || round.Number != 1 || round.Value != "kept" || len(round.Proofs) != 1 || round.Proofs[0].ID != "e1" {
		t.Fatalf("repaired round = %#v", round)
	}
}
