package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

// WebhookSubscription represents a webhook subscription
type WebhookSubscription struct {
	ID            string     `json:"id"`
	URL           string     `json:"url"`
	Events        []string   `json:"events"` // e.g., ["api.created", "api.updated", "api.deprecated"]
	Active        bool       `json:"active"`
	CreatedAt     time.Time  `json:"created_at"`
	LastTriggered *time.Time `json:"last_triggered,omitempty"`
	FailureCount  int        `json:"failure_count"`
}

// WebhookEvent represents an event to be sent to webhooks
type WebhookEvent struct {
	ID        string                 `json:"id"`
	Event     string                 `json:"event"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// WebhookDelivery represents a webhook delivery attempt with retry information
type WebhookDelivery struct {
	WebhookID   string
	WebhookURL  string
	Event       WebhookEvent
	RetryCount  int
	NextRetryAt time.Time
	MaxRetries  int
}

// WebhookManager manages webhook subscriptions and delivery
type WebhookManager struct {
	db         *sql.DB
	mu         sync.RWMutex
	deliveryCh chan WebhookEvent
	retryCh    chan WebhookDelivery
	retryQueue []WebhookDelivery
}

// NewWebhookManager creates a new webhook manager
func NewWebhookManager(db *sql.DB) *WebhookManager {
	wm := &WebhookManager{
		db:         db,
		deliveryCh: make(chan WebhookEvent, 100),
		retryCh:    make(chan WebhookDelivery, 100),
		retryQueue: make([]WebhookDelivery, 0),
	}

	// Start the delivery worker
	go wm.deliveryWorker()

	// Start the retry worker
	go wm.retryWorker()

	return wm
}

// deliveryWorker processes webhook events in the background
func (wm *WebhookManager) deliveryWorker() {
	for event := range wm.deliveryCh {
		wm.deliverEvent(event)
	}
}

// retryWorker processes webhook retries with exponential backoff
func (wm *WebhookManager) retryWorker() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case delivery := <-wm.retryCh:
			// Add to retry queue
			wm.mu.Lock()
			wm.retryQueue = append(wm.retryQueue, delivery)
			wm.mu.Unlock()

		case <-ticker.C:
			// Check retry queue for deliveries that are ready
			wm.processRetryQueue()
		}
	}
}

// processRetryQueue processes webhooks that are ready for retry
func (wm *WebhookManager) processRetryQueue() {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	now := time.Now()
	readyDeliveries := make([]WebhookDelivery, 0)
	remainingQueue := make([]WebhookDelivery, 0)

	for _, delivery := range wm.retryQueue {
		if now.After(delivery.NextRetryAt) {
			readyDeliveries = append(readyDeliveries, delivery)
		} else {
			remainingQueue = append(remainingQueue, delivery)
		}
	}

	wm.retryQueue = remainingQueue

	// Process ready deliveries
	for _, delivery := range readyDeliveries {
		go wm.attemptDelivery(delivery)
	}
}

// calculateBackoff calculates exponential backoff with jitter
func calculateBackoff(retryCount int) time.Duration {
	// Base delay: 1s, 2s, 4s, 8s, 16s, 32s, 64s (max)
	baseDelay := time.Duration(1<<uint(retryCount)) * time.Second
	if baseDelay > 64*time.Second {
		baseDelay = 64 * time.Second
	}

	// Add jitter (0-20% of base delay)
	jitter := time.Duration(float64(baseDelay) * 0.2 * (0.5 + 0.5*float64(time.Now().UnixNano()%1000)/1000))
	return baseDelay + jitter
}

// deliverEvent sends an event to all relevant webhooks
func (wm *WebhookManager) deliverEvent(event WebhookEvent) {
	// Get all active webhooks subscribed to this event type
	rows, err := wm.db.Query(`
		SELECT id, url, failure_count 
		FROM webhook_subscriptions 
		WHERE active = true AND $1 = ANY(events)
	`, event.Event)
	if err != nil {
		log.Printf("Error fetching webhooks: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, url string
		var failureCount int
		if err := rows.Scan(&id, &url, &failureCount); err != nil {
			continue
		}

		// Create initial delivery attempt
		delivery := WebhookDelivery{
			WebhookID:   id,
			WebhookURL:  url,
			Event:       event,
			RetryCount:  0,
			NextRetryAt: time.Now(),
			MaxRetries:  5, // Maximum of 5 retries
		}

		// Attempt immediate delivery
		go wm.attemptDelivery(delivery)
	}
}

// attemptDelivery attempts to deliver a webhook with retry logic
func (wm *WebhookManager) attemptDelivery(delivery WebhookDelivery) {
	// Prepare the payload
	payload, err := json.Marshal(delivery.Event)
	if err != nil {
		log.Printf("Error marshaling webhook payload: %v", err)
		return
	}

	// Send the webhook with timeout
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", delivery.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		log.Printf("Error creating webhook request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event", delivery.Event.Event)
	req.Header.Set("X-Webhook-ID", delivery.Event.ID)
	req.Header.Set("X-Webhook-Retry", fmt.Sprintf("%d", delivery.RetryCount))

	resp, err := client.Do(req)

	success := err == nil && resp != nil && resp.StatusCode < 400

	if resp != nil {
		resp.Body.Close()
	}

	if success {
		// Reset failure count on success
		wm.db.Exec(`
			UPDATE webhook_subscriptions 
			SET failure_count = 0, last_triggered = $1
			WHERE id = $2
		`, time.Now(), delivery.WebhookID)

		// Log successful delivery with retry info
		if delivery.RetryCount > 0 {
			log.Printf("Webhook delivered successfully to %s after %d retries", delivery.WebhookURL, delivery.RetryCount)
		}
	} else {
		// Delivery failed
		if delivery.RetryCount < delivery.MaxRetries {
			// Schedule retry
			delivery.RetryCount++
			delivery.NextRetryAt = time.Now().Add(calculateBackoff(delivery.RetryCount))

			// Add to retry queue
			select {
			case wm.retryCh <- delivery:
				log.Printf("Webhook delivery to %s failed (attempt %d/%d), scheduled retry at %s",
					delivery.WebhookURL, delivery.RetryCount, delivery.MaxRetries+1, delivery.NextRetryAt.Format(time.RFC3339))
			default:
				log.Printf("Retry queue full, dropping webhook retry for %s", delivery.WebhookURL)
			}
		} else {
			// Max retries exceeded, disable webhook
			wm.db.Exec(`
				UPDATE webhook_subscriptions 
				SET failure_count = failure_count + 1,
				    active = false
				WHERE id = $1
			`, delivery.WebhookID)
			log.Printf("Webhook delivery to %s failed after %d attempts, webhook disabled", delivery.WebhookURL, delivery.MaxRetries+1)
		}
	}
}

// TriggerEvent queues an event for delivery
func (wm *WebhookManager) TriggerEvent(eventType string, data map[string]interface{}) {
	event := WebhookEvent{
		ID:        uuid.New().String(),
		Event:     eventType,
		Timestamp: time.Now(),
		Data:      data,
	}

	select {
	case wm.deliveryCh <- event:
		// Event queued successfully
	case <-time.After(time.Second):
		log.Printf("Warning: webhook delivery queue full, dropping event %s", event.ID)
	}
}

// Handler functions for webhook management

// createWebhookHandler creates a new webhook subscription
func createWebhookHandler(w http.ResponseWriter, r *http.Request) {
	var subscription struct {
		URL    string   `json:"url"`
		Events []string `json:"events"`
	}

	if err := json.NewDecoder(r.Body).Decode(&subscription); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate URL
	if subscription.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Validate events
	validEvents := map[string]bool{
		"api.created":    true,
		"api.updated":    true,
		"api.deleted":    true,
		"api.deprecated": true,
		"api.configured": true,
		"note.added":     true,
		"version.added":  true,
		"price.updated":  true,
	}

	for _, event := range subscription.Events {
		if !validEvents[event] {
			http.Error(w, fmt.Sprintf("Invalid event type: %s", event), http.StatusBadRequest)
			return
		}
	}

	// Create the subscription
	id := uuid.New().String()
	_, err := db.Exec(`
		INSERT INTO webhook_subscriptions (id, url, events, active, created_at, failure_count)
		VALUES ($1, $2, $3, true, $4, 0)
	`, id, subscription.URL, pq.Array(subscription.Events), time.Now())

	if err != nil {
		log.Printf("Error creating webhook: %v", err)
		http.Error(w, "Failed to create webhook", http.StatusInternalServerError)
		return
	}

	response := WebhookSubscription{
		ID:           id,
		URL:          subscription.URL,
		Events:       subscription.Events,
		Active:       true,
		CreatedAt:    time.Now(),
		FailureCount: 0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// listWebhooksHandler lists all webhook subscriptions
func listWebhooksHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT id, url, events, active, created_at, last_triggered, failure_count
		FROM webhook_subscriptions
		ORDER BY created_at DESC
	`)
	if err != nil {
		http.Error(w, "Failed to fetch webhooks", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var webhooks []WebhookSubscription
	for rows.Next() {
		var webhook WebhookSubscription
		var lastTriggered sql.NullTime
		var events pq.StringArray

		err := rows.Scan(
			&webhook.ID,
			&webhook.URL,
			&events,
			&webhook.Active,
			&webhook.CreatedAt,
			&lastTriggered,
			&webhook.FailureCount,
		)
		if err != nil {
			continue
		}

		webhook.Events = []string(events)
		if lastTriggered.Valid {
			webhook.LastTriggered = &lastTriggered.Time
		}

		webhooks = append(webhooks, webhook)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(webhooks)
}

// deleteWebhookHandler deletes a webhook subscription
func deleteWebhookHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	result, err := db.Exec("DELETE FROM webhook_subscriptions WHERE id = $1", id)
	if err != nil {
		http.Error(w, "Failed to delete webhook", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Webhook not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// testWebhookHandler tests a webhook by sending a test event
func testWebhookHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var url string
	err := db.QueryRow("SELECT url FROM webhook_subscriptions WHERE id = $1", id).Scan(&url)
	if err != nil {
		http.Error(w, "Webhook not found", http.StatusNotFound)
		return
	}

	// Send test event
	testEvent := WebhookEvent{
		ID:        uuid.New().String(),
		Event:     "webhook.test",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"message": "This is a test webhook event",
		},
	}

	payload, _ := json.Marshal(testEvent)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(payload))

	if err != nil {
		http.Error(w, fmt.Sprintf("Test failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		http.Error(w, fmt.Sprintf("Webhook returned status %d", resp.StatusCode), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"message":     "Test webhook delivered successfully",
		"status_code": resp.StatusCode,
	})
}
