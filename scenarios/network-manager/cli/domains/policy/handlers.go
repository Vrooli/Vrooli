package policy

import (
	"fmt"
	"net/http"

	"github.com/vrooli/cli-core/cliapp"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/policy"
	policyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/policy/policy_v1connect"
	"google.golang.org/protobuf/proto"
)

type handlers struct{ core *cliapp.ScenarioApp }

func (h handlers) preview(ctx cliapp.RunContext) error {
	values := []string{}
	if v := ctx.Flag("value"); v != "" {
		values = append(values, v)
	}
	resp, err := cliapp.Call[*policyv1.PreviewPolicyChangeRequest, *policyv1.PreviewPolicyChangeResponse](h.core, http.MethodPost, policyconnect.PolicyServicePreviewPolicyChangeProcedure, &policyv1.PreviewPolicyChangeRequest{Target: ctx.Flag("target"), Action: ctx.Flag("action"), Values: values})
	if err != nil {
		return cliapp.WrapAPIError("preview policy change", err, nil)
	}
	return renderChange(ctx, resp, resp.GetPreview())
}

func (h handlers) apply(ctx cliapp.RunContext) error {
	resp, err := cliapp.Call[*policyv1.ApplyPolicyChangeRequest, *policyv1.ApplyPolicyChangeResponse](h.core, http.MethodPost, policyconnect.PolicyServiceApplyPolicyChangeProcedure, &policyv1.ApplyPolicyChangeRequest{PreviewId: ctx.Positional("preview_id"), Approved: ctx.BoolFlag("approved")})
	if err != nil {
		return cliapp.WrapAPIError("apply policy change", err, nil)
	}
	return renderMutation(ctx, resp, resp.GetChange())
}

func (h handlers) rollback(ctx cliapp.RunContext) error {
	resp, err := cliapp.Call[*policyv1.RollbackPolicyChangeRequest, *policyv1.RollbackPolicyChangeResponse](h.core, http.MethodPost, policyconnect.PolicyServiceRollbackPolicyChangeProcedure, &policyv1.RollbackPolicyChangeRequest{Id: ctx.Positional("id")})
	if err != nil {
		return cliapp.WrapAPIError("rollback policy change", err, nil)
	}
	return renderMutation(ctx, resp, resp.GetChange())
}

func (h handlers) pause(ctx cliapp.RunContext) error {
	resp, err := cliapp.Call[*policyv1.PauseFilteringRequest, *policyv1.PauseFilteringResponse](h.core, http.MethodPost, policyconnect.PolicyServicePauseFilteringProcedure, &policyv1.PauseFilteringRequest{Target: ctx.Flag("target"), Duration: ctx.Flag("duration")})
	if err != nil {
		return cliapp.WrapAPIError("pause filtering", err, nil)
	}
	return renderMutation(ctx, resp, resp.GetChange())
}

func (h handlers) resume(ctx cliapp.RunContext) error {
	resp, err := cliapp.Call[*policyv1.ResumeFilteringRequest, *policyv1.ResumeFilteringResponse](h.core, http.MethodPost, policyconnect.PolicyServiceResumeFilteringProcedure, &policyv1.ResumeFilteringRequest{Target: ctx.Flag("target")})
	if err != nil {
		return cliapp.WrapAPIError("resume filtering", err, nil)
	}
	return renderMutation(ctx, resp, resp.GetChange())
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
