package agentaccess

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/browser-automation-studio/automation/harness"
	"github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
	actionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	"google.golang.org/protobuf/encoding/protojson"
)

// HTTPHandler is the transport adapter for CLI, MCP and program-runtime
// clients. All session operations delegate to Broker; this handler has no
// driver or browser lifecycle of its own.
type HTTPHandler struct {
	Broker                *Broker
	Workflows             WorkflowCatalog
	ProjectScopes         ProjectScopeAuthorizer
	Harnesses             *harness.Registry
	BridgeOrigin          string
	BridgeExtensionOrigin string
	EvidenceTokenVerifier authn.TokenVerifier
}

func (h *HTTPHandler) RegisterRoutes(r chi.Router) {
	r.Route("/browser/internal/evidence-proof", func(internal chi.Router) {
		internal.Use(h.evidenceVerifierMiddleware)
		internal.Post("/verify", h.verifyEvidenceProof)
		internal.Post("/consume", h.consumeEvidenceProof)
	})
	r.Get("/browser/bridge/socket", h.selectedTabSocket)
	r.Post("/browser/bridge/pair", h.pairSelectedTab)
	r.Options("/browser/bridge/pair", h.pairSelectedTab)
	r.Post("/browser/sessions/{sessionID}/stream-cookie", h.streamCookie)
	r.Route("/browser", func(browser chi.Router) {
		browser.Use(h.agentContext)
		browser.Post("/sessions", h.open)
		browser.Get("/capabilities", h.capabilities)
		browser.Get("/capabilities/profile", h.profileCapabilities)
		browser.Post("/sessions/{sessionID}/attach", h.attach)
		browser.Post("/sessions/{sessionID}/observe", h.observe)
		// /task is a deliberately small, Broker-owned harness vocabulary for
		// external clients. It never accepts a driver/CDP handle or page script.
		browser.Post("/sessions/{sessionID}/task", h.runHarnessTask)
		browser.Get("/sessions/{sessionID}/accessibility-tree", h.accessibilityTree)
		browser.Post("/sessions/{sessionID}/app-monitor-capture", h.appMonitorCapture)
		browser.Get("/sessions/{sessionID}/extensions/inventory", h.extensionInventory)
		browser.Get("/sessions/{sessionID}/history", h.navigationHistory)
		browser.Post("/sessions/{sessionID}/handoff", h.handoff)
		browser.Post("/sessions/{sessionID}/act", h.act)
		browser.Post("/sessions/{sessionID}/audio/diagnostics", h.audioDiagnostics)
		browser.Post("/sessions/{sessionID}/audio/{command}", h.audioPlayback)
		browser.Get("/sessions/{sessionID}/trace", h.trace)
		browser.Post("/sessions/{sessionID}/trace/draft", h.draftTrace)
		browser.Post("/sessions/{sessionID}/trace/promote", h.promote)
		browser.Post("/sessions/{sessionID}/workflows/{workflowID}/replay", h.replayPromotedWorkflow)
		browser.Post("/sessions/{sessionID}/extensions/install", h.installExtension)
		browser.Post("/sessions/{sessionID}/extensions/preflight", h.preflightExtensionLifecycle)
		browser.Post("/sessions/{sessionID}/extensions/{extensionID}/reload", h.reloadExtension)
		browser.Post("/sessions/{sessionID}/extensions/{extensionID}/load", h.loadExtension)
		browser.Post("/sessions/{sessionID}/extensions/{extensionID}/unload", h.unloadExtension)
		browser.Get("/sessions/{sessionID}/extensions/{extensionID}/validate", h.validateExtension)
		browser.Post("/sessions/{sessionID}/close", h.close)
		browser.Get("/harnesses", h.listHarnesses)
		browser.Post("/sessions/{sessionID}/harnesses/{adapterID}/qualify", h.qualify)
	})
}

