package validationprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"

	"test-genie/internal/shared"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/maturity-go/assessment"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

type Provider struct {
	Phase            string
	ProviderScenario string
	FindingSource    architecturev1.FindingSource
	Emoji            string
	SkipEnvVar       string
	DetailCommand    string
	Optional         bool
	Timeout          time.Duration
	IncludeExecution bool
}

type Summary struct {
	Scenario          string `json:"scenario"`
	Status            string `json:"status"`
	Errors            int    `json:"errors"`
	Warnings          int    `json:"warnings"`
	Infos             int    `json:"infos"`
	LocalCurrentLevel string `json:"local_current_level,omitempty"`
	LocalNextLevel    string `json:"local_next_level,omitempty"`
	Skipped           bool   `json:"skipped,omitempty"`
}

func (s Summary) String() string {
	if s.Skipped {
		return "skipped"
	}
	text := fmt.Sprintf("%s status=%s errors=%d warnings=%d infos=%d", s.Scenario, s.Status, s.Errors, s.Warnings, s.Infos)
	if s.LocalCurrentLevel != "" || s.LocalNextLevel != "" {
		text += fmt.Sprintf(" local=%s next=%s", s.LocalCurrentLevel, s.LocalNextLevel)
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
		out.Error = fmt.Errorf("%d %s finding(s) at ERROR severity", summary.Errors, provider.Phase)
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
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.TrimPrefix(s, "finding_severity_")
	s = strings.TrimPrefix(s, "severity_")
	switch s {
	case "blocker":
		return "blocker"
	case "error", "failure", "critical", "high":
		return "error"
	case "warn", "warning", "medium":
		return "warning"
	default:
		return "info"
	}
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
