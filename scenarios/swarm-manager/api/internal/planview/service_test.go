package planview

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/eta"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/operations"
	"swarm-manager/internal/stats"

	"github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

var testNow = time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)

type stubBacklog struct {
	items []backlog.BacklogItem
	err   error
}

func (s stubBacklog) LoadAll(_ []backlog.BacklogKind) ([]backlog.BacklogItem, error) {
	return s.items, s.err
}

type stubGates struct{ gates []Gate }

func (s stubGates) Enumerate(context.Context) []Gate { return s.gates }

type stubExecs struct {
	records []execution.Record
	err     error
}

func (s stubExecs) List(context.Context, execution.ListFilters) ([]execution.Record, error) {
	return s.records, s.err
}

type stubOps struct {
	view *operations.OperationsView
	err  error
}

func (s stubOps) Aggregate(context.Context, operations.Filters) (*operations.OperationsView, error) {
	return s.view, s.err
}

func newTestService(t *testing.T, cfg Config) *Service {
	t.Helper()
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return testNow }
	}
	svc, err := NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func bItem(kind, name string, status backlog.BacklogStatus, deps ...string) backlog.BacklogItem {
	return backlog.BacklogItem{
		Kind:      backlog.BacklogKind(kind),
		Name:      name,
		Title:     name + " title",
		Status:    status,
		DependsOn: deps,
		Updated:   testNow.Add(-time.Hour).Format(time.RFC3339),
	}
}

func decideGate(kind, name string, count int, blocks ...string) Gate {
	return Gate{
		ID:        fmt.Sprintf("decide:backlog/%s/%s", kind, name),
		Kind:      KindDecide,
		OwnerType: "backlog",
		OwnerKind: kind, OwnerName: name, OwnerTitle: name + " title",
		Count: count, Blocks: blocks,
	}
}

func workshopGate(kind, name, suggested string) Gate {
	return Gate{
		ID:        fmt.Sprintf("workshop:backlog/%s/%s", kind, name),
		Kind:      KindWorkshop,
		OwnerType: "backlog",
		OwnerKind: kind, OwnerName: name, OwnerTitle: name + " title",
		Count: 1, Suggested: suggested,
	}
}

func findGroup(t *testing.T, col Column, id string) CardGroup {
	t.Helper()
	for _, g := range col.Groups {
		if g.ID == id {
			return g
		}
	}
	t.Fatalf("group %q not found in %+v", id, col.Groups)
	return CardGroup{}
}

func cardIDs(cards []Card) []string {
	out := make([]string, 0, len(cards))
	for _, c := range cards {
		out = append(out, c.ID)
	}
	return out
}

func TestBuild_NextSplitsReadyWorkshopAndGates(t *testing.T) {
	items := []backlog.BacklogItem{
		bItem("fix", "runnable", backlog.StatusReady),
		bItem("fix", "raw", backlog.StatusBacklog),
		bItem("fix", "questions", backlog.StatusBacklog),
	}
	svc := newTestService(t, Config{
		Backlog: stubBacklog{items: items},
		Gates: stubGates{gates: []Gate{
			workshopGate("fix", "raw", "workshop"),
			decideGate("fix", "questions", 3),
		}},
	})

	board, err := svc.Build(context.Background(), Params{})
	if err != nil {
		t.Fatal(err)
	}

	gatesGroup := findGroup(t, board.Next, "gates")
	if len(gatesGroup.Cards) != 1 || gatesGroup.Cards[0].Action != ActionDecide {
		t.Errorf("expected one decide gate card, got %+v", gatesGroup.Cards)
	}
	if gatesGroup.Cards[0].Gate == nil || gatesGroup.Cards[0].Gate.Count != 3 {
		t.Errorf("gate payload missing: %+v", gatesGroup.Cards[0].Gate)
	}

	ready := findGroup(t, board.Next, "ready")
	if len(ready.Cards) != 1 || ready.Cards[0].ID != "backlog-item/fix/runnable" || ready.Cards[0].Action != ActionRun {
		t.Errorf("unexpected ready cards: %+v", ready.Cards)
	}

	workshop := findGroup(t, board.Next, "workshop")
	if len(workshop.Cards) != 1 || workshop.Cards[0].Action != ActionWorkshop {
		t.Errorf("unexpected workshop cards: %+v", workshop.Cards)
	}

	if board.Next.CardCount != 3 {
		t.Errorf("expected card count 3, got %d", board.Next.CardCount)
	}
}

