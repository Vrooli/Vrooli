package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ROIAnalysisEngine orchestrates all ROI analysis workflows
type ROIAnalysisEngine struct {
	db           *sql.DB
	httpClient   *http.Client
	ollamaClient *OllamaClient
	ctx          context.Context
	cancel       context.CancelFunc
}

// ComprehensiveAnalysisRequest contains all analysis parameters
type ComprehensiveAnalysisRequest struct {
	Idea        string   `json:"idea"`
	Budget      float64  `json:"budget"`
	Timeline    string   `json:"timeline"`
	Skills      []string `json:"skills"`
	MarketFocus string   `json:"market_focus"`
	Location    string   `json:"location,omitempty"`
	RiskTolerance string `json:"risk_tolerance,omitempty"` // low, medium, high
	Industry    string   `json:"industry,omitempty"`
}

// ComprehensiveAnalysisResponse contains full analysis results
type ComprehensiveAnalysisResponse struct {
	AnalysisID       string              `json:"analysis_id"`
	Timestamp        time.Time           `json:"timestamp"`
	Input            *ComprehensiveAnalysisRequest `json:"input"`
	ROIAnalysis      *ROIAnalysisResult  `json:"roi_analysis"`
	MarketResearch   *MarketResearchResult `json:"market_research"`
	FinancialMetrics *FinancialCalculationResult `json:"financial_metrics"`
	Competitive      *CompetitiveAnalysisResult `json:"competitive_analysis"`
	Summary          *ExecutiveSummary   `json:"executive_summary"`
	ExecutionTime    int64               `json:"execution_time_ms"`
	Success          bool                `json:"success"`
	Error            string              `json:"error,omitempty"`
}

// ROIAnalysisResult from the core ROI analyzer
type ROIAnalysisResult struct {
	ROIPercentage     float64  `json:"roi_percentage"`
	NPV               float64  `json:"npv"`
	IRR               float64  `json:"irr"`
	PaybackMonths     int      `json:"payback_months"`
	RiskLevel         string   `json:"risk_level"`
	ConfidenceScore   float64  `json:"confidence_score"`
	BreakevenPoint    float64  `json:"breakeven_point"`
	EstimatedRevenue  float64  `json:"estimated_revenue"`
	EstimatedCosts    float64  `json:"estimated_costs"`
	ProfitMargin      float64  `json:"profit_margin"`
	Recommendations   []string `json:"recommendations"`
}

// MarketResearchResult from market researcher workflows
type MarketResearchResult struct {
	MarketSize        float64             `json:"market_size"`
	MarketGrowthRate  float64             `json:"market_growth_rate"`
	TargetMarketSize  float64             `json:"target_market_size"`
	MarketSaturation  string              `json:"market_saturation"`
	CustomerDemand    string              `json:"customer_demand"`
	MarketTrends      []string            `json:"market_trends"`
	EntryBarriers     []string            `json:"entry_barriers"`
	Opportunities     []string            `json:"opportunities"`
	Threats           []string            `json:"threats"`
	PESTEL           *PESTELAnalysis      `json:"pestel_analysis"`
	KeyInsights       []string            `json:"key_insights"`
	DataSources       []string            `json:"data_sources"`
}

// PESTELAnalysis (Political, Economic, Social, Technological, Environmental, Legal)
type PESTELAnalysis struct {
	Political      []string `json:"political"`
	Economic       []string `json:"economic"`
	Social         []string `json:"social"`
	Technological  []string `json:"technological"`
	Environmental  []string `json:"environmental"`
	Legal          []string `json:"legal"`
}

