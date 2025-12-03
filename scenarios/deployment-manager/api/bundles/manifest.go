// Package bundles provides bundle validation, assembly, and secret merging.
package bundles

// Manifest mirrors the v0.1 desktop bundle schema enough for
// lightweight validation without pulling in external schema validators.
type Manifest struct {
	SchemaVersion string            `json:"schema_version"`
	Target        string            `json:"target"`
	App           ManifestApp       `json:"app"`
	IPC           ManifestIPC       `json:"ipc"`
	Telemetry     ManifestTelemetry `json:"telemetry"`
	Ports         *ManifestPorts    `json:"ports,omitempty"`
	Swaps         []ManifestSwap    `json:"swaps,omitempty"`
	Secrets       []ManifestSecret  `json:"secrets,omitempty"`
	Services      []ServiceEntry    `json:"services"`
}

// ManifestApp holds application metadata.
type ManifestApp struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// ManifestIPC holds IPC configuration.
type ManifestIPC struct {
	Mode          string `json:"mode"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	AuthTokenPath string `json:"auth_token_path"`
}

// ManifestTelemetry holds telemetry configuration.
type ManifestTelemetry struct {
	File      string `json:"file"`
	UploadURL string `json:"upload_url"`
}

// ManifestPorts holds port allocation configuration.
type ManifestPorts struct {
	DefaultRange PortRange `json:"default_range"`
	Reserved     []int     `json:"reserved"`
}

// PortRange defines a port range.
type PortRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// ManifestSwap defines a resource swap.
type ManifestSwap struct {
	Original    string `json:"original"`
	Replacement string `json:"replacement"`
	Reason      string `json:"reason"`
	Limitations string `json:"limitations"`
}

// ManifestSecret defines a secret in the manifest.
type ManifestSecret struct {
	ID          string                 `json:"id"`
	Class       string                 `json:"class"`
	Description string                 `json:"description"`
	Format      string                 `json:"format"`
	Required    *bool                  `json:"required"`
	Prompt      *SecretPrompt          `json:"prompt,omitempty"`
	Generator   map[string]interface{} `json:"generator,omitempty"`
	Target      SecretTarget           `json:"target"`
}

// SecretPrompt defines user prompt for a secret.
type SecretPrompt struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// SecretTarget defines how a secret is injected.
type SecretTarget struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// ServiceEntry defines a service in the bundle.
type ServiceEntry struct {
	ID           string                   `json:"id"`
	Type         string                   `json:"type"`
	Description  string                   `json:"description"`
	Binaries     map[string]ServiceBinary `json:"binaries"`
	Env          map[string]string        `json:"env,omitempty"`
	Secrets      []string                 `json:"secrets,omitempty"`
	DataDirs     []string                 `json:"data_dirs,omitempty"`
	LogDir       string                   `json:"log_dir,omitempty"`
	Ports        *ServicePorts            `json:"ports,omitempty"`
	Health       HealthCheck              `json:"health"`
	Readiness    ReadinessCheck           `json:"readiness"`
	Dependencies []string                 `json:"dependencies,omitempty"`
	Migrations   []Migration              `json:"migrations,omitempty"`
	Assets       []Asset                  `json:"assets,omitempty"`
	GPU          *GPUConfig               `json:"gpu,omitempty"`
	Critical     *bool                    `json:"critical,omitempty"`
}

// ServiceBinary defines a platform-specific binary.
type ServiceBinary struct {
	Path string            `json:"path"`
	Args []string          `json:"args,omitempty"`
	Env  map[string]string `json:"env,omitempty"`
	Cwd  string            `json:"cwd,omitempty"`
}

// ServicePorts defines service port configuration.
type ServicePorts struct {
	Requested []RequestedPort `json:"requested"`
}

// RequestedPort defines a requested port.
type RequestedPort struct {
	Name           string    `json:"name"`
	Range          PortRange `json:"range"`
	RequiresSocket bool      `json:"requires_socket"`
}

// HealthCheck defines health check configuration.
type HealthCheck struct {
	Type     string   `json:"type"`
	Path     string   `json:"path"`
	PortName string   `json:"port_name"`
	Command  []string `json:"command,omitempty"`
	Interval int      `json:"interval_ms"`
	Timeout  int      `json:"timeout_ms"`
	Retries  int      `json:"retries"`
}

// ReadinessCheck defines readiness check configuration.
type ReadinessCheck struct {
	Type     string `json:"type"`
	PortName string `json:"port_name"`
	Pattern  string `json:"pattern"`
	Timeout  int    `json:"timeout_ms"`
}

// Migration defines a database migration.
type Migration struct {
	Version string            `json:"version"`
	Command []string          `json:"command"`
	Env     map[string]string `json:"env,omitempty"`
	RunOn   string            `json:"run_on"`
}

// Asset defines a bundled asset.
type Asset struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

// GPUConfig defines GPU requirements.
type GPUConfig struct {
	Requirement string `json:"requirement"`
}
