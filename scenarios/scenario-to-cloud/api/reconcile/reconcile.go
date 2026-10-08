// Package reconcile is the one deployment reconciliation model: desired
// state (what the operator recorded, with its revision) and observed state
// (what the health and target owners reported, with its own timestamp) are
// compared into a typed drift report whose outcome is unchanged, changed,
// blocked or unknown. A correction is only ever proposed here; applying it
// goes through the executable plan and the durable operation owner, so
// observation never mutates a target and an intentional stop is never
// "repaired" into a restart.
package reconcile

import (
	"sort"
	"strings"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
)

// SchemaVersion is the wire shape of the drift report and retirement plan.
const SchemaVersion = "1"

// Outcome is the typed verdict every lifecycle entry point shares.
type Outcome string

// Outcomes.
const (
	Unchanged Outcome = "unchanged"
	Changed   Outcome = "changed"
	Blocked   Outcome = "blocked"
	Unknown   Outcome = "unknown"
)

// Desired is the operator's recorded intent for a deployment.
type Desired struct {
	DeploymentID        string              `json:"deployment_id"`
	Revision            uint64              `json:"desired_revision"`
	State               domain.DesiredState `json:"desired_state"`
	ReleaseDigest       string              `json:"release_digest"`
	ConfigurationDigest string              `json:"configuration_digest"`
	ClosureDigest       string              `json:"closure_digest"`
	RecordedAt          time.Time           `json:"recorded_at"`
}

// Observed is what the health observation and the target owner reported.
// Empty digests mean unobserved; Freshness and Status follow the health
// contract vocabulary and are never inferred.
type Observed struct {
	ReleaseDigest         string    `json:"observed_release_digest"`
	ConfigurationDigest   string    `json:"observed_configuration_digest"`
	Status                string    `json:"status"`
	Freshness             string    `json:"freshness"`
	ObservedAt            time.Time `json:"observed_at"`
	ActivationInterrupted bool      `json:"activation_interrupted"`
	// WorkloadRunning is nil when the process state was not observed.
	WorkloadRunning *bool `json:"workload_running,omitempty"`
	// ProducerRef names who produced the observation.
	ProducerRef string `json:"producer_ref,omitempty"`
}

// Correction is the proposed change. It names the plan scope to compile;
// the caller admits that plan by its reviewed digest through the operation
// owner. A nil correction means nothing may be applied.
type Correction struct {
	Kind   string `json:"kind"`
	Scope  string `json:"scope"`
	Reason string `json:"reason"`
}

// Correction kinds.
const (
	CorrectionApplyRelease     = "apply_release"
	CorrectionRestartWorkload  = "restart_workload"
	CorrectionStopWorkload     = "stop_workload"
	CorrectionResolveActivate  = "resolve_interrupted_activation"
	CorrectionObserveFirst     = "observe_first"
	CorrectionRetireExplicitly = "retire_explicitly"
)

// RebootPolicy is what happens to the workload when the target reboots,
// derived from the closure's supervision declaration and the desired state.
type RebootPolicy struct {
	Decision      string `json:"decision"`
	AutoRestart   bool   `json:"auto_restart"`
	StartupPolicy string `json:"startup_policy,omitempty"`
	Source        string `json:"source,omitempty"`
}

// Reboot decisions.
const (
	RebootLeaveStopped          = "leave_stopped"
	RebootSupervisorRestarts    = "supervisor_restarts"
	RebootOperatorStartRequired = "operator_start_required"
)

// DriftReport is the typed comparison.
type DriftReport struct {
	SchemaVersion string       `json:"schema_version"`
	DeploymentID  string       `json:"deployment_id"`
	Outcome       Outcome      `json:"outcome"`
	Reasons       []string     `json:"reasons"`
	Desired       Desired      `json:"desired"`
	Observed      Observed     `json:"observed"`
	Correction    *Correction  `json:"correction,omitempty"`
	Reboot        RebootPolicy `json:"reboot_policy"`
	ComputedAt    time.Time    `json:"computed_at"`
}

