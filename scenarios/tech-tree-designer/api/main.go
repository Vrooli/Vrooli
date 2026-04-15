package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
)

const (
	treeIDQueryParam   = "tree_id"
	treeSlugQueryParam = "tree_slug"
	treeIDHeader       = "X-Tech-Tree-Id"
	treeSlugHeader     = "X-Tech-Tree-Slug"
)

var slugCleaner = regexp.MustCompile(`[^a-z0-9-]+`)

func scanTechTree(row *sql.Row) (*TechTree, error) {
	var tree TechTree
	var parentID sql.NullString
	err := row.Scan(
		&tree.ID,
		&tree.Slug,
		&tree.Name,
		&tree.Description,
		&tree.Version,
		&tree.TreeType,
		&tree.Status,
		&tree.IsActive,
		&parentID,
		&tree.CreatedAt,
		&tree.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		tree.ParentTree = &parentID.String
	} else {
		tree.ParentTree = nil
	}
	return &tree, nil
}

func fetchTechTreeByID(ctx context.Context, id string) (*TechTree, error) {
	if db == nil {
		return nil, errors.New("database connection is not initialized")
	}
	return scanTechTree(db.QueryRowContext(ctx, `
		SELECT id, slug, name, description, version, tree_type, status, is_active, parent_tree_id, created_at, updated_at
		FROM tech_trees
		WHERE id = $1
	`, id))
}

func fetchTechTreeBySlug(ctx context.Context, slug string) (*TechTree, error) {
	if db == nil {
		return nil, errors.New("database connection is not initialized")
	}
	return scanTechTree(db.QueryRowContext(ctx, `
		SELECT id, slug, name, description, version, tree_type, status, is_active, parent_tree_id, created_at, updated_at
		FROM tech_trees
		WHERE slug = $1
	`, slug))
}

func fetchDefaultTechTree(ctx context.Context) (*TechTree, error) {
	if db == nil {
		return nil, errors.New("database connection is not initialized")
	}
	return scanTechTree(db.QueryRowContext(ctx, `
		SELECT id, slug, name, description, version, tree_type, status, is_active, parent_tree_id, created_at, updated_at
		FROM tech_trees
		WHERE tree_type = 'official' AND status = 'active'
		ORDER BY updated_at DESC
		LIMIT 1
	`))
}

func resolveTreeContext(c *gin.Context) (*TechTree, error) {
	ctx := c.Request.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	if treeID := strings.TrimSpace(c.Query(treeIDQueryParam)); treeID != "" {
		return fetchTechTreeByID(ctx, treeID)
	}

	if treeSlug := strings.TrimSpace(c.Query(treeSlugQueryParam)); treeSlug != "" {
		return fetchTechTreeBySlug(ctx, treeSlug)
	}

	if headerID := strings.TrimSpace(c.GetHeader(treeIDHeader)); headerID != "" {
		return fetchTechTreeByID(ctx, headerID)
	}

	if headerSlug := strings.TrimSpace(c.GetHeader(treeSlugHeader)); headerSlug != "" {
		return fetchTechTreeBySlug(ctx, headerSlug)
	}

	return fetchDefaultTechTree(ctx)
}

func normalizeSlug(value string) string {
	slug := strings.ToLower(strings.TrimSpace(value))
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = slugCleaner.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = fmt.Sprintf("tree-%d", time.Now().Unix())
	}
	return slug
}

func resolveRepoRoot() (string, error) {
	return repocontract.ResolveRepoRoot()
}

func computeNextStageOrder(ctx context.Context, sectorID string) (int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var nextOrder int
	err := db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(stage_order), 0) + 1
		FROM progression_stages
		WHERE sector_id = $1
	`, sectorID).Scan(&nextOrder)
	if err != nil {
		return 0, err
	}
	if nextOrder <= 0 {
		nextOrder = 1
	}
	return nextOrder, nil
}

func fetchTreeStats(ctx context.Context, treeID string) (int, int, int, error) {
	var sectorCount, stageCount, mappingCount int
	err := db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(1) FROM sectors WHERE tree_id = $1) AS sector_count,
			(SELECT COUNT(1)
			 FROM progression_stages ps
			 JOIN sectors s ON ps.sector_id = s.id
			 WHERE s.tree_id = $1) AS stage_count,
			(SELECT COUNT(1)
			 FROM scenario_mappings sm
			 JOIN progression_stages ps ON sm.stage_id = ps.id
			 JOIN sectors s ON ps.sector_id = s.id
			 WHERE s.tree_id = $1) AS mapping_count
	`, treeID).Scan(&sectorCount, &stageCount, &mappingCount)
	if err != nil {
		return 0, 0, 0, err
	}
	return sectorCount, stageCount, mappingCount, nil
}

