package main

import "landing-page-business-suite-api/internal/delivery"

type (
	DownloadStore      = delivery.CatalogStore
	DownloadService    = delivery.CatalogService
	DownloadAsset      = delivery.Asset
	DownloadStorefront = delivery.Storefront
	DownloadApp        = delivery.App
)

var (
	ErrDownloadNotFound    = delivery.ErrAssetNotFound
	ErrDownloadAppNotFound = delivery.ErrAppNotFound
)
