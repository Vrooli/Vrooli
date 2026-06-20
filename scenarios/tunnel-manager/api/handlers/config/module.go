package config

import (
	"log"

	"tunnel-manager/internal/authz"
	"tunnel-manager/internal/clock"
	"tunnel-manager/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	configconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config/config_v1connect"

	internalconfig "tunnel-manager/internal/config"
)

// Module returns the config domain's contribution to the API: the
// generated Connect-RPC ConfigService handler. The center (server.New)
// does not change — adding a domain is one Module() call in main.go plus
// one registration row in modules.AllEndpoints/AllSchemas/AllProtoFiles.
//
// The config service reconciles ingress against the routes manifest, so it
// reads routes through internalroutes.Service (the RoutesReader seam) and
// actuates Cloudflare through the IngressClient built from env credentials
// (canonical CLOUDFLARE_* names, with deterministic legacy CF_* fallback).
// When credentials are absent the IngressClient is nil and remote apply
// operations return ErrRemoteUnavailable (mapped to FailedPrecondition).
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	svc := internalconfig.NewProductionService(db, clk, internalconfig.ProductionOptions{})

	connectPath, connectHandler := configconnect.NewConfigServiceHandler(NewConnectHandler(Deps{
		Service:    svc,
		Logger:     logger,
		Authorizer: authz.FromEnv(),
	}))
	return module.Module{
		Name: "config",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internalconfig.Schema so the modules registry collects
// endpoint descriptors and schema from one symbol per handler package.
func Schema() string { return internalconfig.Schema() }

// Endpoints is the machine-readable description of the config module's
// public surface. Connect-RPC method paths reference the generated
// *Procedure constants from configconnect, so adding or renaming an RPC in
// config.proto breaks this file at compile time. TestProtoConnectParity
// (api/internal/modules/registry_test.go) asserts every rpc has exactly
// one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "config_get",
		Path:        configconnect.ConfigServiceGetConfigProcedure,
		Method:      "POST",
		Summary:     "Get tunnel configuration",
		Description: "Returns the persisted tunnel configuration (mode and Cloudflare coordinates) through the generated Connect-RPC Config service.",
		Category:    "config",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"config": "TunnelConfig", "readiness": "ConfigReadiness"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Get config", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.config.ConfigService/GetConfig -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "config_sync",
		Path:        configconnect.ConfigServiceSyncProcedure,
		Method:      "POST",
		Summary:     "Reconcile ingress with the routes manifest",
		Description: "Computes desired ingress from enabled routes (subdomain.domain hostnames → http://localhost:<port> + catch-all 404), diffs it against live ingress, and applies the difference (Cloudflare API in remote mode; config.yml + cloudflared restart in local mode). dry_run computes the diff without applying. Idempotent: a no-drift sync applies nothing.",
		Category:    "config",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"dry_run": "bool (compute diff without applying)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"mode":           "Mode",
				"added":          "array<string> (hostnames added)",
				"removed":        "array<string> (hostnames removed)",
				"no_changes":     "bool (manifest already matched live ingress)",
				"setup_required": "bool (dry-run found missing setup)",
				"missing_fields": "array<string> (canonical env names)",
				"message":        "string (operator explanation)",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 412, Code: "failed_precondition", Description: "Remote mode requested but Cloudflare credentials are absent"},
			{Status: 500, Code: "internal", Description: "Ingress read/apply failure"},
		},
		Examples: []module.Example{
			{Name: "Dry-run sync", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.config.ConfigService/Sync -H 'Content-Type: application/json' -d '{\"dry_run\":true}'"},
		},
	},
	{
		ID:          "config_switch_mode",
		Path:        configconnect.ConfigServiceSwitchModeProcedure,
		Method:      "POST",
		Summary:     "Switch tunnel management mode",
		Description: "Migrates between remote (Cloudflare API) and local (config.yml) management, applies the current manifest's ingress through the target channel, and persists the new mode.",
		Category:    "config",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"target_mode": "Mode (MODE_REMOTE | MODE_LOCAL)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"previous_mode": "Mode",
				"current_mode":  "Mode",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Unknown target mode"},
			{Status: 412, Code: "failed_precondition", Description: "Remote mode requested but Cloudflare credentials are absent"},
			{Status: 500, Code: "internal", Description: "Ingress apply or persist failure"},
		},
		Examples: []module.Example{
			{Name: "Switch to local", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.config.ConfigService/SwitchMode -H 'Content-Type: application/json' -d '{\"target_mode\":\"MODE_LOCAL\"}'"},
		},
	},
}
