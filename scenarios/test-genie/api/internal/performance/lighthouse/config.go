package lighthouse

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the Lighthouse audit configuration loaded from .vrooli/lighthouse.json.
type Config struct {
	// Enabled controls whether Lighthouse audits run.
	Enabled bool `json:"enabled"`

	// Pages defines the pages to audit with their thresholds.
	Pages []PageConfig `json:"pages"`

	// GlobalOptions contains global Lighthouse and Chrome settings.
	GlobalOptions *GlobalOptions `json:"global_options,omitempty"`

	// Reporting configures how audit results are reported and saved.
	Reporting *ReportingConfig `json:"reporting,omitempty"`
}

// ReportingConfig controls report generation settings.
type ReportingConfig struct {
	// Formats specifies which report formats to generate ("json", "html").
	Formats []string `json:"formats,omitempty"`

	// FailOnError determines if the phase should fail when threshold errors occur.
	// Defaults to true if not specified.
	FailOnError *bool `json:"fail_on_error,omitempty"`
}

// PageConfig defines a single page to audit.
type PageConfig struct {
	// ID is the unique identifier for this page.
	ID string `json:"id"`

	// Path is the URL path relative to the base URL (e.g., "/", "/about").
	Path string `json:"path"`

	// Label is a human-readable name for the page.
	Label string `json:"label,omitempty"`

	// Viewport specifies the device type: "desktop" or "mobile".
	Viewport string `json:"viewport,omitempty"`

	// WaitForSelector is a CSS selector to wait for before auditing.
	// NOTE: This field is currently unsupported by the Lighthouse CLI runner.
	// Use WaitForMs instead for time-based waiting.
	// Deprecated: Will be removed in a future version.
	WaitForSelector string `json:"waitForSelector,omitempty"`

	// WaitForMs is the time to wait in milliseconds before starting the audit.
	WaitForMs int `json:"waitForMs,omitempty"`

	// Thresholds defines the score thresholds for this page.
	Thresholds CategoryThresholds `json:"thresholds"`

	// Requirements lists requirement IDs that this page audit validates.
	Requirements []string `json:"requirements,omitempty"`
}

// CategoryThresholds maps Lighthouse categories to their threshold configuration.
type CategoryThresholds map[string]ThresholdLevel

// ThresholdLevel defines error and warning thresholds for a category.
type ThresholdLevel struct {
	// Error is the minimum score to pass (below this fails the audit).
	Error float64 `json:"error"`

	// Warn is the threshold for warnings (between error and warn shows a warning).
	Warn float64 `json:"warn"`
}

// GlobalOptions contains global Lighthouse and Chrome configuration.
type GlobalOptions struct {
	// Lighthouse contains Lighthouse-specific settings.
	Lighthouse *LighthouseSettings `json:"lighthouse,omitempty"`

	// ChromeFlags are additional flags passed to Chrome.
	ChromeFlags []string `json:"chrome_flags,omitempty"`

	// TimeoutMs is the maximum time for each audit in milliseconds.
	TimeoutMs int `json:"timeout_ms,omitempty"`

	// Retries is the number of times to retry a failed audit before marking it as failed.
	Retries int `json:"retries,omitempty"`
}

// LighthouseSettings contains Lighthouse runner configuration.
type LighthouseSettings struct {
	// Extends specifies the base config (e.g., "lighthouse:default").
	Extends string `json:"extends,omitempty"`

	// Settings contains Lighthouse settings.
	Settings *LighthouseRunnerSettings `json:"settings,omitempty"`
}

// LighthouseRunnerSettings contains detailed Lighthouse settings.
type LighthouseRunnerSettings struct {
	// OnlyCategories limits which categories to audit.
	OnlyCategories []string `json:"onlyCategories,omitempty"`

	// ThrottlingMethod specifies how throttling is applied.
	ThrottlingMethod string `json:"throttlingMethod,omitempty"`

	// FormFactor specifies the device form factor.
	FormFactor string `json:"formFactor,omitempty"`
}

// DefaultConfig returns a minimal default configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled: false,
		Pages:   nil,
	}
}

