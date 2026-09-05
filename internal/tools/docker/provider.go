package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/dockerhost"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	ProviderExistingDaemon = "existing-daemon"
	ProviderRemoteDaemon   = "remote-daemon"
	ProviderColima         = "colima"
	ProviderOrbStack       = "orbstack"
	ProviderRancherDesktop = "rancher-desktop"
	ProviderDockerDesktop  = "docker-desktop"
	ProviderLinuxEngine    = "linux-engine"
	ProviderWindowsManual  = "windows-manual"
)

const (
	ReasonHealthyDaemon       = "healthy_daemon"
	ReasonReachableRemote     = "reachable_remote_daemon"
	ReasonRemoteUnavailable   = "remote_daemon_unavailable"
	ReasonColimaProvision     = "colima_provision"
	ReasonProviderPresent     = "provider_present_manual_start"
	ReasonLinuxRepair         = "linux_daemon_repair"
	ReasonWindowsManual       = "windows_manual_runtime"
	ReasonUnsupportedPlatform = "unsupported_platform"
)

// ProviderDecision is the pure result of the container-runtime ladder. The
// resolver does not execute commands or touch a daemon; Apply owns provider
// repair/provisioning after a decision has been made.
type ProviderDecision struct {
	Provider       string
	Endpoint       string
	Reason         string
	ManualAction   string
	ObservedBefore cliinstall.ObservedBefore
	Action         cliinstall.InstallAction
	Ready          bool
}

type providerPresence struct {
	OrbStack       bool
	RancherDesktop bool
	DockerDesktop  bool
}

var (
	inspectHealthFn = dockerhost.InspectHealth
	envFn           = os.Getenv
	providerPathFn  = defaultProviderPath
	// Runtime owns the production ledger callback. A no-op default keeps the
	// custom handler unit-testable without writing the current user's ledger.
	RecordRuntimeProviderFn = func(string, string, string, cliinstall.ObservedBefore, cliinstall.InstallAction) error {
		return nil
	}
)

func resolveProvider(host hostreqkit.Host, health dockerhost.Health) ProviderDecision {
	remoteEndpoint := strings.TrimSpace(envFn("DOCKER_HOST"))
	if health.InfoOK {
		if remoteEndpoint != "" {
			return ProviderDecision{
				Provider: ProviderRemoteDaemon, Endpoint: remoteEndpoint, Reason: ReasonReachableRemote,
				ObservedBefore: cliinstall.ObservedPresent, Action: cliinstall.ActionAdopted, Ready: true,
			}
		}
		return ProviderDecision{
			Provider: ProviderExistingDaemon, Endpoint: "docker://local", Reason: ReasonHealthyDaemon,
			ObservedBefore: cliinstall.ObservedPresent, Action: cliinstall.ActionAdopted, Ready: true,
		}
	}

	presence := providerPresence{
		OrbStack:       providerPathFn("orbstack"),
		RancherDesktop: providerPathFn("rancher-desktop"),
		DockerDesktop:  providerPathFn("docker-desktop"),
	}
	if remoteEndpoint != "" {
		// An unreachable remote endpoint is not silently adopted. The platform
		// provider may still be a valid fallback, but the reason remains visible.
		if decision, ok := platformProviderDecision(host, presence); ok {
			decision.Reason = ReasonRemoteUnavailable + ":" + decision.Reason
			return decision
		}
		return ProviderDecision{Provider: ProviderRemoteDaemon, Endpoint: remoteEndpoint, Reason: ReasonRemoteUnavailable, ManualAction: "repair or remove DOCKER_HOST, then retry"}
	}
	if decision, ok := platformProviderDecision(host, presence); ok {
		return decision
	}
	return ProviderDecision{
		Provider: ProviderWindowsManual, Endpoint: "manual://container-runtime", Reason: ReasonUnsupportedPlatform,
		ManualAction: fmt.Sprintf("install and start a Docker-compatible container runtime for %s, then retry", host.OS),
	}
}

func platformProviderDecision(host hostreqkit.Host, presence providerPresence) (ProviderDecision, bool) {
	switch strings.ToLower(strings.TrimSpace(host.OS)) {
	case string(hostreqspec.PlatformDarwin), "macos":
		switch {
		case presence.OrbStack:
			return ProviderDecision{Provider: ProviderOrbStack, Endpoint: "docker://orbstack", Reason: ReasonProviderPresent, ManualAction: "start OrbStack once, then retry", ObservedBefore: cliinstall.ObservedPresent, Action: cliinstall.ActionAdopted}, true
		case presence.RancherDesktop:
			return ProviderDecision{Provider: ProviderRancherDesktop, Endpoint: "docker://rancher-desktop", Reason: ReasonProviderPresent, ManualAction: "start Rancher Desktop once, then retry", ObservedBefore: cliinstall.ObservedPresent, Action: cliinstall.ActionAdopted}, true
		case presence.DockerDesktop:
			return ProviderDecision{Provider: ProviderDockerDesktop, Endpoint: "docker://docker-desktop", Reason: ReasonProviderPresent, ManualAction: "start Docker Desktop once, then retry", ObservedBefore: cliinstall.ObservedPresent, Action: cliinstall.ActionAdopted}, true
		default:
			return ProviderDecision{Provider: ProviderColima, Endpoint: "docker://colima", Reason: ReasonColimaProvision, ObservedBefore: cliinstall.ObservedAbsent, Action: cliinstall.ActionInstalled}, true
		}
	case string(hostreqspec.PlatformLinux):
		return ProviderDecision{Provider: ProviderLinuxEngine, Endpoint: "docker://local", Reason: ReasonLinuxRepair, ObservedBefore: cliinstall.ObservedPresent, Action: cliinstall.ActionAdopted}, true
	case string(hostreqspec.PlatformWindows):
		return ProviderDecision{Provider: ProviderWindowsManual, Endpoint: "manual://container-runtime", Reason: ReasonWindowsManual, ManualAction: "install and start Docker Desktop or another Docker-compatible runtime, then retry"}, true
	default:
		return ProviderDecision{}, false
	}
}

func defaultProviderPath(provider string) bool {
	var names []string
	switch provider {
	case "orbstack":
		names = []string{"OrbStack.app"}
	case "rancher-desktop":
		names = []string{"Rancher Desktop.app"}
	case "docker-desktop":
		names = []string{"Docker.app"}
	default:
		return false
	}
	home, _ := os.UserHomeDir()
	for _, root := range []string{"/Applications", filepath.Join(home, "Applications")} {
		for _, name := range names {
			if _, err := os.Stat(filepath.Join(root, name)); err == nil {
				return true
			}
		}
	}
	return false
}

func recordRuntimeProvider(decision ProviderDecision) error {
	if !decision.Ready {
		return nil
	}
	node, _ := os.Hostname()
	return RecordRuntimeProviderFn(decision.Provider, decision.Endpoint, node, decision.ObservedBefore, decision.Action)
}
