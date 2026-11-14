package coreactivation

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"

	platform "github.com/vrooli/platform-go"
)

var (
	ErrConflict      = errors.New("core activation: operation identity or state conflict")
	ErrClaimed       = errors.New("core activation: native execution already claimed or revoked")
	ErrFailedRelease = errors.New("core activation: failed release requires explicit retry")
	idPattern        = regexp.MustCompile(`^[a-f0-9]{32}$`)
	digestPattern    = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type Phase string

const (
	Prepared        Phase = "prepared"
	Publishing      Phase = "publishing"
	Activating      Phase = "activating"
	Verifying       Phase = "verifying"
	Restoring       Phase = "restoring"
	Verified        Phase = "verified"
	FailedPreserved Phase = "failed_preserved"
	FailedRestored  Phase = "failed_restored"
)

type Fence string

const (
	Ready   Fence = "ready"
	Claimed Fence = "claimed"
	Revoked Fence = "revoked"
)

// Record binds a single root activation to frozen artifacts and native facts.
// There are no effectful paths or raw child diagnostics in the JSON record.
// Artifact paths are derived from Store's home and ID by their domain adapter.
type Record struct {
	SchemaVersion          int                      `json:"schema_version"`
	ID                     string                   `json:"id"`
	Home                   string                   `json:"home"`
	SourceRoot             string                   `json:"source_root"`
	Revision               uint64                   `json:"revision"`
	Phase                  Phase                    `json:"phase"`
	Fence                  Fence                    `json:"fence"`
	CandidateSHA           string                   `json:"candidate_sha256"`
	CandidateBuild         string                   `json:"candidate_build_identity"`
	CandidateStateContract string                   `json:"candidate_state_contract"`
	CandidateUnitSHA       string                   `json:"candidate_unit_sha256"`
	LayoutSHA              string                   `json:"layout_sha256"`
	WorkerSHA              string                   `json:"worker_sha256"`
	PriorSHA               string                   `json:"prior_sha256"`
	PriorBuild             string                   `json:"prior_build_identity"`
	PriorStateContract     string                   `json:"prior_state_contract"`
	PriorUnitSHA           string                   `json:"prior_unit_sha256"`
	PriorPID               int                      `json:"prior_pid"`
	PriorIncarnation       string                   `json:"prior_incarnation"`
	PriorNativeInvocation  string                   `json:"prior_native_invocation"`
	NativeOverridesDigest  string                   `json:"native_overrides_sha256"`
	CreatedUnixNano        int64                    `json:"created_unix_nano"`
	DeadlineUnixNano       int64                    `json:"deadline_unix_nano"`
	WorkerIntent           platform.OperationIntent `json:"worker_intent"`
	DispatchCommitted      bool                     `json:"dispatch_committed"`
	Automatic              bool                     `json:"automatic"`
	WorkerRef              *platform.OperationRef   `json:"worker_ref,omitempty"`
	FailureClass           string                   `json:"failure_class"`
	RestorationRequired    bool                     `json:"restoration_required"`
	FailedReleases         []string                 `json:"failed_releases"`
	FailedBuilds           []string                 `json:"failed_builds"`
}

func (r Record) Terminal() bool {
	return r.Phase == Verified || r.Phase == FailedPreserved || r.Phase == FailedRestored
}

func (r Record) validate(home string) error {
	if r.SchemaVersion != recordSchemaVersion || !idPattern.MatchString(r.ID) || r.Home != home || !filepath.IsAbs(r.SourceRoot) || filepath.Clean(r.SourceRoot) != r.SourceRoot || r.Revision == 0 {
		return fmt.Errorf("%w: invalid record binding", ErrConflict)
	}
	for _, digest := range []string{r.CandidateSHA, r.CandidateBuild, r.CandidateStateContract, r.CandidateUnitSHA, r.LayoutSHA, r.WorkerSHA, r.PriorSHA, r.PriorBuild, r.PriorStateContract, r.PriorUnitSHA, r.NativeOverridesDigest} {
		if !digestPattern.MatchString(digest) {
			return fmt.Errorf("%w: incomplete artifact or state contract", ErrConflict)
		}
	}
	if r.CandidateStateContract != r.PriorStateContract {
		return fmt.Errorf("%w: persistent-state compatibility is not qualified", ErrConflict)
	}
	if r.PriorPID <= 0 || !validIncarnation(r.PriorIncarnation) || !idPattern.MatchString(r.PriorNativeInvocation) {
		return fmt.Errorf("%w: prior native identity is incomplete", ErrConflict)
	}
	if r.CreatedUnixNano <= 0 || r.DeadlineUnixNano <= r.CreatedUnixNano || r.DeadlineUnixNano-r.CreatedUnixNano > int64(5*60*1000000000) {
		return fmt.Errorf("%w: invalid activation deadline", ErrConflict)
	}
	if err := r.WorkerIntent.ValidateIdentity(); err != nil {
		return err
	}
	if r.WorkerIntent.Kind != "" || r.WorkerIntent.Owner != "core-activation" || r.WorkerIntent.AttemptID != r.ID {
		return fmt.Errorf("%w: worker intent belongs to another operation", ErrConflict)
	}
	if r.WorkerRef != nil {
		if !r.DispatchCommitted || r.WorkerRef.RootPID <= 0 {
			return fmt.Errorf("%w: worker identity lacks a committed live claim", ErrConflict)
		}
		if err := r.WorkerRef.ValidateIdentity(); err != nil {
			return err
		}
		if r.WorkerRef.Intent != r.WorkerIntent || !idPattern.MatchString(r.WorkerRef.NativeInvocationID) {
			return fmt.Errorf("%w: worker native identity changed", ErrConflict)
		}
	}
	switch r.Phase {
	case Prepared, Publishing, Activating, Verifying, Restoring, Verified, FailedPreserved, FailedRestored:
	default:
		return fmt.Errorf("%w: invalid phase", ErrConflict)
	}
	switch r.Fence {
	case Ready:
		if r.Phase != Prepared || r.WorkerRef != nil {
			return fmt.Errorf("%w: unclaimed operation has execution state", ErrConflict)
		}
	case Claimed:
		if !r.DispatchCommitted || r.WorkerRef == nil || r.WorkerRef.RootPID <= 0 || r.WorkerRef.RootStartTime == "" || r.Terminal() || r.Phase == Restoring {
			return fmt.Errorf("%w: claimed worker is incomplete", ErrConflict)
		}
	case Revoked:
		if !r.Terminal() && r.Phase != Restoring {
			return fmt.Errorf("%w: revoked operation is not recovering", ErrConflict)
		}
	default:
		return fmt.Errorf("%w: invalid exec fence", ErrConflict)
	}
	if r.Terminal() && r.Fence != Revoked {
		return fmt.Errorf("%w: terminal operation permits future effects", ErrConflict)
	}
	if r.Phase == Verified && (r.WorkerRef == nil || !r.DispatchCommitted) {
		return fmt.Errorf("%w: verified activation lacks its native worker", ErrConflict)
	}
	if r.Phase == Restoring && r.FailureClass == "" || !r.Terminal() && r.Phase != Restoring && r.FailureClass != "" {
		return fmt.Errorf("%w: failure state contradicts the active phase", ErrConflict)
	}
	if r.Phase == Verified && r.FailureClass != "" || r.Phase == FailedPreserved && r.FailureClass == "" || r.Phase == FailedRestored && r.FailureClass == "" {
		return fmt.Errorf("%w: contradictory outcome", ErrConflict)
	}
	if r.RestorationRequired && r.Phase != Restoring && r.Phase != FailedRestored || r.Phase == FailedRestored && !r.RestorationRequired {
		return fmt.Errorf("%w: contradictory restoration intent", ErrConflict)
	}
	switch r.FailureClass {
	case "", "publication", "activation", "verification", "worker_lost", "deadline":
	default:
		return fmt.Errorf("%w: invalid failure class", ErrConflict)
	}
	if len(r.FailedReleases) > 32 || len(r.FailedBuilds) > 32 {
		return fmt.Errorf("%w: failure history bound exceeded", ErrConflict)
	}
	seen := map[string]bool{}
	builds := map[string]bool{}
	for _, build := range r.FailedBuilds {
		if !digestPattern.MatchString(build) || builds[build] {
			return fmt.Errorf("%w: invalid failed-build history", ErrConflict)
		}
		builds[build] = true
	}
	for _, sha := range r.FailedReleases {
		if !digestPattern.MatchString(sha) || seen[sha] {
			return fmt.Errorf("%w: invalid failed-release history", ErrConflict)
		}
		seen[sha] = true
	}
	if (r.Phase == FailedPreserved || r.Phase == FailedRestored) && !seen[r.CandidateSHA] {
		return fmt.Errorf("%w: failed release lacks suppression", ErrConflict)
	}
	if (r.Phase == FailedPreserved || r.Phase == FailedRestored) && !builds[r.CandidateBuild] {
		return fmt.Errorf("%w: failed build lacks suppression", ErrConflict)
	}
	return nil
}

func validIncarnation(s string) bool {
	if len(s) < 38 || s[36] != ':' {
		return false
	}
	uuid := s[:36]
	for i, c := range uuid {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	n, err := strconv.ParseUint(s[37:], 10, 64)
	return err == nil && n > 0
}

// Effects belongs to the native adapter. Each effect must compare frozen
// digests/native identities before mutation, contain its child process tree,
// and return only secret-safe diagnostics. ExcludeWorker must prove both native
// quiescence and that a late delivery cannot execute after the durable revoke.
// RestorePrior must be idempotent: a lost acknowledgement causes the same
// frozen restoration to be retried, including when the prior image is healthy.
type Effects interface {
	// QualifyAdmission must verify frozen artifact/native facts and derive
	// compatibility from the actual compiled persistent-state owners. Equal
	// caller-supplied digest strings are necessary metadata, never that proof.
	QualifyAdmission(context.Context, Record) error
	AuthorizeWorker(context.Context, Record, platform.OperationRef) error
	PublishCandidate(context.Context, Record) error
	ActivateCandidate(context.Context, Record) error
	VerifyCandidate(context.Context, Record) error
	ExcludeWorker(context.Context, Record) error
	VerifyPrior(context.Context, Record) error
	RestorePrior(context.Context, Record) error
	RetireInputs(context.Context, Record) error
}
