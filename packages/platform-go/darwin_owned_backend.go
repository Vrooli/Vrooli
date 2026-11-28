package platform

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const darwinOwnedRecordLimit = 256 << 10

type darwinOwnedRecord struct {
	Version          int             `json:"version"`
	Intent           OperationIntent `json:"intent"`
	Spec             OperationSpec   `json:"spec"`
	UID              uint32          `json:"uid"`
	GID              uint32          `json:"gid"`
	BootID           string          `json:"boot_id"`
	PlistHash        string          `json:"plist_hash"`
	Revoked          bool            `json:"revoked"`
	DispatchPrepared bool            `json:"dispatch_prepared"`
}
type darwinOwnedAccount struct {
	Name     string
	UID, GID uint32
}
type darwinOwnedProcess struct {
	PID             int
	Seconds, Micros int64
	UniqueID        uint64
	ExecVersion     uint32
}

func (p darwinOwnedProcess) start() string { return strconv.FormatInt(p.Seconds*1000000+p.Micros, 10) }
func (p darwinOwnedProcess) incarnation() string {
	return fmt.Sprintf("darwin:%d:%d:%d", p.PID, p.Seconds, p.Micros)
}

type darwinOwnedNative interface {
	Run(context.Context, ...string) ([]byte, error)
	ValidateLaunch(context.Context, OperationIntent, OperationSpec) error
	Account(context.Context, string) (darwinOwnedAccount, error)
	Snapshot(context.Context, string, string, string) (darwinJobSnapshot, bool, error)
	Process(context.Context, int, uint32) (darwinOwnedProcess, bool, error)
	Boot(context.Context) (string, error)
}
type darwinOwnedStore interface {
	Lock(context.Context) (func(), error)
	Path(OperationIntent) string
	Activated(OperationIntent) (bool, error)
	Read(OperationIntent) (darwinOwnedRecord, error)
	Admit(darwinOwnedRecord) error
	PrepareDispatch(darwinOwnedRecord) error
	Revoke(darwinOwnedRecord) error
	Remove(darwinOwnedRecord) error
}
type darwinOwnedBackend struct {
	store  darwinOwnedStore
	native darwinOwnedNative
}

