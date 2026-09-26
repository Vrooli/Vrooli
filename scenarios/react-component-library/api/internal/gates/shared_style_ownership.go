package gates

import (
	"strings"
)

func ValidateSharedStyleOwnership(scope Scope) (Result, error) {
	return validateActiveSourceFiles(scope, "shared-style-ownership", func(asset assetDoc, source string) defect {
		if strings.HasSuffix(asset.Asset.ID, ":BaseStyles") || strings.HasSuffix(asset.Asset.ID, ".base-styles") {
			return ok()
		}
		switch {
		case sharedMotionRE.MatchString(source):
			return defect{Message: "declares a local prefers-reduced-motion rule", Remediation: "Use the shared BaseStyles foundation; an asset must not redefine the library-wide motion policy.", DocsRef: "docs/reference/style-ownership.md"}
		case sharedForcedColorsRE.MatchString(source):
			return defect{Message: "declares a local forced-colors rule", Remediation: "Use the shared BaseStyles foundation; an asset must not redefine the library-wide forced-colors policy.", DocsRef: "docs/reference/style-ownership.md"}
		case sharedFocusRE.MatchString(source):
			return defect{Message: "declares a local focus-visible rule", Remediation: "Use the shared BaseStyles focus ring; keep asset-specific focus styling limited to named states.", DocsRef: "docs/reference/style-ownership.md"}
		case sharedVisuallyHiddenRE.MatchString(source):
			return defect{Message: "declares a local visually-hidden rule", Remediation: "Use the shared BaseStyles visually-hidden utility.", DocsRef: "docs/reference/style-ownership.md"}
		default:
			return ok()
		}
	})
}

// ValidateStyleInjection rejects style elements emitted from component output.
// The only supported runtime path is useLibraryStyleSheet, whose document-head
// ownership is independently tested by the foundation package.
