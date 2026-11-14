package apiauth_test

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
	"google.golang.org/protobuf/proto"
)

func TestPublicPolicyTickPreservesActualOffEvidence(t *testing.T) {
	for _, manualBase := range []bool{false, true} {
		for _, receiptPresent := range []bool{true, false} {
			name := "policy base"
			if manualBase {
				name = "manual base"
			}
			if !receiptPresent {
				name += " without previous receipt"
			}
			t.Run(name, func(t *testing.T) {
				ctx := context.Background()
				p := identity.Principal{Kind: identity.ActorHuman, Source: identity.SourceScenarioAuthenticator, Subject: "owner", Verified: true, Scopes: []string{"lighting-control:write"}, ExpiresAt: time.Now().Add(time.Hour)}
				f := newFixture(t, p, true, false)
				if _, err := f.policy.SetManualPreset(ctx, request(f, &policyv1.SetManualPresetRequest{Preset: policyv1.PresetKind_PRESET_KIND_STATIC})); err != nil {
					t.Fatal(err)
				}
				if _, err := f.db.Exec(`DELETE FROM lighting_desired_state`); err != nil {
					t.Fatal(err)
				}
				if _, err := f.db.Exec(`DELETE FROM lighting_zone_ownership`); err != nil {
					t.Fatal(err)
				}
				if manualBase {
					if _, err := f.control.SetZoneState(ctx, request(f, &controlv1.SetZoneStateRequest{Desired: zoneState(f.controllerID, f.zoneID)})); err != nil {
						t.Fatal(err)
					}
				}
				if _, err := f.control.SetManualOff(ctx, request(f, &controlv1.SetManualOffRequest{AllEnrolled: true})); err != nil {
					t.Fatal(err)
				}
				// Exercise the service's durable-state reconciliation path, as used by
				// the native binder, before the later policy tick. This is not OS restart.
				if _, err := f.control.Reconcile(ctx, request(f, &controlv1.ReconcileRequest{})); err != nil {
					t.Fatal(err)
				}
				if !receiptPresent {
					if _, err := f.db.Exec(`UPDATE lighting_desired_state SET last_apply=NULL`); err != nil {
						t.Fatal(err)
					}
				}
				before, err := f.control.GetDesiredState(ctx, request(f, &controlv1.GetDesiredStateRequest{}))
				if err != nil {
					t.Fatal(err)
				}
				if len(before.Msg.Zones) != 1 {
					t.Fatalf("desired zones %+v", before.Msg.Zones)
				}
				previous := before.Msg.Zones[0].LastApply
				if receiptPresent && (previous == nil || previous.Operation != controlv1.OperationState_OPERATION_STATE_BACKEND_ACKNOWLEDGED || previous.Desired.GetOff() == nil) {
					t.Fatalf("missing actual Off ACK: %+v", previous)
				}
				writes := f.backend.writeCalls.Load()
				for range 2 {
					evaluated, err := f.policy.EvaluatePolicy(ctx, request(f, &policyv1.EvaluatePolicyRequest{}))
					if err != nil {
						t.Fatal(err)
					}
					if len(evaluated.Msg.Results) != 1 || evaluated.Msg.Results[0].Result.Operation != controlv1.OperationState_OPERATION_STATE_DEFERRED {
						t.Fatalf("suppressed caller result: %+v", evaluated.Msg.Results)
					}
				}
				if f.backend.writeCalls.Load() != writes {
					t.Fatal("suppressed tick attempted SDK color writes")
				}
				after, err := f.control.GetDesiredState(ctx, request(f, &controlv1.GetDesiredStateRequest{}))
				if err != nil {
					t.Fatal(err)
				}
				if len(after.Msg.Zones) != 1 || !proto.Equal(previous, after.Msg.Zones[0].LastApply) {
					t.Fatalf("suppressed tick replaced previous evidence: before=%v after=%v", previous, after.Msg.Zones)
				}
				if !after.Msg.ManualOverride.Active {
					t.Fatal("policy tick cleared durable Off")
				}
				if after.Msg.Zones[0].Desired.GetStaticColor() == nil {
					t.Fatal("pending/manual desired projection was lost")
				}
				if !receiptPresent && after.Msg.Zones[0].LastApply != nil {
					t.Fatal("missing receipt promoted to an applied result")
				}
			})
		}
	}
}
