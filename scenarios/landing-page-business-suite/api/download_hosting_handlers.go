package main

import (
	"context"
	"errors"

	downloadhttp "landing-page-business-suite-api/handlers/delivery"
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/delivery"
)

// downloadAdminDependencies is the production composition boundary for delivery HTTP transport.
func downloadAdminDependencies(hosting *delivery.Service, plans *commerce.PlanService) downloadhttp.AdminDependencies {
	return downloadhttp.AdminDependencies{
		BundleKey: plans.BundleKey, SettingsSnapshot: hosting.SettingsSnapshot, SaveSettings: hosting.SaveSettings,
		TestConnection: hosting.TestConnection, ListArtifacts: hosting.ListArtifacts, ListArtifactsByApp: hosting.ListArtifactsByApp,
		PresignUpload: hosting.PresignUpload, CommitArtifact: hosting.CommitArtifact, GetArtifact: hosting.GetArtifact,
		PresignGetArtifact: hosting.PresignGetArtifact, DecodeJSON: decodeJSONBody, PathInt64: getPathParamInt64,
		WriteSuccessData: writeJSONSuccessData, WriteSuccessSimple: writeJSONSuccessSimple, WriteError: writeJSONError,
	}
}

func downloadAdminAssetDependencies(downloads *delivery.CatalogService, hosting *delivery.Service, plans *commerce.PlanService) downloadhttp.AdminDependencies {
	deps := downloadAdminDependencies(hosting, plans)
	deps.GetManagedAsset = func(bundleKey, appKey, platform string) (*downloadhttp.ManagedAsset, error) {
		asset, err := downloads.GetAsset(bundleKey, appKey, platform)
		if asset == nil || err != nil {
			return nil, err
		}
		return &downloadhttp.ManagedAsset{BundleKey: asset.BundleKey, AppKey: asset.AppKey, Platform: asset.Platform, VariantKey: asset.VariantKey, ArtifactURL: asset.ArtifactURL, ArtifactSource: asset.ArtifactSource, ArtifactID: asset.ArtifactID, ReleaseVersion: asset.ReleaseVersion, ReleaseNotes: asset.ReleaseNotes, Checksum: asset.Checksum, RequiresEntitlement: asset.RequiresEntitlement, Metadata: asset.Metadata}, nil
	}
	deps.UpsertManagedAsset = func(ctx context.Context, asset downloadhttp.ManagedAsset) (any, error) {
		return downloads.UpsertAsset(ctx, delivery.Asset{BundleKey: asset.BundleKey, AppKey: asset.AppKey, Platform: asset.Platform, VariantKey: asset.VariantKey, ArtifactURL: asset.ArtifactURL, ArtifactSource: asset.ArtifactSource, ArtifactID: asset.ArtifactID, ReleaseVersion: asset.ReleaseVersion, ReleaseNotes: asset.ReleaseNotes, Checksum: asset.Checksum, RequiresEntitlement: asset.RequiresEntitlement, Metadata: asset.Metadata})
	}
	deps.IsAssetNotFound = func(err error) bool { return errors.Is(err, delivery.ErrAssetNotFound) }
	return deps
}
