package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"

	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

// Models
type SaaSScenario struct {
	ID               string                 `json:"id"`
	ScenarioName     string                 `json:"scenario_name"`
	DisplayName      string                 `json:"display_name"`
	Description      string                 `json:"description"`
	SaaSType         string                 `json:"saas_type"`
	Industry         string                 `json:"industry"`
	RevenuePotential string                 `json:"revenue_potential"`
	HasLandingPage   bool                   `json:"has_landing_page"`
	LandingPageURL   string                 `json:"landing_page_url"`
	LastScan         time.Time              `json:"last_scan"`
	ConfidenceScore  float64                `json:"confidence_score"`
	Metadata         map[string]interface{} `json:"metadata"`
}

type LandingPage struct {
	ID                 string                 `json:"id"`
	ScenarioID         string                 `json:"scenario_id"`
	TemplateID         string                 `json:"template_id"`
	Variant            string                 `json:"variant"`
	Title              string                 `json:"title"`
	Description        string                 `json:"description"`
	Content            map[string]interface{} `json:"content"`
	SEOMetadata        map[string]interface{} `json:"seo_metadata"`
	PerformanceMetrics map[string]interface{} `json:"performance_metrics"`
	Status             string                 `json:"status"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

type Template struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Category     string                 `json:"category"`
	SaaSType     string                 `json:"saas_type"`
	Industry     string                 `json:"industry"`
	HTMLContent  string                 `json:"html_content"`
	CSSContent   string                 `json:"css_content"`
	JSContent    string                 `json:"js_content"`
	ConfigSchema map[string]interface{} `json:"config_schema"`
	PreviewURL   string                 `json:"preview_url"`
	UsageCount   int                    `json:"usage_count"`
	Rating       float64                `json:"rating"`
	CreatedAt    time.Time              `json:"created_at"`
}

type ABTestResult struct {
	ID            string    `json:"id"`
	LandingPageID string    `json:"landing_page_id"`
	Variant       string    `json:"variant"`
	MetricName    string    `json:"metric_name"`
	MetricValue   float64   `json:"metric_value"`
	Timestamp     time.Time `json:"timestamp"`
	SessionID     string    `json:"session_id"`
	UserAgent     string    `json:"user_agent"`
}

// Request/Response types
type ScanRequest struct {
	ForceRescan    bool   `json:"force_rescan"`
	ScenarioFilter string `json:"scenario_filter,omitempty"`
}

type ScanResponse struct {
	TotalScenarios int            `json:"total_scenarios"`
	SaaSScenarios  int            `json:"saas_scenarios"`
	NewlyDetected  int            `json:"newly_detected"`
	Scenarios      []SaaSScenario `json:"scenarios"`
}

type GenerateRequest struct {
	ScenarioID      string                 `json:"scenario_id"`
	TemplateID      string                 `json:"template_id,omitempty"`
	CustomContent   map[string]interface{} `json:"custom_content,omitempty"`
	EnableABTesting bool                   `json:"enable_ab_testing"`
}

type GenerateResponse struct {
	LandingPageID    string   `json:"landing_page_id"`
	PreviewURL       string   `json:"preview_url"`
	DeploymentStatus string   `json:"deployment_status"`
	ABTestVariants   []string `json:"ab_test_variants"`
}

type DeployRequest struct {
	TargetScenario   string `json:"target_scenario"`
	DeploymentMethod string `json:"deployment_method"`
	BackupExisting   bool   `json:"backup_existing"`
}

type DeployResponse struct {
	DeploymentID        string    `json:"deployment_id"`
	AgentSessionID      string    `json:"agent_session_id,omitempty"`
	Status              string    `json:"status"`
	EstimatedCompletion time.Time `json:"estimated_completion"`
}

// Global database connection
var db *sql.DB

// Database service
type DatabaseService struct {
	db *sql.DB
}

func NewDatabaseService(database *sql.DB) *DatabaseService {
	return &DatabaseService{db: database}
}

func (ds *DatabaseService) CreateSaaSScenario(scenario *SaaSScenario) error {
	query := `
		INSERT INTO saas_scenarios (id, scenario_name, display_name, description, saas_type, 
			industry, revenue_potential, has_landing_page, landing_page_url, last_scan, 
			confidence_score, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (scenario_name) DO UPDATE SET
			display_name = $3, description = $4, saas_type = $5, industry = $6,
			revenue_potential = $7, confidence_score = $11, metadata = $12, last_scan = $10
	`

	metadataJSON, _ := json.Marshal(scenario.Metadata)

	_, err := ds.db.Exec(query, scenario.ID, scenario.ScenarioName, scenario.DisplayName,
		scenario.Description, scenario.SaaSType, scenario.Industry, scenario.RevenuePotential,
		scenario.HasLandingPage, scenario.LandingPageURL, scenario.LastScan,
		scenario.ConfidenceScore, string(metadataJSON))

	return err
}

func (ds *DatabaseService) GetSaaSScenarios() ([]SaaSScenario, error) {
	query := `
		SELECT id, scenario_name, display_name, description, saas_type, industry,
			revenue_potential, has_landing_page, landing_page_url, last_scan,
			confidence_score, metadata
		FROM saas_scenarios
		ORDER BY confidence_score DESC, last_scan DESC
	`

	rows, err := ds.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scenarios []SaaSScenario
	for rows.Next() {
		var s SaaSScenario
		var metadataJSON string

		err := rows.Scan(&s.ID, &s.ScenarioName, &s.DisplayName, &s.Description,
			&s.SaaSType, &s.Industry, &s.RevenuePotential, &s.HasLandingPage,
			&s.LandingPageURL, &s.LastScan, &s.ConfidenceScore, &metadataJSON)
		if err != nil {
			continue
		}

		json.Unmarshal([]byte(metadataJSON), &s.Metadata)
		scenarios = append(scenarios, s)
	}

	return scenarios, nil
}

func (ds *DatabaseService) CreateLandingPage(page *LandingPage) error {
	query := `
		INSERT INTO landing_pages (id, scenario_id, template_id, variant, title, description,
			content, seo_metadata, performance_metrics, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	contentJSON, _ := json.Marshal(page.Content)
	seoJSON, _ := json.Marshal(page.SEOMetadata)
	metricsJSON, _ := json.Marshal(page.PerformanceMetrics)

	_, err := ds.db.Exec(query, page.ID, page.ScenarioID, page.TemplateID, page.Variant,
		page.Title, page.Description, string(contentJSON), string(seoJSON),
		string(metricsJSON), page.Status, page.CreatedAt, page.UpdatedAt)

	return err
}

func (ds *DatabaseService) GetTemplates(category, saasType string) ([]Template, error) {
	query := `
		SELECT id, name, category, saas_type, industry, html_content, css_content,
			js_content, config_schema, preview_url, usage_count, rating, created_at
		FROM templates
		WHERE ($1 = '' OR category = $1) AND ($2 = '' OR saas_type = $2)
		ORDER BY usage_count DESC, rating DESC
	`

	rows, err := ds.db.Query(query, category, saasType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []Template
	for rows.Next() {
		var t Template
		var configJSON string

		err := rows.Scan(&t.ID, &t.Name, &t.Category, &t.SaaSType, &t.Industry,
			&t.HTMLContent, &t.CSSContent, &t.JSContent, &configJSON,
			&t.PreviewURL, &t.UsageCount, &t.Rating, &t.CreatedAt)
		if err != nil {
			continue
		}

		json.Unmarshal([]byte(configJSON), &t.ConfigSchema)
		templates = append(templates, t)
	}

	return templates, nil
}

// SaaS Detection Service
type SaaSDetectionService struct {
	scenariosPath string
	dbService     *DatabaseService
}

func NewSaaSDetectionService(scenariosPath string, dbService *DatabaseService) *SaaSDetectionService {
	return &SaaSDetectionService{
		scenariosPath: scenariosPath,
		dbService:     dbService,
	}
}

func (sds *SaaSDetectionService) ScanScenarios(forceRescan bool, scenarioFilter string) (*ScanResponse, error) {
	scenariosDir := sds.scenariosPath
	if scenariosDir == "" {
		scenariosDir = "../../" // Default relative path from api directory
	}

	entries, err := ioutil.ReadDir(scenariosDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenarios directory: %w", err)
	}

	var totalScenarios, saasScenarios, newlyDetected int
	var scenarios []SaaSScenario

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		scenarioName := entry.Name()
		if scenarioFilter != "" && !strings.Contains(scenarioName, scenarioFilter) {
			continue
		}

		totalScenarios++

		// Check if scenario is SaaS
		isSaaS, scenario := sds.analyzeSaaSCharacteristics(scenarioName, scenariosDir)
		if isSaaS {
			saasScenarios++

			// Check if this is newly detected
			existing, err := sds.getExistingScenario(scenarioName)
			if err != nil || existing == nil {
				newlyDetected++
			}

			// Store in database
			scenario.ID = uuid.New().String()
			scenario.LastScan = time.Now()

			err = sds.dbService.CreateSaaSScenario(&scenario)
			if err != nil {
				log.Printf("Failed to store scenario %s: %v", scenarioName, err)
			}

			scenarios = append(scenarios, scenario)
		}
	}

	return &ScanResponse{
		TotalScenarios: totalScenarios,
		SaaSScenarios:  saasScenarios,
		NewlyDetected:  newlyDetected,
		Scenarios:      scenarios,
	}, nil
}

func (sds *SaaSDetectionService) analyzeSaaSCharacteristics(scenarioName, scenariosDir string) (bool, SaaSScenario) {
	cleanName := filepath.Clean(scenarioName)
	if strings.Contains(cleanName, "..") {
		return false, SaaSScenario{ScenarioName: scenarioName}
	}
	scenarioPath := filepath.Join(scenariosDir, cleanName)
	scenario := SaaSScenario{
		ScenarioName: cleanName,
		Metadata:     make(map[string]interface{}),
	}

	var confidenceScore float64
	var characteristics []string

	// 1. Analyze service.json for SaaS indicators
	serviceFile := filepath.Join(scenarioPath, ".vrooli", "service.json")
	if serviceContent, err := ioutil.ReadFile(serviceFile); err == nil {
		var serviceConfig map[string]interface{}
		if json.Unmarshal(serviceContent, &serviceConfig) == nil {

			// Extract display name and description
			if service, ok := serviceConfig["service"].(map[string]interface{}); ok {
				if displayName, ok := service["displayName"].(string); ok {
					scenario.DisplayName = displayName
				}
				if description, ok := service["description"].(string); ok {
					scenario.Description = description
				}

				// Check tags for SaaS indicators
				if tags, ok := service["tags"].([]interface{}); ok {
					saasIndicators := []string{"multi-tenant", "billing", "analytics", "a-b-testing",
						"saas", "business-application", "subscription", "api-service"}

					for _, tag := range tags {
						if tagStr, ok := tag.(string); ok {
							for _, indicator := range saasIndicators {
								if strings.Contains(tagStr, indicator) {
									confidenceScore += 0.2
									characteristics = append(characteristics, fmt.Sprintf("tag:%s", tagStr))
								}
							}
						}
					}
				}
			}

			// Check for database requirements (typical for SaaS)
			if resources, ok := serviceConfig["resources"].(map[string]interface{}); ok {
				if postgres, ok := resources["postgres"].(map[string]interface{}); ok {
					if enabled, ok := postgres["enabled"].(bool); ok && enabled {
						confidenceScore += 0.3
						characteristics = append(characteristics, "database:postgres")
					}
				}
			}
		}
	}

	// 2. Analyze PRD.md for business value and revenue indicators
	prdFile := filepath.Join(scenarioPath, "PRD.md")
	if prdContent, err := ioutil.ReadFile(prdFile); err == nil {
		content := string(prdContent)

		// Look for revenue indicators
		revenuePatterns := []string{
			`\$\d+K`,
			`revenue.*potential`,
			`business.*value`,
			`subscription`,
			`pricing`,
			`enterprise`,
			`saas`,
			`multi-tenant`,
		}

		for _, pattern := range revenuePatterns {
			if matched, _ := regexp.MatchString("(?i)"+pattern, content); matched {
				confidenceScore += 0.1
				characteristics = append(characteristics, fmt.Sprintf("prd:%s", pattern))
			}
		}

		// Extract revenue potential
		revenueRegex := regexp.MustCompile(`(?i)revenue.*potential.*\$(\d+K?\s*-\s*\$?\d+K?)`)
		if matches := revenueRegex.FindStringSubmatch(content); len(matches) > 1 {
			scenario.RevenuePotential = matches[1]
			confidenceScore += 0.2
		}

		// Determine SaaS type based on content
		if strings.Contains(strings.ToLower(content), "b2b") {
			scenario.SaaSType = "b2b_tool"
		} else if strings.Contains(strings.ToLower(content), "api") {
			scenario.SaaSType = "api_service"
		} else if strings.Contains(strings.ToLower(content), "marketplace") {
			scenario.SaaSType = "marketplace"
		} else {
			scenario.SaaSType = "b2c_app"
		}
	}

	// 3. Check for existing UI (indicates user-facing application)
	uiPath := filepath.Join(scenarioPath, "ui")
	if _, err := os.Stat(uiPath); err == nil {
		confidenceScore += 0.2
		characteristics = append(characteristics, "has_ui")
	}

	// 4. Check for API (indicates service offering)
	apiPath := filepath.Join(scenarioPath, "api")
	if _, err := os.Stat(apiPath); err == nil {
		confidenceScore += 0.1
		characteristics = append(characteristics, "has_api")
	}

	// 5. Check for existing landing page
	landingPath := filepath.Join(scenarioPath, "landing")
	if _, err := os.Stat(landingPath); err == nil {
		scenario.HasLandingPage = true
		scenario.LandingPageURL = fmt.Sprintf("/scenarios/%s/landing/", scenarioName)
		characteristics = append(characteristics, "existing_landing")
	}

	scenario.ConfidenceScore = confidenceScore
	scenario.Metadata["characteristics"] = characteristics
	scenario.Metadata["analysis_version"] = "1.0"

	// Consider it a SaaS if confidence score is above threshold
	isSaaS := confidenceScore >= 0.5

	return isSaaS, scenario
}

func (sds *SaaSDetectionService) getExistingScenario(scenarioName string) (*SaaSScenario, error) {
	query := `SELECT id FROM saas_scenarios WHERE scenario_name = $1`
	var id string
	err := sds.dbService.db.QueryRow(query, scenarioName).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &SaaSScenario{ID: id}, nil
}

// Landing Page Service
type LandingPageService struct {
	dbService     *DatabaseService
	templatesPath string
}

func NewLandingPageService(dbService *DatabaseService, templatesPath string) *LandingPageService {
	return &LandingPageService{
		dbService:     dbService,
		templatesPath: templatesPath,
	}
}

func (lps *LandingPageService) GenerateLandingPage(req *GenerateRequest) (*GenerateResponse, error) {
	// Get the template
	templates, err := lps.dbService.GetTemplates("", "")
	if err != nil {
		return nil, err
	}

	var selectedTemplate Template
	if req.TemplateID != "" {
		// Find specific template
		for _, t := range templates {
			if t.ID == req.TemplateID {
				selectedTemplate = t
				break
			}
		}
	} else {
		// Use first available template
		if len(templates) > 0 {
			selectedTemplate = templates[0]
		}
	}

	// Create landing page
	landingPageID := uuid.New().String()
	landingPage := &LandingPage{
		ID:                 landingPageID,
		ScenarioID:         req.ScenarioID,
		TemplateID:         selectedTemplate.ID,
		Variant:            "control",
		Title:              "Generated Landing Page",
		Description:        "AI-generated landing page",
		Content:            req.CustomContent,
		SEOMetadata:        make(map[string]interface{}),
		PerformanceMetrics: make(map[string]interface{}),
		Status:             "draft",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	err = lps.dbService.CreateLandingPage(landingPage)
	if err != nil {
		return nil, err
	}

	// Generate variants for A/B testing if enabled
	variants := []string{"control"}
	if req.EnableABTesting {
		// Create A/B test variants
		for _, variant := range []string{"a", "b"} {
			variantPage := *landingPage
			variantPage.ID = uuid.New().String()
			variantPage.Variant = variant
			variantPage.Title = fmt.Sprintf("%s - Variant %s", landingPage.Title, strings.ToUpper(variant))

			err = lps.dbService.CreateLandingPage(&variantPage)
			if err != nil {
				log.Printf("Failed to create variant %s: %v", variant, err)
			} else {
				variants = append(variants, variant)
			}
		}
	}

	return &GenerateResponse{
		LandingPageID:    landingPageID,
		PreviewURL:       fmt.Sprintf("/preview/%s", landingPageID),
		DeploymentStatus: "ready",
		ABTestVariants:   variants,
	}, nil
}

// Claude Code Integration Service
type ClaudeCodeService struct {
	claudeCodeBinary string
}

func NewClaudeCodeService(claudeCodeBinary string) *ClaudeCodeService {
	if claudeCodeBinary == "" {
		claudeCodeBinary = "resource-claude-code" // Default CLI command
	}
	return &ClaudeCodeService{
		claudeCodeBinary: claudeCodeBinary,
	}
}

func (ccs *ClaudeCodeService) DeployLandingPage(landingPageID, targetScenario string, req *DeployRequest) (*DeployResponse, error) {
	deploymentID := uuid.New().String()

	if req.DeploymentMethod == "claude_agent" {
		// Spawn Claude Code agent for intelligent deployment
		agentSessionID, err := ccs.spawnClaudeAgent(targetScenario, landingPageID)
		if err != nil {
			return nil, fmt.Errorf("failed to spawn Claude agent: %w", err)
		}

		return &DeployResponse{
			DeploymentID:        deploymentID,
			AgentSessionID:      agentSessionID,
			Status:              "agent_working",
			EstimatedCompletion: time.Now().Add(10 * time.Minute),
		}, nil
	} else {
		// Direct deployment
		err := ccs.directDeploy(targetScenario, landingPageID, req.BackupExisting)
		if err != nil {
			return nil, fmt.Errorf("direct deployment failed: %w", err)
		}

		return &DeployResponse{
			DeploymentID:        deploymentID,
			Status:              "completed",
			EstimatedCompletion: time.Now(),
		}, nil
	}
}

func (ccs *ClaudeCodeService) spawnClaudeAgent(targetScenario, landingPageID string) (string, error) {
	// Create deployment prompt for Claude Code agent
	prompt := fmt.Sprintf(`
		Deploy landing page %s to scenario %s.
		
		Tasks:
		1. Create landing/ directory in target scenario if it doesn't exist
		2. Generate responsive HTML/CSS/JS files for the landing page
		3. Set up A/B testing routing configuration
		4. Add SEO optimizations and meta tags
		5. Ensure mobile responsiveness and Core Web Vitals compliance
		6. Create backup of existing landing page if present
		
		Use the landing page data from the saas-landing-manager API.
	`, landingPageID, targetScenario)

	// Execute Claude Code agent spawn
	cmd := exec.Command(ccs.claudeCodeBinary, "spawn", "--task", prompt)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse agent session ID from output (simplified)
	sessionID := strings.TrimSpace(string(output))
	if sessionID == "" {
		sessionID = uuid.New().String() // Fallback
	}

	return sessionID, nil
}

func (ccs *ClaudeCodeService) directDeploy(targetScenario, landingPageID string, backup bool) error {
	// Simplified direct deployment - create basic landing page structure
	cleanTarget := filepath.Clean(targetScenario)
	if strings.Contains(cleanTarget, "..") {
		return fmt.Errorf("invalid target scenario path: %s", targetScenario)
	}
	scenarioPath := filepath.Join("..", "..", cleanTarget)
	landingPath := filepath.Join(scenarioPath, "landing")

	// Create landing directory
	err := os.MkdirAll(landingPath, 0755)
	if err != nil {
		return err
	}

	// Create basic index.html
	indexHTML := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Welcome to Our SaaS</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; }
        .hero { text-align: center; padding: 50px 0; }
        .cta { background: #007bff; color: white; padding: 15px 30px; border: none; border-radius: 5px; }
    </style>
</head>
<body>
    <div class="hero">
        <h1>Revolutionary SaaS Solution</h1>
        <p>Transform your business with our innovative platform</p>
        <button class="cta">Get Started</button>
    </div>
</body>
</html>`

	err = ioutil.WriteFile(filepath.Join(landingPath, "index.html"), []byte(indexHTML), 0644)
	if err != nil {
		return err
	}

	log.Printf("Successfully deployed landing page to %s", targetScenario)
	return nil
}

// HTTP Handlers
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := "healthy"
	ready := true
	latency := interface{}(nil)
	dbErr := interface{}(nil)
	connected := false

	if db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		start := time.Now()
		err := db.PingContext(ctx)
		if err != nil {
			status = "degraded"
			ready = false
			dbErr = map[string]interface{}{
				"code":      "DB_PING_FAILED",
				"message":   err.Error(),
				"category":  "resource",
				"retryable": true,
			}
		} else {
			connected = true
			latency = float64(time.Since(start).Milliseconds())
		}
	} else {
		status = "degraded"
		ready = false
		dbErr = map[string]interface{}{
			"code":      "DB_UNINITIALIZED",
			"message":   "database connection not initialized",
			"category":  "configuration",
			"retryable": true,
		}
	}

	response := map[string]interface{}{
		"status":    status,
		"service":   "saas-landing-manager-api",
		"timestamp": time.Now().Format(time.RFC3339),
		"readiness": ready,
		"version":   "1.0.0",
		"dependencies": map[string]interface{}{
			"database": map[string]interface{}{
				"connected":  connected,
				"latency_ms": latency,
				"error":      dbErr,
			},
		},
		"metrics": map[string]interface{}{
			"goroutines": runtime.NumGoroutine(),
		},
	}

	json.NewEncoder(w).Encode(response)
}

func scanScenariosHandler(w http.ResponseWriter, r *http.Request) {
	var req ScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get scenarios path from environment or use default
	scenariosPath := os.Getenv("SCENARIOS_PATH")
	if scenariosPath == "" {
		scenariosPath = "../../" // Default relative path
	}

	dbService := NewDatabaseService(db)
	detectionService := NewSaaSDetectionService(scenariosPath, dbService)

	response, err := detectionService.ScanScenarios(req.ForceRescan, req.ScenarioFilter)
	if err != nil {
		http.Error(w, fmt.Sprintf("Scan failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func generateLandingPageHandler(w http.ResponseWriter, r *http.Request) {
	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	dbService := NewDatabaseService(db)
	landingPageService := NewLandingPageService(dbService, "./templates")

	response, err := landingPageService.GenerateLandingPage(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Generation failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func deployLandingPageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	landingPageID := vars["id"]

	var req DeployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	claudeService := NewClaudeCodeService("")
	response, err := claudeService.DeployLandingPage(landingPageID, req.TargetScenario, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Deployment failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	saasType := r.URL.Query().Get("saas_type")

	dbService := NewDatabaseService(db)
	templates, err := dbService.GetTemplates(category, saasType)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get templates: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"templates": templates,
	})
}

func getDashboardHandler(w http.ResponseWriter, r *http.Request) {
	dbService := NewDatabaseService(db)
	scenarios, err := dbService.GetSaaSScenarios()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get dashboard data: %v", err), http.StatusInternalServerError)
		return
	}

	dashboard := map[string]interface{}{
		"total_pages":             len(scenarios),
		"active_ab_tests":         0,   // TODO: Implement A/B test counting
		"average_conversion_rate": 0.0, // TODO: Calculate from analytics
		"scenarios":               scenarios,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

// Database initialization
func initDatabase() error {
	// Database configuration - support both POSTGRES_URL and individual components
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			return fmt.Errorf("❌ Missing database configuration. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	var err error
	db, err = sql.Open("postgres", postgresURL)
	if err != nil {
		return fmt.Errorf("Failed to open database connection: %v", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("📆 Database URL configured")

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
		return fmt.Errorf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}

	log.Println("🎉 Database connection pool established successfully!")
	return nil
}

func runLifecycleCommand(command string, args []string) error {
	if err := initDatabase(); err != nil {
		return err
	}
	defer func() {
		if db != nil {
			db.Close()
			db = nil
		}
	}()

	dbService := NewDatabaseService(db)

	switch command {
	case "scan-scenarios":
		forceRescan := false
		scenarioFilter := ""
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "--force", "-f":
				forceRescan = true
			case "--filter":
				if i+1 < len(args) {
					scenarioFilter = args[i+1]
					i++
				}
			}
		}

		scenariosPath := os.Getenv("SCENARIOS_PATH")
		if scenariosPath == "" {
			scenariosPath = "../.."
		}

		detectionService := NewSaaSDetectionService(scenariosPath, dbService)
		response, err := detectionService.ScanScenarios(forceRescan, scenarioFilter)
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}

		output, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(output))
		return nil
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start saas-landing-manager

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	if len(os.Args) > 1 {
		if err := runLifecycleCommand(os.Args[1], os.Args[2:]); err != nil {
			log.Fatalf("%v", err)
		}
		return
	}

	// Initialize database
	if err := initDatabase(); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()

	// Setup router
	r := mux.NewRouter()

	// API routes
	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", healthHandler).Methods("GET")
	api.HandleFunc("/scenarios/scan", scanScenariosHandler).Methods("POST")
	api.HandleFunc("/landing-pages/generate", generateLandingPageHandler).Methods("POST")
	api.HandleFunc("/landing-pages/{id}/deploy", deployLandingPageHandler).Methods("POST")
	api.HandleFunc("/templates", getTemplatesHandler).Methods("GET")
	api.HandleFunc("/analytics/dashboard", getDashboardHandler).Methods("GET")

	// Health check endpoint
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(r)

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	log.Printf("SaaS Landing Manager API starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
