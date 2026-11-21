package resources

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

type nativeExecSessionFixture struct {
	material     openRGBNativeMaterial
	ref          platform.ServiceRef
	prepareError error
	claimError   error
	claimed      bool
	events       *[]string
}

func (s *nativeExecSessionFixture) Prepare(context.Context) (openRGBNativeMaterial, platform.ServiceRef, error) {
	return s.material, s.ref, s.prepareError
}
func (s *nativeExecSessionFixture) Claim(context.Context, platform.ServiceRef, openRGBExecWitness) error {
	if s.claimed {
		return ErrManagedUpdateExecClaimed
	}
	*s.events = append(*s.events, "claim")
	if !errors.Is(s.claimError, ErrManagedUpdateExecRevoked) {
		s.claimed = true
	}
	return s.claimError
}
func (s *nativeExecSessionFixture) Close() error { return nil }

type nativeExecFixture struct {
	capsule openRGBNativeCapsule
	owner   openRGBNativeCapsuleExecOwner
	session *nativeExecSessionFixture
	events  []string
	lookups int
}

func newNativeCapsuleExecFixture(t *testing.T) *nativeExecFixture {
	t.Helper()
	c, m, ref, w, _ := nativeTransportFixture(t)
	image := filepath.Join(t.TempDir(), "launcher")
	if err := os.WriteFile(image, []byte("verified launcher bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(image)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := managedUpdateImageDigest(t.Context(), file)
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	c.LauncherSHA256 = digest
	m.Runtime.LauncherSHA256 = digest
	c.MaterialSHA256 = m.digest()
	ref.RootPID = os.Getpid()
	ref.Native.RootIncarnation = fmt.Sprintf("darwin:%d:456:7", os.Getpid())
	w.Incarnation = ref.Native.RootIncarnation
	f := &nativeExecFixture{capsule: c}
	f.session = &nativeExecSessionFixture{material: m, ref: ref, events: &f.events}
	f.owner = openRGBNativeCapsuleExecOwner{
		lookup: func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) {
			f.lookups++
			return ref, nil
		},
		incarnation: func(int) (string, error) { return ref.Native.RootIncarnation, nil },
		selfImage:   func(platform.ServiceRef) (*os.File, error) { return os.Open(image) },
		connect: func(context.Context, openRGBNativeCapsule) (openRGBNativeCapsuleSession, error) {
			return f.session, nil
		},
		prepareImage: func(context.Context, openRGBNativeMaterial) (managedUpdateInvocation, *os.File, error) {
			file, err := os.Open(image)
			return m.Invocation, file, err
		},
		confineTransport: func(_ context.Context, material openRGBNativeMaterial) (string, error) {
			return hostreqkit.NativeOpenRGBTransportPolicySHA256(material.Policy.Platform)
		},
		observe: func(context.Context, openRGBNativeMaterial, platform.ServiceRef, string) (openRGBExecWitness, error) {
			return w, nil
		},
		verifyProcessOnly: func(int) error { f.events = append(f.events, "no-children"); return nil },
		materialize: func(openRGBNativeMaterial, managedUpdateInvocation) error {
			f.events = append(f.events, "config")
			return nil
		},
		chdir:   func(string) error { return nil },
		replace: func(*os.File, []string, []string) error { f.events = append(f.events, "execute"); return nil },
	}
	return f
}
func TestOpenRGBNativeCapsuleExecRequiresDurableClaimBeforeConfigurationOrExec(t *testing.T) {
	f := newNativeCapsuleExecFixture(t)
	if err := executeOpenRGBNativeCapsule(t.Context(), f.capsule, f.owner); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.events, []string{"claim", "config", "no-children", "execute"}) {
		t.Fatal(f.events)
	}
	if err := executeOpenRGBNativeCapsule(t.Context(), f.capsule, f.owner); !errors.Is(err, ErrManagedUpdateExecClaimed) {
		t.Fatalf("replayed capsule: %v", err)
	}
	if len(f.events) != 4 {
		t.Fatal("replay performed side effects", f.events)
	}
}
func TestOpenRGBNativeCapsuleOwnerDeathOrRevocationNeverExecutes(t *testing.T) {
	for _, when := range []string{"before-prepare", "revoked-before-claim", "after-durable-claim-before-ack"} {
		t.Run(when, func(t *testing.T) {
			f := newNativeCapsuleExecFixture(t)
			switch when {
			case "before-prepare":
				f.session.prepareError = io.EOF
			case "revoked-before-claim":
				f.session.claimError = ErrManagedUpdateExecRevoked
			case "after-durable-claim-before-ack":
				f.session.claimError = io.EOF
			}
			if err := executeOpenRGBNativeCapsule(t.Context(), f.capsule, f.owner); err == nil {
				t.Fatal("accepted unavailable owner/claim")
			}
			for _, event := range f.events {
				if event == "config" || event == "execute" {
					t.Fatal("performed side effects after uncertain claim", f.events)
				}
			}
			if when == "after-durable-claim-before-ack" && !f.session.claimed {
				t.Fatal("lost durable fence")
			}
		})
	}
}
func TestOpenRGBNativeCapsuleRejectsChangedWrapperAndMaterial(t *testing.T) {
	for _, change := range []string{"material", "launcher", "root-after-observation"} {
		t.Run(change, func(t *testing.T) {
			f := newNativeCapsuleExecFixture(t)
			switch change {
			case "material":
				f.session.material.Invocation.Args = append(f.session.material.Invocation.Args, "--unadmitted")
			case "launcher":
				f.capsule.LauncherSHA256 = strings.Repeat("0", 64)
			case "root-after-observation":
				lookup := f.owner.lookup
				f.owner.lookup = func(ctx context.Context, i platform.ServiceIntent) (platform.ServiceRef, error) {
					r, e := lookup(ctx, i)
					if f.lookups > 1 {
						r.RootPID++
					}
					return r, e
				}
			}
			if err := executeOpenRGBNativeCapsule(t.Context(), f.capsule, f.owner); err == nil {
				t.Fatal("accepted changed frozen inputs")
			}
			if len(f.events) > 0 {
				t.Fatal("claim or exec after mismatch", f.events)
			}
		})
	}
}

