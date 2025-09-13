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

	query := `
		INSERT INTO chatbots (id, name, description, personality, knowledge_base, 
			model_config, widget_config, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := d.db.Exec(query, chatbot.ID, chatbot.Name, chatbot.Description, 
		chatbot.Personality, chatbot.KnowledgeBase, modelConfigJSON, widgetConfigJSON, 
		chatbot.IsActive, chatbot.CreatedAt, chatbot.UpdatedAt)

	return err
}

// GetChatbot retrieves a chatbot by ID
func (d *Database) GetChatbot(chatbotID string) (*Chatbot, error) {
	var chatbot Chatbot
	var modelConfigJSON, widgetConfigJSON []byte

	query := `SELECT id, name, description, personality, knowledge_base, 
		model_config, widget_config, is_active, created_at, updated_at 
		FROM chatbots WHERE id = $1`

	err := d.db.QueryRow(query, chatbotID).Scan(
		&chatbot.ID, &chatbot.Name, &chatbot.Description, &chatbot.Personality,
		&chatbot.KnowledgeBase, &modelConfigJSON, &widgetConfigJSON, &chatbot.IsActive,
		&chatbot.CreatedAt, &chatbot.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(modelConfigJSON, &chatbot.ModelConfig)
	json.Unmarshal(widgetConfigJSON, &chatbot.WidgetConfig)

	return &chatbot, nil
}

// ListChatbots retrieves all chatbots
func (d *Database) ListChatbots(activeOnly bool) ([]Chatbot, error) {
	query := "SELECT id, name, description, personality, knowledge_base, model_config, widget_config, is_active, created_at, updated_at FROM chatbots"
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
		var modelConfigJSON, widgetConfigJSON []byte

		err := rows.Scan(&chatbot.ID, &chatbot.Name, &chatbot.Description, &chatbot.Personality,
			&chatbot.KnowledgeBase, &modelConfigJSON, &widgetConfigJSON, &chatbot.IsActive,
			&chatbot.CreatedAt, &chatbot.UpdatedAt)

		if err != nil {
			d.logger.Printf("Failed to scan chatbot: %v", err)
			continue
		}

		json.Unmarshal(modelConfigJSON, &chatbot.ModelConfig)
		json.Unmarshal(widgetConfigJSON, &chatbot.WidgetConfig)

		chatbots = append(chatbots, chatbot)
	}

	return chatbots, nil
}

// GetOrCreateConversation gets an existing conversation or creates a new one
func (d *Database) GetOrCreateConversation(chatbotID, sessionID, userIP string) (string, error) {
	// Try to find existing conversation
	var conversationID string
	err := d.db.QueryRow(
		"SELECT id FROM conversations WHERE chatbot_id = $1 AND session_id = $2 AND ended_at IS NULL",
		chatbotID, sessionID,
	).Scan(&conversationID)

	if err == sql.ErrNoRows {
		// Create new conversation
		conversationID = uuid.New().String()
		_, err = d.db.Exec(
			"INSERT INTO conversations (id, chatbot_id, session_id, user_ip, started_at) VALUES ($1, $2, $3, $4, $5)",
			conversationID, chatbotID, sessionID, userIP, time.Now(),
		)
		return conversationID, err
	}

	return conversationID, err
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

	// For now, leave TopIntents empty since the intent_patterns table doesn't exist
	// This could be implemented later with proper intent analysis
	analytics.TopIntents = []Intent{}

	return &analytics, nil
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