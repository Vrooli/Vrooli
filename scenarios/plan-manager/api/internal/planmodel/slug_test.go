package planmodel

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTruncateSlug(t *testing.T) {
	t.Run("short slugs pass through", func(t *testing.T) {
		require.Equal(t, "short-slug", TruncateSlug("short-slug", 60))
	})
	t.Run("truncates at a word boundary", func(t *testing.T) {
		long := "plan-manager-authoring-form-not-wizard-honest-finalize-and-batch-submission"
		out := TruncateSlug(long, 60)
		require.LessOrEqual(t, len(out), 60)
		require.False(t, strings.HasSuffix(out, "-"))
		require.True(t, strings.HasPrefix(long, out))
		require.Equal(t, byte('-'), long[len(out)], "cut must land on a word boundary")
	})
	t.Run("single long word hard-cuts", func(t *testing.T) {
		out := TruncateSlug(strings.Repeat("a", 100), 60)
		require.Len(t, out, 60)
	})
	t.Run("exact boundary is untouched", func(t *testing.T) {
		v := strings.Repeat("a", 60)
		require.Equal(t, v, TruncateSlug(v, 60))
	})
}
