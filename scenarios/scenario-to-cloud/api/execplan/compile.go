package execplan

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/closure"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/edge"
	"scenario-to-cloud/identity"
)

// Observations are the facts about the target and the deployment record the
// compiler binds the plan to. An empty observation means "unknown", never
// "satisfied": the compiler only emits a no-op when every observed digest is
// present and equal to the desired one.
type Observations struct {
	// DeploymentRevision is the current revision of the deployment record.
	DeploymentRevision uint64
	// TargetEnrollment is the observed enrollment generation of the target.
	TargetEnrollment uint64
	// DataSchemaVersion is the observed persistent-data schema version.
	DataSchemaVersion string
	// ActiveReleaseDigest, ActiveConfigurationDigest and ActiveClosureDigest
	// describe what the target currently runs, as reported by the health
	// observation. Empty means unobserved.
	ActiveReleaseDigest       string
	ActiveConfigurationDigest string
	ActiveClosureDigest       string
	// LegacyDataInventory lists legacy preserve paths
	// (scenarios/<id>/<relative>) the target must carry across activation
	// until they are declared as persistent-data bindings.
	LegacyDataInventory []string
	// PersistentDataBindings are the mappings recorded on the deployment by
	// the legacy conversion (<binding id>=<scenario>/<relative>). They win
	// over the heuristic: a legacy path they cover is bound, not carried.
	PersistentDataBindings []string
	// HostReadiness is "ready", "needs_prepare" or "" (unknown).
	HostReadiness string
	// SatisfiedInputs lists credential addresses (logical_id:field) or input
	// keys that are already satisfied, so they do not produce a handoff.
	SatisfiedInputs []string
}

// Policy pins the policy the plan is compiled under.
type Policy struct {
	Version string
	// MaintenanceStrategy forces a strategy; empty selects one from the
	// closure (side_by_side when no persistent data is declared and the
	// lifecycle owner allocates ports, maintenance otherwise).
	MaintenanceStrategy string
	// MaintenanceDowntimeSeconds is the declared bound of the maintenance
	// interval shown in the preview; zero selects DefaultMaintenanceDowntime.
	MaintenanceDowntimeSeconds int
	// RecoveryKeyRef is the credential reference (logical_id:field) recovery
	// points are sealed under; empty selects DefaultRecoveryKeyRef(scenario).
	RecoveryKeyRef string
	// RetentionPolicy is required by the retire scope when persistent data
	// is declared: "retain" keeps the data, "delete" is irreversible.
	RetentionPolicy string
}

// DefaultMaintenanceDowntime is the declared maintenance bound when the
// policy pins none.
const DefaultMaintenanceDowntime = 60

// Retention policies the retire scope accepts.
const (
	RetentionRetain = "retain"
	RetentionDelete = "delete"
)

// DefaultRecoveryKeyRef names the recovery key credential of a scenario.
func DefaultRecoveryKeyRef(scenarioID string) string {
	return "vrooli/" + strings.TrimSpace(scenarioID) + ":recovery-key"
}

// ReleaseArtifacts are the local release artifact set beside the bundle and
// its canonical target-side identity. ID is the 64-hex release digest the
// target owner stages and activates by; empty when the bundle is not part of
// a built release (the target verbs then refuse before any write).
type ReleaseArtifacts struct {
	ID           string
	ManifestPath string
	NativeCLI    string
}

// CompileInputs is everything Compile reads. Nothing else influences the
// plan, which is what makes the digest deterministic.
type CompileInputs struct {
	Deployment      identity.DeploymentRef
	DesiredRevision uint64
	// Release identifies the artifact. Digest may still be the
	// sha256:<bundle_sha> placeholder until the release domain lands.
	Release identity.ReleaseRef
	// ReleaseArtifacts is the built release set (manifest, native CLI,
	// canonical id) when the bundle lives in a release directory.
	ReleaseArtifacts ReleaseArtifacts
	// Closure is optional; when nil the manifest's dependency snapshot and
	// closure digest are used.
	Closure *domain.Closure
	// Manifest supplies the typed configuration: workdir, ports, edge, the
	// dependency snapshot and declared secrets.
	Manifest domain.CloudManifest
	// ArtifactPath is the local path of the release archive to deliver.
	ArtifactPath string
	// Scope selects the action subset (default ScopeFull).
	Scope        string
	Observations Observations
	Policy       Policy
}

// HostPreparePackages is the fixed package set host.prepare ensures. It
// mirrors the privilege broker's apt allowlist.
var HostPreparePackages = []string{"ca-certificates", "curl", "git", "gnupg", "jq", "lsb-release", "tar", "unzip"}

// Directory names the legacy inventory heuristic treats as mutable data.
var LegacyMutableDirNames = []string{"cache", "data", "files", "logs", "runtime", "state", "storage", "tmp", "uploads"}

// Compile turns desired state, closure and observations into a deterministic
// plan. It never touches the target and never creates records.
func Compile(_ context.Context, in CompileInputs) (*Plan, error) {
	if err := validateInputs(&in); err != nil {
		return nil, err
	}
	plan := &Plan{
		SchemaVersion:       SchemaVersion,
		DeploymentID:        in.Deployment.ID,
		ScenarioID:          in.Deployment.ScenarioID,
		Environment:         identity.NormalizeEnvironment(in.Deployment.Environment),
		Target:              targetOf(in.Deployment.Target),
		Scope:               in.Scope,
		DesiredRevision:     in.DesiredRevision,
		ReleaseDigest:       in.Release.Digest,
		ConfigurationDigest: in.Release.ConfigurationDigest,
		ClosureDigest:       closureDigest(in),
		PolicyVersion:       in.Policy.Version,
		Preconditions:       []Precondition{},
		Actions:             []Action{},
	}

	if missing := missingInputs(in); len(missing) > 0 && in.Scope != ScopeRetire && in.Scope != ScopeStop {
		plan.Outcome = OutcomeNeedsInput
		plan.Handoff = &Handoff{
			Owner: HandoffOwner, Kind: HandoffKind,
			Reference: handoffReference(in.Deployment.ID, plan.Target, in.DesiredRevision, plan.ClosureDigest, missing),
			Missing:   missing, DeploymentID: in.Deployment.ID, Target: plan.Target,
			DesiredRevision: in.DesiredRevision, SelectionDigest: plan.ClosureDigest,
		}
		plan.Actions = append(plan.Actions, Action{
			ID:                 OpInputResumeHandoff,
			OwnerOperation:     OpInputResumeHandoff,
			Effect:             EffectNone,
			RequiredCapability: "onboarding:resume",
			Inputs: map[string]string{
				"owner":                 HandoffOwner,
				"kind":                  HandoffKind,
				"reference":             plan.Handoff.Reference,
				"missing":               strings.Join(missing, ","),
				"deployment_id":         in.Deployment.ID,
				"target":                plan.Target.NodeID,
				"enrollment_generation": strconv.FormatUint(plan.Target.EnrollmentGeneration, 10),
				"desired_revision":      strconv.FormatUint(in.DesiredRevision, 10),
				"selection_digest":      plan.ClosureDigest,
			},
			DependsOn:    []string{},
			Verification: "inputs_satisfied",
			Recovery:     "none_required",
			Retry:        RetrySafeReplay,
			CancelPoint:  true,
		})
		plan.Preconditions = preconditions(plan, in)
		plan.Presentation = presentationFor(plan, in)
		return plan, nil
	}

	if in.Scope != ScopeRetire && in.Scope != ScopeStop && desiredStateSatisfied(plan, in.Observations) {
		plan.Outcome = OutcomeNoOp
		plan.Preconditions = preconditions(plan, in)
		plan.Presentation = presentationFor(plan, in)
		return plan, nil
	}

	plan.Outcome = OutcomeApply
	plan.Actions = actionsFor(in)
	plan.Preconditions = preconditions(plan, in)
	plan.Presentation = presentationFor(plan, in)
	return plan, nil
}

