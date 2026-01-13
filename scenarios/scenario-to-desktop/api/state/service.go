package state

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"
)

// Service provides high-level operations for scenario state management.
type Service struct {
	store    *Store
	detector *StalenessDetector
	logger   *slog.Logger
}

// NewService creates a new state service.
func NewService(store *Store, logger *slog.Logger) *Service {
	return &Service{
		store:    store,
		detector: NewStalenessDetector(store),
		logger:   logger,
	}
}

// LoadState retrieves scenario state with optional validation.
func (s *Service) LoadState(ctx context.Context, scenarioName string, req LoadStateRequest) (*LoadStateResponse, error) {
	state, err := s.store.Get(ctx, scenarioName)
	if err != nil {
		return nil, err
	}

	resp := &LoadStateResponse{
		Found: state != nil,
		State: state,
	}

	// Skip logs if not requested
	if state != nil && !req.IncludeLogs {
		stateCopy := *state
		stateCopy.CompressedLogs = nil
		resp.State = &stateCopy
	}

	// Validate manifest hash if requested
	if req.ValidateManifest && state != nil {
		manifestPath := req.ManifestPath
		if manifestPath == "" {
			manifestPath = state.FormState.BundleManifestPath
		}

		if manifestPath != "" {
			currentHash, _, err := ComputeManifestHash(manifestPath)
			if err == nil {
				resp.CurrentHash = currentHash

				if bundleState, ok := state.Stages[StageBundle]; ok {
					resp.StoredHash = bundleState.InputFingerprint.ManifestHash
					resp.ManifestChanged = currentHash != resp.StoredHash
				}
			}
		}
	}

	return resp, nil
}

// SaveState stores or updates scenario state.
func (s *Service) SaveState(ctx context.Context, scenarioName string, req SaveStateRequest) (*SaveStateResponse, error) {
	// Get existing state for conflict detection
	existing, err := s.store.Get(ctx, scenarioName)
	if err != nil {
		return nil, err
	}

	// Check for conflicts if expected hash is provided
	if req.ExpectedHash != "" && existing != nil && existing.Hash != req.ExpectedHash {
		return &SaveStateResponse{
			Success:     false,
			Conflict:    true,
			ServerState: existing,
			Hash:        existing.Hash,
		}, nil
	}

	// Build state
	var state *ScenarioState
	if existing != nil {
		state = existing
	} else {
		state = &ScenarioState{
			ScenarioName:  scenarioName,
			SchemaVersion: SchemaVersion,
			CreatedAt:     time.Now(),
			Stages:        make(map[string]StageState),
		}
	}

	// Update form state
	state.FormState = req.FormState

	// Compute manifest hash if requested
	if req.ComputeHash && req.ManifestPath != "" {
		hash, mtime, err := ComputeManifestHash(req.ManifestPath)
		if err == nil {
			// Update bundle stage fingerprint
			if state.Stages == nil {
				state.Stages = make(map[string]StageState)
			}
			bundleState := state.Stages[StageBundle]
			bundleState.Stage = StageBundle
			bundleState.InputFingerprint.ManifestPath = req.ManifestPath
			bundleState.InputFingerprint.ManifestHash = hash
			bundleState.InputFingerprint.ManifestMtime = mtime
			state.Stages[StageBundle] = bundleState
		}
	}

	// Compress and store logs if provided
	if len(req.LogTails) > 0 {
		compressed, err := CompressLogs(req.LogTails)
		if err != nil {
			s.logger.Warn("failed to compress logs", "error", err)
		} else {
			state.CompressedLogs = MergeLogs(state.CompressedLogs, compressed)
		}
	}

	// Update build artifacts if provided
	if len(req.BuildArtifacts) > 0 {
		state.BuildArtifacts = mergeArtifacts(state.BuildArtifacts, req.BuildArtifacts)
	}

	// Update stage results if provided
	if req.StageResults != nil {
		for stage, result := range req.StageResults {
			stageState := state.Stages[stage]
			stageState.Stage = stage
			stageState.Result = result
			stageState.ValidatedAt = time.Now()
			stageState.Status = StatusValid
			state.Stages[stage] = stageState
		}
	}

	// Compute state hash
	state.Hash = ComputeStateHash(state)
	state.UpdatedAt = time.Now()

	// Save
	if err := s.store.Save(ctx, state); err != nil {
		return nil, err
	}

	return &SaveStateResponse{
		Success:   true,
		UpdatedAt: state.UpdatedAt,
		Hash:      state.Hash,
	}, nil
}

