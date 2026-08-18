// Package resourcedeployment defines the portable, manifest-serialized
// deployment contract shared by the control plane and deployment consumers.
package resourcedeployment

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/vrooli/binaryfetch"
)

type Deployment struct {
	Profiles map[string]Profile `json:"profiles,omitempty"`
}

type Profile struct {
	Linux   *Target `json:"linux,omitempty"`
	MacOS   *Target `json:"macos,omitempty"`
	Windows *Target `json:"windows,omitempty"`
}

type Target struct {
	Support       string   `json:"support"`
	Mode          string   `json:"mode"`
	Architectures []string `json:"architectures,omitempty"`
	Requires      []string `json:"requires,omitempty"`
	Limitations   []string `json:"limitations,omitempty"`
	Fallbacks     []string `json:"fallbacks,omitempty"`
	Evidence      []string `json:"evidence,omitempty"`
	Reason        string   `json:"reason,omitempty"`
}

// ProviderMode identifies who supplies a managed service and, therefore, who
// has lifecycle authority. It is deliberately distinct from a deployment mode:
// a desktop bundle can be a bundled-service while its provider is private or a
// consented shared Vrooli instance.
type ProviderMode string

const (
	ProviderManagedPrivate ProviderMode = "managed-private"
	// ProviderManagedShared is a Vrooli-owned per-user resource host.  The
	// historical wire spelling remains stable because signed desktop plans use
	// it; callers should describe it as "user-hosted", never as an arbitrary
	// shared endpoint.
	ProviderManagedShared ProviderMode = "managed-shared"
	ProviderAttachOnly    ProviderMode = "attach-only"
	ProviderRemoteVrooli  ProviderMode = "remote-vrooli"
	// ProviderManagedDiscovered permits a verified executable candidate. Vrooli
	// never adopts a running endpoint: after checksum/version verification it
	// may supervise the executable with Vrooli-owned state and configuration.
	ProviderManagedDiscovered ProviderMode = "managed-discovered"
)

// ProviderTarget identifies the runtime that is selecting a managed-service
// provider. It is part of the portable policy contract because a control-plane
// resource host and a desktop bundle have intentionally different safe
// defaults.
type ProviderTarget string

const (
	ProviderTargetControlPlane  ProviderTarget = "control-plane"
	ProviderTargetDesktopBundle ProviderTarget = "desktop-bundle"
)

// AccessCapability is the maximum application access an attach-only provider
// may issue. It is intentionally separate from ProviderMode: write access to
// an external Vault never gives Vrooli lifecycle authority over that Vault.
type AccessCapability string

const (
	AccessReadOnly  AccessCapability = "read-only"
	AccessReadWrite AccessCapability = "read-write"
)

// ProviderPolicy is the portable form of resource.json's managed_service
// provider policy. External endpoints are intentionally attach-only: no
// selection result ever grants Vrooli lifecycle authority over them.
type ProviderPolicy struct {
	DefaultMode                ProviderMode                    `json:"default_mode"`
	TargetDefaults             map[ProviderTarget]ProviderMode `json:"target_defaults,omitempty"`
	AllowedModes               []ProviderMode                  `json:"allowed_modes"`
	SharedReuseRequiresConsent bool                            `json:"shared_reuse_requires_consent"`
	ExternalManagement         string                          `json:"external_management"`
	ExternalAccessCapabilities []AccessCapability              `json:"external_access_capabilities,omitempty"`
}

// ServiceShutdown declares the first process signal used when the control
// plane stops a managed service. The stop timeout on the enclosing resource
// remains the total bounded lifecycle budget; if the process is still alive
// when that budget expires, the supervisor escalates to a forced termination
// and reports failure if the process cannot be observed stopped.
//
// "terminate" maps to the platform's ordinary graceful termination request
// (SIGTERM on Unix). "interrupt" maps to SIGINT on Unix and the platform's
// native console-control request on Windows. Services should select the
// signal their own shutdown semantics require instead of making the driver
// guess from the resource name.
type ServiceShutdown struct {
	Signal string `json:"signal"`
}

