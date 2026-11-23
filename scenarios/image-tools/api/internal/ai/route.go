package ai

// Cloud routing policy for model-backed image jobs.
//
// Local stays first. The cloud is used in exactly three situations, each
// bounded by a per-job spend cap and a rolling daily cap that are read from
// recorded cost_usd (never from a counter the process keeps in memory):
//
//  1. The host has no usable GPU for a generative edit, so the best local
//     choice is a weak CPU model: the job goes to the cloud when spend allows.
//  2. A job that waited for local admission longer than its wait budget is
//     re-routed once, before it starts, by the jobs scheduler.
//  3. The caller asked for the cloud (allow_byok, a role, or the gateway model).
//
// See docs/internal/DECISIONS.md (2026-11-21, cloud fallback rule).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	aigateway "image-tools/internal/aigateway"
	"image-tools/internal/backends"
	"image-tools/internal/capability"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/models"
	"image-tools/internal/resolver"
)

// Environment switches. Every value is optional; unset keeps the default.
const (
	EnvCloudFallback      = "IMAGE_TOOLS_CLOUD_FALLBACK"
	EnvCloudWaitBudget    = "IMAGE_TOOLS_CLOUD_WAIT_BUDGET"
	EnvCloudMaxJobUSD     = "IMAGE_TOOLS_CLOUD_MAX_JOB_USD"
	EnvCloudMaxDailyUSD   = "IMAGE_TOOLS_CLOUD_MAX_DAILY_USD"
	EnvCloudCostEstimates = "IMAGE_TOOLS_CLOUD_COST_ESTIMATES"
	EnvCloudCooldown      = "IMAGE_TOOLS_CLOUD_COOLDOWN"
)

// Gateway roles the routing policy chooses between.
const (
	RoleEditIdentity     = "image.edit.identity"
	RoleEditDefault      = "image.edit.default"
	RoleGenerateDefault  = "image.generate.default"
	RoleGenerateQuality  = "image.generate.quality"
	RoleVisionDefault    = "vision.default"
	routeLedgerWindow    = 24 * time.Hour
	routeLedgerRowLimit  = 2000
	routeCloudEstimateS  = 30
	routeChoiceLocal     = "local"
	routeChoiceCloud     = "cloud"
	routeReasonNoGPU     = "no_usable_gpu"
	routeReasonNoLocal   = "no_local_model"
	routeReasonWait      = "wait_budget_exceeded"
	routeReasonRequested = "caller_requested_cloud"

	// Reasons a job that could have used the cloud stays local (route_choice=local).
	routeStayPinned   = "local_model_pinned"
	routeStayLocalReq = "caller_requires_local"
	routeStayNotAllow = "cloud_not_permitted"
	routeStayCap      = "cloud_spend_cap"
	routeStayNoWait   = "no_wait_budget"
	routeStayCooldown = "cloud_cooldown"
	routeStayConsent  = "consent_required"

	// routeReasonFellBack marks a job whose cloud attempt was refused before any
	// billing and which went back to its original local route.
	routeReasonFellBack = "cloud_unavailable_fell_back_local"

	// MetaCloudRefusedAt and MetaCloudRefusal record a refused cloud route on a
	// job. The cooldown reads the first from the job table.
	MetaCloudRefusedAt = "route_cloud_refused_at"
	MetaCloudRefusal   = "route_cloud_refusal"
)

// EditStyle values for AIParams.edit_style / PlanRequest.EditStyle.
const (
	EditStyleIdentity = "identity"
	EditStyleStylized = "stylized"
)

// ErrSpendCap is returned when a cloud route would exceed the per-job or the
// rolling daily spend cap. Nothing has been billed when it is returned.
var ErrSpendCap = internaljobs.Kinded(internaljobs.FailureRefused, errors.New("ai: cloud spend cap"))

// RouteConfig is the deployment-level routing policy. The zero value is
// inactive: no automatic cloud use and no cap enforcement, which is what
// engines built without a host policy (tests, isolated tools) get.
type RouteConfig struct {
	// Active turns the policy on. DefaultRouteConfig sets it.
	Active bool
	// Enabled allows automatic cloud use (GPU-less preference and wait-budget
	// re-route) when the caller states no preference. A caller's
	// fallback_policy=local_only always wins.
	Enabled bool
	// WaitBudget is how long a job may wait for local admission before it is
	// re-routed. Zero disables the wait trigger.
	WaitBudget time.Duration
	// MaxJobUSD and MaxDailyUSD are the per-job and rolling 24-hour caps.
	MaxJobUSD   float64
	MaxDailyUSD float64
	// CostEstimates is the per-role ceiling used BEFORE a call, because the real
	// cost is only known afterwards. Recorded cost_usd replaces it in the ledger.
	CostEstimates map[string]float64
	// DefaultEstimateUSD prices a role with no entry above.
	DefaultEstimateUSD float64
	// Cooldown pauses automatic cloud routing after a cloud route is refused
	// before billing (no credits, provider unavailable, authentication), so each
	// waiting job does not meet the same refusal in turn. Zero disables it. A
	// caller who demands the cloud is never held back by it.
	Cooldown time.Duration
}

