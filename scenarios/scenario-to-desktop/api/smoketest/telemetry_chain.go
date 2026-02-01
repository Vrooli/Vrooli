package smoketest

import "context"

// TelemetryChainParams contains parameters for telemetry chain execution.
type TelemetryChainParams struct {
	// SmokeTestID is the smoke test identifier.
	SmokeTestID string

	// ScenarioName is the scenario being tested.
	ScenarioName string

	// Platform is the target platform (linux, mac, win).
	Platform string

	// ArtifactPath is the path to the built artifact.
	ArtifactPath string

	// Output is the smoke test process output.
	Output string

	// DirectUploadSuccess indicates if the app reported successful telemetry upload.
	DirectUploadSuccess bool
}

// DefaultTelemetryChainExecutor is the default implementation of TelemetryChainExecutor.
type DefaultTelemetryChainExecutor struct {
	telemetryResolver TelemetryPathResolver
	telemetryIngestor TelemetryIngestor
	config            Config
	logger            Logger
}

// NewTelemetryChainExecutor creates a new telemetry chain executor.
func NewTelemetryChainExecutor(
	telemetryResolver TelemetryPathResolver,
	telemetryIngestor TelemetryIngestor,
	config Config,
	logger Logger,
) *DefaultTelemetryChainExecutor {
	return &DefaultTelemetryChainExecutor{
		telemetryResolver: telemetryResolver,
		telemetryIngestor: telemetryIngestor,
		config:            config,
		logger:            logger,
	}
}

// Execute runs the telemetry collection chain.
func (e *DefaultTelemetryChainExecutor) Execute(ctx context.Context, params TelemetryChainParams) TelemetryResult {
	result := TelemetryResult{
		Source:         TelemetrySourceNone,
		AttemptedPaths: []PathAttempt{},
	}

	// Step 1: Check if direct upload already succeeded
	if params.DirectUploadSuccess {
		result.Source = TelemetrySourceUpload
		result.AttemptedPaths = append(result.AttemptedPaths, PathAttempt{
			Source:  TelemetrySourceUpload,
			Success: true,
		})
		e.logger.Info("telemetry_chain_direct_upload",
			"smoke_test_id", params.SmokeTestID,
			"source", string(TelemetrySourceUpload),
		)
		return result
	}

	// Step 2: Try to extract path from output
	if e.telemetryIngestor == nil {
		result.Error = "no telemetry ingestor configured"
		return result
	}

	outputPath := e.telemetryResolver.ExtractFromOutput(params.Output)
	if outputPath != "" {
		attempt := e.attemptIngestion(ctx, params, outputPath, TelemetrySourceOutputExtraction)
		result.AttemptedPaths = append(result.AttemptedPaths, attempt.PathAttempt)
		if attempt.Success {
			result.Source = TelemetrySourceOutputExtraction
			result.Path = outputPath
			result.EventsFound = attempt.eventsFound
			result.EventsIngested = attempt.eventsIngested
			e.logger.Info("telemetry_chain_output_extraction_success",
				"smoke_test_id", params.SmokeTestID,
				"path", outputPath,
				"events_ingested", attempt.eventsIngested,
			)
			return result
		}
	} else {
		result.AttemptedPaths = append(result.AttemptedPaths, PathAttempt{
			Source:  TelemetrySourceOutputExtraction,
			Success: false,
			Error:   "no telemetry path found in output",
		})
	}

	// Step 3: Try artifact-based resolution
	artifactPath := e.telemetryResolver.ResolveFromArtifact(params.Platform, params.ArtifactPath, params.ScenarioName)
	if artifactPath != "" {
		attempt := e.attemptIngestion(ctx, params, artifactPath, TelemetrySourceArtifactResolution)
		result.AttemptedPaths = append(result.AttemptedPaths, attempt.PathAttempt)
		if attempt.Success {
			result.Source = TelemetrySourceArtifactResolution
			result.Path = artifactPath
			result.EventsFound = attempt.eventsFound
			result.EventsIngested = attempt.eventsIngested
			e.logger.Info("telemetry_chain_artifact_resolution_success",
				"smoke_test_id", params.SmokeTestID,
				"path", artifactPath,
				"events_ingested", attempt.eventsIngested,
			)
			return result
		}
	} else {
		result.AttemptedPaths = append(result.AttemptedPaths, PathAttempt{
			Source:  TelemetrySourceArtifactResolution,
			Success: false,
			Error:   "could not resolve telemetry path from artifact",
		})
	}

	// All attempts failed
	result.Source = TelemetrySourceNone
	result.Error = "all telemetry collection methods failed"
	e.logger.Warn("telemetry_chain_all_failed",
		"smoke_test_id", params.SmokeTestID,
		"attempts", len(result.AttemptedPaths),
	)
	return result
}

// extendedPathAttempt includes additional fields for internal use.
type extendedPathAttempt struct {
	PathAttempt
	eventsFound    int
	eventsIngested int
}

// attemptIngestion tries to read and ingest telemetry from the given path.
func (e *DefaultTelemetryChainExecutor) attemptIngestion(
	ctx context.Context,
	params TelemetryChainParams,
	path string,
	source TelemetrySource,
) extendedPathAttempt {
	attempt := extendedPathAttempt{
		PathAttempt: PathAttempt{
			Source: source,
			Path:   path,
		},
	}

	// Read events from file
	events, err := e.telemetryResolver.ReadTelemetryEvents(path, e.config.MaxTelemetryEvents)
	if err != nil {
		attempt.Error = err.Error()
		e.logger.Error("telemetry_chain_read_failed",
			"smoke_test_id", params.SmokeTestID,
			"source", string(source),
			"path", path,
			"error", err.Error(),
		)
		return attempt
	}

	if len(events) == 0 {
		attempt.Error = "no events found in telemetry file"
		e.logger.Info("telemetry_chain_no_events",
			"smoke_test_id", params.SmokeTestID,
			"source", string(source),
			"path", path,
		)
		return attempt
	}

	attempt.eventsFound = len(events)

	// Ingest events
	_, ingested, err := e.telemetryIngestor.IngestEvents(params.ScenarioName, "", "smoke-test-fallback", events)
	if err != nil {
		attempt.Error = err.Error()
		e.logger.Error("telemetry_chain_ingest_failed",
			"smoke_test_id", params.SmokeTestID,
			"source", string(source),
			"path", path,
			"events_found", len(events),
			"error", err.Error(),
		)
		return attempt
	}

	attempt.Success = true
	attempt.eventsIngested = ingested
	return attempt
}
