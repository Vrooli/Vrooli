package control

import (
	"encoding/json"
	"net/http"

	internal "device-control/internal/control"
	internalflows "device-control/internal/flows"
	"device-control/internal/module"
	"github.com/gorilla/mux"
)

type ModuleDeps struct{ Service *internal.Service }

func Module(s *internal.Service) module.Module {
	var anchors *internalflows.AnchorStore
	if s != nil && s.Anchors() != nil {
		anchors = s.Anchors()
	} else {
		anchors = internalflows.NewAnchorStore()
	}
	h := &handler{service: s, anchors: anchors}
	return module.Module{Name: "device-control", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/devices", h.listDevices).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/devices/discover", h.discoverDevices).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/devices/relations", h.deviceRelations).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/devices/relations/confirm", h.confirmRelation).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/devices/diagnostics", h.deviceDiagnostics).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/devices/{id}", h.describeDevice).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/devices/{id}", h.forgetDevice).Methods(http.MethodDelete)
		r.HandleFunc("/api/v1/devices/{id}/state", h.deviceState).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/devices/{id}/semantics", h.deviceSemantics).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/devices/{id}/pair/start", h.startPairDevice).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/devices/{id}/pair/complete", h.completePairDevice).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/devices/{id}/pair", h.pairDevice).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/devices/{id}/actuate", h.actuateDevice).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/devices/{id}/screen", h.screenDevice).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/devices/{id}/capture", h.captureDevice).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/devices/{id}/launch", h.launchApp).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/devices/{id}/install", h.installApp).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/devices/{id}/apps", h.appLifecycle).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/devices/{id}/volume", h.volumeDevice).Methods(http.MethodPost)
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
		r.HandleFunc("/api/v1/devices/onboard-network", h.onboardNetworkDevice).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/strategies", h.listStrategies).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/strategies/{id}/verify", h.verifyStrategy).Methods(http.MethodGet, http.MethodPost)
		r.HandleFunc("/api/v1/sessions", h.listSessions).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/sessions/acquire", h.acquire).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/sessions/{id}/kill", h.kill).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/sessions/{id}/release", h.release).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/sessions/{id}/validate", h.validateLease).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/recordings/start", h.startExternalRecording).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/recordings/inspect", h.inspectExternalRecording).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/recordings/stop", h.stopExternalRecording).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/recordings/concat", h.concatRecordings).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/flows/validate", h.validateFlow).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/flows/run", h.runFlow).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/flows/runs", h.liveFlowRuns).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/flows/runs/{id}/cache", h.forgetLiveFlowRun).Methods(http.MethodDelete)
		r.HandleFunc("/api/v1/flows/retained", h.retainedFlowRuns).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/flows/{id}/export", h.exportFlow).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/flows/{id}/retained", h.retainedFlowRun).Methods(http.MethodGet, http.MethodPost, http.MethodDelete)
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

type handler struct {
	service *internal.Service
	anchors *internalflows.AnchorStore
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
	{ID: "flow_runs_live", Path: "/api/v1/flows/runs", Method: "GET", Summary: "List process-local completed run IDs and active flow count", Category: "flows", RESTException: module.ThirdPartyJSONREST("GET")},
	{ID: "flow_run_cache_forget", Path: "/api/v1/flows/runs/{id}/cache", Method: "DELETE", Summary: "Forget one completed process-local run cache entry", Category: "flows", RESTException: module.ThirdPartyJSONREST("DELETE")},
	{ID: "flow_runs_retained", Path: "/api/v1/flows/retained", Method: "GET", Summary: "List explicitly retained run IDs", Category: "flows", RESTException: module.ThirdPartyJSONREST("GET")},
	{ID: "flow_run_retain", Path: "/api/v1/flows/{id}/retained", Method: "POST", Summary: "Explicitly retain a completed run and flow", Category: "flows", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "flow_run_retained_get", Path: "/api/v1/flows/{id}/retained", Method: "GET", Summary: "Read a retained run and flow", Category: "flows", RESTException: module.ThirdPartyJSONREST("GET")},
	{ID: "flow_run_retained_delete", Path: "/api/v1/flows/{id}/retained", Method: "DELETE", Summary: "Delete a retained run record", Category: "flows", RESTException: module.ThirdPartyJSONREST("DELETE")},
	{ID: "devices_list", Path: "/api/v1/devices", Method: "GET", Summary: "List devices and probed capabilities", Category: "devices", RESTException: module.ThirdPartyJSONREST("GET")},
	{ID: "devices_discover", Path: "/api/v1/devices/discover", Method: "GET", Summary: "Browse the LAN for DNS-SD device services", Category: "devices", RESTException: module.ThirdPartyJSONREST("GET")},
	{ID: "devices_semantics", Path: "/api/v1/devices/{id}/semantics", Method: "POST", Summary: "Read a bounded accessibility snapshot under a device lease", Category: "devices", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "devices_state", Path: "/api/v1/devices/{id}/state", Method: "GET", Summary: "Read live device state", Category: "devices", RESTException: module.ThirdPartyJSONREST("GET")},
	{ID: "devices_pair_start", Path: "/api/v1/devices/{id}/pair/start", Method: "POST", Summary: "Start the owner-present Android TV Remote pairing handshake", Category: "devices", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "devices_pair_complete", Path: "/api/v1/devices/{id}/pair/complete", Method: "POST", Summary: "Complete an owner-present Android TV Remote pairing handshake", Category: "devices", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "devices_pair", Path: "/api/v1/devices/{id}/pair", Method: "POST", Summary: "Pair a Google TV Android TV Remote transport", Category: "devices", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "devices_actuate", Path: "/api/v1/devices/{id}/actuate", Method: "POST", Summary: "Send one lease-owned direct device command", Category: "devices", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "devices_screen", Path: "/api/v1/devices/{id}/screen", Method: "POST", Summary: "Wake, sleep or prepare a device with verified state", Category: "devices", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "devices_capture", Path: "/api/v1/devices/{id}/capture", Method: "POST", Summary: "Capture a still or bounded recording through the flow owner", Category: "devices", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "devices_install", Path: "/api/v1/devices/{id}/install", Method: "POST", Summary: "Install a local APK, verify it, and optionally open and screenshot it", Category: "devices", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "devices_launch", Path: "/api/v1/devices/{id}/launch", Method: "POST", Summary: "Open an app by name or package and verify it reached the foreground", Category: "devices", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "devices_relations_confirm", Path: "/api/v1/devices/relations/confirm", Method: "POST", Summary: "Confirm a same-host transport candidate as one physical device", Category: "devices", RESTException: module.ThirdPartyJSONREST("POST")},
	{ID: "devices_app_lifecycle", Path: "/api/v1/devices/{id}/apps", Method: "POST", Summary: "Run one confirmed, lease-owned app lifecycle operation", Category: "devices", RESTException: module.ThirdPartyJSONREST("POST")},
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
	{ID: "devices_onboard_network", Path: "/api/v1/devices/onboard-network", Method: "POST", Summary: "Onboard a directly-addressable network ADB endpoint", Category: "devices", RESTException: module.ThirdPartyJSONREST("POST")},
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
