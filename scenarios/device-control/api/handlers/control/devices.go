package control

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	internal "device-control/internal/control"
	"device-control/strategy"
	"github.com/gorilla/mux"
)

func (h *handler) listDevices(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("cached") == "true" {
		write(w, http.StatusOK, map[string]any{"devices": h.service.CachedDevices()})
		return
	}
	write(w, http.StatusOK, map[string]any{"devices": h.service.Devices(r.Context())})
}

func (h *handler) deviceRelations(w http.ResponseWriter, r *http.Request) {
	write(w, http.StatusOK, map[string]any{"candidates": h.service.CorrelationCandidates(r.Context())})
}

func (h *handler) deviceDiagnostics(w http.ResponseWriter, r *http.Request) {
	write(w, http.StatusOK, map[string]any{"devices": h.service.DiagnosticDevices(r.Context())})
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

func (h *handler) deviceSemantics(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Actor      string `json:"actor"`
		LeaseToken string `json:"lease_token"`
		Transport  string `json:"transport"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "semantic observation request is invalid")
		return
	}
	result, err := h.service.ObserveSemantics(r.Context(), mux.Vars(r)["id"], in.Actor, in.LeaseToken, in.Transport)
	if err != nil {
		writeError(w, http.StatusConflict, "semantic_observation_unavailable", err.Error())
		return
	}
	write(w, http.StatusOK, result)
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

func (h *handler) appLifecycle(w http.ResponseWriter, r *http.Request) {
	var in internal.AppLifecycleOperation
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "app lifecycle request is invalid")
		return
	}
	response, err := h.service.ExecuteAppLifecycle(r.Context(), mux.Vars(r)["id"], in)
	if err != nil {
		code, status := "app_lifecycle_failed", http.StatusConflict
		if strings.Contains(err.Error(), "requires explicit confirmation") {
			code, status = "confirmation_required", http.StatusPreconditionRequired
		} else if errors.As(err, new(*strategy.UnsupportedCapabilityError)) || errors.As(err, new(*strategy.AvailabilityError)) {
			code = "capability_unavailable"
		}
		write(w, status, map[string]any{"status": "failed", "code": code, "message": err.Error(), "response": response})
		return
	}
	write(w, http.StatusOK, response)
}

func (h *handler) confirmRelation(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CandidateID string `json:"candidate_id"`
		Force       bool   `json:"force"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "relation confirmation request is invalid")
		return
	}
	confirmation, err := h.service.ConfirmRelation(r.Context(), in.CandidateID, in.Force)
	if err != nil {
		writeError(w, http.StatusConflict, "relation_confirm_failed", err.Error())
		return
	}
	write(w, http.StatusOK, confirmation)
}

func (h *handler) volumeDevice(w http.ResponseWriter, r *http.Request) {
	var in internal.VolumeRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "volume request is invalid")
		return
	}
	in.Device = mux.Vars(r)["id"]
	result, err := h.service.ExecuteVolume(r.Context(), in)
	if err != nil {
		status := http.StatusConflict
		code := "volume_operation_failed"
		var typed *internal.VolumeError
		if errors.As(err, &typed) && typed.Class == "invalid_input" {
			status, code = http.StatusBadRequest, "invalid_request"
		}
		write(w, status, map[string]any{"result": result, "error": map[string]any{"code": code, "message": safeVolumeError(err)}})
		return
	}
	write(w, http.StatusOK, result)
}

func safeVolumeError(err error) string {
	if err == nil {
		return "volume operation failed"
	}
	var typed *internal.VolumeError
	if errors.As(err, &typed) {
		return typed.Message
	}
	var availability *strategy.AvailabilityError
	if errors.As(err, &availability) && availability.Reason != "" {
		return availability.Reason
	}
	if errors.Is(err, internal.ErrVolumeSessionLost) {
		return "volume session was refreshed once but the transport did not complete the operation"
	}
	return "volume operation failed; inspect the typed result and next action"
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
	Transport  string `json:"transport,omitempty"`
}

func (h *handler) attachWebView(w http.ResponseWriter, r *http.Request) {
	var in webViewAttachRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	endpoint, err := h.service.AttachWebView(r.Context(), mux.Vars(r)["id"], in.Actor, in.LeaseToken, in.Package, in.Transport)
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

type onboardNetworkRequest struct {
	Endpoint string `json:"endpoint"`
	AttachTo string `json:"attach_to,omitempty"`
}

// onboardNetworkDevice adopts a directly-addressable Android ADB endpoint
// (classic TCP 5555 for a TV/box, or Wireless Debugging/TLS) as a governed
// device. The durable identity is derived from the verified hardware serial, so
// the network address is never treated as the device identity.
func (h *handler) onboardNetworkDevice(w http.ResponseWriter, r *http.Request) {
	var in onboardNetworkRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	device, err := h.service.OnboardNetworkADB(r.Context(), in.Endpoint)
	if err != nil {
		writeError(w, http.StatusConflict, "network_onboarding_failed", err.Error())
		return
	}
	response := map[string]any{"device": device, "transport": device.Transport}
	if attachTo := strings.TrimSpace(in.AttachTo); attachTo != "" {
		confirmation, confirmErr := h.service.AttachNetworkTransport(r.Context(), device.ID, attachTo)
		if confirmErr != nil {
			response["attach_error"] = confirmErr.Error()
			response["next_actions"] = []string{"device-control device relations", "device-control device relation-confirm <candidate-id>"}
		} else {
			response["attached"] = confirmation
		}
	}
	write(w, http.StatusOK, response)
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
