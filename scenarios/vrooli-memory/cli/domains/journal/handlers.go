package journal

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/journal"
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/journal/journal_v1connect"
)

type handlers struct {
	client journalconnect.JournalServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	http, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: journalconnect.NewJournalServiceClient(http, base)}
}

func (h *handlers) noteCall(ctx cliapp.OperationContext) (*journalv1.AppendEntryResponse, error) {
	resp, err := h.client.AppendEntry(context.Background(), connect.NewRequest(&journalv1.AppendEntryRequest{Body: ctx.Positional("body"), Kind: ctx.Flag("kind")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("append memory", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Entry == nil {
		return nil, fmt.Errorf("server returned no entry")
	}
	return resp.Msg, nil
}

func (h *handlers) noteReport(_ cliapp.OperationContext, msg *journalv1.AppendEntryResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Remembered %s.", msg.Entry.Id)}, Changes: []string{fmt.Sprintf("[%s] %s", msg.Entry.FacetId, msg.Entry.Body)}, NextCommand: []string{"`recall <query>` — retrieve related memory"}}
}
