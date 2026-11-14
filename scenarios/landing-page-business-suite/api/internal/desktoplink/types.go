package desktoplink

import "time"

// AuthorizationRequest is the consented LPBS-side request to connect one
// desktop installation to one local identity. BusinessAccountID must have
// already been resolved through LPBS membership authorization.
type AuthorizationRequest struct {
	LPBSUserID        string
	BusinessAccountID string
	InstallationID    string
	Resource          string
	Audience          string
	Scopes            []string
	CodeChallenge     string
	RedirectURI       string
	ExpiresAt         time.Time
}

type Authorization struct {
	ID                string
	CodeHash          string
	LPBSUserID        string
	BusinessAccountID string
	InstallationID    string
	Resource          string
	Audience          string
	Scopes            []string
	CodeChallenge     string
	RedirectURI       string
	ExpiresAt         time.Time
}

// Link is the durable relationship between one LPBS business context and one
// verified local principal for one installation and resource audience.
type Link struct {
	ID                string     `json:"id"`
	LPBSUserID        string     `json:"lpbs_user_id"`
	BusinessAccountID string     `json:"business_account_id"`
	LocalProvider     string     `json:"local_provider"`
	LocalPrincipal    string     `json:"local_principal"`
	InstallationID    string     `json:"installation_id"`
	Resource          string     `json:"resource"`
	Audience          string     `json:"audience"`
	Scopes            []string   `json:"scopes"`
	CreatedAt         time.Time  `json:"created_at"`
	RevokedAt         *time.Time `json:"revoked_at,omitempty"`
	RevokedBy         string     `json:"revoked_by,omitempty"`
}

type EntitlementLease struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}
