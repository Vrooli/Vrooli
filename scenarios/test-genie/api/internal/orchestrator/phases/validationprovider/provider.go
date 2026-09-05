package validationprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"test-genie/internal/shared"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/maturity-go/assessment"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/audit"
	cartosharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/shared"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

type GateMode string

const (
	GateModeOff            GateMode = "off"
	GateModeHighConfidence GateMode = "high-confidence"
	GateModeAll            GateMode = "all"
)

type Provider struct {
	Phase            string
	ProviderScenario string
	FindingSource    architecturev1.FindingSource
	Emoji            string
	DetailCommand    string
	Optional         bool
	Timeout          time.Duration
	IncludeExecution bool
	CapabilitySubset []string
	Exclude          []string
	DeliveryMode     string
	GateEnvVar       string
	DefaultGateMode  GateMode
	// OnStarted receives the provider-owned child reference immediately after a
	// durable Start acknowledgement is accepted and before the parent waits.
	// Test Genie uses this to persist/reconcile the parent-child link; providers
	// never depend on this callback for their own lifecycle authority.
	OnStarted func(RunReference)
}

// RunReference is the durable provider child identity Test Genie persists in
// its parent run artifacts. It intentionally carries only transport-neutral
// lifecycle metadata; the provider ledger remains authoritative.
type RunReference struct {
	RunID       string
	ParentRunID string
	ETASeconds  int64
	State       string
}

type Summary struct {
	Scenario                  string              `json:"scenario"`
	Status                    string              `json:"status"`
	Blockers                  int                 `json:"blockers"`
	Errors                    int                 `json:"errors"`
	Warnings                  int                 `json:"warnings"`
	Infos                     int                 `json:"infos"`
	LocalCurrentLevel         string              `json:"local_current_level,omitempty"`
	LocalNextLevel            string              `json:"local_next_level,omitempty"`
	LocalClean                bool                `json:"local_clean"`
	LocalUnknownCount         int                 `json:"local_unknown_count,omitempty"`
	Capabilities              []CapabilitySummary `json:"capabilities,omitempty"`
	HighestPriorityCapability *PriorityFocus      `json:"highest_priority_capability,omitempty"`
	AuthorityConfidence       string              `json:"authority_confidence,omitempty"`
	GateMode                  string              `json:"gate_mode,omitempty"`
	GatedBlockers             int                 `json:"gated_blockers,omitempty"`
	Categories                []CategorySummary   `json:"categories,omitempty"`
	Skipped                   bool                `json:"skipped,omitempty"`
	ProviderRunID             string              `json:"provider_run_id,omitempty"`
	ProviderParentRunID       string              `json:"provider_parent_run_id,omitempty"`
	ProviderRunState          string              `json:"provider_run_state,omitempty"`
	ProviderETASeconds        int64               `json:"provider_eta_seconds,omitempty"`
}

type CategorySummary struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	Score float64 `json:"score"`
}

type CapabilitySummary struct {
	ID                   string `json:"id"`
	Label                string `json:"label"`
	CurrentLevel         string `json:"current_level,omitempty"`
	NextLevel            string `json:"next_level,omitempty"`
	CurrentSummary       string `json:"current_summary,omitempty"`
	NextUnlock           string `json:"next_unlock,omitempty"`
	Clean                bool   `json:"clean"`
	UnknownCount         int    `json:"unknown_count,omitempty"`
	BlockingFindingCount int    `json:"blocking_finding_count,omitempty"`
	PriorityRank         int    `json:"priority_rank,omitempty"`
	PriorityReason       string `json:"priority_reason,omitempty"`
}

type PriorityFocus struct {
	CapabilityID    string `json:"capability_id"`
	CapabilityLabel string `json:"capability_label,omitempty"`
	CurrentLevel    string `json:"current_level,omitempty"`
	NextLevel       string `json:"next_level,omitempty"`
	Reason          string `json:"reason,omitempty"`
}

func (s Summary) String() string {
	if s.Skipped {
		return "skipped"
	}
	text := fmt.Sprintf("%s status=%s blockers=%d errors=%d warnings=%d infos=%d", s.Scenario, s.Status, s.Blockers, s.Errors, s.Warnings, s.Infos)
	if s.LocalCurrentLevel != "" || s.LocalNextLevel != "" {
		text += fmt.Sprintf(" local=%s next=%s", s.LocalCurrentLevel, s.LocalNextLevel)
	}
	if s.LocalClean || s.LocalUnknownCount > 0 {
		text += fmt.Sprintf(" clean=%t unknown=%d", s.LocalClean, s.LocalUnknownCount)
	}
	if s.HighestPriorityCapability != nil && s.HighestPriorityCapability.CapabilityID != "" {
		text += fmt.Sprintf(" focus=%s", s.HighestPriorityCapability.CapabilityID)
		if s.HighestPriorityCapability.NextLevel != "" {
			text += "->" + s.HighestPriorityCapability.NextLevel
		}
	}
	if s.AuthorityConfidence != "" {
		text += fmt.Sprintf(" authority=%s", s.AuthorityConfidence)
	}
	if s.GateMode != "" {
		text += fmt.Sprintf(" gate=%s", s.GateMode)
	}
	return text
}

type Result struct {
	shared.RunResult[Summary]
	Findings []*architecturev1.ArchitectureFinding
	// Assessment is the provider-owned shared maturity response. Test Genie
	// transports it unchanged so descriptor-owned recommendations remain
	// available to evidence consumers alongside normalized findings.
	Assessment *commonv1.MaturityAssessment
	// Metrics is the provider's reported execution metrics (timing, stages,
	// resources, host environment), present only when the provider has adopted
	// the metrics contract. nil for un-migrated providers.
	Metrics *commonv1.ExecutionMetrics
	// Presentation is the provider-owned canonical phase presentation. Test
	// Genie transports it unchanged; it does not reconstruct phase semantics.
	Presentation *commonv1.PhasePresentation
	// FindingsSummary is the per-severity tally for this phase (non-nil whenever
	// an assessment was returned; all-zero for a clean phase).
	FindingsSummary *runspb.PhaseFindingsSummary
}

