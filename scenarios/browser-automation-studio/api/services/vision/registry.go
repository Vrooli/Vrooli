package vision

import (
	"context"
	"errors"
	"sync"
)

// Registry errors.
var (
	// ErrNavigatorNotFound is returned when a navigator type is not registered.
	ErrNavigatorNotFound = errors.New("navigator not found")

	// ErrNavigatorNotAvailable is returned when a navigator exists but is not available.
	ErrNavigatorNotAvailable = errors.New("navigator is not available")

	// ErrNavigatorNotAllowed is returned when a navigator is not allowed for the client source.
	ErrNavigatorNotAllowed = errors.New("navigator is not allowed for this client source")

	// ErrNoNavigatorsAvailable is returned when no navigators are available.
	ErrNoNavigatorsAvailable = errors.New("no navigators available")
)

// NavigatorRegistry manages available vision navigators.
type NavigatorRegistry struct {
	mu         sync.RWMutex
	navigators map[NavigatorType]VisionNavigator
	order      []NavigatorType // Order for selection priority
}

// NewNavigatorRegistry creates a new empty registry.
func NewNavigatorRegistry() *NavigatorRegistry {
	return &NavigatorRegistry{
		navigators: make(map[NavigatorType]VisionNavigator),
		order:      make([]NavigatorType, 0),
	}
}

// Register adds a navigator to the registry.
// Later registered navigators have lower selection priority.
func (r *NavigatorRegistry) Register(nav VisionNavigator) {
	r.mu.Lock()
	defer r.mu.Unlock()

	navType := nav.Type()
	if _, exists := r.navigators[navType]; !exists {
		r.order = append(r.order, navType)
	}
	r.navigators[navType] = nav
}

// Get returns a navigator by type, or ErrNavigatorNotFound if not registered.
func (r *NavigatorRegistry) Get(navType NavigatorType) (VisionNavigator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nav, exists := r.navigators[navType]
	if !exists {
		return nil, ErrNavigatorNotFound
	}
	return nav, nil
}

// SelectNavigator finds the best available navigator for the given client source.
// If preferredType is specified and available, it will be selected.
// Otherwise, the first available navigator that allows the client source is selected.
func (r *NavigatorRegistry) SelectNavigator(ctx context.Context, source ClientSource, preferredType NavigatorType) (VisionNavigator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// If a preferred type is specified, try to use it
	if preferredType != "" {
		nav, exists := r.navigators[preferredType]
		if !exists {
			return nil, ErrNavigatorNotFound
		}

		if !nav.ClientSourcePolicy().IsAllowed(source) {
			return nil, ErrNavigatorNotAllowed
		}

		if !nav.IsAvailable(ctx) {
			return nil, ErrNavigatorNotAvailable
		}

		return nav, nil
	}

	// Auto-select: find first available navigator that allows this source
	for _, navType := range r.order {
		nav := r.navigators[navType]

		if !nav.ClientSourcePolicy().IsAllowed(source) {
			continue
		}

		if !nav.IsAvailable(ctx) {
			continue
		}

		return nav, nil
	}

	return nil, ErrNoNavigatorsAvailable
}

// ListNavigators returns information about all registered navigators.
func (r *NavigatorRegistry) ListNavigators(ctx context.Context, source ClientSource) []NavigatorInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]NavigatorInfo, 0, len(r.navigators))

	for _, navType := range r.order {
		nav := r.navigators[navType]

		info := NavigatorInfo{
			Type:           navType,
			Available:      nav.IsAvailable(ctx) && nav.ClientSourcePolicy().IsAllowed(source),
			Description:    nav.Description(),
			CreditPolicy:   nav.CreditPolicy().ToInfo(),
			AllowedSources: nav.ClientSourcePolicy().AllowedSources,
		}

		// If not available, provide reason
		if !nav.IsAvailable(ctx) {
			info.UnavailableReason = nav.UnavailableReason(ctx)
		} else if !nav.ClientSourcePolicy().IsAllowed(source) {
			info.UnavailableReason = "not allowed for this client source"
		}

		// If allowed sources is nil, show all sources
		if info.AllowedSources == nil {
			info.AllowedSources = []ClientSource{ClientSourceUI, ClientSourceCLI, ClientSourceAPI}
		}

		result = append(result, info)
	}

	return result
}

// GetDefault returns the default navigator type (first in order).
func (r *NavigatorRegistry) GetDefault() NavigatorType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.order) == 0 {
		return ""
	}
	return r.order[0]
}

// Count returns the number of registered navigators.
func (r *NavigatorRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.navigators)
}
