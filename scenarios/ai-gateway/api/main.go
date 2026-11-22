package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai-gateway/internal/catalog"
	"ai-gateway/internal/gatewaycatalog"
	internalidentity "ai-gateway/internal/identity"
	inference "ai-gateway/internal/inference"
	"ai-gateway/internal/module"
	"ai-gateway/internal/modules"
	"ai-gateway/internal/providers"
	"ai-gateway/internal/rolecatalog"
	"ai-gateway/internal/server"

	"github.com/vrooli/api-core/schedule"

	sharedpolicy "github.com/vrooli/ai-provider-go/policy"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
	_ "modernc.org/sqlite"

	catalogH "ai-gateway/handlers/catalog"
	conformanceH "ai-gateway/handlers/conformance"
	gatewayH "ai-gateway/handlers/gateway"
	healthH "ai-gateway/handlers/health"
	inferenceH "ai-gateway/handlers/inference"
	inventoryH "ai-gateway/handlers/inventory"
	measuresH "ai-gateway/handlers/measures"
	privacyconsentH "ai-gateway/handlers/privacyconsent"
	routingH "ai-gateway/handlers/routing"
	"ai-gateway/internal/privacyconsent"
	internalrouting "ai-gateway/internal/routing"
)

func repoRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		if st, err := os.Stat(filepath.Join(dir, "VISION.md")); err == nil && !st.IsDir() {
			return dir
		}
		if st, err := os.Stat(filepath.Join(dir, ".git")); err == nil && st.IsDir() {
			return dir
		}
		next := filepath.Dir(dir)
		if next == dir {
			return ""
		}
	}
}