// DefaultRouteConfig is the operator-approved default: on, a 10 minute wait
// budget, $0.25 per job and $3 per rolling day.
func DefaultRouteConfig() RouteConfig {
	return RouteConfig{
		Active:             true,
		Enabled:            true,
		WaitBudget:         10 * time.Minute,
		MaxJobUSD:          0.25,
		MaxDailyUSD:        3,
		DefaultEstimateUSD: 0.15,
		Cooldown:           30 * time.Minute,
		CostEstimates: map[string]float64{
			RoleEditIdentity:      0.14,
			RoleEditDefault:       0.05,
			RoleGenerateDefault:   0.05,
			RoleGenerateQuality:   0.14,
			"image.generate.fast": 0.03,
			// Vision roles answer in text: a call costs a fraction of a cent, so
			// a caption never counts as an image against the daily cap.
			RoleVisionDefault: 0.005,
		},
	}
}

// RouteConfigFromEnv overlays the environment on DefaultRouteConfig. A
// malformed value is an error rather than a silent default, because a typo in
// a spending limit must not quietly become a different limit.
func RouteConfigFromEnv(getenv func(string) string) (RouteConfig, error) {
	cfg := DefaultRouteConfig()
	if raw := strings.TrimSpace(getenv(EnvCloudFallback)); raw != "" {
		switch strings.ToLower(raw) {
		case "1", "true", "yes", "on":
			cfg.Enabled = true
		case "0", "false", "no", "off":
			cfg.Enabled = false
		default:
			return cfg, fmt.Errorf("%s: %q is not on/off", EnvCloudFallback, raw)
		}
	}
	if raw := strings.TrimSpace(getenv(EnvCloudWaitBudget)); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil || d < 0 {
			return cfg, fmt.Errorf("%s: %q is not a non-negative duration such as 10m (0 disables the wait trigger)", EnvCloudWaitBudget, raw)
		}
		cfg.WaitBudget = d
	}
	for _, item := range []struct {
		name string
		dst  *float64
	}{{EnvCloudMaxJobUSD, &cfg.MaxJobUSD}, {EnvCloudMaxDailyUSD, &cfg.MaxDailyUSD}} {
		if raw := strings.TrimSpace(getenv(item.name)); raw != "" {
			v, err := strconv.ParseFloat(raw, 64)
			if err != nil || v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
				return cfg, fmt.Errorf("%s: %q is not a non-negative dollar amount such as 0.25", item.name, raw)
			}
			*item.dst = v
		}
	}
	if raw := strings.TrimSpace(getenv(EnvCloudCooldown)); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil || d < 0 {
			return cfg, fmt.Errorf("%s: %q is not a non-negative duration such as 30m (0 disables the cooldown)", EnvCloudCooldown, raw)
		}
		cfg.Cooldown = d
	}
	if raw := strings.TrimSpace(getenv(EnvCloudCostEstimates)); raw != "" {
		for _, pair := range strings.Split(raw, ",") {
			role, value, ok := strings.Cut(strings.TrimSpace(pair), "=")
			v, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
			if !ok || strings.TrimSpace(role) == "" || err != nil || v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
				return cfg, fmt.Errorf("%s: %q is not role=usd[,role=usd]", EnvCloudCostEstimates, pair)
			}
			cfg.CostEstimates[strings.TrimSpace(role)] = v
		}
	}
	return cfg, nil
}

func (c RouteConfig) estimate(role string, variations int) float64 {
	per, ok := c.CostEstimates[role]
	if !ok {
		per = c.DefaultEstimateUSD
	}
	return per * float64(max(variations, 1))
}

// JobSource is the durable job history the spend ledger reads. *jobs.Manager
// satisfies it.
type JobSource interface {
	ListCostRecords(ctx context.Context, limit int) ([]internaljobs.Job, error)
	ListActive(ctx context.Context) ([]internaljobs.Job, error)
	ListRouteRefusals(ctx context.Context, limit int) ([]internaljobs.Job, error)
}

// RouteState is the routing record pinned into a job payload. It is what the
// scheduler reads to decide a re-route and what the result meta reports, so a
// route is never hidden from the caller.
type RouteState struct {
	// Choice is where the job runs now: local or cloud.
	Choice string `json:"choice"`
	// Reason names why the cloud was chosen (empty while the job is local).
	Reason string `json:"reason,omitempty"`
	// Fallback arms the wait-budget re-route for a local job.
	Fallback          bool    `json:"fallback,omitempty"`
	WaitBudgetSeconds int     `json:"wait_budget_seconds,omitempty"`
	MaxCostUSD        float64 `json:"max_cost_usd,omitempty"`
	EstimatedCostUSD  float64 `json:"estimated_cost_usd,omitempty"`
	Role              string  `json:"role,omitempty"`
	RoleReason        string  `json:"role_reason,omitempty"`
	// Local* record the original local choice when the job moved to the cloud.
	LocalModel    string `json:"local_model,omitempty"`
	LocalProvider string `json:"local_provider,omitempty"`
	LocalTier     string `json:"local_tier,omitempty"`
	Rerouted      bool   `json:"rerouted,omitempty"`
	WaitedSeconds int    `json:"waited_seconds,omitempty"`
	// LocalPayload is the job's payload before it moved to the cloud, kept so a
	// refused cloud attempt can return the job to exactly its local route.
	LocalPayload          json.RawMessage `json:"local_payload,omitempty"`
	LocalEstimatedSeconds int             `json:"local_estimated_seconds,omitempty"`
	// CloudRefusedAt and CloudRefusal record a cloud route refused before
	// billing (RFC 3339 time and the provider's reason).
	CloudRefusedAt string `json:"cloud_refused_at,omitempty"`
	CloudRefusal   string `json:"cloud_refusal,omitempty"`
}

