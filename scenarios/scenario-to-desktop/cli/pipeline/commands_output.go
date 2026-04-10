package pipeline

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func pipelineManifestFilename(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "win", "windows":
		return "latest.yml"
	case "mac", "macos", "darwin":
		return "latest-mac.yml"
	case "linux":
		return "latest-linux.yml"
	default:
		return ""
	}
}

// stderrPattern represents a pattern to match in stderr with associated recovery hint.
type stderrPattern struct {
	pattern  *regexp.Regexp
	hint     string
	category string // For grouping related errors
}

// stderrPatterns contains patterns to match common errors in stderr.
var stderrPatterns = []stderrPattern{
	{
		pattern:  regexp.MustCompile(`no go\.mod|unable to resolve paths for staleness`),
		hint:     "The bundled binary is trying to find Go source files. Ensure VROOLI_API_SKIP_STALE_CHECK=true is set in the bundle environment.",
		category: "go_module",
	},
	{
		pattern:  regexp.MustCompile(`(?i)permission denied`),
		hint:     "Check file permissions. Ensure the artifact is executable: chmod +x <artifact>",
		category: "permissions",
	},
	{
		pattern:  regexp.MustCompile(`(?i)GLIBC.*not found|version.*GLIBC`),
		hint:     "System GLIBC version mismatch. The binary was built for a different Linux version. Rebuild on a compatible system or use a container.",
		category: "glibc",
	},
	{
		pattern:  regexp.MustCompile(`(?i)ENOENT|no such file or directory`),
		hint:     "Required file or dependency not found. Check the bundle contains all required files.",
		category: "missing_file",
	},
	{
		pattern:  regexp.MustCompile(`(?i)EACCES|access denied`),
		hint:     "Access denied. Check permissions and ensure the app isn't blocked by security software.",
		category: "access",
	},
	{
		pattern:  regexp.MustCompile(`(?i)libgtk|libX11|cannot open shared object|libGL`),
		hint:     "Missing system library. Install Electron dependencies: sudo apt-get install libgtk-3-0 libnotify4 libnss3 libxss1",
		category: "shared_lib",
	},
	{
		pattern:  regexp.MustCompile(`(?i)ECONNREFUSED|connection refused`),
		hint:     "Server connection refused. Ensure the target server is running and accessible.",
		category: "connection",
	},
	{
		pattern:  regexp.MustCompile(`(?i)ETIMEDOUT|timeout|timed out`),
		hint:     "Connection or operation timed out. Check network connectivity and increase timeout if needed.",
		category: "timeout",
	},
	{
		pattern:  regexp.MustCompile(`(?i)out of memory|OOM|heap`),
		hint:     "Out of memory. The system may not have enough RAM. Try closing other applications.",
		category: "memory",
	},
	{
		pattern:  regexp.MustCompile(`(?i)segmentation fault|SIGSEGV`),
		hint:     "App crashed with segmentation fault. This may indicate a binary incompatibility or corrupted artifact.",
		category: "crash",
	},
}

// analyzeStderr matches stderr against known patterns and returns a targeted hint.
func analyzeStderr(stderr string) string {
	for _, p := range stderrPatterns {
		if p.pattern.MatchString(stderr) {
			return p.hint
		}
	}
	return ""
}

