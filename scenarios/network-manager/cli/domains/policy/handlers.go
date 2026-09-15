package policy

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/policy"
	policyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/policy/policy_v1connect"
	"google.golang.org/protobuf/proto"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client policyconnect.PolicyServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return handlers{
		core:   core,
		client: policyconnect.NewPolicyServiceClient(httpClient, baseURL),
	}
}

func (h handlers) preview(ctx cliapp.RunContext) error {
	values := []string{}
	if v := ctx.Flag("value"); v != "" {
		values = append(values, v)
	}
	resp, err := h.client.PreviewPolicyChange(context.Background(), connect.NewRequest(&policyv1.PreviewPolicyChangeRequest{Target: ctx.Flag("target"), Action: ctx.Flag("action"), Values: values}))
	if err != nil {
		return cliapp.WrapAPIError("preview policy change", err, nil)
	}
	return renderChange(ctx, resp.Msg, resp.Msg.GetPreview())
}

func (h handlers) apply(ctx cliapp.RunContext) error {
	resp, err := h.client.ApplyPolicyChange(context.Background(), connect.NewRequest(&policyv1.ApplyPolicyChangeRequest{PreviewId: ctx.Positional("preview_id"), Approved: ctx.BoolFlag("approved")}))
	if err != nil {
		return cliapp.WrapAPIError("apply policy change", err, nil)
	}
	return renderMutation(ctx, resp.Msg, resp.Msg.GetChange())
}

func (h handlers) rollback(ctx cliapp.RunContext) error {
	resp, err := h.client.RollbackPolicyChange(context.Background(), connect.NewRequest(&policyv1.RollbackPolicyChangeRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return cliapp.WrapAPIError("rollback policy change", err, nil)
	}
	return renderMutation(ctx, resp.Msg, resp.Msg.GetChange())
}

func (h handlers) pause(ctx cliapp.RunContext) error {
	resp, err := h.client.PauseFiltering(context.Background(), connect.NewRequest(&policyv1.PauseFilteringRequest{Target: ctx.Flag("target"), Duration: ctx.Flag("duration")}))
	if err != nil {
		return cliapp.WrapAPIError("pause filtering", err, nil)
	}
	return renderMutation(ctx, resp.Msg, resp.Msg.GetChange())
}

func (h handlers) resume(ctx cliapp.RunContext) error {
	resp, err := h.client.ResumeFiltering(context.Background(), connect.NewRequest(&policyv1.ResumeFilteringRequest{Target: ctx.Flag("target")}))
	if err != nil {
		return cliapp.WrapAPIError("resume filtering", err, nil)
	}
	return renderMutation(ctx, resp.Msg, resp.Msg.GetChange())
}

func (h handlers) profiles(ctx cliapp.RunContext) error {
	resp, err := h.client.ListPolicyProfiles(context.Background(), connect.NewRequest(&policyv1.ListPolicyProfilesRequest{DeviceGroup: ctx.Flag("group")}))
	if err != nil {
		return cliapp.WrapAPIError("list policy profiles", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetProfiles()))
	for _, profile := range resp.Msg.GetProfiles() {
		results = append(results, formatProfile(profile))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("%d policy profile(s)", len(results))}, ResultsHeading: "Profiles", Results: results})
}

func (h handlers) profileSet(ctx cliapp.RunContext) error {
	resp, err := h.client.UpsertPolicyProfile(context.Background(), connect.NewRequest(&policyv1.UpsertPolicyProfileRequest{
		Profile: &policyv1.PolicyProfile{
			Id:                ctx.Flag("id"),
			Name:              ctx.Flag("name"),
			DeviceGroup:       ctx.Flag("group"),
			FilteringStrength: ctx.Flag("strength"),
			Schedule:          ctx.Flag("schedule"),
			OverrideBehavior:  ctx.Flag("override"),
			Status:            ctx.Flag("status"),
		},
	}))
	if err != nil {
		return cliapp.WrapAPIError("set policy profile", err, nil)
	}
	profile := resp.Msg.GetProfile()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{formatProfile(profile)}, Changes: profile.GetEffects()})
}