// FinancialCalculationResult from financial calculator
type FinancialCalculationResult struct {
	InitialInvestment   float64            `json:"initial_investment"`
	MonthlyBurnRate     float64            `json:"monthly_burn_rate"`
	MonthlyRevenue      []float64          `json:"monthly_revenue_forecast"`
	CashFlowProjection  []float64          `json:"cash_flow_projection"`
	BreakevenMonth      int                `json:"breakeven_month"`
	FundingNeeded       float64            `json:"funding_needed"`
	RunwayMonths        int                `json:"runway_months"`
	CostStructure       map[string]float64 `json:"cost_structure"`
	RevenueStreams      map[string]float64 `json:"revenue_streams"`
	KeyAssumptions      []string           `json:"key_assumptions"`
	SensitivityAnalysis map[string]float64 `json:"sensitivity_analysis"`
}

// CompetitiveAnalysisResult from competitive landscape analyzer
type CompetitiveAnalysisResult struct {
	DirectCompetitors     []Competitor     `json:"direct_competitors"`
	IndirectCompetitors   []Competitor     `json:"indirect_competitors"`
	CompetitiveAdvantages []string         `json:"competitive_advantages"`
	Weaknesses            []string         `json:"weaknesses"`
	MarketPosition        string           `json:"market_position"`
	PricingStrategy       string           `json:"pricing_strategy"`
	DifferentiationOpp    []string         `json:"differentiation_opportunities"`
	ThreatLevel           string           `json:"threat_level"`
	CompetitiveScore      float64          `json:"competitive_score"`
}

// Competitor information
type Competitor struct {
	Name            string   `json:"name"`
	MarketShare     float64  `json:"market_share"`
	Strengths       []string `json:"strengths"`
	Weaknesses      []string `json:"weaknesses"`
	PricingModel    string   `json:"pricing_model"`
	ThreatLevel     string   `json:"threat_level"`
	FundingStatus   string   `json:"funding_status,omitempty"`
}

// ExecutiveSummary provides high-level insights
type ExecutiveSummary struct {
	OverallRating     string   `json:"overall_rating"`  // Excellent, Good, Fair, Poor
	InvestmentAdvice  string   `json:"investment_advice"`
	KeySuccessFactors []string `json:"key_success_factors"`
	MajorRisks        []string `json:"major_risks"`
	Timeline          string   `json:"timeline_assessment"`
	FinalRecommendation string `json:"final_recommendation"`
	ActionItems       []string `json:"action_items"`
	NextSteps         []string `json:"next_steps"`
}

// NewROIAnalysisEngine creates a new ROI analysis engine
func NewROIAnalysisEngine(db *sql.DB) *ROIAnalysisEngine {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ROIAnalysisEngine{
		db:           db,
		httpClient:   &http.Client{Timeout: 120 * time.Second},
		ollamaClient: NewOllamaClient(),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// PerformComprehensiveAnalysis runs all analysis workflows
func (engine *ROIAnalysisEngine) PerformComprehensiveAnalysis(req *ComprehensiveAnalysisRequest) (*ComprehensiveAnalysisResponse, error) {
	startTime := time.Now()
	analysisID := uuid.New().String()
	
	log.Printf("Starting comprehensive ROI analysis: %s for idea: %s", analysisID, req.Idea)
	
	response := &ComprehensiveAnalysisResponse{
		AnalysisID: analysisID,
		Timestamp:  time.Now(),
		Input:      req,
		Success:    false,
	}
	
	// Store the analysis request
	if err := engine.storeAnalysisRequest(analysisID, req); err != nil {
		log.Printf("Failed to store analysis request: %v", err)
	}
	
	// Run all analysis workflows in parallel for efficiency
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := []string{}
	
	// 1. Core ROI Analysis
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, err := engine.performROIAnalysis(req)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			errors = append(errors, fmt.Sprintf("ROI analysis failed: %v", err))
		} else {
			response.ROIAnalysis = result
		}
	}()
	
	// 2. Market Research
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, err := engine.performMarketResearch(req)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			errors = append(errors, fmt.Sprintf("Market research failed: %v", err))
		} else {
			response.MarketResearch = result
		}
	}()
	
	// 3. Financial Calculations
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, err := engine.performFinancialCalculations(req)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			errors = append(errors, fmt.Sprintf("Financial calculations failed: %v", err))
		} else {
			response.FinancialMetrics = result
		}
	}()
	
	// 4. Competitive Analysis
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, err := engine.performCompetitiveAnalysis(req)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			errors = append(errors, fmt.Sprintf("Competitive analysis failed: %v", err))
		} else {
			response.Competitive = result
		}
	}()
	
	// Wait for all analyses to complete
	wg.Wait()
	
	// Generate executive summary based on all results
	if len(errors) == 0 {
		summary, err := engine.generateExecutiveSummary(req, response)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Executive summary failed: %v", err))
		} else {
			response.Summary = summary
			response.Success = true
		}
	}
	
	response.ExecutionTime = time.Since(startTime).Milliseconds()
	
	if len(errors) > 0 {
		response.Error = strings.Join(errors, "; ")
		log.Printf("Analysis completed with errors: %s", response.Error)
	} else {
		log.Printf("Analysis completed successfully in %dms", response.ExecutionTime)
	}
	
	// Store the completed analysis
	engine.storeAnalysisResult(response)
	
	return response, nil
}

