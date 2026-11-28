//go:build !windows

package platform

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

type darwinFixtureNative struct {
	account                                                              darwinOwnedAccount
	boot                                                                 string
	process                                                              darwinOwnedProcess
	present, uncertainBootstrap, uncertainBootout, retainRoot, deferRoot bool
	domainMissing, domainCreationFails, domainBootstrapLost              bool
	beforeDomainBootstrap                                                func()
	calls                                                                []string
	managerErr                                                           error
	beforeBootstrap                                                      func()
	queries                                                              int
	transition                                                           func(int)
}

func (f *darwinFixtureNative) Run(_ context.Context, args ...string) ([]byte, error) {
	f.calls = append(f.calls, strings.Join(args, " "))
	switch args[0] {
	case "bootstrap":
		if len(args) == 2 {
			if f.beforeDomainBootstrap != nil {
				f.beforeDomainBootstrap()
			}
			if args[1] != "user/501" {
				return nil, ErrOperationIdentity
			}
			if f.domainCreationFails {
				return nil, errors.New("domain creation refused")
			}
			f.domainMissing = false
			if f.domainBootstrapLost {
				return nil, errors.New("domain bootstrap response lost")
			}
			return nil, nil
		}
		if f.beforeBootstrap != nil {
			f.beforeBootstrap()
		}
		if args[1] != "user/501" {
			return nil, ErrOperationIdentity
		}
		f.present = true
		if !f.deferRoot {
			f.process = darwinOwnedProcess{PID: 42, Seconds: 100, Micros: 123, UniqueID: 999, ExecVersion: 1}
		}
		if f.uncertainBootstrap {
			return nil, errors.New("lost submission")
		}
	case "bootout":
		if !strings.HasPrefix(args[1], "user/501/com.vrooli.owned.") {
			return nil, ErrOperationIdentity
		}
		if f.uncertainBootout {
			return nil, errors.New("lost retirement")
		}
		f.present = false
		if !f.retainRoot {
			f.process = darwinOwnedProcess{}
		}
	default:
		return nil, ErrUnsupported
	}
	return nil, nil
}
func (f *darwinFixtureNative) ValidateLaunch(_ context.Context, i OperationIntent, s OperationSpec) error {
	return validateDarwinMaintainedLaunchShape(i, s)
}
func (f *darwinFixtureNative) Account(_ context.Context, name string) (darwinOwnedAccount, error) {
	if name != f.account.Name {
		return darwinOwnedAccount{}, ErrOperationIdentity
	}
	return f.account, nil
}
func (f *darwinFixtureNative) Boot(context.Context) (string, error) { return f.boot, nil }
func (f *darwinFixtureNative) Snapshot(_ context.Context, domain, label, expected string) (darwinJobSnapshot, bool, error) {
	f.queries++
	if f.transition != nil {
		f.transition(f.queries)
	}
	if f.domainMissing {
		return darwinJobSnapshot{}, false, errors.New("user domain not yet ready")
	}
	if f.managerErr != nil {
		return darwinJobSnapshot{}, false, f.managerErr
	}
	if domain != "user/501" || !strings.HasPrefix(label, "com.vrooli.owned.") {
		return darwinJobSnapshot{}, false, ErrOperationIdentity
	}
	if !f.present {
		return darwinJobSnapshot{}, false, nil
	}
	state := "not running"
	if f.process.PID > 0 {
		state = "running"
	}
	return darwinJobSnapshot{PID: f.process.PID, State: state}, true, nil
}
func (f *darwinFixtureNative) Process(_ context.Context, pid int, uid uint32) (darwinOwnedProcess, bool, error) {
	if uid != 501 {
		return darwinOwnedProcess{}, false, ErrOperationIdentity
	}
	if pid != f.process.PID || pid <= 0 {
		return darwinOwnedProcess{}, false, nil
	}
	return f.process, true, nil
}
func newDarwinFixture(t *testing.T) (darwinOwnedBackend, *darwinFixtureNative, OperationIntent, OperationSpec) {
	t.Helper()
	root := t.TempDir()
	verify := func(path string, dir bool) error {
		st, e := os.Lstat(path)
		if e != nil {
			return e
		}
		if st.Mode()&os.ModeSymlink != 0 || st.IsDir() != dir || st.Mode().Perm()&0077 != 0 {
			return ErrOperationIdentity
		}
		return nil
	}
	store := darwinOwnedFileStore{root: root, plistDir: root, verify: verify, prepare: func() error { return nil }}
	f := &darwinFixtureNative{account: darwinOwnedAccount{Name: "uid:501", UID: 501, GID: 20}, boot: "11111111-2222-3333-4444-555555555555"}
	s := OwnedServiceSpec{Path: "/sealed/vrooli", Dir: "/sealed", NativeAccount: "uid:501", ObserverPrincipal: "uid:501", Args: []string{"__vrooli-openrgb-native-capsule", "/Library/Application Support/Vrooli/resources/openrgb/launch/fixture-1/capsule.json"}, Env: []string{}, Limits: ServiceLimits{MemoryMaxBytes: 1024, TasksMax: 128, CPUWeight: 100, Enforcement: "native-advisory"}}
	i, e := NewServiceIntent("resource-initial", "fixture-1", s)
	if e != nil {
		t.Fatal(e)
	}
	return darwinOwnedBackend{store: store, native: f}, f, OperationIntent(i), s.processSpec()
}
func startDarwinFixture(t *testing.T, b darwinOwnedBackend, i OperationIntent, s OperationSpec) OperationRef {
	t.Helper()
	op, e := b.start(context.Background(), i, s)
	if e != nil {
		t.Fatal(e)
	}
	return op.Ref
}
func TestDarwinOperatorJobsUncertainSubmissionAndRetirement(t *testing.T) {
	b, f, i, s := newDarwinFixture(t)
	f.uncertainBootstrap = true
	f.beforeBootstrap = func() {
		r, e := b.store.Read(i)
		if e != nil || r.Version != 2 || r.Spec.NativeAccount != "uid:501" {
			t.Fatalf("no durable predispatch intent: %+v %v", r, e)
		}
	}
	if _, e := b.start(context.Background(), i, s); e == nil {
		t.Fatal("lost submission became success")
	}
	ref, e := b.lookup(context.Background(), i)
	if e != nil {
		t.Fatal(e)
	}
	if ref.Native.Domain != "user/501" || ref.Native.RootUniqueID != "999" || !ref.Native.ProcessOnly {
		t.Fatalf("invalid operator identity %+v", ref)
	}
	if _, e = b.start(context.Background(), i, s); !errors.Is(e, ErrOperationExists) {
		t.Fatal("redispatch", e)
	}
	r, e := b.stop(context.Background(), ref)
	if e != nil || !r.NativeQuiescent {
		t.Fatalf("retirement %+v %v", r, e)
	}
	if e = b.release(context.Background(), ref); e != nil {
		t.Fatal(e)
	}
	if ok, e := b.released(context.Background(), ref, ref.Native.BootID); e != nil || !ok {
		t.Fatalf("absence %v %v", ok, e)
	}
	for _, c := range f.calls {
		if strings.Contains(c, "kill") || strings.Contains(c, "system") {
			t.Fatal("global account effect", c)
		}
	}
}
func TestDarwinOperatorJobsPrivateStateNoUIDQuota(t *testing.T) {
	b, _, i, s := newDarwinFixture(t)
	startDarwinFixture(t, b, i, s)
	for _, p := range []string{b.store.Path(i), b.store.(darwinOwnedFileStore).recordPath(i)} {
		st, e := os.Stat(p)
		if e != nil || st.Mode().Perm() != 0600 {
			t.Fatalf("private state %s %v %v", p, st, e)
		}
	}
	for _, bad := range []string{"UserName", "NumberOfProcesses", "_vrooli_"} {
		if strings.Contains(darwinOwnedPlist(i, s), bad) {
			t.Fatal("unsafe operator config", bad)
		}
	}
}
func TestDarwinOperatorJobsRejectPIDReuseAndTransientExec(t *testing.T) {
	for _, kind := range []string{"reuse", "exec", "manager", "account"} {
		t.Run(kind, func(t *testing.T) {
			b, f, i, s := newDarwinFixture(t)
			ref := startDarwinFixture(t, b, i, s)
			switch kind {
			case "reuse":
				f.process.UniqueID++
			case "exec":
				f.queries = 0
				f.transition = func(n int) {
					if n%2 == 0 {
						f.process.ExecVersion++
					}
				}
			case "manager":
				f.managerErr = errors.New("RPC down")
			case "account":
				f.account.UID++
			}
			n := len(f.calls)
			if _, e := b.inspect(context.Background(), ref); e == nil {
				t.Fatal("uncertain identity accepted")
			}
			if _, e := b.stop(context.Background(), ref); e == nil {
				t.Fatal("unsafe stop accepted")
			}
			if len(f.calls) != n {
				t.Fatal("identity failure dispatched effect")
			}
		})
	}
}
func TestDarwinOperatorJobsStableBirthAcrossExec(t *testing.T) {
	b, f, i, s := newDarwinFixture(t)
	ref := startDarwinFixture(t, b, i, s)
	f.process.ExecVersion++
	current, e := b.lookup(context.Background(), i)
	if e != nil || !current.Equal(ref) {
		t.Fatalf("samePID handoff changed receipt %v", e)
	}
}
func TestDarwinOperatorJobsUnassociatedSurvivorIsNotQuiescent(t *testing.T) {
	b, f, i, s := newDarwinFixture(t)
	ref := startDarwinFixture(t, b, i, s)
	f.retainRoot = true
	if r, e := b.stop(context.Background(), ref); !errors.Is(e, ErrOperationNotQuiescent) || r.NativeQuiescent {
		t.Fatalf("survivor %+v %v", r, e)
	}
	if e := b.release(context.Background(), ref); !errors.Is(e, ErrOperationNotQuiescent) {
		t.Fatal(e)
	}
}
func TestDarwinOperatorJobsOldBootCannotClaimCurrentProcess(t *testing.T) {
	b, f, i, s := newDarwinFixture(t)
	ref := startDarwinFixture(t, b, i, s)
	f.boot = "22222222-2222-3333-4444-555555555555"
	if _, e := b.lookup(context.Background(), i); e == nil {
		t.Fatal("old record claimed newboot")
	}
	f.present = false
	f.process = darwinOwnedProcess{PID: 42, Seconds: 200, UniqueID: 4321, ExecVersion: 1}
	if e := b.release(context.Background(), ref); e != nil {
		t.Fatal("oldboot root extinct but blocked", e)
	}
}
func TestDarwinOperatorJobsUncertainBootoutRetainsClaim(t *testing.T) {
	b, f, i, s := newDarwinFixture(t)
	ref := startDarwinFixture(t, b, i, s)
	f.uncertainBootout = true
	if r, e := b.stop(context.Background(), ref); e == nil || r.NativeQuiescent {
		t.Fatal("lost bootout became quiescence")
	}
	if _, e := b.store.Read(i); e != nil {
		t.Fatal(e)
	}
	f.uncertainBootout = false
	if r, e := b.stop(context.Background(), ref); e != nil || !r.NativeQuiescent {
		t.Fatalf("recovery %+v %v", r, e)
	}
}
func TestDarwinOperatorJobsNaturalExitRetainsUnknownOutcomeAndCanRetire(t *testing.T) {
	b, f, i, s := newDarwinFixture(t)
	ref := startDarwinFixture(t, b, i, s)
	f.process = darwinOwnedProcess{}
	o, e := b.inspect(context.Background(), ref)
	if e != nil || o.Populated || o.ExitKnown || o.State != OperationOutcomeUnknown {
		t.Fatalf("natural exit manufactured outcome %+v %v", o, e)
	}
	r, e := b.stop(context.Background(), ref)
	if e != nil || !r.NativeQuiescent {
		t.Fatalf("natural exit retirement %+v %v", r, e)
	}
	if e = b.release(context.Background(), ref); e != nil {
		t.Fatal(e)
	}
}
func TestDarwinOperatorJobsRevocationPreventsDelayedCapsuleExec(t *testing.T) {
	b, f, i, s := newDarwinFixture(t)
	ref := startDarwinFixture(t, b, i, s)
	f.uncertainBootout = true
	if _, e := b.stop(context.Background(), ref); e == nil {
		t.Fatal("expected uncertain bootout")
	}
	if _, e := b.frozenEnvironment(context.Background(), ref); !errors.Is(e, ErrOperationIdentity) {
		t.Fatalf("revoked still-live capsule got execution environment: %v", e)
	}
}
func TestDarwinOperatorJobsRefuseArbitraryEntrypointsBeforePersistenceOrEffects(t *testing.T) {
	for _, kind := range []string{"owner", "verb", "argv", "environment", "cwd"} {
		t.Run(kind, func(t *testing.T) {
			b, f, _, s := newDarwinFixture(t)
			owner := "resource-initial"
			switch kind {
			case "owner":
				owner = "arbitrary-app"
			case "verb":
				s.Args[0] = "shell"
			case "argv":
				s.Args[1] = "/tmp/capsule.json"
			case "environment":
				s.Env = []string{"INJECT=1"}
			case "cwd":
				s.Dir = "/tmp"
			}
			intent, e := NewServiceIntent(owner, "fixture-1", OwnedServiceSpec{Path: s.Path, Dir: s.Dir, NativeAccount: s.NativeAccount, ObserverPrincipal: s.ObserverPrincipal, Args: s.Args, Env: s.Env, Limits: ServiceLimits{MemoryMaxBytes: s.Limits.MemoryMaxBytes, TasksMax: s.Limits.TasksMax, CPUWeight: s.Limits.CPUWeight, Enforcement: s.Limits.Enforcement}})
			if e != nil {
				t.Fatal(e)
			}
			if _, e = b.start(context.Background(), OperationIntent(intent), s); e == nil {
				t.Fatal("arbitrary native entrypoint accepted")
			}
			if len(f.calls) != 0 {
				t.Fatal("refused launch caused native effect")
			}
			if _, e = b.store.Read(OperationIntent(intent)); !errors.Is(e, ErrOperationNotFound) {
				t.Fatal("refused launch persisted", e)
			}
		})
	}
}
func TestDarwinOperatorJobsWaitForRealRootBirthAfterBootstrap(t *testing.T) {
	b, f, i, s := newDarwinFixture(t)
	f.deferRoot = true
	f.transition = func(n int) {
		if n == 4 {
			f.process = darwinOwnedProcess{PID: 42, Seconds: 100, Micros: 123, UniqueID: 999, ExecVersion: 1}
		}
	}
	op, e := b.start(context.Background(), i, s)
	if e != nil || op.Ref.RootPID != 42 || op.Ref.Native.RootUniqueID != "999" {
		t.Fatalf("bootstrap delay returned incomplete birth %+v %v", op, e)
	}
}

