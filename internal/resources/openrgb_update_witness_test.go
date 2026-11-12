package resources

import (
	"context"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func TestOpenRGBUpdateQualifiesThreeRolesAgainstOnePhysicalGrant(t *testing.T) {
	x := newOpenRGBExecFixture(t)
	u := x.update
	u.Original.UpdateLaunch = &ManagedUpdateServingLaunch{Contract: u.Rollback}
	u.Target.Artifact.SHA256 = updateTestDigest("target artifact")
	u.Target.ConfigSHA256 = updateTestDigest("target settings")
	u.Rollback.ConfigSHA256 = updateTestDigest("retained original settings")
	original := hostreqkit.OpenRGBLinuxPolicy{
		MachineID: strings.Repeat("a", 32), ArtifactSHA256: u.Rollback.Artifact.SHA256,
		DetectorSHA256: updateTestDigest("original detector catalog"), RuleSHA256: updateTestDigest("same installed rule"), GroupID: 1001,
		Devices: []hostreqkit.OpenRGBLinuxDevice{{Kind: "hidraw", Detector: "Original detector", Topology: "pci0000:00/usb1/1-5/1-5:1.0", HardwareID: "0b05:19af:00"}},
	}
	target := original
	target.ArtifactSHA256, target.DetectorSHA256 = u.Target.Artifact.SHA256, updateTestDigest("different detector catalog")
	target.Devices = append([]hostreqkit.OpenRGBLinuxDevice(nil), original.Devices...)
	target.Devices[0].Detector = "Target detector"
	var loaded []string
	load := func(contract ManagedUpdateContract) (openRGBUpdateRolePolicy, error) {
		loaded = append(loaded, contract.ConfigSHA256)
		policy := original
		if contract.Artifact.SHA256 == target.ArtifactSHA256 {
			policy = target
		}
		id, err := policy.ID()
		return openRGBUpdateRolePolicy{policy: policy, binding: openRGBExecPolicy{ID: id, ArtifactSHA256: policy.ArtifactSHA256, DetectorCatalogSHA256: policy.DetectorSHA256, ConfigSHA256: contract.ConfigSHA256}}, err
	}
	roles, err := qualifyOpenRGBUpdatePolicies(u, load)
	if err != nil || len(loaded) != 3 || roles.target.binding.ID == roles.original.binding.ID || roles.target.binding.DetectorCatalogSHA256 == roles.original.binding.DetectorCatalogSHA256 || !sameOpenRGBRollbackPolicy(roles.original.binding, roles.rollback.binding, u.Rollback) {
		t.Fatalf("the exact role policies were not qualified: %+v, %v", roles, err)
	}
	if roles.rollback.binding.ConfigSHA256 == roles.original.binding.ConfigSHA256 {
		t.Fatal("the exact frozen rollback settings were replaced by the original launch settings")
	}
	target.Devices[0].Topology = "pci0000:00/usb1/1-6/1-6:1.0"
	if _, err := qualifyOpenRGBUpdatePolicies(u, load); err == nil {
		t.Fatal("update qualified a different target attachment")
	}
	target.Devices[0].Topology = original.Devices[0].Topology
	if _, err := qualifyOpenRGBUpdatePolicies(u, func(contract ManagedUpdateContract) (openRGBUpdateRolePolicy, error) {
		if contract.ConfigSHA256 == u.Rollback.ConfigSHA256 {
			return openRGBUpdateRolePolicy{}, os.ErrNotExist
		}
		return load(contract)
	}); !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), "rollback policy") {
		t.Fatalf("missing rollback evidence was not attributed: %v", err)
	}
}