// performROIAnalysis replaces the roi-analyzer n8n workflow
func (engine *ROIAnalysisEngine) performROIAnalysis(req *ComprehensiveAnalysisRequest) (*ROIAnalysisResult, error) {
	prompt := fmt.Sprintf(`Perform comprehensive ROI analysis for the following business idea:

Business Idea: %s
Budget Available: $%.2f
Timeline: %s
Skills Available: %s
Market Focus: %s
Risk Tolerance: %s

Analyze and provide:
1. Estimated initial investment needed
2. Projected monthly revenue (conservative estimate)
3. ROI percentage (annual)
4. Payback period in months
5. Risk level assessment (low/medium/high)
6. Confidence score (0-100)
7. Key recommendations for maximizing ROI

Return as JSON: {"roi_percentage": 0, "npv": 0, "irr": 0, "payback_months": 0, "risk_level": "", "confidence_score": 0, "breakeven_point": 0, "estimated_revenue": 0, "estimated_costs": 0, "profit_margin": 0, "recommendations": []}`, 
		req.Idea, req.Budget, req.Timeline, strings.Join(req.Skills, ", "), req.MarketFocus, req.RiskTolerance)

	response, err := engine.ollamaClient.Generate(prompt, "llama3.2")
	if err != nil {
		return nil, err
	}
	
	var analysis ROIAnalysisResult
	if err := json.Unmarshal([]byte(response), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse ROI analysis response: %w", err)
	}
	
	return &analysis, nil
}

// performMarketResearch replaces market-researcher and enhanced-market-researcher n8n workflows
func (engine *ROIAnalysisEngine) performMarketResearch(req *ComprehensiveAnalysisRequest) (*MarketResearchResult, error) {
	prompt := fmt.Sprintf(`Conduct comprehensive market research for this business opportunity:

Business Idea: %s
Market Focus: %s
Target Industry: %s
Location: %s

Research and analyze:
1. Total addressable market (TAM) size in USD
2. Market growth rate percentage
3. Target market size and saturation level
4. Customer demand assessment
5. Current market trends (top 5)
6. Market entry barriers
7. Business opportunities and threats
8. PESTEL analysis (Political, Economic, Social, Technological, Environmental, Legal factors)
9. Key market insights
10. Data sources and validation

Provide realistic estimates based on current market conditions.

Return as JSON: {"market_size": 0, "market_growth_rate": 0, "target_market_size": 0, "market_saturation": "", "customer_demand": "", "market_trends": [], "entry_barriers": [], "opportunities": [], "threats": [], "pestel_analysis": {"political": [], "economic": [], "social": [], "technological": [], "environmental": [], "legal": []}, "key_insights": [], "data_sources": []}`,
		req.Idea, req.MarketFocus, req.Industry, req.Location)

	response, err := engine.ollamaClient.Generate(prompt, "llama3.2")
	if err != nil {
		return nil, err
	}
	
	var research MarketResearchResult
	if err := json.Unmarshal([]byte(response), &research); err != nil {
		return nil, fmt.Errorf("failed to parse market research response: %w", err)
	}
	
	return &research, nil
}

