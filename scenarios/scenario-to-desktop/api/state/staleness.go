package state

import (
	"context"
	"sort"
	"time"
)

// StalenessDetector checks for changes that invalidate cached state.
type StalenessDetector struct {
	store *Store
}

// NewStalenessDetector creates a new staleness detector.
func NewStalenessDetector(store *Store) *StalenessDetector {
	return &StalenessDetector{store: store}
}

// DetectChanges compares current config against stored state and returns detected changes.
func (d *StalenessDetector) DetectChanges(
	ctx context.Context,
	scenarioName string,
	current *InputFingerprint,
) ([]StateChange, error) {
	stored, err := d.store.Get(ctx, scenarioName)
	if err != nil {
		return nil, err
	}
	if stored == nil {
		return nil, nil // No stored state = everything needs to run
	}

	var changes []StateChange

	// Check bundle stage inputs
	if bundleState, ok := stored.Stages[StageBundle]; ok {
		fp := bundleState.InputFingerprint

		if fp.ManifestPath != current.ManifestPath {
			changes = append(changes, StateChange{
				ChangeType:    "manifest_path",
				AffectedStage: StageBundle,
				Reason:        "Manifest path changed",
				OldValue:      fp.ManifestPath,
				NewValue:      current.ManifestPath,
			})
		} else if current.ManifestPath != "" {
			// Same path - check content hash
			if current.ManifestHash != "" && fp.ManifestHash != "" {
				if current.ManifestHash != fp.ManifestHash {
					changes = append(changes, StateChange{
						ChangeType:    "manifest_content",
						AffectedStage: StagePreflight, // Content change only affects preflight+
						Reason:        "Manifest file content changed",
						OldValue:      truncateHash(fp.ManifestHash),
						NewValue:      truncateHash(current.ManifestHash),
					})
				} else if current.ManifestMtime != fp.ManifestMtime && fp.ManifestMtime != 0 {
					// Content same but mtime different - file was touched
					changes = append(changes, StateChange{
						ChangeType:    "manifest_touched",
						AffectedStage: StagePreflight,
						Reason:        "Manifest file was touched (content unchanged)",
					})
				}
			}
		}
	}

	// Check preflight inputs
	if preflightState, ok := stored.Stages[StagePreflight]; ok {
		fp := preflightState.InputFingerprint

		if !stringSlicesEqual(fp.PreflightSecretKeys, current.PreflightSecretKeys) {
			changes = append(changes, StateChange{
				ChangeType:    "preflight_secrets",
				AffectedStage: StagePreflight,
				Reason:        "Preflight secrets configuration changed",
			})
		}

		if fp.PreflightTimeout != current.PreflightTimeout && current.PreflightTimeout > 0 {
			changes = append(changes, StateChange{
				ChangeType:    "preflight_timeout",
				AffectedStage: StagePreflight,
				Reason:        "Preflight timeout changed",
			})
		}
	}

	// Check generate inputs
	if genState, ok := stored.Stages[StageGenerate]; ok {
		fp := genState.InputFingerprint

		if fp.TemplateType != current.TemplateType && current.TemplateType != "" {
			changes = append(changes, StateChange{
				ChangeType:    "template_type",
				AffectedStage: StageGenerate,
				Reason:        "Template type changed",
				OldValue:      fp.TemplateType,
				NewValue:      current.TemplateType,
			})
		}

		if fp.Framework != current.Framework && current.Framework != "" {
			changes = append(changes, StateChange{
				ChangeType:    "framework",
				AffectedStage: StageGenerate,
				Reason:        "Framework changed",
				OldValue:      fp.Framework,
				NewValue:      current.Framework,
			})
		}

		if fp.DeploymentMode != current.DeploymentMode && current.DeploymentMode != "" {
			changes = append(changes, StateChange{
				ChangeType:    "deployment_mode",
				AffectedStage: StageGenerate,
				Reason:        "Deployment mode changed",
				OldValue:      fp.DeploymentMode,
				NewValue:      current.DeploymentMode,
			})
		}

		if fp.AppDisplayName != current.AppDisplayName && current.AppDisplayName != "" {
			changes = append(changes, StateChange{
				ChangeType:    "app_display_name",
				AffectedStage: StageGenerate,
				Reason:        "App display name changed",
				OldValue:      fp.AppDisplayName,
				NewValue:      current.AppDisplayName,
			})
		}

		if fp.AppDescription != current.AppDescription && current.AppDescription != "" {
			changes = append(changes, StateChange{
				ChangeType:    "app_description",
				AffectedStage: StageGenerate,
				Reason:        "App description changed",
			})
		}

		if fp.IconPath != current.IconPath && current.IconPath != "" {
			changes = append(changes, StateChange{
				ChangeType:    "icon_path",
				AffectedStage: StageGenerate,
				Reason:        "Icon path changed",
				OldValue:      fp.IconPath,
				NewValue:      current.IconPath,
			})
		}
	}

	// Check build inputs
	if buildState, ok := stored.Stages[StageBuild]; ok {
		fp := buildState.InputFingerprint

		if !stringSlicesEqual(fp.Platforms, current.Platforms) {
			changes = append(changes, StateChange{
				ChangeType:    "platforms",
				AffectedStage: StageBuild,
				Reason:        "Target platforms changed",
			})
		}

		if fp.SigningEnabled != current.SigningEnabled {
			changes = append(changes, StateChange{
				ChangeType:    "signing_enabled",
				AffectedStage: StageBuild,
				Reason:        "Signing configuration toggled",
			})
		}

		if fp.SigningConfigHash != current.SigningConfigHash && current.SigningConfigHash != "" {
			changes = append(changes, StateChange{
				ChangeType:    "signing_config",
				AffectedStage: StageBuild,
				Reason:        "Signing certificates changed",
			})
		}

		if fp.OutputLocation != current.OutputLocation && current.OutputLocation != "" {
			changes = append(changes, StateChange{
				ChangeType:    "output_location",
				AffectedStage: StageBuild,
				Reason:        "Output location changed",
				OldValue:      fp.OutputLocation,
				NewValue:      current.OutputLocation,
			})
		}
	}

	return changes, nil
}

