package build

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	signing "scenario-to-desktop-api/signing"
	signingtypes "scenario-to-desktop-api/signing/types"
	"scenario-to-desktop-api/smoketest"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// DefaultService is the default implementation of Service.
type DefaultService struct {
	store         Store
	wineChecker   WineChecker
	packageFinder PackageFinder
	runner        commandRunner
	logger        Logger
	pathLocksMu   sync.Mutex
	pathLocks     map[string]chan struct{}
}

// DefaultBuildTimeout bounds owner-launched standalone and pipeline builds.
const DefaultBuildTimeout = 30 * time.Minute

type commandRunner interface {
	Run(ctx context.Context, dir, command string) ([]byte, error)
	RunWithEnv(ctx context.Context, dir, command string, env map[string]string) ([]byte, error)
}

type shellCommandRunner struct{}

func (r *shellCommandRunner) Run(ctx context.Context, dir, command string) ([]byte, error) {
	return r.RunWithEnv(ctx, dir, command, nil)
}

func (r *shellCommandRunner) RunWithEnv(ctx context.Context, dir, command string, env map[string]string) ([]byte, error) {
	result, err := smoketest.NewProcessExecutor(&noopLogger{}).ExecuteWithResult(ctx, dir, "bash", []string{"-c", command}, environPairs(env), DefaultBuildTimeout)
	output := result.Combined
	if result.Truncated {
		output += fmt.Sprintf("\n[build output truncated: %d bytes omitted]\n", result.TruncatedBytes)
	}
	return []byte(output), errors.Join(ctx.Err(), err)
}

// environPairs renders env overrides as KEY=VALUE entries. The process
// environment is retained; managed signing values are only added for the
// duration of the signing command.
func environPairs(env map[string]string) []string {
	pairs := make([]string, 0, len(env))
	for key, value := range env {
		pairs = append(pairs, key+"="+value)
	}
	return pairs
}

// ServiceOption configures a DefaultService.
type ServiceOption func(*DefaultService)

// WithStore sets the build store.
func WithStore(store Store) ServiceOption {
	return func(s *DefaultService) {
		s.store = store
	}
}

// WithWineChecker sets the wine checker.
func WithWineChecker(checker WineChecker) ServiceOption {
	return func(s *DefaultService) {
		s.wineChecker = checker
	}
}

// WithPackageFinder sets the package finder.
func WithPackageFinder(finder PackageFinder) ServiceOption {
	return func(s *DefaultService) {
		s.packageFinder = finder
	}
}

// WithCommandRunner sets the command runner.
func WithCommandRunner(runner commandRunner) ServiceOption {
	return func(s *DefaultService) {
		s.runner = runner
	}
}

// WithLogger sets the logger.
func WithLogger(logger Logger) ServiceOption {
	return func(s *DefaultService) {
		s.logger = logger
	}
}

