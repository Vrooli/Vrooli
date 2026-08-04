package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Manifest represents bundle.json (desktop v0.1).
type Manifest struct {
	SchemaVersion string     `json:"schema_version"`
	Target        string     `json:"target"`
	App           App        `json:"app"`
	IPC           IPC        `json:"ipc"`
	Telemetry     Telemetry  `json:"telemetry"`
	Ports         *PortRules `json:"ports,omitempty"`
	Swaps         []Swap     `json:"swaps,omitempty"`
	Secrets       []Secret   `json:"secrets,omitempty"`
	Services      []Service  `json:"services"`
}

// InvalidIPCHostError identifies a manifest that would expose the authenticated
// control API beyond the local machine.
type InvalidIPCHostError struct{ Host string }

func (e InvalidIPCHostError) Error() string {
	return fmt.Sprintf("ipc.host %q is not a loopback address", e.Host)
}

type App struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

type IPC struct {
	Mode         string `json:"mode"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	AuthTokenRel string `json:"auth_token_path"`
}

type Telemetry struct {
	File     string `json:"file"`
	UploadTo string `json:"upload_url,omitempty"`
}

type PortRules struct {
	DefaultRange *PortRange `json:"default_range,omitempty"`
	Reserved     []int      `json:"reserved,omitempty"`
}

type PortRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type Swap struct {
	Original    string `json:"original"`
	Replacement string `json:"replacement"`
	Reason      string `json:"reason,omitempty"`
	Limitations string `json:"limitations,omitempty"`
}

type Secret struct {
	ID          string            `json:"id"`
	Class       string            `json:"class"`
	Description string            `json:"description,omitempty"`
	Format      string            `json:"format,omitempty"`
	Prompt      map[string]string `json:"prompt,omitempty"`
	Generator   map[string]any    `json:"generator,omitempty"`
	Required    *bool             `json:"required,omitempty"`
	Target      SecretTarget      `json:"target"`

	// LogicalID and Field are the durable, backend-neutral name this secret
	// resolves to, exactly as a resource or scenario manifest declares it.
	//
	// They exist so a bundle reads the credential the operator already
	// provisioned. Without them a bundle invented its own namespace from the
	// app's display name, so the OpenRouter key entered during onboarding and
	// the OpenRouter key a packaged bundle looked for were two different stored
	// values with no declared relationship — provision-once was not true across
	// tiers, and neither was a recovery bundle taken on one of them.
	//
	// Both are optional in the file: LogicalIdentity falls back to the bundle's
	// own namespace so an existing manifest still resolves somewhere, and the
	// generator fills them in from the scenario's declaration.
	LogicalID string `json:"logical_id,omitempty"`
	Field     string `json:"field,omitempty"`
}

// CredentialField is the durable field this secret addresses. It matches the
// normalization every other tier uses, so SESSION_SECRET, session_secret, and
// session.secret name one stored value rather than three empty ones.
func (s Secret) CredentialField() string {
	raw := strings.TrimSpace(s.Field)
	if raw == "" {
		raw = strings.TrimSpace(s.ID)
	}
	if raw == "" {
		raw = strings.TrimSpace(s.Target.Name)
	}
	if raw == "" {
		return ""
	}
	return strings.ToLower(strings.NewReplacer("_", "-", ".", "-").Replace(raw))
}

type SecretTarget struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type Service struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Description  string                 `json:"description,omitempty"`
	Binaries     map[string]Binary      `json:"binaries"`
	Build        *BuildConfig           `json:"build,omitempty"`
	Env          map[string]string      `json:"env,omitempty"`
	Secrets      []string               `json:"secrets,omitempty"`
	DataDirs     []string               `json:"data_dirs,omitempty"`
	LogDir       string                 `json:"log_dir,omitempty"`
	Ports        *ServicePorts          `json:"ports,omitempty"`
	Health       HealthCheck            `json:"health"`
	Readiness    ReadinessCheck         `json:"readiness"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Migrations   []Migration            `json:"migrations,omitempty"`
	Assets       []Asset                `json:"assets,omitempty"`
	GPU          *GPURequirements       `json:"gpu,omitempty"`
	Critical     *bool                  `json:"critical,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	// DistRoot is the directory to serve static files from for ui-bundle services.
	// If not specified, the runtime will automatically detect it by finding index.html
	// in the assets list and using its parent directory.
	DistRoot string `json:"dist_root,omitempty"`
}

// BuildConfig specifies how to compile a service binary when not pre-built.
type BuildConfig struct {
	// Type is the build system: "go", "rust", "npm", "python", or "custom"
	Type string `json:"type"`
	// SourceDir is the relative path to the source code directory
	SourceDir string `json:"source_dir"`
	// EntryPoint is the build entry point (e.g., "./cmd/api" for Go, "src/main.rs" for Rust)
	EntryPoint string `json:"entry_point,omitempty"`
	// OutputPattern is the output path pattern with {{platform}} and {{ext}} placeholders
	// e.g., "bin/{{platform}}/api{{ext}}" -> "bin/linux-x64/api" or "bin/win-x64/api.exe"
	OutputPattern string `json:"output_pattern,omitempty"`
	// Args are additional build arguments
	Args []string `json:"args,omitempty"`
	// Env are additional environment variables for the build
	Env map[string]string `json:"env,omitempty"`
}

type GPURequirements struct {
	Requirement string `json:"requirement,omitempty"`
}

func (s Service) GPURequirement() string {
	if s.GPU == nil {
		return ""
	}
	return strings.TrimSpace(s.GPU.Requirement)
}

type Binary struct {
	Path string            `json:"path"`
	Args []string          `json:"args,omitempty"`
	Env  map[string]string `json:"env,omitempty"`
	CWD  string            `json:"cwd,omitempty"`
}

type ServicePorts struct {
	Requested []PortRequest `json:"requested,omitempty"`
}

// PortRequest defines a requested port with optional environment variable binding.
// DOC: docs/internal/SEAMS.md#port-environment-seam-feb-2026
type PortRequest struct {
	Name           string    `json:"name"`
	EnvVar         string    `json:"env_var,omitempty"` // Environment variable name (e.g., "API_PORT")
	Range          PortRange `json:"range"`
	RequiresSocket bool      `json:"requires_socket,omitempty"`
}

type HealthCheck struct {
	Type       string   `json:"type"`
	Path       string   `json:"path,omitempty"`
	PortName   string   `json:"port_name,omitempty"`
	Command    []string `json:"command,omitempty"`
	IntervalMs int      `json:"interval_ms,omitempty"`
	TimeoutMs  int      `json:"timeout_ms,omitempty"`
	Retries    int      `json:"retries,omitempty"`
}

type ReadinessCheck struct {
	Type      string `json:"type"`
	PortName  string `json:"port_name,omitempty"`
	Pattern   string `json:"pattern,omitempty"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
}

