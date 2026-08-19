package safety

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

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

// populateShadow restores a scenario's safety snapshots into the caller-named
// shadow namespaces (the data half of `baseline start` in shadow mode). The
// restores run asynchronously — poll `restores get <id>` for each.
func (h *handlers) populateShadow(ctx cliapp.RunContext) error {
	mappings, err := parseShadowMappings(ctx.Flag("mappings"))
	if err != nil {
		return fmt.Errorf("--mappings: %w", err)
	}
	resp, err := h.client.PopulateShadow(context.Background(), connect.NewRequest(&safetyv1.PopulateShadowRequest{
		Scenario: ctx.Flag("scenario"),
		RunId:    ctx.Flag("run-id"),
		Mappings: mappings,
	}))
	if err != nil {
		return cliapp.WrapAPIError("populate shadow", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no response")
	}
	m := resp.Msg
	changes := make([]string, 0, len(m.Restores)+len(m.Skipped))
	next := make([]string, 0, len(m.Restores))
	for _, r := range m.Restores {
		changes = append(changes, fmt.Sprintf("restoring %s [%s] → %s (restore=%s status=%s)", r.TargetName, r.SnapshotId, r.Location, r.RestoreId, r.Status))
		next = append(next, fmt.Sprintf("`restores get %s` — poll the restore until it reaches a terminal state", r.RestoreId))
	}
	for _, s := range m.Skipped {
		changes = append(changes, fmt.Sprintf("skipped %s — %s", s.TargetName, s.Reason))
	}
	result := fmt.Sprintf("Populating shadow for %q from safety run %s: %d restore(s) enqueued, %d skipped.", m.Scenario, m.RunId, len(m.Restores), len(m.Skipped))
	if len(m.Restores) == 0 {
		result = fmt.Sprintf("No shadow restores enqueued for %q (run %s); %d mapping(s) skipped.", m.Scenario, m.RunId, len(m.Skipped))
	}
	return cliapp.RenderProtoMutation(ctx, m, cliapp.MutationReport{
		Result:      []string{result},
		Changes:     changes,
		NextCommand: next,
	})
}

// parseShadowMappings parses the canonical JSON mapping array into the proto
// mapping slice. The legacy comma-separated target_name=location form remains
// accepted so existing operator scripts continue to work while the manifest's
// structured binding advertises the lossless representation.
func parseShadowMappings(raw string) ([]*safetyv1.ShadowTargetMapping, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("at least one target_name=location mapping is required")
	}
	if strings.HasPrefix(raw, "[") {
		var values []json.RawMessage
		if err := json.Unmarshal([]byte(raw), &values); err != nil {
			return nil, fmt.Errorf("decode JSON mappings: %w", err)
		}
		if len(values) == 0 {
			return nil, fmt.Errorf("at least one target_name=location mapping is required")
		}
		out := make([]*safetyv1.ShadowTargetMapping, 0, len(values))
		for _, value := range values {
			mapping := new(safetyv1.ShadowTargetMapping)
			if err := protojson.Unmarshal(value, mapping); err != nil {
				return nil, fmt.Errorf("decode JSON mapping: %w", err)
			}
			if strings.TrimSpace(mapping.GetTargetName()) == "" || strings.TrimSpace(mapping.GetLocation()) == "" {
				return nil, fmt.Errorf("each mapping requires target_name and location")
			}
			out = append(out, mapping)
		}
		return out, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]*safetyv1.ShadowTargetMapping, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		name, location, ok := strings.Cut(p, "=")
		name = strings.TrimSpace(name)
		location = strings.TrimSpace(location)
		if !ok || name == "" || location == "" {
			return nil, fmt.Errorf("invalid mapping %q (expected target_name=location)", p)
		}
		out = append(out, &safetyv1.ShadowTargetMapping{TargetName: name, Location: location})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one target_name=location mapping is required")
	}
	return out, nil
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
