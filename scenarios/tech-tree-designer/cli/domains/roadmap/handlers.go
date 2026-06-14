package roadmap

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	roadmapv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/roadmap"
	roadmapconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/roadmap/roadmap_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	client roadmapconnect.RoadmapServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: roadmapconnect.NewRoadmapServiceClient(httpClient, baseURL)}
}

func (h *handlers) listSectors(ctx cliapp.RunContext) error {
	resp, err := h.client.ListSectors(context.Background(), connect.NewRequest(&roadmapv1.ListSectorsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list roadmap sectors", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetSectors()))
	for _, sector := range resp.Msg.GetSectors() {
		results = append(results, sectorLine(sector))
	}
	if len(results) == 0 {
		results = append(results, "(no sectors)")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d sector(s).", len(resp.Msg.GetSectors()))},
		ResultsHeading: "Sectors",
		Results:        results,
	})
}

func (h *handlers) upsertSector(ctx cliapp.RunContext) error {
	resp, err := h.client.UpsertSector(context.Background(), connect.NewRequest(&roadmapv1.UpsertSectorRequest{
		Sector: &roadmapv1.Sector{
			Slug:        ctx.Positional("slug"),
			Name:        ctx.Flag("name"),
			Description: ctx.Flag("description"),
		},
	}))
	if err != nil {
		return cliapp.WrapAPIError("upsert roadmap sector", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{sectorLine(resp.Msg)},
		Changes: []string{fmt.Sprintf("Sector %s saved.", resp.Msg.GetSlug())},
	})
}

func (h *handlers) listMilestones(ctx cliapp.RunContext) error {
	resp, err := h.client.ListMilestones(context.Background(), connect.NewRequest(&roadmapv1.ListMilestonesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list roadmap milestones", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetMilestones()))
	for _, milestone := range resp.Msg.GetMilestones() {
		results = append(results, milestoneLine(milestone))
	}
	if len(results) == 0 {
		results = append(results, "(no milestones)")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d milestone(s).", len(resp.Msg.GetMilestones()))},
		ResultsHeading: "Milestones",
		Results:        results,
	})
}

func (h *handlers) upsertMilestone(ctx cliapp.RunContext) error {
	resp, err := h.client.UpsertMilestone(context.Background(), connect.NewRequest(&roadmapv1.UpsertMilestoneRequest{
		Milestone: &roadmapv1.Milestone{
			Id:                ctx.Positional("id"),
			Name:              ctx.Flag("name"),
			Description:       ctx.Flag("description"),
			RequiredScenarios: splitCSV(ctx.Flag("required")),
		},
	}))
	if err != nil {
		return cliapp.WrapAPIError("upsert roadmap milestone", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{milestoneLine(resp.Msg)},
		Changes: []string{fmt.Sprintf("Milestone %s saved.", resp.Msg.GetId())},
	})
}

func (h *handlers) progress(ctx cliapp.RunContext) error {
	resp, err := h.client.GetProgress(context.Background(), connect.NewRequest(&roadmapv1.GetProgressRequest{
		Sector: ctx.Flag("sector"),
		Tier:   ctx.Flag("tier"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("get roadmap progress", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetBuckets()))
	for _, bucket := range resp.Msg.GetBuckets() {
		results = append(results, fmt.Sprintf("%s/%s planned=%d live=%d beta=%d stable=%d",
			bucket.GetSector(), bucket.GetTier(), bucket.GetPlanned(), bucket.GetLive(), bucket.GetBeta(), bucket.GetStable()))
	}
	if len(results) == 0 {
		results = append(results, "(no progress buckets)")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d progress bucket(s).", len(resp.Msg.GetBuckets()))},
		ResultsHeading: "Progress",
		Results:        results,
	})
}

func sectorLine(sector *roadmapv1.Sector) string {
	return fmt.Sprintf("%s — %s", sector.GetSlug(), sector.GetName())
}

func milestoneLine(milestone *roadmapv1.Milestone) string {
	return fmt.Sprintf("%s — %s required=%d", milestone.GetId(), milestone.GetName(), len(milestone.GetRequiredScenarios()))
}

func splitCSV(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
