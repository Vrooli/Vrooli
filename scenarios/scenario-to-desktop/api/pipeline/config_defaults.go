// Package pipeline provides default configuration constants.
//
// This file centralizes timeout values, polling intervals, and other
// tunable parameters that were previously hard-coded in stage implementations.
// Extracting these as named constants makes them:
//   - Discoverable: developers can find all timeouts in one place
//   - Tunable: values can be adjusted without hunting through code
//   - Testable: tests can verify behavior at boundary conditions
package pipeline

import "time"

// Stage timeout defaults.
// These define how long each stage will wait before timing out.
//
// TUNING GUIDE: Each timeout has trade-offs between allowing slow operations
// to complete vs. failing fast on stuck builds. The values below work well
// for typical scenarios but may need adjustment for specific environments.
const (
	// DefaultBuildTimeout is the maximum time to wait for a build to complete.
	// Builds can take significant time for large Electron apps with multiple platforms.
	//
	// Lever impacts:
	//   Increase: Allow larger apps with multiple platforms to complete; prevents
	//             false failures on slow machines or when building all platforms.
	//   Decrease: Fail faster on stuck builds; reduces resource waste but may
	//             cause false failures on complex builds.
	//   Range: 10m (single platform, fast machine) to 60m (all platforms, slow machine)
	//   Default rationale: 30m allows for typical multi-platform builds with some margin.
	DefaultBuildTimeout = 30 * time.Minute

	// DefaultGenerationTimeout is the maximum time to wait for desktop wrapper generation.
	//
	// Lever impacts:
	//   Increase: Allow slow template engines or network-dependent template fetching.
	//   Decrease: Fail faster when generation hangs (usually indicates a bug).
	//   Range: 1m (simple templates) to 10m (complex templates with heavy customization)
	//   Default rationale: 5m covers most generation scenarios with margin for slowness.
	DefaultGenerationTimeout = 5 * time.Minute

	// DefaultSmokeTestTimeout is the maximum time to wait for smoke test completion.
	//
	// Lever impacts:
	//   Increase: Allow apps with slow startup (heavy initialization, large bundles).
	//   Decrease: Fail faster on apps that hang during startup.
	//   Range: 30s (simple apps) to 5m (apps with heavy initialization)
	//   Default rationale: 2m allows for typical Electron app startup with assertions.
	DefaultSmokeTestTimeout = 2 * time.Minute

	// DefaultDistributionTimeout is the maximum time to wait for artifact uploads.
	// Large artifacts to slow endpoints may need this full duration.
	//
	// Lever impacts:
	//   Increase: Allow large artifacts (500MB+) to upload on slow connections.
	//   Decrease: Fail faster on network issues; may cause unnecessary failures.
	//   Range: 5m (small artifacts, fast connection) to 60m (large artifacts, slow upload)
	//   Default rationale: 30m accounts for typical 100-200MB artifacts on moderate connections.
	DefaultDistributionTimeout = 30 * time.Minute

	// DefaultPreflightTimeout is the default timeout for preflight validation.
	// Can be overridden via Config.PreflightTimeoutSeconds.
	//
	// Lever impacts:
	//   Increase: Allow slow dependency installations or network checks.
	//   Decrease: Fail faster when preflight hangs (indicates system issue).
	//   Range: 30s (quick checks only) to 5m (with npm install or system updates)
	//   Default rationale: 60s covers typical system prerequisite checks.
	DefaultPreflightTimeout = 60 * time.Second
)

// Polling interval defaults.
// These define how frequently stages poll for completion.
//
// TUNING GUIDE: Polling intervals balance responsiveness against CPU/network load.
// Shorter intervals detect completion faster but increase resource usage.
const (
	// DefaultBuildPollInterval is how often to check build status.
	//
	// Lever impacts:
	//   Increase: Reduce CPU load during builds; completion detection slightly delayed.
	//   Decrease: Detect build completion faster; increases CPU polling overhead.
	//   Range: 1s (responsive) to 10s (resource-constrained environments)
	DefaultBuildPollInterval = 2 * time.Second

	// DefaultGenerationPollInterval is how often to check generation status.
	//
	// Lever impacts:
	//   Increase: Reduce polling overhead; generation is typically fast anyway.
	//   Decrease: Detect generation completion faster for latency-sensitive pipelines.
	//   Range: 100ms (very responsive) to 2s (batch processing)
	DefaultGenerationPollInterval = 500 * time.Millisecond

	// DefaultSmokePollInterval is how often to check smoke test status.
	//
	// Lever impacts:
	//   Increase: Reduce polling overhead during smoke tests.
	//   Decrease: Detect test completion faster; useful for CI/CD pipelines.
	//   Range: 100ms (very responsive) to 2s (batch processing)
	DefaultSmokePollInterval = 500 * time.Millisecond

	// DefaultDistributionPollInterval is how often to check distribution status.
	//
	// Lever impacts:
	//   Increase: Reduce API calls to distribution targets; be kind to S3/R2.
	//   Decrease: Detect upload completion faster; may hit rate limits.
	//   Range: 1s (local testing) to 10s (production with rate limits)
	DefaultDistributionPollInterval = 2 * time.Second

	// DefaultPipelinePollInterval is how often to poll pipeline completion in blocking mode.
	//
	// Lever impacts:
	//   Increase: Reduce polling overhead for long-running pipelines.
	//   Decrease: Return pipeline results faster; more responsive to cancellation.
	//   Range: 500ms (interactive CLI) to 10s (batch jobs)
	DefaultPipelinePollInterval = 2 * time.Second
)

