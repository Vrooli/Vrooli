package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"scenario-authenticator/auth"
	"scenario-authenticator/db"
	"scenario-authenticator/models"
	"scenario-authenticator/utils"
	"time"

	"github.com/go-chi/chi/v5"
)

// APIKey represents an API key in the database
type APIKey struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	Key         string     `json:"key,omitempty"` // Only returned on creation
	Permissions []string   `json:"permissions"`
	RateLimit   int        `json:"rate_limit"`
	LastUsed    *time.Time `json:"last_used,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// GenerateAPIKey generates a new secure API key
func GenerateAPIKey() (string, error) {
	// Generate 32 bytes of random data
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Encode as base64 URL-safe string
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// CreateAPIKeyHandler creates a new API key for the authenticated user
func CreateAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*models.Claims)

	var req struct {
		Name        string   `json:"name"`
		Permissions []string `json:"permissions,omitempty"`
		RateLimit   int      `json:"rate_limit,omitempty"`
		ExpiresIn   int      `json:"expires_in,omitempty"` // Days until expiration
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		utils.SendError(w, "API key name is required", http.StatusBadRequest)
		return
	}

	// Generate the API key
	apiKey, err := GenerateAPIKey()
	if err != nil {
		utils.SendError(w, "Failed to generate API key", http.StatusInternalServerError)
		return
	}

	// Hash the key for storage
	keyHash := auth.HashToken(apiKey)

	// Set default rate limit if not specified
	if req.RateLimit == 0 {
		req.RateLimit = 1000
	}

	// Calculate expiration date if specified
	var expiresAt *time.Time
	if req.ExpiresIn > 0 {
		exp := time.Now().AddDate(0, 0, req.ExpiresIn)
		expiresAt = &exp
	}

	// Set default permissions if not specified
	if len(req.Permissions) == 0 {
		req.Permissions = []string{"read:own", "write:own"}
	}

	// Convert permissions to JSON
	permissionsJSON, err := json.Marshal(req.Permissions)
	if err != nil {
		utils.SendError(w, "Invalid permissions format", http.StatusBadRequest)
		return
	}

	// Store in database
	var id string
	err = db.DB.QueryRow(`
		INSERT INTO api_keys (user_id, key_hash, name, permissions, rate_limit, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		claims.UserID, keyHash, req.Name, permissionsJSON, req.RateLimit, expiresAt,
	).Scan(&id)

	if err != nil {
		utils.SendError(w, "Failed to create API key", http.StatusInternalServerError)
		return
	}

	// Log the creation event
	logAuthEvent(claims.UserID, "api_key.created", auth.GetClientIP(r), r.Header.Get("User-Agent"), true,
		map[string]interface{}{"key_id": id, "name": req.Name})

	// Return the key (only time it's visible)
	response := APIKey{
		ID:          id,
		UserID:      claims.UserID,
		Name:        req.Name,
		Key:         fmt.Sprintf("sk_%s", apiKey), // Prefix for easy identification
		Permissions: req.Permissions,
		RateLimit:   req.RateLimit,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
	}

	utils.SendJSON(w, response, http.StatusCreated)
}

// ListAPIKeysHandler lists all API keys for the authenticated user
func ListAPIKeysHandler(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*models.Claims)

	rows, err := db.DB.Query(`
		SELECT id, name, permissions, rate_limit, last_used, expires_at, created_at
		FROM api_keys
		WHERE user_id = $1 AND revoked_at IS NULL
		ORDER BY created_at DESC`,
		claims.UserID,
	)
	if err != nil {
		utils.SendError(w, "Failed to fetch API keys", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var key APIKey
		var permissionsJSON string
		var lastUsed, expiresAt sql.NullTime

		err := rows.Scan(&key.ID, &key.Name, &permissionsJSON, &key.RateLimit,
			&lastUsed, &expiresAt, &key.CreatedAt)
		if err != nil {
			continue
		}

		// Parse permissions
		json.Unmarshal([]byte(permissionsJSON), &key.Permissions)

		// Handle nullable fields
		if lastUsed.Valid {
			key.LastUsed = &lastUsed.Time
		}
		if expiresAt.Valid {
			key.ExpiresAt = &expiresAt.Time
		}

		key.UserID = claims.UserID
		keys = append(keys, key)
	}

	utils.SendJSON(w, keys, http.StatusOK)
}

// RevokeAPIKeyHandler revokes an API key
func RevokeAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*models.Claims)
	keyID := chi.URLParam(r, "id")

	if keyID == "" {
		utils.SendError(w, "API key ID is required", http.StatusBadRequest)
		return
	}

	// Verify ownership and revoke
	result, err := db.DB.Exec(`
		UPDATE api_keys
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`,
		keyID, claims.UserID,
	)

	if err != nil {
		utils.SendError(w, "Failed to revoke API key", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		utils.SendError(w, "API key not found or already revoked", http.StatusNotFound)
		return
	}

	// Log the revocation event
	logAuthEvent(claims.UserID, "api_key.revoked", auth.GetClientIP(r), r.Header.Get("User-Agent"), true,
		map[string]interface{}{"key_id": keyID})

	utils.SendJSON(w, map[string]interface{}{
		"success": true,
		"message": "API key revoked successfully",
	}, http.StatusOK)
}

// ValidateAPIKeyHandler validates an API key (for external services)
func ValidateAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		APIKey string `json:"api_key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.APIKey == "" {
		utils.SendError(w, "API key is required", http.StatusBadRequest)
		return
	}

	// Remove prefix if present
	apiKey := req.APIKey
	if len(apiKey) > 3 && apiKey[:3] == "sk_" {
		apiKey = apiKey[3:]
	}

	// Hash the key to compare with stored hash
	keyHash := auth.HashToken(apiKey)

	// Look up the key in database
	var userID string
	var permissions string
	var rateLimit int
	var expiresAt sql.NullTime

	err := db.DB.QueryRow(`
		SELECT user_id, permissions, rate_limit, expires_at
		FROM api_keys
		WHERE key_hash = $1 AND revoked_at IS NULL`,
		keyHash,
	).Scan(&userID, &permissions, &rateLimit, &expiresAt)

	if err == sql.ErrNoRows {
		utils.SendJSON(w, map[string]interface{}{
			"valid": false,
			"error": "Invalid API key",
		}, http.StatusUnauthorized)
		return
	}

	if err != nil {
		utils.SendError(w, "Failed to validate API key", http.StatusInternalServerError)
		return
	}

	// Check if expired
	if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {
		utils.SendJSON(w, map[string]interface{}{
			"valid": false,
			"error": "API key expired",
		}, http.StatusUnauthorized)
		return
	}

	// Update last used timestamp
	go func() {
		db.DB.Exec("UPDATE api_keys SET last_used = CURRENT_TIMESTAMP WHERE key_hash = $1", keyHash)
	}()

	// Parse permissions
	var perms []string
	json.Unmarshal([]byte(permissions), &perms)

	utils.SendJSON(w, map[string]interface{}{
		"valid":       true,
		"user_id":     userID,
		"permissions": perms,
		"rate_limit":  rateLimit,
	}, http.StatusOK)
}
