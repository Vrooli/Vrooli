// Package mocks provides test mocks for the smoketest package.
package mocks

import (
	"context"
	"os"
	"time"

	"scenario-to-desktop-api/screenrecording"
	"scenario-to-desktop-api/smoketest"
)

// MockStore implements smoketest.Store for testing.
type MockStore struct {
	Statuses map[string]*smoketest.Status

	// SaveFunc allows custom save logic
	SaveFunc func(status *smoketest.Status)

	// GetFunc allows custom get logic
	GetFunc func(id string) (*smoketest.Status, bool)

	// UpdateFunc allows custom update logic
	UpdateFunc func(id string, fn func(status *smoketest.Status)) bool

	// UpdateCalls records all Update calls
	UpdateCalls []UpdateCall
}

// UpdateCall records an update invocation.
type UpdateCall struct {
	ID string
}

// NewMockStore creates a new mock store.
func NewMockStore() *MockStore {
	return &MockStore{
		Statuses:    make(map[string]*smoketest.Status),
		UpdateCalls: []UpdateCall{},
	}
}

// AddStatus adds a status to the mock store.
func (m *MockStore) AddStatus(status *smoketest.Status) *MockStore {
	m.Statuses[status.SmokeTestID] = status
	return m
}

// Save inserts or replaces a smoke test status.
func (m *MockStore) Save(status *smoketest.Status) {
	if m.SaveFunc != nil {
		m.SaveFunc(status)
		return
	}
	m.Statuses[status.SmokeTestID] = status
}

// Get returns the status for the given smoke test ID if it exists.
func (m *MockStore) Get(id string) (*smoketest.Status, bool) {
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	status, ok := m.Statuses[id]
	return status, ok
}

// Update executes fn while holding a write lock on the requested smoke test.
func (m *MockStore) Update(id string, fn func(status *smoketest.Status)) bool {
	m.UpdateCalls = append(m.UpdateCalls, UpdateCall{ID: id})
	if m.UpdateFunc != nil {
		return m.UpdateFunc(id, fn)
	}
	status, ok := m.Statuses[id]
	if !ok {
		return false
	}
	fn(status)
	return true
}

// MockCancelManager implements smoketest.CancelManager for testing.
type MockCancelManager struct {
	Cancels map[string]context.CancelFunc

	// ClearCalls records all Clear calls
	ClearCalls []string
}

// NewMockCancelManager creates a new mock cancel manager.
func NewMockCancelManager() *MockCancelManager {
	return &MockCancelManager{
		Cancels:    make(map[string]context.CancelFunc),
		ClearCalls: []string{},
	}
}

// SetCancel registers a cancellation function for a smoke test.
func (m *MockCancelManager) SetCancel(id string, cancel context.CancelFunc) {
	m.Cancels[id] = cancel
}

// TakeCancel retrieves and removes the cancellation function for a smoke test.
func (m *MockCancelManager) TakeCancel(id string) context.CancelFunc {
	cancel, ok := m.Cancels[id]
	if !ok {
		return nil
	}
	delete(m.Cancels, id)
	return cancel
}

// Clear removes the cancellation function without calling it.
func (m *MockCancelManager) Clear(id string) {
	m.ClearCalls = append(m.ClearCalls, id)
	delete(m.Cancels, id)
}

// MockTelemetryIngestor implements smoketest.TelemetryIngestor for testing.
type MockTelemetryIngestor struct {
	// IngestResult is returned by IngestEvents
	IngestResult struct {
		ID    string
		Count int
		Err   error
	}

	// IngestCalls records all IngestEvents calls
	IngestCalls []IngestCall
}

// IngestCall records an ingest invocation.
type IngestCall struct {
	ScenarioName string
	InstanceID   string
	Source       string
	Events       []map[string]interface{}
}

// NewMockTelemetryIngestor creates a new mock telemetry ingestor.
func NewMockTelemetryIngestor() *MockTelemetryIngestor {
	return &MockTelemetryIngestor{}
}

// IngestEvents ingests telemetry events from a smoke test.
func (m *MockTelemetryIngestor) IngestEvents(scenarioName, instanceID, source string, events []map[string]interface{}) (string, int, error) {
	m.IngestCalls = append(m.IngestCalls, IngestCall{
		ScenarioName: scenarioName,
		InstanceID:   instanceID,
		Source:       source,
		Events:       events,
	})
	return m.IngestResult.ID, m.IngestResult.Count, m.IngestResult.Err
}

