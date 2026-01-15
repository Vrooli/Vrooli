package distribution

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const DistributionConfigFilename = "distribution.json"

// GlobalRepository implements Repository using global file storage.
type GlobalRepository struct {
	fs         FileSystem
	vrooliRoot string
}

// GlobalRepositoryOption configures a GlobalRepository.
type GlobalRepositoryOption func(*GlobalRepository)

// WithGlobalFileSystem sets a custom file system.
func WithGlobalFileSystem(fs FileSystem) GlobalRepositoryOption {
	return func(r *GlobalRepository) {
		r.fs = fs
	}
}

// WithVrooliRoot sets the Vrooli root path.
func WithVrooliRoot(root string) GlobalRepositoryOption {
	return func(r *GlobalRepository) {
		r.vrooliRoot = root
	}
}

// NewGlobalRepository creates a new global repository.
func NewGlobalRepository(opts ...GlobalRepositoryOption) *GlobalRepository {
	r := &GlobalRepository{
		fs: &RealFileSystem{},
	}

	for _, opt := range opts {
		opt(r)
	}

	// Determine Vrooli root if not set
	if r.vrooliRoot == "" {
		r.vrooliRoot = detectVrooliRoot()
	}

	return r
}

// detectVrooliRoot finds the Vrooli root directory.
func detectVrooliRoot() string {
	// 1. Check environment variable
	if root := os.Getenv("VROOLI_ROOT"); root != "" {
		return root
	}

	// 2. Try default home directory location
	if homeDir, err := os.UserHomeDir(); err == nil {
		defaultRoot := filepath.Join(homeDir, "Vrooli")
		if info, err := os.Stat(defaultRoot); err == nil && info.IsDir() {
			return defaultRoot
		}
	}

	// 3. Try to find by walking up from current directory
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	for dir := cwd; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
		vrooliDir := filepath.Join(dir, ".vrooli")
		if info, err := os.Stat(vrooliDir); err == nil && info.IsDir() {
			return dir
		}
	}

	return ""
}

// GetPath returns the path to distribution.json.
func (r *GlobalRepository) GetPath() string {
	return filepath.Join(r.vrooliRoot, ".vrooli", DistributionConfigFilename)
}

// Get retrieves the global distribution config.
func (r *GlobalRepository) Get(ctx context.Context) (*DistributionConfig, error) {
	path := r.GetPath()

	if !r.fs.Exists(path) {
		// Return empty config with no targets
		return &DistributionConfig{
			SchemaVersion: SchemaVersion,
			Targets:       make(map[string]*DistributionTarget),
		}, nil
	}

	data, err := r.fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read distribution config: %w", err)
	}

	var config DistributionConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse distribution config: %w", err)
	}

	// Ensure maps are initialized
	if config.Targets == nil {
		config.Targets = make(map[string]*DistributionTarget)
	}

	return &config, nil
}

// Save stores the global distribution config.
func (r *GlobalRepository) Save(ctx context.Context, config *DistributionConfig) error {
	if config.SchemaVersion == "" {
		config.SchemaVersion = SchemaVersion
	}

	path := r.GetPath()

	// Ensure .vrooli directory exists
	dir := filepath.Dir(path)
	if err := r.fs.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal distribution config: %w", err)
	}

	if err := r.fs.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write distribution config: %w", err)
	}

	return nil
}

// GetTarget retrieves a specific target config.
func (r *GlobalRepository) GetTarget(ctx context.Context, name string) (*DistributionTarget, error) {
	config, err := r.Get(ctx)
	if err != nil {
		return nil, err
	}

	target, ok := config.Targets[name]
	if !ok {
		return nil, nil
	}

	return target, nil
}

// SaveTarget creates or updates a specific target.
func (r *GlobalRepository) SaveTarget(ctx context.Context, name string, target *DistributionTarget) error {
	config, err := r.Get(ctx)
	if err != nil {
		return err
	}

	// Set timestamps
	now := time.Now().UTC().Format(time.RFC3339)
	if target.CreatedAt == "" {
		// Check if this is an update to existing target
		if existing, ok := config.Targets[name]; ok && existing.CreatedAt != "" {
			target.CreatedAt = existing.CreatedAt
		} else {
			target.CreatedAt = now
		}
	}
	target.UpdatedAt = now

	config.Targets[name] = target
	return r.Save(ctx, config)
}

// DeleteTarget removes a target.
func (r *GlobalRepository) DeleteTarget(ctx context.Context, name string) error {
	config, err := r.Get(ctx)
	if err != nil {
		return err
	}

	delete(config.Targets, name)
	return r.Save(ctx, config)
}

// ListTargets returns all target names.
func (r *GlobalRepository) ListTargets(ctx context.Context) ([]string, error) {
	config, err := r.Get(ctx)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(config.Targets))
	for name := range config.Targets {
		names = append(names, name)
	}

	return names, nil
}
