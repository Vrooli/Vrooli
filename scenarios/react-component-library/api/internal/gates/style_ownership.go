package gates

import "regexp"

var cssImportRE = regexp.MustCompile(`(?m)\bimport\s+(?:[^"']+\s+from\s+)?["']([^"']+\.css)["']`)

// ValidateStyleOwnership enforces the source-side styling boundary. CSS is an
// authored companion file, so local side-effect imports are accepted as the
// authoring seam; catalog build replaces them with the shared registration
// runtime in the disposable package artifact. External stylesheet imports are
// rejected because they cannot travel with an adopted asset.
func ValidateStyleOwnership(scope Scope) (Result, error) {
	return validateActiveSourceFiles(scope, "style-ownership", func(_ assetDoc, source string) defect {
		for _, match := range cssImportRE.FindAllStringSubmatch(source, -1) {
			if len(match) > 1 && match[1] != "" && match[1][0] != '.' {
				return defect{Message: "runtime source imports an external CSS file", Remediation: "Keep the stylesheet beside the authored version and let catalog build inline it through the shared stylesheet runtime.", DocsRef: "docs/reference/style-ownership.md"}
			}
		}
		return ok()
	})
}
