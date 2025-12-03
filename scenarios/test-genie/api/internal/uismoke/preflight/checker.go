package preflight

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"test-genie/internal/uismoke/orchestrator"
)

// Checker validates preconditions for UI smoke tests.
type Checker struct {
	browserlessURL string
	httpClient     *http.Client
	appRoot        string
}

// NewChecker creates a new preflight Checker.
func NewChecker(browserlessURL string, opts ...CheckerOption) *Checker {
	c := &Checker{
		browserlessURL: strings.TrimSuffix(browserlessURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
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
		pattern := filepath.Join(scenarioDir, glob)
		matches, err := filepath.Glob(pattern)
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
func (c *Checker) CheckUIPort(ctx context.Context, scenarioName string) (int, error) {
	// Method 1: Try vrooli scenario port command
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "port", scenarioName, "UI_PORT")
	output, err := cmd.Output()
	if err == nil {
		var port int
		if _, err := fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &port); err == nil && port > 0 {
			return port, nil
		}
	}

	// Method 2: Try to get all ports and look for UI_PORT
	cmd = exec.CommandContext(ctx, "vrooli", "scenario", "port", scenarioName)
	output, err = cmd.Output()
	if err == nil {
		for _, line := range strings.Split(string(output), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "UI_PORT=") {
				var port int
				if _, err := fmt.Sscanf(line, "UI_PORT=%d", &port); err == nil && port > 0 {
					return port, nil
				}
			}
		}
	}

	// Method 3: Try to parse the scenario logs for UI port
	cmd = exec.CommandContext(ctx, "vrooli", "scenario", "logs", scenarioName, "--step", "start-ui", "--lines", "50")
	output, err = cmd.Output()
	if err == nil {
		port := parseUIPortFromLogs(string(output))
		if port > 0 {
			return port, nil
		}
	}

	return 0, nil
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
