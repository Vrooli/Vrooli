package control

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	internal "device-control/internal/control"
	internalflows "device-control/internal/flows"
	"device-control/internal/module"
	"device-control/strategy"
	"github.com/gorilla/mux"
)

type ModuleDeps struct{ Service *internal.Service }

func Module(s *internal.Service) module.Module {
	anchors := internalflows.NewAnchorStore()
	if s != nil && s.Anchors() != nil {
		anchors = s.Anchors()
	}
	h := &handler{service: s, anchors: anchors}
	return module.Module{Name: "device-control", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/devices", h.listDevices).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/devices/discover", h.discoverDevices).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/devices/{id}", h.describeDevice).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/devices/{id}", h.forgetDevice).Methods(http.MethodDelete)
		r.HandleFunc("/api/v1/devices/{id}/state", h.deviceState).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/devices/{id}/pair/start", h.startPairDevice).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/devices/{id}/pair/complete", h.completePairDevice).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/devices/{id}/pair", h.pairDevice).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/devices/{id}/actuate", h.actuateDevice).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/devices/{id}/merge", h.mergeDevice).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/devices/{id}/split", h.splitDevice).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/devices/{id}/events", h.deviceEvents).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/devices/{id}/webview/attach", h.attachWebView).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/auth/profiles", h.listAuthProfiles).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/auth/profiles", h.createAuthProfile).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/auth/profiles/{id}", h.getAuthProfile).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/auth/profiles/{id}", h.updateAuthProfile).Methods(http.MethodPut)
		r.HandleFunc("/api/v1/auth/profiles/{id}", h.revokeAuthProfile).Methods(http.MethodDelete)
		r.HandleFunc("/api/v1/auth/profiles/{id}/provision", h.provisionAuthCredential).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/auth/profiles/{id}/credential", h.deleteAuthCredential).Methods(http.MethodDelete)
		r.HandleFunc("/api/v1/auth/profiles/{id}/test", h.testAuthProfile).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/auth/unlock", h.unlockDevice).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/devices/{id}/promote", h.promoteDevice).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/devices/{id}/reconnect", h.reconnectDevice).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/conformance/android", h.androidConformancePlan).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/conformance/android/run", h.runAndroidConformance).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/devices/connect", h.connectDevice).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/strategies", h.listStrategies).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/strategies/{id}/verify", h.verifyStrategy).Methods(http.MethodGet, http.MethodPost)
		r.HandleFunc("/api/v1/sessions", h.listSessions).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/sessions/acquire", h.acquire).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/sessions/{id}/kill", h.kill).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/sessions/{id}/release", h.release).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/sessions/{id}/validate", h.validateLease).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/recordings/start", h.startExternalRecording).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/recordings/stop", h.stopExternalRecording).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/recordings/concat", h.concatRecordings).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/flows/validate", h.validateFlow).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/flows/run", h.runFlow).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/flows/{id}/export", h.exportFlow).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/evidence/audit", h.audit).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/evidence/{id}", h.artifact).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/anchors", h.listAnchors).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/anchors", h.createAnchor).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/anchors/{id}", h.deleteAnchor).Methods(http.MethodDelete)
		r.HandleFunc("/api/v1/agents", h.listAgents).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/agents/start", h.startAgent).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/agents/{id}/abort", h.abortAgent).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/agents/{id}/promote", h.promoteAgent).Methods(http.MethodPost)
		registerConnectServices(r, h)
	}, Endpoints: Endpoints}
}

func (h *handler) exportFlow(w http.ResponseWriter, r *http.Request) {
	export, err := h.service.ExportFlow(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "run_not_found", err.Error())
		return
	}
	write(w, http.StatusOK, export)
}

type handler struct {
	service *internal.Service
	anchors *internalflows.AnchorStore
}

func (h *handler) listDevices(w http.ResponseWriter, r *http.Request) {
	write(w, http.StatusOK, map[string]any{"devices": h.service.Devices(r.Context())})
}

func (h *handler) describeDevice(w http.ResponseWriter, r *http.Request) {
	device, err := h.service.DescribeDevice(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "device_not_found", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"device": device})
}