// cloudEligibleOp reports whether the gateway backend serves op.
func cloudEligibleOp(op string) bool {
	switch op {
	case "text_to_image", "image_to_image", "edit_instruct", "face_swap", "colorize":
		return true
	}
	return false
}

// stylizedPrompt matches instructions that restyle the image (a change of
// medium or look) rather than edit something inside it.
var stylizedPrompt = regexp.MustCompile(`(?i)\b(anime|manga|cartoon\w*|comic\w*|illustrat\w*|cel[- ]?shad\w*|watercolou?r\w*|oil[- ]paint\w*|painting|painterly|sketch\w*|line[- ]?art|pixel[- ]?art|vector|flat colou?rs?|low[- ]poly|3d render\w*|claymation|ghibli|pop[- ]art|impressionis\w*|stylised|stylized|caricature|chibi|woodcut|ukiyo\W?e|concept art|digital art|in the style of|art style)\b`)

// ChooseCloudRole picks the gateway role for a cloud route and says why.
//
//   - A role the caller named wins.
//   - Generation: image.generate.quality under quality_policy=quality, else
//     image.generate.default.
//   - Edits: an explicit edit_style decides; otherwise a prompt that restyles
//     the image (anime, illustration, painting, ...) is stylised and goes to
//     image.edit.default, and anything else is an identity-preserving edit and
//     goes to image.edit.identity. With no signal the identity role is used:
//     it keeps the subject recognisable, which is the harder thing to repair.
func ChooseCloudRole(op, callerRole, editStyle, qualityPolicy, prompt string) (role, reason string) {
	if r := strings.TrimSpace(callerRole); r != "" {
		return r, "caller named the role"
	}
	if op == "text_to_image" {
		if qualityPolicy == "quality" {
			return RoleGenerateQuality, "quality policy asked for the quality generator"
		}
		return RoleGenerateDefault, "default generator"
	}
	switch strings.ToLower(strings.TrimSpace(editStyle)) {
	case EditStyleIdentity:
		return RoleEditIdentity, "edit_style=identity"
	case EditStyleStylized, "stylised":
		return RoleEditDefault, "edit_style=stylized"
	}
	if m := stylizedPrompt.FindString(prompt); m != "" {
		return RoleEditDefault, fmt.Sprintf("prompt restyles the image (%q)", strings.ToLower(m))
	}
	return RoleEditIdentity, "no restyle in the prompt; keeping the subject's identity"
}

func (e *Engine) now() time.Time {
	if e.deps.Now != nil {
		return e.deps.Now()
	}
	return time.Now()
}

// cloudSpend sums what the cloud has cost or is committed to cost inside the
// rolling window: recorded cost_usd of finished jobs plus the pinned estimate of
// every cloud job still in flight that was created before ref (so two jobs
// admitted at once cannot both spend the last dollar, and neither starves the
// other). Both reads come from the durable job table, so a restart cannot reset
// the day.
func (e *Engine) cloudSpend(ctx context.Context, self string, ref time.Time) (recorded, committed float64, err error) {
	if e.deps.Jobs == nil {
		return 0, 0, errors.New("the job history that records cloud cost is not wired")
	}
	cutoff := e.now().Add(-routeLedgerWindow)
	finished, err := e.deps.Jobs.ListCostRecords(ctx, routeLedgerRowLimit)
	if err != nil {
		return 0, 0, fmt.Errorf("read recorded cloud cost: %w", err)
	}
	for _, job := range finished {
		if job.ID == self || job.FinishedAt == nil || job.FinishedAt.Before(cutoff) {
			continue
		}
		recorded += recordedJobCost(job)
	}
	active, err := e.deps.Jobs.ListActive(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("read in-flight cloud jobs: %w", err)
	}
	for _, job := range active {
		if job.ID == self || job.CreatedAt.After(ref) {
			continue
		}
		var pl Payload
		if json.Unmarshal(job.Payload, &pl) != nil || pl.Tier != backends.TierBYOK.String() {
			continue
		}
		est := e.deps.Route.DefaultEstimateUSD
		if pl.Route != nil && pl.Route.EstimatedCostUSD > 0 {
			est = pl.Route.EstimatedCostUSD
		}
		committed += est
	}
	return recorded, committed, nil
}

