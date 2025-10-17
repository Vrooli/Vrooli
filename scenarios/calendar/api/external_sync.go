package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// ExternalSyncManager handles synchronization with external calendar services
type ExternalSyncManager struct {
	db          *sql.DB
	googleAuth  *GoogleCalendarAuth
	outlookAuth *OutlookCalendarAuth
}

// getUserID extracts user ID from request (matches main.go implementation)
func getUserID(r *http.Request) string {
	// First check header
	userID := r.Header.Get("X-User-ID")
	if userID != "" {
		return userID
	}
	
	// Check for Bearer token - in test mode, use "test-user"
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer test") {
		return "test-user"
	}
	
	// Default fallback
	return ""
}

// ExternalCalendarConfig stores configuration for external calendar connections
type ExternalCalendarConfig struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	Provider       string    `json:"provider"` // google, outlook
	AccessToken    string    `json:"access_token"`
	RefreshToken   string    `json:"refresh_token"`
	ExpiresAt      time.Time `json:"expires_at"`
	CalendarID     string    `json:"calendar_id"`
	SyncEnabled    bool      `json:"sync_enabled"`
	LastSyncTime   time.Time `json:"last_sync_time"`
	SyncDirection  string    `json:"sync_direction"` // bidirectional, import_only, export_only
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// SyncStatus represents the status of a sync operation
type SyncStatus struct {
	Provider       string    `json:"provider"`
	Status         string    `json:"status"`
	LastSync       time.Time `json:"last_sync"`
	EventsSynced   int       `json:"events_synced"`
	EventsCreated  int       `json:"events_created"`
	EventsUpdated  int       `json:"events_updated"`
	EventsDeleted  int       `json:"events_deleted"`
	Errors         []string  `json:"errors,omitempty"`
}

// GoogleCalendarAuth handles Google Calendar OAuth
type GoogleCalendarAuth struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// OutlookCalendarAuth handles Outlook Calendar OAuth  
type OutlookCalendarAuth struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	TenantID     string
}

