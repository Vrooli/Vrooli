package coreactivation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/process"
)

// NativeWorker supplies the native fencing portion of Effects. Its zero value
// uses the platform owner. It does not publish artifacts, launch a command,
// enroll updates or grant authority to an operation.
type NativeWorker struct{ native *nativeWorkerOps }

type nativeWorkerOps struct {
	lookup      func(context.Context, platform.OperationIntent) (platform.OperationRef, error)
	inspect     func(context.Context, platform.OperationRef) (platform.OperationObservation, error)
	stop        func(context.Context, platform.OperationRef) (platform.OperationStopReceipt, error)
	release     func(context.Context, platform.OperationRef) error
	absent      func(context.Context, platform.OperationIntent) (bool, error)
	released    func(context.Context, platform.OperationRef, string) (bool, error)
	members     func(context.Context, platform.ScopeRef) ([]int, error)
	incarnation func(int) (string, error)
	imageDigest func(context.Context, int) (string, error)
	pid         func() int
}

func (n NativeWorker) ops() nativeWorkerOps {
	if n.native != nil {
		return *n.native
	}
	return nativeWorkerOps{lookup: platform.LookupOwnedOperation, inspect: platform.InspectOwnedOperation, stop: platform.StopOwnedOperation, release: platform.ReleaseOwnedOperation, absent: platform.OwnedOperationIntentAbsent, released: platform.OwnedOperationReleasedForIncarnation, members: platform.ScopeProcessesContext, incarnation: process.Incarnation, imageDigest: runningImageDigest, pid: os.Getpid}
}

// ResolveSelf proves that this process is the admitted worker's exact live
// native root, in that native tree and on the admission boot. The retained
// worker executable must call this before RunWorker, never trust argv or an
// environment invocation token alone.
func (n NativeWorker) ResolveSelf(ctx context.Context, r Record) (platform.OperationRef, error) {
	return n.resolveSelf(ctx, r, false)
}

// VerifyClaimedSelf repeats native/image membership proof at a destructive
// effect, after the engine has durably claimed this exact worker.
func (n NativeWorker) VerifyClaimedSelf(ctx context.Context, r Record) error {
	actual, err := n.resolveSelf(ctx, r, true)
	if err != nil {
		return err
	}
	if r.WorkerRef == nil || !reflect.DeepEqual(actual, *r.WorkerRef) {
		return ErrClaimed
	}
	return nil
}

func (n NativeWorker) resolveSelf(ctx context.Context, r Record, claimed bool) (platform.OperationRef, error) {
	if ctx == nil {
		return platform.OperationRef{}, ErrConflict
	}
	if err := ctx.Err(); err != nil {
		return platform.OperationRef{}, err
	}
	if err := r.validate(r.Home); err != nil {
		return platform.OperationRef{}, err
	}
	if !r.DispatchCommitted || !claimed && (r.Phase != Prepared || r.Fence != Ready) || claimed && (r.Fence != Claimed || r.WorkerRef == nil || r.Phase != Publishing && r.Phase != Activating && r.Phase != Verifying) {
		return platform.OperationRef{}, ErrClaimed
	}
	o := n.ops()
	ref, err := o.lookup(ctx, r.WorkerIntent)
	if err != nil {
		return platform.OperationRef{}, fmt.Errorf("core activation native worker identity unavailable")
	}
	if err := ref.ValidateIdentity(); err != nil {
		return platform.OperationRef{}, err
	}
	if ref.Intent != r.WorkerIntent || ref.RootPID != o.pid() {
		return platform.OperationRef{}, ErrConflict
	}
	incarnation, err := o.incarnation(ref.RootPID)
	boot, ticks, ok := strings.Cut(incarnation, ":")
	priorBoot, _, _ := strings.Cut(r.PriorIncarnation, ":")
	if err != nil || !ok || !validIncarnation(incarnation) || boot != priorBoot || ticks != ref.RootStartTime {
		return platform.OperationRef{}, ErrConflict
	}
	observation, err := o.inspect(ctx, ref)
	if err != nil || observation.State != platform.OperationRunning || !observation.RootAlive || !observation.Populated {
		return platform.OperationRef{}, fmt.Errorf("core activation worker is not a verified live native root")
	}
	members, err := o.members(ctx, ref.Scope)
	if err != nil {
		return platform.OperationRef{}, fmt.Errorf("core activation native worker membership unavailable")
	}
	for _, pid := range members {
		if pid == ref.RootPID {
			sha, err := o.imageDigest(ctx, ref.RootPID)
			if err != nil || sha != r.WorkerSHA {
				return platform.OperationRef{}, fmt.Errorf("core activation worker executable is not the admitted frozen image")
			}
			after, err := o.incarnation(ref.RootPID)
			if err != nil || after != incarnation {
				return platform.OperationRef{}, ErrConflict
			}
			return ref, nil
		}
	}
	return platform.OperationRef{}, fmt.Errorf("core activation worker is outside its admitted native tree")
}