type Migration struct {
	Version string            `json:"version"`
	Command []string          `json:"command"`
	Env     map[string]string `json:"env,omitempty"`
	RunOn   string            `json:"run_on,omitempty"`
}

type Asset struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
}

// LoadManifest reads and parses bundle.json into a Manifest.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	return &m, nil
}

// Validate performs lightweight structural validation for the current OS/arch.
func (m *Manifest) Validate(targetOS, targetArch string) error {
	if err := m.validateHeader(); err != nil {
		return err
	}
	if len(m.Services) == 0 {
		return errors.New("services must not be empty")
	}
	keys := PlatformKeys(targetOS, targetArch)
	for _, svc := range m.Services {
		if err := validateService(svc, keys); err != nil {
			return err
		}
	}
	return nil
}

// validateHeader checks top-level manifest fields.
func (m *Manifest) validateHeader() error {
	if m.SchemaVersion == "" {
		return errors.New("schema_version missing")
	}
	if m.Target != "desktop" {
		return fmt.Errorf("unexpected target %q (expected desktop)", m.Target)
	}
	if m.App.Name == "" || m.App.Version == "" {
		return errors.New("app.name and app.version are required")
	}
	if m.IPC.Host == "" || m.IPC.Port == 0 {
		return errors.New("ipc.host and ipc.port are required")
	}
	if !isLoopbackHost(m.IPC.Host) {
		return InvalidIPCHostError{Host: m.IPC.Host}
	}
	return nil
}

func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(host)
	if strings.EqualFold(host, "localhost") {
		return true
	}
	return net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback()
}

// validateService checks a single service definition.
func validateService(svc Service, platformKeys []string) error {
	if svc.ID == "" {
		return errors.New("service.id is required")
	}
	if svc.Health.Type == "" || svc.Readiness.Type == "" {
		return fmt.Errorf("service %s requires health and readiness definitions", svc.ID)
	}
	if svc.Health.Type == "http" && strings.TrimSpace(svc.Health.PortName) == "" {
		return fmt.Errorf("service %s health port_name is required for http health", svc.ID)
	}
	if len(svc.Binaries) == 0 {
		return fmt.Errorf("service %s missing binaries", svc.ID)
	}
	for _, key := range platformKeys {
		if bin, ok := svc.Binaries[key]; ok && bin.Path != "" {
			return nil
		}
	}
	return fmt.Errorf("service %s missing binary for platform %s", svc.ID, platformKeys[0])
}

// PlatformKey converts GOOS/GOARCH into the manifest map key (e.g., linux-x64).
func PlatformKey(goos, goarch string) string {
	arch := goarch
	if goarch == "amd64" {
		arch = "x64"
	}
	return fmt.Sprintf("%s-%s", goos, arch)
}

// PlatformKeys returns the canonical platform key plus any aliases (win/windows, mac/darwin).
func PlatformKeys(goos, goarch string) []string {
	primary := PlatformKey(goos, goarch)
	keys := []string{primary}
	if alias := platformAlias(primary); alias != "" && alias != primary {
		keys = append(keys, alias)
	}
	return keys
}

func platformAlias(key string) string {
	if strings.HasPrefix(key, "windows-") {
		return "win-" + strings.TrimPrefix(key, "windows-")
	}
	if strings.HasPrefix(key, "win-") {
		return "windows-" + strings.TrimPrefix(key, "win-")
	}
	if strings.HasPrefix(key, "darwin-") {
		return "mac-" + strings.TrimPrefix(key, "darwin-")
	}
	if strings.HasPrefix(key, "mac-") {
		return "darwin-" + strings.TrimPrefix(key, "mac-")
	}
	return ""
}

// ResolveBinary returns the binary config for the current platform.
func (m *Manifest) ResolveBinary(svc Service) (Binary, bool) {
	keys := PlatformKeys(runtime.GOOS, runtime.GOARCH)
	for _, key := range keys {
		if bin, ok := svc.Binaries[key]; ok {
			return bin, true
		}
	}
	return Binary{}, false
}

// ResolvePath resolves a bundle-relative path to an absolute path rooted at bundleDir.
func ResolvePath(bundleDir, rel string) string {
	// Keep manifest Windows paths usable across OSes.
	clean := filepath.Clean(strings.ReplaceAll(rel, "\\", string(filepath.Separator)))
	return filepath.Join(bundleDir, clean)
}
