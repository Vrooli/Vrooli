package vps

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/internal/shellutil"
	"scenario-to-cloud/reach/sshadapter"
	"scenario-to-cloud/release"
)

// PlanRequest is everything the VPS planner needs to compile a plan. It is
// the one seam the REST plan endpoints, the apply endpoints and the pipeline
// share, so preview and execution always compile from the same inputs.
type PlanRequest struct {
	Manifest domain.CloudManifest
	// BundlePath is the local release archive. Required for install scope.
	BundlePath string
	// BundleSHA256 pins the archive digest. When empty and BundlePath is
	// readable the digest is computed from the file.
	BundleSHA256 string
	// Deployment binds the plan to a stored deployment identity. When nil the
	// plan is ad hoc: identity is derived from the manifest target.
	Deployment   *domain.Deployment
	Closure      *domain.Closure
	Scope        string
	Observations execplan.Observations
	Policy       execplan.Policy
}

// CompilePlan compiles the executable plan for a request. It reads the
// bundle file to pin its digest and nothing else; it never touches the
// target.
func CompilePlan(ctx context.Context, req PlanRequest) (*execplan.Plan, error) {
	if req.Manifest.Target.VPS == nil {
		return nil, fmt.Errorf("target.vps is required")
	}
	if req.Scope == "" {
		req.Scope = execplan.ScopeFull
	}
	// An ad hoc install plan needs a local bundle; a stored deployment may
	// plan before its bundle is built (the pipeline builds it).
	if req.Deployment == nil && req.Scope != execplan.ScopeRuntime && req.Scope != execplan.ScopeStart && strings.TrimSpace(req.BundlePath) == "" {
		return nil, fmt.Errorf("bundle_path is required")
	}
	// The runtime scope validates the dependency snapshot against the
	// repository declaration (legacy BuildDeployPlan behaviour); a supplied
	// closure is the declared snapshot and supersedes that check.
	if req.Closure == nil && req.Scope != execplan.ScopeInstall {
		if err := validateManifestResourceDependencies(req.Manifest); err != nil {
			return nil, err
		}
	}
	release, err := releaseRefFor(req)
	if err != nil {
		return nil, err
	}
	return execplan.Compile(ctx, execplan.CompileInputs{
		Deployment:       deploymentRefFor(req),
		DesiredRevision:  req.Observations.DeploymentRevision + 1,
		Release:          release,
		ReleaseArtifacts: ReleaseArtifactsFor(req.BundlePath),
		Closure:          req.Closure,
		Manifest:         req.Manifest,
		ArtifactPath:     strings.TrimSpace(req.BundlePath),
		Scope:            req.Scope,
		Observations:     req.Observations,
		Policy:           req.Policy,
	})
}

func deploymentRefFor(req PlanRequest) identity.DeploymentRef {
	vps := req.Manifest.Target.VPS
	if req.Deployment != nil {
		ref := identity.DeploymentRef{
			ID:          req.Deployment.ID,
			ScenarioID:  req.Deployment.ScenarioID,
			Environment: identity.NormalizeEnvironment(req.Deployment.Environment),
			Target:      req.Deployment.Target,
		}
		if ref.Target.Transport == "" {
			ref.Target.Transport = identity.TransportSSH
		}
		if ref.Target.Locator == (identity.TargetLocator{}) {
			ref.Target.Locator = identity.TargetLocator{Host: vps.Host, Port: vps.Port, User: vps.User, Workdir: vps.Workdir}
		}
		return ref
	}
	return identity.DeploymentRef{
		ID:          "adhoc:" + strings.TrimSpace(vps.Host) + ":" + strings.TrimSpace(req.Manifest.Scenario.ID),
		ScenarioID:  strings.TrimSpace(req.Manifest.Scenario.ID),
		Environment: identity.NormalizeEnvironment(req.Manifest.Environment),
		Target: identity.TargetRef{
			Transport: identity.TransportSSH,
			Locator:   identity.TargetLocator{Host: vps.Host, Port: vps.Port, User: vps.User, Workdir: vps.Workdir},
		},
	}
}

// ReleaseArtifactsFor resolves the built release set beside a bundle: the
// canonical release id the target owner stages by, the manifest and the
// native control plane. A bare bundle (no release-manifest.json beside it)
// yields an empty set, and the target verbs refuse to stage it.
func ReleaseArtifactsFor(bundlePath string) execplan.ReleaseArtifacts {
	bundlePath = strings.TrimSpace(bundlePath)
	if bundlePath == "" {
		return execplan.ReleaseArtifacts{}
	}
	rel, err := release.LoadDir(filepath.Dir(bundlePath))
	if err != nil {
		return execplan.ReleaseArtifacts{}
	}
	return execplan.ReleaseArtifacts{ID: rel.Digest, ManifestPath: rel.ManifestPath(), NativeCLI: rel.NativeCLIPath()}
}