func validateInputs(in *CompileInputs) error {
	if strings.TrimSpace(in.Deployment.ID) == "" {
		return apierrors.New(apierrors.CodeInvalidRequest, "Plan compile requires a deployment id")
	}
	if strings.TrimSpace(in.Deployment.ScenarioID) == "" {
		in.Deployment.ScenarioID = strings.TrimSpace(in.Manifest.Scenario.ID)
	}
	if in.Deployment.ScenarioID == "" {
		return apierrors.New(apierrors.CodeInvalidRequest, "Plan compile requires a scenario id")
	}
	if in.Manifest.Target.VPS == nil || strings.TrimSpace(in.Manifest.Target.VPS.Workdir) == "" {
		return apierrors.New(apierrors.CodeInvalidRequest, "Plan compile requires target.vps.workdir")
	}
	if strings.TrimSpace(in.Release.Digest) == "" {
		return apierrors.New(apierrors.CodeInvalidRequest, "Plan compile requires a release digest")
	}
	if strings.TrimSpace(in.Deployment.Target.Transport) == "" {
		in.Deployment.Target.Transport = identity.TransportSSH
	}
	switch in.Scope {
	case "":
		in.Scope = ScopeFull
	case ScopeFull, ScopeInstall, ScopeRuntime, ScopeStart, ScopeRetire, ScopeStop:
	default:
		return apierrors.Newf(apierrors.CodeInvalidRequest, "Unknown plan scope %q", in.Scope)
	}
	if in.Policy.Version == "" {
		in.Policy.Version = DefaultPolicyVersion
	}
	switch in.Policy.MaintenanceStrategy {
	case "":
		in.Policy.MaintenanceStrategy = SelectStrategy(in.Closure, in.Manifest)
	case StrategyMaintenance, StrategySideBySide:
	default:
		return apierrors.Newf(apierrors.CodeInvalidRequest, "Unknown maintenance strategy %q", in.Policy.MaintenanceStrategy)
	}
	if in.Policy.MaintenanceDowntimeSeconds <= 0 {
		in.Policy.MaintenanceDowntimeSeconds = DefaultMaintenanceDowntime
	}
	if in.Policy.RecoveryKeyRef == "" {
		in.Policy.RecoveryKeyRef = DefaultRecoveryKeyRef(in.Deployment.ScenarioID)
	}
	if in.Scope == ScopeRetire {
		switch in.Policy.RetentionPolicy {
		case RetentionRetain, RetentionDelete:
		case "":
			if len(persistentBindings(*in)) > 0 || len(legacyInventory(*in)) > 0 {
				return apierrors.New(apierrors.CodeInvalidRequest, "retire requires an explicit retention_policy (retain|delete) because the deployment declares persistent data").
					WithDetail("persistent_data", persistentBindings(*in)).WithDetail("legacy_data", legacyInventory(*in))
			}
			in.Policy.RetentionPolicy = RetentionRetain
		default:
			return apierrors.Newf(apierrors.CodeInvalidRequest, "Unknown retention policy %q", in.Policy.RetentionPolicy)
		}
	}
	if in.Closure != nil && !in.Closure.Supported() {
		names := make([]string, 0, len(in.Closure.Unsupported))
		for _, u := range in.Closure.Unsupported {
			names = append(names, u.Component+":"+u.ReasonCode)
		}
		return apierrors.New(apierrors.CodeUnsupportedCapability, "Closure has unsupported components").
			WithDetail("unsupported", names)
	}
	return nil
}

// SelectStrategy chooses the activation strategy the closure permits: a
// workload without persistent data whose ports the lifecycle owner allocates
// (no pinned manifest ports) may run its candidate beside the current
// release; anything stateful or port-pinned needs a declared maintenance
// interval. The choice is shown in the preview and is material.
func SelectStrategy(closure *domain.Closure, manifest domain.CloudManifest) string {
	if closure == nil || len(closure.PersistentData) > 0 || len(manifest.Target.VPS.PreservePaths) > 0 {
		return StrategyMaintenance
	}
	if len(manifest.Ports) > 0 {
		return StrategyMaintenance
	}
	return StrategySideBySide
}

func targetOf(ref identity.TargetRef) Target {
	return Target{
		MachineID:            ref.MachineID,
		NodeID:               ref.NodeID,
		EnrollmentGeneration: ref.EnrollmentGeneration,
		Transport:            ref.Transport,
	}
}

func closureDigest(in CompileInputs) string {
	if in.Closure != nil && in.Closure.Digest != "" {
		return in.Closure.Digest
	}
	if in.Release.ClosureDigest != "" {
		return in.Release.ClosureDigest
	}
	return in.Manifest.Dependencies.ClosureDigest
}

