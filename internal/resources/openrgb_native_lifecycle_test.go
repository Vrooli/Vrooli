package resources

import (
	"context"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/lightingruntime"
)

type nativeLifecycleFenceSession struct {
	material openRGBNativeMaterial
	ref      platform.ServiceRef
	store    *managedUpdateExecStore
	expected managedUpdateExecFence
}

func (s *nativeLifecycleFenceSession) Prepare(context.Context) (openRGBNativeMaterial, platform.ServiceRef, error) {
	return s.material, s.ref, nil
}
func (s *nativeLifecycleFenceSession) Claim(ctx context.Context, ref platform.ServiceRef, w openRGBExecWitness) error {
	if !ref.Equal(s.ref) {
		return errors.New("changed native peer")
	}
	if err := validateOpenRGBNativeTransportWitness(s.material, ref, w); err != nil {
		return err
	}
	_, err := s.store.ClaimOpenRGB(ctx, s.expected, ref, w)
	return err
}
func (*nativeLifecycleFenceSession) Close() error { return nil }

type nativeLifecycleInitialFlow struct {
	fixture  *nativeExecFixture
	attempt  openRGBInitialAttempt
	store    *openRGBInitialStore
	fence    *managedUpdateExecStore
	ref      platform.ServiceRef
	executed bool
}

// Native manager, kernel image/access and SDK edges are simulated. Actual
// persisted initial/fence/capsule/boot/intent owners compose here; this fixture
// cannot qualify native APIs, SDK linkage, hardware access or physical output.
func newNativeLifecycleInitialFlow(t *testing.T, id, boot string, previous ...*nativeExecFixture) *nativeLifecycleInitialFlow {
	t.Helper()
	var f *nativeExecFixture
	if len(previous) == 0 {
		f = newNativeCapsuleExecFixture(t)
	} else {
		// Reuse one leased machine/storage scope across the simulated reboot.
		// Creating another fixture would change global canonical capsule paths
		// and invalidate the old persisted launch's exact specification.
		copy := *previous[0]
		f = &copy
		f.events = nil
		session := *copy.session
		f.session = &session
		f.owner.materialize = func(openRGBNativeMaterial, managedUpdateInvocation) error {
			f.events = append(f.events, "config")
			return nil
		}
		f.owner.verifyProcessOnly = func(int) error { f.events = append(f.events, "no-children"); return nil }
	}
	m := f.session.material
	a := openRGBInitialAttempt{SchemaVersion: openRGBInitialSchemaVersion, ID: id, State: openRGBInitialPrepared, Contract: m.Contract, Runtime: m.Runtime, Policy: m.Policy, NativeDevicePolicy: &m.NativePolicy, Catalog: m.Catalog, TokenHash: m.TokenHash}
	var err error
	a.Intent, err = platform.NewServiceIntent("resource-initial", id, a.wrapperSpec())
	if err != nil {
		t.Fatal(err)
	}
	m.Invocation, err = renderManagedServiceInvocation(a.Contract, a.Runtime, a.root(), "private-token")
	if err != nil {
		t.Fatal(err)
	}
	ref := f.session.ref
	ref.Intent = platform.OperationIntent(a.Intent)
	ref.NativeInvocationID = a.Intent.InvocationID
	ref.Unit, _ = a.Intent.UnitName()
	n := *ref.Native
	ref.Native = &n
	n.InvocationID = a.Intent.InvocationID
	n.Account = a.Intent.NativeAccount
	n.BootID = boot
	n.Name = "com.vrooli.owned." + strings.TrimSuffix(ref.Unit, ".service")
	ref.Scope.Path = n.Domain + "/" + n.Name
	n.ContainmentID = ref.Scope.Path
	fs := execFenceTestStore(filepath.Join(t.TempDir(), "fence.json"), a.fence())
	fs.admissionKind = managedExecAdmissionInitial
	if _, err = fs.Prepare(t.Context(), a.fence()); err != nil {
		t.Fatal(err)
	}
	store := &openRGBInitialStore{dir: a.Runtime.StateDir, write: writeResourceOwnedFileDurable}
	if err = store.prepare(t.Context(), a); err != nil {
		t.Fatal(err)
	}
	a, err = store.submit(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	m.ClaimSHA256 = a.claimDigest(ref)
	f.capsule.ID = id
	f.capsule.Intent = a.Intent
	f.capsule.RuntimeRoot = m.Invocation.RuntimeRoot
	f.capsule.MaterialSHA256 = m.digest()
	session := &nativeLifecycleFenceSession{material: m, ref: ref, store: fs, expected: a.fence()}
	flow := &nativeLifecycleInitialFlow{fixture: f, attempt: a, store: store, fence: fs, ref: ref}
	f.owner.lookup = func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) { return ref, nil }
	f.owner.incarnation = func(int) (string, error) { return ref.Native.RootIncarnation, nil }
	f.owner.connect = func(context.Context, openRGBNativeCapsule) (openRGBNativeCapsuleSession, error) { return session, nil }
	prepareImage := f.owner.prepareImage
	f.owner.prepareImage = func(ctx context.Context, material openRGBNativeMaterial) (managedUpdateInvocation, *os.File, error) {
		_, image, err := prepareImage(ctx, material)
		return material.Invocation, image, err
	}
	observe := f.owner.observe
	f.owner.observe = func(ctx context.Context, material openRGBNativeMaterial, r platform.ServiceRef, inc string) (openRGBExecWitness, error) {
		w, err := observe(ctx, material, r, inc)
		w.Policy = material.Policy
		w.Incarnation = inc
		w.ClaimSHA256 = material.ClaimSHA256
		return w, err
	}
	f.owner.replace = func(_ *os.File, _ []string, _ []string) error {
		saved, err := fs.read()
		if err != nil {
			return err
		}
		if saved.State != managedUpdateExecFenceClaimed || saved.Claim == nil || !saved.Claim.Equal(ref) {
			return errors.New("execution preceded durable native claim")
		}
		flow.executed = true
		return nil
	}
	return flow
}

