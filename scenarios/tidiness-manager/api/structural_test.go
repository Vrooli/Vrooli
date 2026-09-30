package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// descriptorBlock is a real-shaped uniform EndpointDescriptor list — the exact
// kind of cross-file uniformity screaming architecture enforces and naive
// duplication detection flags. The Description strings deliberately contain the
// English words "if" and "for" to prove prose does not register as control flow.
const descriptorBlock = `
	{
		Method:      "POST",
		Path:        "/campaigns",
		Handler:     h.CreateCampaign,
		Description: "Create a campaign if none exists for the scenario",
	},
	{
		Method:      "GET",
		Path:        "/campaigns",
		Handler:     h.ListCampaigns,
		Description: "List campaigns for the current scenario",
	},
	{
		Method:      "DELETE",
		Path:        "/campaigns/{id}",
		Handler:     h.DeleteCampaign,
		Description: "Delete a campaign",
	},
`

// logicBlock is a real-shaped branchy function body — genuine logic that must
// keep its normal severity even when it appears in a role-named file.
const logicBlock = `
	for _, item := range items {
		if item.Score > threshold {
			switch item.Kind {
			case KindA:
				results = append(results, transform(item))
			case KindB:
				if item.Valid && item.Ready {
					results = append(results, item)
				}
			default:
				return nil, fmt.Errorf("unknown kind %v", item.Kind)
			}
		}
	}
`

func TestIsStructuralBlock(t *testing.T) {
	if !IsStructuralBlock(strings.Split(descriptorBlock, "\n")) {
		t.Error("uniform descriptor list should be classified structural")
	}
	if IsStructuralBlock(strings.Split(logicBlock, "\n")) {
		t.Error("branchy function body should be classified logic, not structural")
	}
}

func TestIsStructuralBlock_TooSmallIsLogic(t *testing.T) {
	// Below the minimum-signal floor: default to logic (do not cap).
	if IsStructuralBlock([]string{"x := 1", "y := 2"}) {
		t.Error("tiny block should default to logic")
	}
}

func TestReadBlockLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")
	content := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := readBlockLines(path, 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"line2", "line3", "line4"}
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: got %q, want %q", i, got[i], want[i])
		}
	}

	// Reading past EOF returns the available lines without error.
	tail, err := readBlockLines(path, 4, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(tail) != 2 {
		t.Errorf("past-EOF read: got %d lines, want 2: %v", len(tail), tail)
	}
}
