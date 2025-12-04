package preflight

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"test-genie/internal/structure/smoke/orchestrator"
)

// CommandExecutor abstracts command execution for testing.
type CommandExecutor interface {
	// Execute runs a command and returns its output.
	Execute(ctx context.Context, name string, args ...string) ([]byte, error)
}

// PortValidator abstracts port validation for testing.
type PortValidator interface {
	// ValidateListening checks if a port is accepting connections.
	ValidateListening(port int) error
}

// defaultExecutor uses exec.CommandContext for real command execution.
type defaultExecutor struct{}

func (d defaultExecutor) Execute(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

// defaultPortValidator uses net.DialTimeout for real port validation.
type defaultPortValidator struct{}

func (d defaultPortValidator) ValidateListening(port int) error {
	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return fmt.Errorf("connection refused on %s", addr)
	}
	conn.Close()
	return nil
}

// Checker validates preconditions for UI smoke tests.
type Checker struct {
	browserlessURL string
	httpClient     *http.Client
	appRoot        string
	cmdExecutor    CommandExecutor
	portValidator  PortValidator
}

// NewChecker creates a new preflight Checker.
func NewChecker(browserlessURL string, opts ...CheckerOption) *Checker {
	c := &Checker{
		browserlessURL: strings.TrimSuffix(browserlessURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cmdExecutor:   defaultExecutor{},
		portValidator: defaultPortValidator{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// CheckerOption configures a Checker.
type CheckerOption func(*Checker)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) CheckerOption {
	return func(c *Checker) {
		c.httpClient = hc
	}
}

// WithAppRoot sets the application root directory.
func WithAppRoot(appRoot string) CheckerOption {
	return func(c *Checker) {
		c.appRoot = appRoot
	}
}

// WithCommandExecutor sets a custom command executor (for testing).
func WithCommandExecutor(executor CommandExecutor) CheckerOption {
	return func(c *Checker) {
		c.cmdExecutor = executor
	}
}

// WithPortValidator sets a custom port validator (for testing).
func WithPortValidator(validator PortValidator) CheckerOption {
	return func(c *Checker) {
		c.portValidator = validator
	}
}

// Ensure Checker implements orchestrator.PreflightChecker.
var _ orchestrator.PreflightChecker = (*Checker)(nil)

// CheckBrowserless verifies the Browserless service is available and healthy.
func (c *Checker) CheckBrowserless(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.browserlessURL+"/pressure", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("browserless unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("browserless returned status %d", resp.StatusCode)
	}

	return nil
}

// CheckBundleFreshness verifies the UI bundle is up-to-date.
func (c *Checker) CheckBundleFreshness(ctx context.Context, scenarioDir string) (*orchestrator.BundleStatus, error) {
	// Check if ui/dist/index.html exists
	distIndex := filepath.Join(scenarioDir, "ui", "dist", "index.html")
	if _, err := os.Stat(distIndex); os.IsNotExist(err) {
		return &orchestrator.BundleStatus{
			Fresh:  false,
			Reason: "UI bundle missing (ui/dist/index.html not found)",
		}, nil
	}

	// Load service.json to check for ui-bundle configuration
	serviceJSON := filepath.Join(scenarioDir, ".vrooli", "service.json")
	config, err := loadBundleConfig(serviceJSON)
	if err != nil {
		// If we can't load config, assume bundle is fresh if dist exists
		return &orchestrator.BundleStatus{Fresh: true}, nil
	}

	// Check if bundle needs rebuild based on source file timestamps
	if config != nil {
		fresh, reason := c.checkBundleTimestamps(scenarioDir, config)
		return &orchestrator.BundleStatus{
			Fresh:  fresh,
			Reason: reason,
			Config: config.Raw,
		}, nil
	}

	return &orchestrator.BundleStatus{Fresh: true}, nil
}

// bundleConfig holds ui-bundle check configuration from service.json.
type bundleConfig struct {
	Type        string          `json:"type"`
	SourceGlobs []string        `json:"source_globs"`
	DistPath    string          `json:"dist_path"`
	Raw         json.RawMessage `json:"-"`
}

func loadBundleConfig(serviceJSONPath string) (*bundleConfig, error) {
	data, err := os.ReadFile(serviceJSONPath)
	if err != nil {
		return nil, err
	}

	var manifest struct {
		Lifecycle struct {
			Setup struct {
				Condition struct {
					Checks []json.RawMessage `json:"checks"`
				} `json:"condition"`
			} `json:"setup"`
		} `json:"lifecycle"`
	}

	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	for _, checkRaw := range manifest.Lifecycle.Setup.Condition.Checks {
		var check bundleConfig
		if err := json.Unmarshal(checkRaw, &check); err != nil {
			continue
		}
		if check.Type == "ui-bundle" {
			check.Raw = checkRaw
			return &check, nil
		}
	}

	return nil, nil
}

func (c *Checker) checkBundleTimestamps(scenarioDir string, config *bundleConfig) (bool, string) {
	distPath := config.DistPath
	if distPath == "" {
		distPath = "ui/dist"
	}

	distDir := filepath.Join(scenarioDir, distPath)
	distInfo, err := os.Stat(distDir)
	if err != nil {
		return false, fmt.Sprintf("dist directory not found: %s", distPath)
	}

	distModTime := distInfo.ModTime()

	// Check source files against dist
	sourceGlobs := config.SourceGlobs
	if len(sourceGlobs) == 0 {
		sourceGlobs = []string{"ui/src/**/*", "ui/package.json"}
	}

	for _, glob := range sourceGlobs {
		matches, err := expandGlob(scenarioDir, glob)
		if err != nil {
			continue
		}

		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				continue
			}
			if info.ModTime().After(distModTime) {
				rel, _ := filepath.Rel(scenarioDir, match)
				return false, fmt.Sprintf("Source file newer than bundle: %s", rel)
			}
		}
	}

	return true, ""
}

// expandGlob expands a glob pattern that may contain ** for recursive matching.
// Unlike filepath.Glob, this properly supports ** to match any number of directories.
func expandGlob(baseDir, pattern string) ([]string, error) {
	// Check if pattern contains **
	if !strings.Contains(pattern, "**") {
		// Use standard filepath.Glob for non-recursive patterns
		fullPattern := filepath.Join(baseDir, pattern)
		return filepath.Glob(fullPattern)
	}

	// Split pattern at **
	parts := strings.SplitN(pattern, "**", 2)
	prefix := strings.TrimSuffix(parts[0], string(filepath.Separator))
	suffix := ""
	if len(parts) > 1 {
		suffix = strings.TrimPrefix(parts[1], string(filepath.Separator))
	}

	// Start directory for walking
	startDir := filepath.Join(baseDir, prefix)
	if _, err := os.Stat(startDir); os.IsNotExist(err) {
		return nil, nil // Directory doesn't exist, return empty matches
	}

	var matches []string
	err := filepath.WalkDir(startDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip files/dirs we can't access
		}

		// Skip directories themselves, only match files
		if d.IsDir() {
			return nil
		}

		// If there's a suffix pattern, check if the filename matches
		if suffix != "" {
			// Get the path relative to startDir
			relPath, err := filepath.Rel(startDir, path)
			if err != nil {
				return nil
			}

			// Check if the relative path matches the suffix pattern
			// For patterns like **/*.ts, suffix is "*.ts"
			// We need to check if the filename matches
			matched, err := filepath.Match(suffix, filepath.Base(relPath))
			if err != nil || !matched {
				// Also try matching the full relative path for patterns like **/foo/bar.ts
				matched, err = filepath.Match(suffix, relPath)
				if err != nil || !matched {
					return nil
				}
			}
		}

		matches = append(matches, path)
		return nil
	})

	return matches, err
}

