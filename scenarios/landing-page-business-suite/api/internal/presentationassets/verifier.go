package presentationassets

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	backdroprelease "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/release"
	presentation "landing-page-business-suite-api/internal/presentation"
)

func NewVerifier(cfg Config) (*Verifier, error) {
	if cfg.Owner == nil && cfg.OwnerClientFactory == nil {
		return nil, errors.New("presentation assets: Backdrop owner client or per-request client factory is required")
	}
	if cfg.ResolveOwnerEndpoint == nil {
		return nil, errors.New("presentation assets: server-side owner endpoint resolver is required")
	}
	var ownerClientOrigin *url.URL
	if cfg.Owner != nil {
		var err error
		ownerClientOrigin, err = secureOwnerBaseURL(cfg.OwnerClientOrigin)
		if err != nil {
			return nil, errors.New("presentation assets: the initial Backdrop client origin is required for a static owner client")
		}
	}
	if cfg.Cache == nil {
		return nil, errors.New("presentation assets: durable cache is required")
	}
	if cfg.SurfacePolicy == nil {
		return nil, errors.New("presentation assets: surface policy is required")
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	publicPath := cfg.PublicPath
	if strings.TrimSpace(publicPath) == "" {
		publicPath = defaultPublicPath
	}
	providerName := strings.TrimSpace(cfg.ProviderName)
	if providerName == "" {
		providerName = "backdrop-studio"
	}
	requestTimeout := cfg.RequestTimeout
	if requestTimeout == 0 {
		requestTimeout = 15 * time.Second
	}
	if requestTimeout < 0 {
		return nil, errors.New("presentation assets: request timeout must not be negative")
	}
	return &Verifier{owner: cfg.Owner, ownerClientFactory: cfg.OwnerClientFactory, resolveOwnerEndpoint: cfg.ResolveOwnerEndpoint, cache: cfg.Cache, surfacePolicy: cfg.SurfacePolicy, httpClient: client, requestTimeout: requestTimeout, ownerClientOrigin: ownerClientOrigin, publicPath: publicPath, providerName: providerName}, nil
}

// VerifyPublication is the read-only adapter expected by ConfigStore. It
// validates the complete document first, then resolves only the asset closure
// reachable from public profiles/pages. Private profiles remain structurally
// validated but their candidate assets are not made public or required to be
// available.
func (v *Verifier) VerifyPublication(ctx context.Context, document presentation.Document) error {
	if v == nil {
		return errors.New("presentation assets: verifier is nil")
	}
	if err := presentation.Validate(document); err != nil {
		return fmt.Errorf("presentation assets: document validation: %w", err)
	}
	used, err := publicAssetIDs(document)
	if err != nil {
		return fmt.Errorf("presentation assets: resolve public asset closure: %w", err)
	}
	for _, asset := range document.Assets {
		if !used[asset.ID] {
			continue
		}
		proof, err := v.ResolveAsset(ctx, asset)
		if err != nil {
			return fmt.Errorf("presentation assets: asset %q: %w", asset.ID, err)
		}
		if err := comparePublicationAsset(asset, proof); err != nil {
			return fmt.Errorf("presentation assets: asset %q publication proof: %w", asset.ID, err)
		}
	}
	return nil
}

// ResolveAsset returns the actual owner-derived proof and writes its verified
// bytes to the content-addressed LPBS cache. It does not enforce the
// publication-only document URL/proof equality, so draft importers can use it
// to obtain the expected generated URL and verified descriptor.
func (v *Verifier) ResolveAsset(ctx context.Context, asset presentation.Asset) (AssetProof, error) {
	requestCtx, cancel := context.WithTimeout(ctx, v.requestTimeout)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		return AssetProof{}, err
	}
	base, err := v.resolveOwnerEndpoint(requestCtx)
	if err != nil {
		return AssetProof{}, fmt.Errorf("resolve Backdrop owner endpoint: %w", err)
	}
	baseURL, err := secureOwnerBaseURL(base)
	if err != nil {
		return AssetProof{}, err
	}
	requestCtx = WithResolvedOwnerOrigin(requestCtx, baseURL.String())
	if v.ownerClientFactory == nil && !sameOrigin(baseURL, v.ownerClientOrigin) {
		return AssetProof{}, errors.New("Backdrop owner client origin does not match the discovered owner origin")
	}
	owner := v.owner
	if v.ownerClientFactory != nil {
		owner, err = v.ownerClientFactory(requestCtx, baseURL.String())
		if err != nil {
			return AssetProof{}, fmt.Errorf("construct Backdrop owner client: %w", err)
		}
	}
	if owner == nil {
		return AssetProof{}, errors.New("Backdrop owner client is unavailable")
	}
	response, err := owner.GetReference(requestCtx, connect.NewRequest(&backdroprelease.GetReferenceRequest{Id: asset.ReleaseRef}))
	if err != nil {
		return AssetProof{}, fmt.Errorf("get released backdrop reference: %w", err)
	}
	if response == nil || response.Msg == nil {
		return AssetProof{}, errors.New("Backdrop owner returned an empty release reference")
	}
	released := response.Msg
	if released.GetId() != asset.ReleaseRef {
		return AssetProof{}, fmt.Errorf("owner release id %q does not match document release %q", released.GetId(), asset.ReleaseRef)
	}
	if released.GetSurfaceId() != asset.Surface {
		return AssetProof{}, fmt.Errorf("owner surface %q does not match document surface %q", released.GetSurfaceId(), asset.Surface)
	}
	if released.GetWidth() != int32(asset.Width) || released.GetHeight() != int32(asset.Height) {
		return AssetProof{}, fmt.Errorf("owner dimensions %dx%d do not match document %dx%d", released.GetWidth(), released.GetHeight(), asset.Width, asset.Height)
	}
	if released.GetCandidateId() == "" || asset.Provenance.CandidateRef != released.GetCandidateId() {
		return AssetProof{}, errors.New("owner candidate provenance does not match document asset")
	}
	if released.GetJobId() == "" || asset.Provenance.JobRef != released.GetJobId() {
		return AssetProof{}, errors.New("owner job provenance does not match document asset")
	}
	if strings.TrimSpace(asset.Provenance.Provider) != v.providerName || strings.TrimSpace(asset.Provenance.JobRef) == "" {
		return AssetProof{}, errors.New("document asset provenance is incomplete")
	}
	if len(released.GetReservedRegions()) == 0 || len(released.GetReservedRegions()) != len(asset.OverlayRegions) {
		return AssetProof{}, errors.New("owner measured regions do not match document regions")
	}
	if released.GetContrastRatio() <= 0 || released.GetContrastThreshold() <= 0 || released.GetContrastRatio() < released.GetContrastThreshold() {
		return AssetProof{}, errors.New("owner legibility evidence does not pass")
	}
	for i, region := range released.GetReservedRegions() {
		expected := asset.OverlayRegions[i]
		if region.GetKind() == "" || region.GetKind() != expected.Name || !sameRegion(region.GetX(), region.GetY(), region.GetWidth(), region.GetHeight(), expected) {
			return AssetProof{}, fmt.Errorf("owner region %d does not match document region", i)
		}
	}
	if err := v.surfacePolicy.Verify(requestCtx, asset.Surface, asset.Width, asset.Height, asset.CropPolicy, asset.FocalPoint, released.GetPlacement()); err != nil {
		return AssetProof{}, fmt.Errorf("surface/crop policy: %w", err)
	}

	assetURL, err := secureOwnerAssetURL(baseURL, released.GetUri())
	if err != nil {
		return AssetProof{}, err
	}
	imageBytes, contentType, contentLength, etag, ownerHash, err := v.fetchBytes(requestCtx, assetURL)
	if err != nil {
		return AssetProof{}, err
	}
	digest := sha256.Sum256(imageBytes)
	actualHash := hex.EncodeToString(digest[:])
	if released.GetContentHash() == "" || released.GetContentHash() != actualHash || ownerHash != actualHash || asset.ContentHash != actualHash {
		return AssetProof{}, errors.New("released asset SHA-256 does not match owner/document evidence")
	}
	if contentLength != len(imageBytes) || etag != actualHash {
		return AssetProof{}, errors.New("released asset immutable headers do not match bytes")
	}
	actualWidth, actualHeight, format, err := imageMetadata(imageBytes)
	if err != nil || format != "png" || actualWidth != asset.Width || actualHeight != asset.Height || actualWidth != int(released.GetWidth()) || actualHeight != int(released.GetHeight()) {
		return AssetProof{}, errors.New("released asset MIME or dimensions do not match owner/document evidence")
	}
	if released.GetMimeType() == "" || released.GetMimeType() != contentType || contentType != asset.MIME || contentType != "image/png" {
		return AssetProof{}, fmt.Errorf("released asset MIME %q does not match expected %q", contentType, asset.MIME)
	}
	regions := make([]presentation.OverlayRegion, len(asset.OverlayRegions))
	for i := range asset.OverlayRegions {
		ownerRegion := released.GetReservedRegions()[i]
		regions[i] = presentation.OverlayRegion{Name: ownerRegion.GetKind(), X: ownerRegion.GetX(), Y: ownerRegion.GetY(), Width: ownerRegion.GetWidth(), Height: ownerRegion.GetHeight(), Measurement: presentation.LegibilityMeasurement{ContrastRatio: released.GetContrastRatio(), MinimumContrastRatio: released.GetContrastThreshold(), Threshold: released.GetContrastThreshold(), Verdict: presentation.LegibilityPass, MeasurementRef: "backdrop-release:" + released.GetId()}}
	}
	proof := AssetProof{AssetID: asset.ID, ReleaseID: released.GetId(), ContentHash: actualHash, MIME: contentType, Width: actualWidth, Height: actualHeight, Surface: released.GetSurfaceId(), Placement: released.GetPlacement(), CropPolicy: asset.CropPolicy, FocalPoint: asset.FocalPoint, Provenance: presentation.AssetProvenance{Provider: v.providerName, JobRef: released.GetJobId(), CandidateRef: released.GetCandidateId()}, OverlayRegions: regions}
	return v.cache.Put(requestCtx, proof, imageBytes)
}

