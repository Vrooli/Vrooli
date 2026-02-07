// Package ssh provides SSH key management commands for the CLI.
package ssh

// Outcome is the base API outcome shape used by SSH endpoints.
type Outcome struct {
	OK        bool   `json:"ok"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	Hint      string `json:"hint,omitempty"`
	Timestamp string `json:"timestamp"`
}

// SSHKey mirrors api/ssh.KeyInfo.
type SSHKey struct {
	Path        string `json:"path"`
	Type        string `json:"type"`
	Bits        int    `json:"bits,omitempty"`
	Fingerprint string `json:"fingerprint"`
	Comment     string `json:"comment,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}

// KeysResponse is the response from listing SSH keys.
type KeysResponse struct {
	Outcome
	Keys   []SSHKey `json:"keys"`
	SSHDir string   `json:"ssh_dir"`
}

// GenerateRequest is the request for generating an SSH key.
type GenerateRequest struct {
	Type     string `json:"type,omitempty"` // ed25519, rsa
	Bits     int    `json:"bits,omitempty"` // For RSA: 2048, 4096
	Comment  string `json:"comment,omitempty"`
	Filename string `json:"filename,omitempty"`
	Password string `json:"password,omitempty"`
}

// GenerateResponse is the response from generating an SSH key.
type GenerateResponse struct {
	Outcome
	Key SSHKey `json:"key"`
}

// DeleteRequest is the request for deleting an SSH key.
type DeleteRequest struct {
	KeyPath string `json:"key_path"`
}

// DeleteResponse is the response from deleting an SSH key.
type DeleteResponse struct {
	Outcome
	PrivateDeleted bool `json:"private_deleted"`
	PublicDeleted  bool `json:"public_deleted"`
}

// TestRequest is the request for testing SSH connection.
type TestRequest struct {
	Host    string `json:"host"`
	Port    int    `json:"port,omitempty"` // Default: 22
	User    string `json:"user,omitempty"` // Default: root
	KeyPath string `json:"key_path"`
}

// TestResponse is the response from testing SSH connection.
type TestResponse struct {
	Outcome
	ServerInfo  string `json:"server_info,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	LatencyMs   int64  `json:"latency_ms,omitempty"`
}

// CopyKeyRequest is the request for copying an SSH key to a remote host.
type CopyKeyRequest struct {
	Host     string `json:"host"`
	Port     int    `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
	KeyPath  string `json:"key_path"`
	Password string `json:"password,omitempty"`
}

// CopyKeyResponse is the response from copying an SSH key.
type CopyKeyResponse struct {
	Outcome
	KeyCopied     bool `json:"key_copied"`
	AlreadyExists bool `json:"already_exists"`
}