// RunningImageDigest reads the actual process image, including after atomic
// replacement of its installed pathname. It does not grant signal authority.
func RunningImageDigest(ctx context.Context, pid int) (string, error) {
	return runningImageDigest(ctx, pid)
}

func runningImageDigest(ctx context.Context, pid int) (string, error) {
	if ctx == nil {
		return "", ErrConflict
	}
	f, err := platform.OpenProcessExecutable(pid)
	if err != nil {
		return "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > imageLimit {
		return "", ErrConflict
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(&contextReader{ctx: ctx, reader: f}, imageLimit+1))
	if err != nil {
		return "", err
	}
	if n <= 0 || n > imageLimit {
		return "", ErrConflict
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (n NativeWorker) AuthorizeWorker(ctx context.Context, r Record, ref platform.OperationRef) error {
	actual, err := n.ResolveSelf(ctx, r)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(actual, ref) {
		return ErrConflict
	}
	return nil
}

// ExcludeWorker requires a durable revoke before native cancellation. Unknown
// manager state is an error. Neither a failed lookup nor an old boot alone is
// enough: both deterministic unit and cgroup absence must be observed.
func (n NativeWorker) ExcludeWorker(ctx context.Context, r Record) error {
	if ctx == nil {
		return ErrConflict
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.validate(r.Home); err != nil {
		return err
	}
	if r.Fence != Revoked {
		return ErrClaimed
	}
	if !r.DispatchCommitted {
		return nil
	}
	o := n.ops()
	ref, err := n.resolveForRecovery(ctx, r, o)
	if err != nil {
		return err
	}
	if ref == nil {
		return nil
	}
	if ref.RootPID == o.pid() {
		return fmt.Errorf("core activation recovery requires an independent native owner")
	}
	receipt, err := o.stop(ctx, *ref)
	if err == nil {
		if !receipt.NativeQuiescent || !reflect.DeepEqual(receipt.Ref, *ref) {
			return fmt.Errorf("core activation native worker quiescence is unproven")
		}
		return nil
	}
	if ref.RootPID > 0 {
		boot, _, _ := strings.Cut(r.PriorIncarnation, ":")
		if released, observeErr := o.released(ctx, *ref, boot+":"+ref.RootStartTime); observeErr == nil && released {
			return nil
		}
	}
	return fmt.Errorf("core activation native worker exclusion is unresolved")
}

func (n NativeWorker) resolveForRecovery(ctx context.Context, r Record, o nativeWorkerOps) (*platform.OperationRef, error) {
	if r.WorkerRef != nil {
		return r.WorkerRef, nil
	}
	ref, err := o.lookup(ctx, r.WorkerIntent)
	if err == nil {
		if ref.Intent != r.WorkerIntent {
			return nil, ErrConflict
		}
		if err := ref.ValidateIdentity(); err != nil {
			return nil, err
		}
		return &ref, nil
	}
	absent, absenceErr := o.absent(ctx, r.WorkerIntent)
	if absenceErr != nil || !absent {
		return nil, fmt.Errorf("core activation native submission is unresolved")
	}
	return nil, nil
}

// ReleaseWorker retires native manager evidence only after a terminal record
// exists and this recovery owner has proved the worker tree quiescent.
func (n NativeWorker) ReleaseWorker(ctx context.Context, r Record) error {
	if !r.Terminal() {
		return ErrConflict
	}
	if err := n.ExcludeWorker(ctx, r); err != nil {
		return err
	}
	if !r.DispatchCommitted {
		return nil
	}
	o := n.ops()
	ref, err := n.resolveForRecovery(ctx, r, o)
	if err != nil || ref == nil {
		return err
	}
	if err := o.release(ctx, *ref); err != nil {
		// A lost release acknowledgement is reconcilable through absence.
		absent, observeErr := o.absent(ctx, r.WorkerIntent)
		if observeErr != nil || !absent {
			return fmt.Errorf("core activation native worker release is unresolved")
		}
	}
	return nil
}
