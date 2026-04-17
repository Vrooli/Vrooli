package sshidentity

import (
	"os"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/ssh"
	"strings"
	"time"
)

// Resolver defines the seam for canonical identity resolution.
// DOC: docs/internal/SEAMS.md#ssh-identity-seams
type Resolver interface {
	Resolve(manifest domain.CloudManifest, existing *DeploymentSSHIdentity) (DeploymentSSHIdentity, error)
}

// DefaultResolver applies canonical SSH identity precedence.
type DefaultResolver struct{}

// Resolve determines the canonical identity with this precedence:
// 1) explicit key in manifest
// 2) explicit key from persisted identity (if present)
// 3) ambient SSH transport (agent/default ssh)
func (DefaultResolver) Resolve(manifest domain.CloudManifest, existing *DeploymentSSHIdentity) (DeploymentSSHIdentity, error) {
	resolved := DeploymentSSHIdentity{
		AuthMode:          AuthModeUnknown,
		VerificationState: VerificationUnknown,
	}

	manifestKey := ""
	if manifest.Target.VPS != nil {
		manifestKey = strings.TrimSpace(manifest.Target.VPS.KeyPath)
	}

	if manifestKey != "" {
		resolved.AuthMode = AuthModeExplicitKey
		resolved.KeyPath = manifestKey
		_, fp, err := ReadPublicKeyAndFingerprint(manifestKey)
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

// ApplyToManifest writes the canonical identity into SSH manifest config for command execution.
func ApplyToManifest(manifest domain.CloudManifest, identity DeploymentSSHIdentity) domain.CloudManifest {
	m := manifest
	if m.Target.VPS == nil {
		return m
	}
	if identity.AuthMode == AuthModeExplicitKey {
		m.Target.VPS.KeyPath = identity.KeyPath
	} else {
		m.Target.VPS.KeyPath = ""
	}
	return m
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

// EffectiveSSHConfig returns the SSH config derived from manifest + identity.
func EffectiveSSHConfig(manifest domain.CloudManifest, identity DeploymentSSHIdentity) ssh.Config {
	m := ApplyToManifest(manifest, identity)
	return ssh.ConfigFromManifest(m)
}
