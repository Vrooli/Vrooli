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

// ValidateKeyFilename ensures a key filename is a safe basename.
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
	c := filename[0]
	if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
		return fmt.Errorf("filename must start with an alphanumeric character or underscore")
	}
	return nil
}

// pubKeyPath returns the public-key path for a private-key path.
func pubKeyPath(keyPath string) string {
	if strings.HasSuffix(keyPath, ".pub") {
		return keyPath
	}
	return keyPath + ".pub"
}

// GenerateKey generates a new SSH key pair under the bridge state dir. The
// private key is written 0600 and the state dir 0700; an existing key is left
// untouched (callers treat "already exists" as a signal, not an error).
func (s *Service) GenerateKey(req GenerateKeyRequest) (KeyInfo, error) {
	if req.Type == "" {
		req.Type = KeyTypeEd25519
	}
	if req.Type != KeyTypeEd25519 && req.Type != KeyTypeRSA {
		return KeyInfo{}, fmt.Errorf("key type must be 'ed25519' or 'rsa'")
	}
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
	if err := ValidateKeyFilename(req.Filename); err != nil {
		return KeyInfo{}, err
	}

	if err := ensureDir0700(s.stateDir); err != nil {
		return KeyInfo{}, fmt.Errorf("cannot create SSH state dir: %w", err)
	}

	keyPath := filepath.Join(s.stateDir, req.Filename)
	if _, err := os.Stat(keyPath); err == nil {
		return KeyInfo{}, fmt.Errorf("key already exists: %s", keyPath)
	}

	args := []string{"-t", string(req.Type)}
	if req.Type == KeyTypeRSA && req.Bits > 0 {
		args = append(args, "-b", fmt.Sprintf("%d", req.Bits))
	}
	args = append(args, "-f", keyPath)
	if req.Comment != "" {
		args = append(args, "-C", req.Comment)
	} else {
		args = append(args, "-C", "generated-by-vrooli-bridge")
	}
	// Empty passphrase: the private key is protected by 0600 file perms under
	// the server-owned state dir, matching the control-plane identity key.
	args = append(args, "-N", req.Password)

	_, stderr, err := s.cmd.Run(context.Background(), "ssh-keygen", args...)
	if err != nil {
		return KeyInfo{}, fmt.Errorf("ssh-keygen failed: %s", string(stderr))
	}

	// ssh-keygen already writes 0600, but assert the invariant explicitly so a
	// stricter umask or a fake CommandRunner cannot leave a wider mode.
	if err := os.Chmod(keyPath, 0o600); err != nil {
		return KeyInfo{}, fmt.Errorf("secure private key perms: %w", err)
	}

	slog.Info("ssh.key_generated", "key_type", string(req.Type), "filename", req.Filename)

	return s.parseKeyFile(keyPath)
}

// ReadPublicKey reads the public key content and fingerprint for keyPath.
func (s *Service) ReadPublicKey(keyPath string) (publicKey, fingerprint string, err error) {
	pkPath := pubKeyPath(keyPath)
	content, err := os.ReadFile(pkPath)
	if err != nil {
		return "", "", fmt.Errorf("cannot read public key: %w", err)
	}
	publicKey = strings.TrimSpace(string(content))
	fp, _, _, _ := s.getKeyFingerprint(pkPath)
	return publicKey, fp, nil
}

// parseKeyFile extracts KeyInfo from a private-key file.
func (s *Service) parseKeyFile(keyPath string) (KeyInfo, error) {
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

	keyType := KeyTypeUnknown
	switch {
	case strings.Contains(firstLine, "RSA PRIVATE KEY"):
		keyType = KeyTypeRSA
	case strings.Contains(firstLine, "DSA PRIVATE KEY"):
		keyType = KeyTypeDSA
	case strings.Contains(firstLine, "EC PRIVATE KEY"):
		keyType = KeyTypeECDSA
	}

	fingerprint, keyTypeFromFP, bits, comment := s.getKeyFingerprint(pubKeyPath(keyPath))
	if keyType == KeyTypeUnknown && keyTypeFromFP != KeyTypeUnknown {
		keyType = keyTypeFromFP
	}

	createdAt := ""
	if stat, err := os.Stat(keyPath); err == nil {
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

// getKeyFingerprint runs ssh-keygen -lf to read a public key's fingerprint.
func (s *Service) getKeyFingerprint(pubKeyPath string) (fingerprint string, keyType KeyType, bits int, comment string) {
	output, _, err := s.cmd.Run(context.Background(), "ssh-keygen", "-lf", pubKeyPath)
	if err != nil {
		return "", KeyTypeUnknown, 0, ""
	}

	// Output format: "256 SHA256:xxxx comment (ED25519)"
	line := strings.TrimSpace(string(output))
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return "", KeyTypeUnknown, 0, ""
	}

	if _, err := fmt.Sscanf(parts[0], "%d", &bits); err != nil {
		bits = 0
	}
	fingerprint = parts[1]

	lastField := strings.Trim(parts[len(parts)-1], "()")
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

	if len(parts) > 3 {
		comment = strings.Join(parts[2:len(parts)-1], " ")
	}

	return fingerprint, keyType, bits, comment
}
