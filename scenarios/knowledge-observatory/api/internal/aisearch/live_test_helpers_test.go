package aisearch

import (
	"context"
	"testing"

	pkg "github.com/vrooli/ai-go/search"
)

func liveEmbeddingPolicy(t *testing.T, ctx context.Context) pkg.EmbeddingPolicy {
	t.Helper()
	policy, err := pkg.ResolveEmbeddingPolicy(ctx, pkg.DefaultEmbedRole)
	if err != nil {
		t.Fatalf("resolve live embedding policy: %v", err)
	}
	return policy
}
