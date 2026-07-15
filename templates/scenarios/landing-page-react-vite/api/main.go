// Command landing-page-react-vite-api is the Connect-RPC API for the landing
// page business template. It follows the api-core module architecture: each
// business domain lives in handlers/<domain> and returns a module.Module that
// mounts its generated Connect-RPC service; main.go only connects the database,
// applies migration-owned schemas, seeds demo data, and wires the modules into
// the shared server. There is no central routes.go.
package main

import (
	"context"
	"landing-page-react-vite-api/internal/clock"
	"landing-page-react-vite-api/internal/modules"
	"landing-page-react-vite-api/internal/server"
	"landing-page-react-vite-api/internal/variantspace"
	"log"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"

	accountH "landing-page-react-vite-api/handlers/account"
	adminH "landing-page-react-vite-api/handlers/admin"
	assetsH "landing-page-react-vite-api/handlers/assets"
	brandingH "landing-page-react-vite-api/handlers/branding"
	bundlesH "landing-page-react-vite-api/handlers/bundles"
	contentH "landing-page-react-vite-api/handlers/content"
	docsH "landing-page-react-vite-api/handlers/docs"
	downloadH "landing-page-react-vite-api/handlers/download"
	healthH "landing-page-react-vite-api/handlers/health"
	metricsH "landing-page-react-vite-api/handlers/metrics"
	paymentsH "landing-page-react-vite-api/handlers/payments"
	resetH "landing-page-react-vite-api/handlers/reset"
	seoH "landing-page-react-vite-api/handlers/seo"
	variantH "landing-page-react-vite-api/handlers/variant"
	variantspaceH "landing-page-react-vite-api/handlers/variantspace"

	accountsvc "landing-page-react-vite-api/internal/account"
	adminsvc "landing-page-react-vite-api/internal/admin"
	adminresetsvc "landing-page-react-vite-api/internal/adminreset"
	assetssvc "landing-page-react-vite-api/internal/assets"
	brandingsvc "landing-page-react-vite-api/internal/branding"
	contentsvc "landing-page-react-vite-api/internal/content"
	docssvc "landing-page-react-vite-api/internal/docs"
	downloadsvc "landing-page-react-vite-api/internal/download"
	metricssvc "landing-page-react-vite-api/internal/metrics"
	paymentsettingssvc "landing-page-react-vite-api/internal/paymentsettings"
	plansvc "landing-page-react-vite-api/internal/plan"
	seosvc "landing-page-react-vite-api/internal/seo"
	stripesvc "landing-page-react-vite-api/internal/stripe"
	variantsvc "landing-page-react-vite-api/internal/variant"
)

// downloadEntitlements adapts the account service to the download authorizer's
// entitlement-provider interface (only the status label is needed for gating).
type downloadEntitlements struct{ svc *accountsvc.Service }

func (d downloadEntitlements) GetEntitlements(userIdentity string) (*downloadsvc.Entitlements, error) {
	ent, err := d.svc.GetEntitlements(userIdentity)
	if err != nil {
		return nil, err
	}
	if ent == nil {
		return nil, nil
	}
	return &downloadsvc.Entitlements{Status: ent.Status}, nil
}

const (
	serviceName = "landing-page-react-vite-api"
	version     = "1.0.0"
)

func main() {
	// Preflight must run first so the binary can re-exec itself after a
	// stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "{{SCENARIO_ID}}"}) {
		return
	}

	ctx := context.Background()

	// Connect to PostgreSQL with automatic retry/backoff. DSN and pool
	// settings are resolved from the environment by api-core/database.
	db, err := database.Connect(ctx, database.Config{Driver: database.DriverPostgres})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	// Apply every domain's migration-owned schema (system first).
	if err := database.EnsureSchemas(ctx, db, modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	// Seed idempotent demo/default data (admin user, singletons, defaults).
	if err := seedDefaultData(ctx, db); err != nil {
		log.Fatalf("seed default data failed: %v", err)
	}

	// Shared domain services (constructed once, wired into modules below).
	brandingService := brandingsvc.NewService(db)
	contentService := contentsvc.NewService(db)
	variantSpace := variantspace.Load()
	variantService := variantsvc.NewService(db, variantSpace, contentService)

	// Financial cluster services (plan -> stripe/account share the plan service).
	planService := plansvc.NewService(db)
	paymentSettings := paymentsettingssvc.NewService(db)
	stripeService := stripesvc.NewService(db, planService, paymentSettings)
	accountService := accountsvc.NewService(db, planService)

	// Admin cluster services: session auth, env-gated demo reset (reseeds via
	// seedDefaultData), and the entitlement-gated download catalog.
	adminService := adminsvc.NewService(db)
	resetService := adminresetsvc.NewService(db, func(ctx context.Context) error { return seedDefaultData(ctx, db) })
	downloadService := downloadsvc.NewService(db)
	downloadAuthorizer := downloadsvc.NewAuthorizer(downloadService, downloadEntitlements{svc: accountService}, planService.BundleKey())

	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: log.Default()},
		healthH.Module(db, serviceName, version),
		brandingH.Module(brandingService, log.Default()),
		variantH.Module(variantService, log.Default()),
		contentH.Module(contentService, log.Default()),
		metricsH.Module(metricssvc.NewService(db), log.Default()),
		seoH.Module(seosvc.NewService(brandingService, variantService), log.Default()),
		docsH.Module(docssvc.NewService(""), log.Default()),
		variantspaceH.Module(variantSpace, log.Default()),
		paymentsH.Module(paymentsH.Deps{Stripe: stripeService, Plan: planService, PaymentSettings: paymentSettings, Logger: log.Default()}),
		bundlesH.Module(planService, log.Default()),
		accountH.Module(accountService, log.Default()),
		adminH.Module(adminService, log.Default()),
		resetH.Module(resetService, adminService, log.Default()),
		downloadH.Module(downloadService, downloadAuthorizer, planService, log.Default()),
		assetsH.Module(assetssvc.NewService(db), log.Default()),
	)

	if err := apiserver.Run(apiserver.Config{
		Handler: srv.Handler(),
		Cleanup: func(context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