type Client interface {
	ValidateScenario(context.Context, *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error)
}

// TargetClient is the additive target-aware contract. Keeping it separate
// from Client preserves the test seam and the permanent legacy alias for
// providers that have not adopted ValidateTarget yet.
type TargetClient interface {
	ValidateTarget(context.Context, *connect.Request[scenariovalidationv1.ValidateTargetRequest]) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error)
}

type DurableClient interface {
	StartValidationRun(context.Context, *connect.Request[scenariovalidationv1.StartValidationRunRequest]) (*connect.Response[scenariovalidationv1.StartValidationRunResponse], error)
	WaitValidationRun(context.Context, *connect.Request[scenariovalidationv1.WaitValidationRunRequest]) (*connect.Response[scenariovalidationv1.WaitValidationRunResponse], error)
	AbortValidationRun(context.Context, *connect.Request[scenariovalidationv1.AbortValidationRunRequest]) (*connect.Response[scenariovalidationv1.AbortValidationRunResponse], error)
}

var (
	ResolveBaseURL = func(ctx context.Context, scenario string) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, scenario)
	}
	NewClient = func(timeout time.Duration, baseURL string) Client {
		return scenariovalidationconnect.NewScenarioValidationServiceClient(&http.Client{Timeout: timeout}, baseURL)
	}
	NewTargetClient = func(timeout time.Duration, baseURL string) TargetClient {
		return scenariovalidationconnect.NewScenarioValidationServiceClient(&http.Client{Timeout: timeout}, baseURL)
	}
	NewDurableClient = func(timeout time.Duration, baseURL string) DurableClient {
		return scenariovalidationconnect.NewDurableValidationRunServiceClient(&http.Client{Timeout: timeout}, baseURL)
	}
)

// Run invokes a provider's ValidateScenario RPC for targetScenario. scenarioPath
// is the resolved physical scenario directory (env.ScenarioDir); it is sent as
// ValidateScenarioRequest.path so providers can validate a scenario that does not
// live under the repo's scenarios/ registry — notably the temp-generated scenario
// used by deep template validation. Providers that honor path resolve from it;
// providers that only key off the name ignore it harmlessly. Empty is allowed
// (the provider falls back to registry-by-name resolution).
func Run(ctx context.Context, provider Provider, targetScenario, scenarioPath string) *Result {
	targetScenario = strings.TrimSpace(targetScenario)
	if targetScenario == "" {
		return failure(provider, targetScenario, shared.FailureClassMisconfiguration, errors.New("target scenario is required"), "")
	}
	if strings.TrimSpace(provider.ProviderScenario) == "" {
		return failure(provider, targetScenario, shared.FailureClassMisconfiguration, errors.New("provider scenario is required"), "")
	}

	baseURL, err := ResolveBaseURL(ctx, provider.ProviderScenario)
	if err != nil {
		return unavailable(provider, targetScenario, fmt.Errorf("resolve %s URL: %w", provider.ProviderScenario, err))
	}
	if strings.TrimSpace(baseURL) == "" {
		return unavailable(provider, targetScenario, fmt.Errorf("%s base URL is empty", provider.ProviderScenario))
	}

	resp, err := NewClient(provider.Timeout, baseURL).ValidateScenario(ctx, connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario:         targetScenario,
		Path:             strings.TrimSpace(scenarioPath),
		IncludeExecution: provider.IncludeExecution,
		CapabilitySubset: append([]string(nil), provider.CapabilitySubset...),
	}))
	if err != nil {
		return unavailable(provider, targetScenario, fmt.Errorf("%s validation RPC failed: %w", provider.ProviderScenario, err))
	}
	if resp == nil || resp.Msg == nil {
		return failure(provider, targetScenario, shared.FailureClassSystem, errors.New("provider returned an empty validation response"), "")
	}
	return translate(provider, targetScenario, resp.Msg)
}

// RunTarget invokes the typed validation RPC for a non-legacy target. The
// provider descriptor/applicability gate decides whether this phase applies;
// this function only transports the resolved tuple and never encodes a
// resource, package, or team as a fake scenario name.
func RunTarget(ctx context.Context, provider Provider, target *commonv1.ValidationTarget, targetPath string) *Result {
	if target == nil || strings.TrimSpace(target.GetId()) == "" {
		return failure(provider, "", shared.FailureClassMisconfiguration, errors.New("validation target is required"), "")
	}
	if strings.TrimSpace(provider.ProviderScenario) == "" {
		return failure(provider, target.GetId(), shared.FailureClassMisconfiguration, errors.New("provider scenario is required"), "")
	}
	baseURL, err := ResolveBaseURL(ctx, provider.ProviderScenario)
	if err != nil {
		return unavailable(provider, target.GetId(), fmt.Errorf("resolve %s URL: %w", provider.ProviderScenario, err))
	}
	if strings.TrimSpace(baseURL) == "" {
		return unavailable(provider, target.GetId(), fmt.Errorf("%s base URL is empty", provider.ProviderScenario))
	}
	requestTarget := proto.Clone(target).(*commonv1.ValidationTarget)
	if strings.TrimSpace(requestTarget.GetRoot()) == "" {
		requestTarget.Root = strings.TrimSpace(targetPath)
	}
	resp, err := NewTargetClient(provider.Timeout, baseURL).ValidateTarget(ctx, connect.NewRequest(&scenariovalidationv1.ValidateTargetRequest{
		Target:           requestTarget,
		IncludeExecution: provider.IncludeExecution,
		Path:             strings.TrimSpace(targetPath),
		CapabilitySubset: append([]string(nil), provider.CapabilitySubset...),
		Exclude:          append([]string(nil), provider.Exclude...),
	}))
	if err != nil {
		return unavailable(provider, target.GetId(), fmt.Errorf("%s target validation RPC failed: %w", provider.ProviderScenario, err))
	}
	if resp == nil || resp.Msg == nil {
		return failure(provider, target.GetId(), shared.FailureClassSystem, errors.New("provider returned an empty target validation response"), "")
	}
	legacy := &scenariovalidationv1.ValidateScenarioResponse{
		Status:                resp.Msg.GetStatus(),
		Assessment:            resp.Msg.GetAssessment(),
		NativeDetail:          resp.Msg.GetNativeDetail(),
		Metrics:               resp.Msg.GetMetrics(),
		Scenario:              target.GetId(),
		FailureClassification: resp.Msg.GetFailureClassification(),
	}
	return translate(provider, target.GetId(), legacy)
}

