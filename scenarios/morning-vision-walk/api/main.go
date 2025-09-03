package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

type ConversationMessage struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type ConversationRequest struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	Message   string `json:"message"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

type InsightRequest struct {
	SessionID string                `json:"session_id"`
	UserID    string                `json:"user_id"`
	History   []ConversationMessage `json:"conversation_history"`
	Context   map[string]interface{} `json:"context"`
}

type TaskPrioritizationRequest struct {
	SessionID string                   `json:"session_id"`
	UserID    string                   `json:"user_id"`
	Tasks     []map[string]interface{} `json:"tasks"`
	Context   map[string]interface{}   `json:"context"`
}

type Session struct {
	ID           string                  `json:"id"`
	UserID       string                  `json:"user_id"`
	StartTime    time.Time               `json:"start_time"`
	EndTime      *time.Time              `json:"end_time,omitempty"`
	Messages     []ConversationMessage   `json:"messages"`
	Insights     []map[string]interface{} `json:"insights"`
	DailyVision  string                  `json:"daily_vision"`
	ActionItems  []string                `json:"action_items"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

var activeSessions = make(map[string]*Session)

func main() {
	port := getEnv("API_PORT", getEnv("PORT", ""))

	router := mux.NewRouter()

	// API routes
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/api/conversation/start", startConversationHandler).Methods("POST")
	router.HandleFunc("/api/conversation/message", sendMessageHandler).Methods("POST")
	router.HandleFunc("/api/conversation/end", endConversationHandler).Methods("POST")
	router.HandleFunc("/api/insights/generate", generateInsightsHandler).Methods("POST")
	router.HandleFunc("/api/tasks/prioritize", prioritizeTasksHandler).Methods("POST")
	router.HandleFunc("/api/context/gather", gatherContextHandler).Methods("GET")
	router.HandleFunc("/api/session/{sessionId}", getSessionHandler).Methods("GET")
	router.HandleFunc("/api/session/{sessionId}/export", exportSessionHandler).Methods("GET")
	
	// WebSocket for real-time conversation
	router.HandleFunc("/ws/conversation", handleWebSocket)

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)
	
	log.Printf("Morning Vision Walk API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status": "healthy",
		"service": "morning-vision-walk",
		"timestamp": time.Now().Unix(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func startConversationHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sessionID := fmt.Sprintf("session-%d", time.Now().Unix())
	session := &Session{
		ID:        sessionID,
		UserID:    req.UserID,
		StartTime: time.Now(),
		Messages:  []ConversationMessage{},
		Insights:  []map[string]interface{}{},
		ActionItems: []string{},
	}
	
	activeSessions[sessionID] = session

	// Call context gatherer workflow
	contextData := callN8nWorkflow("vision-context-gatherer", map[string]interface{}{
		"session_id": sessionID,
		"user_id": req.UserID,
		"context_type": "vrooli_state",
	})

	response := map[string]interface{}{
		"session_id": sessionID,
		"started_at": session.StartTime,
		"context": contextData,
		"welcome_message": "Good morning! Ready for our vision walk? Let's explore what's on your mind about Vrooli today.",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	var req ConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	session, exists := activeSessions[req.SessionID]
	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Add user message to history
	userMsg := ConversationMessage{
		ID:        fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		SessionID: req.SessionID,
		Role:      "user",
		Content:   req.Message,
		Timestamp: time.Now(),
	}
	session.Messages = append(session.Messages, userMsg)

	// Call vision conversation workflow
	conversationResponse := callN8nWorkflow("vision-conversation", map[string]interface{}{
		"session_id": req.SessionID,
		"user_id": req.UserID,
		"message": req.Message,
		"conversation_history": session.Messages,
		"context": req.Context,
	})

	// Add assistant response to history
	assistantContent := "I understand. Let me help you explore that idea further."
	if resp, ok := conversationResponse["response"].(string); ok {
		assistantContent = resp
	}

	assistantMsg := ConversationMessage{
		ID:        fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		SessionID: req.SessionID,
		Role:      "assistant",
		Content:   assistantContent,
		Timestamp: time.Now(),
	}
	session.Messages = append(session.Messages, assistantMsg)

	// Generate insights periodically
	if len(session.Messages) % 6 == 0 { // Every 3 exchanges
		go generateInsights(session)
	}

	response := map[string]interface{}{
		"message": assistantMsg,
		"session_stats": map[string]interface{}{
			"message_count": len(session.Messages),
			"duration_minutes": time.Since(session.StartTime).Minutes(),
			"insights_generated": len(session.Insights),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func endConversationHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID string `json:"session_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	session, exists := activeSessions[req.SessionID]
	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	now := time.Now()
	session.EndTime = &now

	// Generate final insights and daily planning
	finalInsights := callN8nWorkflow("insight-generator", map[string]interface{}{
		"session_id": session.ID,
		"user_id": session.UserID,
		"conversation_history": session.Messages,
		"context": map[string]interface{}{
			"session_type": "morning_vision_walk",
			"duration": time.Since(session.StartTime).Minutes(),
		},
	})

	// Create daily plan
	dailyPlan := callN8nWorkflow("daily-planning", map[string]interface{}{
		"session_id": session.ID,
		"user_id": session.UserID,
		"insights": finalInsights,
		"conversation_summary": summarizeConversation(session.Messages),
	})

	response := map[string]interface{}{
		"session_id": session.ID,
		"duration_minutes": time.Since(session.StartTime).Minutes(),
		"total_messages": len(session.Messages),
		"insights_generated": len(session.Insights),
		"daily_plan": dailyPlan,
		"final_insights": finalInsights,
	}

	// Clean up session after sending response
	delete(activeSessions, req.SessionID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func generateInsightsHandler(w http.ResponseWriter, r *http.Request) {
	var req InsightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	insights := callN8nWorkflow("insight-generator", map[string]interface{}{
		"session_id": req.SessionID,
		"user_id": req.UserID,
		"conversation_history": req.History,
		"context": req.Context,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(insights)
}

func prioritizeTasksHandler(w http.ResponseWriter, r *http.Request) {
	var req TaskPrioritizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	priorities := callN8nWorkflow("task-prioritizer", map[string]interface{}{
		"session_id": req.SessionID,
		"user_id": req.UserID,
		"tasks": req.Tasks,
		"context": req.Context,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(priorities)
}

func gatherContextHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	userID := r.URL.Query().Get("user_id")
	
	context := callN8nWorkflow("context-gatherer", map[string]interface{}{
		"session_id": sessionID,
		"user_id": userID,
		"context_type": "full",
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(context)
}

func getSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]
	
	session, exists := activeSessions[sessionID]
	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func exportSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]
	
	session, exists := activeSessions[sessionID]
	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	export := map[string]interface{}{
		"session": session,
		"export_time": time.Now(),
		"summary": summarizeConversation(session.Messages),
		"key_insights": session.Insights,
		"action_items": session.ActionItems,
		"daily_vision": session.DailyVision,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"vision-walk-%s.json\"", sessionID))
	json.NewEncoder(w).Encode(export)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	sessionID := r.URL.Query().Get("session_id")
	session, exists := activeSessions[sessionID]
	if !exists {
		conn.WriteJSON(map[string]string{"error": "Session not found"})
		return
	}

	for {
		var msg ConversationMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		msg.SessionID = sessionID
		msg.Timestamp = time.Now()
		session.Messages = append(session.Messages, msg)

		// Echo back with response
		response := map[string]interface{}{
			"type": "message",
			"data": msg,
		}
		
		if err := conn.WriteJSON(response); err != nil {
			log.Printf("WebSocket write error: %v", err)
			break
		}
	}
}

// Helper functions
func callN8nWorkflow(workflow string, data map[string]interface{}) map[string]interface{} {
	// Use the CLI approach to call n8n workflows
	dataJSON, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling data for workflow %s: %v", workflow, err)
		return map[string]interface{}{
			"success": false,
			"error": err.Error(),
		}
	}

	// Use vrooli resource CLI to execute the workflow
	cmd := exec.Command("bash", "-c", fmt.Sprintf("vrooli resource n8n execute-workflow '%s' '%s'", workflow, string(dataJSON)))
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error executing workflow %s: %v, output: %s", workflow, err, string(output))
		return map[string]interface{}{
			"success": false,
			"error": fmt.Sprintf("Workflow execution failed: %v", err),
			"output": string(output),
		}
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		// If output is not JSON, return it as a string
		return map[string]interface{}{
			"success": true,
			"workflow": workflow,
			"response": string(output),
			"timestamp": time.Now().Unix(),
		}
	}

	result["workflow"] = workflow
	result["timestamp"] = time.Now().Unix()
	return result
}

func generateInsights(session *Session) {
	insights := callN8nWorkflow("insight-generator", map[string]interface{}{
		"session_id": session.ID,
		"user_id": session.UserID,
		"conversation_history": session.Messages,
	})
	
	if insights != nil {
		session.Insights = append(session.Insights, insights)
	}
}

func summarizeConversation(messages []ConversationMessage) string {
	if len(messages) == 0 {
		return "No conversation to summarize"
	}
	
	// In production, this would use AI to summarize
	return fmt.Sprintf("Conversation with %d messages discussing Vrooli improvements and daily planning", len(messages))
}
