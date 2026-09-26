package capabilities

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveDeliveryFormatsUsesExplicitSourceCapability(t *testing.T) {
	require.Equal(t, []string{"source_repository"}, ResolveDeliveryFormats([]string{"runtime.local", SourceDeliveryCapability}))
	require.Empty(t, ResolveDeliveryFormats([]string{"runtime.desktop"}))
}
