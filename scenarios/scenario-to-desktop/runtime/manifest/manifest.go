package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Manifest represents bundle.json (desktop v0.1).
type Manifest struct {
	SchemaVersion  string                 `json:"schema_version"`
	Target         string                 `json:"target"`
	App            App                    `json:"app"`
	IPC            IPC                    `json:"ipc"`
	Telemetry      Telemetry              `json:"telemetry"`
	Authentication *AuthenticationProfile `json:"authentication,omitempty"`
	Ports          *PortRules             `json:"ports,omitempty"`
	Swaps          []Swap                 `json:"swaps,omitempty"`
	Peers          []Peer                 `json:"peers,omitempty"`
	Secrets        []Secret               `json:"secrets,omitempty"`
	Services       []Service              `json:"services"`
	// CatalogRequirements are immutable catalog paths the bundled application
	// needs in order to boot every declared capability. Packaging validation
	// fails before launch when one is absent.
	CatalogRequirements []string `json:"catalog_requirements,omitempty"`
}

// AuthenticationProfile declares the bundle's explicit identity mode. It is
// non-secret metadata; credentials and leases remain in the native store.
type AuthenticationProfile struct {
	Version int    `json:"version"`
	Mode    string `json:"mode"`
	// ModeProfiles declares the non-secret configuration for modes that an
	// operator may explicitly select after installation. The legacy fields
	// above remain the selected mode's profile for backwards compatibility.
	ModeProfiles          map[string]AuthenticationModeProfile `json:"mode_profiles,omitempty"`
	Provider              string                               `json:"provider,omitempty"`
	Resource              string                               `json:"resource,omitempty"`
	Audience              string                               `json:"audience,omitempty"`
	ProviderEndpoint      string                               `json:"provider_endpoint,omitempty"`
	ProviderServiceID     string                               `json:"provider_service_id,omitempty"`
	HumanSignIn           string                               `json:"human_sign_in"`
	Offline               bool                                 `json:"offline"`
	PublicRoutes          []string                             `json:"public_routes,omitempty"`
	ProtectedRoutes       []string                             `json:"protected_routes,omitempty"`
	LeasePath             string                               `json:"lease_path,omitempty"`
	RecoveryURL           string                               `json:"recovery_url,omitempty"`
	RequiresAuthenticator bool                                 `json:"requires_authenticator"`
}

// AuthenticationModeProfile is the non-secret configuration for one optional
// authentication mode. Credentials, refresh material, and leases are never
// represented here.
type AuthenticationModeProfile struct {
	Provider              string   `json:"provider,omitempty"`
	Resource              string   `json:"resource,omitempty"`
	Audience              string   `json:"audience,omitempty"`
	ProviderEndpoint      string   `json:"provider_endpoint,omitempty"`
	ProviderServiceID     string   `json:"provider_service_id,omitempty"`
	HumanSignIn           string   `json:"human_sign_in"`
	Offline               bool     `json:"offline"`
	PublicRoutes          []string `json:"public_routes,omitempty"`
	ProtectedRoutes       []string `json:"protected_routes,omitempty"`
	LeasePath             string   `json:"lease_path,omitempty"`
	RecoveryURL           string   `json:"recovery_url,omitempty"`
	RequiresAuthenticator bool     `json:"requires_authenticator"`
}

// ProfileForMode returns the selected profile or an explicitly declared
// alternate profile. An absent alternate is intentionally not inferred: mode
// changes must be declared by the bundle author.
func (p *AuthenticationProfile) ProfileForMode(mode string) (*AuthenticationProfile, bool) {
	if p == nil {
		return nil, false
	}
	if mode == p.Mode {
		copy := *p
		return &copy, true
	}
	config, ok := p.ModeProfiles[mode]
	if !ok {
		return nil, false
	}
	return &AuthenticationProfile{
		Version:               p.Version,
		Mode:                  mode,
		Provider:              config.Provider,
		Resource:              config.Resource,
		Audience:              config.Audience,
		ProviderEndpoint:      config.ProviderEndpoint,
		ProviderServiceID:     config.ProviderServiceID,
		HumanSignIn:           config.HumanSignIn,
		Offline:               config.Offline,
		PublicRoutes:          append([]string(nil), config.PublicRoutes...),
		ProtectedRoutes:       append([]string(nil), config.ProtectedRoutes...),
		LeasePath:             config.LeasePath,
		RecoveryURL:           config.RecoveryURL,
		RequiresAuthenticator: config.RequiresAuthenticator,
		ModeProfiles:          p.ModeProfiles,
	}, true
}

