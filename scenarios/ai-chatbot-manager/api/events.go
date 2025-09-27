package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

// Event represents a system event
type Event struct {
	Name      string                 `json:"name"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

// EventPublisher handles event publishing to other scenarios
type EventPublisher struct {
	logger     *Logger
	httpClient *http.Client
	// In a production system, this would be configured with actual subscriber endpoints
	subscribers map[string][]string
}

// NewEventPublisher creates a new event publisher
func NewEventPublisher(logger *Logger) *EventPublisher {
	return &EventPublisher{
		logger:     logger,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		subscribers: map[string][]string{
			// These would be configured from environment or config file
			// For now, using placeholder endpoints
			"chatbot.conversation.started": {
				"http://localhost:8540/api/events", // analytics-engine placeholder
				"http://localhost:8541/api/events", // lead-scoring-system placeholder
			},
			"chatbot.lead.captured": {
				"http://localhost:8540/api/events",
				"http://localhost:8542/api/events", // crm-integration placeholder
			},
			"chatbot.conversation.ended": {
				"http://localhost:8540/api/events",
			},
		},
	}
}

// PublishEvent sends an event to all registered subscribers
func (ep *EventPublisher) PublishEvent(eventName string, payload map[string]interface{}) {
	event := Event{
		Name:      eventName,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}
	
	subscribers, exists := ep.subscribers[eventName]
	if !exists {
		ep.logger.Printf("No subscribers for event: %s", eventName)
		return
	}
	
	eventData, err := json.Marshal(event)
	if err != nil {
		ep.logger.Printf("Failed to marshal event %s: %v", eventName, err)
		return
	}
	
	// Publish to each subscriber asynchronously
	for _, endpoint := range subscribers {
		go ep.sendToSubscriber(endpoint, eventData)
	}
	
	ep.logger.Printf("Published event %s to %d subscribers", eventName, len(subscribers))
}

// sendToSubscriber sends event data to a specific subscriber
func (ep *EventPublisher) sendToSubscriber(endpoint string, eventData []byte) {
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(eventData))
	if err != nil {
		ep.logger.Printf("Failed to create request for %s: %v", endpoint, err)
		return
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Event-Source", "ai-chatbot-manager")
	
	resp, err := ep.httpClient.Do(req)
	if err != nil {
		// Log but don't fail - events are best-effort
		ep.logger.Printf("Failed to send event to %s: %v", endpoint, err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		ep.logger.Printf("Subscriber %s returned error status: %d", endpoint, resp.StatusCode)
	}
}

// ConversationStartedEvent publishes when a new conversation begins
func (ep *EventPublisher) ConversationStartedEvent(chatbotID, conversationID string, userContext map[string]interface{}) {
	payload := map[string]interface{}{
		"chatbot_id":      chatbotID,
		"conversation_id": conversationID,
		"user_context":    userContext,
	}
	ep.PublishEvent("chatbot.conversation.started", payload)
}

// LeadCapturedEvent publishes when lead information is captured
func (ep *EventPublisher) LeadCapturedEvent(chatbotID, conversationID string, leadData map[string]interface{}) {
	payload := map[string]interface{}{
		"chatbot_id":      chatbotID,
		"conversation_id": conversationID,
		"lead_data":       leadData,
		"captured_at":     time.Now().UTC().Format(time.RFC3339),
	}
	ep.PublishEvent("chatbot.lead.captured", payload)
}

// ConversationEndedEvent publishes when a conversation ends
func (ep *EventPublisher) ConversationEndedEvent(chatbotID, conversationID string, duration int, messageCount int) {
	payload := map[string]interface{}{
		"chatbot_id":      chatbotID,
		"conversation_id": conversationID,
		"duration_seconds": duration,
		"message_count":   messageCount,
	}
	ep.PublishEvent("chatbot.conversation.ended", payload)
}