type nativePreparedChildFixture struct {
	id      openRGBNativeExecutionChild
	events  *[]string
	resumed bool
}

func (c *nativePreparedChildFixture) identity() openRGBNativeExecutionChild { return c.id }
func (c *nativePreparedChildFixture) Resume() error {
	c.resumed = true
	*c.events = append(*c.events, "resume")
	return nil
}
func (c *nativePreparedChildFixture) Wait() error { *c.events = append(*c.events, "wait"); return nil }
func (c *nativePreparedChildFixture) Abort() error {
	if c.resumed {
		return errors.New("already resumed")
	}
	*c.events = append(*c.events, "abort")
	return nil
}
func nativeWindowsCapsuleExecFixture(t *testing.T) (*nativeExecFixture, *nativePreparedChildFixture) {
	t.Helper()
	f := newNativeCapsuleExecFixture(t)
	r := f.session.ref
	r.Intent.NativeAccount = "sid:S-1-5-21-1-2-3-1001"
	r.Unit, _ = platform.ServiceIntent(r.Intent).UnitName()
	n := *r.Native
	n.Version = 2
	n.Platform, n.Manager, n.Domain, n.Name = "windows", "operator-job", "operator", "vrooli-owned-"+r.Intent.Owner
	n.ProcessOnly, n.RootUniqueID = false, ""
	n.Account = r.Intent.NativeAccount
	n.ContainmentID = `Global\VrooliOwned-` + r.Intent.InvocationID
	n.RootIncarnation = fmt.Sprintf("windows:%d:45600", r.RootPID)
	r.RootStartTime = "116444736000000456"
	r.Native = &n
	r.Scope.Path = "operator/" + n.Name
	if err := r.ValidateIdentity(); err != nil {
		t.Fatal(err)
	}
	m := f.session.material
	m.Runtime.NativeAccount = n.Account
	m.NativePolicy.Platform = "windows"
	m.NativePolicy.Account = n.Account
	m.Runtime.OwnerPrincipal = n.Account
	m.NativePolicy.Devices[0].StableID = "windows-hid-port8"
	var err error
	m.Policy, err = freezeOpenRGBNativeExecPolicy(m.NativePolicy, m.Contract.Artifact.SHA256, m.Catalog, m.Contract.ConfigBytes)
	if err != nil {
		t.Fatal(err)
	}
	f.session.material, f.session.ref = m, r
	f.capsule.Intent = platform.ServiceIntent(r.Intent)
	f.capsule.Account = n.Account
	f.capsule.MaterialSHA256 = m.digest()
	f.owner.lookup = func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) { return r, nil }
	f.owner.incarnation = func(int) (string, error) { return n.RootIncarnation, nil }
	f.owner.observe = func(context.Context, openRGBNativeMaterial, platform.ServiceRef, string) (openRGBExecWitness, error) {
		return openRGBExecWitness{Policy: m.Policy, Incarnation: n.RootIncarnation, ClaimSHA256: m.ClaimSHA256, DeviceGenerationSHA256: strings.Repeat("e", 64), SelectedNodes: []string{"windows-hid-port8"}}, nil
	}
	child := &nativePreparedChildFixture{id: openRGBNativeExecutionChild{PID: r.RootPID + 1, Incarnation: fmt.Sprintf("windows:%d:77700", r.RootPID+1), ImageID: "windows-file:00000001:0000000000000002", ImageHandle: 92, SecurityProof: strings.Repeat("f", 64), NativeAccount: n.Account, ContainmentID: n.ContainmentID}, events: &f.events}
	f.owner.prepareChild = func(*os.File, managedUpdateInvocation) (openRGBNativePreparedChild, error) {
		return child, nil
	}
	return f, child
}
func TestOpenRGBNativeCapsuleWindowsChildIsClaimedBeforeResume(t *testing.T) {
	f, _ := nativeWindowsCapsuleExecFixture(t)
	if err := executeOpenRGBNativeCapsule(t.Context(), f.capsule, f.owner); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.events, []string{"claim", "config", "resume", "wait"}) {
		t.Fatal(f.events)
	}
}
func TestOpenRGBNativeCapsuleWindowsChildAbortedOnUncertainClaim(t *testing.T) {
	f, child := nativeWindowsCapsuleExecFixture(t)
	f.session.claimError = io.EOF
	if err := executeOpenRGBNativeCapsule(t.Context(), f.capsule, f.owner); !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	if child.resumed || !reflect.DeepEqual(f.events, []string{"claim", "abort"}) || !f.session.claimed {
		t.Fatal("uncertain claim lost fence or executed child", f.events)
	}
}

