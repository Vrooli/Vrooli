// Package deployment - swap_decisions.go
//
// This file centralizes decisions about resource swap suggestions.
// Swaps allow heavy local resources to be replaced with lighter alternatives
// for specific deployment tiers.

package deployment

import (
	"scenario-dependency-analyzer/internal/config"
	types "scenario-dependency-analyzer/internal/types"
)

// SwapRecommendation describes a suggested resource replacement.
type SwapRecommendation struct {
	Alternative     string   // The replacement resource
	Reason          string   // Why this swap is recommended
	ApplicableTiers []string // Which tiers benefit from this swap
	Relationship    string   // Type of relationship (api_alternative, managed_service, etc.)
	Impact          string   // Description of the impact
}

// DecideResourceSwaps determines what alternatives are available for a resource.
//
// Swap decisions are based on:
//   - Resource operational requirements (CPU, memory, disk)
//   - Target tier constraints (mobile can't run Ollama)
//   - Deployment model (local vs cloud)
//
// The goal is to suggest lighter alternatives that provide similar functionality
// with reduced resource requirements or better availability.
func DecideResourceSwaps(resourceName string) []SwapRecommendation {
	normalized := config.NormalizeName(resourceName)

	switch normalized {
	case "ollama":
		return decideOllamaSwaps()

	case "postgres":
		return decidePostgresSwaps()

	case "minio":
		return decideMinioSwaps()

	case "redis":
		return decideRedisSwaps()

	default:
		return nil
	}
}

// decideOllamaSwaps: Local LLM can be swapped for cloud AI APIs.
//
// Decision rationale:
//   - Mobile/SaaS can't run local LLMs efficiently
//   - Cloud APIs provide same capability with better scalability
//   - OpenRouter for cost-effective SaaS, Anthropic for enterprise reliability
func decideOllamaSwaps() []SwapRecommendation {
	return []SwapRecommendation{
		{
			Alternative:     "openrouter",
			Reason:          "Lighter alternative for SaaS/mobile deployments",
			ApplicableTiers: []string{"mobile", "saas"},
			Relationship:    "api_alternative",
			Impact:          "Reduces deployment size and resource requirements",
		},
		{
			Alternative:     "anthropic",
			Reason:          "Managed AI API for production deployments",
			ApplicableTiers: []string{"saas", "enterprise"},
			Relationship:    "api_alternative",
			Impact:          "Better scalability and reliability",
		},
	}
}

// decidePostgresSwaps: Full Postgres can be swapped for managed DBs.
//
// Decision rationale:
//   - Mobile apps should use remote databases
//   - SaaS deployments benefit from managed database services
//   - Supabase provides Postgres + extras with zero ops overhead
func decidePostgresSwaps() []SwapRecommendation {
	return []SwapRecommendation{
		{
			Alternative:     "supabase",
			Reason:          "Managed PostgreSQL with built-in APIs",
			ApplicableTiers: []string{"mobile", "saas"},
			Relationship:    "managed_service",
			Impact:          "Eliminates database management overhead",
		},
	}
}

// decideMinioSwaps: Self-hosted S3 can be swapped for cloud storage.
//
// Decision rationale:
//   - Production deployments benefit from cloud storage reliability
//   - Global availability and built-in CDN features
func decideMinioSwaps() []SwapRecommendation {
	return []SwapRecommendation{
		{
			Alternative:     "s3",
			Reason:          "Cloud object storage for production deployments",
			ApplicableTiers: []string{"saas", "enterprise"},
			Relationship:    "cloud_alternative",
			Impact:          "Better reliability and global availability",
		},
	}
}

// decideRedisSwaps: Full Redis can be swapped for embedded caching.
//
// Decision rationale:
//   - Desktop apps should minimize external service dependencies
//   - Simple caching needs don't require full Redis
//   - Embedded caches eliminate the need for a separate process
func decideRedisSwaps() []SwapRecommendation {
	return []SwapRecommendation{
		{
			Alternative:     "in-memory-cache",
			Reason:          "Embedded caching for desktop deployments",
			ApplicableTiers: []string{"desktop"},
			Relationship:    "embedded_alternative",
			Impact:          "Simplifies deployment, no separate service needed",
		},
	}
}

// BuildSwapSuggestion constructs a ResourceSwapSuggestion from a recommendation.
// This bridges the decision output to the types layer.
func BuildSwapSuggestion(resourceName string, rec SwapRecommendation) types.ResourceSwapSuggestion {
	return types.ResourceSwapSuggestion{
		OriginalResource:    resourceName,
		AlternativeResource: rec.Alternative,
		Reason:              rec.Reason,
		ApplicableTiers:     rec.ApplicableTiers,
		Relationship:        rec.Relationship,
		ImpactDescription:   rec.Impact,
	}
}