// recordedJobCost is the cost a finished job recorded: its total on success, or
// the known per-attempt costs in the gateway evidence of a failed one.
func recordedJobCost(job internaljobs.Job) float64 {
	if job.State == internaljobs.StateSucceeded {
		cost, _ := aigateway.ParseCostUSD(job.Meta)
		return cost
	}
	raw := job.Meta[internaljobs.MetaGatewayExecutionEvidence]
	if raw == "" {
		return 0
	}
	var evidence aigateway.Evidence
	if json.Unmarshal([]byte(raw), &evidence) != nil {
		return 0
	}
	total := 0.0
	for _, v := range evidence.Variations {
		if v.CostUSD != nil && *v.CostUSD > 0 && !math.IsNaN(*v.CostUSD) && !math.IsInf(*v.CostUSD, 0) {
			total += *v.CostUSD
		}
	}
	return total
}

// checkSpend returns an ErrSpendCap-wrapped refusal when estimate cannot be
// afforded. maxJob < 0 skips the per-job check. It fails closed when the ledger
// cannot be read: an unverifiable cap is not a cap.
func (e *Engine) checkSpend(ctx context.Context, self string, ref time.Time, estimate, maxJob float64) error {
	cfg := e.deps.Route
	if estimate > maxJob+1e-9 {
		return fmt.Errorf("%w: estimated cost $%.2f exceeds the per-job cap $%.2f (raise it with --max-cost)", ErrSpendCap, estimate, maxJob)
	}
	recorded, committed, err := e.cloudSpend(ctx, self, ref)
	if err != nil {
		return fmt.Errorf("%w: cannot verify the daily cap: %v", ErrSpendCap, err)
	}
	if recorded+committed+estimate > cfg.MaxDailyUSD+1e-9 {
		return fmt.Errorf("%w: $%.2f recorded + $%.2f committed + $%.2f estimated exceeds the rolling daily cap $%.2f (%s)", ErrSpendCap, recorded, committed, estimate, cfg.MaxDailyUSD, EnvCloudMaxDailyUSD)
	}
	return nil
}

// routeLimits resolves the per-request knobs against the deployment defaults.
func (e *Engine) routeLimits(req PlanRequest) (waitBudget time.Duration, maxJob float64) {
	cfg := e.deps.Route
	waitBudget, maxJob = cfg.WaitBudget, cfg.MaxJobUSD
	if req.MaxWaitSeconds != nil {
		waitBudget = time.Duration(max(*req.MaxWaitSeconds, 0)) * time.Second
	}
	if req.MaxCostUSD != nil && *req.MaxCostUSD >= 0 {
		maxJob = *req.MaxCostUSD
	}
	return waitBudget, maxJob
}

// cloudPermitted reports whether the cloud tier may be used for this request.
// The caller's fallback_policy=local_only always wins; an explicit allow or
// policy permits it; otherwise the deployment default decides.
func (e *Engine) cloudPermitted(req PlanRequest) bool {
	switch strings.ToLower(strings.TrimSpace(req.FallbackPolicy)) {
	case "local_only":
		return false
	case "cloud_allowed", "any":
		return true
	}
	return req.AllowBYOK || e.deps.Route.Enabled
}

// RouteConsentFunc reports whether a request of op at consent weight
// (none|low|high), resolved to a route tier, needs a consent affirmation, and
// says why. It is the safety gate's rule handed to the engine as a function so
// routing never moves a request off this machine under a rule the submit edge
// would refuse. nil means no rule applies.
//
// faces counts the people in the request's input; a low-weight rule asks for
// consent only when it finds some, and keeps the requirement when it is nil or
// fails.
type RouteConsentFunc func(ctx context.Context, op, weight, tier string, faces func(context.Context) (int, error)) (required bool, reason string)

// MetaConsentBasis is the job meta key that says why consent was or was not
// needed: faces_detected=N, detector_unavailable, not_required_local, ...
const MetaConsentBasis = "consent_basis"

// ErrConsentRequired is returned when a route that leaves this machine needs the
// caller's consent affirmation and the request does not carry one.
var ErrConsentRequired = internaljobs.Kinded(internaljobs.FailureRefused, errors.New("ai: consent affirmation required for this route"))

// consentBlocks reports whether consent is missing for the cloud route of op at
// weight, with the rule's reason. cleared is true when the caller affirmed or the
// submit edge already found nobody in the input; faces lets a rule that depends on
// the input look at it now.
func (e *Engine) consentBlocks(ctx context.Context, op, weight string, cleared bool, faces func(context.Context) (int, error)) (bool, string) {
	if e.deps.RouteConsent == nil || cleared {
		return false, ""
	}
	return e.deps.RouteConsent(ctx, op, weight, backends.TierBYOK.String(), faces)
}

// demandsCloud reports whether the caller asked for the cloud (--byok, or a
// named role) rather than leaving the choice to the routing policy. Such a
// caller is never held back by the cooldown: they get the cloud, or the honest
// error from it.
func demandsCloud(req PlanRequest) bool {
	return req.AllowBYOK || strings.TrimSpace(req.Role) != ""
}

