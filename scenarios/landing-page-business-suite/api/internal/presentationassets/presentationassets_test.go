package presentationassets

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	backdroprelease "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/release"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/shared"
	presentation "landing-page-business-suite-api/internal/presentation"
	"landing-page-business-suite-api/internal/presentationseed"
)

type fakeBackdropOwner struct {
	release *backdroprelease.ReleasedBackdrop
	calls   int
}

func (f *fakeBackdropOwner) GetReference(context.Context, *connect.Request[backdroprelease.GetReferenceRequest]) (*connect.Response[backdroprelease.ReleasedBackdrop], error) {
	f.calls++
	return connect.NewResponse(f.release), nil
}

func newCache(t *testing.T) (*Cache, *filerouting.RoutedRoots, storage.Paths) {
	t.Helper()
	paths := storage.Paths{
		ConfigDir:   t.TempDir(),
		DataDir:     t.TempDir(),
		CacheDir:    t.TempDir(),
		LogsDir:     t.TempDir(),
		StateDir:    t.TempDir(),
		TestRunsDir: t.TempDir(),
	}
	roots := filerouting.New(paths)
	return NewCache(roots, defaultPublicPath), roots, paths
}

func pngFixture(t *testing.T, width, height int) ([]byte, string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 13), G: uint8(y * 17), B: 90, A: 255})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatal(err)
	}
	data := buffer.Bytes()
	digest := sha256.Sum256(data)
	return data, hex.EncodeToString(digest[:])
}

func testAsset(hash string) presentation.Asset {
	return presentation.Asset{
		ID:          "hero-art",
		ReleaseRef:  "release-1",
		ContentHash: hash,
		Width:       8,
		Height:      8,
		MIME:        "image/png",
		Surface:     "web.hero",
		CropPolicy:  "center",
		FocalPoint:  presentation.FocalPoint{X: 0.5, Y: 0.5},
		Provenance:  presentation.AssetProvenance{Provider: "backdrop-studio", JobRef: "job-1", CandidateRef: "candidate-1"},
		OverlayRegions: []presentation.OverlayRegion{{
			Name: "copy",
			X:    0, Y: 0, Width: 1, Height: 1,
			Measurement: presentation.LegibilityMeasurement{
				ContrastRatio: 7, MinimumContrastRatio: 4.5, Threshold: 4.5,
				Verdict: presentation.LegibilityPass, MeasurementRef: "backdrop-release:release-1",
			},
		}},
	}
}

func testRelease(uri string) *backdroprelease.ReleasedBackdrop {
	return &backdroprelease.ReleasedBackdrop{
		Id: "release-1", CandidateId: "candidate-1", SurfaceId: "web.hero", Placement: "hero",
		JobId: "job-1", MimeType: "image/png",
		Width: 8, Height: 8, ContrastRatio: 7, ContrastThreshold: 4.5,
		ReservedRegions: []*shared.ReservedRegion{{X: 0, Y: 0, Width: 1, Height: 1, Kind: "copy"}},
		Uri:             uri,
	}
}

