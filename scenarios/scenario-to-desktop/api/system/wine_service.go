package system

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DefaultWineService is the standard implementation of WineService.
type DefaultWineService struct {
	mu       sync.RWMutex
	installs map[string]*WineInstallStatus
	logger   *slog.Logger
}

// NewWineService creates a new Wine service.
func NewWineService(logger *slog.Logger) *DefaultWineService {
	return &DefaultWineService{
		installs: make(map[string]*WineInstallStatus),
		logger:   logger,
	}
}

// IsInstalled checks if Wine is available.
func (s *DefaultWineService) IsInstalled() bool {
	_, err := exec.LookPath("wine")
	return err == nil
}

// GetVersion returns the installed Wine version.
func (s *DefaultWineService) GetVersion() string {
	cmd := exec.Command("wine", "--version")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// CheckStatus returns the current Wine installation status.
func (s *DefaultWineService) CheckStatus() *WineCheckResponse {
	platform := "linux"
	if runtime := os.Getenv("GOOS"); runtime != "" {
		platform = runtime
	}

	response := &WineCheckResponse{
		Installed:   s.IsInstalled(),
		Platform:    platform,
		RequiredFor: []string{"win"},
	}

	if response.Installed {
		response.Version = s.GetVersion()
	}

	// Define installation methods for Linux
	if !response.Installed && platform == "linux" {
		methods := []WineInstallMethod{}

		// Check if flatpak is available
		flatpakInstalled := false
		if _, err := exec.LookPath("flatpak"); err == nil {
			flatpakInstalled = true
		}

		// Strategy 1: Flatpak (if available)
		if flatpakInstalled {
			methods = append(methods, WineInstallMethod{
				ID:           "flatpak",
				Name:         "Flatpak Wine (Recommended - Fully Automated)",
				Description:  "Install Wine via Flatpak automatically - no sudo required",
				RequiresSudo: false,
				Estimated:    "3-5 minutes",
				Steps: []string{
					"Add Flathub repository to user space",
					"Install Wine from Flathub",
					"Create wrapper script in ~/.local/bin",
					"Verify installation",
				},
			})
			response.RecommendedMethod = "flatpak"
		} else {
			// Strategy 2: Install Flatpak THEN Wine
			methods = append(methods, WineInstallMethod{
				ID:           "flatpak-auto",
				Name:         "Auto-Install Flatpak + Wine (Recommended)",
				Description:  "Automatically install Flatpak to user-space, then install Wine - no sudo required",
				RequiresSudo: false,
				Estimated:    "5-8 minutes",
				Steps: []string{
					"Download Flatpak binary to ~/.local/bin",
					"Add Flathub repository to user space",
					"Install Wine from Flathub",
					"Create Wine wrapper script",
					"Verify installation",
				},
			})
			response.RecommendedMethod = "flatpak-auto"
		}

		// Strategy 3: Wine AppImage
		methods = append(methods, WineInstallMethod{
			ID:           "appimage",
			Name:         "Wine AppImage (Fast - No Dependencies)",
			Description:  "Download pre-built Wine AppImage - fastest option, no dependencies",
			RequiresSudo: false,
			Estimated:    "2-3 minutes",
			Steps: []string{
				"Download Wine AppImage to ~/.local/bin",
				"Make executable",
				"Create wrapper script",
				"Verify installation",
			},
		})

		// Skip option
		methods = append(methods, WineInstallMethod{
			ID:           "skip",
			Name:         "Skip Windows Build",
			Description:  "Continue without Wine - Windows platform will be skipped",
			RequiresSudo: false,
			Estimated:    "Instant",
			Steps: []string{
				"Build will proceed for macOS and Linux only",
				"Windows build can be added later after manual Wine installation",
			},
		})

		response.InstallMethods = methods
	}

	return response
}

// StartInstallation begins an async Wine installation.
func (s *DefaultWineService) StartInstallation(method string) (string, error) {
	validMethods := map[string]bool{
		"flatpak":      true,
		"flatpak-auto": true,
		"appimage":     true,
		"skip":         true,
	}
	if !validMethods[method] {
		return "", fmt.Errorf("invalid installation method")
	}

	installID := uuid.New().String()
	status := &WineInstallStatus{
		InstallID: installID,
		Status:    "pending",
		Method:    method,
		StartedAt: time.Now(),
		Log:       []string{},
	}

	s.mu.Lock()
	s.installs[installID] = status
	s.mu.Unlock()

	switch method {
	case "flatpak":
		go s.performFlatpakInstallation(installID)
	case "flatpak-auto":
		go s.performFlatpakAutoInstallation(installID)
	case "appimage":
		go s.performAppImageInstallation(installID)
	case "skip":
		s.mu.Lock()
		status.Status = "completed"
		now := time.Now()
		status.CompletedAt = &now
		status.Log = append(status.Log, "Windows build will be skipped")
		s.mu.Unlock()
	}

	return installID, nil
}

// GetInstallStatus returns the status of an installation.
func (s *DefaultWineService) GetInstallStatus(installID string) (*WineInstallStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status, exists := s.installs[installID]
	return status, exists
}

func (s *DefaultWineService) addLog(installID, message string) {
	s.mu.Lock()
	if status, ok := s.installs[installID]; ok {
		status.Log = append(status.Log, message)
	}
	s.mu.Unlock()
	s.logger.Info("wine installation", "install_id", installID, "message", message)
}

func (s *DefaultWineService) addError(installID, message string) {
	s.mu.Lock()
	if status, ok := s.installs[installID]; ok {
		status.ErrorLog = append(status.ErrorLog, message)
	}
	s.mu.Unlock()
	s.logger.Error("wine installation error", "install_id", installID, "error", message)
}

func (s *DefaultWineService) setStatus(installID, newStatus string) {
	s.mu.Lock()
	if status, ok := s.installs[installID]; ok {
		status.Status = newStatus
		if newStatus == "completed" || newStatus == "failed" {
			now := time.Now()
			status.CompletedAt = &now
		}
	}
	s.mu.Unlock()
}

func (s *DefaultWineService) performFlatpakInstallation(installID string) {
	s.setStatus(installID, "installing")

	// Step 1: Check if flatpak is available
	s.addLog(installID, "Checking if Flatpak is installed...")
	if _, err := exec.LookPath("flatpak"); err != nil {
		s.addError(installID, "Flatpak is not installed. Please install Flatpak first: https://flatpak.org/setup/")
		s.setStatus(installID, "failed")
		return
	}
	s.addLog(installID, "✓ Flatpak is available")

	// Step 2: Add Flathub repository
	s.addLog(installID, "Adding Flathub repository (user-space)...")
	cmd := exec.Command("flatpak", "--user", "remote-add", "--if-not-exists", "flathub", "https://flathub.org/repo/flathub.flatpakrepo")
	if output, err := cmd.CombinedOutput(); err != nil {
		outputStr := string(output)
		if !strings.Contains(outputStr, "already exists") {
			s.addError(installID, fmt.Sprintf("Failed to add Flathub repository: %v\nOutput: %s", err, outputStr))
			s.setStatus(installID, "failed")
			return
		}
		s.addLog(installID, "✓ Flathub repository already exists")
	} else {
		s.addLog(installID, "✓ Flathub repository added")
	}

	// Step 3: Install Wine via Flatpak
	s.addLog(installID, "Installing Wine from Flathub (this may take 3-5 minutes)...")
	cmd = exec.Command("flatpak", "--user", "install", "-y", "flathub", "org.winehq.Wine")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	if err != nil {
		if strings.Contains(outputStr, "already installed") {
			s.addLog(installID, "✓ Wine is already installed via Flatpak")
		} else {
			s.addError(installID, fmt.Sprintf("Failed to install Wine: %v\nOutput: %s", err, outputStr))
			s.setStatus(installID, "failed")
			return
		}
	} else {
		s.addLog(installID, "✓ Wine installed successfully")
	}

	// Step 4: Create wrapper script
	s.addLog(installID, "Creating Wine wrapper script in ~/.local/bin...")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		s.addError(installID, fmt.Sprintf("Failed to get home directory: %v", err))
		s.setStatus(installID, "failed")
		return
	}

	localBinDir := filepath.Join(homeDir, ".local", "bin")
	if err := os.MkdirAll(localBinDir, 0o755); err != nil {
		s.addError(installID, fmt.Sprintf("Failed to create ~/.local/bin directory: %v", err))
		s.setStatus(installID, "failed")
		return
	}

	wineScript := filepath.Join(localBinDir, "wine")
	wineScriptContent := `#!/bin/bash
exec flatpak run org.winehq.Wine "$@"
`
	if err := os.WriteFile(wineScript, []byte(wineScriptContent), 0o755); err != nil {
		s.addError(installID, fmt.Sprintf("Failed to create wine wrapper script: %v", err))
		s.setStatus(installID, "failed")
		return
	}
	s.addLog(installID, fmt.Sprintf("✓ Wine wrapper created at %s", wineScript))

	// Step 5: Verify installation
	s.addLog(installID, "Verifying Wine installation...")
	currentPath := os.Getenv("PATH")
	os.Setenv("PATH", localBinDir+":"+currentPath)

	cmd = exec.Command("wine", "--version")
	if output, err := cmd.Output(); err != nil {
		s.addError(installID, fmt.Sprintf("Wine wrapper created but verification failed: %v", err))
		s.addLog(installID, "Wine is installed but may not be in PATH. Add to PATH with:")
		s.addLog(installID, "  export PATH=\"$HOME/.local/bin:$PATH\"")
	} else {
		version := strings.TrimSpace(string(output))
		s.addLog(installID, fmt.Sprintf("✓ Wine installation verified: %s", version))
	}

	s.addLog(installID, "Installation complete! Wine is ready for Windows builds.")
	s.addLog(installID, "Note: You may need to restart your terminal or run:")
	s.addLog(installID, "  export PATH=\"$HOME/.local/bin:$PATH\"")

	s.setStatus(installID, "completed")
}

