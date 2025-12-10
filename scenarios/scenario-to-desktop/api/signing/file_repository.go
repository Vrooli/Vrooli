package signing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"scenario-to-desktop-api/signing/types"
)

// FileRepository implements Repository using file-based storage.
// Each scenario stores its signing configuration in signing.json in the scenario directory.
type FileRepository struct {
	fs              FileSystem
	scenarioLocator ScenarioLocator
}

// FileRepositoryOption configures a FileRepository.
type FileRepositoryOption func(*FileRepository)

// WithFileSystemRepo sets a custom file system.
func WithFileSystemRepo(fs FileSystem) FileRepositoryOption {
	return func(r *FileRepository) {
		r.fs = fs
	}
}

// WithScenarioLocator sets a custom scenario locator.
func WithScenarioLocator(locator ScenarioLocator) FileRepositoryOption {
	return func(r *FileRepository) {
		r.scenarioLocator = locator
	}
}

// NewFileRepository creates a new file-based repository.
func NewFileRepository(opts ...FileRepositoryOption) *FileRepository {
	r := &FileRepository{
		fs:              NewRealFileSystem(),
		scenarioLocator: NewDefaultScenarioLocator(),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Get retrieves the signing config for a scenario.
func (r *FileRepository) Get(ctx context.Context, scenario string) (*types.SigningConfig, error) {
	path := r.GetPath(scenario)
	if !r.fs.Exists(path) {
		return nil, nil // No config is not an error
	}

	data, err := r.fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read signing config: %w", err)
	}

	var config types.SigningConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse signing config: %w", err)
	}

	return &config, nil
}

// Save stores or updates the signing config for a scenario.
func (r *FileRepository) Save(ctx context.Context, scenario string, config *types.SigningConfig) error {
	// Ensure schema version is set
	if config.SchemaVersion == "" {
		config.SchemaVersion = types.SchemaVersion
	}

	path := r.GetPath(scenario)

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := r.fs.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// Marshal with indentation for readability
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal signing config: %w", err)
	}

	if err := r.fs.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write signing config: %w", err)
	}

	return nil
}

// Delete removes the signing config for a scenario.
func (r *FileRepository) Delete(ctx context.Context, scenario string) error {
	path := r.GetPath(scenario)
	if !r.fs.Exists(path) {
		return nil // Already deleted
	}

	if err := r.fs.Remove(path); err != nil {
		return fmt.Errorf("delete signing config: %w", err)
	}

	return nil
}

// Exists checks if a signing config exists for a scenario.
func (r *FileRepository) Exists(ctx context.Context, scenario string) (bool, error) {
	path := r.GetPath(scenario)
	return r.fs.Exists(path), nil
}

// GetPath returns the path to the signing.json file for a scenario.
func (r *FileRepository) GetPath(scenario string) string {
	scenarioPath, err := r.scenarioLocator.GetScenarioPath(scenario)
	if err != nil {
		// Fall back to relative path
		return filepath.Join("scenarios", scenario, SigningConfigFilename)
	}
	return filepath.Join(scenarioPath, SigningConfigFilename)
}

// GetForPlatform retrieves config for a specific platform only.
func (r *FileRepository) GetForPlatform(ctx context.Context, scenario string, platform string) (interface{}, error) {
	config, err := r.Get(ctx, scenario)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, nil
	}

	switch platform {
	case types.PlatformWindows:
		return config.Windows, nil
	case types.PlatformMacOS:
		return config.MacOS, nil
	case types.PlatformLinux:
		return config.Linux, nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
}

// SaveForPlatform updates only a specific platform's config.
func (r *FileRepository) SaveForPlatform(ctx context.Context, scenario string, platform string, platformConfig interface{}) error {
	config, err := r.Get(ctx, scenario)
	if err != nil {
		return err
	}
	if config == nil {
		config = NewSigningConfig()
	}

	switch platform {
	case types.PlatformWindows:
		if c, ok := platformConfig.(*types.WindowsSigningConfig); ok {
			config.Windows = c
		} else {
			return fmt.Errorf("invalid Windows config type")
		}
	case types.PlatformMacOS:
		if c, ok := platformConfig.(*types.MacOSSigningConfig); ok {
			config.MacOS = c
		} else {
			return fmt.Errorf("invalid macOS config type")
		}
	case types.PlatformLinux:
		if c, ok := platformConfig.(*types.LinuxSigningConfig); ok {
			config.Linux = c
		} else {
			return fmt.Errorf("invalid Linux config type")
		}
	default:
		return fmt.Errorf("unsupported platform: %s", platform)
	}

	return r.Save(ctx, scenario, config)
}

// DeleteForPlatform removes config for a specific platform.
func (r *FileRepository) DeleteForPlatform(ctx context.Context, scenario string, platform string) error {
	config, err := r.Get(ctx, scenario)
	if err != nil {
		return err
	}
	if config == nil {
		return nil // Nothing to delete
	}

	switch platform {
	case types.PlatformWindows:
		config.Windows = nil
	case types.PlatformMacOS:
		config.MacOS = nil
	case types.PlatformLinux:
		config.Linux = nil
	default:
		return fmt.Errorf("unsupported platform: %s", platform)
	}

	// If all platforms are nil, delete the entire config
	if config.Windows == nil && config.MacOS == nil && config.Linux == nil {
		return r.Delete(ctx, scenario)
	}

	return r.Save(ctx, scenario, config)
}

// DefaultScenarioLocator implements ScenarioLocator using VROOLI_ROOT.
type DefaultScenarioLocator struct {
	vrooliRoot string
}

// NewDefaultScenarioLocator creates a new default scenario locator.
func NewDefaultScenarioLocator() *DefaultScenarioLocator {
	root := os.Getenv("VROOLI_ROOT")
	if root == "" {
		// Try common default
		homeDir, _ := os.UserHomeDir()
		if homeDir != "" {
			root = filepath.Join(homeDir, "Vrooli")
		}
	}
	return &DefaultScenarioLocator{vrooliRoot: root}
}

// GetScenarioPath returns the absolute path to a scenario directory.
func (l *DefaultScenarioLocator) GetScenarioPath(scenario string) (string, error) {
	if l.vrooliRoot == "" {
		return "", fmt.Errorf("VROOLI_ROOT not set")
	}

	// Check if scenario exists
	path := filepath.Join(l.vrooliRoot, "scenarios", scenario)
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return path, nil
	}

	return "", fmt.Errorf("scenario not found: %s", scenario)
}

// ListScenarios returns all available scenario names.
func (l *DefaultScenarioLocator) ListScenarios() ([]string, error) {
	if l.vrooliRoot == "" {
		return nil, fmt.Errorf("VROOLI_ROOT not set")
	}

	scenariosDir := filepath.Join(l.vrooliRoot, "scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		return nil, fmt.Errorf("read scenarios directory: %w", err)
	}

	var scenarios []string
	for _, entry := range entries {
		if entry.IsDir() && !isHiddenDir(entry.Name()) {
			scenarios = append(scenarios, entry.Name())
		}
	}

	return scenarios, nil
}

// isHiddenDir checks if a directory name should be hidden.
func isHiddenDir(name string) bool {
	return len(name) > 0 && name[0] == '.'
}