// desiredStateSatisfied is true only when every observed digest is present
// and equal to the desired one. An unobserved target is never "satisfied".
func desiredStateSatisfied(plan *Plan, obs Observations) bool {
	if obs.ActiveReleaseDigest == "" || obs.ActiveReleaseDigest != plan.ReleaseDigest {
		return false
	}
	if obs.ActiveConfigurationDigest != plan.ConfigurationDigest {
		return false
	}
	if plan.ClosureDigest != "" && obs.ActiveClosureDigest != plan.ClosureDigest {
		return false
	}
	return true
}

// missingInputs derives the operator inputs the closure and manifest declare
// that nothing has satisfied yet. Only references are handled; values never
// enter the plan.
func missingInputs(in CompileInputs) []string {
	satisfied := map[string]bool{}
	for _, key := range in.Observations.SatisfiedInputs {
		if key = strings.TrimSpace(key); key != "" {
			satisfied[key] = true
		}
	}
	missing := map[string]bool{}
	if in.Closure != nil {
		for _, component := range in.Closure.ComponentsOfKind(domain.ClosureKindCredentialDescriptor) {
			if component.Credential == nil || !component.Credential.Required {
				continue
			}
			address := strings.TrimSpace(component.Credential.LogicalID) + ":" + strings.TrimSpace(component.Credential.Field)
			if !satisfied[address] && !(component.Credential.Env != "" && satisfied[component.Credential.Env]) {
				missing[address] = true
			}
		}
	}
	if in.Manifest.Secrets != nil {
		for _, secret := range in.Manifest.Secrets.BundleSecrets {
			if secret.Class != "user_prompt" || !secret.Required {
				continue
			}
			key := strings.TrimSpace(secret.Target.Name)
			if key == "" {
				key = strings.TrimSpace(secret.ID)
			}
			address := key
			if secret.Descriptor != nil {
				address = strings.TrimSpace(secret.Descriptor.LogicalID) + ":" + strings.TrimSpace(secret.Descriptor.Field)
			}
			if satisfied[key] || satisfied[address] {
				continue
			}
			missing[address] = true
		}
	}
	return sortedKeys(missing)
}

func handoffReference(deploymentID string, target Target, desiredRevision uint64, selectionDigest string, missing []string) string {
	parts := []string{deploymentID, target.MachineID, target.NodeID, strconv.FormatUint(target.EnrollmentGeneration, 10), strconv.FormatUint(desiredRevision, 10), selectionDigest, strings.Join(missing, "\n")}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return fmt.Sprintf("vrooli-onboarding://deployments/%s/resume/%s", deploymentID, hex.EncodeToString(sum[:8]))
}

// actionsFor builds the apply-outcome action graph for the scope.
func actionsFor(in CompileInputs) []Action {
	var actions []Action
	switch in.Scope {
	case ScopeInstall:
		actions = installActions(in)
	case ScopeRuntime:
		actions = runtimeActions(in, false)
	case ScopeStart:
		actions = runtimeActions(in, true)
	case ScopeRetire:
		actions = retireActions(in)
	case ScopeStop:
		actions = stopActions(in)
	default:
		actions = append(installActions(in), runtimeActions(in, false)...)
	}
	// Dependencies only reference actions present in this plan.
	present := map[string]bool{}
	for _, action := range actions {
		present[action.ID] = true
	}
	for i := range actions {
		kept := make([]string, 0, len(actions[i].DependsOn))
		for _, dep := range actions[i].DependsOn {
			if present[dep] {
				kept = append(kept, dep)
			}
		}
		actions[i].DependsOn = kept
		if actions[i].Inputs == nil {
			actions[i].Inputs = map[string]string{}
		}
	}
	return actions
}

// layout resolves the target paths every scope shares.
type layout struct {
	workdir       string
	bundleDir     string
	archive       string
	manifestPath  string
	nativeCLIPath string
	releaseDir    string
}

func layoutFor(in CompileInputs) layout {
	workdir := strings.TrimSpace(in.Manifest.Target.VPS.Workdir)
	bundleDir := remoteJoin(workdir, ".vrooli", "cloud", "bundles", releaseDirName(in.Release.Digest))
	archiveName := path.Base(in.ArtifactPath)
	if strings.TrimSpace(in.ArtifactPath) == "" {
		archiveName = "bundle.tar.gz"
	}
	return layout{
		workdir:       workdir,
		bundleDir:     bundleDir,
		archive:       remoteJoin(bundleDir, archiveName),
		manifestPath:  remoteJoin(bundleDir, "release-manifest.json"),
		nativeCLIPath: remoteJoin(workdir, ".vrooli", "bin", "vrooli"),
		releaseDir:    remoteJoin(workdir, ".vrooli", "cloud", "releases", releaseDirName(in.Release.Digest)),
	}
}

