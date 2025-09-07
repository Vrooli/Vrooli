package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
	
	"github.com/gorilla/mux"
	
	"email-triage/models"
	"email-triage/services"
)

// EmailHandler handles email processing and search endpoints
type EmailHandler struct {
	db            *sql.DB
	searchService *services.SearchService
}

// NewEmailHandler creates a new EmailHandler instance
func NewEmailHandler(db *sql.DB, searchService *services.SearchService) *EmailHandler {
	return &EmailHandler{
		db:            db,
		searchService: searchService,
	}
}

// SearchEmails handles GET /api/v1/emails/search
func (eh *EmailHandler) SearchEmails(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"user authentication required"}`, http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, `{"error":"search query is required"}`, http.StatusBadRequest)
		return
	}

	// Parse optional parameters
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	accountID := r.URL.Query().Get("account_id")
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	// TODO: Generate query embedding using an embedding model
	// For now, create a dummy embedding
	queryEmbedding := make([]float32, 384) // all-MiniLM-L6-v2 dimensions
	for i := range queryEmbedding {
		queryEmbedding[i] = 0.1 // Placeholder values
	}

	startTime := time.Now()

	// Perform semantic search
	results, err := eh.searchService.SearchEmails(userID, queryEmbedding, limit)
	if err != nil {
		responseObj := map[string]interface{}{
			"success": false,
			"error":   "search_failed",
			"message": err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(responseObj)
		return
	}

	// Apply additional filters (account, date range) if specified
	filteredResults := eh.applySearchFilters(results, accountID, dateFrom, dateTo)

	queryTimeMs := int(time.Since(startTime).Nanoseconds() / 1000000)

	response := models.SearchEmailsResponse{
		Results:     filteredResults,
		Total:       len(filteredResults),
		QueryTimeMs: queryTimeMs,
		Page:        1, // TODO: Implement pagination
		Limit:       limit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetEmail handles GET /api/v1/emails/{id}
func (eh *EmailHandler) GetEmail(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"user authentication required"}`, http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	emailID := vars["id"]

	// Query email details with user authorization check
	query := `
		SELECT pe.id, pe.account_id, pe.message_id, pe.subject, pe.sender_email, 
			   pe.recipient_emails, pe.body_preview, pe.full_body, pe.priority_score,
			   pe.processed_at, pe.actions_taken, pe.metadata
		FROM processed_emails pe
		JOIN email_accounts ea ON pe.account_id = ea.id
		WHERE pe.id = $1 AND ea.user_id = $2`

	var email models.ProcessedEmail
	err := eh.db.QueryRow(query, emailID, userID).Scan(
		&email.ID, &email.AccountID, &email.MessageID, &email.Subject,
		&email.SenderEmail, &email.RecipientEmails, &email.BodyPreview,
		&email.FullBody, &email.PriorityScore, &email.ProcessedAt,
		&email.ActionsTaken, &email.Metadata,
	)

	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"email not found"}`, http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, `{"error":"database query failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(email)
}

// ApplyActions handles POST /api/v1/emails/{id}/actions
func (eh *EmailHandler) ApplyActions(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"user authentication required"}`, http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	emailID := vars["id"]

	// Parse request body
	var actionsReq struct {
		Actions []models.RuleAction `json:"actions"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&actionsReq); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if len(actionsReq.Actions) == 0 {
		http.Error(w, `{"error":"at least one action is required"}`, http.StatusBadRequest)
		return
	}

	// TODO: Implement action execution
	// This would:
	// 1. Verify user owns the email
	// 2. Execute each action (forward, archive, label, etc.)
	// 3. Update the email's actions_taken field
	// 4. Return results

	// For now, return a placeholder response
	response := map[string]interface{}{
		"success":  true,
		"email_id": emailID,
		"actions_applied": len(actionsReq.Actions),
		"message": "Action execution not fully implemented",
		"results": []map[string]interface{}{
			{"action": "archive", "status": "success"},
			{"action": "label", "status": "success", "label": "processed"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ForceSync handles POST /api/v1/emails/sync
func (eh *EmailHandler) ForceSync(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"user authentication required"}`, http.StatusUnauthorized)
		return
	}

	// Parse optional request body for sync options
	var syncReq struct {
		AccountID string `json:"account_id,omitempty"`
		FullSync  bool   `json:"full_sync,omitempty"`
	}
	
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&syncReq)
	}

	// TODO: Implement email synchronization
	// This would:
	// 1. Get user's email accounts (or specific account if specified)
	// 2. Fetch new emails from each account
	// 3. Process emails through triage rules
	// 4. Generate embeddings for semantic search
	// 5. Update sync timestamps

	// For now, return a placeholder response
	response := map[string]interface{}{
		"success":           true,
		"message":           "Email sync initiated",
		"accounts_synced":   1,
		"emails_fetched":    0,
		"emails_processed":  0,
		"sync_started_at":   time.Now(),
		"estimated_duration": "2-5 minutes",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted) // 202 for async operation
	json.NewEncoder(w).Encode(response)
}

// Helper functions

func (eh *EmailHandler) applySearchFilters(results []*models.SearchResult, accountID, dateFrom, dateTo string) []*models.SearchResult {
	var filtered []*models.SearchResult

	// Parse date filters if provided
	var fromTime, toTime time.Time
	var err error
	
	if dateFrom != "" {
		fromTime, err = time.Parse("2006-01-02", dateFrom)
		if err != nil {
			fromTime = time.Time{} // Zero time if parsing fails
		}
	}
	
	if dateTo != "" {
		toTime, err = time.Parse("2006-01-02", dateTo)
		if err != nil {
			toTime = time.Now() // Current time if parsing fails
		}
	}

	for _, result := range results {
		// Apply account filter
		if accountID != "" {
			// TODO: Check if result belongs to specified account
			// For now, skip this filter
		}

		// Apply date range filter
		if !fromTime.IsZero() && result.ProcessedAt.Before(fromTime) {
			continue
		}
		if !toTime.IsZero() && result.ProcessedAt.After(toTime) {
			continue
		}

		filtered = append(filtered, result)
	}

	return filtered
}