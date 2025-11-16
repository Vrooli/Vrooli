package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	orderedmap "github.com/iancoleman/orderedmap"
	"github.com/joho/godotenv"
	"github.com/rs/cors"

	pq "github.com/lib/pq"
)

const (
	deploymentReportVersion = 1
	tierBlockerThreshold    = 0.75
)

// Domain models
type ServiceConfig struct {
	Schema  string `json:"$schema"`
	Version string `json:"version"`
	Service struct {
		Name        string   `json:"name"`
		DisplayName string   `json:"displayName"`
		Description string   `json:"description"`
		Version     string   `json:"version"`
		Tags        []string `json:"tags"`
	} `json:"service"`
	Ports        map[string]interface{} `json:"ports"`
	Resources    map[string]Resource    `json:"resources"`
	Dependencies struct {
		Resources map[string]Resource               `json:"resources"`
		Scenarios map[string]ScenarioDependencySpec `json:"scenarios"`
	} `json:"dependencies"`
	Deployment *ServiceDeployment `json:"deployment"`
}

type Resource struct {
	Type           string                   `json:"type"`
	Enabled        bool                     `json:"enabled"`
	Required       bool                     `json:"required"`
	Purpose        string                   `json:"purpose"`
	Initialization []map[string]interface{} `json:"initialization,omitempty"`
	Models         []string                 `json:"models,omitempty"`
}

type ScenarioDependencySpec struct {
	Required     bool   `json:"required"`
	Version      string `json:"version"`
	VersionRange string `json:"versionRange"`
	Description  string `json:"description"`
}

type ServiceDeployment struct {
	MetadataVersion       int                         `json:"metadata_version"`
	LastAnalyzedAt        string                      `json:"last_analyzed_at"`
	Analyzer              *DeploymentAnalyzerInfo     `json:"analyzer"`
	AggregateRequirements *DeploymentRequirements     `json:"aggregate_requirements"`
	Tiers                 map[string]DeploymentTier   `json:"tiers"`
	Dependencies          DeploymentDependencyCatalog `json:"dependencies"`
	Overrides             []DeploymentOverride        `json:"overrides"`
}

type DeploymentAnalyzerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type DeploymentDependencyCatalog struct {
	Resources map[string]DeploymentDependency `json:"resources"`
	Scenarios map[string]DeploymentDependency `json:"scenarios"`
}

type DeploymentDependency struct {
	ResourceType    string                           `json:"resource_type"`
	Footprint       *DeploymentRequirements          `json:"footprint"`
	PlatformSupport map[string]DependencyTierSupport `json:"platform_support"`
	SwappableWith   []DependencySwap                 `json:"swappable_with"`
	PackagingHints  []string                         `json:"packaging_hints"`
}

type DeploymentRequirements struct {
	RAMMB            *float64 `json:"ram_mb,omitempty"`
	DiskMB           *float64 `json:"disk_mb,omitempty"`
	CPUCores         *float64 `json:"cpu_cores,omitempty"`
	GPU              *bool    `json:"gpu,omitempty"`
	Network          string   `json:"network,omitempty"`
	StorageMBPerUser *float64 `json:"storage_mb_per_user,omitempty"`
	StartupTimeMS    *float64 `json:"startup_time_ms,omitempty"`
	Bucket           string   `json:"bucket,omitempty"`
	Source           string   `json:"source,omitempty"`
	Confidence       string   `json:"confidence,omitempty"`
}

type DependencyTierSupport struct {
	Supported    *bool                   `json:"supported,omitempty"`
	FitnessScore *float64                `json:"fitness_score,omitempty"`
	Reason       string                  `json:"reason,omitempty"`
	Requirements *DeploymentRequirements `json:"requirements,omitempty"`
	Alternatives []string                `json:"alternatives,omitempty"`
	Notes        string                  `json:"notes,omitempty"`
}

type DependencySwap struct {
	ID           string `json:"id"`
	Relationship string `json:"relationship"`
	Notes        string `json:"notes"`
}

type DeploymentTier struct {
	Status       string                  `json:"status"`
	FitnessScore *float64                `json:"fitness_score,omitempty"`
	Constraints  []string                `json:"constraints,omitempty"`
	Requirements *DeploymentRequirements `json:"requirements,omitempty"`
	Adaptations  []DeploymentAdaptation  `json:"adaptations,omitempty"`
	Secrets      []DeploymentTierSecret  `json:"secrets,omitempty"`
	Artifacts    []DeploymentArtifact    `json:"artifacts,omitempty"`
	Notes        string                  `json:"notes,omitempty"`
}

type DeploymentAdaptation struct {
	Dependency string  `json:"dependency"`
	Swap       string  `json:"swap"`
	Impact     string  `json:"impact"`
	EffortDays float64 `json:"effort_days"`
	Notes      string  `json:"notes"`
}

type DeploymentTierSecret struct {
	SecretID       string `json:"secret_id"`
	Classification string `json:"classification"`
	StrategyRef    string `json:"strategy_ref"`
	Notes          string `json:"notes"`
}

type DeploymentArtifact struct {
	Type     string `json:"type"`
	Producer string `json:"producer"`
	Status   string `json:"status"`
	Notes    string `json:"notes"`
}

type DeploymentOverride struct {
	Tier      string      `json:"tier"`
	Field     string      `json:"field"`
	Value     interface{} `json:"value"`
	Reason    string      `json:"reason"`
	ExpiresAt string      `json:"expires_at"`
}

type ScenarioDependency struct {
	ID             string                 `json:"id" db:"id"`
	ScenarioName   string                 `json:"scenario_name" db:"scenario_name"`
	DependencyType string                 `json:"dependency_type" db:"dependency_type"`
	DependencyName string                 `json:"dependency_name" db:"dependency_name"`
	Required       bool                   `json:"required" db:"required"`
	Purpose        string                 `json:"purpose" db:"purpose"`
	AccessMethod   string                 `json:"access_method" db:"access_method"`
	Configuration  map[string]interface{} `json:"configuration" db:"configuration"`
	DiscoveredAt   time.Time              `json:"discovered_at" db:"discovered_at"`
	LastVerified   time.Time              `json:"last_verified" db:"last_verified"`
}