// lifecycleStateDescription returns a human-readable description of where the failure occurred.
func lifecycleStateDescription(state string) string {
	switch state {
	case "":
		return "App crashed before smoke test initialization code ran. This usually indicates an Electron startup failure or missing dependencies."
	case "init":
		return "App started smoke test but crashed during initialization. A bundled service likely failed to start."
	case "bundle_resolving":
		return "App is locating the bundle directory. Check if bundle is packaged correctly in extraResources."
	case "runtime_starting":
		return "App is spawning the bundled runtime process. Check runtime binary permissions and dependencies."
	case "waiting_for_token":
		return "App is waiting for runtime auth token file. The runtime process started but may not be creating its token. Check if the bundled API supports --token-path flag."
	case "runtime_healthz":
		return "App is waiting for runtime /healthz endpoint. The runtime may still be starting or crashed."
	case "runtime_readyz":
		return "App is waiting for runtime /readyz endpoint. Runtime started but services not ready."
	case "runtime_ports":
		return "App is querying runtime /ports endpoint. Services are ready but port configuration may be wrong."
	case "ui_server_check":
		return "App is verifying the UI server responds with HTTP 2xx. The server is returning an error status code (e.g., 404 Not Found)."
	case "ready":
		return "App initialized but failed during server connectivity check. The target server may not be running or accessible."
	case "result":
		return "App reported a result but crashed during cleanup. This is usually non-fatal."
	case "exit":
		return "App completed the smoke test lifecycle. If still failing, there may be a race condition in result reporting."
	default:
		return fmt.Sprintf("App reached state '%s' before failing.", state)
	}
}

// smokeTestErrorPattern matches SMOKE_TEST_ERROR markers in app output.
// Format: SMOKE_TEST_ERROR kind=<kind> msg="<message>"
var smokeTestErrorPattern = regexp.MustCompile(`SMOKE_TEST_ERROR kind=(\w+) msg="([^"]+)"`)

// smokeTestStagePattern matches SMOKE_TEST_STAGE markers in app output.
var smokeTestStagePattern = regexp.MustCompile(`SMOKE_TEST_STAGE=(\w+)`)

// extractProgressStages extracts SMOKE_TEST_STAGE markers from stdout.
// Returns the stages in order, representing how far the app progressed.
func extractProgressStages(stdout string) []string {
	if stdout == "" {
		return nil
	}
	matches := smokeTestStagePattern.FindAllStringSubmatch(stdout, -1)
	if len(matches) == 0 {
		return nil
	}

	var stages []string
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) >= 2 {
			stage := match[1]
			if !seen[stage] {
				seen[stage] = true
				stages = append(stages, stage)
			}
		}
	}
	return stages
}

// stageDisplayName returns a human-friendly name for a smoke test stage.
func stageDisplayName(stage string) string {
	names := map[string]string{
		"bundle_resolving":  "Bundle resolved",
		"runtime_starting":  "Runtime starting",
		"waiting_for_token": "Waiting for auth token",
		"runtime_healthz":   "Runtime health check",
		"runtime_readyz":    "Waiting for services ready",
		"runtime_ports":     "Getting port configuration",
		"ui_server_check":   "Verifying UI server",
		"ready":             "App ready",
		"result":            "Smoke test completed",
	}
	if name, ok := names[stage]; ok {
		return name
	}
	return stage
}

// extractSmokeTestErrorHint extracts the first config/validation error from smoke test output.
// Returns the error message if found, empty string otherwise.
func extractSmokeTestErrorHint(stdout, stderr string) string {
	// Check stdout first (where structured markers are written)
	matches := smokeTestErrorPattern.FindAllStringSubmatch(stdout, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			kind := match[1]
			msg := match[2]
			if kind == "config" || kind == "validation" {
				return msg
			}
		}
	}

	// Check stderr as fallback
	matches = smokeTestErrorPattern.FindAllStringSubmatch(stderr, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			return match[2]
		}
	}

	return ""
}

// =============================================================================
// printPipelineSuccess and helpers
// =============================================================================

func printPipelineSuccess(status *pipelineStatus, notice *versionUpdateNotice) {
	fmt.Printf("Pipeline completed: %s\n", status.PipelineID)
	fmt.Println()

	printSuccessHeader(status, notice)
	printSuccessDuration(status)
	printSuccessArtifacts(status)
	printSuccessDeployURLs(status)
	printSuccessScreenRecording(status)
	printSuccessNextSteps(status)
}

