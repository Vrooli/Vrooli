package hosthandlers

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	gocacheapp "github.com/vrooli/vrooli/internal/app/gocache"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/gocache"
)

func runHostGoCache(runCtx cliapp.RunContext, parent *rootcli.CommandContext) error {
	return runHostGoCacheWith(runCtx, parent, gocacheapp.Service{})
}

func runHostGoCacheWith(runCtx cliapp.RunContext, parent *rootcli.CommandContext, service gocacheapp.Service) error {
	const command = "host go-cache"
	action := strings.ToLower(strings.TrimSpace(runCtx.Positional("action")))
	cache := strings.TrimSpace(runCtx.Flag("cache"))
	jsonOutput := parent.Globals.JSON || runCtx.JSON()
	out := runCtx.Stdout()
	switch action {
	case "status":
		status, err := service.Status(cache)
		if err != nil {
			return err
		}
		if jsonOutput {
			return cliout.WriteJSONValue(out, status)
		}
		return renderGoCacheStatus(out, status)
	case "plan":
		plan, err := service.Plan(cache, runCtx.BoolFlag("fresh"))
		if err != nil {
			return err
		}
		if jsonOutput {
			return cliout.WriteJSONValue(out, plan)
		}
		return renderGoCachePlan(out, plan)
	case "reclaim":
		maxBytes, err := parseOptionalInt(runCtx.Flag("max-bytes"))
		if err != nil {
			return rootcli.UsageErrorf(command, "--max-bytes: %v", err)
		}
		var budget time.Duration
		if value := strings.TrimSpace(runCtx.Flag("time-budget")); value != "" {
			if budget, err = time.ParseDuration(value); err != nil || budget <= 0 {
				return rootcli.UsageErrorf(command, "--time-budget must be a positive duration such as 40s")
			}
		}
		receipt, err := service.Reclaim(gocacheapp.ReclaimOptions{Cache: cache, Token: strings.TrimSpace(runCtx.Flag("token")), MaxBytes: maxBytes, TimeBudget: budget})
		if err != nil {
			return err
		}
		if jsonOutput {
			return cliout.WriteJSONValue(out, receipt)
		}
		return renderGoCacheReceipt(out, receipt)
	case "activate", "deactivate":
		run := service.Activate
		if action == "deactivate" {
			run = service.Deactivate
		}
		activation, err := run(parent.OperationContext())
		if jsonOutput {
			if writeErr := cliout.WriteJSONValue(out, activation); writeErr != nil {
				return writeErr
			}
			return err
		}
		if err != nil {
			return err
		}
		return renderGoCacheActivation(out, activation)
	default:
		return rootcli.UsageErrorf(command, "the action must be status, plan, reclaim, activate, or deactivate")
	}
}

func parseOptionalInt(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("want a non-negative byte count, got %q", value)
	}
	return parsed, nil
}

func renderGoCacheActivation(out io.Writer, activation gocache.Activation) error {
	state := "inactive: " + activation.Reason
	if activation.Active {
		state = "active"
	}
	return cliout.WriteSection(out, cliout.Section{Headers: []string{"field", "value"}, Rows: [][]string{
		{"protocol", state},
		{"helper", activation.Program},
		{"go env file", activation.EnvFile},
		{"GOCACHEPROG", activation.ConfiguredProgram},
	}})
}

func renderGoCacheStatus(out io.Writer, status gocache.Status) error {
	if err := renderGoCacheActivation(out, status.Activation); err != nil {
		return err
	}
	rows := [][]string{{"cache", status.Cache}, {"store id", status.StoreID}, {"live sessions", strconv.Itoa(status.LiveSessions)}}
	switch {
	case status.AuditError != "":
		rows = append(rows, []string{"audit", "unavailable: " + status.AuditError})
	case status.Audit != nil:
		rows = append(rows, []string{"go commands", fmt.Sprintf("%d live, %d mediated, %d on other caches, %d outside the protocol", status.Audit.GoCommands, status.Audit.Mediated, status.Audit.OtherStores, len(status.Audit.Unmediated))})
		for _, finding := range status.Audit.Unmediated {
			rows = append(rows, []string{fmt.Sprintf("pid %d", finding.PID), finding.Reason})
		}
	}
	if receipt := status.LastReclaim; receipt != nil {
		rows = append(rows, []string{"last reclaim", fmt.Sprintf("%s: freed %s in %d entries (%s)", receipt.FinishedAt.Format(time.RFC3339), humanBytes(receipt.FreedBytes), receipt.RemovedEntries, receipt.StopReason)})
	}
	return cliout.WriteSection(out, cliout.Section{Rows: rows})
}

func renderGoCachePlan(out io.Writer, plan gocache.Plan) error {
	rows := [][]string{{"cache", plan.Cache}, {"budget", humanBytes(plan.BudgetBytes)}}
	if plan.BlockedReason != "" {
		rows = append(rows, []string{"blocked", plan.BlockedReason})
		return cliout.WriteSection(out, cliout.Section{Rows: rows})
	}
	rows = append(rows, []string{"store", fmt.Sprintf("%s in %d entries", humanBytes(plan.StoreBytes), plan.StoreEntries)})
	if plan.Cutoff == nil {
		rows = append(rows, []string{"reclaimable", "nothing: the store is within budget"})
	} else {
		rows = append(rows,
			[]string{"reclaimable", fmt.Sprintf("%s in %d entries last used before %s", humanBytes(plan.ReclaimableBytes), plan.ReclaimableEntries, plan.Cutoff.Format(time.RFC3339))},
			[]string{"token", plan.Token})
	}
	return cliout.WriteSection(out, cliout.Section{Rows: rows})
}

func renderGoCacheReceipt(out io.Writer, receipt gocache.Receipt) error {
	rows := [][]string{
		{"cache", receipt.Cache},
		{"result", receipt.StopReason},
		{"freed", fmt.Sprintf("%s in %d entries", humanBytes(receipt.FreedBytes), receipt.RemovedEntries)},
		{"protected", fmt.Sprintf("%d held by live builds, %d used since the plan", receipt.SkippedPinned, receipt.SkippedRecentlyUsed)},
	}
	if receipt.BlockedReason != "" {
		rows = append(rows, []string{"blocked", receipt.BlockedReason})
	}
	if receipt.FailedRemovals > 0 {
		rows = append(rows, []string{"failed removals", fmt.Sprintf("%d (first: %s)", receipt.FailedRemovals, receipt.FirstFailure)})
	}
	return cliout.WriteSection(out, cliout.Section{Rows: rows})
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	value, suffix := float64(n), "KMGTPE"
	index := -1
	for value >= unit && index < len(suffix)-1 {
		value /= unit
		index++
	}
	return fmt.Sprintf("%.1f %ciB", value, suffix[index])
}
