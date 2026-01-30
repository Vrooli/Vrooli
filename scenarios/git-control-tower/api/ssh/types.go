package ssh

import "time"

// KeyType represents the type of SSH key.
type KeyType string

const (
	KeyTypeEd25519 KeyType = "ed25519"
	KeyTypeRSA     KeyType = "rsa"
	KeyTypeECDSA   KeyType = "ecdsa"
	KeyTypeDSA     KeyType = "dsa"
	KeyTypeUnknown KeyType = "unknown"
)

// KeyInfo represents information about an SSH key.
type KeyInfo struct {
	Path        string  `json:"path"`
	Filename    string  `json:"filename"`
	Type        KeyType `json:"type"`
	Bits        int     `json:"bits,omitempty"`
	Fingerprint string  `json:"fingerprint"`
	Comment     string  `json:"comment,omitempty"`
	CreatedAt   string  `json:"created_at,omitempty"`
	HasPublic   bool    `json:"has_public"`
}

// ListKeysResponse is the response for listing SSH keys.
type ListKeysResponse struct {
	Keys      []KeyInfo `json:"keys"`
	SSHDir    string    `json:"ssh_dir"`
	Timestamp string    `json:"timestamp"`
}

// GenerateKeyRequest is the request for generating a new SSH key.
type GenerateKeyRequest struct {
	Type     KeyType `json:"type"`
	Bits     int     `json:"bits,omitempty"`
	Comment  string  `json:"comment,omitempty"`
	Filename string  `json:"filename,omitempty"`
}

// GenerateKeyResponse is the response after generating a key.
type GenerateKeyResponse struct {
	Success   bool    `json:"success"`
	Key       KeyInfo `json:"key,omitempty"`
	PublicKey string  `json:"public_key,omitempty"`
	Error     string  `json:"error,omitempty"`
	Timestamp string  `json:"timestamp"`
}

// GetPublicKeyRequest is the request for retrieving a public key.
type GetPublicKeyRequest struct {
	KeyPath string `json:"key_path"`
}

// GetPublicKeyResponse is the response containing the public key.
type GetPublicKeyResponse struct {
	Success     bool   `json:"success"`
	PublicKey   string `json:"public_key,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	Error       string `json:"error,omitempty"`
	Timestamp   string `json:"timestamp"`
}

// TestConnectionRequest is the request for testing SSH connection to GitHub.
type TestConnectionRequest struct {
	KeyPath string `json:"key_path"`
}

// TestConnectionResponse is the response from testing SSH connection.
type TestConnectionResponse struct {
	Success     bool   `json:"success"`
	Status      string `json:"status"`
	Message     string `json:"message,omitempty"`
	Hint        string `json:"hint,omitempty"`
	GitHubUser  string `json:"github_user,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	LatencyMs   int64  `json:"latency_ms,omitempty"`
	Timestamp   string `json:"timestamp"`
}

// DeleteKeyRequest is the request for deleting an SSH key.
type DeleteKeyRequest struct {
	KeyPath string `json:"key_path"`
}

// DeleteKeyResponse is the response from deleting an SSH key.
type DeleteKeyResponse struct {
	Success        bool   `json:"success"`
	Message        string `json:"message,omitempty"`
	Error          string `json:"error,omitempty"`
	PrivateDeleted bool   `json:"private_deleted"`
	PublicDeleted  bool   `json:"public_deleted"`
	Timestamp      string `json:"timestamp"`
}

// timestamp returns the current UTC time in RFC3339 format.
func timestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}
