package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// isValidUUID checks if a string is a valid UUID
func isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

// HealthHandler handles health check requests
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	overallStatus := "healthy"
	startTime := time.Now()
	
	// Check if we have a cached health check result (cache for 30 seconds)
	cacheDuration := 30 * time.Second
	s.healthCacheMutex.RLock()
	if time.Since(s.healthCacheTime) < cacheDuration && len(s.healthCache) > 0 {
		cachedResponse := s.healthCache
		s.healthCacheMutex.RUnlock()
		// Return cached response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cachedResponse)
		return
	}
	s.healthCacheMutex.RUnlock()
	
	// Schema-compliant health response
	healthResponse := map[string]interface{}{
		"status":    overallStatus,
		"service":   serviceName,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"readiness": true, // Service is ready to accept requests
		"version":   apiVersion,
		"dependencies": map[string]interface{}{},
		"metrics": map[string]interface{}{
			"uptime_seconds": time.Since(startTime).Seconds(), // Simplified uptime
		},
	}
	
	dependencies := healthResponse["dependencies"].(map[string]interface{})
	
	// 1. Check database connection (critical for chatbot CRUD operations)
	dbHealth := s.checkDatabase()
	dependencies["database"] = dbHealth
	if dbHealth["connected"] == false {
		overallStatus = "unhealthy" // Database is critical
	}
	
	// 2. Check Ollama connection (critical for AI chat responses)
	ollamaHealth := s.checkOllama()
	dependencies["ollama"] = ollamaHealth
	if ollamaHealth["connected"] == false {
		overallStatus = "unhealthy" // Ollama is critical for functionality
	}
	
	// 3. Check widget file system (for generated widget files)
	widgetHealth := s.checkWidgetFileSystem()
	dependencies["widget_storage"] = widgetHealth
	if widgetHealth["connected"] == false {
		if overallStatus == "healthy" {
			overallStatus = "degraded"
		}
	}
	
	// 4. Test basic AI inference capability (if Ollama is available)
	// Skip inference check for basic health checks to improve response time
	if r.URL.Query().Get("detailed") == "true" && ollamaHealth["connected"] == true {
		inferenceHealth := s.checkAIInference()
		dependencies["ai_inference"] = inferenceHealth
		if inferenceHealth["connected"] == false {
			if overallStatus == "healthy" {
				overallStatus = "degraded"
			}
		}
	}
	
	// Update overall status
	healthResponse["status"] = overallStatus
	
	// Cache the health check result
	s.healthCacheMutex.Lock()
	s.healthCache = healthResponse
	s.healthCacheTime = time.Now()
	s.healthCacheMutex.Unlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(healthResponse)
}

// checkDatabase tests PostgreSQL database connectivity and schema
func (s *Server) checkDatabase() map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}
	
	// Test database ping
	if err := s.db.Ping(); err != nil {
		result["error"] = map[string]interface{}{
			"code":      "DATABASE_PING_FAILED",
			"message":   fmt.Sprintf("Database connection failed: %v", err),
			"category":  "resource",
			"retryable": true,
		}
		return result
	}
	
	// Test basic schema by trying to count chatbots
	chatbots, err := s.db.ListChatbots(false)
	if err != nil {
		result["error"] = map[string]interface{}{
			"code":      "DATABASE_QUERY_FAILED",
			"message":   fmt.Sprintf("Database query failed: %v", err),
			"category":  "resource",
			"retryable": true,
		}
		return result
	}
	
	result["connected"] = true
	result["chatbot_count"] = len(chatbots)
	return result
}

// checkOllama tests Ollama AI service connectivity and model availability
func (s *Server) checkOllama() map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}
	
	if s.config.OllamaURL == "" {
		result["error"] = map[string]interface{}{
			"code":      "OLLAMA_URL_MISSING",
			"message":   "Ollama URL not configured",
			"category":  "configuration",
			"retryable": false,
		}
		return result
	}
	
	// Test Ollama API availability
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(s.config.OllamaURL + "/api/tags")
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			result["error"] = map[string]interface{}{
				"code":      "CONNECTION_REFUSED",
				"message":   "Ollama service not running",
				"category":  "network",
				"retryable": true,
			}
		} else {
			result["error"] = map[string]interface{}{
				"code":      "OLLAMA_CONNECTION_FAILED",
				"message":   fmt.Sprintf("Cannot connect to Ollama: %v", err),
				"category":  "network",
				"retryable": true,
			}
		}
		return result
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		result["error"] = map[string]interface{}{
			"code":      fmt.Sprintf("HTTP_%d", resp.StatusCode),
			"message":   fmt.Sprintf("Ollama returned status %d", resp.StatusCode),
			"category":  "network",
			"retryable": resp.StatusCode >= 500,
		}
		return result
	}
	
	// Parse response to check available models
	var tagsResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tagsResponse); err != nil {
		result["error"] = map[string]interface{}{
			"code":      "OLLAMA_RESPONSE_INVALID",
			"message":   fmt.Sprintf("Cannot parse Ollama response: %v", err),
			"category":  "internal",
			"retryable": true,
		}
		return result
	}
	
	result["connected"] = true
	
	// Check if models array exists and count models
	if models, ok := tagsResponse["models"]; ok {
		if modelsArray, ok := models.([]interface{}); ok {
			result["available_models"] = len(modelsArray)
		}
	}
	
	return result
}

