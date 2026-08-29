package agentsessions

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/agent-sessions", h.List).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/agent-sessions", h.Create).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/agent-sessions/{session_id}", h.Get).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/agent-sessions/{session_id}/kind", h.ChangeKind).Methods(http.MethodPatch)
	r.HandleFunc("/api/v1/agent-sessions/{session_id}/context", h.AttachContext).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/agent-sessions/{session_id}", h.Delete).Methods(http.MethodDelete)
	r.HandleFunc("/api/v1/agent-sessions/{session_id}/startup-brief", h.StartupBrief).Methods(http.MethodGet, http.MethodPost)
	r.HandleFunc("/api/v1/agent-sessions/{session_id}/attachments", h.UploadAttachments).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/agent-sessions/{session_id}/attachments/{attachment_id}", h.GetAttachment).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/agent-sessions/{session_id}/start", h.Start).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/agent-sessions/{session_id}/continue", h.Continue).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/agent-sessions/{session_id}/prompt-preview", h.PreviewPrompt).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/agent-sessions/{session_id}/events", h.ListEvents).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/agent-sessions/{session_id}/refresh", h.Refresh).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/agent-sessions/{session_id}/cancel", h.Cancel).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/agent-sessions/{session_id}/proposals/{proposal_id}/apply", h.ApplyProposal).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/agent-sessions/{session_id}/proposals/{proposal_id}/accept-keep", h.AcceptNoChangeRecommendation).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/agent-sessions/{session_id}/proposals/{proposal_id}/revise", h.ReviseMutationProposal).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/proposal-sessions", h.CreateProposalSession).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/proposal-sessions", h.ListProposalSessions).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/agent-sessions/{session_id}/artifacts", h.ListArtifacts).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/artifacts/by-entity", h.GetArtifactsByEntity).Methods(http.MethodGet)
}

type proposalSessionRequest struct {
	Kind       Kind           `json:"kind"`
	Title      string         `json:"title"`
	Target     ProposalTarget `json:"target"`
	StarterJob string         `json:"starter_job_id,omitempty"`
}

func (h *Handler) CreateProposalSession(w http.ResponseWriter, r *http.Request) {
	var req proposalSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.MapError(w, "[agent-sessions] create proposal session", apierr.BadRequest("invalid request body"))
		return
	}
	if req.Kind == "" {
		req.Kind = KindSwarmOperations
	}
	if strings.TrimSpace(req.Title) == "" {
		req.Title = "Proposal for " + req.Target.Name
	}
	session, err := h.service.Create(r.Context(), CreateRequest{Kind: req.Kind, Title: req.Title, ProposalTarget: &req.Target, StarterJob: req.StarterJob})
	if err != nil {
		apierr.MapError(w, "[agent-sessions] create proposal session", err)
		return
	}
	_ = httputil.JSONWithStatus(w, http.StatusCreated, session)
}

func (h *Handler) ListProposalSessions(w http.ResponseWriter, r *http.Request) {
	targetType, targetRef := ContextType(strings.TrimSpace(r.URL.Query().Get("target_type"))), strings.TrimSpace(r.URL.Query().Get("target_ref"))
	// Proposals are session-level state, so the per-session artifact hydration
	// List performs is pure cost here — and it dominates a whole-store scan.
	sessions, err := h.service.ListWithoutArtifacts(r.Context(), ListFilters{})
	if err != nil {
		apierr.MapError(w, "[agent-sessions] list proposal sessions", err)
		return
	}
	filtered := make([]Session, 0, len(sessions))
	for _, session := range sessions {
		if session.ProposalTarget == nil {
			continue
		}
		if targetType != "" && session.ProposalTarget.Type != targetType {
			continue
		}
		if targetRef != "" && session.ProposalTarget.Ref != targetRef {
			continue
		}
		filtered = append(filtered, session)
	}
	_ = httputil.JSONWithStatus(w, http.StatusOK, map[string]any{"sessions": filtered})
}

