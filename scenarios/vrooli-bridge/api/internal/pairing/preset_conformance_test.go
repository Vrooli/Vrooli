package pairing

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scopecatalog"
	"vrooli-bridge/internal/session"
)

func TestEveryEnforcedTransportCapabilityIsPresetReachable(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../.."))
	catalog, err := scopecatalog.Build(root)
	require.NoError(t, err)
	presets := PermissionPresets(catalog)
	for _, enforced := range []string{session.TransportScope} {
		reachable := false
		for _, preset := range presets {
			for _, scope := range preset.Scopes {
				if scope == enforced {
					reachable = true
				}
			}
		}
		require.Truef(t, reachable, "enforced transport capability %q is unreachable from every permission preset", enforced)
	}
}
