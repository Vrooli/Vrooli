package closure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

// ErrNotFound is returned by a Catalog when a component is not declared.
var ErrNotFound = errors.New("component not declared")

// Catalog reads component declarations. Every method reports a typed error
// when the underlying declaration is unreadable; a Catalog never returns an
// empty declaration for a missing component.
type Catalog interface {
	// Scenario loads one scenario declaration by id.
	Scenario(ctx context.Context, id string) (*ScenarioDeclaration, error)
	// Resource loads one resource declaration by id.
	Resource(ctx context.Context, id string) (*ResourceDeclaration, error)
	// SystemRequiredScenarios lists scenarios whose service.system_required is true.
	SystemRequiredScenarios(ctx context.Context) ([]string, error)
	// Name identifies the catalog in Closure.Sources.
	Name() string
}

// DependencyEdge is one declared dependency edge from a scenario manifest.
type DependencyEdge struct {
	Enabled       bool
	Required      bool
	StartupPolicy string
	RuntimeOnly   bool
	Description   string
}

// HostRequirementDeclaration is a hostTools/hostSafeguards entry as declared.
type HostRequirementDeclaration struct {
	Name      string   `json:"name"`
	Required  bool     `json:"required"`
	Reason    string   `json:"reason"`
	Platforms []string `json:"platforms,omitempty"`
	Privilege string   `json:"privilege,omitempty"`
	Bundling  string   `json:"bundling,omitempty"`
}

// CredentialDescriptor mirrors common.schema.json#/definitions/credentialDescriptor.
type CredentialDescriptor struct {
	LogicalID string `json:"logical_id"`
	Field     string `json:"field"`
	Env       string `json:"env,omitempty"`
	Required  bool   `json:"required"`
	Label     string `json:"label,omitempty"`
}

// Address returns the credential address in the shared logical_id:field form.
func (d CredentialDescriptor) Address() string {
	field := d.Field
	if field == "" {
		field = "value"
	}
	return d.LogicalID + ":" + field
}

// Requirements is the declared footprint (service.schema deploymentRequirements).
type Requirements struct {
	RAMMB    float64 `json:"ram_mb"`
	DiskMB   float64 `json:"disk_mb"`
	CPUCores float64 `json:"cpu_cores"`
	Class    string  `json:"class,omitempty"`
}

// PortDeclaration is one entry of a scenario's ports map.
type PortDeclaration struct {
	Port   int
	EnvVar string
}

// SupportedTarget is one deployment.supported_targets entry.
type SupportedTarget struct {
	OS            string   `json:"os"`
	Distribution  string   `json:"distribution,omitempty"`
	Version       string   `json:"version,omitempty"`
	Architectures []string `json:"architectures"`
}

// PersistentDataDeclaration is one deployment.persistent_data entry.
type PersistentDataDeclaration struct {
	ID             string `json:"id"`
	Owner          string `json:"owner,omitempty"`
	Binding        string `json:"binding"`
	BackupProvider string `json:"backup_provider,omitempty"`
	MigrationOwner string `json:"migration_owner"`
}

// ListenerDeclaration is one deployment.listeners entry.
type ListenerDeclaration struct {
	ID         string                `json:"id"`
	Port       string                `json:"port"`
	Visibility string                `json:"visibility"`
	Readiness  *ReadinessDeclaration `json:"readiness,omitempty"`
}

// ReadinessDeclaration mirrors service.schema componentReadiness.
type ReadinessDeclaration struct {
	Type      string `json:"type"`
	Path      string `json:"path,omitempty"`
	TimeoutMS int    `json:"timeout_ms,omitempty"`
}

// RecoveryDeclaration is the deployment.recovery block.
type RecoveryDeclaration struct {
	CodeRollback   string `json:"code_rollback"`
	SchemaStrategy string `json:"schema_strategy"`
}

// DeploymentDeclaration is the additive deployment block shared by scenario
// and resource manifests.
type DeploymentDeclaration struct {
	SupportedTargets []SupportedTarget           `json:"supported_targets,omitempty"`
	PersistentData   []PersistentDataDeclaration `json:"persistent_data,omitempty"`
	Listeners        []ListenerDeclaration       `json:"listeners,omitempty"`
	Recovery         *RecoveryDeclaration        `json:"recovery,omitempty"`
}

