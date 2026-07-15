// DOC: docs/internal/SEAMS.md#clarification-api
// DOC: docs/reference/api-endpoints.md#clarification
//
// Domain logic helpers for clarification threads: attachment storage, round
// persistence, and proto conversion.
package backlog

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"

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
	if err := os.MkdirAll(attDir, 0o750); err != nil {
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
			closeClarificationFile(src, "backlog: close clarification source")
			return nil, fmt.Errorf("create attachment file: %w", err)
		}
		_, copyErr := io.Copy(dst, src)
		closeClarificationFile(src, "backlog: close clarification source")
		closeClarificationFile(dst, "backlog: close clarification dest")
		if copyErr != nil {
			return nil, fmt.Errorf("write attachment file: %w", copyErr)
		}
		ids = append(ids, filepath.Join("workshop", "attachments", destName))
	}
	return ids, nil
}

// saveRound writes a round back to disk.
func (h *Handler) saveRound(itemDir string, round *workshop.Round) {
	data, err := json.MarshalIndent(round, "", "  ")
	if err != nil {
		slog.Error("saveRound marshal error", "err", err)
		return
	}
	path := filepath.Join(itemDir, "workshop", fmt.Sprintf("round-%03d.json", round.RoundNum))
	if err := os.WriteFile(path, data, 0o600); err != nil {
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

func closeClarificationFile(c io.Closer, msg string) {
	if err := c.Close(); err != nil {
		slog.Debug(msg, "err", err)
	}
}
