package nodes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry/registry_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each RunCtx-func has
// typed access to the API client without re-resolving it. The owner JWT rides
// the configured token source (set it via `configure token` or $VROOLI_BRIDGE_API_TOKEN).
type handlers struct {
	core   *cliapp.ScenarioApp
	client registryconnect.NodeRegistryServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: registryconnect.NewNodeRegistryServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) register(ctx cliapp.RunContext) error {
	resp, err := h.client.RegisterNode(context.Background(), connect.NewRequest(&registryv1.RegisterNodeRequest{
		Name:         ctx.Flag("name"),
		Os:           ctx.Flag("os"),
		Arch:         ctx.Flag("arch"),
		Endpoint:     ctx.Flag("endpoint"),
		Capabilities: splitCSV(ctx.Flag("capabilities")),
		Scopes:       splitCSV(ctx.Flag("scopes")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("register node (set a token via `configure token` or $VROOLI_BRIDGE_API_TOKEN if unauthenticated)", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Node == nil {
		return fmt.Errorf("server returned no node")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Registered node %s.", resp.Msg.Node.Id)},
		Changes: []string{formatNode(resp.Msg.Node)},
		NextCommand: []string{
			fmt.Sprintf("`nodes get %s` — show this node", resp.Msg.Node.Id),
			"`nodes list` — show the fleet",
		},
	})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListNodes(context.Background(), connect.NewRequest(&registryv1.ListNodesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list nodes", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no nodes response")
	}
	results := make([]string, 0, len(resp.Msg.Nodes))
	for _, n := range resp.Msg.Nodes {
		results = append(results, formatNode(n))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d node(s) in the fleet.", len(resp.Msg.Nodes))},
		ResultsHeading: "Nodes",
		Results:        results,
		RetrievalHints: []string{
			"`nodes get <id>` — show a single node",
			"`nodes register --name <n> --os <os> --arch <arch>` — add a node",
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetNode(context.Background(), connect.NewRequest(&registryv1.GetNodeRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get node %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Node == nil {
		return fmt.Errorf("server returned no node")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched node %s.", resp.Msg.Node.Id)},
		ResultsHeading: "Node",
		Results:        []string{formatNode(resp.Msg.Node)},
	})
}

func (h *handlers) update(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.UpdateNode(context.Background(), connect.NewRequest(&registryv1.UpdateNodeRequest{
		Id:           id,
		Name:         ctx.Flag("name"),
		Endpoint:     ctx.Flag("endpoint"),
		Capabilities: splitCSV(ctx.Flag("capabilities")),
		Scopes:       splitCSV(ctx.Flag("scopes")),
		Revision:     ctx.Flag("revision"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("update node %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Node == nil {
		return fmt.Errorf("server returned no node")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated node %s.", resp.Msg.Node.Id)},
		Changes:     []string{formatNode(resp.Msg.Node)},
		NextCommand: []string{fmt.Sprintf("`nodes get %s` — show this node", resp.Msg.Node.Id)},
	})
}

func (h *handlers) revoke(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.RevokeNode(context.Background(), connect.NewRequest(&registryv1.RevokeNodeRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("revoke node %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Node == nil {
		return fmt.Errorf("server returned no node")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Revoked node %s — its access is severed.", resp.Msg.Node.Id)},
		Changes:     []string{formatNode(resp.Msg.Node)},
		NextCommand: []string{"`nodes list` — confirm the fleet"},
	})
}

// splitCSV parses a comma-separated flag value into a trimmed, empty-free slice.
func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// formatNode produces a one-line representation for ListReport / MutationReport.
func formatNode(n *registryv1.Node) string {
	if n == nil {
		return "(nil)"
	}
	created := ""
	if n.CreatedAt != nil {
		created = n.CreatedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — %s [%s/%s status=%s online=%t rev=%s scopes=%d created=%s]",
		n.Id, n.Name, n.Os, n.Arch, statusLabel(n.Status), n.Online, formatRevision(n.Revision), len(n.Scopes), created)
}

// formatRevision renders a node's provenance revision for a one-line listing,
// shortening a long commit sha but PRESERVING the "+dirty" working-tree marker so
// a dirty node reads loudly as "e767613fca+dirty" — visibly not a pinned node.
func formatRevision(rev string) string {
	rev = strings.TrimSpace(rev)
	if rev == "" {
		return "-"
	}
	base, dirty := strings.CutSuffix(rev, "+dirty")
	if len(base) > 12 {
		base = base[:12]
	}
	if dirty {
		return base + "+dirty"
	}
	return base
}

// statusLabel renders the NodeStatus enum as a short lowercase word.
func statusLabel(s registryv1.NodeStatus) string {
	switch s {
	case registryv1.NodeStatus_NODE_STATUS_ONLINE:
		return "online"
	case registryv1.NodeStatus_NODE_STATUS_OFFLINE:
		return "offline"
	case registryv1.NodeStatus_NODE_STATUS_NEEDS_UPDATE:
		return "needs-update"
	case registryv1.NodeStatus_NODE_STATUS_REVOKED:
		return "revoked"
	default:
		return "unknown"
	}
}