// routePlan is the outcome of planRoute.
type routePlan struct {
	res      resolver.Resolution
	state    *RouteState
	warnings []string
	params   map[string]string
	// allowBYOK is true when the pinned provider is the cloud, which admission
	// must be allowed to revalidate.
	allowBYOK bool
}

// planRoute applies the cloud routing policy to the resolver's local-first
// answer. base is the request that produced res/resolveErr.
func (e *Engine) planRoute(ctx context.Context, req PlanRequest, base resolver.Request, res resolver.Resolution, resolveErr error, pinned bool) (routePlan, error) {
	out := routePlan{res: res}
	cfg := e.deps.Route
	if !cfg.Active {
		return out, resolveErr
	}
	waitBudget, maxJob := e.routeLimits(req)
	role, roleReason := ChooseCloudRole(req.Operation, req.Role, req.EditStyle, req.QualityPolicy, req.Prompt)
	if models.IsVisionOperation(req.Operation) && res.Model.Runtime.GatewayRole != "" {
		// A vision row names its own gateway role; the image roles above are for
		// image generation and editing.
		role, roleReason = res.Model.Runtime.GatewayRole, "the model row's gateway role"
	}
	estimate := cfg.estimate(role, req.Variations)
	cloudNow := resolveErr == nil && res.Tier == backends.TierBYOK.String()
	forcedLocal := req.RequireGPU || req.ForceCPU || len(req.Adapters) > 0

	state := &RouteState{Choice: routeChoiceLocal, MaxCostUSD: maxJob, EstimatedCostUSD: estimate, Role: role, RoleReason: roleReason}

	switch {
	case cloudNow:
		// The caller (or a pinned model) already selected the cloud: it is still
		// bounded by the caps.
		if err := e.checkSpend(ctx, "", e.now(), estimate, maxJob); err != nil {
			return out, err
		}
		state.Choice, state.Reason = routeChoiceCloud, routeReasonRequested
		out.state, out.allowBYOK = state, true
		out.params = map[string]string{"openrouter_role": role}
		return out, nil

	case resolveErr != nil:
		// Nothing local runs here. The cloud is the answer only when it is
		// eligible, permitted and affordable; otherwise the local refusal stands.
		if pinned || forcedLocal || !cloudEligibleOp(req.Operation) || !e.cloudPermitted(req) {
			return out, resolveErr
		}
		if !demandsCloud(req) {
			if err := e.checkCooldown(ctx); err != nil {
				return out, fmt.Errorf("%w (cloud fallback declined: %v)", resolveErr, err)
			}
		}
		if err := e.checkSpend(ctx, "", e.now(), estimate, maxJob); err != nil {
			return out, fmt.Errorf("%w (cloud fallback declined: %v)", resolveErr, err)
		}
		cloud, err := e.resolveCloud(ctx, base)
		if err != nil {
			return out, fmt.Errorf("%w (cloud fallback unavailable: %v)", resolveErr, err)
		}
		if blocked, why := e.consentBlocks(ctx, req.Operation, cloud.Weight, req.ConsentAffirmed, req.InputFaces); blocked {
			return out, fmt.Errorf("%w (cloud fallback declined: %s)", resolveErr, why)
		}
		state.Choice, state.Reason = routeChoiceCloud, routeReasonNoLocal
		state.LocalModel, state.LocalTier = "", "none"
		out.res, out.state, out.allowBYOK = cloud, state, true
		out.params = map[string]string{"openrouter_role": role}
		out.warnings = append(out.warnings, fmt.Sprintf("no local model can run %s on this host (%v); routed to the cloud role %s (estimated $%.2f, cap $%.2f)", req.Operation, resolveErr, role, estimate, maxJob))
		return out, nil
	}

	// A local answer exists. Decide whether the cloud is better or merely armed.
	if !cloudEligibleOp(req.Operation) {
		return out, nil
	}
	// Past this point the job could have used the cloud, so the meta says why it
	// did not: staying local is a routing decision a reader may ask about.
	stay := func(reason string) (routePlan, error) {
		state.Reason = reason
		state.LocalModel, state.LocalProvider, state.LocalTier = res.Model.ID, res.Provider, res.Tier
		out.state = state
		return out, nil
	}
	switch {
	case pinned:
		return stay(routeStayPinned)
	case forcedLocal || strings.EqualFold(strings.TrimSpace(req.FallbackPolicy), "local_only"):
		return stay(routeStayLocalReq)
	case !e.cloudPermitted(req):
		return stay(routeStayNotAllow)
	}
	if blocked, why := e.consentBlocks(ctx, req.Operation, res.Weight, req.ConsentAffirmed, req.InputFaces); blocked {
		// An automatic cloud route is a choice the engine makes for the caller, so
		// it never sends an identity-altering request off this machine without the
		// caller's affirmation. Say so, and keep the local answer.
		out.warnings = append(out.warnings, "cloud route not used for this job: "+why+" (affirm consent to allow it)")
		return stay(routeStayConsent)
	}
	if !demandsCloud(req) {
		if err := e.checkCooldown(ctx); err != nil {
			out.warnings = append(out.warnings, "cloud route not available for this job: "+err.Error())
			return stay(routeStayCooldown)
		}
	}
	if err := e.checkSpend(ctx, "", e.now(), estimate, maxJob); err != nil {
		out.warnings = append(out.warnings, "cloud route not available for this job: "+strings.TrimPrefix(err.Error(), ErrSpendCap.Error()+": "))
		return stay(routeStayCap)
	}
	state.LocalModel, state.LocalProvider, state.LocalTier = res.Model.ID, res.Provider, res.Tier
	if res.Tier == backends.TierLocalCPU.String() {
		// No usable GPU for this model: the local answer is a weak CPU model.
		cloud, err := e.resolveCloud(ctx, base)
		if err != nil {
			out.warnings = append(out.warnings, "cloud route preferred over the CPU model but unavailable: "+err.Error())
		} else {
			state.Choice, state.Reason = routeChoiceCloud, routeReasonNoGPU
			out.res, out.state, out.allowBYOK = cloud, state, true
			out.params = map[string]string{"openrouter_role": role}
			out.warnings = append(out.warnings, fmt.Sprintf("this host has no usable GPU for %s; routed to the cloud role %s instead of the CPU model %q (estimated $%.2f, cap $%.2f; pass --fallback-policy local_only to stay local)", req.Operation, role, res.Model.ID, estimate, maxJob))
			return out, nil
		}
	}
	if waitBudget > 0 && len(res.Adapters) == 0 {
		state.Fallback = true
		state.WaitBudgetSeconds = int(math.Ceil(waitBudget.Seconds()))
		out.state = state
		return out, nil
	}
	if len(res.Adapters) > 0 {
		return stay(routeStayLocalReq)
	}
	return stay(routeStayNoWait)
}

