package system

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "wine", "--version")
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

	// Windows cross-builds use host Wine. This scenario observes host tools;
	// it does not install or repair them.

	return response
}

// StartInstallation begins an async Wine installation.
func (s *DefaultWineService) StartInstallation(method string) (string, error) {
	if method != "skip" {
		return "", fmt.Errorf("Wine installation and repair are owned by the control plane; provide a working host Wine or build on Windows")
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

	s.mu.Lock()
	status.Status = "completed"
	now := time.Now()
	status.CompletedAt = &now
	status.Log = append(status.Log, "Windows build will be skipped")
	s.mu.Unlock()

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
