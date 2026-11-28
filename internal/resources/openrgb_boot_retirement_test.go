package resources

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
)

func nativeBootFixture(t *testing.T) (*managedUpdateExecStore, managedUpdateExecFence, platform.ServiceRef) {
	t.Helper()
	capsule, material, ref, _, _ := nativeTransportFixture(t)
	at := time.Now().Add(-time.Minute).UTC()
	f := managedUpdateExecFence{SchemaVersion: 1, Resource: openRGBResourceName, OperationID: "boot-cleanup", Intent: ManagedUpdateLaunchIntent{InstanceID: capsule.Intent.AttemptID, NativeIntent: capsule.Intent, Runtime: material.Runtime, OwnershipTokenHash: material.TokenHash, ContractDigest: material.Contract.Digest}, State: managedUpdateExecFenceRevoked, Claim: &ref, ClaimedAt: &at, RevokedAt: &at}
	s := execFenceTestStore(filepath.Join(t.TempDir(), "fence.json"), f)
	if err := s.save(f); err != nil {
		t.Fatal(err)
	}
	return s, f, ref
}
func nextBootRef(ref platform.ServiceRef, boot string, pid int) platform.ServiceRef {
	n := *ref.Native
	ref.Native = &n
	n.BootID = boot
	ref.RootPID = pid
	if pid == 0 {
		n.RootIncarnation = ""
		n.RootUniqueID = ""
		ref.RootStartTime = ""
	} else {
		n.RootIncarnation = "darwin:456:789:1"
		n.RootUniqueID = "100"
		ref.RootStartTime = "789000001"
	}
	return ref
}

func TestOpenRGBNativeBootCleanupPreservesClaimAndRecoversLostRelease(t *testing.T) {
	s, f, original := nativeBootFixture(t)
	current := nextBootRef(original, "22345678-1234-1234-1234-123456789abc", 456)
	stops, releases := 0, 0
	gone := false
	o := openRGBInitialCleanupOwner{stop: func(_ context.Context, ref platform.ServiceRef) (platform.OperationStopReceipt, error) {
		saved, err := s.read()
		if err != nil {
			t.Fatal(err)
		}
		if len(saved.NativeBootRetirements) != 1 || !saved.NativeBootRetirements[0].Observed.Equal(ref) {
			t.Fatal("stop occurred before exact durable capture")
		}
		stops++
		return platform.OperationStopReceipt{Ref: platform.OperationRef(ref), NativeQuiescent: true}, nil
	}, released: func(context.Context, platform.ServiceRef) (bool, error) { return gone, nil }, release: func(_ context.Context, ref platform.ServiceRef) error {
		saved, err := s.read()
		if err != nil {
			t.Fatal(err)
		}
		if saved.NativeBootRetirements[0].QuiescentAt == nil {
			t.Fatal("release preceded quiescence receipt")
		}
		releases++
		gone = true
		return errors.New("lost release acknowledgement")
	}}
	if handled, err := retireOpenRGBNativeBootWith(context.Background(), s, f, &current, o); !handled || err == nil {
		t.Fatalf("expected retained uncertain release: %v %v", handled, err)
	}
	saved, err := s.read()
	if err != nil {
		t.Fatal(err)
	}
	if !saved.Claim.Equal(original) || saved.NativeBootRetirements[0].ReleasedAt != nil {
		t.Fatal("old claim rewritten or uncertain release invented")
	}
	if handled, err := retireOpenRGBNativeBootWith(context.Background(), s, saved, nil, o); !handled || err != nil {
		t.Fatalf("release recovery: %v %v", handled, err)
	}
	saved, err = s.read()
	if err != nil {
		t.Fatal(err)
	}
	if saved.NativeBootRetirements[0].ReleasedAt == nil || stops != 1 || releases != 1 || !saved.Claim.Equal(original) {
		t.Fatal("recovery repeated effects or lost original identity")
	}
}

