package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"brand-manager/internal/modules"
	"brand-manager/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	applyH "brand-manager/handlers/apply"
	assetsH "brand-manager/handlers/assets"
	assignmentsH "brand-manager/handlers/assignments"
	brandsH "brand-manager/handlers/brands"
	designH "brand-manager/handlers/design"
	discoveryH "brand-manager/handlers/discovery"
	generationH "brand-manager/handlers/generation"
	healthH "brand-manager/handlers/health"
	validationH "brand-manager/handlers/validation"
)

// assetsBaseDir resolves the root directory under which the assets domain
// stores uploaded brand asset files (one subdirectory per brand). Resolution
// mirrors sqliteDSN so the asset tree lands beside the database in the same
// variant-aware namespace (shadow-safe with zero per-scenario work):
//
//  1. ASSETS_PATH env — the canonical override.
//  2. storage.NewResolver(ProfileAuto) — ClassData/assets, the
//     filesystem-safe-by-default location.
func assetsBaseDir() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("ASSETS_PATH")); dir != "" {
		return dir, nil
	}
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("brand-manager")
	if err != nil {
		return "", fmt.Errorf("resolve brand-manager storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		"assets",
	)
	if err != nil {
		return "", fmt.Errorf("resolve brand-manager assets dir: %w", err)
	}
	return path, nil
}

// scenariosBaseDir resolves the directory that contains scenario source trees
// the apply domain writes brand files into (one subdirectory per scenario).
// Resolution order:
//
//  1. SCENARIOS_PATH env — the canonical override.
//  2. SCENARIOS_DIR env — alias accepted for symmetry with the old REST surface.
//  3. "scenarios" relative to the working directory — the repo's scenarios root
//     when the control plane runs apply from the repository root.
//
// Apply guards every write to stay within the resolved target scenario's
// directory (see internal/apply/workspace.go), so a missing or misconfigured
// root simply yields "scenario not found" rather than an unsafe write.
func scenariosBaseDir() string {
	for _, key := range []string{"SCENARIOS_PATH", "SCENARIOS_DIR"} {
		if dir := strings.TrimSpace(os.Getenv(key)); dir != "" {
			return dir
		}
	}
	return "scenarios"
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "brand-manager"}) {
		return
	}

	assetsDir, err := assetsBaseDir()
	if err != nil {
		log.Fatalf("assets configuration failed: %v", err)
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "brand-manager",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "brand-manager-api", "1.0.0"),
		assetsH.Module(db, schedule.System(), log.Default(), assetsDir),
		assignmentsH.Module(db, schedule.System(), log.Default()),
		brandsH.Module(db, schedule.System(), log.Default()),
		// Generation owns no table; it composes the brands + assets domains
		// behind two adapters and writes generated images into the same assets
		// tree (hence assetsDir). The provider chain is built from the
		// environment (OLLAMA_ROLE, OPENROUTER_API_KEY, …).
		generationH.Module(db, schedule.System(), log.Default(), assetsDir),
		// Apply owns no table; it composes the brands + assets + assignments
		// domains and writes brand files into a target scenario's source tree
		// (scenariosDir) using the same assets blob root (assetsDir) for image
		// bytes.
		applyH.Module(db, schedule.System(), log.Default(), scenariosBaseDir(), assetsDir),
		// Discovery owns no table; it scans a scenario's source tree
		// (scenariosDir) for branding state and imports the inferred draft as a
		// new brand through the brands domain.
		discoveryH.Module(db, schedule.System(), log.Default(), scenariosBaseDir()),
		// Design owns no table; it composes the brands domain behind one adapter
		// and renders a brand into a canonical DESIGN.md document (read-only).
		designH.Module(db, schedule.System(), log.Default()),
		// Branding validation: the served ScenarioValidationService test-genie's
		// `branding` delegated phase calls. brand-manager both authors and
		// validates branding, so the provider lives in this one scenario.
		validationH.Module(),
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
		Handler: handler,
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