func comparePublicationAsset(asset presentation.Asset, proof AssetProof) error {
	if asset.PublicURL != proof.PublicURL {
		return fmt.Errorf("document public URL %q does not match generated proof URL %q", asset.PublicURL, proof.PublicURL)
	}
	if asset.ReleaseRef != proof.ReleaseID || asset.ContentHash != proof.ContentHash || asset.MIME != proof.MIME || asset.Width != proof.Width || asset.Height != proof.Height || asset.Surface != proof.Surface || asset.CropPolicy != proof.CropPolicy || asset.FocalPoint != proof.FocalPoint {
		return errors.New("document asset descriptor does not match verified proof")
	}
	if asset.Provenance.Provider != proof.Provenance.Provider || asset.Provenance.JobRef != proof.Provenance.JobRef || asset.Provenance.CandidateRef != proof.Provenance.CandidateRef {
		return errors.New("document asset provenance does not match verified proof")
	}
	if len(asset.OverlayRegions) != len(proof.OverlayRegions) {
		return errors.New("document asset measured-region count does not match verified proof")
	}
	for i, region := range asset.OverlayRegions {
		verified := proof.OverlayRegions[i]
		if region.Name != verified.Name || !sameRegion(verified.X, verified.Y, verified.Width, verified.Height, region) ||
			region.Measurement.ContrastRatio != verified.Measurement.ContrastRatio ||
			region.Measurement.MinimumContrastRatio != verified.Measurement.MinimumContrastRatio ||
			region.Measurement.Threshold != verified.Measurement.Threshold ||
			region.Measurement.Verdict != verified.Measurement.Verdict ||
			region.Measurement.MeasurementRef != verified.Measurement.MeasurementRef {
			return fmt.Errorf("document measured region %d does not match verified proof", i)
		}
	}
	return nil
}

