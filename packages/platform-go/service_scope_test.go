package platform

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func serviceScopeFixtureRef() ServiceScopeRef {
	unit := "vrooli-service-resource-fixture-service-0123456789abcdef"
	return ServiceScopeRef{Scope: ScopeRef{Kind: ScopeKindCgroup, Name: unit, Path: "/user.slice/user-1000.slice/" + ServicesSlice + "/" + unit + ".scope"}, RootPID: 123, RootStartTime: "456", NativeInvocationID: strings.Repeat("a", 32), ScopeDevice: 31, ScopeInode: 99}
}

func TestServiceScopeIdentityRejectsOtherNamespacesAndIncompleteNativeEvidence(t *testing.T) {
	good := serviceScopeFixtureRef()
	if err := good.ValidateIdentity(); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*ServiceScopeRef){
		"agent scope": func(r *ServiceScopeRef) {
			r.Scope.Name = "vrooli-agent-session"
			r.Scope.Path = "/user.slice/" + ServicesSlice + "/" + r.Scope.Name + ".scope"
		},
		"operations slice": func(r *ServiceScopeRef) {
			r.Scope.Path = strings.Replace(r.Scope.Path, ServicesSlice, OperationsSlice, 1)
		},
		"new service unit":    func(r *ServiceScopeRef) { r.Scope.Path = strings.TrimSuffix(r.Scope.Path, ".scope") + ".service" },
		"name mismatch":       func(r *ServiceScopeRef) { r.Scope.Name += "-other" },
		"path traversal":      func(r *ServiceScopeRef) { r.Scope.Path += "/../elsewhere.scope" },
		"missing inode":       func(r *ServiceScopeRef) { r.ScopeInode = 0 },
		"missing device":      func(r *ServiceScopeRef) { r.ScopeDevice = 0 },
		"missing invocation":  func(r *ServiceScopeRef) { r.NativeInvocationID = "" },
		"missing root":        func(r *ServiceScopeRef) { r.RootPID = 0 },
		"unbound incarnation": func(r *ServiceScopeRef) { r.RootStartTime = "boot:456" },
	} {
		t.Run(name, func(t *testing.T) {
			r := good
			change(&r)
			if r.ValidateIdentity() == nil {
				t.Fatal("invalid service-scope receipt accepted")
			}
			if _, err := StopServiceScope(context.Background(), r); !errors.Is(err, ErrOperationIdentity) {
				t.Fatalf("invalid receipt reached native stop: %v", err)
			}
		})
	}
}

func TestServiceScopeCancellationPrecedesNativeEffects(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := serviceScopeFixtureRef()
	if _, err := CaptureServiceScope(ctx, r.Scope, r.RootPID, r.RootStartTime); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := StopServiceScope(ctx, r); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := InspectServiceScope(ctx, r); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
