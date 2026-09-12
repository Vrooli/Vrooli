package debt

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	debtv1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/debt"
	debtconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/debt/debt_v1connect"
)

type handlers struct {
	client debtconnect.DebtServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: debtconnect.NewDebtServiceClient(httpClient, baseURL)}
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*debtv1.ListDebtResponse, error) {
	resp, err := h.client.ListDebt(context.Background(), connect.NewRequest(&debtv1.ListDebtRequest{
		TemplateId: ctx.Flag("template"),
		Status:     ctx.Flag("status"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list debt", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, msg *debtv1.ListDebtResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Entries))
	for _, entry := range msg.Entries {
		results = append(results, formatDebt(entry))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d debt entrie(s).", len(msg.Entries))}, ResultsHeading: "Debt", Results: results}
}

func (h *handlers) showCall(ctx cliapp.OperationContext) (*debtv1.GetDebtResponse, error) {
	key := ctx.Positional("key")
	resp, err := h.client.GetDebt(context.Background(), connect.NewRequest(&debtv1.GetDebtRequest{Key: key}))
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("show debt %q", key), err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) showReport(_ cliapp.OperationContext, msg *debtv1.GetDebtResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Fetched debt entry %s.", msg.Entry.Key)}, ResultsHeading: "Debt entry", Results: []string{formatDebt(msg.Entry)}}
}

func formatDebt(entry *debtv1.DebtEntry) string {
	if entry == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s template=%s severity=%s status=%s title=%s", entry.Key, entry.TemplateId, entry.Severity, entry.Status, entry.Title)
}
