package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	backdroprelease "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/release"
	releaseconnect "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/release/release_v1connect"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/shared"
	surfacesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/surfaces"
	surfacesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/surfaces/surfaces_v1connect"
	"landing-page-business-suite-api/internal/presentation"
)

type compositionReleaseService struct {
	release *backdroprelease.ReleasedBackdrop
}

func (s compositionReleaseService) Release(context.Context, *connect.Request[backdroprelease.ReleaseRequest]) (*connect.Response[backdroprelease.ReleasedBackdrop], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("release fixture is read-only"))
}

func (s compositionReleaseService) GetReference(context.Context, *connect.Request[backdroprelease.GetReferenceRequest]) (*connect.Response[backdroprelease.ReleasedBackdrop], error) {
	return connect.NewResponse(s.release), nil
}

type compositionSurfaceService struct {
	surfaces []*surfacesv1.Surface
}

func (s compositionSurfaceService) ListSurfaces(context.Context, *connect.Request[surfacesv1.ListSurfacesRequest]) (*connect.Response[surfacesv1.ListSurfacesResponse], error) {
	return connect.NewResponse(&surfacesv1.ListSurfacesResponse{Surfaces: s.surfaces}), nil
}

func compositionPNG(t *testing.T, width, height int) ([]byte, string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 20, G: uint8(x * 10), B: uint8(y * 10), A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	body := encoded.Bytes()
	digest := sha256.Sum256(body)
	return body, hex.EncodeToString(digest[:])
}

func compositionBackdropServer(t *testing.T, body []byte, hash string, width, height int, observedHeaders *atomic.Int32) *httptest.Server {
	t.Helper()
	releasePath, releaseHandler := releaseconnect.NewReleaseServiceHandler(compositionReleaseService{release: &backdroprelease.ReleasedBackdrop{
		Id: "release-1", CandidateId: "candidate-1", JobId: "job-1", MimeType: "image/png", ContentHash: hash,
		SurfaceId: "web.hero", Placement: "full_bleed", Width: int32(width), Height: int32(height), ContrastRatio: 7, ContrastThreshold: 4.5,
		Uri: "/api/v1/backdrops/release-1/asset", ReservedRegions: []*shared.ReservedRegion{{X: 0, Y: 0, Width: 1, Height: 1, Kind: "copy"}},
	}})
	surfacePath, surfaceHandler := surfacesconnect.NewSurfacesServiceHandler(compositionSurfaceService{surfaces: []*surfacesv1.Surface{{
		Id: "web.hero", Width: int32(width), Height: int32(height), Placements: []string{"full_bleed"}, Kind: "product", Authority: "fixture", ConfirmedOn: "2026-09-15",
	}}})
	mux := http.NewServeMux()
	mux.Handle(releasePath, releaseHandler)
	mux.Handle(surfacePath, surfaceHandler)
	mux.HandleFunc("/api/v1/backdrops/release-1/asset", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.Header().Set("ETag", `"`+hash+`"`)
		w.Header().Set("X-Content-SHA256", hash)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		_, _ = w.Write(body)
	})
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Vrooli-Test-Mode") == "1" {
			observedHeaders.Add(1)
		}
		mux.ServeHTTP(w, r)
	}))
}

func compositionSurfaceCatalogServer(t *testing.T, surfaces []*surfacesv1.Surface) *httptest.Server {
	t.Helper()
	surfacePath, surfaceHandler := surfacesconnect.NewSurfacesServiceHandler(compositionSurfaceService{surfaces: surfaces})
	mux := http.NewServeMux()
	mux.Handle(surfacePath, surfaceHandler)
	return httptest.NewServer(mux)
}

