package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

const (
	apiVersion  = "1.0.0"
	serviceName = "ai-chatbot-manager"

	// Defaults
	defaultPort        = "8090"
	defaultOllamaURL   = "http://localhost:11434"
	httpTimeout        = 30 * time.Second
	maxDBConnections   = 25
	maxIdleConnections = 5
	connMaxLifetime    = 5 * time.Minute
)

// Logger provides structured logging
type Logger struct {
	*log.Logger
}

func NewLogger() *Logger {
	return &Logger{log.New(os.Stdout, "[CHATBOT-API] ", log.LstdFlags|log.Lshortfile)}
}

// Database models
type Chatbot struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Personality  string                 `json:"personality"`
	KnowledgeBase string                `json:"knowledge_base"`
	ModelConfig  map[string]interface{} `json:"model_config"`
	WidgetConfig map[string]interface{} `json:"widget_config"`
	IsActive     bool                   `json:"is_active"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
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
	Message     string                 `json:"message"`
	SessionID   string                 `json:"session_id"`
	Context     map[string]interface{} `json:"context,omitempty"`
	UserIP      string                 `json:"user_ip,omitempty"`
}

type ChatResponse struct {
	Response          string                 `json:"response"`
	Confidence        float64                `json:"confidence"`
	ShouldEscalate    bool                   `json:"should_escalate"`
	LeadQualification map[string]interface{} `json:"lead_qualification,omitempty"`
	ConversationID    string                 `json:"conversation_id"`
}

type OllamaRequest struct {
	Model    string             `json:"model"`
	Messages []OllamaMessage    `json:"messages"`
	Stream   bool               `json:"stream"`
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
	TotalConversations      int     `json:"total_conversations"`
	TotalMessages          int     `json:"total_messages"`
	LeadsCaptured          int     `json:"leads_captured"`
	AvgConversationLength  float64 `json:"avg_conversation_length"`
	EngagementScore        float64 `json:"engagement_score"`
	ConversionRate         float64 `json:"conversion_rate"`
	TopIntents            []Intent `json:"top_intents"`
}

type Intent struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// WebSocket connection manager
type ConnectionManager struct {
	connections map[string]*websocket.Conn
	register    chan *websocket.Conn
	unregister  chan *websocket.Conn
	logger      *Logger
}

func NewConnectionManager(logger *Logger) *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*websocket.Conn),
		register:    make(chan *websocket.Conn),
		unregister:  make(chan *websocket.Conn),
		logger:      logger,
	}
}

func (cm *ConnectionManager) Start() {
	for {
		select {
		case conn := <-cm.register:
			sessionID := conn.RemoteAddr().String() // Simplified session management
			cm.connections[sessionID] = conn
			cm.logger.Printf("WebSocket client connected: %s", sessionID)

		case conn := <-cm.unregister:
			sessionID := conn.RemoteAddr().String()
			if _, ok := cm.connections[sessionID]; ok {
				delete(cm.connections, sessionID)
				conn.Close()
				cm.logger.Printf("WebSocket client disconnected: %s", sessionID)
			}
		}
	}
}

// Main server struct
type Server struct {
	db               *sql.DB
	logger           *Logger
	ollamaURL        string
	connectionManager *ConnectionManager
	upgrader         websocket.Upgrader
}

func NewServer(db *sql.DB, logger *Logger, ollamaURL string) *Server {
	cm := NewConnectionManager(logger)
	go cm.Start()

	return &Server{
		db:               db,
		logger:           logger,
		ollamaURL:        ollamaURL,
		connectionManager: cm,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
	}
}

// Health check endpoint
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	if err := s.db.Ping(); err != nil {
		s.logger.Printf("Database health check failed: %v", err)
		http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
		return
	}

	// Check Ollama connection
	resp, err := http.Get(s.ollamaURL + "/api/tags")
	if err != nil || resp.StatusCode != http.StatusOK {
		s.logger.Printf("Ollama health check failed: %v", err)
		http.Error(w, "Ollama unavailable", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"version":   apiVersion,
		"service":   serviceName,
		"timestamp": time.Now().UTC(),
		"database":  "connected",
		"ollama":    "connected",
	})
}

// Chatbot CRUD operations
func (s *Server) createChatbotHandler(w http.ResponseWriter, r *http.Request) {
	var chatbot Chatbot
	if err := json.NewDecoder(r.Body).Decode(&chatbot); err != nil {
		s.logger.Printf("Failed to decode chatbot: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if strings.TrimSpace(chatbot.Name) == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(chatbot.Personality) == "" {
		chatbot.Personality = "You are a helpful assistant."
	}

	// Set defaults
	if chatbot.ModelConfig == nil {
		chatbot.ModelConfig = map[string]interface{}{
			"model":       "llama3.2",
			"temperature": 0.7,
			"max_tokens":  1000,
		}
	}

	if chatbot.WidgetConfig == nil {
		chatbot.WidgetConfig = map[string]interface{}{
			"theme":        "light",
			"position":     "bottom-right",
			"primaryColor": "#007bff",
		}
	}

	chatbot.ID = uuid.New().String()
	chatbot.IsActive = true
	chatbot.CreatedAt = time.Now()
	chatbot.UpdatedAt = time.Now()

	// Save to database
	modelConfigJSON, _ := json.Marshal(chatbot.ModelConfig)
	widgetConfigJSON, _ := json.Marshal(chatbot.WidgetConfig)

	query := `
		INSERT INTO chatbots (id, name, description, personality, knowledge_base, model_config, widget_config, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := s.db.Exec(query, chatbot.ID, chatbot.Name, chatbot.Description, chatbot.Personality,
		chatbot.KnowledgeBase, modelConfigJSON, widgetConfigJSON, chatbot.IsActive, chatbot.CreatedAt, chatbot.UpdatedAt)

	if err != nil {
		s.logger.Printf("Failed to create chatbot: %v", err)
		http.Error(w, "Failed to create chatbot", http.StatusInternalServerError)
		return
	}

	// Generate widget embed code
	embedCode := s.generateWidgetEmbedCode(chatbot.ID, chatbot.WidgetConfig)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"chatbot":         chatbot,
		"widget_embed_code": embedCode,
	})
}

