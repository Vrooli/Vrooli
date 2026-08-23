package portability

import (
	"context"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/deployability"
	"github.com/vrooli/vrooli/packages/hostreq"
)

// Service is the domain's read surface. It is read-only by construction: the
// instrument reports what the manifests declare and never writes a manifest,
// starts a scenario, or reconciles a platform gap. This scenario has no
// controller letter, so an actuation path here would be a contract violation
// rather than a missing feature.
type Service struct {
	root           string
	now            func() time.Time
	platformSource PlatformVerdictSource
}

// NewService binds the domain to one repository root. The root is validated
// lazily, at read time, so a scenario started outside a repository still
// serves an explicit error instead of failing to boot — but it never serves an
// empty grid.
func NewService(root string, now func() time.Time, sources ...PlatformVerdictSource) *Service {
	if now == nil {
		now = time.Now
	}
	var source PlatformVerdictSource
	if len(sources) > 0 {
		source = sources[0]
	}
	return &Service{root: strings.TrimSpace(root), now: now, platformSource: source}
}

// Root returns the configured repository root, resolved or not.
func (s *Service) Root() string { return s.root }

// Reader validates the root and returns the typed manifest reader.
func (s *Service) Reader() (*Reader, error) { return NewReader(s.root) }

// Grid computes the whole capability grid.
func (s *Service) Grid(ctx context.Context) (Grid, error) {
	if err := ctx.Err(); err != nil {
		return Grid{}, err
	}
	reader, err := s.Reader()
	if err != nil {
		return Grid{}, err
	}
	grid, err := reader.Grid(s.now())
	if err != nil {
		return Grid{}, err
	}
	observed, err := hostreq.ListObservedSafeguards(s.root, s.now)
	if err != nil {
		return Grid{}, err
	}
	grid.ObservedSafeguards = observed
	grid = AttachObservedQualifications(grid, observed, currentHostOS())
	return grid, nil
}

// Fleet computes the fleet view over the same grid.
func (s *Service) Fleet(ctx context.Context) (FleetReadout, error) {
	if err := ctx.Err(); err != nil {
		return FleetReadout{}, err
	}
	reader, err := s.Reader()
	if err != nil {
		return FleetReadout{}, err
	}
	if s.platformSource == nil {
		return FleetReadout{ManifestRoot: s.root, ComputedAt: s.now().UTC(), Available: false, Reason: "scenario-dependency-analyzer platform verdict source is not configured"}, nil
	}
	grid, err := reader.Grid(s.now())
	if err != nil {
		return FleetReadout{}, err
	}
	derived, err := s.platformSource.ListPlatformFleet(ctx)
	if err != nil {
		return FleetReadout{ManifestRoot: s.root, ComputedAt: s.now().UTC(), Available: false, Reason: err.Error()}, nil
	}
	readout := FleetReadout{ManifestRoot: s.root, ComputedAt: s.now().UTC(), Available: true, BlockedByOS: []ScenarioBlock{}, Peerless: []ScenarioPeerless{}, TierUpgrades: []TierUpgrade{}}
	for _, item := range derived.Scenarios {
		if item.Status != "blocked" || item.BlockingDependency == "" {
			continue
		}
		readout.BlockedByOS = append(readout.BlockedByOS, ScenarioBlock{
			Scenario:     item.Scenario,
			HostOS:       item.HostOS,
			Dependencies: []deployability.DependencyResult{{Kind: "resource", Name: item.BlockingDependency, Required: true, Verdict: deployability.VerdictIneligible, Reasons: []deployability.Reason{{Code: "sda_platform_verdict", Dependency: item.BlockingDependency, Message: item.Reason}}}},
		})
	}
	for _, item := range derived.DockerBlocked {
		readout.DockerBlocked = append(readout.DockerBlocked, ScenarioBlock{Scenario: item.Scenario, HostOS: item.HostOS, Dependencies: []deployability.DependencyResult{{Kind: "resource", Name: item.Dependency, Required: true, Verdict: deployability.VerdictIneligible, Reasons: []deployability.Reason{{Code: "sda_platform_verdict", Dependency: item.Dependency, Message: item.Reason}}}}})
	}
	for _, item := range derived.TierUpgrades {
		readout.TierUpgrades = append(readout.TierUpgrades, TierUpgrade{Scenario: item.Scenario, HostOS: item.HostOS, CurrentTier: item.CurrentTier, NextTier: item.NextTier, Change: item.Change, BlockingDependency: item.BlockingDependency})
	}
	// Peerless is a capability-grid fact and remains owned by this read domain;
	// it does not resolve scenario dependencies.
	for _, entry := range grid.Capabilities {
		for _, platform := range entry.Platforms {
			if platform.Status != deployability.CapabilityPeerless {
				continue
			}
			readout.Peerless = append(readout.Peerless, ScenarioPeerless{HostOS: platform.HostOS, Capabilities: []string{entry.Capability}})
		}
	}
	return readout, nil
}