// RunDurable performs one Start and one server-owned Wait. The parent run and
// phase identity make retries replay-safe without any scenario-name branch.
func RunDurable(ctx context.Context, provider Provider, targetScenario, scenarioPath, parentRunID string) *Result {
	baseURL, err := ResolveBaseURL(ctx, provider.ProviderScenario)
	if err != nil {
		return unavailable(provider, targetScenario, fmt.Errorf("resolve %s URL: %w", provider.ProviderScenario, err))
	}
	key := strings.TrimSpace(parentRunID) + ":" + provider.Phase
	if strings.TrimSpace(parentRunID) == "" {
		return failure(provider, targetScenario, shared.FailureClassMisconfiguration, errors.New("durable provider requires parent run id"), "")
	}
	client := NewDurableClient(provider.Timeout+time.Minute, baseURL)
	started, err := client.StartValidationRun(ctx, connect.NewRequest(&scenariovalidationv1.StartValidationRunRequest{
		Scenario:         targetScenario,
		Path:             scenarioPath,
		IdempotencyKey:   key,
		ParentRunId:      parentRunID,
		CapabilitySubset: append([]string(nil), provider.CapabilitySubset...),
	}))
	if err != nil || started == nil || started.Msg == nil || started.Msg.GetRun() == nil {
		if err == nil {
			err = errors.New("provider returned empty durable start response")
		}
		return unavailable(provider, targetScenario, fmt.Errorf("start durable provider run: %w", err))
	}
	run := started.Msg.GetRun()
	if provider.OnStarted != nil {
		provider.OnStarted(RunReference{
			RunID:       run.GetRunId(),
			ParentRunID: parentRunID,
			ETASeconds:  int64(run.GetEstimatedRemaining().AsDuration().Seconds()),
			State:       strings.TrimPrefix(strings.ToLower(run.GetState().String()), "validation_run_state_"),
		})
	}
	waited, err := client.WaitValidationRun(ctx, connect.NewRequest(&scenariovalidationv1.WaitValidationRunRequest{RunId: run.GetRunId()}))
	if err != nil || waited == nil || waited.Msg == nil || waited.Msg.GetRun() == nil {
		if ctx.Err() != nil {
			// A Test Genie parent cancellation is explicit lifecycle intent, unlike
			// a lost client connection. Propagate it to the provider exactly once;
			// the provider's own durable ledger remains authoritative for recovery.
			_, _ = client.AbortValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.AbortValidationRunRequest{RunId: run.GetRunId(), Reason: "parent Test Genie run canceled"}))
		}
		if err == nil {
			err = errors.New("provider returned empty durable wait response")
		}
		out := unavailable(provider, targetScenario, fmt.Errorf("wait for durable provider run %s: %w", run.GetRunId(), err))
		// A wait failure does not erase a successfully-started durable child.
		// Preserve its identity so the parent run provides an actionable recovery
		// path instead of a bare transport timeout.
		out.Summary.ProviderRunID = run.GetRunId()
		out.Summary.ProviderParentRunID = parentRunID
		out.Summary.ProviderRunState = strings.TrimPrefix(strings.ToLower(run.GetState().String()), "validation_run_state_")
		out.Summary.ProviderETASeconds = int64(run.GetEstimatedRemaining().AsDuration().Seconds())
		out.Remediation = fmt.Sprintf("Provider run %s may still be active. %s", run.GetRunId(), out.Remediation)
		return out
	}
	terminal := waited.Msg.GetRun().GetTerminalResult()
	if terminal == nil {
		return failure(provider, targetScenario, shared.FailureClassMaturityContract, errors.New("durable provider terminal run has no shared validation response"), maturityRemediation(provider, targetScenario))
	}
	out := translate(provider, targetScenario, terminal)
	out.Summary.ProviderRunID = run.GetRunId()
	out.Summary.ProviderParentRunID = parentRunID
	out.Summary.ProviderRunState = strings.TrimPrefix(strings.ToLower(waited.Msg.GetRun().GetState().String()), "validation_run_state_")
	out.Summary.ProviderETASeconds = int64(run.GetEstimatedRemaining().AsDuration().Seconds())
	return out
}