// Reasons.
const (
	ReasonDesiredStopped         = "desired_stopped"
	ReasonDesiredRetired         = "desired_retired"
	ReasonObservationStale       = "observation_stale"
	ReasonObservationUnknown     = "observation_unknown"
	ReasonReleaseUnobserved      = "release_unobserved"
	ReasonActivationInterrupted  = "activation_interrupted"
	ReasonReleaseDiffers         = "release_differs"
	ReasonConfigurationDiffers   = "configuration_differs"
	ReasonWorkloadUnhealthy      = "workload_unhealthy"
	ReasonWorkloadRunningStopped = "workload_running_but_desired_stopped"
	ReasonDesiredSatisfied       = "desired_state_observed"
)

// Detect compares desired and observed state. It is pure.
func Detect(desired Desired, observed Observed, supervision *domain.ClosureSupervision, now time.Time) DriftReport {
	report := DriftReport{SchemaVersion: SchemaVersion, DeploymentID: desired.DeploymentID, Desired: desired, Observed: observed, Reasons: []string{}, ComputedAt: now.UTC(), Reboot: Reboot(desired.State, supervision)}
	state := desired.State.Normalized()
	switch state {
	case domain.DesiredRetired:
		report.Outcome = Unchanged
		report.Reasons = append(report.Reasons, ReasonDesiredRetired)
		return report
	case domain.DesiredStopped:
		// An intentional stop is not drift: observation never proposes a
		// start. A workload still running is the only correction, and it is a
		// stop.
		report.Reasons = append(report.Reasons, ReasonDesiredStopped)
		if observed.WorkloadRunning != nil && *observed.WorkloadRunning {
			report.Outcome = Changed
			report.Reasons = append(report.Reasons, ReasonWorkloadRunningStopped)
			report.Correction = &Correction{Kind: CorrectionStopWorkload, Scope: execplan.ScopeStop, Reason: ReasonWorkloadRunningStopped}
			return report
		}
		report.Outcome = Unchanged
		return report
	}
	if observed.ActivationInterrupted {
		report.Outcome = Blocked
		report.Reasons = append(report.Reasons, ReasonActivationInterrupted)
		report.Correction = &Correction{Kind: CorrectionResolveActivate, Scope: execplan.ScopeRuntime, Reason: ReasonActivationInterrupted}
		return report
	}
	switch strings.ToLower(strings.TrimSpace(observed.Freshness)) {
	case "current":
	case "stale":
		report.Outcome = Unknown
		report.Reasons = append(report.Reasons, ReasonObservationStale)
		report.Correction = &Correction{Kind: CorrectionObserveFirst, Reason: ReasonObservationStale}
		return report
	default:
		report.Outcome = Unknown
		report.Reasons = append(report.Reasons, ReasonObservationUnknown)
		report.Correction = &Correction{Kind: CorrectionObserveFirst, Reason: ReasonObservationUnknown}
		return report
	}
	if strings.TrimSpace(observed.ReleaseDigest) == "" {
		report.Outcome = Unknown
		report.Reasons = append(report.Reasons, ReasonReleaseUnobserved)
		report.Correction = &Correction{Kind: CorrectionObserveFirst, Reason: ReasonReleaseUnobserved}
		return report
	}
	if !sameDigest(observed.ReleaseDigest, desired.ReleaseDigest) {
		report.Reasons = append(report.Reasons, ReasonReleaseDiffers)
	}
	if desired.ConfigurationDigest != "" && observed.ConfigurationDigest != "" && !sameDigest(observed.ConfigurationDigest, desired.ConfigurationDigest) {
		report.Reasons = append(report.Reasons, ReasonConfigurationDiffers)
	}
	if len(report.Reasons) > 0 {
		report.Outcome = Changed
		report.Correction = &Correction{Kind: CorrectionApplyRelease, Scope: execplan.ScopeRuntime, Reason: strings.Join(report.Reasons, ",")}
		return report
	}
	switch strings.ToLower(strings.TrimSpace(observed.Status)) {
	case "healthy", "degraded":
		report.Outcome = Unchanged
		report.Reasons = append(report.Reasons, ReasonDesiredSatisfied)
	case "unhealthy":
		report.Outcome = Changed
		report.Reasons = append(report.Reasons, ReasonWorkloadUnhealthy)
		report.Correction = &Correction{Kind: CorrectionRestartWorkload, Scope: execplan.ScopeStart, Reason: ReasonWorkloadUnhealthy}
	default:
		report.Outcome = Unknown
		report.Reasons = append(report.Reasons, ReasonObservationUnknown)
		report.Correction = &Correction{Kind: CorrectionObserveFirst, Reason: ReasonObservationUnknown}
	}
	return report
}

