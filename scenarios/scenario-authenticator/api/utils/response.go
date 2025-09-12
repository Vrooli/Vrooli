package utils

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"scenario-authenticator/models"
	"time"
)

// SendError sends an error response
func SendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(models.ErrorResponse{
		Error: message,
	})
}

// SendJSON sends a JSON response
func SendJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// SendValidationResponse sends token validation response
func SendValidationResponse(w http.ResponseWriter, valid bool, claims *models.Claims) {
	response := models.ValidationResponse{
		Valid: valid,
	}
	
	if valid && claims != nil {
		response.UserID = claims.UserID
		response.Email = claims.Email
		response.Roles = claims.Roles
		response.ExpiresAt = time.Unix(claims.ExpiresAt, 0)
	}
	
	SendJSON(w, response, http.StatusOK)
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to time-based token if crypto/rand fails
		return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))[:length]
	}
	return hex.EncodeToString(bytes)[:length]
}