const (
	ServiceShutdownTerminate = "terminate"
	ServiceShutdownInterrupt = "interrupt"
)

func (s ServiceShutdown) Validate() error {
	switch strings.ToLower(strings.TrimSpace(s.Signal)) {
	case ServiceShutdownTerminate, ServiceShutdownInterrupt:
		return nil
	default:
		return fmt.Errorf("managed-service shutdown signal must be %q or %q", ServiceShutdownTerminate, ServiceShutdownInterrupt)
	}
}

// ManagedService is the manifest-facing managed_service block. Keeping the
// nesting here makes the JSON contract shared by resource manifests, the
// control plane, and deployment consumers.
type ManagedService struct {
	ProviderPolicy ProviderPolicy `json:"provider_policy"`
	// ArtifactRole and ProvenanceClass are release metadata declared by the
	// resource. The distributor uses them instead of inferring trust from an
	// artifact filename.
	ArtifactRole    string          `json:"artifact_role,omitempty"`
	ProvenanceClass string          `json:"provenance_class,omitempty"`
	Artifact        ServiceArtifact `json:"artifact,omitempty"`
	// Acquisition is the one declared source contract used to stage this
	// service for control-plane and desktop deployments. Artifact remains the
	// launch-time verification contract for the staged result.
	Acquisition *binaryfetch.Acquisition `json:"acquisition,omitempty"`
	// DataArtifacts are checksum-verified, non-executable artifacts that must
	// arrive in the resource-owned data directory before the service starts.
	// Model files are the primary use case; keeping them separate from the
	// launch artifact prevents a service archive from silently becoming a model
	// supply chain.
	DataArtifacts    []ManagedServiceDataArtifact `json:"data_artifacts,omitempty"`
	AttachHealthPath string                       `json:"attach_health_path,omitempty"`
	Arguments        []string                     `json:"arguments,omitempty"`
	Environment      map[string]string            `json:"environment,omitempty"`
	Bootstrap        *ServiceBootstrap            `json:"bootstrap,omitempty"`
	ProcessLimits    *ProcessLimits               `json:"process_limits,omitempty"`
	Shutdown         *ServiceShutdown             `json:"shutdown,omitempty"`
	// EnvironmentFile is an optional resource-owned, line-oriented KEY=VALUE
	// file loaded from RESOURCE_DATA_DIR immediately before launch. It gives a
	// resource a durable, non-shell model/config switch without granting the
	// manifest arbitrary command or host-environment authority.
	EnvironmentFile string `json:"environment_file,omitempty"`
	// CredentialFiles are materialized only for the supervised child and are
	// removed when it stops. Their contents never enter the process
	// environment, argv, logs, or manifest.
	CredentialFiles []ServiceCredentialFile `json:"credential_files,omitempty"`
	Config          *ServiceConfig          `json:"config,omitempty"`
}

// ServiceCredentialFile maps one authority field to a short-lived file below
// RESOURCE_STATE_DIR. Path is relative to that runtime directory and is never
// accepted as an absolute host path.
type ServiceCredentialFile struct {
	LogicalID string `json:"logical_id"`
	Field     string `json:"field,omitempty"`
	Path      string `json:"path"`
}

