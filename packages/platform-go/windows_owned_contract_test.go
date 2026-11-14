package platform

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func windowsOwnedFixture(t *testing.T) windowsOwnedPayload {
	t.Helper()
	dir := t.TempDir()
	s := OperationSpec{NativeAccount: "sid:S-1-5-21-100-200-300-1001", Path: filepath.Join(dir, "fixture.exe"), Dir: dir, Env: []string{"APPDATA=private"}, Limits: OperationLimits{MemoryMaxBytes: 64 << 20, TasksMax: 32, CPUWeight: 100, Enforcement: "native-process-limits"}}
	i, e := windowsOwnedFixtureIntent(s)
	if e != nil {
		t.Fatal(e)
	}
	return windowsOwnedPayload{Version: 2, Intent: i, Spec: s, BootID: "12345678-1234-1234-1234-123456789abc"}
}
func TestWindowsOwnedPayloadRetainsFrozenIntent(t *testing.T) {
	p := windowsOwnedFixture(t)
	raw, e := encodeWindowsPayload(p)
	if e != nil {
		t.Fatal(e)
	}
	got, e := decodeWindowsPayload(raw)
	if e != nil {
		t.Fatal(e)
	}
	if got.Intent != p.Intent || got.Spec.Env[0] != p.Spec.Env[0] {
		t.Fatal("frozen intent or private environment changed")
	}
	got.Spec.Args = []string{"changed"}
	if e = validateWindowsPayload(got); !errors.Is(e, ErrOperationIdentity) {
		t.Fatalf("changed invocation accepted: %v", e)
	}
}
func TestWindowsOwnedBroadAccountsRefused(t *testing.T) {
	for _, account := range []string{"LocalSystem", `NT AUTHORITY\SYSTEM`, `NT AUTHORITY\LocalService`, "Administrator"} {
		t.Run(account, func(t *testing.T) {
			p := windowsOwnedFixture(t)
			p.Spec.NativeAccount = account
			i, e := windowsOwnedFixtureIntent(p.Spec)
			if e != nil {
				t.Fatal(e)
			}
			p.Intent = i
			if e = validateWindowsPayload(p); !errors.Is(e, ErrOperationIdentity) {
				t.Fatalf("broad account accepted: %v", e)
			}
		})
	}
}
func TestWindowsOwnedCaseInsensitiveEnvironmentRefused(t *testing.T) {
	p := windowsOwnedFixture(t)
	p.Spec.Env = []string{"APPDATA=private", "appdata=ambient"}
	i, e := windowsOwnedFixtureIntent(p.Spec)
	if e != nil {
		t.Fatal(e)
	}
	p.Intent = i
	if e = validateWindowsPayload(p); e == nil {
		t.Fatal("case-insensitive duplicate environment accepted")
	}
}
func TestWindowsOwnedRefBindsExactContainment(t *testing.T) {
	p := windowsOwnedFixture(t)
	r := windowsRef(p)
	if e := r.ValidateIdentity(); e != nil {
		t.Fatal(e)
	}
	if r.Native.Account != windowsOwnedAccount(p.Intent) || !strings.HasPrefix(r.Native.ContainmentID, `Global\VrooliOwned-`) {
		t.Fatal("actual operator account or invocation Job missing")
	}
	next := r
	copyNative := *r.Native
	next.Native = &copyNative
	next.Native.InvocationID = strings.Repeat("a", 32)
	if e := next.ValidateIdentity(); !errors.Is(e, ErrOperationIdentity) {
		t.Fatal("another incarnation accepted")
	}
	copyNative = *r.Native
	copyNative.Account = "LocalSystem"
	if windowsSameNative(r, next) {
		t.Fatal("account substitution matched")
	}
}
func TestWindowsOwnedPayloadBounds(t *testing.T) {
	if _, e := decodeWindowsPayload(strings.Repeat("x", 24001)); !errors.Is(e, ErrOperationIdentity) {
		t.Fatal("unbounded payload accepted")
	}
	p := windowsOwnedFixture(t)
	p.Spec.Args = []string{strings.Repeat("x", 24001)}
	if _, e := encodeWindowsPayload(p); e == nil {
		t.Fatal("SCM command-line overflow accepted")
	}
}

