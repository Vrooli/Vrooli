package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/operatingmode"

	"github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func TestOperatingModeBacklogMutatorEmitsAuditedStatusMetadata(t *testing.T) {
	root := t.TempDir()
	store := backlog.NewFileStore(root)
	if err := os.MkdirAll(store.ItemDir(backlog.KindExecute, "do-thing"), 0o755); err != nil {
		t.Fatalf("mkdir item: %v", err)
	}
	if err := store.SaveItem(backlog.BacklogItem{
		Kind:     backlog.KindExecute,
		Name:     "do-thing",
		Title:    "Do thing",
		Status:   backlog.StatusReady,
		Priority: 5,
	}); err != nil {
		t.Fatalf("save item: %v", err)
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	repo := eventlog.NewSQLiteRepository(database.NewFromPrimary(db))
	if err := repo.InitSchema(context.Background()); err != nil {
		t.Fatalf("init event schema: %v", err)
	}

	mutator := operatingModeBacklogMutator{
		store:  store,
		events: eventlog.NewEmitter(repo),
	}
	result, err := mutator.MarkBacklogItemCompleted(context.Background(), "execute", "do-thing", operatingmode.BacklogMutationSource{
		Entrypoint:     "initiative.operating_mode.complete_items",
		InitiativeName: "init-a",
		Mode:           "holistic-loop",
		Phase:          "execute",
		Round:          3,
		RunID:          "run-123",
		RequestedBy:    "operator",
	})
	if err != nil {
		t.Fatalf("mark completed: %v", err)
	}
	if result.ItemRef != "execute/do-thing" || result.FromStatus != string(backlog.StatusReady) || result.ToStatus != string(backlog.StatusCompleted) {
		t.Fatalf("completion result = %+v", result)
	}

	events, err := repo.All(context.Background())
	if err != nil {
		t.Fatalf("load events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	event := events[0]
	if event.EventType != eventlog.EventBacklogStatusChanged || event.EntityID != "execute/do-thing" {
		t.Fatalf("event = %+v", event)
	}

	var payload eventlog.StatusChangePayload
	if err := json.Unmarshal(event.Metadata, &payload); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if payload.Source == nil {
		t.Fatalf("source missing from payload: %+v", payload)
	}
	if payload.Source.Mode != "holistic-loop" || payload.Source.Phase != "execute" || payload.Source.Round != 3 || payload.Source.RunID != "run-123" || payload.Source.RequestedBy != "operator" {
		t.Fatalf("source payload = %+v", payload.Source)
	}
	if len(payload.ItemRefs) != 1 || payload.ItemRefs[0] != "execute/do-thing" {
		t.Fatalf("item refs = %+v", payload.ItemRefs)
	}
}

// TestOperatingModeBacklogTargetReaderProjectsFullItem locks the production
// BacklogItemTargetReader wiring contract: the backlog-item target must carry
// the item's description, canonical spec.json document, plan_ref, and
// write-scope containment. This reader existing but being UNWIRED was a live
// defect (empty ITEM_SPEC/ITEM_DESCRIPTION/ITEM_PLAN_REF inputs and an
// unconstrained sandbox for every backlog-item operation), so the projection
// is pinned here against the real file store.
// [REQ:REQ-P0-011-TARGET-CAPABILITIES]
func TestOperatingModeBacklogTargetReaderProjectsFullItem(t *testing.T) {
	root := t.TempDir()
	store := backlog.NewFileStore(root)
	if err := os.MkdirAll(store.ItemDir(backlog.KindFix, "flaky-test"), 0o755); err != nil {
		t.Fatalf("mkdir item: %v", err)
	}
	if err := store.SaveItem(backlog.BacklogItem{
		Kind:            backlog.KindFix,
		Name:            "flaky-test",
		Title:           "Fix flaky test",
		Description:     "It flakes on CI",
		Status:          backlog.StatusReady,
		Priority:        5,
		AcceptanceAllow: []string{"scenarios/foo/**"},
		AcceptanceDeny:  []string{"scenarios/foo/secrets/**"},
		Creates:         []string{"scenarios/foo/newdir"},
		PlanRef:         &backlog.PlanRef{Provider: "plan-manager", PlanID: "p-1", Slug: "fix-flaky", Role: backlog.PlanRefRoleExecutionSpec},
	}); err != nil {
		t.Fatalf("save item: %v", err)
	}

	reader := operatingModeBacklogTargetReader{store: store}
	bt, err := reader.LoadBacklogItemTarget("fix/flaky-test")
	if err != nil {
		t.Fatalf("load target: %v", err)
	}
	if bt.Title != "Fix flaky test" || bt.Description != "It flakes on CI" || bt.Status != string(backlog.StatusReady) {
		t.Fatalf("target projection = %+v", bt)
	}
	if bt.SpecDocument == "" {
		t.Fatalf("spec document must carry the canonical spec.json bytes")
	}
	if bt.PlanRef == nil || bt.PlanRef.PlanID != "p-1" || bt.PlanRef.Role != backlog.PlanRefRoleExecutionSpec {
		t.Fatalf("plan_ref projection = %+v", bt.PlanRef)
	}
	if len(bt.Containment.AcceptanceAllow) != 1 || bt.Containment.AcceptanceAllow[0] != "scenarios/foo/**" ||
		len(bt.Containment.AcceptanceDeny) != 1 || len(bt.Containment.Creates) != 1 {
		t.Fatalf("containment projection = %+v", bt.Containment)
	}

	if _, err := reader.LoadBacklogItemTarget("not-a-ref"); err == nil {
		t.Fatalf("malformed ref must fail")
	}
}
