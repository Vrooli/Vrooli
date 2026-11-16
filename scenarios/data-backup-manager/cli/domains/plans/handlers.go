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
	retention, err := mergeRetentionFlags(ctx, retention)
	if err != nil {
		return err
	}

	resp, err := h.client.CreatePlan(context.Background(), connect.NewRequest(&plansv1.CreatePlanRequest{
		Name:              ctx.Flag("name"),
		TargetIds:         targetIDs,
		TargetSelector:    selectorFlag(ctx.Flag("target-selector")),
		ExcludedTargetIds: parseCommaSeparated(ctx.Flag("exclude-targets")),
		DestinationIds:    destIDs,
		Schedule:          ctx.Flag("schedule"),
		Retention:         retention,
		// CreatePlan's contract creates enabled plans. The CLI deliberately
		// exposes --enabled only on update because proto3 cannot distinguish
		// omitted false from an explicit request for a disabled new plan.
		Enabled:                 true,
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

// addDestination prevalidates the entire selected set, then optionally applies
// an idempotent membership-only update to each plan. Preview is the default.
func (h *handlers) addDestination(ctx cliapp.RunContext) error {
	ids := parseCommaSeparated(ctx.Flag("plans"))
	destination := strings.TrimSpace(ctx.Flag("destination"))
	if len(ids) == 0 || destination == "" {
		return fmt.Errorf("--plans and --destination are required")
	}
	selected := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, exists := selected[id]; exists {
			return fmt.Errorf("duplicate plan id %q", id)
		}
		selected[id] = struct{}{}
	}
	resp, err := h.client.ListPlans(context.Background(), connect.NewRequest(&plansv1.ListPlansRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list plans", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no plans response")
	}
	byID := make(map[string]*plansv1.Plan, len(resp.Msg.Plans))
	for _, p := range resp.Msg.Plans {
		if p != nil {
			byID[p.Id] = p
		}
	}
	plans := make([]*plansv1.Plan, 0, len(ids))
	for _, id := range ids {
		p := byID[id]
		if p == nil {
			return fmt.Errorf("plan %q does not exist", id)
		}
		plans = append(plans, p)
	}
	apply := ctx.BoolFlag("apply")
	changes := make([]string, 0, len(plans))
	for _, p := range plans {
		if containsID(p.DestinationIds, destination) {
			changes = append(changes, fmt.Sprintf("%s: destination %s already present (unchanged)", p.Id, destination))
			continue
		}
		if apply {
			destinations := append(append([]string(nil), p.DestinationIds...), destination)
			_, err := h.client.UpdatePlan(context.Background(), connect.NewRequest(&plansv1.UpdatePlanRequest{
				Id: p.Id, Name: p.Name, TargetIds: p.TargetIds, TargetSelector: p.TargetSelector, ExcludedTargetIds: p.ExcludedTargetIds, DestinationIds: destinations,
				Schedule: p.Schedule, Retention: p.Retention, Enabled: p.Enabled,
				ProtectionTier: p.ProtectionTier, RecoveryDrillSchedule: p.RecoveryDrillSchedule,
			}))
			if err != nil {
				return cliapp.WrapAPIError(fmt.Sprintf("add destination to plan %q", p.Id), err, nil)
			}
		}
		action := "would add"
		if apply {
			action = "added"
		}
		changes = append(changes, fmt.Sprintf("%s: %s destination %s", p.Id, action, destination))
	}
	mode := "Preview only; pass --apply to write these changes."
	if apply {
		mode = "Applied membership changes."
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{mode}, ResultsHeading: "Plan destination changes", Results: changes})
}

func containsID(ids []string, wanted string) bool {
	for _, id := range ids {
		if id == wanted {
			return true
		}
	}
	return false
}

// mergeUpdate builds a full UpdatePlanRequest from the current plan, replacing
// only the fields whose flags the caller supplied.
func mergeUpdate(ctx cliapp.RunContext, p *plansv1.Plan) (*plansv1.UpdatePlanRequest, error) {
	req := &plansv1.UpdatePlanRequest{
		Id:                      p.Id,
		Name:                    p.Name,
		TargetIds:               p.TargetIds,
		TargetSelector:          p.TargetSelector,
		ExcludedTargetIds:       p.ExcludedTargetIds,
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
		req.TargetSelector, req.ExcludedTargetIds = "", nil // an explicit list replaces the rule
	}
	if ctx.FlagProvided("target-selector") {
		req.TargetSelector = selectorFlag(ctx.Flag("target-selector"))
	}
	if ctx.FlagProvided("exclude-targets") {
		req.ExcludedTargetIds = parseCommaSeparated(ctx.Flag("exclude-targets"))
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
		if req.Retention == nil {
			req.Retention = &plansv1.RetentionPolicy{}
		}
		req.Retention.KeepLatest = int32(n)
	}
	retention, err := mergeRetentionFlags(ctx, req.Retention)
	if err != nil {
		return nil, err
	}
	req.Retention = retention
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

func mergeRetentionFlags(ctx cliapp.RunContext, policy *plansv1.RetentionPolicy) (*plansv1.RetentionPolicy, error) {
	flags := []struct {
		name string
		set  func(int32)
	}{
		{"keep-hourly", func(v int32) { policy.KeepHourly = v }},
		{"keep-daily", func(v int32) { policy.KeepDaily = v }},
		{"keep-weekly", func(v int32) { policy.KeepWeekly = v }},
		{"keep-monthly", func(v int32) { policy.KeepMonthly = v }},
		{"keep-annual", func(v int32) { policy.KeepAnnual = v }},
	}
	for _, flag := range flags {
		if !ctx.FlagProvided(flag.name) {
			continue
		}
		n, err := strconv.ParseInt(ctx.Flag(flag.name), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("--%s: %w", flag.name, err)
		}
		if policy == nil {
			policy = &plansv1.RetentionPolicy{}
		}
		flag.set(int32(n))
	}
	return policy, nil
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
	return fmt.Sprintf("%s — %s [tier=%s targets=%s destinations=%s schedule=%q drill-schedule=%q keep-latest=%d GFS(hourly/daily/weekly/monthly/annual)=%d/%d/%d/%d/%d enabled=%v physically-independent=%v warnings=%s created=%s]",
		p.Id, p.Name,
		p.ProtectionTier.String(),
		describeTargets(p),
		strings.Join(p.DestinationIds, ","),
		p.Schedule, p.RecoveryDrillSchedule, keepLatest, p.Retention.GetKeepHourly(), p.Retention.GetKeepDaily(), p.Retention.GetKeepWeekly(), p.Retention.GetKeepMonthly(), p.Retention.GetKeepAnnual(), p.Enabled, p.DestinationsPhysicallyIndependent, strings.Join(p.SharedRiskWarnings, " | "), created)
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

// selectorFlag reads --target-selector: "none" (or empty) means an explicit list.
func selectorFlag(v string) string {
	v = strings.TrimSpace(v)
	if strings.EqualFold(v, "none") {
		return ""
	}
	return v
}

// describeTargets names a plan's targets: the list, or for a selector plan the
// rule and how many targets it names right now (a list of hundreds is no answer).
func describeTargets(p *plansv1.Plan) string {
	if p.TargetSelector == "" {
		return strings.Join(p.TargetIds, ",")
	}
	text := fmt.Sprintf("selector:%s (%d now", p.TargetSelector, len(p.TargetIds))
	if n := len(p.ExcludedTargetIds); n > 0 {
		text += fmt.Sprintf(", %d excluded: %s", n, strings.Join(p.ExcludedTargetIds, ","))
	}
	return text + ")"
}
