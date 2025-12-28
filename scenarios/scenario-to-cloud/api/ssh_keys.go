package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHKeyType represents the type of SSH key
type SSHKeyType string

const (
	KeyTypeEd25519 SSHKeyType = "ed25519"
	KeyTypeRSA     SSHKeyType = "rsa"
	KeyTypeECDSA   SSHKeyType = "ecdsa"
	KeyTypeDSA     SSHKeyType = "dsa"
	KeyTypeUnknown SSHKeyType = "unknown"
)

// SSHKeyInfo represents information about an SSH key
type SSHKeyInfo struct {
	Path        string     `json:"path"`
	Type        SSHKeyType `json:"type"`
	Bits        int        `json:"bits,omitempty"`
	Fingerprint string     `json:"fingerprint"`
	Comment     string     `json:"comment,omitempty"`
	CreatedAt   string     `json:"created_at,omitempty"`
}

// ListSSHKeysResponse is the response for listing SSH keys
type ListSSHKeysResponse struct {
	Keys      []SSHKeyInfo `json:"keys"`
	SSHDir    string       `json:"ssh_dir"`
	Timestamp string       `json:"timestamp"`
}

// GenerateSSHKeyRequest is the request for generating a new SSH key
type GenerateSSHKeyRequest struct {
	Type     SSHKeyType `json:"type"`
	Bits     int        `json:"bits,omitempty"`
	Comment  string     `json:"comment,omitempty"`
	Filename string     `json:"filename,omitempty"`
	Password string     `json:"password,omitempty"`
}

// GenerateSSHKeyResponse is the response after generating a key
type GenerateSSHKeyResponse struct {
	Key       SSHKeyInfo `json:"key"`
	Timestamp string     `json:"timestamp"`
}

// GetPublicKeyRequest is the request for retrieving a public key
type GetPublicKeyRequest struct {
	KeyPath string `json:"key_path"`
}

// GetPublicKeyResponse is the response containing the public key
type GetPublicKeyResponse struct {
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
	Timestamp   string `json:"timestamp"`
}

// TestSSHConnectionRequest is the request for testing SSH connection
type TestSSHConnectionRequest struct {
	Host    string `json:"host"`
	Port    int    `json:"port,omitempty"`
	User    string `json:"user,omitempty"`
	KeyPath string `json:"key_path"`
}

// TestSSHConnectionResponse is the response from testing SSH connection
type TestSSHConnectionResponse struct {
	OK          bool   `json:"ok"`
	Status      string `json:"status"`
	Message     string `json:"message,omitempty"`
	Hint        string `json:"hint,omitempty"`
	ServerInfo  string `json:"server_info,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	LatencyMs   int64  `json:"latency_ms,omitempty"`
	Timestamp   string `json:"timestamp"`
}

// CopySSHKeyRequest is the request for copying SSH key to a server
type CopySSHKeyRequest struct {
	Host     string `json:"host"`
	Port     int    `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
	KeyPath  string `json:"key_path"`
	Password string `json:"password"`
}

// CopySSHKeyResponse is the response from copying SSH key
type CopySSHKeyResponse struct {
	OK            bool   `json:"ok"`
	Status        string `json:"status"`
	Message       string `json:"message,omitempty"`
	Hint          string `json:"hint,omitempty"`
	KeyCopied     bool   `json:"key_copied"`
	AlreadyExists bool   `json:"already_exists"`
	Timestamp     string `json:"timestamp"`
}

// getSSHDir returns the user's ~/.ssh directory
func getSSHDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(homeDir, ".ssh"), nil
}

// validateSSHPath ensures the path is within ~/.ssh/ to prevent path traversal
func validateSSHPath(path string) error {
	// Handle ~ expansion
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		path = filepath.Join(homeDir, path[2:])
	}

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Get ~/.ssh directory
	sshDir, err := getSSHDir()
	if err != nil {
		return err
	}

	// Ensure path is within ~/.ssh
	if !strings.HasPrefix(absPath, sshDir+string(os.PathSeparator)) && absPath != sshDir {
		return fmt.Errorf("path must be within ~/.ssh")
	}

	// Check for path traversal
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	return nil
}