func writeTreeResponse(c *gin.Context, tree *TechTree, statusCode int) {
	ctx := c.Request.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	sectorCount, stageCount, mappingCount, err := fetchTreeStats(ctx, tree.ID)
	response := gin.H{"tree": tree}
	if err == nil {
		response["stats"] = gin.H{
			"sectors":           sectorCount,
			"stages":            stageCount,
			"scenario_mappings": mappingCount,
		}
	}
	c.JSON(statusCode, response)
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "tech-tree-designer",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Connect to database with automatic retry and backoff.
	// Reads POSTGRES_* environment variables set by the lifecycle system.
	var err error
	db, err = database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Initialize service layer
	treeService = NewTreeService(db)
	sectorService = NewSectorService(db)
	stageService = NewStageService(db)
	graphService = NewGraphService(db)
	graphQueryService = NewGraphQueryService(db)

	repoRoot, rootErr := resolveRepoRoot()
	if rootErr != nil {
		log.Printf("WARNING: scenario catalog disabled (repo root unresolved): %v", rootErr)
	} else {
		log.Printf("INFO: Resolved repo root: %s", repoRoot)
		visibilityPath, pathErr := resolveVisibilityPath(repoRoot)
		if pathErr != nil {
			log.Printf("WARNING: scenario visibility path unavailable: %v", pathErr)
		} else {
			log.Printf("INFO: Visibility path: %s", visibilityPath)
			manager, catalogErr := NewScenarioCatalogManager(repoRoot, visibilityPath)
			if catalogErr != nil {
				log.Printf("WARNING: scenario catalog unavailable: %v", catalogErr)
			} else {
				catalogManager = manager
				log.Printf("INFO: Scenario catalog manager initialized successfully")
				catalogManager.StartBackgroundRefresh(24 * time.Hour)
			}
		}
	}

	// Initialize Gin router
	r := gin.Default()

	// Add CORS middleware with explicit allowed origins
	r.Use(func(c *gin.Context) {
		allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
		if allowedOrigins == "" {
			allowedOrigins = "http://localhost:3000,http://localhost:35000"
		}

		origin := c.Request.Header.Get("Origin")
		// Only set CORS header if origin is in allowed list
		for _, allowed := range []string{"http://localhost:3000", "http://localhost:35000"} {
			if origin == allowed {
				c.Header("Access-Control-Allow-Origin", origin)
				break
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint
	r.GET("/health", gin.WrapF(health.New().Version("1.0.0").Check(health.DB(db), health.Critical).Handler()))

	// API routes
	api := r.Group("/api/v1")
	{
		api.GET("/health", gin.WrapF(health.New().Version("1.0.0").Check(health.DB(db), health.Critical).Handler()))

		// Tech tree routes
		api.GET("/tech-tree", getTechTree)
		api.GET("/tech-tree/sectors", getSectors)
		api.POST("/tech-tree/sectors", createSectorHandler)
		api.PATCH("/tech-tree/sectors/:id", updateSectorHandler)
		api.DELETE("/tech-tree/sectors/:id", deleteSectorHandler)
		api.GET("/tech-tree/sectors/:id", getSector)
		api.GET("/tech-tree/stages/:id", getStage)
		api.GET("/tech-tree/stages/:id/children", getStageChildren)
		api.POST("/tech-tree/stages", createStageHandler)
		api.PATCH("/tech-tree/stages/:id", updateStageHandler)
		api.DELETE("/tech-tree/stages/:id", deleteStageHandler)
		api.PUT("/tech-tree/graph", updateGraph)
		api.GET("/tech-tree/graph/dot", exportGraphAsDOT)
		api.GET("/tech-trees", listTechTrees)
		api.POST("/tech-trees", createTechTreeHandler)
		api.PATCH("/tech-trees/:id", updateTechTreeHandler)
		api.POST("/tech-trees/:id/clone", cloneTechTreeHandler)
		api.POST("/tech-tree/ai/stage-ideas", aiStageIdeasHandler)
		api.GET("/tech-tree/scenario-catalog", getScenarioCatalogHandler)
		api.POST("/tech-tree/scenario-catalog/refresh", refreshScenarioCatalogHandler)
		api.POST("/tech-tree/scenario-catalog/visibility", updateScenarioVisibilityHandler)

		// Progress tracking routes
		api.GET("/progress/scenarios", getScenarioMappings)
		api.POST("/progress/scenarios", updateScenarioMapping)
		api.DELETE("/progress/scenarios/:id", deleteScenarioMapping)
		api.PUT("/progress/scenarios/:scenario", updateScenarioStatus)

		// Maturity tracking routes
		api.PUT("/stages/:id/maturity", updateStageMaturity)
		api.GET("/stages/:id/maturity/events", getStageMaturityEvents)

		// Strategic analysis routes
		api.POST("/tech-tree/analyze", analyzeStrategicPath)
		api.GET("/milestones", getStrategicMilestones)
		api.POST("/milestones", createStrategicMilestone)
		api.PATCH("/milestones/:id", updateStrategicMilestone)
		api.DELETE("/milestones/:id", deleteStrategicMilestone)
		api.GET("/recommendations", getRecommendations)

		// Dependencies and connections
		api.GET("/dependencies", getDependencies)
		api.GET("/connections", getCrossSectorConnections)

		// Graph query endpoints for agents
		api.GET("/graph/neighborhood", getStageNeighborhood)
		api.GET("/graph/path", getShortestPath)
		api.GET("/graph/ancestors", getStageAncestors)
		api.GET("/graph/export/view", exportGraphViewAsText)
	}

	// Get port from environment (required)
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("API_PORT environment variable is required (no default port allowed for security)")
	}

	log.Printf("🚀 Tech Tree Designer API starting on port %s", port)
	log.Printf("🌟 Strategic Intelligence System ready for superintelligence guidance")

	r.Run(":" + port)
}

func resolveVisibilityPath(repoRoot string) (string, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", err
	}
	path, err := resolver.Path(storage.Options{ScenarioID: "tech-tree-designer"}, storage.ClassConfig, "scenario_visibility.json")
	if err != nil {
		return "", err
	}
	return path, nil
}

func migrateVisibilityPath(src, dst string) error {
	if src == dst {
		return nil
	}
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if _, err := os.Stat(dst); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.Rename(src, dst)
}

// Get the main tech tree
