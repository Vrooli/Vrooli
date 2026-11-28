package seo

import (
	"landing-page-react-vite-api/internal/module"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"
)

// Endpoints describes the seo module's surface for codegen: two Connect RPCs
// plus the raw sitemap.xml / robots.txt endpoints, which serve
// crawler-standard text/xml payloads and are declared as REST exceptions.
var Endpoints = []module.EndpointDescriptor{
	{
		ID: "seo_get_variant", Path: landingconnect.SeoServiceGetVariantSEOProcedure, Method: "POST",
		Summary: "Get variant SEO", Description: "Returns the resolved SEO head payload for a variant (public).", Category: "seo",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"slug": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"*": "resolved SEO head fields"}},
	},
	{
		ID: "seo_update_variant", Path: landingconnect.SeoServiceUpdateVariantSEOProcedure, Method: "POST",
		Summary: "Update variant SEO", Description: "Stores SEO overrides for a variant (admin).", Category: "seo",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"slug": "string", "config": "VariantSEOConfig"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"success": "bool", "updated_at": "string"}},
	},
	{
		ID: "seo_sitemap", Path: "/sitemap.xml", Method: "GET",
		Summary: "Sitemap XML", Description: "Serves the dynamic sitemap.xml for active, indexable variants.", Category: "seo",
		Response: &module.Schema{Type: "string", Properties: map[string]string{"content_type": "application/xml"}},
		RESTException: &module.RESTException{
			Reason: module.RESTReasonThirdPartyShape,
			Note:   "sitemaps.org XML schema consumed by search-engine crawlers; not a Connect payload.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
				Response: module.RESTPayload{Transport: "xml", Conformance: "none"},
			},
		},
	},
	{
		ID: "seo_robots", Path: "/robots.txt", Method: "GET",
		Summary: "Robots TXT", Description: "Serves the configurable robots.txt with a sitemap reference.", Category: "seo",
		Response: &module.Schema{Type: "string", Properties: map[string]string{"content_type": "text/plain"}},
		RESTException: &module.RESTException{
			Reason: module.RESTReasonThirdPartyShape,
			Note:   "Robots Exclusion Standard plain-text format consumed by crawlers; not a Connect payload.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
				Response: module.RESTPayload{Transport: "text", Conformance: "none"},
			},
		},
	},
}
