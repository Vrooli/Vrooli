//go:build linux

package platform

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOwnedOperationAbsencePreservesUnknownAndNativeNamespace(t *testing.T) {
	intent, err := NewOperationIntent("core-activation", "absence-probe", operationTestSpec())
	if err != nil {
		t.Fatal(err)
	}
	unit, _ := intent.UnitName()
	oldQuery, oldSlice, oldRoot := queryOwnedUnit, queryOperationsSlice, operationCgroupRoot
	t.Cleanup(func() { queryOwnedUnit, queryOperationsSlice, operationCgroupRoot = oldQuery, oldSlice, oldRoot })
	operationCgroupRoot = t.TempDir()
	base := "/user.slice/" + OperationsSlice
	queryOperationsSlice = func(context.Context) (string, error) { return base, nil }
	state := ownedUnitState{LoadState: "not-found"}
	var queryErr error
	queryOwnedUnit = func(_ context.Context, got string) (ownedUnitState, error) {
		if got != unit {
			t.Fatal("wrong native operation queried")
		}
		return state, queryErr
	}
	ctx := context.Background()
	if absent, err := OwnedOperationIntentAbsent(ctx, intent); err != nil || !absent {
		t.Fatalf("absent operation not observed: %t %v", absent, err)
	}
	state.LoadState = "loaded"
	if absent, err := OwnedOperationIntentAbsent(ctx, intent); err != nil || absent {
		t.Fatalf("loaded unit lost: %t %v", absent, err)
	}
	state.LoadState = "not-found"
	cgroup := filepath.Join(operationCgroupRoot, base, unit)
	if err := os.MkdirAll(cgroup, 0o700); err != nil {
		t.Fatal(err)
	}
	if absent, err := OwnedOperationIntentAbsent(ctx, intent); err != nil || absent {
		t.Fatalf("remaining cgroup ignored: %t %v", absent, err)
	}
	if err := os.Remove(cgroup); err != nil {
		t.Fatal(err)
	}
	queryErr = errors.New("manager unavailable")
	if absent, err := OwnedOperationIntentAbsent(ctx, intent); !errors.Is(err, queryErr) || absent {
		t.Fatalf("outage became absence: %t %v", absent, err)
	}
	queryErr = nil
	base = "/user.slice/" + ServicesSlice
	if absent, err := OwnedOperationIntentAbsent(ctx, intent); !errors.Is(err, ErrOperationIdentity) || absent {
		t.Fatalf("foreign native namespace accepted: %t %v", absent, err)
	}
	intent.Kind = ownedServiceKind
	if absent, err := OwnedOperationIntentAbsent(ctx, intent); !errors.Is(err, ErrOperationIdentity) || absent {
		t.Fatalf("service accepted as bounded operation: %t %v", absent, err)
	}
}

func TestOwnedOperationReleaseRequiresBootAndNativeAbsence(t *testing.T) {
	intent, err := NewOperationIntent("core-activation", "boot-probe", operationTestSpec())
	if err != nil {
		t.Fatal(err)
	}
	unit, _ := intent.UnitName()
	stamp, alive, err := linuxProcessStartTime(os.Getpid())
	if err != nil || !alive {
		t.Fatal("fixture process unavailable")
	}
	ref := OperationRef{Intent: intent, Unit: unit, Scope: ScopeRef{Kind: ScopeKindCgroup, Path: "/user.slice/" + OperationsSlice + "/" + unit}, RootPID: os.Getpid(), RootStartTime: stamp, NativeInvocationID: "0123456789abcdef0123456789abcdef"}
	oldQuery, oldRoot, oldBoot := queryOwnedUnit, operationCgroupRoot, serviceBootIDPath
	t.Cleanup(func() { queryOwnedUnit, operationCgroupRoot, serviceBootIDPath = oldQuery, oldRoot, oldBoot })
	operationCgroupRoot = t.TempDir()
	serviceBootIDPath = filepath.Join(t.TempDir(), "boot")
	const boot = "11111111-2222-3333-4444-555555555555"
	const old = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	if err := os.WriteFile(serviceBootIDPath, []byte(boot), 0o600); err != nil {
		t.Fatal(err)
	}
	state := ownedUnitState{LoadState: "not-found"}
	queryOwnedUnit = func(context.Context, string) (ownedUnitState, error) { return state, nil }
	ctx := context.Background()
	if released, err := OwnedOperationReleasedForIncarnation(ctx, ref, boot+":"+stamp); err != nil || released {
		t.Fatalf("same-boot live PID excluded: %t %v", released, err)
	}
	if released, err := OwnedOperationReleasedForIncarnation(ctx, ref, old+":"+stamp); err != nil || !released {
		t.Fatalf("old boot confused with PID reuse: %t %v", released, err)
	}
	state = ownedUnitState{LoadState: "loaded", InvocationID: ref.NativeInvocationID, ControlGroup: ref.Scope.Path}
	if released, err := OwnedOperationReleasedForIncarnation(ctx, ref, old+":"+stamp); err != nil || released {
		t.Fatalf("old boot hid loaded native unit: %t %v", released, err)
	}
	state.InvocationID = "fedcba9876543210fedcba9876543210"
	if released, err := OwnedOperationReleasedForIncarnation(ctx, ref, old+":"+stamp); !errors.Is(err, ErrOperationIdentity) || released {
		t.Fatalf("replacement invocation ignored: %t %v", released, err)
	}
}