func (h *handler) deviceState(w http.ResponseWriter, r *http.Request) {
	state, err := h.service.ReadDeviceState(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusConflict, "device_state_unavailable", err.Error())
		return
	}
	write(w, http.StatusOK, state)
}

func (h *handler) discoverDevices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if seconds, parseErr := strconv.Atoi(r.URL.Query().Get("timeout_seconds")); parseErr == nil && seconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(seconds)*time.Second)
		defer cancel()
	}
	services, err := h.service.DiscoverLAN(ctx)
	if err != nil && len(services) == 0 {
		write(w, http.StatusServiceUnavailable, map[string]any{"services": services, "health": "unreachable", "reason": err.Error()})
		return
	}
	requested := r.URL.Query()["service"]
	if len(requested) > 0 {
		allowed := map[string]bool{}
		for _, service := range requested {
			allowed[strings.ToLower(strings.TrimSpace(service))] = true
		}
		filtered := services[:0]
		for _, service := range services {
			if allowed[strings.ToLower(service.Service)] {
				filtered = append(filtered, service)
			}
		}
		services = filtered
	}
	response := map[string]any{"services": services}
	if err != nil {
		response["health"] = "degraded"
		response["reason"] = err.Error()
	}
	write(w, http.StatusOK, response)
}

type pairRequest struct {
	PIN string `json:"pin"`
}

type pairCompleteRequest struct {
	PairingID string `json:"pairing_id"`
	PIN       string `json:"pin"`
}

func (h *handler) startPairDevice(w http.ResponseWriter, r *http.Request) {
	pairingID, err := h.service.BeginPairDevice(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusConflict, "pairing_start_failed", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"pairing_started": true, "pairing_id": pairingID, "device_id": mux.Vars(r)["id"], "transport": "android-tv-remote"})
}

func (h *handler) completePairDevice(w http.ResponseWriter, r *http.Request) {
	var in pairCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "pairing completion request is invalid")
		return
	}
	secret := []byte(strings.TrimSpace(in.PIN))
	defer func() {
		for i := range secret {
			secret[i] = 0
		}
	}()
	result, err := h.service.CompletePairDevice(r.Context(), mux.Vars(r)["id"], in.PairingID, secret)
	if err != nil {
		writeError(w, http.StatusConflict, "pairing_failed", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"paired": true, "device_id": mux.Vars(r)["id"], "outcome": result.Outcome, "transport": result.Transport, "detail": result.Detail})
}

func (h *handler) pairDevice(w http.ResponseWriter, r *http.Request) {
	var in pairRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "pairing request is invalid")
		return
	}
	secret := []byte(strings.TrimSpace(in.PIN))
	defer func() {
		for i := range secret {
			secret[i] = 0
		}
	}()
	result, err := h.service.PairDeviceSecret(r.Context(), mux.Vars(r)["id"], secret)
	if err != nil {
		writeError(w, http.StatusConflict, "pairing_failed", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"paired": true, "device_id": mux.Vars(r)["id"], "outcome": result.Outcome, "transport": result.Transport, "detail": result.Detail})
}

func (h *handler) actuateDevice(w http.ResponseWriter, r *http.Request) {
	var in struct {
		internal.DirectActuation
		Actor      string `json:"actor"`
		LeaseToken string `json:"lease_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "actuation request is invalid")
		return
	}
	record, err := h.service.ActuateDevice(r.Context(), mux.Vars(r)["id"], in.Actor, in.LeaseToken, in.DirectActuation)
	if err != nil {
		writeError(w, http.StatusConflict, leaseErrorCode(err), err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"audit": record, "interactive": true, "evidence_backed": false})
}

func (h *handler) mergeDevice(w http.ResponseWriter, r *http.Request) {
	var in struct {
		MemberID string `json:"member_id"`
		Claim    string `json:"claim"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "merge request is invalid")
		return
	}
	device, err := h.service.MergeDevices(r.Context(), mux.Vars(r)["id"], in.MemberID, in.Claim)
	if err != nil {
		writeError(w, http.StatusConflict, "identity_merge_failed", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"device": device, "merged": true})
}

