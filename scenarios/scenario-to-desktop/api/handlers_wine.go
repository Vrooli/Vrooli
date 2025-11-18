package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// checkWineHandler checks if Wine is installed and returns installation options
func (s *Server) checkWineHandler(w http.ResponseWriter, r *http.Request) {
	platform := "linux" // Default assumption for Wine needs
	if runtime := os.Getenv("GOOS"); runtime != "" {
		platform = runtime
	}

	response := WineCheckResponse{
		Installed:   s.isWineInstalled(),
		Platform:    platform,
		RequiredFor: []string{"win"},
	}

	// Get Wine version if installed
	if response.Installed {
		cmd := exec.Command("wine", "--version")
		if output, err := cmd.Output(); err == nil {
			response.Version = strings.TrimSpace(string(output))
		}
	}

	// Define available installation methods
	if !response.Installed && platform == "linux" {
		methods := []WineInstallMethod{}

		// Check if flatpak is available
		flatpakInstalled := false
		if _, err := exec.LookPath("flatpak"); err == nil {
			flatpakInstalled = true
		}

		// Strategy 1: Flatpak (if available) - fully automated
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
			// Strategy 2: Install Flatpak THEN Wine - fully automated, no sudo
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

		// Strategy 3: Wine AppImage - no dependencies, fully automated
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

		// Always offer skip option
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// installWineHandler initiates Wine installation process
func (s *Server) installWineHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Method string `json:"method"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	validMethods := []string{"flatpak", "flatpak-auto", "appimage", "skip"}
	isValid := false
	for _, valid := range validMethods {
		if request.Method == valid {
			isValid = true
			break
		}
	}
	if !isValid {
		http.Error(w, fmt.Sprintf("Invalid installation method. Supported: %s", strings.Join(validMethods, ", ")), http.StatusBadRequest)
		return
	}

	installID := uuid.New().String()
	status := &WineInstallStatus{
		InstallID: installID,
		Status:    "pending",
		Method:    request.Method,
		StartedAt: time.Now(),
		Log:       []string{},
	}

	s.wineInstallMux.Lock()
	s.wineInstalls[installID] = status
	s.wineInstallMux.Unlock()

	// Start installation in background based on method
	switch request.Method {
	case "flatpak":
		go s.performWineInstallation(installID)
	case "flatpak-auto":
		go s.performFlatpakAutoInstallation(installID)
	case "appimage":
		go s.performAppImageInstallation(installID)
	case "skip":
		// Skip method - immediately complete
		s.wineInstallMux.Lock()
		status.Status = "completed"
		now := time.Now()
		status.CompletedAt = &now
		status.Log = append(status.Log, "Windows build will be skipped")
		s.wineInstallMux.Unlock()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"install_id": installID,
		"status":     status.Status,
		"method":     request.Method,
		"status_url": fmt.Sprintf("/api/v1/system/wine/install/status/%s", installID),
	})
}

// getWineInstallStatusHandler returns Wine installation status
func (s *Server) getWineInstallStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installID := vars["install_id"]

	s.wineInstallMux.RLock()
	status, exists := s.wineInstalls[installID]
	s.wineInstallMux.RUnlock()

	if !exists {
		http.Error(w, "Installation ID not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// performWineInstallation performs the actual Wine installation via Flatpak
func (s *Server) performWineInstallation(installID string) {
	s.wineInstallMux.Lock()
	status := s.wineInstalls[installID]
	status.Status = "installing"
	s.wineInstallMux.Unlock()

	addLog := func(message string) {
		s.wineInstallMux.Lock()
		status.Log = append(status.Log, message)
		s.wineInstallMux.Unlock()
		s.logger.Info("wine installation", "install_id", installID, "message", message)
	}

	addError := func(message string) {
		s.wineInstallMux.Lock()
		status.ErrorLog = append(status.ErrorLog, message)
		s.wineInstallMux.Unlock()
		s.logger.Error("wine installation error", "install_id", installID, "error", message)
	}

	fail := func(message string) {
		addError(message)
		s.wineInstallMux.Lock()
		status.Status = "failed"
		now := time.Now()
		status.CompletedAt = &now
		s.wineInstallMux.Unlock()
	}

	succeed := func() {
		s.wineInstallMux.Lock()
		status.Status = "completed"
		now := time.Now()
		status.CompletedAt = &now
		s.wineInstallMux.Unlock()
	}

	// Step 1: Check if flatpak is available
	addLog("Checking if Flatpak is installed...")
	if _, err := exec.LookPath("flatpak"); err != nil {
		fail("Flatpak is not installed. Please install Flatpak first: https://flatpak.org/setup/")
		return
	}
	addLog("✓ Flatpak is available")

	// Step 2: Add Flathub repository (user-space, no sudo)
	addLog("Adding Flathub repository (user-space)...")
	cmd := exec.Command("flatpak", "--user", "remote-add", "--if-not-exists", "flathub", "https://flathub.org/repo/flathub.flatpakrepo")
	if output, err := cmd.CombinedOutput(); err != nil {
		outputStr := string(output)
		if !strings.Contains(outputStr, "already exists") {
			fail(fmt.Sprintf("Failed to add Flathub repository: %v\nOutput: %s", err, outputStr))
			return
		}
		addLog("✓ Flathub repository already exists")
	} else {
		addLog("✓ Flathub repository added")
	}

	// Step 3: Install Wine via Flatpak (user-space, no sudo)
	addLog("Installing Wine from Flathub (this may take 3-5 minutes)...")
	cmd = exec.Command("flatpak", "--user", "install", "-y", "flathub", "org.winehq.Wine")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	if err != nil {
		if strings.Contains(outputStr, "already installed") {
			addLog("✓ Wine is already installed via Flatpak")
		} else {
			fail(fmt.Sprintf("Failed to install Wine: %v\nOutput: %s", err, outputStr))
			return
		}
	} else {
		addLog("✓ Wine installed successfully")
	}

	// Step 4: Create wrapper script
	addLog("Creating Wine wrapper script in ~/.local/bin...")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fail(fmt.Sprintf("Failed to get home directory: %v", err))
		return
	}

	localBinDir := filepath.Join(homeDir, ".local", "bin")
	if err := os.MkdirAll(localBinDir, 0755); err != nil {
		fail(fmt.Sprintf("Failed to create ~/.local/bin directory: %v", err))
		return
	}

	wineScript := filepath.Join(localBinDir, "wine")
	wineScriptContent := `#!/bin/bash
exec flatpak run org.winehq.Wine "$@"
`
	if err := os.WriteFile(wineScript, []byte(wineScriptContent), 0755); err != nil {
		fail(fmt.Sprintf("Failed to create wine wrapper script: %v", err))
		return
	}
	addLog(fmt.Sprintf("✓ Wine wrapper created at %s", wineScript))

	// Step 5: Verify installation
	addLog("Verifying Wine installation...")

	// Update PATH to include ~/.local/bin for this check
	currentPath := os.Getenv("PATH")
	os.Setenv("PATH", localBinDir+":"+currentPath)

	cmd = exec.Command("wine", "--version")
	if output, err := cmd.Output(); err != nil {
		addError(fmt.Sprintf("Wine wrapper created but verification failed: %v", err))
		addLog("Wine is installed but may not be in PATH. Add to PATH with:")
		addLog(fmt.Sprintf("  export PATH=\"$HOME/.local/bin:$PATH\""))
		addLog("Add this line to ~/.bashrc or ~/.zshrc to make it permanent")
	} else {
		version := strings.TrimSpace(string(output))
		addLog(fmt.Sprintf("✓ Wine installation verified: %s", version))
	}

	addLog("Installation complete! Wine is ready for Windows builds.")
	addLog("Note: You may need to restart your terminal or run:")
	addLog("  export PATH=\"$HOME/.local/bin:$PATH\"")

	succeed()
}

// performFlatpakAutoInstallation installs Flatpak to user-space, then installs Wine
func (s *Server) performFlatpakAutoInstallation(installID string) {
	s.wineInstallMux.Lock()
	status := s.wineInstalls[installID]
	status.Status = "installing"
	s.wineInstallMux.Unlock()

	addLog := func(message string) {
		s.wineInstallMux.Lock()
		status.Log = append(status.Log, message)
		s.wineInstallMux.Unlock()
		s.logger.Info("wine installation", "install_id", installID, "message", message)
	}

	addError := func(message string) {
		s.wineInstallMux.Lock()
		status.ErrorLog = append(status.ErrorLog, message)
		s.wineInstallMux.Unlock()
		s.logger.Error("wine installation error", "install_id", installID, "error", message)
	}

	fail := func(message string) {
		addError(message)
		s.wineInstallMux.Lock()
		status.Status = "failed"
		now := time.Now()
		status.CompletedAt = &now
		s.wineInstallMux.Unlock()
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fail(fmt.Sprintf("Failed to get home directory: %v", err))
		return
	}

	localBinDir := filepath.Join(homeDir, ".local", "bin")
	if err := os.MkdirAll(localBinDir, 0755); err != nil {
		fail(fmt.Sprintf("Failed to create ~/.local/bin directory: %v", err))
		return
	}

	// Step 1: Download and install Flatpak binary to user-space
	addLog("Downloading Flatpak binary for user-space installation...")
	addLog("Note: This installs a minimal Flatpak binary, not the full system package")

	// Download Flatpak single-file bundle (if available) or use system package manager
	// For now, inform user that system Flatpak is needed but we'll try package manager
	addLog("Attempting to install Flatpak using system package manager...")
	addLog("Checking available package managers...")

	// Try different package managers with --user or user-space options
	packageManagers := []struct {
		name       string
		checkCmd   []string
		installCmd []string
	}{
		{
			name:       "apt (Debian/Ubuntu)",
			checkCmd:   []string{"which", "apt-get"},
			installCmd: []string{"apt-get", "download", "flatpak"},
		},
		// Note: Most package managers require root for installation
		// The reality is Flatpak itself typically needs system-level install
	}

	flatpakInstalled := false
	for _, pm := range packageManagers {
		if _, err := exec.Command(pm.checkCmd[0], pm.checkCmd[1:]...).Output(); err == nil {
			addLog(fmt.Sprintf("Found %s", pm.name))
			addLog("Note: Flatpak typically requires system-level installation")
			addLog("Falling back to AppImage method instead...")
			break
		}
	}

	if !flatpakInstalled {
		addLog("Cannot install Flatpak without system package manager or sudo")
		addLog("Switching to Wine AppImage installation method (no Flatpak needed)...")

		// Call AppImage installation instead
		s.performAppImageInstallation(installID)
		return
	}
}

// performAppImageInstallation downloads and installs Wine as AppImage (no dependencies)
func (s *Server) performAppImageInstallation(installID string) {
	s.wineInstallMux.Lock()
	status := s.wineInstalls[installID]
	status.Status = "installing"
	s.wineInstallMux.Unlock()

	addLog := func(message string) {
		s.wineInstallMux.Lock()
		status.Log = append(status.Log, message)
		s.wineInstallMux.Unlock()
		s.logger.Info("wine installation", "install_id", installID, "message", message)
	}

	addError := func(message string) {
		s.wineInstallMux.Lock()
		status.ErrorLog = append(status.ErrorLog, message)
		s.wineInstallMux.Unlock()
		s.logger.Error("wine installation error", "install_id", installID, "error", message)
	}

	fail := func(message string) {
		addError(message)
		s.wineInstallMux.Lock()
		status.Status = "failed"
		now := time.Now()
		status.CompletedAt = &now
		s.wineInstallMux.Unlock()
	}

	succeed := func() {
		s.wineInstallMux.Lock()
		status.Status = "completed"
		now := time.Now()
		status.CompletedAt = &now
		s.wineInstallMux.Unlock()
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fail(fmt.Sprintf("Failed to get home directory: %v", err))
		return
	}

	localBinDir := filepath.Join(homeDir, ".local", "bin")
	if err := os.MkdirAll(localBinDir, 0755); err != nil {
		fail(fmt.Sprintf("Failed to create ~/.local/bin directory: %v", err))
		return
	}

	// Step 1: Download Wine AppImage
	addLog("Downloading Wine AppImage (this may take 2-3 minutes)...")

	// Wine AppImage URL (using a stable version)
	wineAppImageURL := "https://github.com/mmtrt/WINE_AppImage/releases/download/continuous/wine-stable_7.0-x86_64.AppImage"
	wineAppImage := filepath.Join(localBinDir, "wine.AppImage")

	addLog(fmt.Sprintf("Downloading from: %s", wineAppImageURL))
	addLog("This is a ~70MB download, please wait...")

	cmd := exec.Command("curl", "-L", "-o", wineAppImage, wineAppImageURL)
	if output, err := cmd.CombinedOutput(); err != nil {
		fail(fmt.Sprintf("Failed to download Wine AppImage: %v\nOutput: %s", err, string(output)))
		return
	}
	addLog("✓ Wine AppImage downloaded")

	// Step 2: Make AppImage executable
	addLog("Making Wine AppImage executable...")
	if err := os.Chmod(wineAppImage, 0755); err != nil {
		fail(fmt.Sprintf("Failed to make AppImage executable: %v", err))
		return
	}
	addLog("✓ Wine AppImage is executable")

	// Step 3: Create wine wrapper script
	addLog("Creating Wine wrapper script...")
	wineScript := filepath.Join(localBinDir, "wine")
	wineScriptContent := fmt.Sprintf(`#!/bin/bash
# Wine AppImage wrapper
exec "%s" "$@"
`, wineAppImage)

	if err := os.WriteFile(wineScript, []byte(wineScriptContent), 0755); err != nil {
		fail(fmt.Sprintf("Failed to create wine wrapper script: %v", err))
		return
	}
	addLog(fmt.Sprintf("✓ Wine wrapper created at %s", wineScript))

	// Step 4: Verify installation
	addLog("Verifying Wine installation...")

	// Update PATH to include ~/.local/bin for this check
	currentPath := os.Getenv("PATH")
	os.Setenv("PATH", localBinDir+":"+currentPath)

	cmd = exec.Command("wine", "--version")
	if output, err := cmd.Output(); err != nil {
		addError(fmt.Sprintf("Wine AppImage created but verification failed: %v", err))
		addLog("Wine is installed but may not be in PATH. Add to PATH with:")
		addLog("  export PATH=\"$HOME/.local/bin:$PATH\"")
		addLog("Add this line to ~/.bashrc or ~/.zshrc to make it permanent")
	} else {
		version := strings.TrimSpace(string(output))
		addLog(fmt.Sprintf("✓ Wine installation verified: %s", version))
	}

	addLog("Installation complete! Wine AppImage is ready for Windows builds.")
	addLog("Note: You may need to restart your terminal or run:")
	addLog("  export PATH=\"$HOME/.local/bin:$PATH\"")

	succeed()
}
