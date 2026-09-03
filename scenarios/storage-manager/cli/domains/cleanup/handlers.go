package cleanup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	cleanupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/cleanup"
	cleanupconnect "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/cleanup/cleanup_v1connect"
)

type handlers struct {
	client cleanupconnect.CleanupServiceClient
}

type rootInfo struct {
	ID           string `json:"id"`
	DeclaredRoot string `json:"declared_root"`
	ResolvedRoot string `json:"resolved_root"`
	Tier         string `json:"tier"`
	MaxAge       string `json:"max_age,omitempty"`
	MaxBytes     string `json:"max_bytes,omitempty"`
	Applicable   bool   `json:"applicable"`
	Present      bool   `json:"present"`
}

func (h *handlers) rootsCall(cliapp.OperationContext) ([]rootInfo, error) {
	repoRoot := strings.TrimSpace(os.Getenv("VROOLI_ROOT"))
	if repoRoot == "" {
		repoRoot, _ = os.Getwd()
	}
	for {
		if _, err := os.Stat(filepath.Join(repoRoot, ".vrooli", "repo-contract.json")); err == nil {
			break
		}
		parent := filepath.Dir(repoRoot)
		if parent == repoRoot {
			return nil, fmt.Errorf("cannot locate .vrooli/repo-contract.json from %s", repoRoot)
		}
		repoRoot = parent
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		return nil, err
	}
	var doc struct {
		Storage struct {
			Roots []struct {
				ID        string   `json:"id"`
				Root      string   `json:"root"`
				Tier      string   `json:"tier"`
				MaxAge    string   `json:"max_age"`
				MaxBytes  string   `json:"max_bytes"`
				Platforms []string `json:"platforms"`
			} `json:"roots"`
		} `json:"storage"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	home, _ := os.UserHomeDir()
	vrooliHome := filepath.Join(home, ".vrooli")
	platform := runtime.GOOS
	if platform == "darwin" {
		platform = "macos"
	}
	result := make([]rootInfo, 0, len(doc.Storage.Roots))
	for _, root := range doc.Storage.Roots {
		applicable := false
		for _, candidate := range root.Platforms {
			if candidate == platform {
				applicable = true
				break
			}
		}
		resolved := strings.ReplaceAll(root.Root, "$USER_HOME", home)
		resolved = strings.ReplaceAll(resolved, "$VROOLI_HOME", vrooliHome)
		resolved = strings.ReplaceAll(resolved, "$TMPDIR", os.TempDir())
		if strings.HasPrefix(resolved, "~/") {
			resolved = filepath.Join(home, strings.TrimPrefix(resolved, "~/"))
		}
		resolved = filepath.Clean(resolved)
		_, statErr := os.Stat(resolved)
		result = append(result, rootInfo{ID: root.ID, DeclaredRoot: root.Root, ResolvedRoot: resolved, Tier: root.Tier, MaxAge: root.MaxAge, MaxBytes: root.MaxBytes, Applicable: applicable, Present: statErr == nil})
	}
	return result, nil
}

func rootsReport(_ cliapp.OperationContext, roots []rootInfo) cliapp.MutationReport {
	rows := make([]string, 0, len(roots))
	for _, root := range roots {
		rows = append(rows, fmt.Sprintf("%s %s tier=%s applicable=%t present=%t", root.ID, root.ResolvedRoot, root.Tier, root.Applicable, root.Present))
	}
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Found %d governed root(s).", len(rows))}, Changes: rows}
}

type tierInfo struct {
	Tier          string `json:"tier"`
	Autonomous    bool   `json:"autonomous"`
	ProofRequired bool   `json:"proof_required"`
}

func (h *handlers) tiersCall(cliapp.OperationContext) ([]tierInfo, error) {
	return []tierInfo{
		{Tier: "safe", Autonomous: true},
		{Tier: "regenerable", Autonomous: true, ProofRequired: true},
		{Tier: "safe_with_owner", ProofRequired: false},
		{Tier: "conditional", ProofRequired: false},
		{Tier: "forbidden", ProofRequired: false},
	}, nil
}

func tiersReport(_ cliapp.OperationContext, tiers []tierInfo) cliapp.MutationReport {
	rows := make([]string, 0, len(tiers))
	for _, tier := range tiers {
		rows = append(rows, fmt.Sprintf("%s autonomous=%t proof_required=%t", tier.Tier, tier.Autonomous, tier.ProofRequired))
	}
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Found %d recovery safety tier(s).", len(rows))}, Changes: rows}
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: cleanupconnect.NewCleanupServiceClient(httpClient, baseURL)}
}

func (h *handlers) listProvidersCall(cliapp.OperationContext) (*cleanupv1.ListProvidersResponse, error) {
	resp, err := h.client.ListProviders(context.Background(), connect.NewRequest(&cleanupv1.ListProvidersRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list cleanup providers", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) listProvidersReport(_ cliapp.OperationContext, msg *cleanupv1.ListProvidersResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.GetProviders()))
	for _, provider := range msg.GetProviders() {
		results = append(results, fmt.Sprintf("%s — %s [%s, default=%s/%s]", provider.GetId(), provider.GetName(), provider.GetSafetyTier(), provider.GetDefaultMode(), provider.GetDefaultApproval()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d cleanup provider(s).", len(results))}, ResultsHeading: "Providers", Results: results}
}

func (h *handlers) getPolicyCall(cliapp.OperationContext) (*cleanupv1.GetPolicyResponse, error) {
	resp, err := h.client.GetPolicy(context.Background(), connect.NewRequest(&cleanupv1.GetPolicyRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get cleanup policy", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) getPolicyReport(_ cliapp.OperationContext, msg *cleanupv1.GetPolicyResponse) cliapp.ListReport {
	pol := msg.GetPolicy()
	results := make([]string, 0, len(pol.GetProviders()))
	for _, provider := range pol.GetProviders() {
		results = append(results, fmt.Sprintf("%s enabled=%t approval=%s min_age=%ds max_bytes=%d", provider.GetProviderId(), provider.GetEnabled(), provider.GetApprovalMode(), provider.GetMinAgeSeconds(), provider.GetMaxBytes()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Policy %s (%s).", pol.GetVersion(), pol.GetProfile())}, ResultsHeading: "Provider policy", Results: results}
}

func (h *handlers) setPolicyProfileCall(ctx cliapp.OperationContext) (*cleanupv1.SetPolicyProfileResponse, error) {
	resp, err := h.client.SetPolicyProfile(context.Background(), connect.NewRequest(&cleanupv1.SetPolicyProfileRequest{Profile: ctx.Flag("profile")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("set cleanup policy profile", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) setPolicyProfileReport(_ cliapp.OperationContext, msg *cleanupv1.SetPolicyProfileResponse) cliapp.MutationReport {
	pol := msg.GetPolicy()
	return cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Policy set to %s.", pol.GetProfile())},
		Changes: []string{
			fmt.Sprintf("version=%s providers=%d", pol.GetVersion(), len(pol.GetProviders())),
		},
		NextCommand: []string{"`cleanup policy` — inspect provider gates", "`cleanup plan` — preview reclaimable data"},
	}
}

func (h *handlers) createPlanCall(cliapp.OperationContext) (*cleanupv1.CreatePlanResponse, error) {
	resp, err := h.client.CreatePlan(context.Background(), connect.NewRequest(&cleanupv1.CreatePlanRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create cleanup plan", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) createPlanReport(_ cliapp.OperationContext, msg *cleanupv1.CreatePlanResponse) cliapp.OperationalReport {
	plan := msg.GetPlan()
	results := make([]string, 0, len(plan.GetProviders()))
	for _, provider := range plan.GetProviders() {
		row := fmt.Sprintf("%s %d bytes %d item(s) blocked=%q approval=%s", provider.GetProviderId(), provider.GetEstimatedBytes(), provider.GetItemCount(), provider.GetBlockedReason(), provider.GetApprovalMode())
		for _, warning := range provider.GetWarnings() {
			row += "\n  warning: " + warning
		}
		results = append(results, row)
	}
	return cliapp.OperationalReport{
		Status:    []string{fmt.Sprintf("Plan %s estimates %d bytes across %d item(s); census=%s (%s).", plan.GetId(), plan.GetTotalBytes(), plan.GetTotalItems(), plan.GetCensusId(), plan.GetCensusStatus())},
		Triage:    []cliapp.TriageGroup{{Heading: "Providers", Items: results}},
		NextSteps: []string{fmt.Sprintf("`cleanup apply --plan-id %s --policy-version %s --idempotency-key <key> --approval-mode operator --approval-token <token>`", plan.GetId(), plan.GetPolicyVersion())},
	}
}

func (h *handlers) applyPlanCall(ctx cliapp.OperationContext) (*cleanupv1.ApplyPlanResponse, error) {
	resp, err := h.client.ApplyPlan(context.Background(), connect.NewRequest(&cleanupv1.ApplyPlanRequest{
		PlanId:         ctx.Flag("plan-id"),
		PolicyVersion:  ctx.Flag("policy-version"),
		ApprovalMode:   ctx.Flag("approval-mode"),
		ApprovalToken:  ctx.Flag("approval-token"),
		IdempotencyKey: ctx.Flag("idempotency-key"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("apply cleanup plan", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) applyPlanReport(_ cliapp.OperationContext, msg *cleanupv1.ApplyPlanResponse) cliapp.MutationReport {
	changes := make([]string, 0, len(msg.GetResults()))
	for _, result := range msg.GetResults() {
		changes = append(changes, fmt.Sprintf("%s applied=%t reclaimed=%d skipped=%d", result.GetProviderId(), result.GetApplied(), result.GetReclaimedBytes(), len(result.GetSkippedItems())))
	}
	return cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Apply %s reclaimed %d bytes (replay=%t).", msg.GetIdempotencyKey(), msg.GetReclaimedBytes(), msg.GetAlreadyApplied())},
		Changes:     changes,
		NextCommand: []string{"`cleanup audit` — inspect immutable apply history"},
	}
}

func (h *handlers) listAuditCall(cliapp.OperationContext) (*cleanupv1.ListAuditResponse, error) {
	resp, err := h.client.ListAudit(context.Background(), connect.NewRequest(&cleanupv1.ListAuditRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list cleanup audit", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) listAuditReport(_ cliapp.OperationContext, msg *cleanupv1.ListAuditResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.GetEvents()))
	for _, event := range msg.GetEvents() {
		results = append(results, fmt.Sprintf("%s %s plan=%s provider=%s redacted=%t %s", event.GetId(), event.GetType(), event.GetPlanId(), event.GetProviderId(), event.GetRedacted(), event.GetMessage()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d audit event(s).", len(results))}, ResultsHeading: "Audit events", Results: results}
}

// reportPressureCall reports disk pressure.
//
// The band is parsed here rather than passed through as a string so an
// operator typo fails locally with the valid options listed, instead of
// becoming a request the server has to interpret. Critical is the band that
// authorises deletion with no operator present, so it must never be reached by
// accident.
func (h *handlers) reportPressureCall(ctx cliapp.OperationContext) (*cleanupv1.ReportPressureResponse, error) {
	band, err := parsePressureBandFlag(ctx.Flag("band"))
	if err != nil {
		return nil, err
	}

	usedPercent, err := parseOptionalFloat(ctx.Flag("used-percent"), "used-percent")
	if err != nil {
		return nil, err
	}
	availableBytes, err := parseOptionalInt(ctx.Flag("available-bytes"), "available-bytes")
	if err != nil {
		return nil, err
	}

	source := ctx.Flag("source")
	if source == "" {
		source = "cli"
	}

	resp, err := h.client.ReportPressure(context.Background(), connect.NewRequest(&cleanupv1.ReportPressureRequest{
		SourceScenario: source,
		Partition:      ctx.Flag("partition"),
		UsedPercent:    usedPercent,
		Band:           band,
		AvailableBytes: availableBytes,
		Trigger:        cleanupv1.PressureTrigger_PRESSURE_TRIGGER_BAND,
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("report disk pressure", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) reportPressureReport(_ cliapp.OperationContext, msg *cleanupv1.ReportPressureResponse) cliapp.MutationReport {
	changes := make([]string, 0, 2)
	if applied := msg.GetProvidersApplied(); len(applied) > 0 {
		changes = append(changes, fmt.Sprintf("ran %s, reclaiming %d bytes", strings.Join(applied, ", "), msg.GetReclaimedBytes()))
	}
	// Withheld providers are reported, never silently dropped: an operator
	// needs to know what the safety tier refused to touch.
	if withheld := msg.GetProvidersWithheld(); len(withheld) > 0 {
		changes = append(changes, fmt.Sprintf("withheld above safe tier: %s", strings.Join(withheld, ", ")))
	}

	result := []string{fmt.Sprintf("Band %s: %s.", shortBandName(msg.GetBand()), shortActionName(msg.GetAction()))}
	if msg.GetPlanId() != "" {
		result = append(result, fmt.Sprintf("Plan %s estimated %d bytes reclaimable.", msg.GetPlanId(), msg.GetEstimatedBytes()))
	}
	if msg.GetReason() != "" {
		result = append(result, msg.GetReason())
	}
	if !msg.GetAutonomousApplyEnabled() {
		result = append(result, "Autonomous apply is disabled by the kill switch.")
	}

	return cliapp.MutationReport{
		Result:      result,
		Changes:     changes,
		NextCommand: []string{"`cleanup audit` — inspect what the pressure signal caused"},
	}
}

func (h *handlers) startRecoveryCall(ctx cliapp.OperationContext) (*cleanupv1.RecoveryRunResponse, error) {
	trigger := parsePressureTriggerFlag(ctx.Flag("trigger"))
	used, err := parseOptionalFloat(ctx.Flag("used-percent"), "used-percent")
	if err != nil {
		return nil, err
	}
	available, err := parseOptionalInt(ctx.Flag("available-bytes"), "available-bytes")
	if err != nil {
		return nil, err
	}
	target, err := parseOptionalFloat(ctx.Flag("target-percent"), "target-percent")
	if err != nil {
		return nil, err
	}
	resp, err := h.client.StartRecovery(context.Background(), connect.NewRequest(&cleanupv1.RecoveryRunRequest{Trigger: trigger, Partition: ctx.Flag("partition"), UsedPercent: used, AvailableBytes: available, TargetFreePercent: target, DryRun: ctx.BoolFlag("dry-run")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("start recovery", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) waitRecoveryCall(ctx cliapp.OperationContext) (*cleanupv1.RecoveryRunResponse, error) {
	resp, err := h.client.WaitRecovery(context.Background(), connect.NewRequest(&cleanupv1.RecoveryWaitRequest{RunId: ctx.Flag("run-id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("wait for recovery", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) listRecoveryCall(ctx cliapp.OperationContext) (*cleanupv1.RecoveryHistoryResponse, error) {
	limit := int32(0)
	if raw := ctx.Flag("limit"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("limit: %w", err)
		}
		limit = int32(parsed)
	}
	resp, err := h.client.ListRecovery(context.Background(), connect.NewRequest(&cleanupv1.RecoveryHistoryRequest{Limit: limit}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list recovery history", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) recoveryReport(_ cliapp.OperationContext, msg *cleanupv1.RecoveryRunResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Recovery %s started (%s).", msg.GetRunId(), msg.GetStatus())}, NextCommand: []string{"`cleanup wait --run-id " + msg.GetRunId() + "`"}}
}

func (h *handlers) recoveryRunReport(_ cliapp.OperationContext, msg *cleanupv1.RecoveryRunResponse) cliapp.OperationalReport {
	return cliapp.OperationalReport{Status: []string{fmt.Sprintf("Recovery %s is %s; action=%s reclaimed=%d reason=%s", msg.GetRunId(), msg.GetStatus(), msg.GetAction(), msg.GetReclaimedBytes(), msg.GetReason())}}
}

func (h *handlers) listRecoveryReport(_ cliapp.OperationContext, msg *cleanupv1.RecoveryHistoryResponse) cliapp.ListReport {
	rows := make([]string, 0, len(msg.GetRuns()))
	for _, run := range msg.GetRuns() {
		rows = append(rows, fmt.Sprintf("%s %s trigger=%s reclaimed=%d", run.GetRunId(), run.GetStatus(), run.GetTrigger(), run.GetReclaimedBytes()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d recovery run(s).", len(rows))}, ResultsHeading: "Recovery history", Results: rows}
}

// parsePressureBandFlag maps the operator-facing band name onto the proto enum.
func parsePressureBandFlag(raw string) (cleanupv1.PressureBand, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "warning":
		return cleanupv1.PressureBand_PRESSURE_BAND_WARNING, nil
	case "high":
		return cleanupv1.PressureBand_PRESSURE_BAND_HIGH, nil
	case "critical":
		return cleanupv1.PressureBand_PRESSURE_BAND_CRITICAL, nil
	default:
		return cleanupv1.PressureBand_PRESSURE_BAND_UNSPECIFIED,
			fmt.Errorf("unknown band %q: expected warning, high, or critical", raw)
	}
}

func parsePressureTriggerFlag(raw string) cleanupv1.PressureTrigger {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "floor":
		return cleanupv1.PressureTrigger_PRESSURE_TRIGGER_FLOOR
	case "rate":
		return cleanupv1.PressureTrigger_PRESSURE_TRIGGER_RATE
	case "manual":
		return cleanupv1.PressureTrigger_PRESSURE_TRIGGER_MANUAL
	default:
		return cleanupv1.PressureTrigger_PRESSURE_TRIGGER_BAND
	}
}

func parseOptionalFloat(raw, flag string) (float64, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("--%s must be a number: %w", flag, err)
	}
	return value, nil
}

func parseOptionalInt(raw, flag string) (int64, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("--%s must be an integer: %w", flag, err)
	}
	return value, nil
}

// shortBandName trims the proto enum prefix for human output.
func shortBandName(band cleanupv1.PressureBand) string {
	return strings.ToLower(strings.TrimPrefix(band.String(), "PRESSURE_BAND_"))
}

func shortActionName(action cleanupv1.PressureAction) string {
	return strings.ToLower(strings.TrimPrefix(action.String(), "PRESSURE_ACTION_"))
}