type mutationProposalDecisionRequest struct {
	AcceptedMutationIDs []string `json:"accepted_mutation_ids"`
	Note                string   `json:"note"`
}

func (h *Handler) DecideMutationProposal(w http.ResponseWriter, r *http.Request) {
	var req mutationProposalDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.MapError(w, "[agent-sessions] decide mutation proposal", apierr.BadRequest("invalid request body"))
		return
	}
	vars := mux.Vars(r)
	session, err := h.service.DecideMutationListProposal(r.Context(), vars["session_id"], vars["proposal_id"], req.AcceptedMutationIDs, req.Note)
	if err != nil {
		apierr.MapError(w, "[agent-sessions] decide mutation proposal", err)
		return
	}
	_ = httputil.JSONWithStatus(w, http.StatusOK, session)
}

func (h *Handler) AcceptNoChangeRecommendation(w http.ResponseWriter, r *http.Request) {
	var req mutationProposalRevisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.MapError(w, "[agent-sessions] accept no-change recommendation", apierr.BadRequest("invalid request body"))
		return
	}
	vars := mux.Vars(r)
	session, err := h.service.AcceptNoChangeRecommendation(r.Context(), vars["session_id"], vars["proposal_id"], req.Note)
	if err != nil {
		apierr.MapError(w, "[agent-sessions] accept no-change recommendation", err)
		return
	}
	_ = httputil.JSONWithStatus(w, http.StatusOK, session)
}

type mutationProposalRevisionRequest struct {
	Note string `json:"note"`
}

func (h *Handler) ReviseMutationProposal(w http.ResponseWriter, r *http.Request) {
	var req mutationProposalRevisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.MapError(w, "[agent-sessions] revise mutation proposal", apierr.BadRequest("invalid request body"))
		return
	}
	vars := mux.Vars(r)
	session, err := h.service.RequestMutationProposalRevision(r.Context(), vars["session_id"], vars["proposal_id"], req.Note)
	if err != nil {
		apierr.MapError(w, "[agent-sessions] revise mutation proposal", err)
		return
	}
	_ = httputil.JSONWithStatus(w, http.StatusOK, session)
}

