package pipeline

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
	
	"github.com/google/uuid"
)

// Pipeline orchestrates the entire scenario generation process
type Pipeline struct {
	db           *sql.DB
	claude       *ClaudeClient
	fileManager  *FileManager
	validator    *Validator
}

// NewPipeline creates a new generation pipeline
func NewPipeline(db *sql.DB) *Pipeline {
	return &Pipeline{
		db:          db,
		claude:      NewClaudeClient(),
		fileManager: NewFileManager(),
		validator:   NewValidator(),
	}
}

// GenerationRequest contains the parameters for generating a scenario
type GenerationRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Prompt      string            `json:"prompt"`
	Complexity  string            `json:"complexity"`
	Category    string            `json:"category"`
	Resources   []string          `json:"resources"`
	Iterations  IterationLimits   `json:"iterations"`
	Metadata    map[string]string `json:"metadata"`
}

// IterationLimits defines max iterations for each phase
type IterationLimits struct {
	Planning       int `json:"planning"`
	Implementation int `json:"implementation"`
	Validation     int `json:"validation"`
}

// GenerationResult contains the complete generation output
type GenerationResult struct {
	ScenarioID       string                 `json:"scenario_id"`
	GenerationID     string                 `json:"generation_id"`
	Status           string                 `json:"status"`
	Files            map[string]string      `json:"files"`
	Resources        []string               `json:"resources"`
	Metrics          GenerationMetrics      `json:"metrics"`
	ValidationResult ValidationResult       `json:"validation_result"`
	Error            string                 `json:"error,omitempty"`
}

// GenerationMetrics tracks performance and quality metrics
type GenerationMetrics struct {
	StartTime         time.Time     `json:"start_time"`
	EndTime           time.Time     `json:"end_time"`
	TotalDuration     time.Duration `json:"total_duration"`
	PlanningTime      time.Duration `json:"planning_time"`
	ImplementationTime time.Duration `json:"implementation_time"`
	ValidationTime    time.Duration `json:"validation_time"`
	PlanIterations    int           `json:"plan_iterations"`
	ImplIterations    int           `json:"impl_iterations"`
	ValIterations     int           `json:"val_iterations"`
	FilesGenerated    int           `json:"files_generated"`
	EstimatedRevenue  int           `json:"estimated_revenue"`
}

// Generate runs the complete generation pipeline
func (p *Pipeline) Generate(req GenerationRequest) (*GenerationResult, error) {
	startTime := time.Now()
	
	// Initialize result
	result := &GenerationResult{
		ScenarioID:   uuid.New().String(),
		GenerationID: uuid.New().String(),
		Status:       "started",
		Files:        make(map[string]string),
		Resources:    []string{},
		Metrics: GenerationMetrics{
			StartTime: startTime,
		},
	}
	
	// Set default iteration limits if not provided
	if req.Iterations.Planning == 0 {
		req.Iterations.Planning = 3
	}
	if req.Iterations.Implementation == 0 {
		req.Iterations.Implementation = 2
	}
	if req.Iterations.Validation == 0 {
		req.Iterations.Validation = 5
	}
	
	// Store initial scenario in database
	if err := p.createScenarioRecord(result.ScenarioID, req); err != nil {
		return nil, fmt.Errorf("failed to create scenario record: %w", err)
	}
	
	// Update status callback
	updateStatus := func(status string) {
		result.Status = status
		p.updateScenarioStatus(result.ScenarioID, status)
	}
	
	// Phase 1: Planning
	log.Printf("Starting planning phase for scenario %s", req.Name)
	updateStatus("planning")
	planStartTime := time.Now()
	
	plan, planMetrics, err := p.runPlanningPhase(req)
	if err != nil {
		result.Error = fmt.Sprintf("Planning failed: %v", err)
		updateStatus("failed")
		p.logError(result.ScenarioID, "planning", err)
		return result, err
	}
	
	result.Metrics.PlanningTime = time.Since(planStartTime)
	result.Metrics.PlanIterations = planMetrics.Iterations
	result.Resources = planMetrics.IdentifiedResources
	
	// Phase 2: Implementation
	log.Printf("Starting implementation phase for scenario %s", req.Name)
	updateStatus("implementing")
	implStartTime := time.Now()
	
	files, implMetrics, err := p.runImplementationPhase(req, plan)
	if err != nil {
		result.Error = fmt.Sprintf("Implementation failed: %v", err)
		updateStatus("failed")
		p.logError(result.ScenarioID, "implementation", err)
		return result, err
	}
	
	result.Files = files
	result.Metrics.ImplementationTime = time.Since(implStartTime)
	result.Metrics.ImplIterations = implMetrics.Iterations
	result.Metrics.FilesGenerated = len(files)
	
	// Phase 3: Validation
	log.Printf("Starting validation phase for scenario %s", req.Name)
	updateStatus("validating")
	valStartTime := time.Now()
	
	validationResult, valMetrics, err := p.runValidationPhase(req, files)
	if err != nil {
		result.Error = fmt.Sprintf("Validation failed: %v", err)
		updateStatus("failed")
		p.logError(result.ScenarioID, "validation", err)
		return result, err
	}
	
	result.ValidationResult = validationResult
	result.Metrics.ValidationTime = time.Since(valStartTime)
	result.Metrics.ValIterations = valMetrics.Iterations
	
	// Calculate final metrics
	result.Metrics.EndTime = time.Now()
	result.Metrics.TotalDuration = result.Metrics.EndTime.Sub(result.Metrics.StartTime)
	result.Metrics.EstimatedRevenue = p.estimateRevenue(req.Complexity)
	
	// Update final status
	if validationResult.Success {
		updateStatus("completed")
		log.Printf("Successfully generated scenario %s in %v", req.Name, result.Metrics.TotalDuration)
	} else {
		updateStatus("validation_failed")
		result.Error = "Scenario generated but validation failed"
	}
	
	// Store final results
	p.storeGenerationResults(result)
	
	return result, nil
}

