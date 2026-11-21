package audiotools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFixedURLResolver(t *testing.T) {
	url, err := FixedURLResolver("http://x:1").ResolveURL(context.Background())
	require.NoError(t, err)
	require.Equal(t, "http://x:1", url)
}

func TestDefaultResolverImplementsInterface(t *testing.T) {
	var _ URLResolver = DefaultResolver()
}
