// Package brands is the HTTP/Connect boundary for the brands domain: it adapts
// the generated BrandsService handler onto the transport-agnostic
// internal/brands service and exports the domain's static metadata
// (Endpoints, Schema) for the modules registry.
package brands

import (
	"log"

	"brand-manager/internal/module"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	brandsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/brands/brands_v1connect"

	internalbrands "brand-manager/internal/brands"
)

// Module returns the brands domain's contribution to the API: the generated
// Connect-RPC BrandsService handler over the sqlite-backed service.
//
// Adding a domain means copying this file into handlers/<dom>/module.go and
// pointing it at <dom>'s proto-generated handler and service.
func Module(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger) module.Module {
	repo := internalbrands.NewSQLiteRepository(db, clk)
	versions := internalbrands.NewSQLiteVersionRepository(db, clk)
	svc := internalbrands.NewService(repo, versions, logger)
	connectPath, connectHandler := brandsconnect.NewBrandsServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "brands",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internalbrands.Schema so the modules registry collects both
// endpoint descriptors and schema from one symbol per handler package.
func Schema() string { return internalbrands.Schema() }

// Endpoints is the machine-readable description of the brands module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants, so renaming an RPC in brands.proto breaks this file at compile
// time. The global parity test in modules/registry_test.go asserts every rpc
// has exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "brands_list",
		Path:        brandsconnect.BrandsServiceListBrandsProcedure,
		Method:      "POST",
		Summary:     "List brands",
		Description: "Returns brands ordered newest-updated first, with optional name-substring filter and pagination.",
		Category:    "brands",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"name_contains": "string (case-insensitive substring filter)",
			"limit":         "int32 (default 100, max 500)",
			"offset":        "int32 (rows to skip)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"brands": "array<Brand>"}},
		Errors:   []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Repository read failure"}},
		Examples: []module.Example{{Name: "List brands", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.brands.BrandsService/ListBrands -H 'Content-Type: application/json' -d '{}'"}},
	},
	{
		ID:          "brands_create",
		Path:        brandsconnect.BrandsServiceCreateBrandProcedure,
		Method:      "POST",
		Summary:     "Create a brand",
		Description: "Persists a new brand at version 1 and snapshots that version. Name is required.",
		Category:    "brands",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"name":       "string (required, non-whitespace)",
			"identity":   "Identity",
			"colors":     "Colors",
			"typography": "Typography",
			"voice":      "Voice",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"brand": "Brand"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing or whitespace-only name"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{{Name: "Create brand", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.brands.BrandsService/CreateBrand -H 'Content-Type: application/json' -d '{\"name\":\"Acme\"}'"}},
	},
	{
		ID:          "brands_get",
		Path:        brandsconnect.BrandsServiceGetBrandProcedure,
		Method:      "POST",
		Summary:     "Get a brand by id",
		Description: "Returns the brand matching the request id.",
		Category:    "brands",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"brand": "Brand"}},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No brand with that id exists"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{{Name: "Get brand", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.brands.BrandsService/GetBrand -H 'Content-Type: application/json' -d '{\"id\":\"abc123\"}'"}},
	},
	{
		ID:          "brands_update",
		Path:        brandsconnect.BrandsServiceUpdateBrandProcedure,
		Method:      "POST",
		Summary:     "Update a brand (partial)",
		Description: "Merges non-empty fields onto the stored brand, increments the version, and snapshots it. expected_version (when > 0) enforces optimistic locking.",
		Category:    "brands",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"id":               "string (required)",
			"expected_version": "int32 (optional optimistic lock)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"brand": "Brand"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Resulting name is empty"},
			{Status: 404, Code: "not_found", Description: "No brand with that id exists"},
			{Status: 409, Code: "failed_precondition", Description: "expected_version does not match current version"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{{Name: "Update brand", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.brands.BrandsService/UpdateBrand -H 'Content-Type: application/json' -d '{\"id\":\"abc123\",\"description\":\"updated\"}'"}},
	},
	{
		ID:          "brands_delete",
		Path:        brandsconnect.BrandsServiceDeleteBrandProcedure,
		Method:      "POST",
		Summary:     "Delete a brand (idempotent)",
		Description: "Removes the brand. Deleting a missing brand returns success.",
		Category:    "brands",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{}},
		Errors:      []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Repository write failure"}},
		Examples:    []module.Example{{Name: "Delete brand", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.brands.BrandsService/DeleteBrand -H 'Content-Type: application/json' -d '{\"id\":\"abc123\"}'"}},
	},
	{
		ID:          "brands_versions",
		Path:        brandsconnect.BrandsServiceListBrandVersionsProcedure,
		Method:      "POST",
		Summary:     "List a brand's version history",
		Description: "Returns the brand's immutable version snapshots, newest-first.",
		Category:    "brands",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"brand_id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"versions": "array<BrandVersion>"}},
		Errors:      []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Repository read failure"}},
		Examples:    []module.Example{{Name: "List versions", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.brands.BrandsService/ListBrandVersions -H 'Content-Type: application/json' -d '{\"brand_id\":\"abc123\"}'"}},
	},
	{
		ID:          "brands_tokens",
		Path:        brandsconnect.BrandsServiceGetTokensProcedure,
		Method:      "POST",
		Summary:     "Get brand design tokens",
		Description: "Projects a brand's color system into stable $brand.* tokens for downstream image and content composition.",
		Category:    "brands",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"brand_id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"tokens": "array<Token>"}},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No brand with that id exists"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{{Name: "Get brand tokens", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.brands.BrandsService/GetTokens -H 'Content-Type: application/json' -d '{\"brand_id\":\"abc123\"}'"}},
	},
}
