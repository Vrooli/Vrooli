package catalog

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
)

// CensusRow is the declaration/host-state snapshot used while reconciling
// resource archetype contracts. Keep this shape stable: later phases compare
// their changes against the Phase 1 baseline.
type CensusRow struct {
	Name           string `json:"name"`
	Driver         string `json:"driver"`
	InRootContract bool   `json:"in_root_contract"`
	Enabled        bool   `json:"enabled"`
	DeclaresCLI    bool   `json:"declares_cli"`
	CLIInstalled   bool   `json:"cli_installed"`
	CLIStateReason string `json:"cli_state_reason"`
	EmptyDirs      int    `json:"empty_dirs"`
	ModulePath     string `json:"module_path"`
}

var goModuleLine = regexp.MustCompile(`^\s*module\s+(\S+)\s*$`)

// Census discovers every resource directory named by the repository contract
// and reports declaration facts separately from host observations.
func (s *Service) Census(opts DiscoverOptions) ([]CensusRow, error) {
	contract, err := repocontract.LoadDefault(s.Root)
	if err != nil {
		return nil, err
	}
	resourceRoot, err := contract.TopLevelDir(s.Root, "resources")
	if err != nil {
		return nil, err
	}
	entries, err := s.readRootConfigEntries()
	if err != nil {
		return nil, err
	}
	dirs, err := os.ReadDir(resourceRoot)
	if err != nil {
		return nil, err
	}
	rows := make([]CensusRow, 0, len(dirs))
	for _, entry := range dirs {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		manifestPath := manifestpkg.DefaultPath(s.Root, name)
		manifest, err := manifestpkg.Load(manifestPath)
		if err != nil {
			return nil, err
		}
		config, inContract := entries[name]
		installed := false
		if opts.ResolveCLIPath != nil {
			path, ok := opts.ResolveCLIPath(name)
			installed = ok && strings.TrimSpace(path) != ""
		}
		reason := censusReason(inContract, config.Enabled, manifest.CLI != nil && manifest.CLI.Enabled, installed)
		rows = append(rows, CensusRow{
			Name:           name,
			Driver:         manifest.Driver,
			InRootContract: inContract,
			Enabled:        config.Enabled,
			DeclaresCLI:    manifest.CLI != nil && manifest.CLI.Enabled,
			CLIInstalled:   installed,
			CLIStateReason: reason,
			EmptyDirs:      countEmptyDirs(filepath.Join(resourceRoot, name)),
			ModulePath:     readModulePath(filepath.Join(resourceRoot, name, "cli", "go.mod")),
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	return rows, nil
}

func (s *Service) readRootConfigEntries() (map[string]ConfigEntry, error) {
	manifest, err := scenario.LoadServiceManifest(filepath.Join(s.Root, ResourceConfigPath))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]ConfigEntry{}, nil
		}
		return nil, err
	}
	entries := make(map[string]ConfigEntry, len(manifest.Dependencies.Resources))
	for name, dependency := range manifest.Dependencies.Resources {
		entries[name] = ConfigEntry{Enabled: dependency.Enabled, Required: dependency.Required, Description: dependency.Description}
	}
	return entries, nil
}

func censusReason(inContract, enabled, declaresCLI, installed bool) string {
	switch {
	case !inContract:
		return "absent_from_contract"
	case !enabled:
		return "resource_disabled"
	case !declaresCLI:
		return "cli_not_declared"
	case !installed:
		return "declared_not_installed"
	default:
		return ""
	}
}

func countEmptyDirs(root string) int {
	empty := 0
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || !entry.IsDir() || path == root {
			return err
		}
		if entry.Name() == "target" || entry.Name() == "node_modules" {
			return filepath.SkipDir
		}
		children, readErr := os.ReadDir(path)
		if readErr != nil {
			return nil
		}
		if len(children) == 0 {
			empty++
		}
		return nil
	})
	return empty
}

func readModulePath(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		match := goModuleLine.FindStringSubmatch(scanner.Text())
		if len(match) > 1 {
			return match[1]
		}
	}
	return ""
}
