package detection

import "regexp"

// patterns.go - Centralized pattern definitions and heuristics
//
// This file contains all regex patterns and resource heuristics used for
// dependency detection, keeping them separate from scanning logic.

// Resource detection patterns
var (
	// resourceCommandPattern matches resource CLI commands like "resource-postgres"
	resourceCommandPattern = regexp.MustCompile(`resource-([a-z0-9-]+)`)

	// resourceHeuristicCatalog defines patterns for detecting resource usage
	// without explicit CLI commands (e.g., connection strings, env vars)
	resourceHeuristicCatalog = []resourceHeuristic{
		{
			Name: "postgres",
			Type: "postgres",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`postgres(ql)?:\/\/`),
				regexp.MustCompile(`PGHOST`),
			},
		},
		{
			Name: "redis",
			Type: "redis",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`redis:\/\/`),
				regexp.MustCompile(`REDIS_URL`),
			},
		},
		{
			Name: "ollama",
			Type: "ollama",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`ollama`),
			},
		},
		{
			Name: "qdrant",
			Type: "qdrant",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`qdrant`),
			},
		},
		{
			Name: "n8n",
			Type: "n8n",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`resource-?n8n`),
				regexp.MustCompile(`N8N_[A-Z0-9_]+`),
				regexp.MustCompile(`RESOURCE_PORT_N8N`),
				regexp.MustCompile(`n8n/(?:api|webhook)`),
			},
		},
		{
			Name: "minio",
			Type: "minio",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`minio`),
			},
		},
	}
)

// Scenario detection patterns
var (
	// Patterns for detecting scenario variable declarations/aliases
	scenarioAliasDeclPattern  = regexp.MustCompile(`(?m)(?:const|var)\s+([A-Za-z0-9_]+)\s*=\s*"([a-z0-9-]+)"`)
	scenarioAliasShortPattern = regexp.MustCompile(`(?m)([A-Za-z0-9_]+)\s*:=\s*"([a-z0-9-]+)"`)
	scenarioAliasBlockPattern = regexp.MustCompile(`(?m)^\s*([A-Za-z0-9_]+)\s*=\s*"([a-z0-9-]+)"`)

	// Pattern for detecting scenario port resolution calls
	scenarioPortCallPattern = regexp.MustCompile(`resolveScenarioPortViaCLI\s*\(\s*[^,]+,\s*(?:"([a-z0-9-]+)"|([A-Za-z0-9_]+))\s*,`)

	// Patterns for detecting scenario references in code
	vrooliScenarioPattern = regexp.MustCompile(`vrooli\s+scenario\s+(?:run|test|status)\s+([a-z0-9-]+)`)
	cliScenarioPattern    = regexp.MustCompile(`([a-z0-9-]+)-cli\.sh|\b([a-z0-9-]+)\s+(?:analyze|process|generate|run)`)

	// Pattern for detecting shared workflow references
	sharedWorkflowPattern = regexp.MustCompile(`initialization/(?:automation/)?(?:n8n|huginn|windmill)/([^/]+\.json)`)
)

// File extension filters for different scan types
var (
	// resourceDetectionExtensions defines which file types to scan for resource usage
	resourceDetectionExtensions = []string{".go", ".js", ".ts", ".tsx", ".sh", ".py", ".md", ".json", ".yml", ".yaml"}

	// scenarioDetectionExtensions defines which file types to scan for scenario references
	scenarioDetectionExtensions = []string{".go", ".js", ".sh", ".py", ".md"}
)

// resourceHeuristic defines a heuristic for detecting a resource without explicit CLI usage
type resourceHeuristic struct {
	Name     string            // Canonical resource name
	Type     string            // Resource type identifier
	Patterns []*regexp.Regexp  // Patterns that indicate usage of this resource
}
