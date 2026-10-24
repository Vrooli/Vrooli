package gates

import (
	"fmt"
)

func ValidateRestyleContract(scope Scope) (Result, error) {
	return validateActiveSources(scope, "restyle-contract", func(asset assetDoc, source string) defect {
		finding := analyzeRestyleSource(source)
		if finding.Message == "" {
			return ok()
		}
		finding.Message = fmt.Sprintf("%s: %s", asset.Asset.ID, finding.Message)
		return finding
	})
}

// ValidateManifestIdentity keeps the source-manifest join explicit. Legacy
// domain catalog ids remain valid because they are the public catalog asset
// identity; library-prefixed ids are the identity form used by assets that do
// not yet have a domain projection and must equal libraryId exactly.
