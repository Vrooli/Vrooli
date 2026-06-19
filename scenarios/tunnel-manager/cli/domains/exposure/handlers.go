package exposure

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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

func (h *handlers) expose(ctx cliapp.RunContext) error {
	req := &exposurev1.ExposeRequest{
		Scenario:    ctx.Positional("scenario"),
		RequestedBy: ctx.Flag("requested-by"),
	}
	if v := strings.TrimSpace(ctx.Flag("ttl-seconds")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("--ttl-seconds must be an integer: %w", err)
		}
		req.TtlSeconds = n
	}
	resp, err := h.client.Expose(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("expose scenario", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no expose response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Exposed at %s.", resp.Msg.PublicUrl)},
		Changes: []string{formatLease(resp.Msg.Lease)},
		NextCommand: []string{
			"`exposure leases` — show active leases",
			fmt.Sprintf("`exposure check %s` — confirm reachability", req.Scenario),
		},
	})
}

func (h *handlers) extend(ctx cliapp.RunContext) error {
	req := &exposurev1.ExtendLeaseRequest{LeaseId: ctx.Positional("lease_id")}
	if v := strings.TrimSpace(ctx.Flag("ttl-seconds")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("--ttl-seconds must be an integer: %w", err)
		}
		req.TtlSeconds = n
	}
	resp, err := h.client.ExtendLease(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("extend lease", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Lease == nil {
		return fmt.Errorf("server returned no lease")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Extended lease %s.", resp.Msg.Lease.Id)},
		Changes: []string{formatLease(resp.Msg.Lease)},
	})
}

func (h *handlers) revoke(ctx cliapp.RunContext) error {
	id := ctx.Positional("lease_id")
	resp, err := h.client.RevokeLease(context.Background(), connect.NewRequest(&exposurev1.RevokeLeaseRequest{LeaseId: id}))
	if err != nil {
		return cliapp.WrapAPIError("revoke lease", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no revoke response")
	}
	msg := fmt.Sprintf("Revoked lease %s; ingress retracted.", id)
	if !resp.Msg.Retracted {
		msg = fmt.Sprintf("Revoked lease %s; scenario stays exposed (also CORE).", id)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{msg}})
}

func (h *handlers) leases(ctx cliapp.RunContext) error {
	status, err := leaseStatusFlag(ctx.Flag("status"))
	if err != nil {
		return err
	}
	resp, err := h.client.ListLeases(context.Background(), connect.NewRequest(&exposurev1.ListLeasesRequest{Status: status}))
	if err != nil {
		return cliapp.WrapAPIError("list leases", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no leases response")
	}
	results := make([]string, 0, len(resp.Msg.Leases))
	for _, l := range resp.Msg.Leases {
		results = append(results, formatLease(l))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d lease(s).", len(resp.Msg.Leases))},
		ResultsHeading: "Leases",
		Results:        results,
	})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListExposures(context.Background(), connect.NewRequest(&exposurev1.ListExposuresRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list exposures", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no exposures response")
	}
	results := make([]string, 0, len(resp.Msg.Exposures))
	for _, e := range resp.Msg.Exposures {
		results = append(results, formatExposure(e))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d exposure(s).", len(resp.Msg.Exposures))},
		ResultsHeading: "Exposures",
		Results:        results,
	})
}

func (h *handlers) check(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.IsExposed(context.Background(), connect.NewRequest(&exposurev1.IsExposedRequest{Scenario: scenario}))
	if err != nil {
		return cliapp.WrapAPIError("check exposure", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no exposure status")
	}
	line := fmt.Sprintf("%s is NOT exposed.", scenario)
	if resp.Msg.Exposed {
		line = fmt.Sprintf("%s is exposed at %s.", scenario, resp.Msg.PublicUrl)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{line},
		ResultsHeading: "Exposure",
		Results:        []string{line},
	})
}

func (h *handlers) reconcile(ctx cliapp.RunContext) error {
	resp, err := h.client.Reconcile(context.Background(), connect.NewRequest(&exposurev1.ReconcileRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("reconcile exposure", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no reconcile response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Reconciled: %d core route(s) ensured, %d lease(s) reaped.", resp.Msg.CoreEnsured, resp.Msg.LeasesReaped)},
	})
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