type DependencyGraph struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"graph_type"`
	Nodes    []GraphNode            `json:"nodes"`
	Edges    []GraphEdge            `json:"edges"`
	Metadata map[string]interface{} `json:"metadata"`
}

type GraphNode struct {
	ID       string                 `json:"id"`
	Label    string                 `json:"label"`
	Type     string                 `json:"type"`
	Group    string                 `json:"group"`
	Metadata map[string]interface{} `json:"metadata"`
}

type GraphEdge struct {
	Source   string                 `json:"source"`
	Target   string                 `json:"target"`
	Label    string                 `json:"label"`
	Type     string                 `json:"type"`
	Required bool                   `json:"required"`
	Weight   float64                `json:"weight"`
	Metadata map[string]interface{} `json:"metadata"`
}

type AnalysisRequest struct {
	ScenarioName      string `json:"scenario_name"`
	IncludeTransitive bool   `json:"include_transitive"`
}

type ProposedScenarioRequest struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Requirements     []string `json:"requirements"`
	SimilarScenarios []string `json:"similar_scenarios,omitempty"`
}

type DependencyAnalysisResponse struct {
	Scenario              string                            `json:"scenario"`
	Resources             []ScenarioDependency              `json:"resources"`
	DetectedResources     []ScenarioDependency              `json:"detected_resources"`
	Scenarios             []ScenarioDependency              `json:"scenarios"`
	DeclaredScenarioSpecs map[string]ScenarioDependencySpec `json:"declared_scenarios"`
	SharedWorkflows       []ScenarioDependency              `json:"shared_workflows"`
	TransitiveDepth       int                               `json:"transitive_depth"`
	ResourceDiff          DependencyDiff                    `json:"resource_diff"`
	ScenarioDiff          DependencyDiff                    `json:"scenario_diff"`
	DeploymentReport      *DeploymentAnalysisReport         `json:"deployment_report,omitempty"`
}

type DependencyDiff struct {
	Missing []DependencyDrift `json:"missing"`
	Extra   []DependencyDrift `json:"extra"`
}

type DependencyDrift struct {
	Name    string                 `json:"name"`
	Details map[string]interface{} `json:"details,omitempty"`
}

type DeploymentAnalysisReport struct {
	Scenario       string                             `json:"scenario"`
	ReportVersion  int                                `json:"report_version"`
	GeneratedAt    time.Time                          `json:"generated_at"`
	Dependencies   []DeploymentDependencyNode         `json:"dependencies"`
	Aggregates     map[string]DeploymentTierAggregate `json:"aggregates"`
	BundleManifest BundleManifest                     `json:"bundle_manifest"`
}

type DeploymentDependencyNode struct {
	Name         string                        `json:"name"`
	Type         string                        `json:"type"`
	ResourceType string                        `json:"resource_type,omitempty"`
	Path         string                        `json:"path,omitempty"`
	Requirements *DeploymentRequirements       `json:"requirements,omitempty"`
	TierSupport  map[string]TierSupportSummary `json:"tier_support,omitempty"`
	Alternatives []string                      `json:"alternatives,omitempty"`
	Notes        string                        `json:"notes,omitempty"`
	Source       string                        `json:"source,omitempty"`
	Children     []DeploymentDependencyNode    `json:"children,omitempty"`
	Metadata     map[string]interface{}        `json:"metadata,omitempty"`
}

type TierSupportSummary struct {
	Supported    *bool                   `json:"supported,omitempty"`
	FitnessScore *float64                `json:"fitness_score,omitempty"`
	Reason       string                  `json:"reason,omitempty"`
	Notes        string                  `json:"notes,omitempty"`
	Requirements *DeploymentRequirements `json:"requirements,omitempty"`
	Alternatives []string                `json:"alternatives,omitempty"`
}

type DeploymentTierAggregate struct {
	FitnessScore          float64                `json:"fitness_score"`
	DependencyCount       int                    `json:"dependency_count"`
	BlockingDependencies  []string               `json:"blocking_dependencies,omitempty"`
	EstimatedRequirements AggregatedRequirements `json:"estimated_requirements"`
}

type AggregatedRequirements struct {
	RAMMB    float64 `json:"ram_mb"`
	DiskMB   float64 `json:"disk_mb"`
	CPUCores float64 `json:"cpu_cores"`
}

type BundleManifest struct {
	Scenario     string                  `json:"scenario"`
	GeneratedAt  time.Time               `json:"generated_at"`
	Files        []BundleFileEntry       `json:"files"`
	Dependencies []BundleDependencyEntry `json:"dependencies"`
}

type BundleFileEntry struct {
	Path   string `json:"path"`
	Type   string `json:"type"`
	Exists bool   `json:"exists"`
	Notes  string `json:"notes,omitempty"`
}

type BundleDependencyEntry struct {
	Name         string                        `json:"name"`
	Type         string                        `json:"type"`
	ResourceType string                        `json:"resource_type,omitempty"`
	TierSupport  map[string]TierSupportSummary `json:"tier_support,omitempty"`
	Alternatives []string                      `json:"alternatives,omitempty"`
}

type OptimizationRequest struct {
	Scenario string `json:"scenario"`
	Type     string `json:"type"`
	Apply    bool   `json:"apply"`
}

type OptimizationResult struct {
	Scenario          string                       `json:"scenario"`
	Recommendations   []OptimizationRecommendation `json:"recommendations"`
	Summary           OptimizationSummary          `json:"summary"`
	Applied           bool                         `json:"applied"`
	ApplySummary      map[string]interface{}       `json:"apply_summary,omitempty"`
	AnalysisTimestamp time.Time                    `json:"analysis_timestamp"`
	Error             string                       `json:"error,omitempty"`
}

type OptimizationSummary struct {
	RecommendationCount int            `json:"recommendation_count"`
	ByType              map[string]int `json:"by_type"`
	HighPriority        int            `json:"high_priority"`
	PotentialImpact     map[string]int `json:"potential_impact"`
}

type OptimizationRecommendation struct {
	ID                 string                 `json:"id"`
	ScenarioName       string                 `json:"scenario_name"`
	RecommendationType string                 `json:"recommendation_type"`
	Title              string                 `json:"title"`
	Description        string                 `json:"description"`
	CurrentState       map[string]interface{} `json:"current_state"`
	RecommendedState   map[string]interface{} `json:"recommended_state"`
	EstimatedImpact    map[string]interface{} `json:"estimated_impact"`
	ConfidenceScore    float64                `json:"confidence_score"`
	Priority           string                 `json:"priority"`
	Status             string                 `json:"status"`
	CreatedAt          time.Time              `json:"created_at"`
}

type ScenarioSummary struct {
	Name        string     `json:"name"`
	DisplayName string     `json:"display_name"`
	Description string     `json:"description"`
	LastScanned *time.Time `json:"last_scanned,omitempty"`
	Tags        []string   `json:"tags"`
}

type ScenarioDetailResponse struct {
	Scenario                    string                            `json:"scenario"`
	DisplayName                 string                            `json:"display_name"`
	Description                 string                            `json:"description"`
	LastScanned                 *time.Time                        `json:"last_scanned,omitempty"`
	DeclaredResources           map[string]Resource               `json:"declared_resources"`
	DeclaredScenarios           map[string]ScenarioDependencySpec `json:"declared_scenarios"`
	StoredDependencies          map[string][]ScenarioDependency   `json:"stored_dependencies"`
	ResourceDiff                DependencyDiff                    `json:"resource_diff"`
	ScenarioDiff                DependencyDiff                    `json:"scenario_diff"`
	OptimizationRecommendations []OptimizationRecommendation      `json:"optimization_recommendations"`
	DeploymentReport            *DeploymentAnalysisReport         `json:"deployment_report,omitempty"`
}

type ScanRequest struct {
	Apply          bool `json:"apply"`
	ApplyResources bool `json:"apply_resources"`
	ApplyScenarios bool `json:"apply_scenarios"`
}

// Database connection and detection helpers
var (
	db                       *sql.DB
	applyDiffsHook           func(string, *ServiceConfig)
	resourceCommandPattern   = regexp.MustCompile(`resource-([a-z0-9-]+)`)
	resourceHeuristicCatalog = []resourceHeuristic{
		{
			Name: "postgres",
			Type: "postgres",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`postgres(ql)?:\/\/`),
				regexp.MustCompile(`PGHOST`),
			},
		},
		{
			Name: "redis",
			Type: "redis",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`redis:\/\/`),
				regexp.MustCompile(`REDIS_URL`),
			},
		},
		{
			Name: "ollama",
			Type: "ollama",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`ollama`),
			},
		},
		{
			Name: "qdrant",
			Type: "qdrant",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`qdrant`),
			},
		},
		{
			Name: "n8n",
			Type: "n8n",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`resource-?n8n`),
				regexp.MustCompile(`N8N_[A-Z0-9_]+`),
				regexp.MustCompile(`RESOURCE_PORT_N8N`),
				regexp.MustCompile(`n8n/(?:api|webhook)`),
			},
		},
		{
			Name: "minio",
			Type: "minio",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`minio`),
			},
		},
	}

	dependencyCatalogMu     sync.RWMutex
	dependencyCatalogLoaded bool
	knownScenarioNames      map[string]struct{}
	knownResourceNames      map[string]struct{}
	analysisIgnoreSegments  = map[string]struct{}{
		"docs":          {},
		"doc":           {},
		"documentation": {},
		"readme":        {},
		"test":          {},
		"tests":         {},
		"testdata":      {},
		"__tests__":     {},
		"spec":          {},
		"specs":         {},
		"coverage":      {},
		"examples":      {},
		"playbooks":     {},
		"data":          {},
		"draft":         {},
		"drafts":        {},
		"prd-drafts":    {},
		"dist":          {},
		"build":         {},
		"out":           {},
		"outputs":       {},
	}

	skipDirectoryNames = map[string]struct{}{
		"node_modules":     {},
		"dist":             {},
		"build":            {},
		"coverage":         {},
		"logs":             {},
		"tmp":              {},
		"temp":             {},
		"vendor":           {},
		"__pycache__":      {},
		".pytest_cache":    {},
		".nyc_output":      {},
		"storybook-static": {},
		".next":            {},
		".nuxt":            {},
		".svelte-kit":      {},
		".vercel":          {},
		".parcel-cache":    {},
		".turbo":           {},
		".git":             {},
		".hg":              {},
		".svn":             {},
		".idea":            {},
		".vscode":          {},
		".cache":           {},
		".output":          {},
		".yalc":            {},
		".yarn":            {},
		".pnpm":            {},
	}
	docExtensions = map[string]struct{}{
		".md":  {},
		".mdx": {},
		".rst": {},
		".txt": {},
	}
	docFileNames = map[string]struct{}{
		"readme":           {},
		"readme.md":        {},
		"readme.mdx":       {},
		"prd.md":           {},
		"prd.mdx":          {},
		"problems.md":      {},
		"requirements.md":  {},
		"requirements.mdx": {},
	}
	scenarioPortCallPattern       = regexp.MustCompile(`resolveScenarioPortViaCLI\s*\(\s*[^,]+,\s*(?:"([a-z0-9-]+)"|([A-Za-z0-9_]+))\s*,`)
	scenarioAliasDeclPattern      = regexp.MustCompile(`(?m)(?:const|var)\s+([A-Za-z0-9_]+)\s*=\s*"([a-z0-9-]+)"`)
	scenarioAliasShortPattern     = regexp.MustCompile(`(?m)([A-Za-z0-9_]+)\s*:=\s*"([a-z0-9-]+)"`)
	scenarioAliasBlockPattern     = regexp.MustCompile(`(?m)^\s*([A-Za-z0-9_]+)\s*=\s*"([a-z0-9-]+)"`)
	resourceCLIDirectoryAllowList = map[string]struct{}{
		"":               {},
		"api":            {},
		"cli":            {},
		"cmd":            {},
		"scripts":        {},
		"script":         {},
		"ui":             {},
		"src":            {},
		"server":         {},
		"services":       {},
		"service":        {},
		"lib":            {},
		"pkg":            {},
		"internal":       {},
		"tools":          {},
		"initialization": {},
		"automation":     {},
		"test":           {},
		"tests":          {},
		"integration":    {},
		"config":         {},
	}
)

type resourceHeuristic struct {
	Name     string
	Type     string
	Patterns []*regexp.Regexp
}

func ensureDependencyCatalogs() {
	dependencyCatalogMu.RLock()
	if dependencyCatalogLoaded {
		dependencyCatalogMu.RUnlock()
		return
	}
	dependencyCatalogMu.RUnlock()

	dependencyCatalogMu.Lock()
	defer dependencyCatalogMu.Unlock()
	if dependencyCatalogLoaded {
		return
	}

	scenariosDir := determineScenariosDir()
	knownScenarioNames = discoverAvailableScenarios(scenariosDir)
	resourcesDir := filepath.Join(filepath.Dir(scenariosDir), "resources")
	knownResourceNames = discoverAvailableResources(resourcesDir)
	dependencyCatalogLoaded = true
}

func determineScenariosDir() string {
	dir := os.Getenv("VROOLI_SCENARIOS_DIR")
	if dir == "" {
		dir = "../.."
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return dir
	}
	return absDir
}

func discoverAvailableScenarios(dir string) map[string]struct{} {
	results := map[string]struct{}{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("⚠️  Unable to read scenarios directory %s: %v", dir, err)
		return results
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		servicePath := filepath.Join(dir, entry.Name(), ".vrooli", "service.json")
		if _, err := os.Stat(servicePath); err == nil {
			results[normalizeName(entry.Name())] = struct{}{}
		}
	}
	return results
}

func discoverAvailableResources(dir string) map[string]struct{} {
	results := map[string]struct{}{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("⚠️  Unable to read resources directory %s: %v", dir, err)
		return results
	}
	for _, entry := range entries {
		if entry.IsDir() {
			results[normalizeName(entry.Name())] = struct{}{}
		}
	}
	return results
}

func shouldIgnoreDetectionFile(relPath string) bool {
	if relPath == "" {
		return false
	}
	lowerPath := strings.ToLower(relPath)
	base := strings.ToLower(filepath.Base(lowerPath))
	if _, ok := docFileNames[base]; ok {
		return true
	}
	if ext := strings.ToLower(filepath.Ext(base)); ext != "" {
		if _, ok := docExtensions[ext]; ok {
			return true
		}
	}
	if strings.HasPrefix(base, "readme") {
		return true
	}
	segments := strings.Split(lowerPath, string(filepath.Separator))
	for _, segment := range segments {
		if _, ok := analysisIgnoreSegments[segment]; ok {
			return true
		}
	}
	return false
}

func isAllowedResourceCLIPath(relPath string) bool {
	if relPath == "" {
		return true
	}
	segments := strings.Split(relPath, string(filepath.Separator))
	if len(segments) == 0 {
		return true
	}
	root := strings.ToLower(strings.TrimSpace(segments[0]))
	if root == "." {
		root = ""
	}
	_, ok := resourceCLIDirectoryAllowList[root]
	return ok
}

func isKnownScenario(name string) bool {
	ensureDependencyCatalogs()
	dependencyCatalogMu.RLock()
	defer dependencyCatalogMu.RUnlock()
	_, ok := knownScenarioNames[normalizeName(name)]
	return ok
}

func isKnownResource(name string) bool {
	ensureDependencyCatalogs()
	dependencyCatalogMu.RLock()
	defer dependencyCatalogMu.RUnlock()
	_, ok := knownResourceNames[normalizeName(name)]
	return ok
}

func refreshDependencyCatalogs() {
	dependencyCatalogMu.Lock()
	defer dependencyCatalogMu.Unlock()
	dependencyCatalogLoaded = false
	knownScenarioNames = nil
	knownResourceNames = nil
}

func normalizeName(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}

func shouldSkipDirectoryEntry(d fs.DirEntry) bool {
	if !d.IsDir() {
		return false
	}
	name := d.Name()
	if _, ok := skipDirectoryNames[strings.ToLower(name)]; ok {
		return true
	}
	if strings.HasPrefix(strings.ToLower(name), "node_modules") {
		return true
	}
	if strings.HasPrefix(strings.ToLower(name), ".ignored") {
		return true
	}
	if strings.HasPrefix(name, ".") && name != ".vrooli" {
		return true
	}
	return false
}

// Configuration
type Config struct {
	Port         string
	DatabaseURL  string
	ScenariosDir string
}

func loadConfig() Config {
	godotenv.Load()

	// Port configuration - use standard API_PORT
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	// Database configuration - support both DATABASE_URL and individual components
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Try to build from individual components
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Missing database configuration. Provide DATABASE_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}

		dbURL = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPassword, dbName)
	}

	// Optional scenarios directory (has reasonable default)
	scenariosDir := os.Getenv("VROOLI_SCENARIOS_DIR")
	if scenariosDir == "" {
		scenariosDir = "../.." // Reasonable default for project structure
	}

	return Config{
		Port:         port,
		DatabaseURL:  dbURL,
		ScenariosDir: scenariosDir,
	}
}

// Removed getEnvWithDefault function - no defaults allowed for sensitive config

func initDatabase(dbURL string) error {
	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Println("📊 Database URL configured")

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt+1)
			break
		}

		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))

		// Add random jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(rand.Float64() * jitterRange)
		actualDelay := delay + jitter

		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)

		// Provide detailed status every few attempts
		if attempt > 0 && attempt%3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt+1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt*2)*baseDelay)
			log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}

		time.Sleep(actualDelay)
	}

	if pingErr != nil {
		return fmt.Errorf("❌ Database connection failed after %d attempts: %w", maxRetries, pingErr)
	}

	log.Println("🎉 Database connection pool established successfully!")
	return nil
}

// Core analysis functions
func analyzeScenario(scenarioName string) (*DependencyAnalysisResponse, error) {
	config := loadConfig()
	scenarioPath := filepath.Join(config.ScenariosDir, scenarioName)
	serviceConfig, err := loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		return nil, err
	}

	response := &DependencyAnalysisResponse{
		Scenario:              scenarioName,
		Resources:             []ScenarioDependency{},
		DetectedResources:     []ScenarioDependency{},
		Scenarios:             []ScenarioDependency{},
		DeclaredScenarioSpecs: map[string]ScenarioDependencySpec{},
		SharedWorkflows:       []ScenarioDependency{},
		TransitiveDepth:       0,
		ResourceDiff:          DependencyDiff{},
		ScenarioDiff:          DependencyDiff{},
	}

	declaredResources := extractDeclaredResources(scenarioName, serviceConfig)
	response.Resources = declaredResources
	response.DeclaredScenarioSpecs = normalizeScenarioSpecs(serviceConfig.Dependencies.Scenarios)

	detectedResources, err := scanForResourceUsage(scenarioPath, scenarioName)
	if err != nil {
		log.Printf("Warning: failed to scan for resource usage: %v", err)
	} else {
		response.DetectedResources = detectedResources
	}

	scenarioDeps, err := scanForScenarioDependencies(scenarioPath, scenarioName)
	if err != nil {
		log.Printf("Warning: failed to scan for scenario dependencies: %v", err)
	} else {
		response.Scenarios = append(response.Scenarios, scenarioDeps...)
	}

	workflowDeps, err := scanForSharedWorkflows(scenarioPath, scenarioName)
	if err != nil {
		log.Printf("Warning: failed to scan for shared workflows: %v", err)
	} else {
		response.SharedWorkflows = append(response.SharedWorkflows, workflowDeps...)
	}

	declaredScenarioDeps := convertDeclaredScenariosToDependencies(scenarioName, response.DeclaredScenarioSpecs)
	response.ResourceDiff = buildResourceDiff(resolvedResourceMap(serviceConfig), response.DetectedResources)
	response.ScenarioDiff = buildScenarioDiff(response.DeclaredScenarioSpecs, response.Scenarios)

	if err := storeDependencies(response, declaredScenarioDeps); err != nil {
		log.Printf("Warning: failed to store dependencies in database: %v", err)
	}

	if err := updateScenarioMetadata(scenarioName, serviceConfig, scenarioPath); err != nil {
		log.Printf("Warning: failed to update scenario metadata: %v", err)
	}

	if deploymentReport := buildDeploymentReport(scenarioName, scenarioPath, config.ScenariosDir, serviceConfig); deploymentReport != nil {
		response.DeploymentReport = deploymentReport
		if err := persistDeploymentReport(scenarioPath, deploymentReport); err != nil {
			log.Printf("Warning: failed to persist deployment report: %v", err)
		}
	}

	return response, nil
}

func runScenarioOptimization(scenarioName string, req OptimizationRequest) (*OptimizationResult, error) {
	analysis, err := analyzeScenario(scenarioName)
	if err != nil {
		return nil, err
	}
	scenarioPath := filepath.Join(loadConfig().ScenariosDir, scenarioName)
	cfg, err := loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		cfg = nil
	}
	recommendations := generateOptimizationRecommendations(scenarioName, analysis, cfg)
	if err := persistOptimizationRecommendations(scenarioName, recommendations); err != nil {
		return nil, err
	}
	var applySummary map[string]interface{}
	applied := false
	if req.Apply && len(recommendations) > 0 {
		applySummary, err = applyOptimizationRecommendations(scenarioName, recommendations)
		if err != nil {
			return nil, err
		}
		if changed, ok := applySummary["changed"].(bool); ok && changed {
			applied = true
			analysis, err = analyzeScenario(scenarioName)
			if err != nil {
				return nil, err
			}
			recommendations = generateOptimizationRecommendations(scenarioName, analysis, cfg)
			if err := persistOptimizationRecommendations(scenarioName, recommendations); err != nil {
				return nil, err
			}
		}
	}
	return &OptimizationResult{
		Scenario:          scenarioName,
		Recommendations:   recommendations,
		Summary:           buildOptimizationSummary(recommendations),
		Applied:           applied,
		ApplySummary:      applySummary,
		AnalysisTimestamp: time.Now().UTC(),
	}, nil
}

func buildDeploymentReport(scenarioName, scenarioPath, scenariosDir string, cfg *ServiceConfig) *DeploymentAnalysisReport {
	if cfg == nil {
		return nil
	}

	generatedAt := time.Now().UTC()
	visited := map[string]struct{}{}
	visited[normalizeName(scenarioName)] = struct{}{}
	nodes := buildDependencyNodeList(scenariosDir, scenarioName, cfg, visited)
	aggregates := computeTierAggregates(nodes)
	manifest := buildBundleManifest(scenarioName, scenarioPath, generatedAt, nodes)

	return &DeploymentAnalysisReport{
		Scenario:       scenarioName,
		ReportVersion:  deploymentReportVersion,
		GeneratedAt:    generatedAt,
		Dependencies:   nodes,
		Aggregates:     aggregates,
		BundleManifest: manifest,
	}
}

func buildDependencyNodeList(scenariosDir, scenarioName string, cfg *ServiceConfig, visited map[string]struct{}) []DeploymentDependencyNode {
	nodes := []DeploymentDependencyNode{}
	if cfg == nil {
		return nodes
	}

	var dependencyCatalog DeploymentDependencyCatalog
	if cfg.Deployment != nil {
		dependencyCatalog = cfg.Deployment.Dependencies
	}

	resources := resolvedResourceMap(cfg)
	resourceNames := make([]string, 0, len(resources))
	for name := range resources {
		resourceNames = append(resourceNames, name)
	}
	sort.Strings(resourceNames)
	for _, name := range resourceNames {
		var meta *DeploymentDependency
		if dependencyCatalog.Resources != nil {
			if resourceMeta, ok := dependencyCatalog.Resources[name]; ok {
				copyMeta := resourceMeta
				meta = &copyMeta
			}
		}
		node := buildResourceDependencyNode(name, meta)
		node.Source = "declared"
		nodes = append(nodes, node)
	}

	if cfg.Dependencies.Scenarios != nil {
		scenarioNames := make([]string, 0, len(cfg.Dependencies.Scenarios))
		for name := range cfg.Dependencies.Scenarios {
			scenarioNames = append(scenarioNames, name)
		}
		sort.Strings(scenarioNames)
		for _, depName := range scenarioNames {
			var meta *DeploymentDependency
			if dependencyCatalog.Scenarios != nil {
				if scenarioMeta, ok := dependencyCatalog.Scenarios[depName]; ok {
					copyMeta := scenarioMeta
					meta = &copyMeta
				}
			}
			node := buildScenarioDependencyNode(scenariosDir, depName, meta, visited)
			node.Source = "declared"
			nodes = append(nodes, node)
		}
	}

	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Type == nodes[j].Type {
			return nodes[i].Name < nodes[j].Name
		}
		return nodes[i].Type < nodes[j].Type
	})

	return nodes
}

func buildResourceDependencyNode(name string, meta *DeploymentDependency) DeploymentDependencyNode {
	node := DeploymentDependencyNode{
		Name: name,
		Type: "resource",
	}
	if meta == nil {
		return node
	}
	node.ResourceType = meta.ResourceType
	node.Requirements = meta.Footprint
	node.TierSupport = convertTierSupportMap(meta.PlatformSupport)
	node.Alternatives = collectDependencyAlternatives(meta)
	return node
}

func buildScenarioDependencyNode(scenariosDir, scenarioName string, parentMeta *DeploymentDependency, visited map[string]struct{}) DeploymentDependencyNode {
	node := DeploymentDependencyNode{
		Name: scenarioName,
		Type: "scenario",
	}

	normalized := normalizeName(scenarioName)
	if _, exists := visited[normalized]; exists {
		node.Notes = "cycle detected"
		if parentMeta != nil {
			node.TierSupport = convertTierSupportMap(parentMeta.PlatformSupport)
			node.Requirements = parentMeta.Footprint
			node.Alternatives = collectDependencyAlternatives(parentMeta)
		}
		return node
	}

	if visited == nil {
		visited = map[string]struct{}{}
	}
	visited[normalized] = struct{}{}
	defer delete(visited, normalized)

	scenarioPath := filepath.Join(scenariosDir, scenarioName)
	node.Path = scenarioPath
	cfg, err := loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		node.Notes = fmt.Sprintf("unable to load scenario: %v", err)
		if parentMeta != nil {
			node.TierSupport = convertTierSupportMap(parentMeta.PlatformSupport)
			node.Requirements = parentMeta.Footprint
			node.Alternatives = collectDependencyAlternatives(parentMeta)
		}
		return node
	}

	var scenarioTierSupport map[string]TierSupportSummary
	if cfg.Deployment != nil {
		node.Requirements = cfg.Deployment.AggregateRequirements
		scenarioTierSupport = convertTierTierMap(cfg.Deployment.Tiers)
		node.Alternatives = append(node.Alternatives, collectAdaptationAlternatives(cfg.Deployment.Tiers)...)
	}

	if node.Requirements == nil && parentMeta != nil {
		node.Requirements = parentMeta.Footprint
	}
	fallbackSupport := convertTierSupportMap(nil)
	if parentMeta != nil {
		fallbackSupport = convertTierSupportMap(parentMeta.PlatformSupport)
		node.Alternatives = append(node.Alternatives, collectDependencyAlternatives(parentMeta)...)
	}
	node.TierSupport = mergeTierSupportMaps(scenarioTierSupport, fallbackSupport)
	node.Alternatives = dedupeStrings(node.Alternatives)
	node.Children = buildDependencyNodeList(scenariosDir, scenarioName, cfg, visited)
	return node
}

func convertTierSupportMap(support map[string]DependencyTierSupport) map[string]TierSupportSummary {
	if len(support) == 0 {
		return nil
	}
	result := make(map[string]TierSupportSummary, len(support))
	for tier, value := range support {
		result[tier] = TierSupportSummary{
			Supported:    value.Supported,
			FitnessScore: value.FitnessScore,
			Reason:       value.Reason,
			Notes:        value.Notes,
			Requirements: value.Requirements,
			Alternatives: append([]string(nil), value.Alternatives...),
		}
	}
	return result
}

func convertTierTierMap(tiers map[string]DeploymentTier) map[string]TierSupportSummary {
	if len(tiers) == 0 {
		return nil
	}
	result := make(map[string]TierSupportSummary, len(tiers))
	for tier, value := range tiers {
		var supported *bool
		switch strings.ToLower(value.Status) {
		case "ready", "supported":
			flag := true
			supported = &flag
		case "limited", "blocked":
			flag := false
			supported = &flag
		}
		result[tier] = TierSupportSummary{
			Supported:    supported,
			FitnessScore: value.FitnessScore,
			Notes:        value.Notes,
			Requirements: value.Requirements,
		}
	}
	return result
}

func mergeTierSupportMaps(preferred, fallback map[string]TierSupportSummary) map[string]TierSupportSummary {
	if len(preferred) == 0 && len(fallback) == 0 {
		return nil
	}
	merged := make(map[string]TierSupportSummary)
	for tier, value := range fallback {
		merged[tier] = value
	}
	for tier, value := range preferred {
		merged[tier] = value
	}
	return merged
}

func collectDependencyAlternatives(meta *DeploymentDependency) []string {
	if meta == nil {
		return nil
	}
	set := map[string]struct{}{}
	for _, swap := range meta.SwappableWith {
		if swap.ID == "" {
			continue
		}
		set[swap.ID] = struct{}{}
	}
	for _, support := range meta.PlatformSupport {
		for _, alt := range support.Alternatives {
			if alt == "" {
				continue
			}
			set[alt] = struct{}{}
		}
	}
	return mapKeys(set)
}

func collectAdaptationAlternatives(tiers map[string]DeploymentTier) []string {
	set := map[string]struct{}{}
	for _, tier := range tiers {
		for _, adaptation := range tier.Adaptations {
			if adaptation.Swap == "" {
				continue
			}
			set[adaptation.Swap] = struct{}{}
		}
	}
	return mapKeys(set)
}

func computeTierAggregates(nodes []DeploymentDependencyNode) map[string]DeploymentTierAggregate {
	accum := map[string]*tierAccumulator{}
	var walk func(DeploymentDependencyNode)
	walk = func(node DeploymentDependencyNode) {
		for tier, support := range node.TierSupport {
			acc := accum[tier]
			if acc == nil {
				acc = &tierAccumulator{Blockers: map[string]struct{}{}}
				accum[tier] = acc
			}
			acc.Count++
			req := selectRequirements(node.Requirements, support.Requirements)
			acc.RAM += req.RAMMB
			acc.Disk += req.DiskMB
			acc.CPU += req.CPUCores
			if support.FitnessScore != nil {
				acc.FitnessSum += *support.FitnessScore
			}
			if (support.Supported != nil && !*support.Supported) || (support.FitnessScore != nil && *support.FitnessScore < tierBlockerThreshold) {
				acc.Blockers[node.Name] = struct{}{}
			}
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	for _, node := range nodes {
		walk(node)
	}

	result := make(map[string]DeploymentTierAggregate, len(accum))
	for tier, acc := range accum {
		if acc == nil || acc.Count == 0 {
			continue
		}
		aggregate := DeploymentTierAggregate{
			DependencyCount: acc.Count,
			EstimatedRequirements: AggregatedRequirements{
				RAMMB:    acc.RAM,
				DiskMB:   acc.Disk,
				CPUCores: acc.CPU,
			},
		}
		aggregate.FitnessScore = acc.FitnessSum / float64(acc.Count)
		aggregate.BlockingDependencies = mapKeys(acc.Blockers)
		sort.Strings(aggregate.BlockingDependencies)
		result[tier] = aggregate
	}
	return result
}

func selectRequirements(base, override *DeploymentRequirements) requirementNumbers {
	if override != nil {
		if numbers := requirementsToNumbers(override); numbers.hasValues() {
			return numbers
		}
	}
	return requirementsToNumbers(base)
}

func requirementsToNumbers(req *DeploymentRequirements) requirementNumbers {
	var numbers requirementNumbers
	if req == nil {
		return numbers
	}
	if req.RAMMB != nil {
		numbers.RAMMB = *req.RAMMB
	}
	if req.DiskMB != nil {
		numbers.DiskMB = *req.DiskMB
	}
	if req.CPUCores != nil {
		numbers.CPUCores = *req.CPUCores
	}
	return numbers
}

type requirementNumbers struct {
	RAMMB    float64
	DiskMB   float64
	CPUCores float64
}

func (r requirementNumbers) hasValues() bool {
	return r.RAMMB != 0 || r.DiskMB != 0 || r.CPUCores != 0
}

type tierAccumulator struct {
	RAM        float64
	Disk       float64
	CPU        float64
	FitnessSum float64
	Count      int
	Blockers   map[string]struct{}
}

func persistDeploymentReport(scenarioPath string, report *DeploymentAnalysisReport) error {
	if report == nil {
		return nil
	}
	reportDir := filepath.Join(scenarioPath, ".vrooli", "deployment")
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return err
	}
	reportPath := filepath.Join(reportDir, "deployment-report.json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := reportPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, reportPath)
}

func loadPersistedDeploymentReport(scenarioPath string) (*DeploymentAnalysisReport, error) {
	reportPath := filepath.Join(scenarioPath, ".vrooli", "deployment", "deployment-report.json")
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, err
	}
	var report DeploymentAnalysisReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}
	return &report, nil
}

func buildBundleManifest(scenarioName, scenarioPath string, generatedAt time.Time, nodes []DeploymentDependencyNode) BundleManifest {
	return BundleManifest{
		Scenario:     scenarioName,
		GeneratedAt:  generatedAt,
		Files:        discoverBundleFiles(scenarioName, scenarioPath),
		Dependencies: flattenBundleDependencies(nodes),
	}
}

func discoverBundleFiles(scenarioName, scenarioPath string) []BundleFileEntry {
	candidates := []struct {
		Path  string
		Type  string
		Notes string
	}{
		{Path: filepath.Join(".vrooli", "service.json"), Type: "service-config"},
		{Path: "api", Type: "api-source"},
		{Path: filepath.Join("api", fmt.Sprintf("%s-api", scenarioName)), Type: "api-binary"},
		{Path: filepath.Join("ui", "dist"), Type: "ui-bundle"},
		{Path: filepath.Join("ui", "dist", "index.html"), Type: "ui-entry"},
		{Path: filepath.Join("cli", scenarioName), Type: "cli-binary"},
	}
	entries := make([]BundleFileEntry, 0, len(candidates))
	for _, candidate := range candidates {
		absolute := filepath.Join(scenarioPath, candidate.Path)
		_, err := os.Stat(absolute)
		entries = append(entries, BundleFileEntry{
			Path:   filepath.ToSlash(candidate.Path),
			Type:   candidate.Type,
			Exists: err == nil,
			Notes:  candidate.Notes,
		})
	}
	return entries
}

func flattenBundleDependencies(nodes []DeploymentDependencyNode) []BundleDependencyEntry {
	seen := map[string]BundleDependencyEntry{}
	var walk func(DeploymentDependencyNode)
	walk = func(node DeploymentDependencyNode) {
		key := fmt.Sprintf("%s:%s", node.Type, node.Name)
		if _, exists := seen[key]; !exists {
			seen[key] = BundleDependencyEntry{
				Name:         node.Name,
				Type:         node.Type,
				ResourceType: node.ResourceType,
				TierSupport:  node.TierSupport,
				Alternatives: dedupeStrings(node.Alternatives),
			}
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	for _, node := range nodes {
		walk(node)
	}
	entries := make([]BundleDependencyEntry, 0, len(seen))
	for _, entry := range seen {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Type == entries[j].Type {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Type < entries[j].Type
	})
	return entries
}

func buildDependencyNodeIndex(nodes []DeploymentDependencyNode) map[string]DeploymentDependencyNode {
	index := map[string]DeploymentDependencyNode{}
	var walk func(DeploymentDependencyNode)
	walk = func(node DeploymentDependencyNode) {
		key := strings.ToLower(node.Name)
		if key != "" {
			if _, exists := index[key]; !exists {
				index[key] = node
			}
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	for _, node := range nodes {
		walk(node)
	}
	return index
}

func dedupeStrings(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	return mapKeys(set)
}

func generateOptimizationRecommendations(scenarioName string, analysis *DependencyAnalysisResponse, cfg *ServiceConfig) []OptimizationRecommendation {
	if analysis == nil {
		return nil
	}
	timestamp := time.Now().UTC()
	recommendations := []OptimizationRecommendation{}
	recommendations = append(recommendations, buildUnusedResourceRecommendations(scenarioName, analysis, timestamp)...)
	recommendations = append(recommendations, buildUnusedScenarioRecommendations(scenarioName, analysis, timestamp)...)
	recommendations = append(recommendations, buildTierBlockerRecommendations(scenarioName, analysis, timestamp)...)
	recommendations = append(recommendations, buildSecretStrategyRecommendations(scenarioName, cfg, timestamp)...)
	sort.SliceStable(recommendations, func(i, j int) bool {
		if recommendations[i].Priority == recommendations[j].Priority {
			return recommendations[i].Title < recommendations[j].Title
		}
		priorityOrder := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
		return priorityOrder[strings.ToLower(recommendations[i].Priority)] < priorityOrder[strings.ToLower(recommendations[j].Priority)]
	})
	return recommendations
}

func buildUnusedResourceRecommendations(scenarioName string, analysis *DependencyAnalysisResponse, timestamp time.Time) []OptimizationRecommendation {
	if analysis == nil || len(analysis.ResourceDiff.Extra) == 0 {
		return nil
	}
	recommendations := make([]OptimizationRecommendation, 0, len(analysis.ResourceDiff.Extra))
	for _, drift := range analysis.ResourceDiff.Extra {
		name := strings.TrimSpace(drift.Name)
		if name == "" {
			continue
		}
		recommendations = append(recommendations, OptimizationRecommendation{
			ID:                 uuid.New().String(),
			ScenarioName:       scenarioName,
			RecommendationType: "dependency_reduction",
			Title:              fmt.Sprintf("Remove unused resource '%s'", name),
			Description:        fmt.Sprintf("Resource '%s' is declared but not detected in the scenario codebase.", name),
			CurrentState: map[string]interface{}{
				"declared": true,
				"detected": false,
				"details":  drift.Details,
			},
			RecommendedState: map[string]interface{}{
				"action":        "remove_resource",
				"resource_name": name,
			},
			EstimatedImpact: map[string]interface{}{
				"complexity_score": -1,
				"description":      "Removes unused infrastructure dependency",
			},
			ConfidenceScore: 0.85,
			Priority:        "medium",
			Status:          "pending",
			CreatedAt:       timestamp,
		})
	}
	return recommendations
}

func buildUnusedScenarioRecommendations(scenarioName string, analysis *DependencyAnalysisResponse, timestamp time.Time) []OptimizationRecommendation {
	if analysis == nil || len(analysis.ScenarioDiff.Extra) == 0 {
		return nil
	}
	recommendations := make([]OptimizationRecommendation, 0, len(analysis.ScenarioDiff.Extra))
	for _, drift := range analysis.ScenarioDiff.Extra {
		name := strings.TrimSpace(drift.Name)
		if name == "" {
			continue
		}
		recommendations = append(recommendations, OptimizationRecommendation{
			ID:                 uuid.New().String(),
			ScenarioName:       scenarioName,
			RecommendationType: "dependency_reduction",
			Title:              fmt.Sprintf("Remove unused scenario '%s'", name),
			Description:        fmt.Sprintf("Scenario dependency '%s' is declared but not referenced in the codebase.", name),
			CurrentState: map[string]interface{}{
				"declared": true,
				"detected": false,
			},
			RecommendedState: map[string]interface{}{
				"action":          "remove_scenario_dependency",
				"scenario_name":   name,
				"dependency_type": "scenario",
			},
			EstimatedImpact: map[string]interface{}{
				"complexity_score": -1,
				"description":      "Removes unused scenario coupling",
			},
			ConfidenceScore: 0.8,
			Priority:        "medium",
			Status:          "pending",
			CreatedAt:       timestamp,
		})
	}
	return recommendations
}

func buildTierBlockerRecommendations(scenarioName string, analysis *DependencyAnalysisResponse, timestamp time.Time) []OptimizationRecommendation {
	report := analysis.DeploymentReport
	if report == nil || len(report.Dependencies) == 0 {
		return nil
	}
	nodeIndex := buildDependencyNodeIndex(report.Dependencies)
	recommendations := []OptimizationRecommendation{}
	blockingByTier := map[string][]string{}
	for tier, aggregate := range report.Aggregates {
		for _, blocker := range aggregate.BlockingDependencies {
			key := strings.ToLower(blocker)
			blockingByTier[key] = append(blockingByTier[key], tier)
		}
	}
	for blockerKey, tiers := range blockingByTier {
		node, ok := nodeIndex[blockerKey]
		if !ok {
			continue
		}
		for _, tier := range tiers {
			recommendations = append(recommendations, buildBlockerRecommendation(scenarioName, node, tier, timestamp))
		}
	}
	return recommendations
}

func buildSecretStrategyRecommendations(scenarioName string, cfg *ServiceConfig, timestamp time.Time) []OptimizationRecommendation {
	if cfg == nil || cfg.Deployment == nil || len(cfg.Deployment.Tiers) == 0 {
		return nil
	}
	recommendations := []OptimizationRecommendation{}
	for tierName, tier := range cfg.Deployment.Tiers {
		for _, secret := range tier.Secrets {
			if strings.TrimSpace(secret.StrategyRef) != "" {
				continue
			}
			recommendations = append(recommendations, OptimizationRecommendation{
				ID:                 uuid.New().String(),
				ScenarioName:       scenarioName,
				RecommendationType: "performance_improvement",
				Title:              fmt.Sprintf("Annotate secret '%s' on %s", secret.SecretID, tierName),
				Description:        fmt.Sprintf("Secret '%s' on tier '%s' lacks a strategy reference. Link it to a secrets-manager playbook to unblock deployment readiness.", secret.SecretID, tierName),
				CurrentState: map[string]interface{}{
					"tier":           tierName,
					"secret_id":      secret.SecretID,
					"classification": secret.Classification,
				},
				RecommendedState: map[string]interface{}{
					"action":    "annotate_secret_strategy",
					"tier":      tierName,
					"secret_id": secret.SecretID,
					"guidance":  "Use secrets-manager playbooks to define strategy_ref",
				},
				EstimatedImpact: map[string]interface{}{
					"risk":        "secrets_gap",
					"description": "Missing secret strategies block deployment",
				},
				ConfidenceScore: 0.75,
				Priority:        "high",
				Status:          "pending",
				CreatedAt:       timestamp,
			})
		}
	}
	return recommendations
}

func buildBlockerRecommendation(scenarioName string, node DeploymentDependencyNode, tier string, timestamp time.Time) OptimizationRecommendation {
	alt := ""
	if len(node.Alternatives) > 0 {
		alt = node.Alternatives[0]
	}
	tierSupport := map[string]interface{}{}
	if node.TierSupport != nil {
		if support, ok := node.TierSupport[tier]; ok {
			tierSupport = map[string]interface{}{
				"supported":     support.Supported,
				"fitness_score": support.FitnessScore,
				"notes":         support.Notes,
				"alternatives":  support.Alternatives,
			}
		}
	}
	priority := "high"
	if node.Type != "resource" {
		priority = "medium"
	}
	estImpact := map[string]interface{}{
		"tier":       tier,
		"risk":       "deployment_blocker",
		"dependency": node.Name,
	}
	return OptimizationRecommendation{
		ID:                 uuid.New().String(),
		ScenarioName:       scenarioName,
		RecommendationType: "resource_swap",
		Title:              fmt.Sprintf("Resolve %s blocker on %s", node.Name, tier),
		Description:        fmt.Sprintf("Dependency '%s' blocks tier '%s'. Consider annotated alternatives or lighter swaps to restore fitness.", node.Name, tier),
		CurrentState: map[string]interface{}{
			"dependency":   node.Name,
			"tier":         tier,
			"tier_support": tierSupport,
		},
		RecommendedState: map[string]interface{}{
			"action":                "swap_dependency",
			"dependency":            node.Name,
			"tier":                  tier,
			"suggested_alternative": alt,
		},
		EstimatedImpact: estImpact,
		ConfidenceScore: 0.7,
		Priority:        priority,
		Status:          "pending",
		CreatedAt:       timestamp,
	}
}

func mapKeys(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func scanForScenarioDependencies(scenarioPath, scenarioName string) ([]ScenarioDependency, error) {
	var dependencies []ScenarioDependency
	ensureDependencyCatalogs()
	scenarioNameNormalized := normalizeName(scenarioName)
	aliasCatalog := buildScenarioAliasCatalog(scenarioPath)

	// Pattern to match vrooli scenario commands
	scenarioPattern := regexp.MustCompile(`vrooli\s+scenario\s+(?:run|test|status)\s+([a-z0-9-]+)`)

	// Pattern to match direct CLI calls to other scenarios
	cliPattern := regexp.MustCompile(`([a-z0-9-]+)-cli\.sh|\b([a-z0-9-]+)\s+(?:analyze|process|generate|run)`)

	// Walk through all files in the scenario
	err := filepath.WalkDir(scenarioPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}

		if d.IsDir() && path != scenarioPath && shouldSkipDirectoryEntry(d) {
			return filepath.SkipDir
		}

		// Only scan certain file types
		ext := strings.ToLower(filepath.Ext(path))
		if !contains([]string{".go", ".js", ".sh", ".py", ".md"}, ext) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		contentStr := string(content)
		relPath, relErr := filepath.Rel(scenarioPath, path)
		if relErr != nil {
			relPath = path
		}
		if shouldIgnoreDetectionFile(relPath) {
			return nil
		}

		// Find scenario references
		matches := scenarioPattern.FindAllStringSubmatch(contentStr, -1)
		for _, match := range matches {
			if len(match) > 1 {
				depName := normalizeName(match[1])
				if depName == scenarioNameNormalized || !isKnownScenario(depName) {
					continue
				}
				dep := ScenarioDependency{
					ID:             uuid.New().String(),
					ScenarioName:   scenarioName,
					DependencyType: "scenario",
					DependencyName: depName,
					Required:       false, // Inter-scenario deps are typically optional
					Purpose:        fmt.Sprintf("Referenced in %s", filepath.Base(path)),
					AccessMethod:   "vrooli scenario",
					Configuration: map[string]interface{}{
						"found_in_file": relPath,
						"pattern_type":  "vrooli_cli",
					},
					DiscoveredAt: time.Now(),
					LastVerified: time.Now(),
				}
				dependencies = append(dependencies, dep)
			}
		}

		// Find CLI references
		cliMatches := cliPattern.FindAllStringSubmatch(contentStr, -1)
		for _, match := range cliMatches {
			var scenarioRef string
			if match[1] != "" {
				scenarioRef = match[1]
			} else if match[2] != "" {
				scenarioRef = match[2]
			}

			scenarioRef = normalizeName(scenarioRef)
			if scenarioRef == "" || scenarioRef == scenarioNameNormalized || !isKnownScenario(scenarioRef) {
				continue
			}
			dep := ScenarioDependency{
				ID:             uuid.New().String(),
				ScenarioName:   scenarioName,
				DependencyType: "scenario",
				DependencyName: scenarioRef,
				Required:       false,
				Purpose:        fmt.Sprintf("CLI reference in %s", filepath.Base(path)),
				AccessMethod:   "direct_cli",
				Configuration: map[string]interface{}{
					"found_in_file": relPath,
					"pattern_type":  "cli_reference",
				},
				DiscoveredAt: time.Now(),
				LastVerified: time.Now(),
			}
			dependencies = append(dependencies, dep)
		}

		// Find scenario references via resolveScenarioPortViaCLI helpers
		portMatches := scenarioPortCallPattern.FindAllStringSubmatch(contentStr, -1)
		for _, match := range portMatches {
			var depName string
			if len(match) > 1 && match[1] != "" {
				depName = normalizeName(match[1])
			} else if len(match) > 2 && match[2] != "" {
				if aliasDep, ok := aliasCatalog[match[2]]; ok {
					depName = aliasDep
				} else {
					depName = normalizeName(match[2])
				}
			}

			if depName == "" || depName == scenarioNameNormalized || !isKnownScenario(depName) {
				continue
			}

			dep := ScenarioDependency{
				ID:             uuid.New().String(),
				ScenarioName:   scenarioName,
				DependencyType: "scenario",
				DependencyName: depName,
				Required:       true,
				Purpose:        fmt.Sprintf("References %s port via CLI", depName),
				AccessMethod:   "scenario_port_cli",
				Configuration: map[string]interface{}{
					"found_in_file": relPath,
					"pattern_type":  "resolve_cli_port",
				},
				DiscoveredAt: time.Now(),
				LastVerified: time.Now(),
			}
			dependencies = append(dependencies, dep)
		}

		return nil
	})

	return dependencies, err
}

func buildScenarioAliasCatalog(scenarioPath string) map[string]string {
	aliases := map[string]string{}
	addAlias := func(identifier, scenario string) {
		if identifier == "" {
			return
		}
		normalized := normalizeName(scenario)
		if !isKnownScenario(normalized) {
			return
		}
		aliases[identifier] = normalized
	}

	filepath.WalkDir(scenarioPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && path != scenarioPath && shouldSkipDirectoryEntry(d) {
			return filepath.SkipDir
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !contains([]string{".go", ".js", ".sh", ".py", ".md"}, ext) {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		contentStr := string(content)
		relPath, relErr := filepath.Rel(scenarioPath, path)
		if relErr != nil {
			relPath = path
		}
		if shouldIgnoreDetectionFile(relPath) {
			return nil
		}
		for _, match := range scenarioAliasDeclPattern.FindAllStringSubmatch(contentStr, -1) {
			if len(match) < 3 {
				continue
			}
			addAlias(match[1], match[2])
		}
		for _, match := range scenarioAliasShortPattern.FindAllStringSubmatch(contentStr, -1) {
			if len(match) < 3 {
				continue
			}
			addAlias(match[1], match[2])
		}
		for _, match := range scenarioAliasBlockPattern.FindAllStringSubmatch(contentStr, -1) {
			if len(match) < 3 {
				continue
			}
			addAlias(match[1], match[2])
		}
		return nil
	})

	return aliases
}

func scanForSharedWorkflows(scenarioPath, scenarioName string) ([]ScenarioDependency, error) {
	var dependencies []ScenarioDependency

	// Look for references to shared workflows in initialization directory
	initPath := filepath.Join(scenarioPath, "initialization")
	if _, err := os.Stat(initPath); os.IsNotExist(err) {
		return dependencies, nil
	}

	// Pattern to match shared workflow references
	sharedPattern := regexp.MustCompile(`initialization/(?:automation/)?(?:n8n|huginn|windmill)/([^/]+\.json)`)

	err := filepath.WalkDir(initPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Check if this is a workflow file that might reference shared workflows
		if strings.HasSuffix(path, ".json") {
			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			matches := sharedPattern.FindAllStringSubmatch(string(content), -1)
			for _, match := range matches {
				if len(match) > 1 {
					dep := ScenarioDependency{
						ID:             uuid.New().String(),
						ScenarioName:   scenarioName,
						DependencyType: "shared_workflow",
						DependencyName: match[1],
						Required:       true, // Shared workflows are typically required
						Purpose:        "Shared workflow dependency",
						AccessMethod:   "workflow_trigger",
						Configuration: map[string]interface{}{
							"found_in_file": strings.TrimPrefix(path, scenarioPath),
							"workflow_type": strings.Split(match[0], "/")[1],
						},
						DiscoveredAt: time.Now(),
						LastVerified: time.Now(),
					}
					dependencies = append(dependencies, dep)
				}
			}
		}

		return nil
	})

	return dependencies, err
}

func loadServiceConfigFromFile(scenarioPath string) (*ServiceConfig, error) {
	serviceConfigPath := filepath.Join(scenarioPath, ".vrooli", "service.json")
	if _, err := os.Stat(serviceConfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("scenario %s not found or missing service.json", filepath.Base(scenarioPath))
	}

	configData, err := os.ReadFile(serviceConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read service.json: %w", err)
	}

	var serviceConfig ServiceConfig
	if err := json.Unmarshal(configData, &serviceConfig); err != nil {
		return nil, fmt.Errorf("failed to parse service.json: %w", err)
	}

	return &serviceConfig, nil
}

func collectAnalysisMetrics() (gin.H, error) {
	metrics := gin.H{
		"scenarios_found":     0,
		"resources_available": 0,
		"database_status":     "unknown",
		"last_analysis":       nil,
	}

	if db == nil {
		return metrics, fmt.Errorf("database connection not initialized")
	}

	if err := db.Ping(); err != nil {
		metrics["database_status"] = "unreachable"
		return metrics, err
	}

	metrics["database_status"] = "connected"

	var scenarioCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM scenario_metadata").Scan(&scenarioCount); err != nil {
		metrics["database_status"] = "error"
		return metrics, err
	}
	metrics["scenarios_found"] = scenarioCount

	var resourceCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM scenario_dependencies WHERE dependency_type = 'resource'").Scan(&resourceCount); err != nil {
		metrics["database_status"] = "error"
		return metrics, err
	}
	metrics["resources_available"] = resourceCount

	var lastAnalysis sql.NullTime
	if err := db.QueryRow("SELECT MAX(last_scanned) FROM scenario_metadata").Scan(&lastAnalysis); err == nil && lastAnalysis.Valid {
		metrics["last_analysis"] = lastAnalysis.Time.UTC().Format(time.RFC3339)
	}

	return metrics, nil
}

func resolvedResourceMap(cfg *ServiceConfig) map[string]Resource {
	if cfg.Dependencies.Resources != nil && len(cfg.Dependencies.Resources) > 0 {
		return cfg.Dependencies.Resources
	}
	if cfg.Resources == nil {
		return map[string]Resource{}
	}
	return cfg.Resources
}

func extractDeclaredResources(scenarioName string, cfg *ServiceConfig) []ScenarioDependency {
	resources := resolvedResourceMap(cfg)
	declared := make([]ScenarioDependency, 0, len(resources))
	for resourceName, resource := range resources {
		dep := ScenarioDependency{
			ID:             uuid.New().String(),
			ScenarioName:   scenarioName,
			DependencyType: "resource",
			DependencyName: resourceName,
			Required:       resource.Required,
			Purpose:        resource.Purpose,
			AccessMethod:   fmt.Sprintf("resource-%s", resourceName),
			Configuration: map[string]interface{}{
				"type":           resource.Type,
				"enabled":        resource.Enabled,
				"initialization": resource.Initialization,
				"models":         resource.Models,
				"source":         "declared",
			},
			DiscoveredAt: time.Now(),
			LastVerified: time.Now(),
		}
		declared = append(declared, dep)
	}
	sort.Slice(declared, func(i, j int) bool {
		return declared[i].DependencyName < declared[j].DependencyName
	})
	return declared
}

func normalizeScenarioSpecs(specs map[string]ScenarioDependencySpec) map[string]ScenarioDependencySpec {
	if specs == nil {
		return map[string]ScenarioDependencySpec{}
	}
	return specs
}

func convertDeclaredScenariosToDependencies(scenarioName string, specs map[string]ScenarioDependencySpec) []ScenarioDependency {
	declared := make([]ScenarioDependency, 0, len(specs))
	for depName, spec := range specs {
		dep := ScenarioDependency{
			ID:             uuid.New().String(),
			ScenarioName:   scenarioName,
			DependencyType: "scenario",
			DependencyName: depName,
			Required:       spec.Required,
			Purpose:        spec.Description,
			AccessMethod:   "declared",
			Configuration: map[string]interface{}{
				"source":        "declared",
				"version":       spec.Version,
				"version_range": spec.VersionRange,
			},
			DiscoveredAt: time.Now(),
			LastVerified: time.Now(),
		}
		declared = append(declared, dep)
	}
	return declared
}

func buildResourceDiff(declared map[string]Resource, detected []ScenarioDependency) DependencyDiff {
	declaredSet := map[string]Resource{}
	for name, cfg := range declared {
		declaredSet[name] = cfg
	}
	detectedSet := map[string]ScenarioDependency{}
	for _, dep := range detected {
		detectedSet[dep.DependencyName] = dep
	}

	missing := []DependencyDrift{}
	for name, dep := range detectedSet {
		if _, ok := declaredSet[name]; !ok {
			missing = append(missing, DependencyDrift{
				Name:    name,
				Details: dep.Configuration,
			})
		}
	}

	extra := []DependencyDrift{}
	for name, cfg := range declaredSet {
		if _, ok := detectedSet[name]; !ok {
			extra = append(extra, DependencyDrift{
				Name: name,
				Details: map[string]interface{}{
					"type":     cfg.Type,
					"required": cfg.Required,
				},
			})
		}
	}

	sort.Slice(missing, func(i, j int) bool { return missing[i].Name < missing[j].Name })
	sort.Slice(extra, func(i, j int) bool { return extra[i].Name < extra[j].Name })

	return DependencyDiff{Missing: missing, Extra: extra}
}

func buildScenarioDiff(declared map[string]ScenarioDependencySpec, detected []ScenarioDependency) DependencyDiff {
	declaredSet := map[string]ScenarioDependencySpec{}
	for name, spec := range declared {
		declaredSet[name] = spec
	}
	detectedSet := map[string]ScenarioDependency{}
	for _, dep := range detected {
		detectedSet[dep.DependencyName] = dep
	}

	missing := []DependencyDrift{}
	for name, dep := range detectedSet {
		if _, ok := declaredSet[name]; !ok {
			missing = append(missing, DependencyDrift{
				Name:    name,
				Details: dep.Configuration,
			})
		}
	}

	extra := []DependencyDrift{}
	for name, spec := range declaredSet {
		if _, ok := detectedSet[name]; !ok {
			extra = append(extra, DependencyDrift{
				Name: name,
				Details: map[string]interface{}{
					"required": spec.Required,
					"version":  spec.Version,
				},
			})
		}
	}

	sort.Slice(missing, func(i, j int) bool { return missing[i].Name < missing[j].Name })
	sort.Slice(extra, func(i, j int) bool { return extra[i].Name < extra[j].Name })

	return DependencyDiff{Missing: missing, Extra: extra}
}

func updateScenarioMetadata(name string, cfg *ServiceConfig, scenarioPath string) error {
	if db == nil {
		return nil
	}

	serviceConfigJSON, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO scenario_metadata (scenario_name, display_name, description, tags, service_config, file_path, last_scanned)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (scenario_name) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			description = EXCLUDED.description,
			tags = EXCLUDED.tags,
			service_config = EXCLUDED.service_config,
			file_path = EXCLUDED.file_path,
			last_scanned = EXCLUDED.last_scanned,
			updated_at = NOW();
	`

	_, err = db.Exec(query,
		name,
		cfg.Service.DisplayName,
		cfg.Service.Description,
		pqArray(cfg.Service.Tags),
		serviceConfigJSON,
		scenarioPath,
		time.Now(),
	)
	return err
}

