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

// HealthHandler handles health check requests
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	overallStatus := "healthy"
	startTime := time.Now()
	
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
	if ollamaHealth["connected"] == true {
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