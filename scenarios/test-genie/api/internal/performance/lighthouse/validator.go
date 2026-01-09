package lighthouse

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"
)

// Validator runs Lighthouse audits and validates against thresholds.
type Validator interface {
	// Audit runs Lighthouse audits for all configured pages.
	Audit(ctx context.Context) *AuditResult
}

// ValidatorConfig holds configuration for the validator.
type ValidatorConfig struct {
	// BaseURL is the base URL of the UI to audit (e.g., "http://localhost:3000").
	BaseURL string

	// Config is the Lighthouse configuration from .vrooli/lighthouse.json.
	Config *Config

	// ChromePath is an optional path to Chrome executable.
	// If empty, Lighthouse will auto-detect Chrome or use CHROME_PATH env var.
	ChromePath string
}

// ArtifactWriter defines the interface for writing Lighthouse artifacts.
type ArtifactWriter interface {
	// WritePageReport writes the JSON report for a single page audit.
	WritePageReport(pageID string, rawResponse []byte) (string, error)
	// WriteHTMLReport writes the HTML report for a single page audit.
	WriteHTMLReport(pageID string, htmlContent []byte) (string, error)
	// WritePhaseResults writes the phase results JSON.
	WritePhaseResults(result *AuditResult) error
	// WriteSummary writes a summary JSON with all page results.
	WriteSummary(result *AuditResult) (string, error)
}

// validator is the default implementation of Validator.
type validator struct {
	config         ValidatorConfig
	client         Client
	logWriter      io.Writer
	artifactWriter ArtifactWriter
}

// ValidatorOption configures a validator.
type ValidatorOption func(*validator)

// New creates a new Lighthouse validator.
func New(config ValidatorConfig, opts ...ValidatorOption) Validator {
	v := &validator{
		config:    config,
		logWriter: io.Discard,
	}

	for _, opt := range opts {
		opt(v)
	}

	// Set default client if not provided - use CLI runner (official Lighthouse CLI)
	if v.client == nil {
		timeout := 120 * time.Second
		if config.Config != nil {
			timeout = time.Duration(config.Config.GetTimeout()) * time.Millisecond
		}

		cliOpts := []CLIRunnerOption{WithCLITimeout(timeout)}
		if config.ChromePath != "" {
			cliOpts = append(cliOpts, WithChromePath(config.ChromePath))
		}
		v.client = NewCLIRunner(cliOpts...)
	}

	return v
}

// WithLogger sets the log writer for the validator.
func WithLogger(w io.Writer) ValidatorOption {
	return func(v *validator) {
		v.logWriter = w
	}
}

// WithClient sets a custom client (for testing).
func WithClient(c Client) ValidatorOption {
	return func(v *validator) {
		v.client = c
	}
}

// WithArtifactWriter sets the artifact writer for saving reports.
func WithArtifactWriter(w ArtifactWriter) ValidatorOption {
	return func(v *validator) {
		v.artifactWriter = w
	}
}

