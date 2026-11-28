package main

import (
	"landing-page-business-suite-api/internal/content"
	"landing-page-business-suite-api/internal/envx"
	"landing-page-business-suite-api/internal/logx"
)

func NewAssetsService(db content.AssetStore) *content.AssetsService {
	return NewAssetsServiceWithOptions(content.AssetsOptions{DB: db, UploadDir: envx.Get("UPLOAD_DIR"), LogError: logx.Error})
}

func NewAssetsServiceWithOptions(opts content.AssetsOptions) *content.AssetsService {
	if opts.LogError == nil {
		opts.LogError = logx.Error
	}
	return content.NewAssetsServiceWithOptions(opts)
}

func detectMimeType(filename string) string { return content.DetectMimeType(filename) }
