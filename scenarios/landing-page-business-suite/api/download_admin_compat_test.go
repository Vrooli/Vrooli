package main

import (
	"net/http"

	downloadhttp "landing-page-business-suite-api/handlers/delivery"
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/delivery"
)

type (
	downloadAppRequest   = downloadhttp.AppRequest
	downloadAssetRequest = downloadhttp.AssetRequest
)

func handleAdminListDownloadApps(d *delivery.CatalogService, p *commerce.PlanService) http.HandlerFunc {
	return downloadhttp.ListApps(deliveryAppDependencies(p), d)
}

func handleAdminCreateDownloadApp(d *delivery.CatalogService, p *commerce.PlanService) http.HandlerFunc {
	return downloadhttp.CreateApp(deliveryAppDependencies(p), d)
}

func handleAdminSaveDownloadApp(d *delivery.CatalogService, p *commerce.PlanService) http.HandlerFunc {
	return downloadhttp.SaveApp(deliveryAppDependencies(p), d)
}

func handleAdminDeleteDownloadApp(d *delivery.CatalogService, p *commerce.PlanService) http.HandlerFunc {
	return downloadhttp.DeleteApp(deliveryAppDependencies(p), d)
}

func buildDownloadAppFromPayload(p downloadAppRequest, b, k string) (delivery.App, error) {
	return downloadhttp.BuildAppFromPayload(p, b, k)
}
func filterStrings(v []string) []string { return downloadhttp.FilterStrings(v) }