func (m ManagedService) ValidateCredentialFiles() error {
	seen := make(map[string]struct{}, len(m.CredentialFiles))
	for index, declaration := range m.CredentialFiles {
		identity := strings.TrimSpace(declaration.LogicalID)
		if identity == "" || !strings.Contains(identity, "/") {
			return fmt.Errorf("managed-service credential_files[%d] logical_id must be namespaced", index)
		}
		field := strings.TrimSpace(declaration.Field)
		if field == "" {
			field = "value"
		}
		if strings.ContainsAny(field, `/\\`) {
			return fmt.Errorf("managed-service credential_files[%d] field cannot contain a path separator", index)
		}
		path := filepath.Clean(filepath.FromSlash(strings.TrimSpace(declaration.Path)))
		if path == "." || path == ".." || filepath.IsAbs(declaration.Path) || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
			return fmt.Errorf("managed-service credential_files[%d] path must remain under RESOURCE_STATE_DIR", index)
		}
		key := identity + ":" + field
		if _, exists := seen[key]; exists {
			return fmt.Errorf("managed-service credential_files declares %s more than once", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

// ManagedServiceDataArtifact describes one durable, regenerable artifact
// staged below RESOURCE_DATA_DIR. Path is relative to that directory.
type ManagedServiceDataArtifact struct {
	Name        string                   `json:"name"`
	Path        string                   `json:"path"`
	Acquisition *binaryfetch.Acquisition `json:"acquisition"`
}

// ProcessLimits are applied to the supervised process after it starts.
// Memory percentages map to kernel soft/hard address-space limits on hosts
// where Vrooli cannot create a delegated service cgroup. OOMScoreAdjust is
// Linux-specific and optional.
type ProcessLimits struct {
	MemoryHighPercent uint8 `json:"memory_high_percent,omitempty"`
	MemoryMaxPercent  uint8 `json:"memory_max_percent,omitempty"`
	OOMScoreAdjust    int   `json:"oom_score_adjust,omitempty"`
}

func (l ProcessLimits) Validate() error {
	if l.MemoryHighPercent == 0 && l.MemoryMaxPercent == 0 && l.OOMScoreAdjust == 0 {
		return fmt.Errorf("managed-service process_limits must declare at least one limit")
	}
	if l.MemoryHighPercent > 100 || l.MemoryMaxPercent > 100 {
		return fmt.Errorf("managed-service process_limits memory percentages must be between 1 and 100")
	}
	if l.MemoryHighPercent > 0 && l.MemoryMaxPercent > 0 && l.MemoryHighPercent > l.MemoryMaxPercent {
		return fmt.Errorf("managed-service process_limits memory_high_percent must not exceed memory_max_percent")
	}
	if l.OOMScoreAdjust < -1000 || l.OOMScoreAdjust > 1000 {
		return fmt.Errorf("managed-service process_limits oom_score_adjust must be between -1000 and 1000")
	}
	return nil
}

// ServiceBootstrap describes one idempotent, manifest-owned initialization
// command for a directory service. It runs only when Marker is absent and
// uses the already verified artifact root. PasswordFile is written from the
// named PasswordEnv at launch time and is never part of the manifest itself.
type ServiceBootstrap struct {
	Marker       string   `json:"marker"`
	Executable   string   `json:"executable"`
	Arguments    []string `json:"arguments,omitempty"`
	PasswordEnv  string   `json:"password_env,omitempty"`
	PasswordFile string   `json:"password_file,omitempty"`
}

func (b ServiceBootstrap) Validate() error {
	if !safeRelativeDeploymentPath(b.Marker) || !safeRelativeDeploymentPath(b.Executable) {
		return fmt.Errorf("managed-service bootstrap marker and executable must be safe relative paths")
	}
	if strings.TrimSpace(b.PasswordEnv) == "" && strings.TrimSpace(b.PasswordFile) != "" {
		return fmt.Errorf("managed-service bootstrap password_file requires password_env")
	}
	if strings.TrimSpace(b.PasswordFile) != "" && !safeRelativeDeploymentPath(b.PasswordFile) {
		return fmt.Errorf("managed-service bootstrap password_file must be a safe relative path")
	}
	return nil
}

func safeRelativeDeploymentPath(value string) bool {
	value = filepath.Clean(filepath.FromSlash(strings.TrimSpace(value)))
	return value != "" && value != "." && value != ".." && !filepath.IsAbs(value) && !strings.HasPrefix(value, ".."+string(filepath.Separator))
}

func (m ManagedService) ValidateAttachHealthPath() error {
	path := strings.TrimSpace(m.AttachHealthPath)
	if path == "" {
		return fmt.Errorf("managed-service attach_health_path is required when attach-only is allowed")
	}
	parsed, err := url.Parse(path)
	if err != nil || !strings.HasPrefix(path, "/") || parsed.IsAbs() || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("managed-service attach_health_path must be an absolute request path without query or fragment")
	}
	return nil
}

// ValidateDataArtifacts validates the durable, non-executable acquisition
// declarations attached to a managed service.
func (m ManagedService) ValidateDataArtifacts() error {
	seen := make(map[string]struct{}, len(m.DataArtifacts))
	for i, artifact := range m.DataArtifacts {
		name := strings.TrimSpace(artifact.Name)
		if name == "" {
			return fmt.Errorf("data artifact %d name is required", i)
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("data artifact %q is declared more than once", name)
		}
		seen[name] = struct{}{}
		path := filepath.Clean(filepath.FromSlash(strings.TrimSpace(artifact.Path)))
		if path == "." || path == ".." || filepath.IsAbs(artifact.Path) || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
			return fmt.Errorf("data artifact %q path must remain below RESOURCE_DATA_DIR", name)
		}
		if artifact.Acquisition == nil {
			return fmt.Errorf("data artifact %q acquisition is required", name)
		}
		if err := artifact.Acquisition.Validate(); err != nil {
			return fmt.Errorf("data artifact %q acquisition: %w", name, err)
		}
	}
	return nil
}

// ServiceConfig is a non-secret, template-rendered configuration file written
// beneath the resource configuration root before a private service starts.
type ServiceConfig struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (c ServiceConfig) Validate() error {
	if path := strings.TrimSpace(c.Path); path == "" || filepath.IsAbs(path) || filepath.Clean(path) == "." || filepath.Clean(path) == ".." || strings.HasPrefix(filepath.Clean(path), ".."+string(filepath.Separator)) {
		return fmt.Errorf("service config path must be a non-empty relative path")
	}
	if strings.TrimSpace(c.Content) == "" || strings.ContainsRune(c.Content, '\x00') {
		return fmt.Errorf("service config content must be non-empty and contain no NUL")
	}
	return nil
}

// ServiceArtifact identifies the exact executable that a managed-service
// supervisor is permitted to launch. Paths are deliberately relative to the
// resource/bundle root: a manifest cannot use this contract to silently adopt
// an arbitrary executable elsewhere on the host.
type ServiceArtifact struct {
	Path             string            `json:"path"`
	Version          string            `json:"version"`
	SHA256           string            `json:"sha256,omitempty"`
	SHA256ByPlatform map[string]string `json:"sha256_by_platform,omitempty"`
	// Layout declares what the supervisor verifies and launches. A directory
	// artifact is authenticated by binaryfetch.TreeDigest and launches its
	// explicitly declared EntryPath; the default remains a single file for
	// backwards-compatible manifests.
	Layout    string `json:"layout,omitempty"`
	EntryPath string `json:"entry_path,omitempty"`
	// BundleArtifact is the target-expanded basename used when this server is
	// included in a signed desktop release.  It is distinct from Path: Path is
	// the control-plane launch location, while BundleArtifact identifies the
	// immutable release asset staged into a desktop bundle.
	BundleArtifact string `json:"bundle_artifact,omitempty"`
	// Verification permits an explicitly governed host tool to be adopted by a
	// managed service. The default remains checksum verification for every
	// staged artifact.
	Verification string `json:"verification,omitempty"`
}

// Validate checks that an artifact declaration is safe to resolve beneath its
// owning resource root and that it pins a complete SHA-256 digest.
func (a ServiceArtifact) Validate() error {
	path := strings.TrimSpace(a.Path)
	if path == "" || filepath.IsAbs(path) || filepath.Clean(path) == "." {
		return fmt.Errorf("artifact path must be a non-empty relative path")
	}
	if strings.HasPrefix(filepath.Clean(path), ".."+string(filepath.Separator)) || filepath.Clean(path) == ".." {
		return fmt.Errorf("artifact path must not escape its resource root")
	}
	if strings.TrimSpace(a.Version) == "" {
		return fmt.Errorf("artifact version is required")
	}
	verification := strings.ToLower(strings.TrimSpace(a.Verification))
	if verification != "" && verification != "checksum" && verification != "host-tool" {
		return fmt.Errorf("artifact verification %q is invalid", a.Verification)
	}
	if verification == "host-tool" && strings.TrimSpace(a.BundleArtifact) != "" {
		return fmt.Errorf("host-tool artifacts cannot declare bundle_artifact")
	}
	layout := strings.ToLower(strings.TrimSpace(a.Layout))
	if layout != "" && layout != "file" && layout != "dir" {
		return fmt.Errorf("artifact layout %q is invalid", a.Layout)
	}
	if entry := strings.TrimSpace(a.EntryPath); entry != "" {
		clean := filepath.Clean(filepath.FromSlash(entry))
		if filepath.IsAbs(entry) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("artifact entry_path must be a non-empty relative path")
		}
		if layout != "dir" {
			return fmt.Errorf("artifact entry_path requires dir layout")
		}
	}
	if layout == "dir" && strings.TrimSpace(a.EntryPath) == "" {
		return fmt.Errorf("directory artifact entry_path is required")
	}
	if template := strings.TrimSpace(a.BundleArtifact); template != "" {
		// Expanding one concrete target validates both the placeholder shape and
		// the resulting basename. ArtifactName performs the same platform-specific
		// Windows suffix normalization used by the release pipeline.
		if _, err := ArtifactName(template, "linux", "amd64"); err != nil {
			return fmt.Errorf("artifact bundle_artifact: %w", err)
		}
	}
	if strings.TrimSpace(a.SHA256) != "" {
		if err := validateSHA256("artifact sha256", a.SHA256); err != nil {
			return err
		}
	}
	for platform, sum := range a.SHA256ByPlatform {
		if _, err := ParsePlatform(platform); err != nil {
			return fmt.Errorf("artifact sha256_by_platform key %q: %w", platform, err)
		}
		if err := validateSHA256("artifact sha256_by_platform "+platform, sum); err != nil {
			return err
		}
	}
	if verification != "host-tool" && strings.TrimSpace(a.SHA256) == "" && len(a.SHA256ByPlatform) == 0 {
		return fmt.Errorf("artifact sha256 or sha256_by_platform is required")
	}
	return nil
}

