package main

import (
	"context"
	"errors"
	"net/http"

	assethttp "landing-page-business-suite-api/handlers/assets"
)

func assetDependencies(service *AssetsService) assethttp.Dependencies {
	return assethttp.Dependencies{
		Upload: func(ctx context.Context, input assethttp.UploadInput) (assethttp.UploadResult, error) {
			asset, err := service.UploadContext(ctx, &AssetUploadRequest{File: input.File, Header: input.Header, Category: input.Category, AltText: input.AltText, UploadedBy: input.UploadedBy})
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
		IsNotFound:         func(err error) bool { return errors.Is(err, ErrAssetNotFound) },
		DetectMimeType:     detectMimeType,
	}
}

// handleAssetUpload is the multipart REST exception documented by assets.proto.
func handleAssetUpload(service *AssetsService) http.HandlerFunc {
	return assethttp.Upload(assetDependencies(service))
}
func handleAssetsList(service *AssetsService) http.HandlerFunc {
	return assethttp.List(assetDependencies(service))
}
func handleAssetGet(service *AssetsService) http.HandlerFunc {
	return assethttp.Get(assetDependencies(service))
}
func handleAssetDelete(service *AssetsService) http.HandlerFunc {
	return assethttp.Delete(assetDependencies(service))
}

//nolint:unused // reserved for debug-only asset preview handler
func handleServeUpload(service *AssetsService) http.HandlerFunc {
	return assethttp.Serve(assetDependencies(service))
}