// checkWidgetFileSystem tests widget file generation and storage
func (s *Server) checkWidgetFileSystem() map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}
	
	// Check if static directory exists for widget files
	staticDir := "static"
	if _, err := os.Stat(staticDir); err != nil {
		if os.IsNotExist(err) {
			// Try to create it
			if err := os.MkdirAll(staticDir, 0755); err != nil {
				result["error"] = map[string]interface{}{
					"code":      "WIDGET_DIR_CREATE_FAILED",
					"message":   fmt.Sprintf("Cannot create widget directory: %v", err),
					"category":  "resource",
					"retryable": false,
				}
				return result
			}
		} else {
			result["error"] = map[string]interface{}{
				"code":      "WIDGET_DIR_ACCESS_FAILED",
				"message":   fmt.Sprintf("Cannot access widget directory: %v", err),
				"category":  "resource",
				"retryable": false,
			}
			return result
		}
	}
	
	// Test writing a small test file
	testFile := filepath.Join(staticDir, ".health_check_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		result["error"] = map[string]interface{}{
			"code":      "WIDGET_WRITE_TEST_FAILED",
			"message":   fmt.Sprintf("Cannot write widget test file: %v", err),
			"category":  "resource",
			"retryable": true,
		}
		return result
	}
	
	// Clean up test file
	os.Remove(testFile)
	
	result["connected"] = true
	return result
}

