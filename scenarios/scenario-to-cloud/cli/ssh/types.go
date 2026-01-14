// Package ssh provides SSH key management commands for the CLI.
package ssh

// KeysResponse is the response from listing SSH keys.
type KeysResponse struct {
	Keys      []SSHKey `json:"keys"`
	Timestamp string   `json:"timestamp"`
}

// SSHKey represents an SSH key.
type SSHKey struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // rsa, ed25519, ecdsa
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"public_key,omitempty"`
	CreatedAt   string `json:"created_at"`
	Comment     string `json:"comment,omitempty"`
}

// GenerateRequest is the request for generating a new SSH key.
type GenerateRequest struct {
	Name    string `json:"name"`
	Type    string `json:"type,omitempty"`    // rsa, ed25519 (default: ed25519)
	Bits    int    `json:"bits,omitempty"`    // For RSA: 2048, 4096
	Comment string `json:"comment,omitempty"` // Key comment
}

// GenerateResponse is the response from generating an SSH key.
type GenerateResponse struct {
	Success     bool   `json:"success"`
	Key         SSHKey `json:"key,omitempty"`
	PrivateKey  string `json:"private_key,omitempty"` // Only returned on generation
	Message     string `json:"message,omitempty"`
	Timestamp   string `json:"timestamp"`
}

// DeleteRequest is the request for deleting an SSH key.
type DeleteRequest struct {
	Name string `json:"name"`
}

// DeleteResponse is the response from deleting an SSH key.
type DeleteResponse struct {
	Success   bool   `json:"success"`
	Name      string `json:"name"`
	Message   string `json:"message,omitempty"`
	Timestamp string `json:"timestamp"`
}

// TestRequest is the request for testing SSH connection.
type TestRequest struct {
	Host       string `json:"host"`
	Port       int    `json:"port,omitempty"` // Default: 22
	User       string `json:"user,omitempty"` // Default: root
	KeyName    string `json:"key_name,omitempty"`
	Timeout    int    `json:"timeout,omitempty"` // Seconds
}

// TestResponse is the response from testing SSH connection.
type TestResponse struct {
	Success      bool   `json:"success"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	User         string `json:"user"`
	Latency      string `json:"latency,omitempty"`
	ServerBanner string `json:"server_banner,omitempty"`
	Message      string `json:"message,omitempty"`
	Timestamp    string `json:"timestamp"`
}

// CopyKeyRequest is the request for copying an SSH key to a remote host.
type CopyKeyRequest struct {
	Host     string `json:"host"`
	Port     int    `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
	KeyName  string `json:"key_name,omitempty"`
	Password string `json:"password,omitempty"` // For initial auth
}

// CopyKeyResponse is the response from copying an SSH key.
type CopyKeyResponse struct {
	Success   bool   `json:"success"`
	Host      string `json:"host"`
	User      string `json:"user"`
	KeyName   string `json:"key_name"`
	Message   string `json:"message,omitempty"`
	Timestamp string `json:"timestamp"`
}