func installActions(in CompileInputs) []Action {
	lay := layoutFor(in)
	targetScenarios := targetScenarioIDs(in.Manifest)
	legacy := legacyInventory(in)

	actions := []Action{{
		ID:                 OpHostPrepare,
		OwnerOperation:     OpHostPrepare,
		Effect:             EffectHostWrite,
		RequiredCapability: "privilegebroker:apt.packages.ensure",
		Inputs: map[string]string{
			"packages":   strings.Join(HostPreparePackages, ","),
			"workdir":    lay.workdir,
			"bundle_dir": lay.bundleDir,
		},
		DependsOn:    []string{},
		Verification: "packages_present",
		Recovery:     "none_required",
		Retry:        RetrySafeReplay,
		CancelPoint:  true,
	}}
	if in.Manifest.Edge.Caddy.Enabled {
		actions = append(actions, Action{
			ID:                 OpEdgeFirewallAllow,
			OwnerOperation:     OpEdgeFirewallAllow,
			Effect:             EffectEdgeWrite,
			RequiredCapability: "privilegebroker:edge.ufw.allow",
			Inputs:             map[string]string{"ports": "80,443", "protocol": "tcp"},
			DependsOn:          []string{OpHostPrepare},
			Verification:       "rules_present_or_firewall_inactive",
			Recovery:           "none_required",
			Retry:              RetrySafeReplay,
			CancelPoint:        true,
		})
	}
	actions = append(actions,
		Action{
			ID:                 OpDataInventory,
			OwnerOperation:     OpDataInventory,
			Effect:             EffectNone,
			RequiredCapability: "cloud-target:data.inventory",
			Inputs: map[string]string{
				"workdir":         lay.workdir,
				"scenarios":       strings.Join(targetScenarios, ","),
				"bindings":        strings.Join(persistentBindings(in), ","),
				"data_bindings":   strings.Join(dataBindingSpecs(in), ","),
				"legacy_preserve": strings.Join(legacy, ","),
				"mutable_names":   strings.Join(LegacyMutableDirNames, ","),
			},
			DependsOn:    []string{OpHostPrepare},
			Verification: "inventory_recorded",
			Recovery:     "none_required",
			Retry:        RetrySafeReplay,
			CancelPoint:  true,
		},
		Action{
			ID:                 OpReleaseDeliver,
			OwnerOperation:     OpReleaseDeliver,
			Effect:             EffectDeploymentWrite,
			RequiredCapability: "reach:deliver",
			Inputs: map[string]string{
				"artifact_ref":     in.Release.Digest,
				"release_id":       in.ReleaseArtifacts.ID,
				"bundle_sha256":    bundleSHA(in),
				"artifact_path":    in.ArtifactPath,
				"destination":      lay.archive,
				"release_manifest": lay.manifestPath,
				"native_cli":       lay.nativeCLIPath,
			},
			DependsOn:    []string{OpHostPrepare},
			Verification: "delivered_sha256_matches",
			Recovery:     "redeliver",
			Retry:        RetrySafeReplay,
			CancelPoint:  true,
		},
		Action{
			ID:                 OpReleaseVerify,
			OwnerOperation:     OpReleaseVerify,
			Effect:             EffectNone,
			RequiredCapability: "cloud-target:release.verify",
			Inputs: map[string]string{
				"release_digest":   in.Release.Digest,
				"release_id":       in.ReleaseArtifacts.ID,
				"bundle_sha256":    bundleSHA(in),
				"archive":          lay.archive,
				"release_manifest": lay.manifestPath,
			},
			DependsOn:    []string{OpReleaseDeliver},
			Verification: "archive_sha256_matches_release",
			Recovery:     "none_required",
			Retry:        RetrySafeReplay,
			CancelPoint:  true,
		},
		Action{
			ID:                 OpReleaseStage,
			OwnerOperation:     OpReleaseStage,
			Effect:             EffectDeploymentWrite,
			RequiredCapability: "cloud-target:release.stage",
			Inputs: map[string]string{
				"release_digest":   in.Release.Digest,
				"release_id":       in.ReleaseArtifacts.ID,
				"archive":          lay.archive,
				"release_manifest": lay.manifestPath,
				"release_dir":      lay.releaseDir,
			},
			DependsOn:    []string{OpReleaseVerify},
			Verification: "staged_tree_complete",
			Recovery:     "discard_owned_inactive_stage",
			Retry:        RetrySafeReplay,
			CancelPoint:  true,
		},
		Action{
			ID:                 OpConfigApply,
			OwnerOperation:     OpConfigApply,
			Effect:             EffectHostWrite,
			RequiredCapability: "vrooli:setup",
			Inputs: map[string]string{
				"workdir":             lay.workdir,
				"environment":         "production",
				"selection":           "setup/v1",
				"selection_json_b64":  selectionJSON(in),
				"scenario_id":         in.Manifest.Scenario.ID,
				"resources":           strings.Join(sortedUnique(in.Manifest.Dependencies.Resources), ","),
				"scenarios":           strings.Join(sortedUnique(in.Manifest.Dependencies.Scenarios), ","),
				"autoheal":            strconv.FormatBool(in.Manifest.Bundle.IncludeAutoheal),
				"autoheal_scope_path": remoteJoin(lay.workdir, ".vrooli", "cloud", "autoheal-scope.json"),
			},
			DependsOn:    []string{OpReleaseStage},
			Verification: "vrooli_setup_completed",
			Recovery:     "rerun_setup",
			Retry:        RetrySafeReplay,
			CancelPoint:  true,
		},
	)
	return actions
}

// selectionJSON is the one cloud-side projection of the shared setup/v1
// contract. The target action still owns application, but the executable plan
// carries the exact closure-derived selection it reviewed so onboarding and
// setup cannot silently reconstruct a different workload union.
func selectionJSON(in CompileInputs) string {
	if in.Closure == nil {
		return ""
	}
	target := in.Deployment.Target.NodeID
	if target == "" {
		target = in.Deployment.Target.MachineID
	}
	selection := closure.ToSelection(*in.Closure, target, closure.Overrides{})
	encoded, err := json.Marshal(selection)
	if err != nil {
		return ""
	}
	// Action inputs are argv-shaped metadata and must remain shell-free. The
	// setup owner decodes this URL-safe payload at the onboarding handoff.
	return base64.RawURLEncoding.EncodeToString(encoded)
}

