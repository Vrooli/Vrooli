package teams

import (
	"encoding/json"
	"testing"
)

func TestHeartbeatEnableStandingSupervisionPortableConfiguration(t *testing.T) {
	ctx := &fakeContext{getResponse: HeartbeatConfig{TeamID: "arbitrary-team", AgentID: "leader"}}
	err := cmdHeartbeatEnable(ctx, []string{"arbitrary-team", "leader", "--supervision", "--accounting-ref=owner:shared-diagnostic", "--healthy-sample-interval-seconds=3600", "--max-healthy-samples-per-wake=1"})
	if err != nil {
		t.Fatal(err)
	}
	var req UpdateHeartbeatRequest
	if err := json.Unmarshal(ctx.gotPayload, &req); err != nil {
		t.Fatal(err)
	}
	if ctx.gotMethod != "PUT" || req.Supervision == nil || req.Supervision.DiagnosticAllowance.AccountingRef != "owner:shared-diagnostic" || req.Supervision.MaxHealthySamplesPerWake != 1 {
		t.Fatalf("configuration not passed through: %s", ctx.gotPayload)
	}
	if req.ProfileKey != nil {
		t.Fatal("standing configuration replaced qualified resource policy")
	}
}

func TestHeartbeatEnableStandingSupervisionRequiresExplicitAccounting(t *testing.T) {
	ctx := &fakeContext{}
	if err := cmdHeartbeatEnable(ctx, []string{"team", "leader", "--supervision"}); err == nil {
		t.Fatal("missing shared diagnostic attribution accepted")
	}
	if ctx.gotMethod != "" {
		t.Fatal("invalid allowance reached API")
	}
}

func TestHeartbeatEnableSupervisorDispatchBindingIsExplicitMetadata(t *testing.T) {
	ctx := &fakeContext{getResponse: HeartbeatConfig{TeamID: "supervisors", AgentID: "leader"}}
	if err := cmdHeartbeatEnable(ctx, []string{"supervisors", "leader", "--supervision", "--accounting-ref=service:supervision", "--dispatch-effort-ref=service:supervision", "--dispatch-authorization-id=authorization-id", "--profile=qualified"}); err != nil {
		t.Fatal(err)
	}
	var req UpdateHeartbeatRequest
	if err := json.Unmarshal(ctx.gotPayload, &req); err != nil {
		t.Fatal(err)
	}
	if req.Supervision.DispatchAuthorization.EffortRef != "service:supervision" || req.Supervision.DispatchAuthorization.AuthorizationID != "authorization-id" {
		t.Fatal("missing explicit binding")
	}
	for _, args := range [][]string{{"supervisors", "leader", "--dispatch-effort-ref=service:supervision"}, {"supervisors", "leader", "--supervision", "--accounting-ref=service:supervision", "--dispatch-authorization-id=authorization-id"}} {
		invalid := &fakeContext{}
		if err := cmdHeartbeatEnable(invalid, args); err == nil || invalid.gotMethod != "" {
			t.Fatal("partial or non-supervisor binding reached API")
		}
	}
}