// releaseRefFor pins the release identity. Until the release domain binds a
// canonical release digest, the digest is the sha256:<bundle_sha> placeholder;
// runtime-only plans without a known bundle bind to the configuration digest.
func releaseRefFor(req PlanRequest) (identity.ReleaseRef, error) {
	configDigest, err := ConfigurationDigest(req.Manifest)
	if err != nil {
		return identity.ReleaseRef{}, err
	}
	sha := strings.TrimSpace(req.BundleSHA256)
	if sha == "" && req.Deployment != nil && req.Deployment.BundleSHA256 != nil {
		sha = strings.TrimSpace(*req.Deployment.BundleSHA256)
	}
	pending := ""
	if sha == "" && strings.TrimSpace(req.BundlePath) != "" {
		computed, err := fileSHA256(req.BundlePath)
		if err != nil {
			// Preview may run before the bundle exists locally. The artifact
			// is then unpinned: release.verify cannot pass at apply time until
			// a real digest is bound, which is the intended refusal.
			sum := sha256.Sum256([]byte(strings.TrimSpace(req.BundlePath)))
			pending = "pending:" + hex.EncodeToString(sum[:16])
		} else {
			sha = computed
		}
	}
	ref := identity.ReleaseRef{ConfigurationDigest: configDigest}
	if req.Closure != nil {
		ref.ClosureDigest = req.Closure.Digest
	} else {
		ref.ClosureDigest = req.Manifest.Dependencies.ClosureDigest
	}
	switch {
	case sha != "":
		ref.Digest = "sha256:" + strings.TrimPrefix(sha, "sha256:")
	case pending != "":
		ref.Digest = pending
	default:
		ref.Digest = "manifest:" + strings.TrimPrefix(configDigest, "sha256:")
	}
	return ref, nil
}

// ConfigurationDigest is sha256 over the manifest with volatile fields
// removed (analyzer timestamps).
func ConfigurationDigest(manifest domain.CloudManifest) (string, error) {
	semantic := manifest
	semantic.Dependencies.Analyzer.GeneratedAt = ""
	raw, err := json.Marshal(semantic)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// RenderSteps derives the legacy VPSPlanStep view from a plan: id = action
// id, command = the shell preview. The preview is display only; execution
// iterates the plan's actions.
func RenderSteps(plan *execplan.Plan, manifest domain.CloudManifest) []domain.VPSPlanStep {
	if plan == nil {
		return nil
	}
	preview := execplan.Render(plan, execplan.WithShellRenderer(ShellRenderer(manifest)))
	commands := map[string]string{}
	for _, line := range preview.ShellPreview {
		commands[line.ActionID] = line.Command
	}
	steps := make([]domain.VPSPlanStep, 0, len(plan.Actions))
	for i, action := range plan.Actions {
		change := preview.Changes[i]
		steps = append(steps, domain.VPSPlanStep{
			ID:          action.ID,
			Title:       change.Summary,
			Description: fmt.Sprintf("%s · effect %s · verification %s · recovery %s · retry %s", action.OwnerOperation, action.Effect, action.Verification, action.Recovery, action.Retry),
			Command:     commands[action.ID],
		})
	}
	return steps
}

// RenderPreview renders the review surface with the shell preview.
func RenderPreview(plan *execplan.Plan, manifest domain.CloudManifest) execplan.Preview {
	return execplan.Render(plan, execplan.WithShellRenderer(ShellRenderer(manifest)))
}

// PreviewIdentity is the placeholder operation identity the preview renders
// with; execution substitutes the admitted operation id and fence.
var PreviewIdentity = Identity{OperationID: "op-preview", Fence: 0}

// ShellRenderer derives the convenience shell preview for an action from the
// typed invocations the executor will run, rendered through the same quoting
// policy the SSH adapter uses. Nothing it returns is executed.
func ShellRenderer(manifest domain.CloudManifest) execplan.ShellRenderer {
	locator := domain.TargetRefFromManifest(manifest).Locator
	workdir := locator.Workdir
	return func(action execplan.Action) string {
		cc := CommandContext{DeploymentID: "deployment", ScenarioID: manifest.Scenario.ID, Identity: PreviewIdentity, Manifest: manifest}
		remote := func(argv []string) string {
			return sshadapter.LocalSSHCommand(locator, sshadapter.RemoteCommand(workdir, argv))
		}
		scp := func(local, remotePath string) string { return sshadapter.LocalSCPCommand(locator, local, remotePath) }
		return ShellPreviewFor(action, cc, remote, scp)
	}
}

// bundleDestination mirrors the compiler's destination for a local bundle.
func bundleDestination(workdir, bundlePath string) string {
	return shellutil.SafeRemoteJoin(workdir, ".vrooli", "cloud", "bundles", filepath.Base(bundlePath))
}