func TestOpenRGBNativeCapsuleMaterialHashCanPrecedeNativeRoot(t *testing.T) {
	c, m, _, _, _ := nativeTransportFixture(t)
	m.ClaimSHA256 = ""
	frozen := m.digest()
	m.ClaimSHA256 = strings.Repeat("f", 64)
	if m.digest() != frozen {
		t.Fatal("immutable material hash depends on post-dispatch native root claim")
	}
	c.MaterialSHA256 = frozen
	if err := m.validate(c); err != nil {
		t.Fatal(err)
	}
	m.Invocation.Args = append(m.Invocation.Args, "--unadmitted")
	if m.validate(c) == nil {
		t.Fatal("excluding dynamic claim weakened frozen invocation binding")
	}
}
func TestOpenRGBNativeCapsuleOwnerMappingAcceptsOnlyActualOperator(t *testing.T) {
	for _, item := range []struct {
		owner, actual string
		allowed       bool
	}{
		{"uid:501", "unix:501", true}, {"uid:501", "unix:0", false}, {"uid:501", "unix:502", false},
		{"sid:S-1-5-21-1-2-3-1001", "windows:S-1-5-21-1-2-3-1001", true}, {"sid:S-1-5-21-1-2-3-1001", "windows:S-1-5-18", false}, {"sid:S-1-5-21-1-2-3-1001", "windows:S-1-5-19", false},
		{"uid:501", "windows:S-1-5-18", false},
	} {
		if actual := nativeCapsuleOwnerAllowed(item.owner, item.actual); actual != item.allowed {
			t.Errorf("owner %s peer %s allowed %v", item.owner, item.actual, actual)
		}
	}
}

