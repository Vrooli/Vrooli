package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"scenario-completeness-scoring/pkg/analysis"
	"scenario-completeness-scoring/pkg/circuitbreaker"
	"scenario-completeness-scoring/pkg/collectors"
	"scenario-completeness-scoring/pkg/config"
	"scenario-completeness-scoring/pkg/health"
	"scenario-completeness-scoring/pkg/history"
	"scenario-completeness-scoring/pkg/scoring"
	"scenario-completeness-scoring/pkg/validators"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// Config holds minimal runtime configuration
type Config struct {
	Port       string
	VrooliRoot string
}

// Server wires the HTTP router
// [REQ:SCS-CB-001] Server includes circuit breaker registry
// [REQ:SCS-HEALTH-001] Server includes health tracker
// [REQ:SCS-CFG-004] Server includes config loader for persistence
// [REQ:SCS-HIST-001] Server includes history repository
// [REQ:SCS-ANALYSIS-001] Server includes what-if analyzer
// [REQ:SCS-ANALYSIS-003] Server includes bulk refresher
type Server struct {
	config         *Config
	router         *mux.Router
	collector      *collectors.MetricsCollector
	cbRegistry     *circuitbreaker.Registry
	healthTracker  *health.Tracker
	configLoader   *config.Loader
	historyDB      *history.DB
	historyRepo    *history.Repository
	trendAnalyzer  *history.TrendAnalyzer
	whatIfAnalyzer *analysis.WhatIfAnalyzer
	bulkRefresher  *analysis.BulkRefresher
}

// NewServer initializes configuration and routes
// [REQ:SCS-CB-001] Initialize circuit breaker with default config
// [REQ:SCS-HEALTH-001] Initialize health tracker
// [REQ:SCS-CFG-004] Initialize config loader
// [REQ:SCS-HIST-001] Initialize history database and repository
// [REQ:SCS-ANALYSIS-001] Initialize what-if analyzer
// [REQ:SCS-ANALYSIS-003] Initialize bulk refresher
func NewServer() (*Server, error) {
	cfg := &Config{
		Port:       requireEnv("API_PORT"),
		VrooliRoot: getEnvWithDefault("VROOLI_ROOT", os.Getenv("HOME")+"/Vrooli"),
	}

	// Initialize circuit breaker registry with default config
	cbRegistry := circuitbreaker.NewRegistry(circuitbreaker.DefaultConfig())

	// Initialize history database
	// [REQ:SCS-HIST-002] SQLite database for history storage
	dataDir := filepath.Join(cfg.VrooliRoot, "scenarios", "scenario-completeness-scoring", "data")
	historyDB, err := history.NewDB(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize history database: %w", err)
	}
	historyRepo := history.NewRepository(historyDB)
	trendAnalyzer := history.NewTrendAnalyzer(historyRepo, 5) // Stall after 5 unchanged

	// Initialize metrics collector
	collector := collectors.NewMetricsCollector(cfg.VrooliRoot)

	srv := &Server{
		config:         cfg,
		router:         mux.NewRouter(),
		collector:      collector,
		cbRegistry:     cbRegistry,
		healthTracker:  health.NewTracker(cbRegistry),
		configLoader:   config.NewLoader(cfg.VrooliRoot),
		historyDB:      historyDB,
		historyRepo:    historyRepo,
		trendAnalyzer:  trendAnalyzer,
		whatIfAnalyzer: analysis.NewWhatIfAnalyzer(collector),
		bulkRefresher:  analysis.NewBulkRefresher(cfg.VrooliRoot, collector, historyRepo),
	}

	srv.setupRoutes()
	return srv, nil
}

