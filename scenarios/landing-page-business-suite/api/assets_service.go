package main

import (
	"landing-page-business-suite-api/internal/content"
	"landing-page-business-suite-api/internal/envx"
)

type (
	Asset              = content.Asset
	AssetUploadRequest = content.AssetUploadRequest
	AssetsService      = content.AssetsService
	AssetStore         = content.AssetStore
	AssetsOptions      = content.AssetsOptions
)

var (
	ErrAssetNotFound    = content.ErrAssetNotFound
	ErrInvalidFileType  = content.ErrInvalidFileType
	ErrFileTooLarge     = content.ErrFileTooLarge
	ErrUploadFailed     = content.ErrUploadFailed
	ErrInvalidAssetPath = content.ErrInvalidAssetPath
)

func NewAssetsService(db AssetStore) *AssetsService {
	return NewAssetsServiceWithOptions(AssetsOptions{DB: db, UploadDir: envx.Get("UPLOAD_DIR"), LogError: logStructuredError})
}

func NewAssetsServiceWithOptions(opts AssetsOptions) *AssetsService {
	if opts.LogError == nil {
		opts.LogError = logStructuredError
	}
	return content.NewAssetsServiceWithOptions(opts)
}

func detectMimeType(filename string) string { return content.DetectMimeType(filename) }
