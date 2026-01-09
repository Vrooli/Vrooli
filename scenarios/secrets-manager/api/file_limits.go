// Package main provides file size limits and extension utilities for the secrets-manager API.
//
// This file consolidates file-related constants that were previously scattered
// throughout the codebase as magic numbers. Having them in one place makes it
// easier to understand the rationale and maintain consistent limits.
package main

// -----------------------------------------------------------------------------
// File Size Limits
// -----------------------------------------------------------------------------
//
// These constants define maximum file sizes for different operations.
// The limits balance thoroughness with performance and memory safety.

const (
	// MaxConfigFileSize is the limit for reading small config files (10KB).
	// Used when reading .env files, secrets.conf, and similar small configs
	// where we only need to find key=value patterns.
	MaxConfigFileSize = 10 * 1024

	// MaxResourceFileScanSize is the limit for scanning resource files (50KB).
	// Used for security scanning of resource configuration files (shell scripts,
	// YAML, JSON). Resource files are typically smaller than scenario source code.
	MaxResourceFileScanSize = 50 * 1024

	// MaxSecretScanFileSize is the limit for secret discovery scanning (100KB).
	// Used by the SecretScanner when looking for secret patterns in source files.
	// Large enough to cover most config files while preventing memory issues.
	MaxSecretScanFileSize = 100 * 1024

	// MaxResourceWalkFileSize is the limit for resource directory walks (200KB).
	// Used during bulk resource scanning operations. Resource files rarely
	// exceed this size; larger files are likely binaries or data files.
	MaxResourceWalkFileSize = 200 * 1024

	// MaxScenarioWalkFileSize is the limit for scenario directory walks (500KB).
	// Scenario source files can be larger than resource files due to
	// generated code, embedded data, or comprehensive test files.
	MaxScenarioWalkFileSize = 500 * 1024
)

// -----------------------------------------------------------------------------
// File Extensions for Scanning
// -----------------------------------------------------------------------------
//
// Different scan operations target different file types based on their purpose.

// TextFileExtensions are file types that should be scanned for secrets and
// environment variable patterns. These are human-readable config and source files.
var TextFileExtensions = map[string]bool{
	".sh":         true,
	".bash":       true,
	".yml":        true,
	".yaml":       true,
	".json":       true,
	".env":        true,
	".conf":       true,
	".config":     true,
	".md":         true,
	".txt":        true,
	".go":         true,
	".js":         true,
	".ts":         true,
	".py":         true,
	".dockerfile": true,
	".sql":        true,
}

// ScenarioSourceExtensions are file types to scan in scenario directories
// for vulnerability patterns. These are the primary source code files.
var ScenarioSourceExtensions = map[string]bool{
	".go": true,
	".js": true,
	".ts": true,
	".py": true,
	".sh": true,
}

// ResourceConfigExtensions are file types to scan in resource directories.
// Resources primarily use shell scripts and configuration files.
var ResourceConfigExtensions = map[string]bool{
	".sh":   true,
	".yaml": true,
	".yml":  true,
	".json": true,
	".env":  true,
}

// IsTextFileExtension reports whether the given extension represents a
// text file that should be scanned for secrets.
func IsTextFileExtension(ext string) bool {
	return TextFileExtensions[ext]
}

// IsScenarioSourceExtension reports whether the extension is a scenario
// source file type that should be scanned for vulnerabilities.
func IsScenarioSourceExtension(ext string) bool {
	return ScenarioSourceExtensions[ext]
}

// IsResourceConfigExtension reports whether the extension is a resource
// configuration file type that should be scanned.
func IsResourceConfigExtension(ext string) bool {
	return ResourceConfigExtensions[ext]
}
