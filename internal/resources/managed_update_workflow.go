package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

const (
	managedUpdateSchemaVersion   = 1
	managedUpdateProofMaxAge     = 10 * time.Second
	managedUpdateRestoreAttempts = 4
)

var (
	managedUpdateKey      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	managedUpdateResource = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
)

type ManagedUpdatePhase string

const (
	UpdatePrepared               ManagedUpdatePhase = "prepared"
	UpdateRollbackProbeStarting  ManagedUpdatePhase = "rollback_probe_starting"
	UpdateRollbackProbeVerifying ManagedUpdatePhase = "rollback_probe_verifying"
	UpdateCandidateStarting      ManagedUpdatePhase = "candidate_starting"
	UpdateCandidateVerifying     ManagedUpdatePhase = "candidate_verifying"
	UpdateCandidateReady         ManagedUpdatePhase = "candidate_ready"
	UpdatePrimaryStopping        ManagedUpdatePhase = "primary_stopping"
	UpdatePrimaryStarting        ManagedUpdatePhase = "primary_starting"
	UpdatePrimaryVerifying       ManagedUpdatePhase = "primary_verifying"
	UpdateCommitting             ManagedUpdatePhase = "committing"
	UpdatePreserving             ManagedUpdatePhase = "preserving"
	UpdatePreserveCommitting     ManagedUpdatePhase = "preserve_committing"
	UpdateRestoring              ManagedUpdatePhase = "restoring"
	UpdateRestoreStarting        ManagedUpdatePhase = "restore_starting"
	UpdateRestoreVerifying       ManagedUpdatePhase = "restore_verifying"
	UpdateRestoreCommitting      ManagedUpdatePhase = "restore_committing"
	UpdateRecoveryNeeded         ManagedUpdatePhase = "recovery_needed"
	UpdateSucceeded              ManagedUpdatePhase = "succeeded"
	UpdateFailedPreserved        ManagedUpdatePhase = "failed_preserved"
	UpdateFailedRestored         ManagedUpdatePhase = "failed_restored"
	UpdateAborted                ManagedUpdatePhase = "aborted"
)

// ManagedUpdateContract freezes source declarations and a verified artifact.
// It does not capture the caller's environment or resolve credential references.
type ManagedUpdateContract struct {
	Manifest ResourceManifest       `json:"manifest"`
	Artifact ManagedServiceArtifact `json:"artifact"`
	Digest   string                 `json:"digest"`
}

func newManagedUpdateContract(m ResourceManifest, artifact ManagedServiceArtifact) (ManagedUpdateContract, error) {
	encoded, err := json.Marshal(m)
	if err != nil {
		return ManagedUpdateContract{}, err
	}
	var frozen ResourceManifest
	if err := json.Unmarshal(encoded, &frozen); err != nil {
		return ManagedUpdateContract{}, err
	}
	c := ManagedUpdateContract{Manifest: frozen, Artifact: artifact}
	c.Digest = c.digest()
	return c, c.validate()
}