// ComputeAffectedStages returns stages that need re-running based on changes.
// Returns stages from the earliest affected stage through the end.
func (d *StalenessDetector) ComputeAffectedStages(changes []StateChange) []string {
	if len(changes) == 0 {
		return nil
	}

	earliestIdx := len(StageOrder)

	for _, change := range changes {
		for i, stage := range StageOrder {
			if stage == change.AffectedStage && i < earliestIdx {
				earliestIdx = i
				break
			}
		}
	}

	if earliestIdx >= len(StageOrder) {
		return nil
	}

	return StageOrder[earliestIdx:]
}

// CheckManifestFreshness checks if a manifest file has changed since state was stored.
// Returns (isFresh, change, error).
func (d *StalenessDetector) CheckManifestFreshness(
	ctx context.Context,
	scenarioName string,
	manifestPath string,
) (bool, *StateChange, error) {
	if manifestPath == "" {
		return true, nil, nil
	}

	stored, err := d.store.Get(ctx, scenarioName)
	if err != nil {
		return false, nil, err
	}
	if stored == nil {
		return true, nil, nil // No stored state
	}

	bundleState, ok := stored.Stages[StageBundle]
	if !ok {
		return true, nil, nil // No bundle state
	}

	fp := bundleState.InputFingerprint

	// Compute current hash
	currentHash, currentMtime, err := ComputeManifestHash(manifestPath)
	if err != nil {
		return false, nil, err
	}

	// Same hash = fresh
	if fp.ManifestHash == currentHash {
		return true, nil, nil
	}

	_ = currentMtime // Could be used for future optimization

	return false, &StateChange{
		ChangeType:    "manifest_content",
		AffectedStage: StagePreflight,
		Reason:        "Manifest file content changed",
		OldValue:      truncateHash(fp.ManifestHash),
		NewValue:      truncateHash(currentHash),
	}, nil
}

// BuildValidationStatus creates a ValidationStatus from current state and changes.
func (d *StalenessDetector) BuildValidationStatus(
	scenarioName string,
	stored *ScenarioState,
	changes []StateChange,
) *ValidationStatus {
	status := &ValidationStatus{
		ScenarioName:   scenarioName,
		OverallStatus:  StatusNone,
		Stages:         make(map[string]StageStatus),
		PendingChanges: changes,
	}

	if stored == nil {
		for _, stage := range StageOrder {
			status.Stages[stage] = StageStatus{
				Stage:    stage,
				Status:   StatusNone,
				CanReuse: false,
			}
		}
		return status
	}

	// Mark affected stages
	affectedStages := d.ComputeAffectedStages(changes)
	affectedSet := make(map[string]bool)
	for _, s := range affectedStages {
		affectedSet[s] = true
	}

	var lastValidated *time.Time
	hasValid := false
	hasStale := false

	for _, stage := range StageOrder {
		ss, ok := stored.Stages[stage]
		if !ok {
			status.Stages[stage] = StageStatus{
				Stage:    stage,
				Status:   StatusNone,
				CanReuse: false,
			}
			continue
		}

		stageStatus := StageStatus{
			Stage:   stage,
			Status:  ss.Status,
			LastRun: ss.ValidatedAt.Format(time.RFC3339),
		}

		if affectedSet[stage] {
			stageStatus.Status = StatusStale
			stageStatus.CanReuse = false
			// Find the reason for this stage
			for _, c := range changes {
				if c.AffectedStage == stage {
					stageStatus.StalenessReason = c.Reason
					break
				}
			}
			hasStale = true
		} else if ss.Status == StatusValid {
			stageStatus.CanReuse = true
			hasValid = true
			if lastValidated == nil || ss.ValidatedAt.After(*lastValidated) {
				lastValidated = &ss.ValidatedAt
			}
		}

		status.Stages[stage] = stageStatus
	}

	// Determine overall status
	if hasStale {
		if hasValid {
			status.OverallStatus = "partial"
		} else {
			status.OverallStatus = StatusStale
		}
	} else if hasValid {
		status.OverallStatus = StatusValid
	}

	status.LastValidated = lastValidated

	return status
}

// --- Helper functions ---

func truncateHash(hash string) string {
	if len(hash) > 12 {
		return hash[:12]
	}
	return hash
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	// Sort copies for comparison
	aCopy := make([]string, len(a))
	bCopy := make([]string, len(b))
	copy(aCopy, a)
	copy(bCopy, b)
	sort.Strings(aCopy)
	sort.Strings(bCopy)
	for i := range aCopy {
		if aCopy[i] != bCopy[i] {
			return false
		}
	}
	return true
}