func TestBuild_DecideGateCarriesItemNotDuplicated(t *testing.T) {
	items := []backlog.BacklogItem{
		bItem("fix", "questions", backlog.StatusBacklog),
	}
	svc := newTestService(t, Config{
		Backlog: stubBacklog{items: items},
		Gates:   stubGates{gates: []Gate{decideGate("fix", "questions", 2)}},
	})
	board, err := svc.Build(context.Background(), Params{})
	if err != nil {
		t.Fatal(err)
	}
	if board.Next.CardCount != 1 {
		t.Errorf("item with decide gate must appear exactly once, got %d cards: %+v",
			board.Next.CardCount, board.Next.Groups)
	}
	if board.Later.CardCount != 0 {
		t.Errorf("expected empty Later, got %+v", board.Later.Groups)
	}
}

func TestBuild_LaterGroupsByNearestBlocker(t *testing.T) {
	items := []backlog.BacklogItem{
		bItem("fix", "gated", backlog.StatusBacklog),                   // wave 0, has decide gate
		bItem("fix", "after-gate", backlog.StatusBacklog, "fix/gated"), // blocked by gated item
		bItem("fix", "base", backlog.StatusReady),                      // wave 0 runnable
		bItem("fix", "after-base", backlog.StatusBacklog, "fix/base"),  // item-blocked
		bItem("fix", "deep", backlog.StatusBacklog, "fix/after-base"),  // wave 2
		bItem("fix", "loop-a", backlog.StatusBacklog, "fix/loop-b"),    // cycle
		bItem("fix", "loop-b", backlog.StatusBacklog, "fix/loop-a"),    // cycle
	}
	svc := newTestService(t, Config{
		Backlog: stubBacklog{items: items},
		Gates:   stubGates{gates: []Gate{decideGate("fix", "gated", 1, "fix/after-gate")}},
	})
	board, err := svc.Build(context.Background(), Params{})
	if err != nil {
		t.Fatal(err)
	}

	if len(board.Later.Groups) != 4 {
		t.Fatalf("expected 4 later groups, got %+v", board.Later.Groups)
	}

	// Gate-blocked group sorts first.
	first := board.Later.Groups[0]
	if first.BlockerKind != BlockerGate || first.GateID != "decide:backlog/fix/gated" {
		t.Errorf("expected gate-blocked group first, got %+v", first)
	}
	if got := cardIDs(first.Cards); len(got) != 1 || got[0] != "backlog-item/fix/after-gate" {
		t.Errorf("unexpected gate group cards: %v", got)
	}

	// Item-blocked groups next; cycle last.
	last := board.Later.Groups[len(board.Later.Groups)-1]
	if last.BlockerKind != BlockerCycle || len(last.Cards) != 2 {
		t.Errorf("expected cycle group last with 2 cards, got %+v", last)
	}
	for _, c := range last.Cards {
		if c.Wave != -1 {
			t.Errorf("cycle card should carry wave -1, got %+v", c)
		}
	}

	// deep sits behind after-base: its group is item-blocked on fix/after-base.
	var deepGroup *CardGroup
	for i := range board.Later.Groups {
		for _, c := range board.Later.Groups[i].Cards {
			if c.ID == "backlog-item/fix/deep" {
				deepGroup = &board.Later.Groups[i]
			}
		}
	}
	if deepGroup == nil || deepGroup.BlockerKind != BlockerItems {
		t.Fatalf("deep item missing or misgrouped: %+v", board.Later.Groups)
	}
	// Wave badge: after-base is wave 1, deep is wave 2.
	for _, c := range findGroup(t, board.Later, deepGroup.ID).Cards {
		if c.ID == "backlog-item/fix/deep" && c.Wave != 2 {
			t.Errorf("expected wave 2 on deep, got %d", c.Wave)
		}
	}

	if len(board.Meta.Cycles) != 1 {
		t.Errorf("expected 1 cycle diagnostic, got %v", board.Meta.Cycles)
	}
}

