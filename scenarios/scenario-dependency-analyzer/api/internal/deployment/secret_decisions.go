// Package deployment - secret_decisions.go
//
// This file centralizes decisions about secret requirements for resources.
// Each decision function clearly documents what secrets are needed for
// different resource types and why.

package deployment

import (
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
//   - Automation (n8n, huginn, windmill): API key + webhook secret
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
		return classifyObjectStorageSecrets(normalized)

	case "n8n", "huginn", "windmill":
		return classifyAutomationSecrets(normalized)

	case "claude-code", "anthropic", "openai":
		return classifyAIAPISecrets(normalized)

	case "qdrant":
		return classifyVectorDBSecrets()

	case "browserless", "playwright":
		return classifyBrowserAutomationSecrets(normalized)

	case "ollama":
		// Decision: Local Ollama typically doesn't need secrets
		return nil

	default:
		return nil
	}
}

// classifyDatabaseSecrets: Full SQL databases need user/password pairs.
func classifyDatabaseSecrets(normalized string) *SecretClassification {
	return &SecretClassification{
		SecretType:      "database_credentials",
		RequiredSecrets: []string{normalized + "_password", normalized + "_user"},
		PlaybookRef:     "secrets-manager/playbooks/database-credentials.md",
	}
}

// classifyCacheSecrets: Redis needs only a password.
func classifyCacheSecrets() *SecretClassification {
	return &SecretClassification{
		SecretType:      "cache_credentials",
		RequiredSecrets: []string{"redis_password"},
		PlaybookRef:     "secrets-manager/playbooks/cache-credentials.md",
	}
}

// classifyObjectStorageSecrets: S3-compatible storage needs access key + secret.
func classifyObjectStorageSecrets(normalized string) *SecretClassification {
	return &SecretClassification{
		SecretType:      "object_storage_credentials",
		RequiredSecrets: []string{normalized + "_access_key", normalized + "_secret_key"},
		PlaybookRef:     "secrets-manager/playbooks/object-storage.md",
	}
}

// classifyAutomationSecrets: Workflow platforms need API key + webhook secret.
func classifyAutomationSecrets(normalized string) *SecretClassification {
	return &SecretClassification{
		SecretType:      "automation_credentials",
		RequiredSecrets: []string{normalized + "_api_key", normalized + "_webhook_secret"},
		PlaybookRef:     "secrets-manager/playbooks/automation-platform.md",
	}
}

// classifyAIAPISecrets: Cloud AI APIs need an API key.
func classifyAIAPISecrets(normalized string) *SecretClassification {
	return &SecretClassification{
		SecretType:      "ai_api_key",
		RequiredSecrets: []string{normalized + "_api_key"},
		PlaybookRef:     "secrets-manager/playbooks/ai-api-keys.md",
	}
}

// classifyVectorDBSecrets: Vector databases need an API key.
func classifyVectorDBSecrets() *SecretClassification {
	return &SecretClassification{
		SecretType:      "vector_db_credentials",
		RequiredSecrets: []string{"qdrant_api_key"},
		PlaybookRef:     "secrets-manager/playbooks/vector-db.md",
	}
}

// classifyBrowserAutomationSecrets: Browser automation services need auth tokens.
func classifyBrowserAutomationSecrets(normalized string) *SecretClassification {
	return &SecretClassification{
		SecretType:      "browser_automation_token",
		RequiredSecrets: []string{normalized + "_token"},
		PlaybookRef:     "secrets-manager/playbooks/browser-automation.md",
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
