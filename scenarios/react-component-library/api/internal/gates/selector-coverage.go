package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func ValidateSelectorCoverage(scope Scope) (Result, error) {
	root := scope.Root
	factsIndex, indexErr := readSourceFactsIndex(root, scope)
	if indexErr != nil && !os.IsNotExist(indexErr) {
		return Result{}, indexErr
	}
	return validateActiveSourcesWithPath(scope, "selector-coverage", func(asset assetDoc, path, source string) defect {
		if !selectorCoverageRenderableKind(asset.Asset.Kind) {
			return ok()
		}
		factErr := indexErr
		facts := []sourceFacts{}
		if fact, ok := factsIndex[filepath.Clean(path)]; ok {
			facts = []sourceFacts{fact}
			factErr = nil
		}
		if factErr != nil && !os.IsNotExist(factErr) {
			return defect{Message: factErr.Error(), Remediation: "Keep the shared AST facts analyzer available to selector validation.", DocsRef: "docs/concepts/ARCHITECTURE.md#automation-selectors"}
		}
		for _, fact := range facts {
			for _, element := range fact.Elements {
				if element.Tag != "button" && element.Tag != "a" && element.Tag != "input" && element.Tag != "select" && element.Tag != "textarea" {
					continue
				}
				testIDs := element.Attributes["data-testid"]
				if len(testIDs) > 0 && strings.Contains(strings.Join(testIDs, " "), asset.Asset.ID) {
					continue
				}
				return defect{
					Message:     fmt.Sprintf("interactive <%s> has no data-testid derived from %s", element.Tag, asset.Asset.ID),
					Remediation: fmt.Sprintf("Add data-testid=%q or a derived selector rooted at %s to the interactive element.", asset.Asset.ID, asset.Asset.ID),
					DocsRef:     "docs/concepts/ARCHITECTURE.md#automation-selectors",
				}
			}
		}
		if factErr == nil {
			return defect{}
		}
		for _, tag := range interactiveElements(source) {
			match := interactiveElementStart.FindStringSubmatch(tag)
			if len(match) == 0 {
				continue
			}
			testID := regexp.MustCompile(`data-testid\s*=`).FindStringIndex(tag)
			if testID == nil || !strings.Contains(tag, asset.Asset.ID) {
				return defect{
					Message:     fmt.Sprintf("interactive <%s> has no data-testid derived from %s", match[1], asset.Asset.ID),
					Remediation: fmt.Sprintf("Add data-testid=%q or a derived selector rooted at %s to the interactive element.", asset.Asset.ID, asset.Asset.ID),
					DocsRef:     "docs/concepts/ARCHITECTURE.md#automation-selectors",
				}
			}
		}
		return defect{}
	})
}

func selectorCoverageRenderableKind(kind string) bool {
	switch kind {
	case "", "component", "primitive", "pattern", "page-template", "navigation":
		return true
	default:
		return false
	}
}

// ValidateRestyleContract checks the public seam used by linked consumers.
// Native HTML attribute inheritance is accepted as an explicit className
// contract when the component forwards its remaining props to its root
// element. Components with bespoke props must name className and use it in
// rendered markup so a consumer never has to copy the implementation merely
// to change presentation.