func TestBuild_LockedTerminalReviewItemsExcludedFromNextLater(t *testing.T) {
	items := []backlog.BacklogItem{
		bItem("fix", "running", backlog.StatusInProgress),
		bItem("fix", "queued", backlog.StatusQueued),
		bItem("fix", "reviewing", backlog.StatusInReview),
		bItem("fix", "waiting", backlog.StatusReviewPending),
		bItem("fix", "old-done", backlog.StatusCompleted),
	}
	// review_pending surfaces via its review gate only.
	reviewGate := Gate{
		ID: "review:backlog/fix/waiting", Kind: KindReview,
		OwnerType: "backlog", OwnerKind: "fix", OwnerName: "waiting", OwnerTitle: "waiting title", Count: 1,
	}
	svc := newTestService(t, Config{
		Backlog: stubBacklog{items: items},
		Gates:   stubGates{gates: []Gate{reviewGate}},
	})
	board, err := svc.Build(context.Background(), Params{})
	if err != nil {
		t.Fatal(err)
	}
	if board.Next.CardCount != 1 {
		t.Fatalf("expected only the review gate card, got %+v", board.Next.Groups)
	}
	card := findGroup(t, board.Next, "gates").Cards[0]
	if card.Action != ActionReview || card.ID != "backlog-item/fix/waiting" {
		t.Errorf("unexpected review card: %+v", card)
	}
	if board.Later.CardCount != 0 {
		t.Errorf("expected empty Later, got %+v", board.Later.Groups)
	}
}

func TestBuild_CompletedDepCollapsesWave(t *testing.T) {
	items := []backlog.BacklogItem{
		bItem("fix", "done-dep", backlog.StatusCompleted),
		bItem("fix", "now-free", backlog.StatusReady, "fix/done-dep"),
	}
	svc := newTestService(t, Config{
		Backlog: stubBacklog{items: items},
		Gates:   stubGates{},
	})
	board, err := svc.Build(context.Background(), Params{})
	if err != nil {
		t.Fatal(err)
	}
	ready := findGroup(t, board.Next, "ready")
	if len(ready.Cards) != 1 || ready.Cards[0].Wave != 0 {
		t.Errorf("completed dep should leave item at wave 0: %+v", ready.Cards)
	}
}

