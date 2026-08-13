package flows

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnchorStoreSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "anchors.json")
	first := NewAnchorStoreAt(path)
	created, err := first.Create("submit", "hello-mobile-input", []float64{.1, .2, .3, .4}, .99)
	require.NoError(t, err)

	restarted := NewAnchorStoreAt(path)
	resolved, ok := restarted.Resolve("hello-mobile-input")
	require.True(t, ok)
	require.Equal(t, created.ID, resolved.ID)
	require.Equal(t, created.Bounds, resolved.Bounds)
}

