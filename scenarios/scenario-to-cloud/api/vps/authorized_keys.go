package vps

import (
	"context"
	"path"
	"strings"

	"scenario-to-cloud/sshidentity"
)

// VerifyAuthorizedKey reports whether the explicit key the identity names
// is present in the bound user's authorized_keys on the target. The read is
// one observation program; a non-explicit identity is unknown by definition.
func VerifyAuthorizedKey(ctx context.Context, prober Prober, identity sshidentity.DeploymentSSHIdentity) (sshidentity.VerificationState, error) {
	if identity.AuthMode != sshidentity.AuthModeExplicitKey || strings.TrimSpace(identity.KeyPath) == "" {
		return sshidentity.VerificationUnknown, nil
	}
	publicKey, _, err := sshidentity.ReadPublicKeyAndFingerprint(identity.KeyPath)
	if err != nil {
		return sshidentity.VerificationUnknown, err
	}
	parts := strings.Fields(publicKey)
	if len(parts) < 2 {
		return sshidentity.VerificationUnknown, nil
	}
	needle := parts[0] + " " + parts[1]
	res, err := prober.Observe(ctx, "cat", path.Join(HomeDir(prober.Target.Locator.User), ".ssh", "authorized_keys"))
	if err != nil {
		return sshidentity.VerificationUnknown, err
	}
	if strings.Contains(res.Stdout, needle) {
		return sshidentity.VerificationAuthorized, nil
	}
	return sshidentity.VerificationUnauthorized, nil
}
