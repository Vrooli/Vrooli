// Package backdrop is the Android deployment scenario's narrow consumer seam
// for listing-asset composition. Screenshot capture remains here; backdrop
// generation and device framing remain owned by Backdrop Studio.
package backdrop

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	composev1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/compose"
	composeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/compose/compose_v1connect"
)

type ListingAssetRequest struct {
	SurfaceID, Arrangement, Caption string
	Width, Height                   int32
	BackdropPNG, ScreenshotPNG      []byte
}

type Client struct {
	compose composeconnect.ComposeServiceClient
}

func NewClient(httpClient *http.Client, baseURL string) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{compose: composeconnect.NewComposeServiceClient(httpClient, strings.TrimRight(baseURL, "/"))}
}

func (c *Client) ComposeListingAsset(ctx context.Context, in ListingAssetRequest) (*composev1.ComposeDeviceFrameResponse, error) {
	if c == nil || c.compose == nil {
		return nil, fmt.Errorf("backdrop: client is not configured")
	}
	if in.SurfaceID == "" || in.Width <= 0 || in.Height <= 0 || len(in.BackdropPNG) == 0 || len(in.ScreenshotPNG) == 0 {
		return nil, fmt.Errorf("backdrop: surface, positive geometry, backdrop, and screenshot are required")
	}
	resp, err := c.compose.ComposeDeviceFrame(ctx, connect.NewRequest(&composev1.ComposeDeviceFrameRequest{SurfaceId: in.SurfaceID, Arrangement: in.Arrangement, Caption: in.Caption, Width: in.Width, Height: in.Height, BackdropPng: in.BackdropPNG, ScreenshotPng: in.ScreenshotPNG}))
	if err != nil {
		return nil, fmt.Errorf("backdrop: compose listing asset: %w", err)
	}
	if resp.Msg.GetWidth() != in.Width || resp.Msg.GetHeight() != in.Height {
		return nil, fmt.Errorf("backdrop: composed geometry %dx%d does not match target %dx%d", resp.Msg.GetWidth(), resp.Msg.GetHeight(), in.Width, in.Height)
	}
	return resp.Msg, nil
}
