// DOC: docs/concepts/architecture.md#key-management-lifecycle — key lifecycle diagram
package ssh

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// KeyService manages SSH key operations with explicit dependencies.
type KeyService struct {
	cmd    CommandRunner
	sshDir string
}

// NewKeyService creates a KeyService. If cmd is nil, ExecCommandRunner is used.
func NewKeyService(cmd CommandRunner, sshDir string) *KeyService {
	if cmd == nil {
		cmd = ExecCommandRunner{}
	}
	return &KeyService{cmd: cmd, sshDir: sshDir}
}

// getSSHDir returns the configured SSH directory or falls back to ~/.ssh.
func (ks *KeyService) getSSHDir() (string, error) {
	if ks.sshDir != "" {
		return ks.sshDir, nil
	}
	return GetSSHDir()
}

// pubKeyPath returns the public key path for the given key path.
// If the path already ends with ".pub", it is returned as-is.
func pubKeyPath(keyPath string) string {
	if strings.HasSuffix(keyPath, ".pub") {
		return keyPath
	}
	return keyPath + ".pub"
}

// DiscoverKeys scans the SSH directory for SSH key files.
func (ks *KeyService) DiscoverKeys() ([]KeyInfo, error) {
	sshDir, err := ks.getSSHDir()
	if err != nil {
		return nil, err
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
	skipped := 0
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
		keyInfo, err := ks.parseKeyFile(keyPath)
		if err != nil {
			// Skip files that aren't valid SSH keys
			slog.Debug("ssh.key_skipped", "path", name, "reason", err.Error())
			skipped++
			continue
		}
		keys = append(keys, keyInfo)
	}

	slog.Debug("ssh.key_discovery", "keys_found", len(keys), "files_skipped", skipped)

	return keys, nil
}

// parseKeyFile extracts information from an SSH key file.
func (ks *KeyService) parseKeyFile(keyPath string) (KeyInfo, error) {
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
	fingerprint, keyTypeFromFP, bits, comment := ks.getKeyFingerprint(pubKeyPath(keyPath))

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
func (ks *KeyService) getKeyFingerprint(pubKeyPath string) (fingerprint string, keyType KeyType, bits int, comment string) {
	output, _, err := ks.cmd.Run(context.Background(), "ssh-keygen", "-lf", pubKeyPath)
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
func (ks *KeyService) ReadPublicKey(keyPath string) (publicKey, fingerprint string, err error) {
	keyPath = ExpandPath(keyPath)

	// Validate path
	if err := ValidateSSHPath(keyPath); err != nil {
		return "", "", err
	}

	pkPath := pubKeyPath(keyPath)

	// Read public key file
	content, err := os.ReadFile(pkPath)
	if err != nil {
		return "", "", fmt.Errorf("cannot read public key: %w", err)
	}

	publicKey = strings.TrimSpace(string(content))

	// Get fingerprint
	fp, _, _, _ := ks.getKeyFingerprint(pkPath)
	fingerprint = fp

	return publicKey, fingerprint, nil
}

// DeleteKey deletes an SSH key pair (private and public key files).
func (ks *KeyService) DeleteKey(req DeleteKeyRequest) DeleteKeyResponse {
	timestamp := nowTimestamp()

	keyPath := ExpandPath(req.KeyPath)

	// Validate key path is within ~/.ssh
	if err := ValidateSSHPath(keyPath); err != nil {
		return DeleteKeyResponse{
			Outcome: Outcome{
				OK:        false,
				Status:    StatusInvalidInput,
				Message:   fmt.Sprintf("Invalid key path: %s", err.Error()),
				Timestamp: timestamp,
			},
		}
	}

	// Ensure we're not deleting .pub file directly - we want the base key path
	keyPath = strings.TrimSuffix(keyPath, ".pub")

	// Don't allow deleting special files
	baseName := filepath.Base(keyPath)
	if baseName == "authorized_keys" || baseName == "known_hosts" || baseName == "config" {
		return DeleteKeyResponse{
			Outcome: Outcome{
				OK:        false,
				Status:    StatusInvalidInput,
				Message:   fmt.Sprintf("Cannot delete special file: %s", baseName),
				Hint:      "authorized_keys, known_hosts, and config are protected system files",
				Timestamp: timestamp,
			},
		}
	}

	var privateDeleted, publicDeleted bool

	// Delete private key
	if _, err := os.Stat(keyPath); err == nil {
		if err := os.Remove(keyPath); err != nil {
			return DeleteKeyResponse{
				Outcome: Outcome{
					OK:        false,
					Status:    StatusError,
					Message:   fmt.Sprintf("Failed to delete private key: %s", err.Error()),
					Timestamp: timestamp,
				},
			}
		}
		privateDeleted = true
	}

	// Delete public key
	pubKeyPath := keyPath + ".pub"
	if _, err := os.Stat(pubKeyPath); err == nil {
		if err := os.Remove(pubKeyPath); err != nil {
			return DeleteKeyResponse{
				Outcome: Outcome{
					OK:        false,
					Status:    StatusError,
					Message:   fmt.Sprintf("Deleted private key but failed to delete public key: %s", err.Error()),
					Timestamp: timestamp,
				},
				PrivateDeleted: privateDeleted,
			}
		}
		publicDeleted = true
	}

	if !privateDeleted && !publicDeleted {
		return DeleteKeyResponse{
			Outcome: Outcome{
				OK:        false,
				Status:    StatusNotFound,
				Message:   "Key files not found",
				Timestamp: timestamp,
			},
		}
	}

	slog.Info("ssh.key_deleted", "filename", filepath.Base(keyPath))

	return DeleteKeyResponse{
		Outcome: Outcome{
			OK:        true,
			Status:    StatusSuccess,
			Message:   "SSH key deleted successfully",
			Timestamp: timestamp,
		},
		PrivateDeleted: privateDeleted,
		PublicDeleted:  publicDeleted,
	}
}
