package skills

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"prompt-manager/internal/store"
)

type retirementMetrics struct {
	metrics *SkillMetrics
	deleted int
}

func (m *retirementMetrics) Get(string) (*SkillMetrics, error) { return m.metrics, nil }
func (m *retirementMetrics) RecordUsage(string) (int, time.Time, error) {
	return 0, time.Time{}, nil
}
func (m *retirementMetrics) SetRating(string, int, *string) error { return nil }
func (m *retirementMetrics) Delete(string) error                  { m.deleted++; return nil }

func TestRetireSkillRequiresCompleteZeroCallerReview(t *testing.T) {
	fs, h, metrics, config := newRetirementHandler(t, 0)
	review := completeRetirementReview()
	review.ExternalCallersReviewed = false
	if err := h.RetireSkill(context.Background(), "retire-demo", review); err == nil {
		t.Fatal("incomplete external caller review must be refused")
	}
	if _, err := fs.Get(context.Background(), "retire-demo"); err != nil {
		t.Fatalf("refusal changed active skill: %v", err)
	}
	if metrics.deleted != 0 {
		t.Fatalf("refusal deleted skill metrics %d times", metrics.deleted)
	}
	if _, err := os.Stat(filepath.Join(config, "skills", "retired", "core", "retire-demo")); !os.IsNotExist(err) {
		t.Fatalf("refusal moved skill to retirement custody: stat err=%v", err)
	}
}

func TestRetireSkillRefusesPersistedUseAndPreservesIt(t *testing.T) {
	fs, h, metrics, _ := newRetirementHandler(t, 2)
	if err := h.RetireSkill(context.Background(), "retire-demo", completeRetirementReview()); err == nil || !strings.Contains(err.Error(), "persisted usage") {
		t.Fatalf("persisted use should refuse retirement, got %v", err)
	}
	if _, err := fs.Get(context.Background(), "retire-demo"); err != nil {
		t.Fatalf("usage refusal changed active skill: %v", err)
	}
	if metrics.deleted != 0 {
		t.Fatalf("usage refusal deleted skill metrics %d times", metrics.deleted)
	}
}

func TestRetireSkillMovesOnlyAfterCompleteOwnerReviewAndPreservesHistory(t *testing.T) {
	fs, h, metrics, config := newRetirementHandler(t, 0)
	ctx := context.Background()
	historyPath := filepath.Join(config, "skills", "packs", "core", "retire-demo", "history.jsonl")
	historyBefore, err := os.ReadFile(historyPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.RetireSkill(ctx, "retire-demo", completeRetirementReview()); err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Get(ctx, "retire-demo"); err == nil {
		t.Fatal("retired skill remains active")
	}
	retiredDir := filepath.Join(config, "skills", "retired", "core", "retire-demo")
	historyAfter, err := os.ReadFile(filepath.Join(retiredDir, "history.jsonl"))
	if err != nil || string(historyAfter) != string(historyBefore) {
		t.Fatalf("retirement lost skill history: err=%v before=%s after=%s", err, historyBefore, historyAfter)
	}
	lifecycle, err := os.ReadFile(filepath.Join(retiredDir, "lifecycle.jsonl"))
	if err != nil || !strings.Contains(string(lifecycle), `"action":"retired"`) || !strings.Contains(string(lifecycle), "graph:inventory") {
		t.Fatalf("retirement did not preserve decision evidence: %s err=%v", lifecycle, err)
	}
	if metrics.deleted != 0 {
		t.Fatalf("successful retirement deleted retained metric history %d times", metrics.deleted)
	}
}

func newRetirementHandler(t *testing.T, usage int) (*store.FileSkillStore, *Handlers, *retirementMetrics, string) {
	t.Helper()
	config := t.TempDir()
	if err := os.MkdirAll(filepath.Join(config, "skills", "packs", "core"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveJSON(filepath.Join(config, "skills", "_pack-order.json"), &store.PackOrder{ActivePacks: []string{"core"}}); err != nil {
		t.Fatal(err)
	}
	fs := store.NewFileSkillStore(config)
	if err := fs.Create(context.Background(), "core", &store.Skill{ID: "retire-demo", Name: "Retire demo", Status: store.StatusActive}, "Useful authored content."); err != nil {
		t.Fatal(err)
	}
	metrics := &retirementMetrics{metrics: &SkillMetrics{SkillID: "retire-demo", UsageCount: usage}}
	adapter := NewStoreAdapter(fs, store.NewFileContentIO())
	return fs, NewHandlers(adapter, metrics, config), metrics, config
}

func completeRetirementReview() SkillRetirementReview {
	return SkillRetirementReview{
		Actor: "owner", Reason: "candidate reviewed across active caller surfaces",
		GraphInboundEdges: 0, RepositoryMentions: 0, UsageReads90d: 0,
		ExternalCallersReviewed: true, ExternalCallerCount: 0,
		EvidenceRefs: []string{"graph:inventory.json", "repository:references.json", "usage:90d.json", "external-callers:review.md"},
	}
}
