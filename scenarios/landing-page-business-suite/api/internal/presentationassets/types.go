// Package presentationassets owns the LPBS-side, read-only seam for released
// Backdrop Studio assets. It never trusts a document URL or qualification
// claim as delivery proof: the owner metadata and the bytes are verified
// together before a content-addressed LPBS URL is returned.
package presentationassets

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"connectrpc.com/connect"
	backdroprelease "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/release"
	backdropconnect "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/release/release_v1connect"
	presentation "landing-page-business-suite-api/internal/presentation"
)

const defaultPublicPath = "/api/v1/presentation-assets/"

// BackdropOwner is the generated typed Backdrop ReleaseService boundary. The
// verifier never accepts an owner endpoint from a presentation document.
type BackdropOwner interface {
	GetReference(context.Context, *connect.Request[backdroprelease.GetReferenceRequest]) (*connect.Response[backdroprelease.ReleasedBackdrop], error)
}

var _ BackdropOwner = (backdropconnect.ReleaseServiceClient)(nil)

// OwnerEndpointResolver is a server-side lifecycle/discovery seam. Its result
// is resolved per verification request and is used for the typed owner client
// and byte fetch; document-controlled URLs never reach it.
type OwnerEndpointResolver func(context.Context) (string, error)

// OwnerClientFactory constructs the typed owner client for the exact origin
// discovered for one request. It prevents a lifecycle restart or port change
// from leaving the verifier with a stale startup URL.
type OwnerClientFactory func(context.Context, string) (BackdropOwner, error)

type resolvedOwnerOriginContextKey struct{}

// WithResolvedOwnerOrigin pins the owner origin selected for one verifier
// request so secondary typed owner reads use the same lifecycle instance.
func WithResolvedOwnerOrigin(ctx context.Context, origin string) context.Context {
	return context.WithValue(ctx, resolvedOwnerOriginContextKey{}, origin)
}

// ResolvedOwnerOrigin returns the request-pinned owner origin, if one exists.
func ResolvedOwnerOrigin(ctx context.Context) (string, bool) {
	origin, ok := ctx.Value(resolvedOwnerOriginContextKey{}).(string)
	return origin, ok && origin != ""
}

// SurfacePolicy verifies the owner surface, dimensions, placement, and the
// LPBS crop/focal policy against the canonical surface catalog. The parent
// composition root supplies this owner later; absence fails publication.
type SurfacePolicy interface {
	Verify(context.Context, string, int, int, string, presentation.FocalPoint, string) error
}

type SurfacePolicyFunc func(context.Context, string, int, int, string, presentation.FocalPoint, string) error

func (f SurfacePolicyFunc) Verify(ctx context.Context, surface string, width, height int, cropPolicy string, focal presentation.FocalPoint, placement string) error {
	return f(ctx, surface, width, height, cropPolicy, focal, placement)
}

// Config contains only server-owned dependencies. BaseURL is deliberately not
// a config-document field; ResolveOwnerEndpoint should use api-core discovery.
type Config struct {
	Owner              BackdropOwner
	OwnerClientFactory OwnerClientFactory
	// OwnerClientOrigin is the origin used when constructing Owner. The
	// resolver must return this same origin at request time. It is required for
	// the legacy static Owner path and intentionally omitted for a dynamic
	// OwnerClientFactory.
	OwnerClientOrigin    string
	ResolveOwnerEndpoint OwnerEndpointResolver
	Cache                *Cache
	SurfacePolicy        SurfacePolicy
	HTTPClient           *http.Client
	RequestTimeout       time.Duration
	PublicPath           string
	ProviderName         string
}

// AssetProof is the verified LPBS descriptor. Hash, MIME, dimensions,
// provenance, regions, and legibility are derived from the owner response and
// fetched bytes; document claims are comparison expectations only.
type AssetProof struct {
	AssetID        string                       `json:"asset_id"`
	ReleaseID      string                       `json:"release_id"`
	PublicURL      string                       `json:"public_url"`
	ContentHash    string                       `json:"content_hash"`
	MIME           string                       `json:"mime"`
	Width          int                          `json:"width"`
	Height         int                          `json:"height"`
	Surface        string                       `json:"surface"`
	Placement      string                       `json:"placement"`
	CropPolicy     string                       `json:"crop_policy"`
	FocalPoint     presentation.FocalPoint      `json:"focal_point"`
	Provenance     presentation.AssetProvenance `json:"provenance"`
	OverlayRegions []presentation.OverlayRegion `json:"overlay_regions"`
}

// Verifier is safe to pass directly to ConfigStore.SetPresentationStorage as
// verifier.VerifyPublication. It has no mutation path into presentation
// documents or the publication pointer.
type Verifier struct {
	owner                BackdropOwner
	ownerClientFactory   OwnerClientFactory
	resolveOwnerEndpoint OwnerEndpointResolver
	cache                *Cache
	surfacePolicy        SurfacePolicy
	httpClient           *http.Client
	requestTimeout       time.Duration
	ownerClientOrigin    *url.URL
	publicPath           string
	providerName         string
}