func verifierFixture(t *testing.T, body []byte, release *backdroprelease.ReleasedBackdrop) (*Verifier, *fakeBackdropOwner, *Cache, *filerouting.RoutedRoots, *httptest.Server) {
	t.Helper()
	cache, roots, _ := newCache(t)
	digest := sha256.Sum256(body)
	if release.ContentHash == "" {
		release.ContentHash = hex.EncodeToString(digest[:])
	}
	owner := &fakeBackdropOwner{release: release}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/backdrops/release-1/asset" {
			http.NotFound(w, r)
			return
		}
		hash := hex.EncodeToString(digest[:])
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", stringInt(len(body)))
		w.Header().Set("ETag", `"`+hash+`"`)
		w.Header().Set("X-Content-SHA256", hash)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	verifier, err := NewVerifier(Config{
		Owner:                owner,
		OwnerClientOrigin:    server.URL,
		ResolveOwnerEndpoint: func(context.Context) (string, error) { return server.URL, nil },
		Cache:                cache,
		SurfacePolicy: SurfacePolicyFunc(func(_ context.Context, surface string, width, height int, crop string, _ presentation.FocalPoint, placement string) error {
			if surface != "web.hero" || width != 8 || height != 8 || crop != "center" || placement != "hero" {
				return os.ErrInvalid
			}
			return nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	return verifier, owner, cache, roots, server
}

func verifierFixtureForSurface(t *testing.T, body []byte, release *backdroprelease.ReleasedBackdrop, policy SurfacePolicy) *Verifier {
	t.Helper()
	cache, _, _ := newCache(t)
	digest := sha256.Sum256(body)
	hash := hex.EncodeToString(digest[:])
	if release.ContentHash == "" {
		release.ContentHash = hash
	}
	owner := &fakeBackdropOwner{release: release}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != release.GetUri() {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", stringInt(len(body)))
		w.Header().Set("ETag", `"`+hash+`"`)
		w.Header().Set("X-Content-SHA256", hash)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	verifier, err := NewVerifier(Config{
		Owner: owner, OwnerClientOrigin: server.URL,
		ResolveOwnerEndpoint: func(context.Context) (string, error) { return server.URL, nil },
		Cache:                cache, SurfacePolicy: policy,
	})
	if err != nil {
		t.Fatal(err)
	}
	return verifier
}

func stringInt(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for value > 0 {
		i--
		digits[i] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[i:])
}

func TestResolveAssetServesVerifiedBytesAfterCacheRestart(t *testing.T) {
	body, hash := pngFixture(t, 8, 8)
	verifier, owner, cache, roots, _ := verifierFixture(t, body, testRelease("/api/v1/backdrops/release-1/asset"))
	proof, err := verifier.ResolveAsset(context.Background(), testAsset(hash))
	if err != nil {
		t.Fatal(err)
	}
	if owner.calls != 1 || proof.PublicURL != cache.PublicURL(hash) || proof.ContentHash != hash {
		t.Fatalf("unexpected proof or owner calls: %#v calls=%d", proof, owner.calls)
	}

	serve := func(ctx context.Context, c *Cache) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, proof.PublicURL, nil).WithContext(ctx)
		c.Handler().ServeHTTP(recorder, request)
		return recorder
	}
	if response := serve(context.Background(), cache); response.Code != http.StatusOK || !bytes.Equal(response.Body.Bytes(), body) || response.Header().Get("Cache-Control") == "" {
		t.Fatalf("live cache response = %d, headers=%v", response.Code, response.Header())
	}

	restarted := NewCache(roots, defaultPublicPath)
	if response := serve(context.Background(), restarted); response.Code != http.StatusOK || !bytes.Equal(response.Body.Bytes(), body) {
		t.Fatalf("restart cache response = %d", response.Code)
	}

	head := httptest.NewRecorder()
	restarted.Handler().ServeHTTP(head, httptest.NewRequest(http.MethodHead, proof.PublicURL, nil))
	if head.Code != http.StatusOK || head.Body.Len() != 0 || head.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("HEAD response = %d body=%d headers=%v", head.Code, head.Body.Len(), head.Header())
	}
	post := httptest.NewRecorder()
	restarted.Handler().ServeHTTP(post, httptest.NewRequest(http.MethodPost, proof.PublicURL, nil))
	if post.Code != http.StatusMethodNotAllowed || post.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("POST response = %d headers=%v", post.Code, post.Header())
	}
	query := httptest.NewRecorder()
	restarted.Handler().ServeHTTP(query, httptest.NewRequest(http.MethodGet, proof.PublicURL+"?cache=public", nil))
	if query.Code != http.StatusNotFound {
		t.Fatalf("query response = %d, want %d", query.Code, http.StatusNotFound)
	}

	assetPath := filepath.Join("presentation-assets", hash, "asset.png")
	dataRoot, err := roots.PickRequired(context.Background(), storage.ClassData)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataRoot, assetPath), []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if response := serve(context.Background(), restarted); response.Code != http.StatusInternalServerError {
		t.Fatalf("tampered cache response = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}

func TestCacheKeepsReferenceSpecificProofsForSharedBytes(t *testing.T) {
	body, hash := pngFixture(t, 8, 8)
	cache, _, paths := newCache(t)
	first := AssetProof{ReleaseID: "release-1", ContentHash: hash, MIME: "image/png", Width: 8, Height: 8, Surface: "web.hero"}
	second := first
	second.ReleaseID = "release-2"
	second.Surface = "mobile.hero"
	if _, err := cache.Put(context.Background(), first, body); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Put(context.Background(), second, body); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(paths.DataDir, cacheDir, hash, proofsDir))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("reference-specific proofs = %d, want 2", len(entries))
	}
}