// GenerateAsync runs generation in background and returns immediately
func (p *Pipeline) GenerateAsync(req GenerationRequest) (string, error) {
	// Create initial scenario record
	scenarioID := uuid.New().String()
	if err := p.createScenarioRecord(scenarioID, req); err != nil {
		return "", err
	}
	
	// Run generation in background
	go func() {
		result, err := p.Generate(req)
		if err != nil {
			log.Printf("Async generation failed for %s: %v", scenarioID, err)
		} else {
			log.Printf("Async generation completed for %s", scenarioID)
		}
		_ = result // Result is stored in DB by Generate()
	}()
	
	return scenarioID, nil
}

// GetStatus retrieves the current status of a generation
func (p *Pipeline) GetStatus(scenarioID string) (map[string]interface{}, error) {
	query := `
		SELECT status, generation_id, created_at, updated_at, completed_at, 
		       generation_error, estimated_revenue
		FROM scenarios 
		WHERE id = $1
	`
	
	var status string
	var generationID sql.NullString
	var createdAt, updatedAt time.Time
	var completedAt sql.NullTime
	var generationError sql.NullString
	var estimatedRevenue int
	
	err := p.db.QueryRow(query, scenarioID).Scan(
		&status, &generationID, &createdAt, &updatedAt, 
		&completedAt, &generationError, &estimatedRevenue,
	)
	
	if err != nil {
		return nil, err
	}
	
	result := map[string]interface{}{
		"scenario_id":       scenarioID,
		"status":           status,
		"created_at":       createdAt,
		"updated_at":       updatedAt,
		"estimated_revenue": estimatedRevenue,
	}
	
	if generationID.Valid {
		result["generation_id"] = generationID.String
	}
	if completedAt.Valid {
		result["completed_at"] = completedAt.Time
		result["duration"] = completedAt.Time.Sub(createdAt)
	}
	if generationError.Valid {
		result["error"] = generationError.String
	}
	
	// Calculate progress percentage
	progressMap := map[string]int{
		"requested":    0,
		"planning":     25,
		"implementing": 50,
		"validating":   75,
		"completed":    100,
		"failed":       0,
	}
	
	if progress, ok := progressMap[status]; ok {
		result["progress"] = progress
	}
	
	return result, nil
}

// createScenarioRecord creates initial database record
func (p *Pipeline) createScenarioRecord(scenarioID string, req GenerationRequest) error {
	resourcesJSON, _ := json.Marshal(req.Resources)
	
	query := `
		INSERT INTO scenarios (
			id, name, description, prompt, status, 
			complexity, category, estimated_revenue, 
			resources_used, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	
	_, err := p.db.Exec(query,
		scenarioID, req.Name, req.Description, req.Prompt, "requested",
		req.Complexity, req.Category, p.estimateRevenue(req.Complexity),
		resourcesJSON, time.Now(), time.Now(),
	)
	
	return err
}

// updateScenarioStatus updates the scenario status in database
func (p *Pipeline) updateScenarioStatus(scenarioID string, status string) error {
	query := `UPDATE scenarios SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := p.db.Exec(query, status, time.Now(), scenarioID)
	return err
}

// storeGenerationResults stores final generation results
func (p *Pipeline) storeGenerationResults(result *GenerationResult) error {
	filesJSON, _ := json.Marshal(result.Files)
	resourcesJSON, _ := json.Marshal(result.Resources)
	
	query := `
		UPDATE scenarios SET 
			files_generated = $1,
			resources_used = $2,
			status = $3,
			generation_id = $4,
			completed_at = $5,
			updated_at = $6
		WHERE id = $7
	`
	
	_, err := p.db.Exec(query,
		filesJSON, resourcesJSON, result.Status, result.GenerationID,
		result.Metrics.EndTime, time.Now(), result.ScenarioID,
	)
	
	return err
}

// logError logs an error to the generation_logs table
func (p *Pipeline) logError(scenarioID string, phase string, err error) {
	logQuery := `
		INSERT INTO generation_logs (scenario_id, step, prompt, success, error_message, started_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	
	p.db.Exec(logQuery, scenarioID, phase+"_error", "", false, err.Error(), time.Now())
}

// estimateRevenue estimates revenue based on complexity
func (p *Pipeline) estimateRevenue(complexity string) int {
	switch complexity {
	case "simple":
		return 15000
	case "advanced":
		return 40000
	default:
		return 25000
	}
}