func pqArray(values []string) interface{} {
	if len(values) == 0 {
		return pq.StringArray([]string{})
	}
	return pq.StringArray(values)
}

func scanForResourceUsage(scenarioPath, scenarioName string) ([]ScenarioDependency, error) {
	results := map[string]ScenarioDependency{}
	ensureDependencyCatalogs()

	relevantExts := []string{".go", ".js", ".ts", ".tsx", ".sh", ".py", ".md", ".json", ".yml", ".yaml"}

	recordDetection := func(name, method, pattern, file string, resourceType string) {
		canonical := normalizeName(name)
		if canonical == "" || !isKnownResource(canonical) {
			return
		}
		existing, ok := results[canonical]
		if !ok {
			existing = ScenarioDependency{
				ID:             uuid.New().String(),
				ScenarioName:   scenarioName,
				DependencyType: "resource",
				DependencyName: canonical,
				Required:       true,
				Purpose:        "Detected via static analysis",
				AccessMethod:   method,
				Configuration:  map[string]interface{}{"source": "detected"},
				DiscoveredAt:   time.Now(),
				LastVerified:   time.Now(),
			}
		}

		if existing.Configuration == nil {
			existing.Configuration = map[string]interface{}{}
		}
		existing.Configuration["resource_type"] = resourceType
		match := map[string]interface{}{
			"pattern": pattern,
			"method":  method,
			"file":    file,
		}
		matches, _ := existing.Configuration["matches"].([]map[string]interface{})
		matches = append(matches, match)
		existing.Configuration["matches"] = matches
		results[canonical] = existing
	}

	err := filepath.WalkDir(scenarioPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != scenarioPath && shouldSkipDirectoryEntry(d) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !contains(relevantExts, ext) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		contentStr := string(content)
		relPath, relErr := filepath.Rel(scenarioPath, path)
		if relErr != nil {
			relPath = path
		}
		if shouldIgnoreDetectionFile(relPath) {
			return nil
		}

		cmdMatches := resourceCommandPattern.FindAllStringSubmatch(contentStr, -1)
		for _, match := range cmdMatches {
			if len(match) > 1 {
				if !isAllowedResourceCLIPath(relPath) {
					continue
				}
				resourceName := normalizeName(match[1])
				recordDetection(resourceName, "resource_cli", "resource-cli", relPath, resourceName)
			}
		}

		for _, heuristic := range resourceHeuristicCatalog {
			for _, pattern := range heuristic.Patterns {
				if pattern.MatchString(contentStr) {
					recordDetection(heuristic.Name, "heuristic", pattern.String(), relPath, heuristic.Type)
					break
				}
			}
		}

		return nil
	})

	augmentResourceDetectionsWithInitialization(results, scenarioPath, scenarioName)

	deps := make([]ScenarioDependency, 0, len(results))
	for _, dep := range results {
		deps = append(deps, dep)
	}
	return deps, err
}

