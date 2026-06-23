// Package preflight validates the engine-agnostic preconditions for a UI smoke
// test: the scenario has a UI directory and a defined UI port, the
// @vrooli/iframe-bridge dependency is present, and the UI port is discoverable
// and listening. It is a dependency-light leaf: it knows nothing about the
// browser engine that ultimately drives the smoke capture. Bundle freshness is
// no longer its concern — the runner delegates that to the canonical
// content-hash freshness engine (`vrooli scenario freshness`).
package preflight

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// BridgeStatus describes iframe-bridge dependency status.
type BridgeStatus struct {
	// DependencyPresent indicates whether @vrooli/iframe-bridge is installed.
	DependencyPresent bool
	// Version is the installed version of iframe-bridge.
	Version string
	// Details provides additional information.
	Details string
}

// UIPortDefinition describes whether a scenario defines a UI port in service.json.
type UIPortDefinition struct {
	// Defined is true when service.json defines a UI port.
	Defined bool
	// EnvVar is the environment variable name (e.g. "UI_PORT").
	EnvVar string
	// Description is the port description from service.json.
	Description string
}

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
	appRoot       string
	cmdExecutor   CommandExecutor
	portValidator PortValidator
}

// NewChecker creates a new preflight Checker.
func NewChecker(opts ...CheckerOption) *Checker {
	c := &Checker{
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

// CheckIframeBridge verifies @vrooli/iframe-bridge is installed.
func (c *Checker) CheckIframeBridge(ctx context.Context, scenarioDir string) (*BridgeStatus, error) {
	packageJSON := filepath.Join(scenarioDir, "ui", "package.json")

	data, err := os.ReadFile(packageJSON)
	if err != nil {
		return &BridgeStatus{
			DependencyPresent: false,
			Details:           "ui/package.json not found",
		}, nil
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return &BridgeStatus{
			DependencyPresent: false,
			Details:           fmt.Sprintf("failed to parse package.json: %v", err),
		}, nil
	}

	version := pkg.Dependencies["@vrooli/iframe-bridge"]
	if version == "" {
		version = pkg.DevDependencies["@vrooli/iframe-bridge"]
	}

	if version == "" {
		return &BridgeStatus{
			DependencyPresent: false,
			Details:           "@vrooli/iframe-bridge not listed in dependencies",
		}, nil
	}

	return &BridgeStatus{
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
func (c *Checker) CheckUIPortDefined(scenarioDir string) (*UIPortDefinition, error) {
	serviceJSON := filepath.Join(scenarioDir, ".vrooli", "service.json")
	data, err := os.ReadFile(serviceJSON)
	if err != nil {
		return &UIPortDefinition{Defined: false}, nil
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
		return &UIPortDefinition{Defined: false}, nil
	}

	if manifest.Ports.UI != nil && manifest.Ports.UI.EnvVar != "" {
		return &UIPortDefinition{
			Defined:     true,
			EnvVar:      manifest.Ports.UI.EnvVar,
			Description: manifest.Ports.UI.Description,
		}, nil
	}

	return &UIPortDefinition{Defined: false}, nil
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
