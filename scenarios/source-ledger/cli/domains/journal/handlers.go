package journal

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
)

type handlers struct {
	client journalconnect.JournalServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	http, base := cliapp.NewConnectHTTPClientWithTimeout(core, 0)
	return &handlers{client: journalconnect.NewJournalServiceClient(http, base)}
}

func (h *handlers) noteCall(ctx cliapp.OperationContext) (*journalv1.AppendEntryResponse, error) {
	response, err := h.client.AppendEntry(context.Background(), connect.NewRequest(&journalv1.AppendEntryRequest{Body: ctx.Positional("body"), Scope: ctx.Flag("scope"), Kind: ctx.Flag("kind"), Trigger: ctx.Flag("trigger"), Approach: ctx.Flag("approach"), Evidence: ctx.Flag("evidence"), Outcome: ctx.Flag("outcome")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("append source-ledger entry", err, nil)
	}
	if response == nil || response.Msg == nil || response.Msg.Entry == nil {
		return nil, fmt.Errorf("server returned no source-ledger entry")
	}
	return response.Msg, nil
}

func (h *handlers) noteReport(_ cliapp.OperationContext, msg *journalv1.AppendEntryResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Recorded %s.", msg.GetEntry().GetId())}, Changes: []string{fmt.Sprintf("[%s] %s", msg.GetEntry().GetFacetId(), msg.GetEntry().GetBody())}}
}