func compositionAsset(hash string, width, height int) presentation.Asset {
	return presentation.Asset{
		ID: "hero-art", ReleaseRef: "release-1", ContentHash: hash, Width: width, Height: height, MIME: "image/png", Surface: "web.hero", CropPolicy: "center",
		FocalPoint: presentation.FocalPoint{X: 0.5, Y: 0.5}, Provenance: presentation.AssetProvenance{Provider: "backdrop-studio", JobRef: "job-1", CandidateRef: "candidate-1"},
		OverlayRegions: []presentation.OverlayRegion{{Name: "copy", X: 0, Y: 0, Width: 1, Height: 1, Measurement: presentation.LegibilityMeasurement{ContrastRatio: 7, MinimumContrastRatio: 4.5, Threshold: 4.5, Verdict: presentation.LegibilityPass, MeasurementRef: "backdrop-release:release-1"}}},
	}
}

func TestComposePresentationAssetVerificationDefersOwnerAccessAndPropagatesLease(t *testing.T) {
	body, hash := compositionPNG(t, 1440, 720)
	var ownerHeadersA, ownerHeadersB atomic.Int32
	serverA := compositionBackdropServer(t, body, hash, 1440, 720, &ownerHeadersA)
	defer serverA.Close()
	serverB := compositionBackdropServer(t, body, hash, 1440, 720, &ownerHeadersB)
	defer serverB.Close()
	var resolverCalls atomic.Int32
	roots := filerouting.New(storage.Paths{DataDir: t.TempDir(), CacheDir: t.TempDir(), StateDir: t.TempDir(), LogsDir: t.TempDir(), ConfigDir: t.TempDir(), TestRunsDir: t.TempDir()})
	verifier, cache, err := composePresentationAssetVerification(roots, func(context.Context) (string, error) {
		if resolverCalls.Add(1) == 1 {
			return serverA.URL, nil
		}
		return serverB.URL, nil
	}, serverA.Client())
	if err != nil {
		t.Fatal(err)
	}
	if resolverCalls.Load() != 0 {
		t.Fatalf("composition contacted Backdrop at startup: %d calls", resolverCalls.Load())
	}
	if cache == nil || verifier == nil {
		t.Fatal("composition returned nil capability")
	}
	if _, err := roots.InstallLeasedTestRoots("presentation-asset-composition", 0, true); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = roots.ClearTestRoots("presentation-asset-composition") }()
	if _, err := verifier.ResolveAsset(database.WithTestMode(context.Background()), compositionAsset(hash, 1440, 720)); err != nil {
		t.Fatal(err)
	}
	if resolverCalls.Load() != 1 {
		t.Fatalf("request-time lifecycle discovery calls = %d, want one pinned origin", resolverCalls.Load())
	}
	if ownerHeadersA.Load() != 3 || ownerHeadersB.Load() != 0 {
		t.Fatalf("owner requests mixed lifecycle origins: origin A=%d, origin B=%d", ownerHeadersA.Load(), ownerHeadersB.Load())
	}
	// [REQ:LP-PRES-009] Exercise the actual mounted GET/HEAD route, not merely
	// the cache helper. A test-populated artifact must stay in its leased root.
	server := &Server{router: mux.NewRouter(), presentationAssetHandler: cache.Handler()}
	registerPresentationAssetRoutes(server)
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		request := httptest.NewRequest(method, cache.PublicURL(hash), nil).WithContext(database.WithTestMode(context.Background()))
		response := httptest.NewRecorder()
		server.router.ServeHTTP(response, request)
		if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "image/png" {
			t.Fatalf("mounted %s delivery = %d, headers=%v", method, response.Code, response.Header())
		}
		if method == http.MethodGet && !bytes.Equal(response.Body.Bytes(), body) {
			t.Fatal("mounted GET did not serve the verified owner bytes")
		}
		if method == http.MethodHead && response.Body.Len() != 0 {
			t.Fatal("mounted HEAD returned image bytes")
		}
	}
	primaryRead := httptest.NewRecorder()
	server.router.ServeHTTP(primaryRead, httptest.NewRequest(http.MethodGet, cache.PublicURL(hash), nil))
	if primaryRead.Code != http.StatusNotFound {
		t.Fatalf("leased artifact became visible in primary storage: %d", primaryRead.Code)
	}
}

