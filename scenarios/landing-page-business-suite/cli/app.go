package main

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	appName        = "landing-page-business-suite"
	appVersion     = "1.0.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

type App struct {
	core *cliapp.ScenarioApp
}

type endpointDef struct {
	Name         string
	Method       string
	Path         string
	Description  string
	Root         bool
	AllowRawPath bool
	Stream       bool
}

func NewApp() (*App, error) {
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{
		ExtraAPIEnvVars:     []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		ExtraAPIPortEnvVars: []string{"API_PORT"},
	})
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:              appName,
		Version:           appVersion,
		Description:       "Landing Page Business Suite CLI",
		DefaultAPIBase:    defaultAPIBase,
		APIEnvVars:        env.APIEnvVars,
		APIPortEnvVars:    env.APIPortEnvVars,
		APIPortDetector:   cliutil.DetectPortFromVrooli(appName, "API_PORT"),
		ConfigDirEnvVars:  env.ConfigDirEnvVars,
		SourceRootEnvVars: env.SourceRootEnvVars,
		TokenEnvVars:      env.TokenEnvVars,
		BuildFingerprint:  buildFingerprint,
		BuildTimestamp:    buildTimestamp,
		BuildSourceRoot:   buildSourceRoot,
		AllowAnonymous:    true,
	})
	if err != nil {
		return nil, err
	}
	app := &App{core: core}
	app.core.SetCommands(app.registerCommands())
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	health := cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Check API health (/health)", Run: a.cmdStatus},
			a.endpointCommand(endpointDef{Name: "health", Method: "GET", Path: "/health", Description: "Check API health (/api/v1/health)"}),
			{Name: "service-auth-status", NeedsAPI: true, Description: "Check LPBS service-to-service auth readiness", Run: a.cmdServiceAuthStatus},
			{Name: "deploy-readiness", NeedsAPI: true, Description: "Run LPBS readiness checks for desktop deploy handoff", Run: a.cmdDeployReadiness},
		},
	}

	landing := cliapp.CommandGroup{
		Title: "Landing",
		Commands: a.endpointCommands([]endpointDef{
			{Name: "landing-config", Method: "GET", Path: "/landing-config", Description: "Fetch landing configuration"},
			{Name: "plans", Method: "GET", Path: "/plans", Description: "List pricing plans"},
			{Name: "variant-space", Method: "GET", Path: "/variant-space", Description: "Fetch variant space"},
			{Name: "customize", Method: "POST", Path: "/customize", Description: "Customize landing content"},
		}),
	}

	auth := cliapp.CommandGroup{
		Title: "Auth",
		Commands: a.endpointCommands([]endpointDef{
			{Name: "auth-magic-link", Method: "POST", Path: "/auth/magic-link", Description: "Request a magic link"},
			{Name: "auth-verify", Method: "GET", Path: "/auth/verify", Description: "Verify magic link"},
			{Name: "auth-refresh", Method: "POST", Path: "/auth/refresh", Description: "Refresh auth token"},
			{Name: "auth-logout", Method: "POST", Path: "/auth/logout", Description: "Logout current user"},
			{Name: "auth-me", Method: "GET", Path: "/auth/me", Description: "Get authenticated user"},
		}),
	}

	account := cliapp.CommandGroup{
		Title: "Account",
		Commands: a.endpointCommands([]endpointDef{
			{Name: "me-subscription", Method: "GET", Path: "/me/subscription", Description: "Get current subscription"},
			{Name: "me-credits", Method: "GET", Path: "/me/credits", Description: "Get credit balance"},
			{Name: "entitlements", Method: "GET", Path: "/entitlements", Description: "Get current entitlements"},
			{Name: "downloads", Method: "GET", Path: "/downloads", Description: "List available downloads"},
		}),
	}

	billing := cliapp.CommandGroup{
		Title: "Billing & Payments",
		Commands: a.endpointCommands([]endpointDef{
			{Name: "billing-checkout", Method: "POST", Path: "/billing/create-checkout-session", Description: "Create billing checkout session"},
			{Name: "billing-credits-checkout", Method: "POST", Path: "/billing/create-credits-checkout-session", Description: "Create credits checkout session"},
			{Name: "billing-portal", Method: "GET", Path: "/billing/portal-url", Description: "Get billing portal URL"},
			{Name: "checkout-create", Method: "POST", Path: "/checkout/create", Description: "Create checkout"},
			{Name: "webhook-stripe", Method: "POST", Path: "/webhooks/stripe", Description: "Send Stripe webhook payload"},
			{Name: "subscription-verify", Method: "GET", Path: "/subscription/verify", Description: "Verify subscription"},
			{Name: "subscription-cancel", Method: "POST", Path: "/subscription/cancel", Description: "Cancel subscription (admin)"},
		}),
	}

	variants := cliapp.CommandGroup{
		Title: "Variants",
		Commands: a.endpointCommands([]endpointDef{
			{Name: "variants-select", Method: "GET", Path: "/variants/select", Description: "Select a variant for visitor"},
			{Name: "public-variant", Method: "GET", Path: "/public/variants/{slug}", Description: "Get public variant by slug"},
			{Name: "public-variant-sections", Method: "GET", Path: "/public/variants/{variant_slug}/sections", Description: "Get public sections for a variant"},
			{Name: "variants-list", Method: "GET", Path: "/variants", Description: "List variants (admin)"},
			{Name: "variants-get", Method: "GET", Path: "/variants/{slug}", Description: "Get variant by slug (admin)"},
			{Name: "variants-update", Method: "PATCH", Path: "/variants/{slug}", Description: "Update variant by slug (admin)"},
			{Name: "variants-delete", Method: "DELETE", Path: "/variants/{slug}", Description: "Delete variant by slug (admin)"},
			{Name: "variants-sections", Method: "GET", Path: "/variants/{variant_slug}/sections", Description: "Get variant sections (admin)"},
			{Name: "admin-variants-sync", Method: "POST", Path: "/admin/variants/sync", Description: "Sync variants from snapshots"},
			{Name: "admin-variants-export", Method: "GET", Path: "/admin/variants/{slug}/export", Description: "Export variant snapshot"},
			{Name: "admin-variants-import", Method: "PUT", Path: "/admin/variants/{slug}/import", Description: "Import variant snapshot"},
		}),
	}

	content := cliapp.CommandGroup{
		Title: "Content",
		Commands: append(
			a.endpointCommands([]endpointDef{
				{Name: "branding", Method: "GET", Path: "/branding", Description: "Get public branding"},
				{Name: "admin-branding-get", Method: "GET", Path: "/admin/branding", Description: "Get branding (admin)"},
				{Name: "admin-branding-update", Method: "PUT", Path: "/admin/branding", Description: "Update branding (admin)"},
				{Name: "admin-branding-clear-field", Method: "POST", Path: "/admin/branding/clear-field", Description: "Clear branding field (admin)"},
				{Name: "admin-assets-list", Method: "GET", Path: "/admin/assets", Description: "List assets (admin)"},
				{Name: "admin-assets-get", Method: "GET", Path: "/admin/assets/{id}", Description: "Get asset (admin)"},
				{Name: "admin-assets-delete", Method: "DELETE", Path: "/admin/assets/{id}", Description: "Delete asset (admin)"},
				{Name: "uploads-get", Method: "GET", Path: "/uploads/{path}", Description: "Fetch uploaded asset by path", AllowRawPath: true},
				{Name: "seo", Method: "GET", Path: "/seo/{slug}", Description: "Get SEO metadata for variant"},
				{Name: "admin-variant-seo-update", Method: "PUT", Path: "/admin/variants/{slug}/seo", Description: "Update variant SEO"},
				{Name: "sitemap", Method: "GET", Path: "/sitemap.xml", Description: "Fetch sitemap", Root: true},
				{Name: "robots", Method: "GET", Path: "/robots.txt", Description: "Fetch robots.txt", Root: true},
			}),
			cliapp.Command{Name: "admin-assets-upload", NeedsAPI: true, Description: "Upload asset (admin)", Run: a.cmdAssetsUpload},
		),
	}

	metrics := cliapp.CommandGroup{
		Title: "Metrics",
		Commands: a.endpointCommands([]endpointDef{
			{Name: "metrics-track", Method: "POST", Path: "/metrics/track", Description: "Track metrics event"},
			{Name: "metrics-summary", Method: "GET", Path: "/metrics/summary", Description: "Get metrics summary (admin)"},
			{Name: "metrics-variants", Method: "GET", Path: "/metrics/variants", Description: "Get variant metrics (admin)"},
		}),
	}

	feedback := cliapp.CommandGroup{
		Title: "Engagement - Feedback",
		Commands: a.endpointCommands([]endpointDef{
			{Name: "feedback-create", Method: "POST", Path: "/api/feedback", Description: "Submit feedback", Root: true},
			{Name: "admin-feedback-list", Method: "GET", Path: "/admin/feedback", Description: "List feedback (admin)"},
			{Name: "admin-feedback-bulk-delete", Method: "POST", Path: "/admin/feedback/bulk-delete", Description: "Bulk delete feedback (admin)"},
			{Name: "admin-feedback-get", Method: "GET", Path: "/admin/feedback/{id}", Description: "Get feedback (admin)"},
			{Name: "admin-feedback-delete", Method: "DELETE", Path: "/admin/feedback/{id}", Description: "Delete feedback (admin)"},
			{Name: "admin-feedback-status-update", Method: "PATCH", Path: "/admin/feedback/{id}/status", Description: "Update feedback status (admin)"},
		}),
	}

	waitlist := cliapp.CommandGroup{
		Title: "Engagement - Waitlist",
		Commands: a.endpointCommands([]endpointDef{
			{Name: "waitlist-create", Method: "POST", Path: "/waitlist", Description: "Create waitlist entry"},
			{Name: "admin-waitlist-list", Method: "GET", Path: "/admin/waitlist", Description: "List waitlist entries (admin)"},
			{Name: "admin-waitlist-delete", Method: "DELETE", Path: "/admin/waitlist/{id}", Description: "Delete waitlist entry (admin)"},
			{Name: "admin-waitlist-export", Method: "GET", Path: "/admin/waitlist/export", Description: "Export waitlist entries (admin)"},
		}),
	}

	ai := cliapp.CommandGroup{
		Title: "AI Gateway",
		Commands: append(
			a.endpointCommands([]endpointDef{
				{Name: "ai-models", Method: "GET", Path: "/ai/models", Description: "List AI models"},
				{Name: "ai-health", Method: "GET", Path: "/ai/health", Description: "AI health"},
				{Name: "ai-chat", Method: "POST", Path: "/ai/chat", Description: "Run AI chat completion"},
				{Name: "ai-usage", Method: "GET", Path: "/ai/usage", Description: "Get AI usage"},
			}),
			cliapp.Command{Name: "ai-stream", NeedsAPI: true, Description: "Stream AI chat completion", Run: a.cmdAIStream},
		),
	}

	adminCommands := []cliapp.Command{
		{Name: "admin-login", NeedsAPI: true, Description: "Admin login (stores session)", Run: a.cmdAdminLogin},
		{Name: "admin-logout", NeedsAPI: true, Description: "Admin logout (clears session)", Run: a.cmdAdminLogout},
		{Name: "admin-session", NeedsAPI: true, Description: "Admin session status", Run: a.cmdAdminSession},
	}
	adminCommands = append(adminCommands, a.endpointCommands([]endpointDef{
		{Name: "admin-profile", Method: "GET", Path: "/admin/profile", Description: "Admin profile"},
		{Name: "admin-profile-update", Method: "PUT", Path: "/admin/profile", Description: "Update admin profile"},
		{Name: "admin-stripe-settings", Method: "GET", Path: "/admin/settings/stripe", Description: "Get Stripe settings"},
		{Name: "admin-stripe-settings-update", Method: "PUT", Path: "/admin/settings/stripe", Description: "Update Stripe settings"},
		{Name: "admin-stripe-secret", Method: "GET", Path: "/admin/settings/stripe/reveal", Description: "Reveal Stripe secret"},
		{Name: "admin-stripe-verify-price", Method: "GET", Path: "/admin/stripe/verify-price", Description: "Verify Stripe price"},
		{Name: "admin-reset-demo-data", Method: "POST", Path: "/admin/reset-demo-data", Description: "Reset demo data"},
	})...)
	admin := cliapp.CommandGroup{
		Title:    "Admin Core",
		Commands: adminCommands,
	}

	remoteProfiles := cliapp.CommandGroup{
		Title: "Admin Remote Profiles",
		Commands: []cliapp.Command{
			{Name: "remote-profiles-list", NeedsAPI: true, Description: "List remote profiles (admin)", Run: a.cmdRemoteProfilesList},
			{Name: "remote-profiles-create", NeedsAPI: true, Description: "Create remote profile (admin)", Run: a.cmdRemoteProfilesCreate},
			{Name: "remote-profiles-update", NeedsAPI: true, Description: "Update remote profile (admin)", Run: a.cmdRemoteProfilesUpdate},
			{Name: "remote-profiles-delete", NeedsAPI: true, Description: "Delete remote profile (admin)", Run: a.cmdRemoteProfilesDelete},
			{Name: "remote-profiles-login", NeedsAPI: true, Description: "Login remote profile (admin)", Run: a.cmdRemoteProfilesLogin},
			{Name: "remote-profiles-logout", NeedsAPI: true, Description: "Logout remote profile (admin)", Run: a.cmdRemoteProfilesLogout},
			{Name: "remote-profiles-test", NeedsAPI: true, Description: "Test remote profile session (admin)", Run: a.cmdRemoteProfilesTest},
			{Name: "remote-profiles-proxy", NeedsAPI: true, Description: "Proxy remote admin request via profile session (admin)", Run: a.cmdRemoteProfilesProxy},
		},
	}

	downloadsAdmin := cliapp.CommandGroup{
		Title: "Admin Commerce - Downloads",
		Commands: append(a.endpointCommands([]endpointDef{
			{Name: "admin-download-apps-list", Method: "GET", Path: "/admin/download-apps", Description: "List download apps"},
			{Name: "admin-download-apps-create", Method: "POST", Path: "/admin/download-apps", Description: "Create download app"},
			{Name: "admin-download-apps-save", Method: "PUT", Path: "/admin/download-apps/{app_key}", Description: "Update download app"},
			{Name: "admin-download-apps-delete", Method: "DELETE", Path: "/admin/download-apps/{app_key}", Description: "Delete download app"},
			{Name: "admin-download-storage-get", Method: "GET", Path: "/admin/download-storage", Description: "Get download storage"},
			{Name: "admin-download-storage-update", Method: "PUT", Path: "/admin/download-storage", Description: "Update download storage"},
			{Name: "admin-download-storage-test", Method: "POST", Path: "/admin/download-storage/test", Description: "Test download storage"},
			{Name: "admin-download-artifacts-list", Method: "GET", Path: "/admin/download-artifacts", Description: "List download artifacts"},
			{Name: "admin-download-artifacts-by-app", Method: "GET", Path: "/admin/download-artifacts/by-app", Description: "List download artifacts by app"},
			{Name: "admin-download-artifacts-presign-upload", Method: "POST", Path: "/admin/download-artifacts/presign-upload", Description: "Presign upload for artifact"},
			{Name: "admin-download-artifacts-commit", Method: "POST", Path: "/admin/download-artifacts/commit", Description: "Commit download artifact"},
			{Name: "admin-download-artifacts-presign-get", Method: "GET", Path: "/admin/download-artifacts/{artifact_id}/presign-get", Description: "Presign get for artifact"},
			{Name: "admin-download-assets-apply", Method: "POST", Path: "/admin/download-assets/apply", Description: "Apply download artifact"},
			{Name: "admin-download-assets-set-current", Method: "POST", Path: "/admin/download-assets/set-current", Description: "Set artifact as current"},
		}),
			cliapp.Command{
				Name:        "admin-downloads-upload-managed",
				NeedsAPI:    true,
				Description: "Upload + apply managed artifact (presign → upload → commit → apply)",
				Run:         a.cmdAdminDownloadsUploadManaged,
			},
		),
	}

	bundles := cliapp.CommandGroup{
		Title: "Admin Commerce - Bundles",
		Commands: a.endpointCommands([]endpointDef{
			{Name: "admin-bundles", Method: "GET", Path: "/admin/bundles", Description: "List bundles"},
			{Name: "admin-bundle-price-update", Method: "PATCH", Path: "/admin/bundles/{bundle_key}/prices/{price_id}", Description: "Update bundle price"},
			{Name: "admin-bundle-price-create", Method: "POST", Path: "/admin/bundles/{bundle_key}/prices", Description: "Create bundle price"},
			{Name: "admin-bundle-price-delete", Method: "DELETE", Path: "/admin/bundles/{bundle_key}/prices/{price_id}", Description: "Delete bundle price"},
		}),
	}

	coupons := cliapp.CommandGroup{
		Title: "Admin Commerce - Coupons",
		Commands: a.endpointCommands([]endpointDef{
			{Name: "admin-coupons-list", Method: "GET", Path: "/admin/coupons", Description: "List coupons"},
			{Name: "admin-coupons-create", Method: "POST", Path: "/admin/coupons", Description: "Create coupon"},
			{Name: "admin-coupons-usage", Method: "GET", Path: "/admin/coupons/usage", Description: "Coupon usage"},
			{Name: "admin-coupons-get", Method: "GET", Path: "/admin/coupons/{coupon_id}", Description: "Get coupon"},
			{Name: "admin-coupons-update", Method: "PATCH", Path: "/admin/coupons/{coupon_id}", Description: "Update coupon"},
			{Name: "admin-coupons-delete", Method: "DELETE", Path: "/admin/coupons/{coupon_id}", Description: "Delete coupon"},
			{Name: "admin-coupon-mappings", Method: "GET", Path: "/admin/coupon-mappings", Description: "List coupon mappings"},
			{Name: "admin-plan-coupon-set", Method: "PUT", Path: "/admin/plans/{price_id}/coupon", Description: "Set coupon for plan"},
			{Name: "admin-plan-coupon-remove", Method: "DELETE", Path: "/admin/plans/{price_id}/coupon", Description: "Remove coupon from plan"},
			{Name: "admin-stripe-coupons-preview", Method: "GET", Path: "/admin/stripe/coupons-preview", Description: "Stripe coupons preview"},
		}),
	}

	stripeImport := cliapp.CommandGroup{
		Title: "Admin Commerce - Stripe Import",
		Commands: a.endpointCommands([]endpointDef{
			{Name: "admin-stripe-import-preview", Method: "GET", Path: "/admin/stripe/import-preview", Description: "Stripe import preview"},
			{Name: "admin-stripe-import", Method: "POST", Path: "/admin/stripe/import", Description: "Run Stripe import"},
		}),
	}

	users := cliapp.CommandGroup{
		Title: "Admin Users",
		Commands: a.endpointCommands([]endpointDef{
			{Name: "admin-users-list", Method: "GET", Path: "/admin/users", Description: "List users"},
			{Name: "admin-users-get", Method: "GET", Path: "/admin/users/{id}", Description: "Get user"},
			{Name: "admin-users-sessions", Method: "GET", Path: "/admin/users/{id}/sessions", Description: "List user sessions"},
			{Name: "admin-users-session-revoke", Method: "DELETE", Path: "/admin/users/{id}/sessions/{sid}", Description: "Revoke user session"},
			{Name: "admin-users-sessions-revoke-all", Method: "POST", Path: "/admin/users/{id}/sessions/revoke-all", Description: "Revoke all user sessions"},
		}),
	}

	credits := cliapp.CommandGroup{
		Title: "Credits",
		Commands: a.endpointCommands([]endpointDef{
			{Name: "admin-api-keys-list", Method: "GET", Path: "/admin/api-keys", Description: "List API keys (admin)"},
			{Name: "admin-api-keys-create", Method: "POST", Path: "/admin/api-keys", Description: "Create API key (admin)"},
			{Name: "admin-api-keys-delete", Method: "DELETE", Path: "/admin/api-keys", Description: "Delete API key (admin)"},
			{Name: "admin-api-keys-test", Method: "POST", Path: "/admin/api-keys/test", Description: "Test API key (admin)"},
			{Name: "admin-api-keys-toggle", Method: "POST", Path: "/admin/api-keys/toggle", Description: "Toggle API key (admin)"},
			{Name: "admin-tiers-limits", Method: "GET", Path: "/admin/tiers/limits", Description: "List tier limits (admin)"},
			{Name: "admin-tier-limits", Method: "GET", Path: "/admin/tiers/{tier}/limits", Description: "Get tier limits (admin)"},
			{Name: "admin-tier-limits-update", Method: "PUT", Path: "/admin/tiers/{tier}/limits", Description: "Update tier limits (admin)"},
			{Name: "admin-limits-create", Method: "POST", Path: "/admin/limits", Description: "Create tier limit (admin)"},
			{Name: "admin-limits-delete", Method: "DELETE", Path: "/admin/limits", Description: "Delete tier limit (admin)"},
			{Name: "admin-app-limits", Method: "GET", Path: "/admin/apps/{app}/limits", Description: "Get app limits (admin)"},
			{Name: "usage-report", Method: "POST", Path: "/usage/report", Description: "Report usage (service auth)"},
			{Name: "usage-summary", Method: "GET", Path: "/usage/summary", Description: "Get usage summary"},
			{Name: "usage-check", Method: "GET", Path: "/usage/check", Description: "Check usage limits"},
			{Name: "usage-health", Method: "GET", Path: "/usage/health", Description: "Usage health"},
			{Name: "admin-usage-summary", Method: "GET", Path: "/admin/usage", Description: "Admin usage summary"},
		}),
	}

	docs := cliapp.CommandGroup{
		Title: "Docs",
		Commands: a.endpointCommands([]endpointDef{
			{Name: "admin-docs-tree", Method: "GET", Path: "/admin/docs/tree", Description: "Get docs tree (admin)"},
			{Name: "admin-docs-content", Method: "GET", Path: "/admin/docs/content", Description: "Get docs content (admin)"},
		}),
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{
		health,
		landing,
		auth,
		billing,
		account,
		variants,
		content,
		metrics,
		feedback,
		waitlist,
		credits,
		ai,
		admin,
		remoteProfiles,
		downloadsAdmin,
		bundles,
		coupons,
		stripeImport,
		users,
		docs,
		config,
	}
}

