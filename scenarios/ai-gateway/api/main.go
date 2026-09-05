package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	inference "ai-gateway/internal/inference"
	"ai-gateway/internal/modules"
	"ai-gateway/internal/providers"
	"ai-gateway/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	_ "modernc.org/sqlite"

	conformanceH "ai-gateway/handlers/conformance"
	gatewayH "ai-gateway/handlers/gateway"
	healthH "ai-gateway/handlers/health"
	inferenceH "ai-gateway/handlers/inference"
	inventoryH "ai-gateway/handlers/inventory"
	measuresH "ai-gateway/handlers/measures"
	routingH "ai-gateway/handlers/routing"
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

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "ai-gateway"}) {
		return
	}

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

	measuresModule, err := measuresH.Module(internalrouting.NewSQLRepository(db.Primary()), nil)
	if err != nil {
		log.Fatalf("measures module init failed: %v", err)
	}

	// A malformed inference catalog degrades typed inference to a stated
	// reason; it does not take down routing, measures, or conformance. Those
	// domains have no dependency on the inference role catalog, and failing
	// them closed would turn one bad config edit into a full gateway outage.
	catalogPath := filepath.Join(repoRoot(), "scenarios", "ai-gateway", "config", "inference-role-catalog.json")
	var inferenceRepository inference.Repository
	roleCatalog, err := inference.LoadCatalog(catalogPath)
	if err != nil {
		log.Printf("typed inference disabled: %v", err)
		inferenceRepository = inference.UnavailableRepository{Reason: err.Error()}
		roleCatalog = inference.RoleCatalog{}
	} else {
		adapters := providers.NewDefaultAdapters(nil)
		if endpoint := strings.TrimSpace(os.Getenv("LPBS_METERED_INFERENCE_URL")); endpoint != "" {
			var resolveAccessToken func(context.Context, string) (string, error)
			if authority, authorityErr := credentialauthority.Default(); authorityErr != nil {
				log.Printf("subscription-backed inference shared credential unavailable: %v", authorityErr)
			} else if client, clientErr := credentialclient.NewClient(credentialclient.ClientOptions{
				Authority: authority,
				Descriptors: func() ([]credentialclient.CredentialRef, error) {
					return credentialclient.DiscoverDescriptors(repoRoot())
				},
			}); clientErr != nil {
				log.Printf("subscription-backed inference credential client unavailable: %v", clientErr)
			} else {
				resolver := &credentialclient.ConsumerSessionResolver{Credentials: client}
				resolveAccessToken = func(ctx context.Context, baseURL string) (string, error) {
					access, err := resolver.ResolveAt(ctx, baseURL)
					if err != nil {
						return "", err
					}
					return "Bearer " + access.AccessToken, nil
				}
			}
			adapters = append(adapters, providers.Adapter{Provider: providers.ProviderMetered, Locality: "remote", Metered: providers.NewMeteredClient(providers.MeteredClientOptions{
				BaseURL: endpoint, ResolveAccessToken: resolveAccessToken,
			})})
		}
		if resourceRepository, repositoryErr := inference.NewResourceRepository(roleCatalog, adapters); repositoryErr != nil {
			log.Printf("typed inference disabled: %v", repositoryErr)
			inferenceRepository = inference.UnavailableRepository{Reason: repositoryErr.Error()}
			roleCatalog = inference.RoleCatalog{}
		} else {
			inferenceRepository = resourceRepository
		}
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "ai-gateway-api", "1.0.0"),
		conformanceH.Module(log.Default(), repoRoot()),
		gatewayH.Module(),
		inferenceH.Module(inferenceH.Deps{Service: inference.NewService(inferenceRepository)}),
		inventoryH.Module(inventoryH.Deps{InferenceRoles: roleCatalog.RoleNames()}),
		measuresModule,
		routingH.Module(routingH.Deps{
			DB:            db.Primary(),
			MediaExecutor: internalrouting.NewResourceOpenRouterMediaExecutor(nil),
		}),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.Register(rootMux, db)

	rootMux.Handle("/", srv.Handler())

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 2 * time.Minute,
		Cleanup:      func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
