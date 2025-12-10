// Package mocks provides test mocks for the signing package.
package mocks

import (
	"context"
	"os"
	"time"

	"scenario-to-desktop-api/signing/types"
)

// MockFileSystem implements signing.FileSystem for testing.
type MockFileSystem struct {
	// Files maps paths to their contents
	Files map[string][]byte

	// Directories tracks which paths are directories
	Directories map[string]bool

	// FileInfo maps paths to their file info
	FileInfos map[string]os.FileInfo

	// ExistsFunc allows custom existence checking
	ExistsFunc func(path string) bool

	// ReadFileFunc allows custom file reading
	ReadFileFunc func(path string) ([]byte, error)

	// WriteFileFunc allows custom file writing
	WriteFileFunc func(path string, data []byte, perm os.FileMode) error

	// StatFunc allows custom stat
	StatFunc func(path string) (os.FileInfo, error)
}

// NewMockFileSystem creates a new mock file system.
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		Files:       make(map[string][]byte),
		Directories: make(map[string]bool),
		FileInfos:   make(map[string]os.FileInfo),
	}
}

// AddFile adds a file to the mock file system.
func (m *MockFileSystem) AddFile(path string, content []byte) *MockFileSystem {
	m.Files[path] = content
	return m
}

// AddDirectory adds a directory to the mock file system.
func (m *MockFileSystem) AddDirectory(path string) *MockFileSystem {
	m.Directories[path] = true
	return m
}

// Exists checks if a file or directory exists.
func (m *MockFileSystem) Exists(path string) bool {
	if m.ExistsFunc != nil {
		return m.ExistsFunc(path)
	}
	_, fileExists := m.Files[path]
	_, dirExists := m.Directories[path]
	return fileExists || dirExists
}

// ReadFile reads the file content.
func (m *MockFileSystem) ReadFile(path string) ([]byte, error) {
	if m.ReadFileFunc != nil {
		return m.ReadFileFunc(path)
	}
	if content, ok := m.Files[path]; ok {
		return content, nil
	}
	return nil, os.ErrNotExist
}

// WriteFile writes data to a file.
func (m *MockFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	if m.WriteFileFunc != nil {
		return m.WriteFileFunc(path, data, perm)
	}
	m.Files[path] = data
	return nil
}

// Stat returns file info.
func (m *MockFileSystem) Stat(path string) (os.FileInfo, error) {
	if m.StatFunc != nil {
		return m.StatFunc(path)
	}
	if info, ok := m.FileInfos[path]; ok {
		return info, nil
	}
	if _, ok := m.Files[path]; ok {
		return &mockFileInfo{name: path, size: int64(len(m.Files[path]))}, nil
	}
	if _, ok := m.Directories[path]; ok {
		return &mockFileInfo{name: path, isDir: true}, nil
	}
	return nil, os.ErrNotExist
}

// MkdirAll creates a directory.
func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	m.Directories[path] = true
	return nil
}

// Remove removes a file or empty directory.
func (m *MockFileSystem) Remove(path string) error {
	delete(m.Files, path)
	delete(m.Directories, path)
	return nil
}

// mockFileInfo implements os.FileInfo for testing.
type mockFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (f *mockFileInfo) Name() string       { return f.name }
func (f *mockFileInfo) Size() int64        { return f.size }
func (f *mockFileInfo) Mode() os.FileMode  { return f.mode }
func (f *mockFileInfo) ModTime() time.Time { return f.modTime }
func (f *mockFileInfo) IsDir() bool        { return f.isDir }
func (f *mockFileInfo) Sys() interface{}   { return nil }

// MockCommandRunner implements signing.CommandRunner for testing.
type MockCommandRunner struct {
	// RunResults maps command names to their outputs
	RunResults map[string]CommandResult

	// LookPathResults maps command names to their paths
	LookPathResults map[string]string

	// LookPathErrors maps command names to errors
	LookPathErrors map[string]error

	// RunFunc allows custom run logic
	RunFunc func(ctx context.Context, name string, args ...string) ([]byte, []byte, error)

	// LookPathFunc allows custom lookup logic
	LookPathFunc func(name string) (string, error)

	// CallLog records all Run calls
	CallLog []CommandCall
}

