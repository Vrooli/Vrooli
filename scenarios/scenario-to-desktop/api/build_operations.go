package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// performDesktopBuild performs desktop build asynchronously
func (s *Server) performDesktopBuild(buildID string, request *struct {
	DesktopPath string   `json:"desktop_path"`
	Platforms   []string `json:"platforms"`
	Sign        bool     `json:"sign"`
	Publish     bool     `json:"publish"`
}) {
	s.buildMutex.RLock()
	status := s.buildStatuses[buildID]
	s.buildMutex.RUnlock()

	// Build steps: install dependencies, build, package
	steps := []string{"npm install", "npm run build", "npm run dist"}

	for _, step := range steps {
		cmd := exec.Command("bash", "-c", step)
		cmd.Dir = request.DesktopPath

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		s.buildMutex.Lock()
		status.BuildLog = append(status.BuildLog, fmt.Sprintf("%s: %s", step, outputStr))

		if err != nil {
			status.Status = "failed"
			status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("%s failed: %v", step, err))
			now := time.Now()
			status.CompletedAt = &now
			s.buildMutex.Unlock()
			return
		}
		s.buildMutex.Unlock()
	}

	s.buildMutex.Lock()
	status.Status = "ready"
	now := time.Now()
	status.CompletedAt = &now
	s.buildMutex.Unlock()
}

// performScenarioDesktopBuild performs scenario desktop build asynchronously
func (s *Server) performScenarioDesktopBuild(buildID, scenarioName, desktopPath string, platforms []string, clean bool) {
	s.buildMutex.RLock()
	status := s.buildStatuses[buildID]
	s.buildMutex.RUnlock()

	defer func() {
		if r := recover(); r != nil {
			s.buildMutex.Lock()
			status.Status = "failed"
			status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Panic during build: %v", r))
			now := time.Now()
			status.CompletedAt = &now
			s.buildMutex.Unlock()
		}
	}()

	s.logger.Info("build started", "scenario", scenarioName, "build_id", buildID)

	// Phase 1: Common build steps (clean, install, compile)
	var commonSteps []struct {
		name    string
		command string
	}

	if clean {
		commonSteps = append(commonSteps, struct {
			name    string
			command string
		}{"clean", "npm run clean"})
	}

	commonSteps = append(commonSteps, []struct {
		name    string
		command string
	}{
		{"install", "npm install"},
		{"compile", "npm run build"},
	}...)

	// Execute common steps
	for i, step := range commonSteps {
		s.logger.Info("executing build step",
			"scenario", scenarioName,
			"step", step.name,
			"progress", fmt.Sprintf("%d/%d", i+1, len(commonSteps)))

		cmd := exec.Command("bash", "-c", step.command)
		cmd.Dir = desktopPath

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		s.buildMutex.Lock()
		logEntry := fmt.Sprintf("[%s] %s", step.name, step.command)
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
		status.BuildLog = append(status.BuildLog, logEntry)

		if err != nil {
			// Common step failed - mark all platforms as failed
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
			s.buildMutex.Unlock()

			s.logger.Error("common build step failed",
				"scenario", scenarioName,
				"build_id", buildID,
				"step", step.name,
				"error", err)
			return
		}
		s.buildMutex.Unlock()
	}

	// Phase 2: Build each platform independently
	distPath := filepath.Join(desktopPath, "dist-electron")
	var wg sync.WaitGroup

	for _, platform := range platforms {
		wg.Add(1)
		go func(plt string) {
			defer wg.Done()
			s.buildPlatform(buildID, scenarioName, desktopPath, distPath, plt)
		}(platform)
	}

	// Wait for all platform builds to complete
	wg.Wait()

	// Determine overall build status
	s.buildMutex.Lock()
	successCount := 0
	failedCount := 0
	skippedCount := 0

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

	if successCount > 0 && failedCount == 0 && skippedCount == 0 {
		status.Status = "ready"
	} else if successCount > 0 {
		status.Status = "partial"
	} else {
		status.Status = "failed"
	}

	now := time.Now()
	status.CompletedAt = &now
	s.buildMutex.Unlock()

	s.logger.Info("build completed",
		"scenario", scenarioName,
		"build_id", buildID,
		"status", status.Status,
		"success", successCount,
		"failed", failedCount,
		"skipped", skippedCount)
}

