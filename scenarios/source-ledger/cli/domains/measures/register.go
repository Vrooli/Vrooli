package measures

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

const GroupName = "measures"

// Register exposes the journal's read-only analytical RPC through the same
// typed command surface as the other source-ledger domains.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"JournalService.CountEntries": h.appended,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("measures: load manifest: %w", err)
	}
	return group, nil
}

type handlers struct{ client journalconnect.JournalServiceClient }

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, base := cliapp.NewConnectHTTPClientWithTimeout(core, 0)
	return &handlers{client: journalconnect.NewJournalServiceClient(httpClient, base)}
}

func (h *handlers) appended(ctx cliapp.RunContext) error {
	window := strings.TrimSpace(ctx.Flag("window"))
	if window == "" {
		window = "this_week"
	}
	token, ok := measuresv1.TimeWindowToken_value["TIME_WINDOW_TOKEN_"+strings.ToUpper(window)]
	if !ok || token == 0 {
		return fmt.Errorf("unknown time window %q", window)
	}
	response, err := h.client.CountEntries(context.Background(), connect.NewRequest(&journalv1.CountEntriesRequest{
		Window: &measuresv1.TimeWindow{Window: &measuresv1.TimeWindow_Token{Token: measuresv1.TimeWindowToken(token)}},
		Scope:  ctx.Flag("scope"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("count source-ledger entries", err, nil)
	}
	return cliapp.RenderProtoList(ctx, response.Msg, cliapp.ListReport{
		Summary: []string{fmt.Sprintf("%d journal entries appended (%s).", response.Msg.GetCount(), window)},
		ResultsHeading: "Source-ledger measure",
		Results: []string{fmt.Sprintf("%d journal entries appended (%s)", response.Msg.GetCount(), window)},
	})
}
