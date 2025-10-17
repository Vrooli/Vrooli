package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// AuthService handles authentication with scenario-authenticator
type AuthService struct {
	authURL string
	client  *http.Client
}

// AuthValidationResponse represents the response from auth validation
type AuthValidationResponse struct {
	Valid     bool      `json:"valid"`
	UserID    string    `json:"user_id"`
	Roles     []string  `json:"roles"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NewAuthService creates a new AuthService instance
func NewAuthService(authURL string) *AuthService {
	return &AuthService{
		authURL: authURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// ValidateToken validates a JWT token with the scenario-authenticator service
func (as *AuthService) ValidateToken(authHeader string) (string, error) {
	// Extract token from "Bearer <token>" format
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}
	
	token := tokenParts[1]
	
	// Create request to auth service
	req, err := http.NewRequest("GET", as.authURL+"/api/v1/auth/validate", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create auth request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	
	// Send request
	resp, err := as.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("auth service request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("invalid token: status %d", resp.StatusCode)
	}
	
	// Parse response
	var authResp AuthValidationResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", fmt.Errorf("failed to parse auth response: %w", err)
	}
	
	// Validate response
	if !authResp.Valid {
		return "", fmt.Errorf("token is not valid")
	}
	
	if authResp.UserID == "" {
		return "", fmt.Errorf("no user ID in token")
	}
	
	// Check if token is expired
	if time.Now().After(authResp.ExpiresAt) {
		return "", fmt.Errorf("token is expired")
	}
	
	return authResp.UserID, nil
}

// HealthCheck checks if the auth service is available
func (as *AuthService) HealthCheck() bool {
	req, err := http.NewRequest("GET", as.authURL+"/health", nil)
	if err != nil {
		return false
	}
	
	resp, err := as.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == http.StatusOK
}