func runtimeActions(in CompileInputs, startOnly bool) []Action {
	lay := layoutFor(in)
	uiPort := in.Manifest.Ports["ui"]
	routeHost, routePort, routeListener := deriveEdgeRoute(in, uiPort)
	ports := portList(in.Manifest.Ports)
	targetScenarios := targetScenarioIDs(in.Manifest)
	hasSecrets := in.Manifest.Secrets != nil && len(in.Manifest.Secrets.BundleSecrets) > 0
	bindings := persistentBindings(in)
	legacy := legacyInventory(in)
	update := in.Observations.ActiveReleaseDigest != "" && in.Observations.ActiveReleaseDigest != in.Release.Digest

	var actions []Action
	if hasSecrets {
		actions = append(actions, Action{
			ID:                 OpCredentialsProvision,
			OwnerOperation:     OpCredentialsProvision,
			Effect:             EffectCredentialWrite,
			RequiredCapability: "credential-authority:provision",
			Inputs: map[string]string{
				"workdir":     lay.workdir,
				"scenario":    in.Manifest.Scenario.ID,
				"descriptors": strings.Join(credentialDescriptors(in.Manifest), ","),
			},
			DependsOn:    []string{},
			Verification: "descriptors_resolvable",
			Recovery:     "none_required",
			Retry:        RetrySafeReplay,
			CancelPoint:  true,
		})
	}
	depScenarios := []string{}
	for _, scen := range sortedUnique(in.Manifest.Dependencies.Scenarios) {
		if scen != in.Manifest.Scenario.ID {
			depScenarios = append(depScenarios, scen)
		}
	}
	resources := sortedUnique(in.Manifest.Dependencies.Resources)
	hasDeps := len(resources) > 0 || len(depScenarios) > 0
	if hasDeps {
		actions = append(actions, Action{
			ID:                 OpRuntimeStartDeps,
			OwnerOperation:     OpRuntimeStartDeps,
			Effect:             EffectRuntimeStart,
			RequiredCapability: "vrooli:resource.start,scenario.start",
			Inputs: map[string]string{
				"workdir":   lay.workdir,
				"resources": strings.Join(resources, ","),
				"scenarios": strings.Join(depScenarios, ","),
			},
			DependsOn:    []string{OpCredentialsProvision},
			Verification: "dependencies_running",
			Recovery:     "restart_dependency",
			Retry:        RetryObserveThenReplay,
			CancelPoint:  true,
		})
	}
	if startOnly {
		actions = append(actions,
			Action{
				ID:                 OpWorkloadStart,
				OwnerOperation:     OpWorkloadStart,
				Effect:             EffectRuntimeStart,
				RequiredCapability: "cloud-target:release.activate",
				Inputs: map[string]string{
					"workdir":       lay.workdir,
					"scenario":      in.Manifest.Scenario.ID,
					"scenarios":     strings.Join(targetScenarios, ","),
					"ports":         ports,
					"ui_port":       strconv.Itoa(uiPort),
					"release_id":    in.ReleaseArtifacts.ID,
					"data_bindings": strings.Join(dataBindingSpecs(in), ","),
				},
				DependsOn:    []string{OpCredentialsProvision, OpRuntimeStartDeps},
				Verification: "runtime_active_release_running",
				Recovery:     "rollback_to_predecessor",
				Retry:        RetryObserveThenReplay,
				CancelPoint:  false,
			},
			readinessAction(in, []string{OpWorkloadStart}),
		)
		return actions
	}
	activateDeps := []string{OpCredentialsProvision, OpRuntimeStartDeps}
	if len(bindings) > 0 || len(legacy) > 0 {
		verification := "recovery_point_recorded"
		precondition := ""
		if update && in.Observations.DataSchemaVersion != "" {
			precondition = in.Release.Digest + "@" + in.Observations.DataSchemaVersion
		}
		actions = append(actions, Action{
			ID:                 OpDataBackup,
			OwnerOperation:     OpDataBackup,
			Effect:             EffectDataWrite,
			RequiredCapability: "cloud-target:data.backup",
			Inputs: map[string]string{
				"bindings":                strings.Join(bindings, ","),
				"binding_specs":           strings.Join(engineBindingSpecs(in), ";"),
				"legacy_preserve":         strings.Join(legacy, ","),
				"recovery_key_ref":        in.Policy.RecoveryKeyRef,
				"schema_version":          in.Observations.DataSchemaVersion,
				"configuration_digest":    in.Release.ConfigurationDigest,
				"release_digest":          in.Release.Digest,
				"migration_posture":       migrationPosture(in, update),
				"retention_policy":        "retain_until_superseded",
				"recovery_point_required": precondition,
			},
			DependsOn:    []string{OpRuntimeStartDeps, OpCredentialsProvision},
			Verification: verification,
			Recovery:     "none_required",
			Retry:        RetryObserveThenReplay,
			CancelPoint:  true,
		})
		activateDeps = append(activateDeps, OpDataBackup)
	}
	if in.Policy.MaintenanceStrategy == StrategyMaintenance {
		actions = append(actions, Action{
			ID:                 OpWorkloadStop,
			OwnerOperation:     OpWorkloadStop,
			Effect:             EffectRuntimeStop,
			RequiredCapability: "privilegebroker:process.stop.scoped",
			Inputs: map[string]string{
				"workdir":  lay.workdir,
				"scenario": in.Manifest.Scenario.ID,
				"ports":    ports,
			},
			DependsOn:    []string{OpDataBackup, OpRuntimeStartDeps, OpCredentialsProvision},
			Verification: "scenario_stopped_shared_demand_kept",
			Recovery:     "none_required",
			Retry:        RetrySafeReplay,
			CancelPoint:  true,
			Downtime:     &Downtime{ExpectedSeconds: in.Policy.MaintenanceDowntimeSeconds, Reason: "maintenance strategy stops the workload before the new release starts"},
		})
		activateDeps = append(activateDeps, OpWorkloadStop)
	}
	activate := Action{
		ID:                 OpReleaseActivate,
		OwnerOperation:     OpReleaseActivate,
		Effect:             EffectDeploymentWrite,
		RequiredCapability: "cloud-target:release.activate",
		Inputs: map[string]string{
			"release_digest":     in.Release.Digest,
			"release_id":         in.ReleaseArtifacts.ID,
			"release_dir":        lay.releaseDir,
			"scenarios":          strings.Join(targetScenarios, ","),
			"strategy":           in.Policy.MaintenanceStrategy,
			"ports":              ports,
			"data_bindings":      strings.Join(dataBindingSpecs(in), ","),
			"legacy_carry":       strings.Join(legacyCarry(in), ","),
			"legacy_root":        lay.workdir,
			"predecessor_digest": in.Observations.ActiveReleaseDigest,
		},
		DependsOn:    activateDeps,
		Verification: "runtime_ready_then_pointer_committed",
		Recovery:     "rollback_to_predecessor",
		Retry:        RetryObserveThenReplay,
		CancelPoint:  false,
	}
	actions = append(actions, activate)
	if in.Manifest.Edge.Caddy.Enabled || strings.TrimSpace(in.Manifest.Edge.Domain) != "" {
		actions = append(actions, Action{
			ID:                 OpEdgeRouteApply,
			OwnerOperation:     OpEdgeRouteApply,
			Effect:             EffectEdgeWrite,
			RequiredCapability: "cloud-target:edge.route-apply",
			Inputs: map[string]string{
				"domain":        strings.TrimSpace(in.Manifest.Edge.Domain),
				"route_host":    routeHost,
				"listener_id":   routeListener,
				"upstream_port": strconv.Itoa(routePort),
				"tls_email":     strings.TrimSpace(in.Manifest.Edge.Caddy.Email),
				"tls_enabled":   strconv.FormatBool(in.Manifest.Edge.Caddy.Enabled),
				"config_path":   "/etc/caddy/Caddyfile",
				"release_id":    in.ReleaseArtifacts.ID,
			},
			DependsOn:    []string{OpReleaseActivate},
			Verification: "caddy_config_validates",
			Recovery:     "restore_previous_caddyfile",
			Retry:        RetrySafeReplay,
			CancelPoint:  true,
		})
	}
	actions = append(actions, readinessAction(in, []string{OpReleaseActivate, OpEdgeRouteApply}))
	if update {
		actions = append(actions, Action{
			ID:                 OpReleaseRetainPredecesor,
			OwnerOperation:     OpReleaseRetainPredecesor,
			Effect:             EffectNone,
			RequiredCapability: "cloud-target:release.list",
			Inputs: map[string]string{
				"predecessor_digest": in.Observations.ActiveReleaseDigest,
				"release_id":         in.ReleaseArtifacts.ID,
				"rollback_eligible":  strconv.FormatBool(rollbackEligible(in)),
				"rollback_reason":    rollbackReason(in),
			},
			DependsOn:    []string{OpVerifyReadiness},
			Verification: "predecessor_retained",
			Recovery:     "none_required",
			Retry:        RetrySafeReplay,
			CancelPoint:  true,
		})
	}
	return actions
}