func (h *handler) splitDevice(w http.ResponseWriter, r *http.Request) {
	devices, err := h.service.SplitDevice(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusConflict, "identity_split_failed", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"devices": devices, "split": true})
}

func (h *handler) deviceEvents(w http.ResponseWriter, r *http.Request) {
	subscription := h.service.SubscribeStateChanges(32)
	defer subscription.Cancel()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusNotImplemented, "stream_unavailable", "event streaming is unavailable")
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case event, open := <-subscription.Events:
			if !open {
				return
			}
			if event.DeviceID != mux.Vars(r)["id"] {
				continue
			}
			payload, err := json.Marshal(event)
			if err != nil {
				continue
			}
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(payload)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}
}

type webViewAttachRequest struct {
	Actor      string `json:"actor"`
	LeaseToken string `json:"lease_token"`
	Package    string `json:"package"`
}

func (h *handler) attachWebView(w http.ResponseWriter, r *http.Request) {
	var in webViewAttachRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	endpoint, err := h.service.AttachWebView(r.Context(), mux.Vars(r)["id"], in.Actor, in.LeaseToken, in.Package)
	if err != nil {
		writeError(w, http.StatusConflict, leaseErrorCode(err), err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"endpoint": endpoint})
}

func (h *handler) forgetDevice(w http.ResponseWriter, r *http.Request) {
	if !h.service.ForgetDeviceContext(r.Context(), mux.Vars(r)["id"]) {
		writeError(w, http.StatusNotFound, "device_not_found", "unknown device "+mux.Vars(r)["id"])
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) promoteDevice(w http.ResponseWriter, r *http.Request) {
	device, err := h.service.PromoteWireless(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusConflict, "wireless_promotion_failed", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"device": device, "transport": device.Transport})
}

func (h *handler) reconnectDevice(w http.ResponseWriter, r *http.Request) {
	device, err := h.service.ReconnectWireless(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusConflict, "wireless_reconnect_failed", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"device": device, "transport": device.Transport})
}

type connectRequest struct {
	Kind string `json:"kind"`
}

type androidConformanceRequest struct {
	DeviceID   string `json:"device_id"`
	Actor      string `json:"actor"`
	LeaseToken string `json:"lease_token,omitempty"`
}

func (h *handler) androidConformancePlan(w http.ResponseWriter, _ *http.Request) {
	write(w, http.StatusOK, h.service.AndroidCapabilitySelfTestPlan())
}

func (h *handler) runAndroidConformance(w http.ResponseWriter, r *http.Request) {
	var in androidConformanceRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := h.service.RunAndroidCapabilitySelfTest(r.Context(), in.DeviceID, in.Actor, in.LeaseToken)
	if err != nil {
		if strings.HasPrefix(err.Error(), "unknown device ") {
			writeError(w, http.StatusBadRequest, "unknown_device", err.Error())
			return
		}
		writeError(w, http.StatusConflict, leaseErrorCode(err), err.Error())
		return
	}
	write(w, http.StatusOK, result)
}

func (h *handler) connectDevice(w http.ResponseWriter, r *http.Request) {
	var in connectRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	rungs := h.service.OnboardingLive(r.Context(), in.Kind)
	next := ""
	wirelessReady := false
	for _, v := range rungs {
		if v["id"] == "wireless-adb" && v["status"] == "available" {
			wirelessReady = true
		}
	}
	for _, v := range rungs {
		if v["status"] != "available" && !(wirelessReady && v["id"] == "usb-bus") {
			next = v["next_action"]
			break
		}
	}
	write(w, http.StatusOK, map[string]any{"kind": strings.ToLower(in.Kind), "rungs": rungs, "first_next_action": next})
}

