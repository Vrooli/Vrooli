// Package usage hosts the `audio-tools usage ...` subtree.
package usage

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	usagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/usage"
	usageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/usage/usage_v1connect"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := usageconnect.NewUsageServiceClient(httpClient, baseURL)
	return cliapp.SubcommandGroup{
		Name:        "usage",
		Description: "Inspect usage rows recorded by the provider chains",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "list",
				Description: "List recent usage rows",
				RunCtx: func(ctx cliapp.RunContext) error {
					resp, err := client.ListRecent(context.Background(), connect.NewRequest(&usagev1.ListRecentRequest{SinceSeconds: 86400, Limit: 50}))
					if err != nil {
						return cliapp.WrapAPIError("list usage", err, nil)
					}
					out := ctx.Stdout()
					for _, r := range resp.Msg.GetRows() {
						fmt.Fprintf(out, "%s  %-10s %-12s %s/%s  %.0fms  err=%q\n",
							r.GetEmittedAt(), r.GetCapability(), r.GetOperation(),
							r.GetProviderTier(), r.GetProviderId(), r.GetLatencyMs(), r.GetError())
					}
					if len(resp.Msg.GetRows()) == 0 {
						fmt.Fprintln(out, "(no usage rows)")
					}
					return nil
				},
			},
			{
				Name:        "summary",
				Description: "Show usage summary for the last 24h",
				RunCtx: func(ctx cliapp.RunContext) error {
					resp, err := client.GetSummary(context.Background(), connect.NewRequest(&usagev1.GetSummaryRequest{SinceSeconds: 86400}))
					if err != nil {
						return cliapp.WrapAPIError("usage summary", err, nil)
					}
					s := resp.Msg.GetSummary()
					out := ctx.Stdout()
					fmt.Fprintf(out, "Operations: %d\nCredits:    %d\nErrors:     %d\n\n",
						s.GetOperationsTotal(), s.GetCreditsTotal(), s.GetErrorCount())
					fmt.Fprintln(out, "Provider distribution:")
					for _, d := range s.GetDistribution() {
						fmt.Fprintf(out, "  %s/%-12s  count=%d credits=%d avg_ms=%.1f\n",
							d.GetProviderTier(), d.GetProviderId(), d.GetCount(), d.GetCreditsTotal(), d.GetAvgLatencyMs())
					}
					return nil
				},
			},
		},
	}
}
