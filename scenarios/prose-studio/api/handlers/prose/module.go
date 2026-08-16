package prose

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"prose-studio/internal/module"
	internal "prose-studio/internal/prose"

	"github.com/gorilla/mux"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/prose-studio/v1/prose"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prose-studio/v1/prose/prose_v1connect"
)

type handler struct{ service *internal.Service }

func Module(service *internal.Service) module.Module {
	h := &handler{service: service}
	return module.Module{Name: "prose", Mount: func(r *mux.Router) {
		path, connectHandler := connectv1.NewProseStudioServiceHandler(h)
		r.PathPrefix(path).Handler(connectHandler)
		for _, op := range []string{"registry", "styles", "profiles/resolve", "generate", "sessions/reroll", "sessions/action", "declarations/reindex", "declarations/validate", "documents", "documents/assemble", "conformance"} {
			r.HandleFunc("/api/v1/prose/"+op, h.rest).Methods(http.MethodPost)
		}
	}, Endpoints: Endpoints}
}

func (h *handler) Registry(ctx context.Context, req *connect.Request[v1.JsonRequest]) (*connect.Response[v1.JsonResponse], error) {
	return h.call(ctx, "registry", req.Msg.GetJson())
}
func (h *handler) CreateStyle(ctx context.Context, req *connect.Request[v1.JsonRequest]) (*connect.Response[v1.JsonResponse], error) {
	return h.call(ctx, "create_style", req.Msg.GetJson())
}
func (h *handler) ResolveProfile(ctx context.Context, req *connect.Request[v1.JsonRequest]) (*connect.Response[v1.JsonResponse], error) {
	return h.call(ctx, "resolve_profile", req.Msg.GetJson())
}
func (h *handler) Generate(ctx context.Context, req *connect.Request[v1.JsonRequest]) (*connect.Response[v1.JsonResponse], error) {
	return h.call(ctx, "generate", req.Msg.GetJson())
}
func (h *handler) Reroll(ctx context.Context, req *connect.Request[v1.JsonRequest]) (*connect.Response[v1.JsonResponse], error) {
	return h.call(ctx, "reroll", req.Msg.GetJson())
}
func (h *handler) SessionAction(ctx context.Context, req *connect.Request[v1.JsonRequest]) (*connect.Response[v1.JsonResponse], error) {
	return h.call(ctx, "session_action", req.Msg.GetJson())
}
func (h *handler) ReindexDeclarations(ctx context.Context, req *connect.Request[v1.JsonRequest]) (*connect.Response[v1.JsonResponse], error) {
	return h.call(ctx, "reindex", req.Msg.GetJson())
}
func (h *handler) ValidateDeclarations(ctx context.Context, req *connect.Request[v1.JsonRequest]) (*connect.Response[v1.JsonResponse], error) {
	return h.call(ctx, "validate", req.Msg.GetJson())
}
func (h *handler) CreateDocument(ctx context.Context, req *connect.Request[v1.JsonRequest]) (*connect.Response[v1.JsonResponse], error) {
	return h.call(ctx, "create_document", req.Msg.GetJson())
}
func (h *handler) AssembleDocument(ctx context.Context, req *connect.Request[v1.JsonRequest]) (*connect.Response[v1.JsonResponse], error) {
	return h.call(ctx, "assemble", req.Msg.GetJson())
}
func (h *handler) Conformance(ctx context.Context, req *connect.Request[v1.JsonRequest]) (*connect.Response[v1.JsonResponse], error) {
	return h.call(ctx, "conformance", req.Msg.GetJson())
}

