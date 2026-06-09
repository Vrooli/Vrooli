package disputes

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	findingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/findings"
	findingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/findings/findings_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client findingsconnect.FindingsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: findingsconnect.NewFindingsServiceClient(httpClient, baseURL),
	}
}

// list surfaces the disputed findings by filtering ListFindings to the DISPUTED
// lifecycle state — the review queue's read side.
func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListDisputes(context.Background(), connect.NewRequest(&findingsv1.ListDisputesRequest{
		Limit: parseInt32(ctx.Flag("limit")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list disputed findings", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no findings response")
	}
	results := make([]string, 0, len(resp.Msg.Findings))
	for _, f := range resp.Msg.Findings {
		results = append(results, formatFinding(f))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d disputed finding(s) awaiting resolution.", len(resp.Msg.Findings))},
		ResultsHeading: "Disputed findings",
		Results:        results,
		RetrievalHints: []string{
			"`disputes resolve <id> --resolution keep` — clear the dispute, return to active",
			"`disputes resolve <id> --resolution supersede --replacement <id>` — retire it",
		},
	})
}

func (h *handlers) resolve(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.ResolveDispute(context.Background(), connect.NewRequest(&findingsv1.ResolveDisputeRequest{
		Id:          id,
		Resolution:  strings.ToLower(strings.TrimSpace(ctx.Flag("resolution"))),
		Replacement: ctx.Flag("replacement"),
		Reason:      ctx.Flag("reason"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("resolve dispute %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Finding == nil {
		return fmt.Errorf("server returned no finding")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Resolved dispute on finding %s (now %s).", resp.Msg.Finding.Id, resp.Msg.Finding.Status.String())},
		Changes: []string{formatFinding(resp.Msg.Finding)},
	})
}

func parseInt32(s string) int32 {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return int32(n)
}

func formatFinding(f *findingsv1.Finding) string {
	if f == nil {
		return "(nil)"
	}
	created := ""
	if f.CreatedAt != nil {
		created = f.CreatedAt.AsTime().Format(time.RFC3339)
	}
	note := strings.TrimSpace(f.DisputeNote)
	if note != "" {
		note = " dispute=" + note
	}
	return fmt.Sprintf("%s — %s [status=%s confidence=%.2f%s created=%s]",
		f.Id, f.Claim, f.Status.String(), f.Confidence, note, created)
}