// MockLogger implements smoketest.Logger for testing.
type MockLogger struct {
	InfoCalls  []LogCall
	WarnCalls  []LogCall
	ErrorCalls []LogCall
}

// LogCall records a logging invocation.
type LogCall struct {
	Msg  string
	Args []interface{}
}

// NewMockLogger creates a new mock logger.
func NewMockLogger() *MockLogger {
	return &MockLogger{
		InfoCalls:  []LogCall{},
		WarnCalls:  []LogCall{},
		ErrorCalls: []LogCall{},
	}
}

// Info logs an info message.
func (m *MockLogger) Info(msg string, args ...interface{}) {
	m.InfoCalls = append(m.InfoCalls, LogCall{Msg: msg, Args: args})
}

// Warn logs a warning message.
func (m *MockLogger) Warn(msg string, args ...interface{}) {
	m.WarnCalls = append(m.WarnCalls, LogCall{Msg: msg, Args: args})
}

// Error logs an error message.
func (m *MockLogger) Error(msg string, args ...interface{}) {
	m.ErrorCalls = append(m.ErrorCalls, LogCall{Msg: msg, Args: args})
}

// MockProcessExecutor implements smoketest.ProcessExecutor for testing.
type MockProcessExecutor struct {
	// ExecuteResult is the default result for Execute
	ExecuteResult struct {
		Output string
		Err    error
	}

	// ExecuteWithResultResult is the default result for ExecuteWithResult
	ExecuteWithResultResult struct {
		Result *smoketest.ExecutionResult
		Err    error
	}

	// ExecuteFunc allows custom execute logic
	ExecuteFunc func(ctx context.Context, workDir, command string, args, env []string, timeout time.Duration) (string, error)

	// ExecuteWithResultFunc allows custom execute logic returning ExecutionResult
	ExecuteWithResultFunc func(ctx context.Context, workDir, command string, args, env []string, timeout time.Duration) (*smoketest.ExecutionResult, error)

	// LookPathResults maps command names to their paths
	LookPathResults map[string]string

	// LookPathErrors maps command names to errors
	LookPathErrors map[string]error

	// LookPathFunc allows custom lookup logic
	LookPathFunc func(name string) (string, error)

	// ExecuteCalls records all Execute calls
	ExecuteCalls []ExecuteCall
}

// ExecuteCall records an execute invocation.
type ExecuteCall struct {
	WorkDir string
	Command string
	Args    []string
	Env     []string
	Timeout time.Duration
}

// NewMockProcessExecutor creates a new mock process executor.
func NewMockProcessExecutor() *MockProcessExecutor {
	return &MockProcessExecutor{
		LookPathResults: make(map[string]string),
		LookPathErrors:  make(map[string]error),
		ExecuteCalls:    []ExecuteCall{},
	}
}

// Execute runs a command and returns combined stdout/stderr output.
func (m *MockProcessExecutor) Execute(ctx context.Context, workDir, command string, args, env []string, timeout time.Duration) (string, error) {
	m.ExecuteCalls = append(m.ExecuteCalls, ExecuteCall{
		WorkDir: workDir,
		Command: command,
		Args:    args,
		Env:     env,
		Timeout: timeout,
	})
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, workDir, command, args, env, timeout)
	}
	return m.ExecuteResult.Output, m.ExecuteResult.Err
}

// ExecuteWithResult runs a command and returns detailed execution result.
func (m *MockProcessExecutor) ExecuteWithResult(ctx context.Context, workDir, command string, args, env []string, timeout time.Duration) (*smoketest.ExecutionResult, error) {
	m.ExecuteCalls = append(m.ExecuteCalls, ExecuteCall{
		WorkDir: workDir,
		Command: command,
		Args:    args,
		Env:     env,
		Timeout: timeout,
	})
	if m.ExecuteWithResultFunc != nil {
		return m.ExecuteWithResultFunc(ctx, workDir, command, args, env, timeout)
	}
	if m.ExecuteWithResultResult.Result != nil {
		return m.ExecuteWithResultResult.Result, m.ExecuteWithResultResult.Err
	}
	// Fallback to old behavior for backward compatibility
	return &smoketest.ExecutionResult{
		Stdout:   m.ExecuteResult.Output,
		Stderr:   "",
		Combined: m.ExecuteResult.Output,
		ExitCode: 0,
	}, m.ExecuteResult.Err
}

