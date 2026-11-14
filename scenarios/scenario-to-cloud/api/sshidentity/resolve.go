package sshidentity

import (
	"os"
	"strings"
	"time"
)

// Resolver defines the seam for canonical identity resolution.
// DOC: docs/internal/SEAMS.md#ssh-identity-seams
type Resolver interface {
	Resolve(boundKeyPath string, existing *DeploymentSSHIdentity) (DeploymentSSHIdentity, error)
}

// DefaultResolver applies canonical SSH identity precedence.
type DefaultResolver struct{}

// Resolve determines the canonical identity with this precedence:
// 1) the key file the deployment's credential binding names (boundKeyPath)
// 2) explicit key from persisted identity (if present)
// 3) ambient SSH transport (agent/default ssh)
func (DefaultResolver) Resolve(boundKeyPath string, existing *DeploymentSSHIdentity) (DeploymentSSHIdentity, error) {
	resolved := DeploymentSSHIdentity{
		AuthMode:          AuthModeUnknown,
		VerificationState: VerificationUnknown,
	}

	if boundKey := strings.TrimSpace(boundKeyPath); boundKey != "" {
		resolved.AuthMode = AuthModeExplicitKey
		resolved.KeyPath = boundKey
		_, fp, err := ReadPublicKeyAndFingerprint(boundKey)
		if err == nil {
			resolved.PublicKeyFingerprint = fp
		}
		return resolved, resolved.Normalize()
	}

	if existing != nil {
		candidate := existing.Clone()
		if err := candidate.Normalize(); err == nil && candidate.AuthMode == AuthModeExplicitKey && strings.TrimSpace(candidate.KeyPath) != "" {
			if _, statErr := os.Stat(candidate.KeyPath); statErr == nil {
				candidate.VerificationState = VerificationUnknown
				candidate.LastVerifiedAt = ""
				return candidate, nil
			}
		}
	}

	resolved.AuthMode = detectAmbientAuthMode()
	return resolved, resolved.Normalize()
}

func detectAmbientAuthMode() AuthMode {
	if strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK")) != "" {
		return AuthModeAgent
	}
	return AuthModeDefaultSSH
}

// ApplyVerificationResult stamps verification status and timestamp onto identity.
func ApplyVerificationResult(identity DeploymentSSHIdentity, state VerificationState, verifiedAt time.Time) DeploymentSSHIdentity {
	updated := identity.Clone()
	updated.VerificationState = state
	if !verifiedAt.IsZero() {
		updated.LastVerifiedAt = verifiedAt.UTC().Format(time.RFC3339)
	}
	if updated.AuthMode != AuthModeExplicitKey && state != VerificationUnknown {
		// Non-explicit auth cannot be directly matched in authorized_keys.
		updated.VerificationState = VerificationUnknown
	}
	return updated
}
