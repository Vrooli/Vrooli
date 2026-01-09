package lighthouse

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CLIRunner implements the Client interface using the official Lighthouse CLI.
// This is the recommended approach as it uses Google's canonical implementation.
type CLIRunner struct {
	timeout    time.Duration
	chromePath string // Optional: custom Chrome path (uses CHROME_PATH env or auto-detect if empty)
}

// CLIRunnerOption configures a CLIRunner.
type CLIRunnerOption func(*CLIRunner)

// NewCLIRunner creates a new Lighthouse CLI runner.
func NewCLIRunner(opts ...CLIRunnerOption) *CLIRunner {
	r := &CLIRunner{
		timeout: 120 * time.Second,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// WithCLITimeout sets the timeout for Lighthouse CLI execution.
func WithCLITimeout(d time.Duration) CLIRunnerOption {
	return func(r *CLIRunner) {
		r.timeout = d
	}
}

// WithChromePath sets a custom Chrome executable path.
func WithChromePath(path string) CLIRunnerOption {
	return func(r *CLIRunner) {
		r.chromePath = path
	}
}

// Health checks if Lighthouse CLI is available and Chrome can be found.
func (r *CLIRunner) Health(ctx context.Context) error {
	// First, check if lighthouse command is available
	lighthousePath, err := r.findLighthouse(ctx)
	if err != nil {
		return fmt.Errorf("lighthouse CLI not found: %w. Install with: npm install -g lighthouse", err)
	}

	// Verify lighthouse works by checking version
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if lighthousePath == "npx" {
		cmd = exec.CommandContext(ctx, "npx", "lighthouse", "--version")
	} else {
		cmd = exec.CommandContext(ctx, lighthousePath, "--version")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("lighthouse --version failed: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}

	// Chrome availability is checked by Lighthouse itself during audit
	// We don't need to verify it here - Lighthouse will give a clear error if Chrome is missing

	return nil
}

// Audit runs a Lighthouse audit using the CLI.
func (r *CLIRunner) Audit(ctx context.Context, req AuditRequest) (*AuditResponse, error) {
	if req.URL == "" {
		return nil, errors.New("URL is required for Lighthouse audit")
	}

	// Find lighthouse
	lighthousePath, err := r.findLighthouse(ctx)
	if err != nil {
		return nil, fmt.Errorf("lighthouse CLI not found: %w", err)
	}

	// Create temp directory for outputs
	tempDir, err := os.MkdirTemp("", "lighthouse-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Base path for output files (Lighthouse appends extensions)
	outputBasePath := filepath.Join(tempDir, "report")

	// Build command arguments
	args := r.buildArgs(req, outputBasePath)

	// Create command with timeout
	timeout := r.timeout
	if req.GotoOptions != nil && req.GotoOptions.Timeout > 0 {
		// Add gotoOptions timeout to our timeout
		timeout += time.Duration(req.GotoOptions.Timeout) * time.Millisecond
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	if lighthousePath == "npx" {
		fullArgs := append([]string{"lighthouse"}, args...)
		cmd = exec.CommandContext(ctx, "npx", fullArgs...)
	} else {
		cmd = exec.CommandContext(ctx, lighthousePath, args...)
	}

	// Set Chrome path if configured
	if r.chromePath != "" {
		cmd.Env = append(os.Environ(), "CHROME_PATH="+r.chromePath)
	}

	// Capture stderr for error messages
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Run the audit
	if err := cmd.Run(); err != nil {
		// Check if it's a context timeout
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("lighthouse audit timed out after %v", timeout)
		}
		// Include stderr in error message for debugging
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return nil, fmt.Errorf("lighthouse audit failed: %w\nstderr: %s", err, truncateString(stderrStr, 500))
		}
		return nil, fmt.Errorf("lighthouse audit failed: %w", err)
	}

	// Read the JSON output
	// Lighthouse naming convention (as of v12.x):
	// - Single output (json): uses exact outputPath (no extension added)
	// - Multiple outputs (json,html): outputPath.report.json and outputPath.report.html
	var jsonPath string
	if req.IncludeHTML {
		jsonPath = outputBasePath + ".report.json"
	} else {
		jsonPath = outputBasePath // Lighthouse uses exact path for single output
	}

	jsonData, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read lighthouse JSON output: %w", err)
	}

	response, err := parseLighthouseOutput(jsonData)
	if err != nil {
		return nil, err
	}

	// Read HTML output if requested
	if req.IncludeHTML {
		htmlPath := outputBasePath + ".report.html"
		htmlData, err := os.ReadFile(htmlPath)
		if err != nil {
			// HTML read failure is not fatal - log but continue
			// The HTML might not exist if Lighthouse had issues
		} else {
			response.RawHTML = htmlData
		}
	}

	return response, nil
}

// findLighthouse locates the lighthouse CLI.
// Returns "npx" if lighthouse should be run via npx, or the direct path to lighthouse.
func (r *CLIRunner) findLighthouse(ctx context.Context) (string, error) {
	// First, try to find lighthouse directly in PATH
	if path, err := exec.LookPath("lighthouse"); err == nil {
		return path, nil
	}

	// Fall back to npx
	if _, err := exec.LookPath("npx"); err == nil {
		// Verify npx can find lighthouse (don't actually run it, just check)
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		// Use npx --no to avoid installing if not present
		cmd := exec.CommandContext(ctx, "npx", "--no", "lighthouse", "--version")
		if err := cmd.Run(); err == nil {
			return "npx", nil
		}

		// npx exists but lighthouse isn't installed - still return npx
		// as it will auto-install lighthouse on first run
		return "npx", nil
	}

	return "", errors.New("neither 'lighthouse' nor 'npx' found in PATH")
}

// buildArgs constructs CLI arguments from the audit request.
func (r *CLIRunner) buildArgs(req AuditRequest, outputBasePath string) []string {
	// Determine output formats
	outputFormat := "json"
	if req.IncludeHTML {
		outputFormat = "json,html"
	}

	args := []string{
		req.URL,
		"--output=" + outputFormat,
		"--output-path=" + outputBasePath,
		"--chrome-flags=--headless --no-sandbox --disable-gpu",
	}

	// Extract settings from config
	if req.Config != nil {
		if settings, ok := req.Config["settings"].(map[string]interface{}); ok {
			// Categories
			if categories, ok := settings["onlyCategories"].([]string); ok && len(categories) > 0 {
				args = append(args, "--only-categories="+strings.Join(categories, ","))
			} else if categories, ok := settings["onlyCategories"].([]interface{}); ok && len(categories) > 0 {
				cats := make([]string, len(categories))
				for i, c := range categories {
					cats[i] = fmt.Sprint(c)
				}
				args = append(args, "--only-categories="+strings.Join(cats, ","))
			}

			// Form factor / preset
			if formFactor, ok := settings["formFactor"].(string); ok {
				if formFactor == "desktop" {
					args = append(args, "--preset=desktop")
				} else if formFactor == "mobile" {
					// Mobile is the default, but we can be explicit
					args = append(args, "--form-factor=mobile")
				}
			}

			// Throttling method
			if throttling, ok := settings["throttlingMethod"].(string); ok {
				args = append(args, "--throttling-method="+throttling)
			}

			// Screen emulation for mobile
			if screenEmu, ok := settings["screenEmulation"].(map[string]interface{}); ok {
				if mobile, ok := screenEmu["mobile"].(bool); ok && mobile {
					args = append(args, "--screenEmulation.mobile=true")
				}
				if width, ok := screenEmu["width"].(float64); ok {
					args = append(args, fmt.Sprintf("--screenEmulation.width=%d", int(width)))
				}
				if height, ok := screenEmu["height"].(float64); ok {
					args = append(args, fmt.Sprintf("--screenEmulation.height=%d", int(height)))
				}
			}
		}
	}

	return args
}

// parseLighthouseOutput parses the JSON output from Lighthouse CLI.
func parseLighthouseOutput(data []byte) (*AuditResponse, error) {
	if len(data) == 0 {
		return nil, errors.New("empty lighthouse output")
	}

	// Lighthouse CLI outputs the LHR (Lighthouse Result) directly
	// Structure: { "categories": {...}, "audits": {...}, ... }
	var lhr struct {
		Categories map[string]struct {
			ID    string   `json:"id"`
			Title string   `json:"title"`
			Score *float64 `json:"score"`
		} `json:"categories"`
		Audits json.RawMessage `json:"audits"`
	}

	if err := json.Unmarshal(data, &lhr); err != nil {
		return nil, fmt.Errorf("failed to parse lighthouse output: %w (first 200 chars: %s)",
			err, truncateString(string(data), 200))
	}

	// Convert to our response format
	response := &AuditResponse{
		Categories: make(map[string]CategoryResult),
		Raw:        data,
	}

	for catID, cat := range lhr.Categories {
		response.Categories[catID] = CategoryResult{
			ID:    cat.ID,
			Title: cat.Title,
			Score: cat.Score,
		}
	}

	response.Audits = lhr.Audits

	return response, nil
}

// truncateString truncates a string for error messages.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Ensure CLIRunner implements Client.
var _ Client = (*CLIRunner)(nil)
