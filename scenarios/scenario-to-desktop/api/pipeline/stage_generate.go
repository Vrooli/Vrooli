package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/shared/errors"
)

// GenerateStage implements the desktop wrapper generation stage of the pipeline.
type GenerateStage struct {
	service      generation.Service
	analyzer     generation.ScenarioAnalyzer
	buildStore   generation.BuildStore // for polling build status
	timeProvider TimeProvider
	scenarioRoot string
}

// GenerateStageOption configures a GenerateStage.
type GenerateStageOption func(*GenerateStage)

// WithGenerateService sets the generation service.
func WithGenerateService(svc generation.Service) GenerateStageOption {
	return func(s *GenerateStage) {
		s.service = svc
	}
}

// WithScenarioAnalyzer sets the scenario analyzer.
func WithScenarioAnalyzer(a generation.ScenarioAnalyzer) GenerateStageOption {
	return func(s *GenerateStage) {
		s.analyzer = a
	}
}

// WithGenerateTimeProvider sets the time provider.
func WithGenerateTimeProvider(tp TimeProvider) GenerateStageOption {
	return func(s *GenerateStage) {
		s.timeProvider = tp
	}
}

// WithGenerateScenarioRoot sets the scenario root path.
func WithGenerateScenarioRoot(root string) GenerateStageOption {
	return func(s *GenerateStage) {
		s.scenarioRoot = root
	}
}

// WithGenerateBuildStore sets the build store for polling build status.
func WithGenerateBuildStore(store generation.BuildStore) GenerateStageOption {
	return func(s *GenerateStage) {
		s.buildStore = store
	}
}

// NewGenerateStage creates a new generate stage.
func NewGenerateStage(opts ...GenerateStageOption) *GenerateStage {
	s := &GenerateStage{
		timeProvider: NewRealTimeProvider(),
	}
	for _, opt := range opts {
		opt(s)
	}
	// Default scenario root
	if s.scenarioRoot == "" {
		home, _ := os.UserHomeDir()
		s.scenarioRoot = filepath.Join(home, "Vrooli", "scenarios")
	}
	return s
}

// Name returns the stage name.
func (s *GenerateStage) Name() string {
	return StageGenerate
}

// Dependencies returns stages that must complete before this one.
func (s *GenerateStage) Dependencies() []string {
	// Depends on preflight (which may have been skipped)
	return []string{StagePreflight}
}

// CanSkip returns whether this stage can be skipped.
// Generation is never skipped - it's always required.
func (s *GenerateStage) CanSkip(input *StageInput) bool {
	return false
}