func TestOpenRGBNativeBootCleanupRetainsInterruptedBootAcrossAnotherReboot(t *testing.T) {
	s, f, original := nativeBootFixture(t)
	first := nextBootRef(original, "22345678-1234-1234-1234-123456789abc", 456)
	o := openRGBInitialCleanupOwner{stop: func(context.Context, platform.ServiceRef) (platform.OperationStopReceipt, error) {
		return platform.OperationStopReceipt{}, errors.New("crash before stop acknowledgement")
	}}
	if _, err := retireOpenRGBNativeBootWith(context.Background(), s, f, &first, o); err == nil {
		t.Fatal("uncertain stop accepted")
	}
	f, err := s.read()
	if err != nil {
		t.Fatal(err)
	}
	second := nextBootRef(original, "32345678-1234-1234-1234-123456789abc", 456)
	gone := false
	stops := 0
	o = openRGBInitialCleanupOwner{stop: func(_ context.Context, r platform.ServiceRef) (platform.OperationStopReceipt, error) {
		stops++
		return platform.OperationStopReceipt{Ref: platform.OperationRef(r), NativeQuiescent: true}, nil
	}, release: func(context.Context, platform.ServiceRef) error { gone = true; return nil }, released: func(context.Context, platform.ServiceRef) (bool, error) { return gone, nil }}
	if handled, err := retireOpenRGBNativeBootWith(context.Background(), s, f, &second, o); !handled || err != nil {
		t.Fatalf("repeat boot cleanup: %v %v", handled, err)
	}
	f, err = s.read()
	if err != nil {
		t.Fatal(err)
	}
	if len(f.NativeBootRetirements) != 2 || stops != 1 || !f.Claim.Equal(original) {
		t.Fatal("repeat boot replaced history/claim or stopped stale boot")
	}
	for _, r := range f.NativeBootRetirements {
		if r.ReleasedAt == nil {
			t.Fatal("historical boot remained unreconciled")
		}
	}
}

func TestOpenRGBNativeBootCleanupRejectsForeignManagerAndSameBootRoot(t *testing.T) {
	for _, mode := range []string{"same-boot-root", "account", "system-label", "intent", "boot-case"} {
		t.Run(mode, func(t *testing.T) {
			s, f, original := nativeBootFixture(t)
			current := nextBootRef(original, "22345678-1234-1234-1234-123456789abc", 456)
			switch mode {
			case "same-boot-root":
				current.Native.BootID = original.Native.BootID
			case "account":
				current.Native.Account = "_foreign"
			case "system-label":
				current.Native.Name += "foreign"
			case "intent":
				current.Intent.AttemptID += "foreign"
			case "boot-case":
				current.Native.BootID = strings.ToUpper(original.Native.BootID)
			}
			stops := 0
			o := openRGBInitialCleanupOwner{stop: func(context.Context, platform.ServiceRef) (platform.OperationStopReceipt, error) {
				stops++
				return platform.OperationStopReceipt{}, nil
			}}
			if _, err := retireOpenRGBNativeBootWith(context.Background(), s, f, &current, o); err == nil {
				t.Fatal("foreign native cleanup accepted")
			}
			if stops != 0 {
				t.Fatal("foreign root stopped")
			}
		})
	}
}

func TestOpenRGBNativeBootCleanupRequiresExactQuiescenceReceipt(t *testing.T) {
	s, f, original := nativeBootFixture(t)
	current := nextBootRef(original, "22345678-1234-1234-1234-123456789abc", 456)
	o := openRGBInitialCleanupOwner{stop: func(context.Context, platform.ServiceRef) (platform.OperationStopReceipt, error) {
		return platform.OperationStopReceipt{Ref: platform.OperationRef(original), NativeQuiescent: true}, nil
	}}
	if _, err := retireOpenRGBNativeBootWith(context.Background(), s, f, &current, o); err == nil {
		t.Fatal("different incarnation quiescence accepted")
	}
}

