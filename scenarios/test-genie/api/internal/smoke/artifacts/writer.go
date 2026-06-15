// Package artifacts persists UI smoke artifacts (screenshot, console, network,
// raw evidence) and the human-readable summary under coverage/runs/<runID>/. It
// is a dependency-light leaf: it accepts engine-agnostic input types so it never
// depends on the smoke runner or the browser engine.
package artifacts

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	sharedartifacts "test-genie/internal/shared/artifacts"
)

// ConsoleEntry is one captured browser console message.
type ConsoleEntry struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

// NetworkEntry is one failed network request.
type NetworkEntry struct {
	URL          string `json:"url"`
	Method       string `json:"method,omitempty"`
	ResourceType string `json:"resource_type,omitempty"`
	Status       *int   `json:"status,omitempty"`
	ErrorText    string `json:"error_text,omitempty"`
}

// Input carries the observations to persist for one smoke capture.
type Input struct {
	// Screenshot is the raw PNG bytes of the captured frame (may be empty).
	Screenshot []byte
	// Console is the captured console output.
	Console []ConsoleEntry
	// Network is the captured network failures.
	Network []NetworkEntry
	// Raw is the raw evidence JSON for diagnostics (may be empty).
	Raw json.RawMessage
}

// Paths records where each artifact was written (absolute paths).
type Paths struct {
	Screenshot string `json:"screenshot,omitempty"`
	Console    string `json:"console,omitempty"`
	Network    string `json:"network,omitempty"`
	Raw        string `json:"raw,omitempty"`
	Readme     string `json:"readme,omitempty"`
}

// Summary is the engine-agnostic smoke outcome used to render the README and the
// phase-results pointer. It mirrors the fields the smoke Result carries without
// creating an import cycle back into the smoke package.
type Summary struct {
	Scenario     string
	Status       string
	Message      string
	Timestamp    time.Time
	DurationMs   int64
	UIURL        string
	Handshake    HandshakeSummary
	BundleFresh  bool
	BundleReason string
	BundleKnown  bool
	Paths        Paths
}

// HandshakeSummary is the handshake outcome for the README.
type HandshakeSummary struct {
	Signaled   bool
	TimedOut   bool
	DurationMs int64
	Error      string
}

// Writer persists smoke artifacts under coverage/runs/<runID>/.
type Writer struct {
	fs    sharedartifacts.FileSystem
	runID string
}

