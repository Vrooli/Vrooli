package capacity

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	capacitypb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/capacity"
	capacityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/capacity/capacityconnect"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/proto"

	"system-monitor/cli/internal/support"
)

// Register exposes the capacity governance commands. They are read-only over the
// platform claim ledger (overview/claims/reconcile/policy get) plus a single
// policy mutation (policy set). Claim mutation flows through `vrooli capacity`,
// never this scenario CLI.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "capacity",
		Description: "Inspect the host capacity claim ledger and tune policy levers",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "overview", Description: "Show per-GPU contention and the active claim table", RunCtx: h.overview},
			{Name: "claims", Description: "List capacity claims (--owner, --active)", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "owner", Description: "Filter to a single owner id"}, {Name: "active", Description: "Only show active claims", Bool: true}}}, RunCtx: h.claims},
			{Name: "reconcile", Description: "Classify observed GPU consumers against the ledger", RunCtx: h.reconcile},
			{Name: "policy", Description: "Show or set capacity policy levers (get|set)", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "action", Description: "get or set"}, {Name: "key", Description: "Policy lever key"}, {Name: "value", Description: "Policy lever value"}}}, RunCtx: h.policy},
		},
	}
}

type handlers struct {
	client capacityconnect.CapacityServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		client: capacityconnect.NewCapacityServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) overview(ctx cliapp.RunContext) error {
	resp, err := h.client.GetCapacityOverview(context.Background(), connect.NewRequest(&capacitypb.GetCapacityOverviewRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get capacity overview", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no capacity overview")
	}

	gpuGroups := make([]cliapp.TriageGroup, 0, len(resp.Msg.GetGpus()))
	for _, g := range resp.Msg.GetGpus() {
		gpuGroups = append(gpuGroups, cliapp.TriageGroup{
			Heading: fmt.Sprintf("GPU %d (%s)", g.GetIndex(), support.FormatMaybeString(g.GetName(), "unknown")),
			Items: []string{
				fmt.Sprintf("Total: %s", formatBytes(g.GetTotalBytes())),
				fmt.Sprintf("Used (observed): %s", formatBytes(g.GetUsedBytes())),
				fmt.Sprintf("Free: %s", formatBytes(g.GetFreeBytes())),
				fmt.Sprintf("Claimed: %s", formatBytes(g.GetClaimedBytes())),
				fmt.Sprintf("Mem util: %.1f%%", g.GetMemoryUtilizationPercent()),
			},
		})
	}
	if len(gpuGroups) == 0 {
		gpuGroups = append(gpuGroups, cliapp.TriageGroup{Heading: "GPUs", Items: []string{"(no GPUs detected)"}})
	}
	gpuGroups = append(gpuGroups, cliapp.TriageGroup{Heading: "Active claims", Items: claimLines(resp.Msg.GetClaims())})

	status := []string{
		fmt.Sprintf("%d active claim(s) across %d GPU(s).", len(resp.Msg.GetClaims()), len(resp.Msg.GetGpus())),
		fmt.Sprintf("Capacity sensing: %s", support.BoolString(resp.Msg.GetSensingAvailable(), "available", "unavailable")),
	}
	for _, warn := range resp.Msg.GetWarnings() {
		status = append(status, "⚠ "+warn)
	}

	return renderProtoOperational(ctx, resp.Msg, cliapp.OperationalReport{
		Status: status,
		Triage: gpuGroups,
		NextSteps: []string{
			"system-monitor capacity reconcile",
			"system-monitor capacity claims --active",
			"system-monitor capacity policy get",
		},
	})
}