func TestWindowsOwnedExplicitAccountStableAcrossWorkflowOwners(t *testing.T) {
	account := "sid:S-1-5-21-100-200-300-1001"
	var names []string
	for _, owner := range []string{"resource-initial", "resource-update"} {
		p := windowsOwnedFixture(t)
		p.Spec.NativeAccount = account
		s := OwnedServiceSpec{Path: p.Spec.Path, Dir: p.Spec.Dir, Env: p.Spec.Env, NativeAccount: account, Limits: ServiceLimits{MemoryMaxBytes: 64 << 20, TasksMax: 32, CPUWeight: 100, Enforcement: "native-process-limits"}}
		i, e := NewServiceIntent(owner, "attempt", s)
		if e != nil {
			t.Fatal(e)
		}
		p.Intent = OperationIntent(i)
		p.Spec = s.processSpec()
		if e = validateWindowsPayload(p); e != nil {
			t.Fatal(e)
		}
		r := windowsRef(p)
		if e = r.ValidateIdentity(); e != nil {
			t.Fatal(e)
		}
		names = append(names, r.Native.Name)
		if r.Native.Account != account {
			t.Fatal("account changed across admission")
		}
	}
	if names[0] == names[1] {
		t.Fatal("separate workflow roles shared native slot")
	}
	if account == "" {
		t.Fatal("missing actual operator")
	}
}

func TestWindowsOwnedStableSlotRejectsNewIncarnation(t *testing.T) {
	account := "sid:S-1-5-21-100-200-300-1001"
	p := windowsOwnedFixture(t)
	s := OwnedServiceSpec{Path: p.Spec.Path, Dir: p.Spec.Dir, Env: p.Spec.Env, NativeAccount: account, Limits: ServiceLimits{MemoryMaxBytes: 64 << 20, TasksMax: 32, CPUWeight: 100, Enforcement: "native-process-limits"}}
	first, e := NewServiceIntent("resource-initial", "first", s)
	if e != nil {
		t.Fatal(e)
	}
	second, e := NewServiceIntent("resource-update", "second", s)
	if e != nil {
		t.Fatal(e)
	}
	a := windowsRef(windowsOwnedPayload{Intent: OperationIntent(first), BootID: p.BootID})
	b := windowsRef(windowsOwnedPayload{Intent: OperationIntent(second), BootID: p.BootID})
	if a.Native.Name == b.Native.Name {
		t.Fatal("different workflow roles shared native slot")
	}
	if windowsSameNative(a, b) {
		t.Fatal("retired receipt matched replacement native invocation")
	}
	if a.Native.ContainmentID == b.Native.ContainmentID {
		t.Fatal("new invocation reused old containment")
	}
}
func TestWindowsOwnedRefusesAdvisoryLimits(t *testing.T) {
	p := windowsOwnedFixture(t)
	p.Spec.Limits.Enforcement = "native-advisory"
	if e := validateWindowsPayload(p); !errors.Is(e, ErrUnsupported) {
		t.Fatalf("Windows silently accepted advisory limits: %v", e)
	}
}

func windowsOwnedFixtureIntent(s OperationSpec) (OperationIntent, error) {
	i, e := NewServiceIntent("fixture", "attempt", OwnedServiceSpec{Path: s.Path, Dir: s.Dir, Args: s.Args, Env: s.Env, NativeAccount: s.NativeAccount, ObserverPrincipal: s.ObserverPrincipal, NativeLauncher: s.NativeLauncher, Limits: ServiceLimits{MemoryMaxBytes: s.Limits.MemoryMaxBytes, TasksMax: s.Limits.TasksMax, CPUWeight: s.Limits.CPUWeight, Enforcement: s.Limits.Enforcement}})
	return OperationIntent(i), e
}
func TestWindowsOwnedRefusesLegacyThreadQuotaClaim(t *testing.T) {
	p := windowsOwnedFixture(t)
	p.Spec.Limits.Enforcement = ""
	if e := validateWindowsPayload(p); !errors.Is(e, ErrUnsupported) {
		t.Fatalf("legacy thread-task hard guarantee accepted: %v", e)
	}
}
func TestWindowsOwnedBoundedThreadQuotaUnsupported(t *testing.T) {
	p := windowsOwnedFixture(t)
	p.Spec.Limits.Enforcement = ""
	p.Spec.Limits.WallTime = time.Minute
	i, e := NewOperationIntent("fixture", "bounded", p.Spec)
	if e != nil {
		t.Fatal(e)
	}
	p.Intent = i
	if e = validateWindowsPayload(p); !errors.Is(e, ErrUnsupported) {
		t.Fatalf("bounded hard thread guarantee accepted: %v", e)
	}
}
func TestWindowsOwnedTerminalEvidenceBindsExactRoot(t *testing.T) {
	p := windowsOwnedFixture(t)
	r := windowsOwnedRootRecord{PID: 42, Start: "116444736000000010", Incarnation: "windows:42:1000", ImageID: "windows-file:fixture", ImageHandle: 123}
	term := windowsOwnedTerminalRecord{Version: 1, Intent: p.Intent, BootID: p.BootID, Root: r, ExitCode: 7}
	if !term.matches(p, r) {
		t.Fatal("exact root terminal record rejected")
	}
	other := r
	other.PID++
	if term.matches(p, other) {
		t.Fatal("terminal outcome attributed to another root")
	}
	q := p
	q.Intent.InvocationID = strings.Repeat("a", 32)
	if term.matches(q, r) {
		t.Fatal("terminal outcome attributed to later invocation")
	}
	q = p
	q.BootID = "87654321-1234-1234-1234-123456789abc"
	if term.matches(q, r) {
		t.Fatal("terminal outcome attributed across reboot")
	}
}