func (s *Server) listChatbotsHandler(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active") == "true"

	query := "SELECT id, name, description, personality, knowledge_base, model_config, widget_config, is_active, created_at, updated_at FROM chatbots"
	args := []interface{}{}

	if activeOnly {
		query += " WHERE is_active = $1"
		args = append(args, true)
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		s.logger.Printf("Failed to query chatbots: %v", err)
		http.Error(w, "Failed to retrieve chatbots", http.StatusInternalServerError)
		return
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
			s.logger.Printf("Failed to scan chatbot: %v", err)
			continue
		}

		json.Unmarshal(modelConfigJSON, &chatbot.ModelConfig)
		json.Unmarshal(widgetConfigJSON, &chatbot.WidgetConfig)

		chatbots = append(chatbots, chatbot)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatbots)
}

func (s *Server) getChatbotHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatbotID := vars["id"]

	var chatbot Chatbot
	var modelConfigJSON, widgetConfigJSON []byte

	query := "SELECT id, name, description, personality, knowledge_base, model_config, widget_config, is_active, created_at, updated_at FROM chatbots WHERE id = $1"
	err := s.db.QueryRow(query, chatbotID).Scan(
		&chatbot.ID, &chatbot.Name, &chatbot.Description, &chatbot.Personality,
		&chatbot.KnowledgeBase, &modelConfigJSON, &widgetConfigJSON, &chatbot.IsActive,
		&chatbot.CreatedAt, &chatbot.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Chatbot not found", http.StatusNotFound)
		return
	} else if err != nil {
		s.logger.Printf("Failed to get chatbot: %v", err)
		http.Error(w, "Failed to retrieve chatbot", http.StatusInternalServerError)
		return
	}

	json.Unmarshal(modelConfigJSON, &chatbot.ModelConfig)
	json.Unmarshal(widgetConfigJSON, &chatbot.WidgetConfig)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatbot)
}

