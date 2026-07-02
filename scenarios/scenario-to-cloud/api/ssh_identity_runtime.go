package main

import (
	"context"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/sshidentity"
)

func (s *Server) resolveCanonicalIdentity(manifest domain.CloudManifest, dep *domain.Deployment) sshidentity.DeploymentSSHIdentity {
	resolver := sshidentity.DefaultResolver{}
	var existing *sshidentity.DeploymentSSHIdentity
	if parsed, err := sshidentity.FromDeployment(dep); err == nil {
		existing = &parsed
	}
	resolved, err := resolver.Resolve(manifest, existing)
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
