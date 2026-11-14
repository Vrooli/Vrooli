package autofix

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/quality-health/v1/audit"
	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/quality-health/v1/audit/audit_v1connect"
)

type handlers struct {
	client auditconnect.AuditServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: auditconnect.NewAuditServiceClient(httpClient, baseURL)}
}

func (h *handlers) preview(ctx cliapp.RunContext) error {
	return h.fix(ctx, false)
}

func (h *handlers) apply(ctx cliapp.RunContext) error {
	return h.fix(ctx, true)
}

func (h *handlers) fix(ctx cliapp.RunContext, apply bool) error {
	scenario := ctx.Positional("scenario")
	req := connect.NewRequest(&auditv1.FixConfigRequest{
		Scenario: scenario,
		RuleIds:  splitCSV(ctx.FlagValues("rule")),
	})
	var resp *connect.Response[auditv1.FixConfigResponse]
	var err error
	if apply || (ctx.FlagDeclared("apply") && ctx.BoolFlag("apply")) {
		resp, err = h.client.ApplyFixConfig(context.Background(), req)
	} else {
		resp, err = h.client.PreviewFixConfig(context.Background(), req)
	}
	if err != nil {
		return cliapp.WrapAPIError("fix config", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetCandidates()))
	for _, c := range resp.Msg.GetCandidates() {
		results = append(results, fmt.Sprintf("%s %s applied=%v", c.GetRuleId(), c.GetFilePath(), c.GetApplied()))
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("%d config fix candidate(s).", len(resp.Msg.GetCandidates()))},
		Changes: results,
		NextCommand: []string{
			fmt.Sprintf("`quality-health audit run %s --json` - refresh quality findings", scenario),
		},
	})
}

func splitCSV(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" || seen[part] {
				continue
			}
			seen[part] = true
			out = append(out, part)
		}
	}
	return out
}
