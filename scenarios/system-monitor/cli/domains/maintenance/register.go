package maintenance

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	maintenancepb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/maintenance"
	maintenanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/maintenance/maintenanceconnect"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/proto"

	"system-monitor/cli/internal/support"
)

// Register exposes the metrics-lifecycle maintenance commands. Retention prunes
// stale metrics; compaction reclaims database file space. Destructive actions
// require an explicit --confirm flag and are thin wrappers over the API.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "maintenance",
		Description: "Preview and apply metrics retention and database compaction",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "retention", Description: "Preview or apply metrics retention (preview|apply)", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "action", Required: true, Description: "preview or apply"}}, Flags: []cliapp.Flag{{Name: "days", Description: "Retention window in days", Default: "30"}, {Name: "confirm", Description: "Confirm destructive prune", Bool: true}}}, RunCtx: h.retention},
			{Name: "compact", Description: "Preview or apply database compaction (preview|apply)", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "action", Required: true, Description: "preview or apply"}}, Flags: []cliapp.Flag{{Name: "confirm", Description: "Confirm compaction", Bool: true}}}, RunCtx: h.compact},
		},
	}
}

type handlers struct {
	client maintenanceconnect.MaintenanceServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		client: maintenanceconnect.NewMaintenanceServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) retention(ctx cliapp.RunContext) error {
	action := strings.TrimSpace(ctx.Positional("action"))
	switch action {
	case "preview":
		return h.retentionPreview(ctx)
	case "apply":
		return h.retentionApply(ctx)
	default:
		return fmt.Errorf("usage: system-monitor maintenance retention <preview|apply>")
	}
}

func (h *handlers) compact(ctx cliapp.RunContext) error {
	action := strings.TrimSpace(ctx.Positional("action"))
	switch action {
	case "preview":
		return h.compactPreview(ctx)
	case "apply":
		return h.compactApply(ctx)
	default:
		return fmt.Errorf("usage: system-monitor maintenance compact <preview|apply>")
	}
}

func (h *handlers) retentionPreview(ctx cliapp.RunContext) error {
	days, err := daysFlag(ctx)
	if err != nil {
		return err
	}

	resp, err := h.client.MetricsRetentionPreview(context.Background(), connect.NewRequest(&maintenancepb.MetricsRetentionPreviewRequest{
		RetentionDays: int32(days),
	}))
	if err != nil {
		return cliapp.WrapAPIError("preview metrics retention", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no retention preview")
	}
	est := resp.Msg.GetEstimate()
	stats := resp.Msg.GetDatabaseStats()
	return renderProtoOperational(ctx, resp.Msg, cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Retention preview for %d-day window (cutoff %s).", days, support.FormatMaybeString(est.GetCutoff(), "n/a")),
			fmt.Sprintf("%d rows / %s would be pruned.", est.GetRowCount(), formatBytes(est.GetPayloadBytes())),
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "Affected range", Items: []string{
				fmt.Sprintf("Oldest: %s", support.FormatMaybeString(est.GetOldestAffected(), "none")),
				fmt.Sprintf("Newest: %s", support.FormatMaybeString(est.GetNewestAffected(), "none")),
			}},
			{Heading: "Database", Items: databaseStatLines(stats)},
		},
		NextSteps: []string{
			fmt.Sprintf("system-monitor maintenance retention apply --days %d --confirm", days),
			"system-monitor maintenance compact preview",
		},
	})
}