func TestOpenRGBServingObservationBindsRetainedClaimContextAndListener(t *testing.T) {
	ctx := context.Background()
	x := newOpenRGBExecFixture(t)
	if _, err := x.store.Prepare(ctx, x.fence); err != nil {
		t.Fatal(err)
	}
	if err := x.run(); err != nil {
		t.Fatal(err)
	}
	claimed, err := x.store.read()
	if err != nil {
		t.Fatal(err)
	}
	contract := x.update.Rollback
	invocation, err := renderManagedUpdateInvocation(contract, x.fence.Intent.Runtime, x.update.ID, x.fence.Intent.InstanceID, "private-token")
	if err != nil {
		t.Fatal(err)
	}
	w := claimed.OpenRGBWitness
	o, err := newOpenRGBServingObservation(claimed, x.fence.Intent, contract, invocation, x.ref, w.Incarnation)
	if err != nil {
		t.Fatal(err)
	}
	contextFor := func(ref platform.ServiceRef, incarnation string) (string, error) {
		if ref != x.ref || incarnation != w.Incarnation {
			t.Fatal("observer was given a different native process")
		}
		return w.ContextSHA256, nil
	}
	listenerCalls := 0
	listener := func(endpoint netip.AddrPort, pid int) error {
		listenerCalls++
		if endpoint.Addr().String() != "127.0.0.1" || int(endpoint.Port()) != x.fence.Intent.Runtime.Port || pid != x.ref.RootPID {
			t.Fatal("observer did not require the exact native root listener")
		}
		return nil
	}
	if err := o.verify(ctx, contract.Artifact.SHA256, contextFor, listener); err != nil || listenerCalls != 1 {
		t.Fatalf("exact serving observation failed: %v", err)
	}
	if err := o.verify(ctx, updateTestDigest("other serving ELF"), contextFor, listener); err == nil || listenerCalls != 1 {
		t.Fatal("replacement image reached listener observation")
	}
	if err := o.verify(ctx, contract.Artifact.SHA256, func(platform.ServiceRef, string) (string, error) { return "different context", nil }, listener); err == nil || listenerCalls != 1 {
		t.Fatal("changed serving context reached listener observation")
	}
	if err := o.verify(ctx, contract.Artifact.SHA256, contextFor, func(netip.AddrPort, int) error { return errors.New("wildcard listener") }); err == nil {
		t.Fatal("unowned or wildcard listener was considered qualified")
	}
	foreign := x.ref
	foreign.NativeInvocationID = strings.Repeat("b", 32)
	if _, err := newOpenRGBServingObservation(claimed, x.fence.Intent, contract, invocation, foreign, w.Incarnation); err == nil {
		t.Fatal("a different invocation reused the retained witness")
	}
	revoked, err := x.store.Revoke(ctx, x.fence)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newOpenRGBServingObservation(revoked, x.fence.Intent, contract, invocation, x.ref, w.Incarnation); err == nil {
		t.Fatal("revoked witness was accepted for publication")
	}
	if _, err := verifyOpenRGBRetainedClaim(revoked, x.fence.Intent, contract, x.ref, w.Incarnation); err != nil {
		t.Fatalf("revoked witness was lost as historical rollback evidence: %v", err)
	}
}

func TestOpenRGBUnknownEnvelopeAndDirectELFRemainUnqualified(t *testing.T) {
	// An arbitrary valid-looking executable and a known digest alone do not
	// supply the qualified release envelope and serving closure.
	data := []byte("\x7fELF\x02\x01\x01\x00AI\x02\x00\x00\x00\x00\x00")
	path := filepath.Join(t.TempDir(), "apparently-native.elf")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (&Controller{}).verifyOpenRGBRoleArtifact(context.Background(), ManagedServiceArtifact{Path: path, Version: openRGBAppImageVersion, SHA256: updateTestDigest(string(data))}); !errors.Is(err, errOpenRGBReleasePayloadUnqualified) {
		t.Fatalf("unknown envelope did not fail with the explicit profile gap: %v", err)
	}
	nativeSHA, err := managedFileSHA256(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&Controller{}).verifyOpenRGBRoleArtifact(context.Background(), ManagedServiceArtifact{Path: os.Args[0], Version: openRGBAppImageVersion, SHA256: nativeSHA}); !errors.Is(err, errOpenRGBReleasePayloadUnqualified) {
		t.Fatalf("unknown native ELF profile accepted: %v", err)
	}
}

func TestOpenRGBRestoringStopUsesTheRetainedOwnerAndFailsBeforeEffects(t *testing.T) {
	x := newOpenRGBExecFixture(t)
	u := x.update
	u.Phase = UpdateRestoring
	if err := verifyOpenRGBManagedUpdateStop(context.Background(), u); err == nil {
		t.Fatal("restoration without its retained original owner was accepted")
	}
	u.Original.UpdateLaunch = &ManagedUpdateServingLaunch{Contract: u.Rollback, Runtime: x.fence.Intent.Runtime}
	if err := verifyOpenRGBManagedUpdateStop(context.Background(), u); err == nil {
		t.Fatal("unqualified retained artifact was accepted during original stop recovery")
	}
	if x.observed != 0 || x.replaced != 0 {
		t.Fatal("restoration preflight caused a launch effect")
	}
}
