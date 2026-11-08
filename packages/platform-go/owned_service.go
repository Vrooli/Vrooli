package platform

import (
	"context"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
)

const ownedServiceKind = "service"

// ServiceLimits bound resource use without imposing a job deadline on the
// serving process. Services stop only through their owner or native failure.
type ServiceLimits struct {
	MemoryMaxBytes int64 `json:"memory_max_bytes"`
	TasksMax       int   `json:"tasks_max"`
	CPUWeight      int   `json:"cpu_weight"`
}

// OwnedServiceSpec is an explicit, fixed invocation. It never inherits a
// caller environment, invokes a shell or requests automatic native restart.
// The domain owner must persist logical launch inputs and reconcile this
// intent before retrying an ambiguous dispatch.
// Env is for non-secret settings and credential-file references: native
// manager properties and command arguments are visible to the same user.
type OwnedServiceSpec struct {
	Path, Dir string
	Args, Env []string
	Limits    ServiceLimits
}

// ServiceIntent and ServiceRef retain the same exact native identity facts as
// bounded operations, with distinct admission and native namespaces.
type (
	ServiceIntent OperationIntent
	ServiceRef    OperationRef
)

// ValidateIdentity checks receipt structure without performing native I/O or
// asserting backend support, process ownership or current health.
func (i ServiceIntent) ValidateIdentity() error { return validateServiceIntentShape(i) }

// ValidateSpec verifies the frozen resolved invocation without native effects.
func (i ServiceIntent) ValidateSpec(spec OwnedServiceSpec) error {
	return validateServiceIntent(i, spec)
}

// UnitName identifies the native service to reconcile after dispatch. It is
// metadata only; a name is never evidence that the unit exists or is owned.
func (i ServiceIntent) UnitName() (string, error) {
	if err := i.ValidateIdentity(); err != nil {
		return "", err
	}
	return operationUnitName(OperationIntent(i)), nil
}

func (r ServiceRef) ValidateIdentity() error {
	if err := validateServiceIntentShape(ServiceIntent(r.Intent)); err != nil {
		return err
	}
	if err := validateOperationRefIdentity(OperationRef(r)); err != nil {
		return err
	}
	// The first owned-service backend records a Linux cgroup identity. This
	// validates that record on any inspection host without claiming support
	// for native effects there.
	if r.Scope.Kind != ScopeKindCgroup || r.Scope.PID != 0 || !validCgroupServicePath(r.Scope.Path, r.Unit, ServicesSlice) || !operationInvocation.MatchString(r.NativeInvocationID) || (r.ScopeDevice == 0) != (r.ScopeInode == 0) {
		return fmt.Errorf("%w: malformed owned service native identity", ErrOperationIdentity)
	}
	return nil
}

func validCgroupServicePath(scopePath, unit, slice string) bool {
	return path.IsAbs(scopePath) && path.Clean(scopePath) == scopePath && strings.HasSuffix(scopePath, "/"+slice+"/"+unit)
}

func (s OwnedServiceSpec) processSpec() OperationSpec {
	return OperationSpec{Path: s.Path, Dir: s.Dir, Args: s.Args, Env: s.Env, Limits: OperationLimits{
		MemoryMaxBytes: s.Limits.MemoryMaxBytes, TasksMax: s.Limits.TasksMax, CPUWeight: s.Limits.CPUWeight,
	}}
}

// NewServiceIntent creates an identity to persist durably BEFORE dispatch.
// The receipt includes a digest of the resolved invocation, not its secrets.
func NewServiceIntent(owner, attemptID string, spec OwnedServiceSpec) (ServiceIntent, error) {
	process := spec.processSpec()
	if err := validateOwnedProcessSpec(process); err != nil {
		return ServiceIntent{}, err
	}
	intent, err := newOwnedWorkloadIntent(owner, attemptID, operationSpecDigest(process), ownedServiceKind)
	return ServiceIntent(intent), err
}

func validateServiceIntentShape(intent ServiceIntent) error {
	if intent.Kind != ownedServiceKind {
		return fmt.Errorf("%w: service admission requires a service intent", ErrOperationIdentity)
	}
	return validateOperationIntentShape(OperationIntent(intent))
}

func validateServiceIntent(intent ServiceIntent, spec OwnedServiceSpec) error {
	if err := validateServiceIntentShape(intent); err != nil {
		return err
	}
	process := spec.processSpec()
	if err := validateOwnedProcessSpec(process); err != nil {
		return err
	}
	if intent.SpecDigest != operationSpecDigest(process) {
		return fmt.Errorf("%w: service specification changed", ErrOperationIdentity)
	}
	return nil
}

