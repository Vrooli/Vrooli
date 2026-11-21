//go:build linux

package resources

import (
	"bytes"
	"context"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/lightingruntime"
	"github.com/vrooli/vrooli/internal/operatorstate"
	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/setup"
)

type permissionNormalOwners struct{}

func (permissionNormalOwners) VerifyLightingSetup(context.Context, lightingruntime.Selection) error {
	return nil
}
func (permissionNormalOwners) EnsureOpenRGB(context.Context, lightingruntime.Selection) (lightingruntime.Backend, error) {
	return lightingruntime.Backend{Endpoint: netip.MustParseAddrPort("127.0.0.1:6742")}, nil
}
func (permissionNormalOwners) DisableOpenRGB(context.Context) error { return nil }
func (permissionNormalOwners) EnsureLightingReconciler(context.Context, lightingruntime.Backend) error {
	return nil
}
func (permissionNormalOwners) QuiesceLightingReconciler(context.Context) error { return nil }

// Real intent/resource fences, policy publication, pending/applied receipts and
// canonical CLI wire share one private owner root. Only the privileged device
// operation is replaced; no root/udev/device/SDK execution occurs in this test.
func TestOpenRGBPermissionNormalIntentApplyCommitAndReopen(t *testing.T) {
	storagetest.IsolateStorage(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USER", "")
	t.Setenv("SUDO_USER", "")
	t.Setenv("SUDO_UID", "")
	t.Setenv("SUDO_GID", "")
	for _, key := range []string{"VROOLI_RESOURCE_STORAGE_ROOT", "VROOLI_RESOURCE_CONFIG_ROOT", "VROOLI_RESOURCE_DATA_ROOT", "VROOLI_RESOURCE_CACHE_ROOT", "VROOLI_RESOURCE_STATE_ROOT", "VROOLI_RESOURCE_LOGS_ROOT"} {
		t.Setenv(key, "")
	}
	for key, value := range map[string]string{"XDG_CONFIG_HOME": filepath.Join(home, ".config"), "XDG_CACHE_HOME": filepath.Join(home, ".cache"), "XDG_DATA_HOME": filepath.Join(home, ".local", "share"), "XDG_STATE_HOME": filepath.Join(home, ".local", "state")} {
		t.Setenv(key, value)
	}
	ordinaryPaths, err := resourceStoragePaths(openRGBResourceName)
	if err != nil {
		t.Fatal(err)
	}
	rootEnv := map[string]string{"HOME": "/root", "XDG_CONFIG_HOME": "/root/.config", "XDG_CACHE_HOME": "/root/.cache", "XDG_DATA_HOME": "/root/.local/share", "XDG_STATE_HOME": "/root/.local/state"}
	sudoResolver, err := resourceStorageResolverForInvoker(home, true, runtimestorage.ResolverConfig{RuntimeOS: "linux", EnvGet: func(key string) string { return rootEnv[key] }, UserConfigDir: func() (string, error) { return "/root/.config", nil }, UserCacheDir: func() (string, error) { return "/root/.cache", nil }})
	if err != nil {
		t.Fatal(err)
	}
	sudoPaths, err := sudoResolver.Resolve(runtimestorage.Options{ResourceID: openRGBResourceName})
	if err != nil || sudoPaths != ordinaryPaths {
		t.Fatalf("sudo and ordinary owner roots diverged: root=%+v operator=%+v err=%v", sudoPaths, ordinaryPaths, err)
	}
	t.Setenv("VROOLI_RESOURCE_ARTIFACT_DIR", filepath.Join(home, "artifacts"))
	repo := projectRootForResourcesTest(t)
	c := NewController(repo, home)
	c.operatorState = operatorstate.New(operatorstate.Config{RepoRoot: repo, StateRoot: filepath.Join(home, "operator-state")})
	if _, err := c.operatorStateService().Apply(t.Context(), []byte(`{"resources":{"openrgb":{"enabled":false}}}`)); err != nil {
		t.Fatal(err)
	}
	owners := permissionNormalOwners{}
	control, err := lightingruntime.NewController(home, lightingruntime.Owners{Resource: owners, Scenario: owners})
	if err != nil {
		t.Fatal(err)
	}
	if err = control.Disable(t.Context()); err != nil {
		t.Fatal(err)
	}
	policyPath, err := openRGBDevicePolicyPath(home)
	if err != nil {
		t.Fatal(err)
	}
	parent := filepath.Dir(policyPath)
	info, err := os.Stat(parent)
	if err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("intent creator did not establish shared private owner: %v %v", info, err)
	}
	// Simulate exactly the previously created same-owner legacy directory.
	if err = os.Chmod(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err = c.readOpenRGBDevicePolicyRecord(); err == nil {
		t.Fatal("read-only policy observation silently accepted legacy nonprivate root")
	}
	info, _ = os.Stat(parent)
	if info.Mode().Perm() != 0o755 {
		t.Fatal("read-only observation repaired storage")
	}
	catalog, err := os.ReadFile(filepath.Join(repo, "resources", openRGBResourceName, "config", "detectors-release_1.1.json"))
	if err != nil {
		t.Fatal(err)
	}
	uid := os.Geteuid()
	if uid <= 0 {
		uid = 1000
	}
	policy := hostreqkit.OpenRGBLinuxPolicy{MachineID: strings.Repeat("a", 32), ArtifactSHA256: openRGBAppImageSHA256, DetectorSHA256: updateTestDigest(string(catalog)), RuleSHA256: strings.Repeat("d", 64), OwnerUID: uint32(uid), Devices: []hostreqkit.OpenRGBLinuxDevice{{Detector: "ASUS Aura Motherboard", Kind: "hidraw", Topology: "pci0000:00/usb1/1-7/1-7:1.0", HardwareID: "0b05:19af:00"}}}
	id := strings.Repeat("e", 64)
	stored := openRGBDeviceSetupStoredPlan{StateDir: ordinaryPaths.StateDir, Plan: hostreqkit.OpenRGBPermissionPlan{ID: id, Policy: policy}}
	kernelCalls := 0
	execute := func() (hostreqkit.OpenRGBPermissionResult, error) {
		if stored.StateDir != sudoPaths.StateDir {
			t.Fatal("privileged leaf resolved another owner namespace")
		}
		pending, err := c.CurrentOpenRGBDeviceSetupStatus(t.Context())
		if err != nil || !pending.Pending || pending.PlanID != id {
			t.Fatalf("host effect preceded durable pending reference: %+v %v", pending, err)
		}
		kernelCalls++
		return hostreqkit.OpenRGBPermissionResult{PlanID: id, Applied: true, ActivePlanID: id, ActivePolicy: &policy}, nil
	}
	var result OpenRGBDeviceSetupResult
	run := func(publish func(*hostreqkit.OpenRGBLinuxPolicy) (string, error)) error {
		return control.PrepareDisabled(t.Context(), func() error {
			return c.withOpenRGBDevicePermissionFence(t.Context(), func() error {
				var err error
				result, err = c.applyOpenRGBDeviceSetupTransaction(t.Context(), id, stored, execute, publish)
				return err
			})
		})
	}
	uncertain := errors.New("operator policy commit interrupted")
	if err = run(func(*hostreqkit.OpenRGBLinuxPolicy) (string, error) { return "", uncertain }); !errors.Is(err, uncertain) || !result.Pending {
		t.Fatalf("partial commit did not retain recovery: %+v %v", result, err)
	}
	pending, err := c.CurrentOpenRGBDeviceSetupStatus(t.Context())
	if err != nil || !pending.Pending || pending.PlanID != id {
		t.Fatalf("partial receipt lost: %+v %v", pending, err)
	}
	if err = run(func(p *hostreqkit.OpenRGBLinuxPolicy) (string, error) { return c.storeOpenRGBDevicePolicyRecord(*p) }); err != nil || !result.Applied || result.Pending {
		t.Fatalf("normal sameplan commit: %+v %v", result, err)
	}
	if kernelCalls != 2 {
		t.Fatal("modeled idempotent host owner was not retried with same plan")
	}
	reopened := NewController(repo, home)
	actual, err := reopened.readOpenRGBDevicePolicyRecord()
	if err != nil || actual == nil {
		t.Fatalf("operator cannot reopen policy: %+v %v", actual, err)
	}
	policyID, _ := policy.ID()
	actualID, _ := actual.Policy.ID()
	current, err := reopened.CurrentOpenRGBDeviceSetupStatus(t.Context())
	if err != nil || current.Pending || current.PlanID != id || current.PolicyID != policyID || actualID != policyID {
		t.Fatalf("policy/reference disagree after reopen: %+v %v", current, err)
	}
	for _, path := range []string{parent, filepath.Join(parent, "device-access")} {
		st, err := os.Stat(path)
		if err != nil || st.Mode().Perm() != 0o700 {
			t.Fatalf("shared owner directory %s: %v %v", path, st, err)
		}
	}
	for _, path := range []string{policyPath, filepath.Join(parent, "intent.json"), filepath.Join(parent, "operation.lock"), filepath.Join(parent, "device-access", "applied.json")} {
		st, err := os.Stat(path)
		if err != nil || st.Mode().Perm() != 0o600 {
			t.Fatalf("private owner file %s: %v %v", path, st, err)
		}
	}
	var stdout bytes.Buffer
	if err = cliout.WriteProtoJSON(&stdout, &setupv1.LightingDeviceAccessResult{State: "applied", PlanId: result.PlanID, DevicePolicyId: result.PolicyID, Message: result.Message}); err != nil {
		t.Fatal(err)
	}
	parsed, err := parseOpenRGBDeviceSetupChild(id, stdout.Bytes(), false, nil)
	if err != nil || !parsed.Applied || parsed.Pending || parsed.PolicyID != policyID {
		t.Fatalf("canonical CLI receipt did not reach ordinary parent: %+v %v body=%s", parsed, err, stdout.String())
	}
	restored, err := lightingruntime.NewController(home, lightingruntime.Owners{})
	if err != nil {
		t.Fatal(err)
	}
	intent, err := restored.Current()
	if err != nil || intent.Phase != lightingruntime.Disabled {
		t.Fatalf("permission commit changed Off fence: %+v %v", intent, err)
	}
}

func TestOpenRGBPermissionChildCanonicalWireAndErrorStreams(t *testing.T) {
	id, policy := strings.Repeat("a", 64), strings.Repeat("b", 64)
	childErr := errors.New("privileged command returned a diagnostic on stderr")
	var stdout bytes.Buffer
	if err := cliout.WriteProtoJSON(&stdout, &setupv1.LightingDeviceAccessResult{State: "pending", PlanId: id, DevicePolicyId: policy, Message: "retry this exact plan"}); err != nil {
		t.Fatal(err)
	}
	result, err := parseOpenRGBDeviceSetupChild(id, stdout.Bytes(), false, childErr)
	if !errors.Is(err, childErr) || !result.Pending || result.PolicyID != policy || result.PlanID != id {
		t.Fatalf("canonical partial receipt/error lost: %+v %v", result, err)
	}
	for name, data := range map[string][]byte{
		"trailing-error-stdout":   append(append([]byte(nil), stdout.Bytes()...), []byte("Error: operation interrupted\n")...),
		"logical-duplicate-alias": []byte(`{"state":"pending","plan_id":"` + id + `","planId":"` + id + `"}`),
		"unknown-field":           []byte(`{"state":"pending","plan_id":"` + id + `","unexpected":true}`),
		"wrong-plan":              []byte(`{"state":"pending","plan_id":"` + policy + `"}`),
	} {
		t.Run(name, func(t *testing.T) {
			parsed, err := parseOpenRGBDeviceSetupChild(id, data, false, childErr)
			if err == nil || !errors.Is(err, childErr) || parsed.Applied {
				t.Fatalf("uncertain child receipt acknowledged: %+v %v", parsed, err)
			}
		})
	}
}
