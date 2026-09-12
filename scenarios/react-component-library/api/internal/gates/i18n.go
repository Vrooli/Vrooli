package gates

import (
	"fmt"
	"strings"
)

func ValidateI18n(scope Scope) (Result, error) {
	return validateActiveSourcesWithPath(scope, "i18n", func(asset assetDoc, path, source string) defect {
		// Story copy describes the specimen, not the adopted runtime surface.
		// It is validated by story grammar and remains intentionally readable in
		// the catalog; only released implementation files require host locale
		// indirection.
		if isStorySource(path) || isTestSource(path) {
			return ok()
		}
		if legacyI18nBridge.MatchString(source) {
			return defect{
				Message:     "library source still uses the removed locale bridge or legacy translate call",
				Remediation: "Declare a named key in a co-located .strings.ts module and read it through useStrings.",
				DocsRef:     "docs/concepts/ARCHITECTURE.md#internationalization",
			}
		}
		if positionalStringKey.MatchString(source) {
			return defect{
				Message:     "internationalization key is positional rather than semantic",
				Remediation: "Rename the key to describe the meaning of the copy and keep the English default in the strings declaration.",
				DocsRef:     "docs/concepts/ARCHITECTURE.md#internationalization",
			}
		}
		for _, match := range i18nAttributeLiteral.FindAllStringSubmatch(source, -1) {
			return defect{
				Message:     fmt.Sprintf("user-facing %s literal %q is embedded in the library source", match[1], match[2]),
				Remediation: fmt.Sprintf("Replace the literal with the host locale bridge for key %s.%s. The English fallback belongs in the locale catalog, not in the adopted component source.", asset.Asset.ID, strings.ToLower(match[1])),
				DocsRef:     "docs/concepts/ARCHITECTURE.md#internationalization",
			}
		}
		if match := jsxTextLiteral.FindStringSubmatch(source); len(match) > 0 && strings.TrimSpace(match[1]) != "" {
			return defect{
				Message:     fmt.Sprintf("user-facing JSX text %q is embedded in the library source", strings.TrimSpace(match[1])),
				Remediation: fmt.Sprintf("Render a translated value from the host locale bridge using a key under %s.", asset.Asset.ID),
				DocsRef:     "docs/concepts/ARCHITECTURE.md#internationalization",
			}
		}
		return defect{}
	})
}

// ValidateSelectorCoverage requires every native interactive element to carry
// a stable test id rooted at the catalog asset identity. Non-interactive
// exports (hooks, stores, tokens, and styling helpers) do not render a root,
// so they are measured without inventing a selector contract for them. This
// keeps BAS flows portable after a renderable asset is copied into an adopting
// scenario.