// validateKeyFilename ensures the filename is safe
func validateKeyFilename(filename string) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}
	if strings.ContainsAny(filename, "/\\") {
		return fmt.Errorf("filename cannot contain path separators")
	}
	if strings.Contains(filename, "..") {
		return fmt.Errorf("filename cannot contain '..'")
	}
	if len(filename) > 64 {
		return fmt.Errorf("filename too long (max 64 characters)")
	}
	// Must start with alphanumeric or underscore
	if len(filename) > 0 {
		c := filename[0]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return fmt.Errorf("filename must start with alphanumeric character or underscore")
		}
	}
	return nil
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

// DiscoverSSHKeys scans ~/.ssh/ for SSH key files
func DiscoverSSHKeys(sshDir string) ([]SSHKeyInfo, error) {
	if sshDir == "" {
		var err error
		sshDir, err = getSSHDir()
		if err != nil {
			return nil, err
		}
	}

	// Check if directory exists
	info, err := os.Stat(sshDir)
	if os.IsNotExist(err) {
		return []SSHKeyInfo{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot access SSH directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("~/.ssh is not a directory")
	}

	// Read directory contents
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read SSH directory: %w", err)
	}

	keys := []SSHKeyInfo{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Skip public keys, known_hosts, config, etc.
		if strings.HasSuffix(name, ".pub") ||
			name == "known_hosts" ||
			name == "known_hosts.old" ||
			name == "config" ||
			name == "authorized_keys" ||
			name == "environment" {
			continue
		}

		keyPath := filepath.Join(sshDir, name)
		keyInfo, err := parseSSHKeyFile(keyPath)
		if err != nil {
			// Skip files that aren't valid SSH keys
			continue
		}
		keys = append(keys, keyInfo)
	}

	return keys, nil
}

// parseSSHKeyFile extracts information from an SSH key file
func parseSSHKeyFile(keyPath string) (SSHKeyInfo, error) {
	// Read the first few bytes to check if it's a private key
	file, err := os.Open(keyPath)
	if err != nil {
		return SSHKeyInfo{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return SSHKeyInfo{}, fmt.Errorf("empty file")
	}

	firstLine := scanner.Text()
	if !strings.HasPrefix(firstLine, "-----BEGIN") {
		return SSHKeyInfo{}, fmt.Errorf("not a PEM-encoded key")
	}

	// Determine key type from header
	keyType := KeyTypeUnknown
	if strings.Contains(firstLine, "OPENSSH PRIVATE KEY") {
		// OpenSSH format - need to check public key for actual type
		keyType = KeyTypeUnknown // Will be determined from fingerprint output
	} else if strings.Contains(firstLine, "RSA PRIVATE KEY") {
		keyType = KeyTypeRSA
	} else if strings.Contains(firstLine, "DSA PRIVATE KEY") {
		keyType = KeyTypeDSA
	} else if strings.Contains(firstLine, "EC PRIVATE KEY") {
		keyType = KeyTypeECDSA
	}

	// Get fingerprint and more info using ssh-keygen
	pubKeyPath := keyPath + ".pub"
	fingerprint, keyTypeFromFP, bits, comment := getKeyFingerprint(pubKeyPath)

	if keyType == KeyTypeUnknown && keyTypeFromFP != KeyTypeUnknown {
		keyType = keyTypeFromFP
	}

	// Get file modification time
	stat, _ := os.Stat(keyPath)
	createdAt := ""
	if stat != nil {
		createdAt = stat.ModTime().Format(time.RFC3339)
	}

	return SSHKeyInfo{
		Path:        keyPath,
		Type:        keyType,
		Bits:        bits,
		Fingerprint: fingerprint,
		Comment:     comment,
		CreatedAt:   createdAt,
	}, nil
}

// getKeyFingerprint uses ssh-keygen to get the fingerprint of a public key
func getKeyFingerprint(pubKeyPath string) (fingerprint string, keyType SSHKeyType, bits int, comment string) {
	cmd := exec.Command("ssh-keygen", "-lf", pubKeyPath)
	output, err := cmd.Output()
	if err != nil {
		return "", KeyTypeUnknown, 0, ""
	}

	// Output format: "256 SHA256:xxxx comment (ED25519)"
	line := strings.TrimSpace(string(output))
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return "", KeyTypeUnknown, 0, ""
	}

	// Parse bits
	fmt.Sscanf(parts[0], "%d", &bits)

	// Parse fingerprint
	fingerprint = parts[1]

	// Parse key type from last field
	lastField := parts[len(parts)-1]
	lastField = strings.Trim(lastField, "()")
	switch strings.ToLower(lastField) {
	case "ed25519":
		keyType = KeyTypeEd25519
	case "rsa":
		keyType = KeyTypeRSA
	case "ecdsa":
		keyType = KeyTypeECDSA
	case "dsa":
		keyType = KeyTypeDSA
	default:
		keyType = KeyTypeUnknown
	}

	// Parse comment (everything between fingerprint and key type)
	if len(parts) > 3 {
		comment = strings.Join(parts[2:len(parts)-1], " ")
	}

	return fingerprint, keyType, bits, comment
}

