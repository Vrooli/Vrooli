package setup

import (
	"connectrpc.com/connect"
	"context"
	"errors"
	"github.com/vrooli/api-core/identity"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/setup"
	"testing"
	"time"
)

type operationsFake struct {
	calls       int
	applyResult *setupv1.LightingDeviceAccessResult
	applyErr    error
}

func (f *operationsFake) Status(context.Context) (*setupv1.LightingRuntimeStatus, error) {
	f.calls++
	return &setupv1.LightingRuntimeStatus{Phase: "disabled", State: "disabled"}, nil
}
func (f *operationsFake) Setup(context.Context, *setupv1.LightingSelection) (*setupv1.LightingRuntimeStatus, error) {
	f.calls++
	return &setupv1.LightingRuntimeStatus{}, nil
}
func (f *operationsFake) Enable(context.Context, *setupv1.LightingSelection) (*setupv1.LightingRuntimeStatus, error) {
	f.calls++
	return &setupv1.LightingRuntimeStatus{}, nil
}
func (f *operationsFake) Disable(context.Context) (*setupv1.LightingRuntimeStatus, error) {
	f.calls++
	return &setupv1.LightingRuntimeStatus{}, nil
}
func TestSetupRequiresVerifiedUnexpiredScenarioCapability(t *testing.T) {
	for _, tc := range []struct {
		name              string
		verified, expired bool
		scopes            []string
		allow             bool
	}{
		{"unverified", false, false, []string{"lighting-control:write"}, false},
		{"other scenario", true, false, []string{"other-scenario:write"}, false},
		{"read only", true, false, []string{"lighting-control:read"}, false},
		{"unscoped project agent", true, false, nil, false},
		{"expired", true, true, []string{"lighting-control:write"}, false},
		{"scoped owner authority", true, false, []string{"lighting-control:write"}, true},
		{"canonical wildcard", true, false, []string{"lighting-control:*"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expiry := time.Now().Add(time.Hour)
			if tc.expired {
				expiry = time.Now().Add(-time.Hour)
			}
			p := identity.Principal{Kind: identity.ActorAgent, Subject: "run", Owner: "operator", Verified: tc.verified, Scopes: tc.scopes, ExpiresAt: expiry}
			f := &operationsFake{}
			h := &Handler{Operations: f}
			_, err := h.SetupLighting(identity.WithPrincipal(context.Background(), p), connect.NewRequest(&setupv1.SetupLightingRequest{}))
			if (err == nil) != tc.allow || f.calls != map[bool]int{false: 0, true: 1}[tc.allow] {
				t.Fatalf("err=%v calls=%d", err, f.calls)
			}
		})
	}
}

func TestAllSetupEntryPointsRejectMissingIdentityBeforeHostInvocation(t *testing.T) {
	fake := &operationsFake{}
	h := &Handler{Operations: fake}
	ctx := context.Background()
	_, a := h.GetSetupStatus(ctx, connect.NewRequest(&setupv1.GetSetupStatusRequest{}))
	_, b := h.SetupLighting(ctx, connect.NewRequest(&setupv1.SetupLightingRequest{}))
	_, c := h.EnableLighting(ctx, connect.NewRequest(&setupv1.EnableLightingRequest{}))
	_, d := h.DisableLighting(ctx, connect.NewRequest(&setupv1.DisableLightingRequest{}))
	_, e := h.GetSetupDevices(ctx, connect.NewRequest(&setupv1.GetSetupDevicesRequest{}))
	_, f := h.PlanLightingDevices(ctx, connect.NewRequest(&setupv1.PlanLightingDevicesRequest{}))
	_, g := h.ApplyLightingDevices(ctx, connect.NewRequest(&setupv1.ApplyLightingDevicesRequest{}))
	_, i := h.RollbackLightingDevices(ctx, connect.NewRequest(&setupv1.RollbackLightingDevicesRequest{}))
	if a == nil || b == nil || c == nil || d == nil || e == nil || f == nil || g == nil || i == nil || fake.calls != 0 {
		t.Fatalf("unauthenticated host invocation: %d", fake.calls)
	}
}

func (f *operationsFake) Devices(context.Context) (*setupv1.LightingDeviceDiscovery, error) {
	f.calls++
	return &setupv1.LightingDeviceDiscovery{}, nil
}
func (f *operationsFake) PlanDevices(context.Context, []string) (*setupv1.LightingDevicePlan, error) {
	f.calls++
	return &setupv1.LightingDevicePlan{}, nil
}
func (f *operationsFake) ApplyDevices(context.Context, string) (*setupv1.LightingDeviceAccessResult, error) {
	f.calls++
	if f.applyResult != nil {
		return f.applyResult, f.applyErr
	}
	return &setupv1.LightingDeviceAccessResult{}, f.applyErr
}
func (f *operationsFake) RollbackDevices(context.Context, string) (*setupv1.LightingDeviceAccessResult, error) {
	f.calls++
	return &setupv1.LightingDeviceAccessResult{}, nil
}

func TestPermissionApplyPreservesAuthenticatedOwnerAuthorityWithoutInventedGrant(t *testing.T) {
	for _, kind := range []identity.ActorKind{identity.ActorHuman, identity.ActorAgent} {
		p := identity.Principal{Kind: kind, Subject: "owner", Verified: true, Scopes: []string{"lighting-control:write"}, ExpiresAt: time.Now().Add(time.Hour)}
		f := &operationsFake{}
		h := &Handler{Operations: f}
		_, err := h.ApplyLightingDevices(identity.WithPrincipal(context.Background(), p), connect.NewRequest(&setupv1.ApplyLightingDevicesRequest{PlanId: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}))
		if err != nil || f.calls != 1 {
			t.Fatalf("ordinary authenticated owner setup invented another grant: %v calls=%d", err, f.calls)
		}
	}
}

func TestPermissionPartialFailurePreservesTypedRecoveryDetail(t *testing.T) {
	result := &setupv1.LightingDeviceAccessResult{State: "pending", PlanId: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Message: "rule applied; owner publication requires recovery"}
	f := &operationsFake{applyResult: result, applyErr: errors.New("owner publication failed")}
	h := &Handler{Operations: f}
	p := identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Verified: true, Scopes: []string{"lighting-control:write"}}
	_, err := h.ApplyLightingDevices(identity.WithPrincipal(context.Background(), p), connect.NewRequest(&setupv1.ApplyLightingDevicesRequest{PlanId: result.PlanId}))
	var failure *connect.Error
	if !errors.As(err, &failure) || len(failure.Details()) != 1 {
		t.Fatalf("partial result lost: %v", err)
	}
	detail, err := failure.Details()[0].Value()
	if err != nil {
		t.Fatal(err)
	}
	got, ok := detail.(*setupv1.LightingDeviceAccessResult)
	if !ok || got.State != "pending" || got.PlanId != result.PlanId {
		t.Fatalf("wrong recovery detail: %+v", detail)
	}
}
