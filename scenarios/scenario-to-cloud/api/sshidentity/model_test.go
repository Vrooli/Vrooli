package sshidentity

import (
	"testing"
	"time"
)

func TestDeploymentSSHIdentity_NormalizeRejectsInvalid(t *testing.T) {
	id := DeploymentSSHIdentity{AuthMode: "bad", VerificationState: VerificationUnknown}
	if err := id.Normalize(); err == nil {
		t.Fatal("expected invalid auth_mode error")
	}
}

func TestDeploymentSSHIdentity_NormalizeRequiresKeyForExplicit(t *testing.T) {
	id := DeploymentSSHIdentity{AuthMode: AuthModeExplicitKey, VerificationState: VerificationUnknown}
	if err := id.Normalize(); err == nil {
		t.Fatal("expected explicit_key key_path validation error")
	}
}

func TestApplyVerificationResult_NonExplicitForcesUnknown(t *testing.T) {
	id := DeploymentSSHIdentity{AuthMode: AuthModeAgent, VerificationState: VerificationUnknown}
	updated := ApplyVerificationResult(id, VerificationAuthorized, time.Now().UTC())
	if updated.VerificationState != VerificationUnknown {
		t.Fatalf("VerificationState=%q, want %q", updated.VerificationState, VerificationUnknown)
	}
}
