package pairing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	pairingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/pairing"
	pairingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/pairing/pairing_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// controlPlaneKeyFileName is the file the redeemed control-plane public key is
// pinned to inside the agent state dir. It MUST match the agent's
// config.controlPlaneKeyFileName (<state-dir>/control_plane.pub) — the node-agent
// reads exactly this path at startup and refuses to run without it. Change both
// together.
const controlPlaneKeyFileName = "control_plane.pub"

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
	// Resolve where to pin the control-plane key BEFORE burning the single-use
	// code, so we never spend a code we cannot complete — the node-agent refuses
	// to start without a pinned control-plane key (SECURITY.md boundary 2), so a
	// redeem that could not pin would leave the node unstartable and the code
	// spent.
	pinDir, err := resolvePinDir(ctx)
	if err != nil {
		return err
	}

	// The pairing code is a single-use secret. Prefer it from $BRIDGE_PAIRING_CODE
	// so the bootstrap installer never passes it on argv (where `ps` would leak it
	// to any local user); the --code flag stays as an explicit override. Exactly
	// one of the two must be present.
	code, err := resolvePairingCode(ctx)
	if err != nil {
		return err
	}

	resp, err := h.client.RedeemPairingCode(context.Background(), connect.NewRequest(&pairingv1.RedeemPairingCodeRequest{
		Code:          code,
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

	pinPath, perr := writePinnedKey(pinDir, resp.Msg.ControlPlanePublicKey)
	if perr != nil {
		// The code is already burned; surface the key so the operator can pin it by
		// hand rather than having to re-pair.
		return fmt.Errorf("paired as node %s but could not pin the control-plane key (%w) — write this base64 key to %s (0600) before starting the agent: %s",
			resp.Msg.NodeId, perr, filepath.Join(pinDir, controlPlaneKeyFileName), resp.Msg.ControlPlanePublicKey)
	}

	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Paired as node %s.", resp.Msg.NodeId)},
		Changes: []string{
			fmt.Sprintf("node id:           %s", resp.Msg.NodeId),
			fmt.Sprintf("control-plane key: %s", resp.Msg.ControlPlanePublicKey),
			fmt.Sprintf("pinned key file:   %s (0600)", pinPath),
		},
		NextCommand: []string{"Start the node-agent with this node id to hold the dial-out channel."},
	})
}

// pairingCodeEnvVar is the environment variable the bootstrap installer sets so
// the single-use pairing code never appears in the redeem process's argv (which
// `ps` would expose to any local user). The --code flag remains an explicit
// override for interactive use.
const pairingCodeEnvVar = "BRIDGE_PAIRING_CODE"

// resolvePairingCode returns the pairing code from the --code flag, falling back
// to $BRIDGE_PAIRING_CODE. Exactly one source must supply it; an empty result is
// a hard error BEFORE any RPC so a missing code never burns anything.
func resolvePairingCode(ctx cliapp.RunContext) (string, error) {
	return pairingCodeFrom(ctx.Flag("code"), os.Getenv(pairingCodeEnvVar))
}

// pairingCodeFrom picks the pairing code from the flag value, falling back to the
// env value. It is the pure core of resolvePairingCode (kept separate so it is
// testable without a RunContext). Empty (from both) is a hard error.
func pairingCodeFrom(flagValue, envValue string) (string, error) {
	code := strings.TrimSpace(flagValue)
	if code == "" {
		code = strings.TrimSpace(envValue)
	}
	if code == "" {
		return "", fmt.Errorf("no pairing code: pass --code or set $%s (kept out of argv so it cannot leak via ps)", pairingCodeEnvVar)
	}
	return code, nil
}

// resolvePinDir determines the agent state directory the control-plane key is
// pinned into: the --state-dir flag wins, else $BRIDGE_AGENT_STATE_DIR (the same
// env the agent reads). Pinning is mandatory, so an unresolvable directory is a
// hard error BEFORE the code is redeemed.
func resolvePinDir(ctx cliapp.RunContext) (string, error) {
	dir := strings.TrimSpace(ctx.Flag("state-dir"))
	if dir == "" {
		dir = strings.TrimSpace(os.Getenv("BRIDGE_AGENT_STATE_DIR"))
	}
	if dir == "" {
		return "", fmt.Errorf("no agent state directory to pin the control-plane key into: pass --state-dir or set BRIDGE_AGENT_STATE_DIR (the node-agent will not start without a pinned key)")
	}
	return dir, nil
}

// writePinnedKey persists the base64 control-plane public key to
// <dir>/control_plane.pub at 0600 (creating dir at 0700), the exact path the
// node-agent reads at startup. It returns the written path.
func writePinnedKey(dir, keyB64 string) (string, error) {
	key := strings.TrimSpace(keyB64)
	if key == "" {
		return "", fmt.Errorf("server returned no control-plane public key")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create state dir %q: %w", dir, err)
	}
	path := filepath.Join(dir, controlPlaneKeyFileName)
	if err := os.WriteFile(path, []byte(key), 0o600); err != nil {
		return "", fmt.Errorf("write pinned key %q: %w", path, err)
	}
	return path, nil
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