func (h handlers) schedule(ctx cliapp.RunContext) error {
	resp, err := h.client.EvaluatePolicySchedule(context.Background(), connect.NewRequest(&policyv1.EvaluatePolicyScheduleRequest{
		ProfileId: ctx.Positional("profile_id"),
		Target:    ctx.Flag("target"),
		Now:       ctx.Flag("now"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("evaluate policy schedule", err, nil)
	}
	evaluation := resp.Msg.GetEvaluation()
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{formatEvaluation(evaluation)}, ResultsHeading: "Effects", Results: evaluation.GetEffects()})
}

func (h handlers) bypassGuidance(ctx cliapp.RunContext) error {
	resp, err := h.client.DiagnoseEncryptedDnsBypass(context.Background(), connect.NewRequest(&policyv1.DiagnoseEncryptedDnsBypassRequest{
		Target:        ctx.Flag("target"),
		AdapterBacked: ctx.BoolFlag("adapter-backed"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("diagnose encrypted DNS bypass", err, nil)
	}
	return renderGuidance(ctx, resp.Msg, resp.Msg.GetReport())
}

func (h handlers) dohGuidance(ctx cliapp.RunContext) error {
	resp, err := h.client.GetEndpointDohGuidance(context.Background(), connect.NewRequest(&policyv1.GetEndpointDohGuidanceRequest{
		Platform:       ctx.Flag("platform"),
		Browser:        ctx.Flag("browser"),
		ManagementMode: ctx.Flag("management-mode"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("get endpoint DoH guidance", err, nil)
	}
	return renderGuidance(ctx, resp.Msg, resp.Msg.GetReport())
}

func renderChange(ctx cliapp.RunContext, payload proto.Message, c *policyv1.PolicyChange) error {
	return cliapp.RenderProtoList(ctx, payload, cliapp.ListReport{Summary: []string{formatChange(c)}, ResultsHeading: "Effects", Results: c.GetEffects()})
}

func renderMutation(ctx cliapp.RunContext, payload proto.Message, c *policyv1.PolicyChange) error {
	return cliapp.RenderProtoMutation(ctx, payload, cliapp.MutationReport{Result: []string{formatChange(c)}, Changes: c.GetEffects()})
}

func formatChange(c *policyv1.PolicyChange) string {
	if c == nil {
		return "Policy change unavailable."
	}
	return fmt.Sprintf("%s target=%s action=%s status=%s rollback=%t", c.GetId(), c.GetTarget(), c.GetAction(), c.GetStatus(), c.GetRollbackSupported())
}

func formatProfile(p *policyv1.PolicyProfile) string {
	if p == nil {
		return "Policy profile unavailable."
	}
	return fmt.Sprintf("%s name=%s group=%s strength=%s schedule=%s status=%s", p.GetId(), p.GetName(), p.GetDeviceGroup(), p.GetFilteringStrength(), p.GetSchedule(), p.GetStatus())
}

func formatEvaluation(e *policyv1.PolicyScheduleEvaluation) string {
	if e == nil {
		return "Policy schedule unavailable."
	}
	return fmt.Sprintf("%s target=%s active=%t status=%s next=%s", e.GetProfileName(), e.GetTarget(), e.GetActive(), e.GetStatus(), e.GetNextChangeAt())
}

func renderGuidance(ctx cliapp.RunContext, payload proto.Message, report *policyv1.PolicyGuidanceReport) error {
	results := []string{}
	for _, check := range report.GetChecks() {
		results = append(results, formatGuidanceCheck(check))
		results = append(results, check.GetRecommendations()...)
	}
	results = append(results, report.GetManualSteps()...)
	results = append(results, report.GetAdapterActions()...)
	results = append(results, report.GetGuardrails()...)
	return cliapp.RenderProtoList(ctx, payload, cliapp.ListReport{Summary: []string{formatGuidanceReport(report)}, ResultsHeading: "Guidance", Results: results})
}

func formatGuidanceReport(report *policyv1.PolicyGuidanceReport) string {
	if report == nil {
		return "Policy guidance unavailable."
	}
	return fmt.Sprintf("%s target=%s profile=%s status=%s", report.GetId(), report.GetTarget(), report.GetProfile(), report.GetStatus())
}

func formatGuidanceCheck(check *policyv1.GuidanceCheck) string {
	if check == nil {
		return "guidance check unavailable"
	}
	return fmt.Sprintf("%s: %s status=%s evidence=%s", check.GetId(), check.GetTitle(), check.GetStatus(), check.GetEvidence())
}
