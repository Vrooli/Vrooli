package notes

import (
	"database/sql"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/connectx"

	notesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/{{SCENARIO_ID}}/v1/notes/notes_v1connect"

	"{{SCENARIO_ID}}/internal/clock"
	"{{SCENARIO_ID}}/internal/module"
	internalnotes "{{SCENARIO_ID}}/internal/notes"
)

// Module returns the notes domain's contribution to the API: the generated
// Connect-RPC service handler plus the deliberate REST multipart exception.
//
// Adding a real domain to a scenario means copying this file into
// handlers/<dom>/module.go and pointing it at <dom>'s proto-generated handler
// and service. The center (server.New) does not change.
func Module(db *sql.DB, clk clock.Clock, blobs blobstore.BlobStore, logger *log.Logger) module.Module {
	repo := internalnotes.NewSQLiteRepository(db, clk)
	attachmentsRepo := internalnotes.NewSQLiteAttachmentsRepository(db, clk)
	svc := internalnotes.NewService(repo)
	attachmentsSvc := internalnotes.NewAttachmentsService(repo, attachmentsRepo)
	connectPath, connectHandler := notesconnect.NewNotesHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	attachmentsHandler := NewAttachmentsHandler(AttachmentsDeps{
		Service: attachmentsSvc,
		Store:   blobs,
		Logger:  logger,
	})
	return module.Module{
		Name: "notes",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
			r.PathPrefix("/api/v1/notes").Handler(attachmentsHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internalnotes.Schema so the modules registry can
// collect both endpoint descriptors and schema from one symbol per
// handler package. Keeps the registry's per-domain shape uniform:
// handlers/<dom>/{Module, Endpoints, Schema}.
func Schema() string { return internalnotes.Schema() }

// Endpoints is the machine-readable description of the notes module's public
// surface. Connect-RPC method paths come from the proto service descriptor at
// runtime; this slice exists for .vrooli/endpoints.json documentation and for
// the REST multipart exception that is not expressible as a Connect service.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "notes_list",
		Path:        "/vrooli.{{SCENARIO_ID_SNAKE}}.v1.notes.Notes/List",
		Method:      "POST",
		Summary:     "List notes",
		Description: "Returns up to 100 notes ordered newest-first by created_at through the generated Connect-RPC Notes service.",
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
			{Name: "List notes", Curl: "curl http://localhost:${API_PORT}/vrooli.{{SCENARIO_ID_SNAKE}}.v1.notes.Notes/List -H 'Content-Type: application/json' -d '{}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "{{SCENARIO_ID}} notes list",
		},
	},
	{
		ID:          "notes_create",
		Path:        "/vrooli.{{SCENARIO_ID_SNAKE}}.v1.notes.Notes/Create",
		Method:      "POST",
		Summary:     "Create a note",
		Description: "Persists a new note. Title is required and validated by notes.Service; body is optional.",
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
			{Status: 400, Code: "invalid_argument", Description: "Missing or whitespace-only title"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Create note", Curl: "curl http://localhost:${API_PORT}/vrooli.{{SCENARIO_ID_SNAKE}}.v1.notes.Notes/Create -H 'Content-Type: application/json' -d '{\"title\":\"first\",\"body\":\"hello\"}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "{{SCENARIO_ID}} notes create",
			Args:    []string{"--title", "<title>", "--body", "<body>"},
		},
	},
	{
		ID:          "notes_get",
		Path:        "/vrooli.{{SCENARIO_ID_SNAKE}}.v1.notes.Notes/Get",
		Method:      "POST",
		Summary:     "Get a note by id",
		Description: "Returns the note matching the request id through the generated Connect-RPC Notes service.",
		Category:    "notes",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"id": "string",
			},
		},
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
			{Name: "Get note", Curl: "curl http://localhost:${API_PORT}/vrooli.{{SCENARIO_ID_SNAKE}}.v1.notes.Notes/Get -H 'Content-Type: application/json' -d '{\"id\":\"abc123\"}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "{{SCENARIO_ID}} notes get",
			Args:    []string{"<id>"},
		},
	},
	{
		ID:          "notes_attach",
		Path:        "/api/v1/notes/{id}/attachments",
		Method:      "POST",
		Summary:     "Upload a note attachment",
		Description: "Uploads opaque file bytes through the documented REST multipart exception and returns proto-typed attachment metadata.",
		Category:    "notes",
		Request: &module.Schema{
			Type: "multipart/form-data",
			Properties: map[string]string{
				"file": "file (required)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"attachment": "Attachment",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_request", Description: "Missing file or invalid multipart upload"},
			{Status: 404, Code: "not_found", Description: "No note with that id exists"},
			{Status: 500, Code: "internal", Description: "BlobStore or metadata persistence failure"},
		},
		Examples: []module.Example{
			{Name: "Attach file", Curl: "curl -X POST http://localhost:${API_PORT}/api/v1/notes/abc123/attachments -F file=@./example.png"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "{{SCENARIO_ID}} notes attach",
			Args:    []string{"<id>", "--file", "<path>"},
		},
	},
}
