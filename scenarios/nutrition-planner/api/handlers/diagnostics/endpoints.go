package diagnostics

import "nutrition-planner/internal/module"

var Endpoints = []module.EndpointDescriptor{{
	ID: "diagnostics_report", Path: "/api/v1/diagnostics", Method: "GET",
	Summary: "Report scoped operational and data-health findings", Category: "operations",
	RESTException: &module.RESTException{
		Reason: module.RESTReasonOpsProbe,
		Note:   "Authenticated operator and workspace data-health report; no generated RPC is required.",
		ProtoPayloads: &module.RESTProtoPayloads{
			Request:  module.RESTPayload{Transport: "query", Conformance: "external_shape"},
			Response: module.RESTPayload{Transport: "json", Conformance: "external_shape"},
			Error:    module.RESTPayload{Transport: "text", Conformance: "external_shape"},
		},
	},
}}
