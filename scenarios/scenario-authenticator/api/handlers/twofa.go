package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"scenario-authenticator/db"
	"scenario-authenticator/models"
)

// TOTPConfig represents TOTP configuration
type TOTPConfig struct {
	Secret      string   `json:"secret"`
	QRCodeURL   string   `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes"`
}

// Enable2FARequest represents request to enable 2FA
type Enable2FARequest struct {
	Code string `json:"code"` // Verification code from authenticator app
}

// Verify2FARequest represents request to verify 2FA code during login
type Verify2FARequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
	Token string `json:"token"` // Temporary token from login
}

// generateTOTPSecret generates a random TOTP secret
func generateTOTPSecret() (string, error) {
	// Generate 20 random bytes (160 bits for high security)
	randomBytes := make([]byte, 20)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	// Encode to base32 (required for TOTP)
	secret := base32.StdEncoding.EncodeToString(randomBytes)
	return strings.TrimRight(secret, "="), nil // Remove padding
}

// generateBackupCodes generates backup codes for 2FA recovery
func generateBackupCodes(count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		// Generate 8 random bytes
		randomBytes := make([]byte, 8)
		if _, err := rand.Read(randomBytes); err != nil {
			return nil, err
		}

		// Convert to readable format (16 hex characters)
		codes[i] = fmt.Sprintf("%x", randomBytes)
	}
	return codes, nil
}

// verifyTOTPCode verifies a TOTP code against a secret
// Implements RFC 6238 TOTP algorithm
func verifyTOTPCode(secret, code string) bool {
	// Get current Unix timestamp
	now := time.Now().Unix()

	// TOTP uses 30-second time steps
	timeStep := int64(30)

	// Check current time window and ±1 window (90 seconds total) to account for clock drift
	for i := int64(-1); i <= 1; i++ {
		counter := (now / timeStep) + i
		expectedCode := generateTOTPCode(secret, counter)

		if code == expectedCode {
			return true
		}
	}

	return false
}

// generateTOTPCode generates a 6-digit TOTP code for a given counter
func generateTOTPCode(secret string, counter int64) string {
	// Decode base32 secret
	key, err := base32.StdEncoding.DecodeString(secret + strings.Repeat("=", (8-len(secret)%8)%8))
	if err != nil {
		return ""
	}

	// Convert counter to byte array (big-endian)
	counterBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(counterBytes, uint64(counter))

	// HMAC-SHA1
	mac := hmac.New(sha1.New, key)
	mac.Write(counterBytes)
	hash := mac.Sum(nil)

	// Dynamic truncation (RFC 4226)
	offset := hash[len(hash)-1] & 0x0f
	truncatedHash := hash[offset : offset+4]

	// Convert to integer
	code := int(truncatedHash[0]&0x7f)<<24 |
		int(truncatedHash[1])<<16 |
		int(truncatedHash[2])<<8 |
		int(truncatedHash[3])

	// Generate 6-digit code
	code = code % 1000000

	return fmt.Sprintf("%06d", code)
}