// LoadConfig reads Lighthouse configuration from .vrooli/lighthouse.json.
// Returns a default (disabled) config if the file doesn't exist.
func LoadConfig(scenarioDir string) (*Config, error) {
	configPath := filepath.Join(scenarioDir, ".vrooli", "lighthouse.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read lighthouse config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse lighthouse config: %w", err)
	}

	return &cfg, nil
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil // Disabled config is always valid
	}

	if len(c.Pages) == 0 {
		return errors.New("lighthouse config has no pages defined")
	}

	for i, page := range c.Pages {
		if page.ID == "" {
			return fmt.Errorf("page %d has no id", i)
		}
		if page.Path == "" {
			return fmt.Errorf("page %q has no path", page.ID)
		}
		if len(page.Thresholds) == 0 {
			return fmt.Errorf("page %q has no thresholds defined", page.ID)
		}

		// Validate viewport
		if page.Viewport != "" && page.Viewport != "desktop" && page.Viewport != "mobile" {
			return fmt.Errorf("page %q has invalid viewport %q (must be 'desktop' or 'mobile')", page.ID, page.Viewport)
		}

		// Validate waitForMs
		if page.WaitForMs < 0 {
			return fmt.Errorf("page %q has invalid waitForMs %d (must be >= 0)", page.ID, page.WaitForMs)
		}

		// Validate threshold values
		for category, threshold := range page.Thresholds {
			if threshold.Error < 0 || threshold.Error > 1 {
				return fmt.Errorf("page %q category %q: error threshold %.2f must be between 0 and 1",
					page.ID, category, threshold.Error)
			}
			if threshold.Warn < 0 || threshold.Warn > 1 {
				return fmt.Errorf("page %q category %q: warn threshold %.2f must be between 0 and 1",
					page.ID, category, threshold.Warn)
			}
			if threshold.Error > threshold.Warn {
				return fmt.Errorf("page %q category %q: error threshold (%.2f) must be <= warn threshold (%.2f)",
					page.ID, category, threshold.Error, threshold.Warn)
			}
		}
	}

	// Validate global options
	if c.GlobalOptions != nil {
		if c.GlobalOptions.Retries < 0 {
			return fmt.Errorf("retries must be >= 0, got %d", c.GlobalOptions.Retries)
		}
	}

	// Validate reporting config
	if c.Reporting != nil {
		validFormats := map[string]bool{"json": true, "html": true}
		for _, format := range c.Reporting.Formats {
			if !validFormats[format] {
				return fmt.Errorf("invalid report format %q (must be 'json' or 'html')", format)
			}
		}
	}

	return nil
}

// GetTimeout returns the configured timeout in milliseconds, or the default.
func (c *Config) GetTimeout() int {
	if c.GlobalOptions != nil && c.GlobalOptions.TimeoutMs > 0 {
		return c.GlobalOptions.TimeoutMs
	}
	return 90000 // Default 90 seconds
}

// GetCategories returns the list of categories to audit.
func (c *Config) GetCategories() []string {
	if c.GlobalOptions != nil &&
		c.GlobalOptions.Lighthouse != nil &&
		c.GlobalOptions.Lighthouse.Settings != nil &&
		len(c.GlobalOptions.Lighthouse.Settings.OnlyCategories) > 0 {
		return c.GlobalOptions.Lighthouse.Settings.OnlyCategories
	}
	// Default categories
	return []string{"performance", "accessibility", "best-practices", "seo"}
}

// GetRetries returns the configured number of retries, or 0 if not specified.
func (c *Config) GetRetries() int {
	if c.GlobalOptions != nil && c.GlobalOptions.Retries > 0 {
		return c.GlobalOptions.Retries
	}
	return 0
}

// ShouldGenerateReport returns true if the specified format should be generated.
func (c *Config) ShouldGenerateReport(format string) bool {
	if c.Reporting == nil || len(c.Reporting.Formats) == 0 {
		return false
	}
	for _, f := range c.Reporting.Formats {
		if f == format {
			return true
		}
	}
	return false
}

// ShouldFailOnError returns true if the phase should fail on threshold errors.
// Defaults to true if not explicitly set to false.
func (c *Config) ShouldFailOnError() bool {
	if c.Reporting != nil && c.Reporting.FailOnError != nil {
		return *c.Reporting.FailOnError
	}
	return true // Default behavior
}

// BuildLighthouseConfig constructs the config object for Lighthouse audits.
// This is used by the CLI runner to extract settings for command-line arguments.
func (c *Config) BuildLighthouseConfig() map[string]interface{} {
	config := map[string]interface{}{
		"extends": "lighthouse:default",
	}

	settings := map[string]interface{}{
		"onlyCategories": c.GetCategories(),
	}

	// Apply global settings if present
	if c.GlobalOptions != nil && c.GlobalOptions.Lighthouse != nil {
		if c.GlobalOptions.Lighthouse.Extends != "" {
			config["extends"] = c.GlobalOptions.Lighthouse.Extends
		}
		if c.GlobalOptions.Lighthouse.Settings != nil {
			s := c.GlobalOptions.Lighthouse.Settings
			if s.ThrottlingMethod != "" {
				settings["throttlingMethod"] = s.ThrottlingMethod
			}
			if s.FormFactor != "" {
				settings["formFactor"] = s.FormFactor
			}
		}
	}

	config["settings"] = settings
	return config
}


// BuildPageLighthouseConfig constructs the config object for a specific page,
// including page-specific settings like viewport and wait options.
func (c *Config) BuildPageLighthouseConfig(page PageConfig) map[string]interface{} {
	// Start with the base config
	config := c.BuildLighthouseConfig()

	settings, ok := config["settings"].(map[string]interface{})
	if !ok {
		settings = make(map[string]interface{})
		config["settings"] = settings
	}

	// Apply page-specific viewport as formFactor
	// If the page has a viewport and global settings don't override formFactor
	if page.Viewport != "" {
		// Check if formFactor is already set by global options
		if _, hasFormFactor := settings["formFactor"]; !hasFormFactor {
			settings["formFactor"] = page.Viewport
		}
	}

	// Apply screenEmulation for mobile viewport
	if page.Viewport == "mobile" {
		settings["screenEmulation"] = map[string]interface{}{
			"mobile":            true,
			"width":             360,
			"height":            640,
			"deviceScaleFactor": 2,
		}
	}

	return config
}