func sameDigest(a, b string) bool {
	norm := func(v string) string { return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(v), "sha256:")) }
	return norm(a) == norm(b)
}

// Reboot derives the reboot policy: a stopped intent stays stopped; a
// supervised workload with auto-restart declared comes back through the
// supervisor; anything else waits for an operator start.
func Reboot(state domain.DesiredState, supervision *domain.ClosureSupervision) RebootPolicy {
	policy := RebootPolicy{Decision: RebootOperatorStartRequired}
	if supervision != nil {
		policy.AutoRestart, policy.StartupPolicy, policy.Source = supervision.AutoRestart, supervision.StartupPolicy, supervision.AutoRestartSource
	}
	switch state.Normalized() {
	case domain.DesiredStopped, domain.DesiredRetired:
		policy.Decision = RebootLeaveStopped
	default:
		if policy.AutoRestart {
			policy.Decision = RebootSupervisorRestarts
		}
	}
	return policy
}

// OwnedObject is one thing a deployment owns, with its retirement disposition.
type OwnedObject struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Reason string `json:"reason,omitempty"`
}

// Object kinds in retirement order.
const (
	KindRoute         = "route"
	KindRuntime       = "runtime"
	KindGrant         = "grant"
	KindData          = "data"
	KindArtifact      = "artifact"
	KindRecoveryPoint = "recovery_point"
)

// RetirementOrder is the fixed owner order: routing, runtime, grants, data,
// artifacts.
var RetirementOrder = []string{KindRoute, KindRuntime, KindGrant, KindData, KindArtifact}

// RetirementInputs are the owned objects a retirement decides over.
type RetirementInputs struct {
	DeploymentID    string
	Domain          string
	Scenarios       []string
	CredentialRefs  []string
	DataBindings    []string
	LegacyData      []string
	RetentionPolicy string
	// ActiveRelease and PreviousRelease are the target's pointer; Releases
	// are every release the target holds.
	ActiveRelease   string
	PreviousRelease string
	Releases        []string
	// RecoveryPoints are the deployment's recovery points (ids) and
	// RecoveryPointReleases the release digests they reference.
	RecoveryPoints        []string
	RecoveryPointReleases []string
}

// RetirementPlan lists what a retirement removes and what it keeps, in
// owner order, plus why anything is blocked.
type RetirementPlan struct {
	SchemaVersion   string        `json:"schema_version"`
	DeploymentID    string        `json:"deployment_id"`
	Order           []string      `json:"order"`
	RetentionPolicy string        `json:"retention_policy"`
	Deleted         []OwnedObject `json:"deleted"`
	Retained        []OwnedObject `json:"retained"`
	Blocked         []string      `json:"blocked"`
	Outcome         Outcome       `json:"outcome"`
}