func TestBuild_DoneWindowAndOutcomes(t *testing.T) {
	inWindow := testNow.Add(-2 * time.Hour).Format(time.RFC3339)
	outOfWindow := testNow.Add(-30 * time.Hour).Format(time.RFC3339)

	items := []backlog.BacklogItem{
		{Kind: "fix", Name: "shipped", Title: "Shipped", Status: backlog.StatusCompleted, Updated: inWindow},
		{Kind: "fix", Name: "flopped", Title: "Flopped", Status: backlog.StatusFailed, Updated: inWindow},
		{Kind: "fix", Name: "ancient", Title: "Ancient", Status: backlog.StatusCompleted, Updated: outOfWindow},
	}
	execs := []execution.Record{
		{ExecutionID: "e-ok", BacklogKind: "fix", BacklogName: "shipped", Status: execution.StatusCompleted, FinishedAt: inWindow},
		{ExecutionID: "e-rev", BacklogKind: "fix", BacklogName: "flopped", Status: execution.StatusNeedsReview, FinishedAt: inWindow},
		{ExecutionID: "e-old", BacklogKind: "fix", BacklogName: "ancient", Status: execution.StatusFailed, FinishedAt: outOfWindow},
		{ExecutionID: "e-run", BacklogKind: "fix", BacklogName: "shipped", Status: execution.StatusRunning},
	}
	svc := newTestService(t, Config{
		Backlog:    stubBacklog{items: items},
		Gates:      stubGates{},
		Executions: stubExecs{records: execs},
	})
	board, err := svc.Build(context.Background(), Params{WindowSeconds: DefaultWindowSeconds})
	if err != nil {
		t.Fatal(err)
	}
	if board.Done.CardCount != 4 {
		t.Fatalf("expected 4 done cards (2 execs + 2 items in window), got %d: %+v",
			board.Done.CardCount, board.Done.Groups)
	}
	outcomes := map[string]string{}
	for _, c := range findGroup(t, board.Done, "recent").Cards {
		outcomes[c.ID] = c.Outcome
	}
	if outcomes["execution-record/e-ok"] != OutcomeOK ||
		outcomes["execution-record/e-rev"] != OutcomeNeedsReview ||
		outcomes["backlog-item/fix/shipped"] != OutcomeOK ||
		outcomes["backlog-item/fix/flopped"] != OutcomeFailed {
		t.Errorf("unexpected outcomes: %v", outcomes)
	}
}

func TestBuild_DoneCapped(t *testing.T) {
	ts := testNow.Add(-time.Hour).Format(time.RFC3339)
	var execs []execution.Record
	for i := 0; i < DoneCap+10; i++ {
		execs = append(execs, execution.Record{
			ExecutionID: fmt.Sprintf("e%d", i), BacklogKind: "fix", BacklogName: "x",
			Status: execution.StatusCompleted, FinishedAt: ts,
		})
	}
	svc := newTestService(t, Config{
		Backlog:    stubBacklog{},
		Gates:      stubGates{},
		Executions: stubExecs{records: execs},
	})
	board, err := svc.Build(context.Background(), Params{})
	if err != nil {
		t.Fatal(err)
	}
	if board.Done.CardCount != DoneCap {
		t.Errorf("expected done column capped at %d, got %d", DoneCap, board.Done.CardCount)
	}
}

func TestBuild_NowSummaryFromOps(t *testing.T) {
	view := &operations.OperationsView{
		Activities: []operations.ActivityRow{{ActivityID: "a1"}, {ActivityID: "a2"}},
		Queue:      operations.QueueStatus{Depth: 3, MaxDepth: 10},
		Lanes: []operations.LaneStatus{
			{Lane: "execute", Active: 2, Capacity: 5},
		},
	}
	svc := newTestService(t, Config{
		Backlog: stubBacklog{},
		Gates:   stubGates{},
		Ops:     stubOps{view: view},
	})
	board, err := svc.Build(context.Background(), Params{})
	if err != nil {
		t.Fatal(err)
	}
	if board.Now.ActiveCount != 2 || board.Now.QueueDepth != 3 || board.Now.MaxQueueDepth != 10 {
		t.Errorf("unexpected now summary: %+v", board.Now)
	}
	if len(board.Now.Lanes) != 1 || board.Now.Lanes[0].Lane != "execute" {
		t.Errorf("unexpected lanes: %+v", board.Now.Lanes)
	}
}

func TestBuild_OpsFailureDegradesNowToZero(t *testing.T) {
	svc := newTestService(t, Config{
		Backlog: stubBacklog{},
		Gates:   stubGates{},
		Ops:     stubOps{err: errors.New("boom")},
	})
	board, err := svc.Build(context.Background(), Params{})
	if err != nil {
		t.Fatal(err)
	}
	if board.Now.ActiveCount != 0 || len(board.Now.Lanes) != 0 {
		t.Errorf("expected degraded now summary, got %+v", board.Now)
	}
}