// LookPath searches for an executable in the system PATH.
func (m *MockProcessExecutor) LookPath(name string) (string, error) {
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

// AddLookPath adds a lookup path result.
func (m *MockProcessExecutor) AddLookPath(name, path string) *MockProcessExecutor {
	m.LookPathResults[name] = path
	return m
}

// AddLookPathError adds a lookup path error.
func (m *MockProcessExecutor) AddLookPathError(name string, err error) *MockProcessExecutor {
	m.LookPathErrors[name] = err
	return m
}

// MockPlatformResolver implements smoketest.PlatformResolver for testing.
type MockPlatformResolver struct {
	// Platform is the current platform to return
	Platform string

	// PlatformOverride is set by SetPlatformOverride
	PlatformOverride string

	// ResolveResult is the result for ResolveCommand
	ResolveResult struct {
		Cmd     string
		Args    []string
		Display string
		Err     error
	}

	// ResolveFunc allows custom resolve logic
	ResolveFunc func(platform, artifactPath string) (string, []string, string, error)

	// HeadlessResult is the result for RequiresHeadlessWrapper
	HeadlessResult struct {
		Needed      bool
		WrapperCmd  string
		WrapperArgs []string
		Err         error
	}
}

// NewMockPlatformResolver creates a new mock platform resolver.
func NewMockPlatformResolver() *MockPlatformResolver {
	return &MockPlatformResolver{
		Platform: "linux",
	}
}

// SetPlatformOverride sets the platform override for testing.
func (m *MockPlatformResolver) SetPlatformOverride(platform string) {
	m.PlatformOverride = platform
}

// CurrentPlatform returns the current platform identifier.
func (m *MockPlatformResolver) CurrentPlatform() string {
	return m.Platform
}

// ResolveCommand determines the command, args, and display string for running a smoke test.
func (m *MockPlatformResolver) ResolveCommand(platform, artifactPath string) (string, []string, string, error) {
	if m.ResolveFunc != nil {
		return m.ResolveFunc(platform, artifactPath)
	}
	return m.ResolveResult.Cmd, m.ResolveResult.Args, m.ResolveResult.Display, m.ResolveResult.Err
}

// RequiresHeadlessWrapper checks if a headless wrapper is needed.
func (m *MockPlatformResolver) RequiresHeadlessWrapper() (bool, string, []string, error) {
	return m.HeadlessResult.Needed, m.HeadlessResult.WrapperCmd, m.HeadlessResult.WrapperArgs, m.HeadlessResult.Err
}

// MockTelemetryPathResolver implements smoketest.TelemetryPathResolver for testing.
type MockTelemetryPathResolver struct {
	// ExtractResult is returned by ExtractFromOutput
	ExtractResult string

	// ExtractFunc allows custom extract logic
	ExtractFunc func(output string) string

	// ResolveResult is returned by ResolveFromArtifact
	ResolveResult string

	// ResolveFunc allows custom resolve logic
	ResolveFunc func(platform, artifactPath, scenarioName string) string

	// ReadEventsResult is returned by ReadTelemetryEvents
	ReadEventsResult struct {
		Events []map[string]interface{}
		Err    error
	}

	// ReadEventsFunc allows custom read logic
	ReadEventsFunc func(path string, limit int) ([]map[string]interface{}, error)
}

// NewMockTelemetryPathResolver creates a new mock telemetry path resolver.
func NewMockTelemetryPathResolver() *MockTelemetryPathResolver {
	return &MockTelemetryPathResolver{}
}

// ExtractFromOutput attempts to extract the telemetry path from smoke test output.
func (m *MockTelemetryPathResolver) ExtractFromOutput(output string) string {
	if m.ExtractFunc != nil {
		return m.ExtractFunc(output)
	}
	return m.ExtractResult
}

// ResolveFromArtifact attempts to resolve the telemetry path based on platform and artifact.
func (m *MockTelemetryPathResolver) ResolveFromArtifact(platform, artifactPath, scenarioName string) string {
	if m.ResolveFunc != nil {
		return m.ResolveFunc(platform, artifactPath, scenarioName)
	}
	return m.ResolveResult
}

// ReadTelemetryEvents reads telemetry events from the given path.
func (m *MockTelemetryPathResolver) ReadTelemetryEvents(path string, limit int) ([]map[string]interface{}, error) {
	if m.ReadEventsFunc != nil {
		return m.ReadEventsFunc(path, limit)
	}
	return m.ReadEventsResult.Events, m.ReadEventsResult.Err
}

// MockOutputParser implements smoketest.OutputParser for testing.
type MockOutputParser struct {
	// ParseResult is returned by ParseResult
	Result smoketest.OutputResult

	// ParseFunc allows custom parse logic
	ParseFunc func(output string) smoketest.OutputResult

	// SequenceResult is returned by ValidateSequence
	SequenceResult smoketest.SequenceValidation

	// SequenceFunc allows custom sequence validation logic
	SequenceFunc func(output string) smoketest.SequenceValidation

	// AppErrorResult is returned by ExtractAppError
	AppErrorResult *smoketest.AppError

	// AppErrorFunc allows custom app error extraction logic
	AppErrorFunc func(output string) *smoketest.AppError

	// LifecycleStateResult is returned by ExtractLastLifecycleState
	LifecycleStateResult string

	// LifecycleStateFunc allows custom lifecycle state extraction logic
	LifecycleStateFunc func(output string) string

	// SessionIDResult is returned by ExtractSessionID
	SessionIDResult string

	// SessionIDFunc allows custom session ID extraction logic
	SessionIDFunc func(output string) string
}

// NewMockOutputParser creates a new mock output parser.
func NewMockOutputParser() *MockOutputParser {
	return &MockOutputParser{}
}

// ParseResult analyzes smoke test output and returns the result.
func (m *MockOutputParser) ParseResult(output string) smoketest.OutputResult {
	if m.ParseFunc != nil {
		return m.ParseFunc(output)
	}
	return m.Result
}

// ValidateSequence validates the smoke test output sequence.
func (m *MockOutputParser) ValidateSequence(output string) smoketest.SequenceValidation {
	if m.SequenceFunc != nil {
		return m.SequenceFunc(output)
	}
	return m.SequenceResult
}

// ExtractAppError parses SMOKE_TEST_ERROR markers from output.
func (m *MockOutputParser) ExtractAppError(output string) *smoketest.AppError {
	if m.AppErrorFunc != nil {
		return m.AppErrorFunc(output)
	}
	return m.AppErrorResult
}

// ExtractLastLifecycleState returns the last lifecycle marker reached.
func (m *MockOutputParser) ExtractLastLifecycleState(output string) string {
	if m.LifecycleStateFunc != nil {
		return m.LifecycleStateFunc(output)
	}
	return m.LifecycleStateResult
}

// ExtractSessionID parses the session ID from the SMOKE_TEST_INIT marker.
func (m *MockOutputParser) ExtractSessionID(output string) string {
	if m.SessionIDFunc != nil {
		return m.SessionIDFunc(output)
	}
	return m.SessionIDResult
}

// MockEnvironmentReader implements smoketest.EnvironmentReader for testing.
type MockEnvironmentReader struct {
	// Vars maps environment variable names to values
	Vars map[string]string

	// HomeDir is returned by UserHomeDir
	HomeDir string

	// HomeDirErr is returned by UserHomeDir
	HomeDirErr error
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

// GetEnv retrieves the value of an environment variable.
func (m *MockEnvironmentReader) GetEnv(key string) string {
	return m.Vars[key]
}

// UserHomeDir returns the current user's home directory.
func (m *MockEnvironmentReader) UserHomeDir() (string, error) {
	if m.HomeDirErr != nil {
		return "", m.HomeDirErr
	}
	return m.HomeDir, nil
}

// MockFileSystem implements smoketest.FileSystem for testing.
type MockFileSystem struct {
	// Files maps paths to their contents
	Files map[string][]byte

	// Directories tracks which paths are directories
	Directories map[string]bool

	// FileInfos maps paths to their file info
	FileInfos map[string]os.FileInfo

	// DirEntries maps paths to their directory entries
	DirEntries map[string][]os.DirEntry

	// StatFunc allows custom stat logic
	StatFunc func(path string) (os.FileInfo, error)

	// ReadDirFunc allows custom read dir logic
	ReadDirFunc func(path string) ([]os.DirEntry, error)

	// OpenFunc allows custom open logic
	OpenFunc func(path string) (*os.File, error)

	// ChmodFunc allows custom chmod logic
	ChmodFunc func(path string, mode os.FileMode) error

	// ChmodCalls records all Chmod calls
	ChmodCalls []ChmodCall
}

// ChmodCall records a chmod invocation.
type ChmodCall struct {
	Path string
	Mode os.FileMode
}

// NewMockFileSystem creates a new mock file system.
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		Files:       make(map[string][]byte),
		Directories: make(map[string]bool),
		FileInfos:   make(map[string]os.FileInfo),
		DirEntries:  make(map[string][]os.DirEntry),
		ChmodCalls:  []ChmodCall{},
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

// AddFileInfo adds file info for a path.
func (m *MockFileSystem) AddFileInfo(path string, info os.FileInfo) *MockFileSystem {
	m.FileInfos[path] = info
	return m
}

// AddDirEntries adds directory entries for a path.
func (m *MockFileSystem) AddDirEntries(path string, entries []os.DirEntry) *MockFileSystem {
	m.DirEntries[path] = entries
	return m
}

// Stat returns file info for the given path.
func (m *MockFileSystem) Stat(path string) (os.FileInfo, error) {
	if m.StatFunc != nil {
		return m.StatFunc(path)
	}
	if info, ok := m.FileInfos[path]; ok {
		return info, nil
	}
	if _, ok := m.Files[path]; ok {
		return &MockFileInfo{NameVal: path, SizeVal: int64(len(m.Files[path]))}, nil
	}
	if _, ok := m.Directories[path]; ok {
		return &MockFileInfo{NameVal: path, IsDirVal: true}, nil
	}
	return nil, os.ErrNotExist
}

// ReadDir reads the contents of a directory.
func (m *MockFileSystem) ReadDir(path string) ([]os.DirEntry, error) {
	if m.ReadDirFunc != nil {
		return m.ReadDirFunc(path)
	}
	if entries, ok := m.DirEntries[path]; ok {
		return entries, nil
	}
	return nil, os.ErrNotExist
}

// Open opens a file for reading.
func (m *MockFileSystem) Open(path string) (*os.File, error) {
	if m.OpenFunc != nil {
		return m.OpenFunc(path)
	}
	// For testing, we can't easily mock os.File, so this returns an error by default
	return nil, os.ErrNotExist
}

// Chmod changes the mode of the named file.
func (m *MockFileSystem) Chmod(path string, mode os.FileMode) error {
	m.ChmodCalls = append(m.ChmodCalls, ChmodCall{Path: path, Mode: mode})
	if m.ChmodFunc != nil {
		return m.ChmodFunc(path, mode)
	}
	return nil
}

// ReadFile reads a file and returns its contents.
func (m *MockFileSystem) ReadFile(path string) ([]byte, error) {
	if content, ok := m.Files[path]; ok {
		return content, nil
	}
	return nil, os.ErrNotExist
}

// MockFileInfo implements os.FileInfo for testing.
type MockFileInfo struct {
	NameVal    string
	SizeVal    int64
	ModeVal    os.FileMode
	ModTimeVal time.Time
	IsDirVal   bool
}

func (f *MockFileInfo) Name() string       { return f.NameVal }
func (f *MockFileInfo) Size() int64        { return f.SizeVal }
func (f *MockFileInfo) Mode() os.FileMode  { return f.ModeVal }
func (f *MockFileInfo) ModTime() time.Time { return f.ModTimeVal }
func (f *MockFileInfo) IsDir() bool        { return f.IsDirVal }
func (f *MockFileInfo) Sys() interface{}   { return nil }

// MockDirEntry implements os.DirEntry for testing.
type MockDirEntry struct {
	EntryName  string
	EntryIsDir bool
	EntryType  os.FileMode
	EntryInfo  os.FileInfo
	InfoErr    error
}

func (e *MockDirEntry) Name() string               { return e.EntryName }
func (e *MockDirEntry) IsDir() bool                { return e.EntryIsDir }
func (e *MockDirEntry) Type() os.FileMode          { return e.EntryType }
func (e *MockDirEntry) Info() (os.FileInfo, error) { return e.EntryInfo, e.InfoErr }

// MockTelemetryChainExecutor implements smoketest.TelemetryChainExecutor for testing.
type MockTelemetryChainExecutor struct {
	// ExecuteResult is returned by Execute
	ExecuteResult smoketest.TelemetryResult

	// ExecuteFunc allows custom execute logic
	ExecuteFunc func(ctx context.Context, params smoketest.TelemetryChainParams) smoketest.TelemetryResult

	// ExecuteCalls records all Execute calls
	ExecuteCalls []smoketest.TelemetryChainParams
}

// NewMockTelemetryChainExecutor creates a new mock telemetry chain executor.
func NewMockTelemetryChainExecutor() *MockTelemetryChainExecutor {
	return &MockTelemetryChainExecutor{
		ExecuteCalls: []smoketest.TelemetryChainParams{},
	}
}

// Execute runs the telemetry collection chain.
func (m *MockTelemetryChainExecutor) Execute(ctx context.Context, params smoketest.TelemetryChainParams) smoketest.TelemetryResult {
	m.ExecuteCalls = append(m.ExecuteCalls, params)
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, params)
	}
	return m.ExecuteResult
}

