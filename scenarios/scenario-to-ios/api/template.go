package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// TemplateExpander handles iOS project template expansion
type TemplateExpander struct {
	scenarioName string
	scenarioPath string
	appName      string
	appClassName string // CamelCase version for Swift struct names
	bundleID     string
	version      string
	buildNumber  string
}

// NewTemplateExpander creates a new template expander
func NewTemplateExpander(scenarioName string) *TemplateExpander {
	scenarioPath := fmt.Sprintf("/home/matthalloran8/Vrooli/scenarios/%s", scenarioName)
	appName := formatAppName(scenarioName)
	return &TemplateExpander{
		scenarioName: scenarioName,
		scenarioPath: scenarioPath,
		appName:      appName,
		appClassName: formatAppClassName(appName),
		bundleID:     formatBundleID(scenarioName),
		version:      "1.0.0",
		buildNumber:  "1",
	}
}

// formatAppName converts scenario name to app name (e.g., "test-scenario" -> "Test Scenario")
func formatAppName(scenarioName string) string {
	parts := strings.Split(scenarioName, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}

// formatBundleID converts scenario name to bundle ID (e.g., "test-scenario" -> "com.vrooli.test-scenario")
func formatBundleID(scenarioName string) string {
	return fmt.Sprintf("com.vrooli.%s", scenarioName)
}

// formatAppClassName converts app name to Swift class name (e.g., "Simple Test" -> "SimpleTest")
func formatAppClassName(appName string) string {
	// Remove spaces and other non-alphanumeric characters, keep CamelCase
	result := strings.ReplaceAll(appName, " ", "")
	result = strings.ReplaceAll(result, "-", "")
	result = strings.ReplaceAll(result, "_", "")
	return result
}

// ExpandTemplate copies and expands the iOS template
func (te *TemplateExpander) ExpandTemplate(outputDir string) error {
	// Get template source directory
	templateDir := filepath.Join(getProjectRoot(), "initialization", "templates", "ios")

	// Verify template exists
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		return fmt.Errorf("template directory not found: %s", templateDir)
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Copy template files with placeholder replacement
	err := filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}

		// Skip if at template root
		if relPath == "." {
			return nil
		}

		// Construct output path
		outPath := filepath.Join(outputDir, relPath)

		if info.IsDir() {
			// Create directory
			return os.MkdirAll(outPath, info.Mode())
		}

		// Copy and expand file
		return te.expandFile(path, outPath)
	})

	if err != nil {
		return err
	}

	// Copy scenario UI files into the app bundle
	return te.copyScenarioUI(outputDir)
}

// copyScenarioUI copies the scenario's UI files into the iOS app bundle
func (te *TemplateExpander) copyScenarioUI(projectDir string) error {
	// Check for scenario UI directory
	scenarioUIPath := filepath.Join(te.scenarioPath, "ui")
	if _, err := os.Stat(scenarioUIPath); os.IsNotExist(err) {
		// No UI directory - that's okay, some scenarios might not have one
		return nil
	}

	// Destination is project/www/ (web bundle inside iOS app)
	destPath := filepath.Join(projectDir, "project", "www")

	// Remove the default index.html before copying scenario UI
	defaultIndex := filepath.Join(destPath, "index.html")
	if err := os.Remove(defaultIndex); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove default index.html: %w", err)
	}

	// Copy scenario UI files
	return CopyDirectory(scenarioUIPath, destPath)
}

// expandFile copies a file and replaces template placeholders
func (te *TemplateExpander) expandFile(srcPath, dstPath string) error {
	// Read source file
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	// Replace placeholders
	expanded := te.replacePlaceholders(string(content))

	// Write expanded content
	return os.WriteFile(dstPath, []byte(expanded), 0644)
}

// replacePlaceholders replaces template placeholders with actual values
func (te *TemplateExpander) replacePlaceholders(content string) string {
	replacements := map[string]string{
		"{{SCENARIO_NAME}}":  te.scenarioName,
		"{{APP_NAME}}":       te.appName,
		"{{APP_CLASS_NAME}}": te.appClassName, // CamelCase for Swift structs/classes
		"{{BUNDLE_ID}}":      te.bundleID,
		"{{VERSION}}":        te.version,
		"{{APP_VERSION}}":    te.version,      // Used in Info.plist and ContentView
		"{{BUILD_NUMBER}}":   te.buildNumber,
		"{{SCENARIO_URL}}":   "http://localhost:8080", // Placeholder, will be replaced with actual scenario URL
		"VrooliScenarioApp":  te.appClassName + "App", // Swift struct name
		"VrooliScenario":     te.appClassName, // Swift class references
	}

	result := content
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// getProjectRoot returns the scenario root directory
func getProjectRoot() string {
	// First try working directory (works for tests and development)
	wd, err := os.Getwd()
	if err == nil {
		// Check if we're in the api/ directory
		if filepath.Base(wd) == "api" {
			return filepath.Dir(wd)
		}
		// Check if initialization/templates exists in current directory
		templatePath := filepath.Join(wd, "initialization", "templates", "ios")
		if _, err := os.Stat(templatePath); err == nil {
			return wd
		}
		// Try parent directory
		parentPath := filepath.Join(filepath.Dir(wd), "initialization", "templates", "ios")
		if _, err := os.Stat(parentPath); err == nil {
			return filepath.Dir(wd)
		}
	}

	// Fallback to executable location
	exePath, err := os.Executable()
	if err == nil {
		// API binary is in api/, project root is parent
		return filepath.Dir(filepath.Dir(exePath))
	}

	// Last resort: current working directory parent
	return filepath.Join(wd, "..")
}

// CopyDirectory recursively copies a directory, skipping unnecessary files
func CopyDirectory(src, dst string) error {
	// Directories to skip (these aren't needed in the iOS app bundle)
	skipDirs := map[string]bool{
		"node_modules": true,
		".git":         true,
		".next":        true,
		"dist":         true,
		"build":        true,
		".pnpm":        true,
		"coverage":     true,
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Skip unwanted directories
		if info.IsDir() {
			baseName := filepath.Base(path)
			if skipDirs[baseName] {
				return filepath.SkipDir
			}
		}

		// Construct destination path
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			// Create directory
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		return copyFile(path, dstPath)
	})
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
