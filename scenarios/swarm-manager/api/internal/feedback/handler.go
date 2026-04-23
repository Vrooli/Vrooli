package feedback

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
)

// Handler exposes feedback-round HTTP endpoints. All routes are scoped to
// an initiative by name via `{name}` in the URL — a stable anchor that
// mirrors how the UI navigates (everything on the initiative page hangs
// off `/api/v1/initiatives/{name}/*`).
type Handler struct {
	svc *Service
}

// NewHandler constructs the handler around an already-configured service.
// The service owns all domain logic; the handler is transport-only.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts the feedback endpoints on the given router.
// Mirrors backlog.Handler.RegisterRoutes conventions so the overall
// surface feels uniform: REST-style paths, method-specific routes.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	base := "/api/v1/initiatives/{name}/feedback"

	r.HandleFunc(base, h.Start).Methods("POST")
	r.HandleFunc(base, h.List).Methods("GET")
	r.HandleFunc(base+"/{round:[0-9]+}", h.Get).Methods("GET")
	r.HandleFunc(base+"/{round:[0-9]+}/continue", h.Continue).Methods("POST")
	r.HandleFunc(base+"/{round:[0-9]+}/decide", h.Decide).Methods("POST")
	r.HandleFunc(base+"/{round:[0-9]+}/dismiss", h.Dismiss).Methods("POST")
	r.HandleFunc(base+"/{round:[0-9]+}/agent-turn", h.AgentTurn).Methods("POST")
	r.HandleFunc(base+"/{round:[0-9]+}/attachments/{id}", h.GetAttachment).Methods("GET")
	r.HandleFunc(base+"/lock", h.LockStatus).Methods("GET")
}

// --- Start round -------------------------------------------------------

// Start handles POST /api/v1/initiatives/{name}/feedback. Accepts either
// multipart/form-data (with optional files[]) or application/json.
func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if strings.TrimSpace(name) == "" {
		apierr.MapError(w, "[feedback] start", apierr.BadRequest("initiative name is required"))
		return
	}

	req, err := parseStartRequest(r)
	if err != nil {
		apierr.MapError(w, "[feedback] start", apierr.BadRequest("%s", err.Error()))
		return
	}
	req.InitiativeName = name

	// Multipart flow: files land on disk before spawn so we know the
	// attachment IDs. The folder name must match what the service will
	// pick later — use NextRoundNumber + ComputeSlug so both sides agree.
	// JSON flow: no files, AttachmentIDs may be empty.
	if r.MultipartForm != nil {
		n, err := h.svc.store.NextRoundNumber(name)
		if err != nil {
			apierr.MapError(w, "[feedback] start", apierr.Internal("assign round number: %s", err.Error()))
			return
		}
		slug := ComputeSlug(req.SlugHint, req.Text)
		ids, err := h.svc.store.SaveAttachments(name, n, slug, r)
		if err != nil {
			apierr.MapError(w, "[feedback] start", apierr.BadRequest("%s", err.Error()))
			return
		}
		req.AttachmentIDs = ids
	}

	round, err := h.svc.StartRound(r.Context(), req)
	if err != nil {
		h.writeStartError(w, err)
		return
	}
	if err := httputil.JSONWithStatus(w, http.StatusCreated, round); err != nil {
		apierr.MapError(w, "[feedback] start", apierr.Internal("encode response"))
	}
}