// CheckIframeBridge verifies @vrooli/iframe-bridge is installed.
func (c *Checker) CheckIframeBridge(ctx context.Context, scenarioDir string) (*orchestrator.BridgeStatus, error) {
	packageJSON := filepath.Join(scenarioDir, "ui", "package.json")

	data, err := os.ReadFile(packageJSON)
	if err != nil {
		return &orchestrator.BridgeStatus{
			DependencyPresent: false,
			Details:           "ui/package.json not found",
		}, nil
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return &orchestrator.BridgeStatus{
			DependencyPresent: false,
			Details:           fmt.Sprintf("failed to parse package.json: %v", err),
		}, nil
	}

	version := pkg.Dependencies["@vrooli/iframe-bridge"]
	if version == "" {
		version = pkg.DevDependencies["@vrooli/iframe-bridge"]
	}

	if version == "" {
		return &orchestrator.BridgeStatus{
			DependencyPresent: false,
			Details:           "@vrooli/iframe-bridge not listed in dependencies",
		}, nil
	}

	return &orchestrator.BridgeStatus{
		DependencyPresent: true,
		Version:           version,
	}, nil
}

// CheckUIPort discovers and returns the UI port for the scenario.
// Returns an error if a port was discovered but is not listening.
// Returns (0, nil) if no port could be discovered at all.
func (c *Checker) CheckUIPort(ctx context.Context, scenarioName string) (int, error) {
	var discoveredPort int
	var discoveryMethod string

	// Method 1: Try vrooli scenario port command with specific port name
	output, err := c.cmdExecutor.Execute(ctx, "vrooli", "scenario", "port", scenarioName, "UI_PORT")
	if err == nil {
		var port int
		if _, err := fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &port); err == nil && port > 0 {
			discoveredPort = port
			discoveryMethod = "vrooli scenario port (direct)"
		}
	}

	// Method 2: Try to get all ports and look for UI_PORT
	if discoveredPort == 0 {
		output, err = c.cmdExecutor.Execute(ctx, "vrooli", "scenario", "port", scenarioName)
		if err == nil {
			for _, line := range strings.Split(string(output), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "UI_PORT=") {
					var port int
					if _, err := fmt.Sscanf(line, "UI_PORT=%d", &port); err == nil && port > 0 {
						discoveredPort = port
						discoveryMethod = "vrooli scenario port (parsed)"
						break
					}
				}
			}
		}
	}

	// Method 3: Try to parse the scenario logs for UI port
	if discoveredPort == 0 {
		output, err = c.cmdExecutor.Execute(ctx, "vrooli", "scenario", "logs", scenarioName, "--step", "start-ui", "--lines", "50")
		if err == nil {
			port := parseUIPortFromLogs(string(output))
			if port > 0 {
				discoveredPort = port
				discoveryMethod = "log parsing"
			}
		}
	}

	// If we discovered a port, validate it's actually listening
	if discoveredPort > 0 {
		if err := c.validatePortListening(discoveredPort); err != nil {
			return 0, fmt.Errorf("port %d discovered via %s but not listening: %w", discoveredPort, discoveryMethod, err)
		}
		return discoveredPort, nil
	}

	return 0, nil
}

