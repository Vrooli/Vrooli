//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"backdrop-studio/integration"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/discovery"
)

// The release lane, across the Asset Studio boundary.
//
// The unit tests prove the rules; this proves the wire. Both halves are needed
// for the same reason the render lane exists: this scenario has already shipped
// a feature that every unit suite passed and no real caller could use, because
// the two sides of a boundary each tested their own side of it.

func assetStudioURL(t *testing.T) string {
	t.Helper()
	url, err := discovery.ResolveScenarioURLDefault(context.Background(), "asset-studio")
	if err != nil {
		t.Skipf("asset-studio is not reachable: %v", err)
	}
	return strings.TrimRight(url, "/")
}

func postJSON(t *testing.T, url string, payload any, out any) (int, string) {
	t.Helper()
	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(encoded))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: time.Minute}).Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 && out != nil {
		require.NoError(t, json.Unmarshal(body, out), "body: %s", body)
	}
	return resp.StatusCode, string(body)
}

func onePixelPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 30), G: uint8(y * 30), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func ingestPayload(t *testing.T, provenance map[string]any) map[string]any {
	t.Helper()
	return map[string]any{
		"image":      base64.StdEncoding.EncodeToString(onePixelPNG(t)),
		"mediaType":  "image/png",
		"altText":    "A blue duotone halftone of a colonnade above a bay",
		"width":      8,
		"height":     8,
		"actorId":    "backdrop-studio-integration",
		"actorKind":  "agent",
		"provenance": provenance,
	}
}