// BundleArtifactForPlatform resolves the server filename selected by the
// signed desktop release. A resource that does not declare a bundle artifact
// is still valid for control-plane-only deployment, but cannot claim the
// bundled-service delivery mode.
func (a ServiceArtifact) BundleArtifactForPlatform(osName, arch string) (string, error) {
	if strings.TrimSpace(a.BundleArtifact) == "" {
		return "", fmt.Errorf("artifact bundle_artifact is required for bundled-service delivery")
	}
	return ArtifactName(a.BundleArtifact, osName, arch)
}

func validateSHA256(label, value string) error {
	sum := strings.ToLower(strings.TrimSpace(value))
	if len(sum) != sha256.Size*2 {
		return fmt.Errorf("%s must be a %d-character hex digest", label, sha256.Size*2)
	}
	if _, err := hex.DecodeString(sum); err != nil {
		return fmt.Errorf("%s is invalid: %w", label, err)
	}
	return nil
}

// ForCurrentPlatform resolves a target-specific executable checksum. A single
// SHA256 applies to every platform; SHA256ByPlatform is required when release
// executables differ by target, as they do for Vault.
func (a ServiceArtifact) ForCurrentPlatform() (ServiceArtifact, error) {
	return a.ForPlatform(runtime.GOOS, runtime.GOARCH)
}

