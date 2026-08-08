package repocontract

import (
	"path/filepath"
	"slices"
)

const (
	defaultContractRelPath = ".vrooli/repo-contract.json"
	supportedMajorVersion  = 1
)

type contractDoc struct {
	Schema      string          `json:"$schema"`
	Version     string          `json:"version"`
	Platform    Platform        `json:"platform"`
	Root        Root            `json:"root"`
	Layout      Layout          `json:"layout"`
	RuntimeHome RuntimeHomeSpec `json:"runtime_home"`
	Scenario    ScenarioSpec    `json:"scenario"`
	Resource    ResourceSpec    `json:"resource"`
	Globs       GlobSpec        `json:"globs"`
	Targets     TargetSpecSet   `json:"targets"`
	Environment struct {
		Variables map[string]string `json:"variables"`
	} `json:"environment"`
	Sandbox struct {
		FullRepoScopes      []string `json:"full_repo_scopes"`
		ScenarioScopePrefix string   `json:"scenario_scope_prefix"`
	} `json:"sandbox"`
	Profiles map[string]Profile `json:"profiles"`
}

// Contract is an immutable view of the loaded repo contract.
type Contract struct {
	doc          contractDoc
	unknownKinds []string
}

type Platform struct {
	Mode                       string `json:"mode"`
	LegacyProjectBashSupported bool   `json:"legacy_project_bash_supported"`
}

type Root struct {
	Markers RootMarkers `json:"markers"`
}

type RootMarkers struct {
	RequiredDirs  []string `json:"required_dirs"`
	RequiredFiles []string `json:"required_files"`
}

type Layout struct {
	ProjectConfigDir string `json:"project_config_dir"`
	ScenarioDir      string `json:"scenario_dir"`
	ResourceDir      string `json:"resource_dir"`
	TemplateDir      string `json:"template_dir"`
	PackageDir       string `json:"package_dir"`
	CommandDir       string `json:"command_dir"`
	InternalDir      string `json:"internal_dir"`
	DocsDir          string `json:"docs_dir"`
}

type ScenarioSpec struct {
	RequiredFiles  []string          `json:"required_files"`
	WellKnownPaths map[string]string `json:"well_known_paths"`
}

type ResourceSpec struct {
	Manifest       string            `json:"manifest"`
	WellKnownPaths map[string]string `json:"well_known_paths"`
}

type GlobSpec struct {
	Syntax        string `json:"syntax"`
	RootRelative  bool   `json:"root_relative"`
	CaseSensitive bool   `json:"case_sensitive"`
	AllowAbsolute bool   `json:"allow_absolute"`
	PathFormat    string `json:"path_format"`
}

// TargetKind is the repository-level governance axis used by Test Genie.
type TargetKind string

const (
	TargetKindScenario     TargetKind = "scenario"
	TargetKindResource     TargetKind = "resource"
	TargetKindTool         TargetKind = "tool"
	TargetKindSafeguard    TargetKind = "safeguard"
	TargetKindTeam         TargetKind = "team"
	TargetKindPackage      TargetKind = "package"
	TargetKindControlPlane TargetKind = "control-plane"
	TargetKindDocs         TargetKind = "docs"
	TargetKindProject      TargetKind = "project"
)

// TargetSpec describes how one target kind is discovered. Roots are always
// repository-relative globs; Marker is reused only for kinds whose directories
// already have a manifest for another purpose.
type TargetSpec struct {
	Roots   []string `json:"roots"`
	Marker  string   `json:"marker,omitempty"`
	Exclude []string `json:"exclude,omitempty"`
}

// Target is a concrete repository target. Root is slash-normalized and
// repository-relative; ID is the manifest owner slug or positional root.
type Target struct {
	Kind TargetKind `json:"kind"`
	ID   string     `json:"id"`
	Root string     `json:"root"`
}

type TargetSpecSet struct {
	Kinds map[string]TargetSpec `json:"kinds"`
}

type Profile struct {
	Description     string   `json:"description"`
	Parameters      []string `json:"parameters"`
	Include         []string `json:"include"`
	OptionalInclude []string `json:"optional_include"`
	Exclude         []string `json:"exclude"`
}

// ResolveParams carries placeholder values used to expand contract profiles.
type ResolveParams struct {
	Values map[string]string
	Lists  map[string][]string
}

// ResolvedProfile is a parameter-expanded profile ready for consumption.
type ResolvedProfile struct {
	Name            string
	Description     string
	Include         []string
	OptionalInclude []string
	Exclude         []string
}

func (c *Contract) Schema() string {
	return c.doc.Schema
}

func (c *Contract) Version() string {
	return c.doc.Version
}

