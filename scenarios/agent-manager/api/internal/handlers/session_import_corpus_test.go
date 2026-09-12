package handlers

import (
	"testing"

	"agent-manager/internal/domain"
)

func TestSelectCorpusCandidatesIsBoundedDeterministicAndCrossRunner(t *testing.T) {
	candidates := []corpusCandidate{
		{source: runnerSessionSource{RunnerType: domain.RunnerTypeCodex}, session: discoveredSession{Key: "z.jsonl"}, month: "2026-02"},
		{source: runnerSessionSource{RunnerType: domain.RunnerTypeClaudeCode}, session: discoveredSession{Key: "b.jsonl"}, month: "2026-01"},
		{source: runnerSessionSource{RunnerType: domain.RunnerTypeCodex}, session: discoveredSession{Key: "a.jsonl"}, month: "2026-01"},
		{source: runnerSessionSource{RunnerType: domain.RunnerTypeCodex}, session: discoveredSession{Key: "b.jsonl"}, month: "2026-01"},
		{source: runnerSessionSource{RunnerType: domain.RunnerTypeClaudeCode}, session: discoveredSession{Key: "a.jsonl"}, month: "2026-02"},
	}
	selected := selectCorpusCandidates(candidates, 1, 4)
	if len(selected) != 4 {
		t.Fatalf("selected %d candidates, want 4", len(selected))
	}
	want := []string{"2026-01/claude-code/b.jsonl", "2026-01/codex/a.jsonl", "2026-02/claude-code/a.jsonl", "2026-02/codex/z.jsonl"}
	for i, candidate := range selected {
		got := candidate.month + "/" + string(candidate.source.RunnerType) + "/" + candidate.session.Key
		if got != want[i] {
			t.Fatalf("selected[%d] = %q, want %q", i, got, want[i])
		}
	}
}

func TestCorpusWindowRejectsInvalidOrReversedBounds(t *testing.T) {
	if _, _, err := corpusWindow("not-a-time", ""); err == nil {
		t.Fatal("expected invalid timestamp error")
	}
	if _, _, err := corpusWindow("2026-02-01T00:00:00Z", "2026-01-01T00:00:00Z"); err == nil {
		t.Fatal("expected reversed window error")
	}
}

func TestSelectCorpusCandidatesDoesNotLetOlderRunnerConsumeLimit(t *testing.T) {
	candidates := []corpusCandidate{
		{source: runnerSessionSource{RunnerType: domain.RunnerTypeCodex}, session: discoveredSession{Key: "old-1.jsonl"}, month: "2025-01"},
		{source: runnerSessionSource{RunnerType: domain.RunnerTypeCodex}, session: discoveredSession{Key: "old-2.jsonl"}, month: "2025-02"},
		{source: runnerSessionSource{RunnerType: domain.RunnerTypeCodex}, session: discoveredSession{Key: "old-3.jsonl"}, month: "2025-03"},
		{source: runnerSessionSource{RunnerType: domain.RunnerTypeClaudeCode}, session: discoveredSession{Key: "new-1.jsonl"}, month: "2026-06"},
		{source: runnerSessionSource{RunnerType: domain.RunnerTypeClaudeCode}, session: discoveredSession{Key: "new-2.jsonl"}, month: "2026-07"},
	}
	selected := selectCorpusCandidates(candidates, 1, 4)
	if len(selected) != 4 {
		t.Fatalf("selected %d candidates, want 4", len(selected))
	}
	seen := map[domain.RunnerType]int{}
	for _, candidate := range selected {
		seen[candidate.source.RunnerType]++
	}
	if seen[domain.RunnerTypeCodex] != 2 || seen[domain.RunnerTypeClaudeCode] != 2 {
		t.Fatalf("runner distribution = %#v, want 2 each", seen)
	}
}
