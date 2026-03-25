// DOC: docs/internal/SEAMS.md#workshop-auto-trigger
// DOC: docs/reference/api-endpoints.md#workshop-save
//
// Dedicated workshop save endpoint and shared async spawn helper.
// The WorkshopSave handler saves round responses and auto-triggers the next
// round when the item is not yet ready. The spawnWorkshopAsync helper is
// reused by both WorkshopSave (auto-advance) and Create (auto-initialize).
package backlog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/workshop"
)

// workshopLockFile is the filename used for idempotency locking.
const workshopLockFile = ".workshop-lock"

// workshopLockTTL is how long a stale lock file is considered valid.
const workshopLockTTL = 30 * time.Minute

// WorkshopSave saves a workshop round's responses and optionally auto-triggers
// the next round if the item is not yet ready.
func (h *Handler) WorkshopSave(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "workshop-save")
	if !ok {
		return
	}

	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.NotFound(w, "[backlog] workshop-save", "backlog item not found")
			return
		}
		httputil.InternalError(w, "[backlog] workshop-save", "failed to load backlog item")
		return
	}

	var req apipb.WorkshopSaveRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		httputil.BadRequest(w, "[backlog] workshop-save", "invalid request body")
		return
	}
	if !httputil.ValidateProtoRequest(w, "[backlog] workshop-save", "invalid request body", &req) {
		return
	}

	// Parse and validate the round content.
	var round workshop.Round
	if err := json.Unmarshal([]byte(req.Content), &round); err != nil {
		httputil.BadRequest(w, "[backlog] workshop-save", "content is not valid workshop round JSON")
		return
	}

	// Write the round file.
	itemDir := h.store.ItemDir(kind, name)
	workshopDir := filepath.Join(itemDir, "workshop")
	if err := os.MkdirAll(workshopDir, 0o755); err != nil {
		log.Printf("[backlog] workshop-save: failed to create workshop dir: %v", err)
		httputil.InternalError(w, "[backlog] workshop-save", "failed to create workshop directory")
		return
	}

	roundFile := fmt.Sprintf("round-%03d.json", req.RoundNumber)
	roundPath := filepath.Join(workshopDir, roundFile)
	if err := os.WriteFile(roundPath, []byte(req.Content), 0o644); err != nil {
		log.Printf("[backlog] workshop-save: failed to write %s: %v", roundPath, err)
		httputil.InternalError(w, "[backlog] workshop-save", "failed to save round file")
		return
	}

	info, _ := os.Stat(roundPath)
	var fileSize int64
	if info != nil {
		fileSize = info.Size()
	}
	fileNode := backlogFileToProto(BacklogFile{
		Name: roundFile,
		Path: filepath.Join("workshop", roundFile),
		Type: "file",
		Size: fileSize,
	})

	log.Printf("[backlog] workshop-save: saved %s/%s %s (%d bytes)", kind, name, roundFile, fileSize)

	// Determine auto-advance.
	autoAdvance := &apipb.WorkshopAutoAdvance{Triggered: false, Reason: "opt_out"}

	optOut := req.AutoWorkshop != nil && !*req.AutoWorkshop
	if !optOut {
		// Load rounds to get the accurate count after save.
		_, roundCount, loadErr := workshop.LoadLatestRound(itemDir)
		if loadErr != nil {
			log.Printf("[backlog] workshop-save: failed to load rounds for auto-advance check: %v", loadErr)
			autoAdvance.Reason = "error"
		} else {
			result := workshop.ShouldAutoAdvance(&round, roundCount, string(kind))
			autoAdvance.Reason = result.Reason
			if result.Advance {
				runID, taskID, spawnErr := h.spawnWorkshopAsync(item, ResearchModeWorkshop)
				if spawnErr != nil {
					log.Printf("[backlog] workshop-save: auto-advance spawn failed for %s/%s: %v", kind, name, spawnErr)
					autoAdvance.Reason = "error"
				} else {
					autoAdvance.Triggered = true
					autoAdvance.RunId = &runID
					autoAdvance.TaskId = &taskID
				}
			}
		}
	}

	resp := &apipb.WorkshopSaveResponse{
		File:        fileNode,
		AutoAdvance: autoAdvance,
	}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[backlog] workshop-save", "failed to encode response")
	}
}

// spawnWorkshopAsync spawns a workshop/initialize agent for the given item.
// It acquires an idempotency lock, fetches the prompt, and spawns via
// agent-manager. Returns run/task IDs on success.
func (h *Handler) spawnWorkshopAsync(item BacklogItem, mode ResearchMode) (runID, taskID string, err error) {
	itemDir := h.store.ItemDir(item.Kind, item.Name)

	// Idempotency lock.
	release, acquired := tryAcquireWorkshopLock(itemDir)
	if !acquired {
		return "", "", fmt.Errorf("workshop lock held for %s/%s", item.Kind, item.Name)
	}
	// Release lock after spawn completes (or fails).
	defer release()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	service := h.agentService
	if service == nil {
		service = agentmanager.NewAgentService(agentmanager.DefaultServiceConfig())
		h.agentService = service
	}

	selection, promptErr := h.fetchResearchPrompt(ctx, item, mode)
	prompt := selection.Prompt
	if promptErr != nil {
		log.Printf("[backlog] auto-workshop: prompt fetch failed for %s/%s: %v", item.Kind, item.Name, promptErr)
		prompt = "Use the backlog item folder as context and perform the requested workshop refinement."
	}

	runResult, err := service.SpawnBacklog(ctx, agentmanager.BacklogSpawnRequest{
		Kind:        string(item.Kind),
		Name:        item.Name,
		Title:       buildResearchTitle(item, mode),
		Description: prompt,
		Prompt:      prompt,
		ScopePath:   ".",
		ProjectRoot: ".",
		CreatedBy:   "swarm-manager",
		Purpose:     "research",
	})
	if err != nil {
		return "", "", fmt.Errorf("spawn failed: %w", err)
	}

	return runResult.RunID, runResult.TaskID, nil
}

// tryAcquireWorkshopLock attempts to create a lock file atomically.
// Returns a release function and true if acquired, or nil and false if
// the lock is already held (and not stale).
func tryAcquireWorkshopLock(itemDir string) (release func(), ok bool) {
	lockPath := filepath.Join(itemDir, workshopLockFile)

	// Clean stale locks.
	if info, err := os.Stat(lockPath); err == nil {
		if time.Since(info.ModTime()) > workshopLockTTL {
			_ = os.Remove(lockPath)
		}
	}

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, false
	}
	_, _ = f.WriteString(time.Now().UTC().Format(time.RFC3339))
	f.Close()

	return func() { _ = os.Remove(lockPath) }, true
}