func (h *handler) listStrategies(w http.ResponseWriter, r *http.Request) {
	decls := h.service.Strategies(r.Context())
	out := make([]map[string]any, 0, len(decls))
	for _, d := range decls {
		// Tiers are derived from the declared capabilities. Some strategies
		// intentionally leave Declaration.Tiers empty so the capability model
		// remains the single source of truth.
		out = append(out, map[string]any{"id": d.StrategyID, "description": d.Description, "status": d.Status, "reason": d.Reason, "supported_host_os": append([]string{}, d.SupportedHostOS...), "capabilities": d.Capabilities, "tiers": strategy.Tiers(d), "executable_step_kinds": append([]string{}, strategy.StepKinds(d)...), "next_actions": append([]string{}, d.NextActions...), "promotable": d.Promotable, "evidence_class": d.EvidenceClass, "minimum_useful_fps": d.MinimumUsefulFPS})
	}
	write(w, http.StatusOK, map[string]any{"strategies": out})
}

func (h *handler) verifyStrategy(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	report, err := h.service.Verify(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	write(w, http.StatusOK, report)
}

func (h *handler) listSessions(w http.ResponseWriter, r *http.Request) {
	sessions := h.service.ListLiveSessionsContext(r.Context())
	views := make([]map[string]any, 0, len(sessions))
	for _, session := range sessions {
		views = append(views, publicSession(session))
	}
	write(w, http.StatusOK, map[string]any{"sessions": views})
}

func publicSession(session internal.Session) map[string]any {
	return map[string]any{
		"id": session.ID, "device_id": session.DeviceID, "actor": session.Actor,
		"state": session.State, "kill_reason": session.KillReason,
		"expires_at": session.ExpiresAt, "created_at": session.CreatedAt,
	}
}

type acquireRequest struct {
	DeviceID   string `json:"device_id"`
	Actor      string `json:"actor"`
	TTLSeconds int    `json:"ttl_seconds"`
}

type externalRecordingRequest struct {
	DeviceID   string `json:"device_id"`
	Actor      string `json:"actor"`
	LeaseToken string `json:"lease_token"`
	HandleID   string `json:"handle_id,omitempty"`
}

type reviewRecordingRequest struct {
	DeviceID     string   `json:"device_id"`
	Actor        string   `json:"actor"`
	LeaseToken   string   `json:"lease_token"`
	ReferenceIDs []string `json:"reference_ids"`
}

func (h *handler) startExternalRecording(w http.ResponseWriter, r *http.Request) {
	var in externalRecordingRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	handle, err := h.service.StartExternalRecording(r.Context(), in.DeviceID, in.Actor, in.LeaseToken)
	if err != nil {
		writeError(w, http.StatusConflict, leaseErrorCode(err), err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"handle": handle})
}

func (h *handler) stopExternalRecording(w http.ResponseWriter, r *http.Request) {
	var in externalRecordingRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := h.service.StopExternalRecording(r.Context(), in.DeviceID, in.Actor, in.LeaseToken, in.HandleID)
	if err != nil {
		writeError(w, http.StatusConflict, leaseErrorCode(err), err.Error())
		return
	}
	write(w, http.StatusOK, result)
}

func (h *handler) concatRecordings(w http.ResponseWriter, r *http.Request) {
	var in reviewRecordingRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := h.service.FinalizeReviewRecording(r.Context(), in.DeviceID, in.Actor, in.LeaseToken, in.ReferenceIDs)
	if err != nil {
		writeError(w, http.StatusConflict, leaseErrorCode(err), err.Error())
		return
	}
	write(w, http.StatusOK, result)
}

func (h *handler) acquire(w http.ResponseWriter, r *http.Request) {
	var in acquireRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	s, err := h.service.AcquireContext(r.Context(), in.DeviceID, in.Actor, time.Duration(in.TTLSeconds)*time.Second)
	if err != nil {
		if strings.HasPrefix(err.Error(), "unknown device ") {
			writeError(w, http.StatusBadRequest, "unknown_device", err.Error())
			return
		}
		writeError(w, http.StatusConflict, "lease_refused", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"session": s})
}

func (h *handler) kill(w http.ResponseWriter, r *http.Request) {
	s, err := h.service.KillContext(r.Context(), mux.Vars(r)["id"], "operator requested kill")
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"session": publicSession(s)})
}

func (h *handler) release(w http.ResponseWriter, r *http.Request) {
	s, err := h.service.ReleaseContext(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"session": publicSession(s)})
}

