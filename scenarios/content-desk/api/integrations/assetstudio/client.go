// Package assetstudio is Content Desk's metadata-only Asset Studio boundary.
package assetstudio

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	studiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/asset-studio/v1/studio"
	studioconnect "github.com/vrooli/vrooli/packages/proto/gen/go/asset-studio/v1/studio/studio_v1connect"
)

const scenarioID = "asset-studio"

type (
	Reference struct {
		ID, AltText, MediaType string
		Width, Height          int32
	}
	Resolver interface {
		ResolveReleasedAsset(context.Context, string) (Reference, error)
	}
	Client struct {
		resolver interface {
			ResolveScenarioURLDefault(context.Context, string) (string, error)
		}
		http *http.Client
	}
)

func NewClient() *Client {
	return &Client{resolver: discovery.NewResolver(discovery.ResolverConfig{}), http: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) ResolveReleasedAsset(ctx context.Context, assetID string) (Reference, error) {
	if assetID == "" {
		return Reference{}, fmt.Errorf("asset id is required")
	}
	if c == nil || c.resolver == nil || c.http == nil {
		return Reference{}, fmt.Errorf("asset studio integration is not configured")
	}
	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	baseURL, err := c.resolver.ResolveScenarioURLDefault(callCtx, scenarioID)
	if err != nil {
		return Reference{}, fmt.Errorf("resolve asset studio: %w", err)
	}
	response, err := studioconnect.NewStudioServiceClient(c.http, strings.TrimRight(baseURL, "/")).GetReleasedAssetReference(callCtx, connect.NewRequest(&studiov1.GetReleasedAssetReferenceRequest{AssetId: assetID}))
	if err != nil {
		return Reference{}, fmt.Errorf("get released asset reference: %w", err)
	}
	if response == nil || response.Msg == nil || response.Msg.Asset == nil {
		return Reference{}, fmt.Errorf("asset studio returned no released asset reference")
	}
	asset := response.Msg.Asset
	return Reference{ID: asset.Id, AltText: asset.AltText, MediaType: asset.MediaType, Width: asset.Width, Height: asset.Height}, nil
}
