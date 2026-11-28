package integrations

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/integrations"
	vc "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/integrations/integrations_v1connect"
)

type handlers struct{ client vc.IntegrationsServiceClient }

func newHandlers(c *cliapp.ScenarioApp) *handlers {
	httpClient, base := cliapp.NewConnectHTTPClient(c)
	return &handlers{client: vc.NewIntegrationsServiceClient(httpClient, base)}
}

func (h *handlers) listCall(_ cliapp.OperationContext) (*v.ListConnectionsResponse, error) {
	r, err := h.client.ListConnections(context.Background(), connect.NewRequest(&v.ListConnectionsRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list integrations", err, nil)
	}
	return r.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, m *v.ListConnectionsResponse) cliapp.ListReport {
	results := make([]string, 0, len(m.Connections))
	for _, c := range m.Connections {
		results = append(results, fmt.Sprintf("%s [%s] status=%s read_only=%t events=%d busy=%dm revision=%d", c.DisplayName, c.Provider, c.Status, c.ReadOnly, c.ImportedEventCount, c.BusyMinutes, c.Revision))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d calendar connection(s)", len(results))}, ResultsHeading: "Connections", Results: results}
}

func (h *handlers) createFixtureCall(c cliapp.OperationContext) (*v.CreateFixtureConnectionResponse, error) {
	r, err := h.client.CreateFixtureConnection(context.Background(), connect.NewRequest(&v.CreateFixtureConnectionRequest{DisplayName: c.Flag("display-name")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create fixture connection", err, nil)
	}
	return r.Msg, nil
}
func (h *handlers) createReport(_ cliapp.OperationContext, m *v.CreateFixtureConnectionResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created synthetic read-only connection %q at revision %d.", m.Connection.DisplayName, m.Connection.Revision)}, NextCommand: []string{"`integrations sync --id ... --revision ...` — refresh its fixture data"}}
}

func (h *handlers) syncCall(c cliapp.OperationContext) (*v.SyncConnectionResponse, error) {
	revision, err := strconv.ParseInt(c.Flag("revision"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("revision must be an integer: %w", err)
	}
	r, err := h.client.SyncConnection(context.Background(), connect.NewRequest(&v.SyncConnectionRequest{Id: c.Flag("id"), ExpectedRevision: revision}))
	if err != nil {
		return nil, cliapp.WrapAPIError("sync integration", err, nil)
	}
	return r.Msg, nil
}
func (h *handlers) syncReport(_ cliapp.OperationContext, m *v.SyncConnectionResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Synced %q: %d imported events, %d busy minutes.", m.Connection.DisplayName, m.ImportedEventCount, m.BusyMinutes)}}
}

func (h *handlers) disconnectCall(c cliapp.OperationContext) (*v.DisconnectConnectionResponse, error) {
	revision, err := strconv.ParseInt(c.Flag("revision"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("revision must be an integer: %w", err)
	}
	r, err := h.client.DisconnectConnection(context.Background(), connect.NewRequest(&v.DisconnectConnectionRequest{Id: c.Flag("id"), ExpectedRevision: revision}))
	if err != nil {
		return nil, cliapp.WrapAPIError("disconnect integration", err, nil)
	}
	return r.Msg, nil
}
func (h *handlers) disconnectReport(_ cliapp.OperationContext, m *v.DisconnectConnectionResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Disconnected %q; imported facts are no longer included in capacity.", m.Connection.DisplayName)}}
}