// NewService creates a new build service with the given options.
func NewService(opts ...ServiceOption) *DefaultService {
	s := &DefaultService{
		store:         NewStore(),
		wineChecker:   &defaultWineChecker{},
		packageFinder: &defaultPackageFinder{},
		runner:        &shellCommandRunner{},
		logger:        &noopLogger{},
		pathLocks:     make(map[string]chan struct{}),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// PerformDesktopBuild completes a desktop build within its owner context.
func (s *DefaultService) PerformDesktopBuild(ctx context.Context, buildID string, request *BuildRequest) {
	if _, ok := s.store.Get(buildID); !ok {
		return
	}

	// Build steps: install dependencies, build, package
	steps := []string{"npm install", "npm run build", "npm run dist"}

	for _, step := range steps {
		output, err := s.runner.Run(ctx, request.DesktopPath, step)
		outputStr := string(output)

		s.store.Update(buildID, func(status *Status) {
			status.BuildLog = append(status.BuildLog, fmt.Sprintf("%s: %s", step, outputStr))

			if err != nil {
				status.Status = "failed"
				status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("%s failed: %v", step, err))
				now := time.Now()
				status.CompletedAt = &now
			}
		})

		if err != nil {
			return
		}
	}

	s.store.Update(buildID, func(status *Status) {
		status.Status = "ready"
		now := time.Now()
		status.CompletedAt = &now
	})
}

// PerformScenarioDesktopBuild completes a scenario build within its owner context.
func (s *DefaultService) PerformScenarioDesktopBuild(ctx context.Context, buildID, scenarioName, desktopPath string, platforms []string, clean bool) {
	defer func() {
		if ctx.Err() != nil {
			s.failBuild(buildID, ctx.Err().Error())
		}
	}()
	if ctx.Err() != nil {
		return
	}
	if _, ok := s.store.Get(buildID); !ok {
		return
	}

	// Serialize builds per desktop path to avoid concurrent npm/node_modules mutation races.
	unlock, err := s.lockDesktopPath(ctx, desktopPath)
	if err != nil {
		return
	}
	defer unlock()

	defer func() {
		if r := recover(); r != nil {
			s.failBuild(buildID, fmt.Sprintf("Panic during build: %v", r))
		}
	}()

	s.logger.Info("build started", "scenario", scenarioName, "build_id", buildID)

	// If requested, remove common build output directories deterministically.
	if clean {
		if failed := s.cleanBuildOutputs(buildID, desktopPath, platforms); failed {
			return
		}
	}
	if _, err := os.Stat(filepath.Join(desktopPath, "wails-strategy.json")); err == nil {
		s.buildWailsProject(ctx, buildID, scenarioName, desktopPath, platforms)
		return
	}

	// Phase 1: Common build steps (install, compile)
	if failed := s.executeCommonBuildSteps(ctx, buildID, scenarioName, desktopPath, platforms); failed {
		return
	}

	// Phase 2: Build each platform independently
	s.buildAllPlatforms(ctx, buildID, scenarioName, desktopPath, platforms)
}

func (s *DefaultService) buildWailsProject(ctx context.Context, buildID, scenarioName, desktopPath string, platforms []string) {
	if len(platforms) == 0 {
		platforms = []string{"linux"}
	}
	hasLinuxTarget := false
	for _, platform := range platforms {
		if platform == "linux" || strings.HasPrefix(platform, "linux-") {
			hasLinuxTarget = true
			continue
		}
		s.store.UpdatePlatform(buildID, platform, func(_ *Status, result *PlatformResult) {
			result.Status = "failed"
			result.ErrorLog = append(result.ErrorLog, "Wails in-process currently supports Linux packaging only")
			completed := time.Now()
			result.CompletedAt = &completed
		})
	}
	if !hasLinuxTarget {
		s.failBuild(buildID)
		return
	}
	if err := os.RemoveAll(filepath.Join(desktopPath, "bin")); err != nil {
		s.failBuild(buildID, fmt.Sprintf("clear previous Wails artifacts: %v", err))
		return
	}

	dependencyCommand := "scenario-dependency-analyzer deps install go/github.com/wailsapp/wails/v3@v3.0.0-beta.27 --scenario " + shellQuote(scenarioName) + " --surface platforms/wails-inprocess --apply"
	dependencyOutput, dependencyErr := s.runner.Run(ctx, filepath.Dir(filepath.Dir(desktopPath)), dependencyCommand)
	if dependencyErr != nil {
		s.failBuild(buildID, "Scenario Dependency Analyzer could not install the pinned Wails module", string(dependencyOutput))
		return
	}

	for _, platform := range platforms {
		started := time.Now()
		if platform != "linux" && !strings.HasPrefix(platform, "linux-") {
			continue
		}
		s.store.UpdatePlatform(buildID, platform, func(_ *Status, result *PlatformResult) {
			result.Status = "building"
			result.StartedAt = &started
		})
		output, err := s.runner.RunWithEnv(ctx, desktopPath, "go run github.com/wailsapp/wails/v3/cmd/wails3 package GOOS=linux", map[string]string{"EXTRA_TAGS": "gtk3"})
		outputStr := string(output)
		artifactPath := filepath.Join(desktopPath, "bin", scenarioName)
		info, statErr := os.Stat(artifactPath)
		if err == nil && statErr != nil {
			err = fmt.Errorf("Wails package command succeeded but expected Linux application binary %s: %w", artifactPath, statErr)
		}
		s.store.Update(buildID, func(status *Status) {
			result, ok := status.PlatformResults[platform]
			if !ok {
				result = &PlatformResult{Platform: platform}
				status.PlatformResults[platform] = result
			}
			completed := time.Now()
			result.CompletedAt = &completed
			status.BuildLog = append(status.BuildLog, formatStepLog("wails-package", "go run github.com/wailsapp/wails/v3/cmd/wails3 package GOOS=linux (EXTRA_TAGS=gtk3)", outputStr, err))
			if err != nil {
				result.Status = "failed"
				result.ErrorLog = append(result.ErrorLog, err.Error(), outputStr)
				status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Wails package failed for %s", platform))
				return
			}
			result.Status = "ready"
			result.Artifact = artifactPath
			result.FileSize = info.Size()
			status.Artifacts[platform] = artifactPath
		})
	}
	s.store.Update(buildID, func(status *Status) {
		ready, failed := 0, 0
		for _, result := range status.PlatformResults {
			switch result.Status {
			case "ready":
				ready++
			case "failed":
				failed++
			}
		}
		switch {
		case ready > 0 && failed == 0:
			status.Status = "ready"
		case ready > 0:
			status.Status = "partial"
		default:
			status.Status = "failed"
		}
		completed := time.Now()
		status.CompletedAt = &completed
	})
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

// cleanBuildOutputs removes common build output directories. Returns true if
// the build should be aborted due to clean errors.
func (s *DefaultService) cleanBuildOutputs(buildID, desktopPath string, platforms []string) bool {
	// Relying on "npm run clean" is not sufficient (it often only removes dist/ and can leave dist-electron/).
	dirs := []string{
		filepath.Join(desktopPath, "dist-electron"),
		filepath.Join(desktopPath, "dist"),
		filepath.Join(desktopPath, "dist-dev"),
		filepath.Join(desktopPath, "dist-dev-electron"),
	}

	removed := 0
	var cleanErrs []string
	for _, d := range dirs {
		// Best-effort: only attempt if it exists.
		if _, err := os.Stat(d); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			cleanErrs = append(cleanErrs, fmt.Sprintf("stat %s: %v", d, err))
			continue
		}
		if err := os.RemoveAll(d); err != nil {
			cleanErrs = append(cleanErrs, fmt.Sprintf("remove %s: %v", d, err))
			continue
		}
		removed++
	}

	s.store.Update(buildID, func(status *Status) {
		entry := fmt.Sprintf("[clean] removed %d dirs under %s", removed, desktopPath)
		if len(cleanErrs) > 0 {
			entry += "\nWARN: some paths could not be cleaned:\n  - " + strings.Join(cleanErrs, "\n  - ")
		}
		status.BuildLog = append(status.BuildLog, entry)
	})

	// If cleanup had hard errors, fail early. This prevents confusing downstream ENOENT/rename issues.
	if len(cleanErrs) > 0 {
		s.store.Update(buildID, func(status *Status) {
			for _, platform := range platforms {
				if result, ok := status.PlatformResults[platform]; ok {
					result.Status = "failed"
					result.ErrorLog = append(result.ErrorLog, "Clean step failed (could not remove existing build outputs)")
					result.ErrorLog = append(result.ErrorLog, strings.Join(cleanErrs, "\n"))
					now := time.Now()
					result.CompletedAt = &now
				}
			}
			status.Status = "failed"
			status.ErrorLog = append(status.ErrorLog, "clean failed")
			status.ErrorLog = append(status.ErrorLog, strings.Join(cleanErrs, "\n"))
			now := time.Now()
			status.CompletedAt = &now
		})
		return true
	}
	return false
}

// executeCommonBuildSteps runs the install and compile steps shared by all
// platforms. Returns true if the build should be aborted due to a step failure.
func (s *DefaultService) executeCommonBuildSteps(ctx context.Context, buildID, scenarioName, desktopPath string, platforms []string) bool {
	type buildStep struct {
		name    string
		command string
	}

	commonSteps := []buildStep{
		{"install", installCommandForDesktop(desktopPath)},
		{"compile", "npm run build"},
	}

	for i, step := range commonSteps {
		s.logger.Info("executing build step",
			"scenario", scenarioName,
			"step", step.name,
			"progress", fmt.Sprintf("%d/%d", i+1, len(commonSteps)))

		outputStr, err := s.runBuildStep(ctx, buildID, scenarioName, desktopPath, step.name, step.command)

		logEntry := formatStepLog(step.name, step.command, outputStr, err)

		s.store.Update(buildID, func(status *Status) {
			status.BuildLog = append(status.BuildLog, logEntry)

			if err != nil {
				for _, platform := range platforms {
					if result, ok := status.PlatformResults[platform]; ok {
						result.Status = "failed"
						result.ErrorLog = append(result.ErrorLog, fmt.Sprintf("Common build step '%s' failed", step.name))
						result.ErrorLog = append(result.ErrorLog, outputStr)
						now := time.Now()
						result.CompletedAt = &now
					}
				}
				status.Status = "failed"
				status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("%s failed: %v", step.name, err))
				status.ErrorLog = append(status.ErrorLog, outputStr)
				now := time.Now()
				status.CompletedAt = &now
			}
		})

		if err != nil {
			s.logger.Error("common build step failed",
				"scenario", scenarioName,
				"build_id", buildID,
				"step", step.name,
				"error", err)
			return true
		}
	}
	return false
}

