package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"compute-manager/internal/capabilities"
	appconfig "compute-manager/internal/config"
	"compute-manager/internal/modules"
	"compute-manager/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	capsH "compute-manager/handlers/capabilities"
	healthH "compute-manager/handlers/health"
	instanceH "compute-manager/handlers/instance"
	intentH "compute-manager/handlers/intent"
	meterH "compute-manager/handlers/meter"
	reconcileH "compute-manager/handlers/reconcile"
	"compute-manager/internal/enroll"
	"compute-manager/internal/expiry"
	internalinstance "compute-manager/internal/instance"
	"compute-manager/internal/integration"
	"compute-manager/internal/intent"
	"compute-manager/internal/meter"
	"compute-manager/internal/provider"
	"compute-manager/internal/provider/digitalocean"
	"compute-manager/internal/provider/hetzner"
	"compute-manager/internal/provision"
	internalreconcile "compute-manager/internal/reconcile"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

// scenarioStorageRoots resolves all filesystem storage classes once at
// startup. Domain file stores receive the routed roots and select the
// request-appropriate class at their own storage seam.
func scenarioStorageRoots() (storage.Paths, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return storage.Paths{}, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("compute-manager")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve compute-manager storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "compute-manager"}) {
		return
	}
	cfg, err := appconfig.Load()
	if err != nil {
		log.Fatalf("configuration invalid: %v", err)
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "compute-manager",
		MaxOpenConns: 8,
		MaxIdleConns: 8,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	primaryFileRoots, err := scenarioStorageRoots()
	if err != nil {
		log.Fatalf("file storage configuration failed: %v", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)

	credentialAuthority, err := credentialauthority.Default()
	if err != nil {
		log.Fatalf("credential authority configuration failed: %v", err)
	}
	credentials, err := credentialclient.NewClient(credentialclient.ClientOptions{Authority: credentialAuthority})
	if err != nil {
		log.Fatalf("credential client configuration failed: %v", err)
	}
	cloudProvider := &hetzner.Provider{Now: time.Now, Token: func(ctx context.Context) (string, error) {
		return credentials.Resolve(ctx, "vrooli/compute-manager", "hetzner-api-token")
	}}
	providers := provider.NewRegistry(cloudProvider)
	digitalOceanProvider := &digitalocean.Provider{Now: time.Now, Token: func(ctx context.Context) (string, error) {
		return credentials.Resolve(ctx, "vrooli/compute-manager", "digitalocean-api-token")
	}}
	if err := providers.Register(digitalOceanProvider); err != nil {
		log.Fatalf("provider registry configuration failed: %v", err)
	}
	intents := intent.Service{Store: intent.SQLStore{DB: db.Primary()}, Provider: cloudProvider, Now: time.Now}
	lpbsBaseURL := cfg.LPBSBaseURL
	if lpbsBaseURL == "" {
		lpbsBaseURL, _ = discovery.ResolveScenarioURLDefault(context.Background(), "landing-page-business-suite")
	}
	bridgeBaseURL := cfg.BridgeBaseURL
	if bridgeBaseURL == "" {
		bridgeBaseURL, _ = discovery.ResolveScenarioURLDefault(context.Background(), "vrooli-bridge")
	}
	credits := &integration.LPBSCredits{BaseURL: lpbsBaseURL, Token: func(context.Context) (string, error) { return os.Getenv("COMPUTE_MANAGER_API_TOKEN"), nil }}
	provisioner := &provision.Service{Intents: intents, Meter: meter.Service{Credits: credits, DB: db.Primary(), TenantCeilingMinutes: cfg.TenantCeilingMinutes, Now: time.Now}, Provider: cloudProvider, Providers: providers, Window: 24 * time.Hour, DB: db.Primary(), Mu: &sync.Mutex{}}
	bridge := &integration.BridgeClient{BaseURL: bridgeBaseURL, Token: func(context.Context) (string, error) { return os.Getenv("COMPUTE_MANAGER_API_TOKEN"), nil }}
	workerCtx, stopWorkers := context.WithCancel(context.Background())
	instanceModule, err := instanceH.NewModuleWithRuntimeProviders(providers, time.Now, &internalinstance.Repository{DB: db.Primary()}, provisioner, func(ctx context.Context, expiry time.Time) (string, error) {
		key, _, err := bridge.GetOnboardingPublicKey(ctx)
		if err != nil {
			return "", err
		}
		return enroll.RenderFirstBoot(key, expiry)
	}, bridge)
	if err != nil {
		log.Fatalf("instance module configuration failed: %v", err)
	}
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
				if _, err := intents.RecoverOpen(workerCtx); err != nil {
					slog.Error("open intent recovery sweep failed", "error", err)
				}
				if err := instanceH.RetryPendingEnrollments(workerCtx, db.Primary(), bridge, time.Now); err != nil {
					slog.Error("enrollment retry sweep failed", "error", err)
				}
			}
		}
	}()
	go func() {
		ticker := time.NewTicker(cfg.ExpiryInterval)
		defer ticker.Stop()
		for {
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
				sweeper := expiry.Sweeper{DB: db.Primary(), Provider: cloudProvider, Now: time.Now, Finalize: provisioner.FinalizeReservation}
				if err := sweeper.Run(workerCtx); err != nil {
					slog.Error("expiry sweep failed", "error", err)
				}
			}
		}
	}()
	go func() {
		interval := provisioner.Window / 2
		if interval <= 0 {
			return
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
				if err := provisioner.RenewReservations(workerCtx); err != nil {
					slog.Error("reservation renewal sweep failed", "error", err)
				}
			}
		}
	}()
	go func() {
		runner := internalreconcile.DailyCostRunner{
			Source:    digitalOceanProvider,
			DB:        db.Primary(),
			Provider:  digitalOceanProvider.Name(),
			Threshold: 5,
			Now:       time.Now,
			Report: func(observations []internalreconcile.CostObservation) {
				for _, observation := range observations {
					if observation.Alarm {
						slog.Error("provider billing diverges from local meter", "provider", observation.ProviderID, "instance_id", observation.InstanceID, "metered_minutes", observation.MeteredMinutes, "provider_minutes", observation.ProviderMinutes, "delta_minutes", observation.DeltaMinutes)
					}
				}
			},
		}
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
				day := time.Now().UTC().Add(-24 * time.Hour)
				if err := runner.Run(workerCtx, day); err != nil {
					slog.Warn("daily provider billing reconciliation unavailable", "provider", digitalOceanProvider.Name(), "error", err)
				}
			}
		}
	}()
	intentModule, err := intentH.NewModule(db.Primary())
	if err != nil {
		log.Fatalf("intent module configuration failed: %v", err)
	}
	meterModule, err := meterH.NewModuleWithCeiling(db.Primary(), cfg.TenantCeilingMinutes)
	if err != nil {
		log.Fatalf("meter module configuration failed: %v", err)
	}
	reconcileModule, err := reconcileH.NewModuleWithRunnerAndDestroy(db.Primary(), reconcileH.SweepRunner(db.Primary(), internalreconcile.Service{Provider: cloudProvider}, provisioner.FinalizeReservation), func(ctx context.Context, providerName, instanceID string) error {
		selected, err := providers.Get(providerName)
		if err != nil {
			return err
		}
		return selected.Destroy(ctx, instanceID)
	})
	if err != nil {
		log.Fatalf("reconcile module configuration failed: %v", err)
	}
	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "compute-manager-api", "1.0.0"),
		capsH.Module(capabilities.NewRegistry()),
		instanceModule,
		intentModule,
		meterModule,
		reconcileModule,
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)

	rootMux.Handle("/", srv.Handler())

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		Cleanup: func(ctx context.Context) error {
			stopWorkers()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
