package vps

import (
	"context"
	"fmt"
	"strings"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/execplan"
)

// Workload stop and retirement. A stop is always the lifecycle owner's scoped
// stop (privilegebroker process.stop.scoped through `vrooli cloud-target host
// repair`): it releases only this deployment's demand on shared resources, so
// a resource another deployment on the host still needs keeps running. There
// is no process-name kill and no port kill; a stale process the lifecycle
// owner does not know is a host-repair finding, not something the cloud
// side hunts with pkill.

func runEdgeRouteRetire(ctx context.Context, e *executor, action execplan.Action) (string, error) {
	detail, err := e.runCommands(ctx, action)
	if err != nil {
		if apierrors.Is(err, "edge_rollback_not_eligible") {
			return "no route snippet to remove (unchanged)", nil
		}
		return "", err
	}
	return detail, nil
}

func runGrantsRevoke(ctx context.Context, e *executor, action execplan.Action) (string, error) {
	if e.manifest.Secrets == nil || len(e.manifest.Secrets.BundleSecrets) == 0 {
		return "no credential grants declared (unchanged)", nil
	}
	if e.rt.Credentials == nil {
		return "", apierrors.New(apierrors.CodeUnsupportedCapability, "credential authority is not configured for this executor; grants.revoke cannot run").WithDetail("action", action.ID)
	}
	result, err := e.rt.Credentials.Revoke(ctx, CredentialProvisionRequest{DeploymentID: e.deploymentID, Target: e.rt.Target, Identity: e.rt.Identity, Manifest: e.manifest})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d grants revoked", len(result.Revoked)), nil
}

// runDataRetire records the disposition. No target verb deletes persistent
// data: an irreversible policy is refused with the owner that must exist
// first, and the retained policy leaves every binding in place.
func runDataRetire(_ context.Context, _ *executor, action execplan.Action) (string, error) {
	switch action.Inputs["retention_policy"] {
	case execplan.RetentionDelete:
		return "", apierrors.New(apierrors.CodeUnsupportedCapability, "irreversible data retirement has no target owner verb yet; the data stays in place").
			WithDetail("bindings", action.Inputs["bindings"]).WithDetail("required_owner", "cloud-target data retire")
	default:
		return "retained: " + action.Inputs["bindings"] + " (persistent-data root untouched)", nil
	}
}

// runArtifactsRetire lists what the target holds and reports what cleanup
// would remove; active, previous and backup-referenced artifacts are never
// candidates. Deletion itself needs the target prune owner.
func runArtifactsRetire(ctx context.Context, e *executor, action execplan.Action) (string, error) {
	listing, err := e.listReleases(ctx, action.ID+".observe")
	if err != nil {
		return "", err
	}
	var retained, candidates []string
	for _, rel := range listing.Releases {
		switch rel.Role {
		case "active", "previous":
			retained = append(retained, rel.Digest)
		default:
			candidates = append(candidates, rel.Digest)
		}
	}
	return fmt.Sprintf("retained %d (%s); unreferenced %d (%s) await the target prune owner", len(retained), strings.Join(retained, ","), len(candidates), strings.Join(candidates, ",")), nil
}