func (h *HTTPHandler) evidenceVerifierMiddleware(next http.Handler) http.Handler {
	if h.EvidenceTokenVerifier == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeErrorStatus(w, http.StatusServiceUnavailable, errors.New("Agent Manager evidence verifier is not configured"))
		})
	}
	provider := authn.TokenProvider{
		ProviderSource: identity.SourceScenarioAuthenticator,
		Verifier:       h.EvidenceTokenVerifier,
		Extract: func(r *http.Request) string {
			value, found := strings.CutPrefix(strings.TrimSpace(r.Header.Get("Authorization")), "Bearer ")
			if !found {
				return ""
			}
			return strings.TrimSpace(value)
		},
	}
	return authn.Middleware(authn.Config{Providers: []authn.Provider{provider}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := identity.PrincipalFromContext(r.Context())
		if !ok || !principal.IsVerified() || principal.Kind != identity.ActorService {
			writeErrorStatus(w, http.StatusUnauthorized, errors.New("verified Agent Manager service identity required"))
			return
		}
		source, err := BrowserEvidenceVerifierRole.Source(principal)
		if err != nil || source != "agent-manager" {
			writeErrorStatus(w, http.StatusForbidden, errors.New("Agent Manager browser evidence verifier role required"))
			return
		}
		if _, err := authn.RequireCapability(r.Context(), BrowserEvidenceVerifierScope); err != nil {
			writeErrorStatus(w, http.StatusForbidden, err)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

type streamCookieRequest struct {
	Receipt string     `json:"receipt"`
	Role    StreamRole `json:"role"`
}

func (h *HTTPHandler) streamCookie(w http.ResponseWriter, r *http.Request) {
	var request streamCookieRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	issued, err := h.Broker.IssueStreamCookie(r.Context(), chi.URLParam(r, "sessionID"), request.Receipt, request.Role)
	if err != nil {
		writeError(w, err)
		return
	}
	maxAge := int(time.Until(issued.ExpiresAt).Seconds())
	if maxAge < 1 {
		writeErrorStatus(w, http.StatusUnauthorized, errors.New("stream receipt expired"))
		return
	}
	http.SetCookie(w, &http.Cookie{Name: issued.Name, Value: issued.Value, Path: issued.Path, HttpOnly: true, Secure: r.TLS != nil || strings.HasPrefix(strings.ToLower(r.Header.Get("Origin")), "https://"), SameSite: http.SameSiteStrictMode, MaxAge: maxAge, Expires: issued.ExpiresAt})
	writeResponse(w, map[string]any{"session_id": issued.SessionID, "role": issued.Role, "expires_at": issued.ExpiresAt}, nil)
}

func (h *HTTPHandler) extensionInventory(w http.ResponseWriter, r *http.Request) {
	response, err := h.Broker.SelectedTabExtensions(r.Context(), chi.URLParam(r, "sessionID"))
	writeResponse(w, response, err)
}

func (h *HTTPHandler) navigationHistory(w http.ResponseWriter, r *http.Request) {
	response, err := h.Broker.NavigationHistory(r.Context(), chi.URLParam(r, "sessionID"))
	writeResponse(w, response, err)
}

func (h *HTTPHandler) handoff(w http.ResponseWriter, r *http.Request) {
	response, err := h.Broker.Handoff(r.Context(), chi.URLParam(r, "sessionID"))
	writeResponse(w, response, err)
}

func (h *HTTPHandler) pairSelectedTab(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if h.BridgeExtensionOrigin == "" || origin != h.BridgeExtensionOrigin {
		writeErrorStatus(w, http.StatusForbidden, errors.New("selected-tab bridge extension origin is not permitted"))
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", h.BridgeExtensionOrigin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Add("Vary", "Origin")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var request PairTabRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if h.BridgeOrigin == "" {
		writeError(w, errors.New("selected-tab bridge public origin is not configured"))
		return
	}
	response, err := h.Broker.PairSelectedTab(r.Context(), request, h.BridgeOrigin)
	writeResponse(w, response, err)
}

// This single-use ticket route is outside Agent Manager run-header middleware;
// the browser extension cannot send those headers after authenticated pairing.
func (h *HTTPHandler) selectedTabSocket(w http.ResponseWriter, r *http.Request) {
	h.Broker.ConnectSelectedTab(w, r, h.BridgeExtensionOrigin)
}

func (h *HTTPHandler) agentContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokens, grants := r.Header.Values("X-Agent-Identity-Token"), r.Header.Values("X-Vrooli-Browser-Grant-Receipt")
		if len(tokens) != 1 || strings.TrimSpace(tokens[0]) == "" || len(grants) != 1 || strings.TrimSpace(grants[0]) == "" {
			writeErrorStatus(w, http.StatusUnauthorized, errors.New("one Agent Manager identity token and browser grant receipt are required"))
			return
		}
		ctx := WithBrowserAgentCredentials(r.Context(), strings.TrimSpace(tokens[0]), strings.TrimSpace(grants[0]))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeErrorStatus(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func (h *HTTPHandler) capabilities(w http.ResponseWriter, r *http.Request) {
	required, err := parseRequiredCapabilities(r.URL.Query().Get("required"))
	if err != nil {
		writeError(w, err)
		return
	}
	response, err := h.Broker.Capabilities(r.Context(), required)
	writeResponse(w, response, err)
}

func (h *HTTPHandler) profileCapabilities(w http.ResponseWriter, r *http.Request) {
	required, err := parseRequiredCapabilities(r.URL.Query().Get("required"))
	if err != nil {
		writeError(w, err)
		return
	}
	response, err := h.Broker.CapabilityPreview(r.Context(), required)
	writeResponse(w, response, err)
}

func parseRequiredCapabilities(raw string) ([]sessionpermission.Capability, error) {
	known := map[string]sessionpermission.Capability{
		"observe": sessionpermission.CapabilityObserve, "control": sessionpermission.CapabilityControl,
		"accessibility_tree": sessionpermission.CapabilityAccessibilityTree,
		"approval_receipt":   sessionpermission.CapabilityApprovalReceipt, "credit_budget": sessionpermission.CapabilityCreditBudget,
		"byte_budget":       sessionpermission.CapabilityByteBudget,
		"history_retention": sessionpermission.CapabilityHistoryRetention,
		"history_read":      sessionpermission.CapabilityHistoryRead, "history_export": sessionpermission.CapabilityHistoryExport,
		"credential_use": sessionpermission.CapabilityCredentialUse, "extension_inventory": sessionpermission.CapabilityExtensionInventory,
		"extension_host_permissions": sessionpermission.CapabilityExtensionHostPermissions, "extension_load": sessionpermission.CapabilityExtensionLoad,
		"extension_install": sessionpermission.CapabilityExtensionInstall, "handoff": sessionpermission.CapabilityHandoff,
		"reconnect": sessionpermission.CapabilityReconnect, "preserve_user_state": sessionpermission.CapabilityPreserveUserState,
		"audio_capture":         sessionpermission.CapabilityAudioCapture,
		"audio_device_evidence": sessionpermission.CapabilityAudioDeviceEvidence,
		"app_monitor_metrics":   sessionpermission.CapabilityAppMonitorMetrics,
		"app_monitor_trace":     sessionpermission.CapabilityAppMonitorTrace,
	}
	var required []sessionpermission.Capability
	for _, value := range strings.Split(raw, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		capability, ok := known[value]
		if !ok {
			return nil, errors.New("unknown required browser capability: " + value)
		}
		required = append(required, capability)
	}
	return required, nil
}

func (h *HTTPHandler) audioPlayback(w http.ResponseWriter, r *http.Request) {
	command := session.AudioPlaybackCommand(strings.TrimSpace(chi.URLParam(r, "command")))
	if command != session.AudioPlaybackRestart && command != session.AudioPlaybackStop {
		writeErrorStatus(w, http.StatusBadRequest, errors.New("unsupported audio command"))
		return
	}
	if err := h.Broker.AudioPlayback(r.Context(), chi.URLParam(r, "sessionID"), command); err != nil {
		writeError(w, err)
		return
	}
	writeResponse(w, map[string]any{"session_id": chi.URLParam(r, "sessionID"), "command": command, "completed": true}, nil)
}

func (h *HTTPHandler) audioDiagnostics(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Surface string `json:"surface"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.Surface != "audio-tools" && request.Surface != "swarm-manager" {
		writeErrorStatus(w, http.StatusBadRequest, errors.New("unsupported Audio Tools diagnostic surface"))
		return
	}
	diagnostics, err := h.Broker.AudioDiagnostics(r.Context(), chi.URLParam(r, "sessionID"), request.Surface)
	writeResponse(w, map[string]any{"session_id": chi.URLParam(r, "sessionID"), "diagnostics": diagnostics}, err)
}

type installExtensionRequest struct {
	// ExtensionID is an optional assertion for legacy callers. The broker never
	// treats it as authority; the artifact owner derives the installed ID.
	ExtensionID    string `json:"extension_id,omitempty"`
	ArtifactRef    string `json:"artifact_ref"`
	ExpectedDigest string `json:"expected_digest"`
}

func (h *HTTPHandler) installExtension(w http.ResponseWriter, r *http.Request) {
	var request installExtensionRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	response, err := h.Broker.InstallExtension(r.Context(), chi.URLParam(r, "sessionID"), request.ExtensionID, request.ArtifactRef, request.ExpectedDigest)
	writeResponse(w, response, err)
}

func (h *HTTPHandler) preflightExtensionLifecycle(w http.ResponseWriter, r *http.Request) {
	var request installExtensionRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	err := h.Broker.PreflightExtensionLifecycle(r.Context(), chi.URLParam(r, "sessionID"), request.ArtifactRef, request.ExpectedDigest, request.ExtensionID)
	writeResponse(w, map[string]bool{"ready": err == nil}, err)
}

func (h *HTTPHandler) reloadExtension(w http.ResponseWriter, r *http.Request) {
	response, err := h.Broker.ReloadExtension(r.Context(), chi.URLParam(r, "sessionID"), chi.URLParam(r, "extensionID"))
	writeResponse(w, response, err)
}

func (h *HTTPHandler) loadExtension(w http.ResponseWriter, r *http.Request) {
	response, err := h.Broker.LoadExtension(r.Context(), chi.URLParam(r, "sessionID"), chi.URLParam(r, "extensionID"))
	writeResponse(w, response, err)
}

func (h *HTTPHandler) unloadExtension(w http.ResponseWriter, r *http.Request) {
	response, err := h.Broker.UnloadExtension(r.Context(), chi.URLParam(r, "sessionID"), chi.URLParam(r, "extensionID"))
	writeResponse(w, response, err)
}

func (h *HTTPHandler) validateExtension(w http.ResponseWriter, r *http.Request) {
	response, err := h.Broker.ValidateExtension(r.Context(), chi.URLParam(r, "sessionID"), chi.URLParam(r, "extensionID"))
	writeResponse(w, response, err)
}

func (h *HTTPHandler) open(w http.ResponseWriter, r *http.Request) {
	var request OpenRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	response, err := h.Broker.Open(r.Context(), request)
	writeResponse(w, response, err)
}

func (h *HTTPHandler) attach(w http.ResponseWriter, r *http.Request) {
	var request AttachRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	request.SessionID = chi.URLParam(r, "sessionID")
	response, err := h.Broker.Attach(r.Context(), request)
	writeResponse(w, response, err)
}

func (h *HTTPHandler) observe(w http.ResponseWriter, r *http.Request) {
	var request ObserveRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	response, err := h.Broker.Observe(r.Context(), chi.URLParam(r, "sessionID"), request)
	writeResponse(w, response, err)
}

func (h *HTTPHandler) act(w http.ResponseWriter, r *http.Request) {
	var request ActRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	response, err := h.Broker.Act(r.Context(), chi.URLParam(r, "sessionID"), request)
	writeResponse(w, response, err)
}

func (h *HTTPHandler) trace(w http.ResponseWriter, r *http.Request) {
	response, err := h.Broker.Trace(r.Context(), chi.URLParam(r, "sessionID"))
	writeResponse(w, response, err)
}

// draftTrace compiles the accepted trace from this already-authorized BAS
// session without touching the workflow catalog. Persisted promotion remains
// on /trace/promote and continues to require project scope.
func (h *HTTPHandler) draftTrace(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name        string            `json:"name"`
		Description string            `json:"description,omitempty"`
		Assertions  []json.RawMessage `json:"assertions"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.Name) == "" || len(request.Assertions) == 0 {
		writeErrorStatus(w, http.StatusBadRequest, errors.New("name and at least one typed assertion are required"))
		return
	}
	assertions, err := decodeActionDefinitions(request.Assertions)
	if err != nil {
		writeErrorStatus(w, http.StatusBadRequest, fmt.Errorf("decode typed assertions: %w", err))
		return
	}
	response, err := h.Broker.DraftTrace(r.Context(), chi.URLParam(r, "sessionID"), PromoteRequest{
		Name: strings.TrimSpace(request.Name), Description: strings.TrimSpace(request.Description), Assertions: assertions,
	})
	writeTraceDraftResponse(w, response, err)
}

func (h *HTTPHandler) promote(w http.ResponseWriter, r *http.Request) {
	if h.Workflows == nil {
		writeErrorStatus(w, http.StatusServiceUnavailable, errors.New("BAS workflow catalog is unavailable; trace promotion cannot be reported as persisted"))
		return
	}
	var request struct {
		ProjectID   string            `json:"project_id"`
		Name        string            `json:"name"`
		Description string            `json:"description,omitempty"`
		Assertions  []json.RawMessage `json:"assertions"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.ProjectID) == "" {
		writeErrorStatus(w, http.StatusBadRequest, errors.New("project_id is required to persist a promoted BAS workflow"))
		return
	}
	assertions, err := decodeActionDefinitions(request.Assertions)
	if err != nil {
		writeErrorStatus(w, http.StatusBadRequest, fmt.Errorf("decode typed assertions: %w", err))
		return
	}
	response, err := h.Broker.PromoteTrace(r.Context(), chi.URLParam(r, "sessionID"), PromoteRequest{
		Name: strings.TrimSpace(request.Name), Description: strings.TrimSpace(request.Description), Assertions: assertions,
	})
	if err == nil {
		response, err = PersistPromotedFlow(r.Context(), h.Workflows, h.ProjectScopes, request.ProjectID, response)
	}
	writeTraceDraftResponse(w, response, err)
}

func decodeActionDefinitions(encoded []json.RawMessage) ([]*actionsv1.ActionDefinition, error) {
	definitions := make([]*actionsv1.ActionDefinition, 0, len(encoded))
	for index, raw := range encoded {
		var action actionsv1.ActionDefinition
		if err := (protojson.UnmarshalOptions{}).Unmarshal(raw, &action); err != nil {
			return nil, fmt.Errorf("assertion %d: %w", index+1, err)
		}
		definitions = append(definitions, &action)
	}
	return definitions, nil
}

type traceDraftHTTPResponse struct {
	DraftDigest        string                 `json:"draft_digest"`
	Evidence           []TraceCompileEvidence `json:"evidence"`
	TraceID            string                 `json:"trace_id"`
	SessionID          string                 `json:"session_id"`
	WorkflowID         string                 `json:"workflow_id,omitempty"`
	WorkflowVersion    int32                  `json:"workflow_version,omitempty"`
	Definition         json.RawMessage        `json:"definition"`
	RequiredParameters []Parameter            `json:"required_parameters"`
	ActionCount        int                    `json:"action_count"`
}

func writeTraceDraftResponse(w http.ResponseWriter, response PromotedFlow, err error) {
	if err != nil {
		writeError(w, err)
		return
	}
	definition, err := protojson.Marshal(response.Definition)
	if err != nil {
		writeError(w, fmt.Errorf("encode typed workflow draft: %w", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(traceDraftHTTPResponse{
		DraftDigest: response.DraftDigest, Evidence: response.Evidence, TraceID: response.TraceID,
		SessionID: response.SessionID, WorkflowID: response.WorkflowID, WorkflowVersion: response.WorkflowVersion,
		Definition: definition, RequiredParameters: response.RequiredParameters, ActionCount: response.ActionCount,
	})
}

func (h *HTTPHandler) replayPromotedWorkflow(w http.ResponseWriter, r *http.Request) {
	if h.Workflows == nil {
		writeErrorStatus(w, http.StatusServiceUnavailable, errors.New("BAS workflow catalog is unavailable"))
		return
	}
	var request struct {
		WorkflowVersion int32             `json:"workflow_version"`
		Parameters      map[string]string `json:"parameters"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.WorkflowVersion <= 0 {
		writeErrorStatus(w, http.StatusBadRequest, errors.New("positive workflow_version is required for deterministic replay"))
		return
	}
	workflowID := chi.URLParam(r, "workflowID")
	loaded, err := LoadScopedPromotedWorkflow(r.Context(), h.Workflows, h.ProjectScopes, workflowID, request.WorkflowVersion)
	if err != nil {
		writeErrorStatus(w, http.StatusPreconditionFailed, errors.New("saved workflow is unavailable or project scope was denied"))
		return
	}
	result, err := h.Broker.ReplayPromotedFlow(r.Context(), chi.URLParam(r, "sessionID"), loaded.GetWorkflow().GetFlowDefinition(), request.Parameters)
	writeResponse(w, map[string]any{"result": result, "workflow_id": loaded.GetWorkflow().GetId(), "workflow_version": loaded.GetWorkflow().GetVersion()}, err)
}

func (h *HTTPHandler) close(w http.ResponseWriter, r *http.Request) {
	if err := h.Broker.Close(r.Context(), chi.URLParam(r, "sessionID")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandler) listHarnesses(w http.ResponseWriter, _ *http.Request) {
	if h.Harnesses == nil {
		writeError(w, errors.New("harness registry unavailable"))
		return
	}
	writeResponse(w, h.Harnesses.List(), nil)
}

func (h *HTTPHandler) qualify(w http.ResponseWriter, r *http.Request) {
	if h.Harnesses == nil {
		writeError(w, errors.New("harness registry unavailable"))
		return
	}
	borrowed, err := h.Broker.HarnessSession(chi.URLParam(r, "sessionID"))
	if err != nil {
		writeError(w, err)
		return
	}
	result := h.Harnesses.Qualify(r.Context(), chi.URLParam(r, "adapterID"), borrowed)
	writeResponse(w, result, nil)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, err)
		return false
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		writeError(w, errors.New("request body must contain one JSON value"))
		return false
	}
	return true
}

func writeResponse(w http.ResponseWriter, value any, err error) {
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, sessionpermission.ErrDenied) || strings.Contains(strings.ToLower(err.Error()), "authorize") || strings.Contains(strings.ToLower(err.Error()), "permission") {
		status = http.StatusForbidden
	}
	if errors.Is(err, ErrSessionNotOwned) {
		status = http.StatusNotFound
	}
	if errors.Is(err, ErrAuthorizerRequired) || errors.Is(err, ErrExtensionOwnerUnavailable) {
		status = http.StatusServiceUnavailable
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