func (h *handler) validateLease(w http.ResponseWriter, r *http.Request) {
	var in struct {
		DeviceID   string `json:"device_id"`
		LeaseToken string `json:"lease_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if err := h.service.ValidateLease(r.Context(), in.DeviceID, in.LeaseToken); err != nil {
		writeError(w, http.StatusConflict, leaseErrorCode(err), err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"valid": true})
}

type flowRequest struct {
	Flow       internal.Flow `json:"flow"`
	StrategyID string        `json:"strategy_id"`
	DeviceID   string        `json:"device_id"`
	Actor      string        `json:"actor"`
	LeaseToken string        `json:"lease_token,omitempty"`
}

func (h *handler) validateFlow(w http.ResponseWriter, r *http.Request) {
	var in flowRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	write(w, http.StatusOK, h.service.Validate(r.Context(), in.Flow, in.StrategyID))
}

func (h *handler) runFlow(w http.ResponseWriter, r *http.Request) {
	var in flowRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := h.service.RunWithLease(r.Context(), in.Flow, in.DeviceID, in.Actor, in.LeaseToken)
	if err != nil {
		if strings.HasPrefix(err.Error(), "unknown device ") {
			writeError(w, http.StatusBadRequest, "unknown_device", err.Error())
			return
		}
		writeError(w, http.StatusConflict, leaseErrorCode(err), err.Error())
		return
	}
	write(w, http.StatusOK, result)
}

func leaseErrorCode(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "lease expired"):
		return "lease_expired"
	case strings.Contains(message, "lease token is invalid"), strings.Contains(message, "lease is "):
		return "lease_invalid"
	case strings.Contains(message, "bound to device"):
		return "lease_device_mismatch"
	default:
		return "run_failed"
	}
}

func (h *handler) audit(w http.ResponseWriter, r *http.Request) {
	write(w, http.StatusOK, map[string]any{"records": h.service.AuditContext(r.Context())})
}

func (h *handler) artifact(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	data, kind, err := h.service.ArtifactContext(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "artifact_not_found", err.Error())
		return
	}
	contentType := http.DetectContentType(data)
	switch kind {
	case "image":
		if strings.HasPrefix(contentType, "application/octet-stream") {
			contentType = "image/png"
		}
	case "video":
		contentType = "video/mp4"
	case "log":
		contentType = "text/plain; charset=utf-8"
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *handler) listAnchors(w http.ResponseWriter, _ *http.Request) {
	write(w, http.StatusOK, map[string]any{"anchors": h.anchors.List()})
}

func (h *handler) createAnchor(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name       string    `json:"name"`
		Target     string    `json:"target"`
		Bounds     []float64 `json:"bounds"`
		Confidence float64   `json:"confidence"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	a, err := h.anchors.Create(in.Name, in.Target, in.Bounds, in.Confidence)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_anchor", err.Error())
		return
	}
	write(w, http.StatusCreated, map[string]any{"anchor": a})
}

func (h *handler) deleteAnchor(w http.ResponseWriter, r *http.Request) {
	if err := h.anchors.Delete(mux.Vars(r)["id"]); err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) listAgents(w http.ResponseWriter, _ *http.Request) {
	write(w, http.StatusOK, map[string]any{"agents": h.service.ListAgents()})
}