func TestConcurrentCacheInstancesConvergeWithoutStagingNameCollision(t *testing.T) {
	body, hash := pngFixture(t, 8, 8)
	_, roots, _ := newCache(t)
	proof := AssetProof{ReleaseID: "release-1", ContentHash: hash, MIME: "image/png", Width: 8, Height: 8, Surface: "web.hero"}
	caches := []*Cache{NewCache(roots, defaultPublicPath), NewCache(roots, defaultPublicPath)}
	errorsCh := make(chan error, len(caches))
	var waitGroup sync.WaitGroup
	for _, cache := range caches {
		waitGroup.Add(1)
		go func(cache *Cache) {
			defer waitGroup.Done()
			_, err := cache.Put(context.Background(), proof, body)
			errorsCh <- err
		}(cache)
	}
	waitGroup.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestResolveAssetRejectsTamperedBytesAndEvidenceMismatches(t *testing.T) {
	body, hash := pngFixture(t, 8, 8)
	tests := []struct {
		name   string
		mutate func(*backdroprelease.ReleasedBackdrop, []byte)
	}{
		{name: "tampered bytes", mutate: func(_ *backdroprelease.ReleasedBackdrop, body []byte) { body[0] ^= 0xff }},
		{name: "stale dimensions", mutate: func(release *backdroprelease.ReleasedBackdrop, _ []byte) { release.Width = 9 }},
		{name: "surface mismatch", mutate: func(release *backdroprelease.ReleasedBackdrop, _ []byte) { release.SurfaceId = "mobile.hero" }},
		{name: "region mismatch", mutate: func(release *backdroprelease.ReleasedBackdrop, _ []byte) { release.ReservedRegions[0].Width = 0.5 }},
		{name: "missing job proof", mutate: func(release *backdroprelease.ReleasedBackdrop, _ []byte) { release.JobId = "" }},
		{name: "stale MIME proof", mutate: func(release *backdroprelease.ReleasedBackdrop, _ []byte) { release.MimeType = "image/jpeg" }},
		{name: "stale hash proof", mutate: func(release *backdroprelease.ReleasedBackdrop, _ []byte) {
			release.ContentHash = strings.Repeat("0", 64)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := append([]byte(nil), body...)
			release := testRelease("/api/v1/backdrops/release-1/asset")
			test.mutate(release, candidate)
			verifier, _, _, _, _ := verifierFixture(t, candidate, release)
			asset := testAsset(hash)
			if _, err := verifier.ResolveAsset(context.Background(), asset); err == nil {
				t.Fatal("ResolveAsset accepted forged or stale release evidence")
			}
		})
	}
}

func TestResolveAssetSupportsIndependentDesktopAndMobileProofs(t *testing.T) {
	tests := []struct {
		name      string
		width     int
		height    int
		surface   string
		crop      string
		placement string
	}{
		{name: "desktop hero", width: 8, height: 8, surface: "web.hero", crop: "center", placement: "hero"},
		{name: "mobile hero", width: 4, height: 6, surface: "web.hero-mobile", crop: "cover", placement: "full_bleed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body, hash := pngFixture(t, test.width, test.height)
			asset := testAsset(hash)
			asset.Width, asset.Height = test.width, test.height
			asset.Surface, asset.CropPolicy = test.surface, test.crop
			release := testRelease("/api/v1/backdrops/release-1/asset")
			release.Width, release.Height = int32(test.width), int32(test.height)
			release.SurfaceId, release.Placement = test.surface, test.placement
			verifier := verifierFixtureForSurface(t, body, release, SurfacePolicyFunc(func(_ context.Context, surface string, width, height int, crop string, focal presentation.FocalPoint, placement string) error {
				if surface != test.surface || width != test.width || height != test.height || crop != test.crop || placement != test.placement || focal != (presentation.FocalPoint{X: 0.5, Y: 0.5}) {
					return os.ErrInvalid
				}
				return nil
			}))

			proof, err := verifier.ResolveAsset(context.Background(), asset)
			if err != nil {
				t.Fatal(err)
			}
			if proof.Surface != test.surface || proof.Width != test.width || proof.Height != test.height || proof.CropPolicy != test.crop || proof.Placement != test.placement {
				t.Fatalf("resolved surface proof = %#v", proof)
			}
			if proof.Provenance != (presentation.AssetProvenance{Provider: "backdrop-studio", JobRef: "job-1", CandidateRef: "candidate-1"}) {
				t.Fatalf("resolved provenance = %#v", proof.Provenance)
			}
		})
	}
}