func printSuccessHeader(status *pipelineStatus, notice *versionUpdateNotice) {
	if status.ScenarioName != "" {
		fmt.Printf("Scenario: %s\n", status.ScenarioName)
	}
	if status.Config != nil && status.Config.Version != "" {
		fmt.Printf("Version: %s\n", status.Config.Version)
	}
	if notice != nil && notice.requested {
		printVersionUpdateWarning(status, notice)
	}
}

func printSuccessDuration(status *pipelineStatus) {
	if status.StartedAt <= 0 || status.CompletedAt <= 0 {
		return
	}
	durationSec := status.CompletedAt - status.StartedAt
	if durationSec >= 60 {
		fmt.Printf("Duration: %dm %ds\n", durationSec/60, durationSec%60)
	} else {
		fmt.Printf("Duration: %ds\n", durationSec)
	}
}

func printSuccessArtifacts(status *pipelineStatus) {
	if len(status.FinalArtifacts) == 0 {
		return
	}
	fmt.Println()
	fmt.Println("Artifacts:")
	for platform, path := range status.FinalArtifacts {
		fmt.Printf("  %s: %s\n", platform, path)
	}
}

func printSuccessDeployURLs(status *pipelineStatus) {
	deployStage, ok := status.Stages["deploy"]
	if !ok || deployStage == nil || len(deployStage.Details) == 0 {
		return
	}
	var details deployStageDetails
	if err := json.Unmarshal(deployStage.Details, &details); err != nil {
		return
	}
	updateURL := strings.TrimSpace(details.UpdateURL)
	if updateURL == "" {
		return
	}

	channel := "stable"
	if status.Config != nil && status.Config.Deploy != nil && strings.TrimSpace(status.Config.Deploy.Channel) != "" {
		channel = strings.TrimSpace(status.Config.Deploy.Channel)
	}

	fmt.Println()
	fmt.Println("Updates:")
	fmt.Printf("  Base:    %s\n", updateURL)
	fmt.Printf("  Channel: %s\n", channel)
	if len(status.FinalArtifacts) > 0 {
		fmt.Println("  Manifests:")
		for platform := range status.FinalArtifacts {
			manifest := pipelineManifestFilename(platform)
			if manifest == "" {
				continue
			}
			fmt.Printf("    %s: %s/%s/%s\n", platform, updateURL, channel, manifest)
		}
	}
}

func printSuccessScreenRecording(status *pipelineStatus) {
	smokeStage, ok := status.Stages["smoketest"]
	if !ok || smokeStage == nil || len(smokeStage.Details) == 0 {
		return
	}
	var details smokeTestDetails
	if err := json.Unmarshal(smokeStage.Details, &details); err != nil {
		return
	}
	if details.ScreenRecording == nil {
		return
	}
	if details.ScreenRecording.Recorded {
		fmt.Println()
		fmt.Println("Screen Recording:")
		fmt.Printf("  Video: %s\n", details.ScreenRecording.VideoPath)
		if details.ScreenRecording.DurationMs > 0 {
			fmt.Printf("  Duration: %.1fs\n", float64(details.ScreenRecording.DurationMs)/1000)
		}
		if details.ScreenRecording.FileSizeBytes > 0 {
			fmt.Printf("  Size: %.1f MB\n", float64(details.ScreenRecording.FileSizeBytes)/1024/1024)
		}
		if details.SmokeTestID != "" {
			fmt.Printf("  Watch:  %s pipeline-status %s --video\n", appName, status.PipelineID)
			fmt.Printf("  API:    GET /api/v1/smoketest/%s/video\n", details.SmokeTestID)
		}
	} else if details.ScreenRecording.Error != "" {
		fmt.Printf("\nScreen Recording: failed (%s)\n", details.ScreenRecording.Error)
	}
}

func printSuccessNextSteps(status *pipelineStatus) {
	fmt.Println()
	fmt.Println("Next steps:")
	if len(status.FinalArtifacts) == 1 {
		for _, path := range status.FinalArtifacts {
			fmt.Printf("  Run the app:     %s\n", path)
			break
		}
	} else if len(status.FinalArtifacts) > 1 {
		fmt.Printf("  Run an artifact from the paths listed above\n")
	}
	fmt.Printf("  View full logs:  %s pipeline-status %s --verbose\n", appName, status.PipelineID)
	fmt.Printf("  View as JSON:    %s pipeline-status %s --json\n", appName, status.PipelineID)
}