func (h *Handler) StartupBrief(w http.ResponseWriter, r *http.Request) {
	brief, err := h.service.StartupBrief(r.Context(), mux.Vars(r)["session_id"])
	if err != nil {
		apierr.MapError(w, "[agent-sessions] startup-brief", err)
		return
	}
	if err := httputil.ProtoJSON(w, &apipb.GetAgentSessionStartupBriefResponse{Brief: contextItemToProto(brief)}); err != nil {
		apierr.MapError(w, "[agent-sessions] startup-brief", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filters, err := listFiltersFromQuery(r)
	if err != nil {
		apierr.MapError(w, "[agent-sessions] list", err)
		return
	}
	sessions, err := h.service.List(r.Context(), filters)
	if err != nil {
		apierr.MapError(w, "[agent-sessions] list", err)
		return
	}
	resp := &apipb.ListAgentSessionsResponse{}
	for _, session := range sessions {
		resp.Sessions = append(resp.Sessions, SessionToProto(session))
	}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[agent-sessions] list", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	session, err := h.service.Get(r.Context(), mux.Vars(r)["session_id"])
	if err != nil {
		apierr.MapError(w, "[agent-sessions] get", err)
		return
	}
	if err := httputil.ProtoJSON(w, &apipb.GetAgentSessionResponse{Session: SessionToProto(session)}); err != nil {
		apierr.MapError(w, "[agent-sessions] get", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req apipb.CreateAgentSessionRequest
	if err := httputil.DecodeProtoJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[agent-sessions] create", apierr.BadRequest("invalid request body: %s", err))
		return
	}
	if !httputil.ValidateProtoRequest(w, "[agent-sessions] create", "invalid agent session create request", &req) {
		return
	}
	createReq := CreateRequest{
		Kind:       Kind(req.Kind),
		Title:      req.Title,
		StarterJob: req.GetStarterJobId(),
	}
	session, err := h.service.Create(r.Context(), createReq)
	if err != nil {
		apierr.MapError(w, "[agent-sessions] create", err)
		return
	}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, &apipb.CreateAgentSessionResponse{Session: SessionToProto(session)}); err != nil {
		apierr.MapError(w, "[agent-sessions] create", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) ChangeKind(w http.ResponseWriter, r *http.Request) {
	var req apipb.ChangeAgentSessionKindRequest
	if err := httputil.DecodeProtoJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[agent-sessions] change kind", apierr.BadRequest("invalid request body: %s", err))
		return
	}
	req.SessionId = mux.Vars(r)["session_id"]
	if !httputil.ValidateProtoRequest(w, "[agent-sessions] change kind", "invalid agent session kind request", &req) {
		return
	}
	result, err := h.service.ChangeKind(r.Context(), ChangeKindRequest{
		SessionID: req.SessionId, Kind: Kind(req.Kind), ContextRefs: contextRefsFromProto(req.ContextRefs),
	})
	if err != nil {
		apierr.MapError(w, "[agent-sessions] change kind", err)
		return
	}
	if err := httputil.ProtoJSON(w, &apipb.ChangeAgentSessionKindResponse{
		Session: SessionToProto(result.Session), DroppedContextRefs: contextRefsToProto(result.DroppedContext), StarterJobCleared: result.StarterJobCleared,
	}); err != nil {
		apierr.MapError(w, "[agent-sessions] change kind", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) AttachContext(w http.ResponseWriter, r *http.Request) {
	var req apipb.AttachAgentSessionContextRequest
	if err := httputil.DecodeProtoJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[agent-sessions] attach context", apierr.BadRequest("invalid request body: %s", err))
		return
	}
	req.SessionId = mux.Vars(r)["session_id"]
	if !httputil.ValidateProtoRequest(w, "[agent-sessions] attach context", "invalid agent session context request", &req) {
		return
	}
	session, err := h.service.AttachContext(r.Context(), req.SessionId, contextRefsFromProto(req.ContextRefs))
	if err != nil {
		apierr.MapError(w, "[agent-sessions] attach context", err)
		return
	}
	if err := httputil.ProtoJSON(w, &apipb.AttachAgentSessionContextResponse{Session: SessionToProto(session)}); err != nil {
		apierr.MapError(w, "[agent-sessions] attach context", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	var req apipb.StartAgentSessionRequest
	if err := httputil.DecodeProtoJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[agent-sessions] start", apierr.BadRequest("invalid request body: %s", err))
		return
	}
	req.SessionId = mux.Vars(r)["session_id"]
	if !httputil.ValidateProtoRequest(w, "[agent-sessions] start", "invalid agent session start request", &req) {
		return
	}
	session, err := h.service.Start(r.Context(), ContinueRequest{
		SessionID:         req.SessionId,
		Message:           req.Message,
		AttachmentIDs:     req.AttachmentIds,
		ContextRefs:       contextRefsFromProto(req.ContextRefs),
		AutoContextPolicy: AutoContextPolicy(req.GetAutoContextPolicy()),
		StarterJob:        req.GetStarterJobId(),
	})
	if err != nil {
		apierr.MapError(w, "[agent-sessions] start", err)
		return
	}
	if err := httputil.ProtoJSON(w, &apipb.StartAgentSessionResponse{Session: SessionToProto(session)}); err != nil {
		apierr.MapError(w, "[agent-sessions] start", apierr.Internal("failed to encode response"))
	}
}

// PreviewPrompt returns the prompt a message would produce without sending it.
// Read-only: it appends no message, spawns no run, and changes no session state.
func (h *Handler) PreviewPrompt(w http.ResponseWriter, r *http.Request) {
	var req apipb.PreviewAgentSessionPromptRequest
	if err := httputil.DecodeProtoJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[agent-sessions] prompt-preview", apierr.BadRequest("invalid request body: %s", err))
		return
	}
	req.SessionId = mux.Vars(r)["session_id"]
	if !httputil.ValidateProtoRequest(w, "[agent-sessions] prompt-preview", "invalid agent session prompt preview request", &req) {
		return
	}
	preview, err := h.service.PreviewPrompt(r.Context(), ContinueRequest{
		SessionID:         req.SessionId,
		Message:           req.Message,
		AttachmentIDs:     req.AttachmentIds,
		ContextRefs:       contextRefsFromProto(req.ContextRefs),
		AutoContextPolicy: AutoContextPolicy(req.GetAutoContextPolicy()),
		StarterJob:        req.GetStarterJobId(),
	})
	if err != nil {
		apierr.MapError(w, "[agent-sessions] prompt-preview", err)
		return
	}
	if err := httputil.ProtoJSON(w, &apipb.PreviewAgentSessionPromptResponse{Prompt: preview.Prompt, Initial: preview.Initial}); err != nil {
		apierr.MapError(w, "[agent-sessions] prompt-preview", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) Continue(w http.ResponseWriter, r *http.Request) {
	var req apipb.ContinueAgentSessionRequest
	if err := httputil.DecodeProtoJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[agent-sessions] continue", apierr.BadRequest("invalid request body: %s", err))
		return
	}
	req.SessionId = mux.Vars(r)["session_id"]
	if !httputil.ValidateProtoRequest(w, "[agent-sessions] continue", "invalid agent session continue request", &req) {
		return
	}
	session, err := h.service.Continue(r.Context(), ContinueRequest{
		SessionID:     req.SessionId,
		Message:       req.Message,
		AttachmentIDs: req.AttachmentIds,
		ContextRefs:   contextRefsFromProto(req.ContextRefs),
	})
	if err != nil {
		apierr.MapError(w, "[agent-sessions] continue", err)
		return
	}
	if err := httputil.ProtoJSON(w, &apipb.ContinueAgentSessionResponse{Session: SessionToProto(session)}); err != nil {
		apierr.MapError(w, "[agent-sessions] continue", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) UploadAttachments(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(24 << 20); err != nil {
		apierr.MapError(w, "[agent-sessions] upload-attachments", apierr.BadRequest("invalid multipart form"))
		return
	}
	files := r.MultipartForm.File["files"]
	uploads := make([]AttachmentUpload, 0, len(files))
	opened := make([]interface{ Close() error }, 0, len(files))
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			for _, closeable := range opened {
				if closeErr := closeable.Close(); closeErr != nil {
					slog.Debug("agentsessions: close uploaded file failed", "err", closeErr)
				}
			}
			apierr.MapError(w, "[agent-sessions] upload-attachments", apierr.Internal("failed to read uploaded file"))
			return
		}
		opened = append(opened, file)
		uploads = append(uploads, AttachmentUpload{
			Filename:    fileHeader.Filename,
			ContentType: fileHeader.Header.Get("Content-Type"),
			SizeBytes:   fileHeader.Size,
			Reader:      file,
		})
	}
	attachments, err := h.service.UploadAttachments(r.Context(), mux.Vars(r)["session_id"], uploads)
	for _, closeable := range opened {
		if closeErr := closeable.Close(); closeErr != nil {
			slog.Debug("agentsessions: close uploaded file failed", "err", closeErr)
		}
	}
	if err != nil {
		apierr.MapError(w, "[agent-sessions] upload-attachments", err)
		return
	}
	resp := &apipb.UploadAgentSessionAttachmentsResponse{}
	for _, attachment := range attachments {
		resp.Attachments = append(resp.Attachments, attachmentToProto(attachment))
	}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, resp); err != nil {
		apierr.MapError(w, "[agent-sessions] upload-attachments", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) GetAttachment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	path, attachment, err := h.service.AttachmentPath(r.Context(), vars["session_id"], vars["attachment_id"])
	if err != nil {
		apierr.MapError(w, "[agent-sessions] attachment", err)
		return
	}
	if attachment.ContentType != "" {
		w.Header().Set("Content-Type", attachment.ContentType)
	}
	http.ServeFile(w, r, path)
}

func contextRefsFromProto(refs []*apipb.AgentSessionContextRef) []ContextRef {
	if len(refs) == 0 {
		return nil
	}
	result := make([]ContextRef, 0, len(refs))
	for _, ref := range refs {
		if ref == nil {
			continue
		}
		result = append(result, ContextRef{
			Type: ContextType(ref.Type),
			Ref:  ref.Ref,
		})
	}
	return result
}

func contextRefsToProto(refs []ContextRef) []*apipb.AgentSessionContextRef {
	result := make([]*apipb.AgentSessionContextRef, 0, len(refs))
	for _, ref := range refs {
		result = append(result, &apipb.AgentSessionContextRef{Type: string(ref.Type), Ref: ref.Ref})
	}
	return result
}

func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	req, err := listEventsRequestFromQuery(r)
	if err != nil {
		apierr.MapError(w, "[agent-sessions] events", err)
		return
	}
	result, err := h.service.ListEvents(r.Context(), req)
	if err != nil {
		apierr.MapError(w, "[agent-sessions] events", err)
		return
	}
	resp := &apipb.ListAgentSessionEventsResponse{
		HasMore:           result.HasMore,
		NextAfterSequence: result.NextAfterSequence,
	}
	for _, event := range result.Events {
		resp.Events = append(resp.Events, runEventToProto(event))
	}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[agent-sessions] events", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	session, err := h.service.Refresh(r.Context(), mux.Vars(r)["session_id"])
	if err != nil {
		apierr.MapError(w, "[agent-sessions] refresh", err)
		return
	}
	if err := httputil.ProtoJSON(w, &apipb.RefreshAgentSessionResponse{Session: SessionToProto(session)}); err != nil {
		apierr.MapError(w, "[agent-sessions] refresh", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	session, err := h.service.Cancel(r.Context(), mux.Vars(r)["session_id"])
	if err != nil {
		apierr.MapError(w, "[agent-sessions] cancel", err)
		return
	}
	if err := httputil.ProtoJSON(w, &apipb.CancelAgentSessionResponse{Session: SessionToProto(session)}); err != nil {
		apierr.MapError(w, "[agent-sessions] cancel", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	req := apipb.DeleteAgentSessionRequest{SessionId: mux.Vars(r)["session_id"]}
	if !httputil.ValidateProtoRequest(w, "[agent-sessions] delete", "invalid agent session delete request", &req) {
		return
	}
	if err := h.service.Delete(r.Context(), req.SessionId); err != nil {
		apierr.MapError(w, "[agent-sessions] delete", err)
		return
	}
	if err := httputil.ProtoJSON(w, &apipb.DeleteAgentSessionResponse{SessionId: req.SessionId}); err != nil {
		apierr.MapError(w, "[agent-sessions] delete", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) ApplyProposal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var decision mutationProposalDecisionRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&decision); err != nil {
			apierr.MapError(w, "[agent-sessions] apply-proposal", apierr.BadRequest("invalid request body"))
			return
		}
	}
	if len(decision.AcceptedMutationIDs) > 0 {
		session, err := h.service.DecideMutationListProposal(r.Context(), vars["session_id"], vars["proposal_id"], decision.AcceptedMutationIDs, decision.Note)
		if err != nil {
			apierr.MapError(w, "[agent-sessions] apply-proposal", err)
			return
		}
		if err := httputil.ProtoJSON(w, &apipb.ApplyAgentSessionProposalResponse{Session: SessionToProto(session)}); err != nil {
			apierr.MapError(w, "[agent-sessions] apply-proposal", apierr.Internal("failed to encode response"))
		}
		return
	}
	session, artifacts, err := h.service.ApplyProposal(r.Context(), vars["session_id"], vars["proposal_id"])
	if err != nil {
		apierr.MapError(w, "[agent-sessions] apply-proposal", err)
		return
	}
	resp := &apipb.ApplyAgentSessionProposalResponse{Session: SessionToProto(session)}
	for _, artifact := range artifacts {
		resp.Artifacts = append(resp.Artifacts, artifactToProto(artifact))
	}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[agent-sessions] apply-proposal", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) ListArtifacts(w http.ResponseWriter, r *http.Request) {
	artifacts, err := h.service.ListArtifacts(r.Context(), mux.Vars(r)["session_id"])
	if err != nil {
		apierr.MapError(w, "[agent-sessions] artifacts", err)
		return
	}
	resp := &apipb.ListAgentSessionArtifactsResponse{}
	for _, artifact := range artifacts {
		resp.Artifacts = append(resp.Artifacts, artifactToProto(artifact))
	}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[agent-sessions] artifacts", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) GetArtifactsByEntity(w http.ResponseWriter, r *http.Request) {
	artifactType := ArtifactType(strings.TrimSpace(r.URL.Query().Get("type")))
	if artifactType == "" {
		artifactType = ArtifactType(strings.TrimSpace(r.URL.Query().Get("artifact_type")))
	}
	entityRef := strings.TrimSpace(r.URL.Query().Get("ref"))
	if entityRef == "" {
		entityRef = strings.TrimSpace(r.URL.Query().Get("entity_ref"))
	}
	artifacts, err := h.service.ListArtifactsByEntity(r.Context(), artifactType, entityRef)
	if err != nil {
		apierr.MapError(w, "[agent-sessions] artifacts-by-entity", err)
		return
	}
	resp := &apipb.GetArtifactsByEntityResponse{}
	for _, artifact := range artifacts {
		resp.Artifacts = append(resp.Artifacts, artifactToProto(artifact))
	}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[agent-sessions] artifacts-by-entity", apierr.Internal("failed to encode response"))
	}
}

func listEventsRequestFromQuery(r *http.Request) (ListEventsRequest, error) {
	q := r.URL.Query()
	req := ListEventsRequest{SessionID: mux.Vars(r)["session_id"], Limit: 100}
	if value := strings.TrimSpace(q.Get("after_sequence")); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed < 0 {
			return ListEventsRequest{}, apierr.BadRequest("after_sequence must be zero or greater")
		}
		req.AfterSequence = parsed
	}
	if value := strings.TrimSpace(q.Get("limit")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 1000 {
			return ListEventsRequest{}, apierr.BadRequest("limit must be between 1 and 1000")
		}
		req.Limit = int32(parsed) // #nosec G109 -- parsed is bounded to [1,1000] immediately above; the int32 conversion cannot overflow.
	}
	return req, nil
}

func listFiltersFromQuery(r *http.Request) (ListFilters, error) {
	q := r.URL.Query()
	filters := ListFilters{
		Kind:   Kind(strings.TrimSpace(q.Get("kind"))),
		Status: Status(strings.TrimSpace(q.Get("status"))),
	}
	if filters.Kind != "" && !IsKnownKind(filters.Kind) {
		return ListFilters{}, apierr.BadRequest("kind is invalid")
	}
	if filters.Status != "" && !IsKnownStatus(filters.Status) {
		return ListFilters{}, apierr.BadRequest("status is invalid")
	}
	if active := strings.TrimSpace(q.Get("active_only")); active != "" {
		parsed, err := strconv.ParseBool(active)
		if err != nil {
			return ListFilters{}, apierr.BadRequest("active_only must be true or false")
		}
		filters.ActiveOnly = parsed
	}
	if limit := strings.TrimSpace(q.Get("limit")); limit != "" {
		parsed, err := strconv.Atoi(limit)
		if err != nil || parsed < 0 || parsed > 200 {
			return ListFilters{}, apierr.BadRequest("limit must be between 0 and 200")
		}
		filters.Limit = parsed
	}
	return filters, nil
}