// CommandResult represents the result of a command execution.
type CommandResult struct {
	Stdout []byte
	Stderr []byte
	Err    error
}

// CommandCall records a command invocation.
type CommandCall struct {
	Name string
	Args []string
}

// NewMockCommandRunner creates a new mock command runner.
func NewMockCommandRunner() *MockCommandRunner {
	return &MockCommandRunner{
		RunResults:      make(map[string]CommandResult),
		LookPathResults: make(map[string]string),
		LookPathErrors:  make(map[string]error),
		CallLog:         []CommandCall{},
	}
}

// AddCommand adds a command result.
func (m *MockCommandRunner) AddCommand(name string, stdout, stderr []byte, err error) *MockCommandRunner {
	m.RunResults[name] = CommandResult{Stdout: stdout, Stderr: stderr, Err: err}
	return m
}

// AddLookPath adds a lookup path result.
func (m *MockCommandRunner) AddLookPath(name, path string) *MockCommandRunner {
	m.LookPathResults[name] = path
	return m
}

// AddLookPathError adds a lookup path error.
func (m *MockCommandRunner) AddLookPathError(name string, err error) *MockCommandRunner {
	m.LookPathErrors[name] = err
	return m
}

// Run executes a command and returns the output.
func (m *MockCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	m.CallLog = append(m.CallLog, CommandCall{Name: name, Args: args})

	if m.RunFunc != nil {
		return m.RunFunc(ctx, name, args...)
	}

	// Try exact match first (with all args)
	fullCmd := name
	for _, arg := range args {
		fullCmd += " " + arg
	}
	if result, ok := m.RunResults[fullCmd]; ok {
		return result.Stdout, result.Stderr, result.Err
	}

	// Fall back to command name only
	if result, ok := m.RunResults[name]; ok {
		return result.Stdout, result.Stderr, result.Err
	}

	return nil, nil, nil
}

// LookPath searches for an executable.
func (m *MockCommandRunner) LookPath(name string) (string, error) {
	if m.LookPathFunc != nil {
		return m.LookPathFunc(name)
	}

	if err, ok := m.LookPathErrors[name]; ok {
		return "", err
	}

	if path, ok := m.LookPathResults[name]; ok {
		return path, nil
	}

	return "", os.ErrNotExist
}

// MockTimeProvider implements signing.TimeProvider for testing.
type MockTimeProvider struct {
	// CurrentTime is the time to return from Now()
	CurrentTime time.Time
}

// NewMockTimeProvider creates a new mock time provider.
func NewMockTimeProvider(t time.Time) *MockTimeProvider {
	return &MockTimeProvider{CurrentTime: t}
}

// Now returns the mock current time.
func (m *MockTimeProvider) Now() time.Time {
	return m.CurrentTime
}

// MockEnvironmentReader implements signing.EnvironmentReader for testing.
type MockEnvironmentReader struct {
	// Vars maps environment variable names to values
	Vars map[string]string
}

// NewMockEnvironmentReader creates a new mock environment reader.
func NewMockEnvironmentReader() *MockEnvironmentReader {
	return &MockEnvironmentReader{
		Vars: make(map[string]string),
	}
}

// SetEnv sets an environment variable.
func (m *MockEnvironmentReader) SetEnv(key, value string) *MockEnvironmentReader {
	m.Vars[key] = value
	return m
}

// GetEnv retrieves an environment variable value.
func (m *MockEnvironmentReader) GetEnv(key string) string {
	return m.Vars[key]
}

// LookupEnv retrieves an environment variable and reports if it exists.
func (m *MockEnvironmentReader) LookupEnv(key string) (string, bool) {
	val, ok := m.Vars[key]
	return val, ok
}

// MockRepository implements Repository for testing.
type MockRepository struct {
	// Configs maps scenario names to signing configs
	Configs map[string]*types.SigningConfig

	// GetError is returned by Get
	GetError error

	// SaveError is returned by Save
	SaveError error

	// DeleteError is returned by Delete
	DeleteError error
}

// NewMockRepository creates a new mock repository.
func NewMockRepository() *MockRepository {
	return &MockRepository{
		Configs: make(map[string]*types.SigningConfig),
	}
}

