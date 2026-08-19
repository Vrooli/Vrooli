package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"tunnel-manager/internal/authz"
	"tunnel-manager/internal/module"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	configconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config/config_v1connect"

	internalconfig "tunnel-manager/internal/config"
)

// resolveScenariosRoot finds the scenarios directory used to resolve a
// scenario's fixed UI port during adopt. VROOLI_SCENARIOS_ROOT wins; otherwise
// walk up from the working directory for a "scenarios" dir; failing that, fall
// back to "scenarios" relative to cwd. Mirrors the exposure/audit resolvers
// (each domain keeps its own to avoid a cross-handler import).
func resolveScenariosRoot() string {
	if v := strings.TrimSpace(os.Getenv("VROOLI_SCENARIOS_ROOT")); v != "" {
		return v
	}
	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for {
			candidate := filepath.Join(dir, "scenarios")
			if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
				return candidate
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return "scenarios"
}

func NewProductionService(db *database.RoutedDB, clk schedule.Clock, routes internalconfig.Routes) internalconfig.Service {
	return internalconfig.NewProductionService(db, clk, internalconfig.ProductionOptions{
		Routes:       routes,
		RoutesWriter: routes,
		// Lets AdoptIngress auto-classify an adopted hostname as a scenario
		// route (real port) when its subdomain matches a known scenario,
		// instead of always falling back to an external route with port 0.
		Scenarios: internalconfig.NewFileScenarioResolver(resolveScenariosRoot()),
	})
}

// Module returns the config domain's contribution to the API: the
// generated Connect-RPC ConfigService handler. The center (server.New)
// does not change — adding a domain is one Module() call in main.go plus
// one registration row in modules.AllEndpoints/AllSchemas/AllProtoFiles.
//
// The config service reconciles ingress against the routes manifest, so it
// reads routes through internalroutes.Service (the RoutesReader seam) and
// actuates Cloudflare through the IngressClient built from the config-domain
// credential store. Environment variables are not credential sources;
// legacy CF_* aliases are intentionally not accepted. When credentials are
// absent the IngressClient is nil and remote apply operations return
// ErrRemoteUnavailable (mapped to FailedPrecondition).
func Module(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger) module.Module {
	svc := NewProductionService(db, clk, nil)
	return ModuleWithService(svc, logger)
}

func ModuleWithService(svc internalconfig.Service, logger *log.Logger) module.Module {
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
		ID:          "config_credentials_status",
		Path:        configconnect.ConfigServiceGetCredentialStatusProcedure,
		Method:      "POST",
		Summary:     "Get Cloudflare credential status",
		Description: "Returns browser-safe Cloudflare credential presence/source metadata. Secret values are never returned.",
		Category:    "config",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"status": "CredentialStatus"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Credential store read failure"},
		},
		Examples: []module.Example{
			{Name: "Credential status", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.config.ConfigService/GetCredentialStatus -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "config_verify_credentials",
		Path:        configconnect.ConfigServiceVerifyCredentialsProcedure,
		Method:      "POST",
		Summary:     "Verify Cloudflare credentials live",
		Description: "Performs read-only Cloudflare probes (token verify, account/tunnel read, apex zone lookup + DNS-records read) and returns a per-check verdict with remediation. Never returns secret values.",
		Category:    "config",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"checks": "CredentialCheck[]", "ready": "bool"},
		},
		Errors: []module.ErrorDesc{
			{Status: 412, Code: "failed_precondition", Description: "Live credential verification is not configured"},
			{Status: 500, Code: "internal", Description: "Credential resolve failure"},
		},
		Examples: []module.Example{
			{Name: "Verify credentials", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.config.ConfigService/VerifyCredentials -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "config_bootstrap",
		Path:        configconnect.ConfigServiceBootstrapCloudflareProcedure,
		Method:      "POST",
		Summary:     "Bootstrap Cloudflare tunnel",
		Description: "Adopts or creates the named Cloudflare tunnel from one operator API token and writes the complete derived credential set only after the connector token is fetched.",
		Category:    "config",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"api_token": "string (write-only)", "account_id": "string", "tunnel_id": "string", "tunnel_name": "string", "dry_run": "bool",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"account_id": "string", "tunnel_id": "string", "adopted": "bool", "created": "bool", "written": "bool",
		}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Operator token required when authz is enforced"},
			{Status: 403, Code: "permission_denied", Description: "Cloudflare API token lacks the required account permission"},
			{Status: 500, Code: "internal", Description: "Bootstrap or credential-authority failure"},
		},
		Examples: []module.Example{{Name: "Dry-run bootstrap", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.config.ConfigService/BootstrapCloudflare -H 'Content-Type: application/json' -d @- <<'JSON'\n{\"api_token\":\"<token>\",\"dry_run\":true}\nJSON"}},
	},
	{
		ID:          "config_credentials_set",
		Path:        configconnect.ConfigServiceSetCloudflareCredentialsProcedure,
		Method:      "POST",
		Summary:     "Set Cloudflare credentials",
		Description: "Stores write-only Cloudflare credential values in the canonical operator credential authority and returns redacted status metadata.",
		Category:    "config",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"account_id": "string",
				"tunnel_id":  "string",
				"api_token":  "string (write-only)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"status": "CredentialStatus"},
		},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Operator token required when authz is enforced"},
			{Status: 403, Code: "permission_denied", Description: "Operator token denied"},
			{Status: 500, Code: "internal", Description: "Credential store write failure"},
		},
		Examples: []module.Example{
			{Name: "Set credentials", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.config.ConfigService/SetCloudflareCredentials -H 'Content-Type: application/json' -d '{\"account_id\":\"acct\",\"tunnel_id\":\"tun\",\"api_token\":\"<token>\"}'"},
		},
	},
	{
		ID:          "config_credentials_clear",
		Path:        configconnect.ConfigServiceClearCloudflareCredentialsProcedure,
		Method:      "POST",
		Summary:     "Clear Cloudflare credentials",
		Description: "Removes Cloudflare credential values from the canonical operator credential authority.",
		Category:    "config",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"fields": "array<string> (account_id | tunnel_id | api_token | all)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"status": "CredentialStatus"},
		},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Operator token required when authz is enforced"},
			{Status: 403, Code: "permission_denied", Description: "Operator token denied"},
			{Status: 500, Code: "internal", Description: "Credential store delete failure"},
		},
		Examples: []module.Example{
			{Name: "Clear token", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.config.ConfigService/ClearCloudflareCredentials -H 'Content-Type: application/json' -d '{\"fields\":[\"api_token\"]}'"},
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
	{
		ID:          "config_get_drift",
		Path:        configconnect.ConfigServiceGetDriftProcedure,
		Method:      "POST",
		Summary:     "Get ingress drift",
		Description: "Reconciles the desired manifest, the live tunnel, and the ownership ledger into a classified read model (managed/missing/external_ok/orphaned/ignored/unmanaged). Read-only — never applies.",
		Category:    "config",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"mode":    "Mode",
				"entries": "array<IngressEntry> (hostname, service_target, state, source, scenario, lease_id, note)",
				"counts":  "DriftCounts (per-state tally)",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 412, Code: "failed_precondition", Description: "Remote mode requested but Cloudflare credentials are absent"},
			{Status: 500, Code: "internal", Description: "Ingress read failure"},
		},
		Examples: []module.Example{
			{Name: "Get drift", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.config.ConfigService/GetDrift -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "config_adopt_ingress",
		Path:        configconnect.ConfigServiceAdoptIngressProcedure,
		Method:      "POST",
		Summary:     "Adopt an unmanaged ingress hostname",
		Description: "Brings an unmanaged live hostname under management: as a scenario route when it resolves to a known scenario, otherwise as an external route. Records MANAGED/EXTERNAL ownership in the ledger.",
		Category:    "config",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"hostname": "string (required)",
				"scenario": "string (adopt as this scenario route)",
				"target":   "string (adopt as external route pointing here)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"entry": "IngressEntry (reclassified)"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing hostname or unadoptable target"},
			{Status: 401, Code: "unauthenticated", Description: "Operator token required when authz is enforced"},
			{Status: 409, Code: "already_exists", Description: "A route already exists for the subdomain"},
			{Status: 500, Code: "internal", Description: "Route create or ledger write failure"},
		},
		Examples: []module.Example{
			{Name: "Adopt as external", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.config.ConfigService/AdoptIngress -H 'Content-Type: application/json' -d '{\"hostname\":\"api.itsagitime.com\",\"target\":\"http://127.0.0.1:9000\"}'"},
		},
	},
	{
		ID:          "config_ignore_ingress",
		Path:        configconnect.ConfigServiceIgnoreIngressProcedure,
		Method:      "POST",
		Summary:     "Ignore an external ingress hostname",
		Description: "Acknowledges a live hostname as external and records it IGNORED, so reconcile never pushes or prunes it.",
		Category:    "config",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"hostname": "string (required)",
				"note":     "string (optional operator note)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"entry": "IngressEntry (reclassified)"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing hostname"},
			{Status: 401, Code: "unauthenticated", Description: "Operator token required when authz is enforced"},
			{Status: 500, Code: "internal", Description: "Ledger write failure"},
		},
		Examples: []module.Example{
			{Name: "Ignore hostname", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.config.ConfigService/IgnoreIngress -H 'Content-Type: application/json' -d '{\"hostname\":\"legacy.itsagitime.com\",\"note\":\"operator dashboard\"}'"},
		},
	},
	{
		ID:          "config_prune_ingress",
		Path:        configconnect.ConfigServicePruneIngressProcedure,
		Method:      "POST",
		Summary:     "Prune a single ingress hostname",
		Description: "Removes a single named hostname from live ingress and the ownership ledger. The only path that removes a specific entry.",
		Category:    "config",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"hostname": "string (required)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"pruned": "bool (true when removed)"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing hostname"},
			{Status: 401, Code: "unauthenticated", Description: "Operator token required when authz is enforced"},
			{Status: 412, Code: "failed_precondition", Description: "Remote mode requested but Cloudflare credentials are absent"},
			{Status: 500, Code: "internal", Description: "Ingress apply or ledger delete failure"},
		},
		Examples: []module.Example{
			{Name: "Prune hostname", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.config.ConfigService/PruneIngress -H 'Content-Type: application/json' -d '{\"hostname\":\"legacy.itsagitime.com\"}'"},
		},
	},
	{
		ID:          "config_set_public_exposure",
		Path:        configconnect.ConfigServiceSetPublicExposureProcedure,
		Method:      "POST",
		Summary:     "Toggle the global /public Access-bypass switch",
		Description: "Flips the global public-exposure capability (the /public-asset convention). PURE: persists the flag and performs zero Cloudflare writes — the next Sync reconciles the live Access apps.",
		Category:    "config",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"enabled": "bool (new global state)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"config": "TunnelConfig (persisted)"},
		},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Operator token required when authz is enforced"},
			{Status: 500, Code: "internal", Description: "Config persist failure"},
		},
		Examples: []module.Example{
			{Name: "Enable public exposure", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.config.ConfigService/SetPublicExposure -H 'Content-Type: application/json' -d '{\"enabled\":true}'"},
		},
	},
	{
		ID:          "config_get_access_status",
		Path:        configconnect.ConfigServiceGetAccessStatusProcedure,
		Method:      "POST",
		Summary:     "Get /public Access-bypass status + dry-run plan",
		Description: "Returns the public-exposure read model: the global switch, whether the Access client is configured, per-host effective decisions, and the dry-run plan (apps a reconcile would create/remove). Read-only — no Cloudflare calls.",
		Category:    "config",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"status": "AccessStatus (enabled, configured, hosts[], to_create[], to_remove[])",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Config or ledger read failure"},
		},
		Examples: []module.Example{
			{Name: "Get access status", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.config.ConfigService/GetAccessStatus -H 'Content-Type: application/json' -d '{}'"},
		},
	},
}