// MockPrerequisiteChecker implements smoketest.PrerequisiteCheckerI for testing.
type MockPrerequisiteChecker struct {
	// CheckAllResult is returned by CheckAll
	CheckAllResult []smoketest.PrerequisiteResult

	// CheckAllFunc allows custom check logic
	CheckAllFunc func(artifactPath, platform string, telemetryPort int) []smoketest.PrerequisiteResult

	// HasFatalFailureResult is returned by HasFatalFailure
	HasFatalFailureResult bool

	// HasFatalFailureFunc allows custom logic
	HasFatalFailureFunc func(results []smoketest.PrerequisiteResult) bool

	// CheckAllCalls records all CheckAll calls
	CheckAllCalls []CheckAllCall
}

// CheckAllCall records a CheckAll invocation.
type CheckAllCall struct {
	ArtifactPath  string
	Platform      string
	TelemetryPort int
}

// NewMockPrerequisiteChecker creates a new mock prerequisite checker.
func NewMockPrerequisiteChecker() *MockPrerequisiteChecker {
	return &MockPrerequisiteChecker{
		CheckAllResult: []smoketest.PrerequisiteResult{},
		CheckAllCalls:  []CheckAllCall{},
	}
}

// CheckAll runs all prerequisite checks.
func (m *MockPrerequisiteChecker) CheckAll(artifactPath, platform string, telemetryPort int) []smoketest.PrerequisiteResult {
	m.CheckAllCalls = append(m.CheckAllCalls, CheckAllCall{
		ArtifactPath:  artifactPath,
		Platform:      platform,
		TelemetryPort: telemetryPort,
	})
	if m.CheckAllFunc != nil {
		return m.CheckAllFunc(artifactPath, platform, telemetryPort)
	}
	return m.CheckAllResult
}