func (a ServiceArtifact) ForPlatform(osName, arch string) (ServiceArtifact, error) {
	if err := a.Validate(); err != nil {
		return ServiceArtifact{}, err
	}
	if a.SHA256 != "" {
		return a, nil
	}
	platform, err := CanonicalPlatform(osName, arch)
	if err != nil {
		return ServiceArtifact{}, err
	}
	sum, ok := a.SHA256ByPlatform[platform.String()]
	if !ok {
		return ServiceArtifact{}, fmt.Errorf("artifact has no checksum for %s", platform)
	}
	a.SHA256 = sum
	return a, nil
}

// VerifyFile checks a staged executable or executable tree before it receives lifecycle
// authority. Signature verification belongs to the release/staging boundary;
// this second checksum check protects the local hand-off to the supervisor.
func (a ServiceArtifact) VerifyFile(path string) error {
	if err := a.Validate(); err != nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(a.Verification), "host-tool") {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("resolve managed-service host tool: %w", err)
		}
		if info.IsDir() || info.Mode()&0o111 == 0 {
			return fmt.Errorf("managed-service host tool is not executable")
		}
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(a.Layout), "dir") {
		got, err := binaryfetch.TreeDigest(path)
		if err != nil {
			return fmt.Errorf("hash managed-service artifact tree: %w", err)
		}
		if !strings.EqualFold(got, strings.TrimSpace(a.SHA256)) {
			return fmt.Errorf("managed-service artifact checksum mismatch")
		}
		return nil
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("read managed-service artifact: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("hash managed-service artifact: %w", err)
	}
	if hex.EncodeToString(hash.Sum(nil)) != strings.ToLower(strings.TrimSpace(a.SHA256)) {
		return fmt.Errorf("managed-service artifact checksum mismatch")
	}
	return nil
}

