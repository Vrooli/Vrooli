package smoke

import (
	"fmt"
	"time"
)

// DefaultTimeout is the overall smoke test timeout.
const DefaultTimeout = 90 * time.Second

// DefaultHandshakeTimeout is the maximum wait for iframe-bridge handshake.
const DefaultHandshakeTimeout = 15 * time.Second

// DefaultViewportWidth is the default browser viewport width.
const DefaultViewportWidth = 1280

// DefaultViewportHeight is the default browser viewport height.
const DefaultViewportHeight = 720

// Config holds configuration for a UI smoke test run.
type Config struct {
	// ScenarioName is the name of the scenario being tested.
	ScenarioName string

	// ScenarioDir is the absolute path to the scenario directory.
	ScenarioDir string

	// BrowserlessURL is the base URL for the Browserless service.
	BrowserlessURL string

	// Timeout is the overall timeout for the smoke test.
	Timeout time.Duration

	// HandshakeTimeout is the maximum time to wait for iframe-bridge handshake.
	HandshakeTimeout time.Duration

	// Viewport defines the browser viewport dimensions.
	Viewport Viewport

	// UIURL is the URL where the scenario's UI is served.
	// If empty, it will be constructed from discovered port information.
	UIURL string

	// UIPort is the port where the scenario's UI is served.
	// Used when UIURL is not explicitly set.
	UIPort int
}

// Viewport defines browser viewport dimensions.
type Viewport struct {
	Width  int
	Height int
}

// DefaultViewport returns the standard viewport for smoke tests.
func DefaultViewport() Viewport {
	return Viewport{
		Width:  DefaultViewportWidth,
		Height: DefaultViewportHeight,
	}
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Timeout:          DefaultTimeout,
		HandshakeTimeout: DefaultHandshakeTimeout,
		Viewport:         DefaultViewport(),
	}
}

// Validate checks that the configuration is complete and valid.
func (c *Config) Validate() error {
	if c.ScenarioName == "" {
		return fmt.Errorf("scenario name is required")
	}
	if c.ScenarioDir == "" {
		return fmt.Errorf("scenario directory is required")
	}
	if c.BrowserlessURL == "" {
		return fmt.Errorf("browserless URL is required")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if c.HandshakeTimeout <= 0 {
		return fmt.Errorf("handshake timeout must be positive")
	}
	if c.Viewport.Width <= 0 || c.Viewport.Height <= 0 {
		return fmt.Errorf("viewport dimensions must be positive")
	}
	return nil
}

// TimeoutMs returns the timeout in milliseconds.
func (c *Config) TimeoutMs() int64 {
	return c.Timeout.Milliseconds()
}

// HandshakeTimeoutMs returns the handshake timeout in milliseconds.
func (c *Config) HandshakeTimeoutMs() int64 {
	return c.HandshakeTimeout.Milliseconds()
}

// ResolveUIURL returns the UI URL to test.
// If UIURL is set, it returns that. Otherwise constructs from UIPort.
func (c *Config) ResolveUIURL() string {
	if c.UIURL != "" {
		return c.UIURL
	}
	if c.UIPort > 0 {
		return fmt.Sprintf("http://localhost:%d", c.UIPort)
	}
	return ""
}
