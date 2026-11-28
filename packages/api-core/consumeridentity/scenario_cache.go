package consumeridentity

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

const maxJWKSResponseBytes = 1 << 20

// NewScenarioCache creates a cache that fetches public signing keys from a
// scenario's discovered JWKS endpoint. Resolution uses each verification
// request's context; callers provide the trusted issuer and scenario slug.
func NewScenarioCache(issuer, scenarioSlug string) *Cache {
	return NewCache(issuer, DefaultLeeway, DefaultCacheTTL, scenarioJWKSFetcher(
		func(ctx context.Context, slug string) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, slug)
		},
		&http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		scenarioSlug,
	))
}

type scenarioURLResolver func(context.Context, string) (string, error)

func scenarioJWKSFetcher(resolve scenarioURLResolver, client *http.Client, scenarioSlug string) Fetcher {
	return func(req *http.Request) ([]byte, error) {
		if req == nil {
			return nil, fmt.Errorf("resolve %s signing keys: request is nil", scenarioSlug)
		}
		base, err := resolve(req.Context(), scenarioSlug)
		if err != nil {
			return nil, fmt.Errorf("resolve %s signing keys: %w", scenarioSlug, err)
		}
		keyRequest, err := http.NewRequestWithContext(req.Context(), http.MethodGet, strings.TrimRight(base, "/")+"/.well-known/jwks.json", nil)
		if err != nil {
			return nil, err
		}
		response, err := client.Do(keyRequest)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("%s signing-key endpoint returned %s", scenarioSlug, response.Status)
		}
		return io.ReadAll(io.LimitReader(response.Body, maxJWKSResponseBytes+1))
	}
}
