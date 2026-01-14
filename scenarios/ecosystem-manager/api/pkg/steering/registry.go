package steering

import (
	"strings"

	"github.com/ecosystem-manager/api/pkg/tasks"
)

// Registry manages steering providers and determines which strategy applies to a task.
// It implements the RegistryAPI interface.
type Registry struct {
	providers map[SteeringStrategy]SteeringProvider
}

// Compile-time interface assertion
var _ RegistryAPI = (*Registry)(nil)

// NewRegistry creates a new steering registry with the provided providers.
func NewRegistry(providers map[SteeringStrategy]SteeringProvider) *Registry {
	return &Registry{
		providers: providers,
	}
}

// GetProvider returns the appropriate steering provider for a task.
// The provider is determined by examining the task's steering-related fields.
func (r *Registry) GetProvider(task *tasks.TaskItem) SteeringProvider {
	strategy := r.DetermineStrategy(task)
	if provider, ok := r.providers[strategy]; ok {
		return provider
	}
	// Fallback to none provider if strategy not registered
	if provider, ok := r.providers[StrategyNone]; ok {
		return provider
	}
	return nil
}

// DetermineStrategy inspects task fields to determine which steering strategy applies.
// Priority order:
// 1. Profile (AutoSteerProfileID set) - most specific configuration
// 2. Queue (SteeringQueue has items) - ordered list of modes
// 3. Manual (SteerMode set) - single mode selection
// 4. None (default) - no explicit steering
func (r *Registry) DetermineStrategy(task *tasks.TaskItem) SteeringStrategy {
	if task == nil {
		return StrategyNone
	}

	// Profile takes highest priority - it's the most specific configuration
	if strings.TrimSpace(task.AutoSteerProfileID) != "" {
		return StrategyProfile
	}

	// Queue comes next - ordered list of modes
	if len(task.SteeringQueue) > 0 {
		return StrategyQueue
	}

	// Manual mode - single mode selection
	if strings.TrimSpace(task.SteerMode) != "" {
		return StrategyManual
	}

	// Default to no steering
	return StrategyNone
}

// HasProvider checks if a provider is registered for a strategy.
func (r *Registry) HasProvider(strategy SteeringStrategy) bool {
	_, ok := r.providers[strategy]
	return ok
}

// RegisterProvider adds or replaces a provider for a strategy.
func (r *Registry) RegisterProvider(strategy SteeringStrategy, provider SteeringProvider) {
	if r.providers == nil {
		r.providers = make(map[SteeringStrategy]SteeringProvider)
	}
	r.providers[strategy] = provider
}