// GenerateSSHKey generates a new SSH key pair
func GenerateSSHKey(req GenerateSSHKeyRequest) (SSHKeyInfo, error) {
	// Validate key type
	if req.Type != KeyTypeEd25519 && req.Type != KeyTypeRSA {
		return SSHKeyInfo{}, fmt.Errorf("key type must be 'ed25519' or 'rsa'")
	}

	// Set defaults
	if req.Type == KeyTypeRSA && req.Bits == 0 {
		req.Bits = 4096
	}
	if req.Filename == "" {
		if req.Type == KeyTypeEd25519 {
			req.Filename = "id_ed25519"
		} else {
			req.Filename = "id_rsa"
		}
	}

	// Validate filename
	if err := validateKeyFilename(req.Filename); err != nil {
		return SSHKeyInfo{}, err
	}

	// Determine output path
	sshDir, err := getSSHDir()
	if err != nil {
		return SSHKeyInfo{}, err
	}

	// Ensure ~/.ssh exists with proper permissions
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return SSHKeyInfo{}, fmt.Errorf("cannot create ~/.ssh directory: %w", err)
	}

	keyPath := filepath.Join(sshDir, req.Filename)

	// Check if file already exists
	if _, err := os.Stat(keyPath); err == nil {
		return SSHKeyInfo{}, fmt.Errorf("key already exists: %s", keyPath)
	}

	// Build ssh-keygen command
	args := []string{"-t", string(req.Type)}
	if req.Type == KeyTypeRSA && req.Bits > 0 {
		args = append(args, "-b", fmt.Sprintf("%d", req.Bits))
	}
	args = append(args, "-f", keyPath)
	if req.Comment != "" {
		args = append(args, "-C", req.Comment)
	} else {
		args = append(args, "-C", "generated-by-vrooli")
	}
	// Passphrase (empty string means no passphrase)
	args = append(args, "-N", req.Password)

	cmd := exec.Command("ssh-keygen", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return SSHKeyInfo{}, fmt.Errorf("ssh-keygen failed: %s", stderr.String())
	}

	// Read the generated key info
	keyInfo, err := parseSSHKeyFile(keyPath)
	if err != nil {
		return SSHKeyInfo{}, fmt.Errorf("failed to read generated key: %w", err)
	}

	return keyInfo, nil
}

// ReadPublicKey reads and returns the public key content
func ReadPublicKey(keyPath string) (publicKey, fingerprint string, err error) {
	keyPath = expandPath(keyPath)

	// Validate path
	if err := validateSSHPath(keyPath); err != nil {
		return "", "", err
	}

	// If path doesn't end with .pub, add it
	pubKeyPath := keyPath
	if !strings.HasSuffix(keyPath, ".pub") {
		pubKeyPath = keyPath + ".pub"
	}

	// Read public key file
	content, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return "", "", fmt.Errorf("cannot read public key: %w", err)
	}

	publicKey = strings.TrimSpace(string(content))

	// Get fingerprint
	fp, _, _, _ := getKeyFingerprint(pubKeyPath)
	fingerprint = fp

	return publicKey, fingerprint, nil
}

// isIPv6 checks if the given host is an IPv6 address
func isIPv6(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.To4() == nil
}

// ipv6ConnectivityHint returns a helpful hint for IPv6 connectivity issues
const ipv6ConnectivityHint = "You entered an IPv6 address, but your network may not have IPv6 connectivity. Most ISPs still only provide IPv4. Try using the IPv4 address of your server instead."