func TestResolveAssetRejectsCrossSurfaceAndUnqualifiedMobilePresentation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*presentation.Asset, *backdroprelease.ReleasedBackdrop)
		policy SurfacePolicy
	}{
		{
			name: "cross surface owner response",
			mutate: func(asset *presentation.Asset, release *backdroprelease.ReleasedBackdrop) {
				asset.Surface = "mobile.hero"
				release.SurfaceId = "web.hero"
			},
			policy: SurfacePolicyFunc(func(context.Context, string, int, int, string, presentation.FocalPoint, string) error { return nil }),
		},
		{
			name:   "unqualified mobile crop",
			mutate: func(asset *presentation.Asset, _ *backdroprelease.ReleasedBackdrop) { asset.CropPolicy = "arbitrary" },
			policy: SurfacePolicyFunc(func(_ context.Context, surface string, _ int, _ int, crop string, _ presentation.FocalPoint, placement string) error {
				if surface != "web.hero-mobile" || crop != "cover" || placement != "full_bleed" {
					return os.ErrInvalid
				}
				return nil
			}),
		},
		{
			name: "unqualified mobile placement",
			mutate: func(_ *presentation.Asset, release *backdroprelease.ReleasedBackdrop) {
				release.Placement = "split_panel"
			},
			policy: SurfacePolicyFunc(func(_ context.Context, surface string, _ int, _ int, crop string, _ presentation.FocalPoint, placement string) error {
				if surface != "web.hero-mobile" || crop != "cover" || placement != "full_bleed" {
					return os.ErrInvalid
				}
				return nil
			}),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body, hash := pngFixture(t, 4, 6)
			asset := testAsset(hash)
			asset.Width, asset.Height = 4, 6
			asset.Surface, asset.CropPolicy = "web.hero-mobile", "cover"
			release := testRelease("/api/v1/backdrops/release-1/asset")
			release.Width, release.Height = 4, 6
			release.SurfaceId, release.Placement = "web.hero-mobile", "full_bleed"
			test.mutate(&asset, release)
			verifier := verifierFixtureForSurface(t, body, release, test.policy)
			if _, err := verifier.ResolveAsset(context.Background(), asset); err == nil {
				t.Fatal("ResolveAsset accepted an invalid mobile surface, crop, or placement")
			}
		})
	}
}

