package main

import (
	"net/http"
	"time"

	seohttp "landing-page-business-suite-api/handlers/seo"
)

func seoDependencies(service *SEOService) seohttp.Dependencies {
	deps := seohttp.Dependencies{
		Path:       getPathParam,
		DecodeJSON: decodeJSONBody,
		WriteJSON:  writeJSONSuccessData,
		WriteError: writeJSONError,
		Log:        logStructuredError,
		Now:        time.Now,
	}
	if service != nil {
		deps.Sitemap = service.SitemapXML
		deps.Robots = service.RobotsTXT
	}
	return deps
}

func handleSitemapXML(service *SEOService) http.HandlerFunc {
	return seohttp.Sitemap(seoDependencies(service))
}

func handleRobotsTXT(service *SEOService) http.HandlerFunc {
	return seohttp.Robots(seoDependencies(service))
}