// NewWriter creates a new artifact Writer.
func NewWriter(opts ...WriterOption) *Writer {
	w := &Writer{
		fs: sharedartifacts.OSFileSystem{},
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// WriterOption configures a Writer.
type WriterOption func(*Writer)

// WithFileSystem sets a custom filesystem implementation.
func WithFileSystem(fs sharedartifacts.FileSystem) WriterOption {
	return func(w *Writer) {
		w.fs = fs
	}
}

// WithRunID keys all artifacts under coverage/runs/<runID>/.
func WithRunID(runID string) WriterOption {
	return func(w *Writer) {
		w.runID = runID
	}
}

// coverageDir returns the run-keyed UI smoke artifacts directory.
func (w *Writer) coverageDir(scenarioDir string) string {
	return sharedartifacts.RunUISmokeDir(scenarioDir, w.runID)
}

// WriteAll writes the screenshot, console, network, and raw artifacts and
// returns their paths.
func (w *Writer) WriteAll(ctx context.Context, scenarioDir, scenarioName string, in Input) (*Paths, error) {
	dir := w.coverageDir(scenarioDir)
	if err := w.fs.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create coverage directory: %w", err)
	}

	paths := &Paths{}

	if len(in.Screenshot) > 0 {
		screenshotPath := filepath.Join(dir, "screenshot.png")
		if err := w.fs.WriteFile(screenshotPath, in.Screenshot, 0o644); err != nil {
			return nil, fmt.Errorf("failed to write screenshot: %w", err)
		}
		paths.Screenshot = sharedartifacts.AbsPath(screenshotPath)
	}

	if len(in.Console) > 0 {
		consolePath := filepath.Join(dir, "console.json")
		data, err := json.MarshalIndent(in.Console, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal console: %w", err)
		}
		if err := w.fs.WriteFile(consolePath, data, 0o644); err != nil {
			return nil, fmt.Errorf("failed to write console: %w", err)
		}
		paths.Console = sharedartifacts.AbsPath(consolePath)
	}

	// Write network failures always (even when empty) for visibility.
	networkPath := filepath.Join(dir, "network.json")
	networkData := in.Network
	if networkData == nil {
		networkData = []NetworkEntry{}
	}
	data, err := json.MarshalIndent(networkData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal network: %w", err)
	}
	if err := w.fs.WriteFile(networkPath, data, 0o644); err != nil {
		return nil, fmt.Errorf("failed to write network: %w", err)
	}
	paths.Network = sharedartifacts.AbsPath(networkPath)

	if len(in.Raw) > 0 {
		rawPath := filepath.Join(dir, "raw.json")
		var obj interface{}
		if err := json.Unmarshal(in.Raw, &obj); err == nil {
			prettyData, _ := json.MarshalIndent(obj, "", "  ")
			if err := w.fs.WriteFile(rawPath, prettyData, 0o644); err != nil {
				return nil, fmt.Errorf("failed to write raw: %w", err)
			}
		} else if err := w.fs.WriteFile(rawPath, in.Raw, 0o644); err != nil {
			return nil, fmt.Errorf("failed to write raw: %w", err)
		}
		paths.Raw = sharedartifacts.AbsPath(rawPath)
	}

	return paths, nil
}

// WriteResultJSON writes the smoke result summary as latest.json and drops the
// phase-results pointer.
func (w *Writer) WriteResultJSON(ctx context.Context, scenarioDir, scenarioName string, result any, summary Summary) error {
	dir := w.coverageDir(scenarioDir)
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

	return w.writePhasePointer(scenarioDir, scenarioName, summary)
}

// WriteReadme generates a README.md summarizing the smoke result and returns its
// absolute path.
func (w *Writer) WriteReadme(ctx context.Context, scenarioDir, scenarioName string, summary Summary) (string, error) {
	dir := w.coverageDir(scenarioDir)
	if err := w.fs.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create coverage directory: %w", err)
	}

	readmePath := filepath.Join(dir, "README.md")
	if err := w.fs.WriteFile(readmePath, []byte(generateReadme(summary)), 0o644); err != nil {
		return "", fmt.Errorf("failed to write README: %w", err)
	}
	return sharedartifacts.AbsPath(readmePath), nil
}

// writePhasePointer drops a concise summary into coverage/phase-results so the
// business phase and operators can quickly locate smoke artifacts.
func (w *Writer) writePhasePointer(scenarioDir, scenarioName string, summary Summary) error {
	phaseDir := sharedartifacts.RunPhaseResultsDir(scenarioDir, w.runID)
	if err := w.fs.MkdirAll(phaseDir, 0o755); err != nil {
		return fmt.Errorf("failed to create phase results directory: %w", err)
	}

	payload := map[string]any{
		"phase":      "smoke",
		"scenario":   scenarioName,
		"status":     summary.Status,
		"message":    summary.Message,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}
	if summary.DurationMs > 0 {
		payload["duration_ms"] = summary.DurationMs
	}
	if summary.UIURL != "" {
		payload["ui_url"] = summary.UIURL
	}
	if summary.Paths != (Paths{}) {
		payload["artifacts"] = summary.Paths
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal smoke phase pointer: %w", err)
	}

	path := filepath.Join(phaseDir, sharedartifacts.PhaseResultsSmoke)
	if err := w.fs.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write smoke phase pointer: %w", err)
	}
	return nil
}