// Supports reports whether the declared targets admit the platform. An empty
// declaration admits every platform (the resource artifacts decide).
func (d DeploymentDeclaration) Supports(os, arch string) bool {
	if len(d.SupportedTargets) == 0 {
		return true
	}
	for _, target := range d.SupportedTargets {
		if !strings.EqualFold(target.OS, os) {
			continue
		}
		if len(target.Architectures) == 0 {
			return true
		}
		for _, candidate := range target.Architectures {
			if strings.EqualFold(candidate, arch) {
				return true
			}
		}
	}
	return false
}

// ScenarioDeclaration is the closure's reading of one service.json.
type ScenarioDeclaration struct {
	ID                 string
	Path               string
	ContentIdentity    string
	Version            string
	SystemRequired     bool
	RuntimeKind        string
	AutoRestartDefault bool
	Resources          map[string]DependencyEdge
	Scenarios          map[string]DependencyEdge
	HostTools          []HostRequirementDeclaration
	HostSafeguards     []HostRequirementDeclaration
	Credentials        []CredentialDescriptor
	Ports              map[string]PortDeclaration
	ComponentReadiness map[string]ReadinessDeclaration
	Requirements       *Requirements
	Deployment         DeploymentDeclaration
}

// ResourceDeclaration is the closure's reading of one resource.json.
type ResourceDeclaration struct {
	ID              string
	Path            string
	ContentIdentity string
	Version         string
	Privilege       string
	Bundling        string
	Requirements    *Requirements
	Dependencies    []string
	HostTools       []HostRequirementDeclaration
	HostSafeguards  []HostRequirementDeclaration
	Credentials     []CredentialDescriptor
	Profiles        resourcedeployment.Deployment
	Artifact        *resourcedeployment.ServiceArtifact
	AllowedModes    []string
	Ports           []string
	Deployment      DeploymentDeclaration
}

type rawEdge struct {
	Enabled       *bool  `json:"enabled"`
	Required      *bool  `json:"required"`
	StartupPolicy string `json:"startup_policy"`
	RuntimeOnly   bool   `json:"runtime_only"`
	Description   string `json:"description"`
}

func (e rawEdge) edge() DependencyEdge {
	edge := DependencyEdge{Enabled: true, Required: true, StartupPolicy: e.StartupPolicy, RuntimeOnly: e.RuntimeOnly, Description: e.Description}
	if e.Enabled != nil {
		edge.Enabled = *e.Enabled
	}
	if e.Required != nil {
		edge.Required = *e.Required
	}
	if edge.StartupPolicy == "" {
		if edge.Required {
			edge.StartupPolicy = "must_start"
		} else {
			edge.StartupPolicy = "try_start"
		}
	}
	return edge
}

type rawServiceManifest struct {
	Service struct {
		Name           string `json:"name"`
		Version        string `json:"version"`
		SystemRequired bool   `json:"system_required"`
	} `json:"service"`
	Runtime struct {
		Kind               string `json:"kind"`
		AutoRestartDefault bool   `json:"auto_restart_default"`
	} `json:"runtime"`
	Dependencies struct {
		Resources map[string]rawEdge `json:"resources"`
		Scenarios map[string]rawEdge `json:"scenarios"`
	} `json:"dependencies"`
	HostTools      []HostRequirementDeclaration `json:"hostTools"`
	HostSafeguards []HostRequirementDeclaration `json:"hostSafeguards"`
	Credentials    struct {
		Descriptors []CredentialDescriptor `json:"descriptors"`
	} `json:"credentials"`
	Ports map[string]struct {
		Port   json.RawMessage `json:"port"`
		EnvVar string          `json:"env_var"`
	} `json:"ports"`
	Components map[string]struct {
		Run struct {
			Port      string                `json:"port"`
			Readiness *ReadinessDeclaration `json:"readiness"`
		} `json:"run"`
	} `json:"components"`
	TierFeasibility struct {
		Tiers map[string]struct {
			Requirements *Requirements `json:"requirements"`
		} `json:"tiers"`
	} `json:"tier_feasibility"`
	Deployment DeploymentDeclaration `json:"deployment"`
}

// capacityTierOrder is the preference order for a scenario's declared
// footprint. A VPS is a server tier; tier-1-local is the authored baseline.
var capacityTierOrder = []string{"tier-4-saas", "tier-1-local"}

