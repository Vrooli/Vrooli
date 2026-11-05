package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/storage"
	platform "github.com/vrooli/platform-go"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"google.golang.org/protobuf/proto"
	landinghttp "landing-page-business-suite-api/handlers/config"
	varianthttp "landing-page-business-suite-api/handlers/experimentation"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/landing"
	"landing-page-business-suite-api/internal/presentation"
	"landing-page-business-suite-api/internal/presentationassets"
	"landing-page-business-suite-api/internal/presentationseed"
)

var runLPBSBrowserEvidence = false

func init() {
	// Keep this expensive, browser/media-producing test out of normal unit runs.
	// The explicit flag is intentionally separate from the capture script's
	// testMode option: the latter isolates HTTP/file routing, while this flag is
	// operator consent to build and launch the real UI.
	flag.BoolVar(&runLPBSBrowserEvidence, "lpbs-browser-evidence", false, "run LPBS isolated browser evidence integration")
}

func TestPresentationBrowserEvidenceIntegration(t *testing.T) {
	if !runLPBSBrowserEvidence && os.Getenv("LPBS_RUN_BROWSER_EVIDENCE") != "1" {
		t.Skip("opt-in: set LPBS_RUN_BROWSER_EVIDENCE=1 or use the browser-evidence test flag")
	}
	outputDir := strings.TrimSpace(os.Getenv("LPBS_INTEGRATED_EVIDENCE_OUTPUT"))
	if outputDir == "" {
		t.Fatal("LPBS_INTEGRATED_EVIDENCE_OUTPUT must name a unique effort output directory")
	}

	// The three sequential journeys share a bounded outer process budget. Leave
	// room for the production build and platform-owned cleanup inside this lease.
	harnessContext, cancelHarness := context.WithTimeout(t.Context(), 10*time.Minute)
	defer cancelHarness()
	ctx := database.WithTestMode(harnessContext)
	store, roots, cache, revisions := browserEvidenceStore(t, ctx)
	apiServer := browserEvidenceAPIServer(t, store, cache, roots)
	defer apiServer.Close()

	client := lpbsconnect.NewLandingConfigServiceClient(apihttp.NewTestModeClient(apiServer.Client()), apiServer.URL)
	singleRoot := browserEvidenceRequest(t, ctx, client, "fixture-single", "/")
	singleDetail := browserEvidenceRequest(t, ctx, client, "fixture-single", "/apps/aquila")
	assertReadyPresentation(t, singleRoot, "fixture-single", revisions["fixture-single"], presentation.ModeSingleApp, "/", presentation.ScopeApp, "web-console", "aquila-signal")
	assertReadyPresentation(t, singleDetail, "fixture-single", revisions["fixture-single"], presentation.ModeAppDetail, "/apps/aquila", presentation.ScopeApp, "web-console", "aquila-signal")
	assertParity(t, singleRoot, singleDetail)

	bundleRoot := browserEvidenceRequest(t, ctx, client, "fixture-bundle", "/")
	bundleDetail := browserEvidenceRequest(t, ctx, client, "fixture-bundle", "/apps/aquila")
	assertReadyPresentation(t, bundleRoot, "fixture-bundle", revisions["fixture-bundle"], presentation.ModeBundle, "/", presentation.ScopeBundle, "", "studio-bundle")
	assertReadyPresentation(t, bundleDetail, "fixture-bundle", revisions["fixture-bundle"], presentation.ModeAppDetail, "/apps/aquila", presentation.ScopeApp, "web-console", "aquila-signal")
	assertContentParity(t, singleDetail, bundleDetail)
	backdropDetail := browserEvidenceRequest(t, ctx, client, "fixture-bundle", "/apps/backdrop-studio")
	assertReadyPresentation(t, backdropDetail, "fixture-bundle", revisions["fixture-bundle"], presentation.ModeAppDetail, "/apps/backdrop-studio", presentation.ScopeApp, "backdrop-studio", "backdrop-studio")
	if got := bundleRoot.GetDiagnostics().GetEligibleAppKeys(); len(got) != 2 || got[0] != "web-console" || got[1] != "backdrop-studio" {
		t.Fatalf("isolated bundle membership = %v, want Aquila then fixture-only Backdrop", got)
	}

	if _, err := client.GetLandingConfig(ctx, connect.NewRequest(&lpbsv1.GetLandingConfigRequest{VariantSlug: "fixture-single", Route: "/apps/browser-automation-studio", Locale: "en"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("private BAS detail must return NotFound, got %v", err)
	}
	if _, err := client.GetLandingConfig(ctx, connect.NewRequest(&lpbsv1.GetLandingConfigRequest{VariantSlug: "fixture-bundle", Route: "/apps/browser-automation-studio", Locale: "en"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("private BAS detail in bundle must return NotFound, got %v", err)
	}

	assetHash := revisions["asset"]
	assetRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, apiServer.URL+cache.PublicURL(assetHash), nil)
	if err != nil {
		t.Fatal(err)
	}
	assetRequest.Header.Set("X-Vrooli-Test-Mode", "1")
	assetResponse, err := apiServer.Client().Do(assetRequest)
	if err != nil {
		t.Fatalf("asset cache request: %v", err)
	}
	defer assetResponse.Body.Close()
	if assetResponse.StatusCode != http.StatusOK {
		t.Fatalf("asset cache status = %d, want 200", assetResponse.StatusCode)
	}
	assetBody, err := io.ReadAll(io.LimitReader(assetResponse.Body, 8<<20))
	if err != nil || browserEvidenceAssetHash(t, assetBody) != assetHash {
		t.Fatalf("served asset bytes failed integrity: %v", err)
	}
	decoded, format, err := image.Decode(bytes.NewReader(assetBody))
	if err != nil || format != "png" || decoded.Bounds().Dx() != 1440 || decoded.Bounds().Dy() != 720 {
		t.Fatalf("served asset is not the expected decodable 1440x720 PNG: %v", err)
	}
	beforeCapture := roots.LeaseStats()
	if beforeCapture.PrimaryWritesDuringTestMode != 0 || beforeCapture.TestRootWrites == 0 {
		t.Fatalf("fixture writes did not stay in the leased roots: %+v", beforeCapture)
	}

	distDir := filepath.Join(t.TempDir(), "vite-dist")
	buildProductionVite(t, distDir)
	apiPort := mustURLPort(t, apiServer.URL)
	// Each journey has its own 180-second ceiling and all three share this
	// seven-minute outer ceiling, including SSR checks and receipt persistence.
	captureContext, cancelCapture := context.WithTimeout(harnessContext, 7*time.Minute)
	defer cancelCapture()
	cmd := exec.CommandContext(captureContext, "node", filepath.Join("..", "scripts", "capture-integrated-fixture.mjs"),
		"--api-port", strconv.Itoa(apiPort),
		"--dist-dir", distDir,
		"--output-dir", outputDir,
		"--single-variant", "fixture-single",
		"--single-revision", revisions["fixture-single"],
		"--bundle-variant", "fixture-bundle",
		"--bundle-revision", revisions["fixture-bundle"],
	)
	cmd.Dir = filepath.Join(".")
	cmd.Env = append(os.Environ(), "NODE_ENV=test")
	result, err := browserEvidenceCommand(cmd)
	logPath := outputDir + ".runner.log"
	logFile, logErr := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if logErr != nil {
		t.Fatalf("retain runner output: %v", logErr)
	}
	_, writeErr := logFile.Write(result)
	closeErr := logFile.Close()
	if writeErr != nil || closeErr != nil {
		t.Fatalf("retain runner output: %v %v", writeErr, closeErr)
	}
	t.Logf("integrated browser evidence runner:\n%s", result)
	if err != nil {
		t.Fatalf("integrated browser evidence capture failed: %v", err)
	}
	if after := roots.LeaseStats(); after != beforeCapture {
		t.Fatalf("read-only capture changed routed write counters: before=%+v after=%+v", beforeCapture, after)
	}
}

// Bound the test-owned command tree through the shared platform seam. A timed
// out Node process must not leave a browser running after the test returns.
func browserEvidenceCommand(command *exec.Cmd) ([]byte, error) {
	if err := platform.ConfigureCommand(command, platform.ProcessOptions{Detached: true}); err != nil {
		return nil, err
	}
	command.Cancel = func() error { return platform.GracefulStopProcess(command.Process) }
	command.WaitDelay = 2 * time.Second
	var output bytes.Buffer
	command.Stdout, command.Stderr = &output, &output
	if err := command.Start(); err != nil {
		return output.Bytes(), err
	}
	cleanup, err := platform.AssignProcessContainment(command.Process)
	if err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		return output.Bytes(), err
	}
	defer cleanup()
	err = command.Wait()
	return output.Bytes(), err
}

const browserEvidenceLeaseID = "lpbs-browser-evidence-fixture"

func browserEvidenceStore(t *testing.T, ctx context.Context) (*experimentation.ConfigStore, *filerouting.RoutedRoots, *presentationassets.Cache, map[string]string) {
	t.Helper()
	primary := storage.Paths{
		ConfigDir: t.TempDir(), DataDir: t.TempDir(), CacheDir: t.TempDir(),
		StateDir: t.TempDir(), LogsDir: t.TempDir(), TestRunsDir: t.TempDir(),
	}
	roots := filerouting.New(primary)
	if _, err := roots.InstallLeasedTestRoots(browserEvidenceLeaseID, 10*time.Minute, true); err != nil {
		t.Fatalf("install isolated routed roots: %v", err)
	}
	t.Cleanup(func() { _ = roots.ClearTestRoots(browserEvidenceLeaseID) })

	variantsDir := t.TempDir()
	store := experimentation.NewConfigStore(variantsDir, "", nil)
	cache := presentationassets.NewCache(roots, "/api/v1/presentation-assets/")
	reviewAssets := map[string]string{
		"survey-relief":  filepath.Join("ui", "public", "presentation", "survey-relief.png"),
		"pale-moon":      filepath.Join("ui", "public", "presentation", "pale-moon.png"),
		"tidal-halftone": filepath.Join("ui", "public", "presentation", "tidal-halftone.png"),
	}
	assetBytes := make(map[string][]byte, len(reviewAssets))
	revisions := map[string]string{}
	for assetID, relativePath := range reviewAssets {
		data, err := os.ReadFile(filepath.Join("..", relativePath))
		if err != nil {
			t.Fatalf("read review art %s: %v", assetID, err)
		}
		assetBytes[assetID] = data
	}

	fixtureVerifier := func(verifyContext context.Context, document presentation.Document, _ *presentation.Document) error {
		// Explicitly fixture-only publication owner. This is not an owner receipt,
		// release qualification, Backdrop release, or public-cutover authority.
		if err := presentation.Validate(document); err != nil {
			return err
		}
		for _, asset := range document.Assets {
			if _, _, err := cache.Read(verifyContext, asset.ContentHash); err != nil {
				return fmt.Errorf("fixture owner cache proof %s: %w", asset.ID, err)
			}
		}
		return nil
	}
	store.SetPresentationStorage(roots, fixtureVerifier)

	single := browserEvidenceDocument(t, false, cache, ctx, assetBytes)
	bundle := browserEvidenceDocument(t, true, cache, ctx, assetBytes)
	for _, variant := range []struct {
		slug string
		doc  presentation.Document
	}{
		{"fixture-single", single},
		{"fixture-bundle", bundle},
	} {
		if err := store.SaveVariant(variant.slug, &experimentation.VariantSnapshot{Variant: experimentation.VariantSnapshotMeta{
			Slug: variant.slug, Name: "LPBS isolated browser evidence fixture", Weight: 0, Status: "active",
			Axes: map[string]string{"persona": "soloDev", "jtbd": "testing", "conversionStyle": "technical"},
		}}); err != nil {
			t.Fatalf("save test variant %s: %v", variant.slug, err)
		}
		draft, err := store.SavePresentationDraft(ctx, variant.slug, variant.doc, 0)
		if err != nil {
			t.Fatalf("save %s draft: %v", variant.slug, err)
		}
		published, err := store.PublishPresentation(ctx, variant.slug, draft.DraftRevision, draft.Generation)
		if err != nil {
			t.Fatalf("publish %s fixture revision: %v", variant.slug, err)
		}
		if published.ActiveRevision != draft.DraftRevision || published.DraftRevision != draft.DraftRevision {
			t.Fatalf("%s heads = active %q draft %q, want exact %q", variant.slug, published.ActiveRevision, published.DraftRevision, draft.DraftRevision)
		}
		revisions[variant.slug] = draft.DraftRevision
	}
	revisions["asset"] = browserEvidenceAssetHash(t, assetBytes["survey-relief"])
	return store, roots, cache, revisions
}

func browserEvidenceDocument(t *testing.T, bundle bool, cache *presentationassets.Cache, ctx context.Context, assetBytes map[string][]byte) presentation.Document {
	t.Helper()
	document, err := presentationseed.Recommended()
	if err != nil {
		t.Fatalf("decode canonical seed: %v", err)
	}
	for index := range document.Pages {
		document.Pages[index].Footer.Label = "Configured footer"
		document.Pages[index].Display.Shell.FooterNote = "Isolated integration fixture — not a public release"
	}
	document.Apps[0].Publication = presentation.PublicationPublished
	if bundle {
		for index := range document.Apps {
			if document.Apps[index].Key == "backdrop-studio" {
				document.Apps[index].Enabled = true
				document.Apps[index].Visibility = presentation.VisibilityPublic
				document.Apps[index].Publication = presentation.PublicationPublished
			}
		}
		for pageIndex := range document.Pages {
			if document.Pages[pageIndex].ID != document.Bundle.PageID {
				continue
			}
			page := &document.Pages[pageIndex]
			page.Display.Apps["backdrop-studio"] = presentation.AppDisplay{FixtureRef: "backdrop-studies", Mark: "landscape", Tone: "amber", DetailLabel: "Explore Backdrop Studio"}
			page.Display.FixtureDisplay["backdrop-studies"] = presentation.FixtureDisplay{Mark: "landscape"}
			page.Display.Blocks["hero"] = presentation.BlockDisplay{Eyebrow: page.Display.Blocks["hero"].Eyebrow, HeroFixtureRefs: map[string]string{"web-console": "workspace", "backdrop-studio": "backdrop-studies"}}
			for blockIndex := range page.Blocks {
				switch page.Blocks[blockIndex].Kind {
				case presentation.BlockBundleHero:
					hero := page.Blocks[blockIndex].Content.(presentation.BundleHeroContent)
					hero.HeroItems = append(hero.HeroItems, presentation.HeroItem{AppKey: "backdrop-studio", ExhibitKind: "artwork", DetailLabel: "Your visual studio"})
					page.Blocks[blockIndex].Content = hero
				case presentation.BlockAppSpotlights:
					spotlights := page.Blocks[blockIndex].Content.(presentation.AppSpotlightsContent)
					spotlights.AppKeys = append(spotlights.AppKeys, "backdrop-studio")
					page.Blocks[blockIndex].Content = spotlights
				}
			}
		}
	}
	for index := range document.Pages {
		if document.Pages[index].ID != "aquila-signal" {
			continue
		}
		for blockIndex := range document.Pages[index].Blocks {
			if document.Pages[index].Blocks[blockIndex].Kind != presentation.BlockProductStory {
				continue
			}
			story, ok := document.Pages[index].Blocks[blockIndex].Content.(presentation.ProductStoryContent)
			if !ok || len(story.Items) == 0 {
				t.Fatalf("canonical Aquila story has no typed story item")
			}
			story.Items[0].VisualRef = "survey-relief"
			story.Items[0].AltText = "Review fixture art"
			document.Pages[index].Blocks[blockIndex].Content = story
		}
	}
	for index := range document.Assets {
		data, ok := assetBytes[document.Assets[index].ID]
		if !ok {
			continue
		}
		config, format, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || format != "png" {
			t.Fatalf("review art %s is not PNG: %v", document.Assets[index].ID, err)
		}
		sum := sha256.Sum256(data)
		hash := hex.EncodeToString(sum[:])
		document.Assets[index].ContentHash = hash
		document.Assets[index].Width = config.Width
		document.Assets[index].Height = config.Height
		document.Assets[index].PublicURL = cache.PublicURL(hash)
		proof := presentationassets.AssetProof{
			AssetID: document.Assets[index].ID, ReleaseID: document.Assets[index].ReleaseRef,
			ContentHash: hash, MIME: "image/png", Width: config.Width, Height: config.Height,
			Surface: document.Assets[index].Surface, Placement: "illustration",
			CropPolicy: document.Assets[index].CropPolicy, FocalPoint: document.Assets[index].FocalPoint,
			Provenance: document.Assets[index].Provenance, OverlayRegions: document.Assets[index].OverlayRegions,
		}
		if _, err := cache.Put(ctx, proof, data); err != nil {
			t.Fatalf("cache fixture art %s: %v", document.Assets[index].ID, err)
		}
	}
	if err := document.Validate(); err != nil {
		t.Fatalf("isolated %s fixture document invalid: %v", map[bool]string{true: "bundle", false: "single"}[bundle], err)
	}
	return document
}

func browserEvidenceAssetHash(t *testing.T, data []byte) string {
	t.Helper()
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func browserEvidenceAPIServer(t *testing.T, store *experimentation.ConfigStore, cache *presentationassets.Cache, roots *filerouting.RoutedRoots) *httptest.Server {
	t.Helper()
	service := landing.NewLandingConfigServiceWithConfigStore(store, nil, nil)
	router := mux.NewRouter()
	landinghttp.RegisterLandingConfigConnectRoutes(router, service)
	varianthttp.RegisterBrandingConnectRoutes(router, store, func(http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "fixture is public-read-only", http.StatusUnauthorized)
		}
	})
	router.Handle("/api/v1/presentation-assets/{file}", cache.Handler()).Methods(http.MethodGet, http.MethodHead)
	// Test-only lease transition. This route is reachable only on this
	// httptest owner and is called after the browser journeys, so it cannot
	// alter any production or live publication state. The files remain present;
	// advancing the routed clock proves expiry rather than merely deletion.
	router.HandleFunc("/__lpbs_test__/expire-presentation-lease", func(writer http.ResponseWriter, request *http.Request) {
		if !database.IsTestMode(request.Context()) {
			http.NotFound(writer, request)
			return
		}
		roots.SetClock(schedule.NewFake(time.Now().Add(24 * time.Hour)))
		writer.WriteHeader(http.StatusNoContent)
	}).Methods(http.MethodPost)
	// Production composes apihttp.TestModeMiddleware around its router. This
	// httptest owner uses the same exact-header context adapter locally because
	// that production middleware is intentionally a no-op in production mode;
	// this harness therefore does not claim to certify production middleware
	// composition.
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("X-Vrooli-Test-Mode") == "1" {
			request = request.WithContext(database.WithTestMode(request.Context()))
		}
		router.ServeHTTP(writer, request)
	})
	return httptest.NewServer(handler)
}

