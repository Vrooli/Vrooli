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
	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/eventpublisher"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
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
	"compute-manager/internal/provider/adoptedvps"
	"compute-manager/internal/provider/akash"
	akashclient "compute-manager/internal/provider/akash/client"
	"compute-manager/internal/provider/digitalocean"
	"compute-manager/internal/provider/hetzner"
	qemuprovider "compute-manager/internal/provider/localqemu"
	"compute-manager/internal/provision"
	internalreconcile "compute-manager/internal/reconcile"
	setup "compute-manager/internal/setup"
	"compute-manager/internal/store"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	"github.com/vrooli/vrooli/packages/provider-effect-gate"
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
	if err := store.MigrateCompatibility(context.Background(), db.Primary()); err != nil {
		log.Fatalf("store compatibility migration failed: %v", err)
	}
	if err := setup.MigrateCompatibility(context.Background(), db.Primary()); err != nil {
		log.Fatalf("setup compatibility migration failed: %v", err)
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
	eventbus.UsePublisherCredentials(func(ctx context.Context, source string) (string, error) {
		return credentials.Resolve(ctx, eventpublisher.CredentialIdentity(source), eventpublisher.CredentialField)
	})
	decisionPublisher := eventbus.NewDiscoveredClient(context.Background())
	setupCredential := func(ctx context.Context, field string) (string, error) {
		return credentials.Resolve(ctx, "vrooli/compute-manager", field)
	}
	setupCredentialConfigured := func(ctx context.Context, field string) (bool, error) {
		status, err := credentials.Status(ctx, "vrooli/compute-manager", field)
		if err != nil {
			return false, err
		}
		return status.Configured, nil
	}
	setupCatalog := setup.Catalog{Repo: setup.NewRepository(db.Primary()), Now: time.Now}
	effects, err := effectgate.New(effectgate.Config{
		Default: effectgate.Stub,
		Modes: map[string]effectgate.Mode{
			"hetzner": effectgate.Live, "digitalocean": effectgate.Live,
			"adopted-vps": effectgate.Live, "local-qemu": effectgate.Live,
			"akash": effectgate.Stub,
		},
	})
	if err != nil {
		log.Fatalf("provider effect gate configuration failed: %v", err)
	}
	setupModule := setup.NewModuleWithCredentialStatus(db.Primary(), setupCredential, setupCredentialConfigured, decisionPublisher)
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
	if err := providers.Register(&adoptedvps.Adapter{Now: time.Now}); err != nil {
		log.Fatalf("provider registry configuration failed: %v", err)
	}
	var akashSandboxClient akash.SandboxClient
	if cfg.AkashNetwork == akashclient.NetworkSandbox2 && cfg.AkashChainID != "" && cfg.AkashEndpoint != "" && cfg.AkashRPCEndpoint != "" && cfg.AkashProviderAddress != "" {
		client, clientErr := akashclient.New(akashclient.Config{
			Network: cfg.AkashNetwork, ChainID: cfg.AkashChainID, OwnerAddress: cfg.AkashProviderAddress,
			RESTEndpoint: cfg.AkashEndpoint, RPCEndpoint: cfg.AkashRPCEndpoint,
			CredentialRef: akash.TestnetWalletCredentialRef,
		}, nil, nil)
		if clientErr != nil {
			slog.Warn("Akash sandbox client is not configured; provider remains unavailable", "error", clientErr)
		} else {
			akashSandboxClient = client
		}
	}
	if err := providers.Register(&akash.Adapter{
		Config: akash.Config{
			Network: cfg.AkashNetwork, ChainID: cfg.AkashChainID, Endpoint: cfg.AkashEndpoint,
			ProviderAddress: cfg.AkashProviderAddress, WalletCredentialRef: akash.TestnetWalletCredentialRef,
		},
		Gate: effects, SandboxClient: akashSandboxClient,
		WalletConfigured: func(ctx context.Context) (bool, error) {
			return setupCredentialConfigured(ctx, "akash-testnet-wallet")
		},
	}); err != nil {
		log.Fatalf("provider registry configuration failed: %v", err)
	}
	capacityBroker, capacityErr := sharedcapacity.NewBroker(context.Background(), "")
	if capacityErr != nil {
		slog.Error("local-QEMU capacity gate unavailable; local-QEMU stays fail-closed", "error", capacityErr)
	} else {
		defer capacityBroker.Close()
	}
	localQEMUProvider := &qemuprovider.Adapter{Roots: fileRoots, Now: time.Now, RequireCapacity: true}
	if capacityBroker != nil {
		localQEMUProvider.Admission = qemuprovider.NewBrokerAdmission(capacityBroker)
	}
	if err := localQEMUProvider.RecoverCapacity(context.Background()); err != nil {
		slog.Error("local-QEMU capacity recovery failed; provider is disabled until owner reconciliation", "error", err)
	}
	defer func() {
		if err := localQEMUProvider.CloseCapacity(); err != nil {
			slog.Error("failed to release local-QEMU owner lock", "error", err)
		}
	}()
	if err := providers.Register(localQEMUProvider); err != nil {
		log.Fatalf("provider registry configuration failed: %v", err)
	}
	// Local-QEMU has no provider-side compute or IPv4 fee, but admission still
	// requires a durable, exact offer fact. Declare the supported host offers
	// through the catalogue owner at startup; LookupPrice itself remains read-
	// only and refuses a missing or stale declaration.
	for _, profile := range []string{"headless-linux", "desktop-linux"} {
		if _, err := setupCatalog.RecordLocalQEMUOffer(context.Background(), profile); err != nil {
			log.Fatalf("local-qemu price declaration failed for %s: %v", profile, err)
		}
	}
	intents := intent.Service{Store: intent.SQLStore{DB: db.Primary()}, Provider: cloudProvider, Providers: providers, Now: time.Now}
	lpbsBaseURL := cfg.LPBSBaseURL
	if lpbsBaseURL == "" {
		lpbsBaseURL, _ = discovery.ResolveScenarioURLDefault(context.Background(), "landing-page-business-suite")
	}
	bridgeBaseURL := cfg.BridgeBaseURL
	if bridgeBaseURL == "" {
		bridgeBaseURL, _ = discovery.ResolveScenarioURLDefault(context.Background(), "vrooli-bridge")
	}
	credits := &integration.LPBSCredits{BaseURL: lpbsBaseURL, Token: func(context.Context) (string, error) { return os.Getenv("COMPUTE_MANAGER_API_TOKEN"), nil }}
	provisioner := &provision.Service{Intents: intents, Meter: meter.Service{Credits: credits, DB: db.Primary(), TenantCeilingMinutes: cfg.TenantCeilingMinutes, Now: time.Now}, MoneyCap: provision.MoneyCap{Currency: "USD", MonthlyMinor: 709, TaxBasis: "pre-tax"}, Provider: cloudProvider, Providers: providers, Prices: setupCatalog, Effects: effects, Window: 24 * time.Hour, DB: db.Primary(), Mu: &sync.Mutex{}}
	bridge := &integration.BridgeClient{BaseURL: bridgeBaseURL, Token: func(context.Context) (string, error) { return os.Getenv("COMPUTE_MANAGER_API_TOKEN"), nil }}
	workerCtx, stopWorkers := context.WithCancel(context.Background())
	firstTouchVerifier := authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{Audience: "scenario-authenticator:compute-manager-adoption-first-touch", ExclusiveAudience: true, Kind: identity.ActorService})
	deploymentLifecycleVerifier := authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{Audience: "scenario-authenticator:compute-manager-deployment-lifecycle", ExclusiveAudience: true, Kind: identity.ActorService})
	instanceModule, err := instanceH.NewModuleWithRuntimeProvidersAndLifecycleAuth(providers, time.Now, &internalinstance.Repository{DB: db.Primary()}, provisioner, func(ctx context.Context, expiry time.Time) (string, error) {
		key, _, err := bridge.GetOnboardingPublicKey(ctx)
		if err != nil {
			return "", err
		}
		return enroll.RenderFirstBoot(key, expiry)
	}, bridge, firstTouchVerifier, func(ctx context.Context) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, "vrooli-onboarding")
	}, deploymentLifecycleVerifier)
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
	authConfig, err := authn.FromEnvironment(os.Getenv)
	if err != nil {
		log.Fatalf("authentication configuration invalid: %v", err)
	}
	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default(), Auth: authConfig},
		healthH.Module(db, "compute-manager-api", "1.0.0"),
		capsH.Module(capabilities.NewRegistry()),
		instanceModule,
		intentModule,
		meterModule,
		reconcileModule,
		setupModule,
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)
	secret, secretErr := setup.ResolveWebhookSecret(credentialAuthority)
	if secretErr != nil {
		log.Printf("notification-hub resolution webhook unavailable: %v", secretErr)
	} else if secret != "" {
		reader := askreader.NewReader("compute-manager", func(ctx context.Context) (string, error) {
			return credentials.Resolve(ctx, askreader.CredentialIdentity("compute-manager"), askreader.CredentialField)
		})
		rootMux.Handle(setup.DecisionWebhookPath, setup.DecisionWebhook(secret, reader, setup.NewRepository(db.Primary()), internalreconcile.InvoiceApprovalEffect{DB: db.Primary(), Reader: reader}))
	}

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