// ParseScenarioDeclaration parses service.json bytes into a declaration.
func ParseScenarioDeclaration(id, path string, data []byte) (*ScenarioDeclaration, error) {
	var raw rawServiceManifest
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse service.json for %s: %w", id, err)
	}
	decl := &ScenarioDeclaration{
		ID:                 id,
		Path:               path,
		ContentIdentity:    contentIdentity(data),
		Version:            raw.Service.Version,
		SystemRequired:     raw.Service.SystemRequired,
		RuntimeKind:        raw.Runtime.Kind,
		AutoRestartDefault: raw.Runtime.AutoRestartDefault,
		Resources:          map[string]DependencyEdge{},
		Scenarios:          map[string]DependencyEdge{},
		HostTools:          raw.HostTools,
		HostSafeguards:     raw.HostSafeguards,
		Credentials:        raw.Credentials.Descriptors,
		Ports:              map[string]PortDeclaration{},
		ComponentReadiness: map[string]ReadinessDeclaration{},
		Deployment:         raw.Deployment,
	}
	if decl.RuntimeKind == "" {
		decl.RuntimeKind = "on_demand"
	}
	for name, edge := range raw.Dependencies.Resources {
		decl.Resources[name] = edge.edge()
	}
	for name, edge := range raw.Dependencies.Scenarios {
		decl.Scenarios[name] = edge.edge()
	}
	for name, port := range raw.Ports {
		decl.Ports[name] = PortDeclaration{Port: parsePort(port.Port), EnvVar: port.EnvVar}
	}
	for _, component := range raw.Components {
		if component.Run.Port != "" && component.Run.Readiness != nil {
			decl.ComponentReadiness[component.Run.Port] = *component.Run.Readiness
		}
	}
	for _, tier := range capacityTierOrder {
		if entry, ok := raw.TierFeasibility.Tiers[tier]; ok && entry.Requirements != nil {
			decl.Requirements = entry.Requirements
			break
		}
	}
	return decl, nil
}

func parsePort(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	var number int
	if err := json.Unmarshal(raw, &number); err == nil {
		return number
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		if number, err := strconv.Atoi(strings.TrimSpace(text)); err == nil {
			return number
		}
	}
	return 0
}

type rawResourceManifest struct {
	Name           string                       `json:"name"`
	Privilege      string                       `json:"privilege"`
	Bundling       string                       `json:"bundling"`
	Requirements   *Requirements                `json:"requirements"`
	Dependencies   []string                     `json:"dependencies"`
	HostTools      []HostRequirementDeclaration `json:"hostTools"`
	HostSafeguards []HostRequirementDeclaration `json:"hostSafeguards"`
	Credentials    struct {
		Descriptors []CredentialDescriptor `json:"descriptors"`
	} `json:"credentials"`
	Deployment     json.RawMessage `json:"deployment"`
	ManagedService *struct {
		Artifact       *resourcedeployment.ServiceArtifact `json:"artifact"`
		ProviderPolicy struct {
			AllowedModes []string `json:"allowed_modes"`
		} `json:"provider_policy"`
	} `json:"managed_service"`
	Ports []struct {
		Name string `json:"name"`
	} `json:"ports"`
}

// ParseResourceDeclaration parses resource.json bytes into a declaration.
func ParseResourceDeclaration(id, path string, data []byte) (*ResourceDeclaration, error) {
	var raw rawResourceManifest
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse resource.json for %s: %w", id, err)
	}
	decl := &ResourceDeclaration{
		ID:              id,
		Path:            path,
		ContentIdentity: contentIdentity(data),
		Privilege:       raw.Privilege,
		Bundling:        raw.Bundling,
		Requirements:    raw.Requirements,
		Dependencies:    raw.Dependencies,
		HostTools:       raw.HostTools,
		HostSafeguards:  raw.HostSafeguards,
		Credentials:     raw.Credentials.Descriptors,
	}
	if len(raw.Deployment) > 0 {
		if err := json.Unmarshal(raw.Deployment, &decl.Profiles); err != nil {
			return nil, fmt.Errorf("parse deployment profiles for %s: %w", id, err)
		}
		if err := json.Unmarshal(raw.Deployment, &decl.Deployment); err != nil {
			return nil, fmt.Errorf("parse deployment declaration for %s: %w", id, err)
		}
	}
	if raw.ManagedService != nil {
		decl.Artifact = raw.ManagedService.Artifact
		decl.AllowedModes = raw.ManagedService.ProviderPolicy.AllowedModes
		if raw.ManagedService.Artifact != nil {
			decl.Version = raw.ManagedService.Artifact.Version
		}
	}
	for _, port := range raw.Ports {
		if port.Name != "" {
			decl.Ports = append(decl.Ports, port.Name)
		}
	}
	sort.Strings(decl.Ports)
	return decl, nil
}

