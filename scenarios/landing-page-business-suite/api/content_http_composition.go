package main

import (
	contenthttp "landing-page-business-suite-api/handlers/content"
	"landing-page-business-suite-api/internal/experimentation"
)

// contentHTTPDependencies injects API response and route conventions into the
// content transport while the handler package owns request behavior.
func contentHTTPDependencies(store *experimentation.ConfigStore) contenthttp.Dependencies {
	return contenthttp.Dependencies{
		PublicSections: func(slug string) (any, error) {
			variant, err := store.GetVariant(slug)
			if err != nil {
				return nil, err
			}
			sections := make([]experimentation.VariantSection, 0)
			for _, section := range variant.Sections {
				if section.Enabled {
					sections = append(sections, section)
				}
			}
			return sections, nil
		},
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
		Log:        logStructuredError,
	}
}