// buildPlatform builds for a specific platform with dependency checking
func (s *Server) buildPlatform(buildID, scenarioName, desktopPath, distPath, platform string) {
	s.buildMutex.RLock()
	status := s.buildStatuses[buildID]
	result := status.PlatformResults[platform]
	s.buildMutex.RUnlock()

	// Check platform dependencies
	if platform == "win" {
		if !s.isWineInstalled() {
			s.buildMutex.Lock()
			result.Status = "skipped"
			result.SkipReason = "Wine not installed (required for Windows builds on Linux). Install with: sudo apt install wine"
			now := time.Now()
			result.CompletedAt = &now
			s.buildMutex.Unlock()
			s.logger.Warn("skipping Windows build - Wine not installed", "scenario", scenarioName)
			return
		}
	}

	// Mark platform build as started
	s.buildMutex.Lock()
	result.Status = "building"
	now := time.Now()
	result.StartedAt = &now
	s.buildMutex.Unlock()

	// Determine build command
	var distCommand string
	switch platform {
	case "win":
		distCommand = "npm run dist:win"
	case "mac":
		distCommand = "npm run dist:mac"
	case "linux":
		distCommand = "npm run dist:linux"
	default:
		s.buildMutex.Lock()
		result.Status = "failed"
		result.ErrorLog = append(result.ErrorLog, fmt.Sprintf("Unknown platform: %s", platform))
		now := time.Now()
		result.CompletedAt = &now
		s.buildMutex.Unlock()
		return
	}

	s.logger.Info("building platform",
		"scenario", scenarioName,
		"build_id", buildID,
		"platform", platform)

	cmd := exec.Command("bash", "-c", distCommand)
	cmd.Dir = desktopPath

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	s.buildMutex.Lock()
	logEntry := fmt.Sprintf("[package-%s] %s", platform, distCommand)

	// Check for Wine/rcedit incompatibility (known limitation)
	isRceditIncompatibility := strings.Contains(outputStr, "Unrecognized argument") && strings.Contains(outputStr, "CompanyName")

	// Check for errors in output even if exit code is 0
	// electron-builder sometimes returns 0 even on partial failures
	// Look for electron-builder specific error markers, not npm warnings
	hasElectronBuilderError := strings.Contains(outputStr, "⨯ ") &&
		!strings.Contains(outputStr, "npm warn")

	if isRceditIncompatibility {
		// This is a known Wine limitation, mark as skipped with helpful message
		logEntry += "\nSKIPPED: Wine/rcedit incompatibility"
		result.Status = "skipped"
		result.SkipReason = "Wine's rcedit doesn't support CompanyName metadata. Workaround: Use actual Windows machine or CI/CD with Windows runners (GitHub Actions, AppVeyor). See: https://github.com/electron-userland/electron-builder/issues/6888"
		result.ErrorLog = append(result.ErrorLog, outputStr)
	} else if err != nil || hasElectronBuilderError {
		logEntry += fmt.Sprintf("\nFAILED: %v", err)
		result.Status = "failed"
		result.ErrorLog = append(result.ErrorLog, outputStr)
	} else {
		logEntry += "\nSUCCESS"
		result.Status = "ready"
	}
	if len(outputStr) < 500 {
		logEntry += fmt.Sprintf("\nOutput: %s", outputStr)
	} else {
		logEntry += fmt.Sprintf("\nOutput: %s... (%d bytes)", outputStr[:500], len(outputStr))
	}
	status.BuildLog = append(status.BuildLog, logEntry)
	now = time.Now()
	result.CompletedAt = &now
	s.buildMutex.Unlock()

	if err != nil {
		s.logger.Error("platform build failed",
			"scenario", scenarioName,
			"build_id", buildID,
			"platform", platform,
			"error", err)
		return
	}

	// Find built package
	packageFile, err := s.findBuiltPackage(distPath, platform)
	if err == nil {
		fileInfo, _ := os.Stat(packageFile)
		s.buildMutex.Lock()
		result.Artifact = packageFile
		if fileInfo != nil {
			result.FileSize = fileInfo.Size()
		}
		status.Artifacts[platform] = packageFile
		s.buildMutex.Unlock()

		s.logger.Info("platform build succeeded",
			"scenario", scenarioName,
			"platform", platform,
			"artifact", packageFile)
	} else {
		s.buildMutex.Lock()
		result.Status = "failed"
		result.ErrorLog = append(result.ErrorLog, fmt.Sprintf("Built package not found: %v", err))
		s.buildMutex.Unlock()

		s.logger.Warn("platform package not found",
			"scenario", scenarioName,
			"platform", platform,
			"error", err)
	}
}