// HasFatalFailure checks if any result has a fatal failure.
func (m *MockPrerequisiteChecker) HasFatalFailure(results []smoketest.PrerequisiteResult) bool {
	if m.HasFatalFailureFunc != nil {
		return m.HasFatalFailureFunc(results)
	}
	return m.HasFatalFailureResult
}

// MockClock implements smoketest.Clock for testing.
type MockClock struct {
	// CurrentTime is the time returned by Now()
	CurrentTime time.Time

	// AfterFunc allows custom After logic
	AfterFunc func(d time.Duration) <-chan time.Time

	// NowCalls records the number of Now() calls
	NowCalls int

	// AfterCalls records all After calls
	AfterCalls []time.Duration
}

// NewMockClock creates a new mock clock with the given time.
func NewMockClock(t time.Time) *MockClock {
	return &MockClock{
		CurrentTime: t,
		AfterCalls:  []time.Duration{},
	}
}

// Now returns the mock current time.
func (m *MockClock) Now() time.Time {
	m.NowCalls++
	return m.CurrentTime
}

// After returns a channel that receives the current time after the duration.
func (m *MockClock) After(d time.Duration) <-chan time.Time {
	m.AfterCalls = append(m.AfterCalls, d)
	if m.AfterFunc != nil {
		return m.AfterFunc(d)
	}
	// Return immediately for tests
	ch := make(chan time.Time, 1)
	ch <- m.CurrentTime
	return ch
}

