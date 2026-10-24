// Package manifest is the HTTP/Connect handler edge for the manifest
// domain. It owns proto ↔ domain translation (adapter.go) and
// Connect-RPC method implementations (connect_handler.go) but contains
// no business logic — that lives in internal/manifest/.
package manifest

import (
	"log"

	"github.com/vrooli/api-core/database"

	"development-toolchain-validator/internal/module"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	manifestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/manifest/manifest_v1connect"

	internalmanifest "development-toolchain-validator/internal/manifest"
)

// Module returns the manifest domain's contribution to the API.
func Module(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger) module.Module {
	repo := internalmanifest.NewSQLiteRepository(db, clk)
	svc := internalmanifest.NewService(repo, clk)
	connectPath, connectHandler := manifestconnect.NewManifestServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "manifest",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the internal package schema so the modules registry
// can collect schema from one symbol per handler package.
func Schema() string { return internalmanifest.Schema() }

// Endpoints is the machine-readable description of the manifest module's
// public surface. Connect-RPC method paths reference generated *Procedure
// constants so renaming an RPC in manifest.proto breaks this file at
// compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "manifest_list",
		Path:        manifestconnect.ManifestServiceListManifestsProcedure,
		Method:      "POST",
		Summary:     "List manifests",
		Description: "Returns every stored (skill_id, golden_slug) manifest ordered by skill_id then golden_slug.",
		Category:    "manifest",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"manifests": "array<Manifest>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List manifests", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.manifest.ManifestService/ListManifests -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "manifest_get",
		Path:        manifestconnect.ManifestServiceGetManifestProcedure,
		Method:      "POST",
		Summary:     "Get a manifest by (skill_id, golden_slug)",
		Description: "Returns the manifest pinned for the given tuple.",
		Category:    "manifest",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"skill_id":    "string (required)",
				"golden_slug": "string (required)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"manifest": "Manifest"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing skill_id or golden_slug"},
			{Status: 404, Code: "not_found", Description: "No manifest exists for that tuple"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Get manifest", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.manifest.ManifestService/GetManifest -H 'Content-Type: application/json' -d '{\"skill_id\":\"implementation-plan-authoring\",\"golden_slug\":\"reference-react-vite\"}'"},
		},
	},
	{
		ID:          "manifest_upsert",
		Path:        manifestconnect.ManifestServiceUpsertManifestProcedure,
		Method:      "POST",
		Summary:     "Create or replace a manifest",
		Description: "Stores the manifest for the given (skill_id, golden_slug). Replaces any existing row.",
		Category:    "manifest",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"manifest": "Manifest (required)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"manifest": "Manifest (post-write)"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Validation failure (slug shape, empty paths, etc.)"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Upsert manifest", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.manifest.ManifestService/UpsertManifest -H 'Content-Type: application/json' -d '{\"manifest\":{\"skill_id\":\"implementation-plan-authoring\",\"golden_slug\":\"reference-react-vite\",\"wildcard_allowed\":true}}'"},
		},
	},
	{
		ID:          "manifest_clear_stale",
		Path:        manifestconnect.ManifestServiceClearStaleProcedure,
		Method:      "POST",
		Summary:     "Clear staleness override for a manifest",
		Description: "Records a manual-clear timestamp that suppresses staleness for the tuple until the next manifest upsert.",
		Category:    "manifest",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"skill_id":    "string (required)",
				"golden_slug": "string (required)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"cleared_at": "google.protobuf.Timestamp"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing skill_id or golden_slug"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Clear stale", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.manifest.ManifestService/ClearStale -H 'Content-Type: application/json' -d '{\"skill_id\":\"implementation-plan-authoring\",\"golden_slug\":\"reference-react-vite\"}'"},
		},
	},
}