func browserEvidenceRequest(t *testing.T, ctx context.Context, client lpbsconnect.LandingConfigServiceClient, variant, route string) *sharedv1.ResolvedProductPresentation {
	t.Helper()
	response, err := client.GetLandingConfig(ctx, connect.NewRequest(&lpbsv1.GetLandingConfigRequest{VariantSlug: variant, Route: route, Locale: "en"}))
	if err != nil {
		t.Fatalf("public generated request %s %s: %v", variant, route, err)
	}
	if response == nil || response.Msg == nil || response.Msg.GetPresentation() == nil {
		t.Fatalf("public generated request %s %s returned no presentation", variant, route)
	}
	return response.Msg.GetPresentation()
}

func assertReadyPresentation(t *testing.T, value *sharedv1.ResolvedProductPresentation, variant, revision string, mode presentation.Mode, route string, scope presentation.Scope, appKey, pageID string) {
	t.Helper()
	diagnostics := value.GetDiagnostics()
	// Public requests select the active publication; they do not request an
	// explicit revision. Preserve that distinction in the captured diagnostics.
	if diagnostics.GetRequestedRoute() != route || diagnostics.GetResolvedRoute() != route || diagnostics.GetRequestedVariant() != variant || diagnostics.GetResolvedVariant() != variant || diagnostics.GetRequestedRevision() != "" || diagnostics.GetResolvedRevision() != revision {
		t.Fatalf("diagnostics identity = route (%q,%q), variant (%q,%q), revision (%q,%q); want route %q, variant %q, revision %q", diagnostics.GetRequestedRoute(), diagnostics.GetResolvedRoute(), diagnostics.GetRequestedVariant(), diagnostics.GetResolvedVariant(), diagnostics.GetRequestedRevision(), diagnostics.GetResolvedRevision(), route, variant, revision)
	}
	if diagnostics.GetFallback() || diagnostics.GetPreview() || diagnostics.GetNoindex() || diagnostics.GetNoStore() {
		t.Fatalf("renderer readiness was fallback/preview/private: fallback=%v preview=%v noindex=%v no_store=%v", diagnostics.GetFallback(), diagnostics.GetPreview(), diagnostics.GetNoindex(), diagnostics.GetNoStore())
	}
	if value.GetMode() != string(mode) || value.GetScope() != string(scope) || value.GetAppKey() != appKey || value.GetPage() == nil || value.GetPage().GetId() != pageID || len(value.GetPage().GetBlocks()) == 0 || len(value.GetAssets()) == 0 || len(value.GetFixtures()) == 0 {
		t.Fatalf("renderer readiness incomplete: mode=%q blocks=%d capabilities=%d assets=%d fixtures=%d", value.GetMode(), len(value.GetPage().GetBlocks()), len(value.GetCapabilities()), len(value.GetAssets()), len(value.GetFixtures()))
	}
	// The configured Backdrop page has no capability claims. It must not inherit
	// Aquila's six claims merely because both apps belong to the fixture bundle.
	wantCapabilities := 6
	if appKey == "backdrop-studio" {
		wantCapabilities = 0
	}
	if len(value.GetCapabilities()) != wantCapabilities {
		t.Fatalf("fixture capability projection for %q = %d, want %d", pageID, len(value.GetCapabilities()), wantCapabilities)
	}
	if diagnostics.GetBlockDigest() == "" {
		t.Fatal("renderer readiness omitted block digest")
	}
	if value.GetPage().GetFooter().GetLabel() != "Configured footer" || value.GetPage().GetDisplay().GetShell().GetFooterNote() != "Isolated integration fixture — not a public release" {
		t.Fatal("fixture presentation omitted its configured disclosure")
	}
}

