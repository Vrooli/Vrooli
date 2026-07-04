// Package deterministicfix aggregates the shared scenario-validation Fix RPC
// across a target scenario's delegated health providers. Unlike the agent-based
// fix path (internal/fix), this path is fully deterministic: it asks each
// provider to PreviewFix/ApplyFix its own registered remediations and merges the
// candidates into a single report. Providers that ship no deterministic fixer
// return Unimplemented and are recorded as "no_fixer" rather than failing the
// aggregate; unreachable providers are recorded as "unreachable".
package deterministicfix

import (
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/discovery"
	autofixcore "github.com/vrooli/maturity-go/autofix"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

// Provider-level outcome statuses.
const (
	StatusFixed       = "fixed"       // candidates returned (preview or applied)
	StatusClean       = "clean"       // reachable, no auto-fixable findings
	StatusNoFixer     = "no_fixer"    // provider does not implement the Fix RPC
	StatusUnreachable = "unreachable" // provider could not be resolved/contacted
	StatusError       = "error"       // provider returned an error
)

// DefaultTimeout bounds each provider Fix call.
const DefaultTimeout = 60 * time.Second

// Candidate is one proposed or applied edit from a provider.
type Candidate struct {
	Provider    string `json:"provider"`
	RuleID      string `json:"ruleId"`
	FilePath    string `json:"filePath"`
	Description string `json:"description"`
	Applied     bool   `json:"applied"`
}

// ProviderReport captures one provider's contribution to the aggregate.
type ProviderReport struct {
	Provider   string      `json:"provider"`
	Status     string      `json:"status"`
	Candidates []Candidate `json:"candidates,omitempty"`
	Messages   []string    `json:"messages,omitempty"`
	Error      string      `json:"error,omitempty"`
}

// Report is the unified deterministic-fix result for one scenario.
type Report struct {
	Scenario        string           `json:"scenario"`
	Applied         bool             `json:"applied"`
	Providers       []ProviderReport `json:"providers"`
	TotalCandidates int              `json:"totalCandidates"`
}

// FixClient is the subset of the generated ScenarioValidationService client the
// aggregate needs; the generated client satisfies it.
type FixClient interface {
	PreviewFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error)
	ApplyFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error)
}

// Runner aggregates deterministic fixes across providers. The seams default to
// production behavior and are overridable in tests.
type Runner struct {
	// Providers are the provider scenario names to fix through, in order. When
	// empty, DefaultProviders() (the delegated catalog) is used.
	Providers []string
	// ResolveBaseURL resolves a provider scenario's API base URL.
	ResolveBaseURL func(ctx context.Context, scenario string) (string, error)
	// NewClient builds a shared Fix client for a resolved base URL.
	NewClient func(timeout time.Duration, baseURL string) FixClient
	// Timeout bounds each provider call (DefaultTimeout when zero).
	Timeout time.Duration
}

// NewRunner returns a Runner wired with production seams.
func NewRunner() *Runner {
	return &Runner{
		ResolveBaseURL: discovery.ResolveScenarioURLDefault,
		NewClient: func(timeout time.Duration, baseURL string) FixClient {
			return scenariovalidationconnect.NewScenarioValidationServiceClient(&http.Client{Timeout: timeout}, baseURL)
		},
	}
}

// Run aggregates PreviewFix (apply=false) or ApplyFix (apply=true) across the
// runner's providers for the target scenario.
func (r *Runner) Run(ctx context.Context, scenario string, apply bool, ruleIDs []string) *Report {
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	providers := r.Providers
	if len(providers) == 0 {
		providers = DefaultProviders()
	}
	report := &Report{Scenario: scenario, Applied: apply}
	for _, provider := range providers {
		pr := r.runProvider(ctx, provider, scenario, apply, ruleIDs, timeout)
		report.Providers = append(report.Providers, pr)
		report.TotalCandidates += len(pr.Candidates)
	}
	return report
}

func (r *Runner) runProvider(ctx context.Context, provider, scenario string, apply bool, ruleIDs []string, timeout time.Duration) ProviderReport {
	baseURL, err := r.ResolveBaseURL(ctx, provider)
	if err != nil {
		return ProviderReport{Provider: provider, Status: StatusUnreachable, Error: err.Error()}
	}

	client := r.NewClient(timeout, baseURL)
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req := connect.NewRequest(&scenariovalidationv1.FixRequest{Scenario: scenario, RuleIds: ruleIDs})
	var resp *connect.Response[scenariovalidationv1.FixResponse]
	if apply {
		resp, err = client.ApplyFix(callCtx, req)
	} else {
		resp, err = client.PreviewFix(callCtx, req)
	}
	if err != nil {
		return classifyClientError(provider, err)
	}
	return providerReportFromResponse(provider, resp.Msg)
}

func classifyClientError(provider string, err error) ProviderReport {
	switch connect.CodeOf(err) {
	case connect.CodeUnimplemented:
		return ProviderReport{Provider: provider, Status: StatusNoFixer}
	case connect.CodeUnavailable, connect.CodeDeadlineExceeded:
		return ProviderReport{Provider: provider, Status: StatusUnreachable, Error: err.Error()}
	default:
		return ProviderReport{Provider: provider, Status: StatusError, Error: err.Error()}
	}
}

func providerReportFromResponse(provider string, msg *scenariovalidationv1.FixResponse) ProviderReport {
	pr := ProviderReport{Provider: provider, Messages: msg.GetMessages()}
	for _, c := range autofixcore.CandidatesFromProto(msg.GetCandidates()) {
		pr.Candidates = append(pr.Candidates, Candidate{
			Provider:    provider,
			RuleID:      c.RuleID,
			FilePath:    c.FilePath,
			Description: c.Description,
			Applied:     c.Applied,
		})
	}
	if len(pr.Candidates) == 0 {
		pr.Status = StatusClean
	} else {
		pr.Status = StatusFixed
	}
	return pr
}

// DefaultProviders returns the delegated provider scenarios (deduplicated, in
// catalog order) that the deterministic aggregate fans out to.
func DefaultProviders() []string {
	names := delegatedProviderScenarios()
	if len(names) == 0 {
		// Defensive fallback: the known fixer-capable providers.
		return []string{"structure-health", "quality-health", "knowledge-observatory"}
	}
	return names
}
