package main

import (
	"net/http"

	contenthttp "landing-page-business-suite-api/handlers/content"
)

func contentDependencies(store *ConfigStore) contenthttp.Dependencies {
	return contenthttp.Dependencies{
		PublicSections: func(slug string) (any, error) {
			variant, err := store.GetVariant(slug)
			if err != nil {
				return nil, err
			}
			sections := make([]VariantSection, 0)
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

func handleGetPublicSectionsFromConfigStore(store *ConfigStore) http.HandlerFunc {
	return contenthttp.Public(contentDependencies(store))
}
func handleGetSectionsFromConfigStore(store *ConfigStore) http.HandlerFunc {
	return contenthttp.Admin(contentDependencies(store))
}