func TestOpenRGBNativeBootReceiptRefusesLinuxAndUnknownVersion(t *testing.T) {
	s, f, ref := nativeBootFixture(t)
	current := nextBootRef(ref, "22345678-1234-1234-1234-123456789abc", 456)
	var err error
	f, err = s.changeNativeBootRetirement(context.Background(), f, current, "capture")
	if err != nil {
		t.Fatal(err)
	}
	f.NativeBootRetirements[0].SchemaVersion = 99
	if f.validate() == nil {
		t.Fatal("future boot receipt accepted")
	}
	f.NativeBootRetirements[0].SchemaVersion = 1
	f.Claim.Native = nil
	if f.validate() == nil {
		t.Fatal("Linux fence accepted native receipt")
	}
}

func TestOpenRGBNativeInitialBootRetirementAllowsFreshAdmissionWithoutRewritingOldRef(t *testing.T) {
	capsule, m, original, w, _ := nativeTransportFixture(t)
	a := openRGBInitialAttempt{SchemaVersion: openRGBInitialSchemaVersion, ID: capsule.Intent.AttemptID, State: openRGBInitialPrepared, Contract: m.Contract, Runtime: m.Runtime, TokenHash: m.TokenHash, NativeDevicePolicy: &m.NativePolicy, Catalog: m.Catalog, Policy: m.Policy}
	var err error
	a.Intent, err = platform.NewServiceIntent("resource-initial", a.ID, a.wrapperSpec())
	if err != nil {
		t.Fatal(err)
	}
	original.Intent = platform.OperationIntent(a.Intent)
	original.Unit, _ = a.Intent.UnitName()
	n := *original.Native
	original.Native = &n
	n.Name = "com.vrooli.owned." + strings.TrimSuffix(original.Unit, ".service")
	n.InvocationID = a.Intent.InvocationID
	original.NativeInvocationID = a.Intent.InvocationID
	original.Scope.Path = n.Domain + "/" + n.Name
	n.ContainmentID = original.Scope.Path
	a.NativeRef = &original
	if err = a.validate(); err != nil {
		t.Fatal(err)
	}
	attempts := &openRGBInitialStore{dir: a.Runtime.StateDir, write: writeResourceOwnedFileDurable}
	if err = attempts.prepare(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	a, err = attempts.submit(context.Background(), a)
	if err != nil {
		t.Fatal(err)
	}
	f := a.fence()
	now := time.Now().Add(-time.Minute).UTC()
	f.State = managedUpdateExecFenceClaimed
	f.Claim = &original
	f.ClaimedAt = &now
	w.ClaimSHA256 = f.openRGBClaimDigest(original)
	f.OpenRGBWitness = &w
	fences := execFenceTestStore(filepath.Join(t.TempDir(), "fence.json"), f)
	fences.admissionKind = managedExecAdmissionInitial
	if err = fences.save(f); err != nil {
		t.Fatal(err)
	}
	current := nextBootRef(original, "22345678-1234-1234-1234-123456789abc", 456)
	gone := false
	o := openRGBInitialCleanupOwner{revoke: fences.Revoke, lookup: func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) { return current, nil }, stop: func(_ context.Context, r platform.ServiceRef) (platform.OperationStopReceipt, error) {
		return platform.OperationStopReceipt{Ref: platform.OperationRef(r), NativeQuiescent: true}, nil
	}, released: func(context.Context, platform.ServiceRef) (bool, error) { return gone, nil }, release: func(context.Context, platform.ServiceRef) error { gone = true; return nil }}
	o.retireBoot = func(ctx context.Context, f managedUpdateExecFence, r *platform.ServiceRef) (bool, error) {
		return retireOpenRGBNativeBootWith(ctx, fences, f, r, o)
	}
	if err = retireOpenRGBInitialWith(context.Background(), attempts, a, o); err != nil {
		t.Fatal(err)
	}
	stopped, err := attempts.read(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stopped.State != openRGBInitialStopped || stopped.NativeRef == nil || !stopped.NativeRef.Equal(original) {
		t.Fatal("boot retirement rewrote original initial reference")
	}
	saved, err := fences.read()
	if err != nil {
		t.Fatal(err)
	}
	if saved.State != managedUpdateExecFenceRevoked || !saved.Claim.Equal(original) || len(saved.NativeBootRetirements) != 1 {
		t.Fatal("boot cleanup lost revoked old admission")
	}
	if _, err = fences.ClaimOpenRGB(context.Background(), a.fence(), current, w); !errors.Is(err, ErrManagedUpdateExecRevoked) {
		t.Fatalf("old admission replay accepted: %v", err)
	}
	fresh := a
	fresh.ID = "fresh-after-boot"
	fresh.NativeRef = nil
	fresh.Intent, err = platform.NewServiceIntent("resource-initial", fresh.ID, fresh.wrapperSpec())
	if err != nil {
		t.Fatal(err)
	}
	fresh.State = openRGBInitialPrepared
	if err = attempts.prepare(context.Background(), fresh); err != nil {
		t.Fatalf("new attempt after exact retirement rejected: %v", err)
	}
	if prior, err := attempts.read(a.ID); err != nil || prior.NativeRef == nil || !prior.NativeRef.Equal(original) {
		t.Fatal("re-enable replaced old attempt identity")
	}
}

func TestOpenRGBNativeBootCleanupResumesStoppedManagerWithNoCurrentRoot(t *testing.T) {
	s, f, original := nativeBootFixture(t)
	current := nextBootRef(original, "22345678-1234-1234-1234-123456789abc", 456)
	stops := 0
	releaseError := errors.New("release interrupted before effect")
	o := openRGBInitialCleanupOwner{stop: func(_ context.Context, r platform.ServiceRef) (platform.OperationStopReceipt, error) {
		stops++
		return platform.OperationStopReceipt{Ref: platform.OperationRef(r), NativeQuiescent: true}, nil
	}, released: func(context.Context, platform.ServiceRef) (bool, error) { return false, nil }, release: func(context.Context, platform.ServiceRef) error { return releaseError }}
	if _, err := retireOpenRGBNativeBootWith(context.Background(), s, f, &current, o); !errors.Is(err, releaseError) {
		t.Fatal(err)
	}
	f, err := s.read()
	if err != nil {
		t.Fatal(err)
	}
	current = nextBootRef(current, current.Native.BootID, 0)
	gone := false
	o.release = func(_ context.Context, r platform.ServiceRef) error {
		if r.RootPID != 456 {
			t.Fatal("cleanup substituted PID0 metadata for captured root")
		}
		gone = true
		return nil
	}
	o.released = func(context.Context, platform.ServiceRef) (bool, error) { return gone, nil }
	if _, err := retireOpenRGBNativeBootWith(context.Background(), s, f, &current, o); err != nil {
		t.Fatal(err)
	}
	if stops != 1 {
		t.Fatal("retained quiescence unnecessarily stopped again")
	}
}

func TestOpenRGBNativeBootReceiptFieldCannotWidenLegacyLinuxCodec(t *testing.T) {
	f, _ := execFenceTestInitial(t)
	for _, value := range []string{"null", "[]"} {
		t.Run(value, func(t *testing.T) {
			data, err := json.Marshal(f)
			if err != nil {
				t.Fatal(err)
			}
			data = append(data[:len(data)-1], []byte(`,"native_boot_retirements":`+value+`}`)...)
			s := execFenceTestStore(filepath.Join(t.TempDir(), "fence.json"), f)
			if err = os.WriteFile(s.path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err = s.read(); err == nil {
				t.Fatal("legacy Linux fence accepted native receipt marker")
			}
		})
	}
}

func TestOpenRGBNativeBootDisableRetiresOriginalAndFinishesInOnePass(t *testing.T) {
	s, f, original := nativeBootFixture(t)
	original.Intent.Owner = "resource-initial"
	original.Unit, _ = platform.ServiceIntent(original.Intent).UnitName()
	original.Native.Name = "com.vrooli.owned." + strings.TrimSuffix(original.Unit, ".service")
	original.Scope.Path = original.Native.Domain + "/" + original.Native.Name
	original.Native.ContainmentID = original.Scope.Path
	f.AdmissionKind = managedExecAdmissionInitial
	f.OperationID = f.Intent.InstanceID
	f.Intent.NativeIntent = platform.ServiceIntent(original.Intent)
	f.Claim = &original
	s.admissionKind = managedExecAdmissionInitial
	s.operation = f.OperationID
	if err := s.save(f); err != nil {
		t.Fatal(err)
	}
	current := nextBootRef(original, "22345678-1234-1234-1234-123456789abc", 456)
	u := updateTestInitial(t)
	u.Resource = openRGBResourceName
	for _, c := range []*ManagedUpdateContract{&u.Target, &u.Rollback} {
		c.Manifest.Name = openRGBResourceName
		c.Artifact.Resource = openRGBResourceName
		c.Digest = c.digest()
	}
	u.Original.InitialLaunch = &ManagedInitialServingLaunch{AttemptID: f.Intent.InstanceID, NativeRef: original}
	u.OriginalScope = nil
	u.Original.InstanceID = f.Intent.InstanceID
	u.Original.PID = original.RootPID
	u.Original.Incarnation = original.Native.RootIncarnation
	u.OriginalProof = nil
	gone := false
	stops, releases := 0, 0
	o := openRGBInitialCleanupOwner{stop: func(_ context.Context, r platform.ServiceRef) (platform.OperationStopReceipt, error) {
		stops++
		return platform.OperationStopReceipt{Ref: platform.OperationRef(r), NativeQuiescent: true}, nil
	}, release: func(context.Context, platform.ServiceRef) error { releases++; gone = true; return nil }, released: func(context.Context, platform.ServiceRef) (bool, error) { return gone, nil }}
	stopper := func(ctx context.Context, got ManagedServiceUpdate) (managedUpdateOriginalStopReceipt, error) {
		return stopManagedUpdateOriginalNativeWithBoot(ctx, got, func(ctx context.Context, state ManagedServiceState) (bool, error) {
			return retireOpenRGBNativeBootWith(ctx, s, f, &current, o)
		})
	}
	finished := false
	err := retireOpenRGBUpdateRecords(context.Background(), managedUpdateLedger{Records: []managedUpdateEntry{{State: u}}}, func(context.Context, ManagedServiceUpdate, ManagedUpdateProcess) error { return nil }, func(context.Context, ManagedServiceUpdate, ManagedUpdateProcess) error { return nil }, stopper, func(int) error { return nil }, func(_ context.Context, got ManagedServiceUpdate) error {
		next, err := transitionManagedUpdate(got, UpdateFinishDisabled, got.UpdatedAt.Add(time.Second))
		if err != nil {
			return err
		}
		if next.Phase != UpdateResourceDisabled || !next.Original.InitialLaunch.NativeRef.Equal(original) {
			t.Fatal("disable replaced original or stayed nonterminal")
		}
		finished = true
		return nil
	})
	if err != nil || !finished || stops != 1 || releases != 1 {
		t.Fatalf("disable after reboot: err=%v finished=%v stop=%d release=%d", err, finished, stops, releases)
	}
	saved, err := s.read()
	if err != nil {
		t.Fatal(err)
	}
	if !saved.Claim.Equal(original) || len(saved.NativeBootRetirements) != 1 || saved.NativeBootRetirements[0].ReleasedAt == nil {
		t.Fatal("disable lost original claim or exact newboot release receipt")
	}
}
