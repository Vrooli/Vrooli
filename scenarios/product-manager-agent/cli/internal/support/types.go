package support

import (
	"time"
)

// Feature mirrors the API's Feature shape exposed via /api/features.
type Feature struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	Reach        int       `json:"reach,omitempty"`
	Impact       int       `json:"impact,omitempty"`
	Confidence   float64   `json:"confidence,omitempty"`
	Effort       int       `json:"effort,omitempty"`
	Priority     string    `json:"priority,omitempty"`
	Score        float64   `json:"score,omitempty"`
	Status       string    `json:"status,omitempty"`
	Dependencies []string  `json:"dependencies,omitempty"`
	ROI          float64   `json:"roi,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
}

// Milestone is a roadmap milestone.
type Milestone struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Date        time.Time `json:"date"`
	Description string    `json:"description,omitempty"`
	Features    []string  `json:"features,omitempty"`
}

// Roadmap mirrors /api/roadmap.
type Roadmap struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	StartDate  time.Time   `json:"start_date"`
	EndDate    time.Time   `json:"end_date"`
	Features   []string    `json:"features,omitempty"`
	Milestones []Milestone `json:"milestones,omitempty"`
	Version    int         `json:"version,omitempty"`
	CreatedAt  time.Time   `json:"created_at,omitempty"`
}

// SprintPlan mirrors /api/sprint responses.
type SprintPlan struct {
	ID             string    `json:"id,omitempty"`
	SprintNumber   int       `json:"sprint_number,omitempty"`
	StartDate      time.Time `json:"start_date,omitempty"`
	EndDate        time.Time `json:"end_date,omitempty"`
	Capacity       int       `json:"capacity"`
	Features       []Feature `json:"features,omitempty"`
	TotalEffort    int       `json:"total_effort,omitempty"`
	EstimatedValue float64   `json:"estimated_value,omitempty"`
	Velocity       float64   `json:"velocity,omitempty"`
	RiskLevel      string    `json:"risk_level,omitempty"`
	PlannedAt      time.Time `json:"planned_at,omitempty"`
}

// MarketAnalysis mirrors /api/market/analyze.
type MarketAnalysis struct {
	ID            string    `json:"id,omitempty"`
	ProductName   string    `json:"product_name"`
	MarketSize    string    `json:"market_size,omitempty"`
	GrowthRate    string    `json:"growth_rate,omitempty"`
	Competitors   []string  `json:"competitors,omitempty"`
	Demographics  string    `json:"demographics,omitempty"`
	Opportunities []string  `json:"opportunities,omitempty"`
	Challenges    []string  `json:"challenges,omitempty"`
	Timestamp     time.Time `json:"timestamp,omitempty"`
}

// CompetitorAnalysis mirrors /api/competitor/analyze.
type CompetitorAnalysis struct {
	ID             string    `json:"id,omitempty"`
	CompetitorName string    `json:"competitor_name"`
	Features       []string  `json:"features,omitempty"`
	Pricing        string    `json:"pricing,omitempty"`
	TargetMarket   string    `json:"target_market,omitempty"`
	Strengths      []string  `json:"strengths,omitempty"`
	Weaknesses     []string  `json:"weaknesses,omitempty"`
	MarketShare    string    `json:"market_share,omitempty"`
	AnalyzedAt     time.Time `json:"analyzed_at,omitempty"`
}

// FeedbackAnalysis mirrors /api/feedback/analyze.
type FeedbackAnalysis struct {
	ID              string    `json:"id,omitempty"`
	TotalItems      int       `json:"total_items"`
	Sentiment       string    `json:"sentiment,omitempty"`
	SentimentScore  float64   `json:"sentiment_score,omitempty"`
	KeyThemes       []string  `json:"key_themes,omitempty"`
	FeatureRequests []string  `json:"feature_requests,omitempty"`
	PainPoints      []string  `json:"pain_points,omitempty"`
	AnalyzedAt      time.Time `json:"analyzed_at,omitempty"`
}

// ROICalculation mirrors /api/roi/calculate.
type ROICalculation struct {
	ID            string    `json:"id,omitempty"`
	FeatureID     string    `json:"feature_id,omitempty"`
	RevenueImpact float64   `json:"revenue_impact"`
	CostEstimate  float64   `json:"cost_estimate"`
	ROI           float64   `json:"roi"`
	PaybackPeriod float64   `json:"payback_period"`
	Assumptions   []string  `json:"assumptions,omitempty"`
	CalculatedAt  time.Time `json:"calculated_at,omitempty"`
}

// DecisionOption is one option in a decision analysis.
type DecisionOption struct {
	Name               string   `json:"name"`
	Description        string   `json:"description,omitempty"`
	Pros               []string `json:"pros,omitempty"`
	Cons               []string `json:"cons,omitempty"`
	RiskLevel          string   `json:"risk_level,omitempty"`
	Complexity         string   `json:"complexity,omitempty"`
	Timeline           string   `json:"timeline,omitempty"`
	SuccessProbability float64  `json:"success_probability,omitempty"`
	Score              float64  `json:"score,omitempty"`
}

// DecisionAnalysis mirrors /api/decision/analyze.
type DecisionAnalysis struct {
	DecisionID     string           `json:"decision_id,omitempty"`
	Options        []DecisionOption `json:"options,omitempty"`
	Recommendation string           `json:"recommendation,omitempty"`
	AnalyzedAt     time.Time        `json:"analyzed_at,omitempty"`
}

// DashboardMetrics is the numeric block inside the dashboard response.
type DashboardMetrics struct {
	ActiveFeatures   int     `json:"active_features,omitempty"`
	SprintProgress   int     `json:"sprint_progress,omitempty"`
	TeamVelocity     float64 `json:"team_velocity,omitempty"`
	CustomerNPS      int     `json:"customer_nps,omitempty"`
	CompletedTasks   int     `json:"completed_tasks,omitempty"`
	PendingDecisions int     `json:"pending_decisions,omitempty"`
}

// Dashboard mirrors /api/dashboard.
type Dashboard struct {
	Metrics        DashboardMetrics `json:"metrics"`
	RecentFeatures []Feature        `json:"recent_features,omitempty"`
	CurrentSprint  *SprintPlan      `json:"current_sprint,omitempty"`
	Roadmap        *Roadmap         `json:"roadmap,omitempty"`
}

// PrioritizeResponse mirrors the /api/features/rice envelope.
type PrioritizeResponse struct {
	PrioritizedFeatures []Feature `json:"prioritized_features"`
	Total               int       `json:"total"`
}
