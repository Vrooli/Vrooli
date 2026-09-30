package graph

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph"

	"github.com/vrooli/cli-core/cliapp"
)

// retentionPreview reports what a prune would reclaim, without deleting.
//
// This is the command an operator reaches for during a disk-pressure incident:
// it answers "how much of this database is snapshot history I do not need"
// before anything irreversible happens.
func (h *handlers) retentionPreview(ctx cliapp.RunContext) error {
	keep, err := parseKeepFlag(ctx.Flag("keep"))
	if err != nil {
		return err
	}

	resp, err := h.client.PreviewSnapshotRetention(context.Background(),
		connect.NewRequest(&graphv1.PreviewSnapshotRetentionRequest{KeepPerScenario: keep}))
	if err != nil {
		return cliapp.WrapAPIError("preview snapshot retention", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no retention preview")
	}
	msg := resp.Msg

	summary := []string{
		fmt.Sprintf("Keeping %d snapshot(s) per scenario: %d of %d row(s) reclaimable, %s.",
			msg.GetKeepPerScenario(), msg.GetReclaimableRows(), msg.GetTotalSnapshots(), humanBytes(msg.GetReclaimableBytes())),
	}

	// Only scenarios with something to reclaim are worth listing; the rest are
	// already at the floor and reporting them buries the signal.
	results := make([]string, 0, len(msg.GetScenarios()))
	for _, s := range msg.GetScenarios() {
		if s.GetReclaimableCount() == 0 {
			continue
		}
		results = append(results, fmt.Sprintf("%-40s %d snapshot(s), %d reclaimable",
			s.GetScenario(), s.GetSnapshotCount(), s.GetReclaimableCount()))
	}
	if len(results) == 0 {
		results = append(results, "Every scenario is already at the retention floor.")
	}

	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Scenarios above the floor",
		Results:        results,
		RetrievalHints: []string{
			"`graph retention-apply --confirm` prunes these rows and returns freed pages to the filesystem.",
		},
	})
}

// retentionApply prunes snapshots beyond the retention floor.
func (h *handlers) retentionApply(ctx cliapp.RunContext) error {
	keep, err := parseKeepFlag(ctx.Flag("keep"))
	if err != nil {
		return err
	}

	// Confirmation is checked client-side too, so an operator gets an
	// immediate, readable refusal rather than a round trip.
	if !isConfirmFlagSet(ctx.Flag("confirm")) {
		return fmt.Errorf("--confirm is required: applying retention permanently deletes snapshots")
	}

	resp, err := h.client.ApplySnapshotRetention(context.Background(),
		connect.NewRequest(&graphv1.ApplySnapshotRetentionRequest{KeepPerScenario: keep, Confirm: true}))
	if err != nil {
		return cliapp.WrapAPIError("apply snapshot retention", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no retention result")
	}
	msg := resp.Msg

	return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Removed %d snapshot row(s) across %d scenario(s), reclaiming %s.",
				msg.GetRowsRemoved(), msg.GetScenariosScanned(), humanBytes(msg.GetBytesReclaimed())),
		},
		Changes: []string{
			fmt.Sprintf("%d database page(s) returned to the filesystem", msg.GetPagesFreed()),
		},
		NextCommand: []string{"`graph retention-preview` — confirm the table is at its floor"},
	})
}

// parseKeepFlag reads the optional retention override. Empty means "use the
// server's configured floor".
func parseKeepFlag(raw string) (int32, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("--keep must be an integer: %w", err)
	}
	if value < 1 {
		return 0, fmt.Errorf("--keep must be at least 1; retention never empties the table")
	}
	return int32(value), nil
}

func isConfirmFlagSet(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "false", "0", "no":
		return false
	default:
		return true
	}
}

// humanBytes renders a byte count for operator output.
func humanBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
