package lifecycle

import (
	"slices"

	"github.com/vrooli/api-core/coreset"
	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// Preserve the minimal recovery plane when oomd selects a descendant of the
// user manager. A slice's lack of managed-OOM monitoring is not protection
// from its ancestor. Avoid is scoped to live recovery services and still
// permits last-resort reclamation; agent and shadow workloads remain ordinary
// candidates. The canonical recovery-plane declaration owns membership.
func recoveryServiceContainment(slug string) platform.Containment {
	key, err := scenarioruntime.ParseInstanceKey(slug, "")
	if err != nil || key.Normalize().Variant != scenarioruntime.DefaultVariant {
		return platform.Containment{}
	}
	return platform.Containment{ManagedOOMAvoid: slices.Contains(coreset.RequiredOperationalScenarios, key.Scenario)}
}