// runBuildStep executes a single build step with automatic retry logic for
// known npm failure modes (lockfile drift, ENOTEMPTY). Returns the combined
// output string and any final error.
func (s *DefaultService) runBuildStep(ctx context.Context, buildID, scenarioName, desktopPath, stepName, command string) (string, error) {
	output, err := s.runner.Run(ctx, desktopPath, command)
	outputStr := string(output)
	retryCommand := command

	// Prefer deterministic installs via npm ci, but recover when lockfile is out of sync.
	if err != nil && ctx.Err() == nil && stepName == "install" && looksLikeNpmLockfileOutOfSync(outputStr) {
		s.logger.Warn("npm ci failed due to lockfile drift; falling back to npm install once", "scenario", scenarioName, "build_id", buildID)
		output2, err2 := s.runner.Run(ctx, desktopPath, "npm install --no-audit --no-fund")
		outputStr = outputStr + "\n\n--- lockfile fallback (npm install) ---\n\n" + string(output2)
		err = err2
		// If the repair itself encounters ENOTEMPTY, the later retry must use
		// npm install as well. Retrying npm ci would discard the repair path and
		// fail again on the same lockfile drift.
		retryCommand = "npm install --no-audit --no-fund"
	}

	// npm can fail with ENOTEMPTY during concurrent FS activity or after interrupted installs.
	// Recover by removing node_modules and retrying once.
	if err != nil && ctx.Err() == nil && stepName == "install" && looksLikeNpmENOTEMPTYRename(outputStr) {
		s.logger.Warn("install failed with ENOTEMPTY; removing node_modules and retrying once", "scenario", scenarioName, "build_id", buildID)
		select {
		case <-ctx.Done():
			return outputStr, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
		nodeModulesPath := filepath.Join(desktopPath, "node_modules")
		_ = os.RemoveAll(nodeModulesPath)

		output2, err2 := s.runner.Run(ctx, desktopPath, retryCommand)
		outputStr = outputStr + "\n\n--- retry ---\n\n" + string(output2)
		err = err2
	}

	return outputStr, err
}

// buildAllPlatforms launches parallel platform builds, waits for completion,
// and determines the overall build status.
func (s *DefaultService) buildAllPlatforms(ctx context.Context, buildID, scenarioName, desktopPath string, platforms []string) {
	distPath := filepath.Join(desktopPath, "dist-electron")
	var wg sync.WaitGroup

	for _, platform := range platforms {
		wg.Add(1)
		go func(plt string) {
			defer wg.Done()
			s.BuildPlatform(ctx, buildID, scenarioName, desktopPath, distPath, plt)
		}(platform)
	}

	// Wait for all platform builds to complete
	wg.Wait()

	// Determine overall build status
	successCount := 0
	failedCount := 0
	skippedCount := 0
	finalStatus := ""

	s.store.Update(buildID, func(status *Status) {
		for _, result := range status.PlatformResults {
			switch result.Status {
			case "ready":
				successCount++
			case "failed":
				failedCount++
			case "skipped":
				skippedCount++
			}
		}

		switch {
		case successCount > 0 && failedCount == 0 && skippedCount == 0:
			status.Status = "ready"
		case successCount > 0:
			status.Status = "partial"
		default:
			status.Status = "failed"
		}

		now := time.Now()
		status.CompletedAt = &now
		finalStatus = status.Status
	})

	s.logger.Info("build completed",
		"scenario", scenarioName,
		"build_id", buildID,
		"status", finalStatus,
		"success", successCount,
		"failed", failedCount,
		"skipped", skippedCount)
}

// formatStepLog builds a log entry string for a build step.
func formatStepLog(stepName, command, outputStr string, err error) string {
	logEntry := fmt.Sprintf("[%s] %s", stepName, command)
	if err != nil {
		logEntry += fmt.Sprintf("\nFAILED: %v", err)
	} else {
		logEntry += "\nSUCCESS"
	}
	if err != nil || len(outputStr) < 500 {
		logEntry += fmt.Sprintf("\nOutput: %s", outputStr)
	} else {
		logEntry += fmt.Sprintf("\nOutput: %s... (%d bytes)", outputStr[:500], len(outputStr))
	}
	return logEntry
}

// BuildPlatform builds for a specific platform with dependency checking.
func (s *DefaultService) BuildPlatform(ctx context.Context, buildID, scenarioName, desktopPath, distPath, platform string) {
	if ctx.Err() != nil {
		return
	}
	if _, ok := s.store.Get(buildID); !ok {
		return
	}

	// Pipeline resource planning uses a concrete OS/architecture target (for
	// example linux-amd64). Electron-builder commands are OS-only. Preserve the
	// concrete target as the status key, but map it exactly once at this boundary.
	builderPlatform, ok := electronBuilderPlatform(platform)
	if !ok {
		s.store.UpdatePlatform(buildID, platform, func(_ *Status, result *PlatformResult) {
			result.Status = "failed"
			result.ErrorLog = append(result.ErrorLog, fmt.Sprintf("Unknown platform: %s", platform))
			now := time.Now()
			result.CompletedAt = &now
		})
		return
	}

	// Check platform dependencies
	if builderPlatform == "win" && !s.wineChecker.IsWineInstalled() {
		s.store.UpdatePlatform(buildID, platform, func(_ *Status, result *PlatformResult) {
			result.Status = "skipped"
			result.SkipReason = "Wine not installed (required for Windows builds on Linux). Install with: sudo apt install wine"
			now := time.Now()
			result.CompletedAt = &now
		})
		s.logger.Warn("skipping Windows build - Wine not installed", "scenario", scenarioName)
		return
	}

	distCommand, ok := platformDistCommand(builderPlatform)
	if !ok {
		s.store.UpdatePlatform(buildID, platform, func(_ *Status, result *PlatformResult) {
			result.Status = "failed"
			result.ErrorLog = append(result.ErrorLog, fmt.Sprintf("Unknown platform: %s", platform))
			now := time.Now()
			result.CompletedAt = &now
		})
		return
	}

	// Mark platform build as started
	s.store.UpdatePlatform(buildID, platform, func(_ *Status, result *PlatformResult) {
		result.Status = "building"
		now := time.Now()
		result.StartedAt = &now
	})

	s.logger.Info("building platform",
		"scenario", scenarioName,
		"build_id", buildID,
		"platform", platform)

	signingEnv, cleanupSigning, signingErr := resolveManagedSigningEnvForPlatform(ctx, scenarioName, builderPlatform)
	if signingErr != nil {
		s.store.UpdatePlatform(buildID, platform, func(_ *Status, result *PlatformResult) {
			result.Status = "failed"
			result.ErrorLog = append(result.ErrorLog, fmt.Sprintf("managed signing key unavailable: %v", signingErr))
			now := time.Now()
			result.CompletedAt = &now
		})
		s.logger.Error("managed signing key unavailable",
			"scenario", scenarioName, "build_id", buildID, "error", signingErr)
		return
	}
	defer cleanupSigning()

	output, err := s.runner.RunWithEnv(ctx, desktopPath, distCommand, signingEnv)
	outputStr := string(output)

	s.store.Update(buildID, func(status *Status) {
		result, ok := status.PlatformResults[platform]
		if !ok {
			result = &PlatformResult{Status: "failed"}
			status.PlatformResults[platform] = result
		}
		logEntry, resultStatus, skipReason := classifyPlatformResult(platform, distCommand, outputStr, err)
		result.Status = resultStatus
		if skipReason != "" {
			result.SkipReason = skipReason
		}
		if resultStatus != "ready" {
			result.ErrorLog = append(result.ErrorLog, outputStr)
		}
		status.BuildLog = append(status.BuildLog, logEntry)
		now := time.Now()
		result.CompletedAt = &now
	})

	if err != nil {
		s.logger.Error("platform build failed",
			"scenario", scenarioName,
			"build_id", buildID,
			"platform", platform,
			"error", err)
		return
	}

	s.recordBuiltPackage(buildID, scenarioName, distPath, platform)
}

func electronBuilderPlatform(platform string) (string, bool) {
	platform = strings.ToLower(strings.TrimSpace(platform))
	if before, _, found := strings.Cut(platform, "-"); found {
		platform = before
	}
	switch platform {
	case "linux":
		return "linux", true
	case "win", "windows":
		return "win", true
	case "mac", "macos", "darwin":
		return "mac", true
	default:
		return "", false
	}
}

// platformDistCommand returns the npm dist command for a platform and whether
// the platform is known.
func platformDistCommand(platform string) (string, bool) {
	switch platform {
	case "win":
		return "npm run dist:win", true
	case "mac":
		return "npm run dist:mac", true
	case "linux":
		return "npm run dist:linux", true
	default:
		return "", false
	}
}

// classifyPlatformResult inspects the build output and error to determine the
// log entry, result status, and optional skip reason.
func classifyPlatformResult(platform, distCommand, outputStr string, err error) (logEntry, status, skipReason string) {
	logEntry = fmt.Sprintf("[package-%s] %s", platform, distCommand)

	// Check for Wine/rcedit incompatibility (known limitation)
	isRceditIncompatibility := strings.Contains(outputStr, "Unrecognized argument") && strings.Contains(outputStr, "CompanyName")

	// Check for errors in output even if exit code is 0
	// electron-builder sometimes returns 0 even on partial failures
	// Look for electron-builder specific error markers, not npm warnings
	hasElectronBuilderError := strings.Contains(outputStr, "⨯ ") &&
		!strings.Contains(outputStr, "npm warn")

	switch {
	case isRceditIncompatibility:
		logEntry += "\nSKIPPED: Wine/rcedit incompatibility"
		status = "skipped"
		skipReason = "Wine's rcedit doesn't support CompanyName metadata. Workaround: Use actual Windows machine or CI/CD with Windows runners (GitHub Actions, AppVeyor). See: https://github.com/electron-userland/electron-builder/issues/6888"
	case err != nil || hasElectronBuilderError:
		logEntry += fmt.Sprintf("\nFAILED: %v", err)
		status = "failed"
	default:
		logEntry += "\nSUCCESS"
		status = "ready"
	}

	if len(outputStr) < 500 {
		logEntry += fmt.Sprintf("\nOutput: %s", outputStr)
	} else {
		logEntry += fmt.Sprintf("\nOutput: %s... (%d bytes)", outputStr[:500], len(outputStr))
	}
	return logEntry, status, skipReason
}

// recordBuiltPackage locates the built package artifact and records it in the
// build status.
func (s *DefaultService) recordBuiltPackage(buildID, scenarioName, distPath, platform string) {
	packagePlatform, ok := electronBuilderPlatform(platform)
	if !ok {
		s.store.UpdatePlatform(buildID, platform, func(_ *Status, result *PlatformResult) {
			result.Status = "failed"
			result.ErrorLog = append(result.ErrorLog, fmt.Sprintf("Built package not found: unknown platform: %s", platform))
		})
		return
	}
	packageFile, err := s.packageFinder.FindBuiltPackage(distPath, packagePlatform)
	if err != nil {
		s.store.UpdatePlatform(buildID, platform, func(_ *Status, result *PlatformResult) {
			result.Status = "failed"
			result.ErrorLog = append(result.ErrorLog, fmt.Sprintf("Built package not found: %v", err))
		})
		s.logger.Warn("platform package not found",
			"scenario", scenarioName,
			"platform", platform,
			"error", err)
		return
	}

	fileInfo, _ := os.Stat(packageFile)
	s.store.Update(buildID, func(status *Status) {
		result, ok := status.PlatformResults[platform]
		if !ok {
			result = &PlatformResult{}
			status.PlatformResults[platform] = result
		}
		result.Artifact = packageFile
		if fileInfo != nil {
			result.FileSize = fileInfo.Size()
		}
		status.Artifacts[platform] = packageFile
	})

	s.logger.Info("platform build succeeded",
		"scenario", scenarioName,
		"platform", platform,
		"artifact", packageFile)
}

// defaultWineChecker is the default Wine checker implementation.
type defaultWineChecker struct{}

// IsWineInstalled checks if Wine is installed on the system.
func (c *defaultWineChecker) IsWineInstalled() bool {
	cmd := exec.Command("which", "wine")
	err := cmd.Run()
	return err == nil
}

// defaultPackageFinder is the default package finder implementation.
type defaultPackageFinder struct{}

// FindBuiltPackage finds the built package file for a specific platform.
func (f *defaultPackageFinder) FindBuiltPackage(distPath, platform string) (string, error) {
	// Check if dist-electron directory exists
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		return "", fmt.Errorf("dist-electron directory not found at %s", distPath)
	}

	patterns, ok := platformGlobPatterns(platform)
	if !ok {
		return "", fmt.Errorf("unknown platform: %s", platform)
	}

	var best *packageCandidate

	// Search for matching files and pick the best by preference then mtime.
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(distPath, pattern))
		if err != nil {
			continue
		}
		for _, match := range matches {
			lower := strings.ToLower(match)
			if platform == "mac" && strings.Contains(lower, "blockmap") {
				continue
			}

			pref := platformFilePreference(platform, lower)
			best = considerCandidate(best, match, pref)
		}
	}

	if best != nil {
		return best.path, nil
	}

	return "", fmt.Errorf("no built package found for platform %s in %s", platform, distPath)
}

