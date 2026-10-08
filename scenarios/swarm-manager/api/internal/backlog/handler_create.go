package backlog

import (
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/identity"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

func validateCreateBacklogItemRequest(req *apipb.CreateBacklogItemRequest) string {
	if strings.TrimSpace(req.Title) == "" {
		return "title is required"
	}
	if strings.TrimSpace(req.Kind) == "" {
		return "kind is required"
	}
	if req.Priority != nil {
		if *req.Priority < 1 || *req.Priority > 10 {
			return "priority must be between 1 and 10"
		}
	}
	return ""
}

func normalizeCreateBacklogItemRequest(req *apipb.CreateBacklogItemRequest) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		req.Name = strings.TrimSpace(req.Title)
	}
	req.Kind = strings.ToLower(strings.TrimSpace(req.Kind))
	if req.Effort != nil {
		normalized := strings.ToUpper(strings.TrimSpace(*req.Effort))
		if normalized == "" {
			req.Effort = nil
		} else {
			req.Effort = &normalized
		}
	}
}

// Create creates a new backlog item.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if strings.EqualFold(mediaType, "multipart/form-data") {
		h.createMultipart(w, r)
		return
	}
	if mediaType != "" && !strings.EqualFold(mediaType, "application/json") {
		apierr.MapError(w, "[backlog] create", apierr.Wrap(apierr.ErrBadRequest, http.StatusUnsupportedMediaType, "unsupported content type"))
		return
	}

	var req apipb.CreateBacklogItemRequest
	if err := httputil.DecodeProtoJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[backlog] create", apierr.BadRequest("invalid request body"))
		return
	}

	item, ok := h.backlogItemFromCreateRequest(w, r, &req)
	if !ok {
		return
	}

	if err := h.creationService().Create(item, CreationContext{
		Context:    r.Context(),
		Source:     SourceHumanHTTP,
		Entrypoint: "http.create",
	}); err != nil {
		mapCreateError(w, err)
		return
	}

	slog.Info("item created", "name", item.Name, "kind", item.Kind, "priority", item.Priority, "status", StatusBacklog)
	resp := &apipb.BacklogItemResponse{Item: backlogToProto(item)}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, resp); err != nil {
		apierr.MapError(w, "[backlog] create", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) backlogItemFromCreateRequest(w http.ResponseWriter, r *http.Request, req *apipb.CreateBacklogItemRequest) (BacklogItem, bool) {
	prov := identity.FromContext(r.Context())
	item, err := buildItemFromCreateRequest(req, prov, h.validateMilestoneReference)
	if err != nil {
		var ve *CreateValidationError
		if errors.As(err, &ve) {
			apierr.MapError(w, "[backlog] create", apierr.BadRequest("%s", ve.Msg))
		} else {
			apierr.MapError(w, "[backlog] create", apierr.Internal("%s", err.Error()))
		}
		return BacklogItem{}, false
	}
	return item, true
}

// mapCreateError translates Service.Create errors into HTTP responses.
// Duplicate → Conflict, dep / cycle / validation failures → BadRequest,
// everything else → Internal.
func mapCreateError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrItemExists):
		apierr.MapError(w, "[backlog] create", apierr.Conflict("backlog item already exists"))
	case strings.HasPrefix(err.Error(), "depends_on:") ||
		strings.HasPrefix(err.Error(), "dependency cycle"):
		apierr.MapError(w, "[backlog] create", apierr.BadRequest("%s", err.Error()))
	default:
		slog.Error("backlog create failed", "err", err)
		apierr.MapError(w, "[backlog] create", apierr.Internal("failed to create backlog item"))
	}
}

// creationService builds a backlog.Service per call from Handler's
// current dependencies. Constructed inline so late-bound setters
// (SetMilestoneAssigner, SetEventLogger) are picked up without
// requiring a separate sync step. The Service is the single chokepoint
// for item creation; HTTP callers MUST go through it.
func (h *Handler) creationService() *Service {
	cfg := ServiceConfig{
		Store:        h.store,
		Assigner:     h.milestoneAssigner,
		Invalidator:  graphInvalidatorFunc(h.invalidateAllGraphLenses),
		CycleChecker: CycleCheckerFunc(h.checkDependencyCycles),
	}
	if h.eventLogger != nil {
		cfg.Events = h.eventLogger
	}
	if h.sessionArtifacts != nil {
		cfg.Artifacts = h.sessionArtifacts
	}
	svc, err := NewService(cfg)
	if err != nil {
		// Store is always non-nil on Handler so this cannot fire in
		// practice; if it does, panic loudly rather than create items
		// silently bypassing the service contract.
		panic(fmt.Sprintf("backlog.Handler.creationService: %v", err))
	}
	return svc
}

// graphInvalidatorFunc adapts a func to the GraphInvalidator interface.
type graphInvalidatorFunc func()

func (f graphInvalidatorFunc) ScheduleAll() {
	if f != nil {
		f()
	}
}