func (a *App) endpointCommands(defs []endpointDef) []cliapp.Command {
	commands := make([]cliapp.Command, 0, len(defs))
	for _, def := range defs {
		def := def
		commands = append(commands, a.endpointCommand(def))
	}
	return commands
}

func (a *App) endpointCommand(def endpointDef) cliapp.Command {
	return cliapp.Command{
		Name:        def.Name,
		NeedsAPI:    true,
		Description: def.Description,
		Run: func(args []string) error {
			return a.runEndpoint(def, args)
		},
	}
}

func (a *App) apiPath(v1Path string) string {
	v1Path = strings.TrimSpace(v1Path)
	if v1Path == "" {
		return ""
	}
	if !strings.HasPrefix(v1Path, "/") {
		v1Path = "/" + v1Path
	}
	base := strings.TrimRight(strings.TrimSpace(a.core.APIClient.BaseURL()), "/")
	if strings.HasSuffix(base, "/api/v1") {
		return v1Path
	}
	return "/api/v1" + v1Path
}

func (a *App) apiRoot() (string, error) {
	base := strings.TrimRight(strings.TrimSpace(a.core.APIClient.BaseURL()), "/")
	if base == "" {
		return "", fmt.Errorf("api base URL is empty")
	}
	if strings.HasSuffix(base, "/api/v1") {
		return strings.TrimSuffix(base, "/api/v1"), nil
	}
	return base, nil
}

