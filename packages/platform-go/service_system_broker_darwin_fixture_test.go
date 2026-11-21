//go:build !windows

package platform

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type darwinBrokerFixture struct {
	root                                     string
	roots                                    nativeSystemBrokerRoots
	active                                   bool
	activeSHA                                string
	calls                                    []string
	bootstrapLost, bootoutLost, wrongProcess bool
	verdict                                  VerdictState
	closureErr, preflightErr                 error
}

func (f *darwinBrokerFixture) run(_ context.Context, args ...string) ([]byte, error) {
	f.calls = append(f.calls, strings.Join(args, " "))
	switch args[0] {
	case "bootstrap":
		if args[1] != "system" || args[2] != f.roots.Unit {
			return nil, ErrOperationIdentity
		}
		data, e := os.ReadFile(f.roots.Unit)
		if e != nil {
			return nil, e
		}
		path, e := darwinBrokerDescriptorFromPlist(data)
		if e != nil {
			return nil, e
		}
		raw, e := os.ReadFile(path)
		if e != nil {
			return nil, e
		}
		d, e := decodeNativeSystemServiceBrokerDescriptor(raw)
		if e != nil {
			return nil, e
		}
		f.activeSHA = d.Options.ExecutableSHA256
		f.active = true
		if f.bootstrapLost {
			return nil, errors.New("lost bootstrap response")
		}
		return nil, nil
	case "bootout":
		if args[1] != "system/com.vrooli.runtime-supervisor" {
			return nil, ErrOperationIdentity
		}
		if f.bootoutLost {
			return nil, errors.New("lost bootout response")
		}
		f.active = false
		return nil, nil
	case "enable", "kickstart":
		if args[1] != "system/com.vrooli.runtime-supervisor" {
			return nil, ErrOperationIdentity
		}
		return nil, nil
	}
	return nil, errors.New("unexpected launchctl operation")
}
func newDarwinBrokerFixture(t *testing.T) (darwinSystemBrokerEngine, *darwinBrokerFixture, ServiceInstallOptions) {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	f := &darwinBrokerFixture{root: root, verdict: VerdictAccepted, roots: nativeSystemBrokerRoots{Install: filepath.Join(root, "machine", "install"), State: filepath.Join(root, "machine", "state"), Unit: filepath.Join(root, "machine", "LaunchDaemons", "com.vrooli.runtime-supervisor.plist")}}
	image := []byte("native-approved-broker")
	source := filepath.Join(root, "source-vrooli")
	if err := os.WriteFile(source, image, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(image)
	options := ServiceInstallOptions{Executable: source, SystemBroker: &NativeSystemServiceBrokerOptions{ExecutableSHA256: hex.EncodeToString(sum[:]), OperatorUID: 501, OperatorGID: 20, OperatorPrincipal: "unix:501", OperatorUsername: "operator", OperatorHome: filepath.Join(root, "operator-home")}}
	engine := darwinSystemBrokerEngine{roots: f.roots, privileged: func() error { return nil }, operator: func(o NativeSystemServiceBrokerOptions) error {
		if o.OperatorUID != 501 {
			return ErrOperationIdentity
		}
		return nil
	}, check: func(path string, dir bool) error {
		st, e := os.Lstat(path)
		if e != nil {
			return e
		}
		if !strings.HasPrefix(path, root) {
			return nil
		}
		if st.Mode()&os.ModeSymlink != 0 || st.IsDir() != dir || st.Mode().Perm()&0022 != 0 {
			return ErrOperationIdentity
		}
		return nil
	}, preflight: func() error { return f.preflightErr }, closure: func(*os.File) error { return f.closureErr }, run: f.run, query: func(_ context.Context, expected string) (darwinJobSnapshot, bool, error) {
		f.calls = append(f.calls, "query system/com.vrooli.runtime-supervisor")
		if !f.active {
			return darwinJobSnapshot{}, false, nil
		}
		return darwinJobSnapshot{PID: 4242, State: "running"}, true, nil
	}, runtimeIdentity: func(NativeSystemServiceBrokerOptions) error { return nil }, validate: func(RenderedArtifact) Verdict { return Verdict{State: f.verdict, Validator: "fixture-plutil"} }, process: func(_ context.Context, p nativeSystemBrokerPlan, pid int) error {
		if pid != 4242 || f.activeSHA != p.SHA256 || f.wrongProcess {
			return ErrOperationIdentity
		}
		return nil
	}}
	return engine, f, options
}
func TestDarwinSystemBrokerInstallsSealedMachineDaemon(t *testing.T) {
	e, f, o := newDarwinBrokerFixture(t)
	r, err := e.install(context.Background(), o)
	if err != nil || !r.Active || r.Scope != "system" {
		t.Fatalf("%+v %v", r, err)
	}
	p, err := e.plan(o)
	if err != nil {
		t.Fatal(err)
	}
	for path, mode := range map[string]os.FileMode{p.ImagePath: 0555, p.DescriptorPath: 0444, p.UnitPath: 0644} {
		st, err := os.Stat(path)
		if err != nil || st.Mode().Perm() != mode {
			t.Fatalf("artifact %s mode: %v %v", path, st, err)
		}
	}
	data, err := os.ReadFile(p.UnitPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, fact := range []string{"<string>operator</string>", NativeSystemSupervisorVerb, p.DescriptorPath, p.StateRoot, "/usr/bin:/bin:/usr/sbin:/sbin"} {
		if !bytes.Contains(data, []byte(fact)) {
			t.Fatalf("missing %q", fact)
		}
	}
	if bytes.Contains(data, []byte(o.Executable)) || bytes.Contains(data, []byte("VROOLI_SOURCE_ROOT")) || bytes.Contains(data, []byte("gui/")) {
		t.Fatal("root daemon retained mutable source/user service inputs")
	}
	if f.calls[len(f.calls)-1] != "query system/com.vrooli.runtime-supervisor" {
		t.Fatalf("missing final native identity observation: %v", f.calls)
	}
}
func TestDarwinSystemBrokerRefusesBadInputsBeforeNativeEffects(t *testing.T) {
	for _, kind := range []string{"checksum", "script-closure", "unsupported-observer", "operator-home", "source-root", "missing-explicit-broker"} {
		t.Run(kind, func(t *testing.T) {
			e, f, o := newDarwinBrokerFixture(t)
			switch kind {
			case "checksum":
				o.SystemBroker.ExecutableSHA256 = strings.Repeat("a", 64)
			case "script-closure":
				f.closureErr = errors.New("not Mach-O")
			case "unsupported-observer":
				f.preflightErr = ErrUnsupported
			case "operator-home":
				o.HomeDir = "/different-home"
			case "source-root":
				o.SourceRoot = "/operator/repo"
			case "missing-explicit-broker":
				o.SystemBroker = nil
			}
			if _, err := e.install(context.Background(), o); err == nil {
				t.Fatal("unsafe privileged input accepted")
			}
			if len(f.calls) > 0 {
				t.Fatalf("unsafe input reached native manager: %v", f.calls)
			}
			if _, err := os.Stat(f.roots.Install); !os.IsNotExist(err) {
				t.Fatal("unsafe input staged machine image")
			}
		})
	}
}
func TestDarwinSystemBrokerRequiresNativeValidatorAcceptance(t *testing.T) {
	for _, verdict := range []VerdictState{VerdictUnavailable, VerdictRejected} {
		t.Run(string(verdict), func(t *testing.T) {
			e, f, o := newDarwinBrokerFixture(t)
			f.verdict = verdict
			r, err := e.install(context.Background(), o)
			if err == nil || r.Active {
				t.Fatal("unqualified plist became active")
			}
			for _, call := range f.calls {
				if strings.HasPrefix(call, "bootstrap") {
					t.Fatal("unqualified plist was bootstrapped")
				}
			}
		})
	}
}
func TestDarwinSystemBrokerRejectsUnownedExistingUnit(t *testing.T) {
	e, f, o := newDarwinBrokerFixture(t)
	if err := os.MkdirAll(filepath.Dir(f.roots.Unit), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.roots.Unit, []byte("foreign root service"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := e.install(context.Background(), o); err == nil {
		t.Fatal("foreign unit replaced")
	}
	if len(f.calls) > 0 {
		t.Fatal("foreign unit caused native effects")
	}
}
func TestDarwinSystemBrokerBootstrapUncertaintyRetainsRecoverableState(t *testing.T) {
	e, f, o := newDarwinBrokerFixture(t)
	f.bootstrapLost = true
	r, err := e.install(context.Background(), o)
	if err == nil || r.Active {
		t.Fatal("uncertain bootstrap became success")
	}
	if _, exists, err := e.installed(); err != nil || !exists {
		t.Fatalf("lost sealed recovery artifacts: %v", err)
	}
	f.bootstrapLost = false
	if active, err := e.start(context.Background(), o); err != nil || !active {
		t.Fatalf("bootstrap recovery %v %v", active, err)
	}
}
func TestDarwinSystemBrokerInterruptedBootoutRecoversFromJournal(t *testing.T) {
	e, f, o := newDarwinBrokerFixture(t)
	if _, err := e.install(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	f.bootoutLost = true
	if _, err := e.uninstall(context.Background(), o); err == nil {
		t.Fatal("uncertain bootout became success")
	}
	if _, err := os.Stat(f.roots.Unit); !os.IsNotExist(err) {
		t.Fatal("retirement retained boot activation")
	}
	if !f.active {
		t.Fatal("fixture failed to retain uncertain runtime")
	}
	f.bootoutLost = false
	if _, err := e.uninstall(context.Background(), o); err != nil {
		t.Fatalf("retirement journal recovery: %v", err)
	}
	if f.active {
		t.Fatal("retired runtime still active")
	}
}
func TestDarwinSystemBrokerUpgradeUsesOnlyVerifiedPriorDaemon(t *testing.T) {
	e, f, o := newDarwinBrokerFixture(t)
	if _, err := e.install(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	before := len(f.calls)
	next := []byte("next-approved-native-broker")
	if err := os.WriteFile(o.Executable, next, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(next)
	o.SystemBroker.ExecutableSHA256 = hex.EncodeToString(sum[:])
	r, err := e.install(context.Background(), o)
	if err != nil || !r.Active {
		t.Fatalf("upgrade %+v %v", r, err)
	}
	events := strings.Join(f.calls[before:], "\n")
	if !strings.Contains(events, "bootout system/") || !strings.Contains(events, "bootstrap system ") {
		t.Fatalf("upgrade native transition missing: %s", events)
	}
}
func TestDarwinSystemBrokerMismatchedProcessCannotBeStopped(t *testing.T) {
	e, f, o := newDarwinBrokerFixture(t)
	if _, err := e.install(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	f.wrongProcess = true
	f.calls = nil
	if _, err := e.uninstall(context.Background(), o); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("wrong process accepted: %v", err)
	}
	for _, call := range f.calls {
		if strings.HasPrefix(call, "bootout") {
			t.Fatal("identity mismatch stopped an unrelated root process")
		}
	}
}
func TestDarwinSystemBrokerReadonlyDescriptorCannotDrift(t *testing.T) {
	e, _, o := newDarwinBrokerFixture(t)
	if _, err := e.install(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	p, _ := e.plan(o)
	if _, err := e.descriptor(p.DescriptorPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p.DescriptorPath, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := e.descriptor(p.DescriptorPath); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("mutable root descriptor accepted: %v", err)
	}
}
func TestDarwinSystemBrokerRejectsSymlinkArtifactAncestor(t *testing.T) {
	e, f, o := newDarwinBrokerFixture(t)
	if err := os.Mkdir(filepath.Join(f.root, "machine"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), f.roots.Install); err != nil {
		t.Fatal(err)
	}
	if _, err := e.install(context.Background(), o); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("symlink ancestor accepted: %v", err)
	}
	for _, call := range f.calls {
		if strings.HasPrefix(call, "bootstrap") {
			t.Fatal("symlink ancestor reached native bootstrap")
		}
	}
}
func darwinBrokerMachFixture(lib string) []byte {
	var cmd []byte
	if lib != "" {
		size := (24 + len(lib) + 1 + 7) &^ 7
		cmd = make([]byte, size)
		binary.LittleEndian.PutUint32(cmd[0:4], 0xc)
		binary.LittleEndian.PutUint32(cmd[4:8], uint32(size))
		binary.LittleEndian.PutUint32(cmd[8:12], 24)
		copy(cmd[24:], lib)
	}
	raw := make([]byte, 32+len(cmd))
	for index, value := range []uint32{0xfeedfacf, 0x100000c, 0, 2, 0, 0, 0, 0} {
		binary.LittleEndian.PutUint32(raw[index*4:], value)
	}
	if len(cmd) > 0 {
		binary.LittleEndian.PutUint32(raw[16:], 1)
		binary.LittleEndian.PutUint32(raw[20:], uint32(len(cmd)))
		copy(raw[32:], cmd)
	}
	return raw
}
func TestDarwinSystemBrokerMachOClosureRefusesUserDylibs(t *testing.T) {
	for _, lib := range []string{"/usr/lib/libSystem.B.dylib", "/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation"} {
		if err := validateDarwinBrokerMachO(bytes.NewReader(darwinBrokerMachFixture(lib)), "arm64"); err != nil {
			t.Fatal(err)
		}
	}
	for _, lib := range []string{"@rpath/evil.dylib", "/Users/operator/evil.dylib", "/usr/lib/../local/evil.dylib"} {
		if err := validateDarwinBrokerMachO(bytes.NewReader(darwinBrokerMachFixture(lib)), "arm64"); err == nil {
			t.Fatalf("user/root-loader override allowed: %q", lib)
		}
	}
	if err := validateDarwinBrokerMachO(bytes.NewReader([]byte("#!/bin/sh\necho root")), "arm64"); err == nil {
		t.Fatal("shell script installed as root broker")
	}
	if err := validateDarwinBrokerMachO(bytes.NewReader(darwinBrokerMachFixture("")), "amd64"); err == nil {
		t.Fatal("wrong native architecture accepted")
	}
}

func TestDarwinSystemBrokerRuntimeRequiresExactSealedNativeContext(t *testing.T) {
	for _, kind := range []string{"valid", "wrong-operator", "other-pid", "wrong-mapped-image", "mutable-descriptor", "foreign-unit", "absent-job"} {
		t.Run(kind, func(t *testing.T) {
			e, f, o := newDarwinBrokerFixture(t)
			if _, err := e.install(context.Background(), o); err != nil {
				t.Fatal(err)
			}
			p, _ := e.plan(o)
			pid := 4242
			switch kind {
			case "wrong-operator":
				e.runtimeIdentity = func(NativeSystemServiceBrokerOptions) error { return ErrOperationIdentity }
			case "other-pid":
				pid = 5000
			case "wrong-mapped-image":
				f.wrongProcess = true
			case "mutable-descriptor":
				if err := os.Chmod(p.DescriptorPath, 0644); err != nil {
					t.Fatal(err)
				}
			case "foreign-unit":
				if err := os.WriteFile(p.UnitPath, []byte("foreign"), 0644); err != nil {
					t.Fatal(err)
				}
			case "absent-job":
				f.active = false
			}
			got, err := e.runtime(context.Background(), p.DescriptorPath, pid)
			if kind == "valid" {
				if err != nil || got != *o.SystemBroker {
					t.Fatalf("valid sealed system context failed: %+v %v", got, err)
				}
			} else if err == nil {
				t.Fatalf("unsafe runtime context %s accepted", kind)
			}
		})
	}
}
func TestDarwinSystemBrokerCanceledSetupNeverPublishesBootActivation(t *testing.T) {
	for _, when := range []string{"before-start", "after-validation"} {
		t.Run(when, func(t *testing.T) {
			e, f, o := newDarwinBrokerFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if when == "before-start" {
				cancel()
			} else {
				e.validate = func(RenderedArtifact) Verdict { cancel(); return Verdict{State: VerdictAccepted, Validator: "fixture"} }
			}
			if r, err := e.install(ctx, o); err == nil || r.Active {
				t.Fatal("canceled install became active")
			}
			if _, err := os.Lstat(f.roots.Unit); !os.IsNotExist(err) {
				t.Fatal("canceled setup published boot activation")
			}
			for _, call := range f.calls {
				if strings.HasPrefix(call, "bootstrap") || strings.HasPrefix(call, "enable") {
					t.Fatalf("canceled setup changed native state: %v", f.calls)
				}
			}
		})
	}
}
func TestDarwinSystemBrokerStartRefusesWrongNativeImageBeforeEffects(t *testing.T) {
	e, f, o := newDarwinBrokerFixture(t)
	if _, err := e.install(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	f.wrongProcess = true
	f.calls = nil
	if _, err := e.start(context.Background(), o); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("wrong native image accepted on start: %v", err)
	}
	for _, call := range f.calls {
		if !strings.HasPrefix(call, "query ") {
			t.Fatalf("identity error caused effects: %v", f.calls)
		}
	}
}
func TestDarwinSystemBrokerMissingUnitDoesNotHideUncertainRuntime(t *testing.T) {
	e, f, o := newDarwinBrokerFixture(t)
	if _, err := e.install(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(f.roots.Unit); err != nil {
		t.Fatal(err)
	}
	f.calls = nil
	if active, err := e.start(context.Background(), o); active || !errors.Is(err, ErrOperationOutcomeLost) {
		t.Fatalf("missing plist hid running broker: %v %v", active, err)
	}
	for _, call := range f.calls {
		if !strings.HasPrefix(call, "query ") {
			t.Fatalf("unresolved runtime caused effects: %v", f.calls)
		}
	}
}

func TestDarwinSystemBrokerMetadataAcceptsOnlySealedNativeVarAlias(t *testing.T) {
	cases := []struct {
		name, path string
		mode       os.FileMode
		uid        uint32
		links      uint64
		dir        bool
		target     string
		allowed    bool
	}{
		{"native-var-alias", "/var", os.ModeSymlink | 0777, 0, 1, true, "private/var", true},
		{"native-absolute-var-alias", "/var", os.ModeSymlink | 0777, 0, 1, true, "/private/var", true},
		{"foreign-var-owner", "/var", os.ModeSymlink | 0777, 501, 1, true, "private/var", false},
		{"other-var-target", "/var", os.ModeSymlink | 0777, 0, 1, true, "/Users/operator", false},
		{"foreign-alias", "/Library/Vrooli", os.ModeSymlink | 0777, 0, 1, true, "/private/var", false},
		{"writable-parent", "/Library", os.ModeDir | 0775, 0, 2, true, "", false},
		{"sealed-image", "/Library/Vrooli/vrooli", 0555, 0, 1, false, "", true},
		{"hardlinked-image", "/Library/Vrooli/vrooli", 0555, 0, 2, false, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateDarwinBrokerMetadata(c.path, c.mode, c.uid, c.links, c.dir, c.target)
			if (err == nil) != c.allowed {
				t.Fatalf("metadata policy: allowed=%v err=%v", c.allowed, err)
			}
		})
	}
}

func TestDarwinSystemBrokerACLRejectsEveryNonrootWriteCapability(t *testing.T) {
	for n, name := range []string{"write-data-or-add-file", "append-or-add-directory", "delete", "delete-child", "write-xattr", "write-attributes", "write-security", "change-owner"} {
		t.Run(name, func(t *testing.T) {
			grant := darwinBrokerACLGrant{Allow: true, Rights: 1 << n}
			if err := validateDarwinBrokerACL([]darwinBrokerACLGrant{grant}); !errors.Is(err, ErrOperationIdentity) {
				t.Fatalf("ACL can mutate root execution despite sealed POSIX modes: %v", err)
			}
		})
	}
	if err := validateDarwinBrokerACL([]darwinBrokerACLGrant{{Allow: false, Rights: 4}, {Allow: true, Rights: 0}, {Allow: true, Root: true, Rights: 255}}); err != nil {
		t.Fatalf("deny-only protection/read/root ACL refused: %v", err)
	}
	if err := validateDarwinBrokerACL([]darwinBrokerACLGrant{{Allow: true, Root: true, Rights: 256}}); err == nil {
		t.Fatal("unknown native ACL capability accepted")
	}
}

func TestDarwinSystemBrokerPIDRequiresSealedSlotWithoutProcessObservation(t *testing.T) {
	e, f, o := newDarwinBrokerFixture(t)
	if _, err := e.install(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	e.privileged = func() error { return errors.New("not root") }
	e.process = func(context.Context, nativeSystemBrokerPlan, int) error {
		t.Fatal("manager PID query attempted foreign process observation")
		return ErrOperationIdentity
	}
	originalCheck := e.check
	e.check = func(path string, dir bool) error {
		if path == f.roots.State {
			return errors.New("private state denied")
		}
		return originalCheck(path, dir)
	}
	if pid, err := e.supervisorPID(context.Background()); err != nil || pid != 4242 {
		t.Fatalf("readonly sealed manager PID: %d %v", pid, err)
	}
	f.active = false
	if pid, err := e.supervisorPID(context.Background()); !errors.Is(err, ErrOperationNotFound) || pid != 0 {
		t.Fatalf("absent root manager produced PID: %d %v", pid, err)
	}
}
func TestDarwinOperatorBootNeverCreatesOrMutatesPrivateHome(t *testing.T) {
	e, _, o := newDarwinBrokerFixture(t)
	if _, err := e.install(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(o.SystemBroker.OperatorHome); !os.IsNotExist(err) {
		t.Fatal("root setup created operator private state", err)
	}
	p, err := e.plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if p.StateRoot != o.SystemBroker.OperatorHome || p.Definition.Username != "operator" || p.Definition.Logs.Stdout != "" || p.Definition.Logs.Stderr != "" {
		t.Fatalf("boot identity/log policy incorrect %+v", p.Definition)
	}
}
func TestDarwinOperatorLedgerACLAllowsOnlyOwnerOrRootMutation(t *testing.T) {
	for bit := uint32(1); bit <= 128; bit <<= 1 {
		for _, principal := range []string{"operator", "root", "foreign-user-or-group"} {
			g := darwinBrokerACLGrant{Allow: true, Rights: bit, Owner: principal == "operator", Root: principal == "root"}
			err := validateDarwinOperatorACL([]darwinBrokerACLGrant{g})
			if principal == "foreign-user-or-group" {
				if !errors.Is(err, ErrOperationIdentity) {
					t.Fatal("foreign mutation grant admitted", bit)
				}
			} else if err != nil {
				t.Fatal("owner/root ACL incorrectly refused", principal, bit, err)
			}
		}
	}
	if err := validateDarwinOperatorACL([]darwinBrokerACLGrant{{Allow: false, Rights: 255}, {Allow: true, Rights: 0}}); err != nil {
		t.Fatal("deny-only/read-only ACL refused", err)
	}
	if err := validateDarwinOperatorACL([]darwinBrokerACLGrant{{Allow: true, Owner: true, Rights: 256}}); err == nil {
		t.Fatal("unknown ACL write authority accepted")
	}
	if err := validateDarwinBrokerACL([]darwinBrokerACLGrant{{Allow: true, Owner: true, Rights: 1}}); err == nil {
		t.Fatal("operator grant weakened immutable root artifact policy")
	}
}