// Preflight janitor configuration.
// The janitor periodically cleans up completed/expired preflight jobs to prevent memory leaks.
const (
	// DefaultPreflightCleanupInterval is how often the preflight janitor runs.
	//
	// Lever impacts:
	//   Increase: Reduce CPU overhead from cleanup; memory usage may grow between cleanups.
	//   Decrease: Keep memory usage lower; more frequent cleanup operations.
	//   Range: 30s (high-throughput systems) to 5m (low-traffic systems)
	DefaultPreflightCleanupInterval = 1 * time.Minute

	// DefaultPreflightJobExpiration is how long preflight jobs are kept before cleanup.
	//
	// Lever impacts:
	//   Increase: Allow longer delays between preflight and pipeline start; uses more memory.
	//   Decrease: Free memory faster; may cause "job not found" errors if pipelines start slowly.
	//   Range: 5m (fast pipelines) to 1h (pipelines with long delays between stages)
	DefaultPreflightJobExpiration = 15 * time.Minute
)

// Deployment mode constants.
// These replace magic strings scattered throughout the codebase.
const (
	// DeploymentModeProxy creates a desktop app that proxies to a running web server.
	// No bundling is required; the app loads from a URL.
	// Note: "proxy" is a legacy name; prefer "external-server" for new code.
	DeploymentModeProxy = "proxy"

	// DeploymentModeExternalServer creates a thin-client desktop app that connects
	// to a running Vrooli server. No bundling is required.
	// This is the recommended mode for scenarios with external dependencies (e.g., PostgreSQL).
	DeploymentModeExternalServer = "external-server"

	// DeploymentModeCloudAPI creates a desktop app that connects to a cloud API.
	// Similar to external-server but for cloud deployments.
	DeploymentModeCloudAPI = "cloud-api"

	// DeploymentModeBundled creates a desktop app with the UI bundled inside.
	// Requires the bundle stage to package UI assets.
	DeploymentModeBundled = "bundled"
)

// IsThinClientMode returns true if the deployment mode is a thin-client mode
// (external-server, cloud-api, or proxy) that doesn't require bundling.
func IsThinClientMode(mode string) bool {
	switch mode {
	case DeploymentModeExternalServer, DeploymentModeCloudAPI, DeploymentModeProxy:
		return true
	default:
		return false
	}
}

// Build status constants.
// These replace magic strings in build result handling.
const (
	// BuildStatusBuilding indicates a build is in progress.
	BuildStatusBuilding = "building"

	// BuildStatusReady indicates a build completed successfully.
	BuildStatusReady = "ready"

	// BuildStatusPartial indicates some platforms built successfully, others failed.
	BuildStatusPartial = "partial"

	// BuildStatusFailed indicates the build failed completely.
	BuildStatusFailed = "failed"

	// BuildStatusSkipped indicates a platform was skipped (e.g., Wine not installed).
	BuildStatusSkipped = "skipped"
)

// Smoke test status constants.
const (
	// SmokeTestStatusPassed indicates the smoke test passed.
	SmokeTestStatusPassed = "passed"

	// SmokeTestStatusFailed indicates the smoke test failed.
	SmokeTestStatusFailed = "failed"

	// SmokeTestStatusRunning indicates the smoke test is in progress.
	SmokeTestStatusRunning = "running"
)

// Framework constants.
const (
	// FrameworkElectron is the Electron desktop framework.
	FrameworkElectron = "electron"

	// FrameworkTauri is the Tauri desktop framework (future support).
	FrameworkTauri = "tauri"
)

// Template type constants.
const (
	// TemplateTypeBasic is the basic Electron template.
	TemplateTypeBasic = "basic"

	// TemplateTypeAdvanced is the advanced Electron template with more features.
	TemplateTypeAdvanced = "advanced"
)