func printVersionUpdateWarning(status *pipelineStatus, notice *versionUpdateNotice) {
	reported := ""
	if status.Config != nil {
		reported = status.Config.Version
	}

	if notice.expectedVersion != "" {
		if reported == "" {
			fmt.Fprintf(os.Stderr, "Warning: version update requested (%s), but pipeline reported no version. Check pipeline logs or update the scenario-to-desktop API.\n", notice.expectedVersion)
			return
		}
		if reported != notice.expectedVersion {
			fmt.Fprintf(os.Stderr, "Warning: version update requested (%s), but pipeline reported version %s. Check pipeline logs or update the scenario-to-desktop API.\n", notice.expectedVersion, reported)
		}
		return
	}

	if notice.bumpRequested && reported == "" {
		bump := notice.bumpValue
		if bump == "" {
			bump = "patch"
		}
		fmt.Fprintf(os.Stderr, "Warning: version bump (%s) requested, but pipeline reported no version. Check pipeline logs or update the scenario-to-desktop API.\n", bump)
	}
}

// =============================================================================
// printPipelineError and helpers
// =============================================================================

// printPipelineError prints detailed error information from a failed pipeline.
// When showOutput is true, it displays the full stdout/stderr from the smoke test.
func printPipelineError(status *pipelineStatus, showOutput bool) {
	if status.Error != "" {
		fmt.Printf("Error: %s\n", status.Error)
	}

	// Find and print the first failed stage.
	for stageName, stage := range status.Stages {
		if stage.Status != "failed" {
			continue
		}

		if stage.Error != "" && stage.Error != status.Error {
			fmt.Printf("Stage '%s' failed: %s\n", stageName, stage.Error)
		}

		var smokeDetails *smokeTestDetails
		if stageName == "smoketest" {
			smokeDetails = stage.getSmokeTestDetails()
		}

		printSmokeTestDiagnostics(smokeDetails)
		printStageErrorInfo(stage.ErrorInfo, smokeDetails)
		printBuildFailureDetails(stageName, stage, showOutput)
		printSmokeTestOutput(showOutput, smokeDetails)

		// Only show first failed stage.
		break
	}
}

// printSmokeTestDiagnostics prints lifecycle state, app errors, progress, and prereq warnings.
func printSmokeTestDiagnostics(details *smokeTestDetails) {
	if details == nil {
		return
	}

	appError := details.getAppReportedError()
	lifecycleState := details.getLifecycleState()
	if lifecycleState == "" {
		lifecycleState = details.CurrentState
	}
	appErrorIsStale := details.ErrorSessionMismatch || details.AppReportedErrorStale

	if appErrorIsStale {
		printStaleAppError(lifecycleState, appError, details)
	} else {
		printFreshAppError(lifecycleState, appError, details)
	}

	printProgressStages(details)
	printPrereqWarnings(details)
}

func printStaleAppError(lifecycleState, appError string, details *smokeTestDetails) {
	if lifecycleState != "" {
		fmt.Printf("\nLifecycle state: %s\n", lifecycleState)
		fmt.Printf("  %s\n", lifecycleStateDescription(lifecycleState))
	}
	if appError == "" {
		return
	}
	fmt.Printf("\nHistorical context (from previous session):\n")
	fmt.Printf("  Previous error: %s\n", appError)
	if ctx := details.getAppReportedErrorContext(); ctx != "" {
		fmt.Printf("  (%s)\n", ctx)
	}
	fmt.Printf("  ⚠️  Note: This error is from a different session and may not reflect the current issue.\n")
	fmt.Printf("  The current run reached '%s' state before timing out.\n", lifecycleState)
}

