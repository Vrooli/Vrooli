package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"source-ledger/internal/capabilities"
	"source-ledger/internal/clock"
	"source-ledger/internal/facets"
	"source-ledger/internal/federation"
	"source-ledger/internal/forest"
	"source-ledger/internal/inference"
	"source-ledger/internal/journal"
	"source-ledger/internal/modules"
	"source-ledger/internal/policy"
	internalrecall "source-ledger/internal/recall"
	"source-ledger/internal/server"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
	searchregister "github.com/vrooli/searchregister-go"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
	_ "modernc.org/sqlite"

	capsH "source-ledger/handlers/capabilities"
	facetsH "source-ledger/handlers/facets"
	forestH "source-ledger/handlers/forest"
	healthH "source-ledger/handlers/health"
	journalH "source-ledger/handlers/journal"
	recallH "source-ledger/handlers/recall"
	rulesH "source-ledger/handlers/rules"
	scopesH "source-ledger/handlers/scopes"
)

func sqliteDSN() (string, error) {
	if path := strings.TrimSpace(os.Getenv("SQLITE_PATH")); path != "" {
		return sqliteFileDSN(path)
	}
	if path := strings.TrimSpace(os.Getenv("SQLITE_DB")); path != "" {
		return sqliteFileDSN(path)
	}
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("source-ledger")
	if err != nil {
		return "", fmt.Errorf("resolve source-ledger storage namespace: %w", err)
	}
	path, err := resolver.Path(storage.Options{ScenarioID: scenarioID}, storage.ClassData, "source-ledger.db")
	if err != nil {
		return "", fmt.Errorf("resolve source-ledger db path: %w", err)
	}
	return sqliteFileDSN(path)
}

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

func sqliteFileDSN(path string) (string, error) {
	if strings.HasPrefix(path, "file:") {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("prepare sqlite directory: %w", err)
	}
	return fmt.Sprintf("file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)", path), nil
}

func main() {
	if preflight.Run(preflight.Config{ScenarioName: "source-ledger"}) {
		return
	}

	dsn, err := sqliteDSN()
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
	policyConfig, err := policy.Resolve(os.LookupEnv)
	if err != nil {
		log.Fatalf("source-ledger policy configuration failed: %v", err)
	}
	registry := policy.NewRegistry(db.Primary())
	if err := registry.Ensure(ctx, policyConfig); err != nil {
		log.Fatalf("scope registry initialization failed: %v", err)
	}
	repoRoot, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		log.Fatalf("resolve repository root for search federation: %v", err)
	}
	searchJSONPath := filepath.Join(repoRoot, "scenarios", "source-ledger", ".vrooli", "search.json")
	registerScopeProvider := func(scope string) error {
		if err := federation.AppendScopeProvider(searchJSONPath, scope); err != nil {
			return err
		}
		// A scope created after boot must become visible in Search Hub as well as
		// in the durable descriptor. Registration is best-effort and bounded by
		// searchregister-go; local ledger writes remain authoritative.
		go searchregister.Register(context.Background(), searchregister.Config{
			ScenarioID:     "source-ledger",
			SearchFilePath: searchJSONPath,
			Logger:         log.Default(),
		})
		return nil
	}
	// The policy registry is the source of truth for scopes. Materialize every
	// already-known scope before self-registration reads search.json, then use
	// the same seam for scopes created through the API later in this process.
	scopeDefinitions, err := registry.List(ctx)
	if err != nil {
		log.Fatalf("list scopes for search federation: %v", err)
	}
	for _, definition := range scopeDefinitions {
		if err := registerScopeProvider(definition.ID); err != nil {
			log.Fatalf("materialize search provider for scope %q: %v", definition.ID, err)
		}
	}
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
	recallConfig := internalrecall.Config{FrontierTarget: policyConfig.FrontierTarget, WakeBudget: policyConfig.WakeBudget, MaxEntryLines: policyConfig.MaxEntryLines, FacetBudgets: facetBudgets}
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
	forestService := forestH.NewService(db, gatewayClient, recallConfig.FrontierTarget, registry)

	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: log.Default()},
		healthH.Module(db, "source-ledger-api", "1.0.0"),
		capsH.Module(capabilities.NewRegistry()),
		journalH.Module(db, gatewayClient, facetService, log.Default()),
		facetsH.Module(db, log.Default()),
		forestH.Module(forestService, log.Default()),
		recallH.Module(db, gatewayClient, recallConfig, registry, log.Default()),
		rulesH.Module(db, log.Default(), gatewayClient),
		scopesH.Module(db, registry, registerScopeProvider, log.Default()),
	)

	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)
	rootMux.Handle("/", srv.Handler())
	handler := apihttp.TestModeMiddleware(rootMux)
	if err := apiserver.Run(apiserver.Config{Handler: handler, Cleanup: func(context.Context) error { return db.Close() }}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
