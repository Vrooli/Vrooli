package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func ValidateAPI(scope Scope) (Result, error) {
	root := scope.Root
	assets, err := loadAssets(scope)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	for _, asset := range assets {
		if asset.API == nil {
			continue
		}
		manifest, source, ok, err := implementationSource(root, asset.Asset.ID)
		if err != nil {
			return Result{}, err
		}
		if !ok {
			continue
		}
		data, err := os.ReadFile(source)
		if err != nil {
			return Result{}, err
		}
		result.Inspected++
		text := string(data)
		apiRemediation := func(kind, value string) string {
			return fmt.Sprintf("Either implement %s %q in %s, or remove it from the asset's api block in catalog/assets/. The catalog entry is the contract adopting scenarios read before they call this component; a declared %s that the source does not handle is a promise the library cannot keep, and it fails at the consumer rather than here.", kind, value, repoRel(root, source), kind)
		}
		for group, values := range asset.API.Variants {
			for _, value := range values {
				if !strings.Contains(text, value) {
					result.Findings = append(result.Findings, Finding{
						Code: "catalog.api_mismatch", AssetID: asset.Asset.ID, File: repoRel(root, manifest),
						Message:     fmt.Sprintf("declared %s variant %q is absent from the implementation", group, value),
						Remediation: apiRemediation("variant", value),
						DocsRef:     "docs/concepts/ARCHITECTURE.md#catalog-graph-projection",
					})
				}
			}
		}
		for _, value := range asset.API.Modes {
			if !strings.Contains(text, value) {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.api_mismatch", AssetID: asset.Asset.ID, File: repoRel(root, manifest),
					Message:     fmt.Sprintf("declared mode %q is absent from the implementation", value),
					Remediation: apiRemediation("mode", value),
					DocsRef:     "docs/concepts/ARCHITECTURE.md#catalog-graph-projection",
				})
			}
		}
		for _, rawPart := range asset.API.Parts {
			partID := ""
			if json.Unmarshal(rawPart, &partID) != nil {
				var part struct {
					ID string `json:"id"`
				}
				_ = json.Unmarshal(rawPart, &part)
				partID = part.ID
			}
			if partID != "" && !strings.Contains(text, partID) {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.api_mismatch", AssetID: asset.Asset.ID, File: repoRel(root, manifest),
					Message:     fmt.Sprintf("declared part %q is absent from the implementation", partID),
					Remediation: apiRemediation("part", partID),
					DocsRef:     "docs/concepts/ARCHITECTURE.md#catalog-graph-projection",
				})
			}
		}
	}
	return nonEmpty(result, "api"), nil
}

var (
	i18nAttributeLiteral    = regexp.MustCompile(`(?m)\b(aria-label|placeholder|title|alt|label|description)\s*=\s*["']([^"'\r\n<>]{1,160})["']`)
	jsxTextLiteral          = regexp.MustCompile(`>\s*([[:alpha:]][^<>{}\n]{1,160})\s*</[A-Za-z]`)
	interactiveElementStart = regexp.MustCompile(`<((?:button|a|input|select|textarea))\b`)
	legacyI18nBridge        = regexp.MustCompile(`__vrooliTranslate|library-locale-bridge|\btranslate\(\s*["']`)
	positionalStringKey     = regexp.MustCompile(`(?:useStrings|resolveStrings|defineStrings)\(\s*["'][^"']*\.[0-9]+["']|["'][^"']*\.[0-9]+["']\s*:`)
)

// ValidateI18n derives user-facing strings from component source. Literal
// labels are not a stable adoption contract: the host must supply their
// translation through the shared locale bridge.
