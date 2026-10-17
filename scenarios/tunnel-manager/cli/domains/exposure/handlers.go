package exposure

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	exposurev1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/exposure"
	exposureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/exposure/exposure_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client exposureconnect.ExposureServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: exposureconnect.NewExposureServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) exposeCall(ctx cliapp.OperationContext) (*exposurev1.ExposeResponse, error) {
	req := &exposurev1.ExposeRequest{
		Scenario:    ctx.Positional("scenario"),
		RequestedBy: ctx.Flag("requested-by"),
	}
	ttlSeconds, err := resolveTTLSeconds(ctx)
	if err != nil {
		return nil, err
	}
	req.TtlSeconds = ttlSeconds
	resp, err := h.client.Expose(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("expose scenario", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no expose response")
	}
	// Expose now ensures the DNS CNAME as part of the same call (it errors out
	// above if the token lacks DNS scope), so a success here means ingress AND
	// DNS are configured. Cloudflare edge propagation is near-instant for a
	// proxied record but can take a few seconds on first publish.
	localPort := resp.Msg.LocalPort
	// Fallback to ListExposures to get the port if not present in the ExposeResponse
	// (ensures the report always shows the current pinned port even if the response extension
	// isn't populated on the wire yet).
	if localPort == 0 {
		if listResp, lerr := h.client.ListExposures(context.Background(), connect.NewRequest(&exposurev1.ListExposuresRequest{})); lerr == nil && listResp != nil {
			for _, e := range listResp.Msg.Exposures {
				if e.Scenario == req.Scenario {
					localPort = e.LocalPort
					resp.Msg.LocalPort = e.LocalPort
					break
				}
			}
		}
	}
	return resp.Msg, nil
}

func (h *handlers) exposeReport(ctx cliapp.OperationContext, message *exposurev1.ExposeResponse) cliapp.MutationReport {
	scenario := ctx.Positional("scenario")
	result := fmt.Sprintf("✓ Exposed %s — live at %s (ingress + proxied DNS configured).", scenario, message.PublicUrl)
	changes := []string{
		formatLease(message.Lease),
		"DNS: proxied CNAME → <tunnel>.cfargotunnel.com (created if absent).",
	}
	if message.LocalPort != 0 {
		changes = append(changes, fmt.Sprintf("Local port: %d (the fixed UI port used by the tunnel).", message.LocalPort))
	}
	if message.PortAssigned {
		changes = append(changes, "Assigned a fixed UI port (the scenario previously declared a range in service.json).")
		result += " Scenario cycled to bind the new port."
	}
	next := []string{
		fmt.Sprintf("`exposure check %s` — confirm public reachability", scenario),
		fmt.Sprintf("`exposure unexpose %s` — retract ingress + DNS", scenario),
		"`config credentials-status --verify` — if the URL is unreachable, re-check DNS scope",
	}
	if message.PortAssigned {
		next = append([]string{fmt.Sprintf("`vrooli scenario status %s` — verify it is now on the pinned port", scenario)}, next...)
	}
	return cliapp.MutationReport{
		Result:      []string{result},
		Changes:     changes,
		NextCommand: next,
	}
}

// resolveTTLSeconds reads the lease duration from --ttl (a human Go duration
// like 30m/2h/168h) or the legacy --ttl-seconds integer. --ttl wins when both
// are set. Zero (neither flag) lets the server apply its default.
func resolveTTLSeconds(ctx cliapp.OperationContext) (int64, error) {
	if v := strings.TrimSpace(ctx.Flag("ttl")); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return 0, fmt.Errorf("--ttl must be a duration like 30m, 2h, or 168h: %w", err)
		}
		if d <= 0 {
			return 0, fmt.Errorf("--ttl must be positive")
		}
		return int64(d.Seconds()), nil
	}
	if v := strings.TrimSpace(ctx.Flag("ttl-seconds")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("--ttl-seconds must be an integer: %w", err)
		}
		return n, nil
	}
	return 0, nil
}

// unexpose retracts a scenario's exposure by name — a thin alias over the
// ExposureService.Unexpose RPC (the server finds the active lease and revokes
// it, retracting ingress and the TM-created DNS record). No business logic
// lives here: the lookup-and-revoke is server-side.
func (h *handlers) unexposeCall(ctx cliapp.OperationContext) (*exposurev1.UnexposeResponse, error) {
	scenario := strings.TrimSpace(ctx.Positional("scenario"))
	if scenario == "" {
		return nil, fmt.Errorf("scenario is required")
	}
	resp, err := h.client.Unexpose(context.Background(), connect.NewRequest(&exposurev1.UnexposeRequest{Scenario: scenario}))
	if err != nil {
		return nil, cliapp.WrapAPIError("unexpose scenario", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no unexpose response")
	}
	return resp.Msg, nil
}

func (h *handlers) unexposeReport(ctx cliapp.OperationContext, message *exposurev1.UnexposeResponse) cliapp.MutationReport {
	scenario := ctx.Positional("scenario")
	msg := fmt.Sprintf("Unexposed %s (lease %s); ingress + DNS retracted.", scenario, message.LeaseId)
	if !message.Retracted {
		msg = fmt.Sprintf("Revoked lease %s for %s; scenario stays exposed (also CORE).", message.LeaseId, scenario)
	}
	return cliapp.MutationReport{Result: []string{msg}}
}

func (h *handlers) extendCall(ctx cliapp.OperationContext) (*exposurev1.ExtendLeaseResponse, error) {
	req := &exposurev1.ExtendLeaseRequest{LeaseId: ctx.Positional("lease_id")}
	if v := strings.TrimSpace(ctx.Flag("ttl-seconds")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("--ttl-seconds must be an integer: %w", err)
		}
		req.TtlSeconds = n
	}
	resp, err := h.client.ExtendLease(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("extend lease", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Lease == nil {
		return nil, fmt.Errorf("server returned no lease")
	}
	return resp.Msg, nil
}

func (h *handlers) extendReport(_ cliapp.OperationContext, message *exposurev1.ExtendLeaseResponse) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Extended lease %s.", message.Lease.Id)},
		Changes: []string{formatLease(message.Lease)},
	}
}

