// Package services provides application services for the Agent Inbox scenario.
//
// This file implements the ModelRegistry service for caching model metadata.
// The registry fetches model information from resource-openrouter CLI and caches
// it to avoid repeated CLI calls. This enables efficient lookup of model properties
// like context length for context window management.
//
// ARCHITECTURE:
// - ModelRegistry: Central cache for model metadata
// - Uses FetchModels from integrations package
// - Provides O(1) lookup by model ID
// - TTL-based cache refresh (configurable via config.Integration.ModelCacheTTL)
package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"agent-inbox/config"
	"agent-inbox/integrations"
)

// ModelRegistry caches model metadata from OpenRouter.
// Thread-safe for concurrent access.
type ModelRegistry struct {
	cfg *config.Config

	// Cache state
	mu        sync.RWMutex
	models    map[string]*integrations.ModelInfo // modelID -> info
	modelList []integrations.ModelInfo
	cacheTime time.Time
}

// NewModelRegistry creates a new ModelRegistry with default config.
func NewModelRegistry() *ModelRegistry {
	return NewModelRegistryWithConfig(config.Default())
}

// NewModelRegistryWithConfig creates a ModelRegistry with explicit config.
// This enables testing and custom configuration injection.
func NewModelRegistryWithConfig(cfg *config.Config) *ModelRegistry {
	return &ModelRegistry{
		cfg:    cfg,
		models: make(map[string]*integrations.ModelInfo),
	}
}

// RefreshModels fetches fresh model data from resource-openrouter CLI.
// This is typically called on startup and when cache expires.
func (r *ModelRegistry) RefreshModels(ctx context.Context) error {
	models, err := integrations.FetchModels(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch models: %w", err)
	}

	// Build lookup map
	modelMap := make(map[string]*integrations.ModelInfo, len(models))
	for i := range models {
		modelMap[models[i].ID] = &models[i]
	}

	// Update cache atomically
	r.mu.Lock()
	r.models = modelMap
	r.modelList = models
	r.cacheTime = time.Now()
	r.mu.Unlock()

	log.Printf("[INFO] Model registry refreshed: %d models cached", len(models))
	return nil
}

// GetModels returns all cached models, refreshing if cache is stale.
func (r *ModelRegistry) GetModels(ctx context.Context) ([]integrations.ModelInfo, error) {
	r.mu.RLock()
	cached := r.modelList
	cacheAge := time.Since(r.cacheTime)
	r.mu.RUnlock()

	// Return cached if still valid
	if len(cached) > 0 && cacheAge < r.cfg.Integration.ModelCacheTTL {
		return cached, nil
	}

	// Cache expired or empty - refresh
	if err := r.RefreshModels(ctx); err != nil {
		// If we have stale cache, return it with a warning
		if len(cached) > 0 {
			log.Printf("[WARN] Failed to refresh model cache, using stale data: %v", err)
			return cached, nil
		}
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.modelList, nil
}

// GetModel returns info for a specific model by ID.
// Returns nil if model not found (after cache refresh attempt).
func (r *ModelRegistry) GetModel(ctx context.Context, modelID string) (*integrations.ModelInfo, error) {
	r.mu.RLock()
	model, exists := r.models[modelID]
	cacheAge := time.Since(r.cacheTime)
	r.mu.RUnlock()

	// Return cached if found and cache is valid
	if exists && cacheAge < r.cfg.Integration.ModelCacheTTL {
		return model, nil
	}

	// Cache expired or model not found - refresh and try again
	if err := r.RefreshModels(ctx); err != nil {
		// If we found the model before refresh failed, return it
		if exists {
			log.Printf("[WARN] Failed to refresh model cache: %v", err)
			return model, nil
		}
		return nil, err
	}

	// Check again after refresh
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.models[modelID], nil // May still be nil if model doesn't exist
}

// GetContextLength returns the context length for a model.
// Falls back to DefaultContextLength if model not found or has no context length.
func (r *ModelRegistry) GetContextLength(ctx context.Context, modelID string) (int, error) {
	model, err := r.GetModel(ctx, modelID)
	if err != nil {
		return r.cfg.AI.DefaultContextLength, err
	}

	if model == nil || model.ContextLength == 0 {
		log.Printf("[DEBUG] No context length for model %s, using default %d",
			modelID, r.cfg.AI.DefaultContextLength)
		return r.cfg.AI.DefaultContextLength, nil
	}

	return model.ContextLength, nil
}

// GetEffectiveContextLimit returns the usable context limit for a model,
// accounting for the reserved portion for completion tokens.
// This is the maximum number of tokens available for input messages.
func (r *ModelRegistry) GetEffectiveContextLimit(ctx context.Context, modelID string) (int, error) {
	contextLength, err := r.GetContextLength(ctx, modelID)
	if err != nil {
		// Still return a usable value even on error
		contextLength = r.cfg.AI.DefaultContextLength
	}

	// Reserve percentage for completion
	reservePercent := r.cfg.AI.CompletionReservePercent
	if reservePercent <= 0 || reservePercent >= 100 {
		reservePercent = 25 // Sanity default
	}

	effectiveLimit := contextLength * (100 - reservePercent) / 100
	return effectiveLimit, err
}

// SupportsImageGeneration returns true if the model can generate images.
func (r *ModelRegistry) SupportsImageGeneration(ctx context.Context, modelID string) (bool, error) {
	model, err := r.GetModel(ctx, modelID)
	if err != nil {
		return false, err
	}
	if model == nil {
		return false, nil
	}
	return model.SupportsImageGeneration(), nil
}

// IsCacheValid returns true if the cache is populated and not expired.
func (r *ModelRegistry) IsCacheValid() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.models) > 0 && time.Since(r.cacheTime) < r.cfg.Integration.ModelCacheTTL
}

// CacheStats returns statistics about the cache for debugging.
func (r *ModelRegistry) CacheStats() (modelCount int, cacheAge time.Duration) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.models), time.Since(r.cacheTime)
}
