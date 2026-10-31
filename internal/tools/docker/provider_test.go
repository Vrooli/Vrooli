//nolint:goconst // test data deliberately reuses stable provider fixtures.
package docker

import (
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/dockerhost"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestResolveProviderLadder(t *testing.T) {
	tests := []struct {
		name     string
		host     hostreqkit.Host
		health   dockerhost.Health
		remote   string
		presence providerPresence
		provider string
		reason   string
		ready    bool
	}{
		{name: "healthy local daemon", host: hostreqkit.Host{OS: "linux"}, health: dockerhost.Health{InfoOK: true}, provider: ProviderExistingDaemon, reason: ReasonHealthyDaemon, ready: true},
		{name: "healthy remote daemon", host: hostreqkit.Host{OS: "darwin"}, health: dockerhost.Health{InfoOK: true}, remote: "ssh://docker.example", provider: ProviderRemoteDaemon, reason: ReasonReachableRemote, ready: true},
		{name: "mac colima fallback", host: hostreqkit.Host{OS: "darwin"}, remote: "ssh://unreachable", provider: ProviderColima, reason: ReasonRemoteUnavailable + ":" + ReasonColimaProvision},
		{name: "orbstack adopted", host: hostreqkit.Host{OS: "darwin"}, presence: providerPresence{OrbStack: true}, provider: ProviderOrbStack, reason: ReasonProviderPresent},
		{name: "rancher adopted", host: hostreqkit.Host{OS: "darwin"}, presence: providerPresence{RancherDesktop: true}, provider: ProviderRancherDesktop, reason: ReasonProviderPresent},
		{name: "docker desktop adopted", host: hostreqkit.Host{OS: "darwin"}, presence: providerPresence{DockerDesktop: true}, provider: ProviderDockerDesktop, reason: ReasonProviderPresent},
		{name: "linux repair", host: hostreqkit.Host{OS: "linux"}, provider: ProviderLinuxEngine, reason: ReasonLinuxRepair},
		{name: "windows manual", host: hostreqkit.Host{OS: "windows"}, provider: ProviderWindowsManual, reason: ReasonWindowsManual},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldEnv := envFn
			oldPresence := providerPathFn
			t.Cleanup(func() { envFn = oldEnv; providerPathFn = oldPresence })
			envFn = func(string) string { return tt.remote }
			providerPathFn = func(name string) bool {
				switch name {
				case "orbstack":
					return tt.presence.OrbStack
				case "rancher-desktop":
					return tt.presence.RancherDesktop
				case "docker-desktop":
					return tt.presence.DockerDesktop
				default:
					return false
				}
			}
			decision := resolveProvider(tt.host, tt.health)
			if decision.Provider != tt.provider || decision.Reason != tt.reason || decision.Ready != tt.ready {
				t.Fatalf("decision = %#v, want provider=%q reason=%q ready=%t", decision, tt.provider, tt.reason, tt.ready)
			}
		})
	}
}

func TestColimaProviderInstallsStartsAndVerifies(t *testing.T) {
	oldLookPath := hostreqkit.LookPathFn
	oldCombined := hostreqkit.CombinedOutputFn
	oldRun := hostreqkit.RunCommandFn
	oldHealth := inspectHealthFn
	oldEnv := envFn
	oldPresence := providerPathFn
	oldRecord := RecordRuntimeProviderFn
	oldPackageRecord := hostreqkit.RecordPackageInstallFn
	t.Cleanup(func() {
		hostreqkit.LookPathFn = oldLookPath
		hostreqkit.CombinedOutputFn = oldCombined
		hostreqkit.RunCommandFn = oldRun
		inspectHealthFn = oldHealth
		envFn = oldEnv
		providerPathFn = oldPresence
		RecordRuntimeProviderFn = oldRecord
		hostreqkit.RecordPackageInstallFn = oldPackageRecord
	})

	commands := []string{}
	colimaStarted := false
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "colima" {
			if colimaStarted {
				return "/opt/homebrew/bin/colima", nil
			}
			return "", errors.New("not found")
		}
		return "/opt/homebrew/bin/" + name, nil
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "docker" && len(args) > 0 && args[0] == "--version" {
			return []byte("Docker version 29.0.0"), nil
		}
		return nil, errors.New("unexpected probe")
	}
	hostreqkit.RunCommandFn = func(name string, args []string, _ hostreqkit.EnsureOptions) error {
		commands = append(commands, name+" "+strings.Join(args, " "))
		if name == "brew" && len(args) >= 2 && args[0] == "install" && args[1] == "colima" {
			return nil
		}
		if name == "colima" && len(args) == 1 && args[0] == "start" {
			colimaStarted = true
			return nil
		}
		return errors.New("unexpected command")
	}
	ready := false
	inspectHealthFn = func() dockerhost.Health {
		if ready {
			return dockerhost.Health{ClientInstalled: true, InfoOK: true}
		}
		return dockerhost.Health{ClientInstalled: true, Detail: "daemon unavailable", DaemonUnavailable: true}
	}
	envFn = func(string) string { return "" }
	providerPathFn = func(string) bool { return false }
	RecordRuntimeProviderFn = func(string, string, string, cliinstall.ObservedBefore, cliinstall.InstallAction) error {
		return nil
	}
	hostreqkit.RecordPackageInstallFn = func(hostreqkit.PackageInstallRecord) error { return nil }

	manifest := testManifest()
	h := NewHandler(manifest)
	status := h.Inspect(hostreqkit.Host{OS: "darwin", PackageManager: "brew"}, hostreqspec.ResolvedRequirement{Name: "docker", Kind: hostreqspec.KindTool, Required: true})
	if status.SelectedProvider != ProviderColima {
		t.Fatalf("selected provider = %q, want %q", status.SelectedProvider, ProviderColima)
	}
	// The health seam becomes ready after the headless provider starts.
	baseRun := hostreqkit.RunCommandFn
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		err := baseRun(name, args, opts)
		if name == "colima" && len(args) == 1 && args[0] == "start" && err == nil {
			ready = true
		}
		return err
	}
	updated, err := h.Apply(hostreqkit.Host{OS: "darwin", PackageManager: "brew"}, status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !updated.Installed || updated.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("updated status = %#v", updated)
	}
	if len(commands) != 2 || commands[0] != "brew install colima" || commands[1] != "colima start" {
		t.Fatalf("commands = %v, want brew install and colima start", commands)
	}
}
