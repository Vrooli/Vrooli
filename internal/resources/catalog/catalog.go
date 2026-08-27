package catalog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/vrooli/vrooli/internal/discovery"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
)

const ResourceConfigPath = repocontractmeta.ServiceManifestPathname

type ConfigEntry struct {
	Enabled     bool   `json:"enabled,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Description string `json:"description,omitempty"`
}

type Resource struct {
	Name         string      `json:"name"`
	Path         string      `json:"path"`
	Exists       bool        `json:"exists"`
	Registered   bool        `json:"registered"`
	Enabled      bool        `json:"enabled"`
	Required     bool        `json:"required"`
	HasCLI       bool        `json:"has_cli"`
	Config       ConfigEntry `json:"config"`
	ControlMode  string      `json:"control_mode,omitempty"`
	Driver       string      `json:"driver,omitempty"`
	Template     string      `json:"template,omitempty"`
	ManifestPath string      `json:"manifest_path,omitempty"`
}

type DiscoverOptions struct {
	DeprecatedNames map[string]struct{}
	ResolveCLIPath  func(name string) (string, bool)
}

type Service struct {
	Root string
}

func New(root string) *Service {
	return &Service{Root: filepath.Clean(root)}
}

func (s *Service) Discover(opts DiscoverOptions) ([]Resource, error) {
	report, err := s.DiscoverReport(opts)
	if err != nil {
		return nil, err
	}
	if len(report.Failures) > 0 {
		failure := report.Failures[0]
		return nil, fmt.Errorf("load resource %s: %s", failure.Name, failure.Error)
	}
	return report.Items, nil
}

func (s *Service) DiscoverReport(opts DiscoverOptions) (discovery.Report[Resource], error) {
	configEntries, err := s.ReadConfigEntries()
	if err != nil {
		return discovery.Report[Resource]{}, err
	}

	manifestNames, err := s.ManifestNames()
	if err != nil {
		return discovery.Report[Resource]{}, err
	}

	namesMap := make(map[string]struct{}, len(manifestNames))
	for _, name := range manifestNames {
		namesMap[name] = struct{}{}
	}

	names := make([]string, 0, len(namesMap))
	for name := range namesMap {
		if _, hidden := opts.DeprecatedNames[name]; hidden {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	report := discovery.Report[Resource]{
		Items:    make([]Resource, 0, len(names)),
		Failures: make([]discovery.Failure, 0),
	}
	for _, name := range names {
		configEntry, registered := configEntries[name]
		path := filepath.Join(s.Root, "resources", name)
		_, statErr := os.Stat(path)
		exists := statErr == nil
		cliPath := ""
		hasCLI := false
		if opts.ResolveCLIPath != nil {
			cliPath, hasCLI = opts.ResolveCLIPath(name)
		}
		manifestPath := manifestpkg.DefaultPath(s.Root, name)
		manifest := manifestpkg.ResourceManifest{}
		hasManifest := false
		if _, err := os.Stat(manifestPath); err == nil {
			loaded, err := manifestpkg.Load(manifestPath)
			if err != nil {
				report.Failures = append(report.Failures, discovery.Failure{
					Kind:  "resource",
					Name:  name,
					Path:  manifestPath,
					Stage: "load_manifest",
					Error: err.Error(),
				})
				continue
			}
			manifest = loaded
			hasManifest = true
		}

		item := Resource{
			Name:       name,
			Path:       path,
			Exists:     exists,
			Registered: registered,
			Enabled:    configEntry.Enabled,
			Required:   configEntry.Required,
			HasCLI:     hasCLI && cliPath != "",
			Config:     configEntry,
		}
		if hasManifest {
			item.Driver = manifest.Driver
			item.Template = manifest.Template
			item.ManifestPath = manifestPath
			item.ControlMode = "manifest-native"
		}
		report.Items = append(report.Items, item)
	}

	return report, nil
}

func (s *Service) DiscoverOne(name string, opts DiscoverOptions) (*Resource, error) {
	configEntries, err := s.ReadConfigEntries()
	if err != nil {
		return nil, err
	}
	if _, hidden := opts.DeprecatedNames[name]; hidden {
		return nil, nil
	}
	path := filepath.Join(s.Root, "resources", name)
	_, statErr := os.Stat(path)
	exists := statErr == nil
	manifestPath := manifestpkg.DefaultPath(s.Root, name)
	_, manifestErr := os.Stat(manifestPath)
	if os.IsNotExist(statErr) && os.IsNotExist(manifestErr) {
		return nil, nil
	}
	configEntry, registered := configEntries[name]
	cliPath := ""
	hasCLI := false
	if opts.ResolveCLIPath != nil {
		cliPath, hasCLI = opts.ResolveCLIPath(name)
	}
	item := &Resource{
		Name:       name,
		Path:       path,
		Exists:     exists,
		Registered: registered,
		Enabled:    configEntry.Enabled,
		Required:   configEntry.Required,
		HasCLI:     hasCLI && cliPath != "",
		Config:     configEntry,
	}
	if manifestErr == nil {
		manifest, err := manifestpkg.Load(manifestPath)
		if err != nil {
			return nil, err
		}
		item.Driver = manifest.Driver
		item.Template = manifest.Template
		item.ManifestPath = manifestPath
		item.ControlMode = "manifest-native"
	}
	return item, nil
}

func (s *Service) ReadConfigEntries() (map[string]ConfigEntry, error) {
	configPath := filepath.Join(s.Root, filepath.FromSlash(ResourceConfigPath))
	manifest, err := scenario.LoadServiceManifest(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]ConfigEntry{}, nil
		}
		return nil, err
	}

	entries := make(map[string]ConfigEntry, len(manifest.Dependencies.Resources))
	for name, dependency := range manifest.Dependencies.Resources {
		entries[name] = ConfigEntry{
			Enabled:     dependency.Enabled,
			Required:    dependency.Required,
			Description: dependency.Description,
		}
	}

	// The project manifest supplies defaults. Per-install operator choices are
	// authoritative once present, so resource enable/disable changes are
	// visible to every catalog consumer, including `vrooli resource status`.
	state, err := operatorstate.New(operatorstate.Config{RepoRoot: s.Root}).Effective(context.Background())
	if err != nil {
		return nil, fmt.Errorf("read operator state: %w", err)
	}
	for name, choice := range state.Resources {
		if choice.Enabled == nil {
			continue
		}
		entry := entries[name]
		entry.Enabled = *choice.Enabled
		entries[name] = entry
	}
	return entries, nil
}

func (s *Service) FilesystemNames() ([]string, error) {
	resourceDir := filepath.Join(s.Root, "resources")
	entries, err := os.ReadDir(resourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func (s *Service) ManifestNames() ([]string, error) {
	filesystemNames, err := s.FilesystemNames()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(filesystemNames))
	for _, name := range filesystemNames {
		manifestPath := manifestpkg.DefaultPath(s.Root, name)
		if _, err := os.Stat(manifestPath); err == nil {
			names = append(names, name)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	return names, nil
}
