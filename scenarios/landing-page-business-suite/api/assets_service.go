package main

import (
	"landing-page-business-suite-api/internal/content"
	"landing-page-business-suite-api/internal/envx"
)

func NewAssetsService(db content.AssetStore) *content.AssetsService {
	return NewAssetsServiceWithOptions(content.AssetsOptions{DB: db, UploadDir: envx.Get("UPLOAD_DIR"), LogError: logStructuredError})
}

func NewAssetsServiceWithOptions(opts content.AssetsOptions) *content.AssetsService {
	if opts.LogError == nil {
		opts.LogError = logStructuredError
	}
	return content.NewAssetsServiceWithOptions(opts)
}

func detectMimeType(filename string) string { return content.DetectMimeType(filename) }
