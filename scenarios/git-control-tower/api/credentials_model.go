package main

import "time"

// CredentialType represents the type of authentication.
type CredentialType string

const (
	// CredentialTypeHTTPS uses username/token authentication.
	CredentialTypeHTTPS CredentialType = "https"
	// CredentialTypeSSH uses SSH key authentication.
	CredentialTypeSSH CredentialType = "ssh"
)

// Credential represents stored authentication information for a git remote.
type Credential struct {
	// ID is a unique identifier for this credential (derived from remote name).
	ID string `json:"id"`

	// Remote is the git remote name (e.g., "origin").
	Remote string `json:"remote"`

	// URL is the remote URL (e.g., "https://github.com/user/repo.git").
	URL string `json:"url"`

	// Type indicates the authentication type (https or ssh).
	Type CredentialType `json:"type"`

	// Username is the git username (for HTTPS auth).
	Username string `json:"username,omitempty"`

	// TokenMasked is the masked token for display (e.g., "ghp_****...1234").
	TokenMasked string `json:"token_masked,omitempty"`

	// SSHKeyPath is the path to the SSH private key (for SSH auth).
	SSHKeyPath string `json:"ssh_key_path,omitempty"`

	// IsConfigured indicates if credentials are fully set up.
	IsConfigured bool `json:"is_configured"`

	// CreatedAt is when the credential was first stored.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the credential was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}

// CredentialsListResponse returns all stored credentials.
type CredentialsListResponse struct {
	// Credentials is the list of all stored credentials.
	Credentials []Credential `json:"credentials"`

	// Timestamp is when the response was generated.
	Timestamp time.Time `json:"timestamp"`
}

// CredentialSaveRequest saves or updates a credential.
type CredentialSaveRequest struct {
	// Remote is the git remote name (e.g., "origin").
	Remote string `json:"remote"`

	// URL optionally updates the remote URL.
	URL string `json:"url,omitempty"`

	// Username is the git username (for HTTPS auth).
	Username string `json:"username,omitempty"`

	// Token is the personal access token (plaintext, will be encrypted).
	Token string `json:"token,omitempty"`

	// SSHKeyPath is the path to the SSH private key (for SSH auth).
	SSHKeyPath string `json:"ssh_key_path,omitempty"`
}

// CredentialSaveResponse is the result of saving a credential.
type CredentialSaveResponse struct {
	// Success indicates if the operation succeeded.
	Success bool `json:"success"`

	// Credential is the saved credential (with masked token).
	Credential *Credential `json:"credential,omitempty"`

	// Error contains any error message.
	Error string `json:"error,omitempty"`

	// Timestamp is when the response was generated.
	Timestamp time.Time `json:"timestamp"`
}

// CredentialDeleteRequest deletes a credential.
type CredentialDeleteRequest struct {
	// ID is the credential ID to delete.
	ID string `json:"id"`
}

// CredentialDeleteResponse is the result of deleting a credential.
type CredentialDeleteResponse struct {
	// Success indicates if the operation succeeded.
	Success bool `json:"success"`

	// Error contains any error message.
	Error string `json:"error,omitempty"`

	// Timestamp is when the response was generated.
	Timestamp time.Time `json:"timestamp"`
}

// CredentialTestRequest tests credential authentication.
type CredentialTestRequest struct {
	// Remote is the git remote name to test.
	Remote string `json:"remote"`

	// UseStored uses stored credentials if true, otherwise tests without auth.
	UseStored bool `json:"use_stored,omitempty"`
}

// CredentialTestResponse is the result of testing a credential.
type CredentialTestResponse struct {
	// Success indicates if the test passed (reachable AND authorized).
	Success bool `json:"success"`

	// Reachable indicates if the remote URL is reachable.
	Reachable bool `json:"reachable"`

	// Authorized indicates if authentication succeeded.
	Authorized bool `json:"authorized"`

	// Error contains any error message.
	Error string `json:"error,omitempty"`

	// Timestamp is when the response was generated.
	Timestamp time.Time `json:"timestamp"`
}

// RemoteURLUpdateRequest updates a remote's URL.
type RemoteURLUpdateRequest struct {
	// Remote is the git remote name.
	Remote string `json:"remote"`

	// URL is the new remote URL.
	URL string `json:"url"`
}

// RemoteURLUpdateResponse is the result of updating a remote URL.
type RemoteURLUpdateResponse struct {
	// Success indicates if the operation succeeded.
	Success bool `json:"success"`

	// OldURL is the previous remote URL.
	OldURL string `json:"old_url,omitempty"`

	// NewURL is the new remote URL.
	NewURL string `json:"new_url,omitempty"`

	// Error contains any error message.
	Error string `json:"error,omitempty"`

	// Timestamp is when the response was generated.
	Timestamp time.Time `json:"timestamp"`
}
