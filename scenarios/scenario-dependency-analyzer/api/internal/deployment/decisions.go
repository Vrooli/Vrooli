// Package deployment - decisions.go
//
// This file centralizes key deployment decisions, making them explicit, testable,
// and easy to locate. Each decision function clearly documents what is being
// decided, the criteria used, and the expected outcomes.

package deployment

import (
	"scenario-dependency-analyzer/internal/config"
	types "scenario-dependency-analyzer/internal/types"
)

// ResourceClass categorizes resources by their operational characteristics.
// This classification drives decisions about tier fitness, secret requirements,
// and swap suggestions.
type ResourceClass string

const (
	ResourceClassDatabase   ResourceClass = "database"
	ResourceClassAI         ResourceClass = "ai"
	ResourceClassAutomation ResourceClass = "automation"
	ResourceClassStorage    ResourceClass = "storage"
	ResourceClassBrowser    ResourceClass = "browser"
	ResourceClassExecution  ResourceClass = "execution"
	ResourceClassService    ResourceClass = "service" // Default/unknown
)

// ResourceClassification holds the result of classifying a resource.
type ResourceClassification struct {
	Class      ResourceClass
	IsHeavyOps bool // Indicates resource-intensive operations
}

// ClassifyResource determines the operational category of a resource.
//
// Decision criteria:
//   - Database resources: postgres, mysql, mongodb, redis, qdrant
//   - AI resources: ollama (local LLM), claude-code, openai, anthropic
//   - Automation platforms: n8n, huginn, windmill
//   - Storage services: minio, s3
//   - Browser automation: browserless, playwright
//   - Code execution: judge0, sandbox
//
// Heavy operations are those requiring significant CPU/memory/GPU:
//   - Databases: postgres, mysql, mongodb (not redis/qdrant which are lighter)
//   - AI: ollama (local LLM inference)
//   - Automation: all (workflow engines are resource-intensive)
//   - Browser/Execution: all (sandboxed environments are heavy)
func ClassifyResource(resourceName string) ResourceClassification {
	normalized := config.NormalizeName(resourceName)

	switch normalized {
	case "postgres", "mysql", "mongodb", "redis", "qdrant":
		return ResourceClassification{
			Class:      ResourceClassDatabase,
			IsHeavyOps: isHeavyDatabaseResource(normalized),
		}

	case "ollama", "claude-code", "openai", "anthropic":
		return ResourceClassification{
			Class:      ResourceClassAI,
			IsHeavyOps: isLocalAIResource(normalized),
		}

	case "n8n", "huginn", "windmill":
		return ResourceClassification{
			Class:      ResourceClassAutomation,
			IsHeavyOps: true, // Automation platforms are always heavy
		}

	case "minio", "s3":
		return ResourceClassification{
			Class:      ResourceClassStorage,
			IsHeavyOps: false, // Storage is not compute-intensive
		}

	case "browserless", "playwright":
		return ResourceClassification{
			Class:      ResourceClassBrowser,
			IsHeavyOps: true, // Browser automation requires significant resources
		}

	case "judge0", "sandbox":
		return ResourceClassification{
			Class:      ResourceClassExecution,
			IsHeavyOps: true, // Code execution sandboxes are heavy
		}

	default:
		return ResourceClassification{
			Class:      ResourceClassService,
			IsHeavyOps: false,
		}
	}
}

// isHeavyDatabaseResource decides whether a database is resource-intensive.
// Full SQL databases (postgres, mysql, mongodb) are heavy; embedded/cache stores are not.
func isHeavyDatabaseResource(normalized string) bool {
	switch normalized {
	case "postgres", "mysql", "mongodb":
		return true
	default:
		return false
	}
}

// isLocalAIResource decides whether an AI resource runs locally (vs cloud API).
// Local LLM inference (ollama) is heavy; cloud APIs are lightweight clients.
func isLocalAIResource(normalized string) bool {
	return normalized == "ollama"
}

// TierFitnessDecision captures the result of evaluating a resource's fitness for a tier.
type TierFitnessDecision struct {
	Supported    bool
	FitnessScore float64
	Reason       string
	Alternatives []string
	Notes        string
}

// DecideTierFitness evaluates whether a resource is suitable for a deployment tier.
//
// Decision matrix:
//   - local: Everything works (development environment) - fitness 1.0
//   - desktop: Works but heavy ops reduce fitness - fitness 0.6-0.9
//   - server: Ideal for most resources - fitness 0.95
//   - mobile: Very restrictive, heavy ops blocked - fitness 0.0-0.4
//   - saas: DBs/storage good, heavy AI needs alternatives - fitness 0.3-0.95
//   - enterprise: Full capability - fitness 0.98
//
// Blocking criteria (fitness < TierBlockerThreshold):
//   - Mobile + local LLM = blocked (no local AI on mobile)
//   - Mobile + heavy database = blocked (use remote)
//   - Mobile + heavy ops = blocked
func DecideTierFitness(tier string, classification ResourceClassification) TierFitnessDecision {
	switch tier {
	case "local":
		return decideFitnessForLocal()

	case "desktop":
		return decideFitnessForDesktop(classification)

	case "server":
		return decideFitnessForServer()

	case "mobile":
		return decideFitnessForMobile(classification)

	case "saas":
		return decideFitnessForSaaS(classification)

	case "enterprise":
		return decideFitnessForEnterprise()

	default:
		// Unknown tier - assume moderate fitness
		return TierFitnessDecision{
			Supported:    true,
			FitnessScore: 0.7,
			Notes:        "Unknown tier - using default fitness",
		}
	}
}