func TestCatalogSurfacePolicyUsesTypedCatalogAndFiniteLPBSPolicy(t *testing.T) {
	body, hash := compositionPNG(t, 1440, 720)
	var ignored atomic.Int32
	server := compositionBackdropServer(t, body, hash, 1440, 720, &ignored)
	defer server.Close()
	policy := catalogSurfacePolicy(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	focal := presentation.FocalPoint{X: 0.5, Y: 0.5}
	if err := policy.Verify(context.Background(), "web.hero", 1440, 720, "contain", focal, "full_bleed"); err != nil {
		t.Fatalf("valid catalog surface rejected: %v", err)
	}
	for _, test := range []struct {
		name  string
		width int
		crop  string
		place string
	}{
		{name: "unknown crop", width: 1440, crop: "arbitrary", place: "full_bleed"},
		{name: "stale dimensions", width: 1441, crop: "center", place: "full_bleed"},
		{name: "unlisted placement", width: 1440, crop: "center", place: "split_panel"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := policy.Verify(context.Background(), "web.hero", test.width, 720, test.crop, focal, test.place); err == nil {
				t.Fatal("surface policy accepted invalid presentation geometry")
			}
		})
	}
	for _, invalidFocal := range []presentation.FocalPoint{{X: math.NaN(), Y: 0.5}, {X: math.Inf(1), Y: 0.5}} {
		if err := policy.Verify(context.Background(), "web.hero", 1440, 720, "center", invalidFocal, "full_bleed"); err == nil {
			t.Fatal("surface policy accepted non-finite focal evidence")
		}
	}
	if err := policy.Verify(context.Background(), "web.hero", math.MaxInt32+1, 720, "center", focal, "full_bleed"); err == nil {
		t.Fatal("surface policy accepted dimensions outside the int32 wire range")
	}
	if _, err := secureScenarioOrigin(server.URL + "/nested/backdrop"); err == nil {
		t.Fatal("secureScenarioOrigin accepted an arbitrary path")
	}
}

func TestCatalogSurfacePolicyUsesDeclaredDesktopAndMobileGeometry(t *testing.T) {
	surfaces := []*surfacesv1.Surface{
		{Id: "web.hero", Width: 1440, Height: 720, Placements: []string{"full_bleed", "split_panel", "framed_inset", "corner_bleed"}, Kind: "product", Authority: "fixture", ConfirmedOn: "2026-09-15"},
		{Id: "web.hero-mobile", Width: 390, Height: 844, Placements: []string{"full_bleed", "framed_inset"}, Kind: "product", Authority: "fixture", ConfirmedOn: "2026-09-15"},
	}
	server := compositionSurfaceCatalogServer(t, surfaces)
	defer server.Close()
	policy := catalogSurfacePolicy(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	focal := presentation.FocalPoint{X: 0.5, Y: 0.5}
	tests := []struct {
		name    string
		surface string
		width   int
		height  int
		crop    string
		place   string
		wantErr bool
	}{
		{name: "declared desktop geometry", surface: "web.hero", width: 1440, height: 720, crop: "contain", place: "full_bleed"},
		{name: "declared mobile geometry and placement", surface: "web.hero-mobile", width: 390, height: 844, crop: "cover", place: "full_bleed"},
		{name: "declared mobile inset placement", surface: "web.hero-mobile", width: 390, height: 844, crop: "center", place: "framed_inset"},
		{name: "desktop dimensions cannot claim mobile", surface: "web.hero-mobile", width: 1440, height: 720, crop: "cover", place: "full_bleed", wantErr: true},
		{name: "missing mobile surface", surface: "web.hero-mobile-missing", width: 390, height: 844, crop: "cover", place: "full_bleed", wantErr: true},
		{name: "invalid crop", surface: "web.hero-mobile", width: 390, height: 844, crop: "cover_top", place: "full_bleed", wantErr: true},
		{name: "invalid mobile placement", surface: "web.hero-mobile", width: 390, height: 844, crop: "cover", place: "split_panel", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := policy.Verify(context.Background(), test.surface, test.width, test.height, test.crop, focal, test.place)
			if (err != nil) != test.wantErr {
				t.Fatalf("surface policy error = %v, wantErr=%t", err, test.wantErr)
			}
		})
	}
}