// TestSSHConnection tests SSH connection to a host using key authentication
func TestSSHConnection(ctx context.Context, req TestSSHConnectionRequest) TestSSHConnectionResponse {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Set defaults
	if req.Port == 0 {
		req.Port = 22
	}
	if req.User == "" {
		req.User = "root"
	}

	keyPath := expandPath(req.KeyPath)

	// Validate key path
	if err := validateSSHPath(keyPath); err != nil {
		return TestSSHConnectionResponse{
			OK:        false,
			Status:    "key_not_found",
			Message:   "Invalid SSH key path",
			Hint:      err.Error(),
			Timestamp: timestamp,
		}
	}

	// Check if key file exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return TestSSHConnectionResponse{
			OK:        false,
			Status:    "key_not_found",
			Message:   "SSH key file not found",
			Hint:      fmt.Sprintf("The file %s does not exist", keyPath),
			Timestamp: timestamp,
		}
	}

	// Use the existing ExecSSHRunner for consistency
	cfg := SSHConfig{
		Host:    req.Host,
		Port:    req.Port,
		User:    req.User,
		KeyPath: keyPath,
	}

	start := time.Now()
	result, err := ExecSSHRunner{}.Run(ctx, cfg, "echo ok && cat /etc/os-release 2>/dev/null | head -5")
	latency := time.Since(start).Milliseconds()

	if err != nil {
		// Parse error to provide helpful hints
		errStr := err.Error()
		status := "unknown_error"
		message := "SSH connection failed"
		hint := errStr

		if strings.Contains(errStr, "Permission denied") || result.ExitCode == 255 {
			status = "auth_failed"
			message = "SSH authentication failed"
			hint = "The server rejected the SSH key. Use 'Copy Key to Server' to add your key, or verify the correct key is selected."
		} else if strings.Contains(errStr, "Connection refused") {
			status = "host_unreachable"
			message = "SSH connection refused"
			hint = "Verify the host and port are correct and that SSH is running on the server."
		} else if strings.Contains(errStr, "i/o timeout") || strings.Contains(errStr, "connection timed out") {
			status = "timeout"
			message = "SSH connection timed out"
			if isIPv6(req.Host) {
				status = "ipv6_unavailable"
				message = "SSH connection timed out (IPv6)"
				hint = ipv6ConnectivityHint
			} else {
				hint = "Check network connectivity and firewall rules."
			}
		} else if strings.Contains(errStr, "No route to host") || strings.Contains(errStr, "network is unreachable") {
			if isIPv6(req.Host) {
				status = "ipv6_unavailable"
				message = "IPv6 not available"
				hint = ipv6ConnectivityHint
			} else {
				status = "host_unreachable"
				message = "Host unreachable"
				hint = "The network path to the host could not be found. Check the IP address and network connectivity."
			}
		}

		return TestSSHConnectionResponse{
			OK:        false,
			Status:    status,
			Message:   message,
			Hint:      hint,
			LatencyMs: latency,
			Timestamp: timestamp,
		}
	}

	// Parse OS info from output
	serverInfo := ""
	lines := strings.Split(result.Stdout, "\n")
	if len(lines) > 1 {
		// Skip "ok" line, parse os-release
		for _, line := range lines[1:] {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				serverInfo = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
				break
			}
		}
	}

	return TestSSHConnectionResponse{
		OK:         true,
		Status:     "connected",
		Message:    "SSH connection successful",
		ServerInfo: serverInfo,
		LatencyMs:  latency,
		Timestamp:  timestamp,
	}
}

