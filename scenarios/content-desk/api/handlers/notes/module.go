package notes

import (
	"fmt"
	"log"
	"content-desk/internal/clock"
	"content-desk/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"

	notesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/notes/notes_v1connect"

	internalnotes "content-desk/internal/notes"
)

// Module returns the notes domain's contribution to the API: the generated
// Connect-RPC service handler plus the deliberate REST multipart exception.
//
// Production callers use this entry point; it owns its own blob storage so
// scenarios that don't use notes don't carry an orphan blobstore in
// api/main.go.
// Tests inject an in-memory blob store via ModuleWithBlobStore.
//
// Adding a real domain to a scenario means copying this file into
// handlers/<dom>/module.go and pointing it at <dom>'s proto-generated handler
// and service. The center (server.New) does not change.
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	blobs, err := defaultBlobStore()
	if err != nil {
		logger.Fatalf("notes attachments storage: %v", err)
	}
	return ModuleWithBlobStore(db, clk, blobs, logger)
}

// ModuleWithBlobStore is the explicit-injection variant used by tests
// (typically with blobstore.NewMemoryBlobStore()) and by callers that
// want to swap the blob backend.
func ModuleWithBlobStore(db *database.RoutedDB, clk clock.Clock, blobs blobstore.BlobStore, logger *log.Logger) module.Module {
	repo := internalnotes.NewSQLiteRepository(db, clk)
	attachmentsRepo := internalnotes.NewSQLiteAttachmentsRepository(db, clk)
	svc := internalnotes.NewService(repo)
	attachmentsSvc := internalnotes.NewAttachmentsService(repo, attachmentsRepo)
	connectPath, connectHandler := notesconnect.NewNotesServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Clock:   clk,
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

// defaultBlobStore resolves the storage-steer-mandated attachments
// directory and returns a filesystem-backed blobstore rooted there.
// Lives in this package so attachments storage travels with the notes
// domain — scenarios that don't ship notes never see a "blobstore"
// import in main.go.
func defaultBlobStore() (blobstore.BlobStore, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return nil, fmt.Errorf("create storage resolver: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: "content-desk"},
		storage.ClassData,
		"attachments",
	)
	if err != nil {
		return nil, fmt.Errorf("resolve attachments path: %w", err)
	}
	return blobstore.NewFilesystemBlobStore(path), nil
}

// Schema re-exports internalnotes.Schema so the modules registry can
// collect both endpoint descriptors and schema from one symbol per
// handler package. Keeps the registry's per-domain shape uniform:
// handlers/<dom>/{Module, Endpoints, Schema}.
func Schema() string { return internalnotes.Schema() }

// Endpoints is the machine-readable description of the notes module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants from notesconnect, so adding or renaming an RPC in notes.proto
// breaks this file at compile time. The complementary global parity test
// (TestProtoConnectParity in api/internal/modules/registry_test.go) walks
// the proto FileDescriptor and asserts every rpc has exactly one entry
// here — that runs automatically for every domain registered in
// modules.AllProtoFiles().
//
// notes_attach is the canonical worked example of a REST exception. It
// carries RESTException{Reason: RESTReasonMultipartUpload} so the
// validateTransport pass in gen-endpoints accepts the hand-authored Path.
// Proto is still the source of truth for the response payload (the
// Attachment message is proto-typed); only the request transport is REST
// because multipart/form-data cannot ride a proto Connect call. The four
// allowed RESTReason values are defined in api/internal/module/module.go;
// adding a fifth requires an architectural decision, not a quick fix.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "notes_list",
		Path:        notesconnect.NotesServiceListNotesProcedure,
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
			{Name: "List notes", Curl: "curl http://localhost:${API_PORT}/vrooli.content_desk.v1.notes.NotesService/ListNotes -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "notes_create",
		Path:        notesconnect.NotesServiceCreateNoteProcedure,
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
			{Name: "Create note", Curl: "curl http://localhost:${API_PORT}/vrooli.content_desk.v1.notes.NotesService/CreateNote -H 'Content-Type: application/json' -d '{\"title\":\"first\",\"body\":\"hello\"}'"},
		},
	},
	{
		ID:          "notes_get",
		Path:        notesconnect.NotesServiceGetNoteProcedure,
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
			{Name: "Get note", Curl: "curl http://localhost:${API_PORT}/vrooli.content_desk.v1.notes.NotesService/GetNote -H 'Content-Type: application/json' -d '{\"id\":\"abc123\"}'"},
		},
	},
	{
		ID:          "notes_count",
		Path:        notesconnect.NotesServiceCountNotesProcedure,
		Method:      "POST",
		Summary:     "Count notes in a time window",
		Description: "Counts notes created within the requested canonical time window. Bound to the `notes count` measure (read-only, run-eligible) so search-hub can answer analytical questions like \"how many notes were created this week\".",
		Category:    "notes",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"window": "TimeWindow (vrooli.measures.v1; e.g. {\"token\":\"TIME_WINDOW_TOKEN_THIS_WEEK\"})",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"count": "int64 (notes created in the window)",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Unresolvable or malformed time window"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Count this week", Curl: "curl http://localhost:${API_PORT}/vrooli.content_desk.v1.notes.NotesService/CountNotes -H 'Content-Type: application/json' -d '{\"window\":{\"token\":\"TIME_WINDOW_TOKEN_THIS_WEEK\"}}'"},
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
			Type: "UploadAttachmentResponse",
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

		RESTException: &module.RESTException{
			Reason: module.RESTReasonMultipartUpload,
			Note:   "Multipart/form-data request transport cannot be expressed in proto. The response payload is still the proto-typed UploadAttachmentResponse message.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "multipart/form-data", Conformance: "transport_only"},
				Response: module.RESTPayload{ProtoFullName: "vrooli.content_desk.v1.notes.UploadAttachmentResponse", Transport: "json", Conformance: "protojson"},
				Error:    module.RESTPayload{ProtoFullName: "vrooli.content_desk.v1.shared.ErrorEnvelope", Transport: "json", Conformance: "protojson"},
			},
		},
	},
}