func assertParity(t *testing.T, root, detail *sharedv1.ResolvedProductPresentation) {
	t.Helper()
	if root.GetDiagnostics().GetResolvedRoute() != "/" || detail.GetDiagnostics().GetResolvedRoute() != "/apps/aquila" {
		t.Fatalf("root/detail route identity = %q/%q", root.GetDiagnostics().GetResolvedRoute(), detail.GetDiagnostics().GetResolvedRoute())
	}
	if root.GetDiagnostics().GetResolvedVariant() != detail.GetDiagnostics().GetResolvedVariant() || root.GetDiagnostics().GetResolvedRevision() != detail.GetDiagnostics().GetResolvedRevision() {
		t.Fatalf("root/detail publication identity mismatch: variant %q/%q revision %q/%q", root.GetDiagnostics().GetResolvedVariant(), detail.GetDiagnostics().GetResolvedVariant(), root.GetDiagnostics().GetResolvedRevision(), detail.GetDiagnostics().GetResolvedRevision())
	}
	if root.GetPage().GetFooter().GetLabel() != "Configured footer" || detail.GetPage().GetFooter().GetLabel() != "Configured footer" {
		t.Fatal("root/detail omitted configured footer disclosure")
	}
	if root.GetDiagnostics().GetBlockDigest() != detail.GetDiagnostics().GetBlockDigest() {
		t.Fatalf("single-app root/detail digest mismatch: %q != %q", root.GetDiagnostics().GetBlockDigest(), detail.GetDiagnostics().GetBlockDigest())
	}
	if !proto.Equal(root.GetPage(), detail.GetPage()) {
		t.Fatal("single-app root/detail page payload mismatch despite shared digest")
	}
	if !slices.Equal(presentationBlockIDs(root), presentationBlockIDs(detail)) || !slices.Equal(presentationCapabilityIDs(root), presentationCapabilityIDs(detail)) || !slices.Equal(presentationAssetIDs(root), presentationAssetIDs(detail)) || !slices.Equal(presentationFixtureIDs(root), presentationFixtureIDs(detail)) {
		t.Fatalf("root/detail resource identity mismatch: blocks %v/%v capabilities %v/%v assets %v/%v fixtures %v/%v", presentationBlockIDs(root), presentationBlockIDs(detail), presentationCapabilityIDs(root), presentationCapabilityIDs(detail), presentationAssetIDs(root), presentationAssetIDs(detail), presentationFixtureIDs(root), presentationFixtureIDs(detail))
	}
}

