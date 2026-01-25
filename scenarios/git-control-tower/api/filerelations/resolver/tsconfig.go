// Package resolver handles resolution of import paths to actual file paths
package resolver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// TSConfigResolver resolves TypeScript path aliases from tsconfig.json
type TSConfigResolver struct {
	baseUrl   string
	paths     map[string][]string // pattern -> replacement paths
	repoRoot  string
	initOnce  bool
	initError error
}

// tsConfigJSON represents the tsconfig.json structure
type tsConfigJSON struct {
	Extends         string `json:"extends"`
	CompilerOptions struct {
		BaseUrl string              `json:"baseUrl"`
		Paths   map[string][]string `json:"paths"`
	} `json:"compilerOptions"`
}

// NewTSConfigResolver creates a new TypeScript config resolver
func NewTSConfigResolver() *TSConfigResolver {
	return &TSConfigResolver{
		paths: make(map[string][]string),
	}
}

// Init initializes the resolver by reading tsconfig.json
func (r *TSConfigResolver) Init(repoRoot string) error {
	if r.initOnce && r.repoRoot == repoRoot {
		return r.initError
	}
	r.initOnce = true
	r.repoRoot = repoRoot
	r.paths = make(map[string][]string)
	r.baseUrl = ""

	// Look for tsconfig.json in common locations
	configPaths := []string{
		"tsconfig.json",
		"jsconfig.json",
		"tsconfig.base.json",
	}

	for _, configPath := range configPaths {
		fullPath := filepath.Join(repoRoot, configPath)
		if _, err := os.Stat(fullPath); err == nil {
			if err := r.parseTSConfig(fullPath, repoRoot); err == nil {
				break
			}
		}
	}

	return r.initError
}

// parseTSConfig parses a tsconfig.json file
func (r *TSConfigResolver) parseTSConfig(configPath, repoRoot string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config tsConfigJSON
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// Handle extends
	if config.Extends != "" {
		extendsPath := config.Extends
		if !filepath.IsAbs(extendsPath) {
			extendsPath = filepath.Join(filepath.Dir(configPath), extendsPath)
		}
		// Add .json extension if not present
		if !strings.HasSuffix(extendsPath, ".json") {
			extendsPath += ".json"
		}
		// Parse the extended config first (base config) - ignore errors as extends may not exist
		_ = r.parseTSConfig(extendsPath, repoRoot)
	}

	// Set baseUrl (override from extended config)
	if config.CompilerOptions.BaseUrl != "" {
		// baseUrl is relative to the tsconfig location
		configDir := filepath.Dir(configPath)
		baseUrlAbs := filepath.Join(configDir, config.CompilerOptions.BaseUrl)
		if rel, err := filepath.Rel(repoRoot, baseUrlAbs); err == nil {
			r.baseUrl = rel
		} else {
			r.baseUrl = config.CompilerOptions.BaseUrl
		}
		// Normalize "." to empty string
		if r.baseUrl == "." {
			r.baseUrl = ""
		}
	}

	// Merge paths (override from extended config)
	for pattern, replacements := range config.CompilerOptions.Paths {
		r.paths[pattern] = replacements
	}

	return nil
}

// Resolve resolves a TypeScript path alias to a file path
func (r *TSConfigResolver) Resolve(importSource string, fromFile string, repoRoot string) string {
	// Initialize if needed (ignore errors - continue even with initialization errors)
	_ = r.Init(repoRoot)

	// Skip relative imports
	if strings.HasPrefix(importSource, ".") {
		return ""
	}

	// Try to match against path patterns
	for pattern, replacements := range r.paths {
		match, remainder := matchPattern(pattern, importSource)
		if !match {
			continue
		}

		// Try each replacement path
		for _, replacement := range replacements {
			resolved := applyReplacement(replacement, remainder)

			// Apply baseUrl
			var fullPath string
			if r.baseUrl != "" {
				fullPath = filepath.Join(r.baseUrl, resolved)
			} else {
				fullPath = resolved
			}

			// Try to find the actual file
			result := r.tryResolveFile(fullPath, repoRoot)
			if result != "" {
				return result
			}
		}
	}

	// If no path patterns matched, try baseUrl resolution
	if r.baseUrl != "" {
		fullPath := filepath.Join(r.baseUrl, importSource)
		result := r.tryResolveFile(fullPath, repoRoot)
		if result != "" {
			return result
		}
	}

	return ""
}

// matchPattern matches an import against a path pattern
// Patterns can be:
// - Exact: "@/components" matches "@/components"
// - Wildcard: "@/*" matches "@/anything" with remainder "anything"
func matchPattern(pattern, importSource string) (matched bool, remainder string) {
	// Handle wildcard patterns
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		if strings.HasPrefix(importSource, prefix+"/") {
			return true, strings.TrimPrefix(importSource, prefix+"/")
		}
		// Also match exact prefix without slash
		if importSource == prefix {
			return true, ""
		}
		return false, ""
	}

	// Handle exact matches
	if pattern == importSource {
		return true, ""
	}

	return false, ""
}

// applyReplacement applies a replacement pattern to a remainder
func applyReplacement(replacement, remainder string) string {
	if strings.HasSuffix(replacement, "/*") {
		base := strings.TrimSuffix(replacement, "/*")
		if remainder != "" {
			return filepath.Join(base, remainder)
		}
		return base
	}
	return replacement
}

// tryResolveFile attempts to resolve a path to an actual file
func (r *TSConfigResolver) tryResolveFile(localPath, repoRoot string) string {
	// Extensions to try
	extensions := []string{
		"",
		".ts",
		".tsx",
		".js",
		".jsx",
		".mjs",
		"/index.ts",
		"/index.tsx",
		"/index.js",
		"/index.jsx",
	}

	for _, ext := range extensions {
		candidate := localPath + ext
		absPath := filepath.Join(repoRoot, candidate)
		if info, err := os.Stat(absPath); err == nil && !info.IsDir() {
			return candidate
		}
	}

	// Check if it's a directory with an index file
	absPath := filepath.Join(repoRoot, localPath)
	if info, err := os.Stat(absPath); err == nil && info.IsDir() {
		indexFiles := []string{
			"index.ts",
			"index.tsx",
			"index.js",
			"index.jsx",
		}
		for _, indexFile := range indexFiles {
			candidate := filepath.Join(localPath, indexFile)
			indexPath := filepath.Join(repoRoot, candidate)
			if info, err := os.Stat(indexPath); err == nil && !info.IsDir() {
				return candidate
			}
		}
	}

	return ""
}

// GetPaths returns the discovered paths (for testing/debugging)
func (r *TSConfigResolver) GetPaths() map[string][]string {
	result := make(map[string][]string, len(r.paths))
	for k, v := range r.paths {
		result[k] = v
	}
	return result
}

// GetBaseUrl returns the base URL (for testing/debugging)
func (r *TSConfigResolver) GetBaseUrl() string {
	return r.baseUrl
}
