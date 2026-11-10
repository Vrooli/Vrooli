package orchestration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	runidentity "agent-manager/internal/identity"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/scopecatalog"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const (
	treasurySessionIdentityPrefix = "vrooli/agent-manager-run-"
	treasurySessionKeyField       = "treasury-agent-session-key"
	treasurySessionKeyEnv         = "VROOLI_TREASURY_SESSION_PRIVATE_KEY"
)

func resolveTreasurySessionKey(ctx context.Context, store TreasurySessionKeyStore, runID uuid.UUID, identityToken string, identitySecret []byte) (string, error) {
	if strings.TrimSpace(identityToken) == "" || len(identitySecret) == 0 {
		return "", nil
	}
	claims, err := runidentity.VerifyToken(identityToken, identitySecret)
	if err != nil {
		return "", fmt.Errorf("verify run identity before Treasury session-key injection: %w", err)
	}
	if !scopecatalog.MatchCapability(claims.Scopes, "treasury:write") {
		return "", nil
	}
	if store == nil {
		return "", fmt.Errorf("run has treasury:write capability but the run-owned Treasury session-key store is unavailable")
	}
	key, err := store.ResolveOrMint(ctx, runID)
	if err != nil {
		return "", fmt.Errorf("provision run-owned Treasury session key: %w", err)
	}
	if len(key) != 64 {
		return "", fmt.Errorf("run-owned Treasury session key has invalid encoding")
	}
	if _, err := hex.DecodeString(key); err != nil {
		return "", fmt.Errorf("run-owned Treasury session key has invalid encoding")
	}
	return strings.ToLower(key), nil
}

// CredentialAuthorityTreasurySessionKeys keeps run signing keys in the native
// credential authority. Only the scoped runner process receives the value.
// Run rows and user-controlled CustomEnv never contain private-key material.
type CredentialAuthorityTreasurySessionKeys struct {
	open func() (*credentialauthority.Authority, error)
}

func NewCredentialAuthorityTreasurySessionKeys() *CredentialAuthorityTreasurySessionKeys {
	return &CredentialAuthorityTreasurySessionKeys{open: credentialauthority.Default}
}

func (s *CredentialAuthorityTreasurySessionKeys) ResolveOrMint(ctx context.Context, runID uuid.UUID) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if s == nil || s.open == nil || runID == uuid.Nil {
		return "", fmt.Errorf("Treasury session-key authority and run ID are required")
	}
	identity, err := credentialauthority.ParseIdentity(treasurySessionIdentityPrefix + runID.String())
	if err != nil {
		return "", fmt.Errorf("parse run-scoped Treasury key identity: %w", err)
	}
	authority, err := s.open()
	if err != nil {
		return "", fmt.Errorf("open credential authority for run-scoped Treasury key: %w", err)
	}
	value, err := authority.ResolveOrMint(identity, treasurySessionKeyField, nil, func() (string, error) {
		secret := make([]byte, 32)
		if _, readErr := rand.Read(secret); readErr != nil {
			return "", fmt.Errorf("generate run-scoped Treasury key: %w", readErr)
		}
		return hex.EncodeToString(secret), nil
	})
	if err != nil {
		return "", fmt.Errorf("resolve run-scoped Treasury key: %w", err)
	}
	value = strings.TrimSpace(value)
	if len(value) != 64 {
		return "", fmt.Errorf("run-scoped Treasury key has invalid encoding")
	}
	if _, err := hex.DecodeString(value); err != nil {
		return "", fmt.Errorf("run-scoped Treasury key has invalid encoding")
	}
	return value, nil
}

func (s *CredentialAuthorityTreasurySessionKeys) RevokeRunCredentialUse(ctx context.Context, runID uuid.UUID, _ string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.open == nil || runID == uuid.Nil {
		return fmt.Errorf("Treasury session-key authority and run ID are required")
	}
	identity, err := credentialauthority.ParseIdentity(treasurySessionIdentityPrefix + runID.String())
	if err != nil {
		return fmt.Errorf("parse run-scoped Treasury key identity: %w", err)
	}
	authority, err := s.open()
	if err != nil {
		return fmt.Errorf("open credential authority to revoke run-scoped Treasury key: %w", err)
	}
	if err := authority.Delete(identity, treasurySessionKeyField); err != nil {
		return fmt.Errorf("revoke run-scoped Treasury key: %w", err)
	}
	return nil
}