func (a *App) authToken() string {
	headers := a.core.APIClient.AuthHeaders()
	if auth, ok := headers["Authorization"]; ok {
		const prefix = "Bearer "
		if strings.HasPrefix(auth, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(auth, prefix))
		}
	}
	return ""
}

func (a *App) requestV1(method, path string, query url.Values, payload []byte) ([]byte, error) {
	var body interface{}
	if payload != nil {
		body = json.RawMessage(payload)
	}
	return a.core.APIClient.Request(method, a.apiPath(path), query, body)
}

func (a *App) requestRoot(method, path string, query url.Values, payload []byte) ([]byte, error) {
	root, err := a.apiRoot()
	if err != nil {
		return nil, err
	}
	client := cliutil.NewHTTPClient(cliutil.HTTPClientOptions{
		BaseOptions: cliutil.APIBaseOptions{DefaultBase: root},
		Token:       a.authToken(),
		Timeout:     a.core.HTTPClient.Timeout(),
	})
	var body interface{}
	if payload != nil {
		body = json.RawMessage(payload)
	}
	return client.Do(method, path, query, body)
}

func (a *App) request(def endpointDef, path string, query url.Values, payload []byte) ([]byte, error) {
	if def.Root {
		return a.requestRoot(def.Method, path, query, payload)
	}
	if strings.HasPrefix(path, "/admin/") || path == "/admin" {
		return a.requestAdmin(def.Method, path, query, payload)
	}
	return a.requestV1(def.Method, path, query, payload)
}

func (a *App) resolveURL(path string, root bool, query url.Values) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(a.core.APIClient.BaseURL()), "/")
	if base == "" {
		return "", fmt.Errorf("api base URL is empty")
	}
	if root {
		if strings.HasSuffix(base, "/api/v1") {
			base = strings.TrimSuffix(base, "/api/v1")
		}
	} else if !strings.HasSuffix(base, "/api/v1") {
		base = base + "/api/v1"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	endpoint := base + path
	if query != nil && len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	return endpoint, nil
}

func (a *App) runEndpoint(def endpointDef, args []string) error {
	fs := flag.NewFlagSet(def.Name, flag.ContinueOnError)
	var queries cliutil.StringList
	fs.Var(&queries, "query", "Query parameters (key=value or key=value&key2=value2). Repeatable.")
	body := fs.String("body", "", "JSON body payload or @file.json")
	jsonOut := cliutil.JSONFlag(fs)
	if err := parseFlagSetInterspersed(fs, args); err != nil {
		return err
	}

	path, argNames, err := resolvePath(def.Path, fs.Args(), def.AllowRawPath)
	if err != nil {
		return fmt.Errorf("usage: %s%s [--query k=v] [--body @file.json] [--json]", def.Name, formatArgUsage(argNames))
	}

	payload, err := parseBody(*body)
	if err != nil {
		return err
	}

	query, err := parseQueries(queries.Values())
	if err != nil {
		return err
	}

	if def.Stream {
		return a.streamEndpoint(def, path, query, payload)
	}

	resp, err := a.request(def, path, query, payload)
	if err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	cliutil.PrintJSON(resp)
	return nil
}

func (a *App) streamEndpoint(def endpointDef, path string, query url.Values, payload []byte) error {
	urlString, err := a.resolveURL(path, def.Root, query)
	if err != nil {
		return err
	}

	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(def.Method, urlString, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "text/event-stream")
	for key, value := range a.core.APIClient.AuthHeaders() {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("read response: %w", readErr)
		}
		return cliutil.ParseAPIError(resp.StatusCode, data)
	}

	_, err = io.Copy(os.Stdout, resp.Body)
	return err
}

type healthResponse struct {
	Status     string                 `json:"status"`
	Service    string                 `json:"service"`
	Version    string                 `json:"version"`
	Timestamp  string                 `json:"timestamp"`
	Details    map[string]interface{} `json:"details"`
	Readiness  bool                   `json:"readiness"`
	Deps       map[string]string      `json:"dependencies"`
	Error      string                 `json:"error"`
	Message    string                 `json:"message"`
	Operations map[string]interface{} `json:"operations"`
}

type usageHealthResponse struct {
	Healthy               bool   `json:"healthy"`
	DatabaseConnected     bool   `json:"database_connected"`
	ServiceAuthConfigured bool   `json:"service_auth_configured"`
	ServiceAuthMode       string `json:"service_auth_mode"`
}

type deployReadinessCheck struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Passed   bool   `json:"passed"`
	Blocked  bool   `json:"blocked,omitempty"`
	Detail   string `json:"detail"`
}

type deployReadinessReport struct {
	Ready      bool                   `json:"ready"`
	ProfileTag string                 `json:"profile_tag,omitempty"`
	ProfileID  string                 `json:"profile_id,omitempty"`
	Domain     string                 `json:"domain,omitempty"`
	Checks     []deployReadinessCheck `json:"checks"`
	NextSteps  []string               `json:"next_steps,omitempty"`
	CheckedAt  string                 `json:"checked_at"`
}

type adminLoginResponse struct {
	Email         string `json:"email,omitempty"`
	Authenticated bool   `json:"authenticated"`
	ResetEnabled  bool   `json:"reset_enabled"`
}

