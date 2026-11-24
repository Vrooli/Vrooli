package bindings

import (
	"testing"

	"github.com/stretchr/testify/require"

	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
)

func TestBindingSupportsRowField(t *testing.T) {
	binding := &bindingsv1.Binding{RowFieldCandidates: []string{"items", "results"}}
	require.True(t, bindingSupportsRowField(binding, "items"))
	require.True(t, bindingSupportsRowField(binding, "results"))
	require.False(t, bindingSupportsRowField(binding, "unknown"))
	require.False(t, bindingSupportsRowField(nil, "items"))
}
