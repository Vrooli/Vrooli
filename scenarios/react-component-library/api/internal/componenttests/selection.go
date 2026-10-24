package componenttests

import (
	"sort"
	"strings"

	"react-component-library/internal/components"
)

// VersionSelection describes the bounded version set used by routine provider
// validation. Released history is immutable and is covered by the integrity
// gate, so routine behavior validation only needs the latest release and an
// explicitly open draft. Full corpus sweeps opt into all versions.
type VersionSelection struct {
	Selected             []components.ComponentVersion
	SkippedHistorical    int
	SelectedVersionCount int
}

// SelectValidationVersions returns a stable, duplicate-free version list. The
// component manifest is authoritative for the latest and draft pointers. A
// draft row without a manifest pointer is an abandoned authoring record and is
// intentionally excluded from routine validation.
func SelectValidationVersions(asset components.Component, versions []components.ComponentVersion, allVersions bool) VersionSelection {
	ordered := append([]components.ComponentVersion(nil), versions...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Version < ordered[j].Version
	})
	if allVersions {
		return VersionSelection{Selected: ordered, SelectedVersionCount: len(ordered)}
	}

	selected := make([]components.ComponentVersion, 0, 2)
	seen := map[string]struct{}{}
	add := func(version components.ComponentVersion) {
		key := strings.TrimSpace(version.Version)
		if key == "" {
			return
		}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		selected = append(selected, version)
	}
	for _, version := range ordered {
		if version.Version == asset.LatestVersion || version.Version == asset.DraftVersion {
			add(version)
		}
	}
	sort.SliceStable(selected, func(i, j int) bool {
		return selected[i].Version < selected[j].Version
	})
	return VersionSelection{
		Selected:             selected,
		SkippedHistorical:    len(ordered) - len(selected),
		SelectedVersionCount: len(selected),
	}
}

// FullVersionAuditRequested is the explicit provider-side escape hatch for a
// full historical audit. Test Genie normally sends no capability subset; the
// reserved selector is also usable by direct validation clients without
// expanding the shared validation protobuf.
func FullVersionAuditRequested(capabilities []string) bool {
	for _, capability := range capabilities {
		if strings.EqualFold(strings.TrimSpace(capability), "all-versions") {
			return true
		}
	}
	return false
}
