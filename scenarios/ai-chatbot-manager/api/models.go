package main

import (
	"time"
)

// Database models
type Chatbot struct {
	ID               string                 `json:"id"`
	TenantID         string                 `json:"tenant_id,omitempty"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Personality      string                 `json:"personality"`
	KnowledgeBase    string                 `json:"knowledge_base"`
	ModelConfig      map[string]interface{} `json:"model_config"`
	WidgetConfig     map[string]interface{} `json:"widget_config"`
	EscalationConfig map[string]interface{} `json:"escalation_config"`
	IsActive         bool                   `json:"is_active"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// Tenant model for multi-tenant support
type Tenant struct {
	ID                       string                 `json:"id"`
	Name                     string                 `json:"name"`
	Slug                     string                 `json:"slug"`
	Description              string                 `json:"description,omitempty"`
	Config                   map[string]interface{} `json:"config,omitempty"`
	Plan                     string                 `json:"plan"`
	MaxChatbots              int                    `json:"max_chatbots"`
	MaxConversationsPerMonth int                    `json:"max_conversations_per_month"`
	APIKey                   string                 `json:"api_key,omitempty"`
	IsActive                 bool                   `json:"is_active"`
	CreatedAt                time.Time              `json:"created_at"`
	UpdatedAt                time.Time              `json:"updated_at"`
}

// ABTest model for A/B testing chatbot personalities
type ABTest struct {
	ID           string                 `json:"id"`
	ChatbotID    string                 `json:"chatbot_id"`
	TenantID     string                 `json:"tenant_id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	VariantA     map[string]interface{} `json:"variant_a"`
	VariantB     map[string]interface{} `json:"variant_b"`
	TrafficSplit float64                `json:"traffic_split"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
	Status       string                 `json:"status"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	EndedAt      *time.Time             `json:"ended_at,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// CRMIntegration model for CRM system integration
type CRMIntegration struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	ChatbotID   *string                `json:"chatbot_id,omitempty"`
	Type        string                 `json:"type"` // salesforce, hubspot, pipedrive, webhook
	Config      map[string]interface{} `json:"config"`
	SyncEnabled bool                   `json:"sync_enabled"`
	LastSyncAt  *time.Time             `json:"last_sync_at,omitempty"`
	SyncStatus  string                 `json:"sync_status"`
	SyncErrors  map[string]interface{} `json:"sync_errors,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type Conversation struct {
	ID            string                 `json:"id"`
	ChatbotID     string                 `json:"chatbot_id"`
	SessionID     string                 `json:"session_id"`
	UserIP        string                 `json:"user_ip,omitempty"`
	StartedAt     time.Time              `json:"started_at"`
	EndedAt       *time.Time             `json:"ended_at,omitempty"`
	LeadCaptured  bool                   `json:"lead_captured"`
	LeadData      map[string]interface{} `json:"lead_data,omitempty"`
	ABTestID      *string                `json:"ab_test_id,omitempty"`
	ABTestVariant *string                `json:"ab_test_variant,omitempty"`
}

type Message struct {
	ID             string                 `json:"id"`
	ConversationID string                 `json:"conversation_id"`
	Role           string                 `json:"role"`
	Content        string                 `json:"content"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
}

type ChatRequest struct {
	Message   string                 `json:"message"`
	SessionID string                 `json:"session_id"`
	Context   map[string]interface{} `json:"context,omitempty"`
	UserIP    string                 `json:"user_ip,omitempty"`
}

type ChatResponse struct {
	Response          string                 `json:"response"`
	Confidence        float64                `json:"confidence"`
	ShouldEscalate    bool                   `json:"should_escalate"`
	EscalationReason  string                 `json:"escalation_reason,omitempty"`
	LeadQualification map[string]interface{} `json:"lead_qualification,omitempty"`
	ConversationID    string                 `json:"conversation_id"`
}

type Escalation struct {
	ID              string                 `json:"id"`
	ConversationID  string                 `json:"conversation_id"`
	ChatbotID       string                 `json:"chatbot_id"`
	Reason          string                 `json:"reason"`
	ConfidenceScore float64                `json:"confidence_score"`
	EscalationType  string                 `json:"escalation_type"`
	Status          string                 `json:"status"`
	EscalatedAt     time.Time              `json:"escalated_at"`
	ResolvedAt      *time.Time             `json:"resolved_at,omitempty"`
	ResolutionNotes string                 `json:"resolution_notes,omitempty"`
	WebhookResponse map[string]interface{} `json:"webhook_response,omitempty"`
	EmailSent       bool                   `json:"email_sent"`
}

type OllamaRequest struct {
	Model    string                 `json:"model"`
	Messages []OllamaMessage        `json:"messages"`
	Stream   bool                   `json:"stream"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OllamaResponse struct {
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

type AnalyticsData struct {
	TotalConversations    int               `json:"total_conversations"`
	TotalMessages         int               `json:"total_messages"`
	LeadsCaptured         int               `json:"leads_captured"`
	AvgConversationLength float64           `json:"avg_conversation_length"`
	EngagementScore       float64           `json:"engagement_score"`
	ConversionRate        float64           `json:"conversion_rate"`
	TopIntents            []Intent          `json:"top_intents"`
	ConversionFunnel      *ConversionFunnel `json:"conversion_funnel,omitempty"`
	UserJourney           *UserJourney      `json:"user_journey,omitempty"`
	HourlyDistribution    []HourlyMetric    `json:"hourly_distribution,omitempty"`
}

type Intent struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type ConversionFunnel struct {
	TotalVisitors     int     `json:"total_visitors"`
	EngagedVisitors   int     `json:"engaged_visitors"`
	QualifiedLeads    int     `json:"qualified_leads"`
	CapturedLeads     int     `json:"captured_leads"`
	EngagementRate    float64 `json:"engagement_rate"`
	QualificationRate float64 `json:"qualification_rate"`
	CaptureRate       float64 `json:"capture_rate"`
}

type UserJourney struct {
	AvgTimeToEngagement float64            `json:"avg_time_to_engagement_seconds"`
	AvgTimeToLead       float64            `json:"avg_time_to_lead_seconds"`
	CommonPaths         []ConversationPath `json:"common_paths"`
	DropOffPoints       []DropOffPoint     `json:"drop_off_points"`
}

type ConversationPath struct {
	Path      string  `json:"path"`
	Count     int     `json:"count"`
	LeadsRate float64 `json:"leads_rate"`
}

type DropOffPoint struct {
	MessageNumber int     `json:"message_number"`
	DropOffRate   float64 `json:"drop_off_rate"`
	Reason        string  `json:"reason,omitempty"`
}

type HourlyMetric struct {
	Hour              int     `json:"hour"`
	Conversations     int     `json:"conversations"`
	Messages          int     `json:"messages"`
	Leads             int     `json:"leads"`
	AvgResponseTimeMs float64 `json:"avg_response_time_ms"`
}

// Request types
type CreateChatbotRequest struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description,omitempty"`
	Personality      string                 `json:"personality,omitempty"`
	KnowledgeBase    string                 `json:"knowledge_base,omitempty"`
	ModelConfig      map[string]interface{} `json:"model_config,omitempty"`
	WidgetConfig     map[string]interface{} `json:"widget_config,omitempty"`
	EscalationConfig map[string]interface{} `json:"escalation_config,omitempty"`
}

type UpdateChatbotRequest struct {
	Name             *string                 `json:"name,omitempty"`
	Description      *string                 `json:"description,omitempty"`
	Personality      *string                 `json:"personality,omitempty"`
	KnowledgeBase    *string                 `json:"knowledge_base,omitempty"`
	ModelConfig      *map[string]interface{} `json:"model_config,omitempty"`
	WidgetConfig     *map[string]interface{} `json:"widget_config,omitempty"`
	EscalationConfig *map[string]interface{} `json:"escalation_config,omitempty"`
	IsActive         *bool                   `json:"is_active,omitempty"`
}
