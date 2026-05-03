package notes

import "{{SCENARIO_ID}}/internal/module"

// Endpoints is the machine-readable description of every route this
// module mounts. The codegen at api/cmd/gen-endpoints reads this slice
// (plus health.Endpoints, plus future domains) to emit the canonical
// .vrooli/endpoints.json. Adding or removing a route here without
// regenerating fails the CI drift check.
//
// Field values mirror the wire contract exactly: paths and methods
// match what NewHandler registers; error codes use the wire strings
// of httpx.Code* constants (invalid_request, not_found, internal);
// cli_mapping commands match cli/domains/notes/register.go.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "notes_list",
		Path:        "/api/v1/notes",
		Method:      "GET",
		Summary:     "List notes",
		Description: "Returns up to 100 notes ordered newest-first by created_at. The wire envelope is the proto-generated ListNotesResponse; an empty list serialises to {} (proto3 default-omission).",
		Category:    "notes",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"notes": "array<Note>",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List notes", Curl: "curl http://localhost:${API_PORT}/api/v1/notes"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "{{SCENARIO_ID}} notes list",
		},
	},
	{
		ID:          "notes_create",
		Path:        "/api/v1/notes",
		Method:      "POST",
		Summary:     "Create a note",
		Description: "Persists a new note. Title is required (validated after whitespace trim by notes.Service); body is optional.",
		Category:    "notes",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"title": "string (required, non-whitespace)",
				"body":  "string",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"note": "Note",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_request", Description: "Missing or whitespace-only title; malformed JSON; unknown fields rejected by DisallowUnknownFields"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{
				Name: "Create note",
				Curl: "curl -X POST http://localhost:${API_PORT}/api/v1/notes -H 'Content-Type: application/json' -d '{\"title\":\"first\",\"body\":\"hello\"}'",
			},
		},
		CLIMapping: &module.CLIMapping{
			Command: "{{SCENARIO_ID}} notes create",
			Args:    []string{"--title", "<title>", "--body", "<body>"},
		},
	},
	{
		ID:          "notes_get",
		Path:        "/api/v1/notes/{id}",
		Method:      "GET",
		Summary:     "Get a note by id",
		Description: "Returns the note matching the path id, or a typed not_found envelope.",
		Category:    "notes",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"note": "Note",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No note with that id exists"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Get note", Curl: "curl http://localhost:${API_PORT}/api/v1/notes/abc123"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "{{SCENARIO_ID}} notes get",
			Args:    []string{"<id>"},
		},
	},
}