// writeStartError maps feedback-specific errors to HTTP status codes.
// Lock conflicts surface the current holder so the UI can render the
// override-warning dialog without a separate GET.
func (h *Handler) writeStartError(w http.ResponseWriter, err error) {
	var conflict *LockConflict
	if errors.As(err, &conflict) {
		payload := map[string]any{
			"error":  "initiative is locked",
			"holder": conflict.Holder,
		}
		_ = httputil.JSONWithStatus(w, http.StatusConflict, payload)
		return
	}
	if errors.Is(err, ErrLocked) {
		_ = httputil.JSONWithStatus(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	apierr.MapError(w, "[feedback] start", apierr.BadRequest("%s", err.Error()))
}

// startFormRequest is the parsed multipart/JSON payload. Extracted into a
// helper so the two encodings share a single code path for the service
// call. Field names match the JSON form; the multipart form reuses the
// same names as form-data keys.
type startFormRequest struct {
	InitiativeName string
	Type           RoundType
	Text           string
	AttachmentIDs  []string
	SlugHint       string
	Override       bool
	DecidedBy      string
}

func parseStartRequest(r *http.Request) (StartRoundRequest, error) {
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			return StartRoundRequest{}, fmt.Errorf("parse multipart: %w", err)
		}
		return StartRoundRequest{
			Type:      RoundType(strings.ToLower(strings.TrimSpace(r.FormValue("type")))),
			Text:      r.FormValue("text"),
			SlugHint:  r.FormValue("slug"),
			Override:  parseBool(r.FormValue("override")),
			DecidedBy: r.FormValue("decided_by"),
		}, nil
	}
	// JSON path.
	var body struct {
		Type          RoundType `json:"type"`
		Text          string    `json:"text"`
		Slug          string    `json:"slug,omitempty"`
		Override      bool      `json:"override,omitempty"`
		DecidedBy     string    `json:"decided_by,omitempty"`
		AttachmentIDs []string  `json:"attachment_ids,omitempty"`
	}
	if err := httputil.DecodeJSONStrict(r, &body); err != nil {
		return StartRoundRequest{}, err
	}
	return StartRoundRequest{
		Type:          body.Type,
		Text:          body.Text,
		AttachmentIDs: body.AttachmentIDs,
		SlugHint:      body.Slug,
		Override:      body.Override,
		DecidedBy:     body.DecidedBy,
	}, nil
}

func parseBool(raw string) bool {
	v, _ := strconv.ParseBool(strings.TrimSpace(raw))
	return v
}

// --- List / Get --------------------------------------------------------

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	rounds, err := h.svc.store.ListRounds(name)
	if err != nil {
		apierr.MapError(w, "[feedback] list", apierr.Internal("%s", err.Error()))
		return
	}
	if err := httputil.JSON(w, map[string]any{"rounds": rounds, "count": len(rounds)}); err != nil {
		apierr.MapError(w, "[feedback] list", apierr.Internal("encode response"))
	}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	name, roundNum, ok := h.parseNameAndRound(w, r, "get")
	if !ok {
		return
	}
	round, err := h.svc.store.LoadRound(name, roundNum)
	if err != nil {
		if errors.Is(err, ErrRoundNotFound) {
			apierr.MapError(w, "[feedback] get", apierr.NotFound("feedback round not found"))
			return
		}
		apierr.MapError(w, "[feedback] get", apierr.Internal("%s", err.Error()))
		return
	}
	if err := httputil.JSON(w, round); err != nil {
		apierr.MapError(w, "[feedback] get", apierr.Internal("encode response"))
	}
}

// --- Continue / Decide / Dismiss ---------------------------------------