func (h *handlers) claims(ctx cliapp.RunContext) error {
	resp, err := h.client.ListCapacityClaims(context.Background(), connect.NewRequest(&capacitypb.ListCapacityClaimsRequest{
		OwnerId:    strings.TrimSpace(ctx.Flag("owner")),
		ActiveOnly: ctx.BoolFlag("active"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list capacity claims", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no capacity claims")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d claim(s).", len(resp.Msg.GetClaims()))},
		ResultsHeading: "Claims",
		Results:        claimLines(resp.Msg.GetClaims()),
		RetrievalHints: []string{"system-monitor capacity overview", "system-monitor capacity reconcile"},
	})
}

func (h *handlers) reconcile(ctx cliapp.RunContext) error {
	resp, err := h.client.ReconcileCapacity(context.Background(), connect.NewRequest(&capacitypb.ReconcileCapacityRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("reconcile capacity", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no capacity reconciliation")
	}

	lines := make([]string, 0, len(resp.Msg.GetFindings()))
	warnCount := 0
	for _, f := range resp.Msg.GetFindings() {
		marker := " "
		if f.GetSeverity() == "warn" {
			marker = "⚠"
			warnCount++
		}
		lines = append(lines, fmt.Sprintf("%s [%s] %s", marker, strings.ToUpper(f.GetClass()), f.GetMessage()))
	}
	if len(lines) == 0 {
		lines = append(lines, "No GPU consumers above the tracking threshold.")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d finding(s), %d warning(s).", len(resp.Msg.GetFindings()), warnCount)},
		ResultsHeading: "Reconciliation",
		Results:        lines,
		RetrievalHints: []string{"system-monitor capacity claims --active", "system-monitor capacity policy get"},
	})
}

func (h *handlers) policy(ctx cliapp.RunContext) error {
	action := strings.TrimSpace(ctx.Positional("action"))
	switch action {
	case "get", "":
		return h.policyGet(ctx)
	case "set":
		return h.policySet(ctx)
	default:
		return fmt.Errorf("usage: system-monitor capacity policy <get|set>")
	}
}

func (h *handlers) policyGet(ctx cliapp.RunContext) error {
	resp, err := h.client.GetCapacityPolicy(context.Background(), connect.NewRequest(&capacitypb.GetCapacityPolicyRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get capacity policy", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no capacity policy")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d policy lever(s).", len(resp.Msg.GetLevers()))},
		ResultsHeading: "Policy",
		Results:        leverLines(resp.Msg.GetLevers()),
		RetrievalHints: []string{"system-monitor capacity policy set <key> <value>"},
	})
}

func (h *handlers) policySet(ctx cliapp.RunContext) error {
	key, value := strings.TrimSpace(ctx.Positional("key")), strings.TrimSpace(ctx.Positional("value"))
	if key == "" || value == "" {
		return fmt.Errorf("usage: system-monitor capacity policy set <key> <value>")
	}

	resp, err := h.client.SetCapacityPolicy(context.Background(), connect.NewRequest(&capacitypb.SetCapacityPolicyRequest{Key: key, Value: value}))
	if err != nil {
		return cliapp.WrapAPIError("set capacity policy", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no updated capacity policy")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Set %s = %s.", key, value)},
		Changes:     leverLines(resp.Msg.GetLevers()),
		NextCommand: []string{"system-monitor capacity policy get", "system-monitor capacity overview"},
	})
}

func renderProtoOperational(ctx cliapp.RunContext, payload proto.Message, report cliapp.OperationalReport) error {
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), payload)
	}
	return ctx.RenderOperational(report)
}

func claimLines(claims []*capacitypb.CapacityClaim) []string {
	if len(claims) == 0 {
		return []string{"(no claims)"}
	}
	lines := make([]string, 0, len(claims))
	for _, c := range claims {
		protectedMark := ""
		if c.GetProtected() {
			protectedMark = " 🛡"
		}
		lines = append(lines, fmt.Sprintf("%s/%s  %s  %s  prio=%s  activity=%s  %s%s",
			c.GetOwnerKind(), c.GetOwnerId(), c.GetResourceKind(), c.GetStatus(),
			c.GetPriorityTier(), c.GetActivityState(), formatBytes(c.GetAmountBytes()), protectedMark))
	}
	return lines
}

func leverLines(levers []*capacitypb.PolicyLever) []string {
	lines := make([]string, 0, len(levers))
	for _, l := range levers {
		lines = append(lines, fmt.Sprintf("%s = %s", l.GetKey(), l.GetValue()))
	}
	if len(lines) == 0 {
		return []string{"(no policy levers)"}
	}
	return lines
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
