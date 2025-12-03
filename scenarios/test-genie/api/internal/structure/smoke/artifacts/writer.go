package artifacts

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"test-genie/internal/structure/smoke/orchestrator"
)

// Writer persists test artifacts to the filesystem.
type Writer struct {
	fs FileSystem
}

// FileSystem abstracts filesystem operations for testing.
type FileSystem interface {
	WriteFile(path string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
}

// OSFileSystem is the default filesystem implementation using os package.
type OSFileSystem struct{}

// WriteFile writes data to a file.
func (OSFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// MkdirAll creates a directory and all parents.
func (OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// NewWriter creates a new artifact Writer.
func NewWriter(opts ...WriterOption) *Writer {
	w := &Writer{
		fs: OSFileSystem{},
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// WriterOption configures a Writer.
type WriterOption func(*Writer)

// WithFileSystem sets a custom filesystem implementation.
func WithFileSystem(fs FileSystem) WriterOption {
	return func(w *Writer) {
		w.fs = fs
	}
}

// coverageDir returns the coverage directory path for UI smoke artifacts.
func coverageDir(scenarioDir, scenarioName string) string {
	return filepath.Join(scenarioDir, "coverage", scenarioName, "ui-smoke")
}

// Ensure Writer implements orchestrator.ArtifactWriter.
var _ orchestrator.ArtifactWriter = (*Writer)(nil)

// WriteAll writes all artifacts and returns their paths.
func (w *Writer) WriteAll(ctx context.Context, scenarioDir, scenarioName string, response *orchestrator.BrowserResponse) (*orchestrator.ArtifactPaths, error) {
	dir := coverageDir(scenarioDir, scenarioName)
	if err := w.fs.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create coverage directory: %w", err)
	}

	paths := &orchestrator.ArtifactPaths{}

	// Write screenshot
	if response.Screenshot != "" {
		screenshotPath := filepath.Join(dir, "screenshot.png")
		decoded, err := base64.StdEncoding.DecodeString(response.Screenshot)
		if err != nil {
			return nil, fmt.Errorf("failed to decode screenshot: %w", err)
		}
		if err := w.fs.WriteFile(screenshotPath, decoded, 0o644); err != nil {
			return nil, fmt.Errorf("failed to write screenshot: %w", err)
		}
		paths.Screenshot = relPath(scenarioDir, screenshotPath)
	}

	// Write console logs
	if len(response.Console) > 0 {
		consolePath := filepath.Join(dir, "console.json")
		data, err := json.MarshalIndent(response.Console, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal console: %w", err)
		}
		if err := w.fs.WriteFile(consolePath, data, 0o644); err != nil {
			return nil, fmt.Errorf("failed to write console: %w", err)
		}
		paths.Console = relPath(scenarioDir, consolePath)
	}

	// Write network failures
	if len(response.Network) > 0 {
		networkPath := filepath.Join(dir, "network.json")
		data, err := json.MarshalIndent(response.Network, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal network: %w", err)
		}
		if err := w.fs.WriteFile(networkPath, data, 0o644); err != nil {
			return nil, fmt.Errorf("failed to write network: %w", err)
		}
		paths.Network = relPath(scenarioDir, networkPath)
	}

	// Write DOM snapshot
	if response.HTML != "" {
		htmlPath := filepath.Join(dir, "dom.html")
		if err := w.fs.WriteFile(htmlPath, []byte(response.HTML), 0o644); err != nil {
			return nil, fmt.Errorf("failed to write html: %w", err)
		}
		paths.HTML = relPath(scenarioDir, htmlPath)
	}

	// Write raw response (without screenshot)
	if len(response.Raw) > 0 {
		rawPath := filepath.Join(dir, "raw.json")
		// Pretty-print the raw JSON
		var obj interface{}
		if err := json.Unmarshal(response.Raw, &obj); err == nil {
			prettyData, _ := json.MarshalIndent(obj, "", "  ")
			if err := w.fs.WriteFile(rawPath, prettyData, 0o644); err != nil {
				return nil, fmt.Errorf("failed to write raw: %w", err)
			}
		} else {
			if err := w.fs.WriteFile(rawPath, response.Raw, 0o644); err != nil {
				return nil, fmt.Errorf("failed to write raw: %w", err)
			}
		}
		paths.Raw = relPath(scenarioDir, rawPath)
	}

	return paths, nil
}

// WriteResultJSON writes a result object as JSON.
func (w *Writer) WriteResultJSON(ctx context.Context, scenarioDir, scenarioName string, result interface{}) error {
	dir := coverageDir(scenarioDir, scenarioName)
	if err := w.fs.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create coverage directory: %w", err)
	}

	resultPath := filepath.Join(dir, "latest.json")
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := w.fs.WriteFile(resultPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write result: %w", err)
	}

	return nil
}

// WriteReadme generates a README.md summarizing the test results.
func (w *Writer) WriteReadme(ctx context.Context, scenarioDir, scenarioName string, result *orchestrator.Result) error {
	dir := coverageDir(scenarioDir, scenarioName)
	if err := w.fs.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create coverage directory: %w", err)
	}

	readmePath := filepath.Join(dir, "README.md")
	content := generateReadme(result)
	if err := w.fs.WriteFile(readmePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write README: %w", err)
	}

	return nil
}

// generateReadme creates the README.md content for a UI smoke test result.
func generateReadme(result *orchestrator.Result) string {
	var b strings.Builder

	// Header
	statusEmoji := "✅"
	switch result.Status {
	case orchestrator.StatusFailed:
		statusEmoji = "❌"
	case orchestrator.StatusSkipped:
		statusEmoji = "⏭️"
	case orchestrator.StatusBlocked:
		statusEmoji = "🚫"
	}

	b.WriteString(fmt.Sprintf("# %s UI Smoke Test Results\n\n", result.Scenario))
	b.WriteString(fmt.Sprintf("**Status:** %s %s\n\n", statusEmoji, result.Status))

	// Test Info
	b.WriteString("## Test Information\n\n")
	b.WriteString(fmt.Sprintf("| Property | Value |\n"))
	b.WriteString(fmt.Sprintf("|----------|-------|\n"))
	b.WriteString(fmt.Sprintf("| Scenario | `%s` |\n", result.Scenario))
	b.WriteString(fmt.Sprintf("| Timestamp | %s |\n", result.Timestamp.Format("2006-01-02 15:04:05 UTC")))
	if result.DurationMs > 0 {
		b.WriteString(fmt.Sprintf("| Duration | %dms |\n", result.DurationMs))
	}
	if result.UIURL != "" {
		b.WriteString(fmt.Sprintf("| URL Tested | %s |\n", result.UIURL))
	}
	b.WriteString("\n")

	// Result Message
	b.WriteString("## Result\n\n")
	if result.Message != "" {
		b.WriteString(fmt.Sprintf("%s\n\n", result.Message))
	}

	// Handshake Status (if applicable)
	if result.UIURL != "" {
		b.WriteString("## Handshake Status\n\n")
		if result.Handshake.Signaled {
			b.WriteString(fmt.Sprintf("✅ **iframe-bridge signaled ready** in %dms\n\n", result.Handshake.DurationMs))
		} else if result.Handshake.TimedOut {
			b.WriteString(fmt.Sprintf("⏱️ **Handshake timed out** after %dms\n\n", result.Handshake.DurationMs))
			b.WriteString("The UI failed to signal readiness via the iframe-bridge. This could indicate:\n")
			b.WriteString("- The `@vrooli/iframe-bridge` package is not properly integrated\n")
			b.WriteString("- JavaScript errors prevented the handshake from completing\n")
			b.WriteString("- Network issues blocked required resources\n\n")
		} else if result.Handshake.Error != "" {
			b.WriteString(fmt.Sprintf("❌ **Handshake error:** %s\n\n", result.Handshake.Error))
		}
	}

	// Bundle Status (if applicable)
	if result.Bundle != nil {
		b.WriteString("## Bundle Status\n\n")
		if result.Bundle.Fresh {
			b.WriteString("✅ **UI bundle is fresh** - No stale build artifacts detected\n\n")
		} else {
			b.WriteString(fmt.Sprintf("⚠️ **UI bundle is stale:** %s\n\n", result.Bundle.Reason))
			b.WriteString("Run `vrooli scenario restart <scenario>` to rebuild the UI bundle.\n\n")
		}
	}

	// Artifacts Section
	b.WriteString("## Collected Artifacts\n\n")
	hasArtifacts := false

	if result.Artifacts.Screenshot != "" {
		hasArtifacts = true
		b.WriteString("### Screenshot\n\n")
		b.WriteString("A screenshot of the UI at the time of test completion.\n\n")
		b.WriteString(fmt.Sprintf("📷 [screenshot.png](./%s)\n\n", filepath.Base(result.Artifacts.Screenshot)))
	}

	if result.Artifacts.Console != "" {
		hasArtifacts = true
		b.WriteString("### Console Logs\n\n")
		b.WriteString("Browser console output captured during the test (errors, warnings, logs).\n\n")
		b.WriteString(fmt.Sprintf("📋 [console.json](./%s)\n\n", filepath.Base(result.Artifacts.Console)))
	}

	if result.Artifacts.Network != "" {
		hasArtifacts = true
		b.WriteString("### Network Failures\n\n")
		b.WriteString("Failed network requests detected during page load (4xx/5xx responses, timeouts).\n\n")
		b.WriteString(fmt.Sprintf("🌐 [network.json](./%s)\n\n", filepath.Base(result.Artifacts.Network)))
	}

	if result.Artifacts.HTML != "" {
		hasArtifacts = true
		b.WriteString("### DOM Snapshot\n\n")
		b.WriteString("The complete HTML of the page at test completion.\n\n")
		b.WriteString(fmt.Sprintf("📄 [dom.html](./%s)\n\n", filepath.Base(result.Artifacts.HTML)))
	}

	if result.Artifacts.Raw != "" {
		hasArtifacts = true
		b.WriteString("### Raw Response\n\n")
		b.WriteString("The complete raw response from Browserless (useful for debugging).\n\n")
		b.WriteString(fmt.Sprintf("🔧 [raw.json](./%s)\n\n", filepath.Base(result.Artifacts.Raw)))
	}

	if !hasArtifacts {
		b.WriteString("*No artifacts were collected for this test run.*\n\n")
		if result.Status == orchestrator.StatusSkipped {
			b.WriteString("This is expected for skipped tests (e.g., no UI port detected).\n\n")
		}
	}

	// Troubleshooting Section (for failures)
	if result.Status == orchestrator.StatusFailed || result.Status == orchestrator.StatusBlocked {
		b.WriteString("## Troubleshooting\n\n")

		switch {
		case result.Status == orchestrator.StatusBlocked:
			b.WriteString("### Blocked Test\n\n")
			b.WriteString("The test could not run due to a prerequisite issue:\n\n")
			b.WriteString(fmt.Sprintf("- %s\n\n", result.Message))

		case result.Handshake.TimedOut:
			b.WriteString("### Handshake Timeout\n\n")
			b.WriteString("1. Check if `@vrooli/iframe-bridge` is installed in `ui/package.json`\n")
			b.WriteString("2. Verify the bridge is initialized in your app's entry point\n")
			b.WriteString("3. Check the console.json for JavaScript errors\n")
			b.WriteString("4. Ensure no network requests are blocking the initial render\n\n")

		default:
			b.WriteString("### General Debugging Steps\n\n")
			b.WriteString("1. Review the screenshot to see the visual state\n")
			b.WriteString("2. Check console.json for JavaScript errors\n")
			b.WriteString("3. Check network.json for failed requests\n")
			b.WriteString("4. Inspect dom.html for unexpected page content\n")
			b.WriteString("5. Restart the scenario: `vrooli scenario restart <scenario>`\n\n")
		}
	}

	// Footer
	b.WriteString("---\n\n")
	b.WriteString("*Generated by test-genie UI smoke test*\n")

	return b.String()
}

// relPath returns a relative path from baseDir to path.
// If relativization fails, returns the absolute path.
func relPath(baseDir, path string) string {
	rel, err := filepath.Rel(baseDir, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}
