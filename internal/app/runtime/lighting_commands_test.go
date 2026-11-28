package runtimeapp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/vrooli/api-core/storage/storagetest"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/lightingruntime"
	"github.com/vrooli/vrooli/internal/resources"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/setup"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLightingSelectionRequiresExactBoundedInstallation(t *testing.T) {
	valid := func() *setupv1.LightingSelection {
		return &setupv1.LightingSelection{Source: setupv1.InstallationSource_INSTALLATION_SOURCE_MANAGED, Version: "1.1", Sha256: strings.Repeat("a", 64), DevicePolicyId: "policy"}
	}
	for _, tc := range []struct {
		name   string
		change func(*setupv1.LightingSelection)
	}{
		{"unspecified", func(s *setupv1.LightingSelection) { s.Source = 0 }}, {"unknown", func(s *setupv1.LightingSelection) { s.Source = 99 }}, {"digest", func(s *setupv1.LightingSelection) { s.Sha256 = "latest" }}, {"version", func(s *setupv1.LightingSelection) { s.Version = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := valid()
			tc.change(s)
			if validateLightingSelection(s, true) == nil {
				t.Fatal("invalid selection admitted")
			}
		})
	}
	s := valid()
	s.DevicePolicyId = ""
	if err := validateLightingSelection(s, true); err != nil {
		t.Fatal(err)
	}
	if validateLightingSelection(s, false) == nil {
		t.Fatal("enable accepted missing machine policy")
	}
	s = valid()
	s.Source = setupv1.InstallationSource_INSTALLATION_SOURCE_EXISTING
	s.Sha256 = ""
	if err := validateLightingSelection(s, true); err != nil {
		t.Fatal(err)
	}
	if validateLightingSelection(s, false) == nil {
		t.Fatal("enable admitted unresolved existing executable digest")
	}
}