type adminSessionConfig struct {
	APIBase   string     `json:"api_base"`
	Session   string     `json:"session"`
	Email     string     `json:"email,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type optionalString struct {
	value string
	set   bool
}

func (o *optionalString) String() string {
	return o.value
}

func (o *optionalString) Set(value string) error {
	o.value = value
	o.set = true
	return nil
}

func (a *App) cmdStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	resp, err := a.request(endpointDef{Method: "GET", Path: "/health", Root: true}, "/health", nil, nil)
	if err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}

	var parsed healthResponse
	if err := json.Unmarshal(resp, &parsed); err == nil && parsed.Status != "" {
		fmt.Printf("Status: %s\n", parsed.Status)
		if parsed.Service != "" {
			fmt.Printf("Service: %s\n", parsed.Service)
		}
		if parsed.Version != "" {
			fmt.Printf("Version: %s\n", parsed.Version)
		}
		if parsed.Timestamp != "" {
			fmt.Printf("Timestamp: %s\n", parsed.Timestamp)
		}
		if parsed.Readiness {
			fmt.Printf("Ready: %v\n", parsed.Readiness)
		}
		if len(parsed.Deps) > 0 {
			fmt.Println("Dependencies:")
			for key, value := range parsed.Deps {
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
		return nil
	}

	cliutil.PrintJSON(resp)
	return nil
}

func (a *App) cmdServiceAuthStatus(args []string) error {
	fs := flag.NewFlagSet("service-auth-status", flag.ContinueOnError)
	requireEnabled := fs.Bool("require-enabled", false, "Exit non-zero if service auth is not configured")
	jsonOut := cliutil.JSONFlag(fs)
	if err := parseFlagSetInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() != 0 {
		return fmt.Errorf("usage: service-auth-status [--require-enabled] [--json]")
	}

	resp, err := a.request(endpointDef{Method: "GET", Path: "/usage/health"}, "/usage/health", nil, nil)
	if err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(resp)
		var parsed usageHealthResponse
		if err := json.Unmarshal(resp, &parsed); err != nil {
			return nil
		}
		if *requireEnabled && !parsed.ServiceAuthConfigured {
			return serviceAuthNotConfiguredError()
		}
		return nil
	}

	var parsed usageHealthResponse
	if err := json.Unmarshal(resp, &parsed); err != nil {
		return fmt.Errorf("parse usage health response: %w", err)
	}

	status := "disabled"
	if parsed.ServiceAuthConfigured {
		status = "enabled"
	}
	mode := parsed.ServiceAuthMode
	if strings.TrimSpace(mode) == "" {
		mode = "unknown"
	}

	fmt.Printf("Service auth: %s\n", status)
	fmt.Printf("Mode: %s\n", mode)
	if !parsed.DatabaseConnected {
		fmt.Println("Database: disconnected")
	}

	if *requireEnabled && !parsed.ServiceAuthConfigured {
		return serviceAuthNotConfiguredError()
	}
	return nil
}

func (a *App) cmdDeployReadiness(args []string) error {
	fs := flag.NewFlagSet("deploy-readiness", flag.ContinueOnError)
	profileIDFlag := fs.String("profile-id", "", "Remote profile id to test")
	profileTagFlag := fs.String("profile-tag", "", "Remote profile tag to test")
	domainFlag := fs.String("domain", "", "Deployment domain used for next-step guidance")
	requireServiceAuth := fs.Bool("require-service-auth", true, "Require LPBS service auth to be enabled")
	jsonOut := cliutil.JSONFlag(fs)
	if err := parseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: deploy-readiness [--profile-id <id> | --profile-tag <tag>] [--domain <domain>] [--require-service-auth=true|false] [--json]")
	}

	profileID := strings.TrimSpace(*profileIDFlag)
	profileTag := strings.TrimSpace(*profileTagFlag)
	domain := strings.TrimSpace(*domainFlag)
	if profileID != "" && profileTag != "" {
		return fmt.Errorf("use either --profile-id or --profile-tag, not both")
	}

	checks := make([]deployReadinessCheck, 0, 4)
	nextSteps := make([]string, 0, 6)
	ready := true

	sessionCfg, sessionErr := a.loadAdminSession()
	adminSessionConfigured := sessionErr == nil && strings.TrimSpace(sessionCfg.Session) != ""
	adminSessionCheck := deployReadinessCheck{
		Name:     "admin_session",
		Required: true,
		Passed:   adminSessionConfigured,
	}
	if sessionErr != nil {
		adminSessionCheck.Detail = fmt.Sprintf("failed to load admin session: %v", sessionErr)
	} else if !adminSessionConfigured {
		adminSessionCheck.Detail = "admin session not configured"
	} else {
		adminSessionCheck.Detail = "admin session is configured"
	}
	checks = append(checks, adminSessionCheck)
	if !adminSessionCheck.Passed {
		ready = false
		nextSteps = append(nextSteps, "landing-page-business-suite admin-login --email <local_admin_email> --password @/path/to/local-admin-password.txt")
	}

	storageCheck := deployReadinessCheck{
		Name:     "download_storage",
		Required: true,
	}
	if adminSessionConfigured {
		_, err := a.requestAdmin("POST", "/admin/download-storage/test", nil, nil)
		if err != nil {
			storageCheck.Passed = false
			storageCheck.Detail = err.Error()
			ready = false
			nextSteps = append(nextSteps, "landing-page-business-suite admin-download-storage-test")
		} else {
			storageCheck.Passed = true
			storageCheck.Detail = "download storage test succeeded"
		}
	} else {
		storageCheck.Passed = false
		storageCheck.Blocked = true
		storageCheck.Detail = "skipped: admin session is required"
		ready = false
	}
	checks = append(checks, storageCheck)

	if profileTag != "" || profileID != "" {
		resolvedProfileID := profileID
		profileCheck := deployReadinessCheck{
			Name:     "remote_profile_session",
			Required: true,
		}
		if !adminSessionConfigured {
			profileCheck.Passed = false
			profileCheck.Blocked = true
			profileCheck.Detail = "skipped: admin session is required"
			ready = false
		} else {
			if resolvedProfileID == "" {
				id, err := a.resolveRemoteProfileIDByTag(profileTag)
				if err != nil {
					profileCheck.Passed = false
					profileCheck.Detail = err.Error()
					ready = false
				} else {
					resolvedProfileID = id
				}
			}
			if profileCheck.Detail == "" {
				_, err := a.requestAdmin("POST", "/admin/remote-profiles/"+url.PathEscape(resolvedProfileID)+"/test", nil, nil)
				if err != nil {
					profileCheck.Passed = false
					profileCheck.Detail = err.Error()
					ready = false
				} else {
					profileCheck.Passed = true
					profileCheck.Detail = "remote profile session is active"
				}
			}
		}
		checks = append(checks, profileCheck)
		if !profileCheck.Passed && !profileCheck.Blocked {
			if profileTag != "" {
				nextSteps = append(nextSteps, fmt.Sprintf("landing-page-business-suite remote-profiles-login --tag %s --email <remote_admin_email> --password @/path/to/remote-admin-password.txt", profileTag))
			} else {
				nextSteps = append(nextSteps, fmt.Sprintf("landing-page-business-suite remote-profiles-login %s --email <remote_admin_email> --password @/path/to/remote-admin-password.txt", resolvedProfileID))
			}
		}
		profileID = resolvedProfileID
	}

	serviceAuthCheck := deployReadinessCheck{
		Name:     "service_auth",
		Required: *requireServiceAuth,
	}
	serviceAuthResp, err := a.request(endpointDef{Method: "GET", Path: "/usage/health"}, "/usage/health", nil, nil)
	if err != nil {
		serviceAuthCheck.Passed = false
		serviceAuthCheck.Detail = err.Error()
		if serviceAuthCheck.Required {
			ready = false
		}
	} else {
		var parsed usageHealthResponse
		if err := json.Unmarshal(serviceAuthResp, &parsed); err != nil {
			serviceAuthCheck.Passed = false
			serviceAuthCheck.Detail = fmt.Sprintf("parse usage health response: %v", err)
			if serviceAuthCheck.Required {
				ready = false
			}
		} else if parsed.ServiceAuthConfigured {
			mode := strings.TrimSpace(parsed.ServiceAuthMode)
			if mode == "" {
				mode = "unknown"
			}
			serviceAuthCheck.Passed = true
			serviceAuthCheck.Detail = fmt.Sprintf("service auth enabled (mode=%s)", mode)
		} else {
			serviceAuthCheck.Passed = false
			serviceAuthCheck.Detail = "service auth is disabled"
			if serviceAuthCheck.Required {
				ready = false
			}
		}
	}
	checks = append(checks, serviceAuthCheck)
	if !serviceAuthCheck.Passed {
		domainArg := "<domain>"
		if domain != "" {
			domainArg = domain
		}
		nextSteps = append(nextSteps,
			fmt.Sprintf("scenario-to-cloud secrets set LPBS_SERVICE_SECRET --scenario landing-page-business-suite --generate hex:64 --targets scenario,deployment --domain %s --restart", domainArg),
			"landing-page-business-suite service-auth-status --require-enabled",
		)
		if profileTag != "" {
			nextSteps = append(nextSteps, fmt.Sprintf("scenario-to-desktop deploy-target test %s --require-service-auth", profileTag))
		}
	}

	if ready {
		nextSteps = append(nextSteps, "LPBS deploy readiness passed. Continue with scenario-to-desktop pipeline run ... --deploy-target <target> --app-key <app_key> --wait")
	}

	report := deployReadinessReport{
		Ready:      ready,
		ProfileTag: profileTag,
		ProfileID:  profileID,
		Domain:     domain,
		Checks:     checks,
		NextSteps:  nextSteps,
		CheckedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	if *jsonOut {
		encoded, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("encode readiness report: %w", err)
		}
		cliutil.PrintJSON(encoded)
		if !ready {
			return fmt.Errorf("deploy readiness checks failed")
		}
		return nil
	}

	status := "NOT READY"
	if ready {
		status = "READY"
	}
	fmt.Printf("Status: %s\n", status)
	fmt.Println("Triage:")
	for _, check := range checks {
		checkStatus := "PASS"
		if check.Blocked {
			checkStatus = "BLOCKED"
		} else if !check.Passed {
			checkStatus = "FAIL"
		}
		required := ""
		if !check.Required {
			required = " (optional)"
		}
		fmt.Printf("  [%s] %s%s: %s\n", checkStatus, check.Name, required, check.Detail)
	}
	if len(nextSteps) > 0 {
		fmt.Println("Next steps:")
		for i, step := range nextSteps {
			fmt.Printf("  %d) %s\n", i+1, step)
		}
	}
	if !ready {
		return fmt.Errorf("deploy readiness checks failed")
	}
	return nil
}

func serviceAuthNotConfiguredError() error {
	return fmt.Errorf(
		"service auth is not configured\n\nNext steps:\n  1) Set shared secret (portable): scenario-to-cloud secrets set LPBS_SERVICE_SECRET --scenario landing-page-business-suite --generate hex:64 --targets scenario,deployment --domain <domain> --restart\n  2) Verify LPBS runtime auth gate: landing-page-business-suite service-auth-status --require-enabled\n  3) Verify desktop deploy auth gate: scenario-to-desktop deploy-target test <target-name> --require-service-auth",
	)
}

func (a *App) adminSessionConfigFile() (*cliutil.ConfigFile, error) {
	if a.core == nil || a.core.ConfigFile == nil {
		return nil, fmt.Errorf("config not initialized")
	}
	dir := filepath.Dir(a.core.ConfigFile.Path)
	return cliutil.NewConfigFile(filepath.Join(dir, "admin_session.json"))
}

func (a *App) loadAdminSession() (adminSessionConfig, error) {
	cfgFile, err := a.adminSessionConfigFile()
	if err != nil {
		return adminSessionConfig{}, err
	}
	var cfg adminSessionConfig
	if err := cfgFile.Load(&cfg); err != nil {
		return cfg, err
	}
	if strings.TrimSpace(cfg.Session) == "" {
		return cfg, nil
	}
	base := strings.TrimRight(strings.TrimSpace(a.core.APIClient.BaseURL()), "/")
	if base == "" {
		return cfg, fmt.Errorf("api base URL is empty; configure an API base first")
	}
	if cfg.APIBase != "" && !strings.EqualFold(strings.TrimRight(cfg.APIBase, "/"), base) {
		_ = cfgFile.Save(adminSessionConfig{})
		return adminSessionConfig{}, nil
	}
	if cfg.ExpiresAt != nil && time.Now().After(cfg.ExpiresAt.UTC()) {
		_ = cfgFile.Save(adminSessionConfig{})
		return adminSessionConfig{}, nil
	}
	return cfg, nil
}

func (a *App) saveAdminSession(cfg adminSessionConfig) error {
	cfgFile, err := a.adminSessionConfigFile()
	if err != nil {
		return err
	}
	cfg.UpdatedAt = time.Now().UTC()
	return cfgFile.Save(cfg)
}

func (a *App) clearAdminSession() error {
	cfgFile, err := a.adminSessionConfigFile()
	if err != nil {
		return err
	}
	return cfgFile.Save(adminSessionConfig{})
}

func (a *App) requestAdmin(method, pathValue string, query url.Values, payload []byte) ([]byte, error) {
	session, err := a.loadAdminSession()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(session.Session) == "" {
		return nil, fmt.Errorf("admin session not configured. Run admin-login first")
	}

	endpoint, err := a.resolveURL(pathValue, false, query)
	if err != nil {
		return nil, err
	}

	var body io.Reader
	if len(payload) > 0 {
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if len(payload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range a.core.APIClient.AuthHeaders() {
		req.Header.Set(key, value)
	}
	req.Header.Set("Cookie", fmt.Sprintf("admin_session=%s", session.Session))

	client := &http.Client{Timeout: a.core.HTTPClient.Timeout()}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		_ = a.clearAdminSession()
	}
	if resp.StatusCode >= 400 {
		return nil, cliutil.ParseAPIError(resp.StatusCode, data)
	}
	return data, nil
}

func (a *App) requestRemoteProxy(profileID, method, pathValue string, query url.Values, headers map[string]string, body []byte) ([]byte, error) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return nil, fmt.Errorf("remote profile id is required")
	}
	if strings.TrimSpace(method) == "" {
		return nil, fmt.Errorf("proxy method is required")
	}
	if strings.TrimSpace(pathValue) == "" {
		return nil, fmt.Errorf("proxy path is required")
	}

	payload := map[string]interface{}{
		"method": method,
		"path":   pathValue,
	}
	if len(query) > 0 {
		payload["query"] = flattenQueryValues(query)
	}
	if len(headers) > 0 {
		payload["headers"] = headers
	}
	if len(body) > 0 {
		payload["body"] = json.RawMessage(body)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode proxy payload: %w", err)
	}
	return a.requestAdmin("POST", "/admin/remote-profiles/"+url.PathEscape(profileID)+"/proxy", nil, payloadBytes)
}

func (a *App) requestAdminJSON(profileID, method, pathValue string, payload interface{}) ([]byte, error) {
	var body []byte
	if payload != nil {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("encode request: %w", err)
		}
		body = payloadBytes
	}
	if strings.TrimSpace(profileID) == "" {
		return a.requestAdmin(method, pathValue, nil, body)
	}
	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	return a.requestRemoteProxy(profileID, method, pathValue, nil, headers, body)
}

func resolveSecretArg(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if strings.HasPrefix(trimmed, "@") {
		data, err := cliutil.ReadFileString(strings.TrimPrefix(trimmed, "@"))
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(data), nil
	}
	return trimmed, nil
}

type boolFlag interface {
	IsBoolFlag() bool
}

// parseFlagSetInterspersed allows trailing flags after positional arguments.
// Go's standard flag parser stops parsing flags at the first positional argument.
func parseFlagSetInterspersed(fs *flag.FlagSet, args []string) error {
	return fs.Parse(reorderInterspersedArgs(fs, args))
}

func reorderInterspersedArgs(fs *flag.FlagSet, args []string) []string {
	flags := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positionals = append(positionals, arg)
			continue
		}

		name := parseFlagTokenName(arg)
		flagDef := fs.Lookup(name)
		if flagDef == nil {
			flags = append(flags, arg)
			continue
		}

		flags = append(flags, arg)
		if strings.Contains(arg, "=") || isBoolFlag(flagDef) {
			continue
		}
		if i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}

	return append(flags, positionals...)
}

func parseFlagTokenName(arg string) string {
	token := strings.TrimLeft(arg, "-")
	if token == "" {
		return ""
	}
	if idx := strings.IndexByte(token, '='); idx >= 0 {
		return token[:idx]
	}
	return token
}

func isBoolFlag(f *flag.Flag) bool {
	if f == nil {
		return false
	}
	bf, ok := f.Value.(boolFlag)
	return ok && bf.IsBoolFlag()
}

func (a *App) cmdAdminLogin(args []string) error {
	fs := flag.NewFlagSet("admin-login", flag.ContinueOnError)
	email := fs.String("email", "", "Admin email")
	password := fs.String("password", "", "Admin password or @file")
	jsonOut := cliutil.JSONFlag(fs)
	if err := parseFlagSetInterspersed(fs, args); err != nil {
		return err
	}

	emailValue := strings.TrimSpace(*email)
	if emailValue == "" {
		return fmt.Errorf("usage: admin-login --email <email> --password <password> [--json]")
	}
	passwordValue, err := resolveSecretArg(*password)
	if err != nil {
		return fmt.Errorf("read password: %w", err)
	}
	if strings.TrimSpace(passwordValue) == "" {
		return fmt.Errorf("usage: admin-login --email <email> --password <password> [--json]")
	}

	base := strings.TrimRight(strings.TrimSpace(a.core.APIClient.BaseURL()), "/")
	if base == "" {
		return fmt.Errorf("api base URL is empty; configure an API base first")
	}

	payload, err := json.Marshal(map[string]string{
		"email":    emailValue,
		"password": passwordValue,
	})
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	endpoint, err := a.resolveURL("/admin/login", false, nil)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: a.core.HTTPClient.Timeout()}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return cliutil.ParseAPIError(resp.StatusCode, data)
	}

	var loginResp adminLoginResponse
	if err := json.Unmarshal(data, &loginResp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	if !loginResp.Authenticated {
		return fmt.Errorf("admin login failed")
	}

	cookie := findCookie(resp.Cookies(), "admin_session")
	if cookie == nil || strings.TrimSpace(cookie.Value) == "" {
		return fmt.Errorf("admin login did not return a session cookie")
	}

	cfg := adminSessionConfig{
		APIBase:   base,
		Session:   cookie.Value,
		Email:     loginResp.Email,
		ExpiresAt: deriveCookieExpiry(cookie),
	}
	if strings.TrimSpace(cfg.Email) == "" {
		cfg.Email = emailValue
	}
	if err := a.saveAdminSession(cfg); err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(data)
		return nil
	}
	fmt.Printf("Admin session stored for %s\n", cfg.Email)
	if cfg.ExpiresAt != nil {
		fmt.Printf("Session expires at %s\n", cfg.ExpiresAt.Format(time.RFC3339))
	}
	return nil
}

func (a *App) cmdAdminLogout(args []string) error {
	fs := flag.NewFlagSet("admin-logout", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := parseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 {
		return fmt.Errorf("usage: admin-logout [--json]")
	}

	session, err := a.loadAdminSession()
	if err != nil {
		return err
	}
	if strings.TrimSpace(session.Session) == "" {
		return fmt.Errorf("no admin session configured. Run admin-login first")
	}

	resp, err := a.requestAdmin("POST", "/admin/logout", nil, nil)
	if err != nil {
		if apiErr, ok := err.(*cliutil.APIError); ok {
			if apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden {
				_ = a.clearAdminSession()
				if *jsonOut {
					cliutil.PrintJSON([]byte(`{"success":true}`))
					return nil
				}
				fmt.Println("Admin session cleared")
				return nil
			}
		}
		return err
	}

	if err := a.clearAdminSession(); err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	fmt.Println("Admin session cleared")
	return nil
}

func (a *App) cmdAdminSession(args []string) error {
	fs := flag.NewFlagSet("admin-session", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := parseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 {
		return fmt.Errorf("usage: admin-session [--json]")
	}

	resp, err := a.requestAdmin("GET", "/admin/session", nil, nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	cliutil.PrintJSON(resp)
	return nil
}

func (a *App) cmdRemoteProfilesList(args []string) error {
	fs := flag.NewFlagSet("remote-profiles-list", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := parseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 {
		return fmt.Errorf("usage: remote-profiles-list [--json]")
	}
	resp, err := a.requestAdmin("GET", "/admin/remote-profiles", nil, nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	cliutil.PrintJSON(resp)
	return nil
}

func (a *App) cmdRemoteProfilesCreate(args []string) error {
	fs := flag.NewFlagSet("remote-profiles-create", flag.ContinueOnError)
	tag := fs.String("tag", "", "Profile tag (unique)")
	label := fs.String("label", "", "Profile label")
	apiBase := fs.String("api-base", "", "Remote API base (must end with /api/v1)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := parseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 {
		return fmt.Errorf("usage: remote-profiles-create --tag <tag> --api-base <url> [--label <label>] [--json]")
	}

	tagValue := strings.TrimSpace(*tag)
	apiBaseValue := strings.TrimSpace(*apiBase)
	if tagValue == "" || apiBaseValue == "" {
		return fmt.Errorf("usage: remote-profiles-create --tag <tag> --api-base <url> [--label <label>] [--json]")
	}

	payload := map[string]string{
		"tag":      tagValue,
		"api_base": apiBaseValue,
	}
	if strings.TrimSpace(*label) != "" {
		payload["label"] = strings.TrimSpace(*label)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	resp, err := a.requestAdmin("POST", "/admin/remote-profiles", nil, body)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	cliutil.PrintJSON(resp)
	return nil
}

func (a *App) cmdRemoteProfilesUpdate(args []string) error {
	fs := flag.NewFlagSet("remote-profiles-update", flag.ContinueOnError)
	var tag optionalString
	var label optionalString
	var apiBase optionalString
	fs.Var(&tag, "tag", "Updated tag")
	fs.Var(&label, "label", "Updated label (use empty string to clear)")
	fs.Var(&apiBase, "api-base", "Updated API base")
	jsonOut := cliutil.JSONFlag(fs)
	if err := parseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: remote-profiles-update <id> [--tag <tag>] [--label <label>] [--api-base <url>] [--json]")
	}
	profileID := strings.TrimSpace(fs.Args()[0])
	if profileID == "" {
		return fmt.Errorf("usage: remote-profiles-update <id> [--tag <tag>] [--label <label>] [--api-base <url>] [--json]")
	}

	payload := map[string]string{}
	if tag.set {
		payload["tag"] = strings.TrimSpace(tag.value)
	}
	if label.set {
		payload["label"] = strings.TrimSpace(label.value)
	}
	if apiBase.set {
		payload["api_base"] = strings.TrimSpace(apiBase.value)
	}
	if len(payload) == 0 {
		return fmt.Errorf("at least one of --tag, --label, or --api-base is required")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	resp, err := a.requestAdmin("PUT", "/admin/remote-profiles/"+url.PathEscape(profileID), nil, body)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	cliutil.PrintJSON(resp)
	return nil
}

func (a *App) cmdRemoteProfilesDelete(args []string) error {
	fs := flag.NewFlagSet("remote-profiles-delete", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := parseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: remote-profiles-delete <id> [--json]")
	}
	profileID := strings.TrimSpace(fs.Args()[0])
	if profileID == "" {
		return fmt.Errorf("usage: remote-profiles-delete <id> [--json]")
	}

	resp, err := a.requestAdmin("DELETE", "/admin/remote-profiles/"+url.PathEscape(profileID), nil, nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	cliutil.PrintJSON(resp)
	return nil
}

func (a *App) cmdRemoteProfilesLogin(args []string) error {
	fs := flag.NewFlagSet("remote-profiles-login", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage of remote-profiles-login:")
		fmt.Fprintln(fs.Output(), "  remote-profiles-login <id> --email <email> --password <password> [--json]")
		fmt.Fprintln(fs.Output(), "  remote-profiles-login --profile-id <id> --email <email> --password <password> [--json]")
		fmt.Fprintln(fs.Output(), "  remote-profiles-login --tag <tag> --email <email> --password <password> [--json]")
		fs.PrintDefaults()
	}
	profileIDFlag := fs.String("profile-id", "", "Remote profile id")
	tag := fs.String("tag", "", "Remote profile tag (resolves id automatically)")
	email := fs.String("email", "", "Remote admin email")
	password := fs.String("password", "", "Remote admin password or @file")
	jsonOut := cliutil.JSONFlag(fs)
	if err := parseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) > 1 {
		return fmt.Errorf("usage: remote-profiles-login <id> --email <email> --password <password> [--json]")
	}
	positionalProfileID := ""
	if len(fs.Args()) == 1 {
		positionalProfileID = strings.TrimSpace(fs.Args()[0])
	}
	flagProfileID := strings.TrimSpace(*profileIDFlag)
	tagValue := strings.TrimSpace(*tag)
	if positionalProfileID != "" && flagProfileID != "" {
		return fmt.Errorf("use either positional <id> or --profile-id, not both")
	}
	if tagValue != "" && (positionalProfileID != "" || flagProfileID != "") {
		return fmt.Errorf("use either --tag or an explicit profile id (--profile-id or positional <id>)")
	}

	profileID := positionalProfileID
	if profileID == "" {
		profileID = flagProfileID
	}
	if profileID == "" && tagValue != "" {
		resolvedProfileID, err := a.resolveRemoteProfileIDByTag(tagValue)
		if err != nil {
			return err
		}
		profileID = resolvedProfileID
	}
	if profileID == "" {
		return fmt.Errorf("usage: remote-profiles-login <id> --email <email> --password <password> [--json]")
	}
	emailValue := strings.TrimSpace(*email)
	if emailValue == "" {
		return fmt.Errorf("usage: remote-profiles-login <id> --email <email> --password <password> [--json]")
	}
	passwordValue, err := resolveSecretArg(*password)
	if err != nil {
		return fmt.Errorf("read password: %w", err)
	}
	if strings.TrimSpace(passwordValue) == "" {
		return fmt.Errorf("usage: remote-profiles-login <id> --email <email> --password <password> [--json]")
	}

	body, err := json.Marshal(map[string]string{
		"email":    emailValue,
		"password": passwordValue,
	})
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	resp, err := a.requestAdmin("POST", "/admin/remote-profiles/"+url.PathEscape(profileID)+"/login", nil, body)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	cliutil.PrintJSON(resp)
	return nil
}

func (a *App) resolveRemoteProfileIDByTag(tag string) (string, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return "", fmt.Errorf("--tag is required")
	}

	resp, err := a.requestAdmin("GET", "/admin/remote-profiles", nil, nil)
	if err != nil {
		return "", err
	}

	var payload struct {
		Profiles []struct {
			ID  json.RawMessage `json:"id"`
			Tag string          `json:"tag"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(resp, &payload); err != nil {
		return "", fmt.Errorf("parse remote profiles list: %w", err)
	}

	var matchedID string
	for _, profile := range payload.Profiles {
		if strings.TrimSpace(profile.Tag) != tag {
			continue
		}
		candidateID, err := normalizeRemoteProfileID(profile.ID)
		if err != nil {
			return "", fmt.Errorf("parse remote profile id for tag %q: %w", tag, err)
		}
		if matchedID != "" && matchedID != candidateID {
			return "", fmt.Errorf("remote profile tag %q maps to multiple ids; run remote-profiles-list --json and fix duplicates", tag)
		}
		matchedID = candidateID
	}
	if matchedID == "" {
		return "", fmt.Errorf("remote profile tag %q not found; run remote-profiles-list to inspect available tags", tag)
	}
	return matchedID, nil
}

