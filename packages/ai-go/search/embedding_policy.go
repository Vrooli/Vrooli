package aisearch

import (
	"context"
	"errors"
	"fmt"
	"strings"

	ollamapolicy "github.com/vrooli/ai-go/ollama/policy"
)

type EmbeddingPolicy struct {
	Role                string
	Model               string
	Dimensions          int
	PolicySchemaVersion string
}

func ResolveEmbeddingConfig(ctx context.Context, cfg Config) (Config, error) {
	return ResolveEmbeddingConfigWithResolver(ctx, cfg, ollamapolicy.Resolver{})
}

func ResolveEmbeddingConfigWithResolver(ctx context.Context, cfg Config, resolver ollamapolicy.Resolver) (Config, error) {
	role := strings.TrimSpace(cfg.EmbedRole)
	if role == "" {
		role = DefaultEmbedRole
	}
	resolved, err := resolver.ResolveRole(ctx, role)
	if err != nil {
		return Config{}, err
	}
	return ConfigWithResolvedEmbedding(cfg, resolved)
}

func ResolveEmbeddingPolicy(ctx context.Context, role string) (EmbeddingPolicy, error) {
	return ResolveEmbeddingPolicyWithResolver(ctx, role, ollamapolicy.Resolver{})
}

func ResolveEmbeddingPolicyWithResolver(ctx context.Context, role string, resolver ollamapolicy.Resolver) (EmbeddingPolicy, error) {
	role = strings.TrimSpace(role)
	if role == "" {
		role = DefaultEmbedRole
	}
	resolved, err := resolver.ResolveRole(ctx, role)
	if err != nil {
		return EmbeddingPolicy{}, err
	}
	return EmbeddingPolicyFromResolved(resolved)
}

func NewServiceForTuningResolved(ctx context.Context, tuning TuningConfig, deps EngineDeps) (TunedEngine, error) {
	return NewServiceForTuningResolvedWithResolver(ctx, tuning, deps, ollamapolicy.Resolver{})
}

func NewServiceForTuningResolvedWithResolver(ctx context.Context, tuning TuningConfig, deps EngineDeps, resolver ollamapolicy.Resolver) (TunedEngine, error) {
	var err error
	deps, err = ResolveEngineDepsEmbeddingWithResolver(ctx, deps, resolver)
	if err != nil {
		return TunedEngine{}, err
	}
	return NewServiceForTuning(tuning, deps), nil
}

func ResolveEngineDepsEmbedding(ctx context.Context, deps EngineDeps) (EngineDeps, error) {
	return ResolveEngineDepsEmbeddingWithResolver(ctx, deps, ollamapolicy.Resolver{})
}

func ResolveEngineDepsEmbeddingWithResolver(ctx context.Context, deps EngineDeps, resolver ollamapolicy.Resolver) (EngineDeps, error) {
	cfg, err := ResolveEmbeddingConfigWithResolver(ctx, Config{EmbedRole: deps.EmbedRole}, resolver)
	if err != nil {
		return EngineDeps{}, err
	}
	deps.EmbedRole = cfg.EmbedRole
	deps.EmbedModel = cfg.EmbedModel
	deps.EmbedDimensions = cfg.EmbedDimensions
	deps.EmbedPolicySchemaVersion = cfg.EmbedPolicySchemaVersion
	return deps, nil
}

func ConfigWithResolvedEmbedding(cfg Config, resolved ollamapolicy.ResolvedRole) (Config, error) {
	policy, err := EmbeddingPolicyFromResolved(resolved)
	if err != nil {
		return Config{}, err
	}
	cfg.EmbedRole = policy.Role
	cfg.EmbedModel = policy.Model
	cfg.EmbedDimensions = policy.Dimensions
	cfg.EmbedPolicySchemaVersion = policy.PolicySchemaVersion
	return cfg, nil
}

func EmbeddingPolicyFromResolved(resolved ollamapolicy.ResolvedRole) (EmbeddingPolicy, error) {
	model := strings.TrimSpace(resolved.Model)
	if model == "" {
		return EmbeddingPolicy{}, errors.New("resolved embedding policy missing model")
	}
	if resolved.EmbeddingDimensions <= 0 {
		return EmbeddingPolicy{}, fmt.Errorf("resolved embedding policy for %s missing embedding_dimensions", model)
	}
	role := strings.TrimSpace(resolved.Role)
	if role == "" {
		role = strings.TrimSpace(resolved.Source)
	}
	if role == "" || role == "model" {
		role = DefaultEmbedRole
	}
	return EmbeddingPolicy{
		Role:                role,
		Model:               model,
		Dimensions:          resolved.EmbeddingDimensions,
		PolicySchemaVersion: strings.TrimSpace(resolved.SchemaVersion),
	}, nil
}
