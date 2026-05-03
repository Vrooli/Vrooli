// DOC: docs/internal/SEAMS.md#clarification-api
// DOC: docs/reference/api-endpoints.md#clarification
//
// Domain logic helpers for clarification threads: prompt building, attachment
// storage, round persistence, agent spawning, and proto conversion.
package backlog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/promptcatalog"
	"swarm-manager/internal/workshop"

	"github.com/google/uuid"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
)

// clarificationAllowedImageTypes lists Content-Types accepted for clarification attachments.
var clarificationAllowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// saveClarificationAttachments saves uploaded files from a multipart request
// and returns their attachment IDs (relative paths). Follows the same pattern
// as capture attachment storage.
func (h *Handler) saveClarificationAttachments(itemDir string, r *http.Request) ([]string, error) {
	if r.MultipartForm == nil {
		return nil, nil
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		return nil, nil
	}

	attDir := filepath.Join(itemDir, "workshop", "attachments")
	if err := os.MkdirAll(attDir, 0o755); err != nil {
		return nil, fmt.Errorf("create attachment dir: %w", err)
	}

	var ids []string
	for _, fh := range files {
		mediaType, _, _ := mime.ParseMediaType(fh.Header.Get("Content-Type"))
		if !clarificationAllowedImageTypes[mediaType] {
			return nil, fmt.Errorf("unsupported file type: %s", mediaType)
		}

		attID := uuid.New().String()
		ext := filepath.Ext(fh.Filename)
		destName := attID + ext
		destPath := filepath.Join(attDir, destName)

		src, err := fh.Open()
		if err != nil {
			return nil, fmt.Errorf("open uploaded file: %w", err)
		}
		dst, err := os.Create(destPath)
		if err != nil {
			src.Close()
			return nil, fmt.Errorf("create attachment file: %w", err)
		}
		_, copyErr := io.Copy(dst, src)
		src.Close()
		dst.Close()
		if copyErr != nil {
			return nil, fmt.Errorf("write attachment file: %w", copyErr)
		}
		ids = append(ids, filepath.Join("workshop", "attachments", destName))
	}
	return ids, nil
}

// buildClarificationPrompt constructs the prompt for a clarification agent
// by fetching the skill from prompt-manager.
func (h *Handler) buildClarificationPrompt(
	ctx context.Context,
	item BacklogItem,
	itemDir string,
	decision *workshop.Item,
	userQuestion string,
	priorMessages []workshop.ClarificationMessage,
) (string, error) {
	entry, ok := promptcatalog.ResolveBacklogSkill("clarify", string(item.Kind))
	if !ok {
		return "", fmt.Errorf("no prompt catalog entry for mode=clarify kind=%s", item.Kind)
	}

	vars := buildVariableMap(item, itemDir)
	vars["DECISION_TOPIC"] = decision.Topic
	vars["DECISION_CONTEXT"] = decision.Context
	vars["DECISION_OPTIONS"] = workshop.FormatOptionsForPrompt(decision.Options)
	vars["USER_QUESTION"] = userQuestion
	vars["CLARIFICATION_HISTORY"] = workshop.FormatClarificationHistory(priorMessages)

	prompt, err := h.promptClient.ReadSkill(ctx, entry.SkillID, vars, false)
	if err != nil {
		return "", fmt.Errorf("prompt-manager read: %w", err)
	}
	return prompt, nil
}

// saveRound writes a round back to disk.
func (h *Handler) saveRound(itemDir string, round *workshop.Round) {
	data, err := json.MarshalIndent(round, "", "  ")
	if err != nil {
		slog.Error("saveRound marshal error", "err", err)
		return
	}
	path := filepath.Join(itemDir, "workshop", fmt.Sprintf("round-%03d.json", round.RoundNum))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		slog.Error("saveRound write error", "err", err)
	}
}

// saveRoundItem links a clarification to a decision item and saves the round.
func (h *Handler) saveRoundItem(itemDir string, round *workshop.Round, item *workshop.Item) error {
	for i := range round.Items {
		if round.Items[i].ID == item.ID {
			round.Items[i] = *item
			break
		}
	}
	h.saveRound(itemDir, round)
	return nil
}

// spawnWorkshopForClarification spawns a new workshop round after invalidation.
func (h *Handler) spawnWorkshopForClarification(
	ctx context.Context,
	kind BacklogKind,
	item BacklogItem,
	itemDir string,
) (agentmanager.RunResult, error) {
	selection, err := h.fetchResearchPrompt(ctx, item, ResearchModeWorkshop)
	if err != nil {
		return agentmanager.RunResult{}, fmt.Errorf("fetch workshop prompt: %w", err)
	}

	activityCtx := agentactivity.WithSpec(ctx, agentactivity.Spec{
		OwnerType:   agentactivity.OwnerBacklog,
		OwnerKind:   string(kind),
		OwnerName:   item.Name,
		OwnerTitle:  item.Title,
		Purpose:     agentactivity.PurposeWorkshop,
		PhaseKind:   string(agentactivity.LaneInvestigate),
		RequestedBy: "swarm-manager",
		Metadata: map[string]string{
			"entrypoint": "backlog.clarification_workshop_respawn",
		},
	})

	return h.agentService.SpawnBacklog(activityCtx, agentmanager.BacklogSpawnRequest{
		Kind:        string(kind),
		Name:        item.Name,
		Title:       fmt.Sprintf("Workshop: %s (re-run after clarification)", item.Title),
		Description: selection.Prompt,
		Prompt:      selection.Prompt,
		ScopePath:   ".",
		ProjectRoot: ".",
		CreatedBy:   "swarm-manager",
		Purpose:     "research",
		Environment: map[string]string{
			"VROOLI_SPAWN_SOURCE": string(kind) + "/" + item.Name,
		},
	})
}

// lastAssistantMessage returns the most recent assistant message in a thread.
func lastAssistantMessage(thread *workshop.ClarificationThread) *workshop.ClarificationMessage {
	for i := len(thread.Messages) - 1; i >= 0; i-- {
		if thread.Messages[i].Role == "assistant" {
			return &thread.Messages[i]
		}
	}
	return nil
}

// isTerminalStatus checks if a run status indicates completion.
func isTerminalStatus(status string) bool {
	switch strings.ToLower(status) {
	case "complete", "completed", "success", "failed", "error", "cancelled", "canceled":
		return true
	}
	return false
}

// clarificationThreadToProto converts a workshop.ClarificationThread to its
// proto representation. The domain types (ClarificationThread, etc.) live in
// the domain proto package; the API response messages reference them.
func clarificationThreadToProto(t *workshop.ClarificationThread) *domainpb.ClarificationThread {
	if t == nil {
		return nil
	}
	pb := &domainpb.ClarificationThread{
		Id:          t.ID,
		RoundNumber: int32(t.RoundNumber),
		ItemId:      t.ItemID,
		RunId:       t.RunID,
		Status:      t.Status,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
	for _, msg := range t.Messages {
		pb.Messages = append(pb.Messages, &domainpb.ClarificationMessage{
			Role:          msg.Role,
			Content:       msg.Content,
			CreatedAt:     msg.CreatedAt,
			AttachmentIds: msg.AttachmentIDs,
		})
	}
	if t.LatestImpact != nil {
		pb.LatestImpact = &domainpb.ClarificationImpact{
			Level:           t.LatestImpact.Level,
			Reasoning:       t.LatestImpact.Reasoning,
			ContextNote:     t.LatestImpact.ContextNote,
			SuggestedUpdate: t.LatestImpact.SuggestedUpdate,
		}
	}
	return pb
}