func validDarwinOwnedAccount(s string) bool {
	return s != "" && validNativeObserverPrincipal("darwin", s)
}
func darwinOwnedDomain(i OperationIntent) string {
	return "user/" + strings.TrimPrefix(i.NativeAccount, "uid:")
}
func decodeDarwinOwnedRecord(data []byte) (darwinOwnedRecord, error) {
	var r darwinOwnedRecord
	if len(data) > darwinOwnedRecordLimit {
		return r, ErrOperationIdentity
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if e := d.Decode(&r); e != nil {
		return r, e
	}
	if d.Decode(new(any)) != io.EOF {
		return r, ErrOperationIdentity
	}
	if r.Version != 2 || r.UID == 0 || r.GID == 0 || !serviceBootIDPattern.MatchString(r.BootID) || r.Intent.Kind != ownedServiceKind {
		return r, ErrOperationIdentity
	}
	if e := validateOperationIntentShape(r.Intent); e != nil {
		return r, e
	}
	if e := validateOwnedProcessSpec(r.Spec); e != nil {
		return r, e
	}
	if e := validateDarwinMaintainedLaunchShape(r.Intent, r.Spec); e != nil {
		return r, e
	}
	if r.Spec.Gate != nil || r.Spec.ObserverPrincipal != "" && r.Spec.ObserverPrincipal != r.Spec.NativeAccount || r.Spec.Limits.Enforcement != "native-advisory" || r.Intent.NativeAccount != r.Spec.NativeAccount || !validDarwinOwnedAccount(r.Spec.NativeAccount) || r.Spec.NativeAccount != "uid:"+strconv.FormatUint(uint64(r.UID), 10) || operationSpecDigest(r.Spec) != r.Intent.SpecDigest {
		return r, ErrOperationIdentity
	}
	sum := sha256.Sum256([]byte(darwinOwnedPlist(r.Intent, r.Spec)))
	if hex.EncodeToString(sum[:]) != r.PlistHash {
		return r, ErrOperationIdentity
	}
	return r, nil
}
func (b darwinOwnedBackend) snapshot(ctx context.Context, i OperationIntent) (darwinJobSnapshot, bool, error) {
	expected := ""
	r, e := b.store.Read(i)
	if e == nil {
		expected = darwinOwnedPlist(r.Intent, r.Spec)
	} else if !errors.Is(e, ErrOperationNotFound) {
		return darwinJobSnapshot{}, false, e
	}
	return b.native.Snapshot(ctx, darwinOwnedDomain(i), darwinOwnedLabel(i), expected)
}
func (b darwinOwnedBackend) read(ctx context.Context, i OperationIntent) (darwinOwnedRecord, error) {
	r, e := b.store.Read(i)
	if e != nil {
		return r, e
	}
	a, e := b.native.Account(ctx, r.Spec.NativeAccount)
	if e != nil {
		return r, e
	}
	if a.Name != r.Spec.NativeAccount || a.UID != r.UID || a.GID != r.GID {
		return r, ErrOperationIdentity
	}
	return r, nil
}
func (b darwinOwnedBackend) ref(r darwinOwnedRecord, p darwinJobSnapshot, process darwinOwnedProcess, boot string) (OperationRef, error) {
	name := darwinOwnedLabel(r.Intent)
	domain := darwinOwnedDomain(r.Intent)
	ref := OperationRef{Intent: r.Intent, Unit: operationUnitName(r.Intent), Scope: ScopeRef{Kind: ScopeKindNative, Path: domain + "/" + name}, NativeInvocationID: r.Intent.InvocationID, Native: &NativeWorkloadIdentity{Version: 2, Platform: "darwin", Manager: "launchd", Domain: domain, Name: name, InvocationID: r.Intent.InvocationID, Account: r.Spec.NativeAccount, ContainmentID: domain + "/" + name, BootID: boot, ObserverPrincipal: r.Spec.ObserverPrincipal, ProcessOnly: true}}
	if p.PID > 0 {
		if process.PID != p.PID || process.UniqueID == 0 || process.ExecVersion == 0 || process.Seconds <= 0 || process.Micros < 0 || process.Micros >= 1000000 {
			return OperationRef{}, ErrOperationIdentity
		}
		ref.RootPID = process.PID
		ref.RootStartTime = process.start()
		ref.Native.RootIncarnation = process.incarnation()
		ref.Native.RootUniqueID = strconv.FormatUint(process.UniqueID, 10)
	}
	return ref, ref.ValidateIdentity()
}
func (b darwinOwnedBackend) start(ctx context.Context, i OperationIntent, s OperationSpec) (*OwnedOperation, error) {
	if s.Limits.Enforcement != "native-advisory" || i.Kind != ownedServiceKind || s.Gate != nil {
		return nil, ErrUnsupported
	}
	if !validDarwinOwnedAccount(s.NativeAccount) || s.ObserverPrincipal != "" && s.ObserverPrincipal != s.NativeAccount || operationSpecDigest(s) != i.SpecDigest {
		return nil, ErrOperationIdentity
	}
	if e := b.native.ValidateLaunch(ctx, i, s); e != nil {
		return nil, e
	}
	unlock, e := b.store.Lock(ctx)
	if e != nil {
		return nil, e
	}
	defer unlock()
	if _, e = b.store.Read(i); e == nil {
		return nil, ErrOperationExists
	} else if !errors.Is(e, ErrOperationNotFound) {
		return nil, e
	}
	a, e := b.native.Account(ctx, s.NativeAccount)
	if e != nil {
		return nil, e
	}
	if a.Name != s.NativeAccount || a.UID == 0 || a.GID == 0 {
		return nil, ErrOperationIdentity
	}
	// This first native query does not consult a yet-unpublished local record.
	// UNKNOWN is preserved; it may mean a boot-time user domain is not ready.
	_, present, managerErr := b.native.Snapshot(ctx, darwinOwnedDomain(i), darwinOwnedLabel(i), "")
	if managerErr == nil && present {
		return nil, ErrOperationExists
	}
	boot, e := b.native.Boot(ctx)
	if e != nil {
		return nil, e
	}
	sum := sha256.Sum256([]byte(darwinOwnedPlist(i, s)))
	r := darwinOwnedRecord{Version: 2, Intent: i, Spec: s, UID: a.UID, GID: a.GID, BootID: boot, PlistHash: hex.EncodeToString(sum[:])}
	// Predispatch intent and frozen spec precede any native-manager admission.
	if e = b.store.Admit(r); e != nil {
		return nil, e
	}
	if managerErr != nil {
		// Explicit domain-only bootstrap is the documented OS readiness operation.
		// It occurs only after durable intent, and never itself submits our private
		// plist. Even an "already exists"/lost response is not absence evidence:
		// a new typed query must independently establish the exact label's absence.
		out, domainErr := b.native.Run(ctx, "bootstrap", darwinOwnedDomain(i))
		_, present, queryErr := b.snapshot(ctx, i)
		if queryErr != nil {
			return nil, fmt.Errorf("platform: user-domain readiness remains unknown; retain intent: %w: %s", errors.Join(managerErr, domainErr, queryErr), out)
		}
		if present {
			// Domain creation never dispatched our private plist. Do not adopt an
			// unrelated pre-existing label merely because a local intent now exists.
			if e := b.store.Revoke(r); e != nil {
				return nil, errors.Join(ErrOperationExists, e)
			}
			if e := b.store.Remove(r); e != nil {
				return nil, errors.Join(ErrOperationExists, e)
			}
			return nil, ErrOperationExists
		}
	}
	if e = b.store.PrepareDispatch(r); e != nil {
		return nil, e
	}
	if out, e := b.native.Run(ctx, "bootstrap", darwinOwnedDomain(i), b.store.Path(i)); e != nil {
		return nil, fmt.Errorf("platform: launchd submission uncertain; reconcile persisted intent: %w: %s", e, out)
	}
	settleCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	// A successful bootstrap is not yet a kernel-birth receipt. Wait for the
	// manager's exact root association; a startup delay or immediate exit never
	// becomes successful admission with an empty root identity.
	for {
		ref, e := b.lookup(settleCtx, i)
		if e != nil {
			return nil, e
		}
		if ref.RootPID > 0 {
			return &OwnedOperation{Ref: ref, launcherReaped: true}, nil
		}
		timer := time.NewTimer(operationWaitPoll)
		select {
		case <-settleCtx.Done():
			timer.Stop()
			return nil, settleCtx.Err()
		case <-timer.C:
		}
	}

}
func (b darwinOwnedBackend) sample(ctx context.Context, r darwinOwnedRecord) (darwinJobSnapshot, bool, OperationRef, error) {
	boot, e := b.native.Boot(ctx)
	if e != nil {
		return darwinJobSnapshot{}, false, OperationRef{}, e
	}
	p, present, e := b.snapshot(ctx, r.Intent)
	if e != nil {
		return p, false, OperationRef{}, e
	}
	var before darwinOwnedProcess
	if p.PID > 0 {
		var alive bool
		before, alive, e = b.native.Process(ctx, p.PID, r.UID)
		if e != nil {
			return p, false, OperationRef{}, e
		}
		if !alive {
			return p, false, OperationRef{}, ErrOperationIdentity
		}
	}
	q, presentAfter, e := b.snapshot(ctx, r.Intent)
	if e != nil {
		return p, false, OperationRef{}, e
	}
	if p != q || present != presentAfter {
		return p, false, OperationRef{}, ErrOperationIdentity
	}
	if p.PID > 0 {
		after, alive, e := b.native.Process(ctx, p.PID, r.UID)
		if e != nil {
			return p, false, OperationRef{}, e
		}
		if !alive || before != after {
			return p, false, OperationRef{}, ErrOperationIdentity
		}
	}
	afterBoot, e := b.native.Boot(ctx)
	if e != nil {
		return p, false, OperationRef{}, e
	}
	if present && !r.DispatchPrepared {
		return p, false, OperationRef{}, ErrOperationIdentity
	}
	if boot != afterBoot || present && boot != r.BootID {
		return p, false, OperationRef{}, ErrOperationIdentity
	}
	ref, e := b.ref(r, p, before, boot)
	return p, present, ref, e
}
func (b darwinOwnedBackend) lookup(ctx context.Context, i OperationIntent) (OperationRef, error) {
	r, e := b.read(ctx, i)
	if e != nil {
		return OperationRef{}, e
	}
	_, _, ref, e := b.sample(ctx, r)
	return ref, e
}
func validateDarwinOwnedRef(ref OperationRef) error {
	if e := ref.ValidateIdentity(); e != nil {
		return e
	}
	if ref.Intent.Owner != "resource-initial" && ref.Intent.Owner != "resource-update" {
		return ErrUnsupported
	}
	if ref.Native == nil || ref.Native.Platform != "darwin" {
		return ErrOperationIdentity
	}
	return nil
}
func (b darwinOwnedBackend) claimAlive(ctx context.Context, r darwinOwnedRecord, ref OperationRef) (bool, error) {
	boot, e := b.native.Boot(ctx)
	if e != nil {
		return false, e
	}
	if boot != ref.Native.BootID {
		return false, nil
	}
	if ref.RootPID == 0 {
		return false, nil
	}
	p, alive, e := b.native.Process(ctx, ref.RootPID, r.UID)
	if e != nil {
		return false, e
	}
	if !alive {
		return false, nil
	}
	return p.start() == ref.RootStartTime && p.incarnation() == ref.Native.RootIncarnation && strconv.FormatUint(p.UniqueID, 10) == ref.Native.RootUniqueID, nil
}

// Manager absence cannot identify an unobserved submitted root. In particular,
// a rootless lookup is not evidence that bootstrap never launched its wrapper.
// Preserve that same-boot uncertainty until an exact root is recovered or the
// origin boot is extinct. A never-dispatched intent has no such native effect.
func (b darwinOwnedBackend) rootExtinct(ctx context.Context, r darwinOwnedRecord, ref OperationRef) (bool, error) {
	boot, err := b.native.Boot(ctx)
	if err != nil {
		return false, err
	}
	if boot != r.BootID {
		return true, nil
	}
	if ref.RootPID == 0 {
		if !r.DispatchPrepared {
			return true, nil
		}
		return false, ErrOperationOutcomeLost
	}
	if ref.Native.BootID != r.BootID {
		return false, ErrOperationIdentity
	}
	alive, err := b.claimAlive(ctx, r, ref)
	return !alive, err
}
func (b darwinOwnedBackend) inspect(ctx context.Context, expected OperationRef) (OperationObservation, error) {
	o, _, err := b.inspectAssociation(ctx, expected)
	return o, err
}
func (b darwinOwnedBackend) inspectAssociation(ctx context.Context, expected OperationRef) (OperationObservation, bool, error) {
	if e := validateDarwinOwnedRef(expected); e != nil {
		return OperationObservation{}, false, e
	}
	r, e := b.read(ctx, expected.Intent)
	if e != nil {
		return OperationObservation{}, false, e
	}
	_, present, current, e := b.sample(ctx, r)
	if e != nil {
		return OperationObservation{}, false, e
	}
	if present {
		if current.RootPID > 0 {
			if !current.Equal(expected) {
				return OperationObservation{}, false, ErrOperationIdentity
			}
		} else {
			want, got := *expected.Native, *current.Native
			want.RootIncarnation, got.RootIncarnation = "", ""
			want.RootUniqueID, got.RootUniqueID = "", ""
			if want != got {
				return OperationObservation{}, false, ErrOperationIdentity
			}
		}
	}
	alive, e := b.claimAlive(ctx, r, expected)
	if e != nil {
		return OperationObservation{}, false, e
	}
	o := OperationObservation{Ref: expected, State: OperationOutcomeUnknown, Populated: present && current.RootPID > 0 || alive, RootAlive: alive, ExitKnown: false}
	if o.Populated {
		o.State = OperationRunning
	}
	return o, present, nil
}
func (b darwinOwnedBackend) stop(ctx context.Context, ref OperationRef) (OperationStopReceipt, error) {
	receipt := OperationStopReceipt{Ref: ref, LauncherReap: LauncherReapUnknown}
	unlock, e := b.store.Lock(ctx)
	if e != nil {
		return receipt, e
	}
	defer unlock()
	if _, e = b.inspect(ctx, ref); e != nil {
		return receipt, e
	}
	r, e := b.read(ctx, ref.Intent)
	if e != nil {
		return receipt, e
	}
	if e = b.store.Revoke(r); e != nil {
		return receipt, e
	}
	// Revocation is durable I/O, not an atomic extension of the initial root
	// observation. Rebind the current manager/kernel association before bootout.
	_, present, e := b.inspectAssociation(ctx, ref)
	if e != nil {
		return receipt, e
	}
	// Only launchd's exact role label can terminate the process. Never signal
	// sampled PIDs, a process group, or other processes belonging to the operator.
	if present {
		if out, e := b.native.Run(ctx, "bootout", ref.Scope.Path); e != nil {
			return receipt, fmt.Errorf("platform: bootout uncertain: %w: %s", e, out)
		}
	}
	for {
		_, present, e := b.snapshot(ctx, r.Intent)
		if e != nil {
			return receipt, e
		}
		if !present {
			extinct, e := b.rootExtinct(ctx, r, ref)
			if e != nil {
				return receipt, e
			}
			if !extinct {
				return receipt, ErrOperationNotQuiescent
			}
			receipt.NativeQuiescent = true
			return receipt, nil
		}
		timer := time.NewTimer(operationWaitPoll)
		select {
		case <-ctx.Done():
			timer.Stop()
			return receipt, ctx.Err()
		case <-timer.C:
		}
	}
}
func (b darwinOwnedBackend) release(ctx context.Context, ref OperationRef) error {
	unlock, e := b.store.Lock(ctx)
	if e != nil {
		return e
	}
	defer unlock()
	o, e := b.inspect(ctx, ref)
	if e != nil {
		return e
	}
	if o.Populated {
		return ErrOperationNotQuiescent
	}
	r, e := b.read(ctx, ref.Intent)
	if e != nil {
		return e
	}
	if e = b.store.Revoke(r); e != nil {
		return e
	}
	if _, present, e := b.inspectAssociation(ctx, ref); e != nil {
		return e
	} else if present {
		if out, e := b.native.Run(ctx, "bootout", ref.Scope.Path); e != nil {
			return fmt.Errorf("platform: release bootout uncertain: %w: %s", e, out)
		}
	}
	if _, present, e := b.snapshot(ctx, ref.Intent); e != nil {
		return e
	} else if present {
		return ErrOperationNotQuiescent
	}
	extinct, e := b.rootExtinct(ctx, r, ref)
	if e != nil {
		return e
	}
	if !extinct {
		return ErrOperationNotQuiescent
	}
	return b.store.Remove(r)
}
func (b darwinOwnedBackend) released(ctx context.Context, ref OperationRef, boot string) (bool, error) {
	if e := validateDarwinOwnedRef(ref); e != nil {
		return false, e
	}
	if boot != "" && !strings.EqualFold(boot, ref.Native.BootID) {
		return false, ErrOperationIdentity
	}
	if active, e := b.store.Activated(ref.Intent); e != nil || active {
		return false, e
	}
	if _, present, e := b.snapshot(ctx, ref.Intent); e != nil || present {
		return false, e
	}
	r, e := b.read(ctx, ref.Intent)
	if e == nil {
		// Quiescence is not metadata retirement. The resource owner must still
		// call release to remove the exact revoked native intent, even when
		// its known root has exited. Keep unknown root extinction explicit.
		_, e := b.rootExtinct(ctx, r, ref)
		return false, e
	}
	if !errors.Is(e, ErrOperationNotFound) {
		return false, e
	}
	if ref.RootPID == 0 {
		// Removal follows extinction proof under the native admission lock.
		// Without a remaining intent or manager slot there is no pending submit.
		return true, nil
	}
	a, e := b.native.Account(ctx, ref.Native.Account)
	if e != nil {
		return false, e
	}
	alive, e := b.claimAlive(ctx, darwinOwnedRecord{UID: a.UID}, ref)
	return !alive, e
}
func (b darwinOwnedBackend) absent(ctx context.Context, i OperationIntent) (bool, error) {
	if !validDarwinOwnedAccount(i.NativeAccount) {
		return false, ErrOperationIdentity
	}
	if active, e := b.store.Activated(i); e != nil || active {
		return false, e
	}
	if _, present, e := b.snapshot(ctx, i); e != nil || present {
		return false, e
	}
	// With an unrevoked record, the manager may have lost its root association.
	// Only a durable exact claim can retire that ambiguity through released().
	r, e := b.read(ctx, i)
	if errors.Is(e, ErrOperationNotFound) {
		return true, nil
	}
	if e != nil {
		return false, e
	}
	boot, e := b.native.Boot(ctx)
	if e != nil {
		return false, e
	}
	return boot != r.BootID, nil
}

// The file store uses only explicit roots. Native construction supplies a
// root-owner/mode verifier; tests supply isolated roots and the same I/O path.
type darwinOwnedFileStore struct {
	root, plistDir string
	verify         func(string, bool) error
	prepare        func() error
	prepareRead    func() error
}

func (s darwinOwnedFileStore) Path(i OperationIntent) string {
	return filepath.Join(s.plistDir, darwinOwnedLabel(i)+".plist")
}
func (s darwinOwnedFileStore) recordPath(i OperationIntent) string {
	return filepath.Join(s.root, darwinOwnedLabel(i)+".json")
}
func (s darwinOwnedFileStore) Lock(ctx context.Context) (func(), error) {
	if e := s.prepare(); e != nil {
		return nil, e
	}
	return AcquireFileLockContext(ctx, filepath.Join(s.root, "admission.lock"))
}
func (s darwinOwnedFileStore) Read(i OperationIntent) (darwinOwnedRecord, error) {
	var r darwinOwnedRecord
	if e := s.readPrepare(); e != nil {
		return r, e
	}
	path := s.recordPath(i)
	if e := s.verify(path, false); e != nil {
		if os.IsNotExist(e) {
			return r, ErrOperationNotFound
		}
		return r, e
	}
	f, e := os.Open(path)
	if e != nil {
		return r, e
	}
	data, e := io.ReadAll(io.LimitReader(f, darwinOwnedRecordLimit+1))
	closeErr := f.Close()
	if e != nil {
		return r, e
	}
	if closeErr != nil {
		return r, closeErr
	}
	r, e = decodeDarwinOwnedRecord(data)
	if e != nil {
		return r, e
	}
	if r.Intent != i {
		return r, ErrOperationIdentity
	}
	path = s.Path(i)
	e = s.verify(path, false)
	if os.IsNotExist(e) {
		return r, nil
	}
	if e != nil {
		return r, e
	}
	f, e = os.Open(path)
	if e != nil {
		return r, e
	}
	data, e = io.ReadAll(io.LimitReader(f, darwinOwnedRecordLimit+1))
	closeErr = f.Close()
	if e != nil {
		return r, e
	}
	if closeErr != nil {
		return r, closeErr
	}
	sum := sha256.Sum256(data)
	if len(data) > darwinOwnedRecordLimit || hex.EncodeToString(sum[:]) != r.PlistHash {
		return r, ErrOperationIdentity
	}
	return r, nil
}
func (s darwinOwnedFileStore) Admit(r darwinOwnedRecord) error {
	data, e := json.Marshal(r)
	if e != nil {
		return e
	}
	if _, e = decodeDarwinOwnedRecord(data); e != nil {
		return e
	}
	if e = darwinOwnedWriteExclusive(s.recordPath(r.Intent), data, 0600); e != nil {
		return e
	}
	return darwinOwnedWriteExclusive(s.Path(r.Intent), []byte(darwinOwnedPlist(r.Intent, r.Spec)), 0600)
}
func (s darwinOwnedFileStore) PrepareDispatch(r darwinOwnedRecord) error {
	current, e := s.Read(r.Intent)
	if e != nil {
		return e
	}
	if current.Revoked || current.DispatchPrepared || r.DispatchPrepared || r.Revoked || current.BootID != r.BootID || current.PlistHash != r.PlistHash {
		return ErrOperationIdentity
	}
	current.DispatchPrepared = true
	data, e := json.Marshal(current)
	if e != nil {
		return e
	}
	return darwinOwnedReplace(s.recordPath(r.Intent), data)
}
func (s darwinOwnedFileStore) Revoke(r darwinOwnedRecord) error {
	// The persisted revoked flag must never coexist with boot activation.
	// A crash before the final record update leaves an unrevoked receipt with
	// no plist; the same stop owner can safely resume that partial retirement.
	if e := os.Remove(s.Path(r.Intent)); e != nil && !os.IsNotExist(e) {
		return e
	}
	if e := darwinOwnedSyncDir(s.plistDir); e != nil {
		return e
	}
	r.Revoked = true
	data, e := json.Marshal(r)
	if e != nil {
		return e
	}
	return darwinOwnedReplace(s.recordPath(r.Intent), data)
}
func (s darwinOwnedFileStore) Remove(r darwinOwnedRecord) error {
	if _, e := os.Lstat(s.Path(r.Intent)); !os.IsNotExist(e) {
		if e != nil {
			return e
		}
		return ErrOperationNotQuiescent
	}
	if e := os.Remove(s.recordPath(r.Intent)); e != nil {
		return e
	}
	return darwinOwnedSyncDir(s.root)
}
func darwinOwnedSyncDir(path string) error {
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	return f.Sync()
}
func darwinOwnedWriteExclusive(path string, data []byte, mode os.FileMode) error {
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if e != nil {
		return e
	}
	if _, e = f.Write(data); e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	return darwinOwnedSyncDir(filepath.Dir(path))
}
func darwinOwnedReplace(path string, data []byte) error {
	f, e := os.CreateTemp(filepath.Dir(path), ".owned-record-*")
	if e != nil {
		return e
	}
	temp := f.Name()
	defer os.Remove(temp)
	if e = f.Chmod(0600); e == nil {
		_, e = f.Write(data)
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	if e = os.Rename(temp, path); e != nil {
		return e
	}
	return darwinOwnedSyncDir(filepath.Dir(path))
}

func (s darwinOwnedFileStore) Activated(i OperationIntent) (bool, error) {
	if e := s.readPrepare(); e != nil {
		return false, e
	}
	if e := s.verify(s.Path(i), false); e != nil {
		if os.IsNotExist(e) {
			return false, nil
		}
		return false, e
	}
	return true, nil
}

func (s darwinOwnedFileStore) readPrepare() error {
	if s.prepareRead != nil {
		return s.prepareRead()
	}
	return s.prepare()
}

func (b darwinOwnedBackend) frozenEnvironment(ctx context.Context, ref OperationRef) ([]string, error) {
	if ref.Intent.Kind != ownedServiceKind || ref.RootPID <= 0 {
		return nil, ErrOperationIdentity
	}
	observation, err := b.inspect(ctx, ref)
	if err != nil {
		return nil, err
	}
	if !observation.RootAlive || !observation.Populated {
		return nil, ErrOperationIdentity
	}
	record, err := b.read(ctx, ref.Intent)
	if err != nil {
		return nil, err
	}
	if !record.DispatchPrepared || record.Revoked || ref.Native == nil || !strings.EqualFold(record.BootID, ref.Native.BootID) {
		return nil, fmt.Errorf("%w: stale-boot native capsule execution is forbidden", ErrOperationIdentity)
	}
	return append([]string{}, record.Spec.Env...), nil
}