func translate(provider Provider, fallbackScenario string, resp *scenariovalidationv1.ValidateScenarioResponse) *Result {
	if resp.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_UNSPECIFIED {
		return failure(provider, fallbackScenario, shared.FailureClassMaturityContract, errors.New("provider returned unspecified validation status"), maturityRemediation(provider, fallbackScenario))
	}
	if err := requireAssessment(provider, resp.GetAssessment()); err != nil {
		return failure(provider, fallbackScenario, shared.FailureClassMaturityContract, err, maturityRemediation(provider, fallbackScenario))
	}
	scenario := strings.TrimSpace(resp.GetScenario())
	if scenario == "" {
		scenario = fallbackScenario
	}
	summary := summarize(scenario, resp.GetStatus(), resp.GetAssessment())
	findings := assessment.AssessmentToArchitectureFindings(scenario, resp.GetAssessment(), provider.FindingSource)
	observations := observations(provider, resp.GetAssessment())
	if resp.GetAssessment().GetPresentation() == nil {
		// Presentation is a provider-owned rendering projection. Older providers
		// can still supply a structurally valid assessment whose findings must be
		// evaluated; retain that evidence without inventing a presentation and
		// make the outstanding provider migration explicit in the run output.
		observations = append(observations, shared.NewWarningObservation(
			"presentation compatibility mode: provider returned no canonical phase presentation; assessment findings remain authoritative",
		))
	}
	out := &Result{
		RunResult: shared.RunResult[Summary]{
			Summary:      summary,
			Observations: observations,
		},
		Findings:        findings,
		Assessment:      resp.GetAssessment(),
		Metrics:         resp.GetMetrics(),
		Presentation:    resp.GetAssessment().GetPresentation(),
		FindingsSummary: buildFindingsSummary(summary),
	}
	switch resp.GetStatus() {
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED:
		out.Success = false
		out.FailureClass = shared.FailureClassTestFailure
		out.Error = fmt.Errorf("%d %s finding(s) at ERROR or BLOCKER severity", summary.Errors+summary.Blockers, provider.Phase)
		out.Remediation = "Run `" + detailCommand(provider, scenario) + "` for details."
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR:
		out.Success = false
		out.FailureClass = shared.FailureClassSystem
		out.Error = fmt.Errorf("%s reported validation status ERROR", provider.ProviderScenario)
		out.Remediation = "Inspect " + provider.ProviderScenario + " logs and rerun the provider validation."
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_SKIPPED:
		out.Success = true
		out.Summary.Skipped = true
		out.Observations = append(out.Observations, shared.NewSkipObservation(provider.ProviderScenario+" skipped validation"))
	default:
		out.Success = true
	}
	applyGate(provider, scenario, resp, out)
	return out
}

func detailCommand(provider Provider, scenario string) string {
	if command := strings.TrimSpace(provider.DetailCommand); command != "" {
		return strings.ReplaceAll(command, "{{scenario}}", scenario)
	}
	return provider.ProviderScenario + " validate scenario " + scenario
}

func summarize(scenario string, status scenariovalidationv1.ValidationStatus, a *commonv1.MaturityAssessment) Summary {
	s := Summary{
		Scenario: scenario,
		Status:   statusLabel(status),
		Blockers: countSeverity(a, "SEVERITY_BLOCKER"),
		Errors:   countSeverity(a, "SEVERITY_ERROR"),
		Warnings: countSeverity(a, "SEVERITY_WARNING"),
		Infos:    countSeverity(a, "SEVERITY_INFO"),
	}
	if local := a.GetLocal(); local != nil {
		s.LocalCurrentLevel = local.GetCurrentLevel()
		s.LocalNextLevel = local.GetNextLevel()
		s.LocalClean = local.GetClean()
		s.LocalUnknownCount = int(local.GetUnknownCount())
	}
	s.Capabilities = capabilitySummaries(a.GetCapabilities())
	if focus := a.GetHighestPriorityCapability(); focus != nil && strings.TrimSpace(focus.GetCapabilityId()) != "" {
		s.HighestPriorityCapability = &PriorityFocus{
			CapabilityID:    focus.GetCapabilityId(),
			CapabilityLabel: focus.GetCapabilityLabel(),
			CurrentLevel:    focus.GetCurrentLevel(),
			NextLevel:       focus.GetNextLevel(),
			Reason:          focus.GetReason(),
		}
	}
	return s
}

// FilterCapabilityResult projects a provider result onto one capability. It
// is the durable unit used by mixed-determinism caching: the provider owns the
// capability ladder and finding attribution, while Test Genie only carries
// the selected projection forward.
func FilterCapabilityResult(result *Result, capabilityID string) *Result {
	if result == nil || strings.TrimSpace(capabilityID) == "" {
		return nil
	}
	out := *result
	capabilityID = strings.TrimSpace(capabilityID)
	if result.Assessment != nil {
		assessmentCopy := proto.Clone(result.Assessment).(*commonv1.MaturityAssessment)
		assessmentCopy.Capabilities = nil
		for _, capability := range result.Assessment.GetCapabilities() {
			if capability != nil && capability.GetId() == capabilityID {
				assessmentCopy.Capabilities = append(assessmentCopy.Capabilities, capability)
			}
		}
		assessmentCopy.Findings = filterAssessmentFindings(result.Assessment.GetFindings(), capabilityID)
		assessmentCopy.Presentation = assessment.BuildPhasePresentation(assessmentCopy)
		out.Assessment = assessmentCopy
		out.Presentation = assessmentCopy.GetPresentation()
		out.Summary = summarize(assessmentCopy.GetScenario(), statusForAssessment(assessmentCopy), assessmentCopy)
		out.FindingsSummary = buildFindingsSummary(out.Summary)
		out.Findings = filterArchitectureFindings(result.Findings, capabilityID, assessmentFindingCodes(assessmentCopy.GetFindings()))
		return &out
	}
	out.Findings = filterArchitectureFindings(result.Findings, capabilityID, nil)
	return &out
}

// MergeCapabilityResults combines cached and fresh capability projections.
// Duplicate findings are removed by stable semantic identity, and the final
// status/standing is rebuilt from the merged assessment rather than inherited
// from whichever projection happened to run last.
func MergeCapabilityResults(cached, fresh *Result) *Result {
	if cached == nil {
		return fresh
	}
	if fresh == nil {
		return cached
	}
	base := fresh
	if base.Assessment == nil {
		base = cached
	}
	out := *base
	if cached.Assessment != nil || fresh.Assessment != nil {
		merged := mergeAssessments(cached.Assessment, fresh.Assessment)
		out.Assessment = merged
		out.Presentation = merged.GetPresentation()
		out.Summary = summarize(merged.GetScenario(), statusForAssessment(merged), merged)
		out.FindingsSummary = buildFindingsSummary(out.Summary)
	}
	out.Findings = mergeArchitectureFindings(cached.Findings, fresh.Findings)
	out.RunResult.Observations = append(append([]shared.Observation(nil), cached.RunResult.Observations...), fresh.RunResult.Observations...)
	out.Success = out.Summary.Status == "passed" || out.Summary.Status == "skipped"
	if !out.Success && out.Error == nil {
		out.Error = fmt.Errorf("merged capability result contains blocking findings")
		out.FailureClass = shared.FailureClassTestFailure
	}
	return &out
}

