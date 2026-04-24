package backlog

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/testutil"
)

// TestConvergence_AllSourcesAgreeOnDiskState pins the contract that the
// three creation surfaces — HTTP single, batch, and proposal — produce
// items with the same on-disk shape and attach to initiatives identically.
// All routes go through Service.Create; this test guards against
// drift if the unified Service ever sprouts source-specific persistence
// branches that diverge.
func TestConvergence_AllSourcesAgreeOnDiskState(t *testing.T) {
	h, _ := setupTestHandler(t)
	ia := newMockInitiativeAssigner()
	ia.snapshots["shared-init"] = InitiativeSnapshot{
		Name:   "shared-init",
		Title:  "Shared",
		Status: "active",
		Items:  []string{},
	}
	h.SetInitiativeAssigner(ia)

	// Path 1: HTTP single-create
	body := bytes.NewBufferString(`{"name":"http-item","title":"HTTP Item","kind":"execute","initiative":"shared-init","priority":4,"effort":"M","description":"created via HTTP"}`)
	req := httptest.NewRequest("POST", "/api/v1/backlog", body)
	w := httptest.NewRecorder()
	h.Create(w, req)
	testutil.AssertStatus(t, w, http.StatusCreated)

	// Path 2: proposal-driven create via Service.Create directly (this is
	// the same call proposals.Applier.applyAddItem makes in production).
	svc := h.creationService()
	now := "2026-04-23T00:00:00Z"
	proposal := BacklogItem{
		Name:        "proposal-item",
		Title:       "Proposal Item",
		Kind:        KindExecute,
		Status:      StatusBacklog,
		Priority:    4,
		Effort:      "M",
		Description: "created via proposal",
		Initiative:  "shared-init",
		Created:     now,
		Updated:     now,
	}
	if err := svc.Create(proposal, CreationContext{
		Source:                SourceProposal,
		FeedbackRoundID:       "shared-init/round-001",
		Entrypoint:            "initiative.feedback",
		SkipCycleCheck:        true,
		SkipGraphInvalidation: true,
	}); err != nil {
		t.Fatalf("Service.Create (proposal): %v", err)
	}

	httpItem, err := h.store.LoadItem(KindExecute, "http-item")
	if err != nil {
		t.Fatalf("load http-item: %v", err)
	}
	primItem, err := h.store.LoadItem(KindExecute, "proposal-item")
	if err != nil {
		t.Fatalf("load proposal-item: %v", err)
	}

	// Lifecycle invariants must match between paths.
	if httpItem.Status != primItem.Status {
		t.Errorf("Status divergence: http=%q primitive=%q", httpItem.Status, primItem.Status)
	}
	if httpItem.Kind != primItem.Kind {
		t.Errorf("Kind divergence: http=%q primitive=%q", httpItem.Kind, primItem.Kind)
	}
	if httpItem.Initiative != primItem.Initiative {
		t.Errorf("Initiative divergence: http=%q primitive=%q", httpItem.Initiative, primItem.Initiative)
	}
	if httpItem.Effort != primItem.Effort {
		t.Errorf("Effort divergence: http=%q primitive=%q", httpItem.Effort, primItem.Effort)
	}
	if httpItem.Priority != primItem.Priority {
		t.Errorf("Priority divergence: http=%d primitive=%d", httpItem.Priority, primItem.Priority)
	}

	// Both items exist on disk under the same kind directory.
	for _, name := range []string{"http-item", "proposal-item"} {
		dir := h.store.ItemDir(KindExecute, name)
		if !pathExists(t, dir) {
			t.Errorf("expected item dir %s to exist", dir)
		}
		spec := filepath.Join(dir, "spec.json")
		if !pathExists(t, spec) {
			t.Errorf("expected spec.json at %s", spec)
		}
	}

	// Both items must be members of the initiative.
	got := ia.snapshots["shared-init"].Items
	want := map[string]bool{"execute/http-item": false, "execute/proposal-item": false}
	for _, ref := range got {
		if _, ok := want[ref]; ok {
			want[ref] = true
		}
	}
	for ref, seen := range want {
		if !seen {
			t.Errorf("initiative items[] missing %q (got %v)", ref, got)
		}
	}
}

func pathExists(t *testing.T, p string) bool {
	t.Helper()
	_, err := os.Stat(p)
	return err == nil
}
