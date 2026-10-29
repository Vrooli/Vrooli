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

// update is a partial update: it reads the current plan and overrides only the
// fields whose flags were supplied. UpdatePlan itself replaces the whole plan,
// so sending zero values for omitted flags would silently disable the plan,
// drop its retention and drill schedule, and reset its protection tier.
func (h *handlers) update(ctx cliapp.RunContext) error {
	id := ctx.Flag("id")
	current, err := h.client.GetPlan(context.Background(), connect.NewRequest(&plansv1.GetPlanRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get plan %q", id), err, nil)
	}
	if current == nil || current.Msg == nil || current.Msg.Plan == nil {
		return fmt.Errorf("server returned no plan")
	}
	req, err := mergeUpdate(ctx, current.Msg.Plan)
	if err != nil {
		return err
	}
	resp, err := h.client.UpdatePlan(context.Background(), connect.NewRequest(req))
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

// mergeUpdate builds a full UpdatePlanRequest from the current plan, replacing
// only the fields whose flags the caller supplied.
func mergeUpdate(ctx cliapp.RunContext, p *plansv1.Plan) (*plansv1.UpdatePlanRequest, error) {
	req := &plansv1.UpdatePlanRequest{
		Id:                      p.Id,
		Name:                    p.Name,
		TargetIds:               p.TargetIds,
		DestinationIds:          p.DestinationIds,
		Schedule:                p.Schedule,
		Retention:               p.Retention,
		Enabled:                 p.Enabled,
		AllowIncompleteCoverage: ctx.BoolFlag("allow-incomplete-coverage"),
		ProtectionTier:          p.ProtectionTier,
		RecoveryDrillSchedule:   p.RecoveryDrillSchedule,
	}
	if ctx.FlagProvided("name") {
		req.Name = ctx.Flag("name")
	}
	if ctx.FlagProvided("targets") {
		req.TargetIds = parseCommaSeparated(ctx.Flag("targets"))
	}
	if ctx.FlagProvided("destinations") {
		req.DestinationIds = parseCommaSeparated(ctx.Flag("destinations"))
	}
	if ctx.FlagProvided("schedule") {
		req.Schedule = ctx.Flag("schedule")
	}
	if ctx.FlagProvided("keep-latest") {
		n, err := strconv.ParseInt(ctx.Flag("keep-latest"), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("--keep-latest: %w", err)
		}
		req.Retention = &plansv1.RetentionPolicy{KeepLatest: int32(n)}
	}
	if ctx.FlagProvided("enabled") {
		enabled, err := strconv.ParseBool(ctx.Flag("enabled"))
		if err != nil {
			return nil, fmt.Errorf("--enabled: %w", err)
		}
		req.Enabled = enabled
	}
	if ctx.FlagProvided("protection-tier") {
		req.ProtectionTier = protectionTier(ctx.Flag("protection-tier"))
	}
	if ctx.FlagProvided("recovery-drill-schedule") {
		req.RecoveryDrillSchedule = ctx.Flag("recovery-drill-schedule")
	}
	return req, nil
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
