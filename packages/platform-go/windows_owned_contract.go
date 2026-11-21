package platform

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Windows payload and receipt codecs are platform independent so recovery
// identity and refusal fixtures run without SCM, an account, or a Windows host.
const windowsOwnedPayloadMaxJSONBytes = 32768

type windowsOwnedPayload struct {
	Version int
	Intent  OperationIntent
	Spec    OperationSpec
	BootID  string
}

func windowsOwnedServiceName(i OperationIntent) string {
	if i.Kind == ownedServiceKind {
		return "vrooli-owned-" + i.Owner
	}
	return operationUnitName(i)
}

// WindowsOwnedServiceAccount is retired: Windows workloads use the actual
// operator's canonical SID, resolved by ResolveOwnedServiceAccount.
func WindowsOwnedServiceAccount(owner string) (string, error) { return "", ErrUnsupported }
func windowsOwnedAccount(i OperationIntent) string            { return i.NativeAccount }
func windowsOwnedJob(i OperationIntent) string                { return `Global\VrooliOwned-` + i.InvocationID }
func validateWindowsPayload(p windowsOwnedPayload) error {
	if p.Version != 2 || !serviceBootIDPattern.MatchString(p.BootID) {
		return ErrOperationIdentity
	}
	if e := validateWindowsFrozenText(p.Spec.Path, p.Spec.Dir, p.Spec.Args, p.Spec.Env); e != nil {
		return e
	}
	if p.Intent.Kind != ownedServiceKind {
		return fmt.Errorf("platform: Windows hard thread-count bounded operations unavailable: %w", ErrUnsupported)
	}
	if p.Spec.Limits.Enforcement != "native-process-limits" {
		return fmt.Errorf("platform: Windows services require explicit native-process-limits (no thread quota): %w", ErrUnsupported)
	}
	if !validNativeObserverPrincipal("windows", p.Spec.ObserverPrincipal) || (p.Spec.ObserverPrincipal != "" && p.Spec.ObserverPrincipal != p.Spec.NativeAccount) {
		return ErrOperationIdentity
	}
	var err error
	if p.Intent.Kind == ownedServiceKind {
		err = validateServiceIntent(ServiceIntent(p.Intent), OwnedServiceSpec{Path: p.Spec.Path, Dir: p.Spec.Dir, Args: p.Spec.Args, Env: p.Spec.Env, NativeAccount: p.Spec.NativeAccount, ObserverPrincipal: p.Spec.ObserverPrincipal, NativeLauncher: p.Spec.NativeLauncher, Limits: ServiceLimits{MemoryMaxBytes: p.Spec.Limits.MemoryMaxBytes, TasksMax: p.Spec.Limits.TasksMax, CPUWeight: p.Spec.Limits.CPUWeight, Enforcement: p.Spec.Limits.Enforcement}})
	} else {
		err = validateOperationIntent(p.Intent, p.Spec)
	}
	if err != nil {
		return err
	}
	if !validNativeObserverPrincipal("windows", p.Spec.NativeAccount) || p.Spec.NativeAccount == "" || p.Spec.NativeAccount != p.Intent.NativeAccount {
		return ErrOperationIdentity
	}

	seen := map[string]bool{}
	for _, entry := range p.Spec.Env {
		k, _, _ := strings.Cut(entry, "=")
		key := strings.ToUpper(k)
		if key == "VROOLI_NATIVE_OWNED_PAYLOAD" || key == "VROOLI_NATIVE_OWNED_JOB_HANDLE" {
			return ErrOperationIdentity
		}
		if seen[key] {
			return errors.New("platform: Windows environment has duplicate case-insensitive key")
		}
		seen[key] = true
	}
	if p.Spec.Gate != nil {
		return fmt.Errorf("platform: Windows gate ACL qualification required: %w", ErrUnsupported)
	}
	// The durable reader's bound applies to the encoded JSON, including escape
	// expansion and all intent/specification overhead, before any reservation.
	raw, e := json.Marshal(p)
	if e != nil {
		return e
	}
	if len(raw) > windowsOwnedPayloadMaxJSONBytes {
		return ErrOperationIdentity
	}
	return nil
}
func encodeWindowsPayload(p windowsOwnedPayload) (string, error) {
	b, e := json.Marshal(p)
	if e != nil {
		return "", e
	}
	s := base64.RawURLEncoding.EncodeToString(b)
	if len(s) > 24000 {
		return "", errors.New("platform: Windows operator invocation exceeds command-line bound")
	}
	return s, nil
}
func decodeWindowsPayload(raw string) (windowsOwnedPayload, error) {
	var p windowsOwnedPayload
	if len(raw) > 24000 {
		return p, ErrOperationIdentity
	}
	b, e := base64.RawURLEncoding.DecodeString(raw)
	if e != nil {
		return p, ErrOperationIdentity
	}
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	if e = d.Decode(&p); e != nil {
		return p, e
	}
	if e = d.Decode(new(any)); e != io.EOF {
		return p, ErrOperationIdentity
	}
	return p, validateWindowsPayload(p)
}