// DeclaredModes returns all modes explicitly available to the installation.
func (p *AuthenticationProfile) DeclaredModes() []string {
	if p == nil {
		return nil
	}
	seen := map[string]struct{}{}
	modes := make([]string, 0, len(p.ModeProfiles)+1)
	if mode := strings.TrimSpace(p.Mode); mode != "" {
		seen[mode] = struct{}{}
		modes = append(modes, mode)
	}
	for mode := range p.ModeProfiles {
		if _, ok := seen[mode]; ok {
			continue
		}
		seen[mode] = struct{}{}
		modes = append(modes, mode)
	}
	sort.Strings(modes)
	return modes
}

// InvalidIPCHostError identifies a manifest that would expose the authenticated
// control API beyond the local machine.
type InvalidIPCHostError struct{ Host string }

func (e InvalidIPCHostError) Error() string {
	return fmt.Sprintf("ipc.host %q is not a loopback address", e.Host)
}

// MissingPeerDiscoveryPathError identifies a discover edge that cannot
// produce any runtime binding and would otherwise be silently discarded.
type MissingPeerDiscoveryPathError struct{ Scenario string }

func (e MissingPeerDiscoveryPathError) Error() string {
	return fmt.Sprintf("peer %q declares bundle_policy discover but has no discovery bindings", e.Scenario)
}

type MissingEmbeddedPeerError struct{ Scenario string }

func (e MissingEmbeddedPeerError) Error() string {
	return fmt.Sprintf("peer %q declares bundle_policy embed but contributes no bundled services", e.Scenario)
}

type App struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	Scenario    string `json:"scenario,omitempty"`
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

type Peer struct {
	Scenario         string        `json:"scenario"`
	BundlePolicy     string        `json:"bundle_policy"`
	StartupPolicy    string        `json:"startup_policy,omitempty"`
	DegradedBehavior string        `json:"degraded_behavior,omitempty"`
	Bindings         []PeerBinding `json:"bindings"`
}

