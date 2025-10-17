package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/rs/cors"

	_ "github.com/lib/pq"
)

// Domain models
type ServiceConfig struct {
	Schema   string `json:"$schema"`
	Version  string `json:"version"`
	Service  struct {
		Name        string   `json:"name"`
		DisplayName string   `json:"displayName"`
		Description string   `json:"description"`
		Version     string   `json:"version"`
		Tags        []string `json:"tags"`
	} `json:"service"`
	Ports     map[string]interface{} `json:"ports"`
	Resources map[string]Resource    `json:"resources"`
}

type Resource struct {
	Type           string                   `json:"type"`
	Enabled        bool                     `json:"enabled"`
	Required       bool                     `json:"required"`
	Purpose        string                   `json:"purpose"`
	Initialization []map[string]interface{} `json:"initialization,omitempty"`
	Models         []string                 `json:"models,omitempty"`
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
	ScenarioName       string `json:"scenario_name"`
	IncludeTransitive  bool   `json:"include_transitive"`
}

type ProposedScenarioRequest struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Requirements     []string `json:"requirements"`
	SimilarScenarios []string `json:"similar_scenarios,omitempty"`
}

type DependencyAnalysisResponse struct {
	Scenario          string               `json:"scenario"`
	Resources         []ScenarioDependency `json:"resources"`
	Scenarios         []ScenarioDependency `json:"scenarios"`
	SharedWorkflows   []ScenarioDependency `json:"shared_workflows"`
	TransitiveDepth   int                  `json:"transitive_depth"`
}

// Database connection
var db *sql.DB

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
			log.Printf("✅ Database connected successfully on attempt %d", attempt + 1)
			break
		}
		
		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay) * math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))
		
		// Add random jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(rand.Float64() * jitterRange)
		actualDelay := delay + jitter
		
		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt + 1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)
		
		// Provide detailed status every few attempts
		if attempt > 0 && attempt % 3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt + 1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt * 2) * baseDelay)
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
	scenarioPath := filepath.Join(loadConfig().ScenariosDir, scenarioName)
	serviceConfigPath := filepath.Join(scenarioPath, ".vrooli", "service.json")
	
	// Check if scenario exists
	if _, err := os.Stat(serviceConfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("scenario %s not found or missing service.json", scenarioName)
	}
	
	// Parse service.json
	configData, err := os.ReadFile(serviceConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read service.json: %w", err)
	}
	
	var serviceConfig ServiceConfig
	if err := json.Unmarshal(configData, &serviceConfig); err != nil {
		return nil, fmt.Errorf("failed to parse service.json: %w", err)
	}
	
	response := &DependencyAnalysisResponse{
		Scenario:        scenarioName,
		Resources:       []ScenarioDependency{},
		Scenarios:       []ScenarioDependency{},
		SharedWorkflows: []ScenarioDependency{},
		TransitiveDepth: 0,
	}
	
	// Extract resource dependencies
	for resourceName, resource := range serviceConfig.Resources {
		dependency := ScenarioDependency{
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
			},
			DiscoveredAt: time.Now(),
			LastVerified: time.Now(),
		}
		response.Resources = append(response.Resources, dependency)
	}
	
	// Scan for inter-scenario dependencies in code files
	scenarioDeps, err := scanForScenarioDependencies(scenarioPath, scenarioName)
	if err != nil {
		log.Printf("Warning: failed to scan for scenario dependencies: %v", err)
	} else {
		response.Scenarios = append(response.Scenarios, scenarioDeps...)
	}
	
	// Scan for shared workflow usage
	workflowDeps, err := scanForSharedWorkflows(scenarioPath, scenarioName)
	if err != nil {
		log.Printf("Warning: failed to scan for shared workflows: %v", err)
	} else {
		response.SharedWorkflows = append(response.SharedWorkflows, workflowDeps...)
	}
	
	// Store in database
	if err := storeDependencies(response); err != nil {
		log.Printf("Warning: failed to store dependencies in database: %v", err)
	}
	
	return response, nil
}