// decideFitnessForLocal: Everything works in local development.
func decideFitnessForLocal() TierFitnessDecision {
	return TierFitnessDecision{
		Supported:    true,
		FitnessScore: 1.0,
	}
}

// decideFitnessForDesktop: Heavy ops reduce fitness but don't block.
func decideFitnessForDesktop(classification ResourceClassification) TierFitnessDecision {
	if classification.IsHeavyOps {
		return TierFitnessDecision{
			Supported:    true,
			FitnessScore: 0.6, // Can run but not ideal
		}
	}
	return TierFitnessDecision{
		Supported:    true,
		FitnessScore: 0.9,
	}
}

// decideFitnessForServer: Ideal environment for most resources.
func decideFitnessForServer() TierFitnessDecision {
	return TierFitnessDecision{
		Supported:    true,
		FitnessScore: 0.95,
	}
}

// decideFitnessForMobile: Very restrictive - blocks heavy resources.
func decideFitnessForMobile(classification ResourceClassification) TierFitnessDecision {
	// Local AI blocked on mobile
	if classification.Class == ResourceClassAI && classification.IsHeavyOps {
		return TierFitnessDecision{
			Supported:    false,
			FitnessScore: 0.0,
			Reason:       "Resource-intensive AI not supported on mobile",
		}
	}

	// Databases should be remote on mobile
	if classification.Class == ResourceClassDatabase || classification.Class == ResourceClassStorage {
		return TierFitnessDecision{
			Supported:    false,
			FitnessScore: 0.2,
			Reason:       "Database should be remote for mobile deployments",
			Alternatives: []string{"saas-variant", "cloud-variant"},
		}
	}

	// Other heavy ops blocked
	if classification.IsHeavyOps {
		return TierFitnessDecision{
			Supported:    false,
			FitnessScore: 0.1,
			Reason:       "Heavy operations not suitable for mobile",
		}
	}

	// Lightweight services might work
	return TierFitnessDecision{
		Supported:    true,
		FitnessScore: 0.4,
	}
}

// decideFitnessForSaaS: Good for managed services, heavy compute needs alternatives.
func decideFitnessForSaaS(classification ResourceClassification) TierFitnessDecision {
	// Databases and storage work well in SaaS
	if classification.Class == ResourceClassDatabase || classification.Class == ResourceClassStorage {
		return TierFitnessDecision{
			Supported:    true,
			FitnessScore: 0.95,
		}
	}

	// Local LLM should use cloud APIs for SaaS
	if classification.Class == ResourceClassAI && classification.IsHeavyOps {
		return TierFitnessDecision{
			Supported:    true,
			FitnessScore: 0.3,
			Alternatives: []string{"openai", "anthropic", "openrouter"},
			Notes:        "Consider using managed AI API for SaaS deployment",
		}
	}

	return TierFitnessDecision{
		Supported:    true,
		FitnessScore: 0.85,
	}
}

// decideFitnessForEnterprise: Full capability environment.
func decideFitnessForEnterprise() TierFitnessDecision {
	return TierFitnessDecision{
		Supported:    true,
		FitnessScore: 0.98,
	}
}

// IsTierBlocker decides whether a dependency should block a tier.
//
// Decision criteria:
//   - Explicitly marked as unsupported
//   - Fitness score below TierBlockerThreshold (0.75)
//
// A blocked tier means the deployment cannot proceed without
// resolving the blocking dependency (e.g., via swap or removal).
func IsTierBlocker(support types.TierSupportSummary) bool {
	// Explicitly unsupported
	if support.Supported != nil && !*support.Supported {
		return true
	}

	// Fitness below threshold
	if support.FitnessScore != nil && *support.FitnessScore < TierBlockerThreshold {
		return true
	}

	return false
}

// TierStatus represents the interpreted status of a deployment tier.
type TierStatus int

const (
	TierStatusUnknown TierStatus = iota
	TierStatusReady              // ready, supported
	TierStatusLimited            // limited, blocked
)

// InterpretTierStatus converts a status string to a typed status.
//
// Decision:
//   - "ready", "supported" -> TierStatusReady (fully supported)
//   - "limited", "blocked" -> TierStatusLimited (not fully supported)
//   - anything else -> TierStatusUnknown
func InterpretTierStatus(status string) TierStatus {
	switch status {
	case "ready", "supported":
		return TierStatusReady
	case "limited", "blocked":
		return TierStatusLimited
	default:
		return TierStatusUnknown
	}
}
