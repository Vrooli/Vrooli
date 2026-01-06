// Package domain defines the core domain types for the scenario-to-cloud scenario.
package domain

// DefaultVPSWorkdir is the default directory where Vrooli is installed on the VPS.
// This is the single source of truth for this value - do not hardcode "/root/Vrooli" elsewhere.
const DefaultVPSWorkdir = "/root/Vrooli"

// CloudManifest is the core domain type representing a deployment configuration.
// It describes what scenario to deploy, where to deploy it, and how.
type CloudManifest struct {
	Version      string               `json:"version"`
	Environment  string               `json:"environment,omitempty"`
	Target       ManifestTarget       `json:"target"`
	Scenario     ManifestScenario     `json:"scenario"`
	Dependencies ManifestDependencies `json:"dependencies"`
	Bundle       ManifestBundle       `json:"bundle"`
	Ports        ManifestPorts        `json:"ports"`
	Edge         ManifestEdge         `json:"edge"`
	Secrets      *ManifestSecrets     `json:"secrets,omitempty"`
}

// ManifestTarget specifies where to deploy.
type ManifestTarget struct {
	Type string       `json:"type"`
	VPS  *ManifestVPS `json:"vps,omitempty"`
}

// ManifestVPS contains VPS-specific deployment target configuration.
type ManifestVPS struct {
	Host    string `json:"host"`
	Port    int    `json:"port,omitempty"`
	User    string `json:"user,omitempty"`
	KeyPath string `json:"key_path,omitempty"`
	Workdir string `json:"workdir,omitempty"`
}

// ManifestScenario identifies the scenario to deploy.
type ManifestScenario struct {
	ID  string `json:"id"`
	Ref string `json:"ref,omitempty"`
}

// ManifestDependencies captures the dependency snapshot from scenario-dependency-analyzer.
type ManifestDependencies struct {
	Scenarios []string `json:"scenarios,omitempty"`
	Resources []string `json:"resources,omitempty"`
	Analyzer  struct {
		Tool        string `json:"tool,omitempty"`
		Fingerprint string `json:"fingerprint,omitempty"`
		GeneratedAt string `json:"generated_at,omitempty"`
	} `json:"analyzer,omitempty"`
}

// ManifestBundle specifies what to include in the deployment bundle.
type ManifestBundle struct {
	IncludePackages bool     `json:"include_packages"`
	IncludeAutoheal bool     `json:"include_autoheal"`
	Scenarios       []string `json:"scenarios,omitempty"`
	Resources       []string `json:"resources,omitempty"`
}

// ManifestPorts maps port names to port numbers.
// Standard ports (ui, api, ws) are common, but scenarios can define additional ports
// like playwright_driver, metrics, etc. in their service.json.
type ManifestPorts map[string]int

// ManifestEdge configures edge/TLS settings.
type ManifestEdge struct {
	Domain    string        `json:"domain"`
	DNSPolicy DNSPolicy     `json:"dns_policy,omitempty"`
	Caddy     ManifestCaddy `json:"caddy"`
}

// ManifestCaddy configures Caddy reverse proxy and TLS.
type ManifestCaddy struct {
	Enabled bool   `json:"enabled"`
	Email   string `json:"email,omitempty"`
}

// ManifestSecrets holds the secrets configuration for deployment.
type ManifestSecrets struct {
	BundleSecrets []BundleSecretPlan `json:"bundle_secrets,omitempty"`
	Summary       SecretsSummary     `json:"summary"`
}

// BundleSecretPlan represents a secret from secrets-manager.
// This is included in the manifest so the deploy target can generate/prompt for secrets.
type BundleSecretPlan struct {
	ID          string                 `json:"id"`
	Class       string                 `json:"class"` // per_install_generated, user_prompt, remote_fetch, infrastructure
	Required    bool                   `json:"required"`
	Description string                 `json:"description,omitempty"`
	Format      string                 `json:"format,omitempty"` // validation pattern
	Target      BundleSecretTarget     `json:"target"`
	Prompt      *SecretPromptMetadata  `json:"prompt,omitempty"`
	Generator   map[string]interface{} `json:"generator,omitempty"`
}

// BundleSecretTarget specifies where to inject the secret value.
type BundleSecretTarget struct {
	Type string `json:"type"` // "env" or "file"
	Name string `json:"name"` // env var name or file path
}

// SecretPromptMetadata contains UI hints for user_prompt secrets.
type SecretPromptMetadata struct {
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
}

// SecretsSummary provides a quick overview of secret requirements.
type SecretsSummary struct {
	TotalSecrets        int `json:"total_secrets"`
	PerInstallGenerated int `json:"per_install_generated"`
	UserPrompt          int `json:"user_prompt"`
	RemoteFetch         int `json:"remote_fetch"`
	Infrastructure      int `json:"infrastructure"`
}

// ValidationIssueSeverity indicates how serious a validation issue is.
type ValidationIssueSeverity string

const (
	SeverityError ValidationIssueSeverity = "error"
	SeverityWarn  ValidationIssueSeverity = "warn"
)

// ValidationIssue represents a single validation problem.
type ValidationIssue struct {
	Path     string                  `json:"path"`
	Message  string                  `json:"message"`
	Hint     string                  `json:"hint,omitempty"`
	Severity ValidationIssueSeverity `json:"severity"`
}

// ManifestValidateResponse is the API response for manifest validation.
type ManifestValidateResponse struct {
	Valid      bool              `json:"valid"`
	Issues     []ValidationIssue `json:"issues,omitempty"`
	Manifest   CloudManifest     `json:"manifest"`
	Timestamp  string            `json:"timestamp"`
	SchemaHint string            `json:"schema_hint,omitempty"`
}