func (h *handler) startAgent(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Goal           string `json:"goal"`
		DeviceID       string `json:"device_id"`
		Actor          string `json:"actor"`
		SkillAvailable bool   `json:"skill_available"`
		DryRun         bool   `json:"dry_run"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	a, err := h.service.StartAgentWithOptions(r.Context(), in.Goal, in.DeviceID, in.Actor, in.SkillAvailable, in.DryRun)
	if err != nil {
		writeError(w, http.StatusPreconditionFailed, "agent_unavailable", err.Error())
		return
	}
	write(w, http.StatusAccepted, map[string]any{"agent": a})
}

func (h *handler) abortAgent(w http.ResponseWriter, r *http.Request) {
	a, err := h.service.AbortAgent(mux.Vars(r)["id"], "operator requested abort")
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"agent": a})
}

func (h *handler) promoteAgent(w http.ResponseWriter, r *http.Request) {
	a, err := h.service.PromoteAgent(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusConflict, "promotion_refused", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"agent": a})
}

func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	write(w, status, map[string]any{"status": "failed", "code": code, "message": message})
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "devices_list", Path: "/api/v1/devices", Method: "GET", Summary: "List devices and probed capabilities", Category: "devices", RESTException: module.ThirdPartyJSONREST("GET")},
	{ID: "devices_discover", Path: "/api/v1/devices/discover", Method: "GET", Summary: "Browse the LAN for DNS-SD device services", Category: "devices", RESTException: module.ThirdPartyJSONREST("GET")},
	{ID: "devices_state", Path: "/api/v1/devices/{id}/state", Method: "GET", Summary: "Read live device state", Category: "devices", RESTException: module.ThirdPartyJSONREST("GET")},
	{ID: "devices_pair_start", Path: "/api/v1/devices/{id}/pair/start", Method: "POST", Summary: "Start the owner-present Android TV Remote pairing handshake", Category: "devices", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "devices_pair_complete", Path: "/api/v1/devices/{id}/pair/complete", Method: "POST", Summary: "Complete an owner-present Android TV Remote pairing handshake", Category: "devices", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "devices_pair", Path: "/api/v1/devices/{id}/pair", Method: "POST", Summary: "Pair a Google TV Android TV Remote transport", Category: "devices", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "devices_actuate", Path: "/api/v1/devices/{id}/actuate", Method: "POST", Summary: "Send one lease-owned direct device command", Category: "devices", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "devices_merge", Path: "/api/v1/devices/{id}/merge", Method: "POST", Summary: "Merge identities under an owner-asserted claim", Category: "devices", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "devices_split", Path: "/api/v1/devices/{id}/split", Method: "POST", Summary: "Split a previously merged identity", Category: "devices", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "devices_events", Path: "/api/v1/devices/{id}/events", Method: "GET", Summary: "Stream device state changes over SSE", Category: "devices", RESTException: module.ThirdPartyJSONREST("GET")},
	{ID: "webview_attach", Path: "/api/v1/devices/{id}/webview/attach", Method: "POST", Summary: "Attach to an application WebView under a device lease", Category: "devices", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "auth_profiles_list", Path: "/api/v1/auth/profiles", Method: "GET", Summary: "List authentication profiles", Category: "auth", RESTException: module.ThirdPartyJSONREST("GET")},
	{ID: "auth_profile_create", Path: "/api/v1/auth/profiles", Method: "POST", Summary: "Create a reference-only authentication profile", Category: "auth", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "auth_profile_get", Path: "/api/v1/auth/profiles/{id}", Method: "GET", Summary: "Inspect an authentication profile", Category: "auth", RESTException: module.ThirdPartyJSONREST("GET")},
	{ID: "auth_profile_update", Path: "/api/v1/auth/profiles/{id}", Method: "PUT", Summary: "Update authentication profile metadata", Category: "auth", RESTException: module.ThirdPartyJSONREST("PUT")},
	{ID: "auth_profile_revoke", Path: "/api/v1/auth/profiles/{id}", Method: "DELETE", Summary: "Revoke an authentication profile", Category: "auth", RESTException: module.ThirdPartyJSONREST("DELETE")},
	{ID: "auth_profile_provision", Path: "/api/v1/auth/profiles/{id}/provision", Method: "POST", Summary: "Provision a credential from stdin", Category: "auth", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "auth_profile_delete_credential", Path: "/api/v1/auth/profiles/{id}/credential", Method: "DELETE", Summary: "Delete an authority-held credential", Category: "auth", RESTException: module.ThirdPartyJSONREST("DELETE")},
	{ID: "auth_profile_test", Path: "/api/v1/auth/profiles/{id}/test", Method: "POST", Summary: "Check authentication provider readiness", Category: "auth", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "auth_unlock", Path: "/api/v1/auth/unlock", Method: "POST", Summary: "Unlock a device and verify the live postcondition", Category: "auth", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "flow_export", Path: "/api/v1/flows/{id}/export", Method: "GET", Summary: "Export a completed run as a replayable flow", Category: "flows", RESTException: module.ThirdPartyJSONREST("GET")},
	{ID: "devices_forget", Path: "/api/v1/devices/{id}", Method: "DELETE", Summary: "Forget a retained device identity", Category: "devices", RESTException: module.ThirdPartyJSONREST("DELETE")},
	{ID: "devices_connect", Path: "/api/v1/devices/connect", Method: "POST", Summary: "Show guided device onboarding", Category: "devices", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "android_conformance_plan", Path: "/api/v1/conformance/android", Method: "GET", Summary: "Describe the physical Android conformance plan", Category: "conformance", RESTException: module.ThirdPartyJSONREST("GET")},
	{ID: "android_conformance_run", Path: "/api/v1/conformance/android/run", Method: "POST", Summary: "Run physical Android conformance against a fixture", Category: "conformance", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "strategies_list", Path: "/api/v1/strategies", Method: "GET", Summary: "List strategy dispositions", Category: "strategies", RESTException: module.ThirdPartyJSONREST("GET")},
	{ID: "strategy_verify", Path: "/api/v1/strategies/{id}/verify", Method: "GET", Summary: "Run fixed strategy conformance", Category: "strategies", RESTException: module.ThirdPartyJSONREST("GET")},
	{ID: "sessions_list", Path: "/api/v1/sessions", Method: "GET", Summary: "List live leases", Category: "sessions", RESTException: module.ThirdPartyJSONREST("GET")},
	{ID: "sessions_acquire", Path: "/api/v1/sessions/acquire", Method: "POST", Summary: "Acquire an exclusive lease", Category: "sessions", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "session_kill", Path: "/api/v1/sessions/{id}/kill", Method: "POST", Summary: "Immediately kill a session", Category: "sessions", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "session_release", Path: "/api/v1/sessions/{id}/release", Method: "POST", Summary: "Release a lease", Category: "sessions", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "flow_validate", Path: "/api/v1/flows/validate", Method: "POST", Summary: "Preflight a flow against a strategy", Category: "flows", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "flow_run", Path: "/api/v1/flows/run", Method: "POST", Summary: "Run a bounded flow", Category: "flows", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "audit_list", Path: "/api/v1/evidence/audit", Method: "GET", Summary: "List device verb audit records", Category: "evidence", RESTException: module.ThirdPartyJSONREST("GET")},
	{ID: "artifact_get", Path: "/api/v1/evidence/{id}", Method: "GET", Summary: "Read a retained evidence artifact", Category: "evidence", RESTException: module.ThirdPartyJSONREST("GET")},
	{ID: "anchors_list", Path: "/api/v1/anchors", Method: "GET", Summary: "List saved visual anchors", Category: "flows", RESTException: module.ThirdPartyJSONREST("GET")},
	{ID: "anchors_create", Path: "/api/v1/anchors", Method: "POST", Summary: "Create a normalized visual anchor", Category: "flows", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "anchors_delete", Path: "/api/v1/anchors/{id}", Method: "DELETE", Summary: "Delete a visual anchor", Category: "flows", RESTException: module.ThirdPartyJSONREST("DELETE")},
	{ID: "agents_list", Path: "/api/v1/agents", Method: "GET", Summary: "List deterministic agent runs", Category: "agents", RESTException: &module.RESTException{
		Reason: module.RESTReasonThirdPartyShape,
		Note:   "Agent-run inventory is a scenario-owned JSON projection until the agent-run proto is published.",
		ProtoPayloads: &module.RESTProtoPayloads{
			Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
			Response: module.RESTPayload{Transport: "json", Conformance: "external_shape"},
			Error:    module.RESTPayload{Transport: "json", Conformance: "external_shape"},
		},
	}},
	{ID: "agents_start", Path: "/api/v1/agents/start", Method: "POST", Summary: "Start a skill-gated agent run", Category: "agents", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "agent_abort", Path: "/api/v1/agents/{id}/abort", Method: "POST", Summary: "Abort an agent run", Category: "agents", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "agent_promote", Path: "/api/v1/agents/{id}/promote", Method: "POST", Summary: "Promote a passing agent run", Category: "agents", RESTException: module.ThirdPartyJSONREST("POST")},
}