func (f *nativeLifecycleInitialFlow) start(ctx context.Context) error {
	if err := executeOpenRGBNativeCapsule(ctx, f.fixture.capsule, f.fixture.owner); err != nil {
		return err
	}
	var err error
	f.attempt, err = f.store.change(ctx, f.attempt, func(a *openRGBInitialAttempt) error { a.State = openRGBInitialServing; return nil })
	return err
}

type nativeLifecycleLightingResource struct {
	ensure  func(context.Context, lightingruntime.Selection) (lightingruntime.Backend, error)
	disable func(context.Context) error
}

func (*nativeLifecycleLightingResource) VerifyLightingSetup(context.Context, lightingruntime.Selection) error {
	return nil
}
func (r *nativeLifecycleLightingResource) EnsureOpenRGB(ctx context.Context, s lightingruntime.Selection) (lightingruntime.Backend, error) {
	return r.ensure(ctx, s)
}
func (r *nativeLifecycleLightingResource) DisableOpenRGB(ctx context.Context) error {
	return r.disable(ctx)
}

func TestNativeOpenRGBBootFreshCapsuleSelectedBackendAndDisableRetirement(t *testing.T) {
	prior := newNativeLifecycleInitialFlow(t, "before-boot", "12345678-1234-1234-1234-123456789abc")
	if err := prior.start(t.Context()); err != nil {
		t.Fatal(err)
	}
	oldClaim, err := prior.fence.read()
	if err != nil {
		t.Fatal(err)
	}
	var fresh *nativeLifecycleInitialFlow
	selection := lightingruntime.Selection{Version: prior.attempt.Contract.Artifact.Version, SHA256: prior.attempt.Contract.Artifact.SHA256, DevicePolicyID: prior.attempt.Policy.ID}
	bootRef := nextBootRef(prior.ref, "22345678-1234-1234-1234-123456789abc", 456)
	present := map[string]bool{prior.ref.Native.InvocationID: true}
	quiesced := false
	stops, releases := 0, 0
	cleanup := openRGBInitialCleanupOwner{
		revoke: prior.fence.Revoke,
		lookup: func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) { return bootRef, nil },
		stop: func(_ context.Context, r platform.ServiceRef) (platform.OperationStopReceipt, error) {
			stops++
			return platform.OperationStopReceipt{Ref: platform.OperationRef(r), NativeQuiescent: true}, nil
		},
		release: func(_ context.Context, r platform.ServiceRef) error {
			releases++
			delete(present, r.Native.InvocationID)
			return nil
		},
		released: func(_ context.Context, r platform.ServiceRef) (bool, error) {
			return !present[r.Native.InvocationID], nil
		},
	}
	cleanup.retireBoot = func(ctx context.Context, f managedUpdateExecFence, current *platform.ServiceRef) (bool, error) {
		return retireOpenRGBNativeBootWith(ctx, prior.fence, f, current, cleanup)
	}
	var control *lightingruntime.Controller
	resource := &nativeLifecycleLightingResource{}
	resource.ensure = func(ctx context.Context, s lightingruntime.Selection) (lightingruntime.Backend, error) {
		if s != selection {
			return lightingruntime.Backend{}, errors.New("selection changed across native lifecycle")
		}
		if fresh != nil && fresh.executed && present[fresh.ref.Native.InvocationID] {
			retained, err := fresh.fence.read()
			if err != nil {
				return lightingruntime.Backend{}, err
			}
			if _, err = verifyOpenRGBConsumedClaim(retained, fresh.attempt.fence().Intent, fresh.attempt.Contract, fresh.ref, fresh.ref.Native.RootIncarnation); err != nil {
				return lightingruntime.Backend{}, err
			}
			return lightingruntime.Backend{Endpoint: netip.MustParseAddrPort("127.0.0.1:6742"), Incarnation: fresh.ref.Native.RootIncarnation}, nil
		}
		if err := retireOpenRGBInitialWith(ctx, prior.store, prior.attempt, cleanup); err != nil {
			return lightingruntime.Backend{}, err
		}
		if present[prior.ref.Native.InvocationID] {
			return lightingruntime.Backend{}, errors.New("old native role still present")
		}
		fresh = newNativeLifecycleInitialFlow(t, "after-boot", bootRef.Native.BootID, prior.fixture)
		if err := fresh.start(ctx); err != nil {
			return lightingruntime.Backend{}, err
		}
		present[fresh.ref.Native.InvocationID] = true
		return lightingruntime.Backend{Endpoint: netip.MustParseAddrPort("127.0.0.1:6742"), Incarnation: fresh.ref.Native.RootIncarnation}, nil
	}
	resource.disable = func(ctx context.Context) error {
		intent, err := control.Current()
		if err != nil {
			return err
		}
		if intent.Phase != lightingruntime.Disabling || !quiesced {
			return errors.New("native retirement preceded durable fence or quiesce")
		}
		current := cleanup
		current.revoke = fresh.fence.Revoke
		current.retireBoot = func(ctx context.Context, f managedUpdateExecFence, ref *platform.ServiceRef) (bool, error) {
			return retireOpenRGBNativeBootWith(ctx, fresh.fence, f, ref, current)
		}

		current.lookup = func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) { return fresh.ref, nil }
		return retireOpenRGBInitialWith(ctx, fresh.store, fresh.attempt, current)
	}
	scenario := openRGBNativeStopScenario{quiesce: func(context.Context) error { quiesced = true; return nil }}
	control, err = lightingruntime.NewController(t.TempDir(), lightingruntime.Owners{Resource: resource, Scenario: scenario})
	if err != nil {
		t.Fatal(err)
	}
	if err = control.Enable(t.Context(), selection); err != nil {
		t.Fatal(err)
	}
	backend, err := resource.EnsureOpenRGB(t.Context(), selection)
	if err != nil || backend.Incarnation != fresh.ref.Native.RootIncarnation || !fresh.executed {
		t.Fatalf("fresh selected backend not acknowledged: %+v %v", backend, err)
	}
	retired, err := prior.fence.read()
	if err != nil {
		t.Fatal(err)
	}
	if retired.Claim == nil || !retired.Claim.Equal(*oldClaim.Claim) || len(retired.NativeBootRetirements) != 1 || retired.NativeBootRetirements[0].ReleasedAt == nil {
		t.Fatal("boot recovery rewrote the consumed claim or omitted retirement")
	}
	beforeRecovery, err := fresh.fence.read()
	if err != nil {
		t.Fatal(err)
	}
	eventsBeforeRecovery := slices.Clone(fresh.fixture.events)
	// A fresh operator worker reconnects to the retained backend; it cannot
	// consume a second capsule or replace the original admission receipt.
	adopted, err := resource.EnsureOpenRGB(t.Context(), selection)
	if err != nil {
		t.Fatal(err)
	}
	afterRecovery, err := fresh.fence.read()
	if err != nil {
		t.Fatal(err)
	}
	if adopted != backend || !slices.Equal(fresh.fixture.events, eventsBeforeRecovery) || !afterRecovery.Claim.Equal(*beforeRecovery.Claim) || !afterRecovery.ClaimedAt.Equal(*beforeRecovery.ClaimedAt) {
		t.Fatal("worker reconnect replaced admission or repeated execution", adopted, fresh.fixture.events)
	}
	if err = control.Disable(t.Context()); err != nil {
		t.Fatal(err)
	}
	intent, err := control.Current()
	if err != nil || intent.Phase != lightingruntime.Disabled || len(present) != 0 || stops != 2 || releases != 2 {
		t.Fatalf("disable incomplete: %+v %v active=%v stop=%d release=%d", intent, err, present, stops, releases)
	}
	stopped, err := fresh.store.read(fresh.attempt.ID)
	if err != nil || stopped.State != openRGBInitialStopped {
		t.Fatalf("fresh role retirement not durable: %+v %v", stopped, err)
	}
}