func TestResolveAssetRejectsSSRFAndNonImmutableOwnerResponses(t *testing.T) {
	body, hash := pngFixture(t, 8, 8)
	for _, uri := range []string{"http://127.0.0.1/private", "//127.0.0.1/private", "/api/v1/backdrops/../private"} {
		t.Run(uri, func(t *testing.T) {
			verifier, _, _, _, _ := verifierFixture(t, body, testRelease(uri))
			if _, err := verifier.ResolveAsset(context.Background(), testAsset(hash)); err == nil {
				t.Fatal("ResolveAsset accepted an owner URI that can escape the owner asset route")
			}
		})
	}

	cache, _, _ := newCache(t)
	owner := &fakeBackdropOwner{release: testRelease("/api/v1/backdrops/release-1/asset")}
	digest := sha256.Sum256(body)
	owner.release.ContentHash = hex.EncodeToString(digest[:])
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", stringInt(len(body)))
		digest := sha256.Sum256(body)
		hash := hex.EncodeToString(digest[:])
		w.Header().Set("ETag", `"`+hash+`"`)
		w.Header().Set("X-Content-SHA256", hash)
		w.Header().Set("Cache-Control", "public, max-age=60")
		_, _ = w.Write(body)
	}))
	defer server.Close()
	verifier, err := NewVerifier(Config{Owner: owner, OwnerClientOrigin: server.URL, ResolveOwnerEndpoint: func(context.Context) (string, error) { return server.URL, nil }, Cache: cache, SurfacePolicy: SurfacePolicyFunc(func(context.Context, string, int, int, string, presentation.FocalPoint, string) error { return nil })})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifier.ResolveAsset(context.Background(), testAsset(hash)); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("non-immutable owner response error = %v", err)
	}
}

func TestPublicationRequiresGeneratedURLAndVerifiedDescriptorAgreement(t *testing.T) {
	body, hash := pngFixture(t, 8, 8)
	verifier, _, _, _, _ := verifierFixture(t, body, testRelease("/api/v1/backdrops/release-1/asset"))
	asset := testAsset(hash)
	asset.PrivateEvidenceRefs = []string{"controlled-proof"}
	proof, err := verifier.ResolveAsset(context.Background(), asset)
	if err != nil {
		t.Fatal(err)
	}
	asset.PublicURL = "/wrong.png"
	if err := comparePublicationAsset(asset, proof); err == nil || !strings.Contains(err.Error(), "public URL") {
		t.Fatalf("wrong public URL error = %v", err)
	}
	asset.PublicURL = proof.PublicURL
	asset.OverlayRegions[0].Measurement.ContrastRatio = 6
	if err := comparePublicationAsset(asset, proof); err == nil || !strings.Contains(err.Error(), "measured region") {
		t.Fatalf("stale measurement error = %v", err)
	}
	asset = testAsset(hash)
	asset.PrivateEvidenceRefs = []string{"controlled-proof"}
	asset.PublicURL = proof.PublicURL
	asset.Provenance.CandidateRef = "forged-candidate"
	if err := comparePublicationAsset(asset, proof); err == nil || !strings.Contains(err.Error(), "provenance") {
		t.Fatalf("stale provenance error = %v", err)
	}
	asset = testAsset(hash)
	asset.PublicURL = proof.PublicURL
	if err := comparePublicationAsset(asset, proof); err != nil {
		t.Fatalf("strict real proof was rejected: %v", err)
	}
	asset.Provenance.JobRef = "forged-job"
	if err := comparePublicationAsset(asset, proof); err == nil || !strings.Contains(err.Error(), "provenance") {
		t.Fatalf("forged job provenance was accepted: %v", err)
	}
}

