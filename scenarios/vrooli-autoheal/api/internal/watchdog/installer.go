package watchdog

import "time"

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

// GetInstallStatus is the read surface for older scenario clients. There is
// no scenario-owned install, uninstall or linger mutation any more: the
// control-plane autoheal_watchdog safeguard owns the unit, and `vrooli setup`
// is the one remediation. The authoritative status is `vrooli setup status`.
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