func (h *Handler) Continue(w http.ResponseWriter, r *http.Request) {
	name, roundNum, ok := h.parseNameAndRound(w, r, "continue")
	if !ok {
		return
	}

	req := ContinueRoundRequest{InitiativeName: name, RoundNumber: roundNum}
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			apierr.MapError(w, "[feedback] continue", apierr.BadRequest("parse multipart: %s", err.Error()))
			return
		}
		// Load the round to recover the slug — attachments land in the
		// round's folder.
		existing, err := h.svc.store.LoadRound(name, roundNum)
		if err != nil {
			if errors.Is(err, ErrRoundNotFound) {
				apierr.MapError(w, "[feedback] continue", apierr.NotFound("feedback round not found"))
				return
			}
			apierr.MapError(w, "[feedback] continue", apierr.Internal("%s", err.Error()))
			return
		}
		ids, err := h.svc.store.SaveAttachments(name, existing.Number, existing.Slug, r)
		if err != nil {
			apierr.MapError(w, "[feedback] continue", apierr.BadRequest("%s", err.Error()))
			return
		}
		req.Text = r.FormValue("text")
		req.AttachmentIDs = ids
		req.DecidedBy = r.FormValue("decided_by")
	} else {
		var body struct {
			Text          string   `json:"text"`
			AttachmentIDs []string `json:"attachment_ids,omitempty"`
			DecidedBy     string   `json:"decided_by,omitempty"`
		}
		if err := httputil.DecodeJSONStrict(r, &body); err != nil {
			apierr.MapError(w, "[feedback] continue", apierr.BadRequest("%s", err.Error()))
			return
		}
		req.Text = body.Text
		req.AttachmentIDs = body.AttachmentIDs
		req.DecidedBy = body.DecidedBy
	}

	round, err := h.svc.ContinueRound(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrRoundNotFound) {
			apierr.MapError(w, "[feedback] continue", apierr.NotFound("feedback round not found"))
			return
		}
		apierr.MapError(w, "[feedback] continue", apierr.BadRequest("%s", err.Error()))
		return
	}
	if err := httputil.JSON(w, round); err != nil {
		apierr.MapError(w, "[feedback] continue", apierr.Internal("encode response"))
	}
}

func (h *Handler) Decide(w http.ResponseWriter, r *http.Request) {
	name, roundNum, ok := h.parseNameAndRound(w, r, "decide")
	if !ok {
		return
	}
	var body struct {
		Kind                DecisionKind `json:"kind"`
		AcceptedMutationIDs []string     `json:"accepted_mutation_ids,omitempty"`
		Rationale           string       `json:"rationale,omitempty"`
		DecidedBy           string       `json:"decided_by,omitempty"`
	}
	if err := httputil.DecodeJSONStrict(r, &body); err != nil {
		apierr.MapError(w, "[feedback] decide", apierr.BadRequest("%s", err.Error()))
		return
	}
	round, result, err := h.svc.Decide(r.Context(), DecideRequest{
		InitiativeName:      name,
		RoundNumber:         roundNum,
		Kind:                body.Kind,
		AcceptedMutationIDs: body.AcceptedMutationIDs,
		Rationale:           body.Rationale,
		DecidedBy:           body.DecidedBy,
	})
	if err != nil {
		if errors.Is(err, ErrRoundNotFound) {
			apierr.MapError(w, "[feedback] decide", apierr.NotFound("feedback round not found"))
			return
		}
		apierr.MapError(w, "[feedback] decide", apierr.BadRequest("%s", err.Error()))
		return
	}
	payload := map[string]any{"round": round}
	if result != nil {
		payload["apply_result"] = result
	}
	if err := httputil.JSON(w, payload); err != nil {
		apierr.MapError(w, "[feedback] decide", apierr.Internal("encode response"))
	}
}

// Dismiss is a convenience wrapper around Decide(kind=dismiss). Accepts an
// optional JSON rationale.
func (h *Handler) Dismiss(w http.ResponseWriter, r *http.Request) {
	name, roundNum, ok := h.parseNameAndRound(w, r, "dismiss")
	if !ok {
		return
	}
	var body struct {
		Rationale string `json:"rationale,omitempty"`
		DecidedBy string `json:"decided_by,omitempty"`
	}
	if r.ContentLength > 0 {
		if err := httputil.DecodeJSONStrict(r, &body); err != nil {
			apierr.MapError(w, "[feedback] dismiss", apierr.BadRequest("%s", err.Error()))
			return
		}
	}
	round, _, err := h.svc.Decide(r.Context(), DecideRequest{
		InitiativeName: name,
		RoundNumber:    roundNum,
		Kind:           DecisionDismiss,
		Rationale:      body.Rationale,
		DecidedBy:      body.DecidedBy,
	})
	if err != nil {
		if errors.Is(err, ErrRoundNotFound) {
			apierr.MapError(w, "[feedback] dismiss", apierr.NotFound("feedback round not found"))
			return
		}
		apierr.MapError(w, "[feedback] dismiss", apierr.BadRequest("%s", err.Error()))
		return
	}
	if err := httputil.JSON(w, round); err != nil {
		apierr.MapError(w, "[feedback] dismiss", apierr.Internal("encode response"))
	}
}

