// Package aisearch is the optional semantic-ranking overlay for the fleet
// Dependency & Vulnerability Intelligence corpus. The SQLite corpus
// (internal/dependencies) stays the source of truth; this package embeds each
// dependency record through the Ollama embedding role and upserts it into a
// Qdrant collection so a free-text query can be ranked by vector similarity.
//
// Everything here is best-effort: when Ollama or Qdrant is unreachable, the
// dependencies Service degrades to deterministic TEXT/structured search. The
// Embedder + VectorStore seams mirror cli-health's proven aisearch substrate;
// tests substitute fakes.
package aisearch

import (
	"os"
	"strings"
	"time"
)

// Env vars tune the embedder / vector store. The SECURITY_HEALTH_ prefix keeps
// this tunable independently of other aisearch users on the same host.
const (
	EnvQdrantURL    = "SECURITY_HEALTH_QDRANT_URL"
	EnvQdrantAPIKey = "SECURITY_HEALTH_QDRANT_API_KEY"
	EnvOllamaRole   = "SECURITY_HEALTH_EMBED_ROLE"
	EnvDisabled     = "SECURITY_HEALTH_AISEARCH_DISABLED"

	DefaultCollection = "security-health-deps"
	DefaultVectorSize = 768
	DefaultEmbedRole  = "embedding.default"
	DefaultQdrantURL  = "http://127.0.0.1:6333"

	// httpTimeout bounds every Qdrant request so a wedged backend can never
	// block a reconcile or a search past this.
	httpTimeout = 15 * time.Second
)

// Config holds the embedder/vector-store tunables read from environment.
type Config struct {
	Disabled   bool
	QdrantURL  string
	QdrantKey  string
	EmbedRole  string
	Collection string
	VectorSize int
}

// LoadConfigFromEnv reads tunables from the environment, falling back to
// defaults (with a warning log) when values are absent or malformed.
func LoadConfigFromEnv() Config {
	return Config{
		Disabled:   envBool(EnvDisabled),
		QdrantURL:  envString(EnvQdrantURL, DefaultQdrantURL),
		QdrantKey:  envString(EnvQdrantAPIKey, ""),
		EmbedRole:  envString(EnvOllamaRole, DefaultEmbedRole),
		Collection: DefaultCollection,
		VectorSize: DefaultVectorSize,
	}
}

func envBool(name string) bool {
	switch strings.TrimSpace(strings.ToLower(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func envString(name, def string) string {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return def
	}
	return raw
}
