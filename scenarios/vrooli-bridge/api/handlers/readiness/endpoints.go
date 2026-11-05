package readiness

import "vrooli-bridge/internal/module"

var Endpoints = []module.EndpointDescriptor{{
	ID: "bridge-readiness", Path: "/api/v1/readiness", Method: "GET", Category: "system",
	Summary: "Bridge host readiness", Description: "Owner-visible canonical endpoint, fixed port, local API check, and latest durable candidate-admission evidence.",
	Response:      &module.Schema{Type: "BridgeReadinessResult", Properties: map[string]string{"status": "string", "endpoint": "string", "port": "integer", "reachability_mode": "string", "local_api": "boolean", "checks": "object", "last_candidate": "object", "firewall": "object"}},
	RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Owner-only readiness inspection; no mutation or generated workflow contract.", ProtoPayloads: &module.RESTProtoPayloads{Request: module.RESTPayload{Transport: "none", Conformance: "none"}, Response: module.RESTPayload{Transport: "json", Conformance: "external_shape"}, Error: module.RESTPayload{Transport: "json", Conformance: "external_shape"}}},
}, {
	ID: "bridge-readiness-endpoint", Path: "/api/v1/readiness/endpoint", Method: "PUT", Category: "system",
	Summary: "Configure Bridge advertised endpoint", Description: "Owner-only durable endpoint configuration used as the default for future onboarding; explicit onboarding endpoints still win.",
	Request:       &module.Schema{Type: "BridgeEndpointConfig", Properties: map[string]string{"endpoint": "string", "reachability_mode": "string"}},
	Response:      &module.Schema{Type: "BridgeReadinessResult", Properties: map[string]string{"status": "string", "endpoint": "string", "reachability_mode": "string"}},
	RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Owner-only host configuration; this REST operation has a dedicated CLI wrapper.", ProtoPayloads: &module.RESTProtoPayloads{Request: module.RESTPayload{Transport: "json", Conformance: "external_shape"}, Response: module.RESTPayload{Transport: "json", Conformance: "external_shape"}, Error: module.RESTPayload{Transport: "json", Conformance: "external_shape"}}},
}, {
	ID: "bridge-readiness-firewall", Path: "/api/v1/readiness/firewall", Method: "POST", Category: "system",
	Summary: "Inspect or remediate exact Bridge UFW admission", Description: "Owner-only confirmed action through the setup-managed local privilege broker. Only the candidate source IP and Bridge port 18767 are accepted.",
	Request:       &module.Schema{Type: "BridgeFirewallAction", Properties: map[string]string{"action": "preview|inspect|verify|allow|revoke", "candidate_ip": "string", "confirm": "boolean"}},
	Response:      &module.Schema{Type: "BridgeFirewallActionResult", Properties: map[string]string{"status": "string", "changed": "boolean", "evidence": "object"}},
	RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Owner-only typed host action; the broker rejects arbitrary commands and this wrapper requires explicit confirmation for mutation.", ProtoPayloads: &module.RESTProtoPayloads{Request: module.RESTPayload{Transport: "json", Conformance: "external_shape"}, Response: module.RESTPayload{Transport: "json", Conformance: "external_shape"}, Error: module.RESTPayload{Transport: "json", Conformance: "external_shape"}}},
}}
