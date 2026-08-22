package portability

import (
	"context"
	"time"

	"github.com/vrooli/vrooli/internal/deployability"
)

// FleetReadout is the instrument projection of SDA's derived fleet verdicts.
// The instrument keeps capability-grid facts such as peerless rows local, but
// never re-resolves scenario dependency closures.
type FleetReadout struct {
	BlockedByOS     []ScenarioBlock        `json:"blocked_by_os"`
	DockerBlocked   []ScenarioBlock        `json:"docker_blocked"`
	Peerless        []ScenarioPeerless     `json:"peerless"`
	TierUpgrades    []TierUpgrade          `json:"tier_upgrades"`
	DesktopBundling DesktopBundlingVerdict `json:"desktop_bundling"`
	ManifestRoot    string                 `json:"manifest_root"`
	ComputedAt      time.Time              `json:"computed_at"`
	Available       bool                   `json:"available"`
	Reason          string                 `json:"reason,omitempty"`
}

type PlatformVerdictSource interface {
	ListPlatformFleet(ctx context.Context) (DerivedPlatformFleet, error)
}

type DerivedPlatformFleet struct {
	Scenarios     []DerivedScenarioPlatformVerdict
	DockerBlocked []DerivedDockerBlock
	TierUpgrades  []DerivedTierUpgrade
}

type DerivedTierUpgrade struct {
	Scenario           string
	HostOS             deployability.HostOS
	CurrentTier        deployability.DeliveryTier
	NextTier           deployability.DeliveryTier
	Change             string
	BlockingDependency string
}

type DerivedScenarioPlatformVerdict struct {
	Scenario           string
	HostOS             deployability.HostOS
	Status             string
	Reason             string
	BlockingDependency string
}

type DerivedDockerBlock struct {
	Scenario   string
	HostOS     deployability.HostOS
	Dependency string
	Reason     string
}

type ScenarioBlock struct {
	Scenario     string                           `json:"scenario"`
	HostOS       deployability.HostOS             `json:"host_os"`
	Dependencies []deployability.DependencyResult `json:"dependencies"`
}

type ScenarioPeerless struct {
	Scenario     string               `json:"scenario"`
	HostOS       deployability.HostOS `json:"host_os"`
	Capabilities []string             `json:"capabilities"`
}

type TierUpgrade struct {
	Scenario           string                     `json:"scenario"`
	HostOS             deployability.HostOS       `json:"host_os"`
	CurrentTier        deployability.DeliveryTier `json:"current_tier"`
	NextTier           deployability.DeliveryTier `json:"next_tier"`
	Change             string                     `json:"single_change"`
	BlockingDependency string                     `json:"blocking_dependency"`
}

type DesktopBundlingVerdict struct {
	Resources       int    `json:"resources"`
	HostRequired    int    `json:"host_required"`
	Vendorable      int    `json:"vendorable"`
	Prohibited      int    `json:"prohibited"`
	Unknown         int    `json:"unknown"`
	DatabaseBlocked bool   `json:"database_blocked"`
	Reason          string `json:"reason"`
}