func (h *handlers) retentionApply(ctx cliapp.RunContext) error {
	days, err := daysFlag(ctx)
	if err != nil {
		return err
	}
	if !ctx.BoolFlag("confirm") {
		return fmt.Errorf("--confirm is required to apply retention (this deletes metrics older than %d days)", days)
	}

	resp, err := h.client.MetricsRetentionApply(context.Background(), connect.NewRequest(&maintenancepb.MetricsRetentionApplyRequest{
		RetentionDays: int32(days),
		Confirm:       true,
	}))
	if err != nil {
		return cliapp.WrapAPIError("apply metrics retention", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no retention result")
	}
	before := resp.Msg.GetDatabaseStatsBefore()
	after := resp.Msg.GetDatabaseStatsAfter()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Pruned %d metric rows (cutoff %s).", resp.Msg.GetResult().GetDeletedRows(), support.FormatMaybeString(resp.Msg.GetResult().GetCutoff(), "n/a"))},
		Changes: []string{
			fmt.Sprintf("Metric rows: %d -> %d", before.GetMetricRows(), after.GetMetricRows()),
			fmt.Sprintf("DB size: %s -> %s", formatBytes(before.GetSizeBytes()), formatBytes(after.GetSizeBytes())),
			fmt.Sprintf("Freelist pages: %d -> %d", before.GetFreelistCount(), after.GetFreelistCount()),
		},
		NextCommand: []string{"system-monitor maintenance compact preview", "system-monitor maintenance compact apply --confirm"},
	})
}

func (h *handlers) compactPreview(ctx cliapp.RunContext) error {
	resp, err := h.client.MetricsCompactionPreview(context.Background(), connect.NewRequest(&maintenancepb.MetricsCompactionPreviewRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("preview metrics compaction", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no compaction preview")
	}
	return renderProtoOperational(ctx, resp.Msg, cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Compaction could reclaim approximately %s.", formatBytes(resp.Msg.GetEstimatedReclaimableBytes())),
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "Database", Items: databaseStatLines(resp.Msg.GetDatabaseStats())},
		},
		NextSteps: []string{"system-monitor maintenance compact apply --confirm"},
	})
}

func (h *handlers) compactApply(ctx cliapp.RunContext) error {
	if !ctx.BoolFlag("confirm") {
		return fmt.Errorf("--confirm is required to apply compaction")
	}

	resp, err := h.client.MetricsCompactionApply(context.Background(), connect.NewRequest(&maintenancepb.MetricsCompactionApplyRequest{Confirm: true}))
	if err != nil {
		return cliapp.WrapAPIError("apply metrics compaction", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no compaction result")
	}
	before := resp.Msg.GetDatabaseStatsBefore()
	after := resp.Msg.GetDatabaseStatsAfter()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Compaction reclaimed %s.", formatBytes(resp.Msg.GetReclaimedBytes()))},
		Changes: []string{
			fmt.Sprintf("DB size: %s -> %s", formatBytes(before.GetSizeBytes()), formatBytes(after.GetSizeBytes())),
			fmt.Sprintf("Freelist pages: %d -> %d", before.GetFreelistCount(), after.GetFreelistCount()),
		},
		NextCommand: []string{"system-monitor maintenance retention preview", "system-monitor status"},
	})
}

func daysFlag(ctx cliapp.RunContext) (int, error) {
	raw := strings.TrimSpace(ctx.Flag("days"))
	if raw == "" {
		raw = "30"
	}
	return parseDays(raw)
}

func parseDays(raw string) (int, error) {
	days, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("--days must be an integer")
	}
	if days <= 0 {
		return 0, fmt.Errorf("--days must be greater than 0")
	}
	return days, nil
}

func renderProtoOperational(ctx cliapp.RunContext, payload proto.Message, report cliapp.OperationalReport) error {
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), payload)
	}
	return ctx.RenderOperational(report)
}

func databaseStatLines(s *maintenancepb.DatabaseStats) []string {
	if s == nil {
		return []string{"(no database stats available)"}
	}
	return []string{
		fmt.Sprintf("Size: %s", formatBytes(s.GetSizeBytes())),
		fmt.Sprintf("Metric rows: %d", s.GetMetricRows()),
		fmt.Sprintf("Pages: %d (page size %d)", s.GetPageCount(), s.GetPageSize()),
		fmt.Sprintf("Freelist pages: %d", s.GetFreelistCount()),
	}
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
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
