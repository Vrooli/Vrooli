package main

import (
	"context"

	"scenario-to-cloud/credentials"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/sshidentity"
)

// boundSSHKeyPath is the key file the deployment's credential binding
// names, or "" (ambient identity) when there is none or no ledger.
func (s *Server) boundSSHKeyPath(ctx context.Context, deploymentID string) string {
	if s.repo == nil {
		return ""
	}
	keyPath, err := credentials.ResolveSSHKeyPath(ctx, s.repo, deploymentID)
	if err != nil {
		s.log("resolve ssh key binding failed", map[string]interface{}{"deployment_id": deploymentID, "error": err.Error()})
		return ""
	}
	return keyPath
}

func (s *Server) resolveCanonicalIdentity(ctx context.Context, dep *domain.Deployment) sshidentity.DeploymentSSHIdentity {
	resolver := sshidentity.DefaultResolver{}
	var existing *sshidentity.DeploymentSSHIdentity
	if parsed, err := sshidentity.FromDeployment(dep); err == nil {
		existing = &parsed
	}
	boundKey := ""
	if dep != nil {
		boundKey = s.boundSSHKeyPath(ctx, dep.ID)
	}
	resolved, err := resolver.Resolve(boundKey, existing)
	if err != nil {
		s.log("resolve ssh identity failed", map[string]interface{}{"error": err.Error()})
		return sshidentity.DeploymentSSHIdentity{
			AuthMode:          sshidentity.AuthModeUnknown,
			VerificationState: sshidentity.VerificationUnknown,
		}
	}
	return resolved
}

func (s *Server) persistCanonicalIdentity(ctx context.Context, deploymentID string, identity sshidentity.DeploymentSSHIdentity) {
	payload, err := sshidentity.Marshal(identity)
	if err != nil {
		s.log("marshal ssh identity failed", map[string]interface{}{"deployment_id": deploymentID, "error": err.Error()})
		return
	}
	if err := s.repo.UpdateDeploymentSSHIdentity(ctx, deploymentID, payload); err != nil {
		s.log("persist ssh identity failed", map[string]interface{}{"deployment_id": deploymentID, "error": err.Error()})
	}
}
