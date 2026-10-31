package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/filerouting"
	releaseconnect "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/release/release_v1connect"
	surfacesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/surfaces"
	surfacesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/surfaces/surfaces_v1connect"
	"landing-page-business-suite-api/internal/presentation"
	"landing-page-business-suite-api/internal/presentationassets"
)

const presentationAssetPublicPath = "/api/v1/presentation-assets/"

// composePresentationAssetVerification creates the LPBS-side asset verifier
// without contacting Backdrop. The owner and surface clients are constructed
// from the lifecycle origin discovered for each verification request.
func composePresentationAssetVerification(
	roots *filerouting.RoutedRoots,
	resolveOwnerEndpoint presentationassets.OwnerEndpointResolver,
	httpClient *http.Client,
) (*presentationassets.Verifier, *presentationassets.Cache, error) {
	if roots == nil {
		return nil, nil, errors.New("presentation assets: routed roots are required")
	}
	if resolveOwnerEndpoint == nil {
		resolveOwnerEndpoint = func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "backdrop-studio")
		}
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	ownerHTTPClient := apihttp.NewTestModeClient(httpClient)
	ownerFactory := func(_ context.Context, baseURL string) (presentationassets.BackdropOwner, error) {
		if _, err := secureScenarioOrigin(baseURL); err != nil {
			return nil, err
		}
		return releaseconnect.NewReleaseServiceClient(ownerHTTPClient, strings.TrimRight(baseURL, "/")), nil
	}
	surfacePolicy := catalogSurfacePolicy(resolveOwnerEndpoint, ownerHTTPClient)
	cache := presentationassets.NewCache(roots, presentationAssetPublicPath)
	verifier, err := presentationassets.NewVerifier(presentationassets.Config{
		OwnerClientFactory:   ownerFactory,
		ResolveOwnerEndpoint: resolveOwnerEndpoint,
		Cache:                cache,
		SurfacePolicy:        surfacePolicy,
		HTTPClient:           ownerHTTPClient,
		PublicPath:           presentationAssetPublicPath,
	})
	if err != nil {
		return nil, nil, err
	}
	return verifier, cache, nil
}

func catalogSurfacePolicy(resolveOwnerEndpoint presentationassets.OwnerEndpointResolver, httpClient *http.Client) presentationassets.SurfacePolicy {
	return presentationassets.SurfacePolicyFunc(func(ctx context.Context, surfaceID string, width, height int, cropPolicy string, focal presentation.FocalPoint, placement string) error {
		if _, ok := allowedPresentationCropPolicies[cropPolicy]; !ok {
			return fmt.Errorf("crop policy %q is not allowed", cropPolicy)
		}
		if math.IsNaN(focal.X) || math.IsInf(focal.X, 0) || math.IsNaN(focal.Y) || math.IsInf(focal.Y, 0) || focal.X < 0 || focal.X > 1 || focal.Y < 0 || focal.Y > 1 {
			return errors.New("focal point must be within the unit square")
		}
		if width <= 0 || height <= 0 {
			return errors.New("surface dimensions must be positive")
		}
		if width > math.MaxInt32 || height > math.MaxInt32 {
			return errors.New("surface dimensions exceed the owner wire limit")
		}
		baseURL, pinned := presentationassets.ResolvedOwnerOrigin(ctx)
		if !pinned {
			var err error
			baseURL, err = resolveOwnerEndpoint(ctx)
			if err != nil {
				return fmt.Errorf("resolve Backdrop surface catalog endpoint: %w", err)
			}
		}
		if _, err := secureScenarioOrigin(baseURL); err != nil {
			return err
		}
		client := surfacesconnect.NewSurfacesServiceClient(httpClient, strings.TrimRight(baseURL, "/"))
		response, err := client.ListSurfaces(ctx, connect.NewRequest(&surfacesv1.ListSurfacesRequest{}))
		if err != nil {
			return fmt.Errorf("list Backdrop surfaces: %w", err)
		}
		if response == nil || response.Msg == nil {
			return errors.New("Backdrop surface catalog returned an empty response")
		}
		var found *surfacesv1.Surface
		for _, candidate := range response.Msg.GetSurfaces() {
			if candidate.GetId() != surfaceID {
				continue
			}
			if found != nil {
				return fmt.Errorf("Backdrop surface catalog contains duplicate surface %q", surfaceID)
			}
			found = candidate
		}
		if found == nil {
			return fmt.Errorf("Backdrop surface %q is not in the owner catalog", surfaceID)
		}
		if found.GetWidth() <= 0 || found.GetHeight() <= 0 {
			return fmt.Errorf("Backdrop surface %q has non-positive catalog geometry", surfaceID)
		}
		if found.GetWidth() != int32(width) || found.GetHeight() != int32(height) {
			return fmt.Errorf("Backdrop surface %q geometry %dx%d does not match asset %dx%d", surfaceID, found.GetWidth(), found.GetHeight(), width, height)
		}
		for _, allowedPlacement := range found.GetPlacements() {
			if allowedPlacement == placement {
				return nil
			}
		}
		return fmt.Errorf("Backdrop surface %q does not allow placement %q", surfaceID, placement)
	})
}

var allowedPresentationCropPolicies = map[string]struct{}{
	"center":  {},
	"contain": {},
	"cover":   {},
}

func secureScenarioOrigin(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return nil, errors.New("Backdrop lifecycle endpoint is not a server-owned origin")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("Backdrop lifecycle endpoint must use HTTP or HTTPS")
	}
	return parsed, nil
}