// ClearState deletes scenario state.
func (s *Service) ClearState(ctx context.Context, scenarioName string) (*ClearStateResponse, error) {
	err := s.store.Delete(ctx, scenarioName)
	if err != nil {
		return &ClearStateResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &ClearStateResponse{
		Success: true,
		Message: "State cleared",
	}, nil
}

// CheckStaleness compares current config against stored state.
func (s *Service) CheckStaleness(ctx context.Context, scenarioName string, req CheckStalenessRequest) (*CheckStalenessResponse, error) {
	stored, err := s.store.Get(ctx, scenarioName)
	if err != nil {
		return nil, err
	}

	// If manifest path is set, compute current hash
	if req.CurrentConfig.ManifestPath != "" && req.CurrentConfig.ManifestHash == "" {
		hash, mtime, err := ComputeManifestHash(req.CurrentConfig.ManifestPath)
		if err == nil {
			req.CurrentConfig.ManifestHash = hash
			req.CurrentConfig.ManifestMtime = mtime
		}
	}

	changes, err := s.detector.DetectChanges(ctx, scenarioName, &req.CurrentConfig)
	if err != nil {
		return nil, err
	}

	affectedStages := s.detector.ComputeAffectedStages(changes)
	status := s.detector.BuildValidationStatus(scenarioName, stored, changes)

	resp := &CheckStalenessResponse{
		Valid:          len(changes) == 0,
		Changed:        len(changes) > 0,
		PendingChanges: changes,
		AffectedStages: affectedStages,
		Status:         status,
	}

	if stored != nil {
		resp.StoredHash = stored.Hash
	}

	return resp, nil
}

// GetLogs retrieves and decompresses logs for a specific service.
func (s *Service) GetLogs(ctx context.Context, scenarioName, serviceID string) (*GetLogsResponse, error) {
	state, err := s.store.Get(ctx, scenarioName)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, nil
	}

	cl := FindCompressedLog(state.CompressedLogs, serviceID)
	if cl == nil {
		return nil, nil
	}

	content, err := DecompressLog(*cl)
	if err != nil {
		return nil, err
	}

	return &GetLogsResponse{
		ServiceID:  cl.ServiceID,
		Content:    content,
		Lines:      cl.OriginalLines,
		CapturedAt: cl.CapturedAt,
	}, nil
}

// InvalidateStagesFrom marks a stage and all downstream stages as stale.
func (s *Service) InvalidateStagesFrom(ctx context.Context, scenarioName, fromStage, reason string) error {
	return s.store.Update(ctx, scenarioName, func(state *ScenarioState) {
		foundStart := false

		for _, stage := range StageOrder {
			if stage == fromStage {
				foundStart = true
			}
			if foundStart {
				if stageState, ok := state.Stages[stage]; ok {
					stageState.Status = StatusStale
					stageState.StalenessReason = reason
					state.Stages[stage] = stageState
				}
			}
		}
	})
}

// MarkStageValid updates a stage as successfully validated with its fingerprint.
func (s *Service) MarkStageValid(
	ctx context.Context,
	scenarioName string,
	stage string,
	fingerprint InputFingerprint,
	result json.RawMessage,
) error {
	return s.store.Update(ctx, scenarioName, func(state *ScenarioState) {
		if state.Stages == nil {
			state.Stages = make(map[string]StageState)
		}
		state.Stages[stage] = StageState{
			Stage:            stage,
			Status:           StatusValid,
			InputFingerprint: fingerprint,
			ValidatedAt:      time.Now(),
			Result:           result,
			StalenessReason:  "",
		}
	})
}

// GetValidationStatus returns the current validation status for a scenario.
func (s *Service) GetValidationStatus(ctx context.Context, scenarioName string) (*ValidationStatus, error) {
	stored, err := s.store.Get(ctx, scenarioName)
	if err != nil {
		return nil, err
	}

	return s.detector.BuildValidationStatus(scenarioName, stored, nil), nil
}

// GetStore returns the underlying store.
func (s *Service) GetStore() *Store {
	return s.store
}

// --- Helper functions ---

func mergeArtifacts(existing, new []BuildArtifact) []BuildArtifact {
	if len(new) == 0 {
		return existing
	}

	// Map by platform for quick lookup
	artifactMap := make(map[string]BuildArtifact, len(existing)+len(new))
	for _, a := range existing {
		artifactMap[a.Platform] = a
	}
	for _, a := range new {
		artifactMap[a.Platform] = a // Overwrite existing
	}

	// Convert back to slice
	result := make([]BuildArtifact, 0, len(artifactMap))
	for _, a := range artifactMap {
		result = append(result, a)
	}

	return result
}
