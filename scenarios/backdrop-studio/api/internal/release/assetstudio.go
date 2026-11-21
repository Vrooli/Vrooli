package release

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AssetStudioPublisher hands released bytes to Asset Studio, which owns the
// provenance and disclosure record for every asset Vrooli publishes.
//
// Backdrop Studio deliberately does not write that record itself. It could —
// it knows the model, the prompt and the seed — and that is exactly why it must
// not: a second authority on disclosure is a second place for the rules to
// drift, and the rules here are the ones that decide whether a synthetic image
// is honestly labelled.
//
// Two calls, in this order and for a reason. `IngestExternalAsset` admits the
// bytes and returns an asset in review; `ReleaseAsset` runs Asset Studio's own
// release checks over it. Skipping the second would leave a record that exists
// but was never released, which reads to every consumer as an asset that failed
// review.
type AssetStudioPublisher struct {
	// Resolve locates asset-studio. It is a function rather than a URL so a
	// scenario that is not running produces a named missing capability at the
	// moment of use, rather than a connection refused at startup.
	Resolve    func(context.Context) (string, error)
	HTTPClient *http.Client
}

const (
	ingestProcedure  = "/vrooli.asset_studio.v1.studio.StudioService/IngestExternalAsset"
	releaseProcedure = "/vrooli.asset_studio.v1.studio.StudioService/ReleaseAsset"
)

// Publish stores the bytes with their provenance and returns the released
// asset reference.
func (p *AssetStudioPublisher) Publish(ctx context.Context, r Request, prov Provenance) (string, error) {
	if p == nil || p.Resolve == nil {
		return "", fmt.Errorf("asset-studio capability is not configured")
	}
	base, err := p.Resolve(ctx)
	if err != nil {
		return "", fmt.Errorf("asset-studio is unavailable: %w", err)
	}
	base = strings.TrimRight(base, "/")

	// Conditioning is described generically. A scaffold is not an identity, and
	// the ingress contract was written so that it does not have to pretend to
	// be one.
	conditioning := map[string]any{}
	if prov.Conditioner != "" {
		conditioning = map[string]any{
			"kind":    "scaffold",
			"id":      r.StyleID,
			"version": prov.Conditioner,
		}
	}

	// The Connect edge takes camelCase JSON names, and `bytes` fields arrive
	// base64-encoded. Both are easy to get wrong in the direction that produces
	// a confusing 400 rather than an obvious one — this scenario has already
	// paid for the camelCase half twice.
	ingestBody := map[string]any{
		"image":      base64.StdEncoding.EncodeToString(r.ImagePNG),
		"mediaType":  "image/png",
		"altText":    r.AltText,
		"decorative": r.Decorative,
		"width":      r.Width,
		"height":     r.Height,
		"actorId":    "backdrop-studio",
		"actorKind":  "agent",
		"provenance": map[string]any{
			"producingScenario": "backdrop-studio",
			"strategy":          prov.Strategy,
			"modelBacked":       prov.ModelBacked,
			"model":             prov.Model,
			"prompt":            prov.Prompt,
			"negativePrompt":    prov.Negative,
			"seed":              prov.Seed,
			"conditioning":      conditioning,
			"parameters":        prov.Parameters,
		},
	}

	var ingested struct {
		Asset struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"asset"`
	}
	if err := p.post(ctx, base+ingestProcedure, ingestBody, &ingested); err != nil {
		return "", fmt.Errorf("ingest: %w", err)
	}
	if ingested.Asset.ID == "" {
		return "", fmt.Errorf("ingest returned no asset id")
	}

	var released struct {
		Asset struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"asset"`
	}
	if err := p.post(ctx, base+releaseProcedure, map[string]any{"assetId": ingested.Asset.ID}, &released); err != nil {
		return "", fmt.Errorf("release asset %s: %w", ingested.Asset.ID, err)
	}
	if released.Asset.Status != "released" {
		return "", fmt.Errorf("asset %s did not reach released status (got %q)", ingested.Asset.ID, released.Asset.Status)
	}
	return released.Asset.ID, nil
}

func (p *AssetStudioPublisher) post(ctx context.Context, url string, payload, out any) error {
	client := p.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Minute}
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Asset Studio's own message is the finding — "a model-backed asset
		// must name the model that produced it" is more useful than a status
		// code, and it is the message a caller needs to fix the request.
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(detail)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
