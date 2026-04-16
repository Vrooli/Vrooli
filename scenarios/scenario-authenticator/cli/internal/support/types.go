package support

import (
	"encoding/json"
	"time"
)

// User mirrors models.User exposed via /api/v1/users endpoints.
type User struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	Username      string     `json:"username,omitempty"`
	Roles         []string   `json:"roles"`
	EmailVerified bool       `json:"email_verified"`
	CreatedAt     time.Time  `json:"created_at"`
	LastLogin     *time.Time `json:"last_login,omitempty"`
}

// UsersListResponse is the shape returned by GET /api/v1/users.
type UsersListResponse struct {
	Users []User `json:"users"`
	Total int    `json:"total"`
}

// AuthResponse mirrors models.AuthResponse from register/login/refresh.
type AuthResponse struct {
	Success      bool   `json:"success"`
	Token        string `json:"token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	User         *User  `json:"user,omitempty"`
	Message      string `json:"message,omitempty"`
}

// UpdateUserResponse is the shape returned by PUT /api/v1/users/{id}.
type UpdateUserResponse struct {
	Success bool `json:"success"`
	User    User `json:"user"`
}

// ValidationResponse mirrors models.ValidationResponse from GET /auth/validate.
type ValidationResponse struct {
	Valid     bool      `json:"valid"`
	UserID    string    `json:"user_id,omitempty"`
	Email     string    `json:"email,omitempty"`
	Roles     []string  `json:"roles,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// Session is one entry in the sessions list response.
type Session struct {
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	IPAddress string    `json:"ip_address,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SessionsListResponse is the shape returned by GET /api/v1/sessions.
type SessionsListResponse struct {
	Sessions []Session `json:"sessions"`
	Total    int       `json:"total"`
}

// APIKey is an API key record returned by the apikeys endpoints.
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

// APIKeyValidation is the shape returned by POST /api/v1/apikeys/validate.
type APIKeyValidation struct {
	Valid       bool     `json:"valid"`
	UserID      string   `json:"user_id,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	RateLimit   int      `json:"rate_limit,omitempty"`
	Error       string   `json:"error,omitempty"`
}

// Application is a registered scenario/app record.
type Application struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	DisplayName    string     `json:"display_name"`
	Description    string     `json:"description,omitempty"`
	ScenarioType   string     `json:"scenario_type"`
	AllowedOrigins []string   `json:"allowed_origins"`
	RedirectURIs   []string   `json:"redirect_uris"`
	Permissions    []string   `json:"permissions"`
	RateLimit      int        `json:"rate_limit"`
	MaxUsers       *int       `json:"max_users,omitempty"`
	IsActive       bool       `json:"is_active"`
	LastAccessed   *time.Time `json:"last_accessed,omitempty"`
	TotalUsers     int        `json:"total_users"`
	TotalAuths     int        `json:"total_authentications"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ApplicationStats is the leaner shape returned when /applications?stats=true.
type ApplicationStats struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	DisplayName      string     `json:"display_name"`
	IsActive         bool       `json:"is_active"`
	TotalUsers       int        `json:"total_users"`
	ActiveSessions   int        `json:"active_sessions"`
	TotalEventsToday int        `json:"total_events_today"`
	RateLimit        int        `json:"rate_limit"`
	LastAccessed     *time.Time `json:"last_accessed,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// ApplicationsListResponse is the envelope returned by GET /api/v1/applications.
type ApplicationsListResponse struct {
	Applications json.RawMessage `json:"applications"`
	Total        int             `json:"total"`
}

// AppCredentials is the response from POST /api/v1/applications (registration).
type AppCredentials struct {
	ApplicationID string `json:"application_id"`
	APIKey        string `json:"api_key"`
	APISecret     string `json:"api_secret"`
}

// IntegrationCode is the response from GET /applications/{id}/integration-code.
type IntegrationCode struct {
	ApplicationName string `json:"application_name"`
	IntegrationType string `json:"integration_type"`
	Code            string `json:"code"`
}

// OAuthProvider is one entry in the providers list.
type OAuthProvider struct {
	Name    string `json:"name"`
	Display string `json:"display"`
	Icon    string `json:"icon,omitempty"`
	Enabled bool   `json:"enabled"`
}

// OAuthProvidersResponse is returned by GET /api/v1/auth/oauth/providers.
type OAuthProvidersResponse struct {
	Providers []OAuthProvider `json:"providers"`
}

// OAuthLoginResponse is returned by GET /api/v1/auth/oauth/login.
type OAuthLoginResponse struct {
	AuthURL  string `json:"auth_url"`
	Provider string `json:"provider"`
}

// TOTPConfig is the response from POST /auth/2fa/setup.
type TOTPConfig struct {
	Secret      string   `json:"secret"`
	QRCodeURL   string   `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes"`
}
