package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"search-hub/internal/eval"
	"search-hub/internal/experimentarms"
	"search-hub/internal/modules"
	internalregistry "search-hub/internal/registry"
	"search-hub/internal/server"

	ollamapolicy "github.com/vrooli/ai-go/ollama/policy"
	aisearch "github.com/vrooli/ai-go/search"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	apihealth "github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	corestorage "github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
	capacitybroker "github.com/vrooli/vrooli/packages/capacity"
	_ "modernc.org/sqlite"

	evalH "search-hub/handlers/eval"
	experimentH "search-hub/handlers/experimentarms"
	healthH "search-hub/handlers/health"
	metricsH "search-hub/handlers/metrics"
	registryH "search-hub/handlers/registry"
	routingH "search-hub/handlers/routing"
	validationH "search-hub/handlers/validation"
)

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func resolveQdrantStoragePath(repoRoot string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve process home: %w", err)
	}
	if strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("process home directory is empty")
	}
	seams := corestorage.PlatformSeams{UserHomeDir: func() (string, error) { return home, nil }}
	inventory, err := corestorage.LoadOwnerInventory(corestorage.InventoryOptions{RepoRoot: repoRoot, Platform: corestorage.HostPlatform(), PlatformSeams: seams})
	if err != nil {
		return "", fmt.Errorf("read storage owner inventory: %w", err)
	}
	for _, owner := range inventory.Owners {
		if owner.Kind != corestorage.OwnerResource || owner.ID != "qdrant" {
			continue
		}
		for _, entry := range owner.StorageEntries {
			if entry.Name != "storage" {
				continue
			}
			path, err := corestorage.ResolveOwnerStoragePath(repoRoot, owner, entry, corestorage.HostPlatform(), seams)
			if err != nil {
				return "", fmt.Errorf("resolve Qdrant storage entry: %w", err)
			}
			return path, nil
		}
		return "", fmt.Errorf("Qdrant storage owner has no storage entry")
	}
	return "", fmt.Errorf("Qdrant resource storage owner was not found")
}

