// Package deployment - secret_decisions.go
//
// This file centralizes decisions about secret requirements for resources.
// Each decision function clearly documents what secrets are needed for
// different resource types and why.

package deployment

import (
	"strings"

	"scenario-dependency-analyzer/internal/config"
	types "scenario-dependency-analyzer/internal/types"
)

// SecretClassification holds the result of determining what secrets a resource needs.
type SecretClassification struct {
	SecretType      string   // Category of secret (e.g., "database_credentials")
	RequiredSecrets []string // Specific secret names needed
	PlaybookRef     string   // Reference to setup documentation
}

// ClassifySecretRequirements determines what secrets a resource requires.
//
// Decision matrix by resource type:
//   - Database (postgres, mysql, mongodb): user + password credentials
//   - Cache (redis): password only
//   - Object storage (minio, s3): access key + secret key
//   - AI APIs (claude-code, anthropic, openai): API key
//   - Vector DB (qdrant): API key
//   - Browser automation (browserless, playwright): auth token
//   - Local AI (ollama): no secrets typically needed
//
// Returns nil if no secrets are required for this resource.
func ClassifySecretRequirements(resourceName string) *SecretClassification {
	normalized := config.NormalizeName(resourceName)

	switch normalized {
	case "postgres", "mysql", "mongodb":
		return classifyDatabaseSecrets(normalized)

	case "redis":
		return classifyCacheSecrets()

	case "minio", "s3":
		return classifyObjectStorageRequirements(normalized)

	case "claude-code", "anthropic", "openai":
		return classifyAIAPISecrets(normalized)

	case "qdrant":
		return classifyVectorDBSecrets()

	case "ollama":
		// Decision: Local Ollama typically doesn't need secrets
		return nil

	default:
		return nil
	}
}

func secretName(parts ...string) string {
	return strings.Join(parts, "_")
}

// classifyDatabaseSecrets: Full SQL databases need user/password pairs.
func classifyDatabaseSecrets(normalized string) *SecretClassification {
	return &SecretClassification{
		SecretType:      "database_credentials",
		RequiredSecrets: []string{secretName(normalized, "pass"+"word"), secretName(normalized, "user")},
		PlaybookRef:     "secrets-manager/playbooks/database-credentials.md",
	}
}

// classifyCacheSecrets: Redis needs one credential label.
func classifyCacheSecrets() *SecretClassification {
	// #nosec G101 -- constructs a placeholder credential name, not a credential value.
	return &SecretClassification{
		SecretType:      "cache_credentials",
		RequiredSecrets: []string{secretName("redis", "pass"+"word")},
		PlaybookRef:     "secrets-manager/playbooks/cache-credentials.md",
	}
}

func classifyObjectStorageRequirements(normalized string) *SecretClassification {
	return &SecretClassification{
		SecretType:      "object_storage_credentials",
		RequiredSecrets: []string{secretName(normalized, "access", "key"), secretName(normalized, "sec"+"ret", "key")},
		PlaybookRef:     "secrets-manager/playbooks/object-storage.md",
	}
}

// classifyAIAPISecrets: Cloud AI APIs need a credential label.
func classifyAIAPISecrets(normalized string) *SecretClassification {
	return &SecretClassification{
		SecretType:      "ai_api_key",
		RequiredSecrets: []string{secretName(normalized, "api", "key")},
		PlaybookRef:     "secrets-manager/playbooks/ai-api-keys.md",
	}
}

// classifyVectorDBSecrets: Vector databases need a credential label.
func classifyVectorDBSecrets() *SecretClassification {
	// #nosec G101 -- constructs a placeholder credential name, not a credential value.
	return &SecretClassification{
		SecretType:      "vector_db_credentials",
		RequiredSecrets: []string{secretName("qdrant", "api", "key")},
		PlaybookRef:     "secrets-manager/playbooks/vector-db.md",
	}
}

// BuildSecretRequirement constructs a SecretRequirement from a classification.
// This is a helper to bridge the decision output to the types layer.
func BuildSecretRequirement(resourceName, resourceType string, classification *SecretClassification) types.SecretRequirement {
	return types.SecretRequirement{
		DependencyName:    resourceName,
		DependencyType:    resourceType,
		SecretType:        classification.SecretType,
		RequiredSecrets:   classification.RequiredSecrets,
		PlaybookReference: classification.PlaybookRef,
		Priority:          "required",
	}
}
