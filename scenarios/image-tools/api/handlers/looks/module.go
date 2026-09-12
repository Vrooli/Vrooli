package looks

import (
	"log"

	internallooks "image-tools/internal/looks"
	"image-tools/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	looksconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/looks/looks_v1connect"
)

// Module returns the looks domain's contribution: the generated Connect-RPC
// LooksService handler over the Look library (built-in seed + SQLite-persisted
// custom Looks). The blob store backs RenderPreview's rendered thumbnails.
func Module(db *database.RoutedDB, blobs BlobStore, logger *log.Logger) module.Module {
	store := internallooks.NewStore(db)
	connectPath, connectHandler := looksconnect.NewLooksServiceHandler(NewConnectHandler(Deps{
		Store:  store,
		Blobs:  blobs,
		Logger: logger,
	}))
	return module.Module{
		Name: "looks",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internallooks.Schema so the modules registry collects the
// looks domain's table from one symbol per handler package.
func Schema() string { return internallooks.Schema() }

// Endpoints describes the looks module's public surface. Connect-RPC method
// paths reference the generated *Procedure constants, so renaming an RPC in
// looks.proto breaks this file at compile time; the global parity test asserts
// every rpc has exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "looks_list",
		Path:        looksconnect.LooksServiceListLooksProcedure,
		Method:      "POST",
		Summary:     "List Looks",
		Description: "Returns the Look/Style library (built-in + custom), optionally filtered by kind, built-ins first.",
		Category:    "looks",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"kind": "LookKind (optional filter)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"looks": "array<Look>"}},
		Examples: []module.Example{
			{Name: "List all Looks", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.looks.LooksService/ListLooks -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "looks_get",
		Path:        looksconnect.LooksServiceGetLookProcedure,
		Method:      "POST",
		Summary:     "Get a Look by id",
		Description: "Returns one Look (built-in or custom) by id.",
		Category:    "looks",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"look": "Look"}},
		Errors:      []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No Look with that id"}},
		Examples: []module.Example{
			{Name: "Get the Noir look", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.looks.LooksService/GetLook -H 'Content-Type: application/json' -d '{\"id\":\"noir\"}'"},
		},
	},
	{
		ID:          "looks_create",
		Path:        looksconnect.LooksServiceCreateLookProcedure,
		Method:      "POST",
		Summary:     "Create a custom Look",
		Description: "Persists a new custom Look. The id is slugified from the name when empty and must not collide with a built-in or existing custom Look.",
		Category:    "looks",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"look": "Look (name + steps required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"look": "Look"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing name/steps or an unknown step operation"},
			{Status: 409, Code: "already_exists", Description: "Id collides with a built-in or existing custom Look"},
		},
		Examples: []module.Example{
			{Name: "Create a sepia Look", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.looks.LooksService/CreateLook -H 'Content-Type: application/json' -d '{\"look\":{\"name\":\"Sepia\",\"kind\":\"LOOK_KIND_FILM\",\"steps\":[{\"operation\":\"filter\",\"kind\":\"STEP_KIND_DETERMINISTIC\",\"params\":{\"filter\":\"sepia\"}}]}}'"},
		},
	},
	{
		ID:          "looks_update",
		Path:        looksconnect.LooksServiceUpdateLookProcedure,
		Method:      "POST",
		Summary:     "Update a custom Look",
		Description: "Replaces a custom Look's mutable fields. Built-in Looks are read-only.",
		Category:    "looks",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"look": "Look (id required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"look": "Look"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "failed_precondition", Description: "The Look is a read-only built-in"},
			{Status: 404, Code: "not_found", Description: "No custom Look with that id"},
		},
		Examples: []module.Example{
			{Name: "Rename a custom Look", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.looks.LooksService/UpdateLook -H 'Content-Type: application/json' -d '{\"look\":{\"id\":\"sepia\",\"name\":\"Warm Sepia\",\"steps\":[{\"operation\":\"filter\",\"kind\":\"STEP_KIND_DETERMINISTIC\",\"params\":{\"filter\":\"sepia\"}}]}}'"},
		},
	},
	{
		ID:          "looks_delete",
		Path:        looksconnect.LooksServiceDeleteLookProcedure,
		Method:      "POST",
		Summary:     "Delete a custom Look",
		Description: "Removes a custom Look. Built-in Looks cannot be deleted.",
		Category:    "looks",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"deleted": "bool"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "failed_precondition", Description: "The Look is a read-only built-in"},
			{Status: 404, Code: "not_found", Description: "No custom Look with that id"},
		},
		Examples: []module.Example{
			{Name: "Delete a custom Look", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.looks.LooksService/DeleteLook -H 'Content-Type: application/json' -d '{\"id\":\"sepia\"}'"},
		},
	},
	{
		ID:          "looks_compile",
		Path:        looksconnect.LooksServiceCompileLookProcedure,
		Method:      "POST",
		Summary:     "Compile a Look into request shapes",
		Description: "Resolves a Look + subject/prompt into the ordered concrete op/AI request shapes (prompt template filled, params merged). The compose-seam; executes nothing.",
		Category:    "looks",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"look_id": "string", "subject": "string", "prompt": "string", "has_input": "bool",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"steps": "array<CompiledStep>", "primary_prompt": "string", "requires_image": "bool", "requires_mask": "bool", "warnings": "array<string>",
		}},
		Errors: []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No Look with that id"}},
		Examples: []module.Example{
			{Name: "Compile the Anime look", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.looks.LooksService/CompileLook -H 'Content-Type: application/json' -d '{\"look_id\":\"anime\",\"subject\":\"a golden retriever\",\"has_input\":true}'"},
		},
	},
	{
		ID:          "looks_render_preview",
		Path:        looksconnect.LooksServiceRenderPreviewProcedure,
		Method:      "POST",
		Summary:     "Render a Look's preview thumbnail",
		Description: "Applies the Look's deterministic step chain to a built-in reference image, stores the preview as a blob, and (for custom Looks) records its thumbnail_ref. AI steps are deferred and reported.",
		Category:    "looks",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"look_id": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"thumbnail_ref": "string (blob key)", "deferred_steps": "array<string>",
		}},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No Look with that id"},
			{Status: 503, Code: "unavailable", Description: "Preview storage is not configured"},
		},
		Examples: []module.Example{
			{Name: "Preview the Polaroid look", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.looks.LooksService/RenderPreview -H 'Content-Type: application/json' -d '{\"look_id\":\"polaroid-600\"}'"},
		},
	},
}
