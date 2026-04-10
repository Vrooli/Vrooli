package ssh

// Status vocabulary for API responses.
const (
	StatusSuccess         = "success"
	StatusAlreadyExists   = "already_exists"
	StatusNotFound        = "not_found"
	StatusAuthFailed      = "auth_failed"
	StatusTimeout         = "timeout"
	StatusHostUnreachable = "host_unreachable"
	StatusHostKeyChanged  = "host_key_changed"
	StatusIPv6Unavailable = "ipv6_unavailable"
	StatusInvalidInput    = "invalid_input"
	StatusDiskFull        = "disk_full"
	StatusDNSFailed       = "dns_failed"
	StatusKeyError        = "key_error"
	StatusError           = "error"
)

// Outcome is the base response struct embedded by all SSH response DTOs.
type Outcome struct {
	OK        bool   `json:"ok"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	Hint      string `json:"hint,omitempty"`
	Timestamp string `json:"timestamp"`
}

// ListKeysResponse is the response for listing SSH keys.
type ListKeysResponse struct {
	Outcome
	Keys   []KeyInfo `json:"keys"`
	SSHDir string    `json:"ssh_dir"`
}

// GenerateKeyRequest is the request for generating a new SSH key.
type GenerateKeyRequest struct {
	Type     KeyType `json:"type"`
	Bits     int     `json:"bits,omitempty"`
	Comment  string  `json:"comment,omitempty"`
	Filename string  `json:"filename,omitempty"`
	Password string  `json:"password,omitempty"`
}

// GenerateKeyResponse is the response after generating a key.
type GenerateKeyResponse struct {
	Outcome
	Key KeyInfo `json:"key"`
}

// GetPublicKeyRequest is the request for retrieving a public key.
type GetPublicKeyRequest struct {
	KeyPath string `json:"key_path"`
}

// GetPublicKeyResponse is the response containing the public key.
type GetPublicKeyResponse struct {
	Outcome
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
}

// TestConnectionRequest is the request for testing SSH connection.
type TestConnectionRequest struct {
	Host    string `json:"host"`
	Port    int    `json:"port,omitempty"`
	User    string `json:"user,omitempty"`
	KeyPath string `json:"key_path"`
}

// TestConnectionResponse is the response from testing SSH connection.
type TestConnectionResponse struct {
	Outcome
	ServerInfo  string `json:"server_info,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	LatencyMs   int64  `json:"latency_ms,omitempty"`
}

// CopyKeyRequest is the request for copying SSH key to a server.
type CopyKeyRequest struct {
	Host     string `json:"host"`
	Port     int    `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
	KeyPath  string `json:"key_path"`
	Password string `json:"password"`
}

// CopyKeyResponse is the response from copying SSH key.
type CopyKeyResponse struct {
	Outcome
	KeyCopied     bool `json:"key_copied"`
	AlreadyExists bool `json:"already_exists"`
}

// DeleteKeyRequest is the request for deleting an SSH key.
type DeleteKeyRequest struct {
	KeyPath string `json:"key_path"`
}

// DeleteKeyResponse is the response from deleting an SSH key.
type DeleteKeyResponse struct {
	Outcome
	PrivateDeleted bool `json:"private_deleted"`
	PublicDeleted  bool `json:"public_deleted"`
}