func windowsRef(p windowsOwnedPayload) OperationRef {
	i := p.Intent
	name := windowsOwnedServiceName(i)
	return OperationRef{Intent: i, Unit: operationUnitName(i), Scope: ScopeRef{Kind: ScopeKindNative, Path: "operator/" + name}, NativeInvocationID: i.InvocationID, Native: &NativeWorkloadIdentity{Version: 2, Platform: "windows", Manager: "operator-job", Domain: "operator", Name: name, InvocationID: i.InvocationID, Account: windowsOwnedAccount(i), ContainmentID: windowsOwnedJob(i), BootID: p.BootID, ObserverPrincipal: p.Spec.ObserverPrincipal}}
}
func windowsSameNative(a, b OperationRef) bool {
	if a.Native == nil || b.Native == nil {
		return false
	}
	x, y := *a.Native, *b.Native
	x.RootIncarnation = ""
	y.RootIncarnation = ""
	return x == y
}

type windowsOwnedRootRecord struct {
	PID                int
	Start, Incarnation string
	ImageID            string
	ImageHandle        uint64
	JobHandle          uint64
	TaskInstance       string
}

type windowsOwnedTerminalRecord struct {
	Version  int
	Intent   OperationIntent
	BootID   string
	Root     windowsOwnedRootRecord
	ExitCode uint32
}

func (t windowsOwnedTerminalRecord) matches(p windowsOwnedPayload, r windowsOwnedRootRecord) bool {
	return t.Version == 1 && t.Intent == p.Intent && t.BootID == p.BootID && t.Root == r && r.PID > 0
}

// Acknowledged by the exact trusted wrapper only after native handle/name/DACL
// and membership validation, before any vendor witness or child launch.
type windowsOwnedCustodyRecord struct {
	Version int
	Intent  OperationIntent
	BootID  string
	Root    windowsOwnedRootRecord
}

func (c windowsOwnedCustodyRecord) matches(p windowsOwnedPayload, r windowsOwnedRootRecord) bool {
	return c.Version == 1 && c.Intent == p.Intent && c.BootID == p.BootID && c.Root == r && r.PID > 0 && r.JobHandle != 0 && r.TaskInstance != ""
}

// decodeWindowsOperatorPayloadJSON is the exact strict durable ledger codec
// used by native lookup and task bootstrap.
func decodeWindowsOperatorPayloadJSON(raw string, i OperationIntent) (windowsOwnedPayload, error) {
	var p windowsOwnedPayload
	var e error
	if len(raw) > windowsOwnedPayloadMaxJSONBytes {
		return p, ErrOperationIdentity
	}
	d := json.NewDecoder(strings.NewReader(raw))
	d.DisallowUnknownFields()
	if e = d.Decode(&p); e != nil {
		return p, e
	}
	if e = d.Decode(new(any)); e != io.EOF {
		return p, ErrOperationIdentity
	}
	if e = validateWindowsPayload(p); e != nil {
		return p, e
	}
	if p.Intent != i {
		return p, ErrOperationIdentity
	}
	return p, nil
}