func (c ManagedUpdateContract) digest() string {
	data, _ := json.Marshal(struct {
		Manifest ResourceManifest       `json:"manifest"`
		Artifact ManagedServiceArtifact `json:"artifact"`
	}{c.Manifest, c.Artifact})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func updateDigest(s string) bool {
	if len(s) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(s)
	return err == nil && hex.EncodeToString(decoded) == s
}

func (c ManagedUpdateContract) validate() error {
	if err := manifestpkg.ValidateManagedServiceUpdate(c.Manifest); err != nil {
		return err
	}
	if c.Manifest.ManagedService == nil || !managedUpdateResource.MatchString(c.Manifest.Name) || c.Artifact.Resource != c.Manifest.Name || c.Artifact.Version == "" || c.Manifest.ManagedService.Artifact.Version != c.Artifact.Version || !filepath.IsAbs(c.Artifact.Path) || filepath.Clean(c.Artifact.Path) != c.Artifact.Path || !updateDigest(c.Artifact.SHA256) {
		return fmt.Errorf("update contract requires a retained resource artifact identity")
	}
	if !updateDigest(c.Digest) || c.Digest != c.digest() {
		return fmt.Errorf("update launch contract digest mismatch")
	}
	selected, err := c.Manifest.ManagedService.Artifact.ForCurrentPlatform()
	if err != nil || selected.SHA256 != c.Artifact.SHA256 {
		return fmt.Errorf("update launch artifact differs from the frozen platform contract")
	}
	return nil
}

// ManagedUpdateProof records an observation for one exact process. The effect
// owner must probe native identity and listener ownership again before a stop;
// serializing these fields alone does not prove anything about a live process.
type ManagedUpdateProof struct {
	PID            int                        `json:"pid"`
	Incarnation    string                     `json:"incarnation"`
	InstanceID     string                     `json:"instance_id"`
	ArtifactSHA256 string                     `json:"artifact_sha256"`
	ObservedAt     time.Time                  `json:"observed_at"`
	OwnedListener  bool                       `json:"owned_listener"`
	Ready          bool                       `json:"ready"`
	Metrics        []ManagedUpdateMetricProof `json:"metrics,omitempty"`
}

// ManagedUpdateLaunchIntent is persisted before a resource service dispatch.
// Its identity is independent of a bounded job's deadline: the only working
// connector must remain alive even if rollback needs operator intervention.
// The native service owner must bind this identity to its durable service unit.
// Only the token hash is retained here; the token is never a receipt field.
type ManagedUpdateLaunchIntent struct {
	InstanceID         string                 `json:"instance_id"`
	OwnershipTokenHash string                 `json:"ownership_token_hash"`
	ContractDigest     string                 `json:"contract_digest"`
	NativeIntent       platform.ServiceIntent `json:"native_intent"`
	Runtime            ManagedUpdateRuntime   `json:"runtime"`
}

type ManagedUpdateProcess struct {
	Intent     *ManagedUpdateLaunchIntent `json:"intent,omitempty"`
	State      *ManagedServiceState       `json:"state,omitempty"`
	Proof      *ManagedUpdateProof        `json:"proof,omitempty"`
	Quiescence *ManagedUpdateQuiescence   `json:"quiescence,omitempty"`
	NativeRef  *platform.ServiceRef       `json:"native_ref,omitempty"`
}

// Quiescence is an exact-identity observation, not a failed readiness probe.
// A dispatched intent without a process receipt still needs native unit and
// process exclusion before its identity may be retired.
type ManagedUpdateQuiescence struct {
	InstanceID         string    `json:"instance_id"`
	OwnershipTokenHash string    `json:"ownership_token_hash"`
	PID                int       `json:"pid,omitempty"`
	Incarnation        string    `json:"incarnation,omitempty"`
	ObservedAt         time.Time `json:"observed_at"`
	NativeAbsent       bool      `json:"native_absent"`
	ProcessesAbsent    bool      `json:"processes_absent"`
	ListenerAbsent     bool      `json:"listener_absent"`
}

type ManagedUpdateRetiredRestore struct {
	Process   ManagedUpdateProcess `json:"process"`
	RetiredAt time.Time            `json:"retired_at"`
}

// ManagedServiceUpdate is the durable checkpoint. Intent phases precede their
// effect. Reconciliation must observe the exact persisted launch identity,
// never redispatch because the caller failed to receive an acknowledgement.
type ManagedServiceUpdate struct {
	SchemaVersion                 int                              `json:"schema_version"`
	ID                            string                           `json:"id"`
	RequestID                     string                           `json:"request_id"`
	Resource                      string                           `json:"resource"`
	Revision                      uint64                           `json:"revision"`
	Phase                         ManagedUpdatePhase               `json:"phase"`
	CreatedAt                     time.Time                        `json:"created_at"`
	UpdatedAt                     time.Time                        `json:"updated_at"`
	SourceRevision                string                           `json:"source_revision"`
	RollbackOf                    string                           `json:"rollback_of,omitempty"`
	Target                        ManagedUpdateContract            `json:"target"`
	Rollback                      ManagedUpdateContract            `json:"rollback"`
	Credentials                   *ManagedUpdateCredentialSnapshot `json:"credentials,omitempty"`
	Capacity                      *ManagedUpdateCapacityBudget     `json:"capacity,omitempty"`
	RollbackQualificationRequired bool                             `json:"rollback_qualification_required"`
	RollbackQualifiedAt           *time.Time                       `json:"rollback_qualified_at,omitempty"`
	RollbackProbe                 ManagedUpdateProcess             `json:"rollback_probe"`
	RollbackProbeQuiescent        bool                             `json:"rollback_probe_quiescent"`
	Original                      ManagedServiceState              `json:"original"`
	OriginalScope                 *platform.ServiceScopeRef        `json:"original_scope,omitempty"`
	OriginalProof                 *ManagedUpdateProof              `json:"original_proof,omitempty"`
	Candidate                     ManagedUpdateProcess             `json:"candidate"`
	Primary                       ManagedUpdateProcess             `json:"primary"`
	Restored                      ManagedUpdateProcess             `json:"restored"`
	RetiredRestores               []ManagedUpdateRetiredRestore    `json:"retired_restores,omitempty"`
	Error                         string                           `json:"error,omitempty"`
	CandidateQuiescent            bool                             `json:"candidate_quiescent"`
	PrimaryStopVerified           bool                             `json:"primary_stop_verified"`
	PrimaryQuiescent              bool                             `json:"primary_quiescent"`
	AbortRequested                bool                             `json:"abort_requested"`
	RecoveryFrom                  ManagedUpdatePhase               `json:"recovery_from,omitempty"`
}

func (p ManagedUpdatePhase) Terminal() bool {
	switch p {
	case UpdateSucceeded, UpdateFailedPreserved, UpdateFailedRestored, UpdateAborted:
		return true
	}
	return false
}

func (u ManagedServiceUpdate) Validate() error {
	if u.RollbackOf != "" && (!managedUpdateKey.MatchString(u.RollbackOf) || u.RollbackOf == u.RequestID) {
		return fmt.Errorf("invalid retained rollback request identity")
	}
	if u.SchemaVersion != managedUpdateSchemaVersion || !managedUpdateKey.MatchString(u.ID) || !managedUpdateKey.MatchString(u.RequestID) || !managedUpdateResource.MatchString(u.Resource) || u.Revision == 0 || u.CreatedAt.IsZero() || u.UpdatedAt.Before(u.CreatedAt) || !updateDigest(u.SourceRevision) {
		return fmt.Errorf("invalid update checkpoint identity, version or timestamps")
	}
	if u.Capacity != nil {
		if err := u.Capacity.validate(u); err != nil {
			return err
		}
	}
	for _, c := range []ManagedUpdateContract{u.Target, u.Rollback} {
		if err := c.validate(); err != nil {
			return err
		}
		if c.Manifest.Name != u.Resource {
			return fmt.Errorf("update contract names a different resource")
		}
	}
	if u.Credentials != nil {
		if err := u.Credentials.validate(u); err != nil {
			return err
		}
	}
	if !validOriginalUpdateState(u.Original, u.Rollback) {
		return fmt.Errorf("update requires the exact retained primary identity")
	}
	if u.OriginalScope != nil {
		ref := u.OriginalScope
		_, ticks, found := strings.Cut(u.Original.Incarnation, ":")
		if ref.ValidateIdentity() != nil || ref.RootPID != u.Original.PID || !found || ref.RootStartTime != ticks || ref.Scope.Name != resourceServiceScopeName(u.Resource, "service") {
			return fmt.Errorf("original native service scope differs from the retained primary")
		}
	}
	if u.Original.UpdateLaunch != nil {
		if u.OriginalScope != nil || validateManagedUpdateServing(u.Original) != nil || u.Original.UpdateLaunch.Contract.Artifact.SHA256 != u.Rollback.Artifact.SHA256 {
			return fmt.Errorf("original recorded native launch differs from rollback or has two owners")
		}
	}
	if u.Target.Manifest.ManagedService.UpdatePolicy == nil {
		return fmt.Errorf("update target has no explicit policy")
	}
	if err := u.Target.Manifest.ManagedService.ValidateUpdatePolicy(); err != nil {
		return err
	}
	for _, p := range []struct {
		process  ManagedUpdateProcess
		contract ManagedUpdateContract
	}{{u.RollbackProbe, u.Rollback}, {u.Candidate, u.Target}, {u.Primary, u.Target}, {u.Restored, u.Rollback}} {
		if p.process.Intent != nil && !validUpdateIntent(p.process.Intent, p.contract) {
			return fmt.Errorf("invalid update launch intent")
		}
		if u.Capacity != nil && p.process.Intent != nil && p.process.Intent.Runtime.Limits.MemoryMaxBytes != managedUpdateRoleMemory {
			return fmt.Errorf("update role memory ceiling exceeds or differs from its reserved budget")
		}
		if p.process.State != nil && !validUpdateProcess(p.process, p.contract) {
			return fmt.Errorf("update process differs from its launch intent")
		}
		if p.process.NativeRef != nil && p.process.State == nil {
			return fmt.Errorf("native service receipt has no process observation")
		}
		if p.process.Proof != nil && (p.process.State == nil || !matchingUpdateProof(p.process.Proof, *p.process.State) || !managedUpdateMetricsQualified(p.process.Proof, p.contract)) {
			return fmt.Errorf("update proof differs from its process")
		}
	}
	if u.OriginalProof != nil && (!matchingUpdateProof(u.OriginalProof, u.Original) || !managedUpdateMetricsQualified(u.OriginalProof, u.Rollback)) {
		return fmt.Errorf("original proof differs from retained identity")
	}
	if u.Candidate.Quiescence != nil || u.Primary.Quiescence != nil || u.RollbackProbe.Quiescence != nil {
		return fmt.Errorf("restoration retirement evidence is attached to a different role")
	}
	if u.Restored.Quiescence != nil && (u.Phase != UpdateRestoring || !matchingUpdateQuiescence(u.Restored)) {
		return fmt.Errorf("restoration retirement lacks exact ownership evidence")
	}
	if len(u.RetiredRestores) >= managedUpdateRestoreAttempts {
		return fmt.Errorf("restoration attempt history exceeds its bound")
	}
	previousRetirement := u.CreatedAt
	for _, retired := range u.RetiredRestores {
		if retired.RetiredAt.Before(previousRetirement) || retired.RetiredAt.After(u.UpdatedAt) || !validUpdateIntent(retired.Process.Intent, u.Rollback) || retired.Process.State != nil && !validUpdateProcess(retired.Process, u.Rollback) || !freshUpdateQuiescence(retired.Process, retired.RetiredAt) {
			return fmt.Errorf("invalid retired restoration identity or quiescence")
		}
		if retired.Process.Proof != nil && (retired.Process.State == nil || !matchingUpdateProof(retired.Process.Proof, *retired.Process.State)) {
			return fmt.Errorf("retired restoration proof differs from its process")
		}
		previousRetirement = retired.RetiredAt
	}
	// A reused process/token would let one role accidentally stop another role.
	instances := map[string]bool{u.Original.InstanceID: true}
	tokens := map[string]bool{u.Original.OwnershipTokenHash: true}
	roles := []ManagedUpdateProcess{u.RollbackProbe, u.Candidate, u.Primary, u.Restored}
	for _, retired := range u.RetiredRestores {
		roles = append(roles, retired.Process)
	}
	for _, p := range roles {
		if p.Intent == nil {
			continue
		}
		if instances[p.Intent.InstanceID] || tokens[p.Intent.OwnershipTokenHash] {
			return fmt.Errorf("update process roles reuse an ownership identity")
		}
		instances[p.Intent.InstanceID], tokens[p.Intent.OwnershipTokenHash] = true, true
	}
	if !u.RollbackQualificationRequired && (u.RollbackProbe.Intent != nil || u.RollbackQualifiedAt != nil || u.RollbackProbeQuiescent) {
		return fmt.Errorf("unexpected rollback qualification state")
	}
	if u.RollbackQualifiedAt != nil {
		if u.RollbackQualifiedAt.Before(u.CreatedAt) || u.RollbackQualifiedAt.After(u.UpdatedAt) || !freshUpdateProof(u.RollbackProbe, u.Rollback, *u.RollbackQualifiedAt) {
			return fmt.Errorf("rollback qualification lacks its historical exact readiness proof")
		}
	}
	if u.RollbackQualificationRequired && u.Candidate.Intent != nil && u.RollbackQualifiedAt == nil {
		return fmt.Errorf("target candidate precedes required rollback qualification")
	}
	switch u.Phase {
	case UpdateRollbackProbeStarting:
		if !u.RollbackQualificationRequired || u.RollbackQualifiedAt != nil || !validUpdateIntent(u.RollbackProbe.Intent, u.Rollback) {
			return fmt.Errorf("rollback qualification launch lacks intent")
		}
	case UpdateRollbackProbeVerifying:
		if !u.RollbackQualificationRequired || u.RollbackQualifiedAt != nil || !validUpdateProcess(u.RollbackProbe, u.Rollback) {
			return fmt.Errorf("rollback qualification lacks native process")
		}
	case UpdatePrepared:
	case UpdateCandidateStarting:
		if u.Candidate.Intent == nil {
			return fmt.Errorf("candidate launch has no durable intent")
		}
	case UpdateCandidateVerifying, UpdateCandidateReady:
		if !validUpdateProcess(u.Candidate, u.Target) {
			return fmt.Errorf("candidate identity is absent")
		}
	case UpdatePrimaryStopping, UpdatePrimaryStarting, UpdatePrimaryVerifying, UpdateCommitting, UpdateSucceeded:
		if !validUpdateProcess(u.Candidate, u.Target) || u.Primary.Intent == nil {
			return fmt.Errorf("promotion lacks isolated candidate and primary intent")
		}
		if u.Phase != UpdatePrimaryStopping && !u.PrimaryStopVerified {
			return fmt.Errorf("promotion lacks original stop evidence")
		}
		if u.Phase == UpdatePrimaryVerifying || u.Phase == UpdateCommitting || u.Phase == UpdateSucceeded {
			if !validUpdateProcess(u.Primary, u.Target) {
				return fmt.Errorf("promoted primary identity is absent")
			}
		}
	case UpdatePreserving, UpdatePreserveCommitting, UpdateRestoring:
		if u.Error == "" {
			return fmt.Errorf("recovery lacks a recorded cause")
		}
	case UpdateRestoreStarting, UpdateRestoreVerifying, UpdateRestoreCommitting:
		if !u.PrimaryStopVerified || !u.PrimaryQuiescent || u.Restored.Intent == nil || u.Error == "" {
			return fmt.Errorf("restoration lacks prior quiescence and durable intent")
		}
		if u.Phase != UpdateRestoreStarting && !validUpdateProcess(u.Restored, u.Rollback) {
			return fmt.Errorf("restoration process identity is absent")
		}
	case UpdateRecoveryNeeded:
		if u.Error == "" || (u.RecoveryFrom != UpdatePreserving && u.RecoveryFrom != UpdateRestoring) {
			return fmt.Errorf("unresolved recovery has no continuation")
		}
	case UpdateFailedPreserved, UpdateFailedRestored, UpdateAborted:
	default:
		return fmt.Errorf("unknown update phase %q", u.Phase)
	}
	// Terminal evidence is checked at its own commit time, not wall-clock now:
	// an old receipt remains readable without becoming a current-health claim.
	if u.Phase.Terminal() {
		if u.RollbackProbe.Intent != nil && !u.RollbackProbeQuiescent {
			return fmt.Errorf("terminal update still has rollback qualification effects")
		}
		if !u.CandidateQuiescent {
			return fmt.Errorf("terminal update still has candidate effects")
		}
		switch u.Phase {
		case UpdateSucceeded:
			if u.AbortRequested || !freshUpdateProof(u.Primary, u.Target, u.UpdatedAt) {
				return fmt.Errorf("success lacks verified new primary")
			}
		case UpdateFailedPreserved:
			if u.Error == "" || u.PrimaryStopVerified || !freshStateUpdateProof(u.OriginalProof, u.Original, u.UpdatedAt) {
				return fmt.Errorf("preserved failure lacks serving original")
			}
		case UpdateFailedRestored:
			if u.Error == "" || !u.PrimaryStopVerified || !u.PrimaryQuiescent || !freshUpdateProof(u.Restored, u.Rollback, u.UpdatedAt) {
				return fmt.Errorf("restored failure lacks serving rollback")
			}
		case UpdateAborted:
			if !u.AbortRequested || u.Error == "" {
				return fmt.Errorf("abort lacks operator intent")
			}
			if u.PrimaryStopVerified {
				if !u.PrimaryQuiescent || !freshUpdateProof(u.Restored, u.Rollback, u.UpdatedAt) {
					return fmt.Errorf("abort lacks restored service")
				}
			} else if !freshStateUpdateProof(u.OriginalProof, u.Original, u.UpdatedAt) {
				return fmt.Errorf("abort lacks preserved service")
			}
		}
	}
	return nil
}

type ManagedUpdateEvent string

const (
	UpdateLaunchCandidate       ManagedUpdateEvent = "launch_candidate"
	UpdateLaunchRollbackProbe   ManagedUpdateEvent = "launch_rollback_probe"
	UpdateRollbackProbeStarted  ManagedUpdateEvent = "rollback_probe_started"
	UpdateRollbackProbeVerified ManagedUpdateEvent = "rollback_probe_verified"
	UpdateCandidateStarted      ManagedUpdateEvent = "candidate_started"
	UpdateCandidateVerified     ManagedUpdateEvent = "candidate_verified"
	UpdateStopPrimary           ManagedUpdateEvent = "stop_primary"
	UpdatePrimaryStopped        ManagedUpdateEvent = "primary_stopped"
	UpdatePrimaryStarted        ManagedUpdateEvent = "primary_started"
	UpdatePrimaryVerified       ManagedUpdateEvent = "primary_verified"
	UpdateFinishSuccess         ManagedUpdateEvent = "finish_success"
	UpdateFailure               ManagedUpdateEvent = "failure"
	UpdateOriginalVerified      ManagedUpdateEvent = "original_verified"
	UpdateFinishPreserved       ManagedUpdateEvent = "finish_preserved"
	UpdateRestoreRequired       ManagedUpdateEvent = "restore_required"
	UpdateLaunchRestore         ManagedUpdateEvent = "launch_restore"
	UpdateRestoreStarted        ManagedUpdateEvent = "restore_started"
	UpdateRestoreVerified       ManagedUpdateEvent = "restore_verified"
	UpdateFinishRestored        ManagedUpdateEvent = "finish_restored"
	UpdateRecoveryFailed        ManagedUpdateEvent = "recovery_failed"
	UpdateRetryRecovery         ManagedUpdateEvent = "retry_recovery"
	UpdateRetireRestore         ManagedUpdateEvent = "retire_restore"
	UpdateBeginAbort            ManagedUpdateEvent = "begin_abort"
)

func beforeUpdatePromotion(p ManagedUpdatePhase) bool {
	switch p {
	case UpdatePrepared, UpdateRollbackProbeStarting, UpdateRollbackProbeVerifying, UpdateCandidateStarting, UpdateCandidateVerifying, UpdateCandidateReady:
		return true
	}
	return false
}

func duringUpdatePromotion(p ManagedUpdatePhase) bool {
	switch p {
	case UpdatePrimaryStopping, UpdatePrimaryStarting, UpdatePrimaryVerifying, UpdateCommitting:
		return true
	}
	return false
}

// transitionManagedUpdate is pure. The state owner must perform compare-and-swap
// and preserve contracts/intent identity when committing its observations.
func transitionManagedUpdate(u ManagedServiceUpdate, event ManagedUpdateEvent, at time.Time) (ManagedServiceUpdate, error) {
	if err := u.Validate(); err != nil {
		return ManagedServiceUpdate{}, err
	}
	if u.Phase.Terminal() || at.Before(u.UpdatedAt) || u.Revision == ^uint64(0) {
		return ManagedServiceUpdate{}, fmt.Errorf("terminal, stale or exhausted update checkpoint")
	}
	next := u.Phase
	retiredRestore := false
	switch event {
	case UpdateLaunchRollbackProbe:
		if u.Phase == UpdatePrepared && u.RollbackQualificationRequired && u.RollbackQualifiedAt == nil && validUpdateIntent(u.RollbackProbe.Intent, u.Rollback) {
			next = UpdateRollbackProbeStarting
		}
	case UpdateRollbackProbeStarted:
		if u.Phase == UpdateRollbackProbeStarting && validUpdateProcess(u.RollbackProbe, u.Rollback) {
			next = UpdateRollbackProbeVerifying
		}
	case UpdateRollbackProbeVerified:
		if u.Phase == UpdateRollbackProbeVerifying && freshUpdateProof(u.RollbackProbe, u.Rollback, at) {
			u.RollbackQualifiedAt = &at
			next = UpdatePrepared
		}
	case UpdateLaunchCandidate:
		if u.Phase == UpdatePrepared && (!u.RollbackQualificationRequired || u.RollbackQualifiedAt != nil) && validUpdateIntent(u.Candidate.Intent, u.Target) {
			next = UpdateCandidateStarting
		}
	case UpdateCandidateStarted:
		if u.Phase == UpdateCandidateStarting && validUpdateProcess(u.Candidate, u.Target) {
			next = UpdateCandidateVerifying
		}
	case UpdateCandidateVerified:
		if u.Phase == UpdateCandidateVerifying && freshUpdateProof(u.Candidate, u.Target, at) {
			next = UpdateCandidateReady
		}
	case UpdateStopPrimary:
		if u.Phase == UpdateCandidateReady && validUpdateIntent(u.Primary.Intent, u.Target) && freshUpdateProof(u.Candidate, u.Target, at) {
			next = UpdatePrimaryStopping
		}
	case UpdatePrimaryStopped:
		if u.Phase == UpdatePrimaryStopping && u.PrimaryStopVerified {
			next = UpdatePrimaryStarting
		}
	case UpdatePrimaryStarted:
		if u.Phase == UpdatePrimaryStarting && validUpdateProcess(u.Primary, u.Target) {
			next = UpdatePrimaryVerifying
		}
	case UpdatePrimaryVerified:
		if u.Phase == UpdatePrimaryVerifying && freshUpdateProof(u.Primary, u.Target, at) {
			next = UpdateCommitting
		}
	case UpdateFinishSuccess:
		if u.Phase == UpdateCommitting && u.CandidateQuiescent && freshUpdateProof(u.Primary, u.Target, at) {
			next = UpdateSucceeded
		}
	case UpdateFailure, UpdateBeginAbort:
		if event == UpdateBeginAbort {
			u.AbortRequested = true
			u.Error = "operator requested abort"
		}
		if u.Error != "" {
			if beforeUpdatePromotion(u.Phase) {
				next = UpdatePreserving
			} else if duringUpdatePromotion(u.Phase) {
				next = UpdateRestoring
			}
		}
	case UpdateOriginalVerified:
		if u.Phase == UpdatePreserving && !u.PrimaryStopVerified && freshStateUpdateProof(u.OriginalProof, u.Original, at) {
			next = UpdatePreserveCommitting
		}
	case UpdateFinishPreserved:
		if u.Phase == UpdatePreserveCommitting && !u.PrimaryStopVerified && u.CandidateQuiescent && freshStateUpdateProof(u.OriginalProof, u.Original, at) {
			next = UpdateFailedPreserved
			if u.AbortRequested {
				next = UpdateAborted
			}
		}
	case UpdateRestoreRequired:
		if u.Phase == UpdatePreserving || u.Phase == UpdatePreserveCommitting {
			next = UpdateRestoring
		}
	case UpdateLaunchRestore:
		if u.Phase == UpdateRestoring && u.PrimaryStopVerified && u.PrimaryQuiescent && validUpdateIntent(u.Restored.Intent, u.Rollback) {
			next = UpdateRestoreStarting
		}
	case UpdateRestoreStarted:
		if u.Phase == UpdateRestoreStarting && validUpdateProcess(u.Restored, u.Rollback) {
			next = UpdateRestoreVerifying
		}
	case UpdateRestoreVerified:
		if u.Phase == UpdateRestoreVerifying && freshUpdateProof(u.Restored, u.Rollback, at) {
			next = UpdateRestoreCommitting
		}
	case UpdateFinishRestored:
		if u.Phase == UpdateRestoreCommitting && u.CandidateQuiescent && freshUpdateProof(u.Restored, u.Rollback, at) {
			next = UpdateFailedRestored
			if u.AbortRequested {
				next = UpdateAborted
			}
		}
	case UpdateRecoveryFailed:
		switch u.Phase {
		case UpdatePreserving, UpdatePreserveCommitting:
			u.RecoveryFrom = UpdatePreserving
			next = UpdateRecoveryNeeded
		case UpdateRestoring, UpdateRestoreStarting, UpdateRestoreVerifying, UpdateRestoreCommitting:
			u.RecoveryFrom = UpdateRestoring
			next = UpdateRecoveryNeeded
		}
	case UpdateRetryRecovery:
		if u.Phase == UpdateRecoveryNeeded {
			next = u.RecoveryFrom
			u.RecoveryFrom = ""
		}
	case UpdateRetireRestore:
		if u.Phase == UpdateRestoring && len(u.RetiredRestores) < managedUpdateRestoreAttempts-1 && freshUpdateQuiescence(u.Restored, at) && !u.Restored.Quiescence.ObservedAt.Before(u.UpdatedAt) {
			u.RetiredRestores = append(u.RetiredRestores, ManagedUpdateRetiredRestore{Process: u.Restored, RetiredAt: at})
			u.Restored = ManagedUpdateProcess{}
			retiredRestore = true
		}
	}
	if next == u.Phase && !retiredRestore {
		return ManagedServiceUpdate{}, fmt.Errorf("update event %q is not admissible in phase %q or lacks required evidence", event, u.Phase)
	}
	u.Phase = next
	u.Revision++
	u.UpdatedAt = at
	if err := u.Validate(); err != nil {
		return ManagedServiceUpdate{}, err
	}
	return u, nil
}

func validUpdateIntent(i *ManagedUpdateLaunchIntent, c ManagedUpdateContract) bool {
	return i != nil && managedUpdateKey.MatchString(i.InstanceID) && updateDigest(i.OwnershipTokenHash) && i.ContractDigest == c.Digest && i.NativeIntent.ValidateIdentity() == nil && i.NativeIntent.Owner == "resource-update" && i.NativeIntent.AttemptID == i.InstanceID && i.Runtime.validate() == nil
}

// The retained copy can have a different path from the running executable.
// Its bytes and version must still match the original witnessed launch.
func validOriginalUpdateState(s ManagedServiceState, c ManagedUpdateContract) bool {
	if !filepath.IsAbs(s.ArtifactPath) || filepath.Clean(s.ArtifactPath) != s.ArtifactPath {
		return false
	}
	c.Artifact.Path = s.ArtifactPath
	return validRetainedUpdateState(s, c)
}

func validRetainedUpdateState(s ManagedServiceState, c ManagedUpdateContract) bool {
	return s.PID > 0 && s.Incarnation != "" && managedUpdateKey.MatchString(s.InstanceID) && updateDigest(s.OwnershipTokenHash) && s.ArtifactSHA256 == c.Artifact.SHA256 && s.ArtifactVersion == c.Artifact.Version && s.ArtifactPath == c.Artifact.Path && !s.StartedAt.IsZero() && s.StoppedAt == nil
}

func validUpdateProcess(p ManagedUpdateProcess, c ManagedUpdateContract) bool {
	if !validUpdateIntent(p.Intent, c) || p.State == nil || !validRetainedUpdateState(*p.State, c) || p.State.InstanceID != p.Intent.InstanceID || p.State.OwnershipTokenHash != p.Intent.OwnershipTokenHash || p.NativeRef == nil || p.NativeRef.ValidateIdentity() != nil {
		return false
	}
	_, start, found := strings.Cut(p.State.Incarnation, ":")
	return found && p.NativeRef.Intent == platform.OperationIntent(p.Intent.NativeIntent) && p.NativeRef.RootPID == p.State.PID && p.NativeRef.RootStartTime == start
}

func matchingUpdateProof(p *ManagedUpdateProof, s ManagedServiceState) bool {
	return p != nil && p.PID == s.PID && p.Incarnation == s.Incarnation && p.InstanceID == s.InstanceID && p.ArtifactSHA256 == s.ArtifactSHA256 && !p.ObservedAt.IsZero()
}

func freshStateUpdateProof(p *ManagedUpdateProof, s ManagedServiceState, at time.Time) bool {
	return matchingUpdateProof(p, s) && p.Ready && p.OwnedListener && !p.ObservedAt.Before(s.StartedAt) && !p.ObservedAt.After(at) && at.Sub(p.ObservedAt) <= managedUpdateProofMaxAge
}

func freshUpdateProof(p ManagedUpdateProcess, c ManagedUpdateContract, at time.Time) bool {
	return validUpdateProcess(p, c) && freshStateUpdateProof(p.Proof, *p.State, at) && managedUpdateMetricsQualified(p.Proof, c)
}

func matchingUpdateQuiescence(p ManagedUpdateProcess) bool {
	q := p.Quiescence
	if p.Intent == nil || q == nil || q.InstanceID != p.Intent.InstanceID || q.OwnershipTokenHash != p.Intent.OwnershipTokenHash || q.ObservedAt.IsZero() {
		return false
	}
	if p.State == nil {
		return q.PID == 0 && q.Incarnation == ""
	}
	return q.PID == p.State.PID && q.Incarnation == p.State.Incarnation && !q.ObservedAt.Before(p.State.StartedAt)
}

func freshUpdateQuiescence(p ManagedUpdateProcess, at time.Time) bool {
	return matchingUpdateQuiescence(p) && p.Quiescence.NativeAbsent && p.Quiescence.ProcessesAbsent && p.Quiescence.ListenerAbsent && !p.Quiescence.ObservedAt.After(at) && at.Sub(p.Quiescence.ObservedAt) <= managedUpdateProofMaxAge
}