// LaunchPath resolves the executable within a verified artifact root.
func (a ServiceArtifact) LaunchPath(root string) (string, error) {
	if err := a.Validate(); err != nil {
		return "", err
	}
	if !strings.EqualFold(strings.TrimSpace(a.Layout), "dir") {
		return root, nil
	}
	entry := filepath.Clean(filepath.FromSlash(a.EntryPath))
	path := filepath.Join(root, entry)
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("artifact entry_path escapes artifact root")
	}
	return path, nil
}

// ProviderRequest carries only explicit operator/app choices. A blank mode
// selects the resource default. Shared reuse must be explicitly consented;
// callers cannot accidentally adopt a user-level instance.
type ProviderRequest struct {
	Mode            ProviderMode
	Target          ProviderTarget
	SharedConsented bool
}

// ResolveProvider applies the resource policy without falling back to an
// arbitrary local process or endpoint. Callers use the selected mode to decide
// whether they may ask the broker for a lease, start a private service, or only
// validate an external endpoint.
func (p ProviderPolicy) ResolveProvider(request ProviderRequest) (ProviderMode, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	mode := request.Mode
	if mode == "" {
		if request.Target != "" {
			var ok bool
			mode, ok = p.TargetDefaults[request.Target]
			if !ok {
				return "", fmt.Errorf("provider policy has no default for target %q", request.Target)
			}
		} else {
			mode = p.DefaultMode
		}
	}
	if !isProviderMode(mode) {
		return "", fmt.Errorf("unsupported provider mode %q", mode)
	}
	if !slices.Contains(p.AllowedModes, mode) {
		return "", fmt.Errorf("provider mode %q is not allowed", mode)
	}
	if mode == ProviderManagedShared && p.SharedReuseRequiresConsent && request.Target != ProviderTargetControlPlane && !request.SharedConsented {
		return "", fmt.Errorf("managed-shared provider requires explicit consent")
	}
	return mode, nil
}

