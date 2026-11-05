package domain

import "testing"

// The scheduled sweep and the corpus importer label the same store differently.
// If those labels are not equivalent, each path adopts sessions the other has
// already imported and the corpus duplicates on every sweep.
func TestNormalizeImportHarnessTreatsPathLabelsAsTheSameHarness(t *testing.T) {
	for _, group := range [][]string{
		{"codex", "resource:codex/sessions", "Resource:Codex/Sessions", "  codex  "},
		{"claude-code", "resource:claude-code/projects", "claude-code/projects"},
	} {
		want := NormalizeImportHarness(group[0])
		if want == "" {
			t.Fatalf("normalizing %q produced an empty identity", group[0])
		}
		for _, label := range group[1:] {
			if got := NormalizeImportHarness(label); got != want {
				t.Fatalf("NormalizeImportHarness(%q) = %q, want %q", label, got, want)
			}
		}
	}
}

// Normalization must not collapse genuinely different harnesses into one, or a
// codex session would resolve to a claude-code run.
func TestNormalizeImportHarnessKeepsDistinctHarnessesApart(t *testing.T) {
	if NormalizeImportHarness("resource:codex/sessions") == NormalizeImportHarness("resource:claude-code/projects") {
		t.Fatal("codex and claude-code must not share an identity")
	}
}

func TestNormalizeImportHarnessHandlesEmptyInput(t *testing.T) {
	if got := NormalizeImportHarness("   "); got != "" {
		t.Fatalf("NormalizeImportHarness(blank) = %q, want empty", got)
	}
}
