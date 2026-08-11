package brandpalette

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	brandsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/brands"
	brandsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/brands/brands_v1connect"
)

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