func printFreshAppError(lifecycleState, appError string, details *smokeTestDetails) {
	if appError != "" {
		fmt.Printf("\nApp reported error: %s\n", appError)
		if ctx := details.getAppReportedErrorContext(); ctx != "" {
			fmt.Printf("  (%s)\n", ctx)
		}
	}
	if lifecycleState != "" {
		fmt.Printf("\nLifecycle state: %s\n", lifecycleState)
		fmt.Printf("  %s\n", lifecycleStateDescription(lifecycleState))
	}
}

func printProgressStages(details *smokeTestDetails) {
	stages := details.getProgressStages()
	if len(stages) == 0 {
		return
	}
	fmt.Printf("\nApp progress (from stdout markers):\n")
	for i, stage := range stages {
		if i == len(stages)-1 {
			fmt.Printf("  ⏳ %s (timed out here)\n", stageDisplayName(stage))
		} else {
			fmt.Printf("  ✓ %s\n", stageDisplayName(stage))
		}
	}
}

func printPrereqWarnings(details *smokeTestDetails) {
	prereqWarnings := details.getPrereqWarnings()
	if len(prereqWarnings) == 0 {
		return
	}
	fmt.Printf("\nPotential issues detected during prerequisites:\n")
	for _, warning := range prereqWarnings {
		fmt.Printf("  ⚠️  %s\n", warning)
	}
}

// printStageErrorInfo prints structured error info including recovery hints.
func printStageErrorInfo(info *stageErrorInfo, smokeDetails *smokeTestDetails) {
	if info == nil {
		return
	}

	if info.Code != "" {
		fmt.Printf("Error code: %s\n", info.Code)
	}

	recoveryHint := buildRecoveryHint(info, smokeDetails)
	printStderrExcerpt(smokeDetails)
	printRecoveryGuidance(recoveryHint, info)
	printAutoFix(info.AutoFix)
	printManualSteps(info.ManualSteps)
	printDiagnosticOutput(info.Diagnostic, smokeDetails)
}

func buildRecoveryHint(info *stageErrorInfo, smokeDetails *smokeTestDetails) string {
	if smokeDetails == nil {
		return ""
	}
	stderr := smokeDetails.getStderr()
	if stderr != "" {
		if hint := analyzeStderr(stderr); hint != "" {
			return hint
		}
	}
	return extractSmokeTestErrorHint(smokeDetails.LastStdout, smokeDetails.LastStderr)
}

func printStderrExcerpt(smokeDetails *smokeTestDetails) {
	if smokeDetails == nil {
		return
	}
	stderr := smokeDetails.getStderr()
	if stderr == "" || strings.Contains(stderr, "ExperimentalWarning") {
		return
	}
	stderrDisplay := strings.TrimSpace(stderr)
	if len(stderrDisplay) > 500 {
		stderrDisplay = stderrDisplay[:500] + "...(truncated)"
	}
	if stderrDisplay != "" {
		fmt.Printf("\nRoot cause (stderr):\n  %s\n", strings.ReplaceAll(stderrDisplay, "\n", "\n  "))
	}
}

func printRecoveryGuidance(recoveryHint string, info *stageErrorInfo) {
	switch {
	case recoveryHint != "":
		fmt.Printf("\nRecovery: %s\n", recoveryHint)
	case info.RecoveryHint != "":
		fmt.Printf("\nRecovery: %s\n", info.RecoveryHint)
	case info.Recovery != "":
		fmt.Printf("\nRecovery action: %s\n", info.Recovery)
	}
}

func printAutoFix(fix *autoFix) {
	if fix == nil || fix.Command == "" {
		return
	}
	safeLabel := ""
	if fix.Safe {
		safeLabel = " (safe to run)"
	}
	fmt.Printf("\nAuto-fix%s:\n  %s\n", safeLabel, fix.Command)
	if fix.Description != "" {
		fmt.Printf("  → %s\n", fix.Description)
	}
}

