package toolexecution

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"scenario-to-desktop-api/shared/args"
)

// DistributionExecutor handles distribution-related tool execution.
type DistributionExecutor struct {
	distributionService DistributionService
	distributionStore   DistributionStore
	logger              *slog.Logger
}

// NewDistributionExecutor creates a new DistributionExecutor.
func NewDistributionExecutor(service DistributionService, store DistributionStore, logger *slog.Logger) *DistributionExecutor {
	if logger == nil {
		logger = slog.Default()
	}
	return &DistributionExecutor{
		distributionService: service,
		distributionStore:   store,
		logger:              logger,
	}
}

// UploadArtifact uploads an artifact.
func (e *DistributionExecutor) UploadArtifact(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	scenarioName, err := args.RequireString(toolArgs, "scenario_name")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	artifactPath, err := args.RequireString(toolArgs, "artifact_path")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	targets := args.GetStringArray(toolArgs, "targets")
	version := args.GetString(toolArgs, "version", "latest")

	distributionID := fmt.Sprintf("dist-%s", uuid.New().String()[:8])

	if e.distributionStore != nil {
		e.distributionStore.Save(DistributionStatus{
			DistributionID: distributionID,
			ScenarioName:   scenarioName,
			Status:         "uploading",
			ArtifactPath:   artifactPath,
			Targets:        targets,
			CreatedAt:      time.Now(),
		})
	}

	return AsyncResult(map[string]interface{}{
		"distribution_id": distributionID,
		"scenario_name":   scenarioName,
		"artifact_path":   artifactPath,
		"targets":         targets,
		"version":         version,
		"status":          "uploading",
		"message":         "Upload started. Use check_distribution_status to monitor progress.",
	}, distributionID), nil
}

// PublishRelease publishes a release.
func (e *DistributionExecutor) PublishRelease(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	scenarioName, err := args.RequireString(toolArgs, "scenario_name")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	version, err := args.RequireString(toolArgs, "version")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	artifacts, err := args.RequireMap(toolArgs, "artifacts")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	releaseNotes := args.GetString(toolArgs, "release_notes", "")
	distributionID := fmt.Sprintf("release-%s", uuid.New().String()[:8])

	if e.distributionStore != nil {
		e.distributionStore.Save(DistributionStatus{
			DistributionID: distributionID,
			ScenarioName:   scenarioName,
			Status:         "publishing",
			CreatedAt:      time.Now(),
		})
	}

	return AsyncResult(map[string]interface{}{
		"distribution_id": distributionID,
		"scenario_name":   scenarioName,
		"version":         version,
		"release_notes":   releaseNotes,
		"artifacts":       artifacts,
		"status":          "publishing",
		"message":         "Release publishing started. Use check_distribution_status to monitor progress.",
	}, distributionID), nil
}

// ListArtifacts lists artifacts.
func (e *DistributionExecutor) ListArtifacts(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	scenarioName, err := args.RequireString(toolArgs, "scenario_name")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	target := args.GetString(toolArgs, "target", "")

	// List artifacts from distribution service
	return SuccessResult(map[string]interface{}{
		"scenario_name": scenarioName,
		"target":        target,
		"artifacts":     []interface{}{},
		"message":       "No artifacts found",
	}), nil
}

// ListDistributionTargets lists distribution targets.
func (e *DistributionExecutor) ListDistributionTargets(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	if e.distributionService != nil {
		targets, err := e.distributionService.ListTargets(ctx)
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to list targets: %v", err), CodeInternalError), nil
		}

		targetMaps := make([]map[string]interface{}, len(targets))
		for i, t := range targets {
			targetMaps[i] = map[string]interface{}{
				"name":    t.Name,
				"type":    t.Type,
				"enabled": t.Enabled,
			}
		}
		return SuccessResult(map[string]interface{}{
			"targets": targetMaps,
		}), nil
	}

	return SuccessResult(map[string]interface{}{
		"targets": []interface{}{},
		"message": "No distribution targets configured",
	}), nil
}

// ValidateDistributionTarget validates a distribution target.
func (e *DistributionExecutor) ValidateDistributionTarget(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	targetName, err := args.RequireString(toolArgs, "target_name")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	if e.distributionService != nil {
		err := e.distributionService.ValidateTarget(ctx, targetName)
		if err != nil {
			return SuccessResult(map[string]interface{}{
				"target_name": targetName,
				"valid":       false,
				"error":       err.Error(),
			}), nil
		}
		return SuccessResult(map[string]interface{}{
			"target_name": targetName,
			"valid":       true,
			"message":     "Target is accessible and properly configured",
		}), nil
	}

	return ErrorResult("distribution service not available - server may not be fully initialized; wait and retry or check server logs", CodeInternalError), nil
}

// CheckDistributionStatus checks distribution status.
func (e *DistributionExecutor) CheckDistributionStatus(ctx context.Context, toolArgs map[string]interface{}) (*ExecutionResult, error) {
	distributionID, err := args.RequireString(toolArgs, "distribution_id")
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	if e.distributionStore == nil {
		return ErrorResult("distribution store not available - server may not be fully initialized; wait and retry or check server logs", CodeInternalError), nil
	}

	status, ok := e.distributionStore.Get(distributionID)
	if !ok {
		return ErrorResult("distribution operation not found", CodeNotFound), nil
	}

	return SuccessResult(map[string]interface{}{
		"distribution_id": status.DistributionID,
		"scenario_name":   status.ScenarioName,
		"status":          status.Status,
		"artifact_path":   status.ArtifactPath,
		"targets":         status.Targets,
		"progress":        status.Progress,
		"error":           status.Error,
		"created_at":      status.CreatedAt,
		"completed_at":    status.CompletedAt,
	}), nil
}