func normalizeRemoteProfileID(raw json.RawMessage) (string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return "", fmt.Errorf("missing id")
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		asString = strings.TrimSpace(asString)
		if asString == "" {
			return "", fmt.Errorf("missing id")
		}
		return asString, nil
	}

	var asNumber json.Number
	if err := json.Unmarshal(raw, &asNumber); err == nil {
		s := strings.TrimSpace(asNumber.String())
		if s == "" {
			return "", fmt.Errorf("missing id")
		}
		return s, nil
	}

	return "", fmt.Errorf("unsupported id format %q", trimmed)
}

func (a *App) cmdRemoteProfilesLogout(args []string) error {
	fs := flag.NewFlagSet("remote-profiles-logout", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := parseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: remote-profiles-logout <id> [--json]")
	}
	profileID := strings.TrimSpace(fs.Args()[0])
	if profileID == "" {
		return fmt.Errorf("usage: remote-profiles-logout <id> [--json]")
	}

	resp, err := a.requestAdmin("POST", "/admin/remote-profiles/"+url.PathEscape(profileID)+"/logout", nil, nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	cliutil.PrintJSON(resp)
	return nil
}

func (a *App) cmdRemoteProfilesTest(args []string) error {
	fs := flag.NewFlagSet("remote-profiles-test", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage of remote-profiles-test:")
		fmt.Fprintln(fs.Output(), "  remote-profiles-test <id> [--json]")
		fmt.Fprintln(fs.Output(), "  remote-profiles-test --profile-id <id> [--json]")
		fmt.Fprintln(fs.Output(), "  remote-profiles-test --tag <tag> [--json]")
		fs.PrintDefaults()
	}
	profileIDFlag := fs.String("profile-id", "", "Remote profile id")
	tag := fs.String("tag", "", "Remote profile tag (resolves id automatically)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := parseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) > 1 {
		return fmt.Errorf("usage: remote-profiles-test <id> [--json]")
	}
	positionalProfileID := ""
	if len(fs.Args()) == 1 {
		positionalProfileID = strings.TrimSpace(fs.Args()[0])
	}
	flagProfileID := strings.TrimSpace(*profileIDFlag)
	tagValue := strings.TrimSpace(*tag)
	if positionalProfileID != "" && flagProfileID != "" {
		return fmt.Errorf("use either positional <id> or --profile-id, not both")
	}
	if tagValue != "" && (positionalProfileID != "" || flagProfileID != "") {
		return fmt.Errorf("use either --tag or an explicit profile id (--profile-id or positional <id>)")
	}

	profileID := positionalProfileID
	if profileID == "" {
		profileID = flagProfileID
	}
	if profileID == "" && tagValue != "" {
		resolvedProfileID, err := a.resolveRemoteProfileIDByTag(tagValue)
		if err != nil {
			return err
		}
		profileID = resolvedProfileID
	}
	if profileID == "" {
		return fmt.Errorf("usage: remote-profiles-test <id> [--json]")
	}

	resp, err := a.requestAdmin("POST", "/admin/remote-profiles/"+url.PathEscape(profileID)+"/test", nil, nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	cliutil.PrintJSON(resp)
	return nil
}

// DOC: docs/reference/api/admin.md#post-adminremote-profilesidproxy
func (a *App) cmdRemoteProfilesProxy(args []string) error {
	fs := flag.NewFlagSet("remote-profiles-proxy", flag.ContinueOnError)
	method := fs.String("method", "", "HTTP method (GET, POST, PUT, PATCH, DELETE)")
	pathValue := fs.String("path", "", "Admin path (e.g., /admin/download-artifacts)")
	var queries cliutil.StringList
	var headers cliutil.StringList
	body := fs.String("body", "", "JSON body payload or @file.json")
	fs.Var(&queries, "query", "Query parameters (key=value or key=value&key2=value2). Repeatable.")
	fs.Var(&headers, "header", "Header override (key=value or key:value). Repeatable.")
	jsonOut := cliutil.JSONFlag(fs)
	if err := parseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: remote-profiles-proxy <id> --method <METHOD> --path <path> [--query k=v] [--header k=v] [--body @file.json] [--json]")
	}
	profileID := strings.TrimSpace(fs.Args()[0])
	if profileID == "" {
		return fmt.Errorf("usage: remote-profiles-proxy <id> --method <METHOD> --path <path> [--query k=v] [--header k=v] [--body @file.json] [--json]")
	}
	methodValue := strings.ToUpper(strings.TrimSpace(*method))
	if methodValue == "" {
		return fmt.Errorf("method is required (use --method)")
	}
	pathValueTrimmed := strings.TrimSpace(*pathValue)
	if pathValueTrimmed == "" {
		return fmt.Errorf("path is required (use --path)")
	}
	if !strings.HasPrefix(pathValueTrimmed, "/admin") {
		return fmt.Errorf("path must start with /admin")
	}

	payloadBody, err := parseBody(*body)
	if err != nil {
		return err
	}
	queryValues, err := parseQueries(queries.Values())
	if err != nil {
		return err
	}
	headerValues, err := parseKeyValuePairs(headers.Values())
	if err != nil {
		return err
	}

	resp, err := a.requestRemoteProxy(profileID, methodValue, pathValueTrimmed, queryValues, headerValues, payloadBody)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	cliutil.PrintJSON(resp)
	return nil
}

// DOC: docs/guides/ADMIN_GUIDE.md#downloads
func (a *App) cmdAdminDownloadsUploadManaged(args []string) error {
	fs := flag.NewFlagSet("admin-downloads-upload-managed", flag.ContinueOnError)
	filePath := fs.String("file", "", "Path to artifact file")
	appKey := fs.String("app-key", "", "Download app key")
	platform := fs.String("platform", "", "Platform (windows, mac, linux)")
	releaseVersion := fs.String("release-version", "", "Release version (e.g., 1.2.3)")
	releaseNotes := fs.String("release-notes", "", "Release notes")
	checksum := fs.String("checksum", "", "Checksum string (optional)")
	requiresEntitlement := fs.Bool("requires-entitlement", false, "Require entitlement to download")
	metadata := fs.String("metadata", "", "Asset metadata JSON or @file.json")
	sha512Flag := fs.String("sha512", "", "Precomputed base64-encoded SHA512 (computed automatically if omitted)")
	contentType := fs.String("content-type", "", "Override content-type for upload")
	remoteProfile := fs.String("remote-profile", "", "Remote profile ID for proxying admin calls")
	skipApply := fs.Bool("skip-apply", false, "Skip apply step (upload + commit only)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 {
		return fmt.Errorf("usage: admin-downloads-upload-managed --file <path> --app-key <app> --platform <platform> --release-version <version> [options]")
	}

	pathValue := strings.TrimSpace(*filePath)
	appKeyValue := strings.TrimSpace(*appKey)
	platformValue, err := normalizeDownloadPlatform(*platform)
	if err != nil {
		return err
	}
	releaseVersionValue := strings.TrimSpace(*releaseVersion)
	if pathValue == "" || appKeyValue == "" || releaseVersionValue == "" {
		return fmt.Errorf("usage: admin-downloads-upload-managed --file <path> --app-key <app> --platform <platform> --release-version <version> [options]")
	}
	if _, err := os.Stat(pathValue); err != nil {
		return fmt.Errorf("artifact file not found: %w", err)
	}

	sha512Value := strings.TrimSpace(*sha512Flag)
	if sha512Value == "" {
		var err error
		sha512Value, err = computeSHA512(pathValue)
		if err != nil {
			return fmt.Errorf("compute sha512: %w", err)
		}
	}

	remoteProfileID := strings.TrimSpace(*remoteProfile)

	contentTypeValue := resolveContentType(pathValue, strings.TrimSpace(*contentType))

	var assetMetadata map[string]interface{}
	if strings.TrimSpace(*metadata) != "" {
		metadataBytes, err := parseBody(*metadata)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(metadataBytes, &assetMetadata); err != nil {
			return fmt.Errorf("metadata must be a JSON object: %w", err)
		}
	}

	presignRespBytes, err := a.requestAdminJSON(remoteProfileID, "POST", "/admin/download-artifacts/presign-upload", map[string]interface{}{
		"filename":        filepath.Base(pathValue),
		"content_type":    contentTypeValue,
		"app_key":         appKeyValue,
		"platform":        platformValue,
		"release_version": releaseVersionValue,
	})
	if err != nil {
		return err
	}
	var presignResp struct {
		UploadURL       string            `json:"upload_url"`
		RequiredHeaders map[string]string `json:"required_headers"`
		Bucket          string            `json:"bucket"`
		ObjectKey       string            `json:"object_key"`
	}
	if err := json.Unmarshal(presignRespBytes, &presignResp); err != nil {
		return fmt.Errorf("parse presign response: %w", err)
	}
	if strings.TrimSpace(presignResp.UploadURL) == "" || strings.TrimSpace(presignResp.Bucket) == "" || strings.TrimSpace(presignResp.ObjectKey) == "" {
		return fmt.Errorf("presign response missing required fields")
	}

	artifactFile, err := os.Open(pathValue)
	if err != nil {
		return fmt.Errorf("open artifact file: %w", err)
	}
	defer artifactFile.Close()

	uploadReq, err := http.NewRequest("PUT", presignResp.UploadURL, artifactFile)
	if err != nil {
		return fmt.Errorf("create upload request: %w", err)
	}
	for key, value := range presignResp.RequiredHeaders {
		if strings.EqualFold(key, "host") {
			continue
		}
		if strings.TrimSpace(value) == "" {
			continue
		}
		uploadReq.Header.Set(key, value)
	}
	if uploadReq.Header.Get("Content-Type") == "" {
		uploadReq.Header.Set("Content-Type", contentTypeValue)
	}

	uploadClient := &http.Client{Timeout: a.core.HTTPClient.Timeout()}
	uploadResp, err := uploadClient.Do(uploadReq)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}
	defer uploadResp.Body.Close()
	if uploadResp.StatusCode >= http.StatusBadRequest {
		bodyBytes, _ := io.ReadAll(uploadResp.Body)
		if len(bodyBytes) > 0 {
			return fmt.Errorf("upload failed (%d): %s", uploadResp.StatusCode, strings.TrimSpace(string(bodyBytes)))
		}
		return fmt.Errorf("upload failed (%d)", uploadResp.StatusCode)
	}

	commitRespBytes, err := a.requestAdminJSON(remoteProfileID, "POST", "/admin/download-artifacts/commit", map[string]interface{}{
		"bucket":            presignResp.Bucket,
		"object_key":        presignResp.ObjectKey,
		"original_filename": filepath.Base(pathValue),
		"content_type":      contentTypeValue,
		"app_key":           appKeyValue,
		"platform":          platformValue,
		"release_version":   releaseVersionValue,
		"sha512":            sha512Value,
	})
	if err != nil {
		return err
	}
	var artifactResp struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(commitRespBytes, &artifactResp); err != nil {
		return fmt.Errorf("parse commit response: %w", err)
	}
	if artifactResp.ID == 0 {
		return fmt.Errorf("commit response missing artifact id")
	}

	var applyRespBytes []byte
	if !*skipApply {
		applyPayload := map[string]interface{}{
			"app_key":         appKeyValue,
			"platform":        platformValue,
			"artifact_id":     artifactResp.ID,
			"release_version": releaseVersionValue,
		}
		if strings.TrimSpace(*releaseNotes) != "" {
			applyPayload["release_notes"] = strings.TrimSpace(*releaseNotes)
		}
		if strings.TrimSpace(*checksum) != "" {
			applyPayload["checksum"] = strings.TrimSpace(*checksum)
		}
		if *requiresEntitlement {
			applyPayload["requires_entitlement"] = true
		}
		if assetMetadata != nil {
			applyPayload["metadata"] = assetMetadata
		}
		applyRespBytes, err = a.requestAdminJSON(remoteProfileID, "POST", "/admin/download-assets/apply", applyPayload)
		if err != nil {
			return err
		}
	}

	if *jsonOut {
		result := map[string]json.RawMessage{
			"artifact": json.RawMessage(commitRespBytes),
		}
		if !*skipApply && len(applyRespBytes) > 0 {
			result["asset"] = json.RawMessage(applyRespBytes)
		}
		out, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("encode response: %w", err)
		}
		cliutil.PrintJSON(out)
		return nil
	}

	fmt.Printf("Uploaded artifact %s (id: %d)\n", filepath.Base(pathValue), artifactResp.ID)
	if *skipApply {
		fmt.Println("Skipped apply step (upload + commit only)")
	} else {
		fmt.Printf("Applied artifact to %s/%s\n", appKeyValue, platformValue)
	}
	return nil
}

