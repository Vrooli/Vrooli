package preflight

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChecker_CheckUIDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// No UI directory
	c := NewChecker()
	if c.CheckUIDirectory(tmpDir) {
		t.Error("CheckUIDirectory() should return false when ui dir doesn't exist")
	}

	// Create UI directory
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.Mkdir(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if !c.CheckUIDirectory(tmpDir) {
		t.Error("CheckUIDirectory() should return true when ui dir exists")
	}
}

func TestChecker_CheckIframeBridge_Present(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.Mkdir(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	packageJSON := `{
		"name": "test-ui",
		"dependencies": {
			"@vrooli/iframe-bridge": "^1.0.0"
		}
	}`

	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(packageJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker()
	status, err := c.CheckIframeBridge(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("CheckIframeBridge() error = %v", err)
	}

	if !status.DependencyPresent {
		t.Error("DependencyPresent should be true")
	}
	if status.Version != "^1.0.0" {
		t.Errorf("Version = %q, want %q", status.Version, "^1.0.0")
	}
}

func TestChecker_CheckIframeBridge_DevDependency(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.Mkdir(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	packageJSON := `{
		"name": "test-ui",
		"devDependencies": {
			"@vrooli/iframe-bridge": "file:../../../packages/iframe-bridge"
		}
	}`

	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(packageJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker()
	status, err := c.CheckIframeBridge(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("CheckIframeBridge() error = %v", err)
	}

	if !status.DependencyPresent {
		t.Error("DependencyPresent should be true for devDependencies")
	}
	if status.Version != "file:../../../packages/iframe-bridge" {
		t.Errorf("Version = %q, want %q", status.Version, "file:../../../packages/iframe-bridge")
	}
}

func TestChecker_CheckIframeBridge_Missing(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.Mkdir(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	packageJSON := `{
		"name": "test-ui",
		"dependencies": {
			"react": "^18.0.0"
		}
	}`

	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(packageJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker()
	status, err := c.CheckIframeBridge(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("CheckIframeBridge() error = %v", err)
	}

	if status.DependencyPresent {
		t.Error("DependencyPresent should be false when iframe-bridge is not listed")
	}
}

func TestChecker_CheckIframeBridge_NoPackageJSON(t *testing.T) {
	tmpDir := t.TempDir()

	c := NewChecker()
	status, err := c.CheckIframeBridge(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("CheckIframeBridge() error = %v", err)
	}

	if status.DependencyPresent {
		t.Error("DependencyPresent should be false when package.json doesn't exist")
	}
}

func TestChecker_CheckIframeBridge_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.Mkdir(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte("not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker()
	status, err := c.CheckIframeBridge(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("CheckIframeBridge() error = %v", err)
	}

	if status.DependencyPresent {
		t.Error("DependencyPresent should be false for invalid package.json")
	}
	if status.Details == "" {
		t.Error("Details should describe the parse error")
	}
}

func TestChecker_WithAppRoot(t *testing.T) {
	c := NewChecker(WithAppRoot("/custom/root"))

	if c.appRoot != "/custom/root" {
		t.Errorf("appRoot = %q, want %q", c.appRoot, "/custom/root")
	}
}

func TestParseUIPortFromLogs_ListeningOnPort(t *testing.T) {
	logs := `Starting server...
listening on port 38441
Server ready`

	port := parseUIPortFromLogs(logs)
	if port != 38441 {
		t.Errorf("parseUIPortFromLogs() = %d, want 38441", port)
	}
}

func TestParseUIPortFromLogs_UILocalhost(t *testing.T) {
	logs := `Building UI...
UI: http://localhost:3000
Ready for connections`

	port := parseUIPortFromLogs(logs)
	if port != 3000 {
		t.Errorf("parseUIPortFromLogs() = %d, want 3000", port)
	}
}

func TestParseUIPortFromLogs_ServerPort(t *testing.T) {
	logs := `Initializing...
server listening on port 8080
Accepting connections`

	port := parseUIPortFromLogs(logs)
	if port != 8080 {
		t.Errorf("parseUIPortFromLogs() = %d, want 8080", port)
	}
}

func TestParseUIPortFromLogs_MultipleMatches_ReturnsLast(t *testing.T) {
	// Simulates a restart scenario where the port appears multiple times
	logs := `Starting server...
listening on port 3000
Server crashed, restarting...
listening on port 3001
Server ready`

	port := parseUIPortFromLogs(logs)
	if port != 3001 {
		t.Errorf("parseUIPortFromLogs() = %d, want 3001 (last match)", port)
	}
}

func TestParseUIPortFromLogs_NoMatch(t *testing.T) {
	logs := `Starting application...
Loading configuration...
Ready`

	port := parseUIPortFromLogs(logs)
	if port != 0 {
		t.Errorf("parseUIPortFromLogs() = %d, want 0 (no match)", port)
	}
}

func TestParseUIPortFromLogs_EmptyString(t *testing.T) {
	port := parseUIPortFromLogs("")
	if port != 0 {
		t.Errorf("parseUIPortFromLogs() = %d, want 0 (empty input)", port)
	}
}

func TestParseUIPortFromLogs_MixedPatterns(t *testing.T) {
	// Note: The function iterates through patterns in order:
	// 1. "listening on port (\d+)" -> 6000
	// 2. "UI:\s*http://localhost:(\d+)" -> 4000
	// 3. "server.*port\s+(\d+)" -> 5000
	// The last pattern to match wins, so 5000 is returned.
	logs := `UI: http://localhost:4000
server listening on port 5000
listening on port 6000`

	port := parseUIPortFromLogs(logs)
	if port != 5000 {
		t.Errorf("parseUIPortFromLogs() = %d, want 5000 (last pattern match)", port)
	}
}

func TestParseUIPortFromLogs_SamePatternMultipleTimes(t *testing.T) {
	// When the same pattern matches multiple times, the last match is used
	logs := `UI: http://localhost:3000
UI: http://localhost:4000
UI: http://localhost:5000`

	port := parseUIPortFromLogs(logs)
	if port != 5000 {
		t.Errorf("parseUIPortFromLogs() = %d, want 5000 (last match of same pattern)", port)
	}
}

func TestChecker_CheckUIPortDefined_Defined(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	serviceJSON := `{
		"ports": {
			"ui": {
				"env_var": "TEST_UI_PORT",
				"description": "UI server port"
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(serviceJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker()
	result, err := c.CheckUIPortDefined(tmpDir)
	if err != nil {
		t.Fatalf("CheckUIPortDefined() error = %v", err)
	}

	if !result.Defined {
		t.Error("Defined should be true when UI port is in service.json")
	}
	if result.EnvVar != "TEST_UI_PORT" {
		t.Errorf("EnvVar = %q, want %q", result.EnvVar, "TEST_UI_PORT")
	}
	if result.Description != "UI server port" {
		t.Errorf("Description = %q, want %q", result.Description, "UI server port")
	}
}

func TestChecker_CheckUIPortDefined_NotDefined(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	serviceJSON := `{
		"ports": {
			"api": {
				"env_var": "API_PORT"
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(serviceJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker()
	result, err := c.CheckUIPortDefined(tmpDir)
	if err != nil {
		t.Fatalf("CheckUIPortDefined() error = %v", err)
	}

	if result.Defined {
		t.Error("Defined should be false when no UI port in service.json")
	}
}

func TestChecker_CheckUIPortDefined_NoServiceJSON(t *testing.T) {
	tmpDir := t.TempDir()

	c := NewChecker()
	result, err := c.CheckUIPortDefined(tmpDir)
	if err != nil {
		t.Fatalf("CheckUIPortDefined() error = %v", err)
	}

	if result.Defined {
		t.Error("Defined should be false when service.json doesn't exist")
	}
}

func TestChecker_CheckUIPortDefined_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte("not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker()
	result, err := c.CheckUIPortDefined(tmpDir)
	if err != nil {
		t.Fatalf("CheckUIPortDefined() error = %v", err)
	}

	if result.Defined {
		t.Error("Defined should be false for invalid JSON")
	}
}

func TestChecker_CheckUIPortDefined_EmptyEnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	serviceJSON := `{
		"ports": {
			"ui": {
				"env_var": "",
				"description": "UI server port"
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(serviceJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker()
	result, err := c.CheckUIPortDefined(tmpDir)
	if err != nil {
		t.Fatalf("CheckUIPortDefined() error = %v", err)
	}

	if result.Defined {
		t.Error("Defined should be false when env_var is empty")
	}
}

func TestChecker_CheckUIPortDefined_NullUI(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	serviceJSON := `{
		"ports": {
			"ui": null
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(serviceJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker()
	result, err := c.CheckUIPortDefined(tmpDir)
	if err != nil {
		t.Fatalf("CheckUIPortDefined() error = %v", err)
	}

	if result.Defined {
		t.Error("Defined should be false when UI is null")
	}
}

// mockCommandExecutor allows testing CheckUIPort without real CLI calls.
type mockCommandExecutor struct {
	responses map[string]mockResponse
}

type mockResponse struct {
	output []byte
	err    error
}

func newMockCommandExecutor() *mockCommandExecutor {
	return &mockCommandExecutor{
		responses: make(map[string]mockResponse),
	}
}

func (m *mockCommandExecutor) Execute(ctx context.Context, name string, args ...string) ([]byte, error) {
	// Build a key from the command and args
	key := name + " " + strings.Join(args, " ")
	if resp, ok := m.responses[key]; ok {
		return resp.output, resp.err
	}
	// Default: command not found
	return nil, fmt.Errorf("command not found: %s", key)
}

func (m *mockCommandExecutor) SetResponse(cmd string, output []byte, err error) {
	m.responses[cmd] = mockResponse{output: output, err: err}
}

func TestChecker_CheckUIPort_Method1_DirectPort(t *testing.T) {
	executor := newMockCommandExecutor()
	executor.SetResponse("vrooli scenario port test-scenario UI_PORT", []byte("3000\n"), nil)

	validator := newMockPortValidator()
	validator.SetListening(3000, true)

	c := NewChecker(WithCommandExecutor(executor), WithPortValidator(validator))
	port, err := c.CheckUIPort(context.Background(), "test-scenario")
	if err != nil {
		t.Fatalf("CheckUIPort() error = %v", err)
	}

	if port != 3000 {
		t.Errorf("port = %d, want 3000", port)
	}
}

func TestChecker_CheckUIPort_Method2_AllPorts(t *testing.T) {
	executor := newMockCommandExecutor()
	// Method 1 fails
	executor.SetResponse("vrooli scenario port test-scenario UI_PORT", nil, fmt.Errorf("not found"))
	// Method 2 succeeds
	executor.SetResponse("vrooli scenario port test-scenario", []byte("API_PORT=8080\nUI_PORT=3001\nDB_PORT=5432\n"), nil)

	validator := newMockPortValidator()
	validator.SetListening(3001, true)

	c := NewChecker(WithCommandExecutor(executor), WithPortValidator(validator))
	port, err := c.CheckUIPort(context.Background(), "test-scenario")
	if err != nil {
		t.Fatalf("CheckUIPort() error = %v", err)
	}

	if port != 3001 {
		t.Errorf("port = %d, want 3001", port)
	}
}

func TestChecker_CheckUIPort_Method3_LogParsing(t *testing.T) {
	executor := newMockCommandExecutor()
	// Method 1 fails
	executor.SetResponse("vrooli scenario port test-scenario UI_PORT", nil, fmt.Errorf("not found"))
	// Method 2 fails (no UI_PORT in output)
	executor.SetResponse("vrooli scenario port test-scenario", []byte("API_PORT=8080\n"), nil)
	// Method 3 succeeds via log parsing
	executor.SetResponse("vrooli scenario logs test-scenario --step start-ui --lines 50",
		[]byte("Starting UI server...\nlistening on port 3002\nReady\n"), nil)

	validator := newMockPortValidator()
	validator.SetListening(3002, true)

	c := NewChecker(WithCommandExecutor(executor), WithPortValidator(validator))
	port, err := c.CheckUIPort(context.Background(), "test-scenario")
	if err != nil {
		t.Fatalf("CheckUIPort() error = %v", err)
	}

	if port != 3002 {
		t.Errorf("port = %d, want 3002", port)
	}
}

func TestChecker_CheckUIPort_AllMethodsFail(t *testing.T) {
	executor := newMockCommandExecutor()
	// All methods fail
	executor.SetResponse("vrooli scenario port test-scenario UI_PORT", nil, fmt.Errorf("not found"))
	executor.SetResponse("vrooli scenario port test-scenario", []byte("API_PORT=8080\n"), nil)
	executor.SetResponse("vrooli scenario logs test-scenario --step start-ui --lines 50",
		[]byte("Starting UI server...\nNo port info\n"), nil)

	c := NewChecker(WithCommandExecutor(executor))
	port, err := c.CheckUIPort(context.Background(), "test-scenario")
	if err != nil {
		t.Fatalf("CheckUIPort() error = %v", err)
	}

	if port != 0 {
		t.Errorf("port = %d, want 0 (not found)", port)
	}
}

func TestChecker_CheckUIPort_InvalidPortOutput(t *testing.T) {
	executor := newMockCommandExecutor()
	// Method 1 returns invalid (non-numeric) output
	executor.SetResponse("vrooli scenario port test-scenario UI_PORT", []byte("not-a-port\n"), nil)
	// Method 2 fails
	executor.SetResponse("vrooli scenario port test-scenario", nil, fmt.Errorf("error"))
	// Method 3 fails
	executor.SetResponse("vrooli scenario logs test-scenario --step start-ui --lines 50", nil, fmt.Errorf("error"))

	c := NewChecker(WithCommandExecutor(executor))
	port, err := c.CheckUIPort(context.Background(), "test-scenario")
	if err != nil {
		t.Fatalf("CheckUIPort() error = %v", err)
	}

	if port != 0 {
		t.Errorf("port = %d, want 0 (invalid output)", port)
	}
}

func TestChecker_CheckUIPort_ZeroPort(t *testing.T) {
	executor := newMockCommandExecutor()
	// Method 1 returns zero
	executor.SetResponse("vrooli scenario port test-scenario UI_PORT", []byte("0\n"), nil)
	// Method 2 has UI_PORT=0
	executor.SetResponse("vrooli scenario port test-scenario", []byte("UI_PORT=0\n"), nil)
	// Method 3 fails
	executor.SetResponse("vrooli scenario logs test-scenario --step start-ui --lines 50", nil, fmt.Errorf("error"))

	c := NewChecker(WithCommandExecutor(executor))
	port, err := c.CheckUIPort(context.Background(), "test-scenario")
	if err != nil {
		t.Fatalf("CheckUIPort() error = %v", err)
	}

	// Zero port should be treated as not found
	if port != 0 {
		t.Errorf("port = %d, want 0 (zero is invalid)", port)
	}
}

func TestChecker_CheckUIPort_Method1Preferred(t *testing.T) {
	executor := newMockCommandExecutor()
	// Method 1 returns 3000
	executor.SetResponse("vrooli scenario port test-scenario UI_PORT", []byte("3000\n"), nil)
	// Method 2 would return 4000 (but shouldn't be called)
	executor.SetResponse("vrooli scenario port test-scenario", []byte("UI_PORT=4000\n"), nil)
	// Method 3 would return 5000 (but shouldn't be called)
	executor.SetResponse("vrooli scenario logs test-scenario --step start-ui --lines 50",
		[]byte("listening on port 5000\n"), nil)

	validator := newMockPortValidator()
	validator.SetListening(3000, true)

	c := NewChecker(WithCommandExecutor(executor), WithPortValidator(validator))
	port, err := c.CheckUIPort(context.Background(), "test-scenario")
	if err != nil {
		t.Fatalf("CheckUIPort() error = %v", err)
	}

	// Should use Method 1's result (3000), not Method 2 or 3
	if port != 3000 {
		t.Errorf("port = %d, want 3000 (Method 1 should be preferred)", port)
	}
}

func TestChecker_CheckUIPort_WhitespaceHandling(t *testing.T) {
	executor := newMockCommandExecutor()
	// Output with extra whitespace
	executor.SetResponse("vrooli scenario port test-scenario UI_PORT", []byte("  3000  \n"), nil)

	validator := newMockPortValidator()
	validator.SetListening(3000, true)

	c := NewChecker(WithCommandExecutor(executor), WithPortValidator(validator))
	port, err := c.CheckUIPort(context.Background(), "test-scenario")
	if err != nil {
		t.Fatalf("CheckUIPort() error = %v", err)
	}

	if port != 3000 {
		t.Errorf("port = %d, want 3000 (whitespace should be trimmed)", port)
	}
}

func TestChecker_WithCommandExecutor(t *testing.T) {
	executor := newMockCommandExecutor()
	c := NewChecker(WithCommandExecutor(executor))

	if c.cmdExecutor != executor {
		t.Error("cmdExecutor should be set to custom executor")
	}
}

// mockPortValidator allows testing port validation without real network calls.
type mockPortValidator struct {
	listeningPorts map[int]bool
	err            error
}

func newMockPortValidator() *mockPortValidator {
	return &mockPortValidator{
		listeningPorts: make(map[int]bool),
	}
}

func (m *mockPortValidator) ValidateListening(port int) error {
	if m.err != nil {
		return m.err
	}
	if m.listeningPorts[port] {
		return nil
	}
	return fmt.Errorf("connection refused on localhost:%d", port)
}

func (m *mockPortValidator) SetListening(port int, listening bool) {
	m.listeningPorts[port] = listening
}

func TestChecker_WithPortValidator(t *testing.T) {
	validator := newMockPortValidator()
	c := NewChecker(WithPortValidator(validator))

	if c.portValidator != validator {
		t.Error("portValidator should be set to custom validator")
	}
}

func TestChecker_CheckUIPort_PortListening(t *testing.T) {
	executor := newMockCommandExecutor()
	executor.SetResponse("vrooli scenario port test-scenario UI_PORT", []byte("3000\n"), nil)

	validator := newMockPortValidator()
	validator.SetListening(3000, true)

	c := NewChecker(
		WithCommandExecutor(executor),
		WithPortValidator(validator))

	port, err := c.CheckUIPort(context.Background(), "test-scenario")
	if err != nil {
		t.Fatalf("CheckUIPort() error = %v", err)
	}

	if port != 3000 {
		t.Errorf("port = %d, want 3000", port)
	}
}

func TestChecker_CheckUIPort_PortNotListening(t *testing.T) {
	executor := newMockCommandExecutor()
	executor.SetResponse("vrooli scenario port test-scenario UI_PORT", []byte("3000\n"), nil)

	validator := newMockPortValidator()
	// Don't set port 3000 as listening - it will fail validation

	c := NewChecker(
		WithCommandExecutor(executor),
		WithPortValidator(validator))

	port, err := c.CheckUIPort(context.Background(), "test-scenario")
	if err == nil {
		t.Error("CheckUIPort() should return error when port is not listening")
	}

	if port != 0 {
		t.Errorf("port = %d, want 0 when validation fails", port)
	}

	// Error message should be informative
	if !strings.Contains(err.Error(), "3000") {
		t.Errorf("error should mention port 3000, got: %v", err)
	}
	if !strings.Contains(err.Error(), "not listening") {
		t.Errorf("error should mention 'not listening', got: %v", err)
	}
}

func TestChecker_CheckUIPort_Method2WithValidation(t *testing.T) {
	executor := newMockCommandExecutor()
	executor.SetResponse("vrooli scenario port test-scenario UI_PORT", nil, fmt.Errorf("not found"))
	executor.SetResponse("vrooli scenario port test-scenario", []byte("UI_PORT=4000\n"), nil)

	validator := newMockPortValidator()
	validator.SetListening(4000, true)

	c := NewChecker(
		WithCommandExecutor(executor),
		WithPortValidator(validator))

	port, err := c.CheckUIPort(context.Background(), "test-scenario")
	if err != nil {
		t.Fatalf("CheckUIPort() error = %v", err)
	}

	if port != 4000 {
		t.Errorf("port = %d, want 4000", port)
	}
}

func TestChecker_CheckUIPort_Method3WithValidation(t *testing.T) {
	executor := newMockCommandExecutor()
	executor.SetResponse("vrooli scenario port test-scenario UI_PORT", nil, fmt.Errorf("not found"))
	executor.SetResponse("vrooli scenario port test-scenario", []byte("API_PORT=8080\n"), nil)
	executor.SetResponse("vrooli scenario logs test-scenario --step start-ui --lines 50",
		[]byte("listening on port 5000\n"), nil)

	validator := newMockPortValidator()
	validator.SetListening(5000, true)

	c := NewChecker(
		WithCommandExecutor(executor),
		WithPortValidator(validator))

	port, err := c.CheckUIPort(context.Background(), "test-scenario")
	if err != nil {
		t.Fatalf("CheckUIPort() error = %v", err)
	}

	if port != 5000 {
		t.Errorf("port = %d, want 5000", port)
	}
}

func TestChecker_CheckUIPort_DiscoveryMethodInError(t *testing.T) {
	tests := []struct {
		name          string
		setupExecutor func(*mockCommandExecutor)
		wantMethod    string
	}{
		{
			name: "method1_direct",
			setupExecutor: func(e *mockCommandExecutor) {
				e.SetResponse("vrooli scenario port test-scenario UI_PORT", []byte("3000\n"), nil)
			},
			wantMethod: "vrooli scenario port (direct)",
		},
		{
			name: "method2_parsed",
			setupExecutor: func(e *mockCommandExecutor) {
				e.SetResponse("vrooli scenario port test-scenario UI_PORT", nil, fmt.Errorf("not found"))
				e.SetResponse("vrooli scenario port test-scenario", []byte("UI_PORT=3000\n"), nil)
			},
			wantMethod: "vrooli scenario port (parsed)",
		},
		{
			name: "method3_logs",
			setupExecutor: func(e *mockCommandExecutor) {
				e.SetResponse("vrooli scenario port test-scenario UI_PORT", nil, fmt.Errorf("not found"))
				e.SetResponse("vrooli scenario port test-scenario", []byte("API_PORT=8080\n"), nil)
				e.SetResponse("vrooli scenario logs test-scenario --step start-ui --lines 50",
					[]byte("listening on port 3000\n"), nil)
			},
			wantMethod: "log parsing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := newMockCommandExecutor()
			tt.setupExecutor(executor)

			validator := newMockPortValidator()
			// Port not listening - will fail validation

			c := NewChecker(
				WithCommandExecutor(executor),
				WithPortValidator(validator))

			_, err := c.CheckUIPort(context.Background(), "test-scenario")
			if err == nil {
				t.Error("CheckUIPort() should return error when port is not listening")
			}

			if !strings.Contains(err.Error(), tt.wantMethod) {
				t.Errorf("error should mention discovery method %q, got: %v", tt.wantMethod, err)
			}
		})
	}
}

func TestChecker_CheckUIDirectory_FileNotDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file named "ui" instead of a directory
	uiFile := filepath.Join(tmpDir, "ui")
	if err := os.WriteFile(uiFile, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker()
	if c.CheckUIDirectory(tmpDir) {
		t.Error("CheckUIDirectory() should return false when ui is a file, not a directory")
	}
}

func TestChecker_CheckIframeBridge_BothDepsAndDevDeps(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.Mkdir(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Bridge in both deps and devDeps - should prefer deps
	packageJSON := `{
		"name": "test-ui",
		"dependencies": {
			"@vrooli/iframe-bridge": "^2.0.0"
		},
		"devDependencies": {
			"@vrooli/iframe-bridge": "^1.0.0"
		}
	}`

	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(packageJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker()
	status, err := c.CheckIframeBridge(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("CheckIframeBridge() error = %v", err)
	}

	if !status.DependencyPresent {
		t.Error("DependencyPresent should be true")
	}
	// Should prefer the version from dependencies (^2.0.0)
	if status.Version != "^2.0.0" {
		t.Errorf("Version = %q, want %q (should prefer dependencies over devDependencies)", status.Version, "^2.0.0")
	}
}

func TestParseUIPortFromLogs_InvalidPort(t *testing.T) {
	// Port that's not a valid number
	logs := `listening on port abc`

	port := parseUIPortFromLogs(logs)
	if port != 0 {
		t.Errorf("parseUIPortFromLogs() = %d, want 0 (invalid port)", port)
	}
}

func TestParseUIPortFromLogs_NegativePort(t *testing.T) {
	// Negative port should not match (regex only matches \d+)
	logs := `listening on port -1`

	port := parseUIPortFromLogs(logs)
	// The regex \d+ won't match negative numbers, so it should return 0
	if port != 0 {
		t.Errorf("parseUIPortFromLogs() = %d, want 0 (negative port)", port)
	}
}

func TestChecker_CheckUIPortDefined_EmptyPortsSection(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	serviceJSON := `{
		"ports": {}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(serviceJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker()
	result, err := c.CheckUIPortDefined(tmpDir)
	if err != nil {
		t.Fatalf("CheckUIPortDefined() error = %v", err)
	}

	if result.Defined {
		t.Error("Defined should be false when ports section is empty")
	}
}