// AgentTurn is the inbound hook invoked when the feedback agent produces
// output. Mirrors how clarification wiring routes agent messages back into
// the thread. Accepts `{body: string}` JSON.
//
// In production the hook is invoked by the agent-manager listener or by
// a CLI subcommand that polls the run; in tests it's exercised directly.
func (h *Handler) AgentTurn(w http.ResponseWriter, r *http.Request) {
	name, roundNum, ok := h.parseNameAndRound(w, r, "agent-turn")
	if !ok {
		return
	}
	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apierr.MapError(w, "[feedback] agent-turn", apierr.BadRequest("%s", err.Error()))
		return
	}
	round, err := h.svc.RecordAgentTurn(name, roundNum, body.Body)
	if err != nil {
		if errors.Is(err, ErrRoundNotFound) {
			apierr.MapError(w, "[feedback] agent-turn", apierr.NotFound("feedback round not found"))
			return
		}
		apierr.MapError(w, "[feedback] agent-turn", apierr.BadRequest("%s", err.Error()))
		return
	}
	if err := httputil.JSON(w, round); err != nil {
		apierr.MapError(w, "[feedback] agent-turn", apierr.Internal("encode response"))
	}
}

// GetAttachment serves a previously uploaded attachment. Routed via the
// attachment ID (UUID-prefixed filename) persisted on the round.
func (h *Handler) GetAttachment(w http.ResponseWriter, r *http.Request) {
	name, roundNum, ok := h.parseNameAndRound(w, r, "attachment")
	if !ok {
		return
	}
	id := mux.Vars(r)["id"]
	if strings.TrimSpace(id) == "" {
		apierr.MapError(w, "[feedback] attachment", apierr.BadRequest("attachment id is required"))
		return
	}

	// Recover slug from the round so ResolveAttachment can find the
	// correct folder.
	round, err := h.svc.store.LoadRound(name, roundNum)
	if err != nil {
		if errors.Is(err, ErrRoundNotFound) {
			apierr.MapError(w, "[feedback] attachment", apierr.NotFound("feedback round not found"))
			return
		}
		apierr.MapError(w, "[feedback] attachment", apierr.Internal("%s", err.Error()))
		return
	}

	path, ok := h.svc.store.ResolveAttachment(name, round.Number, round.Slug, id)
	if !ok {
		apierr.MapError(w, "[feedback] attachment", apierr.NotFound("attachment not found"))
		return
	}
	w.Header().Set("Content-Type", ContentTypeForAttachment(id))
	http.ServeFile(w, r, path)
}

// LockStatus returns the current holder if the lock is held. The UI calls
// this before showing the "Add feedback" button so it can pre-populate
// the override dialog.
func (h *Handler) LockStatus(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	holder, err := h.svc.lock.Inspect(name)
	if err != nil {
		apierr.MapError(w, "[feedback] lock", apierr.Internal("%s", err.Error()))
		return
	}
	payload := map[string]any{"locked": holder != nil}
	if holder != nil {
		payload["holder"] = holder
	}
	if err := httputil.JSON(w, payload); err != nil {
		apierr.MapError(w, "[feedback] lock", apierr.Internal("encode response"))
	}
}

// --- helpers -----------------------------------------------------------

func (h *Handler) parseNameAndRound(w http.ResponseWriter, r *http.Request, action string) (string, int, bool) {
	vars := mux.Vars(r)
	name := vars["name"]
	if strings.TrimSpace(name) == "" {
		apierr.MapError(w, "[feedback] "+action, apierr.BadRequest("initiative name is required"))
		return "", 0, false
	}
	n, err := strconv.Atoi(vars["round"])
	if err != nil || n <= 0 {
		apierr.MapError(w, "[feedback] "+action, apierr.BadRequest("round must be a positive integer"))
		return "", 0, false
	}
	return name, n, true
}
