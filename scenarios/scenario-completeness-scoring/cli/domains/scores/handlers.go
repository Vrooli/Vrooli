package scores

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	scoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring"
	scoringconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring/scoring_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers closes over the Connect client so each subcommand has typed API
// access without re-resolving the base URL.
type handlers struct {
	core   *cliapp.ScenarioApp
	client scoringconnect.ScoreServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: scoringconnect.NewScoreServiceClient(httpClient, baseURL),
	}
}

// get calls ScoreService.GetScore. Output routing: --json emits the proto
// wire shape (identical to a direct curl of the RPC); human consumers get
// the formatted status report (format.go).
func (h *handlers) get(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.GetScore(context.Background(), connect.NewRequest(&scoringv1.GetScoreRequest{
		Scenario: scenario,
	}))
	if err != nil {
		return cliapp.WrapAPIError("get score", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no score response")
	}

	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	_, err = fmt.Fprint(ctx.Stdout(), FormatReport(resp.Msg))
	return err
}
