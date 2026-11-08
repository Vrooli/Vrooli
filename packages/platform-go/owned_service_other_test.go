//go:build !linux

package platform

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestOwnedServiceUnsupportedBackendNeverFallsBackToDirectLaunch(t *testing.T) {
	dir := t.TempDir()
	spec := OwnedServiceSpec{Path: filepath.Join(dir, "service"), Dir: dir, Env: []string{}, Limits: ServiceLimits{MemoryMaxBytes: 256 << 20, TasksMax: 32, CPUWeight: 100}}
	intent, err := NewServiceIntent("resource-update", "unsupported-native", spec)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := StartOwnedService(ctx, intent, spec); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("unsupported service start fell back: %v", err)
	}
	if _, err := LookupOwnedService(ctx, intent); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("unsupported native lookup inferred absence: %v", err)
	}
	unit := operationUnitName(OperationIntent(intent))
	ref := ServiceRef{Intent: OperationIntent(intent), Unit: unit, Scope: ScopeRef{Kind: ScopeKindCgroup, Path: "/user.slice/" + ServicesSlice + "/" + unit}, NativeInvocationID: "0123456789abcdef0123456789abcdef"}
	if _, err := InspectOwnedService(ctx, ref); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("unsupported service inspection inferred health: %v", err)
	}
	if _, err := StopOwnedService(ctx, ref); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("unsupported service stop inferred quiescence: %v", err)
	}
	if err := ReleaseOwnedService(ctx, ref); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("unsupported service release lost ownership: %v", err)
	}
}
