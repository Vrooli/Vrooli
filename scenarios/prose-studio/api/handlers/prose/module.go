package prose

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"prose-studio/internal/module"
	internal "prose-studio/internal/prose"

	"connectrpc.com/connect"

	"github.com/gorilla/mux"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/prose-studio/v1/prose"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prose-studio/v1/prose/prose_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type handler struct{ service *internal.Service }

func Module(service *internal.Service) module.Module {
	h := &handler{service: service}
	return module.Module{Name: "prose", Mount: func(r *mux.Router) {
		path, connectHandler := connectv1.NewProseStudioServiceHandler(h)
		r.PathPrefix(path).Handler(connectHandler)
		for _, op := range []string{"registry", "styles", "profiles/resolve", "generate", "sessions/reroll", "sessions/action", "declarations/reindex", "declarations/validate", "documents", "documents/list", "documents/get", "documents/assemble", "documents/resume", "conformance"} {
			r.HandleFunc("/api/v1/prose/"+op, h.rest).Methods(http.MethodPost)
		}
	}, Endpoints: Endpoints}
}

func (h *handler) Registry(ctx context.Context, req *connect.Request[v1.RegistryRequest]) (*connect.Response[v1.RegistryResponse], error) {
	value := h.service.Registry()
	var out v1.RegistryResponse
	if err := encodeResponse(value, &out); err != nil {
		return nil, err
	}
	return connect.NewResponse(&out), nil
}

func (h *handler) CreateStyle(ctx context.Context, req *connect.Request[v1.CreateStyleRequest]) (*connect.Response[v1.CreateStyleResponse], error) {
	var in internal.Style
	if err := decodeMessage(req.Msg.GetStyle(), &in); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out, err := h.service.CreateStyle(ctx, in)
	if err != nil {
		return nil, connect.NewError(codeFor(err), err)
	}
	var response v1.CreateStyleResponse
	if err := encodeResponse(map[string]any{"style": out}, &response); err != nil {
		return nil, err
	}
	return connect.NewResponse(&response), nil
}

func (h *handler) ResolveProfile(ctx context.Context, req *connect.Request[v1.ResolveProfileRequest]) (*connect.Response[v1.ResolveProfileResponse], error) {
	out, err := h.service.ResolveProfile(ctx, req.Msg.GetKey())
	if err != nil {
		return nil, connect.NewError(codeFor(err), err)
	}
	var response v1.ResolveProfileResponse
	if err := encodeResponse(out, &response); err != nil {
		return nil, err
	}
	return connect.NewResponse(&response), nil
}

func (h *handler) Generate(ctx context.Context, req *connect.Request[v1.GenerateRequest]) (*connect.Response[v1.GenerateResponse], error) {
	var in internal.GenerateRequest
	if err := decodeMessage(req.Msg, &in); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out, err := h.service.Generate(ctx, in)
	if err != nil {
		return nil, connect.NewError(codeFor(err), err)
	}
	var response v1.GenerateResponse
	if err := encodeResponse(out, &response); err != nil {
		return nil, err
	}
	return connect.NewResponse(&response), nil
}

func (h *handler) Reroll(ctx context.Context, req *connect.Request[v1.RerollRequest]) (*connect.Response[v1.RerollResponse], error) {
	out, err := h.service.Reroll(ctx, req.Msg.GetSessionId(), req.Msg.GetIncludeCandidates())
	if err != nil {
		return nil, connect.NewError(codeFor(err), err)
	}
	var response v1.RerollResponse
	if err := encodeResponse(map[string]any{"result": out}, &response); err != nil {
		return nil, err
	}
	return connect.NewResponse(&response), nil
}

func (h *handler) SessionAction(ctx context.Context, req *connect.Request[v1.SessionActionRequest]) (*connect.Response[v1.SessionActionResponse], error) {
	out, err := h.service.SessionAction(ctx, req.Msg.GetAction(), req.Msg.GetSessionId(), req.Msg.GetCandidateId())
	if err != nil {
		return nil, connect.NewError(codeFor(err), err)
	}
	var response v1.SessionActionResponse
	if err := encodeResponse(map[string]any{"session": out}, &response); err != nil {
		return nil, err
	}
	return connect.NewResponse(&response), nil
}

func (h *handler) ReindexDeclarations(ctx context.Context, req *connect.Request[v1.ReindexDeclarationsRequest]) (*connect.Response[v1.ReindexDeclarationsResponse], error) {
	out, err := h.service.Reindex(ctx, req.Msg.GetRoot())
	if err != nil {
		return nil, connect.NewError(codeFor(err), err)
	}
	var response v1.ReindexDeclarationsResponse
	if err := encodeResponse(map[string]any{"declarations": out}, &response); err != nil {
		return nil, err
	}
	return connect.NewResponse(&response), nil
}

func (h *handler) ValidateDeclarations(ctx context.Context, req *connect.Request[v1.ValidateDeclarationsRequest]) (*connect.Response[v1.ValidateDeclarationsResponse], error) {
	out, err := h.service.ValidateDeclarations(ctx, req.Msg.GetRoot())
	if err != nil {
		return nil, connect.NewError(codeFor(err), err)
	}
	var response v1.ValidateDeclarationsResponse
	if err := encodeResponse(map[string]any{"declarations": out}, &response); err != nil {
		return nil, err
	}
	return connect.NewResponse(&response), nil
}

func (h *handler) ListDocuments(ctx context.Context, req *connect.Request[v1.ListDocumentsRequest]) (*connect.Response[v1.ListDocumentsResponse], error) {
	out, err := h.service.ListDocuments(ctx, int(req.Msg.GetLimit()), req.Msg.GetStatus())
	if err != nil {
		return nil, connect.NewError(codeFor(err), err)
	}
	var response v1.ListDocumentsResponse
	if err := encodeResponse(map[string]any{"documents": out}, &response); err != nil {
		return nil, err
	}
	return connect.NewResponse(&response), nil
}