// packageCandidate tracks a potential build artifact with its metadata.
type packageCandidate struct {
	path    string
	modTime time.Time
	pref    int
}

// considerCandidate compares a new file against the current best candidate.
// Returns the better of the two (or the new one if best is nil).
func considerCandidate(best *packageCandidate, path string, pref int) *packageCandidate {
	stat, err := os.Stat(path)
	if err != nil {
		return best
	}
	c := &packageCandidate{
		path:    path,
		modTime: stat.ModTime(),
		pref:    pref,
	}
	if best == nil {
		return c
	}
	// Prefer higher preference (file type) first, then newer modification time as tiebreaker.
	// This ensures AppImage is preferred over .deb on Linux since AppImage is directly runnable.
	if c.pref > best.pref || (c.pref == best.pref && c.modTime.After(best.modTime)) {
		return c
	}
	return best
}

// platformGlobPatterns returns the file glob patterns for a given platform.
func platformGlobPatterns(platform string) ([]string, bool) {
	switch platform {
	case "win":
		// Prefer MSI installers; keep exe as a fallback for legacy builds.
		return []string{"*.msi", "*Setup.exe", "*.exe"}, true
	case "mac":
		// Prefer PKG installers; keep DMG/ZIP as a fallback when cross-compiling.
		return []string{"*.pkg", "*.dmg", "*.zip"}, true
	case "linux":
		return []string{"*.AppImage", "*.deb"}, true
	default:
		return nil, false
	}
}