func (h *handler) rest(w http.ResponseWriter, r *http.Request) {
	op := strings.TrimPrefix(r.URL.Path, "/api/v1/prose/")
	op = map[string]string{"registry": "registry", "styles": "create_style", "profiles/resolve": "resolve_profile", "generate": "generate", "sessions/reroll": "reroll", "sessions/action": "session_action", "declarations/reindex": "reindex", "declarations/validate": "validate", "documents": "create_document", "documents/assemble": "assemble", "conformance": "conformance"}[op]
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, connect.CodeInvalidArgument, err)
		return
	}
	resp, err := h.dispatch(r.Context(), op, raw)
	if err != nil {
		writeError(w, codeFor(err), err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(resp)
}

func (h *handler) call(ctx context.Context, op, raw string) (*connect.Response[v1.JsonResponse], error) {
	out, err := h.dispatch(ctx, op, json.RawMessage(raw))
	if err != nil {
		return nil, connect.NewError(codeFor(err), err)
	}
	return connect.NewResponse(&v1.JsonResponse{Json: string(out)}), nil
}

func (h *handler) dispatch(ctx context.Context, op string, raw json.RawMessage) ([]byte, error) {
	if len(raw) == 0 || string(raw) == "null" {
		raw = []byte(`{}`)
	}
	var value any
	switch op {
	case "registry":
		value = h.service.Registry()
	case "create_style":
		var in internal.Style
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, err
		}
		out, err := h.service.CreateStyle(ctx, in)
		if err != nil {
			return nil, err
		}
		value = out
	case "resolve_profile":
		var in struct {
			Key string `json:"key"`
		}
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, err
		}
		out, err := h.service.ResolveProfile(ctx, in.Key)
		if err != nil {
			return nil, err
		}
		value = out
	case "generate":
		var in internal.GenerateRequest
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, err
		}
		out, err := h.service.Generate(ctx, in)
		if err != nil {
			return nil, err
		}
		value = out
	case "session_action":
		var in struct {
			Action      string `json:"action"`
			SessionID   string `json:"session_id"`
			CandidateID string `json:"candidate_id"`
		}
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, err
		}
		out, err := h.service.SessionAction(ctx, in.Action, in.SessionID, in.CandidateID)
		if err != nil {
			return nil, err
		}
		value = out
	case "reroll":
		var in struct {
			SessionID         string `json:"session_id"`
			IncludeCandidates bool   `json:"include_candidates"`
		}
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, err
		}
		out, err := h.service.Reroll(ctx, in.SessionID, in.IncludeCandidates)
		if err != nil {
			return nil, err
		}
		value = out
	case "reindex", "validate":
		var in struct {
			Root string `json:"root"`
		}
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, err
		}
		var out []internal.Declaration
		var err error
		if op == "reindex" {
			out, err = h.service.Reindex(ctx, in.Root)
		} else {
			out, err = h.service.ValidateDeclarations(ctx, in.Root)
		}
		if err != nil {
			return nil, err
		}
		value = out
	case "create_document":
		var in struct {
			Document internal.Document  `json:"document"`
			Sections []internal.Section `json:"sections"`
		}
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, err
		}
		out, err := h.service.CreateDocument(ctx, in.Document, in.Sections)
		if err != nil {
			return nil, err
		}
		value = out
	case "assemble":
		var in struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, err
		}
		out, err := h.service.AssembleDocument(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		value = out
	case "conformance":
		var in struct {
			StyleKey string `json:"style_key"`
			Text     string `json:"text"`
		}
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, err
		}
		out, err := h.service.Conformance(ctx, in.StyleKey, in.Text)
		if err != nil {
			return nil, err
		}
		value = out
	default:
		return nil, errors.New("unknown prose operation: " + op)
	}
	return json.Marshal(value)
}

func codeFor(err error) connect.Code {
	switch {
	case errors.Is(err, internal.ErrStyleResolutionConflict), errors.Is(err, internal.ErrDeclarationCollision):
		return connect.CodeFailedPrecondition
	case errors.Is(err, internal.ErrProfileDeclared), errors.Is(err, internal.ErrProfileUnregistered), errors.Is(err, internal.ErrBudgetExceeded), errors.Is(err, internal.ErrContextInfeasible):
		return connect.CodeFailedPrecondition
	default:
		return connect.CodeInvalidArgument
	}
}
func writeError(w http.ResponseWriter, code connect.Code, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]any{"code": code.String(), "error": err.Error()})
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "prose_registry", Path: connectv1.ProseStudioServiceRegistryProcedure, Method: http.MethodPost, Summary: "List governed sampler, policy, metric, and transform kinds", Category: "prose"},
	{ID: "prose_create_style", Path: connectv1.ProseStudioServiceCreateStyleProcedure, Method: http.MethodPost, Summary: "Create a versioned writing style", Category: "styles"},
	{ID: "prose_resolve_profile", Path: connectv1.ProseStudioServiceResolveProfileProcedure, Method: http.MethodPost, Summary: "Resolve a profile and inspect its exact instruction", Category: "profiles"},
	{ID: "prose_generate", Path: connectv1.ProseStudioServiceGenerateProcedure, Method: http.MethodPost, Summary: "Generate and measure a candidate set through ai-gateway", Category: "generation"},
	{ID: "prose_reroll", Path: connectv1.ProseStudioServiceRerollProcedure, Method: http.MethodPost, Summary: "Reroll a session with pinned and rejected context", Category: "sessions"},
	{ID: "prose_session_action", Path: connectv1.ProseStudioServiceSessionActionProcedure, Method: http.MethodPost, Summary: "Pin, unpin, reject, refine, or abandon a session", Category: "sessions"},
	{ID: "prose_reindex_declarations", Path: connectv1.ProseStudioServiceReindexDeclarationsProcedure, Method: http.MethodPost, Summary: "Register consumer-owned declaration files", Category: "declarations"},
	{ID: "prose_validate_declarations", Path: connectv1.ProseStudioServiceValidateDeclarationsProcedure, Method: http.MethodPost, Summary: "Validate declarations without test-genie", Category: "declarations"},
	{ID: "prose_create_document", Path: connectv1.ProseStudioServiceCreateDocumentProcedure, Method: http.MethodPost, Summary: "Create a section-composed document", Category: "documents"},
	{ID: "prose_assemble_document", Path: connectv1.ProseStudioServiceAssembleDocumentProcedure, Method: http.MethodPost, Summary: "Assemble committed sections to ordered text", Category: "documents"},
	{ID: "prose_conformance", Path: connectv1.ProseStudioServiceConformanceProcedure, Method: http.MethodPost, Summary: "Report style targets and lexicon spans", Category: "styles"},
}
