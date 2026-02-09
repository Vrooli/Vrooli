// Package sshidentity defines the canonical SSH identity model used across
// bootstrap, deploy, and health flows.
// DOC: docs/reference/configuration.md#canonical-ssh-identity-model
package sshidentity

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// AuthMode describes how SSH authentication is performed.
type AuthMode string

const (
	AuthModeExplicitKey AuthMode = "explicit_key"
	AuthModeAgent       AuthMode = "agent"
	AuthModeDefaultSSH  AuthMode = "default_ssh"
	AuthModeUnknown     AuthMode = "unknown"
)

// VerificationState describes whether explicit key authorization is confirmed.
type VerificationState string

const (
	VerificationAuthorized   VerificationState = "authorized"
	VerificationUnauthorized VerificationState = "unauthorized"
	VerificationUnknown      VerificationState = "unknown"
)

// DeploymentSSHIdentity is the canonical persisted identity state for a deployment.
type DeploymentSSHIdentity struct {
	KeyPath              string            `json:"key_path,omitempty"`
	PublicKeyFingerprint string            `json:"public_key_fingerprint,omitempty"`
	AuthMode             AuthMode          `json:"auth_mode"`
	VerificationState    VerificationState `json:"verification_state"`
	LastVerifiedAt       string            `json:"last_verified_at,omitempty"`
}

// Normalize validates and normalizes the identity in place.
func (d *DeploymentSSHIdentity) Normalize() error {
	if d == nil {
		return errors.New("identity is nil")
	}
	d.KeyPath = strings.TrimSpace(d.KeyPath)
	d.PublicKeyFingerprint = strings.TrimSpace(d.PublicKeyFingerprint)
	if d.AuthMode == "" {
		d.AuthMode = AuthModeUnknown
	}
	if d.VerificationState == "" {
		d.VerificationState = VerificationUnknown
	}

	switch d.AuthMode {
	case AuthModeExplicitKey, AuthModeAgent, AuthModeDefaultSSH, AuthModeUnknown:
	default:
		return fmt.Errorf("invalid auth_mode %q", d.AuthMode)
	}
	switch d.VerificationState {
	case VerificationAuthorized, VerificationUnauthorized, VerificationUnknown:
	default:
		return fmt.Errorf("invalid verification_state %q", d.VerificationState)
	}
	if d.AuthMode == AuthModeExplicitKey && d.KeyPath == "" {
		return errors.New("explicit_key auth_mode requires key_path")
	}
	if d.LastVerifiedAt != "" {
		if _, err := time.Parse(time.RFC3339, d.LastVerifiedAt); err != nil {
			return fmt.Errorf("invalid last_verified_at %q: %w", d.LastVerifiedAt, err)
		}
	}
	return nil
}

// Clone returns a deep copy of the identity.
func (d DeploymentSSHIdentity) Clone() DeploymentSSHIdentity {
	return DeploymentSSHIdentity{
		KeyPath:              d.KeyPath,
		PublicKeyFingerprint: d.PublicKeyFingerprint,
		AuthMode:             d.AuthMode,
		VerificationState:    d.VerificationState,
		LastVerifiedAt:       d.LastVerifiedAt,
	}
}

// Marshal marshals an identity into canonical JSON bytes.
func Marshal(identity DeploymentSSHIdentity) ([]byte, error) {
	if err := identity.Normalize(); err != nil {
		return nil, err
	}
	return json.Marshal(identity)
}

// Unmarshal parses canonical identity JSON.
func Unmarshal(data []byte) (DeploymentSSHIdentity, error) {
	var id DeploymentSSHIdentity
	if len(data) == 0 || string(data) == "null" {
		id.AuthMode = AuthModeUnknown
		id.VerificationState = VerificationUnknown
		return id, nil
	}
	if err := json.Unmarshal(data, &id); err != nil {
		return DeploymentSSHIdentity{}, err
	}
	if err := id.Normalize(); err != nil {
		return DeploymentSSHIdentity{}, err
	}
	return id, nil
}
