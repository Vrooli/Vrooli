package adoptions

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"react-component-library/internal/components"
)

var scenarioTokenDeclarationRE = regexp.MustCompile(`(--[A-Za-z0-9_-]+)\s*:`)
var scenarioRuntimeTokenWriteRE = regexp.MustCompile(`\.setProperty\s*\(\s*["'](--[A-Za-z0-9_-]+)["']`)

// TokenVerdict is the read-only styling-contract result for an adoption
// closure. Required is the exact derived set; RequiredPatterns represents
// runtime-selected families; Unsatisfied is the actionable remainder.
type TokenVerdict struct {
	Required         []string
	RequiredPatterns []string
	Defined          []string
	Unsatisfied      []string
	RequiredBy       map[string][]string
}

func (v TokenVerdict) Satisfied() bool { return len(v.Unsatisfied) == 0 }

// ErrAdoptionTokensUnsatisfied is returned before any adoption write when the
// target scenario cannot satisfy the closure's derived styling contract.
type ErrAdoptionTokensUnsatisfied struct {
	ComponentID string
	Scenario    string
	Properties  []string
}

func (e ErrAdoptionTokensUnsatisfied) Error() string {
	return fmt.Sprintf("adoption %s into %s requires unsatisfied CSS custom properties %s; run `react-component-library adoptions tokens-sync %s` or pass --override-validation",
		e.ComponentID, e.Scenario, strings.Join(e.Properties, ", "), e.Scenario)
}

type ScenarioTokenInventoryReader interface {
	DeclaredTokens(ctx context.Context, scenario string) ([]string, error)
}

type ScenarioRuntimeTokenInventoryReader interface {
	RuntimeWrittenTokens(ctx context.Context, scenario string) ([]string, error)
}

func (s *service) resolveTokenVerdict(ctx context.Context, closure components.ClosureReport, scenario string) (TokenVerdict, error) {
	if s.tokenInventory == nil {
		return TokenVerdict{}, nil
	}
	declared, err := s.tokenInventory.DeclaredTokens(ctx, scenario)
	if err != nil {
		return TokenVerdict{}, err
	}
	defined := make(map[string]struct{}, len(declared))
	for _, property := range declared {
		defined[property] = struct{}{}
	}
	required := make(map[string]struct{})
	patterns := make(map[string]struct{})
	by := make(map[string][]string)
	for _, asset := range closure.Assets {
		for _, property := range asset.Version.RequiredTokens {
			required[property] = struct{}{}
			by[property] = appendUnique(by[property], asset.Asset.LibraryID)
		}
		for _, pattern := range asset.Version.RequiredTokenPatterns {
			patterns[pattern] = struct{}{}
			by[pattern] = appendUnique(by[pattern], asset.Asset.LibraryID)
		}
	}
	unsatisfied := make(map[string]struct{})
	for property := range required {
		if _, ok := defined[property]; !ok {
			unsatisfied[property] = struct{}{}
		}
	}
	for pattern := range patterns {
		prefix := strings.TrimSuffix(pattern, "*")
		matched := false
		for property := range defined {
			if strings.HasPrefix(property, prefix) {
				matched = true
				break
			}
		}
		if !matched {
			unsatisfied[pattern] = struct{}{}
		}
	}
	result := TokenVerdict{
		Required:         sortedKeys(required),
		RequiredPatterns: sortedKeys(patterns),
		Defined:          append([]string(nil), declared...),
		Unsatisfied:      sortedKeys(unsatisfied),
		RequiredBy:       by,
	}
	return result, nil
}

func (s *service) requireTokenVerdict(ctx context.Context, componentID, scenario string, closure components.ClosureReport, override bool) (TokenVerdict, error) {
	verdict, err := s.resolveTokenVerdict(ctx, closure, scenario)
	if err != nil {
		return TokenVerdict{}, err
	}
	if !verdict.Satisfied() && !override {
		return verdict, ErrAdoptionTokensUnsatisfied{ComponentID: componentID, Scenario: scenario, Properties: verdict.Unsatisfied}
	}
	return verdict, nil
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func appendUnique(values []string, value string) []string {
	for _, current := range values {
		if current == value {
			return values
		}
	}
	return append(values, value)
}
