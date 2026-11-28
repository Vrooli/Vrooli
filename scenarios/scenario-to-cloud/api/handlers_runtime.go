package main

import (
	"context"
	"encoding/json"
	"strings"

	"scenario-to-cloud/credentials"
	"scenario-to-cloud/deployment"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/release"
	"scenario-to-cloud/releasesvc"
	"scenario-to-cloud/vps"
)

// executionRuntime is the target-facing seam set a plan runs with outside
// the durable owner (the ad hoc /vps/setup|deploy/apply routes). The
// identity is the admitted operation when one exists.
func (s *Server) executionRuntime(manifest domain.CloudManifest, deploymentID, operationID string) vps.Runtime {
	rt := vps.Runtime{
		Reach:      s.reach,
		Target:     domain.TargetRefFromManifest(manifest),
		Identity:   vps.Identity{OperationID: operationID},
		Backups:    s.recoveryPointRecorder(),
		SecretsGen: s.secretsGenerator,
	}
	if s.repo != nil && strings.TrimSpace(deploymentID) != "" {
		if dep, err := s.repo.GetDeployment(context.Background(), deploymentID); err == nil && dep != nil && !dep.Target.IsZero() {
			rt.Target = dep.Target
			if rt.Target.Locator.Workdir == "" && manifest.Target.VPS != nil {
				rt.Target.Locator = identity.TargetLocator{Host: manifest.Target.VPS.Host, Port: manifest.Target.VPS.Port, User: manifest.Target.VPS.User, Workdir: manifest.Target.VPS.Workdir}
			}
		}
	}
	rt.Reach = s.targetReach()
	return rt
}

// targetReach is the server's one transport seam. Production and handler
// tests must select an explicit configured transport; missing reach fails
// closed instead of constructing an SSH adapter from ambient host identity.
func (s *Server) targetReach() reach.Reach {
	if s.reach != nil {
		return s.reach
	}
	return &reach.Router{}
}

// adHocReach selects the same explicitly configured transport seam as every
// bound deployment. It does not infer SSH authority from a host locator.
func (s *Server) adHocReach(identity.TargetRef) reach.Reach {
	return s.targetReach()
}

// recoveryPointRecorder records target-captured recovery points on the
// cloud repository.
func (s *Server) recoveryPointRecorder() vps.RecoveryPointRecorder {
	if s.repo == nil {
		return nil
	}
	return &deployment.RecoveryPointRecorder{Repo: s.repo}
}

// credentialBinder binds the credential lifecycle to a deployment target
// for the executor (same wiring as the credential routes).
func (s *Server) credentialBinder() deployment.CredentialBinder {
	return func(ctx context.Context, deploymentID string, target identity.TargetRef, manifest domain.CloudManifest) (*credentials.Service, error) {
		bound, err := credentialAdapter{s: s}.bind(ctx, deploymentID)
		if err != nil {
			return nil, err
		}
		return bound.service, nil
	}
}

// releaseBuilder builds the release artifact set for a deployment through
// the release service (bundle, manifest, native control plane, lease).
func (s *Server) releaseBuilder() deployment.ReleaseBuilder {
	return func(ctx context.Context, manifest domain.CloudManifest, platform release.Platform) (release.Release, error) {
		svc, err := s.releaseService()
		if err != nil {
			return release.Release{}, err
		}
		closureDigest := manifest.Dependencies.ClosureDigest
		if closureSvc, cerr := s.closureService(); cerr == nil && closureSvc != nil {
			if derived, derr := closureSvc.Resolve(ctx, manifest.Scenario.ID, manifest.Environment); derr == nil {
				closureDigest = derived.Digest
			}
		}
		return svc.Build(ctx, releasesvc.BuildRequest{Manifest: manifest, ClosureDigest: closureDigest, GOOS: platform.GOOS, GOARCH: platform.GOARCH})
	}
}

// persistentDataBindings reads the mappings the legacy conversion recorded
// on a deployment (<binding id>=<scenario>/<path>).
func persistentDataBindings(dep *domain.Deployment) []string {
	if dep == nil || !dep.PersistentData.Valid || len(dep.PersistentData.Data) == 0 {
		return nil
	}
	var doc domain.PersistentDataBindings
	if err := json.Unmarshal(dep.PersistentData.Data, &doc); err != nil {
		return nil
	}
	return doc.Specs()
}

// recoveryPointsFor lists the deployment's recovery points for rollback
// eligibility and retention decisions.
func (s *Server) recoveryPointsFor(ctx context.Context, deploymentID string) []domain.RecoveryPoint {
	if s.repo == nil {
		return nil
	}
	points, err := s.repo.ListRecoveryPoints(ctx, deploymentID)
	if err != nil {
		return nil
	}
	return points
}