// resolveCloud resolves the cloud tier for the request in base, or says why it
// cannot be used.
func (e *Engine) resolveCloud(ctx context.Context, base resolver.Request) (resolver.Resolution, error) {
	cloudReq := base
	cloudReq.AllowBYOK = true
	cloudReq.FallbackPolicy = "cloud_allowed"
	cloudReq.PreferRemote = true
	cloudReq.ModelOverride = ""
	cloudReq.ForceCPU = false
	cloudReq.Adapters = nil
	res, err := resolver.New(e.effectiveRegistry(ctx), e.deps.Backends, e.deps.RightsEvaluator).Resolve(ctx, cloudReq)
	if err != nil {
		return resolver.Resolution{}, err
	}
	if res.Tier != backends.TierBYOK.String() {
		return resolver.Resolution{}, fmt.Errorf("the cloud tier did not resolve (got %s)", res.Tier)
	}
	return res, nil
}

// RerouteToCloud is the jobs scheduler's Rerouter. It moves a job that is still
// queued locally to the cloud once it has waited past its budget and the spend
// caps allow it. It returns nil to leave the job alone and an error when it
// wanted to move the job but could not.
//
// It only reads. The scheduler applies the result only if the job is still
// queued, so a job is never run twice and the cloud call is made once.
func (e *Engine) RerouteToCloud(ctx context.Context, job internaljobs.Job) (*internaljobs.Reroute, error) {
	if !e.deps.Route.Active || len(job.Payload) == 0 {
		return nil, nil
	}
	var pl Payload
	if err := json.Unmarshal(job.Payload, &pl); err != nil {
		return nil, nil
	}
	state := pl.Route
	if state == nil || !state.Fallback || state.Rerouted || state.WaitBudgetSeconds <= 0 || pl.Tier == backends.TierBYOK.String() {
		return nil, nil
	}
	waited := e.now().Sub(job.CreatedAt)
	if waited < time.Duration(state.WaitBudgetSeconds)*time.Second {
		return nil, nil
	}
	if blocked, _ := e.consentBlocks(ctx, pl.Operation, string(capability.ConsentWeightFor(pl.Operation)), pl.ConsentAffirmed || pl.ConsentWaived, nil); blocked {
		return nil, nil
	}
	if err := e.checkCooldown(ctx); err != nil {
		return nil, err
	}
	if err := e.checkSpend(ctx, job.ID, job.CreatedAt, state.EstimatedCostUSD, state.MaxCostUSD); err != nil {
		return nil, err
	}
	base, err := e.resolverRequest(ctx, PlanRequest{Operation: pl.Operation, AllowBYOK: true, FallbackPolicy: "cloud_allowed", Role: state.Role})
	if err != nil {
		return nil, err
	}
	cloud, err := e.resolveCloud(ctx, base)
	if err != nil {
		return nil, err
	}
	moved := *state
	moved.Choice, moved.Reason, moved.Rerouted, moved.Fallback = routeChoiceCloud, routeReasonWait, true, false
	moved.WaitedSeconds = int(waited.Seconds())
	// Keep the local route so a cloud refusal can return the job to it.
	moved.LocalPayload, moved.LocalEstimatedSeconds = append(json.RawMessage(nil), job.Payload...), job.EstimatedSeconds
	pl.ModelID, pl.Backend, pl.Tier, pl.GPU, pl.AllowBYOK = cloud.Model.ID, cloud.Provider, cloud.Tier, false, true
	pl.AutoAcquire, pl.GPUUUID, pl.RequireGPU = false, "", false
	pl.Params = cloneParams(pl.Params)
	if pl.Params["openrouter_role"] == "" {
		pl.Params["openrouter_role"] = moved.Role
	}
	pl.Route = &moved
	raw, err := json.Marshal(pl)
	if err != nil {
		return nil, err
	}
	return &internaljobs.Reroute{
		Payload:          raw,
		Lane:             LaneForTier(pl.Tier),
		EstimatedSeconds: routeCloudEstimateS,
		Message: fmt.Sprintf("re-routed to the cloud (%s): waited %ds for local %s (%s) against a %ds budget; estimated $%.2f, cap $%.2f",
			moved.Role, moved.WaitedSeconds, moved.LocalModel, moved.LocalTier, state.WaitBudgetSeconds, moved.EstimatedCostUSD, moved.MaxCostUSD),
	}, nil
}

