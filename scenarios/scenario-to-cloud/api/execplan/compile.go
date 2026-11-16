package execplan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
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
var HostPreparePackages = []string{"ca-certificates", "curl", "git", "gnupg", "iproute2", "jq", "lsb-release", "postgresql-client", "tar", "unzip"}

func hostPreparePackages(edgeEnabled bool) []string {
	packages := append([]string(nil), HostPreparePackages...)
	if edgeEnabled {
		packages = append(packages, "caddy")
	}
	return packages
}

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
	// Advisories are informational and apply to every outcome, including a
	// no-op: a deployment already at the desired release can still be missing
	// the credential one of its capabilities needs.
	plan.Advisories = capabilityAdvisories(in)

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
	if err := validateDeploymentInputs(in); err != nil {
		return err
	}
	if err := normalizeCompileScope(in); err != nil {
		return err
	}
	if err := normalizeCompilePolicy(in); err != nil {
		return err
	}
	if err := validateRetentionPolicy(in); err != nil {
		return err
	}
	return validateCompileClosure(in)
}

func validateDeploymentInputs(in *CompileInputs) error {
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
	return nil
}

func normalizeCompileScope(in *CompileInputs) error {
	switch in.Scope {
	case "":
		in.Scope = ScopeFull
	case ScopeFull, ScopeInstall, ScopeRuntime, ScopeStart, ScopeRetire, ScopeStop:
	default:
		return apierrors.Newf(apierrors.CodeInvalidRequest, "Unknown plan scope %q", in.Scope)
	}
	return nil
}

func normalizeCompilePolicy(in *CompileInputs) error {
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
	return nil
}

func validateRetentionPolicy(in *CompileInputs) error {
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
	return nil
}

func validateCompileClosure(in *CompileInputs) error {
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

// capabilityAdvisories reports declared capabilities whose secret nothing has
// satisfied. These are the optional secrets missingInputs deliberately skips:
// optional means "do not block", which until now also meant "do not mention",
// so a deployment could go out with its mail or payment provider unconfigured
// and nothing in the review said so.
//
// Only secrets that name a capability produce an advisory. A required secret
// is already a blocker and is not repeated here.
func capabilityAdvisories(in CompileInputs) []Advisory {
	if in.Manifest.Secrets == nil {
		return nil
	}
	satisfied := map[string]bool{}
	for _, key := range in.Observations.SatisfiedInputs {
		if key = strings.TrimSpace(key); key != "" {
			satisfied[key] = true
		}
	}
	var out []Advisory
	for _, secret := range in.Manifest.Secrets.BundleSecrets {
		capability := strings.TrimSpace(secret.Capability)
		if capability == "" || secret.Required {
			continue
		}
		envName := strings.TrimSpace(secret.Target.Name)
		address := envName
		if secret.Descriptor != nil {
			address = strings.TrimSpace(secret.Descriptor.LogicalID) + ":" + strings.TrimSpace(secret.Descriptor.Field)
		}
		if satisfied[envName] || satisfied[address] || satisfied[strings.TrimSpace(secret.ID)] {
			continue
		}
		id := strings.TrimSpace(secret.ID)
		if id == "" {
			id = envName
		}
		out = append(out, Advisory{
			ID:         "secret_unsatisfied:" + id,
			Severity:   AdvisorySeverityWarning,
			Capability: capability,
			Summary:    fmt.Sprintf("%s will not work: no value satisfies %s", capability, id),
			Detail:     strings.TrimSpace(secret.Description),
			Hint:       fmt.Sprintf("scenario-to-cloud secrets set %s --deployment %s --scenario %s", envName, in.Deployment.ID, in.Deployment.ScenarioID),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
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
