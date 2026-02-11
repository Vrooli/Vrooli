package services

import (
	"context"
	"errors"
	"os"

	"app-monitor-api/repository"
)

// PresetService handles business logic for workspace preset management.
type PresetService struct {
	repo repository.WorkspacePresetRepository
}

// NewPresetService creates a new PresetService.
func NewPresetService(repo repository.WorkspacePresetRepository) *PresetService {
	return &PresetService{repo: repo}
}

// ListPresets returns all workspace presets.
func (s *PresetService) ListPresets(ctx context.Context) ([]repository.WorkspacePreset, error) {
	if s.repo == nil {
		return nil, ErrDatabaseUnavailable
	}
	return s.repo.ListPresets(ctx)
}

// GetPreset returns a single workspace preset by ID.
func (s *PresetService) GetPreset(ctx context.Context, id string) (*repository.WorkspacePreset, error) {
	if s.repo == nil {
		return nil, ErrDatabaseUnavailable
	}
	if id == "" {
		return nil, ErrAppIdentifierRequired
	}
	preset, err := s.repo.GetPreset(ctx, id)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrPresetNotFound
		}
		return nil, err
	}
	return preset, nil
}

// CreatePreset validates and creates a new workspace preset.
func (s *PresetService) CreatePreset(ctx context.Context, preset *repository.WorkspacePreset) error {
	if s.repo == nil {
		return ErrDatabaseUnavailable
	}
	if preset.Name == "" {
		return ErrPresetNameRequired
	}
	if len(preset.Name) > 100 {
		preset.Name = preset.Name[:100]
	}
	return s.repo.CreatePreset(ctx, preset)
}

// UpdatePreset validates and updates an existing workspace preset.
func (s *PresetService) UpdatePreset(ctx context.Context, preset *repository.WorkspacePreset) error {
	if s.repo == nil {
		return ErrDatabaseUnavailable
	}
	if preset.Name == "" {
		return ErrPresetNameRequired
	}
	if len(preset.Name) > 100 {
		preset.Name = preset.Name[:100]
	}
	err := s.repo.UpdatePreset(ctx, preset)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrPresetNotFound
		}
		return err
	}
	return nil
}

// DeletePreset removes a workspace preset by ID.
func (s *PresetService) DeletePreset(ctx context.Context, id string) error {
	if s.repo == nil {
		return ErrDatabaseUnavailable
	}
	if id == "" {
		return ErrAppIdentifierRequired
	}
	err := s.repo.DeletePreset(ctx, id)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrPresetNotFound
		}
		return err
	}
	return nil
}
