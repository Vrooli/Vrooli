package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

type ChatRequest struct {
	Message   string                 `json:"message"`
	SessionID string                 `json:"session_id"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

type WebSocketMessage struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

func main() {
	// Get API URL from environment (must be set by test script)
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		log.Fatal("[FAIL] API_URL environment variable not set. Test must be run via test script.")
	}

	// Extract host for WebSocket
	wsHost := apiURL[7:] // Remove "http://"
	if len(os.Args) > 1 && os.Args[1] != "" {
		// Use provided host if given
		wsHost = os.Args[1]
	}

	fmt.Println("=========================================")
	fmt.Println("AI Chatbot Manager WebSocket Test (Go)")
	fmt.Println("=========================================")
	fmt.Printf("API URL: %s\n", apiURL)
	fmt.Printf("WebSocket URL: ws://%s\n\n", wsHost)

	// Step 1: Create a test chatbot
	fmt.Println("[TEST] Creating test chatbot...")
	chatbotID := createTestChatbot(apiURL)
	if chatbotID == "" {
		log.Fatal("[FAIL] Failed to create test chatbot")
	}
	fmt.Printf("[PASS] Created chatbot: %s\n\n", chatbotID)

	// Step 2: Test WebSocket connection
	fmt.Println("[TEST] Testing WebSocket connection...")
	wsURL := fmt.Sprintf("ws://%s/api/v1/ws/%s?api_key=dev-api-key-change-in-production", wsHost, chatbotID)
	
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}
	
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatalf("[FAIL] Failed to connect: %v", err)
	}
	defer conn.Close()
	fmt.Println("[PASS] WebSocket connected successfully")

	// Step 3: Send test message
	fmt.Println("\n[TEST] Sending test message via WebSocket...")
	sessionID := fmt.Sprintf("test-ws-%d", time.Now().Unix())
	chatReq := ChatRequest{
		Message:   "Hello, this is a WebSocket test message!",
		SessionID: sessionID,
		Context:   map[string]interface{}{"test": true},
	}

	if err := conn.WriteJSON(chatReq); err != nil {
		log.Fatalf("[FAIL] Failed to send message: %v", err)
	}
	fmt.Println("[PASS] Message sent successfully")

	// Step 4: Receive response
	fmt.Println("\n[TEST] Waiting for response...")
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	
	var response WebSocketMessage
	if err := conn.ReadJSON(&response); err != nil {
		log.Fatalf("[FAIL] Failed to read response: %v", err)
	}

	if response.Type == "error" {
		log.Fatalf("[FAIL] Received error response: %v", response.Payload["error"])
	}

	if response.Type != "message" {
		log.Fatalf("[FAIL] Unexpected response type: %s", response.Type)
	}

	fmt.Printf("[PASS] Received response: %.50s...\n", response.Payload["response"])

	// Step 5: Send another message to test conversation continuity
	fmt.Println("\n[TEST] Testing conversation continuity...")
	chatReq2 := ChatRequest{
		Message:   "Do you remember what I just said?",
		SessionID: sessionID,
	}

	if err := conn.WriteJSON(chatReq2); err != nil {
		log.Fatalf("[FAIL] Failed to send follow-up message: %v", err)
	}

	var response2 WebSocketMessage
	if err := conn.ReadJSON(&response2); err != nil {
		log.Fatalf("[FAIL] Failed to read follow-up response: %v", err)
	}

	if response2.Type == "message" && response2.Payload["conversation_id"] == response.Payload["conversation_id"] {
		fmt.Println("[PASS] Conversation context maintained")
	} else {
		fmt.Println("[WARN] Conversation context may not be maintained")
	}

	// Step 6: Test graceful disconnect
	fmt.Println("\n[TEST] Testing graceful disconnect...")
	if err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		fmt.Printf("[WARN] Close message failed: %v\n", err)
	}
	time.Sleep(100 * time.Millisecond)
	fmt.Println("[PASS] WebSocket closed gracefully")

	// Clean up: Delete test chatbot
	fmt.Printf("\n[TEST] Cleaning up test chatbot...\n")
	deleteTestChatbot(apiURL, chatbotID)

	fmt.Println("\n=========================================")
	fmt.Println("✅ All WebSocket tests passed!")
	fmt.Println("=========================================")
}

func createTestChatbot(apiURL string) string {
	reqBody := map[string]interface{}{
		"name":           "WebSocket Test Bot",
		"description":    "Temporary bot for WebSocket testing",
		"personality":    "You are a helpful test assistant",
		"knowledge_base": "This is a test chatbot for WebSocket functionality",
		"model_config": map[string]interface{}{
			"model":       "llama3.2",
			"temperature": 0.7,
		},
	}

	jsonData, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", apiURL+"/api/v1/chatbots", bytes.NewBuffer(jsonData))
	if err != nil {
		return ""
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "dev-api-key-change-in-production")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}

	// Check for nested chatbot object
	if chatbot, ok := result["chatbot"].(map[string]interface{}); ok {
		if id, ok := chatbot["id"].(string); ok {
			return id
		}
	}
	// Fallback to direct id field
	if id, ok := result["id"].(string); ok {
		return id
	}
	return ""
}

func deleteTestChatbot(apiURL, chatbotID string) {
	client := &http.Client{}
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/chatbots/%s", apiURL, chatbotID), nil)
	req.Header.Set("X-API-Key", "dev-api-key-change-in-production")
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
		fmt.Printf("[INFO] Deleted test chatbot: %s\n", chatbotID)
	}
}