// performFinancialCalculations replaces the financial-calculator n8n workflow
func (engine *ROIAnalysisEngine) performFinancialCalculations(req *ComprehensiveAnalysisRequest) (*FinancialCalculationResult, error) {
	prompt := fmt.Sprintf(`Perform detailed financial calculations and projections for:

Business Idea: %s
Available Budget: $%.2f
Timeline: %s
Skills: %s

Calculate and project:
1. Initial investment requirements breakdown
2. Monthly burn rate estimation
3. Monthly revenue forecast for 24 months (realistic progression)
4. Monthly cash flow projection
5. Breakeven analysis (month when revenue > costs)
6. Total funding needed beyond initial budget
7. Financial runway in months
8. Cost structure breakdown (fixed vs variable costs)
9. Revenue stream analysis
10. Key financial assumptions
11. Sensitivity analysis (best/worst case scenarios)

Be realistic and conservative in estimates.

Return as JSON: {"initial_investment": 0, "monthly_burn_rate": 0, "monthly_revenue_forecast": [], "cash_flow_projection": [], "breakeven_month": 0, "funding_needed": 0, "runway_months": 0, "cost_structure": {}, "revenue_streams": {}, "key_assumptions": [], "sensitivity_analysis": {}}`,
		req.Idea, req.Budget, req.Timeline, strings.Join(req.Skills, ", "))

	response, err := engine.ollamaClient.Generate(prompt, "llama3.2")
	if err != nil {
		return nil, err
	}
	
	var calculations FinancialCalculationResult
	if err := json.Unmarshal([]byte(response), &calculations); err != nil {
		return nil, fmt.Errorf("failed to parse financial calculations response: %w", err)
	}
	
	return &calculations, nil
}

// performCompetitiveAnalysis replaces the competitive-landscape-analyzer n8n workflow
func (engine *ROIAnalysisEngine) performCompetitiveAnalysis(req *ComprehensiveAnalysisRequest) (*CompetitiveAnalysisResult, error) {
	prompt := fmt.Sprintf(`Analyze the competitive landscape for:

Business Idea: %s
Market Focus: %s
Industry: %s
Budget: $%.2f

Analyze and identify:
1. Top 5 direct competitors (name, market share, strengths, weaknesses, pricing)
2. Top 3 indirect competitors
3. Your competitive advantages given your skills: %s
4. Potential weaknesses compared to competitors
5. Market positioning opportunities
6. Recommended pricing strategy
7. Differentiation opportunities
8. Overall competitive threat level
9. Competitive score (0-100, higher = more competitive position)

Focus on realistic, current market players and positioning.

Return as JSON: {"direct_competitors": [{"name": "", "market_share": 0, "strengths": [], "weaknesses": [], "pricing_model": "", "threat_level": ""}], "indirect_competitors": [], "competitive_advantages": [], "weaknesses": [], "market_position": "", "pricing_strategy": "", "differentiation_opportunities": [], "threat_level": "", "competitive_score": 0}`,
		req.Idea, req.MarketFocus, req.Industry, req.Budget, strings.Join(req.Skills, ", "))

	response, err := engine.ollamaClient.Generate(prompt, "llama3.2")
	if err != nil {
		return nil, err
	}
	
	var competitive CompetitiveAnalysisResult
	if err := json.Unmarshal([]byte(response), &competitive); err != nil {
		return nil, fmt.Errorf("failed to parse competitive analysis response: %w", err)
	}
	
	return &competitive, nil
}

