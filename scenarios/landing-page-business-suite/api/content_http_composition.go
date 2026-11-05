package main

import (
	contenthttp "landing-page-business-suite-api/handlers/content"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/logx"
)

// contentHTTPDependencies injects API response and route conventions into the
// content transport while the handler package owns request behavior.
func contentHTTPDependencies(store *experimentation.ConfigStore) contenthttp.Dependencies {
	return contenthttp.Dependencies{
		AllSections: func(slug string) (any, error) {
			variant, err := store.GetVariant(slug)
			if err != nil {
				return nil, err
			}
			return variant.Sections, nil
		},
		Path:       getPathParam,
		WriteJSON:  writeJSON,
		WriteError: writeJSONError,
		Log:        logx.Error,
	}
}
