package resources

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/lightingruntime"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/operatorstate"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

func openRGBHeadlessFixture(t *testing.T) (*Controller, ResourceManifest, lightingruntime.Selection) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" || os.Geteuid() <= 0 {
		t.Skip("fixture exercises the unprivileged Linux amd64 launch owner")
	}
	root := t.TempDir()
	t.Setenv("USER", "")
	t.Setenv("HOME", root)
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", filepath.Join(root, "resource-storage"))
	t.Setenv("VROOLI_RESOURCE_ARTIFACT_DIR", filepath.Join(root, "artifacts"))
	manifest, err := manifestpkg.Load(filepath.Join(projectRootForResourcesTest(t), "resources", openRGBResourceName, "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifest.ManagedService.Artifact.SHA256ByPlatform["linux-amd64"] = updateTestDigest("fixture-image")
	manifest.ManagedService.Acquisition.Targets[0].SHA256 = updateTestDigest("fixture-image")
	assetPath := filepath.Join(root, "resources", openRGBResourceName, filepath.FromSlash(manifest.ManagedService.Config.ContentFile))
	asset, err := os.ReadFile(filepath.Join(projectRootForResourcesTest(t), "resources", openRGBResourceName, filepath.FromSlash(manifest.ManagedService.Config.ContentFile)))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(assetPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(assetPath, asset, 0o600); err != nil {
		t.Fatal(err)
	}
	writeResourceManifest(t, root, openRGBResourceName, manifest)
	copyOperatorStateSchemaForTest(t, root)
	c := NewController(root, root)
	c.operatorState = operatorstate.New(operatorstate.Config{RepoRoot: root, StateRoot: filepath.Join(root, "operator-state")})
	policy := hostreqkit.OpenRGBLinuxPolicy{
		MachineID: strings.Repeat("a", 32), ArtifactSHA256: updateTestDigest("fixture-image"), DetectorSHA256: updateTestDigest(string(asset)),
		RuleSHA256: strings.Repeat("d", 64), GroupID: 1500,
		Devices: []hostreqkit.OpenRGBLinuxDevice{
			{Detector: "ASUS Aura Motherboard", Kind: "hidraw", Topology: "pci0000:00/0000:03:00.0/usb1/1-8/1-8:1.0", HardwareID: "0b05:19af:00"},
			{Detector: "ASUS TUF GeForce RTX 4070 Ti SUPER Gaming OC", Kind: "i2c", Topology: "pci0000:00/0000:01:00.0", HardwareID: "10de:2705:1043:8957", Adapter: "NVIDIA I2C adapter 1"},
		},
	}
	policyID, err := c.StoreOpenRGBDevicePolicy(policy)
	if err != nil {
		t.Fatal(err)
	}
	selection := lightingruntime.Selection{Version: manifest.ManagedService.Artifact.Version, SHA256: updateTestDigest("fixture-image"), DevicePolicyID: policyID}
	return c, manifest, selection
}

func TestOpenRGBGenericManagedServiceStartFailsBeforeEffects(t *testing.T) {
	c, manifest, _ := openRGBHeadlessFixture(t)
	if _, err := c.SetManagedServiceSource(t.Context(), openRGBResourceName, "managed"); err != nil {
		t.Fatal(err)
	}
	artifactPath, err := managedServiceArtifactPath(c, manifest)
	if err != nil {
		t.Fatal(err)
	}
	for name, start := range map[string]func() error{
		"ordinary resource start": func() error {
			return (managedServiceDriver{}).Run(t.Context(), c, Resource{Name: openRGBResourceName}, manifest, "start", nil, nil, nil)
		},
		"private start helper": func() error {
			return (managedServiceDriver{}).startPrivate(t.Context(), c, manifest, nil)
		},
		"private launch helper": func() error {
			return (managedServiceDriver{}).startPrivateAt(t.Context(), c, manifest, nil, filepath.Join(c.Root, "staged-openrgb"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := start(); !errors.Is(err, ErrOpenRGBPreExecWitnessUnavailable) && !(name == "ordinary resource start" && err != nil && strings.Contains(err.Error(), "explicit resource enablement")) {
				t.Fatalf("generic OpenRGB launch = %v, want pre-exec witness refusal", err)
			}
		})
	}
	paths, err := resourceStoragePaths(openRGBResourceName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(paths.ConfigDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("generic start wrote OpenRGB config before refusal: %v", err)
	}
	if _, err := os.Lstat(artifactPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("generic start acquired an OpenRGB artifact before refusal: %v", err)
	}
}

func TestOpenRGBManagedUpdateAndRollbackLaunchesFailClosedBeforeAdmission(t *testing.T) {
	c := NewController(t.TempDir(), t.TempDir())
	ctx := t.Context()
	if _, _, err := c.AdmitManagedServiceUpdate(ctx, openRGBResourceName, ManagedServiceUpdateAdmission{RequestID: "update-request", ExpectedVersion: "1.1", ExpectedSHA256: updateTestDigest("staged")}); err == nil || errors.Is(err, ErrOpenRGBPreExecWitnessUnavailable) {
		t.Fatalf("OpenRGB update admission = %v, want concrete missing admission/enablement refusal", err)
	}
	if _, err := c.AdvanceManagedServiceUpdate(ctx, openRGBResourceName, "update-request", false); err == nil || errors.Is(err, ErrOpenRGBPreExecWitnessUnavailable) {
		t.Fatalf("OpenRGB update/rollback resume = %v, want concrete missing admission/enablement refusal", err)
	}
	if _, err := c.AdvanceManagedUpdateCandidate(ctx, openRGBResourceName, "update-request"); err == nil || errors.Is(err, ErrOpenRGBPreExecWitnessUnavailable) {
		t.Fatalf("OpenRGB candidate launch = %v, want concrete missing admission/enablement refusal", err)
	}
}

func TestOpenRGBHeadlessInspectsOnlyStagedSelection(t *testing.T) {
	c, manifest, selection := openRGBHeadlessFixture(t)
	var checks []OpenRGBDevicePolicyCheck
	var policyErr error
	owner := NewOpenRGBHeadless(c, func(_ context.Context, check OpenRGBDevicePolicyCheck) error {
		checks = append(checks, check)
		return policyErr
	})
	ctx := t.Context()
	if _, err := NewOpenRGBHeadless(c, nil).EnsureOpenRGB(ctx, selection); !errors.Is(err, ErrOpenRGBDevicePolicyUnavailable) {
		t.Fatalf("absent host verifier = %v", err)
	}
	if err := owner.VerifyLightingSetup(ctx, selection); err == nil || !strings.Contains(err.Error(), "explicit resource enablement and source selection") {
		t.Fatalf("implicit enablement = %v", err)
	}
	if _, err := c.operatorState.Apply(ctx, []byte(`{"resources":{"openrgb":{"enabled":true,"source":{"mode":"managed"}}}}`)); err != nil {
		t.Fatal(err)
	}
	path, err := managedServiceArtifactPath(c, manifest)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := owner.EnsureOpenRGB(ctx, selection)
	if err == nil || !strings.Contains(err.Error(), "verify staged OpenRGB artifact") || backend != (lightingruntime.Backend{}) || len(checks) != 0 {
		t.Fatalf("missing staged bytes reached launch admission: %+v, %v, checks=%v", backend, err, checks)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("autonomous start changed the staged path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("fixture-image"), 0o700); err != nil {
		t.Fatal(err)
	}
	wrong := selection
	wrong.SHA256 = updateTestDigest("other-image")
	if err := owner.VerifyLightingSetup(ctx, wrong); err == nil || len(checks) != 0 {
		t.Fatalf("wrong digest reached policy verification: %v, checks=%v", err, checks)
	}
	wrongPolicy := selection
	wrongPolicy.DevicePolicyID = strings.Repeat("f", 64)
	if err := owner.VerifyLightingSetup(ctx, wrongPolicy); err == nil || !strings.Contains(err.Error(), "machine-local OpenRGB device policy differs") || len(checks) != 0 {
		t.Fatalf("wrong machine-local policy reached permission verification: %v, checks=%v", err, checks)
	}
	policyErr = errors.New("operator ACL permits unrelated I2C adapters")
	backend, err = owner.EnsureOpenRGB(ctx, selection)
	if !errors.Is(err, policyErr) || backend != (lightingruntime.Backend{}) {
		t.Fatalf("broad permissions reached launch: %+v, %v", backend, err)
	}
	policyErr = nil
	if err := c.SetEnabled(openRGBResourceName, false); err != nil {
		t.Fatal(err)
	}
	if err := owner.VerifyLightingSetup(ctx, selection); err != nil {
		t.Fatalf("setup without same-context witness = %v, want fail-closed witness error", err)
	}
	asset, err := os.ReadFile(filepath.Join(c.Root, "resources", openRGBResourceName, filepath.FromSlash(manifest.ManagedService.Config.ContentFile)))
	if err != nil {
		t.Fatal(err)
	}
	want := OpenRGBDevicePolicyCheck{
		PolicyID: selection.DevicePolicyID, Artifact: ManagedServiceArtifact{Resource: openRGBResourceName, Path: path, Version: selection.Version, SHA256: selection.SHA256},
		DetectorSHA256: updateTestDigest(string(asset)), EffectiveUID: os.Geteuid(),
	}
	if !reflect.DeepEqual(checks, []OpenRGBDevicePolicyCheck{want, want}) {
		t.Fatalf("wrong prospective qualification input: %+v", checks)
	}
	paths, err := resourceStoragePaths(openRGBResourceName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(paths.ConfigDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read-only qualification touched the resource config directory: %v", err)
	}
}

func TestOpenRGBHeadlessOptOutWithoutMachinePolicyDoesNotVerifyOrProbe(t *testing.T) {
	c, _, selection := openRGBHeadlessFixture(t)
	if _, err := c.operatorState.Apply(t.Context(), []byte(`{"resources":{"openrgb":{"enabled":true,"source":{"mode":"managed"}}}}`)); err != nil {
		t.Fatal(err)
	}
	path, err := openRGBDevicePolicyPath(c.Home)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	manifest, err := c.managedServiceArtifactManifest(openRGBResourceName)
	if err != nil {
		t.Fatal(err)
	}
	artifactPath, err := managedServiceArtifactPath(c, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, []byte("fixture-image"), 0o700); err != nil {
		t.Fatal(err)
	}
	verified := false
	owner := NewOpenRGBHeadless(c, func(context.Context, OpenRGBDevicePolicyCheck) error {
		verified = true
		return nil
	})
	if err := owner.VerifyLightingSetup(t.Context(), selection); err == nil || !strings.Contains(err.Error(), "machine-local OpenRGB device policy") {
		t.Fatalf("missing explicit policy setup = %v, want fail closed", err)
	}
	if verified {
		t.Fatal("opted-out machine policy reached host permission verifier")
	}
	paths, err := resourceStoragePaths(openRGBResourceName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(paths.ConfigDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("opt-out setup created a detector config before selection: %v", err)
	}
}

func TestOpenRGBHeadlessRejectsActiveManagedUpdate(t *testing.T) {
	c, manifest, selection := openRGBHeadlessFixture(t)
	update := updateTestInitial(t)
	update.Resource = openRGBResourceName
	for _, target := range []*ManagedUpdateContract{&update.Target, &update.Rollback} {
		artifact := target.Artifact
		artifact.Resource = openRGBResourceName
		manifest.ManagedService.Artifact.Version = artifact.Version
		manifest.ManagedService.Artifact.SHA256ByPlatform["linux-amd64"] = artifact.SHA256
		contract, err := newManagedUpdateContractForResource(manifest, artifact, filepath.Join(c.Root, "resources", openRGBResourceName))
		if err != nil {
			t.Fatal(err)
		}
		*target = contract
	}
	store, err := newManagedUpdateStore(openRGBResourceName)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Create(t.Context(), update, false); err != nil {
		t.Fatal(err)
	}
	owner := NewOpenRGBHeadless(c, func(context.Context, OpenRGBDevicePolicyCheck) error {
		t.Fatal("active update reached device qualification")
		return nil
	})
	if _, err := owner.EnsureOpenRGB(t.Context(), selection); !errors.Is(err, ErrManagedUpdateConflict) {
		t.Fatalf("start bypassed update fence: %v", err)
	}
	if err := owner.DisableOpenRGB(t.Context()); err == nil || !strings.Contains(err.Error(), "unknown native ownership") {
		t.Fatalf("disable invented unknown original retirement: %v", err)
	}
	state, err := c.operatorStateService().Load(t.Context())
	if err != nil || state.Resources[openRGBResourceName].Enabled == nil || *state.Resources[openRGBResourceName].Enabled {
		t.Fatalf("opt-out intent not durable: %+v %v", state, err)
	}
	retained, err := store.Lookup(update.RequestID)
	if err != nil || retained.Phase.Terminal() {
		t.Fatalf("unknown cleanup falsely completed: %s %v", retained.Phase, err)
	}
}

func TestOpenRGBHeadlessDisableNeedsExactAbsenceAndPreservesSource(t *testing.T) {
	c, manifest, _ := openRGBHeadlessFixture(t)
	ctx := t.Context()
	if _, err := c.operatorState.Apply(ctx, []byte(`{"resources":{"openrgb":{"enabled":true,"source":{"mode":"managed"}}}}`)); err != nil {
		t.Fatal(err)
	}
	statusErr := errors.New("ownership unavailable")
	absent := network.TCPListenerSnapshot{Known: true}
	occupied := network.TCPListenerSnapshot{Known: true, Ports: map[int][]network.SnapshotListener{manifest.Ports[0].Host: nil}}
	if err := confirmOpenRGBStopped(false, statusErr, manifest, absent); !errors.Is(err, statusErr) {
		t.Fatalf("unavailable process evidence = %v", err)
	}
	if err := confirmOpenRGBStopped(true, nil, manifest, absent); err == nil {
		t.Fatal("running process reported as stopped")
	}
	for _, listeners := range []network.TCPListenerSnapshot{{}, occupied} {
		if err := confirmOpenRGBStopped(false, nil, manifest, listeners); err == nil {
			t.Fatalf("unconfirmed listener reported as absent: %+v", listeners)
		}
	}
	if err := confirmOpenRGBStopped(false, nil, manifest, absent); err != nil {
		t.Fatal(err)
	}
	owner := NewOpenRGBHeadless(c, nil)
	for range 2 {
		if err := owner.persistDisabled(ctx); err != nil {
			t.Fatal(err)
		}
		state, err := c.operatorState.Load(ctx)
		choice := state.Resources[openRGBResourceName]
		if err != nil || choice.Enabled == nil || *choice.Enabled || choice.Source == nil || choice.Source.Mode != "managed" {
			t.Fatalf("disabled readback lost source selection: %+v, %v", choice, err)
		}
	}
}

func TestOpenRGBHeadlessPublishesOnlyVerifiedOwnedProcess(t *testing.T) {
	endpoint := netip.MustParseAddrPort("127.0.0.1:6742")
	artifact := ManagedServiceArtifact{Resource: openRGBResourceName, Path: "/fixture/openrgb", Version: "1.1", SHA256: updateTestDigest("image")}
	launch := openRGBHeadlessLaunch{endpoint: endpoint, ready: ResourceHealthCheck{Target: endpoint.String(), Type: manifestpkg.HealthCheckOpenRGBV6}, policy: OpenRGBDevicePolicyCheck{
		PolicyID: "selected-physical-devices", Artifact: artifact, DetectorSHA256: updateTestDigest("detectors"), EffectiveUID: 1000,
	}}
	state := ManagedServiceState{PID: 123, Incarnation: "boot:123", InstanceID: "instance", OwnershipTokenHash: updateTestDigest("token"), ArtifactPath: artifact.Path, ArtifactVersion: artifact.Version, ArtifactSHA256: artifact.SHA256}
	witnessFailure := errors.New("evidence unavailable")
	for _, test := range []struct {
		name  string
		calls []string
	}{
		{"ready", []string{"listener", "device-policy", "SDK", "listener", "status", "device-policy"}},
		{"foreign listener", []string{"listener"}},
		{"wide device access", []string{"listener", "device-policy"}},
		{"SDK not ready", []string{"listener", "device-policy", "SDK"}},
		{"listener changed", []string{"listener", "device-policy", "SDK", "listener"}},
		{"process changed", []string{"listener", "device-policy", "SDK", "listener", "status"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []string
			owner := NewOpenRGBHeadless(nil, func(_ context.Context, check OpenRGBDevicePolicyCheck) error {
				calls = append(calls, "device-policy")
				want := launch.policy
				want.PID, want.Incarnation = state.PID, state.Incarnation
				if check != want {
					t.Fatalf("device policy process witness = %+v, want %+v", check, want)
				}
				if test.name == "wide device access" {
					return witnessFailure
				}
				return nil
			})
			observed := 0
			observer := openRGBReadiness{
				status: func() (ManagedServiceState, bool, error) {
					calls = append(calls, "status")
					updated := state
					if test.name == "process changed" {
						updated.Incarnation = "boot:999"
					}
					return updated, true, nil
				},
				listener: func(got netip.AddrPort, pid int) error {
					calls = append(calls, "listener")
					if got != endpoint || pid != state.PID {
						t.Fatalf("listener evidence requested for %s / %d", got, pid)
					}
					observed++
					if test.name == "foreign listener" || (test.name == "listener changed" && observed == 2) {
						return witnessFailure
					}
					return nil
				},
				protocol: func(_ context.Context, check ResourceHealthCheck) error {
					calls = append(calls, "SDK")
					if check.Target != endpoint.String() || check.Type != manifestpkg.HealthCheckOpenRGBV6 {
						t.Fatalf("unexpected readiness check: %+v", check)
					}
					if test.name == "SDK not ready" {
						return witnessFailure
					}
					return nil
				},
			}
			backend, err := owner.observeReady(t.Context(), launch, observer, state)
			if test.name == "ready" {
				if err != nil || backend.Endpoint != endpoint || backend.Incarnation != state.Incarnation {
					t.Fatalf("verified binding = %+v, %v", backend, err)
				}
			} else if err == nil || backend != (lightingruntime.Backend{}) {
				t.Fatalf("unqualified binding = %+v, %v", backend, err)
			}
			if !reflect.DeepEqual(calls, test.calls) {
				t.Fatalf("evidence order = %v, want %v", calls, test.calls)
			}
		})
	}
}

func TestOpenRGBHeadlessListenerCountsRejectForeignAndWildcardAddresses(t *testing.T) {
	const header = "sl local_address rem_address st\n"
	row := func(address, state string) string { return fmt.Sprintf("0: %s 00000000:0000 %s\n", address, state) }
	table := header + row("0100007F:1A56", "0A") + row("00000000:1A56", "0A") + row("0100007F:1A56", "01") + row("00000000:1A57", "0A")
	exact, other, err := openRGBListenerCounts(strings.NewReader(table), 6742, false)
	if err != nil || exact != 1 || other != 1 {
		t.Fatalf("IPv4 listener evidence = %d exact, %d other, %v", exact, other, err)
	}
	exact, other, err = openRGBListenerCounts(strings.NewReader(header+row("00000000000000000000000001000000:1A56", "0A")), 6742, true)
	if err != nil || exact != 0 || other != 1 {
		t.Fatalf("IPv6 listener evidence = %d exact, %d other, %v", exact, other, err)
	}
	if _, _, err := openRGBListenerCounts(strings.NewReader("incomplete"), 6742, false); err == nil {
		t.Fatal("unreadable listener table counted as absent")
	}
}
