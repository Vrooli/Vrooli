package artifacts

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/artifacts"
	artifactsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/artifacts/artifacts_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client artifactsconnect.ArtifactsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: artifactsconnect.NewArtifactsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) distribute(ctx cliapp.RunContext) error {
	resp, err := h.client.DistributeArtifact(context.Background(), connect.NewRequest(&artifactsv1.DistributeArtifactRequest{
		NodeId:          ctx.Positional("node-id"),
		Name:            ctx.Flag("name"),
		SourceRef:       ctx.Flag("source"),
		DestinationPath: ctx.Flag("dest"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("distribute artifact (run `auth login` first if unauthenticated)", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no distribution response")
	}
	msg := resp.Msg

	if msg.DryRun {
		return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
			Result:      []string{fmt.Sprintf("[dry-run] would distribute %q to node %s (validated, nothing delivered).", ctx.Flag("name"), ctx.Positional("node-id"))},
			Changes:     []string{"No distribution recorded, nothing handed to device-sync-hub."},
			NextCommand: []string{"Re-run without --dry-run to distribute for real."},
		})
	}
	return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Distribution %s — %s (ref %s).", msg.DistributionId, statusLabel(msg.Status), emptyDash(msg.DeliveryRef))},
		Changes: []string{fmt.Sprintf("handed off to device-sync-hub (ref=%s)", emptyDash(msg.DeliveryRef))},
		NextCommand: []string{
			fmt.Sprintf("`artifacts get %s` — check delivery status", msg.DistributionId),
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetDistribution(context.Background(), connect.NewRequest(&artifactsv1.GetDistributionRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get distribution %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Distribution == nil {
		return fmt.Errorf("server returned no distribution")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Distribution %s — %s.", resp.Msg.Distribution.Id, statusLabel(resp.Msg.Distribution.Status))},
		ResultsHeading: "Distribution",
		Results:        []string{formatDistribution(resp.Msg.Distribution)},
	})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListDistributions(context.Background(), connect.NewRequest(&artifactsv1.ListDistributionsRequest{
		NodeId: ctx.Flag("node"),
		Limit:  int32(parseInt(ctx.Flag("limit"))),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list distributions", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no distributions response")
	}
	results := make([]string, 0, len(resp.Msg.Distributions))
	for _, d := range resp.Msg.Distributions {
		results = append(results, formatDistribution(d))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d distribution(s).", len(resp.Msg.Distributions))},
		ResultsHeading: "Distributions",
		Results:        results,
		RetrievalHints: []string{"`artifacts get <id>` — show one distribution's status"},
	})
}

// ---- formatting helpers ----

func formatDistribution(d *artifactsv1.Distribution) string {
	if d == nil {
		return "(nil)"
	}
	created := ""
	if d.CreatedAt != nil {
		created = d.CreatedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — node=%s name=%s [status=%s ref=%s dest=%s created=%s]",
		d.Id, d.NodeId, emptyDash(d.Name), statusLabel(d.Status), emptyDash(d.DeliveryRef), d.DestinationPath, created)
}

func statusLabel(s artifactsv1.DeliveryStatus) string {
	switch s {
	case artifactsv1.DeliveryStatus_DELIVERY_STATUS_PENDING:
		return "pending"
	case artifactsv1.DeliveryStatus_DELIVERY_STATUS_DELIVERED:
		return "delivered"
	case artifactsv1.DeliveryStatus_DELIVERY_STATUS_FAILED:
		return "failed"
	default:
		return "unspecified"
	}
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func parseInt(raw string) int {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	return v
}
