package sshidentity

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"scenario-to-cloud/ssh"
)

// SSHRunner seam for authorized-keys inspection.
type SSHRunner interface {
	Run(ctx context.Context, cfg ssh.Config, command string, opts ssh.RunOptions) (ssh.Result, error)
}

// AuthorizedKeysInspector seam for matching explicit keys on remote host.
type AuthorizedKeysInspector interface {
	Inspect(ctx context.Context, cfg ssh.Config, identity DeploymentSSHIdentity) (VerificationState, error)
}

// RemoteAuthorizedKeysInspector inspects ~/.ssh/authorized_keys over SSH.
type RemoteAuthorizedKeysInspector struct {
	Runner SSHRunner
}

// Inspect verifies whether the explicit key is authorized remotely.
func (i RemoteAuthorizedKeysInspector) Inspect(ctx context.Context, cfg ssh.Config, identity DeploymentSSHIdentity) (VerificationState, error) {
	if identity.AuthMode != AuthModeExplicitKey || strings.TrimSpace(identity.KeyPath) == "" {
		return VerificationUnknown, nil
	}
	publicKey, _, err := ReadPublicKeyAndFingerprint(identity.KeyPath)
	if err != nil {
		return VerificationUnknown, err
	}
	pubParts := strings.Fields(publicKey)
	if len(pubParts) < 2 {
		return VerificationUnknown, nil
	}
	needle := pubParts[0] + " " + pubParts[1]

	res, err := i.Runner.Run(ctx, cfg, "cat ~/.ssh/authorized_keys 2>/dev/null || echo ''", ssh.DefaultRunOptions())
	if err != nil {
		return VerificationUnknown, err
	}
	if strings.Contains(res.Stdout, needle) {
		return VerificationAuthorized, nil
	}
	return VerificationUnauthorized, nil
}

// ReadPublicKeyAndFingerprint returns the .pub content and fingerprint for a private key path.
func ReadPublicKeyAndFingerprint(privateKeyPath string) (publicKey string, fingerprint string, err error) {
	keyPath := expandHome(strings.TrimSpace(privateKeyPath))
	if keyPath == "" {
		return "", "", os.ErrNotExist
	}
	pubPath := keyPath + ".pub"
	content, readErr := os.ReadFile(pubPath)
	if readErr != nil {
		return "", "", readErr
	}
	publicKey = strings.TrimSpace(string(content))
	if publicKey == "" {
		return "", "", os.ErrInvalid
	}

	out, runErr := exec.Command("ssh-keygen", "-lf", pubPath).Output()
	if runErr != nil {
		return publicKey, "", nil
	}
	parts := strings.Fields(string(out))
	if len(parts) > 1 {
		fingerprint = strings.TrimSpace(parts[1])
	}
	return publicKey, fingerprint, nil
}

func expandHome(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}