// admitCloudSpend is the last check before a cloud call can be billed. It
// repeats the cap check at admission because the job may have queued for a long
// time since the plan was made.
//
// A job that was re-routed here automatically (not demanded by the caller) is
// returned to its local route instead of failing when the cloud is paused or the
// caps no longer allow the call; an explicit cloud job gets the honest error.
func (e *Engine) admitCloudSpend(ctx context.Context, job internaljobs.Job, pl Payload) error {
	if !e.deps.Route.Active {
		return nil
	}
	if pl.Route != nil && pl.Route.Rerouted {
		if err := e.checkCooldown(ctx); err != nil {
			if back := e.fellBackLocal(pl, "cloud_cooldown", err); back != nil {
				return back
			}
			return err
		}
	}
	estimate, maxJob := e.deps.Route.estimate(pl.Params["openrouter_role"], pl.Variations), e.deps.Route.MaxJobUSD
	if pl.Route != nil {
		maxJob = pl.Route.MaxCostUSD
		if pl.Route.EstimatedCostUSD > 0 {
			// Pinned at plan time, already multiplied by the variation count.
			estimate = pl.Route.EstimatedCostUSD
		}
	}
	err := e.checkSpend(ctx, job.ID, job.CreatedAt, estimate, maxJob)
	if err != nil && pl.Route != nil && pl.Route.Rerouted {
		if back := e.fellBackLocal(pl, "spend_cap", err); back != nil {
			return back
		}
	}
	return err
}

// ErrCloudCooldown is returned while automatic cloud routing is paused after a
// cloud route was refused before billing.
var ErrCloudCooldown = internaljobs.Kinded(internaljobs.FailureRefused, errors.New("ai: cloud routing is paused"))

const routeRefusalRowLimit = 50

// cloudCooldown reports whether automatic cloud routing is paused, and until
// when. The pause is derived from jobs whose meta records a refused cloud route
// (MetaCloudRefusedAt), so it is durable, shared by every waiting job and
// survives a restart. An unreadable history is treated as paused: the caller
// then stays local, which costs a wait rather than a refusal.
func (e *Engine) cloudCooldown(ctx context.Context) (until time.Time, active bool, err error) {
	cooldown := e.deps.Route.Cooldown
	if !e.deps.Route.Active || cooldown <= 0 {
		return time.Time{}, false, nil
	}
	if e.deps.Jobs == nil {
		return time.Time{}, true, errors.New("the job history that records cloud refusals is not wired")
	}
	jobs, err := e.deps.Jobs.ListRouteRefusals(ctx, routeRefusalRowLimit)
	if err != nil {
		return time.Time{}, true, fmt.Errorf("read recorded cloud refusals: %w", err)
	}
	now := e.now()
	for _, job := range jobs {
		at, perr := time.Parse(time.RFC3339, job.Meta[MetaCloudRefusedAt])
		if perr != nil {
			continue
		}
		if end := at.Add(cooldown); end.After(now) && end.After(until) {
			until, active = end, true
		}
	}
	return until, active, nil
}

// checkCooldown returns an ErrCloudCooldown-wrapped refusal while the pause holds.
func (e *Engine) checkCooldown(ctx context.Context) error {
	until, active, err := e.cloudCooldown(ctx)
	switch {
	case err != nil:
		return fmt.Errorf("%w: %v", ErrCloudCooldown, err)
	case active:
		return fmt.Errorf("%w until %s after a cloud route was refused (%s)", ErrCloudCooldown, until.UTC().Format("15:04:05Z"), EnvCloudCooldown)
	}
	return nil
}

var (
	refusalCode   = regexp.MustCompile(`"code"\s*:\s*"([a-z0-9_.-]+)"`)
	refusalStatus = regexp.MustCompile(`"http_status"\s*:\s*(\d{3})`)
	refusalDial   = regexp.MustCompile(`(?i)connection refused|no such host|dial tcp|service unavailable|http 503`)
)