// Validate rejects ambiguous provider declarations before a caller can make a
// lifecycle or credential decision. Keeping this in the portable contract
// makes resource manifests and signed bundle plans enforce the same policy.
func (p ProviderPolicy) Validate() error {
	if p.ExternalManagement != "" && p.ExternalManagement != "forbidden" {
		return fmt.Errorf("external management must be forbidden")
	}
	if len(p.AllowedModes) == 0 {
		return fmt.Errorf("provider policy must allow at least one mode")
	}
	if p.DefaultMode != "" {
		if !isProviderMode(p.DefaultMode) {
			return fmt.Errorf("unsupported default provider mode %q", p.DefaultMode)
		}
		if !slices.Contains(p.AllowedModes, p.DefaultMode) {
			return fmt.Errorf("default provider mode %q is not allowed", p.DefaultMode)
		}
	}
	if p.DefaultMode == "" && len(p.TargetDefaults) == 0 {
		return fmt.Errorf("provider policy must declare a default mode or target defaults")
	}
	seenModes := make(map[ProviderMode]bool, len(p.AllowedModes))
	for _, mode := range p.AllowedModes {
		if !isProviderMode(mode) || seenModes[mode] {
			return fmt.Errorf("provider policy contains invalid or duplicate mode %q", mode)
		}
		seenModes[mode] = true
	}
	for target, mode := range p.TargetDefaults {
		if !isProviderTarget(target) {
			return fmt.Errorf("provider policy contains unsupported target %q", target)
		}
		if !isProviderMode(mode) || !slices.Contains(p.AllowedModes, mode) {
			return fmt.Errorf("provider policy target %q has invalid or disallowed default mode %q", target, mode)
		}
	}
	if seenModes[ProviderAttachOnly] != (len(p.ExternalAccessCapabilities) > 0) {
		return fmt.Errorf("external access capabilities must be declared exactly when attach-only is allowed")
	}
	seenCapabilities := make(map[AccessCapability]bool, len(p.ExternalAccessCapabilities))
	for _, capability := range p.ExternalAccessCapabilities {
		if (capability != AccessReadOnly && capability != AccessReadWrite) || seenCapabilities[capability] {
			return fmt.Errorf("provider policy contains invalid or duplicate external access capability %q", capability)
		}
		seenCapabilities[capability] = true
	}
	return nil
}

// ValidateManagedServiceTargets requires a managed service to declare the
// defaults for the two runtimes Vrooli can operate. A static default is valid
// for a non-service host tool, but cannot express the ownership boundary
// between the control plane and a desktop bundle.
func (p ProviderPolicy) ValidateManagedServiceTargets() error {
	if err := p.Validate(); err != nil {
		return err
	}
	if p.DefaultMode != "" {
		return fmt.Errorf("managed-service provider policy must use target_defaults instead of default_mode")
	}
	for _, target := range []ProviderTarget{ProviderTargetControlPlane, ProviderTargetDesktopBundle} {
		if _, err := p.ResolveProvider(ProviderRequest{Target: target}); err != nil {
			return err
		}
	}
	return nil
}

func isProviderTarget(target ProviderTarget) bool {
	switch target {
	case ProviderTargetControlPlane, ProviderTargetDesktopBundle:
		return true
	default:
		return false
	}
}

func isProviderMode(mode ProviderMode) bool {
	switch mode {
	case ProviderManagedPrivate, ProviderManagedShared, ProviderAttachOnly, ProviderRemoteVrooli, ProviderManagedDiscovered:
		return true
	default:
		return false
	}
}

// AllowsExternalAccess rejects escalation. A caller requesting read-write
// must be explicitly authorized for read-write; read-only is not inferred
// from endpoint reachability or a provider's lifecycle policy.
func (p ProviderPolicy) AllowsExternalAccess(capability AccessCapability) bool {
	if capability != AccessReadOnly && capability != AccessReadWrite {
		return false
	}
	return slices.Contains(p.ExternalAccessCapabilities, capability)
}

// Platform is the canonical, concrete desktop target used everywhere after
// command-line input is accepted.  Keeping OS and architecture together avoids
// treating electron-builder identifiers such as linux-amd64 as OS names.
type Platform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// ParsePlatform accepts the public desktop spellings (linux-amd64, mac-arm64,
// win, etc.) and returns a concrete target. A missing architecture is rejected:
// artifact selection must never fall back to "all architectures".
func ParsePlatform(value string) (Platform, error) {
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(value)), func(r rune) bool { return r == '-' || r == '_' || r == '/' })
	if len(parts) != 2 {
		return Platform{}, fmt.Errorf("desktop target %q must include OS and architecture (for example linux-amd64)", value)
	}
	os, ok := canonicalOS(parts[0])
	if !ok {
		return Platform{}, fmt.Errorf("unsupported desktop OS %q", parts[0])
	}
	arch := strings.TrimSpace(parts[1])
	if arch != "amd64" && arch != "arm64" {
		return Platform{}, fmt.Errorf("unsupported desktop architecture %q", arch)
	}
	return Platform{OS: os, Arch: arch}, nil
}