// checkAIInference tests basic AI model inference capability
func (s *Server) checkAIInference() map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}
	
	// Create a simple test request
	testRequest := map[string]interface{}{
		"model": "llama3.2", // Default model
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": "Hello",
			},
		},
		"stream": false,
	}
	
	requestBody, err := json.Marshal(testRequest)
	if err != nil {
		result["error"] = map[string]interface{}{
			"code":      "INFERENCE_REQUEST_MARSHAL_FAILED",
			"message":   "Cannot marshal inference test request",
			"category":  "internal",
			"retryable": true,
		}
		return result
	}
	
	// Send inference request with short timeout
	client := &http.Client{Timeout: 10 * time.Second}
	inferenceStart := time.Now()
	resp, err := client.Post(s.config.OllamaURL+"/api/chat", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		if strings.Contains(err.Error(), "timeout") {
			result["error"] = map[string]interface{}{
				"code":      "INFERENCE_TIMEOUT",
				"message":   "AI inference test timed out",
				"category":  "network",
				"retryable": true,
			}
		} else {
			result["error"] = map[string]interface{}{
				"code":      "INFERENCE_REQUEST_FAILED",
				"message":   fmt.Sprintf("AI inference test failed: %v", err),
				"category":  "network",
				"retryable": true,
			}
		}
		return result
	}
	defer resp.Body.Close()
	
	inferenceTime := time.Since(inferenceStart)
	
	if resp.StatusCode != http.StatusOK {
		result["error"] = map[string]interface{}{
			"code":      fmt.Sprintf("INFERENCE_HTTP_%d", resp.StatusCode),
			"message":   fmt.Sprintf("AI inference returned status %d", resp.StatusCode),
			"category":  "network",
			"retryable": resp.StatusCode >= 500,
		}
		return result
	}
	
	// Parse response
	var inferenceResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&inferenceResponse); err != nil {
		result["error"] = map[string]interface{}{
			"code":      "INFERENCE_RESPONSE_INVALID",
			"message":   fmt.Sprintf("Cannot parse inference response: %v", err),
			"category":  "internal",
			"retryable": true,
		}
		return result
	}
	
	// Check if response has expected structure
	if _, hasMessage := inferenceResponse["message"]; !hasMessage {
		result["error"] = map[string]interface{}{
			"code":      "INFERENCE_RESPONSE_MALFORMED",
			"message":   "AI inference response missing message field",
			"category":  "internal",
			"retryable": true,
		}
		return result
	}
	
	result["connected"] = true
	result["latency_ms"] = float64(inferenceTime.Nanoseconds()) / 1e6 // Convert to milliseconds
	return result
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

	// Set default escalation config if not provided
	if req.EscalationConfig == nil {
		req.EscalationConfig = map[string]interface{}{
			"enabled":     false,
			"threshold":   0.5,
			"webhook_url": nil,
			"email":       nil,
		}
	}

	// Create chatbot
	chatbot := &Chatbot{
		ID:               uuid.New().String(),
		Name:             req.Name,
		Description:      req.Description,
		Personality:      req.Personality,
		KnowledgeBase:    req.KnowledgeBase,
		ModelConfig:      req.ModelConfig,
		WidgetConfig:     req.WidgetConfig,
		EscalationConfig: req.EscalationConfig,
		IsActive:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
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

	// Validate UUID format first
	if !isValidUUID(chatbotID) {
		http.Error(w, "Invalid chatbot ID format", http.StatusBadRequest)
		return
	}

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

	// Validate UUID format first
	if !isValidUUID(chatbotID) {
		http.Error(w, "Invalid chatbot ID format", http.StatusBadRequest)
		return
	}

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

	// Validate UUID format first
	if !isValidUUID(chatbotID) {
		http.Error(w, "Invalid chatbot ID format", http.StatusBadRequest)
		return
	}

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

	// Validate UUID format first
	if !isValidUUID(chatbotID) {
		http.Error(w, "Invalid chatbot ID format", http.StatusBadRequest)
		return
	}

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

	// Validate UUID format first
	if !isValidUUID(chatbotID) {
		http.Error(w, "Invalid chatbot ID format", http.StatusBadRequest)
		return
	}

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
	conversationID, isNew, err := s.db.GetOrCreateConversation(chatbotID, req.SessionID, req.UserIP)
	if err != nil {
		s.logger.Printf("Failed to get/create conversation: %v", err)
		return nil, fmt.Errorf("failed to manage conversation")
	}
	
	// Publish event if this is a new conversation
	if isNew {
		userContext := map[string]interface{}{
			"session_id": req.SessionID,
			"user_ip":    req.UserIP,
			"context":    req.Context,
		}
		s.eventPublisher.ConversationStartedEvent(chatbotID, conversationID, userContext)
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
	
	// Publish lead event if qualified
	if len(leadQualification) > 0 && (leadQualification["email_mentioned"] == true || 
		leadQualification["purchase_intent"] == true || 
		leadQualification["demo_interest"] == true) {
		leadData := map[string]interface{}{
			"qualification": leadQualification,
			"message":       req.Message,
			"session_id":    req.SessionID,
		}
		s.eventPublisher.LeadCapturedEvent(chatbotID, conversationID, leadData)
	}

	// Check if escalation is needed
	shouldEscalate := false
	escalationReason := ""
	
	if escalationConfig, ok := chatbot.EscalationConfig["enabled"].(bool); ok && escalationConfig {
		threshold := 0.5
		if thresh, ok := chatbot.EscalationConfig["threshold"].(float64); ok {
			threshold = thresh
		}
		
		// Check confidence threshold
		if confidence < threshold {
			shouldEscalate = true
			escalationReason = fmt.Sprintf("Low confidence: %.2f < %.2f threshold", confidence, threshold)
		}
		
		// Check for explicit escalation keywords
		escalationKeywords := []string{"speak to human", "talk to agent", "human help", "real person", "escalate", "manager"}
		lowerMessage := strings.ToLower(req.Message)
		for _, keyword := range escalationKeywords {
			if strings.Contains(lowerMessage, keyword) {
				shouldEscalate = true
				escalationReason = "User requested human assistance"
				break
			}
		}
	}
	
	// Handle escalation if needed
	if shouldEscalate {
		escalation := &Escalation{
			ID:              uuid.New().String(),
			ConversationID:  conversationID,
			ChatbotID:       chatbotID,
			Reason:          escalationReason,
			ConfidenceScore: confidence,
			EscalationType:  "low_confidence",
			Status:          "pending",
			EscalatedAt:     time.Now(),
			EmailSent:       false,
		}
		
		// Save escalation to database
		if err := s.db.CreateEscalation(escalation); err != nil {
			s.logger.Printf("Failed to save escalation: %v", err)
		}
		
		// Trigger webhook if configured
		if webhookURL, ok := chatbot.EscalationConfig["webhook_url"].(string); ok && webhookURL != "" {
			go s.TriggerEscalationWebhook(webhookURL, escalation)
		}
		
		// Send email notification if configured
		if email, ok := chatbot.EscalationConfig["email"].(string); ok && email != "" {
			go s.SendEscalationEmail(email, escalation)
		}
	}

	return &ChatResponse{
		Response:          response,
		Confidence:        confidence,
		ShouldEscalate:    shouldEscalate,
		EscalationReason:  escalationReason,
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

// TriggerEscalationWebhook sends escalation notification to webhook
func (s *Server) TriggerEscalationWebhook(webhookURL string, escalation *Escalation) {
	payload := map[string]interface{}{
		"escalation_id":    escalation.ID,
		"conversation_id":  escalation.ConversationID,
		"chatbot_id":       escalation.ChatbotID,
		"reason":           escalation.Reason,
		"confidence_score": escalation.ConfidenceScore,
		"escalation_type":  escalation.EscalationType,
		"escalated_at":     escalation.EscalatedAt,
		"conversation_url": fmt.Sprintf("%s/conversations/%s", s.baseURL, escalation.ConversationID),
	}
	
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		s.logger.Printf("Failed to marshal webhook payload: %v", err)
		return
	}
	
	req, err := http.NewRequest("POST", webhookURL, bytes.NewReader(jsonPayload))
	if err != nil {
		s.logger.Printf("Failed to create webhook request: %v", err)
		return
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "AI-Chatbot-Manager/1.0")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Printf("Webhook request failed: %v", err)
		return
	}
	defer resp.Body.Close()
	
	s.logger.Printf("Escalation webhook triggered for conversation %s (status: %d)", escalation.ConversationID, resp.StatusCode)
}

// SendEscalationEmail sends escalation notification via email
func (s *Server) SendEscalationEmail(email string, escalation *Escalation) {
	// This is a placeholder implementation
	// In production, you would integrate with an email service like SendGrid, AWS SES, etc.
	
	subject := fmt.Sprintf("Escalation Alert: Conversation %s requires attention", escalation.ConversationID[:8])
	body := fmt.Sprintf(`
An escalation has been triggered in the AI Chatbot Manager system.

Details:
- Conversation ID: %s
- Chatbot ID: %s
- Reason: %s
- Confidence Score: %.2f
- Type: %s
- Time: %s

Please review this conversation and take appropriate action.

View conversation: %s/conversations/%s
`,
		escalation.ConversationID,
		escalation.ChatbotID,
		escalation.Reason,
		escalation.ConfidenceScore,
		escalation.EscalationType,
		escalation.EscalatedAt.Format(time.RFC3339),
		s.baseURL,
		escalation.ConversationID,
	)
	
	// Log the email details (in production, send actual email)
	s.logger.Printf("Email notification prepared for %s:\nSubject: %s\nBody: %s", email, subject, body)
	
	// TODO: Integrate with email service provider
	// Example: sendgrid.Send(email, subject, body)
}

// GetEscalationsHandler returns pending escalations for a chatbot
func (s *Server) GetEscalationsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatbotID := vars["id"]
	
	// Validate UUID format
	if !isValidUUID(chatbotID) {
		http.Error(w, "Invalid chatbot ID format", http.StatusBadRequest)
		return
	}
	
	escalations, err := s.db.GetPendingEscalations(chatbotID)
	if err != nil {
		s.logger.Printf("Failed to get escalations: %v", err)
		http.Error(w, "Failed to retrieve escalations", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(escalations)
}

// UpdateEscalationHandler updates the status of an escalation
func (s *Server) UpdateEscalationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	escalationID := vars["id"]
	
	// Validate UUID format
	if !isValidUUID(escalationID) {
		http.Error(w, "Invalid escalation ID format", http.StatusBadRequest)
		return
	}
	
	// Parse request body
	var update struct {
		Status string `json:"status"`
		Notes  string `json:"notes"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Validate status
	validStatuses := []string{"pending", "in_progress", "resolved", "dismissed"}
	isValid := false
	for _, status := range validStatuses {
		if update.Status == status {
			isValid = true
			break
		}
	}
	
	if !isValid {
		http.Error(w, "Invalid status. Must be one of: pending, in_progress, resolved, dismissed", http.StatusBadRequest)
		return
	}
	
	// Update escalation
	if err := s.db.UpdateEscalationStatus(escalationID, update.Status, update.Notes); err != nil {
		s.logger.Printf("Failed to update escalation: %v", err)
		http.Error(w, "Failed to update escalation", http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": "Escalation updated successfully",
	})
}