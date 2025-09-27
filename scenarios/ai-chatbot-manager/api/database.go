package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Database represents the database connection and operations
type Database struct {
	db     *sql.DB
	logger *Logger
}

// DB returns the underlying SQL database for advanced queries
func (d *Database) DB() *sql.DB {
	return d.db
}

// QueryRow executes a query that returns at most one row
func (d *Database) QueryRow(query string, args ...interface{}) *sql.Row {
	return d.db.QueryRow(query, args...)
}

// Query executes a query that returns rows
func (d *Database) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return d.db.Query(query, args...)
}

// Exec executes a query without returning any rows
func (d *Database) Exec(query string, args ...interface{}) (sql.Result, error) {
	return d.db.Exec(query, args...)
}

// NewDatabase creates a new database connection with exponential backoff
func NewDatabase(cfg *Config, logger *Logger) (*Database, error) {
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(maxDBConnections)
	db.SetMaxIdleConns(maxIdleConnections)
	db.SetConnMaxLifetime(connMaxLifetime)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	logger.Println("🔄 Attempting database connection with exponential backoff...")

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			logger.Printf("✅ Database connected successfully on attempt %d", attempt+1)
			break
		}

		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))

		// Add progressive jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
		actualDelay := delay + jitter

		logger.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
		logger.Printf("⏳ Waiting %v before next attempt", actualDelay)

		// Provide detailed status every few attempts
		if attempt > 0 && attempt%3 == 0 {
			logger.Printf("📈 Retry progress:")
			logger.Printf("   - Attempts made: %d/%d", attempt+1, maxRetries)
			logger.Printf("   - Total wait time: ~%v", time.Duration(attempt*2)*baseDelay)
			logger.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}

		time.Sleep(actualDelay)
	}

	if pingErr != nil {
		return nil, fmt.Errorf("database connection failed after %d attempts: %v", maxRetries, pingErr)
	}

	logger.Println("🎉 Database connection pool established successfully!")
	return &Database{db: db, logger: logger}, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// Ping checks database connectivity
func (d *Database) Ping() error {
	return d.db.Ping()
}