// CanonicalPlatform combines separately supplied OS and architecture values.
func CanonicalPlatform(os, arch string) (Platform, error) {
	canonical, ok := canonicalOS(strings.ToLower(strings.TrimSpace(os)))
	if !ok {
		return Platform{}, fmt.Errorf("unsupported desktop OS %q", os)
	}
	return ParsePlatform(canonical + "-" + arch)
}

func (p Platform) String() string { return p.OS + "-" + p.Arch }

func canonicalOS(platform string) (string, bool) {
	switch platform {
	case "linux":
		return "linux", true
	case "mac", "macos", "darwin":
		return "macos", true
	case "win", "windows":
		return "windows", true
	default:
		return "", false
	}
}

// Target resolves one profile/OS/architecture declaration. Empty arch permits
// callers that have not yet selected a concrete architecture to assess OS
// support without making a false artifact-availability claim.
func (d Deployment) Target(profile, platform, arch string) (Target, bool) {
	p, ok := d.Profiles[strings.TrimSpace(profile)]
	if !ok {
		return Target{}, false
	}
	var target *Target
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "linux":
		target = p.Linux
	case "mac", "macos", "darwin":
		target = p.MacOS
	case "win", "windows":
		target = p.Windows
	}
	if target == nil || (target.Support != "unsupported" && strings.TrimSpace(arch) != "" && !slices.Contains(target.Architectures, arch)) {
		return Target{}, false
	}
	return *target, true
}

// ResolveTarget resolves a concrete parsed platform against a profile.
func (d Deployment) ResolveTarget(profile string, platform Platform) (Target, bool) {
	return d.Target(profile, platform.OS, platform.Arch)
}

// ArtifactName expands a manifest artifact template for a concrete target.
// Manifest platform spelling is operator-facing (macos); release artifacts use
// Go's spelling (darwin). Windows executables always retain their .exe suffix.
func ArtifactName(template, platform, arch string) (string, error) {
	os := strings.ToLower(strings.TrimSpace(platform))
	switch os {
	case "mac", "macos", "darwin":
		os = "darwin"
	case "win", "windows":
		os = "windows"
	case "linux":
	default:
		return "", fmt.Errorf("unsupported artifact platform %q", platform)
	}
	arch = strings.TrimSpace(arch)
	if arch == "" {
		return "", fmt.Errorf("artifact architecture is required")
	}
	name := strings.ReplaceAll(strings.ReplaceAll(template, "${os}", os), "${arch}", arch)
	if name == template || strings.Contains(name, "${") {
		return "", fmt.Errorf("invalid artifact template %q", template)
	}
	if os == "windows" && !strings.HasSuffix(strings.ToLower(name), ".exe") {
		name += ".exe"
	}
	if !IsSafeArtifactName(name) {
		return "", fmt.Errorf("artifact name %q is not a basename", name)
	}
	return name, nil
}

// IsSafeArtifactName prevents a manifest from escaping a signed release or
// unpacked bundle directory. Artifact identities are always basenames.
func IsSafeArtifactName(name string) bool {
	name = strings.TrimSpace(name)
	return name != "" && filepath.Base(name) == name && !strings.Contains(name, "..") && !strings.ContainsAny(name, `/\\`)
}

// ArtifactFiles is the immutable resource artifact set. The sibling manifest
// and build metadata travel with the executable and receive independent hashes.
func ArtifactFiles(binary string) ([]string, error) {
	if !IsSafeArtifactName(binary) {
		return nil, fmt.Errorf("artifact name %q is not a basename", binary)
	}
	return []string{binary, binary + ".manifest.json", binary + ".build.json"}, nil
}
