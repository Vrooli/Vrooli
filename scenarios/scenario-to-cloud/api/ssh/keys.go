package ssh

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

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
	Type        KeyType `json:"type"`
	Bits        int     `json:"bits,omitempty"`
	Fingerprint string  `json:"fingerprint"`
	Comment     string  `json:"comment,omitempty"`
	CreatedAt   string  `json:"created_at,omitempty"`
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
	Password string  `json:"password,omitempty"`
}

// GenerateKeyResponse is the response after generating a key.
type GenerateKeyResponse struct {
	Key       KeyInfo `json:"key"`
	Timestamp string  `json:"timestamp"`
}

// GetPublicKeyRequest is the request for retrieving a public key.
type GetPublicKeyRequest struct {
	KeyPath string `json:"key_path"`
}

// GetPublicKeyResponse is the response containing the public key.
type GetPublicKeyResponse struct {
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
	Timestamp   string `json:"timestamp"`
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
	OK          bool   `json:"ok"`
	Status      string `json:"status"`
	Message     string `json:"message,omitempty"`
	Hint        string `json:"hint,omitempty"`
	ServerInfo  string `json:"server_info,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	LatencyMs   int64  `json:"latency_ms,omitempty"`
	Timestamp   string `json:"timestamp"`
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
	OK            bool   `json:"ok"`
	Status        string `json:"status"`
	Message       string `json:"message,omitempty"`
	Hint          string `json:"hint,omitempty"`
	KeyCopied     bool   `json:"key_copied"`
	AlreadyExists bool   `json:"already_exists"`
	Timestamp     string `json:"timestamp"`
}

// DeleteKeyRequest is the request for deleting an SSH key.
type DeleteKeyRequest struct {
	KeyPath string `json:"key_path"`
}

// DeleteKeyResponse is the response from deleting an SSH key.
type DeleteKeyResponse struct {
	OK             bool   `json:"ok"`
	Message        string `json:"message,omitempty"`
	PrivateDeleted bool   `json:"private_deleted"`
	PublicDeleted  bool   `json:"public_deleted"`
	Timestamp      string `json:"timestamp"`
}

// GetSSHDir returns the user's ~/.ssh directory.
func GetSSHDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(homeDir, ".ssh"), nil
}

// ValidateSSHPath ensures the path is within ~/.ssh/ to prevent path traversal.
func ValidateSSHPath(path string) error {
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
	sshDir, err := GetSSHDir()
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

// ValidateKeyFilename ensures the filename is safe.
func ValidateKeyFilename(filename string) error {
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

// ExpandPath expands ~ to home directory.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

// DiscoverKeys scans ~/.ssh/ for SSH key files.
func DiscoverKeys(sshDir string) ([]KeyInfo, error) {
	if sshDir == "" {
		var err error
		sshDir, err = GetSSHDir()
		if err != nil {
			return nil, err
		}
	}

	// Check if directory exists
	info, err := os.Stat(sshDir)
	if os.IsNotExist(err) {
		return []KeyInfo{}, nil
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

	keys := []KeyInfo{}
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
		keyInfo, err := parseKeyFile(keyPath)
		if err != nil {
			// Skip files that aren't valid SSH keys
			continue
		}
		keys = append(keys, keyInfo)
	}

	return keys, nil
}

// parseKeyFile extracts information from an SSH key file.
func parseKeyFile(keyPath string) (KeyInfo, error) {
	// Read the first few bytes to check if it's a private key
	file, err := os.Open(keyPath)
	if err != nil {
		return KeyInfo{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return KeyInfo{}, fmt.Errorf("empty file")
	}

	firstLine := scanner.Text()
	if !strings.HasPrefix(firstLine, "-----BEGIN") {
		return KeyInfo{}, fmt.Errorf("not a PEM-encoded key")
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

	return KeyInfo{
		Path:        keyPath,
		Type:        keyType,
		Bits:        bits,
		Fingerprint: fingerprint,
		Comment:     comment,
		CreatedAt:   createdAt,
	}, nil
}

// getKeyFingerprint uses ssh-keygen to get the fingerprint of a public key.
func getKeyFingerprint(pubKeyPath string) (fingerprint string, keyType KeyType, bits int, comment string) {
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
	if _, err := fmt.Sscanf(parts[0], "%d", &bits); err != nil {
		bits = 0
	}

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

// ReadPublicKey reads and returns the public key content.
func ReadPublicKey(keyPath string) (publicKey, fingerprint string, err error) {
	keyPath = ExpandPath(keyPath)

	// Validate path
	if err := ValidateSSHPath(keyPath); err != nil {
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

// IsIPv6 checks if the given host is an IPv6 address.
func IsIPv6(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.To4() == nil
}

// IPv6ConnectivityHint is a helpful hint for IPv6 connectivity issues.
const IPv6ConnectivityHint = "You entered an IPv6 address, but your network may not have IPv6 connectivity. Most ISPs still only provide IPv4. Try using the IPv4 address of your server instead."

// errorClassification represents a classified SSH connection error with
// actionable status, message, and hint for the user.
type errorClassification struct {
	Status  string
	Message string
	Hint    string
}

// classifySSHError analyzes an SSH error string and returns a user-friendly classification.
func classifySSHError(errStr, host, defaultHint string) errorClassification {
	errLower := strings.ToLower(errStr)

	switch {
	case strings.Contains(errStr, "Host key verification failed"):
		return errorClassification{
			Status:  "host_key_changed",
			Message: "Server host key has changed",
			Hint:    "The server's identity has changed since you last connected. This could indicate a server rebuild or a security issue. Remove the old key with: ssh-keygen -R " + host,
		}

	case strings.Contains(errStr, "Permission denied"):
		return errorClassification{
			Status:  "auth_failed",
			Message: "SSH authentication failed",
			Hint:    "The server rejected the SSH key. Use 'Copy Key to Server' to add your key, or verify the correct key is selected.",
		}

	case strings.Contains(errLower, "unable to authenticate"),
		strings.Contains(errLower, "no supported methods remain"):
		return errorClassification{
			Status:  "auth_failed",
			Message: "Password authentication failed",
			Hint:    "The password may be incorrect, or password authentication may be disabled on the server.",
		}

	case strings.Contains(errLower, "connection refused"):
		return errorClassification{
			Status:  "host_unreachable",
			Message: "SSH connection refused",
			Hint:    "Verify the host and port are correct and that SSH is running on the server.",
		}

	case strings.Contains(errLower, "i/o timeout"),
		strings.Contains(errLower, "connection timed out"),
		strings.Contains(errLower, "timed out"):
		if IsIPv6(host) {
			return errorClassification{
				Status:  "ipv6_unavailable",
				Message: "SSH connection timed out (IPv6)",
				Hint:    IPv6ConnectivityHint,
			}
		}
		return errorClassification{
			Status:  "timeout",
			Message: "SSH connection timed out",
			Hint:    "Check network connectivity and firewall rules.",
		}

	case strings.Contains(errLower, "no route to host"),
		strings.Contains(errLower, "network is unreachable"):
		if IsIPv6(host) {
			return errorClassification{
				Status:  "ipv6_unavailable",
				Message: "IPv6 not available",
				Hint:    IPv6ConnectivityHint,
			}
		}
		return errorClassification{
			Status:  "host_unreachable",
			Message: "Host unreachable",
			Hint:    "The network path to the host could not be found. Check the IP address and network connectivity.",
		}

	default:
		return errorClassification{
			Status:  "unknown_error",
			Message: "SSH connection failed",
			Hint:    defaultHint,
		}
	}
}

// TestConnection tests SSH connection to a host using key authentication.
func TestConnection(ctx context.Context, req TestConnectionRequest) TestConnectionResponse {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Set defaults
	if req.Port == 0 {
		req.Port = DefaultPort
	}
	if req.User == "" {
		req.User = DefaultUser
	}

	keyPath := ExpandPath(req.KeyPath)

	// Validate key path
	if err := ValidateSSHPath(keyPath); err != nil {
		return TestConnectionResponse{
			OK:        false,
			Status:    "key_not_found",
			Message:   "Invalid SSH key path",
			Hint:      err.Error(),
			Timestamp: timestamp,
		}
	}

	// Check if key file exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return TestConnectionResponse{
			OK:        false,
			Status:    "key_not_found",
			Message:   "SSH key file not found",
			Hint:      fmt.Sprintf("The file %s does not exist", keyPath),
			Timestamp: timestamp,
		}
	}

	// Build SSH command with IdentitiesOnly=yes to test ONLY the specified key
	// (not SSH agent keys). This ensures the test accurately reflects whether
	// the key file itself is authorized on the server.
	args := []string{
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "IdentitiesOnly=yes", // Only use the specified key, not SSH agent
		"-p", fmt.Sprintf("%d", req.Port),
		"-i", keyPath,
		fmt.Sprintf("%s@%s", req.User, req.Host),
		"--",
		"echo ok && cat /etc/os-release 2>/dev/null | head -5",
	}

	start := time.Now()
	cmd := exec.CommandContext(ctx, "ssh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	latency := time.Since(start).Milliseconds()

	result := Result{
		Stdout:   strings.TrimRight(stdout.String(), "\n"),
		Stderr:   strings.TrimRight(stderr.String(), "\n"),
		ExitCode: 0,
	}
	if runErr != nil {
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			result.ExitCode = ee.ExitCode()
		} else {
			result.ExitCode = 255
		}
	}

	if runErr != nil {
		// Combine error sources for classification
		errStr := runErr.Error() + " " + result.Stderr
		defaultHint := result.Stderr
		if defaultHint == "" {
			defaultHint = runErr.Error()
		}

		classified := classifySSHError(errStr, req.Host, defaultHint)
		return TestConnectionResponse{
			OK:        false,
			Status:    classified.Status,
			Message:   classified.Message,
			Hint:      classified.Hint,
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

	return TestConnectionResponse{
		OK:         true,
		Status:     "connected",
		Message:    "SSH connection successful",
		ServerInfo: serverInfo,
		LatencyMs:  latency,
		Timestamp:  timestamp,
	}
}

// CopyKeyToServer copies the public key to the server using password authentication.
func CopyKeyToServer(ctx context.Context, req CopyKeyRequest) CopyKeyResponse {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Set defaults
	if req.Port == 0 {
		req.Port = DefaultPort
	}
	if req.User == "" {
		req.User = DefaultUser
	}

	keyPath := ExpandPath(req.KeyPath)

	// Validate key path
	if err := ValidateSSHPath(keyPath); err != nil {
		return CopyKeyResponse{
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
		return CopyKeyResponse{
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
		return CopyKeyResponse{
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
		classified := classifySSHError(err.Error(), req.Host, err.Error())
		return CopyKeyResponse{
			OK:        false,
			Status:    classified.Status,
			Message:   classified.Message,
			Hint:      classified.Hint,
			Timestamp: timestamp,
		}
	}
	defer client.Close()

	// Check if key already exists in authorized_keys
	keyData := pubKeyParts[1] // The base64-encoded key data
	checkSession, err := client.NewSession()
	if err != nil {
		return CopyKeyResponse{
			OK:        false,
			Status:    "error",
			Message:   "Failed to create SSH session",
			Hint:      err.Error(),
			Timestamp: timestamp,
		}
	}

	// Use grep to check if key exists
	checkCmd := fmt.Sprintf("grep -qF %s ~/.ssh/authorized_keys 2>/dev/null", QuoteSingle(keyData))
	checkErr := checkSession.Run(checkCmd)
	checkSession.Close()

	if checkErr == nil {
		// Key already exists
		return CopyKeyResponse{
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
		return CopyKeyResponse{
			OK:        false,
			Status:    "error",
			Message:   "Failed to create SSH session",
			Hint:      err.Error(),
			Timestamp: timestamp,
		}
	}
	defer addSession.Close()

	// Create .ssh directory, add key, set permissions
	// The complex condition ensures we add a newline before the key if the file
	// exists but doesn't end with a newline (which would corrupt both keys)
	addCmd := fmt.Sprintf(
		`mkdir -p ~/.ssh && chmod 700 ~/.ssh && { [ ! -f ~/.ssh/authorized_keys ] || [ -z "$(tail -c1 ~/.ssh/authorized_keys)" ] || echo ''; printf '%%s\n' %s; } >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys`,
		QuoteSingle(pubKeyLine),
	)

	if err := addSession.Run(addCmd); err != nil {
		return CopyKeyResponse{
			OK:        false,
			Status:    "error",
			Message:   "Failed to add key to authorized_keys",
			Hint:      err.Error(),
			Timestamp: timestamp,
		}
	}

	return CopyKeyResponse{
		OK:            true,
		Status:        "copied",
		Message:       "SSH key successfully copied to server",
		KeyCopied:     true,
		AlreadyExists: false,
		Timestamp:     timestamp,
	}
}

// DeleteKey deletes an SSH key pair (private and public key files).
func DeleteKey(req DeleteKeyRequest) DeleteKeyResponse {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	keyPath := ExpandPath(req.KeyPath)

	// Validate key path is within ~/.ssh
	if err := ValidateSSHPath(keyPath); err != nil {
		return DeleteKeyResponse{
			OK:        false,
			Message:   fmt.Sprintf("Invalid key path: %s", err.Error()),
			Timestamp: timestamp,
		}
	}

	// Ensure we're not deleting .pub file directly - we want the base key path
	keyPath = strings.TrimSuffix(keyPath, ".pub")

	// Don't allow deleting special files
	baseName := filepath.Base(keyPath)
	if baseName == "authorized_keys" || baseName == "known_hosts" || baseName == "config" {
		return DeleteKeyResponse{
			OK:        false,
			Message:   fmt.Sprintf("Cannot delete special file: %s", baseName),
			Timestamp: timestamp,
		}
	}

	var privateDeleted, publicDeleted bool

	// Delete private key
	if _, err := os.Stat(keyPath); err == nil {
		if err := os.Remove(keyPath); err != nil {
			return DeleteKeyResponse{
				OK:        false,
				Message:   fmt.Sprintf("Failed to delete private key: %s", err.Error()),
				Timestamp: timestamp,
			}
		}
		privateDeleted = true
	}

	// Delete public key
	pubKeyPath := keyPath + ".pub"
	if _, err := os.Stat(pubKeyPath); err == nil {
		if err := os.Remove(pubKeyPath); err != nil {
			return DeleteKeyResponse{
				OK:             false,
				Message:        fmt.Sprintf("Deleted private key but failed to delete public key: %s", err.Error()),
				PrivateDeleted: privateDeleted,
				Timestamp:      timestamp,
			}
		}
		publicDeleted = true
	}

	if !privateDeleted && !publicDeleted {
		return DeleteKeyResponse{
			OK:        false,
			Message:   "Key files not found",
			Timestamp: timestamp,
		}
	}

	return DeleteKeyResponse{
		OK:             true,
		Message:        "SSH key deleted successfully",
		PrivateDeleted: privateDeleted,
		PublicDeleted:  publicDeleted,
		Timestamp:      timestamp,
	}
}