func (a *App) cmdAssetsUpload(args []string) error {
	fs := flag.NewFlagSet("admin-assets-upload", flag.ContinueOnError)
	filePath := fs.String("file", "", "Path to file")
	category := fs.String("category", "", "Asset category")
	altText := fs.String("alt-text", "", "Alt text")
	uploadedBy := fs.String("uploaded-by", "", "Uploaded by")
	jsonOut := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	pathValue := strings.TrimSpace(*filePath)
	if pathValue == "" {
		return fmt.Errorf("usage: admin-assets-upload --file path [--category=...] [--alt-text=...] [--uploaded-by=...] [--json]")
	}

	file, err := os.Open(pathValue)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filepath.Base(pathValue))
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	if strings.TrimSpace(*category) != "" {
		if err := writer.WriteField("category", strings.TrimSpace(*category)); err != nil {
			return fmt.Errorf("write category: %w", err)
		}
	}
	if strings.TrimSpace(*altText) != "" {
		if err := writer.WriteField("alt_text", strings.TrimSpace(*altText)); err != nil {
			return fmt.Errorf("write alt_text: %w", err)
		}
	}
	if strings.TrimSpace(*uploadedBy) != "" {
		if err := writer.WriteField("uploaded_by", strings.TrimSpace(*uploadedBy)); err != nil {
			return fmt.Errorf("write uploaded_by: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close writer: %w", err)
	}

	endpoint, err := a.resolveURL("/admin/assets/upload", false, nil)
	if err != nil {
		return err
	}
	session, err := a.loadAdminSession()
	if err != nil {
		return err
	}
	if strings.TrimSpace(session.Session) == "" {
		return fmt.Errorf("admin session not configured. Run admin-login first")
	}

	req, err := http.NewRequest("POST", endpoint, &buf)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	for key, value := range a.core.APIClient.AuthHeaders() {
		req.Header.Set(key, value)
	}
	req.Header.Set("Cookie", fmt.Sprintf("admin_session=%s", session.Session))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: a.core.HTTPClient.Timeout()}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		_ = a.clearAdminSession()
	}
	if resp.StatusCode >= 400 {
		return cliutil.ParseAPIError(resp.StatusCode, data)
	}

	if *jsonOut {
		cliutil.PrintJSON(data)
		return nil
	}
	cliutil.PrintJSON(data)
	return nil
}

