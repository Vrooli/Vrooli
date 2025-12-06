package scenarios

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"test-genie/internal/shared"
	"test-genie/internal/smoke"
)

type scenarioSummaryStore interface {
	List(ctx context.Context) ([]ScenarioSummary, error)
	Get(ctx context.Context, scenario string) (*ScenarioSummary, error)
}

// ScenarioLister exposes scenario metadata from the Vrooli CLI.
type ScenarioLister interface {
	ListScenarios(ctx context.Context) ([]ScenarioMetadata, error)
}

// ScenarioMetadata captures high-level scenario info returned by the CLI.
type ScenarioMetadata struct {
	Name        string
	Description string
	Status      string
	Tags        []string
}

// ScenarioDirectoryService orchestrates catalog lookups.
type ScenarioDirectoryService struct {
	repo          scenarioSummaryStore
	lister        ScenarioLister
	scenariosRoot string
	detectTesting func(string) TestingCapabilities
	runTests      func(context.Context, TestingCapabilities, RunScenarioTestOptions) (*TestingRunnerResult, error)
}

// RunScenarioTestOptions carries execution preferences and extra args passed to the runner.
type RunScenarioTestOptions struct {
	Preferred string
	ExtraArgs []string
}

func NewScenarioDirectoryService(repo scenarioSummaryStore, lister ScenarioLister, scenariosRoot string) *ScenarioDirectoryService {
	defaultRunner := TestingRunner{
		Timeout: defaultTestingTimeout,
		Output:  log.Writer(),
		LogDir:  filepath.Join(strings.TrimSpace(scenariosRoot), "_artifacts", "scenario-tests"),
	}
	return &ScenarioDirectoryService{
		repo:          repo,
		lister:        lister,
		scenariosRoot: strings.TrimSpace(scenariosRoot),
		detectTesting: DetectTestingCapabilities,
		runTests: func(ctx context.Context, caps TestingCapabilities, opts RunScenarioTestOptions) (*TestingRunnerResult, error) {
			return defaultRunner.RunWithArgs(ctx, caps, opts.Preferred, opts.ExtraArgs)
		},
	}
}