// Audit implements Validator.
func (v *validator) Audit(ctx context.Context) *AuditResult {
	// Check if disabled or no pages
	if v.config.Config == nil || !v.config.Config.Enabled {
		logInfo(
			v.logWriter,
			"[INFO] Lighthouse audits disabled (enable in .vrooli/lighthouse.json). Setup guide: scenarios/test-genie/docs/phases/performance/lighthouse.md; schema: scenarios/test-genie/schemas/lighthouse.schema.json",
		)
		return &AuditResult{
			Result: OK().WithObservations(
				NewSkipObservation("lighthouse audits disabled"),
				NewInfoObservation("Enable Lighthouse in .vrooli/lighthouse.json; see scenarios/test-genie/docs/phases/performance/lighthouse.md (schema: scenarios/test-genie/schemas/lighthouse.schema.json)"),
			),
			Skipped: true,
		}
	}

	if len(v.config.Config.Pages) == 0 {
		logInfo(v.logWriter, "No pages configured for Lighthouse audits")
		return &AuditResult{
			Result:  OK().WithObservations(NewSkipObservation("no pages configured")),
			Skipped: true,
		}
	}

	// CRITICAL: Check if UI URL is configured. We explicitly skip rather than falling back
	// to a hardcoded URL like localhost:3000, which would silently audit the wrong thing.
	if v.config.BaseURL == "" {
		logInfo(v.logWriter, "Lighthouse audits skipped: no UI URL configured")
		return &AuditResult{
			Result: OK().WithObservations(
				NewSkipObservation("no UI URL configured"),
				NewInfoObservation("To enable Lighthouse: ensure scenario UI is running and UIURL is passed to the performance phase"),
			),
			Skipped: true,
		}
	}

	// Validate config
	if err := v.config.Config.Validate(); err != nil {
		return &AuditResult{
			Result: FailMisconfiguration(err, "Fix .vrooli/lighthouse.json configuration."),
		}
	}

	// Check Lighthouse CLI availability
	logInfo(v.logWriter, "Checking Lighthouse CLI availability")
	if err := v.client.Health(ctx); err != nil {
		logError(v.logWriter, "Lighthouse CLI not available: %v", err)
		return &AuditResult{
			Result: FailUnavailable(
				fmt.Errorf("lighthouse CLI not available: %w", err),
				"Install Lighthouse CLI: npm install -g lighthouse (requires Node.js and Chrome)",
			),
		}
	}

	// Run audits for each page
	var observations []Observation
	var pageResults []PageResult
	allPassed := true
	maxRetries := v.config.Config.GetRetries()

	for _, page := range v.config.Config.Pages {
		pageURL := v.buildPageURL(page.Path)
		logInfo(v.logWriter, "Auditing page %q at %s", page.ID, pageURL)

		// Build page-specific config (handles viewport, etc.)
		lighthouseConfig := v.config.Config.BuildPageLighthouseConfig(page)

		start := time.Now()
		pageResult := v.auditPageWithRetry(ctx, page, pageURL, lighthouseConfig, maxRetries)
		pageResult.DurationMs = time.Since(start).Milliseconds()

		pageResults = append(pageResults, pageResult)

		if !pageResult.Success {
			allPassed = false
		}

		// Build observations for this page
		observations = append(observations, v.buildPageObservations(pageResult)...)
	}

	result := &AuditResult{
		PageResults: pageResults,
	}

	// Write artifacts if configured
	v.writeArtifacts(result)

	if !allPassed {
		// Check if we should fail on error (default true)
		shouldFail := v.config.Config.ShouldFailOnError()

		// Find first error
		var firstErr error
		for _, pr := range pageResults {
			if !pr.Success {
				if pr.Error != nil {
					firstErr = pr.Error
				} else if len(pr.Violations) > 0 {
					firstErr = fmt.Errorf("page %q: %s below threshold", pr.PageID, pr.Violations[0].Category)
				}
				break
			}
		}

		if shouldFail {
			result.Result = FailSystem(firstErr, "Improve Lighthouse scores to meet configured thresholds.")
			result.Observations = observations
			return result
		}

		// FailOnError is false - report as warning but don't fail
		logInfo(v.logWriter, "Lighthouse thresholds not met but fail_on_error is false")
		observations = append(observations, NewWarningObservation("Lighthouse thresholds not met (fail_on_error: false)"))
		result.Result = OK().WithObservations(observations...)
		return result
	}

	logSuccess(v.logWriter, "All Lighthouse audits passed")
	result.Result = OK().WithObservations(observations...)
	return result
}

// writeArtifacts writes all configured artifacts.
func (v *validator) writeArtifacts(result *AuditResult) {
	if v.artifactWriter == nil {
		return
	}
	if result == nil || result.Skipped || len(result.PageResults) == 0 {
		return
	}

	cfg := v.config.Config
	if cfg == nil {
		return
	}

	// Write JSON reports for each page if configured
	if cfg.ShouldGenerateReport("json") {
		for _, pr := range result.PageResults {
			if len(pr.RawResponse) > 0 {
				path, err := v.artifactWriter.WritePageReport(pr.PageID, pr.RawResponse)
				if err != nil {
					logError(v.logWriter, "Failed to write page report for %q: %v", pr.PageID, err)
				} else if path != "" {
					logInfo(v.logWriter, "Wrote page report: %s", path)
				}
			}
		}

		// Write summary
		path, err := v.artifactWriter.WriteSummary(result)
		if err != nil {
			logError(v.logWriter, "Failed to write summary: %v", err)
		} else if path != "" {
			logInfo(v.logWriter, "Wrote summary: %s", path)
		}
	}

	// Write HTML reports for each page if configured
	if cfg.ShouldGenerateReport("html") {
		for _, pr := range result.PageResults {
			if len(pr.RawHTMLResponse) > 0 {
				path, err := v.artifactWriter.WriteHTMLReport(pr.PageID, pr.RawHTMLResponse)
				if err != nil {
					logError(v.logWriter, "Failed to write HTML report for %q: %v", pr.PageID, err)
				} else if path != "" {
					logInfo(v.logWriter, "Wrote HTML report: %s", path)
				}
			}
		}
	}

	// Always write phase results for business phase integration
	if err := v.artifactWriter.WritePhaseResults(result); err != nil {
		logError(v.logWriter, "Failed to write phase results: %v", err)
	}
}

// buildPageURL constructs the full URL for a page path.
// NOTE: This assumes BaseURL is non-empty (checked in Audit before reaching here).
func (v *validator) buildPageURL(path string) string {
	baseURL := v.config.BaseURL

	// Parse and join properly
	base, err := url.Parse(baseURL)
	if err != nil {
		return baseURL + path
	}

	pagePath, err := url.Parse(path)
	if err != nil {
		return baseURL + path
	}

	return base.ResolveReference(pagePath).String()
}

