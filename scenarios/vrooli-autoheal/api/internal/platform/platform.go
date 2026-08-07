// Package platform is the autoheal compatibility view over the control
// plane's host inventory. Host detection belongs in internal/hostinventory;
// this package retains the scenario-facing shape so existing checks and API
// clients do not need a flag-day migration.
package platform

import (
	"context"
	"runtime"

	sharedhost "github.com/vrooli/vrooli/internal/hostinventory"
)

type Type string

const (
	Linux   Type = "linux"
	Windows Type = "windows"
	MacOS   Type = "macos"
	Other   Type = "other"
)

type Capabilities struct {
	Platform            Type                               `json:"platform"`
	SupportsRDP         bool                               `json:"supportsRdp"`
	SupportsSystemd     bool                               `json:"supportsSystemd"`
	SupportsLaunchd     bool                               `json:"supportsLaunchd"`
	SupportsWindowsSvc  bool                               `json:"supportsWindowsServices"`
	IsHeadlessServer    bool                               `json:"isHeadlessServer"`
	HasDocker           bool                               `json:"hasDocker"`
	IsWSL               bool                               `json:"isWsl"`
	SupportsCloudflared bool                               `json:"supportsCloudflared"`
	SessionType         string                             `json:"sessionType,omitempty"`
	Seat                string                             `json:"seat,omitempty"`
	DisplayManager      string                             `json:"displayManager,omitempty"`
	DisplayServer       string                             `json:"displayServer,omitempty"`
	DisplayAttached     bool                               `json:"displayAttached"`
	ActiveSessionUser   string                             `json:"activeSessionUser,omitempty"`
	RemoteDesktop       sharedhost.RemoteDesktopCapability `json:"remoteDesktop"`
	WaylandAttainable   bool                               `json:"waylandAttainable"`
	WaylandReason       string                             `json:"waylandReason,omitempty"`
	ElevationMechanism  string                             `json:"elevationMechanism,omitempty"`
	IsElevated          bool                               `json:"isElevated"`
	CanElevate          bool                               `json:"canElevate"`
}

// Detect deliberately collects on each call. A process-lifetime cache made
// test seams impossible and caused autoheal to report stale host state.
func Detect() *Capabilities {
	return detect()
}

func detect() *Capabilities {
	snapshot, err := sharedhost.Collect(context.Background())
	if err != nil {
		return &Capabilities{Platform: detectPlatform()}
	}
	return capabilitiesFromSnapshot(snapshot)
}

func capabilitiesFromSnapshot(snapshot sharedhost.Snapshot) *Capabilities {
	platform := Type(snapshot.OS)
	if platform == "darwin" {
		platform = MacOS
	}
	return &Capabilities{
		Platform:            platform,
		SupportsRDP:         snapshot.SupportsRDP,
		SupportsSystemd:     snapshot.SupportsSystemd,
		SupportsLaunchd:     snapshot.SupportsLaunchd,
		SupportsWindowsSvc:  snapshot.SupportsWindowsServices,
		IsHeadlessServer:    snapshot.IsHeadless,
		HasDocker:           snapshot.RuntimeTools["docker"].Present,
		IsWSL:               snapshot.IsWSL,
		SupportsCloudflared: snapshot.SupportsCloudflared,
		SessionType:         snapshot.SessionType,
		Seat:                snapshot.Seat,
		DisplayManager:      snapshot.DisplayManager,
		DisplayServer:       snapshot.DisplayServer,
		DisplayAttached:     snapshot.DisplayAttached,
		ActiveSessionUser:   snapshot.ActiveSessionUser,
		RemoteDesktop:       snapshot.RemoteDesktop,
		WaylandAttainable:   snapshot.Wayland.Attainable,
		WaylandReason:       snapshot.Wayland.Reason,
		ElevationMechanism:  snapshot.Elevation.Mechanism,
		IsElevated:          snapshot.Elevation.Elevated,
		CanElevate:          snapshot.Elevation.CanElevate,
	}
}

func detectPlatform() Type {
	switch runtime.GOOS {
	case "linux":
		return Linux
	case "darwin":
		return MacOS
	case "windows":
		return Windows
	default:
		return Other
	}
}

func detectWSL() bool             { return detect().IsWSL }
func detectDocker() bool          { return detect().HasDocker }
func detectSystemd() bool         { return detect().SupportsSystemd }
func detectLaunchd() bool         { return detect().SupportsLaunchd }
func detectWindowsServices() bool { return detect().SupportsWindowsSvc }
func detectHeadless() bool        { return detect().IsHeadlessServer }
func detectCloudflared() bool     { return detect().SupportsCloudflared }