// ListSummaries returns every tracked scenario with queue + execution metadata and hydrates
// missing scenarios with CLI metadata so the catalog includes the entire workspace.
func (s *ScenarioDirectoryService) ListSummaries(ctx context.Context) ([]ScenarioSummary, error) {
	if s == nil || s.repo == nil {
		return nil, sql.ErrConnDone
	}

	summaries, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range summaries {
		s.decorateScenario(&summaries[i])
	}
	if s.lister == nil {
		return summaries, nil
	}

	metadata, err := s.lister.ListScenarios(ctx)
	if err != nil {
		log.Printf("scenario lister failed: %v", err)
		return summaries, nil
	}

	if len(metadata) == 0 {
		return summaries, nil
	}

	summaryMap := make(map[string]ScenarioSummary, len(summaries))
	for _, summary := range summaries {
		key := strings.ToLower(summary.ScenarioName)
		if key == "" {
			continue
		}
		summaryMap[key] = summary
	}

	merged := make([]ScenarioSummary, 0, len(metadata)+len(summaries))
	for _, meta := range metadata {
		name := strings.TrimSpace(meta.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if summary, ok := summaryMap[key]; ok {
			summary := applyScenarioMetadata(summary, meta)
			s.decorateScenario(&summary)
			merged = append(merged, summary)
			delete(summaryMap, key)
			continue
		}
		summary := applyScenarioMetadata(ScenarioSummary{
			ScenarioName:    name,
			PendingRequests: 0,
			TotalRequests:   0,
			TotalExecutions: 0,
		}, meta)
		s.decorateScenario(&summary)
		merged = append(merged, summary)
	}

	// Preserve any tracked scenarios not currently returned by the CLI.
	if len(summaryMap) > 0 {
		leftovers := make([]ScenarioSummary, 0, len(summaryMap))
		for _, summary := range summaryMap {
			s.decorateScenario(&summary)
			leftovers = append(leftovers, summary)
		}
		sort.Slice(leftovers, func(i, j int) bool {
			return strings.ToLower(leftovers[i].ScenarioName) < strings.ToLower(leftovers[j].ScenarioName)
		})
		merged = append(merged, leftovers...)
	}

	return merged, nil
}

// GetSummary loads a single scenario summary by name and falls back to CLI metadata if needed.
func (s *ScenarioDirectoryService) GetSummary(ctx context.Context, scenario string) (*ScenarioSummary, error) {
	if s == nil || s.repo == nil {
		return nil, sql.ErrConnDone
	}
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil, sql.ErrNoRows
	}

	summary, err := s.repo.Get(ctx, scenario)
	if err == nil {
		if meta := s.lookupScenarioMetadata(ctx, scenario); meta != nil {
			result := applyScenarioMetadata(*summary, *meta)
			s.decorateScenario(&result)
			return &result, nil
		}
		s.decorateScenario(summary)
		return summary, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	meta := s.lookupScenarioMetadata(ctx, scenario)
	if meta == nil {
		return nil, err
	}

	placeholder := applyScenarioMetadata(ScenarioSummary{
		ScenarioName:    meta.Name,
		PendingRequests: 0,
		TotalRequests:   0,
		TotalExecutions: 0,
	}, *meta)
	s.decorateScenario(&placeholder)
	return &placeholder, nil
}

func (s *ScenarioDirectoryService) lookupScenarioMetadata(ctx context.Context, scenario string) *ScenarioMetadata {
	if s.lister == nil {
		return nil
	}
	metadata, err := s.lister.ListScenarios(ctx)
	if err != nil {
		log.Printf("scenario metadata lookup failed: %v", err)
		return nil
	}
	target := strings.ToLower(strings.TrimSpace(scenario))
	for _, item := range metadata {
		if strings.ToLower(strings.TrimSpace(item.Name)) == target {
			meta := item
			return &meta
		}
	}
	return nil
}

func applyScenarioMetadata(summary ScenarioSummary, meta ScenarioMetadata) ScenarioSummary {
	summary.ScenarioDescription = strings.TrimSpace(meta.Description)
	summary.ScenarioStatus = strings.TrimSpace(meta.Status)
	if len(meta.Tags) > 0 {
		summary.ScenarioTags = append([]string(nil), meta.Tags...)
	} else {
		summary.ScenarioTags = nil
	}
	return summary
}

func (s *ScenarioDirectoryService) decorateScenario(summary *ScenarioSummary) {
	if summary == nil || summary.ScenarioName == "" {
		return
	}
	if s == nil || s.scenariosRoot == "" || s.detectTesting == nil {
		return
	}
	dir := filepath.Join(s.scenariosRoot, summary.ScenarioName)
	caps := s.detectTesting(dir)
	if !caps.HasTests && !caps.Lifecycle && !caps.Legacy && !caps.Phased {
		summary.Testing = nil
		return
	}
	summary.Testing = &caps
}

// RunScenarioTests executes the preferred testing command for the provided scenario.
func (s *ScenarioDirectoryService) RunScenarioTests(ctx context.Context, scenario string, preferred string, extraArgs []string) (*TestingCommand, *TestingRunnerResult, error) {
	if s == nil || s.runTests == nil {
		return nil, nil, sql.ErrConnDone
	}
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil, nil, shared.NewValidationError("scenario name is required")
	}
	if s.scenariosRoot == "" {
		return nil, nil, fmt.Errorf("scenarios root is not configured")
	}
	dir := filepath.Join(s.scenariosRoot, scenario)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("scenario directory not found: %w", os.ErrNotExist)
		}
		return nil, nil, fmt.Errorf("failed to access scenario directory: %w", err)
	}
	if !info.IsDir() {
		return nil, nil, fmt.Errorf("scenario path is not a directory")
	}
	caps := s.detectTesting(dir)
	if !caps.HasTests {
		return nil, nil, shared.NewValidationError("scenario does not define runnable tests")
	}
	cmd := caps.SelectCommand(preferred)
	if cmd == nil {
		return nil, nil, shared.NewValidationError("requested test type is unavailable for this scenario")
	}
	result, err := s.runTests(ctx, caps, RunScenarioTestOptions{Preferred: preferred, ExtraArgs: extraArgs})
	if err != nil {
		return nil, nil, err
	}
	return cmd, result, nil
}

// UISmokeResult represents the outcome of a UI smoke test.
type UISmokeResult struct {
	Scenario            string          `json:"scenario"`
	Status              string          `json:"status"`
	BlockedReason       string          `json:"blocked_reason,omitempty"`
	Message             string          `json:"message"`
	Timestamp           time.Time       `json:"timestamp"`
	DurationMs          int64           `json:"duration_ms"`
	UIURL               string          `json:"ui_url,omitempty"`
	Handshake           json.RawMessage `json:"handshake,omitempty"`
	NetworkFailureCount int             `json:"network_failure_count"`
	PageErrorCount      int             `json:"page_error_count"`
	ConsoleErrorCount   int             `json:"console_error_count"`
	Artifacts           json.RawMessage `json:"artifacts,omitempty"`
	Bundle              json.RawMessage `json:"bundle,omitempty"`
}

