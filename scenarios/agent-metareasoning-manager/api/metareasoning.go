package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MetareasoningEngine orchestrates all reasoning workflows
type MetareasoningEngine struct {
	db              *sql.DB
	httpClient      *http.Client
	ollamaClient    *OllamaClient
	vectorStore     *VectorStore
	ctx             context.Context
	cancel          context.CancelFunc
	activeChains    map[string]*ReasoningChain
	chainsMutex     sync.RWMutex
}

// ReasoningRequest represents an incoming reasoning request
type ReasoningRequest struct {
	Input       string                 `json:"input"`
	Type        string                 `json:"type"` // pros_cons, swot, risk_assessment, self_review, reasoning_chain
	Model       string                 `json:"model,omitempty"`
	ChainType   string                 `json:"chain_type,omitempty"`
	CustomChain []ReasoningStep        `json:"custom_chain,omitempty"`
	Context     string                 `json:"context,omitempty"`
	Constraints string                 `json:"constraints,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ReasoningResponse contains the analysis results
type ReasoningResponse struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"`
	Analysis        interface{}            `json:"analysis"`
	Scores          map[string]float64     `json:"scores,omitempty"`
	Confidence      float64                `json:"confidence"`
	RecommendationStrength float64        `json:"recommendation_strength,omitempty"`
	Timestamp       time.Time              `json:"timestamp"`
	Model           string                 `json:"model"`
	ExecutionTime   int64                  `json:"execution_time_ms"`
	Success         bool                   `json:"success"`
	Error           string                 `json:"error,omitempty"`
	VectorID        string                 `json:"vector_id,omitempty"`
}

// ProsCons represents structured pros and cons analysis
type ProsCons struct {
	Pros           []WeightedItem `json:"pros"`
	Cons           []WeightedItem `json:"cons"`
	Recommendation string         `json:"recommendation"`
	Confidence     float64        `json:"confidence"`
}

// WeightedItem represents an item with importance weight
type WeightedItem struct {
	Item        string  `json:"item"`
	Weight      float64 `json:"weight"`
	Explanation string  `json:"explanation"`
}

// SWOTAnalysis represents strategic SWOT analysis
type SWOTAnalysis struct {
	Strengths      []WeightedItem `json:"strengths"`
	Weaknesses     []WeightedItem `json:"weaknesses"`
	Opportunities  []WeightedItem `json:"opportunities"`
	Threats        []WeightedItem `json:"threats"`
	Strategic      string         `json:"strategic_recommendation"`
	Priority       string         `json:"priority_focus"`
	Confidence     float64        `json:"confidence"`
}

// RiskAssessment represents risk analysis
type RiskAssessment struct {
	Risks          []Risk    `json:"risks"`
	Mitigations    []Risk    `json:"mitigations"`
	OverallRisk    string    `json:"overall_risk_level"`
	Recommendation string    `json:"recommendation"`
	Confidence     float64   `json:"confidence"`
}

// Risk represents a single risk or mitigation
type Risk struct {
	Description string  `json:"description"`
	Probability float64 `json:"probability"`
	Impact      float64 `json:"impact"`
	Severity    float64 `json:"severity"`
	Category    string  `json:"category"`
}

// SelfReview represents metacognitive analysis
type SelfReview struct {
	BiasesIdentified    []string `json:"biases_identified"`
	AssumptionsChecked  []string `json:"assumptions_checked"`
	AlternativeViews    []string `json:"alternative_views"`
	LogicalFallacies    []string `json:"logical_fallacies"`
	ConfidenceLevel     float64  `json:"confidence_level"`
	RecommendedActions  []string `json:"recommended_actions"`
	QualityAssessment   string   `json:"quality_assessment"`
}

// ReasoningChain orchestrates multi-step reasoning
type ReasoningChain struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Input       string            `json:"input"`
	Steps       []ReasoningStep   `json:"steps"`
	CurrentStep int               `json:"current_step"`
	Results     []StepResult      `json:"results"`
	Status      string            `json:"status"`
	StartedAt   time.Time         `json:"started_at"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	FinalResult *ReasoningResponse `json:"final_result,omitempty"`
}

// ReasoningStep defines a step in the reasoning chain
type ReasoningStep struct {
	Step        string `json:"step"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// StepResult contains the result of a reasoning step