func mergeAssessments(left, right *commonv1.MaturityAssessment) *commonv1.MaturityAssessment {
	var out *commonv1.MaturityAssessment
	if right != nil {
		out = proto.Clone(right).(*commonv1.MaturityAssessment)
	} else {
		out = proto.Clone(left).(*commonv1.MaturityAssessment)
	}
	capabilities := map[string]*commonv1.CapabilityMaturityAssessment{}
	for _, source := range []*commonv1.MaturityAssessment{left, right} {
		if source == nil {
			continue
		}
		for _, capability := range source.GetCapabilities() {
			if capability != nil && capability.GetId() != "" {
				capabilities[capability.GetId()] = capability
			}
		}
	}
	out.Capabilities = out.Capabilities[:0]
	for _, capability := range capabilities {
		out.Capabilities = append(out.Capabilities, capability)
	}
	sort.Slice(out.Capabilities, func(i, j int) bool { return out.Capabilities[i].GetId() < out.Capabilities[j].GetId() })
	out.Findings = mergeAssessmentFindings(left, right)
	out.FindingsBySeverity = countAssessmentFindings(out.Findings)
	out.Presentation = assessment.BuildPhasePresentation(out)
	return out
}

func filterAssessmentFindings(findings []*commonv1.AssessmentFinding, capabilityID string) []*commonv1.AssessmentFinding {
	out := make([]*commonv1.AssessmentFinding, 0, len(findings))
	for _, finding := range findings {
		if finding != nil && finding.GetMaturity().GetCapabilityId() == capabilityID {
			out = append(out, finding)
		}
	}
	return out
}

func filterArchitectureFindings(findings []*architecturev1.ArchitectureFinding, capabilityID string, assessmentCodes map[string]struct{}) []*architecturev1.ArchitectureFinding {
	out := make([]*architecturev1.ArchitectureFinding, 0, len(findings))
	for _, finding := range findings {
		if finding == nil {
			continue
		}
		if _, attributed := assessmentCodes[finding.GetCode()]; attributed {
			out = append(out, finding)
			continue
		}
		for _, location := range finding.GetLocations() {
			if location == "capability:"+capabilityID {
				out = append(out, finding)
				break
			}
		}
	}
	return out
}

func assessmentFindingCodes(findings []*commonv1.AssessmentFinding) map[string]struct{} {
	codes := make(map[string]struct{}, len(findings))
	for _, finding := range findings {
		if finding != nil && finding.GetCode() != "" {
			codes[finding.GetCode()] = struct{}{}
		}
	}
	return codes
}

func mergeAssessmentFindings(left, right *commonv1.MaturityAssessment) []*commonv1.AssessmentFinding {
	seen := map[string]*commonv1.AssessmentFinding{}
	for _, source := range []*commonv1.MaturityAssessment{left, right} {
		if source == nil {
			continue
		}
		for _, finding := range source.GetFindings() {
			if finding == nil {
				continue
			}
			key := finding.GetCode() + "\x00" + finding.GetLocation() + "\x00" + finding.GetMaturity().GetCapabilityId()
			seen[key] = finding
		}
	}
	out := make([]*commonv1.AssessmentFinding, 0, len(seen))
	for _, finding := range seen {
		out = append(out, finding)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GetCode() < out[j].GetCode() })
	return out
}

func mergeArchitectureFindings(left, right []*architecturev1.ArchitectureFinding) []*architecturev1.ArchitectureFinding {
	seen := map[string]*architecturev1.ArchitectureFinding{}
	for _, finding := range append(append([]*architecturev1.ArchitectureFinding(nil), left...), right...) {
		if finding == nil {
			continue
		}
		key := finding.GetCode() + "\x00" + strings.Join(finding.GetLocations(), "\x00")
		seen[key] = finding
	}
	out := make([]*architecturev1.ArchitectureFinding, 0, len(seen))
	for _, finding := range seen {
		out = append(out, finding)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GetCode() < out[j].GetCode() })
	return out
}

func countAssessmentFindings(findings []*commonv1.AssessmentFinding) map[string]int32 {
	counts := map[string]int32{}
	for _, finding := range findings {
		if finding != nil {
			counts[finding.GetSeverity()]++
		}
	}
	return counts
}

func statusForAssessment(a *commonv1.MaturityAssessment) scenariovalidationv1.ValidationStatus {
	for _, finding := range a.GetFindings() {
		if finding == nil {
			continue
		}
		switch finding.GetSeverity() {
		case "SEVERITY_BLOCKER", "SEVERITY_ERROR":
			return scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
		}
	}
	return scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
}

func capabilitySummaries(capabilities []*commonv1.CapabilityMaturityAssessment) []CapabilitySummary {
	out := make([]CapabilitySummary, 0, len(capabilities))
	for _, capability := range capabilities {
		if capability == nil {
			continue
		}
		out = append(out, CapabilitySummary{
			ID:                   capability.GetId(),
			Label:                capability.GetLabel(),
			CurrentLevel:         capability.GetCurrentLevel(),
			NextLevel:            capability.GetNextLevel(),
			CurrentSummary:       capability.GetCurrentSummary(),
			NextUnlock:           capability.GetNextUnlock(),
			Clean:                capability.GetClean(),
			UnknownCount:         int(capability.GetUnknownCount()),
			BlockingFindingCount: len(capability.GetBlockingFindingCodes()),
			PriorityRank:         int(capability.GetPriorityRank()),
			PriorityReason:       capability.GetPriorityReason(),
		})
	}
	return out
}