func (h *handlers) revokeCall(ctx cliapp.OperationContext) (*exposurev1.RevokeLeaseResponse, error) {
	id := ctx.Positional("lease_id")
	resp, err := h.client.RevokeLease(context.Background(), connect.NewRequest(&exposurev1.RevokeLeaseRequest{LeaseId: id}))
	if err != nil {
		return nil, cliapp.WrapAPIError("revoke lease", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no revoke response")
	}
	return resp.Msg, nil
}

func (h *handlers) revokeReport(ctx cliapp.OperationContext, message *exposurev1.RevokeLeaseResponse) cliapp.MutationReport {
	id := ctx.Positional("lease_id")
	msg := fmt.Sprintf("Revoked lease %s; ingress retracted.", id)
	if !message.Retracted {
		msg = fmt.Sprintf("Revoked lease %s; scenario stays exposed (also CORE).", id)
	}
	return cliapp.MutationReport{Result: []string{msg}}
}

func (h *handlers) leasesCall(ctx cliapp.OperationContext) (*exposurev1.ListLeasesResponse, error) {
	status, err := leaseStatusFlag(ctx.Flag("status"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.ListLeases(context.Background(), connect.NewRequest(&exposurev1.ListLeasesRequest{Status: status}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list leases", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no leases response")
	}
	return resp.Msg, nil
}

func (h *handlers) leasesReport(_ cliapp.OperationContext, msg *exposurev1.ListLeasesResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Leases))
	for _, l := range msg.Leases {
		results = append(results, formatLease(l))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d lease(s).", len(msg.Leases))},
		ResultsHeading: "Leases",
		Results:        results,
	}
}

func (h *handlers) listCall(_ cliapp.OperationContext) (*exposurev1.ListExposuresResponse, error) {
	resp, err := h.client.ListExposures(context.Background(), connect.NewRequest(&exposurev1.ListExposuresRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list exposures", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no exposures response")
	}
	return resp.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, msg *exposurev1.ListExposuresResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Exposures))
	for _, e := range msg.Exposures {
		results = append(results, formatExposure(e))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d exposure(s).", len(msg.Exposures))},
		ResultsHeading: "Exposures",
		Results:        results,
	}
}

func (h *handlers) checkCall(ctx cliapp.OperationContext) (*exposurev1.IsExposedResponse, error) {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.IsExposed(context.Background(), connect.NewRequest(&exposurev1.IsExposedRequest{Scenario: scenario}))
	if err != nil {
		return nil, cliapp.WrapAPIError("check exposure", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no exposure status")
	}
	return resp.Msg, nil
}

func (h *handlers) checkReport(ctx cliapp.OperationContext, msg *exposurev1.IsExposedResponse) cliapp.ListReport {
	scenario := ctx.Positional("scenario")
	line := fmt.Sprintf("%s is NOT exposed.", scenario)
	if msg.Exposed {
		line = fmt.Sprintf("%s is exposed at %s.", scenario, msg.PublicUrl)
	}
	return cliapp.ListReport{
		Summary:        []string{line},
		ResultsHeading: "Exposure",
		Results:        []string{line},
	}
}

func (h *handlers) reconcileCall(_ cliapp.OperationContext) (*exposurev1.ReconcileResponse, error) {
	resp, err := h.client.Reconcile(context.Background(), connect.NewRequest(&exposurev1.ReconcileRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("reconcile exposure", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no reconcile response")
	}
	return resp.Msg, nil
}

func (h *handlers) reconcileReport(_ cliapp.OperationContext, msg *exposurev1.ReconcileResponse) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Reconciled: %d core route(s) ensured, %d lease(s) reaped.", msg.CoreEnsured, msg.LeasesReaped)},
	}
}

func leaseStatusFlag(v string) (exposurev1.LeaseStatus, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "":
		return exposurev1.LeaseStatus_LEASE_STATUS_UNSPECIFIED, nil
	case "active":
		return exposurev1.LeaseStatus_LEASE_STATUS_ACTIVE, nil
	case "expired":
		return exposurev1.LeaseStatus_LEASE_STATUS_EXPIRED, nil
	case "revoked":
		return exposurev1.LeaseStatus_LEASE_STATUS_REVOKED, nil
	default:
		return exposurev1.LeaseStatus_LEASE_STATUS_UNSPECIFIED, fmt.Errorf("unknown status %q (use active, expired, or revoked)", v)
	}
}

func formatLease(l *exposurev1.Lease) string {
	if l == nil {
		return "(nil)"
	}
	status := strings.ToLower(strings.TrimPrefix(l.Status.String(), "LEASE_STATUS_"))
	expires := ""
	if l.ExpiresAt != nil {
		expires = l.ExpiresAt.AsTime().Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("%s — %s [%s, expires %s, extended %d, id=%s]",
		l.Scenario, l.RequestedBy, status, expires, l.ExtendedCount, l.Id)
}

func formatExposure(e *exposurev1.Exposure) string {
	if e == nil {
		return "(nil)"
	}
	state := "enabled"
	if !e.Enabled {
		state = "disabled"
	}
	return fmt.Sprintf("%s — %s :%d [%s, %s]", e.Scenario, e.PublicUrl, e.LocalPort, e.Tier, state)
}
