package receiptsigning

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const RuntimeConfigVersion = "vrooli.receipt-signing-runtime.v1"

// RuntimeConfig is written by the lifecycle/Secrets Manager boundary into the
// scenario config directory. It contains routing and identity-file metadata,
// never a signing key or a Vault token.
type RuntimeConfig struct {
	Version             string                            `json:"version"`
	Mode                string                            `json:"mode"`
	CredentialAuthority *CredentialAuthorityRuntimeConfig `json:"credentialAuthority,omitempty"`
	VaultTransit        *VaultTransitRuntimeConfig        `json:"vaultTransit,omitempty"`
	LegacyVaultTransit  *VaultTransitRuntimeConfig        `json:"legacyVaultTransit,omitempty"`
}

const ModeCredentialAuthorityEd25519 = "credential-authority-ed25519"

type CredentialAuthorityRuntimeConfig struct {
	Identity string `json:"identity"`
	Field    string `json:"field"`
}

type VaultTransitRuntimeConfig struct {
	Address        string `json:"address"`
	KeyName        string `json:"keyName"`
	CredentialFile string `json:"credentialFile"`
}

func (c RuntimeConfig) Validate() error {
	if c.Version != RuntimeConfigVersion {
		return fmt.Errorf("unsupported receipt signing runtime config version %q", c.Version)
	}
	switch c.Mode {
	case "development":
		if c.VaultTransit != nil || c.CredentialAuthority != nil || c.LegacyVaultTransit != nil {
			return fmt.Errorf("development receipt signer must not contain provider configuration")
		}
		return nil
	case ModeCredentialAuthorityEd25519:
		if c.CredentialAuthority == nil {
			return fmt.Errorf("credential authority receipt signer configuration is required")
		}
		if strings.TrimSpace(c.CredentialAuthority.Identity) == "" || strings.TrimSpace(c.CredentialAuthority.Field) == "" {
			return fmt.Errorf("credential authority receipt signer requires identity and field")
		}
		if c.VaultTransit != nil {
			return fmt.Errorf("credential authority receipt signer must not contain active Vault Transit configuration")
		}
		return nil
	case "vault-transit":
		if c.VaultTransit == nil {
			return fmt.Errorf("Vault Transit receipt signer configuration is required")
		}
		if strings.TrimSpace(c.VaultTransit.Address) == "" || strings.TrimSpace(c.VaultTransit.KeyName) == "" || strings.TrimSpace(c.VaultTransit.CredentialFile) == "" {
			return fmt.Errorf("Vault Transit receipt signer requires address, key name, and credential file")
		}
		return nil
	default:
		return fmt.Errorf("unsupported receipt signer mode %q", c.Mode)
	}
}

func LoadRuntimeConfig(path string) (RuntimeConfig, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return RuntimeConfig{}, err
	}
	var config RuntimeConfig
	if err := json.Unmarshal(contents, &config); err != nil {
		return RuntimeConfig{}, fmt.Errorf("parse receipt signing runtime config: %w", err)
	}
	if err := config.Validate(); err != nil {
		return RuntimeConfig{}, err
	}
	return config, nil
}

// FileCredentialSource reads a lifecycle-issued, permission-restricted token
// file at request time. It rejects symlinks and group/world-readable files so
// a deployment mistake cannot silently widen the credential boundary.
type FileCredentialSource struct{ Path string }

func (s FileCredentialSource) Credential(context.Context) (string, error) {
	info, err := os.Lstat(s.Path)
	if err != nil {
		return "", fmt.Errorf("read lifecycle credential metadata: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("lifecycle credential must be a regular file")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("lifecycle credential file permissions must not grant group or world access")
	}
	contents, err := os.ReadFile(s.Path)
	if err != nil {
		return "", fmt.Errorf("read lifecycle credential: %w", err)
	}
	credential := strings.TrimSpace(string(contents))
	if credential == "" {
		return "", fmt.Errorf("lifecycle credential is empty")
	}
	return credential, nil
}

func NewSignerFromRuntimeConfig(config RuntimeConfig) (ReceiptSigner, bool, error) {
	if err := config.Validate(); err != nil {
		return nil, false, err
	}
	if config.Mode == "development" {
		return NewDevelopmentSigner(), false, nil
	}
	if config.Mode == ModeCredentialAuthorityEd25519 {
		return nil, true, fmt.Errorf("credential authority signer must be constructed by its credential-authority runtime binding")
	}
	transit := config.VaultTransit
	signer, err := NewVaultTransitSigner(VaultTransitConfig{Address: transit.Address, KeyName: transit.KeyName, Credentials: FileCredentialSource{Path: filepath.Clean(transit.CredentialFile)}, AllowedPurposes: []Purpose{PurposeExperimentAuditReceipt, PurposeExperimentHoldoutReceipt}})
	if err != nil {
		return nil, false, err
	}
	return signer, true, nil
}