// RunUISmoke executes a UI smoke test for the specified scenario.
// If uiURL is provided, it overrides the auto-detected URL.
// If browserlessURL is provided, it overrides the default Browserless endpoint.
// If timeoutMs is > 0, it overrides the default timeout.
func (s *ScenarioDirectoryService) RunUISmoke(ctx context.Context, scenario string, uiURL string, browserlessURL string, timeoutMs int64) (*UISmokeResult, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil, shared.NewValidationError("scenario name is required")
	}
	if s.scenariosRoot == "" {
		return nil, fmt.Errorf("scenarios root is not configured")
	}
	dir := filepath.Join(s.scenariosRoot, scenario)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("scenario directory not found: %w", os.ErrNotExist)
		}
		return nil, fmt.Errorf("failed to access scenario directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scenario path is not a directory")
	}

	// Use default browserless URL if not provided
	if browserlessURL == "" {
		browserlessURL = smoke.DefaultBrowserlessURL
	}

	opts := []smoke.RunnerOption{smoke.WithRunnerLogger(log.Writer())}
	if timeoutMs > 0 {
		opts = append(opts, smoke.WithRunnerTimeout(time.Duration(timeoutMs)*time.Millisecond))
	}
	if uiURL != "" {
		opts = append(opts, smoke.WithUIURL(uiURL))
	}

	runner := smoke.NewRunner(browserlessURL, opts...)
	result, err := runner.Run(ctx, scenario, dir)
	if err != nil {
		return nil, fmt.Errorf("ui smoke test failed: %w", err)
	}

	return convertSmokeResult(result), nil
}

// UISmokeOptions contains options for running a UI smoke test.
type UISmokeOptions struct {
	URL            string
	BrowserlessURL string
	TimeoutMs      int64
	NoRecovery     bool
	SharedMode     bool
	AutoStart      bool
}

// RunUISmokeWithOpts executes a UI smoke test with full options support.
func (s *ScenarioDirectoryService) RunUISmokeWithOpts(ctx context.Context, scenario string, opts UISmokeOptions) (*UISmokeResult, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil, shared.NewValidationError("scenario name is required")
	}
	if s.scenariosRoot == "" {
		return nil, fmt.Errorf("scenarios root is not configured")
	}
	dir := filepath.Join(s.scenariosRoot, scenario)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("scenario directory not found: %w", os.ErrNotExist)
		}
		return nil, fmt.Errorf("failed to access scenario directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scenario path is not a directory")
	}

	// Use default browserless URL if not provided
	browserlessURL := opts.BrowserlessURL
	if browserlessURL == "" {
		browserlessURL = smoke.DefaultBrowserlessURL
	}

	runnerOpts := []smoke.RunnerOption{smoke.WithRunnerLogger(log.Writer())}
	if opts.TimeoutMs > 0 {
		runnerOpts = append(runnerOpts, smoke.WithRunnerTimeout(time.Duration(opts.TimeoutMs)*time.Millisecond))
	}
	if opts.URL != "" {
		runnerOpts = append(runnerOpts, smoke.WithUIURL(opts.URL))
	}
	// Apply recovery options
	if opts.NoRecovery {
		runnerOpts = append(runnerOpts, smoke.WithAutoRecovery(false))
	}
	if opts.SharedMode {
		runnerOpts = append(runnerOpts, smoke.WithSharedMode(true))
	}
	if opts.AutoStart {
		runnerOpts = append(runnerOpts, smoke.WithAutoStart(true))
	}

	runner := smoke.NewRunner(browserlessURL, runnerOpts...)
	result, err := runner.Run(ctx, scenario, dir)
	if err != nil {
		return nil, fmt.Errorf("ui smoke test failed: %w", err)
	}

	return convertSmokeResult(result), nil
}

// convertSmokeResult converts a smoke.Result to a UISmokeResult API response.
func convertSmokeResult(result *smoke.Result) *UISmokeResult {
	apiResult := &UISmokeResult{
		Scenario:            result.Scenario,
		Status:              string(result.Status),
		BlockedReason:       string(result.BlockedReason),
		Message:             result.Message,
		Timestamp:           result.Timestamp,
		DurationMs:          result.DurationMs,
		UIURL:               result.UIURL,
		NetworkFailureCount: result.NetworkFailureCount,
		PageErrorCount:      result.PageErrorCount,
		ConsoleErrorCount:   result.ConsoleErrorCount,
	}

	// Marshal nested structs to JSON
	if handshakeData, err := json.Marshal(result.Handshake); err == nil {
		apiResult.Handshake = handshakeData
	}
	if artifactsData, err := json.Marshal(result.Artifacts); err == nil {
		apiResult.Artifacts = artifactsData
	}
	if result.Bundle != nil {
		if bundleData, err := json.Marshal(result.Bundle); err == nil {
			apiResult.Bundle = bundleData
		}
	}

	return apiResult
}
