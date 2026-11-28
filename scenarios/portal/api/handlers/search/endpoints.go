package search

import (
	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/search/search_v1connect"

	"portal/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "search_suggest", Path: searchconnect.SearchServiceSuggestProcedure, Method: "POST", Summary: "Suggest ecosystem results", Description: "Returns bounded, typed search-hub projections for the composer omnibox.", Category: "search"},
}
