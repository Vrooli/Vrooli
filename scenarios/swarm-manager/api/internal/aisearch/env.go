package aisearch

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Env-var names are declared as constants so the resolvers, the test harness,
// and any future operator tooling share a single source of truth.
const (
	EnvOllamaURL              = "OLLAMA_URL"
	EnvQdrantURL              = "QDRANT_URL"
	EnvQdrantBaseURL          = "QDRANT_BASE_URL"
	EnvQdrantPort             = "QDRANT_PORT"
	EnvQdrantAPIKey           = "QDRANT_API_KEY"
	EnvAISearchModel          = "AI_SEARCH_MODEL"
	EnvAISearchThreshold      = "AI_SEARCH_THRESHOLD"
	EnvAISearchBacklogColl    = "AI_SEARCH_BACKLOG_COLLECTION"
	EnvAISearchInitiativeColl = "AI_SEARCH_INITIATIVE_COLLECTION"

	DefaultEmbeddingModel       = "nomic-embed-text"
	DefaultVectorDimensions     = 768
	DefaultThreshold            = 0.5
	DefaultBacklogCollection    = "swarm-manager-backlog"
	DefaultInitiativeCollection = "swarm-manager-initiatives"
)

// ResolveOllamaURL returns the configured Ollama URL, or the empty string if
// none is configured. Callers treat an empty URL as "AI search disabled" and
// degrade gracefully.
func ResolveOllamaURL() string {
	return strings.TrimSpace(os.Getenv(EnvOllamaURL))
}

// ResolveQdrantURL resolves the Qdrant base URL from the standard precedence:
// QDRANT_URL > QDRANT_BASE_URL (Vrooli resource export) > http://localhost:{QDRANT_PORT}.
// Returns "" if none are set.
func ResolveQdrantURL() string {
	if v := strings.TrimSpace(os.Getenv(EnvQdrantURL)); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv(EnvQdrantBaseURL)); v != "" {
		return v
	}
	if port := strings.TrimSpace(os.Getenv(EnvQdrantPort)); port != "" {
		return fmt.Sprintf("http://localhost:%s", port)
	}
	return ""
}

// Config is the resolved configuration for constructing a swarm-manager
// aisearch Service. Unset string fields mean "use default"; unset URL fields
// mean "that subsystem is disabled and the service degrades to fallback."
type Config struct {
	OllamaURL            string
	QdrantURL            string
	QdrantAPIKey         string
	EmbeddingModel       string
	VectorDimensions     int
	Threshold            float64
	BacklogCollection    string
	InitiativeCollection string
}

// LoadConfigFromEnv reads the full aisearch configuration from the process
// environment, applying defaults. It never returns an error — missing values
// become defaults, and missing URLs are represented as empty strings that the
// caller must interpret as "disabled."
func LoadConfigFromEnv() Config {
	cfg := Config{
		OllamaURL:            ResolveOllamaURL(),
		QdrantURL:            ResolveQdrantURL(),
		QdrantAPIKey:         strings.TrimSpace(os.Getenv(EnvQdrantAPIKey)),
		EmbeddingModel:       DefaultEmbeddingModel,
		VectorDimensions:     DefaultVectorDimensions,
		Threshold:            DefaultThreshold,
		BacklogCollection:    DefaultBacklogCollection,
		InitiativeCollection: DefaultInitiativeCollection,
	}
	if v := strings.TrimSpace(os.Getenv(EnvAISearchModel)); v != "" {
		cfg.EmbeddingModel = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvAISearchBacklogColl)); v != "" {
		cfg.BacklogCollection = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvAISearchInitiativeColl)); v != "" {
		cfg.InitiativeCollection = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvAISearchThreshold)); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed > 0 {
			cfg.Threshold = parsed
		}
	}
	return cfg
}
