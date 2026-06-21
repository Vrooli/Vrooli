package autofix

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildFixResponseEmptyStampsMessage(t *testing.T) {
	resp := BuildFixResponse("demo", false, nil)
	if resp.GetScenario() != "demo" {
		t.Fatalf("scenario = %q, want demo", resp.GetScenario())
	}
	if resp.GetApplied() {
		t.Fatalf("applied = true, want false")
	}
	if len(resp.GetCandidates()) != 0 {
		t.Fatalf("candidates = %d, want 0", len(resp.GetCandidates()))
	}
	if len(resp.GetMessages()) != 1 || resp.GetMessages()[0] != NoFixesMessage {
		t.Fatalf("messages = %v, want [%q]", resp.GetMessages(), NoFixesMessage)
	}
}

func TestCandidatesProtoRoundTrip(t *testing.T) {
	in := []Candidate{{
		RuleID:      "R1",
		FilePath:    "/tmp/a",
		Description: "do thing",
		Before:      "x",
		After:       "y",
		Applied:     true,
	}}
	got := CandidatesFromProto(CandidatesToProto(in))
	if len(got) != 1 {
		t.Fatalf("round-trip len = %d, want 1", len(got))
	}
	if got[0] != in[0] {
		t.Fatalf("round-trip mismatch: %+v != %+v", got[0], in[0])
	}
}

func TestRegistryPreviewAndApplyFixResponse(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "out.txt")
	reg := NewRegistry(Fixer{
		RuleID: "R1",
		Preview: func(root string) ([]Candidate, error) {
			return []Candidate{{RuleID: "R1", FilePath: target, Description: "create", After: "hello"}}, nil
		},
		CanFix: func(root, findingPath string) bool { return true },
	})

	preview, err := reg.PreviewFixResponse("demo", dir, nil)
	if err != nil {
		t.Fatalf("PreviewFixResponse: %v", err)
	}
	if preview.GetApplied() {
		t.Fatalf("preview applied = true, want false")
	}
	if len(preview.GetCandidates()) != 1 || preview.GetCandidates()[0].GetApplied() {
		t.Fatalf("preview candidate unexpected: %+v", preview.GetCandidates())
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("preview must not write file")
	}

	applied, err := reg.ApplyFixResponse("demo", dir, []string{"R1"})
	if err != nil {
		t.Fatalf("ApplyFixResponse: %v", err)
	}
	if !applied.GetApplied() || len(applied.GetCandidates()) != 1 || !applied.GetCandidates()[0].GetApplied() {
		t.Fatalf("apply response unexpected: %+v", applied)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read applied file: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("applied content = %q, want hello", string(data))
	}
}
