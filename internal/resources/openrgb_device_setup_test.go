package resources

import (
	"context"
	"errors"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenRGBDeviceSetupChildPreservesPartialFailure(t *testing.T) {
	id := strings.Repeat("a", 64)
	childErr := errors.New("privileged transaction interrupted")
	result, err := parseOpenRGBDeviceSetupChild(id, []byte(`{"state":"pending","planId":"`+id+`","message":"retry or rollback"}`), false, childErr)
	if !errors.Is(err, childErr) || !result.Pending || result.PlanID != id || result.Applied {
		t.Fatalf("partial receipt lost: %+v %v", result, err)
	}
	for _, data := range []string{`{"state":"ready","planId":"` + id + `"}`, `{"state":"applied","planId":"other"}`, `{"state":"pending","planId":"` + id + `","planId":"` + id + `"}`} {
		if _, err := parseOpenRGBDeviceSetupChild(id, []byte(data), false, nil); err == nil {
			t.Fatalf("accepted uncertain receipt %s", data)
		}
	}
	if _, err := parseOpenRGBDeviceSetupChild(id, []byte(`{}`), true, nil); err == nil {
		t.Fatal("accepted overflow")
	}
	result, err = parseOpenRGBDeviceSetupChild(id, []byte(`{"state":"rolled_back","planId":"`+id+`"}`), false, nil)
	if err != nil || !result.RolledBack || result.Applied || result.Pending {
		t.Fatalf("rollback receipt: %+v %v", result, err)
	}
}

func TestOpenRGBDeviceSetupStatusIsOneMetadataSnapshot(t *testing.T) {
	c := &Controller{Home: t.TempDir()}
	ctx := context.Background()
	current, err := c.CurrentOpenRGBDeviceSetupStatus(ctx)
	if err != nil || current.PlanID != "" || current.Pending {
		t.Fatalf("absent ref: %+v %v", current, err)
	}
	path, err := c.openRGBDeviceSetupPath("applied.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	id, policy := strings.Repeat("a", 64), strings.Repeat("b", 64)
	data := `{"PlanID":"` + id + `","PolicyID":"` + policy + `","Pending":true}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	current, err = c.CurrentOpenRGBDeviceSetupStatus(ctx)
	if err != nil || current.PlanID != id || current.PolicyID != policy || !current.Pending {
		t.Fatalf("pending snapshot: %+v %v", current, err)
	}
	if err := os.WriteFile(path, []byte(`{"PlanID":"not-a-digest","Pending":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := c.CurrentOpenRGBDeviceSetupStatus(ctx); err == nil {
		t.Fatal("accepted malformed authority")
	}
}

func TestOpenRGBDeviceSetupKeepsRecoveryUntilPolicyPublication(t *testing.T) {
	c := &Controller{Home: t.TempDir()}
	policy := hostreqkit.OpenRGBLinuxPolicy{MachineID: strings.Repeat("a", 32), ArtifactSHA256: strings.Repeat("b", 64), DetectorSHA256: strings.Repeat("c", 64), RuleSHA256: strings.Repeat("d", 64), OwnerUID: 1000, Devices: []hostreqkit.OpenRGBLinuxDevice{{Detector: "ASUS Aura Motherboard", Kind: "hidraw", Topology: "pci0000:00/usb1/1-7/1-7:1.0", HardwareID: "0b05:19af:00"}}}
	id := strings.Repeat("e", 64)
	stored := openRGBDeviceSetupStoredPlan{Plan: hostreqkit.OpenRGBPermissionPlan{Policy: policy}}
	publicationFailure := errors.New("policy fsync failed")
	execute := func() (hostreqkit.OpenRGBPermissionResult, error) {
		snapshot, err := c.CurrentOpenRGBDeviceSetupStatus(t.Context())
		if err != nil || !snapshot.Pending || snapshot.PlanID != id {
			t.Fatalf("effect preceded durable recovery intent: %+v %v", snapshot, err)
		}
		return hostreqkit.OpenRGBPermissionResult{PlanID: id, Applied: true, ActivePlanID: id, ActivePolicy: &policy}, nil
	}
	result, err := c.applyOpenRGBDeviceSetupTransaction(t.Context(), id, stored, execute, func(*hostreqkit.OpenRGBLinuxPolicy) (string, error) { return "", publicationFailure })
	if !errors.Is(err, publicationFailure) || !result.Pending {
		t.Fatalf("uncertain publication: %+v %v", result, err)
	}
	snapshot, err := c.CurrentOpenRGBDeviceSetupStatus(t.Context())
	if err != nil || !snapshot.Pending || snapshot.PlanID != id {
		t.Fatalf("pending recovery lost: %+v %v", snapshot, err)
	}
	policyID, _ := policy.ID()
	result, err = c.applyOpenRGBDeviceSetupTransaction(t.Context(), id, stored, execute, func(*hostreqkit.OpenRGBLinuxPolicy) (string, error) { return policyID, nil })
	if err != nil || result.Pending || !result.Applied {
		t.Fatalf("retry: %+v %v", result, err)
	}
	snapshot, err = c.CurrentOpenRGBDeviceSetupStatus(t.Context())
	if err != nil || snapshot.Pending || snapshot.PlanID != id || snapshot.PolicyID != policyID {
		t.Fatalf("retry publication: %+v %v", snapshot, err)
	}
	result, err = c.applyOpenRGBDeviceSetupTransaction(t.Context(), id, stored, func() (hostreqkit.OpenRGBPermissionResult, error) {
		return hostreqkit.OpenRGBPermissionResult{PlanID: id, Applied: true, ActivePlanID: id}, nil
	}, func(*hostreqkit.OpenRGBLinuxPolicy) (string, error) {
		t.Fatal("incomplete active enrollment reached policy publication")
		return "", nil
	})
	if err == nil || !result.Pending {
		t.Fatalf("inconsistent completion cleared recovery: %+v %v", result, err)
	}
	otherID := strings.Repeat("1", 64)
	if _, err := c.applyOpenRGBDeviceSetupTransaction(t.Context(), otherID, stored, func() (hostreqkit.OpenRGBPermissionResult, error) {
		t.Fatal("new plan displaced unresolved recovery")
		return hostreqkit.OpenRGBPermissionResult{}, nil
	}, func(*hostreqkit.OpenRGBLinuxPolicy) (string, error) {
		t.Fatal("new plan published over pending transaction")
		return "", nil
	}); err == nil {
		t.Fatal("unresolved recovery reference overwritten")
	}
	snapshot, err = c.CurrentOpenRGBDeviceSetupStatus(t.Context())
	if err != nil || snapshot.PlanID != id || !snapshot.Pending {
		t.Fatalf("original pending authority lost: %+v %v", snapshot, err)
	}
	previousID := strings.Repeat("f", 64)
	result, err = c.applyOpenRGBDeviceSetupTransaction(t.Context(), id, stored, func() (hostreqkit.OpenRGBPermissionResult, error) {
		return hostreqkit.OpenRGBPermissionResult{PlanID: id, RolledBack: true, ActivePlanID: previousID, ActivePolicy: &policy}, nil
	}, func(*hostreqkit.OpenRGBLinuxPolicy) (string, error) { return policyID, nil })
	snapshot, snapshotErr := c.CurrentOpenRGBDeviceSetupStatus(t.Context())
	if err != nil || snapshotErr != nil || !result.RolledBack || snapshot.PlanID != previousID || snapshot.Pending {
		t.Fatalf("prior enrollment not restored: %+v %+v %v %v", result, snapshot, err, snapshotErr)
	}
}

func TestOpenRGBDeviceSetupPublishesQualifiedPriorPolicyWithoutCurrentManifest(t *testing.T) {
	catalog, err := os.ReadFile(filepath.Join(projectRoot(t), "resources", openRGBResourceName, "config", "detectors-release_1.1.json"))
	if err != nil {
		t.Fatal(err)
	}
	policy := hostreqkit.OpenRGBLinuxPolicy{MachineID: strings.Repeat("a", 32), ArtifactSHA256: openRGBAppImageSHA256, DetectorSHA256: updateTestDigest(string(catalog)), RuleSHA256: strings.Repeat("d", 64), OwnerUID: 1000, Devices: []hostreqkit.OpenRGBLinuxDevice{{Detector: "ASUS Aura Motherboard", Kind: "hidraw", Topology: "pci0000:00/usb1/1-7/1-7:1.0", HardwareID: "0b05:19af:00"}}}
	c := &Controller{Root: t.TempDir(), Home: t.TempDir()}
	if _, err := c.StoreOpenRGBDevicePolicy(policy); err == nil {
		t.Fatal("normal policy store ignored missing current manifest")
	}
	id, err := c.storeOpenRGBDevicePolicyRecord(policy)
	if err != nil {
		t.Fatalf("qualified returned prior policy required mutable manifest: %v", err)
	}
	record, err := c.readOpenRGBDevicePolicyRecord()
	if err != nil || record == nil {
		t.Fatalf("prior enrollment lost: %+v %v", record, err)
	}
	actual, err := record.Policy.ID()
	if err != nil || actual != id {
		t.Fatalf("published prior policy differs: %s %s %v", id, actual, err)
	}
}