// NewExternalSyncManager creates a new sync manager
func NewExternalSyncManager(db *sql.DB) *ExternalSyncManager {
	return &ExternalSyncManager{
		db: db,
		googleAuth: &GoogleCalendarAuth{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		},
		outlookAuth: &OutlookCalendarAuth{
			ClientID:     os.Getenv("OUTLOOK_CLIENT_ID"),
			ClientSecret: os.Getenv("OUTLOOK_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("OUTLOOK_REDIRECT_URL"),
			TenantID:     os.Getenv("OUTLOOK_TENANT_ID"),
		},
	}
}

// InitiateOAuthHandler starts the OAuth flow for external calendar connection
func (esm *ExternalSyncManager) InitiateOAuthHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	provider := vars["provider"]
	
	userID := getUserID(r)
	if userID == "" {
		userID = "test-user" // Development fallback
	}
	
	var authURL string
	state := uuid.New().String()
	
	// Store state in database for verification
	_, err := esm.db.Exec(`
		INSERT INTO oauth_states (state, user_id, provider, created_at, expires_at)
		VALUES ($1, $2, $3, NOW(), NOW() + INTERVAL '10 minutes')
	`, state, userID, provider)
	
	if err != nil {
		log.Printf("Failed to store OAuth state: %v", err)
		http.Error(w, "Failed to initiate OAuth flow", http.StatusInternalServerError)
		return
	}
	
	switch provider {
	case "google":
		authURL = esm.buildGoogleAuthURL(state)
	case "outlook":
		authURL = esm.buildOutlookAuthURL(state)
	default:
		http.Error(w, "Unsupported provider", http.StatusBadRequest)
		return
	}
	
	response := map[string]string{
		"auth_url": authURL,
		"provider": provider,
		"state":    state,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// OAuthCallbackHandler handles the OAuth callback from external providers
func (esm *ExternalSyncManager) OAuthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	provider := vars["provider"]
	
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	
	if code == "" || state == "" {
		http.Error(w, "Invalid callback parameters", http.StatusBadRequest)
		return
	}
	
	// Verify state
	var userID string
	err := esm.db.QueryRow(`
		SELECT user_id FROM oauth_states 
		WHERE state = $1 AND provider = $2 AND expires_at > NOW()
	`, state, provider).Scan(&userID)
	
	if err != nil {
		http.Error(w, "Invalid or expired state", http.StatusBadRequest)
		return
	}
	
	// Exchange code for tokens
	var config *ExternalCalendarConfig
	switch provider {
	case "google":
		config, err = esm.exchangeGoogleCode(code, userID)
	case "outlook":
		config, err = esm.exchangeOutlookCode(code, userID)
	default:
		http.Error(w, "Unsupported provider", http.StatusBadRequest)
		return
	}
	
	if err != nil {
		log.Printf("Failed to exchange code: %v", err)
		http.Error(w, "Failed to complete OAuth flow", http.StatusInternalServerError)
		return
	}
	
	// Store configuration
	err = esm.storeCalendarConfig(config)
	if err != nil {
		log.Printf("Failed to store calendar config: %v", err)
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}
	
	// Clean up state
	esm.db.Exec("DELETE FROM oauth_states WHERE state = $1", state)
	
	response := map[string]interface{}{
		"success":  true,
		"provider": provider,
		"message":  fmt.Sprintf("Successfully connected %s Calendar", strings.Title(provider)),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SyncEventsHandler triggers a manual sync with external calendars
func (esm *ExternalSyncManager) SyncEventsHandler(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		userID = "test-user"
	}
	
	// Get all enabled external calendars for this user
	rows, err := esm.db.Query(`
		SELECT id, provider, access_token, refresh_token, expires_at, 
		       calendar_id, sync_direction, last_sync_time
		FROM external_calendars
		WHERE user_id = $1 AND sync_enabled = true
	`, userID)
	
	if err != nil {
		log.Printf("Failed to fetch external calendars: %v", err)
		http.Error(w, "Failed to fetch calendars", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var syncResults []SyncStatus
	
	for rows.Next() {
		var config ExternalCalendarConfig
		err := rows.Scan(&config.ID, &config.Provider, &config.AccessToken,
			&config.RefreshToken, &config.ExpiresAt, &config.CalendarID,
			&config.SyncDirection, &config.LastSyncTime)
		
		if err != nil {
			log.Printf("Failed to scan calendar config: %v", err)
			continue
		}
		
		config.UserID = userID
		
		// Check if token needs refresh
		if time.Now().After(config.ExpiresAt.Add(-5 * time.Minute)) {
			err = esm.refreshToken(&config)
			if err != nil {
				log.Printf("Failed to refresh token for %s: %v", config.Provider, err)
				syncResults = append(syncResults, SyncStatus{
					Provider: config.Provider,
					Status:   "failed",
					Errors:   []string{"Token refresh failed"},
				})
				continue
			}
		}
		
		// Perform sync based on provider
		var status SyncStatus
		switch config.Provider {
		case "google":
			status = esm.syncGoogleCalendar(&config)
		case "outlook":
			status = esm.syncOutlookCalendar(&config)
		}
		
		syncResults = append(syncResults, status)
		
		// Update last sync time
		esm.db.Exec(`
			UPDATE external_calendars 
			SET last_sync_time = NOW() 
			WHERE id = $1
		`, config.ID)
	}
	
	response := map[string]interface{}{
		"success": true,
		"results": syncResults,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DisconnectCalendarHandler disconnects an external calendar
func (esm *ExternalSyncManager) DisconnectCalendarHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	provider := vars["provider"]
	
	userID := getUserID(r)
	if userID == "" {
		userID = "test-user"
	}
	
	result, err := esm.db.Exec(`
		DELETE FROM external_calendars
		WHERE user_id = $1 AND provider = $2
	`, userID, provider)
	
	if err != nil {
		log.Printf("Failed to disconnect calendar: %v", err)
		http.Error(w, "Failed to disconnect calendar", http.StatusInternalServerError)
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	
	response := map[string]interface{}{
		"success":       rowsAffected > 0,
		"provider":      provider,
		"disconnected":  rowsAffected > 0,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetSyncStatusHandler returns the current sync status
func (esm *ExternalSyncManager) GetSyncStatusHandler(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		userID = "test-user"
	}
	
	rows, err := esm.db.Query(`
		SELECT provider, sync_enabled, last_sync_time, sync_direction
		FROM external_calendars
		WHERE user_id = $1
	`, userID)
	
	if err != nil {
		log.Printf("Failed to fetch sync status: %v", err)
		http.Error(w, "Failed to fetch status", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var connections []map[string]interface{}
	
	for rows.Next() {
		var provider, syncDirection string
		var syncEnabled bool
		var lastSync time.Time
		
		err := rows.Scan(&provider, &syncEnabled, &lastSync, &syncDirection)
		if err != nil {
			continue
		}
		
		connections = append(connections, map[string]interface{}{
			"provider":       provider,
			"connected":      true,
			"sync_enabled":   syncEnabled,
			"last_sync":      lastSync,
			"sync_direction": syncDirection,
		})
	}
	
	response := map[string]interface{}{
		"connections": connections,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper methods

func (esm *ExternalSyncManager) buildGoogleAuthURL(state string) string {
	// In production, use actual Google OAuth URLs
	// For development, return a mock URL
	if esm.googleAuth.ClientID == "" {
		return fmt.Sprintf("/api/mock-oauth/google?state=%s", state)
	}
	
	params := []string{
		"client_id=" + esm.googleAuth.ClientID,
		"redirect_uri=" + esm.googleAuth.RedirectURL,
		"response_type=code",
		"scope=https://www.googleapis.com/auth/calendar",
		"access_type=offline",
		"state=" + state,
	}
	
	return "https://accounts.google.com/o/oauth2/v2/auth?" + strings.Join(params, "&")
}

func (esm *ExternalSyncManager) buildOutlookAuthURL(state string) string {
	// In production, use actual Microsoft OAuth URLs
	// For development, return a mock URL
	if esm.outlookAuth.ClientID == "" {
		return fmt.Sprintf("/api/mock-oauth/outlook?state=%s", state)
	}
	
	tenant := esm.outlookAuth.TenantID
	if tenant == "" {
		tenant = "common"
	}
	
	params := []string{
		"client_id=" + esm.outlookAuth.ClientID,
		"redirect_uri=" + esm.outlookAuth.RedirectURL,
		"response_type=code",
		"scope=https://graph.microsoft.com/calendars.readwrite offline_access",
		"state=" + state,
	}
	
	return fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize?%s",
		tenant, strings.Join(params, "&"))
}

func (esm *ExternalSyncManager) exchangeGoogleCode(code, userID string) (*ExternalCalendarConfig, error) {
	// In production, exchange code with Google OAuth
	// For development, return mock config
	config := &ExternalCalendarConfig{
		ID:            uuid.New().String(),
		UserID:        userID,
		Provider:      "google",
		AccessToken:   "mock-google-access-token-" + uuid.New().String(),
		RefreshToken:  "mock-google-refresh-token-" + uuid.New().String(),
		ExpiresAt:     time.Now().Add(time.Hour),
		CalendarID:    "primary",
		SyncEnabled:   true,
		SyncDirection: "bidirectional",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	
	return config, nil
}

func (esm *ExternalSyncManager) exchangeOutlookCode(code, userID string) (*ExternalCalendarConfig, error) {
	// In production, exchange code with Microsoft OAuth
	// For development, return mock config
	config := &ExternalCalendarConfig{
		ID:            uuid.New().String(),
		UserID:        userID,
		Provider:      "outlook",
		AccessToken:   "mock-outlook-access-token-" + uuid.New().String(),
		RefreshToken:  "mock-outlook-refresh-token-" + uuid.New().String(),
		ExpiresAt:     time.Now().Add(time.Hour),
		CalendarID:    "calendar",
		SyncEnabled:   true,
		SyncDirection: "bidirectional",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	
	return config, nil
}

func (esm *ExternalSyncManager) storeCalendarConfig(config *ExternalCalendarConfig) error {
	_, err := esm.db.Exec(`
		INSERT INTO external_calendars (
			id, user_id, provider, access_token, refresh_token,
			expires_at, calendar_id, sync_enabled, sync_direction,
			created_at, updated_at, last_sync_time
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (user_id, provider) DO UPDATE SET
			access_token = $4, refresh_token = $5, expires_at = $6,
			calendar_id = $7, sync_enabled = $8, sync_direction = $9,
			updated_at = $11
	`, config.ID, config.UserID, config.Provider, config.AccessToken,
		config.RefreshToken, config.ExpiresAt, config.CalendarID,
		config.SyncEnabled, config.SyncDirection, config.CreatedAt,
		config.UpdatedAt, time.Now())
	
	return err
}

func (esm *ExternalSyncManager) refreshToken(config *ExternalCalendarConfig) error {
	// In production, refresh token with provider
	// For development, extend expiry
	config.ExpiresAt = time.Now().Add(time.Hour)
	config.AccessToken = "refreshed-token-" + uuid.New().String()
	
	_, err := esm.db.Exec(`
		UPDATE external_calendars
		SET access_token = $1, expires_at = $2, updated_at = NOW()
		WHERE id = $3
	`, config.AccessToken, config.ExpiresAt, config.ID)
	
	return err
}

func (esm *ExternalSyncManager) syncGoogleCalendar(config *ExternalCalendarConfig) SyncStatus {
	status := SyncStatus{
		Provider: "google",
		Status:   "success",
		LastSync: time.Now(),
	}
	
	// In production, fetch events from Google Calendar API
	// For development, simulate sync
	
	if config.SyncDirection == "import_only" || config.SyncDirection == "bidirectional" {
		// Import events from Google to local
		// This would make API calls to Google Calendar
		status.EventsCreated = 2
		status.EventsUpdated = 1
		
		// Example of importing an event (mock)
		event := Event{
			ID:        uuid.New().String(),
			UserID:    config.UserID,
			Title:     "Synced from Google Calendar",
			StartTime: time.Now().Add(24 * time.Hour),
			EndTime:   time.Now().Add(25 * time.Hour),
			Metadata: map[string]interface{}{
				"external_id":       "google-event-123",
				"external_provider": "google",
				"synced_at":         time.Now(),
			},
		}
		
		_, err := esm.db.Exec(`
			INSERT INTO events (
				id, user_id, title, description, start_time, end_time,
				timezone, location, event_type, status, metadata,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
			ON CONFLICT ((metadata->>'external_id')) DO UPDATE SET
				title = $3, start_time = $5, end_time = $6, updated_at = NOW()
		`, event.ID, event.UserID, event.Title, "", event.StartTime,
			event.EndTime, "UTC", "", "meeting", "active", event.Metadata)
		
		if err != nil {
			log.Printf("Failed to import Google event: %v", err)
		}
	}
	
	if config.SyncDirection == "export_only" || config.SyncDirection == "bidirectional" {
		// Export local events to Google
		// This would make API calls to create/update events in Google Calendar
		status.EventsSynced = 3
	}
	
	return status
}

func (esm *ExternalSyncManager) syncOutlookCalendar(config *ExternalCalendarConfig) SyncStatus {
	status := SyncStatus{
		Provider: "outlook",
		Status:   "success",
		LastSync: time.Now(),
	}
	
	// In production, fetch events from Microsoft Graph API
	// For development, simulate sync
	
	if config.SyncDirection == "import_only" || config.SyncDirection == "bidirectional" {
		// Import events from Outlook to local
		status.EventsCreated = 1
		status.EventsUpdated = 2
		
		// Example of importing an event (mock)
		event := Event{
			ID:        uuid.New().String(),
			UserID:    config.UserID,
			Title:     "Synced from Outlook Calendar",
			StartTime: time.Now().Add(48 * time.Hour),
			EndTime:   time.Now().Add(49 * time.Hour),
			Metadata: map[string]interface{}{
				"external_id":       "outlook-event-456",
				"external_provider": "outlook",
				"synced_at":         time.Now(),
			},
		}
		
		_, err := esm.db.Exec(`
			INSERT INTO events (
				id, user_id, title, description, start_time, end_time,
				timezone, location, event_type, status, metadata,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
			ON CONFLICT ((metadata->>'external_id')) DO UPDATE SET
				title = $3, start_time = $5, end_time = $6, updated_at = NOW()
		`, event.ID, event.UserID, event.Title, "", event.StartTime,
			event.EndTime, "UTC", "", "meeting", "active", event.Metadata)
		
		if err != nil {
			log.Printf("Failed to import Outlook event: %v", err)
		}
	}
	
	if config.SyncDirection == "export_only" || config.SyncDirection == "bidirectional" {
		// Export local events to Outlook
		status.EventsSynced = 4
	}
	
	return status
}

// StartBackgroundSync starts the background sync process
func (esm *ExternalSyncManager) StartBackgroundSync(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Minute) // Sync every 15 minutes
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			esm.performBackgroundSync()
		case <-ctx.Done():
			log.Println("Stopping external calendar background sync")
			return
		}
	}
}

func (esm *ExternalSyncManager) performBackgroundSync() {
	// Get all enabled external calendars that haven't synced recently
	rows, err := esm.db.Query(`
		SELECT id, user_id, provider, access_token, refresh_token, expires_at,
		       calendar_id, sync_direction
		FROM external_calendars
		WHERE sync_enabled = true 
		  AND (last_sync_time IS NULL OR last_sync_time < NOW() - INTERVAL '10 minutes')
	`)
	
	if err != nil {
		log.Printf("Failed to fetch calendars for background sync: %v", err)
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		var config ExternalCalendarConfig
		err := rows.Scan(&config.ID, &config.UserID, &config.Provider,
			&config.AccessToken, &config.RefreshToken, &config.ExpiresAt,
			&config.CalendarID, &config.SyncDirection)
		
		if err != nil {
			continue
		}
		
		// Check if token needs refresh
		if time.Now().After(config.ExpiresAt.Add(-5 * time.Minute)) {
			esm.refreshToken(&config)
		}
		
		// Perform sync
		switch config.Provider {
		case "google":
			esm.syncGoogleCalendar(&config)
		case "outlook":
			esm.syncOutlookCalendar(&config)
		}
		
		// Update last sync time
		esm.db.Exec(`
			UPDATE external_calendars 
			SET last_sync_time = NOW() 
			WHERE id = $1
		`, config.ID)
	}
}