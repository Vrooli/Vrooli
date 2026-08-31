package gates

import (
	"os"
)

func ValidateTokenVocabulary(scope Scope) (Result, error) {
	root := scope.Root
	sources, err := activeLibrarySources(scope)
	if err != nil {
		return Result{}, err
	}
	result := Result{Inspected: len(sources)}
	for _, path := range sources {
		raw, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		if offset := firstRetiredTokenReference(raw); offset >= 0 {
			result.Findings = append(result.Findings, Finding{
				Code:        "catalog.token_vocabulary",
				AssetID:     implementationName(path),
				File:        repoRel(root, path),
				Line:        lineAt(raw, offset),
				Message:     "references the retired --app-* CSS custom property vocabulary",
				Remediation: "Replace each --app-<name> reference with its --color-<name> / --space-<name> equivalent from the canonical ramp at ui/src/design-tokens.css. The --app-* family is defined only for the workspace application shell and is not published to library consumers, so a library asset referencing it renders unstyled in every adopting scenario.",
				DocsRef:     "docs/concepts/ARCHITECTURE.md#design-tokens",
			})
		}
	}
	return nonEmpty(result, "token-vocabulary"), nil
}

// firstRetiredTokenReference returns the first --app-* occurrence that is not
// the name of a custom-property declaration. BaseStyles intentionally keeps
// declaration-only aliases at the compatibility boundary; consumers may not
// reference that retired vocabulary from active component source.
