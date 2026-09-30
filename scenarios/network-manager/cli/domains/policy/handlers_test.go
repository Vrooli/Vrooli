package policy

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	clitest "github.com/vrooli/cli-core/cliapptest"

	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/policy"
	policyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/policy/policy_v1connect"
)

func TestHandlersProfileSetCallsGeneratedClient(t *testing.T) {
	// [REQ:NM-P1-001] The policy profile CLI reaches the shared Connect API contract.
	var got *policyv1.PolicyProfile
	core := newPolicyTestApp(t, &fakePolicyService{
		upsertProfile: func(_ context.Context, req *connect.Request[policyv1.UpsertPolicyProfileRequest]) (*connect.Response[policyv1.UpsertPolicyProfileResponse], error) {
			got = req.Msg.GetProfile()
			return connect.NewResponse(&policyv1.UpsertPolicyProfileResponse{Profile: &policyv1.PolicyProfile{
				Id:                "profile-kids",
				Name:              got.GetName(),
				DeviceGroup:       got.GetDeviceGroup(),
				FilteringStrength: got.GetFilteringStrength(),
				Schedule:          got.GetSchedule(),
				Status:            "enabled",
				Effects:           []string{"stored"},
			}}), nil
		},
	})
	h := newHandlers(core)
	var out bytes.Buffer

	err := h.profileSet(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Core:   core,
		JSON:   true,
		Stdout: &out,
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "id"},
			{Name: "name", Required: true},
			{Name: "group", Required: true},
			{Name: "strength"},
			{Name: "schedule"},
			{Name: "override"},
			{Name: "status"},
		}},
		Flags: map[string]string{
			"name":     "Kids",
			"group":    "kids",
			"strength": "strict",
			"schedule": "daily:20:00-07:00",
		},
	}))

	if err != nil {
		t.Fatalf("profile-set: %v", err)
	}
	if got.GetName() != "Kids" || got.GetDeviceGroup() != "kids" || got.GetSchedule() != "daily:20:00-07:00" {
		t.Fatalf("unexpected profile request: %+v", got)
	}
	if !strings.Contains(out.String(), "profile-kids") {
		t.Fatalf("expected profile in output, got %s", out.String())
	}
}

func TestHandlersScheduleCallsGeneratedClient(t *testing.T) {
	// [REQ:NM-P1-002] The schedule CLI evaluates profile windows without local business logic.
	var gotProfileID string
	var gotNow string
	core := newPolicyTestApp(t, &fakePolicyService{
		evaluateSchedule: func(_ context.Context, req *connect.Request[policyv1.EvaluatePolicyScheduleRequest]) (*connect.Response[policyv1.EvaluatePolicyScheduleResponse], error) {
			gotProfileID = req.Msg.GetProfileId()
			gotNow = req.Msg.GetNow()
			return connect.NewResponse(&policyv1.EvaluatePolicyScheduleResponse{Evaluation: &policyv1.PolicyScheduleEvaluation{
				ProfileId:   gotProfileID,
				ProfileName: "Kids",
				Target:      req.Msg.GetTarget(),
				Active:      true,
				Status:      "active",
				Effects:     []string{"manual approval still required"},
			}}), nil
		},
	})
	h := newHandlers(core)
	var out bytes.Buffer

	err := h.schedule(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Core:   core,
		JSON:   true,
		Stdout: &out,
		Schema: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{{Name: "profile_id", Required: true}},
			Flags:       []cliapp.Flag{{Name: "target"}, {Name: "now"}},
		},
		Positionals: map[string]string{"profile_id": "profile-kids"},
		Flags:       map[string]string{"target": "group:kids", "now": "2026-06-23T20:30:00Z"},
	}))

	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if gotProfileID != "profile-kids" || gotNow != "2026-06-23T20:30:00Z" {
		t.Fatalf("request = profile %q now %q", gotProfileID, gotNow)
	}
	if !strings.Contains(out.String(), "active") {
		t.Fatalf("expected evaluation in output, got %s", out.String())
	}
}

func newPolicyTestApp(t *testing.T, svc policyconnect.PolicyServiceHandler) *cliapp.ScenarioApp {
	t.Helper()
	path, handler := policyconnect.NewPolicyServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return clitest.NewTestApp(t, mux)
}

type fakePolicyService struct {
	policyconnect.UnimplementedPolicyServiceHandler
	upsertProfile    func(context.Context, *connect.Request[policyv1.UpsertPolicyProfileRequest]) (*connect.Response[policyv1.UpsertPolicyProfileResponse], error)
	evaluateSchedule func(context.Context, *connect.Request[policyv1.EvaluatePolicyScheduleRequest]) (*connect.Response[policyv1.EvaluatePolicyScheduleResponse], error)
}

func (f *fakePolicyService) UpsertPolicyProfile(ctx context.Context, req *connect.Request[policyv1.UpsertPolicyProfileRequest]) (*connect.Response[policyv1.UpsertPolicyProfileResponse], error) {
	if f.upsertProfile != nil {
		return f.upsertProfile(ctx, req)
	}
	return connect.NewResponse(&policyv1.UpsertPolicyProfileResponse{}), nil
}

func (f *fakePolicyService) EvaluatePolicySchedule(ctx context.Context, req *connect.Request[policyv1.EvaluatePolicyScheduleRequest]) (*connect.Response[policyv1.EvaluatePolicyScheduleResponse], error) {
	if f.evaluateSchedule != nil {
		return f.evaluateSchedule(ctx, req)
	}
	return connect.NewResponse(&policyv1.EvaluatePolicyScheduleResponse{}), nil
}
