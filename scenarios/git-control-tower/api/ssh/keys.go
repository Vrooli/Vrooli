package ssh

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DiscoverKeys scans the SSH directory for SSH key files.
func DiscoverKeys(platform Platform, sshDir string) ([]KeyInfo, error) {
	if sshDir == "" {
		var err error
		sshDir, err = platform.GetSSHDir()
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
			IsProtectedFile(name) {
			continue
		}

		keyPath := filepath.Join(sshDir, name)
		keyInfo, err := parseKeyFile(platform, keyPath)
		if err != nil {
			// Skip files that aren't valid SSH keys
			continue
		}
		keys = append(keys, keyInfo)
	}

	return keys, nil
}

// parseKeyFile extracts information from an SSH key file.
func parseKeyFile(platform Platform, keyPath string) (KeyInfo, error) {
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

	// Check if public key exists
	pubKeyPath := keyPath + ".pub"
	hasPublic := false
	if _, err := os.Stat(pubKeyPath); err == nil {
		hasPublic = true
	}

	// Get fingerprint and more info using ssh-keygen
	fingerprint, keyTypeFromFP, bits, comment := getKeyFingerprint(platform, pubKeyPath)

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
		Filename:    filepath.Base(keyPath),
		Type:        keyType,
		Bits:        bits,
		Fingerprint: fingerprint,
		Comment:     comment,
		CreatedAt:   createdAt,
		HasPublic:   hasPublic,
	}, nil
}

// getKeyFingerprint uses ssh-keygen to get the fingerprint of a public key.
func getKeyFingerprint(platform Platform, pubKeyPath string) (fingerprint string, keyType KeyType, bits int, comment string) {
	cmd := exec.Command(platform.SSHKeygenPath(), "-lf", pubKeyPath)
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
func ReadPublicKey(platform Platform, keyPath string) (publicKey, fingerprint string, err error) {
	keyPath = ExpandPath(platform, keyPath)

	// Validate path
	if err := ValidateSSHPath(platform, keyPath); err != nil {
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
	fp, _, _, _ := getKeyFingerprint(platform, pubKeyPath)
	fingerprint = fp

	return publicKey, fingerprint, nil
}
