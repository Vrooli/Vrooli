package brief

import (
	"context"
	"database/sql"
	"testing"
	"time"

	agentbrief "github.com/vrooli/agentbrief-go"
	"github.com/vrooli/api-core/scheduletest"
	_ "modernc.org/sqlite"
)

type fakeHub struct{ result agentbrief.QueryResult }

func (f fakeHub) Query(context.Context, agentbrief.QueryInput) (agentbrief.QueryResult, error) {
	return f.result, nil
}

func newBriefDB(t *testing.T) (*sql.DB, *scheduletest.FakeClock) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:brief-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec("PRAGMA foreign_keys = ON;" + Schema()); err != nil {
		t.Fatal(err)
	}
	clock := scheduletest.New(time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC))
	return db, clock
}

func TestWithheldBriefRoundTripsAndDoesNotStoreExternalPrompt(t *testing.T) {
	db, clock := newBriefDB(t)
	repo := NewSQLiteRepository(db, clock)
	service := NewService(Config{Repository: repo, Hub: fakeHub{}, Clock: clock, Thresholds: agentbrief.DefaultThresholds})
	prompt := "external secret prompt that must not be persisted"
	record, err := service.Build(context.Background(), BuildInput{Prompt: prompt, Consumer: agentbrief.ConsumerExternalHarness, SessionRef: "opaque-session"})
	if err != nil {
		t.Fatal(err)
	}
	if record.Verdict != agentbrief.VerdictWithheldLowConfidence {
		t.Fatalf("verdict = %s", record.Verdict)
	}
	got, err := service.Get(context.Background(), record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Reason != record.Reason || got.Verdict != record.Verdict {
		t.Fatalf("round trip = %+v", got)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM briefs WHERE prompt_digest=? OR rendered=?", prompt, prompt).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("raw external prompt was persisted")
	}
	if got.PromptDigest == prompt || got.PromptDigest == "" {
		t.Fatal("prompt digest was not stored as a digest")
	}
	if got.EffectiveQuery != prompt {
		t.Fatalf("effective query = %q, want normalized query", got.EffectiveQuery)
	}
}

func TestRetentionCascadesItemsAndUses(t *testing.T) {
	db, clock := newBriefDB(t)
	repo := NewSQLiteRepository(db, clock)
	old := Record{ID: "old", Consumer: agentbrief.ConsumerPortalAgent, Verdict: agentbrief.VerdictDeliver, PromptDigest: "digest", CreatedAt: clock.Now().Add(-31 * 24 * time.Hour), Items: []agentbrief.Item{{ProviderID: "cli-health.commands", Type: "command", Title: "old"}}}
	if err := repo.Save(context.Background(), old); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.RecordUse(context.Background(), UseInput{BriefID: old.ID, ItemIndex: 0, Kind: "OPENED"}, clock.Now()); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteBefore(context.Background(), clock.Now().Add(-30*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"briefs", "brief_items", "brief_uses"} {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s retained %d rows", table, count)
		}
	}
}

func TestStatsReportsDeliveryWithholdingAndUsageByConsumer(t *testing.T) {
	db, clock := newBriefDB(t)
	repo := NewSQLiteRepository(db, clock)
	delivered := Record{ID: "delivered", Consumer: agentbrief.ConsumerPortalLLM, Verdict: agentbrief.VerdictDeliver, PromptDigest: "d1", CreatedAt: clock.Now(), Items: []agentbrief.Item{{ProviderID: "cli-health.commands", Type: "command", Title: "status"}}}
	withheld := Record{ID: "withheld", Consumer: agentbrief.ConsumerPortalLLM, Verdict: agentbrief.VerdictWithheldLowConfidence, Reason: "weak", PromptDigest: "d2", CreatedAt: clock.Now()}
	for _, record := range []Record{delivered, withheld} {
		if err := repo.Save(context.Background(), record); err != nil {
			t.Fatal(err)
		}
	}
	first, err := repo.RecordUse(context.Background(), UseInput{BriefID: delivered.ID, ItemIndex: 0, Kind: "COPIED"}, clock.Now())
	if err != nil || !first {
		t.Fatalf("first use = %v/%v", first, err)
	}
	second, err := repo.RecordUse(context.Background(), UseInput{BriefID: delivered.ID, ItemIndex: 0, Kind: "COPIED"}, clock.Now())
	if err != nil || second {
		t.Fatalf("duplicate use = %v/%v", second, err)
	}
	rows, err := repo.Stats(context.Background(), StatsInput{WindowDays: 7, Consumer: agentbrief.ConsumerPortalLLM}, clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("stats rows = %d", len(rows))
	}
	row := rows[0]
	if row.BriefsBuilt != 2 || row.BriefsDelivered != 1 || row.ItemsDelivered != 1 || row.ItemsUsed != 1 {
		t.Fatalf("stats counts = %+v", row)
	}
	if row.UsageRate != 1 || row.WithheldRate != 0.5 || row.WithheldByVerdict[agentbrief.VerdictWithheldLowConfidence] != 1 {
		t.Fatalf("stats rates = %+v", row)
	}
}