// configToScoringOptions converts config.ScoringConfig to scoring.ScoringOptions
// Normalizes weights to sum to 100 when some dimensions are disabled
func configToScoringOptions(cfg *config.ScoringConfig) *scoring.ScoringOptions {
	if cfg == nil {
		return scoring.DefaultScoringOptions()
	}

	// Normalize weights to redistribute when dimensions are disabled
	// This ensures the total always sums to 100
	normalizedWeights := cfg.Weights.Normalize(cfg.Components)

	return &scoring.ScoringOptions{
		QualityEnabled:  cfg.Components.Quality.Enabled,
		CoverageEnabled: cfg.Components.Coverage.Enabled,
		QuantityEnabled: cfg.Components.Quantity.Enabled,
		UIEnabled:       cfg.Components.UI.Enabled,
		QualityWeight:   normalizedWeights.Quality,
		CoverageWeight:  normalizedWeights.Coverage,
		QuantityWeight:  normalizedWeights.Quantity,
		UIWeight:        normalizedWeights.UI,
	}
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)

	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/health", s.handleHealth).Methods("GET", "OPTIONS")

	// Scoring endpoints [REQ:SCS-CORE-002]
	s.router.HandleFunc("/api/v1/scores", s.handleListScores).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/scores/{scenario}", s.handleGetScore).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/scores/{scenario}/calculate", s.handleCalculateScore).Methods("POST", "OPTIONS")

	// Configuration endpoints [REQ:SCS-CFG-001] [REQ:SCS-CFG-004]
	s.router.HandleFunc("/api/v1/config", s.handleGetConfig).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/config", s.handleUpdateConfig).Methods("PUT", "OPTIONS")
	s.router.HandleFunc("/api/v1/config/thresholds", s.handleGetThresholds).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/config/thresholds/{category}", s.handleGetCategoryThresholds).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/config/scenarios/{scenario}", s.handleGetScenarioConfig).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/config/scenarios/{scenario}", s.handleUpdateScenarioConfig).Methods("PUT", "OPTIONS")
	s.router.HandleFunc("/api/v1/config/scenarios/{scenario}", s.handleDeleteScenarioConfig).Methods("DELETE", "OPTIONS")
	s.router.HandleFunc("/api/v1/config/presets", s.handleListPresets).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/config/presets/{name}/apply", s.handleApplyPreset).Methods("POST", "OPTIONS")

	// Health monitoring endpoints [REQ:SCS-HEALTH-001]
	s.router.HandleFunc("/api/v1/health/collectors", s.handleGetCollectorHealth).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/health/collectors/{name}/test", s.handleTestCollector).Methods("POST", "OPTIONS")

	// Circuit breaker endpoints [REQ:SCS-CB-004]
	s.router.HandleFunc("/api/v1/health/circuit-breaker", s.handleGetCircuitBreakers).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/health/circuit-breaker/reset", s.handleResetAllCircuitBreakers).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/api/v1/health/circuit-breaker/{collector}/reset", s.handleResetCircuitBreaker).Methods("POST", "OPTIONS")

	// History and trends endpoints [REQ:SCS-HIST-001] [REQ:SCS-HIST-003] [REQ:SCS-HIST-004]
	s.router.HandleFunc("/api/v1/scores/{scenario}/history", s.handleGetHistory).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/scores/{scenario}/trends", s.handleGetTrends).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/trends", s.handleGetAllTrends).Methods("GET", "OPTIONS")

	// Analysis endpoints [REQ:SCS-ANALYSIS-001] [REQ:SCS-ANALYSIS-003] [REQ:SCS-ANALYSIS-004]
	s.router.HandleFunc("/api/v1/scores/{scenario}/what-if", s.handleWhatIf).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/api/v1/scores/refresh-all", s.handleBulkRefresh).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/api/v1/compare", s.handleCompare).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/api/v1/recommendations/{scenario}", s.handleGetRecommendations).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/analysis/components", s.handleListAnalysisComponents).Methods("GET", "OPTIONS")

	// Validation quality analysis endpoint
	s.router.HandleFunc("/api/v1/scores/{scenario}/validation-analysis", s.handleValidationAnalysis).Methods("GET", "OPTIONS")
}