func applyGate(provider Provider, scenario string, resp *scenariovalidationv1.ValidateScenarioResponse, out *Result) {
	if out == nil || provider.GateEnvVar == "" {
		return
	}
	mode, invalid := resolveGateMode(provider)
	out.Summary.GateMode = string(mode)
	native, err := auditNativeDetail(resp)
	if err != nil {
		out.Observations = append(out.Observations, shared.NewWarningObservation(
			fmt.Sprintf("%s gate could not read provider authority detail: %v", provider.Phase, err),
		))
	} else {
		out.Summary.AuthorityConfidence = authorityLabel(native.GetAuthorityConfidence())
		applyNativeFindingClasses(out.Findings, native)
		out.Observations = architectureObservations(provider, out.Findings)
		out.Summary.Categories = categorySummaries(native.GetCategories())
		out.Observations = append(out.Observations, categoryObservations(native.GetCategories())...)
	}
	if invalid != "" {
		out.Observations = append(out.Observations, shared.NewWarningObservation(invalid))
	}
	gateable := gateableFindings(native, out.Findings)
	out.Summary.GatedBlockers = gateable
	if !shouldGate(mode, native, gateable) {
		return
	}
	out.Success = false
	out.FailureClass = shared.FailureClassTestFailure
	out.Error = fmt.Errorf("%d deterministic %s finding(s) gated by %s=%s", gateable, provider.Phase, provider.GateEnvVar, mode)
	out.Remediation = "Run `" + detailCommand(provider, scenario) + "` for details, or set " + provider.GateEnvVar + "=off only for a deliberate advisory rollout."
	out.Observations = append(out.Observations, shared.NewErrorObservation(out.Error.Error()))
}

func categorySummaries(categories []*auditv1.AuditCategory) []CategorySummary {
	out := make([]CategorySummary, 0, len(categories))
	for _, category := range categories {
		if category == nil {
			continue
		}
		out = append(out, CategorySummary{
			Key:   category.GetKey(),
			Label: category.GetLabel(),
			Score: category.GetScore(),
		})
	}
	return out
}

func categoryObservations(categories []*auditv1.AuditCategory) []shared.Observation {
	if len(categories) == 0 {
		return nil
	}
	out := []shared.Observation{shared.NewSectionObservation("▣", "Architecture Score Matrix")}
	for _, category := range categories {
		if category == nil {
			continue
		}
		out = append(out, shared.NewInfoObservation(fmt.Sprintf("%s %s %.0f%%",
			category.GetLabel(), progressBar(category.GetScore(), 12), category.GetScore()*100)))
	}
	var considered int
	for _, category := range categories {
		for _, item := range category.GetTopItems() {
			if item == nil {
				continue
			}
			out = append(out, shared.NewInfoObservation(fmt.Sprintf("Consider %s: [%s/%s] %s",
				category.GetLabel(), cartoSeverityName(item.GetSeverity()), cartoFindingClassName(item.GetFindingClass()), item.GetHeadline())))
			considered++
			if considered == 5 {
				return out
			}
		}
	}
	return out
}

func progressBar(score float64, width int) string {
	if width <= 0 {
		return ""
	}
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	filled := int(score*float64(width) + 0.5)
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "]"
}

func cartoSeverityName(severity cartosharedv1.Severity) string {
	switch severity {
	case cartosharedv1.Severity_SEVERITY_BLOCKER:
		return "blocker"
	case cartosharedv1.Severity_SEVERITY_ERROR:
		return "error"
	case cartosharedv1.Severity_SEVERITY_WARN:
		return "warn"
	case cartosharedv1.Severity_SEVERITY_INFO:
		return "info"
	default:
		return "unspecified"
	}
}

func cartoFindingClassName(class cartosharedv1.FindingClass) string {
	switch class {
	case cartosharedv1.FindingClass_FINDING_CLASS_DETERMINISTIC:
		return "deterministic"
	case cartosharedv1.FindingClass_FINDING_CLASS_HEURISTIC:
		return "heuristic"
	default:
		return "unspecified"
	}
}

func applyNativeFindingClasses(findings []*architecturev1.ArchitectureFinding, native *auditv1.AuditRunResponse) {
	if native == nil || len(findings) == 0 {
		return
	}
	classes := make(map[string]architecturev1.FindingClass, len(native.GetFindings()))
	for _, finding := range native.GetFindings() {
		if finding == nil {
			continue
		}
		classes[nativeFindingKey(finding.GetType(), finding.GetSubtype(), finding.GetLocations())] = architectureClassFromNative(finding.GetFindingClass())
	}
	for _, finding := range findings {
		if finding == nil {
			continue
		}
		if class, ok := classes[nativeFindingKey(finding.GetCode(), "", finding.GetLocations())]; ok {
			finding.FindingClass = class
		}
	}
}

func nativeFindingKey(code, subtype string, locations []string) string {
	code = strings.TrimSpace(code)
	if sub := strings.TrimSpace(subtype); sub != "" && !strings.Contains(code, "/") {
		code += "/" + sub
	}
	return code + "\x1f" + strings.Join(locations, ", ")
}

func architectureClassFromNative(class cartosharedv1.FindingClass) architecturev1.FindingClass {
	switch class {
	case cartosharedv1.FindingClass_FINDING_CLASS_HEURISTIC:
		return architecturev1.FindingClass_FINDING_CLASS_HEURISTIC
	case cartosharedv1.FindingClass_FINDING_CLASS_DETERMINISTIC:
		return architecturev1.FindingClass_FINDING_CLASS_DETERMINISTIC
	default:
		return architecturev1.FindingClass_FINDING_CLASS_UNSPECIFIED
	}
}

func resolveGateMode(provider Provider) (GateMode, string) {
	mode := provider.DefaultGateMode
	if mode == "" {
		mode = GateModeOff
	}
	raw := strings.TrimSpace(os.Getenv(provider.GateEnvVar))
	if raw == "" {
		return mode, ""
	}
	parsed, ok := parseGateMode(raw)
	if ok {
		return parsed, ""
	}
	return mode, fmt.Sprintf("invalid %s=%q; using %s", provider.GateEnvVar, raw, mode)
}