// cloudRefusal reports whether err is a cloud provider refusing the request
// before it did, or could bill, any work: out of credits, bad or missing
// credentials, the provider or gateway unreachable or rate limiting. Anything
// else (a failed generation, a timeout, an unknown error) may have been billed
// and stays an honest failure. It returns the provider's code or status.
func cloudRefusal(err error) (string, bool) {
	if err == nil {
		return "", false
	}
	text := err.Error()
	if m := refusalCode.FindStringSubmatch(text); m != nil {
		switch m[1] {
		case "insufficient_credits", "payment_required", "unauthorized", "forbidden", "invalid_api_key", "missing_api_key",
			"authentication_failed", "missing_credentials", "provider_unavailable", "rate_limited", "quota_exceeded":
			return m[1], true
		}
	}
	if m := refusalStatus.FindStringSubmatch(text); m != nil {
		switch m[1] {
		case "401", "402", "403", "429", "503":
			return "http_" + m[1], true
		}
	}
	if refusalDial.MatchString(text) {
		return "provider_unreachable", true
	}
	return "", false
}

// refusalMeta is the record of a refused cloud route.
func (e *Engine) refusalMeta(code string, cause error) map[string]string {
	text := strings.Join(strings.Fields(cause.Error()), " ")
	if len(text) > 300 {
		text = text[:300] + "..."
	}
	return map[string]string{MetaCloudRefusedAt: e.now().UTC().Format(time.RFC3339), MetaCloudRefusal: code + ": " + text}
}

// fellBackLocal builds the RouteBackError that returns a job re-routed to the
// cloud to exactly the local route it left, recording why. It reports false when
// the job has no local route to return to (the caller asked for the cloud).
func (e *Engine) fellBackLocal(pl Payload, code string, cause error) error {
	if pl.Route == nil || !pl.Route.Rerouted || len(pl.Route.LocalPayload) == 0 {
		return nil
	}
	var local Payload
	if err := json.Unmarshal(pl.Route.LocalPayload, &local); err != nil || local.Tier == backends.TierBYOK.String() {
		return nil
	}
	refusal := e.refusalMeta(code, cause)
	state := pl.Route
	if local.Route != nil {
		restored := *local.Route
		state = &restored
	} else {
		restored := *pl.Route
		state = &restored
	}
	state.Choice, state.Reason, state.Fallback, state.Rerouted = routeChoiceLocal, routeReasonFellBack, false, true
	state.WaitedSeconds = pl.Route.WaitedSeconds
	state.LocalPayload, state.LocalEstimatedSeconds = nil, 0
	state.CloudRefusedAt, state.CloudRefusal = refusal[MetaCloudRefusedAt], refusal[MetaCloudRefusal]
	local.Route = state
	raw, err := json.Marshal(local)
	if err != nil {
		return nil
	}
	meta := routeMeta(state, e.deps.Route)
	return &internaljobs.RouteBackError{
		Move: internaljobs.Reroute{
			Payload:          raw,
			Lane:             LaneForTier(local.Tier),
			EstimatedSeconds: pl.Route.LocalEstimatedSeconds,
			Message:          fmt.Sprintf("cloud route refused (%s); back on the local route %s (%s)", code, state.LocalModel, state.LocalTier),
			Meta:             meta,
		},
		Cause: cause,
	}
}

// routeMeta is the routing record a job reports in its meta. Every key is
// prefixed route_, so a caller can tell where a job ran, why, what it replaced
// and what it was allowed to spend; cost_usd is reported separately by the
// backend.
func routeMeta(state *RouteState, cfg RouteConfig) map[string]string {
	if state == nil {
		return nil
	}
	m := map[string]string{"route_choice": state.Choice}
	set := func(key, value string) {
		if value != "" {
			m[key] = value
		}
	}
	set("route_reason", state.Reason)
	if state.Rerouted {
		m["route_rerouted"] = "true"
		m["route_waited_seconds"] = strconv.Itoa(state.WaitedSeconds)
	}
	if state.Choice == routeChoiceLocal && state.Fallback {
		m["route_fallback"] = "armed"
	}
	if state.WaitBudgetSeconds > 0 {
		m["route_wait_budget_seconds"] = strconv.Itoa(state.WaitBudgetSeconds)
	}
	set(MetaCloudRefusedAt, state.CloudRefusedAt)
	set(MetaCloudRefusal, state.CloudRefusal)
	set("route_local_model", state.LocalModel)
	set("route_local_provider", state.LocalProvider)
	set("route_local_tier", state.LocalTier)
	set("route_role", state.Role)
	set("route_role_reason", state.RoleReason)
	m["route_max_cost_usd"] = strconv.FormatFloat(state.MaxCostUSD, 'f', -1, 64)
	if state.EstimatedCostUSD > 0 {
		m["route_estimated_cost_usd"] = strconv.FormatFloat(state.EstimatedCostUSD, 'f', -1, 64)
	}
	if cfg.Active {
		m["route_daily_cap_usd"] = strconv.FormatFloat(cfg.MaxDailyUSD, 'f', -1, 64)
	}
	return m
}

func cloneParams(in map[string]string) map[string]string {
	out := make(map[string]string, len(in)+1)
	for k, v := range in {
		out[k] = v
	}
	return out
}
