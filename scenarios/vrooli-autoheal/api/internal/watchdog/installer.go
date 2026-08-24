package watchdog

import (
	"context"
	"time"
)

// InstallResult is retained for the scenario API contract. Native scheduler
// mutation belongs to the project control plane; callers should use
// `vrooli setup` rather than invoking scenario-owned installation logic.
type InstallResult struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	ServicePath   string `json:"servicePath,omitempty"`
	NeedsLinger   bool   `json:"needsLinger,omitempty"`
	LingerCommand string `json:"lingerCommand,omitempty"`
	Error         string `json:"error,omitempty"`
}

type UninstallResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

type InstallOptions struct {
	UseSystemService bool `json:"useSystemService"`
	EnableLingering  bool `json:"enableLingering"`
}

// Install is a compatibility façade. It deliberately performs no host
// mutation; the control-plane autoheal_watchdog safeguard owns installation.
func (d *Detector) Install(_ context.Context, _ InstallOptions) *InstallResult {
	return &InstallResult{
		Message: "autoheal watchdog installation is owned by project setup",
		Error:   "run `vrooli setup` to install and verify autoheal boot protection",
	}
}

// Uninstall is retained only for API compatibility. There is no scenario
// uninstall path because setup owns the lifecycle and must preserve its
// declared protection contract.
func (d *Detector) Uninstall(_ context.Context) *UninstallResult {
	return &UninstallResult{
		Message: "autoheal watchdog lifecycle is owned by project setup",
		Error:   "scenario-owned watchdog uninstall is unsupported; use the project setup policy",
	}
}

// EnableLingering is retained only for API compatibility. Dedicated-host
// lingering is applied by the control-plane safeguard according to operator
// policy, never by the scenario.
func (d *Detector) EnableLingering(_ context.Context) *InstallResult {
	return &InstallResult{
		Message: "systemd lingering is owned by project setup",
		Error:   "run `vrooli setup` to apply the selected boot policy",
	}
}

type InstallStatus struct {
	Installed        bool   `json:"installed"`
	Enabled          bool   `json:"enabled"`
	Running          bool   `json:"running"`
	BootProtected    bool   `json:"bootProtected"`
	ServicePath      string `json:"servicePath,omitempty"`
	WatchdogType     string `json:"watchdogType"`
	CanInstall       bool   `json:"canInstall"`
	NeedsLinger      bool   `json:"needsLinger"`
	LingerCommand    string `json:"lingerCommand,omitempty"`
	ProtectionLevel  string `json:"protectionLevel"`
	LastChecked      string `json:"lastChecked"`
	RecommendedSetup string `json:"recommendedSetup"`
}

// GetInstallStatus is read-only and exists for older scenario clients. The
// authoritative setup status is `vrooli setup status`.
func (d *Detector) GetInstallStatus() *InstallStatus {
	status := d.Detect()
	needsLinger := status.IsUserService && !status.LingeringEnabled
	return &InstallStatus{
		Installed:        status.WatchdogInstalled,
		Enabled:          status.WatchdogEnabled,
		Running:          status.WatchdogRunning,
		BootProtected:    status.BootProtectionActive,
		ServicePath:      status.ServicePath,
		WatchdogType:     string(status.WatchdogType),
		CanInstall:       status.CanInstall,
		NeedsLinger:      needsLinger,
		ProtectionLevel:  string(status.ProtectionLevel),
		LastChecked:      time.Now().UTC().Format(time.RFC3339),
		RecommendedSetup: recommendedSetup(string(d.platform.Platform)),
	}
}

func recommendedSetup(platform string) string {
	switch platform {
	case "linux", "macos":
		return "user"
	case "windows":
		return "system"
	default:
		return ""
	}
}