type StepResult struct {
	Step      string      `json:"step"`
	Result    interface{} `json:"result"`
	Timestamp time.Time   `json:"timestamp"`
	Duration  int64       `json:"duration_ms"`
}

// OllamaClient handles communication with Ollama
type OllamaClient struct {
	baseURL    string
	httpClient *http.Client
}

// VectorStore handles vector operations
type VectorStore struct {
	qdrantURL string
	httpClient *http.Client
}

// NewMetareasoningEngine creates a new metareasoning engine
func NewMetareasoningEngine(db *sql.DB) *MetareasoningEngine {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &MetareasoningEngine{
		db:           db,
		httpClient:   &http.Client{Timeout: 120 * time.Second},
		ollamaClient: NewOllamaClient(),
		vectorStore:  NewVectorStore(),
		ctx:          ctx,
		cancel:       cancel,
		activeChains: make(map[string]*ReasoningChain),
	}
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient() *OllamaClient {
	return &OllamaClient{
		baseURL:    getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// NewVectorStore creates a new vector store client
func NewVectorStore() *VectorStore {
	return &VectorStore{
		qdrantURL:  getEnv("QDRANT_URL", "http://localhost:6333"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// ProcessReasoning handles all types of reasoning requests
func (me *MetareasoningEngine) ProcessReasoning(req *ReasoningRequest) (*ReasoningResponse, error) {
	startTime := time.Now()
	id := uuid.New().String()
	
	log.Printf("Processing %s reasoning request: %s", req.Type, id)
	
	var result interface{}
	var scores map[string]float64
	var confidence float64
	var err error
	
	switch req.Type {
	case "pros_cons":
		result, scores, confidence, err = me.analyzeProsCons(req)
	case "swot":
		result, scores, confidence, err = me.analyzeSWOT(req)
	case "risk_assessment":
		result, scores, confidence, err = me.assessRisk(req)
	case "self_review":
		result, scores, confidence, err = me.performSelfReview(req)
	case "reasoning_chain":
		return me.startReasoningChain(req)
	default:
		return nil, fmt.Errorf("unsupported reasoning type: %s", req.Type)
	}
	
	response := &ReasoningResponse{
		ID:            id,
		Type:          req.Type,
		Analysis:      result,
		Scores:        scores,
		Confidence:    confidence,
		Timestamp:     time.Now(),
		Model:         getModel(req.Model),
		ExecutionTime: time.Since(startTime).Milliseconds(),
		Success:       err == nil,
	}
	
	if err != nil {
		response.Error = err.Error()
		log.Printf("Reasoning request failed: %v", err)
	} else {
		// Store result and create vector embedding
		me.storeResult(response)
		vectorID := me.createVectorEmbedding(response)
		response.VectorID = vectorID
	}
	
	return response, nil
}

// analyzeProsCons performs pros and cons analysis
func (me *MetareasoningEngine) analyzeProsCons(req *ReasoningRequest) (interface{}, map[string]float64, float64, error) {
	prompt := fmt.Sprintf(`Analyze the following decision or proposal and create a comprehensive pros and cons list:

%s

Context: %s

Provide:
1. At least 5 pros (advantages, benefits, positive outcomes)
2. At least 5 cons (disadvantages, risks, negative outcomes)  
3. Weight each item on importance (1-10)
4. Overall recommendation
5. Confidence level (0-1)

Format as JSON with structure: {"pros": [{"item": "", "weight": 0, "explanation": ""}], "cons": [{"item": "", "weight": 0, "explanation": ""}], "recommendation": "", "confidence": 0}`, req.Input, req.Context)

	response, err := me.ollamaClient.Generate(prompt, getModel(req.Model))
	if err != nil {
		return nil, nil, 0, err
	}
	
	var analysis ProsCons
	if err := json.Unmarshal([]byte(response), &analysis); err != nil {
		return nil, nil, 0, fmt.Errorf("failed to parse pros/cons response: %w", err)
	}
	
	// Calculate scores
	prosScore := 0.0
	for _, pro := range analysis.Pros {
		prosScore += pro.Weight
	}
	
	consScore := 0.0
	for _, con := range analysis.Cons {
		consScore += con.Weight
	}
	
	netScore := prosScore - consScore
	totalScore := prosScore + consScore
	recommendationStrength := 0.0
	if totalScore > 0 {
		recommendationStrength = math.Abs(netScore) / totalScore
	}
	
	scores := map[string]float64{
		"pros_total":             prosScore,
		"cons_total":             consScore,
		"net_score":              netScore,
		"recommendation_strength": recommendationStrength,
	}
	
	return analysis, scores, analysis.Confidence, nil
}

// analyzeSWOT performs SWOT analysis
func (me *MetareasoningEngine) analyzeSWOT(req *ReasoningRequest) (interface{}, map[string]float64, float64, error) {
	prompt := fmt.Sprintf(`Perform a comprehensive SWOT analysis for:

%s

Context: %s

Analyze:
1. Strengths - internal positive factors (weight 1-10)
2. Weaknesses - internal negative factors (weight 1-10)
3. Opportunities - external positive factors (weight 1-10)
4. Threats - external negative factors (weight 1-10)

Provide strategic recommendation and priority focus area.

Format as JSON: {"strengths": [{"item": "", "weight": 0, "explanation": ""}], "weaknesses": [...], "opportunities": [...], "threats": [...], "strategic_recommendation": "", "priority_focus": "", "confidence": 0}`, req.Input, req.Context)

	response, err := me.ollamaClient.Generate(prompt, getModel(req.Model))
	if err != nil {
		return nil, nil, 0, err
	}
	
	var analysis SWOTAnalysis
	if err := json.Unmarshal([]byte(response), &analysis); err != nil {
		return nil, nil, 0, fmt.Errorf("failed to parse SWOT response: %w", err)
	}
	
	// Calculate SWOT scores
	strengthsScore := calculateItemsScore(analysis.Strengths)
	weaknessesScore := calculateItemsScore(analysis.Weaknesses)
	opportunitiesScore := calculateItemsScore(analysis.Opportunities)
	threatsScore := calculateItemsScore(analysis.Threats)
	
	scores := map[string]float64{
		"strengths":     strengthsScore,
		"weaknesses":    weaknessesScore,
		"opportunities": opportunitiesScore,
		"threats":       threatsScore,
		"internal_net":  strengthsScore - weaknessesScore,
		"external_net":  opportunitiesScore - threatsScore,
	}
	
	return analysis, scores, analysis.Confidence, nil
}

// assessRisk performs risk assessment
func (me *MetareasoningEngine) assessRisk(req *ReasoningRequest) (interface{}, map[string]float64, float64, error) {
	prompt := fmt.Sprintf(`Perform comprehensive risk assessment for the following action:

Action: %s
Constraints: %s
Context: %s

Analyze:
1. Identify potential risks with probability (0-1), impact (1-10), and severity calculation
2. Suggest mitigation strategies for each risk
3. Categorize risks (technical, business, operational, etc.)
4. Overall risk level assessment
5. Recommendation

Format as JSON: {"risks": [{"description": "", "probability": 0, "impact": 0, "severity": 0, "category": ""}], "mitigations": [...], "overall_risk_level": "", "recommendation": "", "confidence": 0}`, req.Input, req.Constraints, req.Context)

	response, err := me.ollamaClient.Generate(prompt, getModel(req.Model))
	if err != nil {
		return nil, nil, 0, err
	}
	
	var analysis RiskAssessment
	if err := json.Unmarshal([]byte(response), &analysis); err != nil {
		return nil, nil, 0, fmt.Errorf("failed to parse risk assessment response: %w", err)
	}
	
	// Calculate risk scores
	totalRisk := 0.0
	highRisks := 0
	for _, risk := range analysis.Risks {
		risk.Severity = risk.Probability * risk.Impact
		totalRisk += risk.Severity
		if risk.Severity > 7.0 {
			highRisks++
		}
	}
	
	scores := map[string]float64{
		"total_risk":       totalRisk,
		"average_risk":     totalRisk / float64(len(analysis.Risks)),
		"high_risk_count":  float64(highRisks),
		"risk_mitigation_ratio": float64(len(analysis.Mitigations)) / float64(len(analysis.Risks)),
	}
	
	return analysis, scores, analysis.Confidence, nil
}

// performSelfReview conducts metacognitive analysis
func (me *MetareasoningEngine) performSelfReview(req *ReasoningRequest) (interface{}, map[string]float64, float64, error) {
	prompt := fmt.Sprintf(`Perform a metacognitive self-review of the following reasoning or decision:

%s

Context: %s

Analyze:
1. Identify potential cognitive biases that may have influenced the reasoning
2. Check underlying assumptions that were made
3. Consider alternative perspectives or viewpoints
4. Look for logical fallacies in the reasoning
5. Assess the overall quality of the reasoning process
6. Recommend actions to improve the reasoning

Format as JSON: {"biases_identified": [], "assumptions_checked": [], "alternative_views": [], "logical_fallacies": [], "confidence_level": 0, "recommended_actions": [], "quality_assessment": ""}`, req.Input, req.Context)

	response, err := me.ollamaClient.Generate(prompt, getModel(req.Model))
	if err != nil {
		return nil, nil, 0, err
	}
	
	var analysis SelfReview
	if err := json.Unmarshal([]byte(response), &analysis); err != nil {
		return nil, nil, 0, fmt.Errorf("failed to parse self-review response: %w", err)
	}
	
	// Calculate metacognitive scores
	scores := map[string]float64{
		"biases_found":        float64(len(analysis.BiasesIdentified)),
		"assumptions_checked": float64(len(analysis.AssumptionsChecked)),
		"alternatives_considered": float64(len(analysis.AlternativeViews)),
		"fallacies_detected":  float64(len(analysis.LogicalFallacies)),
		"improvement_actions": float64(len(analysis.RecommendedActions)),
	}
	
	return analysis, scores, analysis.ConfidenceLevel, nil
}

// startReasoningChain initiates a multi-step reasoning chain
func (me *MetareasoningEngine) startReasoningChain(req *ReasoningRequest) (*ReasoningResponse, error) {
	chainID := uuid.New().String()
	
	// Define predefined chains
	chains := map[string][]ReasoningStep{
		"comprehensive": {
			{Step: "pros_cons", Type: "pros_cons", Description: "Initial pros and cons analysis"},
			{Step: "risk_assessment", Type: "risk_assessment", Description: "Assess potential risks"},
			{Step: "swot", Type: "swot", Description: "Strategic SWOT analysis"},
			{Step: "self_review", Type: "self_review", Description: "Review reasoning process"},
		},
		"strategic": {
			{Step: "swot", Type: "swot", Description: "Strategic SWOT analysis"},
			{Step: "risk_assessment", Type: "risk_assessment", Description: "Strategic risk assessment"},
			{Step: "pros_cons", Type: "pros_cons", Description: "Final pros and cons evaluation"},
		},
		"critical": {
			{Step: "self_review", Type: "self_review", Description: "Validate assumptions and check biases"},
			{Step: "pros_cons", Type: "pros_cons", Description: "Challenge from opposing view"},
			{Step: "risk_assessment", Type: "risk_assessment", Description: "Final risk evaluation"},
		},
	}
	
	var steps []ReasoningStep
	if len(req.CustomChain) > 0 {
		steps = req.CustomChain
	} else {
		chainType := req.ChainType
		if chainType == "" {
			chainType = "comprehensive"
		}
		steps = chains[chainType]
	}
	
	chain := &ReasoningChain{
		ID:          chainID,
		Type:        req.ChainType,
		Input:       req.Input,
		Steps:       steps,
		CurrentStep: 0,
		Results:     []StepResult{},
		Status:      "initiated",
		StartedAt:   time.Now(),
	}
	
	me.chainsMutex.Lock()
	me.activeChains[chainID] = chain
	me.chainsMutex.Unlock()
	
	// Start chain execution in background
	go me.executeReasoningChain(chain, getModel(req.Model), req.Context)
	
	response := &ReasoningResponse{
		ID:        chainID,
		Type:      "reasoning_chain",
		Analysis:  chain,
		Timestamp: time.Now(),
		Model:     getModel(req.Model),
		Success:   true,
	}
	
	return response, nil
}

// executeReasoningChain executes a reasoning chain step by step
func (me *MetareasoningEngine) executeReasoningChain(chain *ReasoningChain, model, context string) {
	defer func() {
		me.chainsMutex.Lock()
		delete(me.activeChains, chain.ID)
		me.chainsMutex.Unlock()
	}()
	
	for chain.CurrentStep < len(chain.Steps) {
		step := chain.Steps[chain.CurrentStep]
		stepStart := time.Now()
		
		log.Printf("Executing chain %s step %d: %s", chain.ID, chain.CurrentStep, step.Step)
		
		// Prepare step input
		stepReq := &ReasoningRequest{
			Input:   chain.Input,
			Type:    step.Type,
			Model:   model,
			Context: context,
		}
		
		// Add context from previous steps
		if len(chain.Results) > 0 {
			previousResults := []string{}
			for _, result := range chain.Results {
				resultJSON, _ := json.Marshal(result.Result)
				previousResults = append(previousResults, fmt.Sprintf("%s: %s", result.Step, string(resultJSON)))
			}
			stepReq.Context = fmt.Sprintf("%s\n\nPrevious reasoning steps:\n%s", context, strings.Join(previousResults, "\n\n"))
		}
		
		// Execute step
		response, err := me.ProcessReasoning(stepReq)
		
		stepResult := StepResult{
			Step:      step.Step,
			Timestamp: time.Now(),
			Duration:  time.Since(stepStart).Milliseconds(),
		}
		
		if err != nil {
			chain.Status = "failed"
			stepResult.Result = map[string]string{"error": err.Error()}
			chain.Results = append(chain.Results, stepResult)
			return
		}
		
		stepResult.Result = response.Analysis
		chain.Results = append(chain.Results, stepResult)
		chain.CurrentStep++
		
		chain.Status = "executing"
		if chain.CurrentStep >= len(chain.Steps) {
			chain.Status = "completed"
			now := time.Now()
			chain.CompletedAt = &now
			
			// Create final consolidated result
			chain.FinalResult = &ReasoningResponse{
				ID:        chain.ID + "_final",
				Type:      "reasoning_chain_result", 
				Analysis:  chain,
				Timestamp: time.Now(),
				Model:     model,
				Success:   true,
			}
			
			me.storeResult(chain.FinalResult)
		}
	}
}

// Generate calls Ollama to generate AI responses
func (oc *OllamaClient) Generate(prompt, model string) (string, error) {
	// For now, use CLI interface to Ollama via vrooli resource command
	cmd := exec.Command("vrooli", "resource", "ollama", "generate", prompt, "--model", model, "--type", "reasoning", "--quiet")
	
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ollama generation failed: %v, stderr: %s", err, stderr.String())
	}
	
	return strings.TrimSpace(out.String()), nil
}

// Helper functions

func calculateItemsScore(items []WeightedItem) float64 {
	total := 0.0
	for _, item := range items {
		total += item.Weight
	}
	return total
}

func getModel(model string) string {
	if model == "" {
		return "llama3.2"
	}
	return model
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// storeResult stores reasoning results in the database
func (me *MetareasoningEngine) storeResult(response *ReasoningResponse) {
	analysisJSON, _ := json.Marshal(response.Analysis)
	scoresJSON, _ := json.Marshal(response.Scores)
	
	query := `
		INSERT INTO reasoning_results (
			id, type, analysis, scores, confidence, 
			model, execution_time_ms, success, error_message,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
	`
	
	me.db.Exec(query,
		response.ID, response.Type, analysisJSON, scoresJSON,
		response.Confidence, response.Model, response.ExecutionTime,
		response.Success, response.Error,
	)
}

// createVectorEmbedding creates vector embeddings for semantic search
func (me *MetareasoningEngine) createVectorEmbedding(response *ReasoningResponse) string {
	// Create text representation for embedding
	text := fmt.Sprintf("Reasoning Type: %s\nAnalysis: %s", response.Type, fmt.Sprintf("%v", response.Analysis))
	
	// Use Ollama's embedding model
	cmd := exec.Command("vrooli", "resource", "ollama", "embeddings", text, "--model", "nomic-embed-text")
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to create embedding: %v", err)
		return ""
	}
	
	return response.ID + "_embedding"
}