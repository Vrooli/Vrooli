package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// HealthHandler handles health check requests
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	if err := s.db.Ping(); err != nil {
		s.logger.Printf("Database health check failed: %v", err)
		http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
		return
	}

	// Check Ollama connection
	resp, err := http.Get(s.config.OllamaURL + "/api/tags")
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

// CreateChatbotHandler handles chatbot creation
func (s *Server) CreateChatbotHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateChatbotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Printf("Failed to decode chatbot: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Set defaults
	if strings.TrimSpace(req.Personality) == "" {
		req.Personality = "You are a helpful assistant."
	}

	if req.ModelConfig == nil {
		req.ModelConfig = map[string]interface{}{
			"model":       "llama3.2",
			"temperature": 0.7,
			"max_tokens":  1000,
		}
	}

	if req.WidgetConfig == nil {
		req.WidgetConfig = map[string]interface{}{
			"theme":        "light",
			"position":     "bottom-right",
			"primaryColor": "#007bff",
		}
	}

	// Create chatbot
	chatbot := &Chatbot{
		ID:            uuid.New().String(),
		Name:          req.Name,
		Description:   req.Description,
		Personality:   req.Personality,
		KnowledgeBase: req.KnowledgeBase,
		ModelConfig:   req.ModelConfig,
		WidgetConfig:  req.WidgetConfig,
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Save to database
	if err := s.db.CreateChatbot(chatbot); err != nil {
		s.logger.Printf("Failed to create chatbot: %v", err)
		http.Error(w, "Failed to create chatbot", http.StatusInternalServerError)
		return
	}

	// Generate widget embed code
	embedCode := s.GenerateWidgetEmbedCode(chatbot.ID, chatbot.WidgetConfig)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"chatbot":           chatbot,
		"widget_embed_code": embedCode,
	})
}

// ListChatbotsHandler handles listing all chatbots
func (s *Server) ListChatbotsHandler(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active") == "true"

	chatbots, err := s.db.ListChatbots(activeOnly)
	if err != nil {
		s.logger.Printf("Failed to list chatbots: %v", err)
		http.Error(w, "Failed to retrieve chatbots", http.StatusInternalServerError)
		return
	}

	// Default to empty array if nil
	if chatbots == nil {
		chatbots = []Chatbot{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatbots)
}

// GetChatbotHandler handles getting a specific chatbot
func (s *Server) GetChatbotHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatbotID := vars["id"]

	chatbot, err := s.db.GetChatbot(chatbotID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			http.Error(w, "Chatbot not found", http.StatusNotFound)
		} else {
			s.logger.Printf("Failed to get chatbot: %v", err)
			http.Error(w, "Failed to retrieve chatbot", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatbot)
}

// UpdateChatbotHandler handles updating a chatbot
func (s *Server) UpdateChatbotHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatbotID := vars["id"]

	var req UpdateChatbotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Verify chatbot exists
	_, err := s.db.GetChatbot(chatbotID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			http.Error(w, "Chatbot not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve chatbot", http.StatusInternalServerError)
		}
		return
	}

	// Update chatbot
	if err := s.db.UpdateChatbot(chatbotID, &req); err != nil {
		s.logger.Printf("Failed to update chatbot: %v", err)
		http.Error(w, "Failed to update chatbot", http.StatusInternalServerError)
		return
	}

	// Get updated chatbot
	chatbot, err := s.db.GetChatbot(chatbotID)
	if err != nil {
		s.logger.Printf("Failed to get updated chatbot: %v", err)
		http.Error(w, "Failed to retrieve updated chatbot", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatbot)
}

// DeleteChatbotHandler handles soft deleting a chatbot
func (s *Server) DeleteChatbotHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatbotID := vars["id"]

	if err := s.db.DeleteChatbot(chatbotID); err != nil {
		s.logger.Printf("Failed to delete chatbot: %v", err)
		http.Error(w, "Failed to delete chatbot", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Chatbot deleted successfully",
		"id":      chatbotID,
	})
}

// ChatHandler handles chat requests
func (s *Server) ChatHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatbotID := vars["id"]

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Process chat message
	response, err := s.ProcessChatMessage(chatbotID, req)
	if err != nil {
		s.logger.Printf("Failed to process chat: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// AnalyticsHandler handles analytics requests
func (s *Server) AnalyticsHandler(w http.ResponseWriter, r *http.Request) {
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
	analytics, err := s.db.GetAnalyticsData(chatbotID, startDate)
	if err != nil {
		s.logger.Printf("Failed to get analytics: %v", err)
		http.Error(w, "Failed to retrieve analytics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

// ProcessChatMessage processes a chat message and returns a response
func (s *Server) ProcessChatMessage(chatbotID string, req ChatRequest) (*ChatResponse, error) {
	// Get chatbot configuration
	chatbot, err := s.db.GetChatbot(chatbotID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, fmt.Errorf("chatbot not found or inactive")
		}
		return nil, fmt.Errorf("failed to retrieve chatbot")
	}

	if !chatbot.IsActive {
		return nil, fmt.Errorf("chatbot is not active")
	}

	// Get or create conversation
	conversationID, err := s.db.GetOrCreateConversation(chatbotID, req.SessionID, req.UserIP)
	if err != nil {
		s.logger.Printf("Failed to get/create conversation: %v", err)
		return nil, fmt.Errorf("failed to manage conversation")
	}

	// Save user message
	userMessage := &Message{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		Role:           "user",
		Content:        req.Message,
		Timestamp:      time.Now(),
	}
	if err := s.db.SaveMessage(userMessage); err != nil {
		s.logger.Printf("Failed to save user message: %v", err)
	}

	// Get conversation history for context
	messages, err := s.db.GetConversationHistory(conversationID)
	if err != nil {
		s.logger.Printf("Failed to get conversation history: %v", err)
		messages = []Message{} // Continue with empty history
	}

	// Generate AI response
	response, confidence, err := s.GenerateAIResponse(chatbot, messages, req.Message)
	if err != nil {
		s.logger.Printf("Failed to generate AI response: %v", err)
		return nil, fmt.Errorf("failed to generate response")
	}

	// Save assistant response
	assistantMessage := &Message{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		Role:           "assistant",
		Content:        response,
		Metadata: map[string]interface{}{
			"confidence":    confidence,
			"model":         chatbot.ModelConfig["model"],
			"response_time": time.Now().Unix(),
		},
		Timestamp: time.Now(),
	}
	if err := s.db.SaveMessage(assistantMessage); err != nil {
		s.logger.Printf("Failed to save assistant message: %v", err)
	}

	// Check for lead qualification
	leadQualification := s.AnalyzeForLeadQualification(req.Message, response)

	return &ChatResponse{
		Response:          response,
		Confidence:        confidence,
		ShouldEscalate:    confidence < 0.5,
		LeadQualification: leadQualification,
		ConversationID:    conversationID,
	}, nil
}

// GenerateAIResponse generates a response using Ollama
func (s *Server) GenerateAIResponse(chatbot *Chatbot, history []Message, userMessage string) (string, float64, error) {
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

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Post(s.config.OllamaURL+"/api/chat", "application/json", bytes.NewBuffer(reqBody))
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

// AnalyzeForLeadQualification analyzes messages for lead qualification signals
func (s *Server) AnalyzeForLeadQualification(userMessage, aiResponse string) map[string]interface{} {
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
	if strings.Contains(userLower, "buy") || strings.Contains(userLower, "purchase") || strings.Contains(userLower, "subscribe") {
		qualification["purchase_intent"] = true
	}

	return qualification
}