// CopySSHKeyToServer copies the public key to the server using password authentication
func CopySSHKeyToServer(ctx context.Context, req CopySSHKeyRequest) CopySSHKeyResponse {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Set defaults
	if req.Port == 0 {
		req.Port = 22
	}
	if req.User == "" {
		req.User = "root"
	}

	keyPath := expandPath(req.KeyPath)

	// Validate key path
	if err := validateSSHPath(keyPath); err != nil {
		return CopySSHKeyResponse{
			OK:        false,
			Status:    "error",
			Message:   "Invalid SSH key path",
			Hint:      err.Error(),
			Timestamp: timestamp,
		}
	}

	// Read public key
	pubKeyPath := keyPath
	if !strings.HasSuffix(keyPath, ".pub") {
		pubKeyPath = keyPath + ".pub"
	}

	pubKeyContent, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return CopySSHKeyResponse{
			OK:        false,
			Status:    "error",
			Message:   "Cannot read public key",
			Hint:      err.Error(),
			Timestamp: timestamp,
		}
	}
	pubKeyLine := strings.TrimSpace(string(pubKeyContent))

	// Validate public key format
	pubKeyParts := strings.Fields(pubKeyLine)
	if len(pubKeyParts) < 2 {
		return CopySSHKeyResponse{
			OK:        false,
			Status:    "error",
			Message:   "Invalid public key format",
			Hint:      "The public key file does not contain a valid SSH public key",
			Timestamp: timestamp,
		}
	}

	// Connect using password authentication
	config := &ssh.ClientConfig{
		User: req.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(req.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TOFU - Trust On First Use
		Timeout:         10 * time.Second,
	}

	addr := net.JoinHostPort(req.Host, fmt.Sprint(req.Port))
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		status := "error"
		message := "SSH connection failed"
		hint := err.Error()

		if strings.Contains(err.Error(), "unable to authenticate") ||
			strings.Contains(err.Error(), "no supported methods remain") {
			status = "auth_failed"
			message = "Password authentication failed"
			hint = "The password may be incorrect, or password authentication may be disabled on the server."
		} else if strings.Contains(err.Error(), "connection refused") {
			status = "error"
			message = "SSH connection refused"
			hint = "Verify the host and port are correct and that SSH is running on the server."
		} else if strings.Contains(err.Error(), "i/o timeout") || strings.Contains(err.Error(), "connection timed out") {
			if isIPv6(req.Host) {
				status = "ipv6_unavailable"
				message = "SSH connection timed out (IPv6)"
				hint = ipv6ConnectivityHint
			} else {
				status = "error"
				message = "SSH connection timed out"
				hint = "Check network connectivity and firewall rules."
			}
		} else if strings.Contains(err.Error(), "no route to host") || strings.Contains(err.Error(), "network is unreachable") {
			if isIPv6(req.Host) {
				status = "ipv6_unavailable"
				message = "IPv6 not available"
				hint = ipv6ConnectivityHint
			} else {
				status = "error"
				message = "Host unreachable"
				hint = "The network path to the host could not be found. Check the IP address and network connectivity."
			}
		}

		return CopySSHKeyResponse{
			OK:        false,
			Status:    status,
			Message:   message,
			Hint:      hint,
			Timestamp: timestamp,
		}
	}
	defer client.Close()

	// Check if key already exists in authorized_keys
	keyData := pubKeyParts[1] // The base64-encoded key data
	checkSession, err := client.NewSession()
	if err != nil {
		return CopySSHKeyResponse{
			OK:        false,
			Status:    "error",
			Message:   "Failed to create SSH session",
			Hint:      err.Error(),
			Timestamp: timestamp,
		}
	}

	// Use grep to check if key exists
	checkCmd := fmt.Sprintf("grep -qF %s ~/.ssh/authorized_keys 2>/dev/null", shellQuoteSingle(keyData))
	checkErr := checkSession.Run(checkCmd)
	checkSession.Close()

	if checkErr == nil {
		// Key already exists
		return CopySSHKeyResponse{
			OK:            true,
			Status:        "already_exists",
			Message:       "SSH key is already authorized on the server",
			KeyCopied:     false,
			AlreadyExists: true,
			Timestamp:     timestamp,
		}
	}

	// Key doesn't exist, add it
	addSession, err := client.NewSession()
	if err != nil {
		return CopySSHKeyResponse{
			OK:        false,
			Status:    "error",
			Message:   "Failed to create SSH session",
			Hint:      err.Error(),
			Timestamp: timestamp,
		}
	}
	defer addSession.Close()

	// Create .ssh directory, add key, set permissions
	addCmd := fmt.Sprintf(
		"mkdir -p ~/.ssh && chmod 700 ~/.ssh && printf '%%s\\n' %s >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys",
		shellQuoteSingle(pubKeyLine),
	)

	if err := addSession.Run(addCmd); err != nil {
		return CopySSHKeyResponse{
			OK:        false,
			Status:    "error",
			Message:   "Failed to add key to authorized_keys",
			Hint:      err.Error(),
			Timestamp: timestamp,
		}
	}

	return CopySSHKeyResponse{
		OK:            true,
		Status:        "copied",
		Message:       "SSH key successfully copied to server",
		KeyCopied:     true,
		AlreadyExists: false,
		Timestamp:     timestamp,
	}
}