// generateReadme renders the README.md content for a smoke result.
func generateReadme(s Summary) string {
	var b strings.Builder

	statusEmoji := "✅"
	switch strings.ToLower(s.Status) {
	case "failed":
		statusEmoji = "❌"
	case "skipped":
		statusEmoji = "⏭️"
	case "blocked":
		statusEmoji = "🚫"
	}

	b.WriteString(fmt.Sprintf("# %s UI Smoke Test Results\n\n", s.Scenario))
	b.WriteString(fmt.Sprintf("**Status:** %s %s\n\n", statusEmoji, s.Status))

	b.WriteString("## Test Information\n\n")
	b.WriteString("| Property | Value |\n")
	b.WriteString("|----------|-------|\n")
	b.WriteString(fmt.Sprintf("| Scenario | `%s` |\n", s.Scenario))
	b.WriteString(fmt.Sprintf("| Timestamp | %s |\n", s.Timestamp.Format("2006-01-02 15:04:05 UTC")))
	if s.DurationMs > 0 {
		b.WriteString(fmt.Sprintf("| Duration | %dms |\n", s.DurationMs))
	}
	if s.UIURL != "" {
		b.WriteString(fmt.Sprintf("| URL Tested | %s |\n", s.UIURL))
	}
	b.WriteString("\n")

	b.WriteString("## Result\n\n")
	if s.Message != "" {
		b.WriteString(fmt.Sprintf("%s\n\n", s.Message))
	}

	if s.UIURL != "" {
		b.WriteString("## Handshake Status\n\n")
		switch {
		case s.Handshake.Signaled:
			b.WriteString(fmt.Sprintf("✅ **iframe-bridge signaled ready** in %dms\n\n", s.Handshake.DurationMs))
		case s.Handshake.TimedOut:
			b.WriteString(fmt.Sprintf("⏱️ **Handshake timed out** after %dms\n\n", s.Handshake.DurationMs))
			b.WriteString("The UI failed to signal readiness via the iframe-bridge. This could indicate:\n")
			b.WriteString("- The `@vrooli/iframe-bridge` package is not properly integrated\n")
			b.WriteString("- JavaScript errors prevented the handshake from completing\n")
			b.WriteString("- Network issues blocked required resources\n\n")
		case s.Handshake.Error != "":
			b.WriteString(fmt.Sprintf("❌ **Handshake error:** %s\n\n", s.Handshake.Error))
		}
	}

	if s.BundleKnown {
		b.WriteString("## Bundle Status\n\n")
		if s.BundleFresh {
			b.WriteString("✅ **UI bundle is fresh** - No stale build artifacts detected\n\n")
		} else {
			b.WriteString(fmt.Sprintf("⚠️ **UI bundle is stale:** %s\n\n", s.BundleReason))
			b.WriteString("Run `vrooli scenario restart <scenario>` to rebuild the UI bundle.\n\n")
		}
	}

	b.WriteString("## Collected Artifacts\n\n")
	hasArtifacts := false
	if s.Paths.Screenshot != "" {
		hasArtifacts = true
		b.WriteString("### Screenshot\n\n")
		b.WriteString("A screenshot of the embedded UI at the time of test completion.\n\n")
		b.WriteString(fmt.Sprintf("📷 [screenshot.png](./%s)\n\n", filepath.Base(s.Paths.Screenshot)))
	}
	if s.Paths.Console != "" {
		hasArtifacts = true
		b.WriteString("### Console Logs\n\n")
		b.WriteString("Browser console output captured during the test (errors, warnings, logs).\n\n")
		b.WriteString(fmt.Sprintf("📋 [console.json](./%s)\n\n", filepath.Base(s.Paths.Console)))
	}
	if s.Paths.Network != "" {
		hasArtifacts = true
		b.WriteString("### Network Failures\n\n")
		b.WriteString("Failed network requests detected during page load (4xx/5xx responses, transport errors).\n\n")
		b.WriteString(fmt.Sprintf("🌐 [network.json](./%s)\n\n", filepath.Base(s.Paths.Network)))
	}
	if s.Paths.Raw != "" {
		hasArtifacts = true
		b.WriteString("### Raw Evidence\n\n")
		b.WriteString("The raw smoke evidence captured from the BAS workflow timeline (useful for debugging).\n\n")
		b.WriteString(fmt.Sprintf("🔧 [raw.json](./%s)\n\n", filepath.Base(s.Paths.Raw)))
	}
	if !hasArtifacts {
		b.WriteString("*No artifacts were collected for this test run.*\n\n")
		if strings.EqualFold(s.Status, "skipped") {
			b.WriteString("This is expected for skipped tests (e.g., no UI port detected).\n\n")
		}
	}

	if strings.EqualFold(s.Status, "failed") || strings.EqualFold(s.Status, "blocked") {
		b.WriteString("## Troubleshooting\n\n")
		switch {
		case strings.EqualFold(s.Status, "blocked"):
			b.WriteString("### Blocked Test\n\n")
			b.WriteString("The test could not run due to a prerequisite issue:\n\n")
			b.WriteString(fmt.Sprintf("- %s\n\n", s.Message))
		case s.Handshake.TimedOut:
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
			b.WriteString("4. Restart the scenario: `vrooli scenario restart <scenario>`\n\n")
		}
	}

	b.WriteString("---\n\n")
	b.WriteString("*Generated by test-genie UI smoke test*\n")
	return b.String()
}
