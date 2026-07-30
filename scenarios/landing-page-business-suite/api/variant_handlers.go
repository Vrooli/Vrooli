// DOC: docs/reference/api/variants.md - A/B testing variant endpoints
// DOC: docs/concepts/CONCEPTS.md#ab-testing-system - A/B testing architecture
// DOC: docs/concepts/CONCEPTS.md#variant-selection-flow - Variant selection algorithm
package main

import (
	varianthttp "landing-page-business-suite-api/handlers/experimentation"
	"landing-page-business-suite-api/internal/experimentation"
)

func variantReadDependencies(cs experimentation.ConfigStoreReader, pathPrefix string) varianthttp.Dependencies {
	return varianthttp.NewReadDependencies(cs, pathPrefix, writeJSON, writeJSONError, logStructuredError)
}