func TestVerifierRequiresOwnerClientAndDiscoveredOriginsToMatch(t *testing.T) {
	_, hash := pngFixture(t, 8, 8)
	cache, _, _ := newCache(t)
	owner := &fakeBackdropOwner{release: testRelease("/api/v1/backdrops/release-1/asset")}
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	verifier, err := NewVerifier(Config{
		Owner:             owner,
		OwnerClientOrigin: server.URL,
		ResolveOwnerEndpoint: func(context.Context) (string, error) {
			return "http://127.0.0.1:1", nil
		},
		Cache:         cache,
		SurfacePolicy: SurfacePolicyFunc(func(context.Context, string, int, int, string, presentation.FocalPoint, string) error { return nil }),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifier.ResolveAsset(context.Background(), testAsset(hash)); err == nil || !strings.Contains(err.Error(), "origin") {
		t.Fatalf("origin mismatch error = %v", err)
	}
}

func TestVerifierUsesBoundedOwnerAssetRequestTimeout(t *testing.T) {
	_, hash := pngFixture(t, 8, 8)
	cache, _, _ := newCache(t)
	owner := &fakeBackdropOwner{release: testRelease("/api/v1/backdrops/release-1/asset")}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	verifier, err := NewVerifier(Config{
		Owner: owner, OwnerClientOrigin: server.URL,
		ResolveOwnerEndpoint: func(context.Context) (string, error) { return server.URL, nil },
		Cache:                cache, RequestTimeout: 20 * time.Millisecond,
		SurfacePolicy: SurfacePolicyFunc(func(context.Context, string, int, int, string, presentation.FocalPoint, string) error { return nil }),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifier.ResolveAsset(context.Background(), testAsset(hash)); err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("timeout error = %v", err)
	}
}

func TestCacheHonorsLeasedTestStorageIsolation(t *testing.T) {
	body, hash := pngFixture(t, 8, 8)
	verifier, _, cache, roots, _ := verifierFixture(t, body, testRelease("/api/v1/backdrops/release-1/asset"))
	asset := testAsset(hash)
	testContext := database.WithTestMode(context.Background())
	if _, err := verifier.ResolveAsset(testContext, asset); err == nil {
		t.Fatal("test-mode write succeeded without an installed lease")
	}
	if _, err := roots.InstallLeasedTestRoots("presentation-assets-test", 0, true); err != nil {
		t.Fatal(err)
	}
	defer roots.ClearTestRoots("presentation-assets-test")
	proof, err := verifier.ResolveAsset(testContext, asset)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		ctx  context.Context
		code int
	}{
		{name: "live", ctx: context.Background(), code: http.StatusNotFound},
		{name: "leased", ctx: testContext, code: http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, proof.PublicURL, nil).WithContext(test.ctx)
			cache.Handler().ServeHTTP(recorder, request)
			if recorder.Code != test.code {
				t.Fatalf("response = %d, want %d", recorder.Code, test.code)
			}
			if test.name == "leased" && recorder.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatalf("leased cache policy = %q", recorder.Header().Get("Cache-Control"))
			}
		})
	}
}

func TestPublicAssetClosureDoesNotIncludePrivateProfileCandidates(t *testing.T) {
	document, err := presentationseed.Recommended()
	if err != nil {
		t.Fatal(err)
	}
	document.Apps[0].Publication = presentation.PublicationPublished
	privateAsset := document.Assets[0]
	privateAsset.ID = "private-only"
	document.Assets = append(document.Assets, privateAsset)
	for i := range document.Pages {
		if document.Pages[i].ID != document.Apps[1].PageID {
			continue
		}
		for blockIndex := range document.Pages[i].Blocks {
			if hero, ok := document.Pages[i].Blocks[blockIndex].Content.(presentation.ProductHeroContent); ok {
				hero.VisualRef = "private-only"
				document.Pages[i].Blocks[blockIndex].Content = hero
				document.Pages[i].Display.AssetLabels["private-only"] = presentation.AssetLabel{Alt: "Private candidate fixture"}
			}
		}
	}
	assets, err := publicAssetIDs(document)
	if err != nil {
		t.Fatal(err)
	}
	if assets["private-only"] {
		t.Fatalf("public asset closure = %#v", assets)
	}
}