func TestBuild_ExecutionsFailureDegradesDone(t *testing.T) {
	svc := newTestService(t, Config{
		Backlog:    stubBacklog{},
		Gates:      stubGates{},
		Executions: stubExecs{err: errors.New("boom")},
	})
	board, err := svc.Build(context.Background(), Params{})
	if err != nil {
		t.Fatal(err)
	}
	if board.Done.CardCount != 0 {
		t.Errorf("expected empty done on executions failure, got %+v", board.Done)
	}
}

func TestBuild_BacklogFailureIsFatal(t *testing.T) {
	svc := newTestService(t, Config{
		Backlog: stubBacklog{err: errors.New("disk")},
		Gates:   stubGates{},
	})
	if _, err := svc.Build(context.Background(), Params{}); err == nil {
		t.Fatal("expected error when backlog load fails")
	}
}

func TestBuild_ExecutionReviewGateCard(t *testing.T) {
	items := []backlog.BacklogItem{
		bItem("fix", "flagged", backlog.StatusInProgress),
	}
	execGate := Gate{
		ID: "review:execution/e1", Kind: KindReview,
		OwnerType: "execution", OwnerKind: "fix", OwnerName: "flagged",
		OwnerTitle: "flagged title", Count: 1,
	}
	svc := newTestService(t, Config{
		Backlog: stubBacklog{items: items},
		Gates:   stubGates{gates: []Gate{execGate}},
	})
	board, err := svc.Build(context.Background(), Params{})
	if err != nil {
		t.Fatal(err)
	}
	card := findGroup(t, board.Next, "gates").Cards[0]
	if card.ID != "execution-record/e1" || card.ExecutionID != "e1" || card.Action != ActionReview {
		t.Errorf("unexpected execution gate card: %+v", card)
	}
}

func TestBuild_GateCardOrdering(t *testing.T) {
	items := []backlog.BacklogItem{
		bItem("fix", "d-small", backlog.StatusBacklog),
		bItem("fix", "d-big", backlog.StatusBacklog),
		bItem("fix", "r-item", backlog.StatusReviewPending),
		bItem("fix", "big-child-1", backlog.StatusBacklog, "fix/d-big"),
	}
	classifyGate := Gate{
		ID: "classify:capture/c1", Kind: KindClassify,
		OwnerType: "capture", OwnerName: "c1", OwnerTitle: "capture text", Count: 2,
	}
	reviewGate := Gate{
		ID: "review:backlog/fix/r-item", Kind: KindReview,
		OwnerType: "backlog", OwnerKind: "fix", OwnerName: "r-item", OwnerTitle: "r title", Count: 1,
	}
	svc := newTestService(t, Config{
		Backlog: stubBacklog{items: items},
		Gates: stubGates{gates: []Gate{
			classifyGate,
			reviewGate,
			decideGate("fix", "d-small", 1),
			decideGate("fix", "d-big", 1, "fix/big-child-1"),
		}},
	})
	board, err := svc.Build(context.Background(), Params{})
	if err != nil {
		t.Fatal(err)
	}
	cards := findGroup(t, board.Next, "gates").Cards
	if len(cards) != 4 {
		t.Fatalf("expected 4 gate cards, got %+v", cards)
	}
	// decide gates first (higher unblocks first), then review, then classify.
	if cards[0].Gate.OwnerName != "d-big" || cards[1].Gate.OwnerName != "d-small" {
		t.Errorf("decide ordering wrong: %v, %v", cards[0].Gate.OwnerName, cards[1].Gate.OwnerName)
	}
	if cards[2].Action != ActionReview || cards[3].Action != ActionClassify {
		t.Errorf("expected review then classify, got %+v", cardIDs(cards))
	}
	if cards[3].ID != "capture/c1" {
		t.Errorf("classify card id: %s", cards[3].ID)
	}
}

