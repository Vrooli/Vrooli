package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
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

type PortRequest struct {
	Name           string    `json:"name"`
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
	if len(m.Services) == 0 {
		return errors.New("services must not be empty")
	}

	for _, svc := range m.Services {
		if svc.ID == "" {
			return errors.New("service.id is required")
		}
		if svc.Health.Type == "" || svc.Readiness.Type == "" {
			return fmt.Errorf("service %s requires health and readiness definitions", svc.ID)
		}
		if len(svc.Binaries) == 0 {
			return fmt.Errorf("service %s missing binaries", svc.ID)
		}
		keys := PlatformKeys(targetOS, targetArch)
		found := false
		for _, key := range keys {
			if bin, ok := svc.Binaries[key]; ok && bin.Path != "" {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("service %s missing binary for platform %s", svc.ID, keys[0])
		}
	}

	return nil
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