func loadProfileCatalog(root string) (*gatewaycatalog.ProfileCatalog, error) {
	paths := []string{filepath.Join(root, "scenarios", "ai-gateway", "config", "profile-catalog.json")}
	if root == "" {
		// Packaged runs may not retain the repository marker, but the scenario
		// config is shipped beside the API source tree.
		paths = []string{filepath.Join("config", "profile-catalog.json"), filepath.Join("..", "config", "profile-catalog.json")}
	}
	var lastErr error
	for _, path := range paths {
		profileCatalog, err := gatewaycatalog.LoadProfiles(path)
		if err == nil {
			return profileCatalog, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func loadPrivateAIFeatures(root string) []internalrouting.PrivateFeature {
	path := filepath.Join(root, "packages", "monetization-go", "ai-features.generated.json")
	data, err := os.ReadFile(path)
	if root == "" {
		if bundle, bundleErr := catalog.Embedded(); bundleErr == nil {
			data, err = bundle.Features, nil
		}
	}
	if err != nil {
		log.Printf("private AI feature registry unavailable: %v", err)
		return nil
	}
	var registry struct {
		Version  int `json:"version"`
		Features []struct {
			Scenario string                         `json:"scenario"`
			Feature  internalrouting.PrivateFeature `json:"feature"`
		} `json:"features"`
	}
	if err := json.Unmarshal(data, &registry); err != nil || registry.Version != 1 {
		log.Printf("private AI feature registry is invalid; private feature routing remains denied")
		return nil
	}
	features := make([]internalrouting.PrivateFeature, 0, len(registry.Features))
	for _, entry := range registry.Features {
		entry.Feature.AppKey = strings.TrimSpace(entry.Scenario)
		features = append(features, entry.Feature)
	}
	return features
}

func loadCreditFeatureOwners(root string) monetization.CreditFeatureRegistry {
	path := filepath.Join(root, "packages", "monetization-go", "credit-features.generated.json")
	data, err := os.ReadFile(path)
	if root == "" {
		if bundle, bundleErr := catalog.Embedded(); bundleErr == nil {
			data, err = bundle.CreditOwners, nil
		}
	}
	if err != nil || len(data) == 0 {
		log.Printf("credit feature owner registry unavailable; unowned metered roles remain denied")
		return monetization.CreditFeatureRegistry{}
	}
	registry, err := monetization.ParseCreditFeatureRegistry(data)
	if err != nil {
		log.Printf("credit feature owner registry is invalid; unowned metered roles remain denied: %v", err)
		return monetization.CreditFeatureRegistry{}
	}
	return registry
}

func openGatewayDatabase() *database.RoutedDB {
	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "ai-gateway",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	return db
}

func initializeConsentAndMeasures(db *database.RoutedDB) (*privacyconsent.Store, module.Module) {
	routeEvidenceRepository := internalrouting.NewSQLRepository(db.Primary())
	consentStore, err := privacyconsent.New(db.Primary(), nil)
	if err != nil {
		log.Fatalf("AI privacy consent authority initialization failed: %v", err)
	}
	measuresModule, err := measuresH.Module(routeEvidenceRepository, nil)
	if err != nil {
		log.Fatalf("measures module init failed: %v", err)
	}
	return consentStore, measuresModule
}

func rightsContextSource() internalrouting.RightsContextSource {
	rightsContextPath := filepath.Join(repoRoot(), "scenarios", "ai-gateway", "config", "model-rights-context.json")
	return func(context.Context) (sharedpolicy.RightsContext, error) {
		return sharedpolicy.LoadRightsContextFile(rightsContextPath)
	}
}

func loadInferenceRuntime(db *database.RoutedDB, consentStore *privacyconsent.Store, rightsSource internalrouting.RightsContextSource, resourceRunner providers.CommandRunner) (inference.Repository, rolecatalog.RoleCatalog) {
	// A malformed inference catalog degrades typed inference to a stated
	// reason; it does not take down routing, measures, or conformance. Those
	// domains have no dependency on the inference role catalog, and failing
	// them closed would turn one bad config edit into a full gateway outage.
	catalogPath := filepath.Join(repoRoot(), "scenarios", "ai-gateway", "config", "inference-role-catalog.json")
	var inferenceRepository inference.Repository
	roleCatalog, err := rolecatalog.LoadCatalog(catalogPath)
	bundle, bundleErr := catalog.Embedded()
	if repoRoot() == "" && bundleErr == nil {
		err = json.Unmarshal(bundle.Roles, &roleCatalog)
		if err == nil {
			err = roleCatalog.Validate()
		}
	}
	if err != nil {
		log.Printf("typed inference disabled: %v", err)
		inferenceRepository = inference.UnavailableRepository{Reason: err.Error()}
		roleCatalog = rolecatalog.RoleCatalog{}
	} else {
		adapters := providers.NewDefaultAdapters(resourceRunner)
		if repoRoot() == "" {
			adapters = providers.NewPackagedAdapters(bundle)
		}
		if endpoint := strings.TrimSpace(os.Getenv("LPBS_AI_DOOR_URL")); endpoint != "" {
			adapters = append(adapters, providers.Adapter{Provider: providers.ProviderMetered, Locality: "remote", Door: &providers.HostedDoorClient{BaseURL: endpoint}})
		}
		if resourceRepository, repositoryErr := inference.NewResourceRepositoryWithRightsContextSource(roleCatalog, adapters, rightsSource, inference.NewSQLAttemptLedger(db.Primary())); repositoryErr != nil {
			log.Printf("typed inference disabled: %v", repositoryErr)
			inferenceRepository = inference.UnavailableRepository{Reason: repositoryErr.Error()}
			roleCatalog = rolecatalog.RoleCatalog{}
		} else {
			resourceRepository.SetRemoteConsentVerifier(consentStore)
			inferenceRepository = resourceRepository
		}
	}

	return inferenceRepository, roleCatalog
}

func loadPrivateFeatures(root string) ([]internalrouting.PrivateFeature, monetization.CreditFeatureRegistry) {
	privateAIFeatures := loadPrivateAIFeatures(root)
	owners := loadCreditFeatureOwners(root)
	if owner, ok := owners.OwnerForFeature("agent.session"); ok {
		privateAIFeatures = append(privateAIFeatures, internalrouting.PrivateFeature{
			ID: owner.Feature.ID, AppKey: owner.AppKey, Role: "chat", Profile: "balanced",
			MaxInputBytes: 1 << 20, MaxOutputBytes: 1 << 20, MaxUserTurns: 1,
		})
	}
	return privateAIFeatures, owners
}

func buildGatewayServer(db *database.RoutedDB, consentStore *privacyconsent.Store, measuresModule module.Module, inferenceRepository inference.Repository, roleCatalog rolecatalog.RoleCatalog, privateAIFeatures []internalrouting.PrivateFeature, rightsSource internalrouting.RightsContextSource, lpbsConsumerProvider *internalidentity.LPBSConsumerProvider, resourceRunner providers.CommandRunner) http.Handler {
	routeEvidenceRepository := internalrouting.NewSQLRepository(db.Primary())
	profileCatalog, profileCatalogErr := loadProfileCatalog(repoRoot())
	if profileCatalogErr != nil {
		log.Printf("capability catalog unavailable: profile catalog could not be loaded: %v", profileCatalogErr)
	}
	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "ai-gateway-api", "1.0.0"),
		conformanceH.Module(log.Default(), repoRoot(), conformanceH.Deps{ModelEvalConsumers: routeEvidenceRepository, ModelEvalRuntimeModels: routeEvidenceRepository.ModelEvalRuntimeModels}),
		catalogH.Module(profileCatalog, roleCatalog),
		gatewayH.Module(),
		inferenceH.Module(inferenceH.Deps{Service: inference.NewService(inferenceRepository)}),
		inventoryH.Module(inventoryH.Deps{Runner: resourceRunner, InferenceRoles: roleCatalog.RoleNames()}),
		measuresModule,
		routingH.Module(routingH.Deps{
			DB:                  db.Primary(),
			Service:             internalrouting.NewSQLService(db.Primary(), providers.NewDefaultAdapters(resourceRunner), internalrouting.WithRightsContextSource(rightsSource), internalrouting.WithRemoteConsentVerifier(consentStore)),
			MediaExecutor:       internalrouting.NewResourceOpenRouterMediaExecutor(nil),
			ProviderDecisionHub: internalrouting.NewDefaultProviderDecisionHubClient(),
			PrivatePolicy:       internalrouting.NewResourcePrivatePolicyService(db.Primary(), privateAIFeatures, repoRoot(), internalrouting.NewDefaultProviderDecisionHubClient()),
			CreditAssertions:    lpbsConsumerProvider,
		}),
	)
	return srv.Handler()
}

func buildRootHandler(db *database.RoutedDB, consentStore *privacyconsent.Store, apiHandler http.Handler) http.Handler {
	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.Register(rootMux, db)
	consentHandler := privacyconsentH.New(consentStore)
	rootMux.Handle("/api/v1/ai/privacy-consent", consentHandler)
	rootMux.Handle("/api/v1/ai/privacy-consent/", consentHandler)

	rootMux.Handle("/", apiHandler)

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	return apihttp.TestModeMiddleware(rootMux)
}

func serveGateway(db *database.RoutedDB, handler http.Handler, lpbsConsumerProvider authn.Provider) {
	// AI consent is user-account scoped through LPBS's signed consumer token.
	// Do not trust app request JSON or a selected provider to choose the owner.
	identityConfig := authn.Config{Providers: []authn.Provider{lpbsConsumerProvider}}
	if err := apiserver.Run(apiserver.Config{
		Handler:        handler,
		Authentication: &identityConfig,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   2 * time.Minute,
		Cleanup:        func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "ai-gateway"}) {
		return
	}
	db := openGatewayDatabase()
	consentStore, measuresModule := initializeConsentAndMeasures(db)
	rightsSource := rightsContextSource()
	// One fact cache serves routing, typed inference and inventory, so a
	// resource's policy and inventory answers are read once per lifetime
	// window instead of once per request and per domain.
	resourceRunner := providers.NewFactCache(providers.ExecRunner{}, providers.DefaultFactLifetime)
	// Fill the cache before the first request needs it, then keep facts that
	// callers use from expiring during quiet spells.
	go providers.WarmFacts(context.Background(), providers.NewDefaultAdapters(resourceRunner))
	go resourceRunner.KeepWarm(context.Background())
	inferenceRepository, roleCatalog := loadInferenceRuntime(db, consentStore, rightsSource, resourceRunner)
	lpbsConsumerProvider := internalidentity.NewLPBSConsumerProvider(os.Getenv("LPBS_CONSUMER_TOKEN_ISSUER"))
	privateFeatures, _ := loadPrivateFeatures(repoRoot())
	apiHandler := buildGatewayServer(db, consentStore, measuresModule, inferenceRepository, roleCatalog, privateFeatures, rightsSource, lpbsConsumerProvider, resourceRunner)
	serveGateway(db, buildRootHandler(db, consentStore, apiHandler), lpbsConsumerProvider)
}