// AddConfig adds a signing config for a scenario.
func (m *MockRepository) AddConfig(scenario string, config *types.SigningConfig) *MockRepository {
	m.Configs[scenario] = config
	return m
}

// Get retrieves the signing config for a scenario.
func (m *MockRepository) Get(ctx context.Context, scenario string) (*types.SigningConfig, error) {
	if m.GetError != nil {
		return nil, m.GetError
	}
	return m.Configs[scenario], nil
}

// Save stores or updates the signing config for a scenario.
func (m *MockRepository) Save(ctx context.Context, scenario string, config *types.SigningConfig) error {
	if m.SaveError != nil {
		return m.SaveError
	}
	m.Configs[scenario] = config
	return nil
}

// Delete removes the signing config for a scenario.
func (m *MockRepository) Delete(ctx context.Context, scenario string) error {
	if m.DeleteError != nil {
		return m.DeleteError
	}
	delete(m.Configs, scenario)
	return nil
}

// Exists checks if a signing config exists for a scenario.
func (m *MockRepository) Exists(ctx context.Context, scenario string) (bool, error) {
	_, ok := m.Configs[scenario]
	return ok, nil
}

// GetPath returns the path to the signing.json file for a scenario.
func (m *MockRepository) GetPath(scenario string) string {
	return "scenarios/" + scenario + "/signing.json"
}

// GetForPlatform retrieves config for a specific platform only.
func (m *MockRepository) GetForPlatform(ctx context.Context, scenario string, platform string) (interface{}, error) {
	config, err := m.Get(ctx, scenario)
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
		return nil, nil
	}
}

// SaveForPlatform updates only a specific platform's config.
func (m *MockRepository) SaveForPlatform(ctx context.Context, scenario string, platform string, config interface{}) error {
	if m.SaveError != nil {
		return m.SaveError
	}

	existing := m.Configs[scenario]
	if existing == nil {
		existing = &types.SigningConfig{}
		m.Configs[scenario] = existing
	}

	switch platform {
	case types.PlatformWindows:
		if c, ok := config.(*types.WindowsSigningConfig); ok {
			existing.Windows = c
		}
	case types.PlatformMacOS:
		if c, ok := config.(*types.MacOSSigningConfig); ok {
			existing.MacOS = c
		}
	case types.PlatformLinux:
		if c, ok := config.(*types.LinuxSigningConfig); ok {
			existing.Linux = c
		}
	}

	return nil
}

// DeleteForPlatform removes config for a specific platform.
func (m *MockRepository) DeleteForPlatform(ctx context.Context, scenario string, platform string) error {
	if m.DeleteError != nil {
		return m.DeleteError
	}

	config := m.Configs[scenario]
	if config == nil {
		return nil
	}

	switch platform {
	case types.PlatformWindows:
		config.Windows = nil
	case types.PlatformMacOS:
		config.MacOS = nil
	case types.PlatformLinux:
		config.Linux = nil
	}

	return nil
}

// MockScenarioLocator implements signing.ScenarioLocator for testing.
type MockScenarioLocator struct {
	// Scenarios maps scenario names to their paths
	Scenarios map[string]string

	// GetPathError is returned by GetScenarioPath
	GetPathError error
}

// NewMockScenarioLocator creates a new mock scenario locator.
func NewMockScenarioLocator() *MockScenarioLocator {
	return &MockScenarioLocator{
		Scenarios: make(map[string]string),
	}
}

// AddScenario adds a scenario to the mock locator.
func (m *MockScenarioLocator) AddScenario(name, path string) *MockScenarioLocator {
	m.Scenarios[name] = path
	return m
}

// GetScenarioPath returns the absolute path to a scenario directory.
func (m *MockScenarioLocator) GetScenarioPath(scenario string) (string, error) {
	if m.GetPathError != nil {
		return "", m.GetPathError
	}
	if path, ok := m.Scenarios[scenario]; ok {
		return path, nil
	}
	return "", os.ErrNotExist
}

// ListScenarios returns all available scenario names.
func (m *MockScenarioLocator) ListScenarios() ([]string, error) {
	scenarios := make([]string, 0, len(m.Scenarios))
	for name := range m.Scenarios {
		scenarios = append(scenarios, name)
	}
	return scenarios, nil
}