func scanForScenarioDependencies(scenarioPath, scenarioName string) ([]ScenarioDependency, error) {
	var dependencies []ScenarioDependency
	
	// Pattern to match vrooli scenario commands
	scenarioPattern := regexp.MustCompile(`vrooli\s+scenario\s+(?:run|test|status)\s+([a-z0-9-]+)`)
	
	// Pattern to match direct CLI calls to other scenarios
	cliPattern := regexp.MustCompile(`([a-z0-9-]+)-cli\.sh|\b([a-z0-9-]+)\s+(?:analyze|process|generate|run)`)
	
	// Walk through all files in the scenario
	err := filepath.WalkDir(scenarioPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip files with errors
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
		
		// Find scenario references
		matches := scenarioPattern.FindAllStringSubmatch(contentStr, -1)
		for _, match := range matches {
			if len(match) > 1 && match[1] != scenarioName {
				dep := ScenarioDependency{
					ID:             uuid.New().String(),
					ScenarioName:   scenarioName,
					DependencyType: "scenario",
					DependencyName: match[1],
					Required:       false, // Inter-scenario deps are typically optional
					Purpose:        fmt.Sprintf("Referenced in %s", filepath.Base(path)),
					AccessMethod:   "vrooli scenario",
					Configuration: map[string]interface{}{
						"found_in_file": strings.TrimPrefix(path, scenarioPath),
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
			
			if scenarioRef != "" && scenarioRef != scenarioName {
				dep := ScenarioDependency{
					ID:             uuid.New().String(),
					ScenarioName:   scenarioName,
					DependencyType: "scenario",
					DependencyName: scenarioRef,
					Required:       false,
					Purpose:        fmt.Sprintf("CLI reference in %s", filepath.Base(path)),
					AccessMethod:   "direct_cli",
					Configuration: map[string]interface{}{
						"found_in_file": strings.TrimPrefix(path, scenarioPath),
						"pattern_type":  "cli_reference",
					},
					DiscoveredAt: time.Now(),
					LastVerified: time.Now(),
				}
				dependencies = append(dependencies, dep)
			}
		}
		
		return nil
	})
	
	return dependencies, err
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

func storeDependencies(analysis *DependencyAnalysisResponse) error {
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
	
	// Insert all dependencies
	allDeps := append(analysis.Resources, analysis.Scenarios...)
	allDeps = append(allDeps, analysis.SharedWorkflows...)
	
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
	
	return tx.Commit()
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

func generateDependencyGraph(graphType string) (*DependencyGraph, error) {
	nodes := []GraphNode{}
	edges := []GraphEdge{}
	
	// Query all dependencies from database
	rows, err := db.Query(`
		SELECT scenario_name, dependency_type, dependency_name, required, purpose, access_method
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
		
		err := rows.Scan(&scenarioName, &depType, &depName, &required, &purpose, &accessMethod)
		if err != nil {
			continue
		}
		
		// Filter by graph type
		if graphType == "resource" && depType != "resource" {
			continue
		}
		if graphType == "scenario" && depType == "resource" {
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
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "scenario-dependency-analyzer",
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
		SELECT scenario_name, dependency_type, dependency_name, required, purpose, access_method, configuration
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
		
		err := rows.Scan(&dep.ScenarioName, &dep.DependencyType, &dep.DependencyName,
			&dep.Required, &dep.Purpose, &dep.AccessMethod, &configJSON)
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
		"scenario":          scenarioName,
		"resources":         response["resources"],
		"scenarios":         response["scenarios"],
		"shared_workflows":  response["shared_workflows"],
		"transitive_depth":  0,
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
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"capabilities": []string{"dependency_analysis", "graph_generation"},
		})
	})
	
	// API routes
	api := router.Group("/api/v1")
	{
		api.GET("/analyze/:scenario", analyzeScenarioHandler)
		api.GET("/scenarios/:scenario/dependencies", getDependenciesHandler)
		api.GET("/graph/:type", getGraphHandler)
		api.POST("/analyze/proposed", analyzeProposedHandler)
	}
	
	log.Printf("Starting Scenario Dependency Analyzer API on port %s", config.Port)
	log.Printf("Scenarios directory: %s", config.ScenariosDir)
	
	if err := http.ListenAndServe(":"+config.Port, handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}