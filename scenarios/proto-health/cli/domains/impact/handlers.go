package impact

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	impactv1 "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/impact"
	impactconnect "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/impact/impact_v1connect"
)

type handlers struct {
	client impactconnect.ImpactServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		client: impactconnect.NewImpactServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) getImpact(ctx cliapp.RunContext) error {
	name := ctx.Positional("scenario")
	against := strings.TrimSpace(ctx.Flag("against"))
	resp, err := h.client.GetImpact(context.Background(), connect.NewRequest(&impactv1.GetImpactRequest{
		Scenario: name,
		Against:  against,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("impact %q", name), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Report == nil {
		return fmt.Errorf("server returned no impact report")
	}
	report := resp.Msg.Report
	results := make([]string, 0, len(report.GetChanges()))
	for _, change := range report.GetChanges() {
		consumerNote := ""
		if count := len(change.GetUnreconciledConsumers()); count > 0 {
			consumerNote = fmt.Sprintf(" | unreconciled_consumers=%d", count)
		}
		results = append(results, fmt.Sprintf("%s | %s | wire=%t json=%t stability=%s%s | %s",
			change.GetKind().String(),
			impactLocation(change),
			change.GetWireBreaking(),
			change.GetJsonBreaking(),
			valueOrUnknown(change.GetStability()),
			consumerNote,
			change.GetMessage(),
		))
	}
	if len(results) == 0 {
		results = append(results, "No breaking proto changes found for this scope.")
	}
	summary := []string{
		fmt.Sprintf("Impact %s against %s (%s)", report.GetScenario(), report.GetScope(), shortSHA(report.GetBaselineSha())),
		fmt.Sprintf("wire_breaking=%d json_breaking=%d unreconciled_consumers=%d stable_unreconciled_breaking=%d",
			report.GetWireBreakingCount(),
			report.GetJsonBreakingCount(),
			report.GetUnreconciledConsumerCount(),
			report.GetStableUnreconciledBreakingCount(),
		),
	}
	if report.GetScopeKind() != "" {
		summary = append(summary, fmt.Sprintf("scope=%s", report.GetScopeKind()))
	}
	if report.GetBaselineName() != "" {
		summary = append(summary, fmt.Sprintf("baseline=%s commits_since=%d likely_stale=%t", report.GetBaselineName(), report.GetCommitsSinceBaseline(), report.GetLikelyStale()))
	}
	if report.GetFallbackReason() != "" {
		summary = append(summary, "fallback="+report.GetFallbackReason())
	}
	if len(report.GetUnreconciledConsumers()) > 0 {
		consumers := make([]string, 0, len(report.GetUnreconciledConsumers()))
		for _, consumer := range report.GetUnreconciledConsumers() {
			consumers = append(consumers, fmt.Sprintf("%s imports %s from %s", consumer.GetScenario(), consumer.GetToFile(), consumer.GetFromFile()))
		}
		results = append(results, append([]string{"Unreconciled consumers:"}, consumers...)...)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Changes",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("`impact %s --against HEAD` - compare against the current commit", name),
			fmt.Sprintf("`impact %s --against merge-base` - compare against the branch merge-base", name),
			fmt.Sprintf("`impact %s --against baseline:<name>` - compare against a named git-control-tower baseline", name),
		},
	})
}

func impactLocation(change *impactv1.ImpactChange) string {
	if strings.TrimSpace(change.GetPath()) != "" {
		return change.GetPath()
	}
	return change.GetFile()
}

func valueOrUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}

func shortSHA(value string) string {
	if len(value) <= 12 {
		return value
	}
	return value[:12]
}