// TestAssetStudioAcceptsAScaffoldConditionedIngress is the cross-scenario
// conformance check PROBLEMS.md asked for, run against the really running
// service rather than the domain object.
//
// The recorded worry was that Asset Studio's composition path assumes an
// identity binding — a character, a product, a scene — and that a producer
// binding a scaffold and a palette would not fit through it. This sends exactly
// that: no identity, a `scaffold` conditioning kind, and a model-backed claim.
func TestAssetStudioAcceptsAScaffoldConditionedIngress(t *testing.T) {
	base := assetStudioURL(t)
	var ingested struct {
		Asset struct {
			ID          string `json:"id"`
			Status      string `json:"status"`
			Disclosure  string `json:"disclosure"`
			AIGenerated bool   `json:"aiGenerated"`
		} `json:"asset"`
	}
	status, body := postJSON(t, base+"/vrooli.asset_studio.v1.studio.StudioService/IngestExternalAsset",
		ingestPayload(t, map[string]any{
			"producingScenario": "backdrop-studio",
			"strategy":          "guided",
			"modelBacked":       true,
			"model":             "sd-1.5/local-gpu",
			"prompt":            "sunlit modernist interior, tall windows, long shadows",
			"seed":              "7",
			"conditioning":      map[string]any{"kind": "scaffold", "id": "arcade", "version": "edge"},
		}), &ingested)
	require.Equalf(t, http.StatusOK, status, "ingest refused a non-identity conditioning kind: %s", body)
	require.NotEmpty(t, ingested.Asset.ID)
	require.Equal(t, "in_review", ingested.Asset.Status,
		"an ingested asset must land in review, not released: the ingress is a door into the release path, not around it")
	require.True(t, ingested.Asset.AIGenerated)
	require.Contains(t, ingested.Asset.Disclosure, "sd-1.5/local-gpu")
	require.Contains(t, ingested.Asset.Disclosure, "scaffold arcade@edge")

	var released struct {
		Asset struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"asset"`
	}
	status, body = postJSON(t, base+"/vrooli.asset_studio.v1.studio.StudioService/ReleaseAsset",
		map[string]any{"assetId": ingested.Asset.ID}, &released)
	require.Equalf(t, http.StatusOK, status, "release of an ingested asset failed: %s", body)
	require.Equal(t, "released", released.Asset.Status)

	// The reference must resolve afterwards, or the release produced a record
	// nobody can fetch.
	var fetched struct {
		Asset struct {
			ID         string `json:"id"`
			Status     string `json:"status"`
			Disclosure string `json:"disclosure"`
		} `json:"asset"`
	}
	status, body = postJSON(t, base+"/vrooli.asset_studio.v1.studio.StudioService/GetReleasedAssetReference",
		map[string]any{"assetId": ingested.Asset.ID}, &fetched)
	require.Equalf(t, http.StatusOK, status, "released reference did not resolve: %s", body)
	require.Equal(t, "released", fetched.Asset.Status)
	require.Contains(t, fetched.Asset.Disclosure, "Prompt:")
}

// TestAssetStudioRefusesUnlabelledSyntheticMedia is the rule that makes the
// disclosure worth carrying. A synthetic image with no model or no prompt
// cannot be reproduced or audited, so it must not be admitted at all.
func TestAssetStudioRefusesUnlabelledSyntheticMedia(t *testing.T) {
	base := assetStudioURL(t)
	for _, tc := range []struct {
		name       string
		provenance map[string]any
		want       string
	}{
		{"no model", map[string]any{
			"producingScenario": "backdrop-studio", "strategy": "synthesized",
			"modelBacked": true, "prompt": "art nouveau celestial chart",
		}, "model"},
		{"no prompt", map[string]any{
			"producingScenario": "backdrop-studio", "strategy": "synthesized",
			"modelBacked": true, "model": "sd-1.5/local-gpu",
		}, "prompt"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := postJSON(t, base+"/vrooli.asset_studio.v1.studio.StudioService/IngestExternalAsset",
				ingestPayload(t, tc.provenance), nil)
			require.NotEqualf(t, http.StatusOK, status, "unlabelled synthetic media was admitted: %s", body)
			require.Contains(t, strings.ToLower(body), tc.want)
		})
	}
}

// TestProceduralReleaseNeedsNoAssetStudio keeps the two lanes distinct. A
// procedural backdrop is not synthetic media and must not acquire an
// AI-generated label or an Asset Studio round trip on its way out.
func TestProceduralReleaseNeedsNoAssetStudio(t *testing.T) {
	env, _ := newEnvironment(t)
	ctx := context.Background()

	styles, err := env.Styles(ctx)
	require.NoError(t, err)
	surfaces, err := env.Surfaces(ctx)
	require.NoError(t, err)

	var chosen integration.Style
	for _, style := range styles {
		if !style.ModelBacked() {
			chosen = style
			break
		}
	}
	require.NotEmpty(t, chosen.ID, "the catalog has no procedural style to release")

	permitted := integration.PermittedSurfaces(chosen, surfaces)
	require.NotEmpty(t, permitted)
	surface := permitted[len(permitted)-1]

	job, err := env.Submit(ctx, integration.SubmitOptions{
		StyleID: chosen.ID, Seed: 7, SurfaceID: surface.ID, Placement: chosen.Placements[0],
	})
	require.NoError(t, err)
	require.NotEmpty(t, job.Candidates)
	candidate := job.Candidates[0]

	var released struct {
		Backdrop struct {
			ID             string `json:"id"`
			AIGenerated    bool   `json:"aiGenerated"`
			AssetStudioRef string `json:"assetStudioRef"`
		} `json:"backdrop"`
	}
	status, body := postJSON(t, env.BackdropURL+"/vrooli.backdrop_studio.v1.release.ReleaseService/Release", map[string]any{
		"candidate_id": candidate.ID, "style_id": chosen.ID, "strategy": chosen.Strategy,
		"surface_id": surface.ID, "placement": chosen.Placements[0],
		"alt_text": "An ambient procedural backdrop", "width": candidate.Width, "height": candidate.Height,
		"legibility_passes": true, "contrast_ratio": 5.0, "contrast_threshold": 4.5,
		"image_png": base64.StdEncoding.EncodeToString(candidate.ImagePNG),
	}, &released)
	require.Equalf(t, http.StatusOK, status, "procedural release failed: %s", body)
	require.False(t, released.Backdrop.AIGenerated, "a procedural backdrop must not be labelled AI-generated")
	require.Empty(t, released.Backdrop.AssetStudioRef,
		"a procedural backdrop must not acquire an Asset Studio reference: it is not synthetic media")
}

// TestModelBackedReleaseGoesThroughAssetStudio is the payoff, and it is honest
// about what this host can prove.
//
// A model-backed candidate must exist before it can be released, and generating
// one at hero aspect exhausts device memory on the reference workstation. When
// that happens the test records a named skip rather than passing — a skip says
// "not proven here", which is true, where a pass would say "proven", which
// would not be.
func TestModelBackedReleaseGoesThroughAssetStudio(t *testing.T) {
	env, caps := newEnvironment(t)
	ctx := context.Background()
	if !caps.GenerationOK {
		t.Skip("SKIP(no-generation): image-tools reports no inference capability")
	}

	styles, err := env.Styles(ctx)
	require.NoError(t, err)
	surfaces, err := env.Surfaces(ctx)
	require.NoError(t, err)

	var attempted int
	for _, style := range styles {
		if !style.ModelBacked() {
			continue
		}
		attempted++
		permitted := integration.PermittedSurfaces(style, surfaces)
		require.NotEmpty(t, permitted)
		surface := permitted[0]

		job, submitErr := env.Submit(ctx, integration.SubmitOptions{
			StyleID: style.ID, Seed: 7, SurfaceID: surface.ID, Placement: style.Placements[0],
		})
		if submitErr != nil {
			if integration.IsGPUCapacityFailure(submitErr) {
				t.Logf("SKIP(gpu-capacity): %s could not be generated on this host: %v", style.ID, submitErr)
				continue
			}
			t.Errorf("model-backed style %q failed to render: %v", style.ID, submitErr)
			continue
		}
		require.NotEmpty(t, job.Candidates)
		candidate := job.Candidates[0]

		var released struct {
			Backdrop struct {
				ID             string `json:"id"`
				AIGenerated    bool   `json:"aiGenerated"`
				AssetStudioRef string `json:"assetStudioRef"`
			} `json:"backdrop"`
		}
		status, body := postJSON(t, env.BackdropURL+"/vrooli.backdrop_studio.v1.release.ReleaseService/Release", map[string]any{
			"candidate_id": candidate.ID, "style_id": style.ID, "strategy": style.Strategy,
			"surface_id": surface.ID, "placement": style.Placements[0],
			"alt_text": fmt.Sprintf("A model-backed backdrop in the %s style", style.ID),
			"width":    candidate.Width, "height": candidate.Height,
			"legibility_passes": true, "contrast_ratio": 5.0, "contrast_threshold": 4.5,
			"image_png": base64.StdEncoding.EncodeToString(candidate.ImagePNG),
		}, &released)
		require.Equalf(t, http.StatusOK, status, "model-backed release of %q failed: %s", style.ID, body)
		require.True(t, released.Backdrop.AIGenerated)
		require.NotEmptyf(t, released.Backdrop.AssetStudioRef,
			"model-backed style %q released without an Asset Studio reference: its provenance went nowhere", style.ID)
		t.Logf("released %s as asset %s", style.ID, released.Backdrop.AssetStudioRef)
	}
	require.Positive(t, attempted, "the catalog declares no model-backed style, so this lane proves nothing")
}
