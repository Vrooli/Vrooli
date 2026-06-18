package validationprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"

	"test-genie/internal/shared"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/maturity-go/assessment"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/audit"
	cartosharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/shared"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
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
	GateEnvVar       string
	DefaultGateMode  GateMode
}

type Summary struct {
	Scenario            string            `json:"scenario"`
	Status              string            `json:"status"`
	Blockers            int               `json:"blockers"`
	Errors              int               `json:"errors"`
	Warnings            int               `json:"warnings"`
	Infos               int               `json:"infos"`
	LocalCurrentLevel   string            `json:"local_current_level,omitempty"`
	LocalNextLevel      string            `json:"local_next_level,omitempty"`
	AuthorityConfidence string            `json:"authority_confidence,omitempty"`
	GateMode            string            `json:"gate_mode,omitempty"`
	GatedBlockers       int               `json:"gated_blockers,omitempty"`
	Categories          []CategorySummary `json:"categories,omitempty"`
	Skipped             bool              `json:"skipped,omitempty"`
}

type CategorySummary struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	Score float64 `json:"score"`
}

func (s Summary) String() string {
	if s.Skipped {
		return "skipped"
	}
	text := fmt.Sprintf("%s status=%s blockers=%d errors=%d warnings=%d infos=%d", s.Scenario, s.Status, s.Blockers, s.Errors, s.Warnings, s.Infos)
	if s.LocalCurrentLevel != "" || s.LocalNextLevel != "" {
		text += fmt.Sprintf(" local=%s next=%s", s.LocalCurrentLevel, s.LocalNextLevel)
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
}

type Client interface {
	ValidateScenario(context.Context, *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error)
}

var (
	ResolveBaseURL = func(ctx context.Context, scenario string) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, scenario)
	}
	NewClient = func(timeout time.Duration, baseURL string) Client {
		return scenariovalidationconnect.NewScenarioValidationServiceClient(&http.Client{Timeout: timeout}, baseURL)
	}
)

func Run(ctx context.Context, provider Provider, targetScenario string) *Result {
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
		IncludeExecution: provider.IncludeExecution,
	}))
	if err != nil {
		return unavailable(provider, targetScenario, fmt.Errorf("%s validation RPC failed: %w", provider.ProviderScenario, err))
	}
	if resp == nil || resp.Msg == nil {
		return failure(provider, targetScenario, shared.FailureClassSystem, errors.New("provider returned an empty validation response"), "")
	}
	return translate(provider, targetScenario, resp.Msg)
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
	out := &Result{
		RunResult: shared.RunResult[Summary]{
			Summary:      summary,
			Observations: observations(provider, resp.GetAssessment()),
		},
		Findings: findings,
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
	}
	return s
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
	if err := assessment.ValidateAssessment(a); err != nil {
		return fmt.Errorf("%s response violates maturity assessment contract: %w", provider.ProviderScenario, err)
	}
	if strings.TrimSpace(a.GetProvider()) == "" || strings.TrimSpace(a.GetPhase()) == "" {
		return fmt.Errorf("%s response violates maturity assessment contract: provider and phase are required", provider.ProviderScenario)
	}
	return nil
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
