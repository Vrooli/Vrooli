package main

import "testing"

func TestParseBranchRefs(t *testing.T) {
	input := []byte("refs/heads/main|main|origin/main|abc123|2025-01-01 00:00:00 +0000\nrefs/remotes/origin/main|origin/main||def456|2025-01-01 01:00:00 +0000\nrefs/remotes/origin/HEAD|origin/HEAD||zzz999|2025-01-01 01:00:00 +0000")
	refs, err := ParseBranchRefs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(refs))
	}
	if refs[0].ShortName != "main" || refs[0].IsRemote {
		t.Fatalf("expected local main branch")
	}
	if refs[1].ShortName != "origin/main" || !refs[1].IsRemote {
		t.Fatalf("expected remote origin/main branch")
	}
}

func TestParseBranchRefs_InvalidLine(t *testing.T) {
	_, err := ParseBranchRefs([]byte("invalid-line"))
	if err == nil {
		t.Fatalf("expected error for invalid line")
	}
}
