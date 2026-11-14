// Package mocks provides controllable collaborators for orchestration tests.
package mocks

import (
	"context"
	"time"

	"agent-manager/internal/repository"
)

var _ repository.StatsRepository = (*FakeStatsRepository)(nil)

// FakeStatsRepository is a reusable fake for the pricing model-ranking seam.
type FakeStatsRepository struct {
	PopularModels []string
	RecentModels  []string

	PopularModelsErr error
	RecentModelsErr  error
}

func NewFakeStatsRepository() *FakeStatsRepository {
	return &FakeStatsRepository{}
}

func (f *FakeStatsRepository) GetPopularModels(context.Context, time.Time, int) ([]string, error) {
	if f.PopularModelsErr != nil {
		return nil, f.PopularModelsErr
	}
	return append([]string(nil), f.PopularModels...), nil
}

func (f *FakeStatsRepository) GetRecentModels(context.Context, int) ([]string, error) {
	if f.RecentModelsErr != nil {
		return nil, f.RecentModelsErr
	}
	return append([]string(nil), f.RecentModels...), nil
}