func contentIdentity(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Layout maps component ids to their manifest paths. It is the only piece of
// filesystem knowledge a file catalog holds.
type Layout struct {
	ScenarioRoot func(id string) (string, error)
	ResourceRoot func(id string) (string, error)
	// ListScenarioRoots enumerates every scenario root for the system-required
	// scan. It may return an empty list when the layout cannot enumerate.
	ListScenarioRoots func() ([]string, error)
}

// FileCatalog reads declarations from a repository layout.
type FileCatalog struct {
	layout Layout
	name   string
}

// NewRepoCatalog builds a catalog over a repository governed by the repo contract.
func NewRepoCatalog(repoRoot string) (*FileCatalog, error) {
	contract, err := repocontract.LoadDefault(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("load repo contract: %w", err)
	}
	scenariosDir, err := contract.TopLevelDir(repoRoot, "scenarios")
	if err != nil {
		return nil, fmt.Errorf("resolve scenarios dir: %w", err)
	}
	return &FileCatalog{name: "repo:" + filepath.Base(repoRoot), layout: Layout{
		ScenarioRoot:      func(id string) (string, error) { return contract.ScenarioRoot(repoRoot, id) },
		ResourceRoot:      func(id string) (string, error) { return contract.ResourceRoot(repoRoot, id) },
		ListScenarioRoots: func() ([]string, error) { return listChildDirs(scenariosDir) },
	}}, nil
}

// NewLayoutCatalog builds a catalog over explicit scenarios/ and resources/
// directories using the conventional <root>/.vrooli/service.json and
// <root>/resource.json manifest locations. Fixtures use it; production uses
// NewRepoCatalog.
func NewLayoutCatalog(scenariosDir, resourcesDir string) *FileCatalog {
	return &FileCatalog{name: "layout:" + filepath.Base(scenariosDir), layout: Layout{
		ScenarioRoot:      func(id string) (string, error) { return filepath.Join(scenariosDir, id), nil },
		ResourceRoot:      func(id string) (string, error) { return filepath.Join(resourcesDir, id), nil },
		ListScenarioRoots: func() ([]string, error) { return listChildDirs(scenariosDir) },
	}}
}

func listChildDirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	roots := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			roots = append(roots, filepath.Join(dir, entry.Name()))
		}
	}
	sort.Strings(roots)
	return roots, nil
}

// Name implements Catalog.
func (c *FileCatalog) Name() string { return c.name }

// Scenario implements Catalog.
func (c *FileCatalog) Scenario(_ context.Context, id string) (*ScenarioDeclaration, error) {
	id = strings.TrimSpace(id)
	if !validComponentID(id) {
		return nil, fmt.Errorf("invalid scenario id %q", id)
	}
	root, err := c.layout.ScenarioRoot(id)
	if err != nil {
		return nil, fmt.Errorf("resolve scenario %s: %w", id, err)
	}
	path := filepath.Join(root, ".vrooli", "service.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("scenario %s: %w", id, ErrNotFound)
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return ParseScenarioDeclaration(id, root, data)
}

// Resource implements Catalog.
func (c *FileCatalog) Resource(_ context.Context, id string) (*ResourceDeclaration, error) {
	id = strings.TrimSpace(id)
	if !validComponentID(id) {
		return nil, fmt.Errorf("invalid resource id %q", id)
	}
	root, err := c.layout.ResourceRoot(id)
	if err != nil {
		return nil, fmt.Errorf("resolve resource %s: %w", id, err)
	}
	path := filepath.Join(root, "resource.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("resource %s: %w", id, ErrNotFound)
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return ParseResourceDeclaration(id, root, data)
}

// SystemRequiredScenarios implements Catalog by scanning every scenario
// manifest for service.system_required.
func (c *FileCatalog) SystemRequiredScenarios(_ context.Context) ([]string, error) {
	roots, err := c.layout.ListScenarioRoots()
	if err != nil {
		return nil, fmt.Errorf("list scenarios: %w", err)
	}
	var ids []string
	for _, root := range roots {
		data, err := os.ReadFile(filepath.Join(root, ".vrooli", "service.json"))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", root, err)
		}
		var raw struct {
			Service struct {
				Name           string `json:"name"`
				SystemRequired bool   `json:"system_required"`
			} `json:"service"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse %s: %w", root, err)
		}
		if raw.Service.SystemRequired {
			id := raw.Service.Name
			if id == "" {
				id = filepath.Base(root)
			}
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

func validComponentID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
		default:
			return false
		}
	}
	return !strings.HasPrefix(id, ".") && !strings.Contains(id, "..")
}
