package main

import (
	"context"
	"errors"

	assethttp "landing-page-business-suite-api/handlers/assets"
	"landing-page-business-suite-api/internal/content"
)

// assetsHTTPDependencies injects API conventions into the assets transport.
func assetsHTTPDependencies(service *content.AssetsService) assethttp.Dependencies {
	return assethttp.Dependencies{
		Upload: func(ctx context.Context, input assethttp.UploadInput) (assethttp.UploadResult, error) {
			asset, err := service.UploadContext(ctx, &content.AssetUploadRequest{File: input.File, Header: input.Header, Category: input.Category, AltText: input.AltText, UploadedBy: input.UploadedBy})
			if err != nil {
				return assethttp.UploadResult{}, err
			}
			return assethttp.UploadResult{Payload: asset, ID: asset.ID, Filename: asset.Filename, Category: asset.Category, Size: asset.SizeBytes}, nil
		},
		List: func(category string) (any, bool, error) {
			assets, err := service.List(category)
			return assets, assets == nil, err
		},
		Get:                func(id int) (any, error) { return service.Get(id) },
		Delete:             func(ctx context.Context, id int) error { return service.DeleteContext(ctx, id) },
		ResolveStoragePath: service.ResolveStoragePathContext,
		PathInt:            getPathParamInt,
		Path:               getPathParam,
		WriteError:         writeJSONError,
		WriteJSON:          writeJSON,
		Log:                logStructuredError,
		IsNotFound:         func(err error) bool { return errors.Is(err, content.ErrAssetNotFound) },
		DetectMimeType:     detectMimeType,
	}
}
