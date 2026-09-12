package mandate_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"treasury/internal/mandate"
)

func TestHMACSignerIsDeterministicAndKeyScoped(t *testing.T) {
	first, err := mandate.NewHMACSigner([]byte("first-key"))
	require.NoError(t, err)
	second, err := mandate.NewHMACSigner([]byte("second-key"))
	require.NoError(t, err)

	a, err := first.Sign(context.Background(), []byte("mandate"))
	require.NoError(t, err)
	b, err := first.Sign(context.Background(), []byte("mandate"))
	require.NoError(t, err)
	c, err := second.Sign(context.Background(), []byte("mandate"))
	require.NoError(t, err)
	require.Equal(t, a, b)
	require.NotEqual(t, a, c)
}
