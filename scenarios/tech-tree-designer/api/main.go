package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
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
	workingDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	log.Printf("DEBUG: resolveRepoRoot - starting from working dir: %s", workingDir)

	current := workingDir
	for depth := 0; depth < 8; depth++ {
		log.Printf("DEBUG: resolveRepoRoot - checking depth %d: %s", depth, current)

		// Always prefer .git as the definitive marker of repo root
		gitPath := filepath.Join(current, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			log.Printf("DEBUG: resolveRepoRoot - found .git at: %s", current)
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			log.Printf("DEBUG: resolveRepoRoot - reached filesystem root")
			break
		}
		current = parent
	}

	// Fallback: if we didn't find .git, look for scenarios/ directory
	// (this handles non-git installations)
	current = workingDir
	for depth := 0; depth < 8; depth++ {
		scenariosPath := filepath.Join(current, "scenarios")
		if info, err := os.Stat(scenariosPath); err == nil && info.IsDir() {
			// Verify this looks like the root scenarios dir, not a subdirectory
			// by checking if it contains multiple scenario directories
			entries, readErr := os.ReadDir(scenariosPath)
			if readErr == nil && len(entries) > 5 {
				log.Printf("DEBUG: resolveRepoRoot - found scenarios/ with %d entries at: %s", len(entries), current)
				return current, nil
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	log.Printf("DEBUG: resolveRepoRoot - failed to find repo root after 8 levels")
	return "", fmt.Errorf("unable to resolve repository root from %s", workingDir)
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

func getAppVersion() string {
	version := os.Getenv("APP_VERSION")
	if version == "" {
		return "1.0.0"
	}
	return version
}

func buildHealthResponse(ctx context.Context) gin.H {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	start := time.Now()
	dbErr := db.PingContext(ctx)
	latency := time.Since(start).Milliseconds()

	databaseHealth := gin.H{
		"connected":  dbErr == nil,
		"latency_ms": float64(latency),
		"error":      nil,
	}

	status := "healthy"
	readiness := true

	if dbErr != nil {
		status = "degraded"
		readiness = false
		databaseHealth["error"] = gin.H{
			"code":      "DATABASE_UNAVAILABLE",
			"message":   dbErr.Error(),
			"category":  "resource",
			"retryable": true,
		}
	}

	return gin.H{
		"status":    status,
		"service":   "tech-tree-designer-api",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"readiness": readiness,
		"version":   getAppVersion(),
		"dependencies": gin.H{
			"database": databaseHealth,
		},
	}
}

func initDB() error {
	_ = godotenv.Load()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		dbHost := os.Getenv("POSTGRES_HOST")
		if dbHost == "" {
			dbHost = os.Getenv("DB_HOST")
		}
		if dbHost == "" {
			dbHost = "localhost"
		}

		dbPort := os.Getenv("POSTGRES_PORT")
		if dbPort == "" {
			dbPort = os.Getenv("DB_PORT")
		}

		dbUser := os.Getenv("POSTGRES_USER")
		if dbUser == "" {
			dbUser = os.Getenv("DB_USER")
		}

		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		if dbPassword == "" {
			dbPassword = os.Getenv("DB_PASSWORD")
		}

		dbName := os.Getenv("POSTGRES_DB")
		if dbName == "" {
			dbName = os.Getenv("DB_NAME")
		}

		if dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			return fmt.Errorf("database configuration missing. Provide DATABASE_URL or all of: POSTGRES_HOST/DB_HOST, POSTGRES_PORT/DB_PORT, POSTGRES_USER/DB_USER, POSTGRES_PASSWORD/DB_PASSWORD, POSTGRES_DB/DB_NAME")
		}

		databaseURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	var err error
	db, err = sql.Open("postgres", databaseURL)
	if err != nil {
		return err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	maxRetries := 10
	baseDelay := 500 * time.Millisecond
	maxDelay := 30 * time.Second
	randSource := rand.New(rand.NewSource(time.Now().UnixNano()))

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			return nil
		}

		delay := time.Duration(math.Min(float64(baseDelay)*math.Pow(2, float64(attempt)), float64(maxDelay)))
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(randSource.Float64() * jitterRange)
		waitTime := delay + jitter

		log.Printf("Database connection attempt %d/%d failed: %v. Retrying in %v", attempt+1, maxRetries, pingErr, waitTime)
		time.Sleep(waitTime)
	}

	return fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, pingErr)
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start tech-tree-designer

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Initialize database
	if err := initDB(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

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
		visibilityPath := filepath.Join(repoRoot, "scenarios", "tech-tree-designer", "data", "scenario_visibility.json")
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

	// Health check endpoints compatible with lifecycle schemas
	healthHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, buildHealthResponse(c.Request.Context()))
	}

	r.GET("/health", healthHandler)

	// API routes
	api := r.Group("/api/v1")
	{
		api.GET("/health", healthHandler)

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

// Get the main tech tree