func (a *App) cmdAIStream(args []string) error {
	fs := flag.NewFlagSet("ai-stream", flag.ContinueOnError)
	body := fs.String("body", "", "JSON body payload or @file.json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	payload, err := parseBody(*body)
	if err != nil {
		return err
	}
	if payload == nil {
		return fmt.Errorf("usage: ai-stream --body '{...}'")
	}
	def := endpointDef{Method: "POST", Path: "/ai/stream"}
	return a.streamEndpoint(def, "/ai/stream", nil, payload)
}

var pathParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

func resolvePath(template string, args []string, allowRaw bool) (string, []string, error) {
	matches := pathParamRegex.FindAllStringSubmatch(template, -1)
	argNames := make([]string, 0, len(matches))
	for _, match := range matches {
		argNames = append(argNames, match[1])
	}
	if len(args) < len(argNames) {
		return "", argNames, fmt.Errorf("missing arguments")
	}
	if len(args) > len(argNames) {
		return "", argNames, fmt.Errorf("too many arguments")
	}
	path := template
	for i, name := range argNames {
		replacement := strings.TrimSpace(args[i])
		if replacement == "" {
			return "", argNames, fmt.Errorf("empty %s", name)
		}
		if !allowRaw {
			replacement = url.PathEscape(replacement)
		}
		path = strings.Replace(path, "{"+name+"}", replacement, 1)
	}
	return path, argNames, nil
}

func formatArgUsage(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return " " + strings.Join(names, " ")
}

func parseBody(value string) ([]byte, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return nil, nil
	}
	if raw == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		raw = strings.TrimSpace(string(data))
	} else if strings.HasPrefix(raw, "@") {
		path := strings.TrimSpace(strings.TrimPrefix(raw, "@"))
		if path == "" {
			return nil, fmt.Errorf("body file path is empty")
		}
		data, err := cliutil.ReadFileString(path)
		if err != nil {
			return nil, fmt.Errorf("read body file: %w", err)
		}
		raw = strings.TrimSpace(data)
	}
	if raw == "" {
		return nil, fmt.Errorf("body is empty")
	}
	var decoded interface{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, fmt.Errorf("body must be valid JSON: %w", err)
	}
	return []byte(raw), nil
}