// platformFilePreference returns a preference score for a file based on its
// platform. Higher scores are preferred.
func platformFilePreference(platform, lowerPath string) int {
	switch platform {
	case "win":
		switch {
		case strings.HasSuffix(lowerPath, ".msi"):
			return 3
		case strings.Contains(lowerPath, "setup"):
			return 2
		default:
			return 1
		}
	case "mac":
		switch {
		case strings.HasSuffix(lowerPath, ".pkg"):
			return 3
		case !strings.Contains(lowerPath, "arm64"):
			return 2
		default:
			return 1
		}
	case "linux":
		if strings.HasSuffix(lowerPath, ".appimage") {
			return 2
		}
		return 1
	default:
		return 0
	}
}

// noopLogger is a no-op logger for when logging is not needed.
type noopLogger struct{}

func (l *noopLogger) Info(msg string, args ...interface{})  {}
func (l *noopLogger) Warn(msg string, args ...interface{})  {}
func (l *noopLogger) Error(msg string, args ...interface{}) {}

func (s *DefaultService) lockDesktopPath(ctx context.Context, desktopPath string) (func(), error) {
	key := normalizeDesktopPath(desktopPath)
	s.pathLocksMu.Lock()
	lock := s.pathLocks[key]
	if lock == nil {
		lock = make(chan struct{}, 1)
		s.pathLocks[key] = lock
	}
	s.pathLocksMu.Unlock()
	select {
	case lock <- struct{}{}:
		return func() { <-lock }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *DefaultService) failBuild(buildID string, messages ...string) {
	s.store.Update(buildID, func(status *Status) {
		status.Status = "failed"
		status.ErrorLog = append(status.ErrorLog, messages...)
		now := time.Now()
		status.CompletedAt = &now
		for _, result := range status.PlatformResults {
			if result.Status == "building" || result.Status == "pending" {
				result.Status = "failed"
				result.CompletedAt = &now
			}
		}
	})
}

func normalizeDesktopPath(desktopPath string) string {
	p := strings.TrimSpace(desktopPath)
	if p == "" {
		return ""
	}
	if abs, err := filepath.Abs(p); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(p)
}

func installCommandForDesktop(desktopPath string) string {
	// Prefer npm ci for deterministic installs when a lockfile exists.
	lockfilePath := filepath.Join(desktopPath, "package-lock.json")
	if _, err := os.Stat(lockfilePath); err == nil {
		return "npm ci --no-audit --no-fund"
	}
	// Fallback for projects without a lockfile.
	return "npm install --no-audit --no-fund"
}

func looksLikeNpmENOTEMPTYRename(output string) bool {
	// Example:
	//   npm error code ENOTEMPTY
	//   npm error syscall rename
	//   npm error path .../node_modules/<pkg>
	//   npm error dest .../node_modules/.<pkg>-<suffix>
	s := strings.ToLower(output)
	return strings.Contains(s, "npm error code enotempty") &&
		strings.Contains(s, "npm error syscall rename") &&
		strings.Contains(s, "node_modules")
}

func looksLikeNpmLockfileOutOfSync(output string) bool {
	s := strings.ToLower(output)
	return strings.Contains(s, "npm error code eusage") &&
		strings.Contains(s, "npm ci") &&
		(strings.Contains(s, "package.json and package-lock.json") ||
			strings.Contains(s, "not in sync") ||
			strings.Contains(s, "missing:"))
}

// resolveManagedSigningEnvForPlatform resolves managed signing material only
// for the platform that consumes it.
func resolveManagedSigningEnvForPlatform(ctx context.Context, scenarioName, builderPlatform string) (map[string]string, func(), error) {
	if builderPlatform != "linux" {
		return nil, func() {}, nil
	}
	return resolveManagedSigningEnv(ctx, scenarioName)
}

// resolveManagedSigningEnv materializes a credential-authority-held signing key
// into an ephemeral GPG home and returns the process environment the signing
// command needs. The private key and passphrase never persist in the generated
// project; the caller must invoke the returned cleanup when the command ends.
func resolveManagedSigningEnv(ctx context.Context, scenarioName string) (map[string]string, func(), error) {
	noop := func() {}
	if strings.TrimSpace(scenarioName) == "" {
		return nil, noop, nil
	}
	config, err := signing.NewFileRepository().Get(ctx, scenarioName)
	if err != nil {
		return nil, noop, fmt.Errorf("load signing config: %w", err)
	}
	return managedSigningEnv(ctx, config)
}

// managedKeyAuthority is the read seam for managed signing material.
type managedKeyAuthority interface {
	Resolve(identity credentialauthority.Identity, field string) (string, error)
	Availability() error
}

// Seams allow the custody/materialization contract to be tested without a live
// credential store or a real gpg binary. Production never reassigns them.
var (
	openSigningAuthority    = func() (managedKeyAuthority, error) { return credentialauthority.Default() }
	importManagedSigningKey = importSecretKey
)

func managedSigningEnv(ctx context.Context, config *signingtypes.SigningConfig) (map[string]string, func(), error) {
	noop := func() {}
	if config == nil || !config.Enabled || config.Linux == nil || config.Linux.ManagedKey == nil {
		return nil, noop, nil
	}

	managed := config.Linux.ManagedKey
	identity, err := credentialauthority.ParseIdentity(managed.LogicalID)
	if err != nil {
		return nil, noop, fmt.Errorf("invalid managed signing identity %q: %w", managed.LogicalID, err)
	}
	authority, err := openSigningAuthority()
	if err != nil {
		return nil, noop, fmt.Errorf("open credential authority: %w", err)
	}
	if err := authority.Availability(); err != nil {
		return nil, noop, fmt.Errorf("credential authority unavailable: %w", err)
	}
	secret, err := authority.Resolve(identity, managed.ResolvedPrivateKeyField())
	if err != nil {
		return nil, noop, fmt.Errorf("resolve signing private key: %w", err)
	}
	passphrase, err := authority.Resolve(identity, managed.ResolvedPassphraseField())
	if err != nil {
		return nil, noop, fmt.Errorf("resolve signing passphrase: %w", err)
	}

	homedir, err := os.MkdirTemp("", "vrooli-signing-gnupg-")
	if err != nil {
		return nil, noop, fmt.Errorf("create signing homedir: %w", err)
	}
	_ = os.Chmod(homedir, 0o700)
	cleanup := func() { _ = os.RemoveAll(homedir) }

	if err := importManagedSigningKey(ctx, homedir, secret); err != nil {
		cleanup()
		return nil, noop, err
	}

	envName := strings.TrimSpace(config.Linux.GPGPassphraseEnv)
	if envName == "" {
		envName = signingtypes.DefaultPassphraseEnvVar
	}
	return map[string]string{
		signingtypes.DefaultManagedHomedirEnv: homedir,
		envName:                               passphrase,
	}, cleanup, nil
}

// importSecretKey imports an ASCII-armored private key into a GPG home.
func importSecretKey(ctx context.Context, homedir, armored string) error {
	cmd := exec.CommandContext(ctx, "gpg", "--batch", "--homedir", homedir, "--import")
	cmd.Stdin = strings.NewReader(armored)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("import signing key: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