func assertContentParity(t *testing.T, singleDetail, bundleDetail *sharedv1.ResolvedProductPresentation) {
	t.Helper()
	if singleDetail.GetDiagnostics().GetBlockDigest() != bundleDetail.GetDiagnostics().GetBlockDigest() || !proto.Equal(singleDetail.GetPage(), bundleDetail.GetPage()) || !slices.Equal(presentationBlockIDs(singleDetail), presentationBlockIDs(bundleDetail)) || !slices.Equal(presentationCapabilityIDs(singleDetail), presentationCapabilityIDs(bundleDetail)) || !slices.Equal(presentationAssetIDs(singleDetail), presentationAssetIDs(bundleDetail)) || !slices.Equal(presentationFixtureIDs(singleDetail), presentationFixtureIDs(bundleDetail)) {
		t.Fatal("Aquila detail content differs between single-app and bundle resolution")
	}
}

func presentationBlockIDs(value *sharedv1.ResolvedProductPresentation) []string {
	result := make([]string, 0, len(value.GetPage().GetBlocks()))
	for _, block := range value.GetPage().GetBlocks() {
		result = append(result, block.GetId()+":"+block.GetKind())
	}
	return result
}

func presentationCapabilityIDs(value *sharedv1.ResolvedProductPresentation) []string {
	result := make([]string, 0, len(value.GetCapabilities()))
	for _, capability := range value.GetCapabilities() {
		result = append(result, capability.GetId())
	}
	return result
}