func TestDarwinOperatorJobsStartupDeadlineRetainsRecoverableIntent(t *testing.T) {
	b, f, i, s := newDarwinFixture(t)
	f.deferRoot = true
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if op, e := b.start(ctx, i, s); !errors.Is(e, context.DeadlineExceeded) || op != nil {
		t.Fatalf("unpublished birth became successful admission %+v %v", op, e)
	}
	if _, e := b.store.Read(i); e != nil {
		t.Fatal("startup timeout lost durable intent", e)
	}
	f.process = darwinOwnedProcess{PID: 42, Seconds: 100, Micros: 123, UniqueID: 999, ExecVersion: 1}
	if ref, e := b.lookup(context.Background(), i); e != nil || ref.RootPID != 42 {
		t.Fatalf("late root could not recover %+v %v", ref, e)
	}
	dispatches := 0
	for _, call := range f.calls {
		if strings.HasPrefix(call, "bootstrap ") {
			dispatches++
		}
	}
	if dispatches != 1 {
		t.Fatal("startup uncertainty redispatched", f.calls)
	}
}
func TestDarwinOperatorJobsCreatesMissingUserDomainWithoutAssumingAbsence(t *testing.T) {
	for _, kind := range []string{"created", "domain-response-lost", "unavailable"} {
		t.Run(kind, func(t *testing.T) {
			b, f, i, s := newDarwinFixture(t)
			f.domainMissing = true
			f.domainBootstrapLost = kind == "domain-response-lost"
			f.domainCreationFails = kind == "unavailable"
			f.beforeDomainBootstrap = func() {
				if record, e := b.store.Read(i); e != nil || record.Intent != i {
					t.Fatalf("domain mutation preceded durable intent %+v %v", record, e)
				}
			}
			op, e := b.start(context.Background(), i, s)
			if kind == "unavailable" {
				if e == nil || op != nil {
					t.Fatal("query failure became absence/start success")
				}
				if len(f.calls) != 1 || f.calls[0] != "bootstrap user/501" {
					t.Fatal("unready domain dispatched service", f.calls)
				}
				if _, e = b.store.Read(i); e != nil {
					t.Fatal("unknown readiness lost intent", e)
				}
			} else {
				if e != nil || op.Ref.RootPID != 42 {
					t.Fatalf("firstboot domain recovery %+v %v", op, e)
				}
				if len(f.calls) != 2 || f.calls[0] != "bootstrap user/501" || !strings.HasPrefix(f.calls[1], "bootstrap user/501 ") {
					t.Fatal("domain/service dispatch invalid", f.calls)
				}
			}
		})
	}
}
func TestDarwinOperatorJobsDomainRecoveryNeverAdoptsPreexistingRole(t *testing.T) {
	b, f, i, s := newDarwinFixture(t)
	f.domainMissing = true
	f.beforeDomainBootstrap = func() {
		f.present = true
		f.process = darwinOwnedProcess{PID: 77, Seconds: 100, UniqueID: 777, ExecVersion: 1}
	}
	if op, e := b.start(context.Background(), i, s); !errors.Is(e, ErrOperationExists) || op != nil {
		t.Fatalf("foreign role adopted %+v %v", op, e)
	}
	if !f.present || f.process.PID != 77 {
		t.Fatal("foreign native role mutated")
	}
	if len(f.calls) != 1 || f.calls[0] != "bootstrap user/501" {
		t.Fatal("role effect issued after preexisting observation", f.calls)
	}
	if _, e := b.lookup(context.Background(), i); !errors.Is(e, ErrOperationNotFound) {
		t.Fatal("new intent blessed foreign role", e)
	}
}
func TestDarwinOperatorJobsOnlyDispatchPreparedIntentCanClaimManagerRoot(t *testing.T) {
	b, f, i, s := newDarwinFixture(t)
	f.domainMissing = true
	f.domainCreationFails = true
	if _, e := b.start(context.Background(), i, s); e == nil {
		t.Fatal("expected unknown userdomain")
	}
	f.domainMissing = false
	f.present = true
	f.process = darwinOwnedProcess{PID: 77, Seconds: 100, UniqueID: 777, ExecVersion: 1}
	if _, e := b.lookup(context.Background(), i); !errors.Is(e, ErrOperationIdentity) {
		t.Fatal("never-dispatched local intent adopted manager root", e)
	}
}