type serviceResource struct {
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

// requiredResourceChecks derives readiness dependencies from the scenario
// manifest. Search Hub observes resource state but does not own remediation;
// adding a required resource therefore changes one declarative file, not the
// health handler. Resource-specific health paths are tied to resource type,
// never to a provider scenario id.
func requiredResourceChecks(repoRoot string) []apihealth.Checker {
	raw, err := os.ReadFile(filepath.Join(repoRoot, "scenarios", "search-hub", ".vrooli", "service.json"))
	if err != nil {
		log.Printf("health: cannot read service resource manifest: %v", err)
		return nil
	}
	var manifest struct {
		Dependencies struct {
			Resources map[string]serviceResource `json:"resources"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		log.Printf("health: cannot parse service resource manifest: %v", err)
		return nil
	}
	checks := make([]apihealth.Checker, 0)
	for name, resource := range manifest.Dependencies.Resources {
		if !resource.Required {
			continue
		}
		path := "/healthz"
		if resource.Type == "ollama" {
			path = "/api/tags"
		}
		envKey := strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_URL"
		fallback := "http://127.0.0.1:6333"
		if resource.Type == "ollama" {
			fallback = "http://127.0.0.1:11434"
		}
		checks = append(checks, apihealth.HTTP(name, envOrDefault(envKey, fallback)+path))
	}
	return checks
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "search-hub"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "search-hub",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil {
		log.Fatalf("repo root resolution failed: %v", err)
	}

	if err := metricsH.Migrate(context.Background(), db); err != nil {
		log.Fatalf("metrics schema migration failed: %v", err)
	}
	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	if err := internalregistry.NewSQLiteStore(db, schedule.System()).MigrateLegacyControlTokens(context.Background()); err != nil {
		log.Fatalf("provider control-token migration failed; credential authority must be restored before Search Hub can serve providers: %v", err)
	}
	searchNamespace, err := corestorage.ResolveNamespace(corestorage.NamespaceConfig{FallbackScenario: "search-hub"})
	if err != nil {
		log.Fatalf("Search Hub storage namespace resolution failed: %v", err)
	}
	armRegistry, err := experimentarms.NewForNamespace(db, searchNamespace)
	if err != nil {
		log.Fatalf("embedding experiment registry construction failed: %v", err)
	}
	// Search Hub owns each embedding-fleet operation. Reserve its bounded Qdrant
	// storage peak through the shared capacity disk-job ledger and persist the
	// operation/arm/target binding beside the fleet catalog. The model role and
	// digest-bound rights identity come from resource-ollama policy; the encoder
	// and peak estimator are owner implementations, never request fields.
	var fleetAdmission aisearch.FleetAdmissionGate
	fleetEncoder, encoderErr := experimentarms.NewTrustedOllamaFleetDescriptionEncoder(context.Background(), ollamapolicy.Resolver{})
	if encoderErr != nil {
		log.Printf("trusted Ollama fleet encoder remains unavailable: %v", encoderErr)
	}
	capacityBroker, capacityErr := capacitybroker.NewBroker(context.Background(), "")
	if capacityErr != nil {
		log.Printf("Qdrant fleet operation admission remains unavailable: open shared capacity broker: %v", capacityErr)
	} else {
		defer capacityBroker.Close()
		storagePath, pathErr := resolveQdrantStoragePath(repoRoot)
		if pathErr != nil {
			log.Printf("Qdrant fleet operation admission remains unavailable: resolve owner storage path: %v", pathErr)
		} else {
			gate, gateErr := experimentarms.NewDiskJobFleetAdmissionGate(db.Primary(), capacityBroker, storagePath, searchNamespace.Root()+".qdrant-fleet")
			if gateErr == nil {
				gateErr = gate.Initialize(context.Background())
			}
			if gateErr != nil {
				log.Printf("Qdrant fleet operation admission remains unavailable: initialize owner gate: %v", gateErr)
			} else {
				fleetAdmission = gate
			}
		}
	}
	fleetStage, fleetStageErr := armRegistry.NewFleetGenerationStage(
		aisearch.QdrantFleetOptions{BaseURL: envOrDefault("QDRANT_URL", aisearch.DefaultQdrantURL), Admission: fleetAdmission},
		fleetEncoder,
		experimentarms.ProviderDescriptionFleetPeakEstimator{},
	)
	if fleetStageErr != nil {
		log.Printf("embedding generation stage remains unavailable until verified runtime and trusted peak estimator are registered: %v", fleetStageErr)
	}
	var fleetSemanticStore aisearch.FleetSemanticStore
	constructedSemanticStore, fleetSemanticErr := armRegistry.NewProviderDescriptionFleetSemanticStore(
		aisearch.QdrantFleetOptions{BaseURL: envOrDefault("QDRANT_URL", aisearch.DefaultQdrantURL)},
		nil,
		aisearch.FleetSemanticOptions{RuntimeRole: "search-hub-embedding"},
	)
	if fleetSemanticErr != nil {
		log.Printf("steady-state embedding search/upsert/delete remains unavailable until verified recipe encoder, bounded peak estimator, and semantic operation admission are registered: %v", fleetSemanticErr)
	} else {
		fleetSemanticStore = constructedSemanticStore
	}

	// Register the SHRINKING set of starter eval suites search-hub still ships
	// (idempotent upsert by suite_id) for the one provider that has not yet
	// adopted corpus self-registration (swarm-manager.records). Adopted providers
	// (cli-health, knowledge-observatory, ui-health) self-register both their
	// descriptor AND their tests corpus from their own .vrooli/search.json at
	// boot, so they need no seed here; a suite only references a provider_id and
	// the runner resolves it at run time. When swarm-manager adopts, this call and
	// the eval/seeds mechanism are deleted. See internal/eval/seeds.go.
	if err := eval.RegisterSeeds(context.Background(), eval.NewSQLiteStore(db, schedule.System())); err != nil {
		log.Fatalf("eval seed registration failed: %v", err)
	}

	// The metrics domain owns the query_telemetry store. Its Recorder bridge is
	// injected into the routing module so each federated query records telemetry
	// (Phase 7), while the routing handler stays free of any metrics-store import.
	telemetryRecorder := metricsH.Recorder(db, schedule.System(), log.Default())
	// The fleet-backed semantic caller is composed only when both its owner
	// store and a source-verified caller context resolver are available. The
	// latter remains absent until rights and quality evidence has a current
	// authenticated owner source; nil keeps routing on lexical fallback without
	// constructing a legacy generic embedder.
	router := routingH.NewRouterWithFleetSemanticStore(db, schedule.System(), log.Default(), telemetryRecorder, fleetSemanticStore, nil)
	routingH.StartRecoveryProbes(router, log.Default())
	resourceChecks := requiredResourceChecks(repoRoot)
	resourceChecks = append(resourceChecks, apihealth.Func("federation", func(ctx context.Context) error {
		share, breached, err := router.CircuitOpenQuorum(ctx)
		if err != nil {
			return err
		}
		if breached {
			return fmt.Errorf("circuit-open share %.1f%% meets federation quorum %.1f%%", share*100, routingH.CircuitOpenQuorumThreshold*100)
		}
		return nil
	}))

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "search-hub-api", "1.0.0", resourceChecks...),
		metricsH.Module(db, schedule.System(), log.Default()),
		registryH.Module(db, schedule.System(), log.Default(), repoRoot),
		experimentH.Module(armRegistry, fleetStage),
		routingH.ModuleWithRouter(router, log.Default()),
		evalH.ModuleWithRoutability(db, schedule.System(), log.Default(), router),
		validationH.Module(log.Default(), repoRoot, db, schedule.System(), router),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.Register(rootMux, db)
	rootMux.Handle("/", srv.Handler())
	// Search Hub's embedding-arm lifecycle writes require a verified principal
	// and operation-scoped authority. Keep API-core's passive middleware enabled
	// even when no interactive provider is configured so Agent Manager identity
	// and separately approved operation grants reach the domain handler.
	authConfig, err := authn.FromEnvironment(os.Getenv)
	if err != nil {
		log.Fatalf("authentication configuration failed: %v", err)
	}

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler:        handler,
		Authentication: &authConfig,
		// EvalService.RunSuite is a synchronous, long-running RPC: it fans out one
		// HTTP call per golden case to the target provider, and a provider whose
		// active reranker leg is an LLM (seconds per query) can push a full suite
		// well past the 30s default. Give the server enough headroom to finish a
		// suite run rather than resetting the connection mid-fan-out (which would
		// discard the immutable run record the harness exists to produce).
		WriteTimeout: 15 * time.Minute,
		Cleanup:      func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
