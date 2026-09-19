package gates

import (
	"strings"
)

func ValidateIntegration(scope Scope) (Result, error) {
	return validateActiveSources(scope, "integration", func(asset assetDoc, source string) defect {
		if strings.TrimSpace(source) == "" {
			return defect{
				Message:     "released integration source is empty",
				Remediation: "Implement this asset, or move it back to draft. An empty released source passes every static gate by having nothing to inspect, which is how an asset reaches production-ready while rendering nothing.",
				DocsRef:     "docs/internal/SEAMS.md",
			}
		}
		// The active component manifest supplies the exact released version;
		// source identity may use the library marker or the established
		// adoption-facade marker. Both are valid integration boundaries, while
		// an unowned source is not.
		if !strings.Contains(source, "@libraryId") && !strings.Contains(source, "@vrooliComponentSource") {
			return defect{
				Message:     "released source carries neither @libraryId nor @vrooliComponentSource identity metadata",
				Remediation: "Add an @libraryId docblock tag naming this asset's library id (or @vrooliComponentSource if the file is an adoption facade). Identity metadata is how the adoption reconciler tells a library-owned file from a scenario-owned one; without it this file is invisible to drift detection and will silently diverge from the version it was copied from.",
				DocsRef:     "docs/internal/SEAMS.md",
			}
		}
		return ok()
	})
}

// ValidateSelfHosting measures whether the catalog application exercises the
// published library surface. This is a corpus observation: the catalog app
// is the consumer and the library asset set is the denominator.
