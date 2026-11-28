package baseline

import (
	"context"
	"encoding/json"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	safetyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/safety"
)

// safetyBackupNow triggers a data-backup-manager safety backup of the
// scenario's registered targets to the secondary safety location — the
// pre-promote rollback snapshot and the source shadow data population copies
// from. It returns the run ID, or "" and a note saying why no backup started:
// the scenario has no registered stateful targets (code-only), the substrate
// call failed, or its output did not carry a run ID. All are non-fatal to the
// callers, but each note says which one happened.
func safetyBackupNow(ctx context.Context, scenario string) (string, string) {
	out, err := runCommand(ctx, "data-backup-manager", "safety", "backup-now", "--scenario", scenario, "--json")
	if err != nil {
		if strings.Contains(err.Error(), "no registered targets") {
			return "", "no registered stateful targets (code-only)"
		}
		return "", "safety backup unavailable: " + firstLine(err.Error())
	}
	// The CLI prints the generated BackupScenarioNowResponse with proto field
	// names ({"run_id": …}, cli-core PrintProtoJSON). Decoding through the
	// typed message accepts that and the lowerCamel spelling alike.
	var resp safetyv1.BackupScenarioNowResponse
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(out, &resp); err != nil {
		return "", "safety backup output unparseable: " + firstLine(err.Error())
	}
	runID := strings.TrimSpace(resp.GetRunId())
	if runID == "" {
		return "", "safety backup returned no run id"
	}
	return runID, ""
}

// awaitSafetyRun polls `runs get` until the backup run reaches a terminal state
// (its snapshots are only safe to restore from once it has finished) and returns
// that status, or "" when the attempt budget is exhausted.
func awaitSafetyRun(ctx context.Context, runID string) string {
	for attempt := 0; attempt < populateMaxAttempts; attempt++ {
		out, err := runCommand(ctx, "data-backup-manager", "runs", "get", runID, "--json")
		if err == nil {
			var resp struct {
				Run struct {
					Status string `json:"status"`
				} `json:"run"`
			}
			if json.Unmarshal(out, &resp) == nil && safetyRunTerminal(resp.Run.Status) {
				return resp.Run.Status
			}
		}
		sleepFn(populatePollInterval)
	}
	return ""
}
