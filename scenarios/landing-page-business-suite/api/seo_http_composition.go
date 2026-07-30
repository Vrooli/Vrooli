package main

import (
	"time"

	seohttp "landing-page-business-suite-api/handlers/seo"
	"landing-page-business-suite-api/internal/content"
)

// seoHTTPDependencies injects API conventions into the SEO transport.
func seoHTTPDependencies(service *content.SEOService) seohttp.Dependencies {
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
