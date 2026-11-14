// This file implements profile lifecycle service operations.
package orchestration

import (
	"context"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// ProfileService is the handler-facing profile and declaration boundary.
// Orchestrator satisfies it today; keeping this slice explicit prevents new
// profile callers from depending on the aggregate orchestration surface.
type ProfileService interface {
	CreateProfile(context.Context, *domain.AgentProfile) (*domain.AgentProfile, error)
	GetProfile(context.Context, uuid.UUID) (*domain.AgentProfile, error)
	ListProfiles(context.Context, ListOptions) ([]*domain.AgentProfile, error)
	UpdateProfile(context.Context, *domain.AgentProfile) (*domain.AgentProfile, error)
	DeleteProfile(context.Context, uuid.UUID) error
	EnsureProfile(context.Context, EnsureProfileRequest) (*EnsureProfileResult, error)
	ReconcileScenarioProfiles(context.Context, ReconcileScenarioProfilesRequest) (*ReconcileScenarioProfilesResult, error)
	ReconcileScenarioDeclarations(context.Context, ReconcileScenarioDeclarationsRequest) (*ReconcileScenarioDeclarationsResult, error)
	ReconcileSelfDeclarations(context.Context, string) (*ReconcileScenarioDeclarationsResult, error)
}

var _ ProfileService = (*Orchestrator)(nil)
