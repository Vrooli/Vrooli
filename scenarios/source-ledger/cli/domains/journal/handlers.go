package journal

import (
	"context"
	"fmt"
	"strconv"

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

func (h *handlers) listCall(ctx cliapp.OperationContext) (*journalv1.ListEntriesResponse, error) {
	limit, err := strconv.Atoi(ctx.Flag("limit"))
	if err != nil || limit < 1 || limit > 500 {
		return nil, fmt.Errorf("limit must be 1-500")
	}
	newest := ctx.BoolFlag("newest-first")
	if ctx.Flag("kind") != "" && !newest {
		return nil, fmt.Errorf("kind requires --newest-first")
	}
	if newest && ctx.Flag("cursor") != "" {
		return nil, fmt.Errorf("newest-first does not accept a cursor")
	}
	response, err := h.client.ListEntries(context.Background(), connect.NewRequest(&journalv1.ListEntriesRequest{Scope: ctx.Flag("scope"), Limit: int32(limit), NewestFirst: newest, Kind: ctx.Flag("kind"), Cursor: ctx.Flag("cursor")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list journal", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, msg *journalv1.ListEntriesResponse) cliapp.ListReport {
	lines := []string{}
	for _, e := range msg.GetEntries() {
		lines = append(lines, fmt.Sprintf("%s [%s] %s", e.GetId(), e.GetKind(), e.GetBody()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d journal entries", len(lines))}, Results: lines}
}

func (h *handlers) getCall(ctx cliapp.OperationContext) (*journalv1.GetEntryResponse, error) {
	response, err := h.client.GetEntry(context.Background(), connect.NewRequest(&journalv1.GetEntryRequest{Id: ctx.Positional("id"), Scope: ctx.Flag("scope")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get journal entry", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) getReport(_ cliapp.OperationContext, msg *journalv1.GetEntryResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{msg.GetEntry().GetId()}, Results: []string{msg.GetEntry().GetBody()}}
}
