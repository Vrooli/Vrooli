package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/lightingruntime"
)

type openRGBPublicFixture struct {
	c                                  *Controller
	initial                            openRGBInitialAttempt
	original                           ManagedServiceState
	target                             ManagedUpdateContract
	supervisor                         *ManagedServiceSupervisor
	sourceAlive                        bool
	active                             map[string]platform.ServiceRef
	starts, execs, stops, publications int
	quiesces                           int
	failTarget, interrupt              bool
}

func newOpenRGBPublicFixture(t *testing.T) *openRGBPublicFixture {
	return newOpenRGBPublicFixtureFor(t, "")
}

// Native variants exercise the public durable state owner with native policy/ref
// records. OS dispatch and payload execution remain modeled here; capsule/kernel
// behavior has separate integration proofs and still needs native qualification.
func newOpenRGBPublicFixtureFor(t *testing.T, nativeTarget string) *openRGBPublicFixture {
	t.Helper()
	c, _, _ := openRGBHeadlessFixture(t)
	x := newInitialExecFixture(t)
	a := x.a
	catalog, err := os.ReadFile(filepath.Join(c.Root, "resources", openRGBResourceName, a.Contract.ConfigFile))
	if err != nil {
		t.Fatal(err)
	}
	policy := hostreqkit.OpenRGBLinuxPolicy{MachineID: strings.Repeat("a", 32), ArtifactSHA256: a.Contract.Artifact.SHA256, DetectorSHA256: updateTestDigest(string(catalog)), RuleSHA256: updateTestDigest("static rule"), GroupID: 1001, Devices: []hostreqkit.OpenRGBLinuxDevice{{Detector: "ASUS Aura Motherboard", Kind: "hidraw", Topology: "pci0000:00/usb1/1-5/1-5:1.0", HardwareID: "0b05:19af:00"}}}
	paths, err := resourceStoragePaths(openRGBResourceName)
	if err != nil {
		t.Fatal(err)
	}
	a.Runtime.Root, a.Runtime.Home, a.Runtime.StateDir = c.Root, c.Home, paths.StateDir
	a.Catalog, a.DevicePolicy = catalog, &policy
	if nativeTarget != "" {
		p, nativeCatalog, settings := nativeResourcePolicyFixture(t)
		p.Platform = nativeTarget
		if nativeTarget == openRGBPlatformWindows {
			p.Account = "sid:S-1-5-21-1-2-3-1001"
			p.Devices[0].StableID = "windows-hid-port8"
		}
		p.ArtifactSHA256 = a.Contract.Artifact.SHA256
		a.Contract, err = a.Contract.withConfigBytes(a.Contract.ConfigFile, settings)
		if err != nil {
			t.Fatal(err)
		}
		a.Policy, err = freezeOpenRGBNativeExecPolicy(p, p.ArtifactSHA256, nativeCatalog, settings)
		if err != nil {
			t.Fatal(err)
		}
		a.Catalog, a.DevicePolicy, a.NativeDevicePolicy = nativeCatalog, nil, &p
		a.Runtime.NativeAccount, a.Runtime.OwnerPrincipal = p.Account, p.Account
	}
	a.State = openRGBInitialServing
	a.Intent, err = platform.NewServiceIntent("resource-initial", a.ID, a.wrapperSpec())
	if err != nil {
		t.Fatal(err)
	}
	ref := x.ref
	ref.Intent = platform.OperationIntent(a.Intent)
	ref.Unit, _ = a.Intent.UnitName()
	ref.Scope.Path = "/user.slice/" + platform.ServicesSlice + "/" + ref.Unit
	if nativeTarget != "" {
		ref = publicFixtureNativeRef(t, a.Intent, nativeTarget, "1000")
	}
	store, err := newOpenRGBInitialStore()
	if err != nil {
		t.Fatal(err)
	}
	a.State = openRGBInitialPrepared
	if err = store.prepare(t.Context(), a); err != nil {
		t.Fatal(err)
	}
	a, err = store.submit(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	fence, err := newManagedInitialExecStore(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = fence.Prepare(t.Context(), a.fence()); err != nil {
		t.Fatal(err)
	}
	witness := openRGBExecWitness{DeviceGenerationSHA256: updateTestDigest("fixture device generation"), Policy: a.Policy, Incarnation: "boot:" + ref.RootStartTime, ClaimSHA256: a.claimDigest(ref), ContextSHA256: updateTestDigest("fixture initial confinement"), SelectedNodes: []string{"/dev/hidraw9"}}
	if ref.Native != nil {
		witness.Incarnation = ref.Native.RootIncarnation
		witness.TransportSHA256, _ = hostreqkit.NativeOpenRGBTransportPolicySHA256(nativeTarget)
		if nativeTarget == openRGBPlatformWindows {
			witness.ExecutionChild = publicFixtureNativeChild(ref)
		}
	}
	claimed, err := fence.ClaimOpenRGB(t.Context(), a.fence(), ref, witness)
	if err != nil {
		t.Fatal(err)
	}
	a, err = store.change(t.Context(), a, func(a *openRGBInitialAttempt) error { a.State = openRGBInitialServing; return nil })
	if err != nil {
		t.Fatal(err)
	}
	original := ManagedServiceState{InitialLaunch: &ManagedInitialServingLaunch{AttemptID: a.ID, NativeRef: ref}, InstanceID: a.ID, OwnershipTokenHash: a.TokenHash, PID: ref.RootPID, Incarnation: witness.Incarnation, ArtifactPath: a.Contract.Artifact.Path, ArtifactSHA256: a.Contract.Artifact.SHA256, ArtifactVersion: a.Contract.Artifact.Version, StartedAt: *claimed.ClaimedAt}
	supervisor := newManagedServiceSupervisor(filepath.Join(paths.StateDir, managedServiceStateFile), filepath.Join(paths.LogsDir, "service.log"))
	if err = supervisor.writeState(original); err != nil {
		t.Fatal(err)
	}
	if a.NativeDevicePolicy != nil {
		_, err = c.writeOpenRGBNativeDevicePolicy(*a.NativeDevicePolicy)
	} else {
		_, err = c.writeOpenRGBDevicePolicyRecord(policy)
	}
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.SetManagedServiceSource(t.Context(), openRGBResourceName, "managed"); err != nil {
		t.Fatal(err)
	}
	if _, err = c.operatorStateService().Apply(t.Context(), []byte(`{"resources":{"openrgb":{"enabled":true}}}`)); err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(t.TempDir(), "qualified-fixture-next")
	if err = os.WriteFile(targetPath, []byte("qualified-next-image"), 0755); err != nil {
		t.Fatal(err)
	}
	var m ResourceManifest
	encoded, _ := json.Marshal(a.Contract.Manifest)
	if err = json.Unmarshal(encoded, &m); err != nil {
		t.Fatal(err)
	}
	service := *m.ManagedService
	service.Artifact.Version = "fixture-1.2"
	service.Artifact.SHA256ByPlatform = nil
	service.Artifact.SHA256 = updateTestDigest("qualified-next-image")
	for i := range service.Acquisition.Targets {
		if service.Acquisition.Targets[i].URL != "" {
			service.Acquisition.Targets[i].SHA256 = service.Artifact.SHA256
		}
	}
	m.ManagedService = &service
	artifact := ManagedServiceArtifact{Resource: openRGBResourceName, Path: targetPath, Version: service.Artifact.Version, SHA256: service.Artifact.SHA256}
	target, err := newManagedUpdateContractWithConfig(m, artifact, a.Contract.ConfigFile, a.Contract.ConfigBytes)
	if err != nil {
		t.Fatal(err)
	}
	writeResourceManifest(t, c.Root, openRGBResourceName, m)
	f := &openRGBPublicFixture{c: c, initial: a, original: original, target: target, supervisor: supervisor, sourceAlive: true, active: map[string]platform.ServiceRef{}}
	owners := c.openRGBPublicOwners()
	owners.quiesce = func(context.Context) error { f.quiesces++; return nil }
	owners.profile = func(_ context.Context, artifact ManagedServiceArtifact) error {
		if artifact.SHA256 != a.Contract.Artifact.SHA256 && artifact.SHA256 != target.Artifact.SHA256 {
			return errors.New("unknown fixture profile")
		}
		return nil
	}
	owners.grant = func(context.Context, openRGBUpdateRolePolicy) error { return nil }
	owners.source = func(context.Context, ResourceManifest) (managedUpdateAdmissionSource, error) {
		state, err := supervisor.readState()
		if err != nil {
			return managedUpdateAdmissionSource{}, err
		}
		contract := a.Contract
		if state.UpdateLaunch != nil {
			contract = state.UpdateLaunch.Contract
		}
		return managedUpdateAdmissionSource{SourceRevision: updateTestDigest("fixture-source"), Process: state, Contract: contract, CredentialStateDir: filepath.Join(a.root(), "state")}, nil
	}
	owners.target = func(ctx context.Context, m ResourceManifest, s *managedUpdateStore, opts ManagedServiceUpdateAdmission) (ManagedUpdateContract, error) {
		if opts.RollbackOf != "" {
			return c.managedUpdateAdmissionTarget(ctx, m, s, opts)
		}
		return f.target, nil
	}
	owners.servingSelection = func(_ context.Context, u ManagedServiceUpdate) (*lightingruntime.Selection, error) {
		state, err := supervisor.readState()
		if err != nil {
			return nil, err
		}
		if state.UpdateLaunch == nil || state.UpdateLaunch.OperationID != u.ID || state.StoppedAt != nil {
			return nil, nil
		}
		role, err := openRGBFrozenUpdateRoleFor(u, state.UpdateLaunch.Contract)
		if err != nil {
			return nil, err
		}
		id, _, _, err := role.policyIdentity()
		return &lightingruntime.Selection{Version: state.ArtifactVersion, SHA256: state.ArtifactSHA256, DevicePolicyID: id}, err
	}
	c.openRGBUpdateEffects = &owners
	effects := defaultManagedUpdateEffects(c)
	effects.reserve = func(context.Context, ManagedServiceUpdate) error { return nil }
	effects.release = effects.reserve
	effects.original = func(_ context.Context, u ManagedServiceUpdate) (*ManagedUpdateProof, error) {
		alive := f.sourceAlive
		if ref := managedUpdateOriginalNativeRef(u.Original); ref != nil && u.Original.UpdateLaunch != nil {
			_, alive = f.active[ref.Intent.InvocationID]
		}
		if !alive {
			return nil, errors.New("original is absent")
		}
		return updateTestProof(u.Original, time.Now().UTC()), nil
	}
	effects.stopOriginal = func(_ context.Context, u ManagedServiceUpdate) (managedUpdateOriginalStopReceipt, error) {
		f.sourceAlive = false
		if ref := managedUpdateOriginalNativeRef(u.Original); ref != nil {
			delete(f.active, ref.Intent.InvocationID)
		}
		f.stops++
		return managedUpdateOriginalStopReceipt{RootGone: true, TreeQuiescent: true}, nil
	}
	effects.stopRestoration = effects.stopOriginal
	effects.originalQuiescent = func(context.Context, ManagedServiceUpdate) (bool, error) { return !f.sourceAlive, nil }
	effects.portUnbound = func(int) error { return nil }
	effects.prepare = func(ctx context.Context, u ManagedServiceUpdate, contract ManagedUpdateContract, instance string, port int) (ManagedUpdateLaunchIntent, error) {
		runtime := a.Runtime
		runtime.Port = port
		intent := ManagedUpdateLaunchIntent{InstanceID: instance, OwnershipTokenHash: managedServiceTokenHash(instance + "-token"), ContractDigest: contract.Digest, Runtime: runtime}
		intent.NativeIntent, err = platform.NewServiceIntent("resource-update", instance, managedUpdateWrapperSpec(u, intent))
		if err != nil {
			return intent, err
		}
		tokenPath, err := runtime.tokenPath(u.ID, instance)
		if err != nil {
			return intent, err
		}
		if err = os.MkdirAll(filepath.Dir(tokenPath), 0700); err != nil {
			return intent, err
		}
		if err = writeResourceOwnedFileDurable(tokenPath, []byte(instance+"-token"), 0600); err != nil {
			return intent, err
		}
		role, err := openRGBFrozenUpdateRoleFor(u, contract)
		if err != nil {
			return intent, err
		}
		binding, err := role.binding(contract)
		if err != nil {
			return intent, err
		}
		fs, err := newManagedUpdateExecStore(u.Resource, u.ID, instance)
		if err != nil {
			return intent, err
		}
		_, err = fs.Prepare(ctx, managedUpdateExecFence{SchemaVersion: managedUpdateExecFenceSchemaVersion, Resource: u.Resource, OperationID: u.ID, Intent: intent, State: managedUpdateExecFencePrepared, OpenRGBPolicy: &binding})
		return intent, err
	}
	effects.start = func(ctx context.Context, intent platform.ServiceIntent, spec platform.OwnedServiceSpec) (platform.ServiceRef, error) {
		f.starts++
		if nativeTarget != "" {
			ref := publicFixtureNativeRef(t, intent, nativeTarget, fmt.Sprint(1000+f.starts))
			u, _, launch, err := loadOpenRGBNativeUpdateDispatch(ctx, intent, spec)
			if err != nil {
				return ref, err
			}
			fs, err := newManagedUpdateExecStore(openRGBResourceName, u.ID, launch.InstanceID)
			if err != nil {
				return ref, err
			}
			expected, err := fs.read()
			if err != nil {
				return ref, err
			}
			w := openRGBExecWitness{Policy: *expected.OpenRGBPolicy, Incarnation: ref.Native.RootIncarnation, ClaimSHA256: expected.openRGBClaimDigest(ref), ContextSHA256: updateTestDigest("native fixture context"), DeviceGenerationSHA256: updateTestDigest("native fixture devices"), SelectedNodes: []string{"native-hid-port8"}}
			w.TransportSHA256, _ = hostreqkit.NativeOpenRGBTransportPolicySHA256(nativeTarget)
			if nativeTarget == openRGBPlatformWindows {
				w.ExecutionChild = publicFixtureNativeChild(ref)
			}
			if _, err = fs.ClaimOpenRGB(ctx, expected, ref, w); err != nil {
				return ref, err
			}
			f.execs++
			f.active[intent.InvocationID] = ref
			return ref, nil
		}
		unit, _ := intent.UnitName()
		stamp := fmt.Sprint(1000 + f.starts)
		ref := platform.ServiceRef{Intent: platform.OperationIntent(intent), Unit: unit, Scope: platform.ScopeRef{Kind: platform.ScopeKindCgroup, Path: "/user.slice/" + platform.ServicesSlice + "/" + unit}, RootPID: os.Getpid(), RootStartTime: stamp, NativeInvocationID: updateTestDigest(intent.InvocationID)[:32], ScopeDevice: 8, ScopeInode: 11}
		s, _ := newManagedUpdateStore(openRGBResourceName)
		u, err := s.Lookup(spec.Args[2])
		if err != nil {
			return ref, err
		}
		fs, _ := newManagedUpdateExecStore(u.Resource, u.ID, spec.Args[3])
		owner := managedUpdateExecOwner{lookup: func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) { return ref, nil }, selfImage: func(int) (*os.File, error) { return os.Open(os.Args[0]) }, incarnation: func(int) (string, error) { return "boot:" + stamp, nil }, chdir: func(string) error { return nil }, capacity: func(context.Context, ManagedServiceUpdate) error { return nil }, replace: func(*os.File, []string, []string) error { f.execs++; return nil }, openRGB: &openRGBExecOwner{readFence: fs.read, claim: fs.ClaimOpenRGB, confineExecThread: func() (func(), error) { return func() {}, nil }}}
		owner.openRGBImage = func(_ context.Context, artifact ManagedServiceArtifact, base managedUpdateInvocation, _ int) (openRGBAppImageStage, managedUpdateInvocation, *os.File, error) {
			image, err := os.Open(artifact.Path)
			return openRGBAppImageStage{}, base, image, err
		}
		owner.openRGBObserve = func(_ context.Context, _ ManagedUpdateContract, _ managedUpdateInvocation, fence managedUpdateExecFence, ref platform.ServiceRef, inc string) (openRGBExecWitness, error) {
			return openRGBExecWitness{DeviceGenerationSHA256: updateTestDigest("fixture device generation"), Policy: *fence.OpenRGBPolicy, Incarnation: inc, ClaimSHA256: fence.openRGBClaimDigest(ref), ContextSHA256: updateTestDigest("fixture final context"), SelectedNodes: []string{"/dev/hidraw9"}}, nil
		}
		if err = executeManagedUpdateRole(ctx, u, spec.Args[3], owner); err != nil {
			return ref, err
		}
		f.active[intent.InvocationID] = ref
		return ref, nil
	}
	effects.observe = func(_ context.Context, u ManagedServiceUpdate, p ManagedUpdateProcess, contract ManagedUpdateContract) (ManagedUpdateProcess, error) {
		ref, ok := f.active[p.Intent.NativeIntent.InvocationID]
		if !ok {
			return p, errors.New("native fixture is absent")
		}
		now := time.Now().UTC()
		started := now
		if p.State != nil {
			started = p.State.StartedAt
		}
		p.NativeRef = &ref
		incarnation := "boot:" + ref.RootStartTime
		if ref.Native != nil {
			incarnation = ref.Native.RootIncarnation
		}
		p.State = &ManagedServiceState{InstanceID: p.Intent.InstanceID, OwnershipTokenHash: p.Intent.OwnershipTokenHash, PID: ref.RootPID, Incarnation: incarnation, ArtifactPath: contract.Artifact.Path, ArtifactSHA256: contract.Artifact.SHA256, ArtifactVersion: contract.Artifact.Version, StartedAt: started}
		p.Proof = updateTestProof(*p.State, now)
		return p, nil
	}
	effects.wait = func(ctx context.Context, u ManagedServiceUpdate, p ManagedUpdateProcess, contract ManagedUpdateContract) (ManagedUpdateProcess, error) {
		if f.interrupt {
			f.interrupt = false
			return p, context.Canceled
		}
		if f.failTarget && contract.Artifact.SHA256 == f.target.Artifact.SHA256 {
			return p, errors.New("fixture target readiness rejected")
		}
		return effects.observe(ctx, u, p, contract)
	}
	effects.cleanup = func(ctx context.Context, u ManagedServiceUpdate, p ManagedUpdateProcess) error {
		if p.Intent == nil {
			return nil
		}
		fs, _ := newManagedUpdateExecStore(u.Resource, u.ID, p.Intent.InstanceID)
		if _, err := fs.Revoke(ctx, managedUpdateExecFence{SchemaVersion: managedUpdateExecFenceSchemaVersion, Resource: u.Resource, OperationID: u.ID, Intent: *p.Intent, State: managedUpdateExecFencePrepared}); err != nil {
			return err
		}
		delete(f.active, p.Intent.NativeIntent.InvocationID)
		return nil
	}
	effects.publish = func(ctx context.Context, u ManagedServiceUpdate, role string) error {
		state, err := managedUpdatePublishedState(u, role)
		if err != nil {
			return err
		}
		frozen, err := openRGBFrozenUpdateRoleFor(u, state.UpdateLaunch.Contract)
		if err != nil {
			return err
		}
		if err = c.publishOpenRGBPolicy(ctx, u, frozen); err != nil {
			return err
		}
		if err = writeManagedUpdateServingState(supervisor, state); err != nil {
			return err
		}
		f.publications++
		return nil
	}
	c.updateEffects = &effects
	return f
}
func (f *openRGBPublicFixture) admit(t *testing.T, request string) ManagedServiceUpdate {
	t.Helper()
	u, replayed, err := f.c.AdmitManagedServiceUpdate(t.Context(), openRGBResourceName, ManagedServiceUpdateAdmission{RequestID: request, ExpectedVersion: f.target.Artifact.Version, ExpectedSHA256: f.target.Artifact.SHA256})
	if err != nil || replayed {
		t.Fatalf("normal admission: replay=%v err=%v", replayed, err)
	}
	return u
}
func TestOpenRGBPublicExclusiveUpdateUsesRetainedInitialAndNativeWitness(t *testing.T) {
	f := newOpenRGBPublicFixture(t)
	u := f.admit(t, "public-success")
	if f.starts != 0 || f.stops != 0 {
		t.Fatal("admission changed native owner")
	}
	if got, err := f.c.AdvanceManagedUpdateCandidate(t.Context(), openRGBResourceName, u.RequestID); err != nil || got.Phase != UpdatePrepared || f.starts != 0 {
		t.Fatalf("exclusive preflight %s %v", got.Phase, err)
	}
	got, err := f.c.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, u.RequestID, false)
	if err != nil || got.Phase != UpdateSucceeded || f.starts != 1 || f.execs != 1 || f.stops != 1 || f.publications != 1 {
		t.Fatalf("normal success phase=%s err=%v start=%d exec=%d stop=%d pub=%d", got.Phase, err, f.starts, f.execs, f.stops, f.publications)
	}
	if _, err = f.c.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, u.RequestID, false); err != nil || f.starts != 1 {
		t.Fatalf("replayed success dispatched again: %v starts=%d", err, f.starts)
	}
	policy, _ := got.OpenRGBPolicies.Target.Policy.ID()
	launch, found, err := NewOpenRGBHeadless(f.c, nil).inspectRetainedOpenRGBSelection(t.Context(), lightingruntime.Selection{Version: got.Target.Artifact.Version, SHA256: got.Target.Artifact.SHA256, DevicePolicyID: policy})
	if err != nil || !found || launch.policy.Artifact.SHA256 != got.Target.Artifact.SHA256 {
		t.Fatalf("headless retained publication: found=%v err=%v", found, err)
	}
}
func TestOpenRGBPublicTargetFailureRestoresWithoutRedispatch(t *testing.T) {
	f := newOpenRGBPublicFixture(t)
	u := f.admit(t, "target-failure")
	f.failTarget = true
	got, err := f.c.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, u.RequestID, false)
	if err != nil || got.Phase != UpdateFailedRestored || f.starts != 2 || f.execs != 2 {
		t.Fatalf("rollback %s %v starts=%d execs=%d", got.Phase, err, f.starts, f.execs)
	}
}
func TestOpenRGBPublicUnknownTargetRejectedBeforeOriginalStop(t *testing.T) {
	f := newOpenRGBPublicFixture(t)
	f.c.openRGBUpdateEffects.profile = func(context.Context, ManagedServiceArtifact) error { return errOpenRGBReleasePayloadUnqualified }
	_, _, err := f.c.AdmitManagedServiceUpdate(t.Context(), openRGBResourceName, ManagedServiceUpdateAdmission{RequestID: "unknown-target", ExpectedVersion: f.target.Artifact.Version, ExpectedSHA256: f.target.Artifact.SHA256})
	if !errors.Is(err, errOpenRGBReleasePayloadUnqualified) || f.starts != 0 || f.stops != 0 {
		t.Fatalf("unqualified target effects err=%v starts=%d stops=%d", err, f.starts, f.stops)
	}
}