// validatePortListening checks if a port is actually accepting connections.
func (c *Checker) validatePortListening(port int) error {
	return c.portValidator.ValidateListening(port)
}

// parseUIPortFromLogs looks for common UI port patterns in log output.
// Returns the most recently mentioned port (last match) since logs may contain restarts.
func parseUIPortFromLogs(logs string) int {
	// Common patterns:
	// "listening on port 38441"
	// "UI: http://localhost:38441"
	// "server listening on port 38441"
	patterns := []string{
		`listening on port (\d+)`,
		`UI:\s*http://localhost:(\d+)`,
		`server.*port\s+(\d+)`,
	}

	var lastPort int
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		// Find ALL matches and use the last one (most recent)
		allMatches := re.FindAllStringSubmatch(logs, -1)
		for _, matches := range allMatches {
			if len(matches) >= 2 {
				var port int
				if _, err := fmt.Sscanf(matches[1], "%d", &port); err == nil && port > 0 {
					lastPort = port
				}
			}
		}
	}
	return lastPort
}

// CheckUIPortDefined checks if the scenario's service.json defines a UI port.
func (c *Checker) CheckUIPortDefined(scenarioDir string) (*orchestrator.UIPortDefinition, error) {
	serviceJSON := filepath.Join(scenarioDir, ".vrooli", "service.json")
	data, err := os.ReadFile(serviceJSON)
	if err != nil {
		return &orchestrator.UIPortDefinition{Defined: false}, nil
	}

	var manifest struct {
		Ports struct {
			UI *struct {
				EnvVar      string `json:"env_var"`
				Description string `json:"description"`
			} `json:"ui"`
		} `json:"ports"`
	}

	if err := json.Unmarshal(data, &manifest); err != nil {
		return &orchestrator.UIPortDefinition{Defined: false}, nil
	}

	if manifest.Ports.UI != nil && manifest.Ports.UI.EnvVar != "" {
		return &orchestrator.UIPortDefinition{
			Defined:     true,
			EnvVar:      manifest.Ports.UI.EnvVar,
			Description: manifest.Ports.UI.Description,
		}, nil
	}

	return &orchestrator.UIPortDefinition{Defined: false}, nil
}

// CheckUIDirectory returns true if the scenario has a UI directory.
func (c *Checker) CheckUIDirectory(scenarioDir string) bool {
	uiDir := filepath.Join(scenarioDir, "ui")
	info, err := os.Stat(uiDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}