func parseGateMode(raw string) (GateMode, bool) {
	switch GateMode(strings.ToLower(strings.TrimSpace(raw))) {
	case GateModeOff:
		return GateModeOff, true
	case GateModeHighConfidence:
		return GateModeHighConfidence, true
	case GateModeAll:
		return GateModeAll, true
	default:
		return "", false
	}
}

func auditNativeDetail(resp *scenariovalidationv1.ValidateScenarioResponse) (*auditv1.AuditRunResponse, error) {
	if resp == nil || resp.GetNativeDetail() == nil {
		return nil, errors.New("native_detail is missing")
	}
	native := &auditv1.AuditRunResponse{}
	if err := resp.GetNativeDetail().UnmarshalTo(native); err != nil {
		return nil, err
	}
	return native, nil
}

func shouldGate(mode GateMode, native *auditv1.AuditRunResponse, gateable int) bool {
	if gateable <= 0 {
		return false
	}
	switch mode {
	case GateModeAll:
		return true
	case GateModeHighConfidence:
		return native != nil && native.GetAuthorityConfidence() == auditv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_HIGH
	default:
		return false
	}
}

func gateableFindings(native *auditv1.AuditRunResponse, findings []*architecturev1.ArchitectureFinding) int {
	if native != nil {
		total := 0
		for _, finding := range native.GetFindings() {
			if finding == nil {
				continue
			}
			if isAdvisoryIntentFinding(finding.GetType()) {
				continue
			}
			if finding.GetFindingClass() != cartosharedv1.FindingClass_FINDING_CLASS_DETERMINISTIC {
				continue
			}
			switch finding.GetSeverity() {
			case cartosharedv1.Severity_SEVERITY_ERROR, cartosharedv1.Severity_SEVERITY_BLOCKER:
				total++
			}
		}
		return total
	}
	total := 0
	for _, finding := range findings {
		if finding == nil {
			continue
		}
		if isAdvisoryIntentFinding(finding.GetCode()) {
			continue
		}
		if finding.GetFindingClass() != architecturev1.FindingClass_FINDING_CLASS_DETERMINISTIC {
			continue
		}
		switch finding.GetSeverity() {
		case architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR,
			architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER:
			total++
		}
	}
	return total
}

func isAdvisoryIntentFinding(code string) bool {
	if !strings.HasPrefix(strings.TrimSpace(code), "intent.") {
		return false
	}
	return strings.ToLower(strings.TrimSpace(os.Getenv("INTENT_ALIGNMENT_GATE"))) != "strict"
}

func authorityLabel(authority auditv1.AuthorityConfidence) string {
	switch authority {
	case auditv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_LOW:
		return "low"
	case auditv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_MEDIUM:
		return "medium"
	case auditv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_HIGH:
		return "high"
	case auditv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_MISSING:
		return "missing"
	default:
		return ""
	}
}

func observations(provider Provider, a *commonv1.MaturityAssessment) []shared.Observation {
	out := []shared.Observation{shared.NewSectionObservation(provider.Emoji, provider.Phase)}
	out = append(out, capabilityObservations(a)...)
	if len(a.GetFindings()) == 0 {
		return append(out, shared.NewSuccessObservation("No "+provider.Phase+" findings detected"))
	}
	for _, finding := range a.GetFindings() {
		if finding == nil {
			continue
		}
		msg := formatFinding(finding)
		switch normalizeSeverity(finding.GetSeverity()) {
		case "error", "blocker":
			out = append(out, shared.NewErrorObservation(msg))
		case "warning":
			out = append(out, shared.NewWarningObservation(msg))
		default:
			out = append(out, shared.NewInfoObservation(msg))
		}
	}
	return out
}

func capabilityObservations(a *commonv1.MaturityAssessment) []shared.Observation {
	if a == nil || len(a.GetCapabilities()) == 0 {
		return nil
	}
	out := make([]shared.Observation, 0, len(a.GetCapabilities())+1)
	if len(a.GetFindings()) == 0 {
		out = append(out, shared.NewSuccessObservation(fmt.Sprintf("all %d capability assessment(s) clean", len(a.GetCapabilities()))))
		return out
	}
	focusID := ""
	if focus := a.GetHighestPriorityCapability(); focus != nil && strings.TrimSpace(focus.GetCapabilityId()) != "" {
		focusID = strings.TrimSpace(focus.GetCapabilityId())
		label := strings.TrimSpace(focus.GetCapabilityLabel())
		if label == "" {
			label = focus.GetCapabilityId()
		}
		msg := "highest priority capability: " + label
		if next := strings.TrimSpace(focus.GetNextLevel()); next != "" {
			msg += " to " + next
		}
		if reason := strings.TrimSpace(focus.GetReason()); reason != "" {
			msg += " - " + reason
		}
		out = append(out, shared.NewInfoObservation(msg))
	}
	for _, capability := range a.GetCapabilities() {
		if capability == nil {
			continue
		}
		blockers := len(capability.GetBlockingFindingCodes())
		unknowns := int(capability.GetUnknownCount())
		if capability.GetId() == focusID || blockers == 0 && unknowns == 0 {
			continue
		}
		label := strings.TrimSpace(capability.GetLabel())
		if label == "" {
			label = capability.GetId()
		}
		msg := fmt.Sprintf("%s capability: current=%s", label, emptyAs(capability.GetCurrentLevel(), "none"))
		if next := strings.TrimSpace(capability.GetNextLevel()); next != "" {
			msg += " next=" + next
		} else {
			msg += " maximum maturity reached"
		}
		if blockers > 0 {
			msg += fmt.Sprintf(" blockers=%d", blockers)
		}
		if unknowns > 0 {
			msg += fmt.Sprintf(" unknown=%d", unknowns)
		}
		if summary := strings.TrimSpace(capability.GetCurrentSummary()); summary != "" {
			msg += " - " + summary
		}
		out = append(out, shared.NewInfoObservation(msg))
	}
	return out
}

