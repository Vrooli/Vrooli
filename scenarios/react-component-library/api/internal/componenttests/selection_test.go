package componenttests

import (
	"testing"

	"react-component-library/internal/components"

	"github.com/stretchr/testify/require"
)

func TestSelectValidationVersionsKeepsLatestAndDraft(t *testing.T) {
	asset := components.Component{LatestVersion: "2.0.0", DraftVersion: "2.1.0-draft.1"}
	versions := []components.ComponentVersion{
		{Version: "1.0.0", Status: components.VersionStatusReleased},
		{Version: "2.1.0-draft.1", Status: components.VersionStatusDraft},
		{Version: "2.0.0", Status: components.VersionStatusReleased},
	}

	selection := SelectValidationVersions(asset, versions, false)
	require.Equal(t, []string{"2.0.0", "2.1.0-draft.1"}, versionNames(selection.Selected))
	require.Equal(t, 1, selection.SkippedHistorical)

	audit := SelectValidationVersions(asset, versions, true)
	require.Equal(t, []string{"1.0.0", "2.0.0", "2.1.0-draft.1"}, versionNames(audit.Selected))
	require.Zero(t, audit.SkippedHistorical)
}

func TestSelectValidationVersionsIgnoresUnownedDraftWhenPointerIsEmpty(t *testing.T) {
	selection := SelectValidationVersions(
		components.Component{LatestVersion: "1.0.0"},
		[]components.ComponentVersion{
			{Version: "1.0.0", Status: components.VersionStatusReleased},
			{Version: "1.1.0-draft.1", Status: components.VersionStatusDraft},
			{Version: "0.9.0", Status: components.VersionStatusReleased},
		},
		false,
	)
	require.Equal(t, []string{"1.0.0"}, versionNames(selection.Selected))
}

func TestFullVersionAuditSelectorIsExplicit(t *testing.T) {
	require.True(t, FullVersionAuditRequested([]string{"all-versions"}))
	require.True(t, FullVersionAuditRequested([]string{"contracts", "ALL-VERSIONS"}))
	require.False(t, FullVersionAuditRequested([]string{"contracts"}))
}

func versionNames(versions []components.ComponentVersion) []string {
	names := make([]string, 0, len(versions))
	for _, version := range versions {
		names = append(names, version.Version)
	}
	return names
}
