//go:build linux

package platform

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOwnedServiceReleaseAcrossBootRequiresExactNativeAbsence(t *testing.T) {
	intent, err := NewServiceIntent("resource-update", "cross-boot-release", ownedServiceTestSpec())
	if err != nil {
		t.Fatal(err)
	}
	unit, _ := intent.UnitName()
	stamp, alive, err := linuxProcessStartTime(os.Getpid())
	if err != nil || !alive {
		t.Fatalf("fixture root unavailable: %v", err)
	}
	ref := ServiceRef{Intent: OperationIntent(intent), Unit: unit, Scope: ScopeRef{Kind: ScopeKindCgroup, Path: "/user.slice/" + ServicesSlice + "/" + unit}, RootPID: os.Getpid(), RootStartTime: stamp, NativeInvocationID: "0123456789abcdef0123456789abcdef"}
	oldQuery, oldRoot, oldBoot := queryOwnedUnit, operationCgroupRoot, serviceBootIDPath
	t.Cleanup(func() { queryOwnedUnit, operationCgroupRoot, serviceBootIDPath = oldQuery, oldRoot, oldBoot })
	operationCgroupRoot = t.TempDir()
	serviceBootIDPath = filepath.Join(t.TempDir(), "boot_id")
	const current = "11111111-2222-3333-4444-555555555555"
	const previous = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	if err := os.WriteFile(serviceBootIDPath, []byte(current+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	state := ownedUnitState{LoadState: "not-found"}
	var queryErr error
	queryOwnedUnit = func(context.Context, string) (ownedUnitState, error) { return state, queryErr }
	ctx := context.Background()
	// Deliberately reuse a real, live PID and the identical start ticks. Only
	// the retained different boot may exclude it as the old resource root.
	if released, err := OwnedServiceReleased(ctx, ref); err != nil || released {
		t.Fatalf("legacy observer lost conservative behavior: %t %v", released, err)
	}
	if released, err := OwnedServiceReleasedForIncarnation(ctx, ref, current+":"+stamp); err != nil || released {
		t.Fatalf("same-boot live root excluded: %t %v", released, err)
	}
	if released, err := OwnedServiceReleasedForIncarnation(ctx, ref, previous+":"+stamp); err != nil || !released {
		t.Fatalf("prior-boot root was mistaken for reused PID: %t %v", released, err)
	}
	for _, incarnation := range []string{"", "bad:" + stamp, previous + ":wrong", previous + ":" + stamp + ":extra"} {
		if released, err := OwnedServiceReleasedForIncarnation(ctx, ref, incarnation); !errors.Is(err, ErrOperationIdentity) || released {
			t.Fatalf("unbound incarnation accepted: %t %v", released, err)
		}
	}
	malformed := ref
	malformed.RootStartTime = "invalid"
	if released, err := OwnedServiceReleasedForIncarnation(ctx, malformed, previous+":invalid"); !errors.Is(err, ErrOperationIdentity) || released {
		t.Fatalf("malformed root start ticks accepted: %t %v", released, err)
	}
	state = ownedUnitState{LoadState: "loaded", InvocationID: ref.NativeInvocationID, ControlGroup: ref.Scope.Path}
	if released, err := OwnedServiceReleasedForIncarnation(ctx, ref, previous+":"+stamp); err != nil || released {
		t.Fatalf("existing native unit ignored: %t %v", released, err)
	}
	state.InvocationID = "fedcba9876543210fedcba9876543210"
	if released, err := OwnedServiceReleasedForIncarnation(ctx, ref, previous+":"+stamp); !errors.Is(err, ErrOperationIdentity) || released {
		t.Fatalf("replacement native unit ignored: %t %v", released, err)
	}
	state = ownedUnitState{LoadState: "not-found"}
	cgroup := filepath.Join(operationCgroupRoot, ref.Scope.Path)
	if err := os.MkdirAll(cgroup, 0o700); err != nil {
		t.Fatal(err)
	}
	if released, err := OwnedServiceReleasedForIncarnation(ctx, ref, previous+":"+stamp); err != nil || released {
		t.Fatalf("existing cgroup ignored: %t %v", released, err)
	}
	if err := os.RemoveAll(cgroup); err != nil {
		t.Fatal(err)
	}
	queryErr = errors.New("manager unavailable")
	if released, err := OwnedServiceReleasedForIncarnation(ctx, ref, previous+":"+stamp); !errors.Is(err, queryErr) || released {
		t.Fatalf("manager outage interpreted as absence: %t %v", released, err)
	}
	queryErr = nil
	if err := os.WriteFile(serviceBootIDPath, []byte("invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	if released, err := OwnedServiceReleasedForIncarnation(ctx, ref, previous+":"+stamp); err == nil || released {
		t.Fatalf("invalid host boot accepted: %t %v", released, err)
	}
	if err := os.Remove(serviceBootIDPath); err != nil {
		t.Fatal(err)
	}
	if released, err := OwnedServiceReleasedForIncarnation(ctx, ref, previous+":"+stamp); err == nil || released {
		t.Fatalf("missing host boot interpreted as exclusion: %t %v", released, err)
	}
}