func architectureObservations(provider Provider, findings []*architecturev1.ArchitectureFinding) []shared.Observation {
	out := []shared.Observation{shared.NewSectionObservation(provider.Emoji, provider.Phase)}
	if len(findings) == 0 {
		return append(out, shared.NewSuccessObservation("No "+provider.Phase+" findings detected"))
	}
	for _, finding := range findings {
		if finding == nil {
			continue
		}
		msg := formatArchitectureFinding(finding)
		switch finding.GetFindingClass() {
		case architecturev1.FindingClass_FINDING_CLASS_HEURISTIC:
			if isErrorPlus(finding.GetSeverity()) {
				out = append(out, shared.NewWarningObservation("advisory: "+msg))
			} else if finding.GetSeverity() == architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING {
				out = append(out, shared.NewWarningObservation(msg))
			} else {
				out = append(out, shared.NewInfoObservation(msg))
			}
		default:
			switch finding.GetSeverity() {
			case architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR,
				architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER:
				out = append(out, shared.NewErrorObservation(msg))
			case architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING:
				out = append(out, shared.NewWarningObservation(msg))
			default:
				out = append(out, shared.NewInfoObservation(msg))
			}
		}
	}
	return out
}

func isErrorPlus(severity architecturev1.FindingSeverity) bool {
	return severity == architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR ||
		severity == architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER
}

func formatArchitectureFinding(f *architecturev1.ArchitectureFinding) string {
	parts := []string{strings.TrimSpace(f.GetCode())}
	if msg := strings.TrimSpace(f.GetMessage()); msg != "" {
		parts = append(parts, msg)
	}
	line := strings.Join(nonEmpty(parts...), ": ")
	if loc := strings.Join(nonEmpty(f.GetLocations()...), ", "); loc != "" {
		line += " [" + loc + "]"
	}
	if suggestion := strings.TrimSpace(f.GetSuggestion()); suggestion != "" {
		line += "\n    suggestion: " + suggestion
	}
	return line
}

func unavailable(provider Provider, scenario string, err error) *Result {
	if provider.Optional {
		summary := Summary{Scenario: scenario, Status: "skipped", Skipped: true}
		return &Result{RunResult: shared.RunResult[Summary]{
			Success: true,
			Summary: summary,
			Observations: []shared.Observation{shared.NewSkipObservation(
				fmt.Sprintf("%s skipped — %s unreachable: %v (start it via `vrooli scenario start %s`)",
					provider.Phase, provider.ProviderScenario, err, provider.ProviderScenario),
			)},
		}}
	}
	return failure(provider, scenario, shared.FailureClassMissingDependency, err, "Ensure "+provider.ProviderScenario+" is running (`vrooli scenario start "+provider.ProviderScenario+"`) and reachable.")
}

func failure(provider Provider, scenario string, class shared.FailureClass, err error, remediation string) *Result {
	return &Result{RunResult: shared.RunResult[Summary]{
		Success:      false,
		Error:        err,
		FailureClass: class,
		Remediation:  remediation,
		Summary:      Summary{Scenario: scenario, Status: "error"},
		Observations: []shared.Observation{shared.NewErrorObservation(err.Error())},
	}}
}

func requireAssessment(provider Provider, a *commonv1.MaturityAssessment) error {
	if err := assessment.RequireIdentity(provider.ProviderScenario, provider.Phase, a); err != nil {
		return fmt.Errorf("%s response violates the provider maturity contract: %w", provider.ProviderScenario, err)
	}
	if err := assessment.ValidateAssessment(a); err != nil {
		return fmt.Errorf("%s response violates the provider maturity contract: %w", provider.ProviderScenario, err)
	}
	if a.GetPresentation() != nil {
		if err := assessment.ValidatePhasePresentation(a); err != nil {
			return fmt.Errorf("%s response violates the phase presentation contract: %w", provider.ProviderScenario, err)
		}
	}
	return nil
}

func buildFindingsSummary(s Summary) *runspb.PhaseFindingsSummary {
	return &runspb.PhaseFindingsSummary{
		Blockers: int32(s.Blockers),
		Errors:   int32(s.Errors),
		Warnings: int32(s.Warnings),
		Infos:    int32(s.Infos),
		Total:    int32(s.Blockers + s.Errors + s.Warnings + s.Infos),
	}
}

func maturityRemediation(provider Provider, scenario string) string {
	return "Run `test-genie provider-contract check " + provider.Phase + " " + scenario + " --json` after restarting " + provider.ProviderScenario + " through lifecycle, then fix the provider maturity assessment."
}

func countSeverity(a *commonv1.MaturityAssessment, severity string) int {
	if a == nil {
		return 0
	}
	want := normalizeSeverity(severity)
	total := 0
	for key, count := range a.GetFindingsBySeverity() {
		if normalizeSeverity(key) == want {
			total += int(count)
		}
	}
	return total
}

func statusLabel(status scenariovalidationv1.ValidationStatus) string {
	return strings.ToLower(strings.TrimPrefix(status.String(), "VALIDATION_STATUS_"))
}

func normalizeSeverity(raw string) string {
	if normalized := shared.NormalizeFindingSeverityLabel(raw); normalized != "" {
		return normalized
	}
	return "info"
}

func formatFinding(f *commonv1.AssessmentFinding) string {
	parts := []string{strings.TrimSpace(f.GetCode())}
	if title := strings.TrimSpace(f.GetTitle()); title != "" {
		parts = append(parts, title)
	}
	if msg := strings.TrimSpace(f.GetMessage()); msg != "" {
		parts = append(parts, msg)
	}
	line := strings.Join(nonEmpty(parts...), ": ")
	if loc := strings.TrimSpace(f.GetLocation()); loc != "" {
		line += " [" + loc + "]"
	}
	if remediation := strings.TrimSpace(f.GetRemediation()); remediation != "" {
		line += "\n    suggestion: " + remediation
	}
	return line
}

func nonEmpty(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func emptyAs(value, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}