// Execute runs the desktop generation stage.
func (s *GenerateStage) Execute(ctx context.Context, input *StageInput) *StageResult {
	result := newStageResult(s.Name(), s.timeProvider)

	if checkCancellation(ctx, result, s.timeProvider) {
		return result
	}

	if s.analyzer == nil {
		failStage(result, s.timeProvider, errors.ErrGenerateAnalyzerNotConfigured())
		return result
	}

	scenarioName := input.Config.ScenarioName
	result.Logs = append(result.Logs, fmt.Sprintf("Analyzing scenario: %s", scenarioName))

	// Analyze the scenario
	metadata, err := s.analyzer.AnalyzeScenario(scenarioName)
	if err != nil {
		failStage(result, s.timeProvider, errors.ErrScenarioAnalysisFailed(err, scenarioName))
		return result
	}

	input.ScenarioMetadata = metadata
	result.Logs = append(result.Logs, fmt.Sprintf("Detected: %s v%s", metadata.DisplayName, metadata.Version))
	if input.Config != nil && input.Config.Version == "" && metadata.Version != "" {
		input.Config.Version = metadata.Version
		result.Logs = append(result.Logs, fmt.Sprintf("Release version: %s", metadata.Version))
	}

	// Validate scenario is ready for desktop
	if err := s.analyzer.ValidateScenarioForDesktop(scenarioName); err != nil {
		failStage(result, s.timeProvider, errors.ErrScenarioValidationFailed(err, scenarioName))
		return result
	}

	// Create desktop config from metadata
	templateType := input.Config.GetTemplateType()
	desktopConfig, err := s.analyzer.CreateDesktopConfigFromMetadata(metadata, templateType)
	if err != nil {
		failStage(result, s.timeProvider, errors.ErrDesktopConfigFailed(err))
		return result
	}

	// Apply pipeline config overrides
	// Always use the pipeline's computed deployment mode (with default "bundled").
	// This ensures the generated code matches the pipeline's behavior.
	// GetDeploymentMode() returns "bundled" by default, which creates fully
	// self-contained desktop applications - the most common production use case.
	desktopConfig.DeploymentMode = input.Config.GetDeploymentMode()
	desktopConfig.Platforms = input.Config.Platforms
	if input.Config.LocationMode != "" {
		desktopConfig.LocationMode = input.Config.LocationMode
	}
	if input.Config.ProxyURL != "" {
		desktopConfig.ProxyURL = input.Config.ProxyURL
	}
	if input.BundleResult != nil {
		desktopConfig.BundleManifestPath = input.BundleResult.ManifestPath
		// Use relative path "bundle" instead of absolute source path.
		// electron-builder copies bundle/ to resources/bundle via extraResources,
		// so the packaged app must look for "bundle" relative to resources path.
		desktopConfig.BundleRuntimeRoot = "bundle"

		// Extract IPC configuration from manifest content.
		// This is critical for bundled mode: the token path in bundle.json must be
		// passed through to the template generator, otherwise the desktop app will
		// wait for a token file at the wrong path and timeout.
		if input.BundleResult.ManifestContent != nil {
			desktopConfig.BundleIPC = extractBundleIPCConfig(input.BundleResult.ManifestContent)
			if desktopConfig.BundleIPC != nil {
				result.Logs = append(result.Logs,
					fmt.Sprintf("Bundle IPC: host=%s port=%d token_path=%s",
						desktopConfig.BundleIPC.Host,
						desktopConfig.BundleIPC.Port,
						desktopConfig.BundleIPC.AuthTokenRel))
			}

			// Extract UI service configuration from manifest content.
			// This ensures the Electron app knows which service ID and port name
			// to use for resolving the UI port, instead of relying on defaults.
			uiSvcID, uiPortName := extractBundleUIServiceConfig(input.BundleResult.ManifestContent)
			if uiSvcID != "" {
				desktopConfig.BundleUISvcID = uiSvcID
				desktopConfig.BundleUIPortName = uiPortName
				result.Logs = append(result.Logs,
					fmt.Sprintf("Bundle UI service: id=%s port_name=%s", uiSvcID, uiPortName))
			}
		}
	}

	// Validate that BundleRuntimeRoot is never an absolute path.
	// Absolute paths would be embedded in the generated code and break after packaging
	// since the app runs from a different location than where it was built.
	if desktopConfig.BundleRuntimeRoot != "" && filepath.IsAbs(desktopConfig.BundleRuntimeRoot) {
		failStage(result, s.timeProvider, errors.New(errors.CodeConfigInvalid,
			fmt.Sprintf("BundleRuntimeRoot must be a relative path, got absolute path: %s", desktopConfig.BundleRuntimeRoot)).
			WithRecovery(errors.RecoveryFixInput,
				"BundleRuntimeRoot should be 'bundle' (relative) so the packaged app finds assets in resources/bundle").
			WithManualSteps([]string{
				"Check that BundleRuntimeRoot is set to 'bundle' (relative path)",
				"The electron-builder extraResources config copies bundle/ to resources/bundle",
				"Absolute paths break after packaging because the app runs from a different location",
			}))
		return result
	}

	// For bundled deployment mode, extraResources should point to the bundle directory
	// (relative to the electron output path) instead of the source ui/dist path.
	// The bundle stage has already packaged all assets (UI, binaries, manifest) into
	// platforms/electron/bundle/, so we reference that directory.
	if desktopConfig.DeploymentMode == "bundled" {
		desktopConfig.ScenarioPath = "bundle"
	}

	// Resolve staging output path when requested so bundle + build share a stable location.
	scenarioPath := input.ScenarioPath
	if scenarioPath == "" {
		scenarioPath = filepath.Join(s.scenarioRoot, scenarioName)
	}
	resolvedDesktopPath := ""
	if isStagingLocation(input.Config.LocationMode) {
		if input.BundleResult != nil && input.BundleResult.BundleDir != "" {
			resolvedDesktopPath = filepath.Dir(input.BundleResult.BundleDir)
		} else {
			_, resolvedDesktopPath = resolvePipelineOutputPaths(input.Config, scenarioPath, input.PipelineID, desktopConfig.Framework)
		}
		if resolvedDesktopPath != "" {
			desktopConfig.OutputPath = resolvedDesktopPath
			result.Logs = append(result.Logs, fmt.Sprintf("Staging output path: %s", resolvedDesktopPath))
		}
	} else {
		_, resolvedDesktopPath = resolvePipelineOutputPaths(input.Config, scenarioPath, input.PipelineID, desktopConfig.Framework)
	}

	// Clean thin-client outputs here (bundle stage is skipped).
	if input.Config.Clean && ShouldSkipBundle(input.Config) && resolvedDesktopPath != "" {
		result.Logs = append(result.Logs, fmt.Sprintf("Cleaning desktop output: %s", resolvedDesktopPath))
		if err := os.RemoveAll(resolvedDesktopPath); err != nil {
			failStage(result, s.timeProvider, errors.ErrGenerationFailed(err).WithDetail("output_path", resolvedDesktopPath))
			return result
		}
	}

	result.Logs = append(result.Logs,
		fmt.Sprintf("Deployment mode: %s", desktopConfig.DeploymentMode),
		fmt.Sprintf("Template type: %s", templateType),
	)

	// Check for update config warnings
	// The default provider is "generic" (self-hosted), which requires a URL.
	// Without a URL, auto-updates are effectively disabled.
	if input.Config.UpdateConfig != nil {
		provider := input.Config.UpdateConfig.Provider
		if provider == "" {
			provider = "generic" // Default provider
		}
		if provider == "generic" {
			if input.Config.UpdateConfig.Generic == nil || input.Config.UpdateConfig.Generic.URL == "" {
				result.Logs = append(result.Logs,
					"WARNING: Generic update provider configured without URL. "+
						"Auto-updates will not work until update_config.generic.url is set. "+
						"Set provider to 'none' to explicitly disable updates.")
			}
		}
	}

	if s.service == nil {
		failStage(result, s.timeProvider, errors.ErrGenerateServiceNotConfigured())
		return result
	}

	// Queue the generation
	buildStatus := s.service.QueueBuild(desktopConfig, metadata, true)
	buildID := buildStatus.BuildID

	// Wait for generation to complete (poll with cancellation support)
	desktopPath, err := s.waitForGeneration(ctx, buildID, buildStatus)
	if err != nil {
		failStage(result, s.timeProvider, errors.ErrGenerationFailed(err).WithDetail("build_id", buildID))
		return result
	}

	// Update input for next stage
	input.DesktopPath = desktopPath
	input.GenerationResult = &generation.GenerateResponse{
		BuildID:     buildID,
		Status:      "ready",
		DesktopPath: desktopPath,
	}

	completeStage(result, s.timeProvider, input.GenerationResult)
	result.Logs = append(result.Logs,
		fmt.Sprintf("Desktop wrapper generated: %s", desktopPath),
	)

	return result
}