func existingInstallationSetupFixture(t *testing.T, data []byte, home string) (LightingActions, *lightingResourceOwnerFake) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("existing source compatibility is explicitly qualified for linux-amd64")
	}
	storagetest.IsolateStorage(t)
	if home == "" {
		home = t.TempDir()
	}
	if err := os.Chmod(home, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	bin := filepath.Join(t.TempDir(), "openrgb")
	if err := os.WriteFile(bin, data, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	owner := &lightingResourceOwnerFake{}
	control, err := NewLightingControl(home, lightingruntime.Owners{Resource: owner}, func(context.Context) (resources.Status, error) {
		t.Fatal("disabled setup sampled live resource")
		return resources.Status{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	actions := LightingActions{Control: control, Resources: resources.NewController(root, home)}
	return actions, owner
}

func TestExistingInstallationSetupRejectsVersionOnlyProgramBeforeExecution(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "unexpected-vendor-execution")
	t.Setenv("VROOLI_TEST_ADOPTION_EXECUTION_MARKER", marker)
	actions, owner := existingInstallationSetupFixture(t, []byte("#!/bin/sh\nprintf executed > \"$VROOLI_TEST_ADOPTION_EXECUTION_MARKER\"\nprintf 'OpenRGB 1.1\\n'\n"), "")
	result, err := actions.Setup(t.Context(), &setupv1.LightingSelection{Source: setupv1.InstallationSource_INSTALLATION_SOURCE_EXISTING, Version: "1.1"})
	if err == nil || result != nil || !strings.Contains(err.Error(), "not the exact qualified native release") {
		t.Fatalf("version-only program qualified as an installation: result=%+v err=%v", result, err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("source inspection executed an unqualified vendor program: %v", err)
	}
	intent, err := actions.Control.controller.Current()
	if err != nil || intent.Phase != lightingruntime.Disabled || intent.Preparation != nil || intent.Selection != nil || len(owner.verified) != 0 || owner.ensured != 0 {
		t.Fatalf("failed source qualification changed intent or invoked native owner: intent=%+v owner=%+v err=%v", intent, owner, err)
	}
}

func TestExistingInstallationSetupSeparatesMetadataFromMissingPolicy(t *testing.T) {
	archive := os.Getenv("VROOLI_OPENRGB_STAGE_FIXTURE")
	if archive == "" {
		t.Skip("set VROOLI_OPENRGB_STAGE_FIXTURE to the approved pinned offline archive; a version-only shell program cannot qualify adoption")
	}
	data, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	// The real sealed stage rejects writable ancestors, including a shared
	// TMPDIR. Create only this fixture below the original operator home; do not
	// chmod the shared temporary root or any existing ancestor to make it pass.
	parent, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	home, err := os.MkdirTemp(parent, ".vrooli-openrgb-setup-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = filepath.WalkDir(home, func(path string, entry os.DirEntry, err error) error {
			if err == nil && entry.IsDir() {
				return os.Chmod(path, 0o700)
			}
			return err
		})
		if err := os.RemoveAll(home); err != nil {
			t.Error(err)
		}
	})
	actions, owner := existingInstallationSetupFixture(t, data, home)
	result, err := actions.Setup(context.Background(), &setupv1.LightingSelection{Source: setupv1.InstallationSource_INSTALLATION_SOURCE_EXISTING, Version: "1.1"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Installed || result.SetupVerified || result.ResourceObserved || result.LaunchBlocker == "" || result.Selection.Sha256 == "" || result.Selection.DevicePolicyId != "" {
		t.Fatalf("metadata and access conflated: %+v", result)
	}
	if len(owner.verified) != 0 || owner.ensured != 0 || result.Phase != "disabled" {
		t.Fatalf("setup invoked native owner: %+v", owner)
	}
	if _, err := actions.Enable(context.Background(), result.Selection); err == nil {
		t.Fatal("enable admitted unqualified installation")
	}
	owner.verifyErr = errors.New("selected policy is bound to different artifact bytes")
	result.Selection.DevicePolicyId = "existing-policy"
	mismatch, err := actions.Setup(context.Background(), result.Selection)
	if err != nil || mismatch == nil || !mismatch.Installed || mismatch.SetupVerified || !strings.Contains(mismatch.LaunchBlocker, "different artifact") {
		t.Fatalf("policy mismatch lost installation truth: %+v %v", mismatch, err)
	}
	if owner.ensured != 0 {
		t.Fatal("metadata-only existing setup started backend")
	}
	refreshed, err := actions.Status(context.Background())
	if err != nil || refreshed.Installed || refreshed.ResourceObserved || refreshed.SetupVerified || refreshed.LaunchBlocker != mismatch.LaunchBlocker || refreshed.Selection.Sha256 != mismatch.Selection.Sha256 {
		t.Fatalf("refresh lost setup result or claimed present bytes: %+v %v", refreshed, err)
	}

}

func TestDisabledStatusRetainsStaticPreparationWithoutClaimingPresentBytes(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("fixture policy is linux-amd64")
	}
	storagetest.IsolateStorage(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	resource := resources.NewController(root, home)
	manifest, err := resource.ResourceManifest(openRGBResource)
	if err != nil {
		t.Fatal(err)
	}
	asset, err := os.ReadFile(filepath.Join(root, "resources/openrgb/config/detectors-release_1.1.json"))
	if err != nil {
		t.Fatal(err)
	}
	policy := hostreqkit.OpenRGBLinuxPolicy{MachineID: strings.Repeat("a", 32), ArtifactSHA256: manifest.ManagedService.Artifact.SHA256ByPlatform["linux-amd64"], DetectorSHA256: fmt.Sprintf("%x", sha256.Sum256(asset)), RuleSHA256: strings.Repeat("d", 64), GroupID: 1500, Devices: []hostreqkit.OpenRGBLinuxDevice{{Detector: "ASUS Aura Motherboard", Kind: "hidraw", Topology: "pci0000:00/0000:03:00.0/usb1/1-8/1-8:1.0", HardwareID: "0b05:19af:00"}}}
	policyID, err := resource.StoreOpenRGBDevicePolicy(policy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resource.SetManagedServiceSource(context.Background(), openRGBResource, "managed"); err != nil {
		t.Fatal(err)
	}
	control, err := NewLightingControl(home, lightingruntime.Owners{}, func(context.Context) (resources.Status, error) {
		t.Fatal("disabled status invoked live owner")
		return resources.Status{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	proof := &lightingruntime.Preparation{Source: lightingruntime.PreparationManaged, Selection: lightingruntime.Selection{Version: manifest.ManagedService.Artifact.Version, SHA256: policy.ArtifactSHA256, DevicePolicyID: policyID}, Qualified: true, LaunchBlocker: "native launch access remains unobserved"}
	if err := control.controller.PrepareDisabledResult(context.Background(), func() (*lightingruntime.Preparation, error) { return proof, nil }); err != nil {
		t.Fatal(err)
	}
	actions := LightingActions{Control: control, Resources: resource}
	status, err := actions.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !status.SetupVerified || status.Installed || status.ResourceObserved || status.Selection.DevicePolicyId != policyID {
		t.Fatalf("lost preparation or inferred current bytes: %+v", status)
	}
	stateRoot, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		t.Fatal(err)
	}
	reference := filepath.Join(stateRoot, "host-lighting/device-access/applied.json")
	if err := os.MkdirAll(filepath.Dir(reference), 0700); err != nil {
		t.Fatal(err)
	}
	planID := strings.Repeat("f", 64)
	referenceData, err := json.Marshal(map[string]any{"PlanID": planID, "PolicyID": policyID, "Pending": true})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reference, referenceData, 0600); err != nil {
		t.Fatal(err)
	}
	pending, err := actions.Status(context.Background())
	if err != nil || pending.SetupVerified || !pending.Degraded || pending.DeviceAccessPlanId != planID || !pending.DeviceAccessPending || pending.ResourceObserved || !strings.Contains(pending.LaunchBlocker, "incomplete") {
		t.Fatalf("pending owner transaction retained static qualification: %+v %v", pending, err)
	}
	if err := os.Remove(reference); err != nil {
		t.Fatal(err)
	}
	policy.RuleSHA256 = strings.Repeat("e", 64)
	if _, err := resource.StoreOpenRGBDevicePolicy(policy); err != nil {
		t.Fatal(err)
	}
	stale, err := actions.Status(context.Background())
	if err != nil || stale.SetupVerified || !strings.Contains(stale.LaunchBlocker, "policy changed") {
		t.Fatalf("stale policy retained qualification: %+v %v", stale, err)
	}
}
