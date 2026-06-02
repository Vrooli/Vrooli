package maintenance

import (
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/api"

	"system-monitor/cli/internal/support"
)

// Register exposes the metrics-lifecycle maintenance commands. Retention prunes
// stale metrics; compaction reclaims database file space. Destructive actions
// require an explicit --confirm flag and are thin wrappers over the API.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "maintenance",
		Description: "Preview and apply metrics retention and database compaction",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "retention", Description: "Preview or apply metrics retention (preview|apply)", Run: func(args []string) error { return runRetention(core, args) }},
			{Name: "compact", Description: "Preview or apply database compaction (preview|apply)", Run: func(args []string) error { return runCompact(core, args) }},
		},
	}
}

func runRetention(core *cliapp.ScenarioApp, args []string) error {
	action, rest := splitAction(args)
	switch action {
	case "preview":
		return runRetentionPreview(core, rest)
	case "apply":
		return runRetentionApply(core, rest)
	default:
		return fmt.Errorf("usage: system-monitor maintenance retention <preview|apply>")
	}
}

func runCompact(core *cliapp.ScenarioApp, args []string) error {
	action, rest := splitAction(args)
	switch action {
	case "preview":
		return runCompactPreview(core, rest)
	case "apply":
		return runCompactApply(core, rest)
	default:
		return fmt.Errorf("usage: system-monitor maintenance compact <preview|apply>")
	}
}

func runRetentionPreview(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("maintenance retention preview")
	days := fs.Int("days", 30, "Retention window in days")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *days <= 0 {
		return fmt.Errorf("--days must be greater than 0")
	}

	body, err := core.Get("/maintenance/metrics/retention/preview", url.Values{"days": {strconv.Itoa(*days)}})
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var resp apipb.MetricsRetentionPreviewResponse
	if err := support.DecodeProto(body, &resp); err != nil {
		return err
	}
	est := resp.GetEstimate()
	stats := resp.GetDatabaseStats()
	return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Retention preview for %d-day window (cutoff %s).", *days, support.FormatMaybeString(est.GetCutoff(), "n/a")),
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
			fmt.Sprintf("system-monitor maintenance retention apply --days %d --confirm", *days),
			"system-monitor maintenance compact preview",
		},
	})
}

func runRetentionApply(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("maintenance retention apply")
	days := fs.Int("days", 30, "Retention window in days")
	confirm := fs.Bool("confirm", false, "Required: confirm the destructive prune")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *days <= 0 {
		return fmt.Errorf("--days must be greater than 0")
	}
	if !*confirm {
		return fmt.Errorf("--confirm is required to apply retention (this deletes metrics older than %d days)", *days)
	}

	body, err := core.Request("POST", "/maintenance/metrics/retention/apply", nil, &apipb.MetricsRetentionApplyRequest{
		RetentionDays: int32(*days),
		Confirm:       true,
	})
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var resp apipb.MetricsRetentionApplyResponse
	if err := support.DecodeProto(body, &resp); err != nil {
		return err
	}
	before := resp.GetDatabaseStatsBefore()
	after := resp.GetDatabaseStatsAfter()
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Pruned %d metric rows (cutoff %s).", resp.GetResult().GetDeletedRows(), support.FormatMaybeString(resp.GetResult().GetCutoff(), "n/a"))},
		Changes: []string{
			fmt.Sprintf("Metric rows: %d -> %d", before.GetMetricRows(), after.GetMetricRows()),
			fmt.Sprintf("DB size: %s -> %s", formatBytes(before.GetSizeBytes()), formatBytes(after.GetSizeBytes())),
			fmt.Sprintf("Freelist pages: %d -> %d", before.GetFreelistCount(), after.GetFreelistCount()),
		},
		NextCommand: []string{"system-monitor maintenance compact preview", "system-monitor maintenance compact apply --confirm"},
	})
}

func runCompactPreview(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("maintenance compact preview")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/maintenance/metrics/compaction/preview", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var resp apipb.MetricsCompactionPreviewResponse
	if err := support.DecodeProto(body, &resp); err != nil {
		return err
	}
	return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Compaction could reclaim approximately %s.", formatBytes(resp.GetEstimatedReclaimableBytes())),
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "Database", Items: databaseStatLines(resp.GetDatabaseStats())},
		},
		NextSteps: []string{"system-monitor maintenance compact apply --confirm"},
	})
}

func runCompactApply(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("maintenance compact apply")
	confirm := fs.Bool("confirm", false, "Required: confirm the compaction")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if !*confirm {
		return fmt.Errorf("--confirm is required to apply compaction")
	}

	body, err := core.Request("POST", "/maintenance/metrics/compaction/apply", nil, &apipb.MetricsCompactionApplyRequest{Confirm: true})
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var resp apipb.MetricsCompactionApplyResponse
	if err := support.DecodeProto(body, &resp); err != nil {
		return err
	}
	before := resp.GetDatabaseStatsBefore()
	after := resp.GetDatabaseStatsAfter()
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Compaction reclaimed %s.", formatBytes(resp.GetReclaimedBytes()))},
		Changes: []string{
			fmt.Sprintf("DB size: %s -> %s", formatBytes(before.GetSizeBytes()), formatBytes(after.GetSizeBytes())),
			fmt.Sprintf("Freelist pages: %d -> %d", before.GetFreelistCount(), after.GetFreelistCount()),
		},
		NextCommand: []string{"system-monitor maintenance retention preview", "system-monitor status"},
	})
}

func splitAction(args []string) (string, []string) {
	if len(args) == 0 {
		return "", nil
	}
	return args[0], args[1:]
}

func databaseStatLines(s *apipb.DatabaseStats) []string {
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
