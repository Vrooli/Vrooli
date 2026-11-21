package deployments

import (
	"fmt"
	"strings"

	"deployment-manager/profiles"
	"deployment-manager/releases"
)

// DesktopDestinationPlan is the immutable, typed routing decision for one
// desktop publication. Owner-specific coordinates are copied only after the
// destination revision has been validated against the governed profile.
type DesktopDestinationPlan struct {
	RevisionID              string
	Kind                    string
	DestinationID           string
	ConfigurationDigest     string
	Channel                 string
	ExpectedChannelRevision string
	LPBSAppKey              string
	LPBSRemoteProfile       string
	UpdateURL               string
}

// resolveDesktopDestinationPlan chooses the owner adapter from the approved
// destination kind. LPBS is the only desktop publisher wired today; other
// typed destinations fail before a pipeline owner call.
func resolveDesktopDestinationPlan(revision *releases.DestinationRevisionRecord, cfg *profiles.LPBSReleaseConfig, requestedChannel string) (DesktopDestinationPlan, error) {
	if revision == nil || strings.TrimSpace(revision.ID) == "" {
		return DesktopDestinationPlan{}, fmt.Errorf("commercial publication requires the approved destination revision")
	}
	canonical, err := revision.Revision.Canonical()
	if err != nil {
		return DesktopDestinationPlan{}, fmt.Errorf("invalid destination revision: %w", err)
	}
	if strings.TrimSpace(revision.ID) == "" {
		return DesktopDestinationPlan{}, fmt.Errorf("destination revision id is required")
	}
	channel := effectiveChannel(requestedChannel, canonical.Channel)
	if channel != canonical.Channel {
		return DesktopDestinationPlan{}, fmt.Errorf("requested channel %q does not match approved destination channel %q", channel, canonical.Channel)
	}
	plan := DesktopDestinationPlan{
		RevisionID: revision.ID, Kind: canonical.Kind, DestinationID: canonical.DestinationID,
		ConfigurationDigest: canonical.ConfigurationDigest, Channel: canonical.Channel,
		ExpectedChannelRevision: canonical.ExpectedChannelRevision,
	}
	switch canonical.Kind {
	case "lpbs":
		if cfg == nil || strings.TrimSpace(cfg.LPBSAppKey) == "" || strings.TrimSpace(cfg.LPBSRemoteProfile) == "" {
			return DesktopDestinationPlan{}, fmt.Errorf("LPBS destination requires exact app key and remote profile")
		}
		if strings.TrimSpace(cfg.LPBSRemoteProfile) != canonical.DestinationID {
			return DesktopDestinationPlan{}, fmt.Errorf("approved LPBS destination does not match the configured remote profile")
		}
		plan.LPBSAppKey = strings.TrimSpace(cfg.LPBSAppKey)
		plan.LPBSRemoteProfile = strings.TrimSpace(cfg.LPBSRemoteProfile)
		plan.UpdateURL = strings.TrimSpace(cfg.UpdateURL)
		return plan, nil
	default:
		return DesktopDestinationPlan{}, fmt.Errorf("desktop destination kind %q has no publisher adapter", canonical.Kind)
	}
}