func TestOpenRGBNativeCapsuleRejectsAmbiguousEnvironmentAndPrivateHomePaths(t *testing.T) {
	for _, change := range []string{"duplicate-env", "stage-in-home", "runtime-in-home", "empty-token"} {
		t.Run(change, func(t *testing.T) {
			c, m, _, _, _ := nativeTransportFixture(t)
			switch change {
			case "duplicate-env":
				m.Invocation.Env = append(m.Invocation.Env, "HOME=/unadmitted")
			case "stage-in-home":
				m.StagePath = filepath.Join(m.Runtime.Home, "sealed")
			case "runtime-in-home":
				m.Invocation.RuntimeRoot = filepath.Join(m.Runtime.Home, "vendor")
				c.RuntimeRoot = m.Invocation.RuntimeRoot
			case "empty-token":
				for i, entry := range m.Invocation.Env {
					if strings.HasPrefix(entry, managedServiceOwnershipTokenEnv+"=") {
						m.Invocation.Env[i] = managedServiceOwnershipTokenEnv + "="
					}
				}
				m.TokenHash = managedServiceTokenHash("")
			}
			c.MaterialSHA256 = m.digest()
			if err := m.validate(c); err == nil {
				t.Fatal("accepted unsafe source-consistent material")
			}
		})
	}
}

func TestOpenRGBNativeCapsuleCannotObserveDevicesOrExecuteWithoutTransportConfinement(t *testing.T) {
	for _, missing := range []bool{false, true} {
		f := newNativeCapsuleExecFixture(t)
		observed := false
		f.owner.observe = func(context.Context, openRGBNativeMaterial, platform.ServiceRef, string) (openRGBExecWitness, error) {
			observed = true
			return openRGBExecWitness{}, nil
		}
		if missing {
			f.owner.confineTransport = nil
		} else {
			f.owner.confineTransport = func(context.Context, openRGBNativeMaterial) (string, error) {
				return "", errors.New("native transport confinement denied")
			}
		}
		if executeOpenRGBNativeCapsule(t.Context(), f.capsule, f.owner) == nil {
			t.Fatal("unconfined native backend accepted")
		}
		if observed || len(f.events) != 0 {
			t.Fatal("unconfined backend observed devices, claimed or executed", f.events)
		}
	}
}

func TestOpenRGBProcessOnlyKernelFailureRetainsClaimWithoutVendorExec(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(fmt.Sprint(missing), func(t *testing.T) {
			f := newNativeCapsuleExecFixture(t)
			f.owner.verifyProcessOnly = func(int) error { return os.ErrPermission }
			if missing {
				f.owner.verifyProcessOnly = nil
			}
			if err := executeOpenRGBNativeCapsule(t.Context(), f.capsule, f.owner); err == nil {
				t.Fatal("process-only payload executed without observed kernel policy")
			}
			if !f.session.claimed || !reflect.DeepEqual(f.events, []string{"claim", "config"}) {
				t.Fatalf("missing durable claim or executed payload: %v %v", f.session.claimed, f.events)
			}
		})
	}
}