type publicLightingFixture struct {
	f       *openRGBPublicFixture
	bound   string
	applies int
}

func (m *publicLightingFixture) VerifyLightingSetup(context.Context, lightingruntime.Selection) error {
	return nil
}
func (m *publicLightingFixture) EnsureOpenRGB(_ context.Context, s lightingruntime.Selection) (lightingruntime.Backend, error) {
	if err := m.f.c.requireOpenRGBEnabled(context.Background()); err != nil {
		return lightingruntime.Backend{}, err
	}
	if err := ensureManagedUpdateLifecycleAvailable(openRGBResourceName, brokerTransportStart); err != nil {
		return lightingruntime.Backend{}, err
	}
	state, err := m.f.supervisor.readState()
	if err != nil {
		return lightingruntime.Backend{}, err
	}
	if state.ArtifactSHA256 != s.SHA256 || state.ArtifactVersion != s.Version {
		return lightingruntime.Backend{}, errors.New("selection does not match canonical native fixture")
	}
	selected, err := m.f.c.readOpenRGBDevicePolicyRecord()
	if err != nil || selected == nil {
		return lightingruntime.Backend{}, errors.New("selected policy unavailable")
	}
	id, _ := selected.ID()
	if id != s.DevicePolicyID {
		return lightingruntime.Backend{}, errors.New("selected policy drift")
	}
	if state.UpdateLaunch != nil {
		l := state.UpdateLaunch
		fs, _ := newManagedUpdateExecStore(openRGBResourceName, l.OperationID, state.InstanceID)
		fence, err := fs.read()
		if err != nil {
			return lightingruntime.Backend{}, err
		}
		i := ManagedUpdateLaunchIntent{InstanceID: state.InstanceID, OwnershipTokenHash: state.OwnershipTokenHash, ContractDigest: l.Contract.Digest, NativeIntent: platform.ServiceIntent(l.NativeRef.Intent), Runtime: l.Runtime}
		if _, err = verifyOpenRGBConsumedClaim(fence, i, l.Contract, l.NativeRef, state.Incarnation); err != nil {
			return lightingruntime.Backend{}, err
		}
	} else if _, _, err := readOpenRGBInitialServing(state); err != nil {
		return lightingruntime.Backend{}, err
	}
	return lightingruntime.Backend{Endpoint: netip.MustParseAddrPort("127.0.0.1:6742"), Incarnation: state.Incarnation}, nil
}
func (m *publicLightingFixture) DisableOpenRGB(ctx context.Context) error {
	return withOpenRGBDisableLock(ctx, func() error {
		if _, err := m.f.c.operatorStateService().Apply(ctx, []byte(`{"resources":{"openrgb":{"enabled":false}}}`)); err != nil {
			return err
		}
		store, err := newManagedUpdateStore(openRGBResourceName)
		if err != nil {
			return err
		}
		ledger, err := store.read()
		if err != nil {
			return err
		}
		e := m.f.c.managedUpdateEffectOwner()
		if err = retireOpenRGBUpdateRecords(ctx, ledger, revokeOpenRGBUpdateRole, e.cleanup, e.stopOriginal, e.portUnbound, func(ctx context.Context, u ManagedServiceUpdate) error {
			_, err := store.Commit(ctx, u, UpdateFinishDisabled)
			return err
		}); err != nil {
			return err
		}
		initial, err := newOpenRGBInitialStore()
		if err != nil {
			return err
		}
		a, err := initial.read(m.f.initial.ID)
		if err != nil {
			return err
		}
		fence, err := newManagedInitialExecStore(a.ID)
		if err != nil {
			return err
		}
		if err = retireOpenRGBInitialWith(ctx, initial, a, openRGBInitialCleanupOwner{revoke: fence.Revoke, lookup: func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) {
			return platform.ServiceRef{}, errors.New("modeled original absent")
		}, absent: func(context.Context, platform.ServiceIntent) (bool, error) { return true, nil }}); err != nil {
			return err
		}
		state, err := m.f.supervisor.readState()
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		state.StoppedAt = &now
		if state.UpdateLaunch != nil {
			return writeManagedUpdateServingState(m.f.supervisor, state)
		}
		return m.f.supervisor.writeState(state)
	})
}