// waitForGeneration polls for generation completion.
func (s *GenerateStage) waitForGeneration(ctx context.Context, buildID string, initialStatus *generation.BuildStatus) (string, error) {
	// Quick check for synchronous completion
	if initialStatus.Status == BuildStatusReady && initialStatus.OutputPath != "" {
		return initialStatus.OutputPath, nil
	}

	// If still building, wait with timeout
	timeout := time.After(DefaultGenerationTimeout)
	ticker := time.NewTicker(DefaultGenerationPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("generation cancelled")
		case <-timeout:
			return "", fmt.Errorf("generation timed out after %v", DefaultGenerationTimeout)
		case <-ticker.C:
			// Poll fresh status from store. QueueBuild spawns an async goroutine
			// that updates the store, so we query the store for latest status.
			if s.buildStore == nil {
				// Fallback to checking initialStatus if no store configured.
				// This won't see async updates, but covers simple/test cases.
				switch initialStatus.Status {
				case BuildStatusReady:
					return initialStatus.OutputPath, nil
				case BuildStatusFailed:
					if len(initialStatus.ErrorLog) > 0 {
						return "", fmt.Errorf("generation failed: %s", initialStatus.ErrorLog[len(initialStatus.ErrorLog)-1])
					}
					return "", fmt.Errorf("generation failed")
				}
				continue
			}

			currentStatus, ok := s.buildStore.Get(buildID)
			if !ok {
				return "", fmt.Errorf("build status not found in store: %s", buildID)
			}

			switch currentStatus.Status {
			case BuildStatusReady:
				return currentStatus.OutputPath, nil
			case BuildStatusFailed:
				if len(currentStatus.ErrorLog) > 0 {
					return "", fmt.Errorf("generation failed: %s", currentStatus.ErrorLog[len(currentStatus.ErrorLog)-1])
				}
				return "", fmt.Errorf("generation failed")
			}
		}
	}
}