type PeerBinding struct {
	EnvVar          string `json:"env_var"`
	Form            string `json:"form"`
	Port            string `json:"port"`
	WhenUnavailable string `json:"when_unavailable"`
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
	ID           string            `json:"id"`
	Type         string            `json:"type"`
	Description  string            `json:"description,omitempty"`
	Binaries     map[string]Binary `json:"binaries"`
	Build        *BuildConfig      `json:"build,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	Secrets      []string          `json:"secrets,omitempty"`
	DataDirs     []string          `json:"data_dirs,omitempty"`
	LogDir       string            `json:"log_dir,omitempty"`
	Ports        *ServicePorts     `json:"ports,omitempty"`
	Health       HealthCheck       `json:"health"`
	Readiness    ReadinessCheck    `json:"readiness"`
	Dependencies []string          `json:"dependencies,omitempty"`
	Migrations   []Migration       `json:"migrations,omitempty"`
	Assets       []Asset           `json:"assets,omitempty"`
	// AssetDirs contains runtime directories that must be copied and preserved
	// as part of a service bundle (for example, a Node service's dependencies).
	AssetDirs []string               `json:"asset_dirs,omitempty"`
	GPU       *GPURequirements       `json:"gpu,omitempty"`
	Critical  *bool                  `json:"critical,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
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
	if err := validateAuthentication(m.Authentication); err != nil {
		return err
	}
	if m.Authentication != nil {
		for mode := range m.Authentication.ModeProfiles {
			profile, ok := m.Authentication.ProfileForMode(mode)
			if !ok {
				return fmt.Errorf("authentication.mode_profiles[%q] is not readable", mode)
			}
			if err := validateAuthentication(profile); err != nil {
				return fmt.Errorf("authentication.mode_profiles[%q]: %w", mode, err)
			}
		}
	}
	if m.Authentication != nil && m.Authentication.RequiresAuthenticator {
		serviceID := strings.TrimSpace(m.Authentication.ProviderServiceID)
		if serviceID == "" {
			return errors.New("authentication.provider_service_id is required when authenticator startup is required")
		}
		found := false
		for _, service := range m.Services {
			if service.ID == serviceID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("authentication provider service %q is not bundled", serviceID)
		}
	}
	if m.Authentication != nil {
		for mode := range m.Authentication.ModeProfiles {
			profile, _ := m.Authentication.ProfileForMode(mode)
			if !profile.RequiresAuthenticator {
				continue
			}
			serviceID := strings.TrimSpace(profile.ProviderServiceID)
			if serviceID == "" {
				return fmt.Errorf("authentication.mode_profiles[%q].provider_service_id is required when authenticator startup is required", mode)
			}
			found := false
			for _, service := range m.Services {
				if service.ID == serviceID {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("authentication mode %q provider service %q is not bundled", mode, serviceID)
			}
		}
	}
	if err := validatePeers(m.Peers, m.Services); err != nil {
		return err
	}
	keys := PlatformKeys(targetOS, targetArch)
	for _, svc := range m.Services {
		if err := validateService(svc, keys); err != nil {
			return err
		}
	}
	return nil
}

func validateAuthentication(profile *AuthenticationProfile) error {
	if profile == nil {
		return nil
	}
	if profile.Version != 1 {
		return fmt.Errorf("authentication.version must be 1")
	}
	switch profile.Mode {
	case "personal_local":
		if profile.Provider != "" || !profile.Offline || profile.RequiresAuthenticator {
			return errors.New("personal_local authentication must be offline, provider-free, and authenticator-free")
		}
		if profile.HumanSignIn != "disabled" && profile.HumanSignIn != "optional" {
			return errors.New("personal_local authentication must disable or make human sign-in optional")
		}
	case "local_multi_user":
		if profile.Provider != "scenario-authenticator" || profile.HumanSignIn != "required" || !profile.RequiresAuthenticator || strings.TrimSpace(profile.ProviderServiceID) == "" {
			return errors.New("local_multi_user authentication requires a bundled scenario-authenticator and human sign-in")
		}
	case "remote_vrooli":
		if profile.Provider != "scenario-authenticator" || profile.HumanSignIn != "required" || strings.TrimSpace(profile.ProviderEndpoint) == "" {
			return errors.New("remote_vrooli authentication requires scenario-authenticator, an endpoint, and human sign-in")
		}
		if err := validateProviderEndpoint(profile.ProviderEndpoint); err != nil {
			return fmt.Errorf("remote_vrooli authentication provider endpoint: %w", err)
		}
	case "shared_provider":
		if profile.Provider == "" || profile.HumanSignIn != "required" || strings.TrimSpace(profile.ProviderEndpoint) == "" || strings.TrimSpace(profile.LeasePath) == "" {
			return errors.New("shared_provider authentication requires a provider, endpoint, lease path, and human sign-in")
		}
		if err := validateProviderEndpoint(profile.ProviderEndpoint); err != nil {
			return fmt.Errorf("shared_provider authentication provider endpoint: %w", err)
		}
	default:
		return fmt.Errorf("authentication.mode %q is unsupported", profile.Mode)
	}
	if profile.Mode != "personal_local" && strings.TrimSpace(profile.Resource) == "" {
		return errors.New("authentication.resource is required for networked modes")
	}
	for _, route := range append(append([]string{}, profile.PublicRoutes...), profile.ProtectedRoutes...) {
		if route == "" || !strings.HasPrefix(route, "/") || strings.ContainsAny(route, "\r\n") || strings.Contains(route, "..") || strings.Contains(route, "//") {
			return fmt.Errorf("authentication route %q is unsafe", route)
		}
	}
	return nil
}

func validateProviderEndpoint(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
		return errors.New("must be an HTTP(S) URL with a host and no credentials")
	}
	if strings.ContainsAny(raw, "\r\n\"'\\") {
		return errors.New("must not contain control characters or quoting")
	}
	return nil
}

func validatePeers(peers []Peer, services []Service) error {
	for _, peer := range peers {
		if strings.TrimSpace(peer.Scenario) == "" {
			return errors.New("peer.scenario is required")
		}
		switch peer.BundlePolicy {
		case "embed":
			prefix := peer.Scenario + "--"
			found := false
			for _, service := range services {
				if strings.HasPrefix(service.ID, prefix) {
					found = true
					break
				}
			}
			if !found {
				return MissingEmbeddedPeerError{Scenario: peer.Scenario}
			}
		case "discover", "either":
			if len(peer.Bindings) == 0 {
				return MissingPeerDiscoveryPathError{Scenario: peer.Scenario}
			}
		default:
			return fmt.Errorf("peer %q has unsupported bundle_policy %q", peer.Scenario, peer.BundlePolicy)
		}
		for _, binding := range peer.Bindings {
			if binding.EnvVar == "" || binding.Port == "" {
				return fmt.Errorf("peer %q has an incomplete discovery binding", peer.Scenario)
			}
			switch binding.Form {
			case "http_base_url", "ws_base_url", "host_port", "port_number":
			default:
				return fmt.Errorf("peer %q binding %s has unsupported form %q", peer.Scenario, binding.EnvVar, binding.Form)
			}
			if binding.WhenUnavailable != "omit" && binding.WhenUnavailable != "fail" {
				return fmt.Errorf("peer %q binding %s has unsupported when_unavailable %q", peer.Scenario, binding.EnvVar, binding.WhenUnavailable)
			}
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
	if m.IPC.Host == "" || m.IPC.Port < 0 {
		return errors.New("ipc.host and a non-negative ipc.port allocator input are required")
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