func readinessAction(in CompileInputs, deps []string) Action {
	checks := []string{"local"}
	if strings.TrimSpace(in.Manifest.Edge.Domain) != "" {
		checks = append(checks, "https", "origin", "public")
	}
	return Action{
		ID:                 OpVerifyReadiness,
		OwnerOperation:     OpVerifyReadiness,
		Effect:             EffectNone,
		RequiredCapability: "health:observe",
		Inputs: map[string]string{
			"checks":     strings.Join(checks, ","),
			"domain":     strings.TrimSpace(in.Manifest.Edge.Domain),
			"ui_port":    strconv.Itoa(in.Manifest.Ports["ui"]),
			"host":       strings.TrimSpace(in.Manifest.Target.VPS.Host),
			"release_id": in.ReleaseArtifacts.ID,
		},
		DependsOn:    deps,
		Verification: "readiness_checks_pass",
		Recovery:     "rollback_to_predecessor",
		Retry:        RetrySafeReplay,
		CancelPoint:  true,
	}
}

// retireActions is the terminal lifecycle in owner order: routing, runtime,
// grants, data, artifacts. Data and artifacts are retained unless the policy
// says otherwise; an irreversible data disposition requires the explicit
// policy validated in validateInputs.
func retireActions(in CompileInputs) []Action {
	lay := layoutFor(in)
	bindings := persistentBindings(in)
	legacy := legacyInventory(in)
	actions := []Action{
		{
			ID:                 OpEdgeRouteRetire,
			OwnerOperation:     OpEdgeRouteRetire,
			Effect:             EffectEdgeWrite,
			RequiredCapability: "cloud-target:edge.route-rollback",
			Inputs:             map[string]string{"domain": strings.TrimSpace(in.Manifest.Edge.Domain)},
			DependsOn:          []string{},
			Verification:       "route_absent",
			Recovery:           "none_required",
			Retry:              RetrySafeReplay,
			CancelPoint:        true,
		},
		{
			ID:                 OpWorkloadStop,
			OwnerOperation:     OpWorkloadStop,
			Effect:             EffectRuntimeStop,
			RequiredCapability: "privilegebroker:process.stop.scoped",
			Inputs:             map[string]string{"workdir": lay.workdir, "scenario": in.Manifest.Scenario.ID, "ports": portList(in.Manifest.Ports)},
			DependsOn:          []string{OpEdgeRouteRetire},
			Verification:       "scenario_stopped_shared_demand_kept",
			Recovery:           "none_required",
			Retry:              RetrySafeReplay,
			CancelPoint:        true,
		},
		{
			ID:                 OpGrantsRevoke,
			OwnerOperation:     OpGrantsRevoke,
			Effect:             EffectCredentialWrite,
			RequiredCapability: "credential-authority:revoke",
			Inputs:             map[string]string{"scenario": in.Manifest.Scenario.ID, "descriptors": strings.Join(credentialDescriptors(in.Manifest), ",")},
			DependsOn:          []string{OpWorkloadStop},
			Verification:       "grants_revoked",
			Recovery:           "none_required",
			Retry:              RetrySafeReplay,
			CancelPoint:        true,
		},
		{
			ID:                 OpDataRetire,
			OwnerOperation:     OpDataRetire,
			Effect:             dataRetireEffect(in),
			RequiredCapability: "cloud-target:data.retire",
			Inputs: map[string]string{
				"bindings":         strings.Join(bindings, ","),
				"legacy_preserve":  strings.Join(legacy, ","),
				"retention_policy": in.Policy.RetentionPolicy,
			},
			DependsOn:    []string{OpGrantsRevoke},
			Verification: "data_disposition_recorded",
			Recovery:     "none_required",
			Retry:        RetryObserveThenReplay,
			CancelPoint:  true,
		},
		{
			ID:                 OpArtifactsRetire,
			OwnerOperation:     OpArtifactsRetire,
			Effect:             EffectDeploymentWrite,
			RequiredCapability: "cloud-target:release.list",
			Inputs:             map[string]string{"release_id": in.ReleaseArtifacts.ID, "protected": "active,previous,backup"},
			DependsOn:          []string{OpDataRetire},
			Verification:       "protected_artifacts_retained",
			Recovery:           "none_required",
			Retry:              RetrySafeReplay,
			CancelPoint:        true,
		},
	}
	return actions
}

// stopActions is the intentional stop of the workload through the lifecycle
// owner. Nothing else changes: routes, data and artifacts stay.
func stopActions(in CompileInputs) []Action {
	lay := layoutFor(in)
	return []Action{{
		ID:                 OpWorkloadStop,
		OwnerOperation:     OpWorkloadStop,
		Effect:             EffectRuntimeStop,
		RequiredCapability: "privilegebroker:process.stop.scoped",
		Inputs:             map[string]string{"workdir": lay.workdir, "scenario": in.Manifest.Scenario.ID, "ports": portList(in.Manifest.Ports)},
		DependsOn:          []string{},
		Verification:       "scenario_stopped_shared_demand_kept",
		Recovery:           "none_required",
		Retry:              RetrySafeReplay,
		CancelPoint:        true,
	}}
}

func dataRetireEffect(in CompileInputs) string {
	if in.Policy.RetentionPolicy == RetentionDelete {
		return EffectDataWrite
	}
	return EffectNone
}