// auditPageWithRetry runs a page audit with optional retries for transient failures.
func (v *validator) auditPageWithRetry(ctx context.Context, page PageConfig, pageURL string, lighthouseConfig map[string]interface{}, maxRetries int) PageResult {
	var lastResult PageResult

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			logInfo(v.logWriter, "Retrying page %q (attempt %d/%d)", page.ID, attempt+1, maxRetries+1)
			// Brief pause between retries
			time.Sleep(time.Second)
		}

		lastResult = v.auditPage(ctx, page, pageURL, lighthouseConfig)

		// If success or a threshold violation (not a transient error), don't retry
		if lastResult.Success || lastResult.Error == nil {
			break
		}

		// Only retry on transient errors (network issues, timeouts, etc.)
		// Threshold violations are not retriable - the page content/scores are stable
		if attempt < maxRetries {
			logInfo(v.logWriter, "Page %q failed with error, will retry: %v", page.ID, lastResult.Error)
		}
	}

	if lastResult.RetryCount == 0 && maxRetries > 0 && !lastResult.Success && lastResult.Error != nil {
		// Record how many retries were used (only if we actually retried)
		// This is already tracked via the loop, but we track final count
	}

	return lastResult
}

// auditPage runs a single page audit.
func (v *validator) auditPage(ctx context.Context, page PageConfig, pageURL string, lighthouseConfig map[string]interface{}) PageResult {
	result := PageResult{
		PageID:       page.ID,
		URL:          pageURL,
		Scores:       make(map[string]float64),
		Requirements: page.Requirements,
	}

	// Build the audit request with wait options
	req := AuditRequest{
		URL:    pageURL,
		Config: lighthouseConfig,
	}

	// Check if HTML output is requested
	if v.config.Config != nil && v.config.Config.ShouldGenerateReport("html") {
		req.IncludeHTML = true
	}

	// Add gotoOptions if waitForMs is set
	if page.WaitForMs > 0 {
		req.GotoOptions = &GotoOptions{
			Timeout: page.WaitForMs + 30000, // Add 30s buffer to the wait time
		}
	}

	resp, err := v.client.Audit(ctx, req)
	if err != nil {
		logError(v.logWriter, "Audit failed for page %q: %v", page.ID, err)
		result.Error = err
		result.Success = false
		return result
	}

	// Store raw responses for potential report generation
	result.RawResponse = resp.Raw
	result.RawHTMLResponse = resp.RawHTML

	// Extract scores
	for catName, catResult := range resp.Categories {
		if catResult.Score != nil {
			result.Scores[catName] = *catResult.Score
		}
	}

	// Check thresholds
	result.Success = true
	for category, threshold := range page.Thresholds {
		score, ok := result.Scores[category]
		if !ok {
			// Category not in response - skip
			continue
		}

		if score < threshold.Error {
			result.Violations = append(result.Violations, CategoryViolation{
				Category:  category,
				Score:     score,
				Threshold: threshold.Error,
				Level:     "error",
			})
			result.Success = false
		} else if score < threshold.Warn {
			result.Warnings = append(result.Warnings, CategoryViolation{
				Category:  category,
				Score:     score,
				Threshold: threshold.Warn,
				Level:     "warn",
			})
		}
	}

	if result.Success {
		logSuccess(v.logWriter, "Page %q passed: %s", page.ID, result.ScoreSummary())
	} else {
		logError(v.logWriter, "Page %q failed: %s", page.ID, result.ScoreSummary())
	}

	return result
}

// buildPageObservations creates observations for a page result.
func (v *validator) buildPageObservations(pr PageResult) []Observation {
	var observations []Observation

	if pr.Error != nil {
		observations = append(observations,
			NewErrorObservation(fmt.Sprintf("page %q: audit error - %v", pr.PageID, pr.Error)))
		return observations
	}

	// Add score summary
	if pr.Success {
		msg := fmt.Sprintf("page %q: %s", pr.PageID, pr.ScoreSummary())
		observations = append(observations, NewSuccessObservation(msg))
	}

	// Add violations
	for _, violation := range pr.Violations {
		observations = append(observations,
			NewErrorObservation(fmt.Sprintf("page %q: %s", pr.PageID, violation.String())))
	}

	// Add warnings
	for _, warning := range pr.Warnings {
		observations = append(observations,
			NewWarningObservation(fmt.Sprintf("page %q: %s", pr.PageID, warning.String())))
	}

	return observations
}

// Logging helpers

func logInfo(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "   %s\n", msg)
}

func logSuccess(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[SUCCESS] %s\n", msg)
}

func logError(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[ERROR] %s\n", msg)
}
