// Convert handler for moving backlog items between kinds (e.g. research -> execute).
package backlog

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/httputil"
)

// Convert moves a backlog item to another kind.
func (h *Handler) Convert(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "convert")
	if !ok {
		return
	}

	var req apipb.ConvertBacklogItemRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		httputil.BadRequest(w, "[backlog] convert", "invalid request body")
		return
	}
	req.TargetKind = strings.ToLower(strings.TrimSpace(req.TargetKind))
	if strings.TrimSpace(req.TargetKind) == "" {
		httputil.BadRequest(w, "[backlog] convert", "target_kind is required")
		return
	}
	if !httputil.ValidateProtoRequest(w, "[backlog] convert", "invalid request body", &req) {
		return
	}

	targetKind, err := ParseBacklogKind(req.TargetKind)
	if err != nil {
		httputil.BadRequest(w, "[backlog] convert", err.Error())
		return
	}

	targetName := name
	if req.TargetName != nil {
		candidate := strings.TrimSpace(*req.TargetName)
		if candidate == "" {
			httputil.BadRequest(w, "[backlog] convert", "target_name is invalid")
			return
		}
		targetName = candidate
	}
	targetName = sanitizeName(targetName)
	if targetName == "" {
		httputil.BadRequest(w, "[backlog] convert", "target_name is invalid")
		return
	}

	sourceDir := h.store.ItemDir(kind, name)
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		httputil.NotFound(w, "[backlog] convert", "backlog item not found")
		return
	}

	targetDir := h.store.ItemDir(targetKind, targetName)
	if _, err := os.Stat(targetDir); err == nil {
		httputil.Conflict(w, "[backlog] convert", "target backlog item already exists")
		return
	}

	if err := os.MkdirAll(h.store.KindDir(targetKind), 0o755); err != nil {
		log.Printf("[backlog] convert: failed to create target dir %s: %v", targetDir, err)
		httputil.InternalError(w, "[backlog] convert", "failed to create target backlog directory")
		return
	}

	if err := os.Rename(sourceDir, targetDir); err != nil {
		log.Printf("[backlog] convert: failed to move %s to %s: %v", sourceDir, targetDir, err)
		httputil.InternalError(w, "[backlog] convert", "failed to move backlog item")
		return
	}

	item, err := h.store.LoadItem(targetKind, targetName)
	if err != nil {
		httputil.InternalError(w, "[backlog] convert", "failed to load moved backlog item")
		return
	}
	item.Name = targetName
	item.Kind = targetKind
	item.Updated = time.Now().UTC().Format(time.RFC3339)
	if item.Kind != KindResearch {
		item.ResearchTarget = ""
	}

	if err := h.store.SaveItem(item); err != nil {
		log.Printf("[backlog] convert: failed to save %q: %v", targetName, err)
		httputil.InternalError(w, "[backlog] convert", "failed to update backlog item")
		return
	}

	resp := &apipb.BacklogItemResponse{Item: backlogToProto(item)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[backlog] convert", "failed to encode response")
	}
}