// Setup2FAHandler initiates 2FA setup for a user
func Setup2FAHandler(w http.ResponseWriter, r *http.Request) {
	// Get claims from context (set by auth middleware)
	claims, ok := r.Context().Value("claims").(*models.Claims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := claims.UserID

	// Check if 2FA is already enabled
	// Use db.DB directly
	var twoFAEnabled bool
	err := db.DB.QueryRow("SELECT two_factor_enabled FROM users WHERE id = $1", userID).Scan(&twoFAEnabled)
	if err != nil {
		http.Error(w, "Failed to check 2FA status", http.StatusInternalServerError)
		return
	}

	if twoFAEnabled {
		http.Error(w, "2FA is already enabled", http.StatusBadRequest)
		return
	}

	// Generate TOTP secret
	secret, err := generateTOTPSecret()
	if err != nil {
		http.Error(w, "Failed to generate secret", http.StatusInternalServerError)
		return
	}

	// Generate backup codes
	backupCodes, err := generateBackupCodes(10)
	if err != nil {
		http.Error(w, "Failed to generate backup codes", http.StatusInternalServerError)
		return
	}

	// Get user email for QR code
	var email string
	err = db.DB.QueryRow("SELECT email FROM users WHERE id = $1", userID).Scan(&email)
	if err != nil {
		http.Error(w, "Failed to get user email", http.StatusInternalServerError)
		return
	}

	// Store secret temporarily (will be activated upon verification)
	_, err = db.DB.Exec(
		"UPDATE users SET two_factor_secret = $1, metadata = metadata || $2 WHERE id = $3",
		secret,
		fmt.Sprintf(`{"backup_codes": %s}`, toJSON(backupCodes)),
		userID,
	)
	if err != nil {
		http.Error(w, "Failed to store 2FA setup", http.StatusInternalServerError)
		return
	}

	// Generate QR code URL (otpauth:// format)
	qrCodeURL := fmt.Sprintf(
		"otpauth://totp/Vrooli:%s?secret=%s&issuer=Vrooli",
		email,
		secret,
	)

	response := TOTPConfig{
		Secret:      secret,
		QRCodeURL:   qrCodeURL,
		BackupCodes: backupCodes,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Enable2FAHandler verifies and enables 2FA for a user
func Enable2FAHandler(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	claims, ok := r.Context().Value("claims").(*models.Claims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := claims.UserID

	var req Enable2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get stored secret
	// Use db.DB directly
	var secret string
	err := db.DB.QueryRow("SELECT two_factor_secret FROM users WHERE id = $1", userID).Scan(&secret)
	if err != nil {
		http.Error(w, "2FA setup not initiated", http.StatusBadRequest)
		return
	}

	// Verify the code
	if !verifyTOTPCode(secret, req.Code) {
		http.Error(w, "Invalid verification code", http.StatusBadRequest)
		return
	}

	// Enable 2FA
	_, err = db.DB.Exec(
		"UPDATE users SET two_factor_enabled = TRUE WHERE id = $1",
		userID,
	)
	if err != nil {
		http.Error(w, "Failed to enable 2FA", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "2FA enabled successfully",
	})
}

// Disable2FAHandler disables 2FA for a user
func Disable2FAHandler(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	claims, ok := r.Context().Value("claims").(*models.Claims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := claims.UserID

	var req Enable2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Verify current 2FA code before disabling
	// Use db.DB directly
	var secret string
	var enabled bool
	err := db.DB.QueryRow("SELECT two_factor_secret, two_factor_enabled FROM users WHERE id = $1", userID).Scan(&secret, &enabled)
	if err != nil {
		http.Error(w, "Failed to get 2FA status", http.StatusInternalServerError)
		return
	}

	if !enabled {
		http.Error(w, "2FA is not enabled", http.StatusBadRequest)
		return
	}

	// Verify the code
	if !verifyTOTPCode(secret, req.Code) {
		http.Error(w, "Invalid verification code", http.StatusBadRequest)
		return
	}

	// Disable 2FA
	_, err = db.DB.Exec(
		"UPDATE users SET two_factor_enabled = FALSE, two_factor_secret = NULL WHERE id = $1",
		userID,
	)
	if err != nil {
		http.Error(w, "Failed to disable 2FA", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "2FA disabled successfully",
	})
}

// Verify2FAHandler verifies a 2FA code during login
func Verify2FAHandler(w http.ResponseWriter, r *http.Request) {
	var req Verify2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get user's 2FA secret
	// Use db.DB directly
	var secret string
	var enabled bool
	var userID string
	err := db.DB.QueryRow(
		"SELECT id, two_factor_secret, two_factor_enabled FROM users WHERE email = $1",
		req.Email,
	).Scan(&userID, &secret, &enabled)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if !enabled {
		http.Error(w, "2FA is not enabled for this user", http.StatusBadRequest)
		return
	}

	// Verify the code
	if !verifyTOTPCode(secret, req.Code) {
		// Check if it's a backup code
		if !checkBackupCode(userID, req.Code) {
			http.Error(w, "Invalid verification code", http.StatusBadRequest)
			return
		}
	}

	// Code is valid, return success
	// In production, this would also validate the temporary token
	// and return a full access token
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.AuthResponse{
		Success: true,
		Message: "2FA verification successful",
	})
}

// checkBackupCode checks if a backup code is valid and marks it as used
func checkBackupCode(userID, code string) bool {
	// Use db.DB directly

	// Get backup codes from metadata
	var metadataJSON string
	err := db.DB.QueryRow("SELECT metadata FROM users WHERE id = $1", userID).Scan(&metadataJSON)
	if err != nil {
		return false
	}

	// Parse metadata
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		return false
	}

	backupCodes, ok := metadata["backup_codes"].([]interface{})
	if !ok {
		return false
	}

	// Check if code exists
	for i, bc := range backupCodes {
		if bc.(string) == code {
			// Remove used backup code
			backupCodes = append(backupCodes[:i], backupCodes[i+1:]...)
			metadata["backup_codes"] = backupCodes

			// Update metadata
			newMetadataJSON := toJSON(metadata)
			_, err := db.DB.Exec("UPDATE users SET metadata = $1 WHERE id = $2", newMetadataJSON, userID)
			return err == nil
		}
	}

	return false
}

// toJSON is a helper to convert data to JSON string
func toJSON(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}