// preconditions records the material facts the plan binds to.
func preconditions(plan *Plan, in CompileInputs) []Precondition {
	pre := []Precondition{
		{Kind: PreconditionDeploymentRevision, Value: strconv.FormatUint(in.Observations.DeploymentRevision, 10)},
		{Kind: PreconditionTargetEnrollment, Value: strconv.FormatUint(in.Deployment.Target.EnrollmentGeneration, 10)},
		{Kind: PreconditionReleaseDigest, Value: plan.ReleaseDigest},
		{Kind: PreconditionClosureDigest, Value: plan.ClosureDigest},
		{Kind: PreconditionConfigurationDigest, Value: plan.ConfigurationDigest},
		{Kind: PreconditionDataSchema, Value: in.Observations.DataSchemaVersion},
		{Kind: PreconditionPrivilegeSet, Value: PrivilegeSet(plan.Actions, in.Closure)},
	}
	if backup := plan.Action(OpDataBackup); backup != nil && backup.Inputs["recovery_point_required"] != "" {
		pre = append(pre, Precondition{Kind: PreconditionRecoveryPoint, Value: backup.Inputs["recovery_point_required"]})
	}
	return pre
}

// PrivilegeSet is the sorted, comma-joined set of privileges a plan needs:
// every capability of an effectful action plus every closure privilege.
func PrivilegeSet(actions []Action, closure *domain.Closure) string {
	set := map[string]bool{}
	for _, action := range actions {
		if action.Effect != EffectNone && action.RequiredCapability != "" {
			set[action.RequiredCapability] = true
		}
	}
	if closure != nil {
		for _, privilege := range closure.Privileges {
			set["closure:"+privilege.Effect+":"+privilege.Subject] = true
		}
	}
	return strings.Join(sortedKeys(set), ",")
}

func presentationFor(plan *Plan, in CompileInputs) Presentation {
	pres := Presentation{}
	switch plan.Outcome {
	case OutcomeNoOp:
		pres.Title = fmt.Sprintf("%s is already at the desired release", plan.ScenarioID)
		pres.Summary = "The observed release, configuration and closure digests match the desired state; no action will run."
	case OutcomeNeedsInput:
		pres.Title = fmt.Sprintf("%s needs operator input before it can be deployed", plan.ScenarioID)
		pres.Summary = fmt.Sprintf("%d required input(s) are not satisfied; resume through the onboarding handoff.", len(plan.Handoff.Missing))
	default:
		verb := "Deploy"
		switch plan.Scope {
		case ScopeRetire:
			verb = "Retire"
		case ScopeStop:
			verb = "Stop"
		}
		pres.Title = fmt.Sprintf("%s %s (%s) on %s", verb, plan.ScenarioID, plan.Scope, strings.TrimSpace(in.Manifest.Target.VPS.Host))
		pres.Summary = fmt.Sprintf("%d actions: %s", len(plan.Actions), strings.Join(plan.ActionIDs(), ", "))
		if seconds := totalDowntime(plan.Actions); seconds > 0 {
			pres.DowntimeNote = fmt.Sprintf("Strategy %s: expect up to %d seconds of unavailability while the workload restarts.", in.Policy.MaintenanceStrategy, seconds)
		} else if plan.Scope == ScopeRetire {
			pres.DowntimeNote = "The workload stops permanently; shared dependencies other deployments demand keep running."
		} else {
			pres.DowntimeNote = fmt.Sprintf("Strategy %s: no downtime is declared for this plan.", in.Policy.MaintenanceStrategy)
		}
		switch {
		case plan.Scope == ScopeRetire:
			pres.RecoveryNote = fmt.Sprintf("Persistent data disposition: %s. Active, previous and backup artifacts and recovery points are retained.", in.Policy.RetentionPolicy)
		case plan.Action(OpReleaseRetainPredecesor) != nil:
			pres.RecoveryNote = fmt.Sprintf("Predecessor %s is retained; rollback eligibility: %s (%s). Staged trees that were never activated are discarded.", in.Observations.ActiveReleaseDigest, strconv.FormatBool(rollbackEligible(in)), rollbackReason(in))
		default:
			pres.RecoveryNote = "A failed activation or readiness check leaves the prior release active; staged trees that were never activated are discarded."
		}
	}
	return pres
}

func totalDowntime(actions []Action) int {
	total := 0
	for _, action := range actions {
		if action.Downtime != nil {
			total += action.Downtime.ExpectedSeconds
		}
	}
	return total
}

func targetScenarioIDs(manifest domain.CloudManifest) []string {
	ids := sortedUnique(manifest.Bundle.Scenarios)
	if len(ids) == 0 {
		ids = sortedUnique([]string{manifest.Scenario.ID})
	}
	return ids
}

// legacyInventory merges the manifest's preserve paths with the observed
// legacy inventory into one sorted list of scenarios/<id>/<relative> paths.
func legacyInventory(in CompileInputs) []string {
	set := map[string]bool{}
	for _, p := range in.Manifest.Target.VPS.PreservePaths {
		if p = path.Clean(strings.TrimSpace(p)); p != "" && p != "." {
			set[p] = true
		}
	}
	for _, p := range in.Observations.LegacyDataInventory {
		if p = path.Clean(strings.TrimSpace(p)); p != "" && p != "." {
			set[p] = true
		}
	}
	return sortedKeys(set)
}

// legacyCarry is the legacy inventory minus every path a recorded or
// declared binding covers: the heuristic applies only to unmapped paths.
func legacyCarry(in CompileInputs) []string {
	covered := map[string]bool{}
	for _, spec := range dataBindingSpecs(in) {
		if _, location, ok := strings.Cut(spec, "="); ok {
			covered["scenarios/"+location] = true
		}
	}
	var out []string
	for _, p := range legacyInventory(in) {
		skip := false
		for c := range covered {
			if p == c || strings.HasPrefix(p, c+"/") || strings.HasPrefix(c, p+"/") {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, strings.TrimPrefix(p, "scenarios/"))
		}
	}
	sort.Strings(out)
	return out
}

func persistentBindings(in CompileInputs) []string {
	if in.Closure == nil {
		return nil
	}
	set := map[string]bool{}
	for _, data := range in.Closure.PersistentData {
		set[data.ID+"@"+data.Owner] = true
	}
	return sortedKeys(set)
}