func (h *handler) GetDocument(ctx context.Context, req *connect.Request[v1.GetDocumentRequest]) (*connect.Response[v1.GetDocumentResponse], error) {
	out, err := h.service.GetDocument(ctx, req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(codeFor(err), err)
	}
	var response v1.GetDocumentResponse
	if err := encodeResponse(map[string]any{"document": out}, &response); err != nil {
		return nil, err
	}
	return connect.NewResponse(&response), nil
}

func (h *handler) CreateDocument(ctx context.Context, req *connect.Request[v1.CreateDocumentRequest]) (*connect.Response[v1.CreateDocumentResponse], error) {
	var in struct {
		Document internal.Document  `json:"document"`
		Sections []internal.Section `json:"sections"`
	}
	if err := decodeMessage(req.Msg, &in); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out, err := h.service.CreateDocument(ctx, in.Document, in.Sections)
	if err != nil {
		return nil, connect.NewError(codeFor(err), err)
	}
	var response v1.CreateDocumentResponse
	if err := encodeResponse(map[string]any{"document": out}, &response); err != nil {
		return nil, err
	}
	return connect.NewResponse(&response), nil
}

func (h *handler) AssembleDocument(ctx context.Context, req *connect.Request[v1.AssembleDocumentRequest]) (*connect.Response[v1.AssembleDocumentResponse], error) {
	out, err := h.service.AssembleDocument(ctx, req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(codeFor(err), err)
	}
	var response v1.AssembleDocumentResponse
	if err := encodeResponse(map[string]any{"document": out}, &response); err != nil {
		return nil, err
	}
	return connect.NewResponse(&response), nil
}

func (h *handler) ResumeDocument(ctx context.Context, req *connect.Request[v1.ResumeDocumentRequest]) (*connect.Response[v1.ResumeDocumentResponse], error) {
	out, err := h.service.ResumeDocument(ctx, req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(codeFor(err), err)
	}
	var response v1.ResumeDocumentResponse
	if err := encodeResponse(map[string]any{"document": out}, &response); err != nil {
		return nil, err
	}
	return connect.NewResponse(&response), nil
}

func (h *handler) Conformance(ctx context.Context, req *connect.Request[v1.ConformanceRequest]) (*connect.Response[v1.ConformanceResponse], error) {
	out, err := h.service.Conformance(ctx, req.Msg.GetStyleKey(), req.Msg.GetText())
	if err != nil {
		return nil, connect.NewError(codeFor(err), err)
	}
	var response v1.ConformanceResponse
	if err := encodeResponse(map[string]any{"report": out}, &response); err != nil {
		return nil, err
	}
	return connect.NewResponse(&response), nil
}

func decodeMessage(message proto.Message, out any) error {
	if message == nil {
		return errors.New("request message is required")
	}
	raw, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(message)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func encodeResponse(value any, message proto.Message) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, message); err != nil {
		return err
	}
	return nil
}

func (h *handler) rest(w http.ResponseWriter, r *http.Request) {
	op := strings.TrimPrefix(r.URL.Path, "/api/v1/prose/")
	op = map[string]string{"registry": "registry", "styles": "create_style", "profiles/resolve": "resolve_profile", "generate": "generate", "sessions/reroll": "reroll", "sessions/action": "session_action", "declarations/reindex": "reindex", "declarations/validate": "validate", "documents": "create_document", "documents/list": "list_documents", "documents/get": "get_document", "documents/assemble": "assemble", "documents/resume": "resume", "conformance": "conformance"}[op]
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
	case "list_documents":
		var in struct {
			Limit  int    `json:"limit"`
			Status string `json:"status"`
		}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &in); err != nil {
				return nil, err
			}
		}
		out, err := h.service.ListDocuments(ctx, in.Limit, in.Status)
		if err != nil {
			return nil, err
		}
		value = map[string]any{"documents": out}
	case "get_document":
		var in struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, err
		}
		out, err := h.service.GetDocument(ctx, in.ID)
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
	case "resume":
		var in struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, err
		}
		out, err := h.service.ResumeDocument(ctx, in.ID)
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
	case errors.Is(err, internal.ErrDocumentNotFound):
		return connect.CodeNotFound
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
	{ID: "prose_list_documents", Path: connectv1.ProseStudioServiceListDocumentsProcedure, Method: http.MethodPost, Summary: "List generated documents, newest first", Category: "documents"},
	{ID: "prose_get_document", Path: connectv1.ProseStudioServiceGetDocumentProcedure, Method: http.MethodPost, Summary: "Read one document with its outline, coherence, and provenance", Category: "documents"},
	{ID: "prose_create_document", Path: connectv1.ProseStudioServiceCreateDocumentProcedure, Method: http.MethodPost, Summary: "Create a section-composed document", Category: "documents"},
	{ID: "prose_assemble_document", Path: connectv1.ProseStudioServiceAssembleDocumentProcedure, Method: http.MethodPost, Summary: "Assemble committed sections to ordered text", Category: "documents"},
	{ID: "prose_resume_document", Path: connectv1.ProseStudioServiceResumeDocumentProcedure, Method: http.MethodPost, Summary: "Resume a partially committed document", Category: "documents"},
	{ID: "prose_conformance", Path: connectv1.ProseStudioServiceConformanceProcedure, Method: http.MethodPost, Summary: "Report style targets and lexicon spans", Category: "styles"},
}