// Chat functionality
func (s *Server) chatHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatbotID := vars["id"]

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Get chatbot configuration
	var chatbot Chatbot
	var modelConfigJSON []byte
	query := "SELECT id, personality, knowledge_base, model_config FROM chatbots WHERE id = $1 AND is_active = true"
	err := s.db.QueryRow(query, chatbotID).Scan(&chatbot.ID, &chatbot.Personality, &chatbot.KnowledgeBase, &modelConfigJSON)

	if err == sql.ErrNoRows {
		http.Error(w, "Chatbot not found or inactive", http.StatusNotFound)
		return
	} else if err != nil {
		s.logger.Printf("Failed to get chatbot: %v", err)
		http.Error(w, "Failed to retrieve chatbot", http.StatusInternalServerError)
		return
	}

	json.Unmarshal(modelConfigJSON, &chatbot.ModelConfig)

	// Get or create conversation
	conversationID, err := s.getOrCreateConversation(chatbotID, req.SessionID, req.UserIP)
	if err != nil {
		s.logger.Printf("Failed to get/create conversation: %v", err)
		http.Error(w, "Failed to manage conversation", http.StatusInternalServerError)
		return
	}

	// Save user message
	messageID := uuid.New().String()
	_, err = s.db.Exec(
		"INSERT INTO messages (id, conversation_id, role, content, timestamp) VALUES ($1, $2, $3, $4, $5)",
		messageID, conversationID, "user", req.Message, time.Now(),
	)
	if err != nil {
		s.logger.Printf("Failed to save user message: %v", err)
	}

	// Get conversation history for context
	messages, err := s.getConversationHistory(conversationID)
	if err != nil {
		s.logger.Printf("Failed to get conversation history: %v", err)
	}

	// Generate AI response
	response, confidence, err := s.generateAIResponse(chatbot, messages, req.Message)
	if err != nil {
		s.logger.Printf("Failed to generate AI response: %v", err)
		http.Error(w, "Failed to generate response", http.StatusInternalServerError)
		return
	}

	// Save assistant response
	assistantMessageID := uuid.New().String()
	metadata := map[string]interface{}{
		"confidence":    confidence,
		"model":         chatbot.ModelConfig["model"],
		"response_time": time.Now().Unix(),
	}
	metadataJSON, _ := json.Marshal(metadata)

	_, err = s.db.Exec(
		"INSERT INTO messages (id, conversation_id, role, content, metadata, timestamp) VALUES ($1, $2, $3, $4, $5, $6)",
		assistantMessageID, conversationID, "assistant", response, metadataJSON, time.Now(),
	)
	if err != nil {
		s.logger.Printf("Failed to save assistant message: %v", err)
	}

	// Check for lead qualification
	leadQualification := s.analyzeForLeadQualification(req.Message, response)

	chatResponse := ChatResponse{
		Response:          response,
		Confidence:        confidence,
		ShouldEscalate:    confidence < 0.5,
		LeadQualification: leadQualification,
		ConversationID:    conversationID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatResponse)
}

// WebSocket handler for real-time chat
func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatbotID := vars["id"]

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	s.connectionManager.register <- conn

	for {
		var req ChatRequest
		if err := conn.ReadJSON(&req); err != nil {
			s.logger.Printf("WebSocket read error: %v", err)
			break
		}

		// Process the chat request (similar to HTTP handler)
		response := s.processChatMessage(chatbotID, req)
		
		if err := conn.WriteJSON(response); err != nil {
			s.logger.Printf("WebSocket write error: %v", err)
			break
		}
	}

	s.connectionManager.unregister <- conn
}