func presentationAssetIDs(value *sharedv1.ResolvedProductPresentation) []string {
	result := make([]string, 0, len(value.GetAssets()))
	for _, asset := range value.GetAssets() {
		result = append(result, asset.GetId())
	}
	return result
}

func presentationFixtureIDs(value *sharedv1.ResolvedProductPresentation) []string {
	result := make([]string, 0, len(value.GetFixtures()))
	for _, fixture := range value.GetFixtures() {
		result = append(result, fixture.GetId())
	}
	return result
}

func buildProductionVite(t *testing.T, distDir string) {
	t.Helper()
	if err := os.MkdirAll(distDir, 0o700); err != nil {
		t.Fatalf("create unique Vite output: %v", err)
	}
	buildContext, cancelBuild := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancelBuild()
	command := exec.CommandContext(buildContext, "pnpm", "exec", "vite", "build", "--outDir", distDir, "--emptyOutDir")
	command.Dir = filepath.Join("..", "ui")
	command.Env = append(os.Environ(), "NODE_ENV=production")
	output, err := browserEvidenceCommand(command)
	if err != nil {
		t.Fatalf("production Vite build failed: %v\n%s", err, output)
	}
	if _, err := os.Stat(filepath.Join(distDir, "index.html")); err != nil {
		t.Fatalf("production Vite build omitted index.html: %v", err)
	}
}

func mustURLPort(t *testing.T, raw string) int {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	port := parsed.Port()
	value, err := strconv.Atoi(port)
	if err != nil || value <= 0 {
		t.Fatalf("invalid test API port %q", port)
	}
	return value
}