// Retirement decides the disposition of every owned object. Data needs an
// explicit policy when any binding exists; without one the plan is blocked
// rather than guessed. Active, previous and recovery-referenced artifacts
// and every recovery point are retained by construction.
func Retirement(in RetirementInputs) RetirementPlan {
	plan := RetirementPlan{SchemaVersion: SchemaVersion, DeploymentID: in.DeploymentID, Order: RetirementOrder, RetentionPolicy: in.RetentionPolicy, Deleted: []OwnedObject{}, Retained: []OwnedObject{}, Blocked: []string{}, Outcome: Changed}
	if strings.TrimSpace(in.Domain) != "" {
		plan.Deleted = append(plan.Deleted, OwnedObject{Kind: KindRoute, ID: in.Domain, Reason: "public route removed first so no traffic reaches a retiring workload"})
	}
	for _, scenario := range in.Scenarios {
		plan.Deleted = append(plan.Deleted, OwnedObject{Kind: KindRuntime, ID: scenario, Reason: "scoped lifecycle stop; shared resources other deployments demand keep running"})
	}
	for _, ref := range in.CredentialRefs {
		plan.Deleted = append(plan.Deleted, OwnedObject{Kind: KindGrant, ID: ref, Reason: "credential grant revoked through the authority"})
	}
	hasData := len(in.DataBindings) > 0 || len(in.LegacyData) > 0
	switch {
	case hasData && in.RetentionPolicy == execplan.RetentionDelete:
		for _, id := range append(append([]string{}, in.DataBindings...), in.LegacyData...) {
			plan.Deleted = append(plan.Deleted, OwnedObject{Kind: KindData, ID: id, Reason: "explicit retention_policy delete (irreversible); requires the target data retire owner"})
		}
		plan.Blocked = append(plan.Blocked, "irreversible data retirement has no target owner verb yet (cloud-target data retire); the plan refuses at data.retire")
	case hasData && in.RetentionPolicy == execplan.RetentionRetain:
		for _, id := range append(append([]string{}, in.DataBindings...), in.LegacyData...) {
			plan.Retained = append(plan.Retained, OwnedObject{Kind: KindData, ID: id, Reason: "retention_policy retain"})
		}
	case hasData:
		for _, id := range append(append([]string{}, in.DataBindings...), in.LegacyData...) {
			plan.Retained = append(plan.Retained, OwnedObject{Kind: KindData, ID: id, Reason: "no retention_policy declared; data is never deleted implicitly"})
		}
		plan.Blocked = append(plan.Blocked, "retention_policy (retain|delete) is required because the deployment holds persistent data")
	}
	protected := map[string]string{}
	if in.ActiveRelease != "" {
		protected[in.ActiveRelease] = "active release"
	}
	if in.PreviousRelease != "" {
		protected[in.PreviousRelease] = "rollback predecessor"
	}
	for _, digest := range in.RecoveryPointReleases {
		if digest != "" {
			if _, ok := protected[digest]; !ok {
				protected[digest] = "referenced by a recovery point"
			}
		}
	}
	releases := append([]string{}, in.Releases...)
	sort.Strings(releases)
	for _, digest := range releases {
		if reason, ok := protected[digest]; ok {
			plan.Retained = append(plan.Retained, OwnedObject{Kind: KindArtifact, ID: digest, Reason: reason})
			continue
		}
		plan.Deleted = append(plan.Deleted, OwnedObject{Kind: KindArtifact, ID: digest, Reason: "unreferenced staged release"})
	}
	for _, id := range in.RecoveryPoints {
		plan.Retained = append(plan.Retained, OwnedObject{Kind: KindRecoveryPoint, ID: id, Reason: "recovery points survive retirement; prune them through retention"})
	}
	if len(plan.Blocked) > 0 {
		plan.Outcome = Blocked
	}
	return plan
}

// CleanupProtection is the reference set cleanup must honour regardless of
// artifact age or lease state: the active and previous releases, every
// release a recovery point references, and every recovery point that a
// retained release or a non-terminal operation holds. A lease that expired
// while its owner crashed does not unprotect an artifact that is still
// referenced.
type CleanupProtection struct {
	ReleaseDigests []string `json:"release_digests"`
	RecoveryPoints []string `json:"recovery_points"`
}

// Protect computes the protection set from the target pointer, the
// recorded recovery points and the non-terminal operations.
func Protect(active, previous string, points []domain.RecoveryPoint, nonTerminalOperations []string) CleanupProtection {
	releases := map[string]bool{}
	for _, digest := range []string{active, previous} {
		if strings.TrimSpace(digest) != "" {
			releases[strings.TrimPrefix(digest, "sha256:")] = true
		}
	}
	protectedPoints := map[string]bool{}
	ops := map[string]bool{}
	for _, op := range nonTerminalOperations {
		ops[op] = true
	}
	for _, rp := range points {
		if rp.ReleaseDigest != "" {
			releases[strings.TrimPrefix(rp.ReleaseDigest, "sha256:")] = true
		}
		if rp.Protected || ops[rp.OperationID] || sameDigest(rp.ReleaseDigest, active) || sameDigest(rp.ReleaseDigest, previous) {
			protectedPoints[rp.ID] = true
		}
	}
	out := CleanupProtection{ReleaseDigests: []string{}, RecoveryPoints: []string{}}
	for digest := range releases {
		out.ReleaseDigests = append(out.ReleaseDigests, digest)
	}
	for id := range protectedPoints {
		out.RecoveryPoints = append(out.RecoveryPoints, id)
	}
	sort.Strings(out.ReleaseDigests)
	sort.Strings(out.RecoveryPoints)
	return out
}
