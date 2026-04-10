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
	sshDir, err := resolveSSHDir(platform, sshDir)
	if err != nil {
		return nil, err
	}

	entries, err := readSSHDirEntries(sshDir)
	if err != nil || entries == nil {
		return []KeyInfo{}, err
	}

	return collectKeys(platform, sshDir, entries), nil
}

// resolveSSHDir returns the SSH directory, using the platform default if empty.
func resolveSSHDir(platform Platform, sshDir string) (string, error) {
	if sshDir != "" {
		return sshDir, nil
	}
	return platform.GetSSHDir()
}

// readSSHDirEntries reads directory entries from the SSH directory.
// Returns nil entries (no error) if the directory does not exist.
func readSSHDirEntries(sshDir string) ([]os.DirEntry, error) {
	info, err := os.Stat(sshDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot access SSH directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("~/.ssh is not a directory")
	}
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read SSH directory: %w", err)
	}
	return entries, nil
}

// isKeyCandidate returns true if the directory entry should be considered as a potential SSH key.
func isKeyCandidate(entry os.DirEntry) bool {
	if entry.IsDir() {
		return false
	}
	name := entry.Name()
	return !strings.HasSuffix(name, ".pub") && !IsProtectedFile(name)
}

// collectKeys parses all key candidates from the given directory entries.
func collectKeys(platform Platform, sshDir string, entries []os.DirEntry) []KeyInfo {
	keys := []KeyInfo{}
	for _, entry := range entries {
		if !isKeyCandidate(entry) {
			continue
		}
		keyInfo, err := parseKeyFile(platform, filepath.Join(sshDir, entry.Name()))
		if err != nil {
			continue
		}
		keys = append(keys, keyInfo)
	}
	return keys
}

// parseKeyFile extracts information from an SSH key file.
func parseKeyFile(platform Platform, keyPath string) (KeyInfo, error) {
	firstLine, err := readFirstLine(keyPath)
	if err != nil {
		return KeyInfo{}, err
	}

	if !strings.HasPrefix(firstLine, "-----BEGIN") {
		return KeyInfo{}, fmt.Errorf("not a PEM-encoded key")
	}

	keyType := keyTypeFromPEMHeader(firstLine)
	pubKeyPath := keyPath + ".pub"
	hasPublic := fileExists(pubKeyPath)

	fingerprint, keyTypeFromFP, bits, comment := getKeyFingerprint(platform, pubKeyPath)
	if keyType == KeyTypeUnknown && keyTypeFromFP != KeyTypeUnknown {
		keyType = keyTypeFromFP
	}

	return KeyInfo{
		Path:        keyPath,
		Filename:    filepath.Base(keyPath),
		Type:        keyType,
		Bits:        bits,
		Fingerprint: fingerprint,
		Comment:     comment,
		CreatedAt:   fileModTime(keyPath),
		HasPublic:   hasPublic,
	}, nil
}

// readFirstLine reads the first line of a file.
func readFirstLine(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return "", fmt.Errorf("empty file")
	}
	return scanner.Text(), nil
}

// keyTypeFromPEMHeader determines the key type from a PEM header line.
func keyTypeFromPEMHeader(header string) KeyType {
	switch {
	case strings.Contains(header, "RSA PRIVATE KEY"):
		return KeyTypeRSA
	case strings.Contains(header, "DSA PRIVATE KEY"):
		return KeyTypeDSA
	case strings.Contains(header, "EC PRIVATE KEY"):
		return KeyTypeECDSA
	default:
		return KeyTypeUnknown
	}
}

// fileExists returns true if the file at path exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// fileModTime returns the modification time of a file as an RFC3339 string, or empty on error.
func fileModTime(path string) string {
	stat, err := os.Stat(path)
	if err != nil {
		return ""
	}
	return stat.ModTime().Format(time.RFC3339)
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
