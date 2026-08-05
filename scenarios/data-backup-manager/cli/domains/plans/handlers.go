package plans

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/plans"
	plansconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/plans/plans_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client plansconnect.PlansServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: plansconnect.NewPlansServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) create(ctx cliapp.RunContext) error {
	targetIDs := parseCommaSeparated(ctx.Flag("targets"))
	destIDs := parseCommaSeparated(ctx.Flag("destinations"))

	var retention *plansv1.RetentionPolicy
	if s := ctx.Flag("keep-latest"); s != "" {
		n, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return fmt.Errorf("--keep-latest: %w", err)
		}
		retention = &plansv1.RetentionPolicy{KeepLatest: int32(n)}
	}

	enabled := true
	if s := ctx.Flag("enabled"); s != "" {
		var err error
		enabled, err = strconv.ParseBool(s)
		if err != nil {
			return fmt.Errorf("--enabled: %w", err)
		}
	}

	resp, err := h.client.CreatePlan(context.Background(), connect.NewRequest(&plansv1.CreatePlanRequest{
		Name:                    ctx.Flag("name"),
		TargetIds:               targetIDs,
		DestinationIds:          destIDs,
		Schedule:                ctx.Flag("schedule"),
		Retention:               retention,
		Enabled:                 enabled,
		AllowIncompleteCoverage: ctx.BoolFlag("allow-incomplete-coverage"),
		ProtectionTier:          protectionTier(ctx.Flag("protection-tier")),
		RecoveryDrillSchedule:   optionalFlag(ctx, "recovery-drill-schedule"),
	}))
	if err != nil {
		// The API returns FAILED_PRECONDITION with the exact remediation commands
		// (coverage report / coverage accept-defaults) in the message, so the
		// wrapped error already guides the operator.
		return cliapp.WrapAPIError("create plan", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Plan == nil {
		return fmt.Errorf("server returned no plan")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Created plan %s.", resp.Msg.Plan.Id)},
		Changes: []string{formatPlan(resp.Msg.Plan)},
		NextCommand: []string{
			fmt.Sprintf("`plans get %s` — show this plan", resp.Msg.Plan.Id),
			fmt.Sprintf("`runs trigger --plan %s` — trigger a run now", resp.Msg.Plan.Id),
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetPlan(context.Background(), connect.NewRequest(&plansv1.GetPlanRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get plan %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Plan == nil {
		return fmt.Errorf("server returned no plan")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched plan %s.", resp.Msg.Plan.Id)},
		ResultsHeading: "Plan",
		Results:        []string{formatPlan(resp.Msg.Plan)},
	})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListPlans(context.Background(), connect.NewRequest(&plansv1.ListPlansRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list plans", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no plans response")
	}
	results := make([]string, 0, len(resp.Msg.Plans))
	for _, p := range resp.Msg.Plans {
		results = append(results, formatPlan(p))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d plan(s).", len(resp.Msg.Plans))},
		ResultsHeading: "Plans",
		Results:        results,
		RetrievalHints: []string{
			"`plans get <id>` — show a single plan",
			"`plans create --name <n> --targets <ids> --destinations <ids>` — create a plan",
		},
	})
}

func (h *handlers) update(ctx cliapp.RunContext) error {
	targetIDs := parseCommaSeparated(ctx.Flag("targets"))
	destIDs := parseCommaSeparated(ctx.Flag("destinations"))

	var retention *plansv1.RetentionPolicy
	if s := ctx.Flag("keep-latest"); s != "" {
		n, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return fmt.Errorf("--keep-latest: %w", err)
		}
		retention = &plansv1.RetentionPolicy{KeepLatest: int32(n)}
	}

	enabled := false
	if s := ctx.Flag("enabled"); s != "" {
		var err error
		enabled, err = strconv.ParseBool(s)
		if err != nil {
			return fmt.Errorf("--enabled: %w", err)
		}
	}

	resp, err := h.client.UpdatePlan(context.Background(), connect.NewRequest(&plansv1.UpdatePlanRequest{
		Id:                      ctx.Flag("id"),
		Name:                    ctx.Flag("name"),
		TargetIds:               targetIDs,
		DestinationIds:          destIDs,
		Schedule:                ctx.Flag("schedule"),
		Retention:               retention,
		Enabled:                 enabled,
		AllowIncompleteCoverage: ctx.BoolFlag("allow-incomplete-coverage"),
		ProtectionTier:          protectionTier(ctx.Flag("protection-tier")),
		RecoveryDrillSchedule:   optionalFlag(ctx, "recovery-drill-schedule"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("update plan", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Plan == nil {
		return fmt.Errorf("server returned no plan")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Updated plan %s.", resp.Msg.Plan.Id)},
		Changes: []string{formatPlan(resp.Msg.Plan)},
	})
}

func (h *handlers) delete(ctx cliapp.RunContext) error {
	resp, err := h.client.DeletePlan(context.Background(), connect.NewRequest(&plansv1.DeletePlanRequest{
		Id: ctx.Flag("id"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("delete plan", err, nil)
	}
	msg := "No matching plan to delete."
	if resp != nil && resp.Msg != nil && resp.Msg.Removed {
		msg = "Deleted plan."
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{msg},
	})
}

func formatPlan(p *plansv1.Plan) string {
	if p == nil {
		return "(nil)"
	}
	created := ""
	if p.CreatedAt != nil {
		created = p.CreatedAt.AsTime().Format(time.RFC3339)
	}
	keepLatest := int32(0)
	if p.Retention != nil {
		keepLatest = p.Retention.KeepLatest
	}
	return fmt.Sprintf("%s — %s [tier=%s targets=%s destinations=%s schedule=%q drill-schedule=%q keep-latest=%d enabled=%v physically-independent=%v warnings=%s created=%s]",
		p.Id, p.Name,
		p.ProtectionTier.String(),
		strings.Join(p.TargetIds, ","),
		strings.Join(p.DestinationIds, ","),
		p.Schedule, p.RecoveryDrillSchedule, keepLatest, p.Enabled, p.DestinationsPhysicallyIndependent, strings.Join(p.SharedRiskWarnings, " | "), created)
}

func protectionTier(value string) plansv1.ProtectionTier {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "critical-primary":
		return plansv1.ProtectionTier_PROTECTION_TIER_CRITICAL_PRIMARY
	case "critical-secondary":
		return plansv1.ProtectionTier_PROTECTION_TIER_CRITICAL_SECONDARY
	default:
		return plansv1.ProtectionTier_PROTECTION_TIER_FULL_PRIMARY
	}
}

func optionalFlag(ctx cliapp.RunContext, name string) string {
	if !ctx.FlagDeclared(name) {
		return ""
	}
	return ctx.Flag(name)
}

// parseCommaSeparated splits a comma-separated string into a slice, trimming spaces.
func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