// generateExecutiveSummary creates final recommendations
func (engine *ROIAnalysisEngine) generateExecutiveSummary(req *ComprehensiveAnalysisRequest, analysis *ComprehensiveAnalysisResponse) (*ExecutiveSummary, error) {
	// Convert analysis components to JSON for context
	roiJSON, _ := json.Marshal(analysis.ROIAnalysis)
	marketJSON, _ := json.Marshal(analysis.MarketResearch)
	financialJSON, _ := json.Marshal(analysis.FinancialMetrics)
	competitiveJSON, _ := json.Marshal(analysis.Competitive)
	
	prompt := fmt.Sprintf(`Based on comprehensive analysis, create an executive summary for:

Business Idea: %s
Budget: $%.2f

Analysis Results:
ROI Analysis: %s
Market Research: %s
Financial Projections: %s
Competitive Analysis: %s

Synthesize findings and provide:
1. Overall investment rating (Excellent/Good/Fair/Poor)
2. Clear investment advice (Go/Caution/Stop)
3. Top 5 key success factors
4. Top 5 major risks
5. Timeline assessment and feasibility
6. Final recommendation with reasoning
7. Immediate action items (next 30 days)
8. Strategic next steps (3-6 months)

Be decisive and actionable in recommendations.

Return as JSON: {"overall_rating": "", "investment_advice": "", "key_success_factors": [], "major_risks": [], "timeline_assessment": "", "final_recommendation": "", "action_items": [], "next_steps": []}`,
		req.Idea, req.Budget, string(roiJSON), string(marketJSON), string(financialJSON), string(competitiveJSON))

	response, err := engine.ollamaClient.Generate(prompt, "llama3.2")
	if err != nil {
		return nil, err
	}
	
	var summary ExecutiveSummary
	if err := json.Unmarshal([]byte(response), &summary); err != nil {
		return nil, fmt.Errorf("failed to parse executive summary response: %w", err)
	}
	
	return &summary, nil
}

// Database operations

func (engine *ROIAnalysisEngine) storeAnalysisRequest(analysisID string, req *ComprehensiveAnalysisRequest) error {
	reqJSON, _ := json.Marshal(req)
	
	query := `
		INSERT INTO roi_analyses (id, idea, budget, timeline, skills, market_focus, 
		                         location, risk_tolerance, industry, request_data, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'processing', NOW())
	`
	
	skillsJSON, _ := json.Marshal(req.Skills)
	_, err := engine.db.Exec(query, analysisID, req.Idea, req.Budget, req.Timeline, 
		skillsJSON, req.MarketFocus, req.Location, req.RiskTolerance, req.Industry, reqJSON)
	
	return err
}

func (engine *ROIAnalysisEngine) storeAnalysisResult(response *ComprehensiveAnalysisResponse) {
	resultJSON, _ := json.Marshal(response)
	status := "completed"
	if !response.Success {
		status = "failed"
	}
	
	query := `
		UPDATE roi_analyses 
		SET result_data = $2, status = $3, execution_time_ms = $4,
		    success = $5, error_message = $6, completed_at = NOW()
		WHERE id = $1
	`
	
	engine.db.Exec(query, response.AnalysisID, resultJSON, status, 
		response.ExecutionTime, response.Success, response.Error)
}

// OllamaClient for AI generation (reused from previous implementation)
type OllamaClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewOllamaClient() *OllamaClient {
	return &OllamaClient{
		baseURL:    "http://localhost:11434", // Default Ollama URL
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (oc *OllamaClient) Generate(prompt, model string) (string, error) {
	// Use vrooli resource command to call Ollama
	cmd := exec.Command("vrooli", "resource", "ollama", "generate", prompt, 
		"--model", model, "--type", "financial-analysis", "--quiet")
	
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("ollama generation failed: %w", err)
	}
	
	return strings.TrimSpace(string(output)), nil
}