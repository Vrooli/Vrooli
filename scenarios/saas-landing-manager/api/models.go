package main

import "time"

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