// Start launches the HTTP server with graceful shutdown
func (s *Server) Start() error {
	s.log("starting server", map[string]interface{}{
		"service":     "scenario-completeness-scoring-api",
		"port":        s.config.Port,
		"vrooli_root": s.config.VrooliRoot,
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", s.config.Port),
		Handler:      handlers.RecoveryHandler()(s.router),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log("server startup failed", map[string]interface{}{"error": err.Error()})
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	// Close history database
	if s.historyDB != nil {
		if err := s.historyDB.Close(); err != nil {
			s.log("failed to close history database", map[string]interface{}{"error": err.Error()})
		}
	}

	s.log("server stopped", nil)
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "Scenario Completeness Scoring API",
		"version":   "1.0.0",
		"readiness": true,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleListScores returns a list of all scenarios with their scores
// [REQ:SCS-CORE-002] Score retrieval API endpoint
func (s *Server) handleListScores(w http.ResponseWriter, r *http.Request) {
	scenariosDir := filepath.Join(s.config.VrooliRoot, "scenarios")

	// Load config for scoring options
	cfg, _ := s.configLoader.LoadGlobal()
	scoringOpts := configToScoringOptions(cfg)

	// Discover scenarios by listing directories
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		http.Error(w, "Failed to read scenarios directory", http.StatusInternalServerError)
		return
	}

	var scenarios []map[string]interface{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Skip hidden directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		scenarioName := entry.Name()

		// Try to collect metrics (gracefully handle failures)
		metrics, err := s.collector.Collect(scenarioName)
		if err != nil {
			// Log but continue with other scenarios
			log.Printf("Warning: failed to collect metrics for %s: %v", scenarioName, err)
			continue
		}

		thresholds := scoring.GetThresholds(metrics.Category)
		breakdown := scoring.CalculateCompletenessScoreWithOptions(*metrics, thresholds, nil, scoringOpts)

		scenarios = append(scenarios, map[string]interface{}{
			"scenario":       scenarioName,
			"category":       metrics.Category,
			"score":          breakdown.Score,
			"classification": breakdown.Classification,
		})
	}

	response := map[string]interface{}{
		"scenarios":     scenarios,
		"total":         len(scenarios),
		"calculated_at": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetScore returns the detailed score for a specific scenario
// [REQ:SCS-CORE-002] Score retrieval API endpoint
func (s *Server) handleGetScore(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	// Collect real metrics from the scenario
	metrics, err := s.collector.Collect(scenarioName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to collect metrics for scenario %s: %v", scenarioName, err), http.StatusNotFound)
		return
	}

	// Load config for scoring options
	cfg, _ := s.configLoader.LoadGlobal()
	scoringOpts := configToScoringOptions(cfg)

	// Perform validation quality analysis
	scenarioRoot := s.collector.GetScenarioRoot(scenarioName)
	rawReqs := s.collector.LoadRequirements(scenarioName)
	validatorReqs := validators.ConvertRequirements(rawReqs)

	validationAnalysis := validators.AnalyzeValidationQuality(
		validators.MetricCounts{
			RequirementsTotal: metrics.Requirements.Total,
			TestsTotal:        metrics.Tests.Total,
		},
		validatorReqs,
		nil, // No operational targets yet
		scenarioRoot,
	)

	// Convert to scoring.ValidationQualityAnalysis for score calculation
	var scoringValidation *scoring.ValidationQualityAnalysis
	if validationAnalysis.TotalPenalty > 0 {
		penalties := make([]scoring.PenaltyDetail, len(validationAnalysis.Issues))
		for i, issue := range validationAnalysis.Issues {
			penalties[i] = scoring.PenaltyDetail{
				Type:    issue.Type,
				Points:  issue.Penalty,
				Message: issue.Message,
			}
		}
		scoringValidation = &scoring.ValidationQualityAnalysis{
			TotalPenalty: validationAnalysis.TotalPenalty,
			Penalties:    penalties,
		}
	}

	thresholds := scoring.GetThresholds(metrics.Category)
	breakdown := scoring.CalculateCompletenessScoreWithOptions(*metrics, thresholds, scoringValidation, scoringOpts)
	recommendations := scoring.GenerateRecommendations(breakdown, thresholds)

	response := map[string]interface{}{
		"scenario":            scenarioName,
		"category":            metrics.Category,
		"score":               breakdown.Score,
		"base_score":          breakdown.BaseScore,
		"validation_penalty":  breakdown.ValidationPenalty,
		"classification":      breakdown.Classification,
		"breakdown":           breakdown,
		"metrics":             metrics,
		"validation_analysis": validationAnalysis,
		"recommendations":     recommendations,
		"calculated_at":       time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCalculateScore forces recalculation of score for a scenario
// [REQ:SCS-CORE-002] Score retrieval API endpoint
// [REQ:SCS-HIST-001] Save score snapshot to history
func (s *Server) handleCalculateScore(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	// Force recalculation by collecting fresh metrics
	metrics, err := s.collector.Collect(scenarioName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to calculate score for scenario %s: %v", scenarioName, err), http.StatusNotFound)
		return
	}

	// Load config for scoring options
	cfg, _ := s.configLoader.LoadGlobal()
	scoringOpts := configToScoringOptions(cfg)

	thresholds := scoring.GetThresholds(metrics.Category)
	breakdown := scoring.CalculateCompletenessScoreWithOptions(*metrics, thresholds, nil, scoringOpts)
	recommendations := scoring.GenerateRecommendations(breakdown, thresholds)

	// Save snapshot to history [REQ:SCS-HIST-001]
	var snapshotID int64
	if s.historyRepo != nil {
		snapshot, err := s.historyRepo.Save(scenarioName, &breakdown, nil)
		if err != nil {
			log.Printf("Warning: failed to save history snapshot: %v", err)
		} else if snapshot != nil {
			snapshotID = snapshot.ID
		}
	}

	response := map[string]interface{}{
		"scenario":        scenarioName,
		"category":        metrics.Category,
		"score":           breakdown.Score,
		"classification":  breakdown.Classification,
		"breakdown":       breakdown,
		"metrics":         metrics,
		"recommendations": recommendations,
		"calculated_at":   time.Now().UTC().Format(time.RFC3339),
		"recalculated":    true,
		"snapshot_id":     snapshotID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetConfig returns the current scoring configuration
// [REQ:SCS-CFG-001] Component toggle API
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	// Load actual config from disk
	cfg, err := s.configLoader.LoadGlobal()
	if err != nil {
		// Fall back to default config if load fails
		defaultCfg := config.DefaultConfig()
		cfg = &defaultCfg
	}

	response := map[string]interface{}{
		"version": cfg.Version,
		"scoring": map[string]interface{}{
			"weights": map[string]int{
				"quality":  cfg.Weights.Quality,
				"coverage": cfg.Weights.Coverage,
				"quantity": cfg.Weights.Quantity,
				"ui":       cfg.Weights.UI,
			},
			"quality_breakdown": map[string]int{
				"requirement_pass_rate": 20,
				"target_pass_rate":      15,
				"test_pass_rate":        15,
			},
			"coverage_breakdown": map[string]int{
				"test_coverage_ratio": 8,
				"depth_score":         7,
			},
			"quantity_breakdown": map[string]int{
				"requirements": 4,
				"targets":      3,
				"tests":        3,
			},
			"ui_breakdown": map[string]float64{
				"template_check":       10,
				"component_complexity": 5,
				"api_integration":      6,
				"routing":              1.5,
				"code_volume":          2.5,
			},
		},
		"components": cfg.Components,
		"penalties":  cfg.Penalties,
		"classifications": map[string]interface{}{
			"production_ready":      map[string]int{"min": 96, "max": 100},
			"nearly_ready":          map[string]int{"min": 81, "max": 95},
			"mostly_complete":       map[string]int{"min": 61, "max": 80},
			"functional_incomplete": map[string]int{"min": 41, "max": 60},
			"foundation_laid":       map[string]int{"min": 21, "max": 40},
			"early_stage":           map[string]int{"min": 0, "max": 20},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetThresholds returns all category thresholds
func (s *Server) handleGetThresholds(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"categories":       scoring.CategoryThresholds,
		"default_category": "utility",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetCategoryThresholds returns thresholds for a specific category
func (s *Server) handleGetCategoryThresholds(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	category := vars["category"]

	thresholds := scoring.GetThresholds(category)
	response := map[string]interface{}{
		"category":   category,
		"thresholds": thresholds,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleUpdateConfig updates the global scoring configuration
// [REQ:SCS-CFG-004] Configuration persistence
func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var cfg config.ScoringConfig
	if err := json.Unmarshal(body, &cfg); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Validate configuration
	if err := config.ValidateConfig(&cfg); err != nil {
		http.Error(w, fmt.Sprintf("Invalid configuration: %v", err), http.StatusBadRequest)
		return
	}

	// Save to disk
	if err := s.configLoader.SaveGlobal(&cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save configuration: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Global configuration updated",
		"config":  cfg,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetScenarioConfig returns configuration for a specific scenario
// [REQ:SCS-CFG-002] Per-scenario overrides
func (s *Server) handleGetScenarioConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	// Get effective config (merged global + scenario override)
	effectiveCfg, err := s.configLoader.GetEffectiveConfig(scenarioName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load configuration: %v", err), http.StatusInternalServerError)
		return
	}

	// Get scenario-specific override
	override, err := s.configLoader.LoadScenarioOverride(scenarioName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load override: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"scenario":  scenarioName,
		"effective": effectiveCfg,
		"override":  override,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleUpdateScenarioConfig updates configuration for a specific scenario
// [REQ:SCS-CFG-002] Per-scenario overrides
func (s *Server) handleUpdateScenarioConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var override config.ScenarioOverride
	if err := json.Unmarshal(body, &override); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	override.Scenario = scenarioName

	// Validate if overrides are provided
	if override.Overrides != nil {
		if err := config.ValidateConfig(override.Overrides); err != nil {
			http.Error(w, fmt.Sprintf("Invalid configuration: %v", err), http.StatusBadRequest)
			return
		}
	}

	// Save override
	if err := s.configLoader.SaveScenarioOverride(&override); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save configuration: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":  true,
		"message":  fmt.Sprintf("Configuration for %s updated", scenarioName),
		"override": override,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDeleteScenarioConfig deletes configuration override for a specific scenario
// [REQ:SCS-CFG-002] Per-scenario overrides
func (s *Server) handleDeleteScenarioConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	if err := s.configLoader.DeleteScenarioOverride(scenarioName); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete configuration: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Configuration override for %s deleted", scenarioName),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleListPresets returns all available configuration presets
// [REQ:SCS-CFG-003] Configuration presets system
func (s *Server) handleListPresets(w http.ResponseWriter, r *http.Request) {
	presets := config.ListPresetInfo()

	response := map[string]interface{}{
		"presets": presets,
		"total":   len(presets),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleApplyPreset applies a preset to global configuration
// [REQ:SCS-CFG-003] Configuration presets system
func (s *Server) handleApplyPreset(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	presetName := vars["name"]

	preset := config.GetPreset(presetName)
	if preset == nil {
		http.Error(w, fmt.Sprintf("Unknown preset: %s", presetName), http.StatusNotFound)
		return
	}

	// Save preset config as global config
	if err := s.configLoader.SaveGlobal(&preset.Config); err != nil {
		http.Error(w, fmt.Sprintf("Failed to apply preset: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Applied preset '%s'", presetName),
		"preset":  preset.Name,
		"config":  preset.Config,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetCollectorHealth returns health status of all collectors
// [REQ:SCS-HEALTH-001] Health status API endpoint
func (s *Server) handleGetCollectorHealth(w http.ResponseWriter, r *http.Request) {
	overall := s.healthTracker.GetOverallHealth()

	response := map[string]interface{}{
		"status":     overall.Status,
		"collectors": overall.Collectors,
		"summary": map[string]int{
			"healthy":  overall.Healthy,
			"degraded": overall.Degraded,
			"failed":   overall.Failed,
			"total":    overall.Total,
		},
		"checked_at": overall.CheckedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleTestCollector tests a specific collector on demand
// [REQ:SCS-HEALTH-003] Collector test endpoint
func (s *Server) handleTestCollector(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collectorName := vars["name"]

	// Define test functions for each collector type
	var testFn health.CollectorTestFunc
	switch collectorName {
	case "requirements":
		testFn = func() error {
			_, err := s.collector.Collect("scenario-completeness-scoring")
			return err
		}
	case "tests", "ui", "service":
		testFn = func() error {
			// These are part of the overall collect, so test via a sample scenario
			_, err := s.collector.Collect("scenario-completeness-scoring")
			return err
		}
	default:
		http.Error(w, fmt.Sprintf("Unknown collector: %s", collectorName), http.StatusNotFound)
		return
	}

	result := s.healthTracker.TestCollector(collectorName, testFn)

	// Also record in circuit breaker
	cb := s.cbRegistry.Get(collectorName)
	if result.Success {
		cb.RecordSuccess()
	} else {
		cb.RecordFailure()
	}

	response := map[string]interface{}{
		"name":        result.Name,
		"success":     result.Success,
		"duration_ms": result.Duration.Milliseconds(),
		"status":      result.Status,
	}
	if result.Error != "" {
		response["error"] = result.Error
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetCircuitBreakers returns status of all circuit breakers
// [REQ:SCS-CB-004] View auto-disabled components
func (s *Server) handleGetCircuitBreakers(w http.ResponseWriter, r *http.Request) {
	statuses := s.cbRegistry.ListStatus()
	stats := s.cbRegistry.Stats()
	openBreakers := s.cbRegistry.OpenBreakers()

	response := map[string]interface{}{
		"breakers": statuses,
		"stats": map[string]int{
			"total":     stats.Total,
			"closed":    stats.Closed,
			"open":      stats.Open,
			"half_open": stats.HalfOpen,
		},
		"tripped": openBreakers,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleResetAllCircuitBreakers resets all circuit breakers
// [REQ:SCS-CB-004] Reset all components
func (s *Server) handleResetAllCircuitBreakers(w http.ResponseWriter, r *http.Request) {
	count := s.cbRegistry.ResetAll()

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Reset %d circuit breakers", count),
		"count":   count,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleResetCircuitBreaker resets a specific circuit breaker
// [REQ:SCS-CB-004] Reset specific component
func (s *Server) handleResetCircuitBreaker(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collectorName := vars["collector"]

	success := s.cbRegistry.Reset(collectorName)
	if !success {
		http.Error(w, fmt.Sprintf("Circuit breaker not found: %s", collectorName), http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"success":   true,
		"collector": collectorName,
		"message":   fmt.Sprintf("Circuit breaker '%s' has been reset", collectorName),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetHistory returns score history for a scenario
// [REQ:SCS-HIST-001] Score history storage
// [REQ:SCS-HIST-004] History API endpoint
func (s *Server) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	// Parse limit from query params
	limit := 30
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	history, err := s.historyRepo.GetHistory(scenarioName, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get history: %v", err), http.StatusInternalServerError)
		return
	}

	count, _ := s.historyRepo.Count(scenarioName)

	response := map[string]interface{}{
		"scenario":   scenarioName,
		"snapshots":  history,
		"count":      len(history),
		"total":      count,
		"limit":      limit,
		"fetched_at": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetTrends returns trend analysis for a scenario
// [REQ:SCS-HIST-003] Trend detection
func (s *Server) handleGetTrends(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	// Parse limit from query params
	limit := 30
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	analysis, err := s.trendAnalyzer.Analyze(scenarioName, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to analyze trends: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"scenario":    scenarioName,
		"analysis":    analysis,
		"analyzed_at": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetAllTrends returns trend summary across all scenarios
// [REQ:SCS-HIST-003] Trend detection
func (s *Server) handleGetAllTrends(w http.ResponseWriter, r *http.Request) {
	// Parse limit from query params
	limit := 30
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	summary, err := s.trendAnalyzer.GetTrendSummary(limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get trend summary: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"summary":     summary,
		"analyzed_at": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleWhatIf performs what-if analysis for a scenario
// [REQ:SCS-ANALYSIS-001] What-if analysis API endpoint
func (s *Server) handleWhatIf(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req analysis.WhatIfRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if len(req.Changes) == 0 {
		http.Error(w, "No changes specified", http.StatusBadRequest)
		return
	}

	result, err := s.whatIfAnalyzer.Analyze(scenarioName, req.Changes)
	if err != nil {
		http.Error(w, fmt.Sprintf("Analysis failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"scenario":    scenarioName,
		"result":      result,
		"analyzed_at": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleBulkRefresh recalculates scores for all scenarios
// [REQ:SCS-ANALYSIS-003] Bulk score refresh API endpoint
func (s *Server) handleBulkRefresh(w http.ResponseWriter, r *http.Request) {
	result, err := s.bulkRefresher.RefreshAll()
	if err != nil {
		http.Error(w, fmt.Sprintf("Bulk refresh failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"result": result,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCompare compares multiple scenarios
// [REQ:SCS-ANALYSIS-004] Cross-scenario comparison API endpoint
func (s *Server) handleCompare(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req struct {
		Scenarios []string `json:"scenarios"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if len(req.Scenarios) < 2 {
		http.Error(w, "At least 2 scenarios required for comparison", http.StatusBadRequest)
		return
	}

	result, err := s.bulkRefresher.Compare(req.Scenarios)
	if err != nil {
		http.Error(w, fmt.Sprintf("Comparison failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"comparison": result,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetRecommendations returns recommendations for a specific scenario
// [REQ:SCS-ANALYSIS-002] Prioritized recommendations API endpoint
func (s *Server) handleGetRecommendations(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	// Collect metrics and calculate score
	metrics, err := s.collector.Collect(scenarioName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to collect metrics for scenario %s: %v", scenarioName, err), http.StatusNotFound)
		return
	}

	// Load config for scoring options
	cfg, _ := s.configLoader.LoadGlobal()
	scoringOpts := configToScoringOptions(cfg)

	thresholds := scoring.GetThresholds(metrics.Category)
	breakdown := scoring.CalculateCompletenessScoreWithOptions(*metrics, thresholds, nil, scoringOpts)
	recommendations := scoring.GenerateRecommendations(breakdown, thresholds)

	// Calculate total potential impact
	totalImpact := 0
	for _, rec := range recommendations {
		totalImpact += rec.Impact
	}

	response := map[string]interface{}{
		"scenario":        scenarioName,
		"current_score":   breakdown.Score,
		"recommendations": recommendations,
		"total_impact":    totalImpact,
		"potential_score": breakdown.Score + totalImpact,
		"generated_at":    time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleListAnalysisComponents returns the list of components available for what-if analysis
func (s *Server) handleListAnalysisComponents(w http.ResponseWriter, r *http.Request) {
	components := analysis.AvailableComponents()

	response := map[string]interface{}{
		"components": components,
		"total":      len(components),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleValidationAnalysis returns validation quality analysis for a scenario
// This detects anti-patterns and gaming behaviors in test validation
func (s *Server) handleValidationAnalysis(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	// Collect metrics to get counts
	metrics, err := s.collector.Collect(scenarioName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to collect metrics for scenario %s: %v", scenarioName, err), http.StatusNotFound)
		return
	}

	// Perform validation quality analysis
	scenarioRoot := s.collector.GetScenarioRoot(scenarioName)
	rawReqs := s.collector.LoadRequirements(scenarioName)
	validatorReqs := validators.ConvertRequirements(rawReqs)

	analysis := validators.AnalyzeValidationQuality(
		validators.MetricCounts{
			RequirementsTotal: metrics.Requirements.Total,
			TestsTotal:        metrics.Tests.Total,
		},
		validatorReqs,
		nil, // No operational targets yet
		scenarioRoot,
	)

	response := map[string]interface{}{
		"scenario":    scenarioName,
		"analysis":    analysis,
		"analyzed_at": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware prints simple request logs
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

func (s *Server) log(msg string, fields map[string]interface{}) {
	if len(fields) == 0 {
		log.Println(msg)
		return
	}
	log.Printf("%s | %v", msg, fields)
}

func requireEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		log.Fatalf("environment variable %s is required. Run the scenario via 'vrooli scenario run <name>' so lifecycle exports it.", key)
	}
	return value
}

func getEnvWithDefault(key, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	return value
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `This binary must be run through the Vrooli lifecycle system.

Instead, use:
   vrooli scenario start scenario-completeness-scoring

The lifecycle system provides environment variables, port allocation,
and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	server, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
