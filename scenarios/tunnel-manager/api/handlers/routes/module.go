package routes

import (
	"log"

	"tunnel-manager/internal/clock"
	"tunnel-manager/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	routesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/routes/routes_v1connect"

	internalroutes "tunnel-manager/internal/routes"
)

// Module returns the routes domain's contribution to the API: the
// generated Connect-RPC RoutesService handler. The center (server.New)
// does not change — adding a domain is one Module() call in main.go plus
// one registration row in modules.AllEndpoints/AllSchemas/AllProtoFiles.
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	repo := internalroutes.NewSQLiteRepository(db, clk)
	svc := internalroutes.NewService(repo)
	connectPath, connectHandler := routesconnect.NewRoutesServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "routes",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internalroutes.Schema so the modules registry collects
// endpoint descriptors and schema from one symbol per handler package.
func Schema() string { return internalroutes.Schema() }

// Endpoints is the machine-readable description of the routes module's
// public surface. Connect-RPC method paths reference the generated
// *Procedure constants from routesconnect, so adding or renaming an RPC in
// routes.proto breaks this file at compile time. TestProtoConnectParity
// (api/internal/modules/registry_test.go) asserts every rpc has exactly
// one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "routes_list",
		Path:        routesconnect.RoutesServiceListRoutesProcedure,
		Method:      "POST",
		Summary:     "List routes",
		Description: "Returns all manifest routes ordered by subdomain, optionally filtered by tier, through the generated Connect-RPC Routes service.",
		Category:    "routes",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"tier": "Tier (TIER_CORE | TIER_LEASED; unset returns all)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"routes": "array<Route>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List routes", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.routes.RoutesService/ListRoutes -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "routes_get",
		Path:        routesconnect.RoutesServiceGetRouteProcedure,
		Method:      "POST",
		Summary:     "Get a route by id",
		Description: "Returns the manifest route matching the request id through the generated Connect-RPC Routes service.",
		Category:    "routes",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"route": "Route"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No route with that id exists"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Get route", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.routes.RoutesService/GetRoute -H 'Content-Type: application/json' -d '{\"id\":\"abc123\"}'"},
		},
	},
	{
		ID:          "routes_create",
		Path:        routesconnect.RoutesServiceCreateRouteProcedure,
		Method:      "POST",
		Summary:     "Create a route",
		Description: "Persists a new manifest route. Subdomain (valid DNS label), scenario, and local_port are required; domain defaults to itsagitime.com, health_path to /health, tier to leased, enabled to true.",
		Category:    "routes",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"subdomain":   "string (required, DNS label)",
				"scenario":    "string (required)",
				"local_port":  "int32 (required, 1-65535)",
				"domain":      "string (default itsagitime.com)",
				"tier":        "Tier (default TIER_LEASED)",
				"health_path": "string (default /health)",
				"enabled":     "bool (default true)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"route": "Route"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Invalid subdomain, missing scenario, or out-of-range port"},
			{Status: 409, Code: "already_exists", Description: "A route already exists for the subdomain"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Create route", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.routes.RoutesService/CreateRoute -H 'Content-Type: application/json' -d '{\"subdomain\":\"agent-manager\",\"scenario\":\"agent-manager\",\"local_port\":21100}'"},
		},
	},
	{
		ID:          "routes_update",
		Path:        routesconnect.RoutesServiceUpdateRouteProcedure,
		Method:      "POST",
		Summary:     "Update a route",
		Description: "Applies a partial update to the route with the given id; only set fields change.",
		Category:    "routes",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"id":          "string (required)",
				"subdomain":   "string",
				"scenario":    "string",
				"domain":      "string",
				"local_port":  "int32",
				"tier":        "Tier",
				"health_path": "string",
				"enabled":     "bool",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"route": "Route"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Invalid field value"},
			{Status: 404, Code: "not_found", Description: "No route with that id exists"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Disable route", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.routes.RoutesService/UpdateRoute -H 'Content-Type: application/json' -d '{\"id\":\"abc123\",\"enabled\":false}'"},
		},
	},
	{
		ID:          "routes_delete",
		Path:        routesconnect.RoutesServiceDeleteRouteProcedure,
		Method:      "POST",
		Summary:     "Delete a route",
		Description: "Removes a manifest route by id. Returns deleted=false when the id did not exist.",
		Category:    "routes",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"deleted": "bool"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Delete route", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.routes.RoutesService/DeleteRoute -H 'Content-Type: application/json' -d '{\"id\":\"abc123\"}'"},
		},
	},
}
