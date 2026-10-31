package pairing

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scopecatalog"
)

func TestPermissionPresetsAreCatalogDerivedAndNarrowByDefault(t *testing.T) {
	catalog := scopecatalog.Catalog{Scopes: []scopecatalog.Scope{
		{Value: "system-monitor:read", Effect: scopecatalog.EffectRead},
		{Value: "system-monitor:write", Effect: scopecatalog.EffectWrite},
		{Value: "system-monitor:destroy", Effect: scopecatalog.EffectDestructive},
		{Value: "*:write", Effect: scopecatalog.EffectWrite},
	}}
	presets := PermissionPresets(catalog)
	require.Len(t, presets, 3)
	require.Equal(t, []string{"system-monitor:read"}, presets[0].Scopes)
	require.Equal(t, []string{"system-monitor:read", "system-monitor:write"}, presets[1].Scopes)
	require.Equal(t, []string{"system-monitor:destroy", "system-monitor:read", "system-monitor:write"}, presets[2].Scopes)
	require.NotContains(t, presets[0].Scopes, "*:write")
	got, ok := ScopesForPreset(catalog, PresetReadOnly)
	require.True(t, ok)
	require.Equal(t, presets[0].Scopes, got)
}
