package portability

import (
	"strings"

	"github.com/vrooli/vrooli/internal/deployability"
	portabilityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/portability"
	internalportability "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/portability"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Every projection below resolves its enum through the generated value table
// rather than a hand-maintained parallel switch. A parallel switch returns the
// zero value on a miss, and an unlabelled enum value is indistinguishable from
// a deliberate UNSPECIFIED — which is how a shipped projection can render as
// "unspecified" on every typed surface without anything failing.

func protoEnum[T ~int32](values map[string]int32, prefix, token string) T {
	name := prefix + strings.ToUpper(strings.ReplaceAll(token, "-", "_"))
	if value, ok := values[name]; ok {
		return T(value)
	}
	var unspecified T
	return unspecified
}

func protoHostOS(hostOS deployability.HostOS) portabilityv1.HostOS {
	return protoEnum[portabilityv1.HostOS](portabilityv1.HostOS_value, "HOST_OS_", string(hostOS))
}

func protoStatus(status deployability.CapabilityResolutionStatus) portabilityv1.ResolutionStatus {
	return protoEnum[portabilityv1.ResolutionStatus](portabilityv1.ResolutionStatus_value, "RESOLUTION_STATUS_", string(status))
}

func protoQualification(qualification deployability.Qualification) portabilityv1.Qualification {
	return protoEnum[portabilityv1.Qualification](portabilityv1.Qualification_value, "QUALIFICATION_", string(qualification))
}

func protoSituation(situation internalportability.CapabilitySituation) portabilityv1.CapabilitySituation {
	return protoEnum[portabilityv1.CapabilitySituation](portabilityv1.CapabilitySituation_value, "CAPABILITY_SITUATION_", string(situation))
}

func protoVerdict(verdict deployability.Verdict) portabilityv1.Verdict {
	return protoEnum[portabilityv1.Verdict](portabilityv1.Verdict_value, "VERDICT_", string(verdict))
}

// protoTier strips the tier vocabulary's ordinal prefix ("tier-2-desktop")
// before resolving, because the enum names the tier, not its position.
func protoTier(tier deployability.DeliveryTier) portabilityv1.DeliveryTier {
	token := string(tier)
	if parts := strings.SplitN(token, "-", 3); len(parts) == 3 && parts[0] == "tier" {
		token = parts[2]
	}
	return protoEnum[portabilityv1.DeliveryTier](portabilityv1.DeliveryTier_value, "DELIVERY_TIER_", token)
}

func protoPlatform(platform internalportability.PlatformEntry) *portabilityv1.PlatformEntry {
	return &portabilityv1.PlatformEntry{
		HostOs:              protoHostOS(platform.HostOS),
		Status:              protoStatus(platform.Status),
		Qualification:       protoQualification(platform.Qualification),
		Implementer:         platform.Implementer,
		Mechanism:           platform.Mechanism,
		Reason:              platform.Reason,
		QualificationReason: platform.QualificationReason,
		HasImplementation:   platform.HasImplementation,
	}
}

func protoEntry(entry internalportability.Entry) *portabilityv1.CapabilityEntry {
	out := &portabilityv1.CapabilityEntry{
		Capability:      entry.Capability,
		Situation:       protoSituation(entry.Situation),
		SituationReason: entry.SituationReason,
		Platforms:       make([]*portabilityv1.PlatformEntry, 0, len(entry.Platforms)),
	}
	for _, platform := range entry.Platforms {
		out.Platforms = append(out.Platforms, protoPlatform(platform))
	}
	return out
}

func protoGrid(grid internalportability.Grid, entries []internalportability.Entry) *portabilityv1.Grid {
	out := &portabilityv1.Grid{
		ManifestRoot:  grid.ManifestRoot,
		ManifestsRead: int32(grid.ManifestsRead),
		ComputedAt:    timestamppb.New(grid.ComputedAt),
		Capabilities:  make([]*portabilityv1.CapabilityEntry, 0, len(entries)),
	}
	for _, entry := range entries {
		out.Capabilities = append(out.Capabilities, protoEntry(entry))
	}
	return out
}

func protoDependency(dependency deployability.DependencyResult) *portabilityv1.DependencyResult {
	out := &portabilityv1.DependencyResult{
		Kind:     dependency.Kind,
		Name:     dependency.Name,
		Required: dependency.Required,
		Verdict:  protoVerdict(dependency.Verdict),
		Reasons:  make([]*portabilityv1.DependencyReason, 0, len(dependency.Reasons)),
	}
	for _, reason := range dependency.Reasons {
		out.Reasons = append(out.Reasons, &portabilityv1.DependencyReason{
			Code: reason.Code, Dependency: reason.Dependency,
			Requirement: reason.Requirement, Message: reason.Message,
		})
	}
	return out
}

func protoBlocks(blocks []internalportability.ScenarioBlock) []*portabilityv1.ScenarioBlock {
	out := make([]*portabilityv1.ScenarioBlock, 0, len(blocks))
	for _, block := range blocks {
		item := &portabilityv1.ScenarioBlock{
			Scenario:     block.Scenario,
			HostOs:       protoHostOS(block.HostOS),
			Dependencies: make([]*portabilityv1.DependencyResult, 0, len(block.Dependencies)),
		}
		for _, dependency := range block.Dependencies {
			item.Dependencies = append(item.Dependencies, protoDependency(dependency))
		}
		out = append(out, item)
	}
	return out
}

func protoFleet(readout internalportability.FleetReadout) *portabilityv1.FleetReadout {
	out := &portabilityv1.FleetReadout{
		BlockedByOs:   protoBlocks(readout.BlockedByOS),
		DockerBlocked: protoBlocks(readout.DockerBlocked),
		Peerless:      make([]*portabilityv1.ScenarioPeerless, 0, len(readout.Peerless)),
		TierUpgrades:  make([]*portabilityv1.TierUpgrade, 0, len(readout.TierUpgrades)),
		DesktopBundling: &portabilityv1.DesktopBundlingVerdict{
			Resources:       int32(readout.DesktopBundling.Resources),
			HostRequired:    int32(readout.DesktopBundling.HostRequired),
			Vendorable:      int32(readout.DesktopBundling.Vendorable),
			Prohibited:      int32(readout.DesktopBundling.Prohibited),
			Unknown:         int32(readout.DesktopBundling.Unknown),
			DatabaseBlocked: readout.DesktopBundling.DatabaseBlocked,
			Reason:          readout.DesktopBundling.Reason,
		},
		ManifestRoot: readout.ManifestRoot,
		ComputedAt:   timestamppb.New(readout.ComputedAt),
	}
	for _, item := range readout.Peerless {
		out.Peerless = append(out.Peerless, &portabilityv1.ScenarioPeerless{
			Scenario: item.Scenario, HostOs: protoHostOS(item.HostOS), Capabilities: item.Capabilities,
		})
	}
	for _, item := range readout.TierUpgrades {
		out.TierUpgrades = append(out.TierUpgrades, &portabilityv1.TierUpgrade{
			Scenario: item.Scenario, HostOs: protoHostOS(item.HostOS),
			CurrentTier: protoTier(item.CurrentTier), NextTier: protoTier(item.NextTier),
			SingleChange: item.Change, BlockingDependency: item.BlockingDependency,
		})
	}
	return out
}