func parseQueries(values []string) (url.Values, error) {
	if len(values) == 0 {
		return nil, nil
	}
	query := url.Values{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		parsed, err := url.ParseQuery(value)
		if err != nil {
			return nil, fmt.Errorf("invalid query %q: %w", value, err)
		}
		for key, vals := range parsed {
			for _, v := range vals {
				query.Add(key, v)
			}
		}
	}
	if len(query) == 0 {
		return nil, nil
	}
	return query, nil
}

func flattenQueryValues(values url.Values) map[string]string {
	if len(values) == 0 {
		return nil
	}
	flat := make(map[string]string)
	for key, vals := range values {
		if len(vals) == 0 {
			continue
		}
		flat[key] = vals[0]
	}
	if len(flat) == 0 {
		return nil
	}
	return flat
}

func parseKeyValuePairs(values []string) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	pairs := map[string]string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		var key string
		var val string
		if strings.Contains(value, "=") {
			parts := strings.SplitN(value, "=", 2)
			key = strings.TrimSpace(parts[0])
			val = strings.TrimSpace(parts[1])
		} else if strings.Contains(value, ":") {
			parts := strings.SplitN(value, ":", 2)
			key = strings.TrimSpace(parts[0])
			val = strings.TrimSpace(parts[1])
		} else {
			return nil, fmt.Errorf("invalid pair %q: expected key=value", value)
		}
		if key == "" {
			return nil, fmt.Errorf("invalid pair %q: empty key", value)
		}
		pairs[key] = val
	}
	if len(pairs) == 0 {
		return nil, nil
	}
	return pairs, nil
}

func computeSHA512(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open file for SHA512: %w", err)
	}
	defer f.Close()
	h := sha512.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("compute SHA512: %w", err)
	}
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

func normalizeDownloadPlatform(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "windows", "win":
		return "windows", nil
	case "mac", "macos", "osx":
		return "mac", nil
	case "linux":
		return "linux", nil
	case "":
		return "", fmt.Errorf("platform is required (windows, mac, linux)")
	default:
		return "", fmt.Errorf("unsupported platform %q (use windows, mac, or linux)", raw)
	}
}

func resolveContentType(pathValue, override string) string {
	trimmed := strings.TrimSpace(override)
	if trimmed != "" {
		return trimmed
	}
	ext := strings.ToLower(filepath.Ext(pathValue))
	if ext != "" {
		if guessed := mime.TypeByExtension(ext); guessed != "" {
			return guessed
		}
	}
	return "application/octet-stream"
}

func deriveCookieExpiry(cookie *http.Cookie) *time.Time {
	if cookie == nil {
		return nil
	}
	if !cookie.Expires.IsZero() {
		expiry := cookie.Expires.UTC()
		return &expiry
	}
	if cookie.MaxAge > 0 {
		expiry := time.Now().Add(time.Duration(cookie.MaxAge) * time.Second).UTC()
		return &expiry
	}
	return nil
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie != nil && cookie.Name == name {
			return cookie
		}
	}
	return nil
}