func TestWindowsOwnedObserverIsFrozenInNativeReceipt(t *testing.T) {
	p := windowsOwnedFixture(t)
	p.Spec.ObserverPrincipal = "sid:S-1-5-21-100-200-300-1001"
	intent, e := windowsOwnedFixtureIntent(p.Spec)
	if e != nil {
		t.Fatal(e)
	}
	p.Intent = intent
	if e = validateWindowsPayload(p); e != nil {
		t.Fatal(e)
	}
	r := windowsRef(p)
	if e = r.ValidateIdentity(); e != nil {
		t.Fatal(e)
	}
	if r.Native.ObserverPrincipal != p.Spec.ObserverPrincipal {
		t.Fatal("observer omitted from durable native identity")
	}
	p.Spec.ObserverPrincipal = "sid:S-1-5-21-100-200-300-1002"
	if e = validateWindowsPayload(p); !errors.Is(e, ErrOperationIdentity) {
		t.Fatal("observer substitution accepted", e)
	}
}

func TestWindowsProofBearingServiceAdmissionAndLedgerCodec(t *testing.T) {
	p := windowsTaskFixture(t)
	spec := OwnedServiceSpec{
		Path: p.Spec.Path, Dir: p.Spec.Dir, Args: p.Spec.Args, Env: p.Spec.Env,
		NativeAccount: p.Spec.NativeAccount, ObserverPrincipal: p.Spec.ObserverPrincipal,
		NativeLauncher: &NativeLauncherProof{SchemaVersion: NativeLauncherProofVersion, Profile: NativeLauncherProfileOpenRGB, SHA256: strings.Repeat("a", 64)},
		Limits:         ServiceLimits{MemoryMaxBytes: p.Spec.Limits.MemoryMaxBytes, TasksMax: p.Spec.Limits.TasksMax, CPUWeight: p.Spec.Limits.CPUWeight, Enforcement: p.Spec.Limits.Enforcement},
	}
	intent, e := NewServiceIntent("resource-initial", "attempt", spec)
	if e != nil {
		t.Fatal(e)
	}
	p.Intent = OperationIntent(intent)
	p.Spec = spec.processSpec()
	if e = validateWindowsPayload(p); e != nil {
		t.Fatal("proof-bearing real Windows admission refused", e)
	}
	raw, e := json.Marshal(p)
	if e != nil {
		t.Fatal(e)
	}
	// This is the actual strict production registry JSON decoder used by lookup
	// and task bootstrap, rather than the retired base64 environment transport.
	recovered, e := decodeWindowsOperatorPayloadJSON(string(raw), p.Intent)
	if e != nil {
		t.Fatal("own proof-bearing ledger unreadable", e)
	}
	if !reflect.DeepEqual(recovered, p) || recovered.Spec.NativeLauncher == nil {
		t.Fatal("frozen proof lost during ledger recovery")
	}
	mutated := recovered
	proof := *mutated.Spec.NativeLauncher
	mutated.Spec.NativeLauncher = &proof
	mutated.Spec.NativeLauncher.SHA256 = strings.Repeat("b", 64)
	changed, e := json.Marshal(mutated)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = decodeWindowsOperatorPayloadJSON(string(changed), p.Intent); !errors.Is(e, ErrOperationIdentity) {
		t.Fatal("changed proof accepted without redigesting frozen intent", e)
	}
	mutated.Spec.NativeLauncher = nil
	changed, e = json.Marshal(mutated)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = decodeWindowsOperatorPayloadJSON(string(changed), p.Intent); !errors.Is(e, ErrOperationIdentity) {
		t.Fatal("dropped proof silently fell back to installed host", e)
	}
	if _, e = decodeWindowsOperatorPayloadJSON(string(raw)+"{}", p.Intent); e == nil {
		t.Fatal("trailing ledger bytes accepted")
	}
	foreign := p.Intent
	foreign.InvocationID = strings.Repeat("b", 32)
	if _, e = decodeWindowsOperatorPayloadJSON(string(raw), foreign); !errors.Is(e, ErrOperationIdentity) {
		t.Fatal("foreign invocation ledger accepted", e)
	}
}
