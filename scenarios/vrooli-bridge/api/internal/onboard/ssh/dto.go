package ssh

// Status vocabulary for SSH operation outcomes.
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

// Outcome is the base result embedded by SSH operation responses.
type Outcome struct {
	OK        bool   `json:"ok"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	Hint      string `json:"hint,omitempty"`
	Timestamp string `json:"timestamp"`
}

// KeyType represents the type of an SSH key.
type KeyType string

// Supported SSH key types.
const (
	KeyTypeEd25519 KeyType = "ed25519"
	KeyTypeRSA     KeyType = "rsa"
	KeyTypeECDSA   KeyType = "ecdsa"
	KeyTypeDSA     KeyType = "dsa"
	KeyTypeUnknown KeyType = "unknown"
)

// KeyInfo describes a generated or discovered SSH key.
type KeyInfo struct {
	Path        string  `json:"path"`
	Type        KeyType `json:"type"`
	Bits        int     `json:"bits,omitempty"`
	Fingerprint string  `json:"fingerprint"`
	Comment     string  `json:"comment,omitempty"`
	CreatedAt   string  `json:"created_at,omitempty"`
}

// GenerateKeyRequest is the request for generating a new SSH key.
type GenerateKeyRequest struct {
	Type     KeyType `json:"type"`
	Bits     int     `json:"bits,omitempty"`
	Comment  string  `json:"comment,omitempty"`
	Filename string  `json:"filename,omitempty"`
	Password string  `json:"password,omitempty"`
}

// TestConnectionRequest is the request for testing key-based SSH.
type TestConnectionRequest struct {
	Host    string `json:"host"`
	Port    int    `json:"port,omitempty"`
	User    string `json:"user,omitempty"`
	KeyPath string `json:"key_path"`
}

// TestConnectionResponse is the response from testing key-based SSH.
type TestConnectionResponse struct {
	Outcome
	ServerInfo  string `json:"server_info,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	LatencyMs   int64  `json:"latency_ms,omitempty"`
}

// CopyKeyRequest is the request for installing a public key on a host using
// password authentication. KnownHostsFile is the bridge-owned known_hosts the
// TOFU host-key callback reads/writes.
type CopyKeyRequest struct {
	Host           string `json:"host"`
	Port           int    `json:"port,omitempty"`
	User           string `json:"user,omitempty"`
	KeyPath        string `json:"key_path"`
	KnownHostsFile string `json:"-"`
	Password       string `json:"-"`
}

// CopyKeyResponse is the response from installing a public key.
type CopyKeyResponse struct {
	Outcome
	KeyCopied     bool `json:"key_copied"`
	AlreadyExists bool `json:"already_exists"`
}
