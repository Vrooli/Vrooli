package seams

import (
	"io/fs"
	"sync"
	"time"
)

// TestClock implements Clock with controllable time for testing.
type TestClock struct {
	mu   sync.Mutex
	time time.Time
}

// NewTestClock creates a TestClock at the given time.
func NewTestClock(t time.Time) *TestClock {
	return &TestClock{time: t}
}

// Now returns the fixed or advanced time.
func (c *TestClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.time
}

// Set changes the clock's time.
func (c *TestClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.time = t
}

// Advance moves the clock forward by the given duration.
func (c *TestClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.time = c.time.Add(d)
}

// SequentialIDGenerator returns predictable IDs for testing.
type SequentialIDGenerator struct {
	mu     sync.Mutex
	prefix string
	next   int
}

// NewSequentialIDGenerator creates an ID generator with a prefix.
func NewSequentialIDGenerator(prefix string) *SequentialIDGenerator {
	return &SequentialIDGenerator{prefix: prefix, next: 1}
}

// NewID returns the next sequential ID.
func (g *SequentialIDGenerator) NewID() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	id := g.prefix + "-" + string(rune('0'+g.next))
	if g.next < 9 {
		g.next++
	}
	return id
}

// MemoryFileSystem implements FileSystem backed by in-memory storage.
type MemoryFileSystem struct {
	mu    sync.RWMutex
	files map[string][]byte
	dirs  map[string]bool
}