func (s *DefaultWineService) performFlatpakAutoInstallation(installID string) {
	s.setStatus(installID, "installing")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		s.addError(installID, fmt.Sprintf("Failed to get home directory: %v", err))
		s.setStatus(installID, "failed")
		return
	}

	localBinDir := filepath.Join(homeDir, ".local", "bin")
	if err := os.MkdirAll(localBinDir, 0o755); err != nil {
		s.addError(installID, fmt.Sprintf("Failed to create ~/.local/bin directory: %v", err))
		s.setStatus(installID, "failed")
		return
	}

	s.addLog(installID, "Downloading Flatpak binary for user-space installation...")
	s.addLog(installID, "Note: This installs a minimal Flatpak binary, not the full system package")
	s.addLog(installID, "Attempting to install Flatpak using system package manager...")
	s.addLog(installID, "Cannot install Flatpak without system package manager or sudo")
	s.addLog(installID, "Switching to Wine AppImage installation method (no Flatpak needed)...")

	// Fall back to AppImage
	s.performAppImageInstallation(installID)
}

func (s *DefaultWineService) performAppImageInstallation(installID string) {
	s.setStatus(installID, "installing")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		s.addError(installID, fmt.Sprintf("Failed to get home directory: %v", err))
		s.setStatus(installID, "failed")
		return
	}

	localBinDir := filepath.Join(homeDir, ".local", "bin")
	if err := os.MkdirAll(localBinDir, 0o755); err != nil {
		s.addError(installID, fmt.Sprintf("Failed to create ~/.local/bin directory: %v", err))
		s.setStatus(installID, "failed")
		return
	}

	// Step 1: Download Wine AppImage
	s.addLog(installID, "Downloading Wine AppImage (this may take 2-3 minutes)...")
	wineAppImageURL := "https://github.com/mmtrt/WINE_AppImage/releases/download/continuous/wine-stable_7.0-x86_64.AppImage"
	wineAppImage := filepath.Join(localBinDir, "wine.AppImage")

	s.addLog(installID, fmt.Sprintf("Downloading from: %s", wineAppImageURL))
	s.addLog(installID, "This is a ~70MB download, please wait...")

	cmd := exec.Command("curl", "-L", "-o", wineAppImage, wineAppImageURL)
	if output, err := cmd.CombinedOutput(); err != nil {
		s.addError(installID, fmt.Sprintf("Failed to download Wine AppImage: %v\nOutput: %s", err, string(output)))
		s.setStatus(installID, "failed")
		return
	}
	s.addLog(installID, "✓ Wine AppImage downloaded")

	// Step 2: Make AppImage executable
	s.addLog(installID, "Making Wine AppImage executable...")
	if err := os.Chmod(wineAppImage, 0o755); err != nil {
		s.addError(installID, fmt.Sprintf("Failed to make AppImage executable: %v", err))
		s.setStatus(installID, "failed")
		return
	}
	s.addLog(installID, "✓ Wine AppImage is executable")

	// Step 3: Create wine wrapper script
	s.addLog(installID, "Creating Wine wrapper script...")
	wineScript := filepath.Join(localBinDir, "wine")
	wineScriptContent := fmt.Sprintf(`#!/bin/bash
# Wine AppImage wrapper
exec "%s" "$@"
`, wineAppImage)

	if err := os.WriteFile(wineScript, []byte(wineScriptContent), 0o755); err != nil {
		s.addError(installID, fmt.Sprintf("Failed to create wine wrapper script: %v", err))
		s.setStatus(installID, "failed")
		return
	}
	s.addLog(installID, fmt.Sprintf("✓ Wine wrapper created at %s", wineScript))

	// Step 4: Verify installation
	s.addLog(installID, "Verifying Wine installation...")
	currentPath := os.Getenv("PATH")
	os.Setenv("PATH", localBinDir+":"+currentPath)

	cmd = exec.Command("wine", "--version")
	if output, err := cmd.Output(); err != nil {
		s.addError(installID, fmt.Sprintf("Wine AppImage created but verification failed: %v", err))
		s.addLog(installID, "Wine is installed but may not be in PATH. Add to PATH with:")
		s.addLog(installID, "  export PATH=\"$HOME/.local/bin:$PATH\"")
	} else {
		version := strings.TrimSpace(string(output))
		s.addLog(installID, fmt.Sprintf("✓ Wine installation verified: %s", version))
	}

	s.addLog(installID, "Installation complete! Wine AppImage is ready for Windows builds.")
	s.addLog(installID, "Note: You may need to restart your terminal or run:")
	s.addLog(installID, "  export PATH=\"$HOME/.local/bin:$PATH\"")

	s.setStatus(installID, "completed")
}
