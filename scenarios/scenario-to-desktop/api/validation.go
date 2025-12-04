package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
	isBundled := config.DeploymentMode == "bundled"

	// Validate framework
	if !isValidFramework(config.Framework) {
		return fmt.Errorf("invalid framework: %s", config.Framework)
	}

	// Validate template type
	if !isValidTemplateType(config.TemplateType) {
		return fmt.Errorf("invalid template_type: %s", config.TemplateType)
	}

	// Set defaults
	if config.License == "" {
		config.License = "MIT"
	}
	if len(config.Platforms) == 0 {
		if isBundled {
			config.Platforms = []string{defaultPlatformKey()}
		} else {
			config.Platforms = []string{"win", "mac", "linux"}
		}
	}
	if config.ServerPort == 0 && (config.ServerType == "node" || config.ServerType == "executable") {
		config.ServerPort = 3000
	}

	if config.ServerType == "" {
		config.ServerType = defaultServerType
	}
	if !isValidServerType(config.ServerType) {
		return fmt.Errorf("invalid server_type: %s", config.ServerType)
	}
	if config.DeploymentMode == "" {
		config.DeploymentMode = defaultDeploymentMode
		isBundled = false
	}
	if !isValidDeploymentMode(config.DeploymentMode) {
		return fmt.Errorf("invalid deployment_mode: %s", config.DeploymentMode)
	}
	if isBundled && config.BundleManifestPath == "" {
		return fmt.Errorf("bundle_manifest_path is required when deployment_mode is 'bundled'")
	}

	if isBundled {
		// Bundled builds run against the packaged runtime; avoid strict proxy requirements.
		if config.ServerType != "external" {
			config.ServerType = "external"
		}
		if config.ServerPath == "" {
			config.ServerPath = "http://127.0.0.1"
		}
		if config.ProxyURL == "" {
			config.ProxyURL = config.ServerPath
		}
		if config.ExternalServerURL == "" {
			config.ExternalServerURL = config.ServerPath
		}
		if config.APIEndpoint == "" {
			config.APIEndpoint = config.ServerPath
		}
		if config.ExternalAPIURL == "" {
			config.ExternalAPIURL = config.APIEndpoint
		}
	} else if config.ServerType == "external" {
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
	if config.APIEndpoint == "" {
		config.APIEndpoint = "http://127.0.0.1"
	}
	if config.Icon != "" {
		iconPath, err := filepath.Abs(config.Icon)
		if err != nil {
			return fmt.Errorf("resolve icon path: %w", err)
		}
		info, err := os.Stat(iconPath)
		if err != nil {
			return fmt.Errorf("icon file not found: %w", err)
		}
		if info.IsDir() {
			return fmt.Errorf("icon must be a file, not a directory")
		}
		if !strings.HasSuffix(strings.ToLower(iconPath), ".png") {
			return fmt.Errorf("icon must be a .png file")
		}
		config.Icon = iconPath
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

func defaultPlatformKey() string {
	switch runtime.GOOS {
	case "darwin":
		return "mac"
	case "windows":
		return "win"
	default:
		return "linux"
	}
}