// Analytics endpoint
func (s *Server) analyticsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatbotID := vars["id"]

	// Parse query parameters
	days := 7
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	}

	startDate := time.Now().AddDate(0, 0, -days)

	// Get analytics data
	analytics, err := s.getAnalyticsData(chatbotID, startDate)
	if err != nil {
		s.logger.Printf("Failed to get analytics: %v", err)
		http.Error(w, "Failed to retrieve analytics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

// Helper functions
func (s *Server) getOrCreateConversation(chatbotID, sessionID, userIP string) (string, error) {
	// Try to find existing conversation
	var conversationID string
	err := s.db.QueryRow(
		"SELECT id FROM conversations WHERE chatbot_id = $1 AND session_id = $2 AND ended_at IS NULL",
		chatbotID, sessionID,
	).Scan(&conversationID)

	if err == sql.ErrNoRows {
		// Create new conversation
		conversationID = uuid.New().String()
		_, err = s.db.Exec(
			"INSERT INTO conversations (id, chatbot_id, session_id, user_ip, started_at) VALUES ($1, $2, $3, $4, $5)",
			conversationID, chatbotID, sessionID, userIP, time.Now(),
		)
		return conversationID, err
	}

	return conversationID, err
}

func (s *Server) getConversationHistory(conversationID string) ([]Message, error) {
	rows, err := s.db.Query(
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

func (s *Server) generateAIResponse(chatbot Chatbot, history []Message, userMessage string) (string, float64, error) {
	// Prepare messages for Ollama
	var ollamaMessages []OllamaMessage

	// Add system message with personality and knowledge base
	systemPrompt := chatbot.Personality
	if chatbot.KnowledgeBase != "" {
		systemPrompt += "\n\nKnowledge Base:\n" + chatbot.KnowledgeBase
	}
	ollamaMessages = append(ollamaMessages, OllamaMessage{
		Role:    "system",
		Content: systemPrompt,
	})

	// Add conversation history (last 10 messages)
	historyStart := len(history) - 10
	if historyStart < 0 {
		historyStart = 0
	}

	for i := historyStart; i < len(history); i++ {
		ollamaMessages = append(ollamaMessages, OllamaMessage{
			Role:    history[i].Role,
			Content: history[i].Content,
		})
	}

	// Add current user message
	ollamaMessages = append(ollamaMessages, OllamaMessage{
		Role:    "user",
		Content: userMessage,
	})

	// Prepare Ollama request
	model := "llama3.2"
	if m, ok := chatbot.ModelConfig["model"].(string); ok {
		model = m
	}

	ollamaReq := OllamaRequest{
		Model:    model,
		Messages: ollamaMessages,
		Stream:   false,
		Options:  chatbot.ModelConfig,
	}

	// Send request to Ollama
	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return "", 0, err
	}

	resp, err := http.Post(s.ollamaURL+"/api/chat", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("Ollama API error: %d", resp.StatusCode)
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", 0, err
	}

	// Calculate confidence (simplified)
	confidence := 0.8 // Default confidence
	if len(ollamaResp.Message.Content) > 10 {
		confidence = 0.9
	}

	return ollamaResp.Message.Content, confidence, nil
}

func (s *Server) processChatMessage(chatbotID string, req ChatRequest) ChatResponse {
	// This is a simplified version for WebSocket handling
	// You could refactor the main chat logic into a shared function
	return ChatResponse{
		Response:       "WebSocket response placeholder",
		Confidence:     0.8,
		ShouldEscalate: false,
		ConversationID: "ws-conv-id",
	}
}

func (s *Server) analyzeForLeadQualification(userMessage, aiResponse string) map[string]interface{} {
	qualification := make(map[string]interface{})

	// Simple keyword-based lead qualification
	userLower := strings.ToLower(userMessage)
	if strings.Contains(userLower, "email") || strings.Contains(userLower, "@") {
		qualification["email_mentioned"] = true
	}
	if strings.Contains(userLower, "price") || strings.Contains(userLower, "cost") || strings.Contains(userLower, "budget") {
		qualification["pricing_interest"] = true
	}
	if strings.Contains(userLower, "demo") || strings.Contains(userLower, "trial") {
		qualification["demo_interest"] = true
	}

	return qualification
}

func (s *Server) getAnalyticsData(chatbotID string, startDate time.Time) (*AnalyticsData, error) {
	query := `
		SELECT 
			COUNT(DISTINCT c.id) as total_conversations,
			COUNT(m.id) as total_messages,
			COUNT(*) FILTER (WHERE c.lead_captured = true) as leads_captured,
			AVG((SELECT COUNT(*) FROM messages WHERE conversation_id = c.id)) as avg_conversation_length,
			AVG(COALESCE((SELECT calculate_engagement_score(c.id)), 0)) as engagement_score
		FROM conversations c
		LEFT JOIN messages m ON c.id = m.conversation_id
		WHERE c.chatbot_id = $1 AND c.started_at >= $2
	`

	var analytics AnalyticsData
	var avgLength, engagement sql.NullFloat64

	err := s.db.QueryRow(query, chatbotID, startDate).Scan(
		&analytics.TotalConversations,
		&analytics.TotalMessages,
		&analytics.LeadsCaptured,
		&avgLength,
		&engagement,
	)

	if err != nil {
		return nil, err
	}

	analytics.AvgConversationLength = avgLength.Float64
	analytics.EngagementScore = engagement.Float64

	// Calculate conversion rate
	if analytics.TotalConversations > 0 {
		analytics.ConversionRate = float64(analytics.LeadsCaptured) / float64(analytics.TotalConversations) * 100
	}

	// Get top intents (simplified)
	intentRows, err := s.db.Query(`
		SELECT intent_name, occurrence_count 
		FROM intent_patterns 
		WHERE chatbot_id = $1 
		ORDER BY occurrence_count DESC 
		LIMIT 5
	`, chatbotID)

	if err == nil {
		defer intentRows.Close()
		for intentRows.Next() {
			var intent Intent
			if err := intentRows.Scan(&intent.Name, &intent.Count); err == nil {
				analytics.TopIntents = append(analytics.TopIntents, intent)
			}
		}
	}

	return &analytics, nil
}

func (s *Server) generateWidgetEmbedCode(chatbotID string, widgetConfig map[string]interface{}) string {
	// This would generate the actual widget embed code
	// For now, return a placeholder
	return fmt.Sprintf(`<script>
(function() {
	var chatbotId = '%s';
	var config = %s;
	// Widget initialization code would go here
	console.log('Chatbot widget loaded:', chatbotId, config);
})();
</script>`, chatbotID, "{}") // Simplified for now
}

// Database setup
func setupDatabase() (*sql.DB, error) {
	// Get database connection string from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/postgres?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(maxDBConnections)
	db.SetMaxIdleConns(maxIdleConnections)
	db.SetConnMaxLifetime(connMaxLifetime)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	return db, nil
}

func main() {
	logger := NewLogger()
	logger.Println("Starting AI Chatbot Manager API...")

	// Setup database
	db, err := setupDatabase()
	if err != nil {
		logger.Fatalf("Database setup failed: %v", err)
	}
	defer db.Close()

	// Get configuration from environment
	port := os.Getenv("API_PORT")
	if port == "" {
		port = defaultPort
	}

	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = defaultOllamaURL
	}

	// Create server
	server := NewServer(db, logger, ollamaURL)

	// Setup routes
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", server.healthHandler).Methods("GET")

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	
	// Chatbot management
	api.HandleFunc("/chatbots", server.createChatbotHandler).Methods("POST")
	api.HandleFunc("/chatbots", server.listChatbotsHandler).Methods("GET")
	api.HandleFunc("/chatbots/{id}", server.getChatbotHandler).Methods("GET")
	
	// Chat functionality
	api.HandleFunc("/chat/{id}", server.chatHandler).Methods("POST")
	api.HandleFunc("/ws/{id}", server.websocketHandler) // WebSocket
	
	// Analytics
	api.HandleFunc("/analytics/{id}", server.analyticsHandler).Methods("GET")

	// CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"}, // Configure appropriately for production
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(router)

	logger.Printf("Server starting on port %s", port)
	logger.Printf("Ollama URL: %s", ollamaURL)
	logger.Printf("Health check: http://localhost:%s/health", port)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		logger.Fatalf("Server failed to start: %v", err)
	}
}