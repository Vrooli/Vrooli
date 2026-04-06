// DOC: docs/internal/SEAMS.md#workshop-auto-trigger
//
// Background ticker that fires deferred auto-advance spawns when their
// countdown expires. Maintains an in-memory registry of pending advances
// and checks every 2 seconds for due items.
package backlog

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const workshopTickerInterval = 2 * time.Second

// WorkshopTicker polls for pending advance files and spawns agents when due.
type WorkshopTicker struct {
	handler *Handler
	pending sync.Map // key: "kind/name" → PendingAdvance
	ctx     context.Context
	cancel  context.CancelFunc
}

// newWorkshopTicker creates a new ticker bound to the given handler.
func newWorkshopTicker(handler *Handler) *WorkshopTicker {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkshopTicker{
		handler: handler,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Register adds a pending advance to the in-memory registry.
func (t *WorkshopTicker) Register(kind, name string, pa PendingAdvance) {
	t.pending.Store(kind+"/"+name, pa)
}

// Unregister removes a pending advance from the in-memory registry.
func (t *WorkshopTicker) Unregister(kind, name string) {
	t.pending.Delete(kind + "/" + name)
}

// Start begins the background polling loop. Call Stop() to terminate.
func (t *WorkshopTicker) Start() {
	go t.run()
}

// Stop terminates the background ticker.
func (t *WorkshopTicker) Stop() {
	t.cancel()
}

// RecoverPending scans all backlog item directories for leftover pending
// advance files (from a server crash) and re-registers them.
func (t *WorkshopTicker) RecoverPending() {
	store := t.handler.store
	for _, kind := range []BacklogKind{KindIdea, KindFix, KindExecute, KindChore, KindResearch} {
		kindDir := store.KindDir(kind)
		entries, err := os.ReadDir(kindDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			itemDir := filepath.Join(kindDir, entry.Name())
			pa, err := readPendingAdvance(itemDir)
			if err != nil || pa == nil {
				continue
			}
			t.Register(pa.Kind, pa.Name, *pa)
			slog.Info("recovered pending advance", "kind", pa.Kind, "name", pa.Name, "advance_at", pa.AdvanceAt)
		}
	}
}

func (t *WorkshopTicker) run() {
	ticker := time.NewTicker(workshopTickerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			t.checkPending()
		}
	}
}

func (t *WorkshopTicker) checkPending() {
	now := time.Now()
	t.pending.Range(func(key, value any) bool {
		pa, ok := value.(PendingAdvance)
		if !ok {
			t.pending.Delete(key)
			return true
		}
		if now.Before(pa.AdvanceAt) {
			return true
		}

		keyStr := key.(string)
		parts := strings.SplitN(keyStr, "/", 2)
		if len(parts) != 2 {
			t.pending.Delete(key)
			return true
		}
		kind, name := BacklogKind(parts[0]), parts[1]

		// Remove from registry immediately to prevent re-fires.
		t.pending.Delete(key)

		// Load the item and spawn.
		item, err := t.handler.store.LoadItem(kind, name)
		if err != nil {
			slog.Error("ticker: failed to load item for pending advance", "kind", kind, "name", name, "err", err)
			deletePendingAdvance(t.handler.store.ItemDir(kind, name))
			return true
		}

		runMode := ResearchModeWorkshop
		if pa.NextMode == string(ResearchModeFinalize) {
			runMode = ResearchModeFinalize
		}

		runID, taskID, spawnErr := t.handler.spawnWorkshopAsync(item, runMode)
		itemDir := t.handler.store.ItemDir(kind, name)
		deletePendingAdvance(itemDir)

		if spawnErr != nil {
			slog.Error("ticker: pending advance spawn failed", "kind", kind, "name", name, "err", spawnErr)
		} else {
			slog.Info("ticker: pending advance spawned", "kind", kind, "name", name, "run_id", runID, "task_id", taskID)
		}

		return true
	})
}