func (v *Verifier) fetchBytes(ctx context.Context, assetURL string) ([]byte, string, int, string, string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL, nil)
	if err != nil {
		return nil, "", 0, "", "", err
	}
	request.Header.Set("Accept-Encoding", "identity")
	client := *v.httpClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return errors.New("redirects are not allowed for released assets")
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, "", 0, "", "", fmt.Errorf("fetch released asset: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, "", 0, "", "", fmt.Errorf("released asset returned HTTP %d", response.StatusCode)
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || mediaType == "" {
		return nil, "", 0, "", "", errors.New("released asset has invalid MIME header")
	}
	if !strings.Contains(strings.ToLower(response.Header.Get("Cache-Control")), "immutable") {
		return nil, "", 0, "", "", errors.New("released asset is missing immutable cache policy")
	}
	ownerHash := strings.ToLower(strings.TrimSpace(response.Header.Get("X-Content-SHA256")))
	if !contentHashPattern.MatchString(ownerHash) {
		return nil, "", 0, "", "", errors.New("released asset is missing a valid SHA-256 header")
	}
	etag := strings.Trim(strings.TrimSpace(response.Header.Get("ETag")), "\"")
	if !contentHashPattern.MatchString(etag) {
		return nil, "", 0, "", "", errors.New("released asset is missing a content hash ETag")
	}
	contentLength, err := strconv.Atoi(response.Header.Get("Content-Length"))
	if err != nil || contentLength <= 0 {
		return nil, "", 0, "", "", errors.New("released asset is missing a valid Content-Length")
	}
	const maxAssetBytes = 32 << 20
	imageBytes, err := io.ReadAll(io.LimitReader(response.Body, maxAssetBytes+1))
	if err != nil {
		return nil, "", 0, "", "", err
	}
	if len(imageBytes) > maxAssetBytes {
		return nil, "", 0, "", "", errors.New("released asset exceeds size limit")
	}
	return imageBytes, mediaType, contentLength, etag, ownerHash, nil
}

func secureOwnerBaseURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("Backdrop owner endpoint is not a secure server-owned origin")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("Backdrop owner endpoint must use HTTP or HTTPS")
	}
	return parsed, nil
}

func sameOrigin(left, right *url.URL) bool {
	return left != nil && right != nil && strings.EqualFold(left.Scheme, right.Scheme) && strings.EqualFold(left.Host, right.Host)
}

func secureOwnerAssetURL(base *url.URL, raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.IsAbs() || parsed.Host != "" || strings.HasPrefix(raw, "//") || parsed.RawQuery != "" || parsed.Fragment != "" || strings.Contains(parsed.Path, "..") {
		return "", errors.New("released asset URI must be a same-origin relative owner path")
	}
	if !strings.HasPrefix(parsed.Path, "/api/v1/backdrops/") {
		return "", errors.New("released asset URI is outside the Backdrop asset endpoint")
	}
	resolved := base.ResolveReference(parsed)
	if resolved.Scheme != base.Scheme || resolved.Host != base.Host || resolved.User != nil {
		return "", errors.New("released asset URI escaped the server-owned owner origin")
	}
	return resolved.String(), nil
}

func imageMetadata(data []byte) (int, int, string, error) {
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	return config.Width, config.Height, format, err
}

func sameRegion(x, y, width, height float64, expected presentation.OverlayRegion) bool {
	const epsilon = 0.000001
	return abs(x-expected.X) < epsilon && abs(y-expected.Y) < epsilon && abs(width-expected.Width) < epsilon && abs(height-expected.Height) < epsilon
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func publicAssetIDs(document presentation.Document) (map[string]bool, error) {
	assets := map[string]bool{}
	views, err := presentation.PublicViews(document)
	if err != nil {
		return nil, fmt.Errorf("canonical public views: %w", err)
	}
	for _, view := range views {
		for _, asset := range view.Assets {
			assets[asset.ID] = true
		}
	}
	return assets, nil
}
