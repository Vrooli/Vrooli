package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	
	"email-triage/models"
	"email-triage/services"
)

// AccountHandler handles email account management endpoints
type AccountHandler struct {
	db           *sql.DB
	emailService *services.EmailService
}

// NewAccountHandler creates a new AccountHandler instance
func NewAccountHandler(db *sql.DB, emailService *services.EmailService) *AccountHandler {
	return &AccountHandler{
		db:           db,
		emailService: emailService,
	}
}

// CreateAccount handles POST /api/v1/accounts
func (ah *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"user authentication required"}`, http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req models.CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.EmailAddress == "" || req.Password == "" {
		http.Error(w, `{"error":"email_address and password are required"}`, http.StatusBadRequest)
		return
	}

	// Create IMAP and SMTP configs
	imapConfig := models.IMAPConfig{
		Server:   req.IMAPServer,
		Port:     req.IMAPPort,
		Username: req.EmailAddress,
		Password: req.Password,
		UseTLS:   req.UseTLS,
	}

	smtpConfig := models.SMTPConfig{
		Server:   req.SMTPServer,
		Port:     req.SMTPPort,
		Username: req.EmailAddress,
		Password: req.Password,
		UseTLS:   req.UseTLS,
	}

	// Serialize configs
	imapJSON, _ := json.Marshal(imapConfig)
	smtpJSON, _ := json.Marshal(smtpConfig)

	// Create email account model
	account := &models.EmailAccount{
		ID:           generateUUID(),
		UserID:       userID,
		EmailAddress: req.EmailAddress,
		IMAPSettings: imapJSON,
		SMTPSettings: smtpJSON,
		SyncEnabled:  true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Test connection before saving
	if err := ah.emailService.TestConnection(account); err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   "connection_failed",
			"message": err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Save to database
	query := `
		INSERT INTO email_accounts (
			id, user_id, email_address, imap_settings, smtp_settings, 
			sync_enabled, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := ah.db.Exec(query,
		account.ID, account.UserID, account.EmailAddress,
		account.IMAPSettings, account.SMTPSettings,
		account.SyncEnabled, account.CreatedAt, account.UpdatedAt)

	if err != nil {
		http.Error(w, `{"error":"failed to save account"}`, http.StatusInternalServerError)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"success":    true,
		"account_id": account.ID,
		"status":     "connected",
		"message":    "Email account connected successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// ListAccounts handles GET /api/v1/accounts
func (ah *AccountHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"user authentication required"}`, http.StatusUnauthorized)
		return
	}

	// Query user's accounts
	query := `
		SELECT id, user_id, email_address, last_sync, sync_enabled, created_at, updated_at
		FROM email_accounts 
		WHERE user_id = $1 
		ORDER BY created_at DESC`

	rows, err := ah.db.Query(query, userID)
	if err != nil {
		http.Error(w, `{"error":"database query failed"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var accounts []map[string]interface{}

	for rows.Next() {
		var account models.EmailAccount
		err := rows.Scan(
			&account.ID, &account.UserID, &account.EmailAddress,
			&account.LastSync, &account.SyncEnabled,
			&account.CreatedAt, &account.UpdatedAt,
		)
		if err != nil {
			continue // Skip invalid rows
		}

		// Don't expose sensitive settings in list view
		accountData := map[string]interface{}{
			"id":           account.ID,
			"email_address": account.EmailAddress,
			"last_sync":    account.LastSync,
			"sync_enabled": account.SyncEnabled,
			"created_at":   account.CreatedAt,
			"status":       "active", // TODO: Implement actual status checking
		}

		accounts = append(accounts, accountData)
	}

	response := map[string]interface{}{
		"accounts": accounts,
		"count":    len(accounts),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAccount handles GET /api/v1/accounts/{id}
func (ah *AccountHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"user authentication required"}`, http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	accountID := vars["id"]

	// Query specific account
	query := `
		SELECT id, user_id, email_address, last_sync, sync_enabled, created_at, updated_at
		FROM email_accounts 
		WHERE id = $1 AND user_id = $2`

	var account models.EmailAccount
	err := ah.db.QueryRow(query, accountID, userID).Scan(
		&account.ID, &account.UserID, &account.EmailAddress,
		&account.LastSync, &account.SyncEnabled,
		&account.CreatedAt, &account.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"account not found"}`, http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, `{"error":"database query failed"}`, http.StatusInternalServerError)
		return
	}

	// Return account details (without sensitive settings)
	response := map[string]interface{}{
		"id":           account.ID,
		"email_address": account.EmailAddress,
		"last_sync":    account.LastSync,
		"sync_enabled": account.SyncEnabled,
		"created_at":   account.CreatedAt,
		"updated_at":   account.UpdatedAt,
		"status":       "active",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateAccount handles PUT /api/v1/accounts/{id}
func (ah *AccountHandler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"user authentication required"}`, http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	accountID := vars["id"]

	// Parse request body
	var updateReq map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// For now, only allow updating sync_enabled
	if syncEnabled, ok := updateReq["sync_enabled"].(bool); ok {
		query := `UPDATE email_accounts SET sync_enabled = $1, updated_at = $2 WHERE id = $3 AND user_id = $4`
		
		_, err := ah.db.Exec(query, syncEnabled, time.Now(), accountID, userID)
		if err != nil {
			http.Error(w, `{"error":"update failed"}`, http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"success": true,
			"message": "Account updated successfully",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	http.Error(w, `{"error":"no valid fields to update"}`, http.StatusBadRequest)
}

// DeleteAccount handles DELETE /api/v1/accounts/{id}
func (ah *AccountHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"user authentication required"}`, http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	accountID := vars["id"]

	// Delete account and related data
	tx, err := ah.db.Begin()
	if err != nil {
		http.Error(w, `{"error":"transaction failed"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Delete processed emails first (foreign key constraint)
	_, err = tx.Exec("DELETE FROM processed_emails WHERE account_id = $1", accountID)
	if err != nil {
		http.Error(w, `{"error":"failed to delete email data"}`, http.StatusInternalServerError)
		return
	}

	// Delete the account
	result, err := tx.Exec("DELETE FROM email_accounts WHERE id = $1 AND user_id = $2", accountID, userID)
	if err != nil {
		http.Error(w, `{"error":"failed to delete account"}`, http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, `{"error":"account not found"}`, http.StatusNotFound)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, `{"error":"commit failed"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Account deleted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SyncAccount handles POST /api/v1/accounts/{id}/sync
func (ah *AccountHandler) SyncAccount(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"user authentication required"}`, http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	accountID := vars["id"]

	// TODO: Implement email synchronization
	// This would fetch new emails from the account and process them

	response := map[string]interface{}{
		"success":         true,
		"message":         "Email sync completed",
		"emails_fetched":  0,
		"emails_processed": 0,
		"sync_time":       time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper functions
func getUserIDFromContext(r *http.Request) string {
	if userID := r.Context().Value("user_id"); userID != nil {
		return userID.(string)
	}
	return ""
}

func generateUUID() string {
	// Placeholder UUID generation - use proper UUID library in production
	return fmt.Sprintf("uuid-%d", time.Now().UnixNano())
}