// extractBundleIPCConfig extracts IPC configuration from manifest content.
// The manifest content is a map[string]interface{} parsed from bundle.json.
// Returns nil if IPC config is not present or cannot be extracted.
func extractBundleIPCConfig(content map[string]interface{}) *generation.BundleIPCConfig {
	ipcRaw, ok := content["ipc"]
	if !ok {
		return nil
	}

	ipc, ok := ipcRaw.(map[string]interface{})
	if !ok {
		return nil
	}

	config := &generation.BundleIPCConfig{}

	// Extract host
	if host, ok := ipc["host"].(string); ok {
		config.Host = host
	}

	// Extract port (JSON numbers come as float64)
	if port, ok := ipc["port"].(float64); ok {
		config.Port = int(port)
	}

	// Extract auth_token_path - this is critical!
	// The bundle.json uses "auth_token_path" as the key.
	if tokenPath, ok := ipc["auth_token_path"].(string); ok {
		config.AuthTokenRel = tokenPath
	}

	return config
}

// extractBundleUIServiceConfig extracts UI service configuration from manifest content.
// Returns the service ID and port name for the Electron app to resolve the UI port.
func extractBundleUIServiceConfig(content map[string]interface{}) (serviceID string, portName string) {
	servicesRaw, ok := content["services"]
	if !ok {
		return "", ""
	}

	services, ok := servicesRaw.([]interface{})
	if !ok {
		return "", ""
	}

	for _, svcRaw := range services {
		svc, ok := svcRaw.(map[string]interface{})
		if !ok {
			continue
		}

		svcType, _ := svc["type"].(string)
		if svcType != "ui-bundle" && svcType != "ui" && svcType != "frontend" {
			continue
		}

		serviceID, _ = svc["id"].(string)
		if serviceID == "" {
			continue
		}

		// Extract port name from ports.requested[0].name
		if portsRaw, ok := svc["ports"].(map[string]interface{}); ok {
			if reqRaw, ok := portsRaw["requested"].([]interface{}); ok && len(reqRaw) > 0 {
				if firstPort, ok := reqRaw[0].(map[string]interface{}); ok {
					if name, ok := firstPort["name"].(string); ok && name != "" {
						return serviceID, name
					}
				}
			}
		}

		// Default port name for UI services
		return serviceID, "ui"
	}

	return "", ""
}
