package board

import "content-desk/internal/module"

// Endpoints describes the board module's public surface. The single route is a
// deliberate browser REST exception: it executes the governed
// content-desk.board-read declared program, which the browser cannot invoke
// without reaching program-runtime directly.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "board_read",
		Path:        "/api/v1/board",
		Method:      "GET",
		Summary:     "Read the Content Desk marketing board",
		Description: "Executes the content-desk.board-read declared program and returns its envelope: current campaign/draft work and next actions, capability readiness gaps and offer release readiness.",
		Category:    "board",
		RESTException: &module.RESTException{
			Reason: module.RESTReasonBrowserSurface,
			Note:   "Browser-facing read of a governed declared program; Connect cannot invoke a declared program without exposing program-runtime to the browser.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
				Response: module.RESTPayload{Transport: "json", Conformance: "external_shape"},
				Error:    module.RESTPayload{Transport: "json", Conformance: "external_shape"},
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"status":   "string",
				"signals":  "object",
				"errors":   "[]object",
				"evidence": "[]string",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 503, Code: "unavailable", Description: "Program runtime or the declared board program is unavailable"},
			{Status: 502, Code: "internal", Description: "The board program returned a non-JSON envelope"},
		},
	},
}
