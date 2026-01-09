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
				regexp.MustCompile(`PGHOST|POSTGRES_HOST|POSTGRES_URL`),
			},
		},
		{
			Name: "redis",
			Type: "redis",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`redis:\/\/`),
				regexp.MustCompile(`REDIS_URL|REDIS_HOST`),
			},
		},
		{
			Name: "ollama",
			Type: "ollama",
			Patterns: []*regexp.Regexp{
				// More specific patterns - require context indicating actual usage
				regexp.MustCompile(`OLLAMA_(?:HOST|URL|API|BASE_URL)`),
				regexp.MustCompile(`ollama(?:\.|\s+)(?:generate|embeddings|list|pull|push)`),
				regexp.MustCompile(`http://[^"'\s]*ollama[^"'\s]*:\d+`),
			},
		},
		{
			Name: "qdrant",
			Type: "qdrant",
			Patterns: []*regexp.Regexp{
				// More specific - require context
				regexp.MustCompile(`QDRANT_(?:HOST|URL|API_KEY)`),
				regexp.MustCompile(`qdrant(?:\.|\s+)(?:search|upsert|delete|create_collection)`),
				regexp.MustCompile(`http://[^"'\s]*qdrant[^"'\s]*:\d+`),
			},
		},
		{
			Name: "n8n",
			Type: "n8n",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`resource-n8n`), // Exact match only
				regexp.MustCompile(`N8N_(?:HOST|URL|API_KEY|WEBHOOK)`),
				regexp.MustCompile(`RESOURCE_PORT_N8N`),
				regexp.MustCompile(`http://[^"'\s]*n8n[^"'\s]*/(?:api|webhook)`),
			},
		},
		{
			Name: "minio",
			Type: "minio",
			Patterns: []*regexp.Regexp{
				// More specific - require context
				regexp.MustCompile(`MINIO_(?:ENDPOINT|ACCESS_KEY|SECRET_KEY|BUCKET)`),
				regexp.MustCompile(`minio\.(?:Client|BucketExists|GetObject|PutObject)`),
				regexp.MustCompile(`http://[^"'\s]*minio[^"'\s]*:\d+`),
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
	vrooliScenarioPattern = regexp.MustCompile(`vrooli\s+scenario\s+(?:run|test|status|start|stop)\s+([a-z0-9-]+)`)
	// More specific CLI pattern - require explicit CLI script reference
	cliScenarioPattern = regexp.MustCompile(`([a-z0-9-]+)-(?:cli|api)(?:\.sh)?`)

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
	Name     string           // Canonical resource name
	Type     string           // Resource type identifier
	Patterns []*regexp.Regexp // Patterns that indicate usage of this resource
}
