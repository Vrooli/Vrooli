package handlers_test

import (
	"fmt"
	"sync/atomic"
	"testing"

	"brand-manager/config"
	"brand-manager/handlers"
	"brand-manager/repository/mocks"

	"github.com/gorilla/mux"
)

// setupRouterWith creates a router with routes from the given handler.
func setupRouterWith(h *handlers.Handlers) *mux.Router {
	router := mux.NewRouter()
	h.RegisterRoutes(router)
	return router
}

// setupMockServerWithConfig creates a handler stack with custom config, returning all repos.
func setupMockServerWithConfigAndRepos(t *testing.T, cfg config.Config) (*handlers.Handlers, *mux.Router, *mocks.BrandRepository, *mocks.VersionRepository, *mocks.AssignmentRepository, *mocks.AssetRepository) {
	t.Helper()
	brandRepo := mocks.NewBrandRepository()
	versionRepo := mocks.NewVersionRepository()
	assignRepo := mocks.NewAssignmentRepository()
	assetRepo := mocks.NewAssetRepository()

	var counter atomic.Int64
	h := handlers.New(brandRepo, versionRepo, assignRepo).
		WithAssets(assetRepo).
		WithConfig(cfg).
		WithIDFunc(func() string {
			return fmt.Sprintf("id-%d", counter.Add(1))
		})

	router := mux.NewRouter()
	h.RegisterRoutes(router)
	return h, router, brandRepo, versionRepo, assignRepo, assetRepo
}

// setupMockServerWithConfig is a convenience wrapper that omits the asset repo return.
func setupMockServerWithConfig(t *testing.T, cfg config.Config) (*handlers.Handlers, *mux.Router, *mocks.BrandRepository, *mocks.VersionRepository, *mocks.AssignmentRepository) {
	t.Helper()
	h, router, brandRepo, versionRepo, assignRepo, _ := setupMockServerWithConfigAndRepos(t, cfg)
	return h, router, brandRepo, versionRepo, assignRepo
}

// testConfig returns a config.Default() with ScenariosDir and AssetBasePath set
// to temp directories. Most handler tests need only these two overrides.
func testConfig(t *testing.T) config.Config {
	t.Helper()
	cfg := config.Default()
	cfg.ScenariosDir = t.TempDir()
	cfg.AssetBasePath = t.TempDir()
	return cfg
}

// testConfigWithScenarioParent returns a config whose ScenariosDir is set to
// parentDir (the parent of an already-created scenario temp dir).
func testConfigWithScenarioParent(t *testing.T, parentDir string) config.Config {
	t.Helper()
	cfg := config.Default()
	cfg.ScenariosDir = parentDir
	cfg.AssetBasePath = t.TempDir()
	return cfg
}