// Advance moves the mock clock forward by the given duration.
func (m *MockClock) Advance(d time.Duration) {
	m.CurrentTime = m.CurrentTime.Add(d)
}

// Set sets the mock clock to a specific time.
func (m *MockClock) Set(t time.Time) {
	m.CurrentTime = t
}

// MockTelemetryErrorExtractor implements smoketest.TelemetryErrorExtractor for testing.
type MockTelemetryErrorExtractor struct {
	// ExtractErrorsResult is returned by ExtractErrors
	ExtractErrorsResult struct {
		Errors []smoketest.TelemetryError
		Err    error
	}

	// ExtractErrorsFunc allows custom extract logic
	ExtractErrorsFunc func(telemetryPath string, limit int) ([]smoketest.TelemetryError, error)

	// ExtractLatestErrorResult is returned by ExtractLatestError
	ExtractLatestErrorResult struct {
		Error *smoketest.TelemetryError
		Err   error
	}

	// ExtractLatestErrorFunc allows custom extract logic
	ExtractLatestErrorFunc func(telemetryPath string) (*smoketest.TelemetryError, error)

	// ExtractLatestErrorForSessionResult is returned by ExtractLatestErrorForSession
	ExtractLatestErrorForSessionResult struct {
		Error *smoketest.TelemetryError
		Err   error
	}

	// ExtractLatestErrorForSessionFunc allows custom extract logic
	ExtractLatestErrorForSessionFunc func(telemetryPath, sessionID string) (*smoketest.TelemetryError, error)

	// ExtractErrorsCalls records all ExtractErrors calls
	ExtractErrorsCalls []ExtractErrorsCall

	// ExtractLatestErrorCalls records all ExtractLatestError calls
	ExtractLatestErrorCalls []string

	// ExtractLatestErrorForSessionCalls records all ExtractLatestErrorForSession calls
	ExtractLatestErrorForSessionCalls []ExtractLatestErrorForSessionCall
}

