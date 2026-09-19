package brandpalette

import (
	"context"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	brandsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/brands"
	brandsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/brands/brands_v1connect"
)

// ResolveBaseURL locates brand-manager the way every other cross-scenario call
// in this repository does: service discovery, with an environment override for
// a deployment that pins the address.
//
// It exists because requiring BRAND_MANAGER_URL made `--brand` fail on a
// correctly running install — brand-manager was up and healthy, and the render
// still refused, because one env var nobody sets was missing. A dependency that
// discovery can find should not need configuring.
func ResolveBaseURL(ctx context.Context) (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("BRAND_MANAGER_URL")); explicit != "" {
		return explicit, nil
	}
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "brand-manager")
	if err != nil {
		return "", fmt.Errorf("brandpalette: brand-manager is not reachable: %w", err)
	}
	return baseURL, nil
}

// Fetch is the real palette-authority client. Backdrop Studio never reads the
// Brand Manager database and never stores a second palette copy.
func Fetch(ctx context.Context, httpClient connect.HTTPClient, baseURL, brandID string) (map[string]string, error) {
	if strings.TrimSpace(baseURL) == "" || strings.TrimSpace(brandID) == "" {
		return nil, fmt.Errorf("brandpalette: base URL and brand id are required")
	}
	client := brandsconnect.NewBrandsServiceClient(httpClient, strings.TrimRight(baseURL, "/"))
	resp, err := client.GetTokens(ctx, connect.NewRequest(&brandsv1.GetTokensRequest{BrandId: brandID}))
	if err != nil {
		return nil, fmt.Errorf("brandpalette: get tokens: %w", err)
	}
	out := make(map[string]string, len(resp.Msg.GetTokens()))
	for _, token := range resp.Msg.GetTokens() {
		out[token.GetName()] = token.GetValue()
	}
	return out, nil
}