func validateServiceRef(ref ServiceRef) error {
	if err := ref.ValidateIdentity(); err != nil {
		return err
	}
	return validateNativeOperationRef(OperationRef(ref))
}

func StartOwnedService(ctx context.Context, intent ServiceIntent, spec OwnedServiceSpec) (ServiceRef, error) {
	if ctx == nil {
		return ServiceRef{}, errors.New("platform: nil service start context")
	}
	if err := ctx.Err(); err != nil {
		return ServiceRef{}, err
	}
	if err := validateServiceIntent(intent, spec); err != nil {
		return ServiceRef{}, err
	}
	started, err := startOwnedOperation(ctx, OperationIntent(intent), spec.processSpec())
	if err != nil {
		return ServiceRef{}, err
	}
	return ServiceRef(started.Ref), nil
}

func LookupOwnedService(ctx context.Context, intent ServiceIntent) (ServiceRef, error) {
	if err := validateServiceIntentShape(intent); err != nil {
		return ServiceRef{}, err
	}
	ref, err := LookupOwnedOperation(ctx, OperationIntent(intent))
	return ServiceRef(ref), err
}

// Inspect/Stop/Release share the native identity and recursive quiescence
// owner with bounded work. Unsupported native backends fail closed; no direct
// process fallback silently drops the pre-dispatch ownership contract.
func InspectOwnedService(ctx context.Context, ref ServiceRef) (OperationObservation, error) {
	if err := validateServiceRef(ref); err != nil {
		return OperationObservation{}, err
	}
	return InspectOwnedOperation(ctx, OperationRef(ref))
}

func StopOwnedService(ctx context.Context, ref ServiceRef) (OperationStopReceipt, error) {
	if err := validateServiceRef(ref); err != nil {
		return OperationStopReceipt{}, err
	}
	return StopOwnedOperation(ctx, OperationRef(ref))
}

func ReleaseOwnedService(ctx context.Context, ref ServiceRef) error {
	if err := validateServiceRef(ref); err != nil {
		return err
	}
	return ReleaseOwnedOperation(ctx, OperationRef(ref))
}

// OwnedServiceReleased observes native unit/cgroup absence and the recorded
// root's absence. It does not recover exit outcome or fence a future dispatch;
// the domain owner must retain its durable exec fence and prior stop receipt.
func OwnedServiceReleased(ctx context.Context, ref ServiceRef) (bool, error) {
	return observeServiceRelease(ctx, ref, "")
}

// OwnedServiceReleasedForIncarnation additionally binds the root to the
// boot-id:start-ticks incarnation retained by its domain owner. A different
// boot excludes that root even if its PID and start ticks have been reused.
// Native unit and cgroup absence remain mandatory. This is observation only;
// it neither authorizes redispatch nor recovers an exit outcome.
func OwnedServiceReleasedForIncarnation(ctx context.Context, ref ServiceRef, incarnation string) (bool, error) {
	boot, ticks, ok := strings.Cut(incarnation, ":")
	start, startErr := strconv.ParseUint(ticks, 10, 64)
	if !ok || !serviceBootIDPattern.MatchString(boot) || startErr != nil || start == 0 || ticks != ref.RootStartTime || ref.RootPID <= 0 {
		return false, fmt.Errorf("%w: service root incarnation differs from native reference", ErrOperationIdentity)
	}
	return observeServiceRelease(ctx, ref, boot)
}

var serviceBootIDPattern = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)

func observeServiceRelease(ctx context.Context, ref ServiceRef, boot string) (bool, error) {
	if ctx == nil {
		return false, errors.New("platform: nil service release observation context")
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := ref.ValidateIdentity(); err != nil {
		return false, err
	}
	return ownedServiceReleased(ctx, ref, boot)
}

// OwnedServiceIntentAbsent observes the deterministic native unit and cgroup
// for an intent whose submission may not have returned a process receipt. It
// supplies no future-exec fence; the domain owner must revoke that separately.
func OwnedServiceIntentAbsent(ctx context.Context, intent ServiceIntent) (bool, error) {
	if ctx == nil {
		return false, errors.New("platform: nil service absence context")
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := intent.ValidateIdentity(); err != nil {
		return false, err
	}
	return ownedServiceIntentAbsent(ctx, intent)
}
