package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// validateDesktopConfig validates desktop configuration
func (s *Server) validateDesktopConfig(config *DesktopConfig) error {
	// Basic validation
	if config.AppName == "" {
		return fmt.Errorf("app_name is required")
	}
	if config.Framework == "" {
		return fmt.Errorf("framework is required")
	}
	if config.TemplateType == "" {
		return fmt.Errorf("template_type is required")
	}
	// Note: output_path is optional - defaults to standard location if empty

	// Validate framework
	validFrameworks := []string{"electron", "tauri", "neutralino"}
	if !contains(validFrameworks, config.Framework) {
		return fmt.Errorf("invalid framework: %s", config.Framework)
	}

	// Validate template type
	validTemplates := []string{"universal", "basic", "advanced", "multi_window", "kiosk"} // basic is alias for universal
	if !contains(validTemplates, config.TemplateType) {
		return fmt.Errorf("invalid template_type: %s", config.TemplateType)
	}

	// Set defaults
	if config.License == "" {
		config.License = "MIT"
	}
	if len(config.Platforms) == 0 {
		config.Platforms = []string{"win", "mac", "linux"}
	}
	if config.ServerPort == 0 && (config.ServerType == "node" || config.ServerType == "executable") {
		config.ServerPort = 3000
	}

	if config.ServerType == "" {
		config.ServerType = "external"
	}
	if config.DeploymentMode == "" {
		config.DeploymentMode = "external-server"
	}
	validDeploymentModes := []string{"external-server", "cloud-api", "bundled"}
	if !contains(validDeploymentModes, config.DeploymentMode) {
		return fmt.Errorf("invalid deployment_mode: %s", config.DeploymentMode)
	}

	if config.ServerType == "external" {
		proxyURL := config.ProxyURL
		if proxyURL == "" {
			proxyURL = config.ExternalServerURL
		}
		if proxyURL == "" {
			proxyURL = config.ServerPath
		}
		if proxyURL == "" {
			return fmt.Errorf("proxy_url is required when server_type is 'external'")
		}
		normalizedProxy, err := normalizeProxyURL(proxyURL)
		if err != nil {
			return err
		}
		config.ProxyURL = normalizedProxy
		config.ExternalServerURL = normalizedProxy
		config.ServerPath = normalizedProxy
		config.ExternalAPIURL = proxyAPIURL(normalizedProxy)
		config.APIEndpoint = config.ExternalAPIURL
	} else {
		if config.ServerPath == "" {
			return fmt.Errorf("server_path is required when server_type is '%s'", config.ServerType)
		}
	}
	if config.ScenarioName == "" {
		config.ScenarioName = config.AppName
	}
	if config.VrooliBinaryPath == "" {
		config.VrooliBinaryPath = "vrooli"
	}
	if config.AutoManageVrooli && config.ServerType != "external" {
		return fmt.Errorf("auto_manage_vrooli is only supported when server_type is 'external'")
	}

	return nil
}

// isWineInstalled checks if Wine is installed on the system
func (s *Server) isWineInstalled() bool {
	cmd := exec.Command("which", "wine")
	err := cmd.Run()
	return err == nil
}

// Desktop test functions
func (s *Server) runDesktopTests(request *struct {
	AppPath   string   `json:"app_path"`
	Platforms []string `json:"platforms"`
	Headless  bool     `json:"headless"`
}) map[string]interface{} {
	results := map[string]interface{}{
		"package_json_valid": s.testPackageJSON(request.AppPath),
		"build_files_exist":  s.testBuildFiles(request.AppPath),
		"dependencies_ok":    s.testDependencies(request.AppPath),
	}

	return results
}

func (s *Server) testPackageJSON(appPath string) bool {
	packagePath := filepath.Join(appPath, "package.json")
	_, err := os.Stat(packagePath)
	return err == nil
}

func (s *Server) testBuildFiles(appPath string) bool {
	requiredFiles := []string{"src/main.ts", "src/preload.ts"}
	for _, file := range requiredFiles {
		filePath := filepath.Join(appPath, file)
		if _, err := os.Stat(filePath); err != nil {
			return false
		}
	}
	return true
}

func (s *Server) testDependencies(appPath string) bool {
	cmd := exec.Command("npm", "list", "--prod")
	cmd.Dir = appPath
	err := cmd.Run()
	return err == nil
}
