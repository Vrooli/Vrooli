package models

import (
	"time"
)

// User represents a user account
type User struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	Username      string     `json:"username,omitempty"`
	PasswordHash  string     `json:"-"`
	Roles         []string   `json:"roles"`
	EmailVerified bool       `json:"email_verified"`
	CreatedAt     time.Time  `json:"created_at"`
	LastLogin     *time.Time `json:"last_login,omitempty"`
}

// RegisterRequest represents registration payload
type RegisterRequest struct {
	Email    string                 `json:"email"`
	Password string                 `json:"password"`
	Username string                 `json:"username,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// LoginRequest represents login payload
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	Success      bool   `json:"success"`
	Token        string `json:"token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	User         *User  `json:"user,omitempty"`
	Message      string `json:"message,omitempty"`
}

// ValidationResponse represents token validation response
type ValidationResponse struct {
	Valid     bool      `json:"valid"`
	UserID    string    `json:"user_id,omitempty"`
	Email     string    `json:"email,omitempty"`
	Roles     []string  `json:"roles,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// ErrorResponse represents error response
type ErrorResponse struct {
	Error string `json:"error"`
}