// ExtractErrorsCall records an ExtractErrors invocation.
type ExtractErrorsCall struct {
	TelemetryPath string
	Limit         int
}

// ExtractLatestErrorForSessionCall records an ExtractLatestErrorForSession invocation.
type ExtractLatestErrorForSessionCall struct {
	TelemetryPath string
	SessionID     string
}

// NewMockTelemetryErrorExtractor creates a new mock telemetry error extractor.
func NewMockTelemetryErrorExtractor() *MockTelemetryErrorExtractor {
	return &MockTelemetryErrorExtractor{
		ExtractErrorsCalls:                []ExtractErrorsCall{},
		ExtractLatestErrorCalls:           []string{},
		ExtractLatestErrorForSessionCalls: []ExtractLatestErrorForSessionCall{},
	}
}

// ExtractErrors reads a telemetry file and extracts any error events.
func (m *MockTelemetryErrorExtractor) ExtractErrors(telemetryPath string, limit int) ([]smoketest.TelemetryError, error) {
	m.ExtractErrorsCalls = append(m.ExtractErrorsCalls, ExtractErrorsCall{
		TelemetryPath: telemetryPath,
		Limit:         limit,
	})
	if m.ExtractErrorsFunc != nil {
		return m.ExtractErrorsFunc(telemetryPath, limit)
	}
	return m.ExtractErrorsResult.Errors, m.ExtractErrorsResult.Err
}

// ExtractLatestError returns the most recent error from a telemetry file.
func (m *MockTelemetryErrorExtractor) ExtractLatestError(telemetryPath string) (*smoketest.TelemetryError, error) {
	m.ExtractLatestErrorCalls = append(m.ExtractLatestErrorCalls, telemetryPath)
	if m.ExtractLatestErrorFunc != nil {
		return m.ExtractLatestErrorFunc(telemetryPath)
	}
	return m.ExtractLatestErrorResult.Error, m.ExtractLatestErrorResult.Err
}