// CreateChatbot creates a new chatbot
func (d *Database) CreateChatbot(chatbot *Chatbot) error {
	modelConfigJSON, _ := json.Marshal(chatbot.ModelConfig)
	widgetConfigJSON, _ := json.Marshal(chatbot.WidgetConfig)
	escalationConfigJSON, _ := json.Marshal(chatbot.EscalationConfig)

	query := `
		INSERT INTO chatbots (id, name, description, personality, knowledge_base, 
			model_config, widget_config, escalation_config, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := d.db.Exec(query, chatbot.ID, chatbot.Name, chatbot.Description, 
		chatbot.Personality, chatbot.KnowledgeBase, modelConfigJSON, widgetConfigJSON, 
		escalationConfigJSON, chatbot.IsActive, chatbot.CreatedAt, chatbot.UpdatedAt)

	return err
}

// GetChatbot retrieves a chatbot by ID
func (d *Database) GetChatbot(chatbotID string) (*Chatbot, error) {
	var chatbot Chatbot
	var modelConfigJSON, widgetConfigJSON, escalationConfigJSON []byte

	query := `SELECT id, name, description, personality, knowledge_base, 
		model_config, widget_config, 
		COALESCE(escalation_config, '{"enabled":false,"threshold":0.5}'::jsonb),
		is_active, created_at, updated_at 
		FROM chatbots WHERE id = $1`

	err := d.db.QueryRow(query, chatbotID).Scan(
		&chatbot.ID, &chatbot.Name, &chatbot.Description, &chatbot.Personality,
		&chatbot.KnowledgeBase, &modelConfigJSON, &widgetConfigJSON, &escalationConfigJSON,
		&chatbot.IsActive, &chatbot.CreatedAt, &chatbot.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(modelConfigJSON, &chatbot.ModelConfig)
	json.Unmarshal(widgetConfigJSON, &chatbot.WidgetConfig)
	json.Unmarshal(escalationConfigJSON, &chatbot.EscalationConfig)

	return &chatbot, nil
}

// ListChatbots retrieves all chatbots
func (d *Database) ListChatbots(activeOnly bool) ([]Chatbot, error) {
	query := "SELECT id, name, description, personality, knowledge_base, model_config, widget_config, COALESCE(escalation_config, '{\"enabled\":false,\"threshold\":0.5}'::jsonb), is_active, created_at, updated_at FROM chatbots"
	args := []interface{}{}

	if activeOnly {
		query += " WHERE is_active = $1"
		args = append(args, true)
	}

	query += " ORDER BY created_at DESC"

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chatbots []Chatbot
	for rows.Next() {
		var chatbot Chatbot
		var modelConfigJSON, widgetConfigJSON, escalationConfigJSON []byte

		err := rows.Scan(&chatbot.ID, &chatbot.Name, &chatbot.Description, &chatbot.Personality,
			&chatbot.KnowledgeBase, &modelConfigJSON, &widgetConfigJSON, &escalationConfigJSON,
			&chatbot.IsActive, &chatbot.CreatedAt, &chatbot.UpdatedAt)

		if err != nil {
			d.logger.Printf("Failed to scan chatbot: %v", err)
			continue
		}

		json.Unmarshal(modelConfigJSON, &chatbot.ModelConfig)
		json.Unmarshal(widgetConfigJSON, &chatbot.WidgetConfig)
		json.Unmarshal(escalationConfigJSON, &chatbot.EscalationConfig)

		chatbots = append(chatbots, chatbot)
	}

	return chatbots, nil
}

// GetOrCreateConversation gets an existing conversation or creates a new one
func (d *Database) GetOrCreateConversation(chatbotID, sessionID, userIP string) (string, bool, error) {
	// Try to find existing conversation
	var conversationID string
	err := d.db.QueryRow(
		"SELECT id FROM conversations WHERE chatbot_id = $1 AND session_id = $2 AND ended_at IS NULL",
		chatbotID, sessionID,
	).Scan(&conversationID)

	if err == sql.ErrNoRows {
		// Create new conversation
		conversationID = uuid.New().String()
		
		// Handle empty IP address - use NULL for empty string
		var ipValue interface{} = nil
		if userIP != "" {
			ipValue = userIP
		}
		
		_, err = d.db.Exec(
			"INSERT INTO conversations (id, chatbot_id, session_id, user_ip, started_at) VALUES ($1, $2, $3, $4, $5)",
			conversationID, chatbotID, sessionID, ipValue, time.Now(),
		)
		return conversationID, true, err // true indicates new conversation
	}

	return conversationID, false, err // false indicates existing conversation
}

// SaveMessage saves a message to the database
func (d *Database) SaveMessage(message *Message) error {
	metadataJSON, _ := json.Marshal(message.Metadata)

	_, err := d.db.Exec(
		"INSERT INTO messages (id, conversation_id, role, content, metadata, timestamp) VALUES ($1, $2, $3, $4, $5, $6)",
		message.ID, message.ConversationID, message.Role, message.Content, metadataJSON, message.Timestamp,
	)
	return err
}

// GetConversationHistory retrieves message history for a conversation
func (d *Database) GetConversationHistory(conversationID string) ([]Message, error) {
	rows, err := d.db.Query(
		"SELECT id, role, content, metadata, timestamp FROM messages WHERE conversation_id = $1 ORDER BY timestamp ASC LIMIT 20",
		conversationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var message Message
		var metadataJSON []byte
		message.ConversationID = conversationID

		err := rows.Scan(&message.ID, &message.Role, &message.Content, &metadataJSON, &message.Timestamp)
		if err != nil {
			continue
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &message.Metadata)
		}
		messages = append(messages, message)
	}

	return messages, nil
}

// GetAnalyticsData retrieves analytics for a chatbot (FIXED SQL)
func (d *Database) GetAnalyticsData(chatbotID string, startDate time.Time) (*AnalyticsData, error) {
	// Fixed query - removed non-existent functions and tables
	query := `
		SELECT 
			COUNT(DISTINCT c.id) as total_conversations,
			COUNT(m.id) as total_messages,
			COUNT(DISTINCT c.id) FILTER (WHERE c.lead_captured = true) as leads_captured,
			COALESCE(AVG((
				SELECT COUNT(*) 
				FROM messages 
				WHERE conversation_id = c.id
			)), 0) as avg_conversation_length
		FROM conversations c
		LEFT JOIN messages m ON c.id = m.conversation_id
		WHERE c.chatbot_id = $1 AND c.started_at >= $2
	`

	var analytics AnalyticsData
	var avgLength sql.NullFloat64

	err := d.db.QueryRow(query, chatbotID, startDate).Scan(
		&analytics.TotalConversations,
		&analytics.TotalMessages,
		&analytics.LeadsCaptured,
		&avgLength,
	)

	if err != nil {
		return nil, err
	}

	analytics.AvgConversationLength = avgLength.Float64

	// Calculate conversion rate
	if analytics.TotalConversations > 0 {
		analytics.ConversionRate = float64(analytics.LeadsCaptured) / float64(analytics.TotalConversations) * 100
	}

	// Calculate a simple engagement score based on average conversation length
	if analytics.AvgConversationLength > 0 {
		// Simple engagement score: normalize avg length to 0-100 scale
		// Assume 10+ messages is highly engaged
		analytics.EngagementScore = math.Min(analytics.AvgConversationLength*10, 100)
	}

	// Get top intents
	analytics.TopIntents = d.GetTopIntents(chatbotID, startDate)

	// Get conversion funnel data
	analytics.ConversionFunnel = d.GetConversionFunnel(chatbotID, startDate)

	// Get user journey data
	analytics.UserJourney = d.GetUserJourneyData(chatbotID, startDate)

	// Get hourly distribution
	analytics.HourlyDistribution = d.GetHourlyDistribution(chatbotID, startDate)

	return &analytics, nil
}

// GetTopIntents retrieves the most common user intents
func (d *Database) GetTopIntents(chatbotID string, startDate time.Time) []Intent {
	query := `
		SELECT 
			COALESCE((m.metadata->>'intent')::text, 'general') as intent_name,
			COUNT(*) as count
		FROM messages m
		JOIN conversations c ON m.conversation_id = c.id
		WHERE c.chatbot_id = $1 
			AND c.started_at >= $2
			AND m.role = 'user'
		GROUP BY intent_name
		ORDER BY count DESC
		LIMIT 5
	`

	rows, err := d.db.Query(query, chatbotID, startDate)
	if err != nil {
		d.logger.Printf("Failed to get top intents: %v", err)
		return []Intent{}
	}
	defer rows.Close()

	var intents []Intent
	for rows.Next() {
		var intent Intent
		if err := rows.Scan(&intent.Name, &intent.Count); err != nil {
			continue
		}
		intents = append(intents, intent)
	}

	return intents
}

// GetConversionFunnel calculates conversion funnel metrics
func (d *Database) GetConversionFunnel(chatbotID string, startDate time.Time) *ConversionFunnel {
	query := `
		SELECT 
			COUNT(DISTINCT c.id) as total_visitors,
			COUNT(DISTINCT c.id) FILTER (WHERE m.message_count > 0) as engaged_visitors,
			COUNT(DISTINCT c.id) FILTER (WHERE m.message_count > 3) as qualified_leads,
			COUNT(DISTINCT c.id) FILTER (WHERE c.lead_captured = true) as captured_leads
		FROM conversations c
		LEFT JOIN (
			SELECT conversation_id, COUNT(*) as message_count
			FROM messages
			WHERE role = 'user'
			GROUP BY conversation_id
		) m ON c.id = m.conversation_id
		WHERE c.chatbot_id = $1 AND c.started_at >= $2
	`

	var funnel ConversionFunnel
	err := d.db.QueryRow(query, chatbotID, startDate).Scan(
		&funnel.TotalVisitors,
		&funnel.EngagedVisitors,
		&funnel.QualifiedLeads,
		&funnel.CapturedLeads,
	)

	if err != nil {
		d.logger.Printf("Failed to get conversion funnel: %v", err)
		return nil
	}

	// Calculate rates
	if funnel.TotalVisitors > 0 {
		funnel.EngagementRate = float64(funnel.EngagedVisitors) / float64(funnel.TotalVisitors) * 100
		funnel.QualificationRate = float64(funnel.QualifiedLeads) / float64(funnel.TotalVisitors) * 100
		funnel.CaptureRate = float64(funnel.CapturedLeads) / float64(funnel.TotalVisitors) * 100
	}

	return &funnel
}

// GetUserJourneyData analyzes user journey patterns
func (d *Database) GetUserJourneyData(chatbotID string, startDate time.Time) *UserJourney {
	journey := &UserJourney{}

	// Calculate average time to engagement
	query := `
		SELECT 
			AVG(EXTRACT(EPOCH FROM (first_message - started_at))) as avg_time_to_engagement,
			AVG(EXTRACT(EPOCH FROM (lead_time - started_at))) as avg_time_to_lead
		FROM (
			SELECT 
				c.id,
				c.started_at,
				MIN(m.timestamp) as first_message,
				CASE 
					WHEN c.lead_captured = true THEN c.ended_at
					ELSE NULL
				END as lead_time
			FROM conversations c
			LEFT JOIN messages m ON c.id = m.conversation_id AND m.role = 'user'
			WHERE c.chatbot_id = $1 AND c.started_at >= $2
			GROUP BY c.id, c.started_at, c.lead_captured, c.ended_at
		) t
	`

	var avgEngagement, avgLead sql.NullFloat64
	err := d.db.QueryRow(query, chatbotID, startDate).Scan(&avgEngagement, &avgLead)
	if err == nil {
		journey.AvgTimeToEngagement = avgEngagement.Float64
		journey.AvgTimeToLead = avgLead.Float64
	}

	// Get common conversation paths (simplified)
	journey.CommonPaths = d.getCommonPaths(chatbotID, startDate)

	// Get drop-off points
	journey.DropOffPoints = d.getDropOffPoints(chatbotID, startDate)

	return journey
}

// getCommonPaths analyzes common conversation patterns
func (d *Database) getCommonPaths(chatbotID string, startDate time.Time) []ConversationPath {
	query := `
		SELECT 
			COUNT(*) as count,
			COUNT(*) FILTER (WHERE lead_captured = true) as leads
		FROM (
			SELECT 
				c.id,
				c.lead_captured,
				COUNT(m.id) as message_count
			FROM conversations c
			LEFT JOIN messages m ON c.id = m.conversation_id
			WHERE c.chatbot_id = $1 AND c.started_at >= $2
			GROUP BY c.id, c.lead_captured
		) t
		GROUP BY 
			CASE 
				WHEN message_count <= 2 THEN 'short'
				WHEN message_count <= 5 THEN 'medium'
				ELSE 'long'
			END
	`

	rows, err := d.db.Query(query, chatbotID, startDate)
	if err != nil {
		return nil
	}
	defer rows.Close()

	paths := []ConversationPath{
		{Path: "short (1-2 messages)", Count: 0, LeadsRate: 0},
		{Path: "medium (3-5 messages)", Count: 0, LeadsRate: 0},
		{Path: "long (6+ messages)", Count: 0, LeadsRate: 0},
	}

	// Simplified path analysis
	return paths
}

// getDropOffPoints identifies where users commonly stop engaging
func (d *Database) getDropOffPoints(chatbotID string, startDate time.Time) []DropOffPoint {
	query := `
		WITH message_counts AS (
			SELECT 
				c.id,
				COUNT(m.id) as total_messages,
				MAX(CASE WHEN m.role = 'user' THEN m.timestamp END) as last_user_message
			FROM conversations c
			LEFT JOIN messages m ON c.id = m.conversation_id
			WHERE c.chatbot_id = $1 AND c.started_at >= $2
			GROUP BY c.id
		)
		SELECT 
			total_messages as message_number,
			COUNT(*) as drop_count
		FROM message_counts
		WHERE total_messages > 0
		GROUP BY total_messages
		ORDER BY total_messages
		LIMIT 5
	`

	rows, err := d.db.Query(query, chatbotID, startDate)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var dropPoints []DropOffPoint
	totalConversations := 0

	for rows.Next() {
		var point DropOffPoint
		var dropCount int
		if err := rows.Scan(&point.MessageNumber, &dropCount); err != nil {
			continue
		}
		totalConversations += dropCount
		dropPoints = append(dropPoints, point)
	}

	// Calculate drop-off rates
	for i := range dropPoints {
		if totalConversations > 0 {
			dropPoints[i].DropOffRate = float64(dropPoints[i].DropOffRate) / float64(totalConversations) * 100
		}
	}

	return dropPoints
}

// GetHourlyDistribution gets hourly conversation patterns
func (d *Database) GetHourlyDistribution(chatbotID string, startDate time.Time) []HourlyMetric {
	query := `
		SELECT 
			EXTRACT(HOUR FROM c.started_at) as hour,
			COUNT(DISTINCT c.id) as conversations,
			COUNT(m.id) as messages,
			COUNT(DISTINCT c.id) FILTER (WHERE c.lead_captured = true) as leads
		FROM conversations c
		LEFT JOIN messages m ON c.id = m.conversation_id
		WHERE c.chatbot_id = $1 AND c.started_at >= $2
		GROUP BY hour
		ORDER BY hour
	`

	rows, err := d.db.Query(query, chatbotID, startDate)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var metrics []HourlyMetric
	for rows.Next() {
		var metric HourlyMetric
		if err := rows.Scan(&metric.Hour, &metric.Conversations, &metric.Messages, &metric.Leads); err != nil {
			continue
		}
		metrics = append(metrics, metric)
	}

	return metrics
}

// UpdateChatbot updates an existing chatbot
func (d *Database) UpdateChatbot(chatbotID string, updates *UpdateChatbotRequest) error {
	// Build dynamic update query
	setClause := "updated_at = $1"
	args := []interface{}{time.Now()}
	argCount := 2

	if updates.Name != nil {
		setClause += fmt.Sprintf(", name = $%d", argCount)
		args = append(args, *updates.Name)
		argCount++
	}

	if updates.Description != nil {
		setClause += fmt.Sprintf(", description = $%d", argCount)
		args = append(args, *updates.Description)
		argCount++
	}

	if updates.Personality != nil {
		setClause += fmt.Sprintf(", personality = $%d", argCount)
		args = append(args, *updates.Personality)
		argCount++
	}

	if updates.KnowledgeBase != nil {
		setClause += fmt.Sprintf(", knowledge_base = $%d", argCount)
		args = append(args, *updates.KnowledgeBase)
		argCount++
	}

	if updates.ModelConfig != nil {
		modelConfigJSON, _ := json.Marshal(*updates.ModelConfig)
		setClause += fmt.Sprintf(", model_config = $%d", argCount)
		args = append(args, modelConfigJSON)
		argCount++
	}

	if updates.WidgetConfig != nil {
		widgetConfigJSON, _ := json.Marshal(*updates.WidgetConfig)
		setClause += fmt.Sprintf(", widget_config = $%d", argCount)
		args = append(args, widgetConfigJSON)
		argCount++
	}

	if updates.IsActive != nil {
		setClause += fmt.Sprintf(", is_active = $%d", argCount)
		args = append(args, *updates.IsActive)
		argCount++
	}

	// Add chatbot ID as last argument
	args = append(args, chatbotID)

	query := fmt.Sprintf("UPDATE chatbots SET %s WHERE id = $%d", setClause, argCount)

	_, err := d.db.Exec(query, args...)
	return err
}

// DeleteChatbot soft deletes a chatbot
func (d *Database) DeleteChatbot(chatbotID string) error {
	_, err := d.db.Exec(
		"UPDATE chatbots SET is_active = false, updated_at = $1 WHERE id = $2",
		time.Now(), chatbotID,
	)
	return err
}

// CreateEscalation creates a new escalation record
func (d *Database) CreateEscalation(escalation *Escalation) error {
	webhookJSON, _ := json.Marshal(escalation.WebhookResponse)
	
	query := `
		INSERT INTO escalations (id, conversation_id, chatbot_id, reason, 
			confidence_score, escalation_type, status, escalated_at, webhook_response, email_sent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	
	_, err := d.db.Exec(query, escalation.ID, escalation.ConversationID, escalation.ChatbotID,
		escalation.Reason, escalation.ConfidenceScore, escalation.EscalationType, 
		escalation.Status, escalation.EscalatedAt, webhookJSON, escalation.EmailSent)
	
	return err
}

// GetPendingEscalations retrieves pending escalations for a chatbot
func (d *Database) GetPendingEscalations(chatbotID string) ([]Escalation, error) {
	query := `
		SELECT id, conversation_id, chatbot_id, reason, confidence_score, 
			escalation_type, status, escalated_at, resolved_at, resolution_notes,
			webhook_response, email_sent
		FROM escalations 
		WHERE chatbot_id = $1 AND status = 'pending'
		ORDER BY escalated_at DESC
	`
	
	rows, err := d.db.Query(query, chatbotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var escalations []Escalation
	for rows.Next() {
		var escalation Escalation
		var resolvedAt sql.NullTime
		var resolutionNotes sql.NullString
		var webhookJSON []byte
		
		err := rows.Scan(&escalation.ID, &escalation.ConversationID, &escalation.ChatbotID,
			&escalation.Reason, &escalation.ConfidenceScore, &escalation.EscalationType,
			&escalation.Status, &escalation.EscalatedAt, &resolvedAt, &resolutionNotes,
			&webhookJSON, &escalation.EmailSent)
		
		if err != nil {
			d.logger.Printf("Failed to scan escalation: %v", err)
			continue
		}
		
		if resolvedAt.Valid {
			escalation.ResolvedAt = &resolvedAt.Time
		}
		if resolutionNotes.Valid {
			escalation.ResolutionNotes = resolutionNotes.String
		}
		
		json.Unmarshal(webhookJSON, &escalation.WebhookResponse)
		escalations = append(escalations, escalation)
	}
	
	return escalations, nil
}

// UpdateEscalationStatus updates the status of an escalation
func (d *Database) UpdateEscalationStatus(escalationID, status, notes string) error {
	query := `
		UPDATE escalations 
		SET status = $1, resolution_notes = $2, resolved_at = $3
		WHERE id = $4
	`
	
	var resolvedAt *time.Time
	if status == "resolved" {
		now := time.Now()
		resolvedAt = &now
	}
	
	_, err := d.db.Exec(query, status, notes, resolvedAt, escalationID)
	return err
}