func (c *Contract) Platform() Platform {
	return c.doc.Platform
}

func (c *Contract) RootMarkers() RootMarkers {
	return RootMarkers{
		RequiredDirs:  slices.Clone(c.doc.Root.Markers.RequiredDirs),
		RequiredFiles: slices.Clone(c.doc.Root.Markers.RequiredFiles),
	}
}

func (c *Contract) Layout() Layout {
	return c.doc.Layout
}

// RuntimeHomeSpec returns a deep copy of the runtime-home structural spec.
func (c *Contract) RuntimeHomeSpec() RuntimeHomeSpec {
	return cloneRuntimeHomeSpec(c.doc.RuntimeHome)
}

func (c *Contract) Scenario() ScenarioSpec {
	return ScenarioSpec{
		RequiredFiles:  slices.Clone(c.doc.Scenario.RequiredFiles),
		WellKnownPaths: cloneStringMap(c.doc.Scenario.WellKnownPaths),
	}
}

func (c *Contract) Resource() ResourceSpec {
	return ResourceSpec{
		Manifest:       c.doc.Resource.Manifest,
		WellKnownPaths: cloneStringMap(c.doc.Resource.WellKnownPaths),
	}
}

func (c *Contract) Globs() GlobSpec {
	return c.doc.Globs
}

// Targets returns a defensive copy of the repository target discovery rules.
func (c *Contract) Targets() TargetSpecSet {
	out := TargetSpecSet{Kinds: make(map[string]TargetSpec, len(c.doc.Targets.Kinds))}
	for kind, spec := range c.doc.Targets.Kinds {
		out.Kinds[kind] = TargetSpec{
			Roots:   slices.Clone(spec.Roots),
			Marker:  spec.Marker,
			Exclude: slices.Clone(spec.Exclude),
		}
	}
	return out
}

// UnknownKinds returns the target kinds skipped while loading the contract.
// Unknown kinds remain non-fatal so known target kinds continue to work.
func (c *Contract) UnknownKinds() []string {
	if c == nil {
		return nil
	}
	return slices.Clone(c.unknownKinds)
}

func (c *Contract) EnvironmentVariables() map[string]string {
	return cloneStringMap(c.doc.Environment.Variables)
}

func (c *Contract) SandboxFullRepoScopes() []string {
	return slices.Clone(c.doc.Sandbox.FullRepoScopes)
}

func (c *Contract) SandboxScenarioScopePrefix() string {
	return c.doc.Sandbox.ScenarioScopePrefix
}

func (c *Contract) Profile(name string) (Profile, error) {
	profile, ok := c.doc.Profiles[name]
	if !ok {
		return Profile{}, &Error{Kind: ErrNotFound, Message: "profile not found", Details: name}
	}
	return Profile{
		Description:     profile.Description,
		Parameters:      slices.Clone(profile.Parameters),
		Include:         slices.Clone(profile.Include),
		OptionalInclude: slices.Clone(profile.OptionalInclude),
		Exclude:         slices.Clone(profile.Exclude),
	}, nil
}

func (c *Contract) Profiles() map[string]Profile {
	if len(c.doc.Profiles) == 0 {
		return nil
	}
	profiles := make(map[string]Profile, len(c.doc.Profiles))
	for name, profile := range c.doc.Profiles {
		profiles[name] = Profile{
			Description:     profile.Description,
			Parameters:      slices.Clone(profile.Parameters),
			Include:         slices.Clone(profile.Include),
			OptionalInclude: slices.Clone(profile.OptionalInclude),
			Exclude:         slices.Clone(profile.Exclude),
		}
	}
	return profiles
}

func (c *Contract) ScenarioRoot(repoRoot, scenario string) (string, error) {
	scenario, err := cleanIdentifier(scenario)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Clean(repoRoot), filepath.FromSlash(c.doc.Layout.ScenarioDir), scenario), nil
}

func (c *Contract) ResourceRoot(repoRoot, resource string) (string, error) {
	resource, err := cleanIdentifier(resource)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Clean(repoRoot), filepath.FromSlash(c.doc.Layout.ResourceDir), resource), nil
}

func cloneRuntimeHomeSpec(in RuntimeHomeSpec) RuntimeHomeSpec {
	out := RuntimeHomeSpec{
		DirName:      in.DirName,
		EnvOverrides: slices.Clone(in.EnvOverrides),
		Scoped:       cloneStringMap(in.Scoped),
	}
	if len(in.Entries) > 0 {
		out.Entries = make(map[string]RuntimeHomeEntrySpec, len(in.Entries))
		for key, entry := range in.Entries {
			out.Entries[key] = entry
		}
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