func printManualSteps(steps []string) {
	if len(steps) == 0 {
		return
	}
	fmt.Printf("\nManual steps:\n")
	for i, step := range steps {
		fmt.Printf("  %d. %s\n", i+1, step)
	}
}

func printDiagnosticOutput(diag *diagnostic, smokeDetails *smokeTestDetails) {
	if diag == nil || diag.Process == nil {
		return
	}
	if diag.Process.LastOutput == "" || smokeDetails != nil {
		return
	}
	output := diag.Process.LastOutput
	if len(output) > 500 {
		output = output[:500] + "...(truncated)"
	}
	fmt.Printf("\nLast output:\n%s\n", output)
}

type buildPlatformResult struct {
	Status   string   `json:"status,omitempty"`
	ErrorLog []string `json:"error_log,omitempty"`
}

type buildDetails struct {
	PlatformResults map[string]buildPlatformResult `json:"platform_results,omitempty"`
	BuildLog        []string                       `json:"build_log,omitempty"`
	ErrorLog        []string                       `json:"error_log,omitempty"`
}

// printBuildFailureDetails surfaces build log excerpts for build stage failures.
func printBuildFailureDetails(stageName string, stage *stageResult, showOutput bool) {
	if stageName != "build" || len(stage.Details) == 0 {
		return
	}

	var details buildDetails
	if err := json.Unmarshal(stage.Details, &details); err != nil {
		return
	}

	printBuildExcerpt(&details)
	if showOutput {
		printBuildLogTail(details.BuildLog)
	}
}

func printBuildExcerpt(details *buildDetails) {
	var excerpt string
	for _, pr := range details.PlatformResults {
		if pr.Status != "failed" || len(pr.ErrorLog) == 0 {
			continue
		}
		excerpt = pr.ErrorLog[len(pr.ErrorLog)-1]
		break
	}
	if excerpt == "" && len(details.ErrorLog) > 0 {
		excerpt = details.ErrorLog[len(details.ErrorLog)-1]
	}
	if excerpt == "" {
		return
	}
	excerpt = strings.TrimSpace(excerpt)
	if len(excerpt) > 1200 {
		excerpt = excerpt[:1200] + "...(truncated)"
	}
	fmt.Printf("\nRoot cause (build output):\n  %s\n", strings.ReplaceAll(excerpt, "\n", "\n  "))
}

func printBuildLogTail(buildLog []string) {
	if len(buildLog) == 0 {
		return
	}
	fmt.Printf("\n--- Build log (tail) ---\n")
	start := len(buildLog) - 3
	if start < 0 {
		start = 0
	}
	for i := start; i < len(buildLog); i++ {
		entry := strings.TrimSpace(buildLog[i])
		if len(entry) > 800 {
			entry = entry[:800] + "...(truncated)"
		}
		fmt.Printf("%s\n\n", entry)
	}
}

// printSmokeTestOutput shows full stdout/stderr when --show-output is enabled.
func printSmokeTestOutput(showOutput bool, details *smokeTestDetails) {
	if !showOutput || details == nil {
		return
	}
	if details.LastStdout != "" {
		fmt.Printf("\n--- App stdout ---\n%s\n", details.LastStdout)
	}
	if details.LastStderr != "" {
		fmt.Printf("\n--- App stderr ---\n%s\n", details.LastStderr)
	}
	if details.LastStdout == "" && details.LastStderr == "" {
		fmt.Printf("\n--- No app output captured ---\n")
		fmt.Printf("Tip: App may have crashed before producing output. Check system logs.\n")
	}
}

// printAPIError displays a structured API error with recovery information.
func printAPIError(err error, debug bool) {
	var apiErr *cliutil.APIError
	if errors.As(err, &apiErr) && apiErr.IsStructured() {
		fmt.Print(apiErr.FormatConcise())
		if debug && len(apiErr.RawResponse) > 0 {
			fmt.Println("\n--- Debug: Raw Response ---")
			cliutil.PrintJSON(apiErr.RawResponse)
		}
	} else {
		fmt.Printf("Error: %s\n", err)
	}
}
