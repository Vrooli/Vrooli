package pairing

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	pairingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/pairing"
	pairingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/pairing/pairing_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the Connect client closure over *cliapp.ScenarioApp.
type handlers struct {
	core   *cliapp.ScenarioApp
	client pairingconnect.PairingServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: pairingconnect.NewPairingServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) issue(ctx cliapp.RunContext) error {
	var ttl int64
	if raw := strings.TrimSpace(ctx.Flag("ttl")); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("--ttl must be a whole number of seconds: %w", err)
		}
		ttl = v
	}
	resp, err := h.client.IssuePairingCode(context.Background(), connect.NewRequest(&pairingv1.IssuePairingCodeRequest{
		Name:       ctx.Flag("name"),
		Scopes:     splitCSV(ctx.Flag("scopes")),
		TtlSeconds: ttl,
	}))
	if err != nil {
		return cliapp.WrapAPIError("issue pairing code (run `auth login` first if unauthenticated)", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no response")
	}
	expires := ""
	if resp.Msg.ExpiresAt != nil {
		expires = resp.Msg.ExpiresAt.AsTime().Format("2006-01-02 15:04:05 MST")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{"Pairing code issued (shown once — copy it now)."},
		Changes: []string{
			fmt.Sprintf("code:            %s", resp.Msg.Code),
			fmt.Sprintf("expires:         %s", expires),
			fmt.Sprintf("control-plane key: %s", resp.Msg.ControlPlanePublicKey),
		},
		NextCommand: []string{
			"On the node, run the bootstrap installer (it redeems this code and pins the control-plane key).",
		},
	})
}

func (h *handlers) redeem(ctx cliapp.RunContext) error {
	resp, err := h.client.RedeemPairingCode(context.Background(), connect.NewRequest(&pairingv1.RedeemPairingCodeRequest{
		Code:          ctx.Flag("code"),
		NodePublicKey: ctx.Flag("public-key"),
		Name:          ctx.Flag("name"),
		Os:            ctx.Flag("os"),
		Arch:          ctx.Flag("arch"),
		Endpoint:      ctx.Flag("endpoint"),
		Capabilities:  splitCSV(ctx.Flag("capabilities")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("redeem pairing code", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Paired as node %s.", resp.Msg.NodeId)},
		Changes: []string{
			fmt.Sprintf("node id:           %s", resp.Msg.NodeId),
			fmt.Sprintf("control-plane key: %s", resp.Msg.ControlPlanePublicKey),
		},
		NextCommand: []string{"Start the node-agent with this node id to hold the dial-out channel."},
	})
}

func (h *handlers) request(ctx cliapp.RunContext) error {
	resp, err := h.client.RequestPairing(context.Background(), connect.NewRequest(&pairingv1.RequestPairingRequest{
		NodePublicKey: ctx.Flag("public-key"),
		Name:          ctx.Flag("name"),
		Os:            ctx.Flag("os"),
		Arch:          ctx.Flag("arch"),
		Endpoint:      ctx.Flag("endpoint"),
		Capabilities:  splitCSV(ctx.Flag("capabilities")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("request pairing", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Pairing request %s submitted (%s).", resp.Msg.RequestId, statusLabel(resp.Msg.Status))},
		NextCommand: []string{"Ask the owner to `pair approve " + resp.Msg.RequestId + "`."},
	})
}

func (h *handlers) approve(ctx cliapp.RunContext) error {
	id := ctx.Positional("request-id")
	approve := !ctx.BoolFlag("reject")
	resp, err := h.client.ApprovePairing(context.Background(), connect.NewRequest(&pairingv1.ApprovePairingRequest{
		RequestId: id,
		Approve:   approve,
		Scopes:    splitCSV(ctx.Flag("scopes")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("decide pairing request %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no response")
	}
	result := fmt.Sprintf("Request %s rejected.", id)
	if resp.Msg.NodeId != "" {
		result = fmt.Sprintf("Request %s approved — minted node %s.", id, resp.Msg.NodeId)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{result},
		NextCommand: []string{"`pair list --all` — review requests", "`nodes list` — see the fleet"},
	})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListPairingRequests(context.Background(), connect.NewRequest(&pairingv1.ListPairingRequestsRequest{
		IncludeDecided: ctx.BoolFlag("all"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list pairing requests", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no response")
	}
	results := make([]string, 0, len(resp.Msg.Requests))
	for _, r := range resp.Msg.Requests {
		results = append(results, formatRequest(r))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d pairing request(s).", len(resp.Msg.Requests))},
		ResultsHeading: "Requests",
		Results:        results,
		RetrievalHints: []string{"`pair approve <request-id>` — approve a request"},
	})
}

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

func formatRequest(r *pairingv1.PairingRequest) string {
	if r == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s — %s [%s/%s status=%s]", r.Id, r.Name, r.Os, r.Arch, statusLabel(r.Status))
}

func statusLabel(s pairingv1.PairingRequestStatus) string {
	switch s {
	case pairingv1.PairingRequestStatus_PAIRING_REQUEST_STATUS_PENDING:
		return "pending"
	case pairingv1.PairingRequestStatus_PAIRING_REQUEST_STATUS_APPROVED:
		return "approved"
	case pairingv1.PairingRequestStatus_PAIRING_REQUEST_STATUS_REJECTED:
		return "rejected"
	default:
		return "unknown"
	}
}
