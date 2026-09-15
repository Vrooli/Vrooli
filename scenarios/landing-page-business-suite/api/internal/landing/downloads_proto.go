package landing

import (
	"fmt"
	"math"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"google.golang.org/protobuf/types/known/structpb"
	"landing-page-business-suite-api/internal/delivery"
)

// ProtoDownloads owns the delivery-to-public-landing projection. The landing
// aggregate deliberately contains delivery metadata in its stable public
// response, so this boundary belongs to landing—not its HTTP handler.
func ProtoDownloads(downloads []delivery.App) ([]*sharedv1.DownloadApp, error) {
	result := make([]*sharedv1.DownloadApp, 0, len(downloads))
	for index, app := range downloads {
		metadata, err := structpb.NewStruct(app.Metadata)
		if err != nil {
			return nil, fmt.Errorf("download %d (%q) metadata: %w", index, app.AppKey, err)
		}
		policy, err := structpb.NewStruct(app.UpdatePolicy)
		if err != nil {
			return nil, fmt.Errorf("download %d (%q) update policy: %w", index, app.AppKey, err)
		}
		platforms, err := protoAssets(app.Platforms)
		if err != nil {
			return nil, fmt.Errorf("download %d (%q): %w", index, app.AppKey, err)
		}
		storefronts := make([]*sharedv1.DownloadStorefront, 0, len(app.Storefronts))
		for _, storefront := range app.Storefronts {
			storefronts = append(storefronts, &sharedv1.DownloadStorefront{Store: storefront.Store, Label: storefront.Label, Url: storefront.URL, Badge: storefront.Badge})
		}
		order, err := downloadInt32(app.DisplayOrder, fmt.Sprintf("download %d (%q) display order", index, app.AppKey))
		if err != nil {
			return nil, err
		}
		result = append(result, &sharedv1.DownloadApp{Id: app.ID, BundleKey: app.BundleKey, AppKey: app.AppKey, Name: app.Name, Tagline: app.Tagline, Description: app.Description, IconUrl: app.IconURL, ScreenshotUrl: app.ScreenshotURL, InstallOverview: app.InstallOverview, InstallSteps: app.InstallSteps, Storefronts: storefronts, Metadata: metadata, DisplayOrder: order, UpdateApiKey: app.UpdateAPIKey, UpdatePolicy: policy, Platforms: platforms})
	}
	return result, nil
}

// ProtoPresentationDownloads projects delivery facts for the typed
// presentation response. Presentation copy owns names and instructions; the
// owner catalog contributes only stable identity, chooser facts, and the
// validated web launch destination. In particular, installer URLs and
// delivery metadata stay behind the authorized download operation.
func ProtoPresentationDownloads(downloads []delivery.App) ([]*sharedv1.DownloadApp, error) {
	public := make([]delivery.App, 0, len(downloads))
	for _, app := range downloads {
		metadata := map[string]interface{}{}
		if raw, ok := app.Metadata["web_url"].(string); ok {
			if webURL := presentationWebURLValue(raw); webURL != "" {
				metadata["web_url"] = webURL
			}
		}
		item := delivery.App{
			ID: app.ID, BundleKey: app.BundleKey, AppKey: app.AppKey,
			Metadata:  metadata,
			Platforms: make([]delivery.Asset, 0, len(app.Platforms)),
		}
		for _, asset := range app.Platforms {
			item.Platforms = append(item.Platforms, delivery.Asset{
				ID: asset.ID, BundleKey: asset.BundleKey, AppKey: asset.AppKey,
				Platform: asset.Platform, ReleaseVersion: asset.ReleaseVersion,
				Checksum: asset.Checksum, RequiresEntitlement: asset.RequiresEntitlement,
				// Keep the stable artifact lookup identity available to the
				// authorized delivery flow; never expose its source or URL.
				ArtifactID: asset.ArtifactID,
			})
		}
		public = append(public, item)
	}
	return ProtoDownloads(public)
}

func protoAssets(assets []delivery.Asset) ([]*sharedv1.DownloadAsset, error) {
	result := make([]*sharedv1.DownloadAsset, 0, len(assets))
	for index, asset := range assets {
		metadata, err := structpb.NewStruct(asset.Metadata)
		if err != nil {
			return nil, fmt.Errorf("asset %d (%q) metadata: %w", index, asset.Platform, err)
		}
		count, err := downloadInt32(asset.ArtifactCount, fmt.Sprintf("asset %d (%q) artifact count", index, asset.Platform))
		if err != nil {
			return nil, err
		}
		result = append(result, &sharedv1.DownloadAsset{Id: asset.ID, BundleKey: asset.BundleKey, AppKey: asset.AppKey, Platform: asset.Platform, ArtifactUrl: asset.ArtifactURL, ReleaseVersion: asset.ReleaseVersion, ReleaseNotes: asset.ReleaseNotes, Checksum: asset.Checksum, RequiresEntitlement: asset.RequiresEntitlement, Metadata: metadata, ArtifactSource: asset.ArtifactSource, ArtifactId: asset.ArtifactID, VariantKey: asset.VariantKey, ArtifactFilename: asset.ArtifactFilename, ArtifactSizeBytes: asset.ArtifactSizeBytes, ArtifactCount: count})
	}
	return result, nil
}

func downloadInt32(value int, field string) (int32, error) {
	if value < math.MinInt32 || value > math.MaxInt32 {
		return 0, fmt.Errorf("%s %d is outside the protobuf int32 range", field, value)
	}
	return int32(value), nil
}