// NewMemoryFileSystem creates an empty in-memory filesystem.
func NewMemoryFileSystem() *MemoryFileSystem {
	return &MemoryFileSystem{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

// AddFile adds a file to the in-memory filesystem.
func (m *MemoryFileSystem) AddFile(path string, content []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[path] = content
}

// AddDir marks a path as a directory.
func (m *MemoryFileSystem) AddDir(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dirs[path] = true
}

// ReadDir reads a directory (stub implementation).
func (m *MemoryFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Simplified: return empty if no files match
	return []fs.DirEntry{}, nil
}

// Stat returns file info (stub implementation).
func (m *MemoryFileSystem) Stat(name string) (fs.FileInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := m.files[name]; ok {
		return &memFileInfo{name: name, size: int64(len(m.files[name]))}, nil
	}
	if m.dirs[name] {
		return &memFileInfo{name: name, isDir: true}, nil
	}
	return nil, fs.ErrNotExist
}

// ReadFile reads file contents.
func (m *MemoryFileSystem) ReadFile(name string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if data, ok := m.files[name]; ok {
		return data, nil
	}
	return nil, fs.ErrNotExist
}

// WriteFile writes file contents.
func (m *MemoryFileSystem) WriteFile(name string, data []byte, perm fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[name] = data
	return nil
}

// MkdirAll creates directories.
func (m *MemoryFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dirs[path] = true
	return nil
}

// memFileInfo is a minimal fs.FileInfo implementation.
type memFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (f *memFileInfo) Name() string       { return f.name }
func (f *memFileInfo) Size() int64        { return f.size }
func (f *memFileInfo) Mode() fs.FileMode  { return 0644 }
func (f *memFileInfo) ModTime() time.Time { return time.Time{} }
func (f *memFileInfo) IsDir() bool        { return f.isDir }
func (f *memFileInfo) Sys() interface{}   { return nil }

// MockStore implements DependencyStore for testing.
type MockStore struct {
	mu                   sync.RWMutex
	dependencies         map[string]map[string]interface{}
	metadata             map[string]interface{}
	recommendations      map[string]interface{}
	metrics              map[string]interface{}
	StoreDependenciesErr error
	LoadDependenciesErr  error
	LoadAllErr           error
	UpdateMetadataErr    error
	LoadMetadataErr      error
	LoadRecsErr          error
	PersistRecsErr       error
	CollectMetricsErr    error
	CleanupErr           error
}

// NewMockStore creates an empty mock store.
func NewMockStore() *MockStore {
	return &MockStore{
		dependencies:    make(map[string]map[string]interface{}),
		metadata:        make(map[string]interface{}),
		recommendations: make(map[string]interface{}),
		metrics: map[string]interface{}{
			"scenarios_found":     0,
			"resources_available": 0,
			"database_status":     "connected",
			"last_analysis":       nil,
		},
	}
}

func (m *MockStore) StoreDependencies(analysis interface{}, extras interface{}) error {
	if m.StoreDependenciesErr != nil {
		return m.StoreDependenciesErr
	}
	return nil
}

func (m *MockStore) LoadStoredDependencies(scenario string) (map[string]interface{}, error) {
	if m.LoadDependenciesErr != nil {
		return nil, m.LoadDependenciesErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if deps, ok := m.dependencies[scenario]; ok {
		return deps, nil
	}
	return map[string]interface{}{
		"resources":        []interface{}{},
		"scenarios":        []interface{}{},
		"shared_workflows": []interface{}{},
	}, nil
}

func (m *MockStore) LoadAllDependencies() (interface{}, error) {
	if m.LoadAllErr != nil {
		return nil, m.LoadAllErr
	}
	return []interface{}{}, nil
}

func (m *MockStore) UpdateScenarioMetadata(name string, cfg interface{}, scenarioPath string) error {
	if m.UpdateMetadataErr != nil {
		return m.UpdateMetadataErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metadata[name] = cfg
	return nil
}

func (m *MockStore) LoadScenarioMetadataMap() (map[string]interface{}, error) {
	if m.LoadMetadataErr != nil {
		return nil, m.LoadMetadataErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]interface{})
	for k, v := range m.metadata {
		result[k] = v
	}
	return result, nil
}

func (m *MockStore) LoadOptimizationRecommendations(scenario string) (interface{}, error) {
	if m.LoadRecsErr != nil {
		return nil, m.LoadRecsErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.recommendations[scenario], nil
}

func (m *MockStore) PersistOptimizationRecommendations(scenario string, recs interface{}) error {
	if m.PersistRecsErr != nil {
		return m.PersistRecsErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recommendations[scenario] = recs
	return nil
}

func (m *MockStore) CollectAnalysisMetrics() (map[string]interface{}, error) {
	if m.CollectMetricsErr != nil {
		return nil, m.CollectMetricsErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]interface{})
	for k, v := range m.metrics {
		result[k] = v
	}
	return result, nil
}

func (m *MockStore) CleanupInvalidScenarioDependencies(known map[string]struct{}) error {
	if m.CleanupErr != nil {
		return m.CleanupErr
	}
	return nil
}

// SetDependencies sets test dependency data for a scenario.
func (m *MockStore) SetDependencies(scenario string, deps map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dependencies[scenario] = deps
}

// SetMetrics sets test metrics data.
func (m *MockStore) SetMetrics(metrics map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics = metrics
}

// MockDetector implements DependencyDetector for testing.
type MockDetector struct {
	mu                  sync.RWMutex
	scenarios           map[string]struct{}
	resources           map[string]struct{}
	detectedResources   interface{}
	detectedScenarios   interface{}
	detectedWorkflows   interface{}
	ScanResourcesErr    error
	ScanScenarioDepsErr error
	ScanWorkflowsErr    error
}

// NewMockDetector creates an empty mock detector.
func NewMockDetector() *MockDetector {
	return &MockDetector{
		scenarios: make(map[string]struct{}),
		resources: make(map[string]struct{}),
	}
}

func (m *MockDetector) RefreshCatalogs() {
	// No-op for testing
}

func (m *MockDetector) KnownScenario(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.scenarios[name]
	return ok
}

func (m *MockDetector) KnownResource(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.resources[name]
	return ok
}

func (m *MockDetector) ScenarioCatalog() map[string]struct{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]struct{})
	for k, v := range m.scenarios {
		result[k] = v
	}
	return result
}

func (m *MockDetector) ScanResources(scenarioPath, scenarioName string, cfg interface{}) (interface{}, error) {
	if m.ScanResourcesErr != nil {
		return nil, m.ScanResourcesErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.detectedResources, nil
}

func (m *MockDetector) ScanScenarioDependencies(scenarioPath, scenarioName string) (interface{}, error) {
	if m.ScanScenarioDepsErr != nil {
		return nil, m.ScanScenarioDepsErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.detectedScenarios, nil
}

func (m *MockDetector) ScanSharedWorkflows(scenarioPath, scenarioName string) (interface{}, error) {
	if m.ScanWorkflowsErr != nil {
		return nil, m.ScanWorkflowsErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.detectedWorkflows, nil
}

// AddScenario adds a known scenario to the mock detector.
func (m *MockDetector) AddScenario(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scenarios[name] = struct{}{}
}

// AddResource adds a known resource to the mock detector.
func (m *MockDetector) AddResource(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resources[name] = struct{}{}
}

// SetDetectedResources sets the resources that will be returned by ScanResources.
func (m *MockDetector) SetDetectedResources(resources interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.detectedResources = resources
}

// SetDetectedScenarios sets the scenarios that will be returned by ScanScenarioDependencies.
func (m *MockDetector) SetDetectedScenarios(scenarios interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.detectedScenarios = scenarios
}

// SetDetectedWorkflows sets the workflows that will be returned by ScanSharedWorkflows.
func (m *MockDetector) SetDetectedWorkflows(workflows interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.detectedWorkflows = workflows
}

// NewTestDependencies creates Dependencies with test doubles.
func NewTestDependencies() *Dependencies {
	return &Dependencies{
		Clock:    NewTestClock(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)),
		IDs:      NewSequentialIDGenerator("test"),
		FS:       NewMemoryFileSystem(),
		Store:    NewMockStore(),
		Detector: NewMockDetector(),
	}
}
