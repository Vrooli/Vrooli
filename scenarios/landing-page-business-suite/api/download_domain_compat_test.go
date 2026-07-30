package main

import "landing-page-business-suite-api/internal/delivery"

// Legacy root-package tests characterize the delivery migration. Production
// code imports the delivery domain directly; these aliases are test-only.
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
