package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"source-ledger/internal/capabilities"
	"source-ledger/internal/facets"
	"source-ledger/internal/files"
	"source-ledger/internal/forest"
	"source-ledger/internal/inference"
	"source-ledger/internal/journal"
	"source-ledger/internal/modules"
	"source-ledger/internal/policy"
	internalrecall "source-ledger/internal/recall"
	"source-ledger/internal/server"
	vectorstore "source-ledger/internal/vector"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/researchhome"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
	searchregister "github.com/vrooli/searchregister-go"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
	_ "modernc.org/sqlite"

	capsH "source-ledger/handlers/capabilities"
	facetsH "source-ledger/handlers/facets"
	filesH "source-ledger/handlers/files"
	forestH "source-ledger/handlers/forest"
	healthH "source-ledger/handlers/health"
	journalH "source-ledger/handlers/journal"
	recallH "source-ledger/handlers/recall"
	rulesH "source-ledger/handlers/rules"
	scopesH "source-ledger/handlers/scopes"
	vectorH "source-ledger/handlers/vectorgenerations"
)

func scenarioStorageRoots() (storage.Paths, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return storage.Paths{}, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("source-ledger")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve source-ledger storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

func main() {
	if preflight.Run(preflight.Config{ScenarioName: "source-ledger"}) {
		return
	}
	dsn, err := storage.SQLiteDSN(storage.SQLiteConfig{Scenario: "source-ledger"})
	if err != nil {
		log.Fatalf("sqlite configuration failed: %v", err)
	}
	db, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: dsn, MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}

	ctx := context.Background()
	if err := journal.EnsureMigrations(ctx, db.Primary()); err != nil {
		log.Fatalf("journal schema migration failed: %v", err)
	}
	if err := forest.EnsureMigrations(ctx, db.Primary()); err != nil {
		log.Fatalf("forest schema migration failed: %v", err)
	}
	if err := policy.EnsureMigrations(ctx, db.Primary()); err != nil {
		log.Fatalf("policy schema migration failed: %v", err)
	}
	if err := facets.EnsureMigrations(ctx, db.Primary()); err != nil {
		log.Fatalf("facet schema migration failed: %v", err)
	}
	if err := database.EnsureSchemas(ctx, db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	generationServing, err := vectorstore.NewSQLiteGenerationServing(db.Primary())
	if err != nil {
		log.Fatalf("Source Ledger generation serving configuration failed: %v", err)
	}
	generationRuntime, err := vectorstore.NewDefaultTrustedSourceLedgerOllamaRuntime(db.Primary())
	if err != nil {
		log.Fatalf("Source Ledger exact-recipe embedding configuration failed: %v", err)
	}
	diskAdmission := vectorstore.UnavailableGenerationDiskAdmission{Reason: "approved packages/capacity dependency is not installed while host disk admission is below the 120 GiB floor"}
	generationBackend, err := vectorstore.NewSQLiteGenerationBackend(db.Primary(), generationRuntime, generationRuntime, diskAdmission, vectorstore.SQLiteGenerationPeakEstimate{}, vectorstore.SQLiteGenerationOptions{})
	if err != nil {
		log.Fatalf("Source Ledger generation lifecycle configuration failed: %v", err)
	}
	log.Printf("Source Ledger vector-generation mutations are fail-closed until the governed shared-capacity disk-job adapter is available")
	if census, err := vectorstore.CensusExistingVectors(ctx, db.Primary(), time.Now().UTC()); err != nil {
		log.Printf("Source Ledger vector census unavailable; historical recipe identity remains unresolved: %v", err)
	} else {
		for _, dataset := range census {
			log.Printf("Source Ledger vector census: dataset=%s identity=%s dimensions=%d vectors=%d missing=%d invalid=%d", dataset.DatasetID, dataset.IdentityState, dataset.Dimensions, dataset.VectorCount, dataset.MissingCount, dataset.InvalidCount)
		}
	}
	facetService := facets.NewService(facets.NewSQLiteRepository(db.Primary()))
	if err := facetService.Seed(ctx); err != nil {
		log.Fatalf("seed facet definitions: %v", err)
	}
	if err := facetService.SeedExamples(ctx); err != nil {
		log.Fatalf("seed facet examples: %v", err)
	}
	primaryFileRoots, err := scenarioStorageRoots()
	if err != nil {
		log.Fatalf("file storage configuration failed: %v", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)

	gatewayURL, err := discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
	if err != nil {
		log.Fatalf("resolve ai-gateway endpoint: %v", err)
	}
	policyConfig, err := policy.LoadDefaults(policy.DefaultPolicyPath())
	if err != nil {
		log.Fatalf("source-ledger policy configuration failed: %v", err)
	}
	registry := policy.NewRegistry(db.Primary(), policyConfig)
	if err := registry.Ensure(ctx, policyConfig); err != nil {
		log.Fatalf("scope registry initialization failed: %v", err)
	}
	repoRoot, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		log.Fatalf("resolve repository root for search federation: %v", err)
	}
	searchJSONPath := filepath.Join(repoRoot, "scenarios", "source-ledger", ".vrooli", "search.json")
	go searchregister.Register(ctx, searchregister.Config{
		ScenarioID:     "source-ledger",
		SearchFilePath: searchJSONPath,
		Logger:         log.Default(),
	})
	resolver := policy.NewRequestResolver(registry, func(ctx context.Context) ([]policy.Facet, error) {
		definitions, err := facetService.List(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]policy.Facet, 0, len(definitions))
		for _, definition := range definitions {
			out = append(out, policy.Facet{ID: definition.ID, Label: definition.Label, Guidance: definition.ClassificationGuidance, Examples: definition.ClassificationExamples})
		}
		return out, nil
	})
	definitions, err := facetService.List(ctx)
	if err != nil {
		log.Fatalf("list facet policies: %v", err)
	}
	facetBudgets := make(map[string]int, len(definitions))
	for _, definition := range definitions {
		facetBudgets[definition.ID] = definition.ResidentBudget
	}
	recallConfig := internalrecall.Config{FrontierTarget: policyConfig.FrontierTarget, WakeBudget: policyConfig.WakeBudget, WakeBudgetChars: policyConfig.WakeBudgetChars, MaxEntryLines: policyConfig.MaxEntryLines, MaxEntryChars: policyConfig.MaxEntryChars, FacetBudgets: facetBudgets}
	gatewayClient := inference.NewGatewayClient(routingconnect.NewRoutingServiceClient(http.DefaultClient, gatewayURL), func(ctx context.Context) ([]inference.VocabularyEntry, error) {
		definitions, err := resolver.Vocabulary(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]inference.VocabularyEntry, 0, len(definitions))
		for _, definition := range definitions {
			out = append(out, inference.VocabularyEntry{ID: definition.ID, Label: definition.Label, Guidance: definition.Guidance, Examples: definition.Examples})
		}
		return out, nil
	})
	gatewayClient.WithSummaryMaxOutputTokens(policyConfig.SummaryMaxOutputTokens)
	forestService := forestH.NewService(db, gatewayClient, recallConfig.FrontierTarget, registry, generationServing)
	gatewayClient.WithBatchEmbedding(inferenceconnect.NewInferenceServiceClient(http.DefaultClient, gatewayURL))
	fileService := openFileCorpus(ctx, inference.Embedder{Client: gatewayClient})
	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "source-ledger-api", "1.0.0"),
		capsH.Module(capabilities.NewRegistry()),
		journalH.Module(db, gatewayClient, facetService, log.Default()),
		facetsH.Module(db, log.Default(), registry),
		filesH.Module(fileService),
		forestH.Module(forestService, log.Default()),
		recallH.Module(db, gatewayClient, recallConfig, registry, log.Default(), generationServing, generationRuntime),
		rulesH.ModuleWithRegistry(db, log.Default(), registry, gatewayClient),
		scopesH.Module(db, registry, log.Default(), internalrecall.NewSQLiteSource(db.Primary(), generationServing)),
		vectorH.Module(generationBackend, log.Default()),
	)

	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)
	rootMux.Handle("/", srv.Handler())
	handler := apihttp.TestModeMiddleware(rootMux)
	authConfig, err := authn.FromEnvironment(os.Getenv)
	if err != nil {
		log.Fatalf("Source Ledger authentication configuration failed: %v", err)
	}
	// Compaction and rebuild are server-owned passes that hold one unary RPC
	// open for minutes (each cluster summary is a bounded model call). The
	// shared server's 30 s WriteTimeout closed that connection mid-pass and
	// every consumer saw "unexpected EOF" while the pass kept running; the
	// write bound must cover vrooli-memory's 30 min CompactTimeout.
	if err := apiserver.Run(apiserver.Config{Handler: handler, Authentication: &authConfig, WriteTimeout: 35 * time.Minute, Cleanup: func(context.Context) error { return db.Close() }}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// openFileCorpus opens the research-home file corpus. Its index is a cache in
// the scenario cache directory, separate from the journal authority; deleting
// it costs one rebuild. A failure here is fatal at boot rather than a silently
// missing corpus.
func openFileCorpus(ctx context.Context, embedder files.Embedder) *files.Service {
	home, err := researchhome.Dir(os.Getenv)
	if err != nil {
		log.Fatalf("resolve research home for the file corpus: %v", err)
	}
	// The index is a rebuildable cache, so it lives in the cache storage class.
	// The storage package owns the file path, parent directory, and pragmas.
	dsn, err := storage.SQLiteDSN(storage.SQLiteConfig{Scenario: "source-ledger", Filename: "files-index.db", Class: storage.ClassCache})
	if err != nil {
		log.Fatalf("file corpus index configuration: %v", err)
	}
	indexDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatalf("open file corpus index: %v", err)
	}
	indexDB.SetMaxOpenConns(1)
	service, err := files.NewService(ctx, indexDB, map[string]string{files.DefaultCorpus: home}, embedder, log.Default())
	if err != nil {
		log.Fatalf("file corpus index: %v", err)
	}
	go service.Run(ctx, 5*time.Minute)
	return service
}
