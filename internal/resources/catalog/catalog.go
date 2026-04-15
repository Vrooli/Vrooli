package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/vrooli/vrooli/internal/repocontractmeta"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

const ResourceConfigPath = repocontractmeta.ServiceManifestPathname

type ConfigEntry struct {
	Enabled     bool   `json:"enabled,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Description string `json:"description,omitempty"`
}

type Resource struct {
	Name            string      `json:"name"`
	Path            string      `json:"path"`
	Exists          bool        `json:"exists"`
	Registered      bool        `json:"registered"`
	Enabled         bool        `json:"enabled"`
	Required        bool        `json:"required"`
	HasCLI          bool        `json:"has_cli"`
	Config          ConfigEntry `json:"config"`
	ControlMode     string      `json:"control_mode,omitempty"`
	Driver          string      `json:"driver,omitempty"`
	Template        string      `json:"template,omitempty"`
	PortabilityTier string      `json:"portability_tier,omitempty"`
	ManifestPath    string      `json:"manifest_path,omitempty"`
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
	configEntries, err := s.ReadConfigEntries()
	if err != nil {
		return nil, err
	}

	manifestNames, err := s.ManifestNames()
	if err != nil {
		return nil, err
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

	items := make([]Resource, 0, len(names))
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
				return nil, err
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
			item.PortabilityTier = manifest.PortabilityTier
			item.ManifestPath = manifestPath
			item.ControlMode = "manifest-native"
		}
		items = append(items, item)
	}

	return items, nil
}

func (s *Service) DiscoverOne(name string, opts DiscoverOptions) (*Resource, error) {
	items, err := s.Discover(opts)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].Name == name {
			return &items[i], nil
		}
	}
	return nil, nil
}

func (s *Service) ReadConfigEntries() (map[string]ConfigEntry, error) {
	configPath := filepath.Join(s.Root, filepath.FromSlash(ResourceConfigPath))
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]ConfigEntry{}, nil
		}
		return nil, err
	}

	var payload struct {
		Dependencies struct {
			Resources map[string]ConfigEntry `json:"resources"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if payload.Dependencies.Resources == nil {
		return map[string]ConfigEntry{}, nil
	}
	return payload.Dependencies.Resources, nil
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
