package resolver

import (
	"context"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	resolverv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/resolver"
	resolverconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/resolver/resolver_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client resolverconnect.ResolverServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return handlers{
		core:   core,
		client: resolverconnect.NewResolverServiceClient(httpClient, baseURL),
	}
}

func (h handlers) status(ctx cliapp.RunContext) error {
	resp, err := h.client.GetResolverStatus(context.Background(), connect.NewRequest(&resolverv1.GetResolverStatusRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get resolver status", err, nil)
	}
	return renderStatus(ctx, resp.Msg)
}

func (h handlers) configureAdGuard(ctx cliapp.RunContext) error {
	req := &resolverv1.ConfigureAdGuardHomeRequest{
		BaseUrl:  firstNonEmpty(ctx.Flag("base-url"), os.Getenv("ADGUARD_HOME_BASE_URL"), os.Getenv("ADGUARD_HOME_URL")),
		Username: firstNonEmpty(ctx.Flag("username"), os.Getenv("ADGUARD_HOME_USERNAME")),
		CredentialRef: firstNonEmpty(ctx.Flag("credential-ref"), os.Getenv("ADGUARD_HOME_CREDENTIAL_REF")),
		DryRun:   ctx.BoolFlag("dry-run"),
	}
	resp, err := h.client.ConfigureAdGuardHome(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("configure AdGuard Home", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{formatStatus(resp.Msg.GetStatus())}, Changes: resp.Msg.GetNextSteps()})
}

func (h handlers) upstreams(ctx cliapp.RunContext) error {
	var upstreams []string
	if v := ctx.Flag("upstream"); v != "" {
		upstreams = []string{v}
	}
	resp, err := h.client.UpdateUpstreams(context.Background(), connect.NewRequest(&resolverv1.UpdateUpstreamsRequest{Upstreams: upstreams, DryRun: ctx.BoolFlag("dry-run")}))
	if err != nil {
		return cliapp.WrapAPIError("update upstreams", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{formatStatus(resp.Msg.GetStatus())}, Changes: resp.Msg.GetChanges()})
}

func (h handlers) health(ctx cliapp.RunContext) error {
	resp, err := h.client.CheckResolverHealth(context.Background(), connect.NewRequest(&resolverv1.CheckResolverHealthRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("check resolver health", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{formatStatus(resp.Msg.GetStatus())}, ResultsHeading: "Checks", Results: resp.Msg.GetChecks()})
}

func (h handlers) rollout(ctx cliapp.RunContext) error {
	resp, err := h.client.GetAdGuardRollout(context.Background(), connect.NewRequest(&resolverv1.GetAdGuardRolloutRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get AdGuard rollout", err, nil)
	}
	rollout := resp.Msg.GetRollout()
	summary := []string{
		fmt.Sprintf("status=%s dns_bind_ip=%s", rollout.GetStatus(), rollout.GetDnsBindIp()),
		rollout.GetSummary(),
	}
	results := rolloutLines(rollout)
	hints := []string{
		"`resolver health --json` — inspect AdGuard resource health",
		"`devices refresh --json` — refresh AdGuard client evidence after router changes",
		"`snapshot run --profile home --json` — capture post-rollout network health",
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: summary, ResultsHeading: "Rollout Checks", Results: results, RetrievalHints: hints})
}

func renderStatus(ctx cliapp.RunContext, resp *resolverv1.GetResolverStatusResponse) error {
	status := resp.GetStatus()
	results := append([]string{}, status.GetWarnings()...)
	if evidence := status.GetEnforcementEvidence(); len(evidence) > 0 {
		results = append(results, "Evidence: "+strings.Join(evidence, " "))
	}
	return cliapp.RenderProtoList(ctx, resp, cliapp.ListReport{Summary: []string{formatStatus(status)}, ResultsHeading: "Status Details", Results: results, RetrievalHints: []string{"`resolver rollout` — show household rollout checklist", "`resolver configure-adguard --dry-run` — preview backend setup"}})
}

func formatStatus(s *resolverv1.ResolverStatus) string {
	if s == nil {
		return "Resolver status unavailable."
	}
	return fmt.Sprintf("backend=%s status=%s filtering=%t enforcement=%s", s.GetBackend(), s.GetStatus(), s.GetFilteringEnabled(), s.GetEnforcementStatus())
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func rolloutLines(rollout *resolverv1.AdGuardRollout) []string {
	if rollout == nil {
		return []string{"AdGuard rollout status unavailable."}
	}
	var lines []string
	for _, check := range rollout.GetChecks() {
		lines = append(lines, fmt.Sprintf("%s: %s — %s", check.GetTitle(), check.GetStatus(), check.GetEvidence()))
		for _, recommendation := range check.GetRecommendations() {
			lines = append(lines, "  next: "+recommendation)
		}
	}
	if settings := rollout.GetRouterSettings(); len(settings) > 0 {
		lines = append(lines, "Router settings to apply:")
		for _, setting := range settings {
			lines = append(lines, "  "+setting)
		}
	}
	if steps := rollout.GetNextSteps(); len(steps) > 0 {
		lines = append(lines, "Next steps:")
		for _, step := range steps {
			lines = append(lines, "  "+step)
		}
	}
	if warnings := rollout.GetWarnings(); len(warnings) > 0 {
		lines = append(lines, "Warnings:")
		for _, warning := range warnings {
			lines = append(lines, "  "+warning)
		}
	}
	return lines
}
