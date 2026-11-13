package externalurl

import "notification-hub/internal/module"

var Endpoints = []module.EndpointDescriptor{{
	ID:       "external_url",
	Path:     "/api/v1/external-url",
	Method:   "GET",
	Summary:  "Resolve an installed scenario's browser-openable URL.",
	Category: "system",
	Response: &module.Schema{Type: "Response", Properties: map[string]string{"scenario": "string", "url": "string"}},
	RESTException: &module.RESTException{
		Reason: module.RESTReasonOpsProbe,
		Note:   "Read-only URL lookup for UI navigation; returns an empty URL when the allowlisted scenario cannot be resolved.",
		ProtoPayloads: &module.RESTProtoPayloads{
			Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
			Response: module.RESTPayload{Transport: "json", Conformance: "external_shape"},
			Error:    module.RESTPayload{Transport: "json", Conformance: "external_shape"},
		},
	},
}}