func augmentResourceDetectionsWithInitialization(results map[string]ScenarioDependency, scenarioPath, scenarioName string) {
	cfg, err := loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		return
	}

	resources := resolvedResourceMap(cfg)
	for resourceName, resource := range resources {
		if len(resource.Initialization) == 0 {
			continue
		}
		canonical := normalizeName(resourceName)
		if canonical == "" || !isKnownResource(canonical) {
			continue
		}

		files := extractInitializationFiles(resource.Initialization)
		entry, exists := results[canonical]
		if !exists {
			entry = ScenarioDependency{
				ID:             uuid.New().String(),
				ScenarioName:   scenarioName,
				DependencyType: "resource",
				DependencyName: canonical,
				Required:       true,
				Purpose:        "Initialization data references this resource",
				AccessMethod:   "initialization",
				Configuration:  map[string]interface{}{},
				DiscoveredAt:   time.Now(),
				LastVerified:   time.Now(),
			}
		}

		if entry.Configuration == nil {
			entry.Configuration = map[string]interface{}{}
		}
		entry.Configuration["initialization_detected"] = true
		if len(files) > 0 {
			entry.Configuration["initialization_files"] = mergeInitializationFiles(entry.Configuration["initialization_files"], files)
		}
		results[canonical] = entry
	}
}

