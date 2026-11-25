package hub_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"notification-hub/internal/hub"
)

// The live leg is opt-in: it needs ai-gateway and the local embedding model.
// Run with NOTIFICATION_HUB_SEMANTIC_TEST=1 when the gateway is healthy.
func TestGatewayEmbedderAgainstTheLiveGateway(t *testing.T) {
	if os.Getenv("NOTIFICATION_HUB_SEMANTIC_TEST") != "1" {
		t.Skip("set NOTIFICATION_HUB_SEMANTIC_TEST=1 to exercise ai-gateway")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	result, err := hub.NewGatewayEmbedder().Embed(ctx, []string{
		"search_query: delete the staging database",
		"search_query: remove the pre-production data store",
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Model)
	require.Greater(t, result.Dimensions, 0)
	require.Len(t, result.Vectors, 2)
	require.Len(t, result.Vectors[0], result.Dimensions)
	require.Greater(t, cosineForTest(result.Vectors[0], result.Vectors[1]), 0.5,
		"the two paraphrases should be close in the model's space")
}

func cosineForTest(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (sqrtForTest(na) * sqrtForTest(nb))
}

func sqrtForTest(v float64) float64 {
	// small, dependency-free square root good enough for a test assertion
	x := v
	for i := 0; i < 40; i++ {
		x = (x + v/x) / 2
	}
	return x
}
