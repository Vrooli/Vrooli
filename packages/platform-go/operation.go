package platform

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	ErrOperationExists       = errors.New("platform: operation identity is already in use")
	ErrOperationNotFound     = errors.New("platform: owned operation is not present")
	ErrOperationIdentity     = errors.New("platform: owned operation identity mismatch")
	ErrOperationOutcomeLost  = errors.New("platform: operation outcome is unavailable")
	ErrOperationNotQuiescent = errors.New("platform: operation process tree is not quiescent")
)

const (
	// OperationsSlice contains manager-owned bounded local operations.
	OperationsSlice         = "vrooli-operations.slice"
	maxOperationMemoryBytes = int64(1 << 40)
	maxOperationTasks       = 65536
	maxOperationArgs        = 1024
	maxOperationStringBytes = 64 << 10
	maxOperationEnvBytes    = 64 << 10
	maxOperationWallTime    = 24 * time.Hour
	minOperationWallTime    = time.Millisecond
	operationWaitPoll       = 100 * time.Millisecond
)

var (
	operationOwnerPattern   = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)
	operationAttemptPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,95}$`)
	operationInvocation     = regexp.MustCompile(`^[a-f0-9]{32}$`)
)

// OperationLimits are mandatory ceilings for an owned native operation.
// CPUWeight is the platform's relative CPU share, not a wall-clock limit.
type OperationLimits struct {
	MemoryMaxBytes int64         `json:"memory_max_bytes"`
	TasksMax       int           `json:"tasks_max"`
	CPUWeight      int           `json:"cpu_weight"`
	WallTime       time.Duration `json:"wall_time_ns"`
}

// OperationSpec is an already-resolved, fixed executable invocation. Callers
// must build argv from typed domain inputs; this type never invokes a shell.
// Env must be non-nil to make inherited environment use an explicit choice.
type OperationSpec struct {
	Path   string
	Args   []string
	Env    []string
	Dir    string
	Limits OperationLimits
}

// OperationIntent is the caller-persisted, pre-dispatch identity. Persist it
// durably before calling StartOwnedOperation and never reuse its attempt or
// invocation ID. SpecDigest binds the intent to the resolved invocation.
type OperationIntent struct {
	Owner        string `json:"owner"`
	AttemptID    string `json:"attempt_id"`
	InvocationID string `json:"invocation_id"`
	SpecDigest   string `json:"spec_digest"`
}

// OperationRef is the exact native operation returned by the owner. Methods
// that inspect or stop work require this ref and reject a later invocation of
// the same unit. NativeInvocationID is required; cgroup inode identity is
// additional evidence when the service still has a cgroup.
type OperationRef struct {
	Intent             OperationIntent `json:"intent"`
	Unit               string          `json:"unit"`
	Scope              ScopeRef        `json:"scope"`
	RootPID            int             `json:"root_pid,omitempty"`
	RootStartTime      string          `json:"root_start_time,omitempty"`
	NativeInvocationID string          `json:"native_invocation_id"`
	ScopeDevice        uint64          `json:"scope_device,omitempty"`
	ScopeInode         uint64          `json:"scope_inode,omitempty"`
}

type OperationState string

const (
	OperationRunning             OperationState = "running"
	OperationStoppedOutcomeKnown OperationState = "stopped_outcome_known"
	OperationOutcomeUnknown      OperationState = "outcome_unknown"
)

type OperationObservation struct {
	Ref            OperationRef   `json:"ref"`
	State          OperationState `json:"state"`
	Populated      bool           `json:"populated"`
	RootAlive      bool           `json:"root_alive"`
	Result         string         `json:"result,omitempty"`
	ExecMainCode   int            `json:"exec_main_code,omitempty"`
	ExecMainStatus int            `json:"exec_main_status,omitempty"`
	ExitKnown      bool           `json:"exit_known"`
}

type LauncherReapState string

const (
	LauncherReapUnknown LauncherReapState = "unknown"
	LauncherReaped      LauncherReapState = "reaped"
)

// OperationStopReceipt separates native descendant quiescence from whether
// this API process reaped its short-lived systemd-run launcher. It never
// claims the manager-owned operation's root was locally reaped.
type OperationStopReceipt struct {
	Ref             OperationRef      `json:"ref"`
	NativeQuiescent bool              `json:"native_quiescent"`
	LauncherReap    LauncherReapState `json:"launcher_reap"`
}

// OwnedOperation is a handle to a manager-owned transient service. Its
// durable identity is Ref; it does not create or schedule work.
type OwnedOperation struct {
	Ref            OperationRef
	launcherReaped bool
}

// NewOperationIntent creates an invocation token and binds it to a complete
// resolved spec. Persist the returned value before dispatch.
func NewOperationIntent(owner, attemptID string, spec OperationSpec) (OperationIntent, error) {
	if err := validateOperationSpec(spec); err != nil {
		return OperationIntent{}, err
	}
	if !operationOwnerPattern.MatchString(owner) {
		return OperationIntent{}, fmt.Errorf("platform: invalid operation owner %q", owner)
	}
	if !operationAttemptPattern.MatchString(attemptID) {
		return OperationIntent{}, fmt.Errorf("platform: invalid operation attempt ID %q", attemptID)
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return OperationIntent{}, fmt.Errorf("platform: generate operation invocation ID: %w", err)
	}
	return OperationIntent{
		Owner: owner, AttemptID: attemptID, InvocationID: hex.EncodeToString(random),
		SpecDigest: operationSpecDigest(spec),
	}, nil
}

func validateOperationIntent(intent OperationIntent, spec OperationSpec) error {
	if !operationOwnerPattern.MatchString(intent.Owner) || !operationAttemptPattern.MatchString(intent.AttemptID) || !operationInvocation.MatchString(intent.InvocationID) {
		return fmt.Errorf("%w: malformed operation intent", ErrOperationIdentity)
	}
	if len(intent.SpecDigest) != sha256.Size*2 || strings.ToLower(intent.SpecDigest) != intent.SpecDigest {
		return fmt.Errorf("%w: malformed specification digest", ErrOperationIdentity)
	}
	if err := validateOperationSpec(spec); err != nil {
		return err
	}
	if operationSpecDigest(spec) != intent.SpecDigest {
		return fmt.Errorf("%w: specification digest changed", ErrOperationIdentity)
	}
	return nil
}

func validateOperationSpec(spec OperationSpec) error {
	if !filepath.IsAbs(spec.Path) || filepath.Clean(spec.Path) != spec.Path || strings.ContainsRune(spec.Path, '\x00') {
		return errors.New("platform: operation executable must be an absolute, clean path")
	}
	if !filepath.IsAbs(spec.Dir) || filepath.Clean(spec.Dir) != spec.Dir || strings.ContainsRune(spec.Dir, '\x00') {
		return errors.New("platform: operation working directory must be an absolute, clean path")
	}
	if spec.Env == nil {
		return errors.New("platform: operation environment must be explicit (use an empty slice to inherit nothing)")
	}
	if len(spec.Args) > maxOperationArgs {
		return fmt.Errorf("platform: operation has too many arguments: %d", len(spec.Args))
	}
	argBytes := 0
	for _, arg := range spec.Args {
		if strings.ContainsRune(arg, '\x00') {
			return errors.New("platform: operation argument contains NUL")
		}
		argBytes += len(arg)
		if argBytes > maxOperationStringBytes {
			return errors.New("platform: operation arguments exceed size bound")
		}
	}
	envBytes := 0
	seen := make(map[string]struct{}, len(spec.Env))
	for _, entry := range spec.Env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || key == "" || strings.ContainsAny(key, "=\x00") || strings.ContainsRune(value, '\x00') {
			return fmt.Errorf("platform: invalid operation environment entry for key %q", key)
		}
		if _, exists := seen[key]; exists {
			return fmt.Errorf("platform: duplicate operation environment key %q", key)
		}
		seen[key] = struct{}{}
		envBytes += len(entry)
		if envBytes > maxOperationEnvBytes {
			return errors.New("platform: operation environment exceeds size bound")
		}
	}
	limits := spec.Limits
	if limits.MemoryMaxBytes <= 0 || limits.MemoryMaxBytes > maxOperationMemoryBytes {
		return fmt.Errorf("platform: operation memory limit must be between 1 and %d bytes", maxOperationMemoryBytes)
	}
	if limits.TasksMax < minimumTasksMax || limits.TasksMax > maxOperationTasks {
		return fmt.Errorf("platform: operation TasksMax must be between %d and %d", minimumTasksMax, maxOperationTasks)
	}
	if limits.CPUWeight < 1 || limits.CPUWeight > 10000 {
		return errors.New("platform: operation CPUWeight must be between 1 and 10000")
	}
	if limits.WallTime < minOperationWallTime || limits.WallTime > maxOperationWallTime {
		return fmt.Errorf("platform: operation wall time must be between %s and %s", minOperationWallTime, maxOperationWallTime)
	}
	return nil
}

func operationSpecDigest(spec OperationSpec) string {
	canonical := struct {
		Path   string          `json:"path"`
		Args   []string        `json:"args"`
		Env    []string        `json:"env"`
		Dir    string          `json:"dir"`
		Limits OperationLimits `json:"limits"`
	}{spec.Path, spec.Args, spec.Env, spec.Dir, spec.Limits}
	data, _ := json.Marshal(canonical)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func operationUnitName(intent OperationIntent) string {
	data, _ := json.Marshal(intent)
	sum := sha256.Sum256(data)
	return "vrooli-op-" + intent.Owner + "-" + hex.EncodeToString(sum[:12]) + ".service"
}

// StartOwnedOperation starts one pre-persisted operation under the native
// operation owner. The caller owns durable job state and must never reuse the
// intent, including after an ambiguous start result.
func StartOwnedOperation(ctx context.Context, intent OperationIntent, spec OperationSpec) (*OwnedOperation, error) {
	if ctx == nil {
		return nil, errors.New("platform: nil operation start context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateOperationIntent(intent, spec); err != nil {
		return nil, err
	}
	return startOwnedOperation(ctx, intent, spec)
}

// LookupOwnedOperation resolves an operation from the intent that was
// persisted before dispatch. It can recover native tree identity after an API
// restart, but it cannot recover an exit code or claim successful output.
func LookupOwnedOperation(ctx context.Context, intent OperationIntent) (OperationRef, error) {
	if ctx == nil {
		return OperationRef{}, errors.New("platform: nil operation lookup context")
	}
	if err := validateOperationIntentShape(intent); err != nil {
		return OperationRef{}, err
	}
	return lookupOwnedOperation(ctx, intent)
}

// InspectOwnedOperation reads only the exact operation ref. A manager error,
// unsupported identity fact, or mismatched invocation is an error, never a
// not-running result.
func InspectOwnedOperation(ctx context.Context, expected OperationRef) (OperationObservation, error) {
	if ctx == nil {
		return OperationObservation{}, errors.New("platform: nil operation inspect context")
	}
	if err := validateOperationRefShape(expected); err != nil {
		return OperationObservation{}, err
	}
	return inspectOwnedOperation(ctx, expected)
}

// StopOwnedOperation signals the exact native tree and waits for recursive
// cgroup quiescence. LauncherReap remains unknown after API recovery.
func StopOwnedOperation(ctx context.Context, expected OperationRef) (OperationStopReceipt, error) {
	if ctx == nil {
		return OperationStopReceipt{}, errors.New("platform: nil operation stop context")
	}
	if err := validateOperationRefShape(expected); err != nil {
		return OperationStopReceipt{}, err
	}
	return stopOwnedOperation(ctx, expected)
}

// ReleaseOwnedOperation unloads a retained terminal unit after the caller has
// durably recorded output and terminal job evidence. It destroys manager-side
// recovery evidence and is never an implicit part of stop.
func ReleaseOwnedOperation(ctx context.Context, expected OperationRef) error {
	if ctx == nil {
		return errors.New("platform: nil operation release context")
	}
	if err := validateOperationRefShape(expected); err != nil {
		return err
	}
	return releaseOwnedOperation(ctx, expected)
}

func validateOperationIntentShape(intent OperationIntent) error {
	if !operationOwnerPattern.MatchString(intent.Owner) || !operationAttemptPattern.MatchString(intent.AttemptID) || !operationInvocation.MatchString(intent.InvocationID) {
		return fmt.Errorf("%w: malformed operation intent", ErrOperationIdentity)
	}
	if len(intent.SpecDigest) != sha256.Size*2 || strings.ToLower(intent.SpecDigest) != intent.SpecDigest {
		return fmt.Errorf("%w: malformed specification digest", ErrOperationIdentity)
	}
	if _, err := hex.DecodeString(intent.SpecDigest); err != nil {
		return fmt.Errorf("%w: malformed specification digest", ErrOperationIdentity)
	}
	return nil
}

func validateOperationRefShape(ref OperationRef) error {
	if err := validateOperationIntentShape(ref.Intent); err != nil {
		return err
	}
	if ref.Unit != operationUnitName(ref.Intent) || ref.Scope.Path == "" || ref.NativeInvocationID == "" {
		return fmt.Errorf("%w: native unit or scope does not match intent", ErrOperationIdentity)
	}
	if ref.RootPID < 0 || (ref.RootPID == 0) != (ref.RootStartTime == "") {
		return fmt.Errorf("%w: malformed root process identity", ErrOperationIdentity)
	}
	return validateNativeOperationRef(ref)
}

// Stop stops the native service and waits for the manager's terminal record.
func (op *OwnedOperation) Stop(ctx context.Context) (OperationStopReceipt, error) {
	if op == nil {
		return OperationStopReceipt{}, errors.New("platform: nil owned operation")
	}
	receipt, err := StopOwnedOperation(ctx, op.Ref)
	if err != nil {
		return receipt, err
	}
	if err := op.Wait(ctx); err != nil {
		return receipt, err
	}
	if op.launcherReaped {
		receipt.LauncherReap = LauncherReaped
	}
	return receipt, nil
}

// Wait observes manager-recorded native termination and exit evidence. It is
// bounded by ctx and does not represent local launcher reaping.
func (op *OwnedOperation) Wait(ctx context.Context) error {
	if op == nil {
		return errors.New("platform: nil owned operation")
	}
	if ctx == nil {
		return errors.New("platform: nil operation wait context")
	}
	for {
		observation, err := InspectOwnedOperation(ctx, op.Ref)
		if err != nil {
			return err
		}
		if observation.ExitKnown && !observation.Populated {
			return nil
		}
		timer := time.NewTimer(operationWaitPoll)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