// ExtractLatestErrorForSession returns the most recent error matching the given session ID.
func (m *MockTelemetryErrorExtractor) ExtractLatestErrorForSession(telemetryPath, sessionID string) (*smoketest.TelemetryError, error) {
	m.ExtractLatestErrorForSessionCalls = append(m.ExtractLatestErrorForSessionCalls, ExtractLatestErrorForSessionCall{
		TelemetryPath: telemetryPath,
		SessionID:     sessionID,
	})
	if m.ExtractLatestErrorForSessionFunc != nil {
		return m.ExtractLatestErrorForSessionFunc(telemetryPath, sessionID)
	}
	return m.ExtractLatestErrorForSessionResult.Error, m.ExtractLatestErrorForSessionResult.Err
}

// WithLatestError configures the mock to return a specific error.
func (m *MockTelemetryErrorExtractor) WithLatestError(err *smoketest.TelemetryError) *MockTelemetryErrorExtractor {
	m.ExtractLatestErrorResult.Error = err
	return m
}

// WithLatestErrorForSession configures the mock to return a specific error for session-filtered extraction.
func (m *MockTelemetryErrorExtractor) WithLatestErrorForSession(err *smoketest.TelemetryError) *MockTelemetryErrorExtractor {
	m.ExtractLatestErrorForSessionResult.Error = err
	return m
}

// MockRecorder implements screenrecording.Recorder for testing.
type MockRecorder struct {
	StartResult struct {
		CaptureID string
		Err       error
	}
	StopResult struct {
		VideoPath     string
		DurationMs    int64
		FileSizeBytes int64
		Err           error
	}
	StartCalls []MockStartCaptureCall
	StopCalls  []string
}

// MockStartCaptureCall records a StartCapture invocation.
type MockStartCaptureCall struct {
	Display string
	Width   int
	Height  int
	FPS     int
}

// NewMockRecorder creates a new mock recorder.
func NewMockRecorder() *MockRecorder {
	return &MockRecorder{
		StartCalls: []MockStartCaptureCall{},
		StopCalls:  []string{},
	}
}

// StartCapture records the call and returns the configured result.
func (m *MockRecorder) StartCapture(_ context.Context, cfg screenrecording.CaptureConfig) (string, error) {
	m.StartCalls = append(m.StartCalls, MockStartCaptureCall{
		Display: cfg.Display,
		Width:   cfg.Width,
		Height:  cfg.Height,
		FPS:     cfg.FPS,
	})
	return m.StartResult.CaptureID, m.StartResult.Err
}

// StopCapture records the call and returns the configured result.
func (m *MockRecorder) StopCapture(_ context.Context, captureID string) (*screenrecording.CaptureResult, error) {
	m.StopCalls = append(m.StopCalls, captureID)
	if m.StopResult.Err != nil {
		return nil, m.StopResult.Err
	}
	return &screenrecording.CaptureResult{
		VideoPath:     m.StopResult.VideoPath,
		DurationMs:    m.StopResult.DurationMs,
		FileSizeBytes: m.StopResult.FileSizeBytes,
	}, nil
}

// MockDisplayManager implements screenrecording.DisplayManager for testing.
type MockDisplayManager struct {
	CreateResult struct {
		DisplayID string
		Err       error
	}
	CreateCalls []MockCreateDisplayCall
	CleanupCalled bool
}

// MockCreateDisplayCall records a CreateDisplay invocation.
type MockCreateDisplayCall struct {
	Width  int
	Height int
}

// NewMockDisplayManager creates a new mock display manager.
func NewMockDisplayManager() *MockDisplayManager {
	return &MockDisplayManager{
		CreateCalls: []MockCreateDisplayCall{},
	}
}

// CreateDisplay records the call and returns the configured result.
func (m *MockDisplayManager) CreateDisplay(width, height int) (string, func(), error) {
	m.CreateCalls = append(m.CreateCalls, MockCreateDisplayCall{Width: width, Height: height})
	if m.CreateResult.Err != nil {
		return "", nil, m.CreateResult.Err
	}
	displayID := m.CreateResult.DisplayID
	if displayID == "" {
		displayID = ":99"
	}
	return displayID, func() { m.CleanupCalled = true }, nil
}