func publicationDocument(asset presentation.Asset) presentation.Document {
	shell := presentation.ShellDisplay{BrandName: "Suite", BrandMark: "suite", BrandTarget: "/", SkipLabel: "Skip", MenuLabel: "Menu", FooterBrandName: "Suite", FooterBrandMark: "suite", FooterBrandTarget: "/", UnavailableReason: "Unavailable", PreviewLabel: "Preview"}
	page := func(id, title string, blocks []presentation.Block) presentation.Page {
		return presentation.Page{
			Display: presentation.PageDisplay{Shell: shell, AssetLabels: map[string]presentation.AssetLabel{asset.ID: {Alt: "Configured asset fixture"}}, FixtureDisplay: map[string]presentation.FixtureDisplay{}, Blocks: map[string]presentation.BlockDisplay{}, Apps: map[string]presentation.AppDisplay{}},
			ID:      id, Locale: "en", Title: title, Description: "Configured description",
			Theme:      presentation.Theme{Variant: "signal", Primary: "#263E38", Background: "#EAEADF", Accent: "#DF7958"},
			Navigation: presentation.Navigation{Label: "Main", Items: []presentation.NavigationItem{{Label: "Home", AccessibleLabel: "Home", Target: "#hero"}}},
			Blocks:     blocks,
			Footer:     presentation.Footer{Label: "Footer", Links: []presentation.NavigationItem{{Label: "Home", AccessibleLabel: "Home", Target: "/"}}},
		}
	}
	blocks := []presentation.Block{
		{ID: "hero", Kind: presentation.BlockProductHero, Version: presentation.SchemaVersion, Variant: "centered", Content: presentation.ProductHeroContent{AppKey: "public", Title: "Public", Description: "Public description", VisualRef: asset.ID, AccessibilityLabel: "Public visual", Actions: []presentation.Action{{Kind: presentation.ActionAnchor, Label: "Learn", AccessibleLabel: "Learn", Target: "#hero"}}}},
		{ID: "closing", Kind: presentation.BlockClosingAction, Version: presentation.SchemaVersion, Variant: "plain", Content: presentation.ClosingActionContent{Heading: "Continue", Description: "Continue description", Actions: []presentation.Action{{Kind: presentation.ActionAnchor, Label: "Continue", AccessibleLabel: "Continue", Target: "#hero"}}}},
	}
	return presentation.Document{
		SchemaVersion: presentation.SchemaVersion,
		Bundle:        presentation.Bundle{Key: "suite", Name: "Suite", AppOrder: []string{"public"}, MaxAppSlides: 2, PageID: "bundle", EmptyPageID: "empty", DefaultLocale: "en", Locales: []string{"en"}},
		Apps:          []presentation.App{{Key: "public", Slug: "public", Name: "Public", Enabled: true, Visibility: presentation.VisibilityPublic, Publication: presentation.PublicationPublished, PageID: "public", Tagline: "Tagline", Description: "Description"}},
		Pages:         []presentation.Page{page("bundle", "Suite", nil), page("empty", "Empty", nil), page("public", "Public", blocks)},
		Assets:        []presentation.Asset{asset},
	}
}

func TestVerifyPublicationEnforcesDocumentURLMeasurementAndProvenance(t *testing.T) {
	body, hash := pngFixture(t, 8, 8)
	verifier, _, _, _, _ := verifierFixture(t, body, testRelease("/api/v1/backdrops/release-1/asset"))
	asset := testAsset(hash)
	asset.PrivateEvidenceRefs = []string{"controlled-proof"}
	asset.PublicURL = "/not-generated.png"
	if err := verifier.VerifyPublication(context.Background(), publicationDocument(asset)); err == nil || !strings.Contains(err.Error(), "public URL") {
		t.Fatalf("publication accepted a stale URL: %v", err)
	}
	proof, err := verifier.ResolveAsset(context.Background(), testAsset(hash))
	if err != nil {
		t.Fatal(err)
	}
	asset.PublicURL = proof.PublicURL
	asset.OverlayRegions[0].Measurement.Threshold = 3
	if err := verifier.VerifyPublication(context.Background(), publicationDocument(asset)); err == nil || !strings.Contains(err.Error(), "measured region") {
		t.Fatalf("publication accepted stale measurement: %v", err)
	}
	asset = testAsset(hash)
	asset.PrivateEvidenceRefs = []string{"controlled-proof"}
	asset.PublicURL = proof.PublicURL
	if err := verifier.VerifyPublication(context.Background(), publicationDocument(asset)); err != nil {
		t.Fatalf("publication rejected strict real proof: %v", err)
	}
	asset.Provenance.JobRef = "forged-job"
	if err := verifier.VerifyPublication(context.Background(), publicationDocument(asset)); err == nil || !strings.Contains(err.Error(), "provenance") {
		t.Fatalf("publication accepted forged job provenance: %v", err)
	}
}