func (m *publicLightingFixture) EnsureLightingReconciler(_ context.Context, b lightingruntime.Backend) error {
	if m.bound != b.Incarnation {
		m.applies++
		m.bound = b.Incarnation
	}
	return nil
}
func (m *publicLightingFixture) QuiesceLightingReconciler(context.Context) error {
	m.bound = ""
	return nil
}
func enablePublicLighting(t *testing.T, f *openRGBPublicFixture) (*lightingruntime.Controller, *publicLightingFixture) {
	t.Helper()
	m := &publicLightingFixture{f: f}
	control, err := lightingruntime.NewController(f.c.Home, lightingruntime.Owners{Resource: m, Scenario: m})
	if err != nil {
		t.Fatal(err)
	}
	selection := lightingruntime.Selection{Version: f.original.ArtifactVersion, SHA256: f.original.ArtifactSHA256, DevicePolicyID: f.initial.Policy.ID}
	if err = control.Enable(t.Context(), selection); err != nil {
		t.Fatal(err)
	}
	if err = control.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	f.c.openRGBUpdateEffects.quiesce = func(ctx context.Context) error { f.quiesces++; return m.QuiesceLightingReconciler(ctx) }
	return control, m
}
func TestOpenRGBPublicUpdateAndExplicitRollbackRebindNewIncarnations(t *testing.T) {
	f := newOpenRGBPublicFixture(t)
	control, model := enablePublicLighting(t, f)
	oldBound := model.bound
	u := f.admit(t, "forward-with-lighting")
	result, err := f.c.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, u.RequestID, false)
	if err != nil || result.Phase != UpdateSucceeded {
		t.Fatalf("forward %s %v", result.Phase, err)
	}
	current, err := control.Current()
	if err != nil || current.Selection.SHA256 != result.Target.Artifact.SHA256 {
		t.Fatalf("forward selection %+v %v", current, err)
	}
	if model.bound != "" || f.quiesces != 1 {
		t.Fatal("original writer was not drained before native replacement")
	}
	if err = control.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	if model.bound == oldBound || model.applies != 2 {
		t.Fatalf("new incarnation not rebound: %q applies=%d", model.bound, model.applies)
	}
	rollback, replayed, err := f.c.AdmitManagedServiceUpdate(t.Context(), openRGBResourceName, ManagedServiceUpdateAdmission{RequestID: "explicit-rollback", ExpectedVersion: u.Rollback.Artifact.Version, ExpectedSHA256: u.Rollback.Artifact.SHA256, RollbackOf: u.RequestID})
	if err != nil || replayed {
		t.Fatalf("rollback admission %v replay=%v", err, replayed)
	}
	restored, err := f.c.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, rollback.RequestID, false)
	if err != nil || restored.Phase != UpdateSucceeded {
		t.Fatalf("explicit rollback %s %v", restored.Phase, err)
	}
	current, err = control.Current()
	if err != nil || current.Selection.SHA256 != u.Rollback.Artifact.SHA256 {
		t.Fatalf("rollback selection %+v %v", current, err)
	}
	if err = control.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	if model.applies != 3 || f.starts != 2 || f.execs != 2 {
		t.Fatalf("rollback reapply=%d starts=%d execs=%d", model.applies, f.starts, f.execs)
	}
	if err = control.Reconcile(t.Context()); err != nil || model.applies != 3 {
		t.Fatalf("unchanged binding reapplied: %v count=%d", err, model.applies)
	}
}
func TestOpenRGBPublicInterruptedStartingPhaseResumesWithoutRedispatch(t *testing.T) {
	f := newOpenRGBPublicFixture(t)
	u := f.admit(t, "interrupted-native")
	wait := f.c.updateEffects.wait
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	f.c.updateEffects.wait = func(_ context.Context, _ ManagedServiceUpdate, p ManagedUpdateProcess, _ ManagedUpdateContract) (ManagedUpdateProcess, error) {
		cancel()
		return p, ctx.Err()
	}
	stopped, err := f.c.AdvanceManagedServiceUpdate(ctx, openRGBResourceName, u.RequestID, false)
	if !errors.Is(err, context.Canceled) || stopped.Phase != UpdatePrimaryStarting || f.starts != 1 || f.execs != 1 {
		t.Fatalf("interruption %s %v starts=%d exec=%d", stopped.Phase, err, f.starts, f.execs)
	}
	f.c.updateEffects.wait = wait
	completed, err := f.c.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, u.RequestID, false)
	if err != nil || completed.Phase != UpdateSucceeded || f.starts != 1 || f.execs != 1 {
		t.Fatalf("resume %s %v starts=%d exec=%d", completed.Phase, err, f.starts, f.execs)
	}
}
func TestOpenRGBPublicResumeAndExplicitAdvanceCannotBypassOptOut(t *testing.T) {
	f := newOpenRGBPublicFixture(t)
	u := f.admit(t, "disabled-admitted")
	if _, err := f.c.operatorStateService().Apply(t.Context(), []byte(`{"resources":{"openrgb":{"enabled":false}}}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.c.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, u.RequestID, false); err == nil {
		t.Fatal("disabled update advanced")
	}
	if err := f.c.ResumeManagedServiceUpdates(t.Context()); err == nil {
		t.Fatal("disabled admitted resume reported eligible")
	}
	if f.starts != 0 || f.execs != 0 || f.stops != 0 || f.publications != 0 {
		t.Fatalf("opt-out effects start=%d exec=%d stop=%d pub=%d", f.starts, f.execs, f.stops, f.publications)
	}
}
func TestOpenRGBDisableCleanupContinuesAfterRetainedRoleFailure(t *testing.T) {
	var calls []string
	failed := errors.New("retained role stop failed")
	err := completeOpenRGBDisableCleanup(func() error { calls = append(calls, "update"); return failed }, func() error { calls = append(calls, "initial"); return nil }, func() error { calls = append(calls, "canonical"); return nil })
	if !errors.Is(err, failed) || strings.Join(calls, ",") != "update,initial,canonical" {
		t.Fatalf("partial disable skipped writers: %v %v", err, calls)
	}
}

func TestOpenRGBPublicUsesFrozenCatalogAfterCheckoutChanges(t *testing.T) {
	f := newOpenRGBPublicFixture(t)
	u := f.admit(t, "frozen-catalog")
	if err := os.WriteFile(filepath.Join(f.c.Root, "resources", openRGBResourceName, u.Target.ConfigFile), []byte("changed checkout catalog"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := f.c.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, u.RequestID, false)
	if err != nil || got.Phase != UpdateSucceeded {
		t.Fatalf("frozen catalog source %s %v", got.Phase, err)
	}
}
func TestOpenRGBOrdinaryStartCannotImplicitlyAdvanceAdmission(t *testing.T) {
	f := newOpenRGBPublicFixture(t)
	u := f.admit(t, "start-no-update")
	_, err := NewOpenRGBHeadless(f.c, func(context.Context, OpenRGBDevicePolicyCheck) error { return nil }).EnsureOpenRGB(t.Context(), lightingruntime.Selection{Version: f.original.ArtifactVersion, SHA256: f.original.ArtifactSHA256, DevicePolicyID: f.initial.Policy.ID})
	if !errors.Is(err, ErrManagedUpdateConflict) || f.starts != 0 || f.stops != 0 {
		t.Fatalf("ordinary start advanced update: %v starts=%d stops=%d", err, f.starts, f.stops)
	}
	retained, _ := LookupManagedServiceUpdate(openRGBResourceName, u.RequestID)
	if retained.Phase != UpdatePrepared {
		t.Fatalf("ordinary start changed admitted phase %s", retained.Phase)
	}
}
func TestOpenRGBPublicNewerLightingChoiceWins(t *testing.T) {
	f := newOpenRGBPublicFixture(t)
	control, _ := enablePublicLighting(t, f)
	u := f.admit(t, "newer-choice")
	newer := lightingruntime.Selection{Version: "operator-next", SHA256: updateTestDigest("operator-next"), DevicePolicyID: "operator-policy"}
	if err := control.Enable(t.Context(), newer); err != nil {
		t.Fatal(err)
	}
	if _, err := f.c.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, u.RequestID, false); !errors.Is(err, ErrManagedUpdateConflict) {
		t.Fatalf("newer choice was not fenced: %v", err)
	}
	current, _ := control.Current()
	if *current.Selection != newer || f.starts != 0 || f.stops != 0 {
		t.Fatalf("newer selection overwritten %+v start=%d stop=%d", current, f.starts, f.stops)
	}
}
func TestOpenRGBPublicDisableAfterInterruptedUpdateLeavesNoModeledService(t *testing.T) {
	f := newOpenRGBPublicFixture(t)
	control, model := enablePublicLighting(t, f)
	u := f.admit(t, "disable-after-interrupt")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	f.c.updateEffects.wait = func(_ context.Context, _ ManagedServiceUpdate, p ManagedUpdateProcess, _ ManagedUpdateContract) (ManagedUpdateProcess, error) {
		cancel()
		return p, ctx.Err()
	}
	if _, err := f.c.AdvanceManagedServiceUpdate(ctx, openRGBResourceName, u.RequestID, false); !errors.Is(err, context.Canceled) {
		t.Fatalf("interruption: %v", err)
	}
	if err := control.Disable(t.Context()); err != nil {
		t.Fatal(err)
	}
	current, _ := control.Current()
	if current.Phase != lightingruntime.Disabled || len(f.active) != 0 || model.bound != "" {
		t.Fatalf("disabled modeled service remains phase=%s active=%d binding=%q", current.Phase, len(f.active), model.bound)
	}
	starts, execs := f.starts, f.execs
	if err := f.c.ResumeManagedServiceUpdates(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.c.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, u.RequestID, false); err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("disabled explicit advance unexpectedly eligible: %v", err)
	}
	if f.starts != starts || f.execs != execs || len(f.active) != 0 {
		t.Fatal("disabled recovery recreated a native fixture")
	}
	ledger, _ := LookupManagedServiceUpdate(openRGBResourceName, u.RequestID)
	if ledger.Phase != UpdateResourceDisabled {
		t.Fatalf("disabled ledger not terminal: %s", ledger.Phase)
	}
	if err := f.c.RunManagedUpdateExec(t.Context(), openRGBResourceName, u.RequestID, ledger.Primary.Intent.InstanceID); err == nil {
		t.Fatal("late native wrapper bypassed disabled current intent")
	}
}

func TestOpenRGBPublicLostPublicationAcknowledgementConvergesAndReplays(t *testing.T) {
	f := newOpenRGBPublicFixture(t)
	control, model := enablePublicLighting(t, f)
	u := f.admit(t, "lost-publish-ack")
	publish := f.c.updateEffects.publish
	uncertain := true
	f.c.updateEffects.publish = func(ctx context.Context, u ManagedServiceUpdate, role string) error {
		if err := publish(ctx, u, role); err != nil {
			return err
		}
		if uncertain {
			uncertain = false
			return ErrManagedUpdateCommitUnknown
		}
		return nil
	}
	pending, err := f.c.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, u.RequestID, false)
	if !errors.Is(err, ErrManagedUpdateCommitUnknown) || pending.Phase != UpdateCommitting {
		t.Fatalf("publication outcome %s %v", pending.Phase, err)
	}
	current, _ := control.Current()
	if current.Selection.SHA256 != u.Target.Artifact.SHA256 {
		t.Fatal("verified serving publication did not converge alongside lost acknowledgement")
	}
	if err = control.Reconcile(t.Context()); !errors.Is(err, ErrManagedUpdateConflict) || model.bound != "" {
		t.Fatalf("uncommitted update bound a writer: %v bound=%q", err, model.bound)
	}
	completed, err := f.c.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, u.RequestID, false)
	if err != nil || completed.Phase != UpdateSucceeded || f.starts != 1 || f.execs != 1 {
		t.Fatalf("publication replay %s %v starts=%d exec=%d", completed.Phase, err, f.starts, f.execs)
	}
	if err = control.Reconcile(t.Context()); err != nil || model.applies != 2 {
		t.Fatalf("completed publication not rebound: %v applies=%d", err, model.applies)
	}
}
func TestOpenRGBPublicNewerResourceSourceWins(t *testing.T) {
	f := newOpenRGBPublicFixture(t)
	u := f.admit(t, "newer-resource-source")
	patch := fmt.Sprintf(`{"resources":{"openrgb":{"source":{"mode":"host","executable":"/fixture/operator-choice","version":"fixture-next","sha256":"%s"}}}}`, updateTestDigest("operator-source"))
	if _, err := f.c.operatorStateService().Apply(t.Context(), []byte(patch)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.c.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, u.RequestID, false); !errors.Is(err, ErrManagedUpdateConflict) {
		t.Fatalf("newer resource source not fenced: %v", err)
	}
	if f.starts != 0 || f.stops != 0 || f.publications != 0 {
		t.Fatal("old update replaced a newer resource source")
	}
}

func TestOpenRGBPublicFailedNativeDisableFencesPostRunObservation(t *testing.T) {
	f := newOpenRGBPublicFixture(t)
	_, model := enablePublicLighting(t, f)
	u := f.admit(t, "failed-native-disable")
	publish := f.c.updateEffects.publish
	f.c.updateEffects.publish = func(ctx context.Context, u ManagedServiceUpdate, role string) error {
		if err := publish(ctx, u, role); err != nil {
			return err
		}
		return ErrManagedUpdateCommitUnknown
	}
	published, err := f.c.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, u.RequestID, false)
	if !errors.Is(err, ErrManagedUpdateCommitUnknown) || published.Phase != UpdateCommitting {
		t.Fatalf("published checkpoint %s %v", published.Phase, err)
	}
	// Model a stop failure after durable resource opt-out. The canonical native
	// publication remains observable, which must not authorize a later probe.
	f.c.updateEffects.cleanup = func(context.Context, ManagedServiceUpdate, ManagedUpdateProcess) error {
		return errors.New("native stop failed")
	}
	if err = model.DisableOpenRGB(t.Context()); err == nil || len(f.active) != 1 {
		t.Fatalf("native failure model: %v active=%d", err, len(f.active))
	}
	observed := 0
	selector := f.c.openRGBUpdateEffects.servingSelection
	f.c.openRGBUpdateEffects.servingSelection = func(ctx context.Context, u ManagedServiceUpdate) (*lightingruntime.Selection, error) {
		observed++
		return selector(ctx, u)
	}
	starts, pubs := f.starts, f.publications
	if _, err = f.c.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, u.RequestID, false); err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("disabled advance: %v", err)
	}
	if err = f.c.ResumeManagedServiceUpdates(t.Context()); err == nil {
		t.Fatal("disabled resume did not report opt-out")
	}
	if observed != 0 || f.starts != starts || f.publications != pubs {
		t.Fatalf("opt-out observed/published native state: observations=%d starts=%d publications=%d", observed, f.starts, f.publications)
	}
}

func publicFixtureNativeRef(t *testing.T, intent platform.ServiceIntent, target, stamp string) platform.ServiceRef {
	t.Helper()
	unit, err := intent.UnitName()
	if err != nil {
		t.Fatal(err)
	}
	birth, _ := strconv.ParseUint(stamp, 10, 64)
	start := strconv.FormatUint(birth*1000000+7, 10)
	n := &platform.NativeWorkloadIdentity{Version: 2, Platform: target, InvocationID: intent.InvocationID, Account: intent.NativeAccount, BootID: "12345678-1234-1234-1234-123456789abc"}
	if target == openRGBPlatformDarwin {
		n.Manager, n.Domain, n.Name = "launchd", "user/501", "com.vrooli.owned."+strings.TrimSuffix(unit, ".service")
		n.ContainmentID, n.ProcessOnly, n.RootUniqueID = n.Domain+"/"+n.Name, true, stamp
		n.RootIncarnation = fmt.Sprintf("darwin:%d:%s:7", os.Getpid(), stamp)
	} else {
		n.Manager, n.Domain, n.Name = "operator-job", "operator", "vrooli-owned-"+intent.Owner
		n.ContainmentID = `Global\VrooliOwned-` + intent.InvocationID
		n.RootIncarnation = fmt.Sprintf("windows:%d:%d", os.Getpid(), birth*100)
		start = strconv.FormatUint(116444736000000000+birth, 10)
	}
	r := platform.ServiceRef{Intent: platform.OperationIntent(intent), Unit: unit, Scope: platform.ScopeRef{Kind: platform.ScopeKindNative, Path: n.Domain + "/" + n.Name}, RootPID: os.Getpid(), RootStartTime: start, NativeInvocationID: intent.InvocationID, Native: n}
	if err := r.ValidateIdentity(); err != nil {
		t.Fatal(err)
	}
	return r
}

func publicFixtureNativeChild(ref platform.ServiceRef) *openRGBNativeExecutionChild {
	return &openRGBNativeExecutionChild{PID: ref.RootPID + 1, Incarnation: fmt.Sprintf("windows:%d:77700", ref.RootPID+1), ImageID: "windows-file:00000001:0000000000000002", ImageHandle: 92, SecurityProof: updateTestDigest("native child security"), NativeAccount: ref.Native.Account, ContainmentID: ref.Native.ContainmentID}
}