func extractInitializationFiles(entries []map[string]interface{}) []string {
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		if file, ok := entry["file"].(string); ok && file != "" {
			files = append(files, file)
		}
	}
	return files
}

func mergeInitializationFiles(existing interface{}, additions []string) []string {
	if len(additions) == 0 {
		return toStringSlice(existing)
	}
	set := map[string]struct{}{}
	merged := make([]string, 0)
	for _, item := range toStringSlice(existing) {
		if _, ok := set[item]; ok {
			continue
		}
		set[item] = struct{}{}
		merged = append(merged, item)
	}
	for _, add := range additions {
		if add == "" {
			continue
		}
		if _, ok := set[add]; ok {
			continue
		}
		set[add] = struct{}{}
		merged = append(merged, add)
	}
	return merged
}

func toStringSlice(value interface{}) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []interface{}:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	default:
		return nil
	}
}

func orderedMapFromStruct(value interface{}) *orderedmap.OrderedMap {
	ordered := orderedmap.New()
	if value == nil {
		return ordered
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return ordered
	}
	if err := json.Unmarshal(payload, ordered); err != nil {
		return orderedmap.New()
	}
	return ordered
}

func storeDependencies(analysis *DependencyAnalysisResponse, extras []ScenarioDependency) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing dependencies for this scenario
	_, err = tx.Exec("DELETE FROM scenario_dependencies WHERE scenario_name = $1", analysis.Scenario)
	if err != nil {
		return err
	}

	// Insert all dependencies (declared + detected)
	allDeps := make([]ScenarioDependency, 0, len(analysis.Resources)+len(analysis.DetectedResources)+len(analysis.Scenarios)+len(analysis.SharedWorkflows)+len(extras))
	allDeps = append(allDeps, analysis.Resources...)
	allDeps = append(allDeps, analysis.DetectedResources...)
	allDeps = append(allDeps, analysis.Scenarios...)
	allDeps = append(allDeps, analysis.SharedWorkflows...)
	allDeps = append(allDeps, extras...)

	for _, dep := range allDeps {
		configJSON, _ := json.Marshal(dep.Configuration)

		_, err = tx.Exec(`
			INSERT INTO scenario_dependencies 
			(scenario_name, dependency_type, dependency_name, required, purpose, access_method, configuration, discovered_at, last_verified)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			dep.ScenarioName, dep.DependencyType, dep.DependencyName, dep.Required,
			dep.Purpose, dep.AccessMethod, configJSON, dep.DiscoveredAt, dep.LastVerified)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	cleanupInvalidScenarioDependencies()
	return nil
}

func cleanupInvalidScenarioDependencies() {
	if db == nil {
		return
	}
	ensureDependencyCatalogs()
	dependencyCatalogMu.RLock()
	defer dependencyCatalogMu.RUnlock()
	if len(knownScenarioNames) == 0 {
		return
	}
	rows, err := db.Query(`
		SELECT DISTINCT dependency_name
		FROM scenario_dependencies
		WHERE dependency_type = 'scenario'`)
	if err != nil {
		log.Printf("Warning: unable to query scenario dependencies for cleanup: %v", err)
		return
	}
	defer rows.Close()

	toDelete := make([]string, 0)
	for rows.Next() {
		var depName string
		if err := rows.Scan(&depName); err != nil {
			continue
		}
		if _, ok := knownScenarioNames[normalizeName(depName)]; !ok {
			toDelete = append(toDelete, depName)
		}
	}

	for _, depName := range toDelete {
		if _, err := db.Exec(`
			DELETE FROM scenario_dependencies
			WHERE dependency_type = 'scenario' AND dependency_name = $1`, depName); err != nil {
			log.Printf("Warning: failed to delete orphaned dependency %s: %v", depName, err)
		}
	}
}

func analyzeAllScenarios() (map[string]*DependencyAnalysisResponse, error) {
	results := make(map[string]*DependencyAnalysisResponse)

	scenariosDir := loadConfig().ScenariosDir
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenarios directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		scenarioName := entry.Name()

		// Check if it has a service.json (valid scenario)
		serviceConfigPath := filepath.Join(scenariosDir, scenarioName, ".vrooli", "service.json")
		if _, err := os.Stat(serviceConfigPath); os.IsNotExist(err) {
			continue
		}

		analysis, err := analyzeScenario(scenarioName)
		if err != nil {
			log.Printf("Warning: failed to analyze scenario %s: %v", scenarioName, err)
			continue
		}

		results[scenarioName] = analysis
	}

	return results, nil
}

func listScenarioNames() ([]string, error) {
	scenariosDir := loadConfig().ScenariosDir
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		return nil, err
	}
	names := []string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		serviceConfigPath := filepath.Join(scenariosDir, entry.Name(), ".vrooli", "service.json")
		if _, err := os.Stat(serviceConfigPath); os.IsNotExist(err) {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

func generateDependencyGraph(graphType string) (*DependencyGraph, error) {
	nodes := []GraphNode{}
	edges := []GraphEdge{}

	// Query all dependencies from database
	rows, err := db.Query(`
		SELECT scenario_name, dependency_type, dependency_name, required, purpose, access_method, configuration, discovered_at, last_verified
		FROM scenario_dependencies
		ORDER BY scenario_name, dependency_type, dependency_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodeSet := make(map[string]bool)

	for rows.Next() {
		var scenarioName, depType, depName, purpose, accessMethod string
		var required bool
		var configJSON []byte
		var discoveredAt, lastVerified time.Time

		err := rows.Scan(&scenarioName, &depType, &depName, &required, &purpose, &accessMethod, &configJSON, &discoveredAt, &lastVerified)
		if err != nil {
			continue
		}

		var configuration map[string]interface{}
		if len(configJSON) > 0 {
			if err := json.Unmarshal(configJSON, &configuration); err != nil {
				configuration = map[string]interface{}{}
			}
		}

		// Filter by graph type
		if graphType == "resource" && depType != "resource" {
			continue
		}
		if graphType == "scenario" && depType == "resource" {
			continue
		}
		if depType == "scenario" && !isKnownScenario(depName) {
			continue
		}

		// Add nodes if not already present
		if !nodeSet[scenarioName] {
			nodes = append(nodes, GraphNode{
				ID:    scenarioName,
				Label: scenarioName,
				Type:  "scenario",
				Group: "scenarios",
				Metadata: map[string]interface{}{
					"node_type": "scenario",
				},
			})
			nodeSet[scenarioName] = true
		}

		if !nodeSet[depName] {
			nodeGroup := "resources"
			nodeType := "resource"
			if depType == "scenario" {
				nodeGroup = "scenarios"
				nodeType = "scenario"
			} else if depType == "shared_workflow" {
				nodeGroup = "workflows"
				nodeType = "workflow"
			}

			nodes = append(nodes, GraphNode{
				ID:    depName,
				Label: depName,
				Type:  nodeType,
				Group: nodeGroup,
				Metadata: map[string]interface{}{
					"node_type": depType,
				},
			})
			nodeSet[depName] = true
		}

		// Add edge
		weight := 1.0
		if required {
			weight = 2.0
		}

		edges = append(edges, GraphEdge{
			Source:   scenarioName,
			Target:   depName,
			Label:    depType,
			Type:     depType,
			Required: required,
			Weight:   weight,
			Metadata: map[string]interface{}{
				"purpose":       purpose,
				"access_method": accessMethod,
				"configuration": configuration,
				"discovered_at": discoveredAt,
				"last_verified": lastVerified,
			},
		})
	}

	graph := &DependencyGraph{
		ID:    uuid.New().String(),
		Type:  graphType,
		Nodes: nodes,
		Edges: edges,
		Metadata: map[string]interface{}{
			"total_nodes":      len(nodes),
			"total_edges":      len(edges),
			"generated_at":     time.Now(),
			"complexity_score": calculateComplexityScore(nodes, edges),
		},
	}

	return graph, nil
}

func calculateComplexityScore(nodes []GraphNode, edges []GraphEdge) float64 {
	if len(nodes) == 0 {
		return 0.0
	}

	// Simple complexity score based on edge-to-node ratio
	ratio := float64(len(edges)) / float64(len(nodes))

	// Normalize to 0-1 scale (assuming max ratio of 5 = complex system)
	score := ratio / 5.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

func analyzeProposedScenario(req ProposedScenarioRequest) (map[string]interface{}, error) {
	predictedResources := []map[string]interface{}{}
	similarPatterns := []map[string]interface{}{}
	recommendations := []map[string]interface{}{}

	// Simple heuristic predictions based on requirements
	for _, reqResource := range req.Requirements {
		predictedResources = append(predictedResources, map[string]interface{}{
			"resource_name": reqResource,
			"confidence":    0.9,
			"reasoning":     "Explicitly mentioned in requirements",
		})
	}

	// Use Claude Code for intelligent analysis of the scenario description
	claudeAnalysis, err := analyzeWithClaudeCode(req.Name, req.Description)
	if err != nil {
		log.Printf("Claude Code analysis failed: %v", err)
	} else {
		// Merge Claude Code predictions with our results
		for _, prediction := range claudeAnalysis.PredictedResources {
			predictedResources = append(predictedResources, prediction)
		}
		for _, recommendation := range claudeAnalysis.Recommendations {
			recommendations = append(recommendations, recommendation)
		}
	}

	// Use Qdrant for semantic similarity matching
	qdrantMatches, err := findSimilarScenariosQdrant(req.Description, req.SimilarScenarios)
	if err != nil {
		log.Printf("Qdrant similarity matching failed: %v", err)
	} else {
		similarPatterns = qdrantMatches
	}

	// Add common patterns based on description keywords (fallback heuristics)
	description := strings.ToLower(req.Description)
	heuristicResources := getHeuristicPredictions(description)
	predictedResources = append(predictedResources, heuristicResources...)

	// Calculate confidence scores
	resourceConfidence := calculateResourceConfidence(predictedResources)
	scenarioConfidence := calculateScenarioConfidence(similarPatterns)

	return map[string]interface{}{
		"predicted_resources": deduplicateResources(predictedResources),
		"similar_patterns":    similarPatterns,
		"recommendations":     recommendations,
		"confidence_scores": map[string]float64{
			"resource": resourceConfidence,
			"scenario": scenarioConfidence,
		},
	}, nil
}

// HTTP Handlers
func healthHandler(c *gin.Context) {
	// Check database connectivity
	dbConnected := false
	var dbLatencyMs float64
	if db != nil {
		start := time.Now()
		err := db.Ping()
		dbLatencyMs = float64(time.Since(start).Milliseconds())
		dbConnected = err == nil
	}

	status := "healthy"
	if !dbConnected {
		status = "degraded"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    status,
		"timestamp": time.Now(),
		"service":   "scenario-dependency-analyzer",
		"readiness": dbConnected,
		"dependencies": gin.H{
			"database": gin.H{
				"connected":  dbConnected,
				"latency_ms": dbLatencyMs,
				"error":      nil,
			},
		},
	})
}

func analyzeScenarioHandler(c *gin.Context) {
	scenarioName := c.Param("scenario")

	if scenarioName == "all" {
		results, err := analyzeAllScenarios()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, results)
		return
	}

	result, err := analyzeScenario(scenarioName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func getDependenciesHandler(c *gin.Context) {
	scenarioName := c.Param("scenario")

	rows, err := db.Query(`
		SELECT id, scenario_name, dependency_type, dependency_name, required, purpose, access_method, configuration, discovered_at, last_verified
		FROM scenario_dependencies
		WHERE scenario_name = $1
		ORDER BY dependency_type, dependency_name`, scenarioName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var dependencies []ScenarioDependency
	for rows.Next() {
		var dep ScenarioDependency
		var configJSON string

		err := rows.Scan(&dep.ID, &dep.ScenarioName, &dep.DependencyType, &dep.DependencyName,
			&dep.Required, &dep.Purpose, &dep.AccessMethod, &configJSON, &dep.DiscoveredAt, &dep.LastVerified)
		if err != nil {
			continue
		}

		json.Unmarshal([]byte(configJSON), &dep.Configuration)
		dependencies = append(dependencies, dep)
	}

	// Group by type
	response := map[string][]ScenarioDependency{
		"resources":        {},
		"scenarios":        {},
		"shared_workflows": {},
	}

	for _, dep := range dependencies {
		switch dep.DependencyType {
		case "resource":
			response["resources"] = append(response["resources"], dep)
		case "scenario":
			response["scenarios"] = append(response["scenarios"], dep)
		case "shared_workflow":
			response["shared_workflows"] = append(response["shared_workflows"], dep)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"scenario":         scenarioName,
		"resources":        response["resources"],
		"scenarios":        response["scenarios"],
		"shared_workflows": response["shared_workflows"],
		"transitive_depth": 0,
	})
}

func getGraphHandler(c *gin.Context) {
	graphType := c.Param("type")

	if !contains([]string{"resource", "scenario", "combined"}, graphType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid graph type"})
		return
	}

	graph, err := generateDependencyGraph(graphType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, graph)
}

func analyzeProposedHandler(c *gin.Context) {
	var req ProposedScenarioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := analyzeProposedScenario(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func optimizeHandler(c *gin.Context) {
	var req OptimizationRequest
	if c.Request.Body != nil {
		if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	scenario := strings.TrimSpace(req.Scenario)
	if scenario == "" {
		scenario = "all"
	}
	var targets []string
	if scenario == "all" {
		names, err := listScenarioNames()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		targets = names
	} else {
		if !isKnownScenario(scenario) {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("scenario %s not found", scenario)})
			return
		}
		targets = []string{scenario}
	}
	results := make(map[string]*OptimizationResult, len(targets))
	for _, target := range targets {
		result, err := runScenarioOptimization(target, req)
		if err != nil {
			results[target] = &OptimizationResult{
				Scenario:          target,
				Recommendations:   nil,
				Summary:           OptimizationSummary{},
				Applied:           false,
				AnalysisTimestamp: time.Now().UTC(),
				Error:             err.Error(),
			}
			continue
		}
		results[target] = result
	}
	c.JSON(http.StatusOK, gin.H{
		"results":      results,
		"generated_at": time.Now().UTC(),
	})
}

func loadScenarioMetadataMap() (map[string]ScenarioSummary, error) {
	results := map[string]ScenarioSummary{}
	if db == nil {
		return results, nil
	}
	rows, err := db.Query("SELECT scenario_name, display_name, description, tags, last_scanned FROM scenario_metadata")
	if err != nil {
		return results, err
	}
	defer rows.Close()

	for rows.Next() {
		var summary ScenarioSummary
		var tags pq.StringArray
		var lastScanned sql.NullTime
		if err := rows.Scan(&summary.Name, &summary.DisplayName, &summary.Description, &tags, &lastScanned); err != nil {
			continue
		}
		summary.Tags = []string(tags)
		if lastScanned.Valid {
			summary.LastScanned = &lastScanned.Time
		}
		results[summary.Name] = summary
	}

	return results, nil
}

func loadStoredDependencies(scenarioName string) (map[string][]ScenarioDependency, error) {
	if db == nil {
		return map[string][]ScenarioDependency{
			"resources":        []ScenarioDependency{},
			"scenarios":        []ScenarioDependency{},
			"shared_workflows": []ScenarioDependency{},
		}, nil
	}
	rows, err := db.Query(`
		SELECT scenario_name, dependency_type, dependency_name, required, purpose, access_method, configuration, discovered_at, last_verified
		FROM scenario_dependencies
		WHERE scenario_name = $1
		ORDER BY dependency_type, dependency_name`, scenarioName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string][]ScenarioDependency{
		"resources":        []ScenarioDependency{},
		"scenarios":        []ScenarioDependency{},
		"shared_workflows": []ScenarioDependency{},
	}

	for rows.Next() {
		var dep ScenarioDependency
		var configJSON []byte
		if err := rows.Scan(&dep.ScenarioName, &dep.DependencyType, &dep.DependencyName, &dep.Required, &dep.Purpose, &dep.AccessMethod, &configJSON, &dep.DiscoveredAt, &dep.LastVerified); err != nil {
			continue
		}
		if len(configJSON) > 0 {
			_ = json.Unmarshal(configJSON, &dep.Configuration)
		}
		result[dep.DependencyType] = append(result[dep.DependencyType], dep)
	}

	return result, nil
}

func loadOptimizationRecommendations(scenarioName string) ([]OptimizationRecommendation, error) {
	if db == nil {
		return nil, nil
	}
	rows, err := db.Query(`
		SELECT id, scenario_name, recommendation_type, title, description, current_state, recommended_state, estimated_impact, confidence_score, priority, status, created_at
		FROM optimization_recommendations
		WHERE scenario_name = $1
		ORDER BY created_at DESC`, scenarioName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	recommendations := []OptimizationRecommendation{}
	for rows.Next() {
		var rec OptimizationRecommendation
		var currentJSON, recommendedJSON, estimatedJSON []byte
		if err := rows.Scan(&rec.ID, &rec.ScenarioName, &rec.RecommendationType, &rec.Title, &rec.Description, &currentJSON, &recommendedJSON, &estimatedJSON, &rec.ConfidenceScore, &rec.Priority, &rec.Status, &rec.CreatedAt); err != nil {
			continue
		}
		_ = json.Unmarshal(currentJSON, &rec.CurrentState)
		_ = json.Unmarshal(recommendedJSON, &rec.RecommendedState)
		_ = json.Unmarshal(estimatedJSON, &rec.EstimatedImpact)
		recommendations = append(recommendations, rec)
	}
	return recommendations, nil
}

func filterDetectedDependencies(deps []ScenarioDependency) []ScenarioDependency {
	filtered := []ScenarioDependency{}
	for _, dep := range deps {
		source := ""
		if dep.Configuration != nil {
			if val, ok := dep.Configuration["source"].(string); ok {
				source = val
			}
		}
		if source == "declared" {
			continue
		}
		filtered = append(filtered, dep)
	}
	return filtered
}

func applyDetectedDiffs(scenarioName string, analysis *DependencyAnalysisResponse, applyResources, applyScenarios bool) (map[string]interface{}, error) {
	updates := map[string]interface{}{}
	scenarioPath := filepath.Join(loadConfig().ScenariosDir, scenarioName)
	cfg, err := loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		return nil, err
	}
	if os.Getenv("SCENARIO_ANALYZER_TRACE") == "1" {
		log.Printf("[dependency-apply] %s declared resources=%d scenarios=%d", scenarioName, len(cfg.Dependencies.Resources), len(cfg.Dependencies.Scenarios))
	}
	if applyDiffsHook != nil {
		applyDiffsHook(scenarioName, cfg)
	}

	rawConfig, err := loadRawServiceConfigMap(scenarioPath)
	if err != nil {
		return nil, err
	}
	rawDependencies := ensureOrderedMap(rawConfig, "dependencies")
	rawResources := ensureOrderedMap(rawDependencies, "resources")
	rawScenarios := ensureOrderedMap(rawDependencies, "scenarios")
	rawDependencies.Set("resources", rawResources)
	rawDependencies.Set("scenarios", rawScenarios)
	if os.Getenv("SCENARIO_ANALYZER_TRACE") == "1" {
		log.Printf("[dependency-apply] raw resources keys=%d", len(rawResources.Keys()))
	}
	if len(rawResources.Keys()) == 0 && len(cfg.Dependencies.Resources) > 0 {
		rawResources = orderedMapFromStruct(cfg.Dependencies.Resources)
		rawDependencies.Set("resources", rawResources)
	}
	if len(rawScenarios.Keys()) == 0 && len(cfg.Dependencies.Scenarios) > 0 {
		rawScenarios = orderedMapFromStruct(cfg.Dependencies.Scenarios)
		rawDependencies.Set("scenarios", rawScenarios)
	}

	resourcesAdded := []string{}
	if applyResources {
		missing := map[string]struct{}{}
		for _, drift := range analysis.ResourceDiff.Missing {
			missing[drift.Name] = struct{}{}
		}
		if len(missing) > 0 {
			if cfg.Dependencies.Resources == nil {
				cfg.Dependencies.Resources = map[string]Resource{}
			}
			for _, dep := range analysis.DetectedResources {
				if _, ok := missing[dep.DependencyName]; !ok {
					continue
				}
				if _, exists := cfg.Dependencies.Resources[dep.DependencyName]; exists {
					continue
				}
				typeHint := "custom"
				if dep.Configuration != nil {
					if val, ok := dep.Configuration["resource_type"].(string); ok && val != "" {
						typeHint = val
					}
				}
				description := fmt.Sprintf("Auto-detected via analyzer (%s)", dep.AccessMethod)
				cfg.Dependencies.Resources[dep.DependencyName] = Resource{
					Type:     typeHint,
					Enabled:  true,
					Required: true,
					Purpose:  description,
				}
				resourceEntry := orderedmap.New()
				resourceEntry.Set("type", typeHint)
				resourceEntry.Set("enabled", true)
				resourceEntry.Set("required", true)
				resourceEntry.Set("description", description)
				rawResources.Set(dep.DependencyName, resourceEntry)
				resourcesAdded = append(resourcesAdded, dep.DependencyName)
			}
		}
	}

	scenariosAdded := []string{}
	if applyScenarios {
		missing := map[string]struct{}{}
		for _, drift := range analysis.ScenarioDiff.Missing {
			missing[drift.Name] = struct{}{}
		}
		if len(missing) > 0 {
			if cfg.Dependencies.Scenarios == nil {
				cfg.Dependencies.Scenarios = map[string]ScenarioDependencySpec{}
			}
			for _, dep := range analysis.Scenarios {
				if _, ok := missing[dep.DependencyName]; !ok {
					continue
				}
				if _, exists := cfg.Dependencies.Scenarios[dep.DependencyName]; exists {
					continue
				}
				description := fmt.Sprintf("Auto-detected dependency via %s", dep.AccessMethod)
				version, versionRange := resolveScenarioVersionSpec(dep.DependencyName)
				cfg.Dependencies.Scenarios[dep.DependencyName] = ScenarioDependencySpec{
					Required:     true,
					Version:      version,
					VersionRange: versionRange,
					Description:  description,
				}
				scenarioEntry := orderedmap.New()
				scenarioEntry.Set("required", true)
				scenarioEntry.Set("version", version)
				scenarioEntry.Set("versionRange", versionRange)
				scenarioEntry.Set("description", description)
				rawScenarios.Set(dep.DependencyName, scenarioEntry)
				scenariosAdded = append(scenariosAdded, dep.DependencyName)
			}
		}
	}

	changed := len(resourcesAdded) > 0 || len(scenariosAdded) > 0
	if changed {
		reordered := reorderTopLevelKeys(rawConfig)
		if err := writeRawServiceConfigMap(scenarioPath, reordered); err != nil {
			return nil, err
		}
		if err := updateScenarioMetadata(scenarioName, cfg, scenarioPath); err != nil {
			return nil, err
		}
		refreshDependencyCatalogs()
	}

	updates["changed"] = changed
	updates["resources_added"] = resourcesAdded
	updates["scenarios_added"] = scenariosAdded
	return updates, nil
}

func loadRawServiceConfigMap(scenarioPath string) (*orderedmap.OrderedMap, error) {
	serviceConfigPath := filepath.Join(scenarioPath, ".vrooli", "service.json")
	data, err := os.ReadFile(serviceConfigPath)
	if err != nil {
		return nil, err
	}
	raw := orderedmap.New()
	if err := json.Unmarshal(data, raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func writeRawServiceConfigMap(scenarioPath string, cfg *orderedmap.OrderedMap) error {
	serviceConfigPath := filepath.Join(scenarioPath, ".vrooli", "service.json")
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cfg); err != nil {
		return err
	}
	payload := bytes.TrimRight(buffer.Bytes(), "\n")
	payload = bytes.ReplaceAll(payload, []byte(`\u003c`), []byte("<"))
	payload = bytes.ReplaceAll(payload, []byte(`\u003e`), []byte(">"))
	payload = bytes.ReplaceAll(payload, []byte(`\u0026`), []byte("&"))
	return os.WriteFile(serviceConfigPath, payload, 0644)
}

func persistOptimizationRecommendations(scenarioName string, recs []OptimizationRecommendation) error {
	if db == nil {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM optimization_recommendations WHERE scenario_name = $1", scenarioName); err != nil {
		return err
	}
	insertStmt := `
		INSERT INTO optimization_recommendations
		(id, scenario_name, recommendation_type, title, description, current_state, recommended_state, estimated_impact, confidence_score, priority, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`
	for _, rec := range recs {
		currentJSON, _ := json.Marshal(rec.CurrentState)
		recommendedJSON, _ := json.Marshal(rec.RecommendedState)
		estimatedJSON, _ := json.Marshal(rec.EstimatedImpact)
		if _, err := tx.Exec(insertStmt,
			rec.ID,
			rec.ScenarioName,
			rec.RecommendationType,
			rec.Title,
			rec.Description,
			currentJSON,
			recommendedJSON,
			estimatedJSON,
			rec.ConfidenceScore,
			rec.Priority,
			rec.Status,
			rec.CreatedAt,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func buildOptimizationSummary(recs []OptimizationRecommendation) OptimizationSummary {
	summary := OptimizationSummary{
		RecommendationCount: len(recs),
		ByType:              map[string]int{},
		PotentialImpact:     map[string]int{},
	}
	for _, rec := range recs {
		summary.ByType[rec.RecommendationType]++
		if strings.EqualFold(rec.Priority, "high") || strings.EqualFold(rec.Priority, "critical") {
			summary.HighPriority++
		}
		if action, ok := rec.RecommendedState["action"].(string); ok {
			summary.PotentialImpact[action]++
		}
	}
	return summary
}

func applyOptimizationRecommendations(scenarioName string, recs []OptimizationRecommendation) (map[string]interface{}, error) {
	if len(recs) == 0 {
		return map[string]interface{}{"changed": false}, nil
	}
	config := loadConfig()
	scenarioPath := filepath.Join(config.ScenariosDir, scenarioName)
	removedResources := []string{}
	scenariosRemoved := []string{}
	for _, rec := range recs {
		action, _ := rec.RecommendedState["action"].(string)
		switch action {
		case "remove_resource":
			resourceName, _ := rec.RecommendedState["resource_name"].(string)
			if resourceName == "" {
				continue
			}
			removed, err := removeResourceFromServiceConfig(scenarioPath, resourceName)
			if err != nil {
				return nil, err
			}
			if removed {
				removedResources = append(removedResources, resourceName)
			}
		case "remove_scenario_dependency":
			depName, _ := rec.RecommendedState["scenario_name"].(string)
			if depName == "" {
				continue
			}
			removed, err := removeScenarioDependencyFromServiceConfig(scenarioPath, depName)
			if err != nil {
				return nil, err
			}
			if removed {
				scenariosRemoved = append(scenariosRemoved, depName)
			}
		}
	}
	changed := len(removedResources) > 0 || len(scenariosRemoved) > 0
	if changed {
		cfg, err := loadServiceConfigFromFile(scenarioPath)
		if err == nil {
			_ = updateScenarioMetadata(scenarioName, cfg, scenarioPath)
		}
		refreshDependencyCatalogs()
	}
	return map[string]interface{}{
		"changed":           changed,
		"resources_removed": removedResources,
		"scenarios_removed": scenariosRemoved,
	}, nil
}

func removeResourceFromServiceConfig(scenarioPath, resourceName string) (bool, error) {
	raw, err := loadRawServiceConfigMap(scenarioPath)
	if err != nil {
		return false, err
	}
	deps := ensureOrderedMap(raw, "dependencies")
	resources := ensureOrderedMap(deps, "resources")
	if _, ok := resources.Get(resourceName); !ok {
		return false, nil
	}
	resources.Delete(resourceName)
	reordered := reorderTopLevelKeys(raw)
	if err := writeRawServiceConfigMap(scenarioPath, reordered); err != nil {
		return false, err
	}
	return true, nil
}

func removeScenarioDependencyFromServiceConfig(scenarioPath, scenarioName string) (bool, error) {
	raw, err := loadRawServiceConfigMap(scenarioPath)
	if err != nil {
		return false, err
	}
	deps := ensureOrderedMap(raw, "dependencies")
	scenarios := ensureOrderedMap(deps, "scenarios")
	if _, ok := scenarios.Get(scenarioName); !ok {
		return false, nil
	}
	scenarios.Delete(scenarioName)
	reordered := reorderTopLevelKeys(raw)
	if err := writeRawServiceConfigMap(scenarioPath, reordered); err != nil {
		return false, err
	}
	return true, nil
}

func ensureOrderedMap(parent *orderedmap.OrderedMap, key string) *orderedmap.OrderedMap {
	if parent == nil {
		return orderedmap.New()
	}
	if val, ok := parent.Get(key); ok {
		switch typed := val.(type) {
		case *orderedmap.OrderedMap:
			return typed
		case orderedmap.OrderedMap:
			converted := orderedmap.New()
			for _, childKey := range typed.Keys() {
				if childVal, ok := typed.Get(childKey); ok {
					converted.Set(childKey, childVal)
				}
			}
			parent.Set(key, converted)
			return converted
		case map[string]interface{}:
			converted := orderedmap.New()
			for k, v := range typed {
				converted.Set(k, v)
			}
			parent.Set(key, converted)
			return converted
		}
	}
	child := orderedmap.New()
	parent.Set(key, child)
	return child
}

func reorderTopLevelKeys(cfg *orderedmap.OrderedMap) *orderedmap.OrderedMap {
	if cfg == nil {
		return orderedmap.New()
	}
	preferred := []string{"$schema", "version", "service", "ports", "lifecycle", "dependencies"}
	reordered := orderedmap.New()
	seen := map[string]struct{}{}
	for _, key := range preferred {
		if val, ok := cfg.Get(key); ok {
			reordered.Set(key, val)
			seen[key] = struct{}{}
		}
	}
	for _, key := range cfg.Keys() {
		if _, ok := seen[key]; ok {
			continue
		}
		if val, ok := cfg.Get(key); ok {
			reordered.Set(key, val)
		}
	}
	return reordered
}

func cloneOrderedMap(src *orderedmap.OrderedMap) *orderedmap.OrderedMap {
	if src == nil {
		return orderedmap.New()
	}
	clone := orderedmap.New()
	for _, key := range src.Keys() {
		if val, ok := src.Get(key); ok {
			clone.Set(key, val)
		}
	}
	return clone
}

func resolveScenarioVersionSpec(dependencyName string) (string, string) {
	scenarioPath := filepath.Join(loadConfig().ScenariosDir, dependencyName)
	cfg, err := loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		return "", ">=0.0.0"
	}
	version := strings.TrimSpace(cfg.Service.Version)
	if version == "" {
		return "", ">=0.0.0"
	}
	return version, fmt.Sprintf(">=%s", version)
}

func listScenariosHandler(c *gin.Context) {
	metadata, err := loadScenarioMetadataMap()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	scenariosDir := loadConfig().ScenariosDir
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	summaries := []ScenarioSummary{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		summary, ok := metadata[name]
		if !ok {
			scenarioPath := filepath.Join(scenariosDir, name)
			cfg, err := loadServiceConfigFromFile(scenarioPath)
			if err != nil {
				continue
			}
			summary = ScenarioSummary{
				Name:        name,
				DisplayName: cfg.Service.DisplayName,
				Description: cfg.Service.Description,
				Tags:        cfg.Service.Tags,
			}
		}
		summaries = append(summaries, summary)
	}

	sort.Slice(summaries, func(i, j int) bool { return summaries[i].Name < summaries[j].Name })
	c.JSON(http.StatusOK, summaries)
}

func getScenarioDetailHandler(c *gin.Context) {
	scenarioName := c.Param("scenario")
	scenarioPath := filepath.Join(loadConfig().ScenariosDir, scenarioName)
	cfg, err := loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	stored, err := loadStoredDependencies(scenarioName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	optRecs, err := loadOptimizationRecommendations(scenarioName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	declaredResources := resolvedResourceMap(cfg)

	declaredScenarios := cfg.Dependencies.Scenarios
	if declaredScenarios == nil {
		declaredScenarios = map[string]ScenarioDependencySpec{}
	}

	resourceDiff := buildResourceDiff(declaredResources, filterDetectedDependencies(stored["resources"]))
	scenarioDiff := buildScenarioDiff(declaredScenarios, filterDetectedDependencies(stored["scenarios"]))

	var lastScanned *time.Time
	if db != nil {
		row := db.QueryRow("SELECT last_scanned FROM scenario_metadata WHERE scenario_name = $1", scenarioName)
		var ts sql.NullTime
		if err := row.Scan(&ts); err == nil && ts.Valid {
			lastScanned = &ts.Time
		}
	}

	detail := ScenarioDetailResponse{
		Scenario:                    scenarioName,
		DisplayName:                 cfg.Service.DisplayName,
		Description:                 cfg.Service.Description,
		LastScanned:                 lastScanned,
		DeclaredResources:           declaredResources,
		DeclaredScenarios:           declaredScenarios,
		StoredDependencies:          stored,
		ResourceDiff:                resourceDiff,
		ScenarioDiff:                scenarioDiff,
		OptimizationRecommendations: optRecs,
	}

	if report := buildDeploymentReport(scenarioName, scenarioPath, loadConfig().ScenariosDir, cfg); report != nil {
		detail.DeploymentReport = report
	}

	c.JSON(http.StatusOK, detail)
}

func getDeploymentReportHandler(c *gin.Context) {
	scenarioName := c.Param("scenario")
	config := loadConfig()
	scenarioPath := filepath.Join(config.ScenariosDir, scenarioName)
	cfg, err := loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	report, err := loadPersistedDeploymentReport(scenarioPath)
	if err != nil {
		report = buildDeploymentReport(scenarioName, scenarioPath, config.ScenariosDir, cfg)
		if report != nil {
			if persistErr := persistDeploymentReport(scenarioPath, report); persistErr != nil {
				log.Printf("Warning: failed to persist deployment report for %s: %v", scenarioName, persistErr)
			}
		}
	}

	if report == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to build deployment report"})
		return
	}

	c.JSON(http.StatusOK, report)
}

func scanScenarioHandler(c *gin.Context) {
	scenarioName := c.Param("scenario")
	var req ScanRequest
	if c.Request.Body != nil {
		if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	analysis, err := analyzeScenario(scenarioName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	applyResources := req.ApplyResources || req.Apply
	applyScenarios := req.ApplyScenarios || req.Apply
	var applySummary map[string]interface{}
	applied := false
	if applyResources || applyScenarios {
		applySummary, err = applyDetectedDiffs(scenarioName, analysis, applyResources, applyScenarios)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "analysis": analysis})
			return
		}
		if changed, ok := applySummary["changed"].(bool); ok && changed {
			applied = true
			analysis, err = analyzeScenario(scenarioName)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis":      analysis,
		"applied":       applied,
		"apply_summary": applySummary,
	})
}

// Integrate with Claude Code resource for intelligent analysis
func analyzeWithClaudeCode(scenarioName, description string) (*ClaudeCodeAnalysis, error) {
	// Create a temporary analysis file
	analysisPrompt := fmt.Sprintf(`
Analyze the following proposed Vrooli scenario and predict its likely dependencies:

Scenario Name: %s
Description: %s

Please identify:
1. Likely resource dependencies (postgres, redis, ollama, n8n, etc.)
2. Similar existing scenarios it might depend on
3. Recommended architecture patterns
4. Potential optimization opportunities

Format your response as structured analysis focusing on technical implementation needs.
`, scenarioName, description)

	// Write prompt to temporary file
	tempFile := "/tmp/claude-analysis-" + uuid.New().String() + ".txt"
	if err := os.WriteFile(tempFile, []byte(analysisPrompt), 0644); err != nil {
		return nil, fmt.Errorf("failed to create analysis prompt file: %w", err)
	}
	defer os.Remove(tempFile)

	// Execute Claude Code analysis
	cmd := exec.Command("resource-claude-code", "analyze", tempFile, "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("claude-code command failed: %w", err)
	}

	// Parse Claude Code response and extract dependency predictions
	analysis := parseClaudeCodeResponse(string(output), description)
	return analysis, nil
}

// Integrate with Qdrant for semantic similarity matching
func findSimilarScenariosQdrant(description string, existingScenarios []string) ([]map[string]interface{}, error) {
	var matches []map[string]interface{}

	// Create embedding for the proposed scenario description
	embeddingCmd := exec.Command("resource-qdrant", "embed", description)
	embeddingOutput, err := embeddingCmd.Output()
	if err != nil {
		return matches, fmt.Errorf("failed to create embedding: %w", err)
	}

	// Search for similar scenarios in the vector database
	searchCmd := exec.Command("resource-qdrant", "search",
		"--collection", "scenario_embeddings",
		"--vector", string(embeddingOutput),
		"--limit", "5",
		"--output", "json")

	searchOutput, err := searchCmd.Output()
	if err != nil {
		// Qdrant search failed - return empty matches rather than error
		// This allows the analysis to continue with other methods
		return matches, nil
	}

	// Parse Qdrant search results
	var searchResults QdrantSearchResults
	if err := json.Unmarshal(searchOutput, &searchResults); err != nil {
		return matches, fmt.Errorf("failed to parse qdrant results: %w", err)
	}

	// Convert to our format
	for _, result := range searchResults.Matches {
		if result.Score > 0.7 { // Only include high-confidence matches
			matches = append(matches, map[string]interface{}{
				"scenario_name": result.ScenarioName,
				"similarity":    result.Score,
				"resources":     result.Resources,
				"description":   result.Description,
			})
		}
	}

	return matches, nil
}

// Helper functions for analysis
func getHeuristicPredictions(description string) []map[string]interface{} {
	var predictions []map[string]interface{}

	heuristics := map[string][]string{
		"postgres": {"data", "database", "store", "persist", "sql", "table"},
		"redis":    {"cache", "session", "temporary", "fast", "memory"},
		"ollama":   {"ai", "llm", "language model", "chat", "generate", "intelligent"},
		"n8n":      {"workflow", "automation", "process", "trigger", "orchestrate"},
		"qdrant":   {"vector", "semantic", "search", "similarity", "embedding"},
		"minio":    {"file", "upload", "storage", "document", "asset", "image"},
	}

	for resource, keywords := range heuristics {
		confidence := 0.0
		matches := 0

		for _, keyword := range keywords {
			if strings.Contains(description, keyword) {
				matches++
				confidence += 0.1
			}
		}

		if confidence > 0 {
			// Normalize confidence based on number of matches
			confidence = math.Min(confidence, 0.8)

			predictions = append(predictions, map[string]interface{}{
				"resource_name": resource,
				"confidence":    confidence,
				"reasoning":     fmt.Sprintf("Heuristic match: %d keywords detected", matches),
			})
		}
	}

	return predictions
}

func deduplicateResources(resources []map[string]interface{}) []map[string]interface{} {
	seen := make(map[string]float64)
	var deduplicated []map[string]interface{}

	for _, resource := range resources {
		name := resource["resource_name"].(string)
		confidence := resource["confidence"].(float64)

		if existingConfidence, exists := seen[name]; !exists || confidence > existingConfidence {
			seen[name] = confidence
		}
	}

	// Rebuild array with highest confidence for each resource
	for _, resource := range resources {
		name := resource["resource_name"].(string)
		confidence := resource["confidence"].(float64)

		if seen[name] == confidence {
			deduplicated = append(deduplicated, resource)
			delete(seen, name) // Prevent duplicates
		}
	}

	return deduplicated
}

func calculateResourceConfidence(predictions []map[string]interface{}) float64 {
	if len(predictions) == 0 {
		return 0.0
	}

	totalConfidence := 0.0
	for _, pred := range predictions {
		totalConfidence += pred["confidence"].(float64)
	}

	return math.Min(totalConfidence/float64(len(predictions)), 1.0)
}

func calculateScenarioConfidence(patterns []map[string]interface{}) float64 {
	if len(patterns) == 0 {
		return 0.0
	}

	totalSimilarity := 0.0
	for _, pattern := range patterns {
		if sim, ok := pattern["similarity"].(float64); ok {
			totalSimilarity += sim
		}
	}

	return math.Min(totalSimilarity/float64(len(patterns)), 1.0)
}

// Data structures for external integrations
type ClaudeCodeAnalysis struct {
	PredictedResources []map[string]interface{} `json:"predicted_resources"`
	Recommendations    []map[string]interface{} `json:"recommendations"`
	ArchitectureNotes  string                   `json:"architecture_notes"`
}

type QdrantSearchResults struct {
	Matches []QdrantMatch `json:"matches"`
}

type QdrantMatch struct {
	ScenarioName string                 `json:"scenario_name"`
	Score        float64                `json:"score"`
	Resources    []string               `json:"resources"`
	Description  string                 `json:"description"`
	Metadata     map[string]interface{} `json:"metadata"`
}

func parseClaudeCodeResponse(response, originalDescription string) *ClaudeCodeAnalysis {
	// Parse Claude Code response and extract structured dependency predictions
	// This is a simplified implementation - in practice, you'd parse the AI response more sophisticatedly

	analysis := &ClaudeCodeAnalysis{
		PredictedResources: []map[string]interface{}{},
		Recommendations:    []map[string]interface{}{},
		ArchitectureNotes:  response,
	}

	// Extract resource mentions from Claude's response
	responseText := strings.ToLower(response)

	resourcePatterns := map[string]float64{
		"postgres":   0.8,
		"postgresql": 0.8,
		"database":   0.7,
		"redis":      0.8,
		"cache":      0.6,
		"ollama":     0.9,
		"llm":        0.7,
		"n8n":        0.8,
		"workflow":   0.6,
		"qdrant":     0.9,
		"vector":     0.7,
		"minio":      0.8,
		"storage":    0.5,
	}

	for pattern, confidence := range resourcePatterns {
		if strings.Contains(responseText, pattern) {
			// Map patterns to actual resource names
			resourceName := mapPatternToResource(pattern)
			if resourceName != "" {
				analysis.PredictedResources = append(analysis.PredictedResources, map[string]interface{}{
					"resource_name": resourceName,
					"confidence":    confidence,
					"reasoning":     fmt.Sprintf("Claude Code analysis mentioned '%s'", pattern),
				})
			}
		}
	}

	return analysis
}

func mapPatternToResource(pattern string) string {
	resourceMap := map[string]string{
		"postgres":   "postgres",
		"postgresql": "postgres",
		"database":   "postgres",
		"redis":      "redis",
		"cache":      "redis",
		"ollama":     "ollama",
		"llm":        "ollama",
		"n8n":        "n8n",
		"workflow":   "n8n",
		"qdrant":     "qdrant",
		"vector":     "qdrant",
		"minio":      "minio",
		"storage":    "minio",
	}

	return resourceMap[pattern]
}

// Utility functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start scenario-dependency-analyzer

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	config := loadConfig()

	// Initialize database
	if err := initDatabase(config.DatabaseURL); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	cleanupInvalidScenarioDependencies()

	// Initialize Gin router
	router := gin.Default()

	// Add CORS support
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	handler := c.Handler(router)

	// Health endpoints
	router.GET("/health", healthHandler)
	router.GET("/api/v1/health/analysis", func(c *gin.Context) {
		// Test analysis capability
		_, err := generateDependencyGraph("combined")
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  "Analysis capability test failed",
			})
			return
		}

		payload := gin.H{
			"status":       "healthy",
			"capabilities": []string{"dependency_analysis", "graph_generation"},
		}

		metrics, metricsErr := collectAnalysisMetrics()
		for k, v := range metrics {
			payload[k] = v
		}

		if metricsErr != nil {
			payload["status"] = "degraded"
			payload["error"] = metricsErr.Error()
		}

		c.JSON(http.StatusOK, payload)
	})

	// API routes
	api := router.Group("/api/v1")
	{
		api.GET("/scenarios", listScenariosHandler)
		api.GET("/scenarios/:scenario", getScenarioDetailHandler)
		api.GET("/scenarios/:scenario/deployment", getDeploymentReportHandler)
		api.GET("/analyze/:scenario", analyzeScenarioHandler)
		api.GET("/scenarios/:scenario/dependencies", getDependenciesHandler)
		api.POST("/scenarios/:scenario/scan", scanScenarioHandler)
		api.GET("/graph/:type", getGraphHandler)
		api.POST("/analyze/proposed", analyzeProposedHandler)
		api.POST("/optimize", optimizeHandler)
	}

	log.Printf("Starting Scenario Dependency Analyzer API on port %s", config.Port)
	log.Printf("Scenarios directory: %s", config.ScenariosDir)

	if err := http.ListenAndServe(":"+config.Port, handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
