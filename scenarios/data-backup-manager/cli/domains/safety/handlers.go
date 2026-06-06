package safety

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	safetyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/safety"
	safetyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/safety/safety_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client safetyconnect.SafetyServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: safetyconnect.NewSafetyServiceClient(httpClient, baseURL),
	}
}

// ensureDestination idempotently provisions the reserved baseline-safety
// destination. Safe to call repeatedly; created=false means it already existed.
func (h *handlers) ensureDestination(ctx cliapp.RunContext) error {
	capBytes, err := parseOptionalInt64(ctx.Flag("cap-bytes"))
	if err != nil {
		return fmt.Errorf("--cap-bytes: %w", err)
	}
	resp, err := h.client.EnsureSafetyDestination(context.Background(), connect.NewRequest(&safetyv1.EnsureSafetyDestinationRequest{
		CapBytes: capBytes,
	}))
	if err != nil {
		return cliapp.WrapAPIError("ensure safety destination", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Destination == nil {
		return fmt.Errorf("server returned no destination")
	}
	d := resp.Msg.Destination
	state := "already existed"
	if resp.Msg.Created {
		state = "created"
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Safety destination %q %s.", d.Name, state)},
		Changes: []string{fmt.Sprintf("%s — %s", d.Id, d.Location)},
		NextCommand: []string{
			"`safety backup-now --scenario <s>` — take a pre-promote snapshot of a scenario's targets",
		},
	})
}

// backupNow backs up a scenario's registered targets now via an ephemeral plan.
// The run is asynchronous — it returns immediately; poll `runs get <id>`.
func (h *handlers) backupNow(ctx cliapp.RunContext) error {
	keepLatest, err := parseOptionalInt32(ctx.Flag("keep-latest"))
	if err != nil {
		return fmt.Errorf("--keep-latest: %w", err)
	}
	resp, err := h.client.BackupScenarioNow(context.Background(), connect.NewRequest(&safetyv1.BackupScenarioNowRequest{
		Scenario:   ctx.Flag("scenario"),
		KeepLatest: keepLatest,
	}))
	if err != nil {
		return cliapp.WrapAPIError("backup scenario now", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no response")
	}
	m := resp.Msg
	return cliapp.RenderProtoMutation(ctx, m, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Queued safety backup run %s for %d target(s) — executing in the background.", m.RunId, m.TargetCount)},
		Changes: []string{fmt.Sprintf("run=%s plan=%s destination=%s status=%s", m.RunId, m.PlanId, m.DestinationId, m.Status)},
		NextCommand: []string{
			fmt.Sprintf("`runs get %s` — poll the run until it reaches a terminal state", m.RunId),
		},
	})
}

// registerTargets derives and registers a scenario's reliably-conventional
// backup targets (Postgres + filesystem data dir) so `backup-now` works without
// a hand-run `targets register`. Idempotent.
func (h *handlers) registerTargets(ctx cliapp.RunContext) error {
	resp, err := h.client.RegisterScenarioTargets(context.Background(), connect.NewRequest(&safetyv1.RegisterScenarioTargetsRequest{
		Scenario: ctx.Flag("scenario"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("register scenario targets", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no response")
	}
	m := resp.Msg
	changes := make([]string, 0, len(m.Registered)+len(m.Skipped))
	for _, t := range m.Registered {
		changes = append(changes, fmt.Sprintf("registered %s [%s] → %s", t.Name, t.SourceKind, t.Locator))
	}
	for _, s := range m.Skipped {
		changes = append(changes, fmt.Sprintf("skipped %s — %s", s.SourceKind, s.Reason))
	}
	result := fmt.Sprintf("Registered %d backup target(s) for %q.", len(m.Registered), m.Scenario)
	if len(m.Registered) == 0 {
		result = fmt.Sprintf("No derivable backup targets for %q — register them explicitly with `targets register`.", m.Scenario)
	}
	return cliapp.RenderProtoMutation(ctx, m, cliapp.MutationReport{
		Result:  []string{result},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("`safety backup-now --scenario %s` — take a pre-promote snapshot of the registered targets", m.Scenario),
		},
	})
}

// parseOptionalInt64 parses an optional non-negative int64 flag; empty -> 0.
func parseOptionalInt64(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("must be a non-negative integer, got %q", s)
	}
	return n, nil
}

// parseOptionalInt32 parses an optional non-negative int32 flag; empty -> 0.
func parseOptionalInt32(s string) (int32, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("must be a non-negative integer, got %q", s)
	}
	return int32(n), nil
}
