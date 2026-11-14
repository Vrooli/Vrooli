package adapters

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewAllIncludesBuiltInAdapters(t *testing.T) {
	all := NewAll()
	ids := make([]string, 0, len(all))
	for _, adapter := range all {
		ids = append(ids, adapter.ID())
	}
	sort.Strings(ids)
	require.Equal(t, []string{"imessage", "in-app", "slack", "telegram"}, ids)
}
