package ssh

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SSHDeps contains dependencies for SSH operations.
type SSHDeps struct {
	Platform Platform
}

// ListKeys returns all SSH keys in ~/.ssh.
func ListKeys(_ context.Context, deps SSHDeps) (*ListKeysResponse, error) {
	sshDir, err := deps.Platform.GetSSHDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine SSH directory: %w", err)
	}

	keys, err := DiscoverKeys(deps.Platform, sshDir)
	if err != nil {
		return nil, fmt.Errorf("failed to discover SSH keys: %w", err)
	}

	return &ListKeysResponse{
		Keys:      keys,
		SSHDir:    sshDir,
		Timestamp: timestamp(),
	}, nil
}

// GenerateKeyService generates a new SSH key pair.
func GenerateKeyService(_ context.Context, deps SSHDeps, req GenerateKeyRequest) (*GenerateKeyResponse, error) {
	// Validate type
	if req.Type != KeyTypeEd25519 && req.Type != KeyTypeRSA {
		return &GenerateKeyResponse{
			Success:   false,
			Error:     "Key type must be 'ed25519' or 'rsa'",
			Timestamp: timestamp(),
		}, nil
	}

	// Set defaults
	if req.Type == KeyTypeRSA && req.Bits == 0 {
		req.Bits = 4096
	}

	keyInfo, publicKey, err := GenerateKey(deps.Platform, req)
	if err != nil {
		return &GenerateKeyResponse{
			Success:   false,
			Error:     err.Error(),
			Timestamp: timestamp(),
		}, nil
	}

	return &GenerateKeyResponse{
		Success:   true,
		Key:       keyInfo,
		PublicKey: strings.TrimSpace(publicKey),
		Timestamp: timestamp(),
	}, nil
}

// GetPublicKeyService retrieves the public key content for a given key path.
func GetPublicKeyService(_ context.Context, deps SSHDeps, req GetPublicKeyRequest) (*GetPublicKeyResponse, error) {
	if req.KeyPath == "" {
		return &GetPublicKeyResponse{
			Success:   false,
			Error:     "key_path is required",
			Timestamp: timestamp(),
		}, nil
	}

	publicKey, fingerprint, err := ReadPublicKey(deps.Platform, req.KeyPath)
	if err != nil {
		return &GetPublicKeyResponse{
			Success:   false,
			Error:     err.Error(),
			Timestamp: timestamp(),
		}, nil
	}

	return &GetPublicKeyResponse{
		Success:     true,
		PublicKey:   publicKey,
		Fingerprint: fingerprint,
		Timestamp:   timestamp(),
	}, nil
}

// TestGitHubConnectionService tests SSH connection to GitHub.
func TestGitHubConnectionService(ctx context.Context, deps SSHDeps, req TestConnectionRequest) (*TestConnectionResponse, error) {
	if req.KeyPath == "" {
		resp := TestConnectionResponse{
			Success:   false,
			Status:    "missing_key_path",
			Message:   "key_path is required",
			Timestamp: timestamp(),
		}
		return &resp, nil
	}

	result := TestGitHubConnection(ctx, deps.Platform, req.KeyPath)
	return &result, nil
}

// DeleteKeyService deletes an SSH key pair.
func DeleteKeyService(_ context.Context, deps SSHDeps, req DeleteKeyRequest) (*DeleteKeyResponse, error) {
	if req.KeyPath == "" {
		return &DeleteKeyResponse{
			Success:   false,
			Error:     "key_path is required",
			Timestamp: timestamp(),
		}, nil
	}

	keyPath := ExpandPath(deps.Platform, req.KeyPath)

	// Validate key path is within ~/.ssh
	if err := ValidateSSHPath(deps.Platform, keyPath); err != nil {
		return &DeleteKeyResponse{
			Success:   false,
			Error:     fmt.Sprintf("Invalid key path: %s", err.Error()),
			Timestamp: timestamp(),
		}, nil
	}

	// Ensure we're not deleting .pub file directly - we want the base key path
	keyPath = strings.TrimSuffix(keyPath, ".pub")

	// Don't allow deleting special files
	baseName := filepath.Base(keyPath)
	if IsProtectedFile(baseName) {
		return &DeleteKeyResponse{
			Success:   false,
			Error:     fmt.Sprintf("Cannot delete protected file: %s", baseName),
			Timestamp: timestamp(),
		}, nil
	}

	var privateDeleted, publicDeleted bool

	// Delete private key
	if _, err := os.Stat(keyPath); err == nil {
		if err := os.Remove(keyPath); err != nil {
			return &DeleteKeyResponse{
				Success:   false,
				Error:     fmt.Sprintf("Failed to delete private key: %s", err.Error()),
				Timestamp: timestamp(),
			}, nil
		}
		privateDeleted = true
	}

	// Delete public key
	pubKeyPath := keyPath + ".pub"
	if _, err := os.Stat(pubKeyPath); err == nil {
		if err := os.Remove(pubKeyPath); err != nil {
			return &DeleteKeyResponse{
				Success:        false,
				Message:        "Deleted private key but failed to delete public key",
				Error:          err.Error(),
				PrivateDeleted: privateDeleted,
				Timestamp:      timestamp(),
			}, nil
		}
		publicDeleted = true
	}

	if !privateDeleted && !publicDeleted {
		return &DeleteKeyResponse{
			Success:   false,
			Error:     "Key files not found",
			Timestamp: timestamp(),
		}, nil
	}

	return &DeleteKeyResponse{
		Success:        true,
		Message:        "SSH key deleted successfully",
		PrivateDeleted: privateDeleted,
		PublicDeleted:  publicDeleted,
		Timestamp:      timestamp(),
	}, nil
}