// dataBindingSpecs lists filesystem bindings as <id>=<scenario>/<relative>:
// the closure's dir bindings and the mappings recorded by the legacy
// conversion (recorded mappings win for the same id).
func dataBindingSpecs(in CompileInputs) []string {
	set := map[string]string{}
	if in.Closure != nil {
		for _, data := range in.Closure.PersistentData {
			kind, locator, ok := strings.Cut(strings.TrimSpace(data.Binding), ":")
			if !ok || kind != "dir" || strings.TrimSpace(data.Owner) == "" {
				continue
			}
			rel := strings.Trim(path.Clean("/"+locator), "/")
			if rel == "" || strings.Contains(locator, "..") {
				continue
			}
			set[data.ID] = data.Owner + "/" + rel
		}
	}
	for _, spec := range in.Observations.PersistentDataBindings {
		id, location, ok := strings.Cut(strings.TrimSpace(spec), "=")
		if !ok || strings.TrimSpace(id) == "" {
			continue
		}
		set[strings.TrimSpace(id)] = strings.TrimPrefix(strings.TrimSpace(location), "scenarios/")
	}
	out := make([]string, 0, len(set))
	for id, location := range set {
		out = append(out, id+"="+location)
	}
	sort.Strings(out)
	return out
}

// engineBindingSpecs renders the persistent data the recovery point must
// cover as <id>:<kind>:<locator>:<owner>:<migration_owner>: every declared
// binding plus every legacy path the heuristic still carries, so undeclared
// data is captured until it is declared or retired.
func engineBindingSpecs(in CompileInputs) []string {
	var out []string
	if in.Closure != nil {
		for _, data := range in.Closure.PersistentData {
			out = append(out, data.ID+":"+strings.TrimSpace(data.Binding)+":"+strings.TrimSpace(data.Owner)+":"+strings.TrimSpace(data.MigrationOwner))
		}
	}
	for _, p := range legacyInventory(in) {
		scenario, rel, ok := strings.Cut(strings.TrimPrefix(p, "scenarios/"), "/")
		if !ok {
			continue
		}
		id := "legacy-" + scenario + "-" + strings.ReplaceAll(rel, "/", "-")
		out = append(out, id+":dir:"+rel+":"+scenario+":scenario")
	}
	sort.Strings(out)
	return out
}

func migrationPosture(in CompileInputs, update bool) string {
	switch {
	case !update:
		return domain.MigrationPostureGreenfield
	case in.Observations.DataSchemaVersion != "":
		return domain.MigrationPostureProductionEvolution
	default:
		return domain.MigrationPostureGreenfieldWithData
	}
}

// rollbackEligible mirrors backup.EvaluateRollback for the preview: a
// predecessor is eligible when the scenario declares code rollback and the
// schema strategy keeps the current data readable by the predecessor.
func rollbackEligible(in CompileInputs) bool {
	return rollbackReason(in) == "predecessor_retained" || rollbackReason(in) == "no_schema"
}

func rollbackReason(in CompileInputs) string {
	recovery := scenarioRecovery(in)
	switch {
	case in.Observations.ActiveReleaseDigest == "":
		return "no_predecessor"
	case recovery != nil && recovery.CodeRollback == "none":
		return "code_rollback_not_declared"
	case len(persistentBindings(in)) == 0:
		return "no_schema"
	case recovery == nil || recovery.SchemaStrategy == "":
		return "schema_strategy_undeclared"
	case recovery.SchemaStrategy == "explicit_restore":
		return "explicit_restore_required"
	case recovery.SchemaStrategy == "none":
		return "no_schema"
	default:
		return "predecessor_retained"
	}
}

func scenarioRecovery(in CompileInputs) *domain.ClosureRecovery {
	if in.Closure == nil {
		return nil
	}
	for _, component := range in.Closure.ComponentsOfKind(domain.ClosureKindScenario) {
		if component.ID == in.Deployment.ScenarioID || component.ID == "scenario:"+in.Deployment.ScenarioID {
			return component.Recovery
		}
	}
	for _, component := range in.Closure.ComponentsOfKind(domain.ClosureKindScenario) {
		if component.Recovery != nil {
			return component.Recovery
		}
	}
	return nil
}

func credentialDescriptors(manifest domain.CloudManifest) []string {
	if manifest.Secrets == nil {
		return nil
	}
	set := map[string]bool{}
	for _, secret := range manifest.Secrets.BundleSecrets {
		ref := strings.TrimSpace(secret.ID) + ":" + strings.TrimSpace(secret.Class)
		if secret.Descriptor != nil {
			ref += ":" + strings.TrimSpace(secret.Descriptor.LogicalID) + ":" + strings.TrimSpace(secret.Descriptor.Field)
		}
		set[ref] = true
	}
	return sortedKeys(set)
}

func bundleSHA(in CompileInputs) string {
	return strings.TrimPrefix(in.Release.Digest, "sha256:")
}

func releaseDirName(digest string) string {
	name := strings.ReplaceAll(digest, ":", "-")
	name = strings.ReplaceAll(name, "/", "_")
	return name
}

func portList(ports domain.ManifestPorts) string {
	keys := make([]string, 0, len(ports))
	for key := range ports {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, ports[key]))
	}
	return strings.Join(parts, ",")
}

func remoteJoin(elem ...string) string {
	cleaned := make([]string, 0, len(elem))
	for _, e := range elem {
		if e = strings.TrimSpace(e); e != "" {
			cleaned = append(cleaned, e)
		}
	}
	return path.Clean(path.Join(cleaned...))
}

func sortedUnique(in []string) []string {
	set := map[string]bool{}
	for _, v := range in {
		if v = strings.TrimSpace(v); v != "" {
			set[v] = true
		}
	}
	return sortedKeys(set)
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// deriveEdgeRoute resolves the one public route from the closure's declared
// listeners through the edge policy (a scenario's public listener need not be
// named "ui"). A deployment without declared listeners keeps the launch
// contract: the manifest's ui port behind the apex host.
func deriveEdgeRoute(in CompileInputs, uiPort int) (host string, port int, listenerID string) {
	domainName := strings.TrimSpace(in.Manifest.Edge.Domain)
	listenerID = in.Deployment.ScenarioID + "/ui"
	if in.Closure != nil && len(in.Closure.Listeners) > 0 && domainName != "" {
		spec, err := edge.Derive(edge.PolicyInputs{
			DeploymentID:    in.Deployment.ID,
			ScenarioID:      in.Deployment.ScenarioID,
			Environment:     in.Deployment.Environment,
			Domain:          domainName,
			Listeners:       in.Closure.Listeners,
			Ports:           in.Manifest.Ports,
			ACMEEmail:       in.Manifest.Edge.Caddy.Email,
			ACMEEnvironment: in.Manifest.Edge.ACMEEnvironment,
		})
		if err == nil && len(spec.Routes) > 0 {
			r := spec.Routes[0]
			return r.Host, r.UpstreamPort, r.ListenerID
		}
	}
	return domainName, uiPort, listenerID
}
