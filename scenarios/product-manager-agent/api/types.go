package main

import (
	"time"
)

// Feature represents a product feature
type Feature struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Reach       int       `json:"reach"`       // Number of users affected
	Impact      int       `json:"impact"`      // Impact level (1-5)
	Confidence  float64   `json:"confidence"`  // Confidence level (0-1)
	Effort      int       `json:"effort"`      // Effort in story points
	Priority    string    `json:"priority"`    // CRITICAL, HIGH, MEDIUM, LOW
	Score       float64   `json:"score"`       // RICE score
	Status      string    `json:"status"`      // proposed, approved, in_progress, completed
	Dependencies []string `json:"dependencies"` // Feature IDs this depends on
	ROI         float64   `json:"roi"`         // Estimated ROI
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Roadmap represents a product roadmap
type Roadmap struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	Features   []string  `json:"features"` // Feature IDs
	Milestones []Milestone `json:"milestones"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
}

// Milestone represents a roadmap milestone
type Milestone struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Date        time.Time `json:"date"`
	Description string    `json:"description"`
	Features    []string  `json:"features"`
}

// SprintPlan represents a sprint planning session
type SprintPlan struct {
	ID             string    `json:"id"`
	SprintNumber   int       `json:"sprint_number"`
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	Capacity       int       `json:"capacity"`       // Total story points
	Features       []Feature `json:"features"`
	TotalEffort    int       `json:"total_effort"`
	EstimatedValue float64   `json:"estimated_value"`
	Velocity       float64   `json:"velocity"`
	RiskLevel      string    `json:"risk_level"`
	PlannedAt      time.Time `json:"planned_at"`
}

// MarketAnalysis represents market research results
type MarketAnalysis struct {
	ID            string    `json:"id"`
	ProductName   string    `json:"product_name"`
	MarketSize    string    `json:"market_size"`
	GrowthRate    string    `json:"growth_rate"`
	Competitors   []string  `json:"competitors"`
	Demographics  string    `json:"demographics"`
	Opportunities []string  `json:"opportunities"`
	Challenges    []string  `json:"challenges"`
	Timestamp     time.Time `json:"timestamp"`
}

// CompetitorAnalysis represents competitor research
type CompetitorAnalysis struct {
	ID             string    `json:"id"`
	CompetitorName string    `json:"competitor_name"`
	Features       []string  `json:"features"`
	Pricing        string    `json:"pricing"`
	TargetMarket   string    `json:"target_market"`
	Strengths      []string  `json:"strengths"`
	Weaknesses     []string  `json:"weaknesses"`
	MarketShare    string    `json:"market_share"`
	AnalyzedAt     time.Time `json:"analyzed_at"`
}

// FeedbackItem represents user feedback
type FeedbackItem struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Content    string    `json:"content"`
	Type       string    `json:"type"` // bug, feature_request, improvement, praise
	Sentiment  string    `json:"sentiment"`
	Priority   int       `json:"priority"`
	ReceivedAt time.Time `json:"received_at"`
}

// FeedbackAnalysis represents analyzed feedback
type FeedbackAnalysis struct {
	ID              string    `json:"id"`
	TotalItems      int       `json:"total_items"`
	Sentiment       string    `json:"sentiment"`
	SentimentScore  float64   `json:"sentiment_score"`
	KeyThemes       []string  `json:"key_themes"`
	FeatureRequests []string  `json:"feature_requests"`
	PainPoints      []string  `json:"pain_points"`
	AnalyzedAt      time.Time `json:"analyzed_at"`
}

// Decision represents a product decision
type Decision struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Options     []DecisionOption `json:"options"`
	Status      string    `json:"status"` // pending, decided, implemented
	DecidedAt   time.Time `json:"decided_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// DecisionOption represents an option in a decision
type DecisionOption struct {
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	Pros               []string `json:"pros"`
	Cons               []string `json:"cons"`
	RiskLevel          string   `json:"risk_level"`
	Complexity         string   `json:"complexity"`
	Timeline           string   `json:"timeline"`
	SuccessProbability float64  `json:"success_probability"`
	Score              float64  `json:"score"`
}

// DecisionAnalysis represents analyzed decision options
type DecisionAnalysis struct {
	DecisionID string           `json:"decision_id"`
	Options    []DecisionOption `json:"options"`
	Recommendation string       `json:"recommendation"`
	AnalyzedAt time.Time        `json:"analyzed_at"`
}

// ROICalculation represents ROI analysis for a feature
type ROICalculation struct {
	ID            string    `json:"id"`
	FeatureID     string    `json:"feature_id"`
	RevenueImpact float64   `json:"revenue_impact"`
	CostEstimate  float64   `json:"cost_estimate"`
	ROI           float64   `json:"roi"`
	PaybackPeriod float64   `json:"payback_period"` // in months
	Assumptions   []string  `json:"assumptions"`
	CalculatedAt  time.Time `json:"calculated_at"`
}

// Dashboard represents the product dashboard metrics
type Dashboard struct {
	Metrics        DashboardMetrics `json:"metrics"`
	RecentFeatures []Feature        `json:"recent_features"`
	CurrentSprint  *SprintPlan      `json:"current_sprint"`
	Roadmap        *Roadmap         `json:"roadmap"`
}

// DashboardMetrics represents key metrics
type DashboardMetrics struct {
	ActiveFeatures   int     `json:"active_features"`
	SprintProgress   int     `json:"sprint_progress"`
	TeamVelocity     float64 `json:"team_velocity"`
	CustomerNPS      int     `json:"customer_nps"`
	CompletedTasks   int     `json:"completed_tasks"`
	PendingDecisions int     `json:"pending_decisions"`
}