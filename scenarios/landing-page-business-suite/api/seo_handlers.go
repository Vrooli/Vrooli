package main

import (
	"encoding/json"
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
		deps.VariantSEO = func(slug string) (any, error) { return service.VariantSEO(slug) }
		deps.Sitemap = service.SitemapXML
		deps.Robots = service.RobotsTXT
	}
	return deps
}

func handleGetVariantSEO(service *SEOService) http.HandlerFunc {
	return seohttp.Variant(seoDependencies(service))
}

func handleUpdateVariantSEOConfigStore(store *ConfigStore) http.HandlerFunc {
	deps := seoDependencies(nil)
	deps.Update = func(slug string, config json.RawMessage) (bool, error) {
		variant, err := store.GetVariant(slug)
		if err != nil {
			return false, err
		}
		variant.Variant.SEOConfig = config
		return true, store.SaveVariant(slug, variant)
	}
	return seohttp.Update(deps)
}

func handleSitemapXML(service *SEOService) http.HandlerFunc {
	return seohttp.Sitemap(seoDependencies(service))
}
func handleRobotsTXT(service *SEOService) http.HandlerFunc {
	return seohttp.Robots(seoDependencies(service))
}
