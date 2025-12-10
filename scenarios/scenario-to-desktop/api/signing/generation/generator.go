// Package generation creates build tool configurations from signing settings.
//
// This package transforms signing configuration into platform-specific
// build configurations such as electron-builder JSON, macOS entitlements.plist,
// and notarization afterSign scripts.
package generation

import (
	"scenario-to-desktop-api/signing/types"
)

// Generator creates build tool configurations from signing settings.
type Generator interface {
	// GenerateElectronBuilder creates electron-builder signing config from a SigningConfig.
	// Returns nil if signing is disabled.
	GenerateElectronBuilder(config *types.SigningConfig) (*types.ElectronBuilderSigningConfig, error)

	// GenerateEntitlements creates macOS entitlements.plist content.
	// The capabilities parameter allows requesting specific entitlements beyond the defaults.
	// Returns nil if macOS signing is not configured.
	GenerateEntitlements(config *types.MacOSSigningConfig, capabilities []string) ([]byte, error)

	// GenerateNotarizeScript creates the afterSign JavaScript for electron-builder.
	// This script handles notarization after the app is signed.
	// Returns nil if notarization is not enabled.
	GenerateNotarizeScript(config *types.MacOSSigningConfig) ([]byte, error)

	// GenerateAll creates all signing-related files and returns them as a map.
	// Keys are relative file paths, values are file contents.
	// This is useful for writing all files at once during bundle export.
	GenerateAll(config *types.SigningConfig) (map[string][]byte, error)
}

// EnvironmentReader abstracts environment variable access.
type EnvironmentReader interface {
	// GetEnv retrieves an environment variable value.
	GetEnv(key string) string

	// LookupEnv retrieves an environment variable and reports if it exists.
	LookupEnv(key string) (string, bool)
}

// Options configures the generator behavior.
type Options struct {
	// OutputDir is the base directory for generated file paths.
	// Default: "build"
	OutputDir string

	// EntitlementsPath is the relative path for the entitlements file.
	// Default: "entitlements.mac.plist"
	EntitlementsPath string

	// NotarizeScriptPath is the relative path for the notarize script.
	// Default: "scripts/notarize.js"
	NotarizeScriptPath string

	// EnvironmentResolver resolves environment variable references.
	// If nil, environment variables are kept as ${VAR} references.
	EnvironmentResolver EnvironmentReader
}

// DefaultOptions returns the default generator options.
func DefaultOptions() *Options {
	return &Options{
		OutputDir:          "build",
		EntitlementsPath:   "entitlements.mac.plist",
		NotarizeScriptPath: "scripts/notarize.js",
	}
}

// DefaultGenerator implements the Generator interface.
type DefaultGenerator struct {
	opts *Options
}

// NewGenerator creates a new Generator with the given options.
// If opts is nil, DefaultOptions() is used.
func NewGenerator(opts *Options) Generator {
	if opts == nil {
		opts = DefaultOptions()
	}
	return &DefaultGenerator{opts: opts}
}

// GenerateElectronBuilder creates electron-builder signing config from a SigningConfig.
func (g *DefaultGenerator) GenerateElectronBuilder(config *types.SigningConfig) (*types.ElectronBuilderSigningConfig, error) {
	if config == nil || !config.Enabled {
		return nil, nil
	}

	result := &types.ElectronBuilderSigningConfig{}

	// Generate Windows config
	if config.Windows != nil {
		result.Win = generateWindowsConfig(config.Windows, g.opts)
	}

	// Generate macOS config
	if config.MacOS != nil {
		result.Mac = generateMacOSConfig(config.MacOS, g.opts)
	}

	return result, nil
}

// GenerateEntitlements creates macOS entitlements.plist content.
func (g *DefaultGenerator) GenerateEntitlements(config *types.MacOSSigningConfig, capabilities []string) ([]byte, error) {
	if config == nil {
		return nil, nil
	}

	return generateEntitlementsPlist(config, capabilities)
}

// GenerateNotarizeScript creates the afterSign JavaScript for electron-builder.
func (g *DefaultGenerator) GenerateNotarizeScript(config *types.MacOSSigningConfig) ([]byte, error) {
	if config == nil || !config.Notarize {
		return nil, nil
	}

	return generateNotarizeJS(config)
}

// GenerateAll creates all signing-related files and returns them as a map.
func (g *DefaultGenerator) GenerateAll(config *types.SigningConfig) (map[string][]byte, error) {
	if config == nil || !config.Enabled {
		return nil, nil
	}

	files := make(map[string][]byte)

	// Generate macOS files if configured
	if config.MacOS != nil {
		// Generate entitlements
		entitlements, err := g.GenerateEntitlements(config.MacOS, nil)
		if err != nil {
			return nil, err
		}
		if entitlements != nil {
			files[g.opts.OutputDir+"/"+g.opts.EntitlementsPath] = entitlements
		}

		// Generate notarize script if notarization is enabled
		if config.MacOS.Notarize {
			script, err := g.GenerateNotarizeScript(config.MacOS)
			if err != nil {
				return nil, err
			}
			if script != nil {
				files[g.opts.NotarizeScriptPath] = script
			}
		}
	}

	return files, nil
}
