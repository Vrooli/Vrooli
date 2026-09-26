package infra

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/reporoot"
)

// RemoteDesktopIntent is the read-only desired-state view consumed by infra-rdp.
// Managed is deliberately separate from Experience: an operator who has not
// opted into the safeguard has not delegated recovery authority to autoheal.
type RemoteDesktopIntent struct {
	Managed    bool
	Experience string
	Provider   string
}

type RemoteDesktopIntentProvider func(context.Context) (RemoteDesktopIntent, error)

const (
	RemoteDesktopVerdictUnmanaged = "unmanaged"
	RemoteDesktopVerdictMatching  = "matching"
	RemoteDesktopVerdictDrifted   = "drifted"
)

// ResolveRemoteDesktopIntent reads the same validated manifest/operator-state
// resolution used by setup explain. It never writes operator state or host
// state. Resolution errors are returned so the caller can fail closed.
func ResolveRemoteDesktopIntent(_ context.Context, caps *platform.Capabilities) (RemoteDesktopIntent, error) {
	root := reporoot.ResolveFromOS()
	if root == "" {
		return RemoteDesktopIntent{}, fmt.Errorf("repository root is unavailable")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return RemoteDesktopIntent{}, fmt.Errorf("resolve home directory: %w", err)
	}
	platformName := ""
	if caps != nil {
		platformName = string(caps.Platform)
	}
	resolution, err := hostreq.ResolveHostRequirements(root, home, hostreq.ResolveOptions{
		Environment: "development",
		When:        "setup",
		Resources:   "none",
		Scenarios:   "none",
		Platform:    platformName,
	})
	if err != nil {
		return RemoteDesktopIntent{}, err
	}
	for _, requirement := range resolution.Safeguards {
		if requirement.Name != "remote_desktop_access" {
			continue
		}
		experience, _ := requirement.ConfigString("experience")
		provider, _ := requirement.ConfigString("provider")
		return RemoteDesktopIntent{
			Managed:    requirement.OperatorChoice == hostreqspec.OperatorChoiceOptedIn,
			Experience: experience,
			Provider:   provider,
		}, nil
	}
	return RemoteDesktopIntent{}, fmt.Errorf("remote_desktop_access is not resolved for platform %q", platformName)
}

func observedRemoteDesktopExperience(service RDPServiceInfo) string {
	switch service.Type {
	case RDPTypeGnome:
		if service.Mode == "user-shared" || service.Mode == "headless" {
			return "direct-desktop"
		}
		return "login-screen"
	case RDPTypeXrdp, RDPTypeTermService:
		return "direct-desktop"
	default:
		return "none"
	}
}

func observedRemoteDesktopProvider(service RDPServiceInfo) string {
	switch service.Type {
	case RDPTypeGnome:
		if service.Mode == "user-shared" {
			return "gnome-user-shared"
		}
		if service.Mode == "headless" {
			return "gnome-headless"
		}
		return "gnome-system"
	case RDPTypeXrdp:
		return "xrdp"
	case RDPTypeTermService:
		return "windows-termservice"
	default:
		return "none"
	}
}

func remoteDesktopProviderMatches(want string, service RDPServiceInfo) bool {
	want = strings.TrimSpace(want)
	return want == "" || want == "auto" || want == observedRemoteDesktopProvider(service)
}

func remoteDesktopIntentVerdict(intent RemoteDesktopIntent, service RDPServiceInfo) string {
	if !intent.Managed {
		return RemoteDesktopVerdictUnmanaged
	}
	if intent.Experience == "observe-only" {
		return RemoteDesktopVerdictMatching
	}
	if intent.Experience == observedRemoteDesktopExperience(service) && remoteDesktopProviderMatches(intent.Provider, service) {
		return RemoteDesktopVerdictMatching
	}
	return RemoteDesktopVerdictDrifted
}
