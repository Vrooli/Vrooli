package main

import (
	"time"
)

// Database models
type Chatbot struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Personality   string                 `json:"personality"`
	KnowledgeBase string                 `json:"knowledge_base"`
	ModelConfig   map[string]interface{} `json:"model_config"`
	WidgetConfig  map[string]interface{} `json:"widget_config"`
	IsActive      bool                   `json:"is_active"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

type Conversation struct {
	ID           string                 `json:"id"`
	ChatbotID    string                 `json:"chatbot_id"`
	SessionID    string                 `json:"session_id"`
	UserIP       string                 `json:"user_ip,omitempty"`
	StartedAt    time.Time              `json:"started_at"`
	EndedAt      *time.Time             `json:"ended_at,omitempty"`
	LeadCaptured bool                   `json:"lead_captured"`
	LeadData     map[string]interface{} `json:"lead_data,omitempty"`
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
	LeadQualification map[string]interface{} `json:"lead_qualification,omitempty"`
	ConversationID    string                 `json:"conversation_id"`
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
	TotalConversations    int      `json:"total_conversations"`
	TotalMessages         int      `json:"total_messages"`
	LeadsCaptured         int      `json:"leads_captured"`
	AvgConversationLength float64  `json:"avg_conversation_length"`
	EngagementScore       float64  `json:"engagement_score"`
	ConversionRate        float64  `json:"conversion_rate"`
	TopIntents            []Intent `json:"top_intents"`
}

type Intent struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Request types
type CreateChatbotRequest struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description,omitempty"`
	Personality   string                 `json:"personality,omitempty"`
	KnowledgeBase string                 `json:"knowledge_base,omitempty"`
	ModelConfig   map[string]interface{} `json:"model_config,omitempty"`
	WidgetConfig  map[string]interface{} `json:"widget_config,omitempty"`
}

type UpdateChatbotRequest struct {
	Name          *string                 `json:"name,omitempty"`
	Description   *string                 `json:"description,omitempty"`
	Personality   *string                 `json:"personality,omitempty"`
	KnowledgeBase *string                 `json:"knowledge_base,omitempty"`
	ModelConfig   *map[string]interface{} `json:"model_config,omitempty"`
	WidgetConfig  *map[string]interface{} `json:"widget_config,omitempty"`
	IsActive      *bool                   `json:"is_active,omitempty"`
}