func TestClampWindow(t *testing.T) {
	tests := []struct{ in, want int }{
		{0, DefaultWindowSeconds},
		{-5, DefaultWindowSeconds},
		{30, MinWindowSeconds},
		{3600, 3600},
		{999999, MaxWindowSeconds},
	}
	for _, tt := range tests {
		if got := clampWindow(tt.in); got != tt.want {
			t.Errorf("clampWindow(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestNewService_RequiresBacklogAndGates(t *testing.T) {
	if _, err := NewService(Config{Gates: stubGates{}}); err == nil {
		t.Error("expected error without backlog")
	}
	if _, err := NewService(Config{Backlog: stubBacklog{}}); err == nil {
		t.Error("expected error without gates")
	}
}

func TestBuild_AttachesETABand(t *testing.T) {
	items := []backlog.BacklogItem{
		bItem("execute", "a", backlog.StatusReady),
		bItem("execute", "b", backlog.StatusBacklog, "execute/a"),
		bItem("execute", "c", backlog.StatusCompleted),
	}
	items[0].Effort = "M"
	items[1].Effort = "M"

	cfg := Config{
		Backlog: stubBacklog{items: items},
		Gates:   stubGates{},
		ETA: func() (*eta.Estimator, error) {
			return eta.NewEstimator(nil, nil, 2, eta.DefaultTrials, eta.DefaultSeed), nil
		},
	}
	svc := newTestService(t, cfg)
	board, err := svc.Build(context.Background(), Params{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if board.Meta.ETA == nil {
		t.Fatal("expected an ETA band in Meta")
	}
	// Two pending items (a, b); c is completed and excluded.
	if board.Meta.ETA.RemainingItems != 2 {
		t.Errorf("remaining = %d, want 2", board.Meta.ETA.RemainingItems)
	}
	if board.Meta.ETA.BasisLabel != "priors only" {
		t.Errorf("basis label = %q, want %q", board.Meta.ETA.BasisLabel, "priors only")
	}
	if board.Meta.ETA.P50Hours > board.Meta.ETA.P80Hours {
		t.Errorf("p50 %v must be <= p80 %v", board.Meta.ETA.P50Hours, board.Meta.ETA.P80Hours)
	}
}

func TestStatsRemainingETAEqualsPlanViewETA(t *testing.T) {
	items := []backlog.BacklogItem{
		{Kind: backlog.KindExecute, Name: "a", Status: backlog.StatusReady, Effort: "M"},
		{Kind: backlog.KindExecute, Name: "b", Status: backlog.StatusBacklog, DependsOn: []string{"execute/a"}, Effort: "L"},
		{Kind: backlog.KindExecute, Name: "done", Status: backlog.StatusCompleted, Effort: "S"},
	}
	estFactory := func() (*eta.Estimator, error) {
		return eta.NewEstimator(nil, nil, 2, eta.DefaultTrials, eta.DefaultSeed), nil
	}
	loader := stubBacklog{items: items}
	planSvc := newTestService(t, Config{
		Backlog: loader,
		Gates:   stubGates{},
		ETA:     estFactory,
	})
	board, err := planSvc.Build(context.Background(), Params{})
	if err != nil {
		t.Fatalf("plan build: %v", err)
	}

	statsEngine := stats.NewEngine(nil, stats.Config{Backlog: loader, ETA: estFactory})
	resp := statsEngine.GetStatsContext(context.Background())

	if board.Meta.ETA == nil {
		t.Fatal("plan ETA is nil")
	}
	if resp.Dashboard.EstimatedRemaining == nil {
		t.Fatal("stats ETA is nil")
	}
	if *resp.Dashboard.EstimatedRemaining != *board.Meta.ETA {
		t.Fatalf("stats ETA = %+v, plan ETA = %+v", *resp.Dashboard.EstimatedRemaining, *board.Meta.ETA)
	}
}

func TestStatsGoalRemainingETAEqualsPlanViewETA(t *testing.T) {
	items := []backlog.BacklogItem{
		{Kind: backlog.KindExecute, Name: "a", Status: backlog.StatusReady, Effort: "S"},
		{Kind: backlog.KindExecute, Name: "b", Status: backlog.StatusBacklog, DependsOn: []string{"execute/a"}, Effort: "XL"},
		{Kind: backlog.KindExecute, Name: "outside", Status: backlog.StatusReady, Effort: "XL"},
	}
	estFactory := func() (*eta.Estimator, error) {
		return eta.NewEstimator(nil, nil, 1, eta.DefaultTrials, eta.DefaultSeed), nil
	}
	loader := stubBacklog{items: items}
	goals := stubGoalScoper{name: "goal-x", closure: []string{"execute/a", "execute/b"}}
	planSvc := newTestService(t, Config{
		Backlog: loader,
		Gates:   stubGates{},
		Goals:   goals,
		ETA:     estFactory,
	})
	board, err := planSvc.Build(context.Background(), Params{Goal: "goal-x"})
	if err != nil {
		t.Fatalf("plan build: %v", err)
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	repo := eventlog.NewSQLiteRepository(database.NewFromPrimary(db))
	if err := repo.InitSchema(context.Background()); err != nil {
		t.Fatalf("init eventlog: %v", err)
	}
	statsEngine := stats.NewEngine(repo, stats.Config{
		Backlog: loader,
		Goals:   goals,
		ETA:     estFactory,
	})
	resp, err := statsEngine.GetStatsForParams(context.Background(), stats.Params{Goal: "goal-x"})
	if err != nil {
		t.Fatalf("stats build: %v", err)
	}

	if board.Meta.ETA == nil {
		t.Fatal("plan ETA is nil")
	}
	if resp.Dashboard.EstimatedRemaining == nil {
		t.Fatal("stats ETA is nil")
	}
	if *resp.Dashboard.EstimatedRemaining != *board.Meta.ETA {
		t.Fatalf("stats goal ETA = %+v, plan goal ETA = %+v", *resp.Dashboard.EstimatedRemaining, *board.Meta.ETA)
	}
}

func TestBuild_NoETAFactoryOmitsBand(t *testing.T) {
	svc := newTestService(t, Config{
		Backlog: stubBacklog{items: []backlog.BacklogItem{bItem("execute", "a", backlog.StatusReady)}},
		Gates:   stubGates{},
	})
	board, err := svc.Build(context.Background(), Params{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if board.Meta.ETA != nil {
		t.Error("expected no ETA band when no factory is configured")
	}
}

// stubGoalScoper resolves a fixed closure for a known goal name; any other name
// returns a not-found error so the ErrGoalScope path can be exercised.
type stubGoalScoper struct {
	name    string
	closure []string
}

func (s stubGoalScoper) ClosureRefs(name string) ([]string, error) {
	if name != s.name {
		return nil, fmt.Errorf("goal %q not found", name)
	}
	return append([]string(nil), s.closure...), nil
}

// allBoardCardIDs collects every card ID across Next, Later, and Done.
func allBoardCardIDs(b Board) map[string]bool {
	out := map[string]bool{}
	for _, col := range []Column{b.Next, b.Later, b.Done} {
		for _, g := range col.Groups {
			for _, id := range cardIDs(g.Cards) {
				out[id] = true
			}
		}
	}
	return out
}

func TestBuild_GoalScopeSubsetsToClosure(t *testing.T) {
	// a <- b <- c chain plus an unrelated item d. Goal closure = {a, b}.
	items := []backlog.BacklogItem{
		bItem("execute", "a", backlog.StatusReady),
		bItem("execute", "b", backlog.StatusReady, "execute/a"),
		bItem("execute", "c", backlog.StatusReady, "execute/b"),
		bItem("execute", "d", backlog.StatusReady),
	}
	cfg := Config{
		Backlog: stubBacklog{items: items},
		Gates:   stubGates{},
		Goals:   stubGoalScoper{name: "goal-x", closure: []string{"execute/a", "execute/b"}},
	}
	svc := newTestService(t, cfg)

	scoped, err := svc.Build(context.Background(), Params{Goal: "goal-x"})
	if err != nil {
		t.Fatalf("scoped Build: %v", err)
	}
	got := allBoardCardIDs(scoped)
	if !got["backlog-item/execute/a"] || !got["backlog-item/execute/b"] {
		t.Errorf("scoped board missing closure items: %v", got)
	}
	if got["backlog-item/execute/c"] || got["backlog-item/execute/d"] {
		t.Errorf("scoped board leaked out-of-closure items: %v", got)
	}
}

func TestBuild_NoGoalIdenticalToUnscoped(t *testing.T) {
	items := []backlog.BacklogItem{
		bItem("execute", "a", backlog.StatusReady),
		bItem("execute", "b", backlog.StatusReady, "execute/a"),
	}
	cfg := Config{
		Backlog: stubBacklog{items: items},
		Gates:   stubGates{},
		Goals:   stubGoalScoper{name: "goal-x", closure: []string{"execute/a"}},
	}
	svc := newTestService(t, cfg)

	// A goal scoper is wired, but no goal is requested: the projection must be
	// identical to a service with no scoper at all.
	withScoper, err := svc.Build(context.Background(), Params{})
	if err != nil {
		t.Fatalf("Build with idle scoper: %v", err)
	}
	plain := newTestService(t, Config{Backlog: stubBacklog{items: items}, Gates: stubGates{}})
	baseline, err := plain.Build(context.Background(), Params{})
	if err != nil {
		t.Fatalf("baseline Build: %v", err)
	}
	if a, b := allBoardCardIDs(withScoper), allBoardCardIDs(baseline); len(a) != len(b) {
		t.Fatalf("idle scoper changed the board: %v vs %v", a, b)
	}
}

func TestBuild_UnknownGoalReturnsGoalScopeError(t *testing.T) {
	svc := newTestService(t, Config{
		Backlog: stubBacklog{items: []backlog.BacklogItem{bItem("execute", "a", backlog.StatusReady)}},
		Gates:   stubGates{},
		Goals:   stubGoalScoper{name: "goal-x", closure: []string{"execute/a"}},
	})
	_, err := svc.Build(context.Background(), Params{Goal: "missing"})
	if !errors.Is(err, ErrGoalScope) {
		t.Fatalf("want ErrGoalScope, got %v", err)
	}
}

func TestBuild_GoalRequestedButNoScoperErrors(t *testing.T) {
	svc := newTestService(t, Config{
		Backlog: stubBacklog{items: []backlog.BacklogItem{bItem("execute", "a", backlog.StatusReady)}},
		Gates:   stubGates{},
	})
	_, err := svc.Build(context.Background(), Params{Goal: "goal-x"})
	if !errors.Is(err, ErrGoalScope) {
		t.Fatalf("want ErrGoalScope when no scoper, got %v", err)
	}
}

func TestAppendGoalActionsPlacesGoalDecisionBeforeBacklogGroups(t *testing.T) {
	next := Column{Groups: []CardGroup{{ID: "ready", Label: "Ready to run", Cards: []Card{{ID: "backlog-item/idea/ready"}}}}, CardCount: 1}
	got := appendGoalActions(next, []GoalAction{{Name: "portfolio", Title: "Portfolio", Action: ActionDecide, Priority: 9}})
	if len(got.Groups) != 2 || got.Groups[0].ID != "goals" || got.CardCount != 2 {
		t.Fatalf("goal group = %#v", got)
	}
	card := got.Groups[0].Cards[0]
	if card.ID != "goal/portfolio" || card.Gate == nil || card.Gate.OwnerType != "goal" || card.Gate.Kind != KindDecide || card.Action != ActionDecide {
		t.Fatalf("goal card = %#v", card)
	}
}
