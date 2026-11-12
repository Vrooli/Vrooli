package resources

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

type openRGBExecFixture struct {
	update   ManagedServiceUpdate
	owner    managedUpdateExecOwner
	fence    managedUpdateExecFence
	ref      platform.ServiceRef
	store    *managedUpdateExecStore
	access   func() (hostreqkit.OpenRGBLinuxEvidence, error)
	observed int
	replaced int
}

func newOpenRGBExecFixture(t *testing.T) *openRGBExecFixture {
	return newOpenRGBExecFixtureAtPort(t, 0)
}

func newOpenRGBExecFixtureAtPort(t *testing.T, port int) *openRGBExecFixture {
	t.Helper()
	u, owner, _ := updateExecTestFixture(t)
	i := *u.Candidate.Intent
	m, err := manifestpkg.Load(filepath.Join(projectRoot(t), "resources", openRGBResourceName, "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	assetPath := m.ManagedService.Config.ContentFile
	catalog, err := readManagedUpdateConfigFile(filepath.Join(projectRoot(t), "resources", openRGBResourceName), assetPath)
	if err != nil {
		t.Fatal(err)
	}
	artifact := u.Target.Artifact
	artifact.Resource, artifact.Version = m.Name, m.ManagedService.Artifact.Version
	m.ManagedService.Artifact.SHA256ByPlatform = nil
	m.ManagedService.Artifact.SHA256 = artifact.SHA256
	const detector = "ASUS Aura Motherboard"
	policy := hostreqkit.OpenRGBLinuxPolicy{
		MachineID: strings.Repeat("a", 32), ArtifactSHA256: artifact.SHA256, DetectorSHA256: updateTestDigest(string(catalog)), RuleSHA256: updateTestDigest("static rule"), GroupID: 1001,
		Devices: []hostreqkit.OpenRGBLinuxDevice{{Detector: detector, Kind: "hidraw", Topology: "pci0000:00/usb1/1-5/1-5:1.0", HardwareID: "0b05:19af:00"}},
	}
	merged, err := mergeOpenRGBConfigContentWithSelection(nil, string(catalog), []string{detector})
	if err != nil {
		t.Fatal(err)
	}
	contract, err := newManagedUpdateContractWithConfig(m, artifact, assetPath, []byte(merged))
	if err != nil {
		t.Fatal(err)
	}
	binding, err := freezeOpenRGBExecPolicy(policy, artifact.SHA256, catalog, contract.ConfigBytes)
	if err != nil {
		t.Fatal(err)
	}
	u.Target, u.Rollback, u.Resource = contract, contract, m.Name
	u.Original.ArtifactPath, u.Original.ArtifactSHA256, u.Original.ArtifactVersion = artifact.Path, artifact.SHA256, artifact.Version
	u.Candidate, u.Phase, u.Error = ManagedUpdateProcess{}, UpdateRestoreStarting, "restore fixture"
	u.PrimaryStopVerified, u.PrimaryQuiescent = true, true
	i.ContractDigest, i.Runtime.Port = contract.Digest, m.Ports[0].Host
	if port > 0 {
		i.Runtime.Port = port
	}
	i.NativeIntent, err = platform.NewServiceIntent("resource-update", i.InstanceID, managedUpdateWrapperSpec(u, i))
	if err != nil {
		t.Fatal(err)
	}
	u.Restored.Intent = &i
	if err := u.Validate(); err != nil {
		t.Fatal(err)
	}
	ref, err := owner.lookup(context.Background(), i.NativeIntent)
	if err != nil {
		t.Fatal(err)
	}
	ref.Intent = platform.OperationIntent(i.NativeIntent)
	ref.Unit, _ = i.NativeIntent.UnitName()
	ref.Scope.Path = "/user.slice/" + platform.ServicesSlice + "/" + ref.Unit
	ref.ScopeDevice, ref.ScopeInode = 8, 11
	f := managedUpdateExecFence{SchemaVersion: managedUpdateExecFenceSchemaVersion, Resource: u.Resource, OperationID: u.ID, Intent: i, State: managedUpdateExecFencePrepared, OpenRGBPolicy: &binding}
	x := &openRGBExecFixture{update: u, owner: owner, fence: f, ref: ref, store: execFenceTestStore(filepath.Join(t.TempDir(), "fence.json"), f)}
	x.owner.lookup = func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) { return ref, nil }
	x.owner.claim = func(context.Context, managedUpdateExecFence, platform.ServiceRef) (managedUpdateExecFence, error) {
		t.Fatal("OpenRGB used the generic exec claim")
		return managedUpdateExecFence{}, nil
	}
	x.owner.replace = func(*os.File, []string, []string) error { x.replaced++; return nil }
	x.owner.openRGBImage = func(_ context.Context, artifact ManagedServiceArtifact, base managedUpdateInvocation, _ int) (openRGBAppImageStage, managedUpdateInvocation, *os.File, error) {
		// Witness fixtures model an already qualified direct native image. The
		// AppImage closure itself has separate content and operated fixtures.
		file, err := os.Open(artifact.Path)
		if err != nil {
			return openRGBAppImageStage{}, base, nil, err
		}
		digest, err := managedUpdateImageDigest(context.Background(), file)
		if err != nil || digest != artifact.SHA256 {
			file.Close()
			return openRGBAppImageStage{}, base, nil, errors.New("synthetic qualified image changed")
		}
		return openRGBAppImageStage{}, base, file, nil
	}
	x.access = func() (hostreqkit.OpenRGBLinuxEvidence, error) {
		return hostreqkit.OpenRGBLinuxEvidence{DeviceGenerationSHA256: updateTestDigest("selected kernel device generation"), StaticGrantConfigured: true, CurrentScopeVerified: true, SelectedNodes: []string{"/dev/hidraw9"}}, nil
	}
	x.owner.openRGB = &openRGBExecOwner{
		readFence: x.store.read, claim: x.store.ClaimOpenRGB, confineExecThread: func() (func(), error) { return func() {}, nil },
	}
	x.owner.openRGBObserve = func(ctx context.Context, c ManagedUpdateContract, inv managedUpdateInvocation, f managedUpdateExecFence, ref platform.ServiceRef, inc string) (openRGBExecWitness, error) {
		x.observed++
		if _, err := os.Stat(inv.ConfigPath); !os.IsNotExist(err) {
			t.Fatalf("role settings were materialized before the witness: %v", err)
		}
		return observeOpenRGBExecWitness(ctx, policy, c, inv, catalog, c.ConfigBytes, f, ref, inc, 1000,
			func(platform.ServiceRef, string) (string, error) { return updateTestDigest("confined launcher"), nil },
			func(_ context.Context, _ hostreqkit.OpenRGBLinuxPolicy, check hostreqkit.OpenRGBLinuxCheck, _ func(int) (string, error)) (hostreqkit.OpenRGBLinuxEvidence, error) {
				if check.PID != 0 || check.Incarnation != "" || check.PolicyID != binding.ID || check.ArtifactSHA256 != binding.ArtifactSHA256 || check.DetectorSHA256 != binding.DetectorCatalogSHA256 {
					t.Fatal("witness observed a different process context or policy")
				}
				return x.access()
			})
	}
	return x
}

func (x *openRGBExecFixture) run() error {
	return executeManagedUpdateRole(context.Background(), x.update, x.fence.Intent.InstanceID, x.owner)
}

func TestOpenRGBExecRequiresVerifiedAccessAndRetainsOneClaimAfterExecFailure(t *testing.T) {
	ctx := context.Background()
	x := newOpenRGBExecFixture(t)
	if _, err := x.store.Prepare(ctx, x.fence); err != nil {
		t.Fatal(err)
	}
	if _, err := x.store.Claim(ctx, x.fence, x.ref); err == nil {
		t.Fatal("generic exec claim bypassed the device witness")
	}
	access := x.access
	denied := errors.New("unrelated I2C access is present")
	x.access = func() (hostreqkit.OpenRGBLinuxEvidence, error) { return hostreqkit.OpenRGBLinuxEvidence{}, denied }
	if err := x.run(); !errors.Is(err, denied) || x.replaced != 0 {
		t.Fatalf("denied access advanced the launch: %v, execs %d", err, x.replaced)
	}
	prepared, err := x.store.read()
	if err != nil || prepared.State != managedUpdateExecFencePrepared || prepared.OpenRGBWitness != nil {
		t.Fatalf("denied access consumed the fence: %+v, %v", prepared, err)
	}
	x.access = access
	crashed := errors.New("exec failed after claim")
	x.owner.replace = func(*os.File, []string, []string) error { x.replaced++; return crashed }
	if err := x.run(); !errors.Is(err, crashed) || x.replaced != 1 {
		t.Fatalf("verified launch did not reach exec: %v, execs %d", err, x.replaced)
	}
	claimed, err := x.store.read()
	if err != nil || claimed.State != managedUpdateExecFenceClaimed || claimed.OpenRGBWitness == nil || *claimed.Claim != x.ref || !reflect.DeepEqual(claimed.OpenRGBWitness.SelectedNodes, []string{"/dev/hidraw9"}) {
		t.Fatalf("exact claim and witness did not survive exec failure: %+v, %v", claimed, err)
	}
	if err := x.run(); !errors.Is(err, ErrManagedUpdateExecClaimed) || x.replaced != 1 || x.observed != 2 {
		t.Fatalf("claimed launch was retried: %v, observations %d, execs %d", err, x.observed, x.replaced)
	}
	other := x.ref
	other.NativeInvocationID = strings.Repeat("b", 32)
	claimed.Claim = &other
	if err := claimed.validate(); err == nil {
		t.Fatal("witness could be moved to another native invocation")
	}
}

func TestOpenRGBExecUncertainClaimCanOnlyBeRecoveredOrRevoked(t *testing.T) {
	ctx := context.Background()
	x := newOpenRGBExecFixture(t)
	if _, err := x.store.Prepare(ctx, x.fence); err != nil {
		t.Fatal(err)
	}
	write := x.store.write
	x.store.write = func(path string, data []byte, mode os.FileMode) error {
		if err := write(path, data, mode); err != nil {
			return err
		}
		return errors.New("lost durable acknowledgement")
	}
	if err := x.run(); !errors.Is(err, ErrManagedUpdateCommitUnknown) || x.replaced != 0 {
		t.Fatalf("uncertain claim permitted exec: %v, execs %d", err, x.replaced)
	}
	x.store.write = write
	if err := x.run(); !errors.Is(err, ErrManagedUpdateExecClaimed) || x.replaced != 0 || x.observed != 1 {
		t.Fatalf("a prior uncertain claim was reused: %v, observations %d, execs %d", err, x.observed, x.replaced)
	}
	legacyExpected := x.fence
	legacyExpected.OpenRGBPolicy = nil // Cleanup loads the exact ref even without the admission input.
	revoked, err := x.store.Revoke(ctx, legacyExpected)
	if err != nil || revoked.State != managedUpdateExecFenceRevoked || revoked.Claim == nil || *revoked.Claim != x.ref || revoked.OpenRGBWitness == nil {
		t.Fatalf("exact recovery witness lost on revocation: %+v, %v", revoked, err)
	}
	if err := x.run(); !errors.Is(err, ErrManagedUpdateExecRevoked) || x.replaced != 0 {
		t.Fatalf("revoked launch was retried: %v, execs %d", err, x.replaced)
	}
}

func TestOpenRGBExecRefusesMissingPolicyAndChangedOwnership(t *testing.T) {
	ctx := context.Background()
	x := newOpenRGBExecFixture(t)
	legacy := x.fence
	legacy.OpenRGBPolicy = nil
	if err := x.store.save(legacy); err != nil {
		t.Fatal(err)
	}
	if err := x.run(); err == nil || x.observed != 0 || x.replaced != 0 {
		t.Fatalf("missing policy advanced launch: %v", err)
	}
	if _, err := x.store.Prepare(ctx, x.fence); !errors.Is(err, ErrManagedUpdateConflict) {
		t.Fatalf("legacy admission was mutable: %v", err)
	}
	if _, err := x.store.Revoke(ctx, legacy); err != nil {
		t.Fatalf("legacy launch could not be revoked: %v", err)
	}
	x = newOpenRGBExecFixture(t)
	if _, err := x.store.Prepare(ctx, x.fence); err != nil {
		t.Fatal(err)
	}
	lookup := x.owner.lookup
	lookups := 0
	x.owner.lookup = func(ctx context.Context, intent platform.ServiceIntent) (platform.ServiceRef, error) {
		ref, err := lookup(ctx, intent)
		lookups++
		if lookups > 1 {
			ref.ScopeInode++
		}
		return ref, err
	}
	if err := x.run(); err == nil || x.observed != 1 || x.replaced != 0 {
		t.Fatalf("changed native ownership advanced launch: %v", err)
	}
	if saved, err := x.store.read(); err != nil || saved.Claim != nil {
		t.Fatalf("changed native ownership consumed the fence: %+v, %v", saved, err)
	}
}

func TestOpenRGBExecRefusesChangedFrozenConfigAndInvocation(t *testing.T) {
	for _, change := range []struct {
		name  string
		apply func(*testing.T, *managedUpdateInvocation)
	}{
		{"config enabled an extra detector", func(t *testing.T, inv *managedUpdateInvocation) {
			data := strings.Replace(inv.ConfigContent, `"ENE SMBus DRAM": false`, `"ENE SMBus DRAM": true`, 1)
			if data == inv.ConfigContent {
				t.Fatal("fixture lost the disabled detector")
			}
			inv.ConfigContent = data
		}},
		{"alternate config directory", func(_ *testing.T, inv *managedUpdateInvocation) { inv.Args[6] = "/outside" }},
		{"non-loopback listener", func(_ *testing.T, inv *managedUpdateInvocation) { inv.Args[2] = "0.0.0.0" }},
		{"AppImage execution mode override", func(_ *testing.T, inv *managedUpdateInvocation) {
			inv.Env = append(inv.Env, "APPIMAGE_EXTRACT_AND_RUN=1")
		}},
	} {
		t.Run(change.name, func(t *testing.T) {
			ctx := context.Background()
			x := newOpenRGBExecFixture(t)
			if _, err := x.store.Prepare(ctx, x.fence); err != nil {
				t.Fatal(err)
			}
			observe := x.owner.openRGBObserve
			x.owner.openRGBObserve = func(ctx context.Context, c ManagedUpdateContract, inv managedUpdateInvocation, f managedUpdateExecFence, ref platform.ServiceRef, inc string) (openRGBExecWitness, error) {
				change.apply(t, &inv)
				return observe(ctx, c, inv, f, ref, inc)
			}
			x.access = func() (hostreqkit.OpenRGBLinuxEvidence, error) {
				t.Fatal("invalid launch reached the device access verifier")
				return hostreqkit.OpenRGBLinuxEvidence{}, nil
			}
			if err := x.run(); err == nil || x.replaced != 0 {
				t.Fatalf("modified frozen inputs advanced launch: %v", err)
			}
			if saved, err := x.store.read(); err != nil || saved.Claim != nil {
				t.Fatalf("modified frozen inputs consumed the fence: %+v, %v", saved, err)
			}
		})
	}
}

func TestOpenRGBExecRequiresExactKernelDeviceGeneration(t *testing.T) {
	x := newOpenRGBExecFixture(t)
	if _, err := x.store.Prepare(t.Context(), x.fence); err != nil {
		t.Fatal(err)
	}
	access := x.access
	x.access = func() (hostreqkit.OpenRGBLinuxEvidence, error) {
		evidence, err := access()
		evidence.DeviceGenerationSHA256 = ""
		return evidence, err
	}
	if err := x.run(); err == nil || x.replaced != 0 {
		t.Fatalf("unknown device generation consumed native exec: %v", err)
	}
	current, err := x.store.read()
	if err != nil || current.State != managedUpdateExecFencePrepared || current.Claim != nil {
		t.Fatalf("failed device generation consumed claim: %+v %v", current, err)
	}
}

func TestOpenRGBExecUsesQualifiedRetainedImageAfterSourcePathDisappears(t *testing.T) {
	x := newOpenRGBExecFixture(t)
	data, err := os.ReadFile(x.update.Rollback.Artifact.Path)
	if err != nil {
		t.Fatal(err)
	}
	retained := filepath.Join(t.TempDir(), "retained-qualified-image")
	if err := os.WriteFile(retained, data, 0500); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(x.update.Rollback.Artifact.Path); err != nil {
		t.Fatal(err)
	}
	prepared := false
	x.owner.openRGBImage = func(ctx context.Context, artifact ManagedServiceArtifact, base managedUpdateInvocation, _ int) (openRGBAppImageStage, managedUpdateInvocation, *os.File, error) {
		prepared = true
		file, err := os.Open(retained)
		if err != nil {
			return openRGBAppImageStage{}, base, nil, err
		}
		digest, err := managedUpdateImageDigest(ctx, file)
		if err != nil || digest != artifact.SHA256 {
			file.Close()
			return openRGBAppImageStage{}, base, nil, errors.New("retained qualified image changed")
		}
		base.Path = retained
		return openRGBAppImageStage{}, base, file, nil
	}
	if _, err := x.store.Prepare(t.Context(), x.fence); err != nil {
		t.Fatal(err)
	}
	if err := x.run(); err != nil || !prepared || x.replaced != 1 {
		t.Fatalf("retained image required missing source pathname: %v prepared=%v execs=%d", err, prepared, x.replaced)
	}
}
