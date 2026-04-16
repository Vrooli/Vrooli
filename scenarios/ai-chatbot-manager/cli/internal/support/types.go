package support

import "time"

// Chatbot mirrors the Chatbot shape returned by /api/v1/chatbots.
type Chatbot struct {
	ID               string                 `json:"id"`
	TenantID         string                 `json:"tenant_id,omitempty"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description,omitempty"`
	Personality      string                 `json:"personality,omitempty"`
	KnowledgeBase    string                 `json:"knowledge_base,omitempty"`
	ModelConfig      map[string]interface{} `json:"model_config,omitempty"`
	WidgetConfig     map[string]interface{} `json:"widget_config,omitempty"`
	EscalationConfig map[string]interface{} `json:"escalation_config,omitempty"`
	IsActive         bool                   `json:"is_active"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// ChatbotCreateResponse is the shape returned from POST /api/v1/chatbots.
type ChatbotCreateResponse struct {
	Chatbot         Chatbot `json:"chatbot"`
	WidgetEmbedCode string  `json:"widget_embed_code,omitempty"`
}

// ChatResponse mirrors the POST /api/v1/chat/{id} response.
type ChatResponse struct {
	Response          string                 `json:"response"`
	Confidence        float64                `json:"confidence"`
	ShouldEscalate    bool                   `json:"should_escalate"`
	EscalationReason  string                 `json:"escalation_reason,omitempty"`
	LeadQualification map[string]interface{} `json:"lead_qualification,omitempty"`
	ConversationID    string                 `json:"conversation_id"`
}

// Escalation mirrors the escalation records surfaced via
// /api/v1/chatbots/{id}/escalations.
type Escalation struct {
	ID              string     `json:"id"`
	ConversationID  string     `json:"conversation_id"`
	ChatbotID       string     `json:"chatbot_id"`
	Reason          string     `json:"reason"`
	ConfidenceScore float64    `json:"confidence_score"`
	EscalationType  string     `json:"escalation_type"`
	Status          string     `json:"status"`
	EscalatedAt     time.Time  `json:"escalated_at"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
	ResolutionNotes string     `json:"resolution_notes,omitempty"`
	EmailSent       bool       `json:"email_sent"`
}
