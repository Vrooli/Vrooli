package adoptions

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"react-component-library/internal/utilityclass"
)

// TokenTranslation records a semantic design-token rewrite that happened at
// adoption time. It is intentionally stable text: provenance and drift tools
// can explain why an adopted copy differs from its catalog source without
// treating the translation as a local edit.
type TokenTranslation struct {
	From string
	To   string
}

// TokenRoleMapping is the scenario-owned destination for one catalog role.
// CSSVariable makes the runtime-theme dependency explicit instead of allowing
// an adopted asset to point at an opaque or hard-coded color.
type TokenRoleMapping struct {
	Target      string `json:"target"`
	CSSVariable string `json:"css_variable"`
}

type TokenContrastPair struct {
	Foreground string  `json:"foreground"`
	Background string  `json:"background"`
	Ratio      float64 `json:"ratio"`
}

// TokenMapping is read from the adopting scenario's UI tree. The component
// library deliberately does not own consumer palette decisions.
type TokenMapping struct {
	Namespace     string                      `json:"namespace"`
	Roles         map[string]TokenRoleMapping `json:"roles"`
	ContrastFloor float64                     `json:"contrast_floor"`
	ContrastPairs []TokenContrastPair         `json:"contrast_pairs"`
}

var (
	tokenRoleRE   = regexp.MustCompile(`^app-[a-z0-9-]+$`)
	cssVariableRE = regexp.MustCompile(`^--[a-z0-9-]+$`)
)

// ValidateTokenMappingInjective checks the semantic roles emitted by one
// adopted asset against the mapping supplied by its consumer scenario.
func ValidateTokenMappingInjective(mapping TokenMapping, roles []string) error {
	return validateTokenMapping(mapping, roles)
}

func validateTokenMapping(mapping TokenMapping, roles []string) error {
	ns := strings.TrimSpace(mapping.Namespace)
	if ns == "" {
		return fmt.Errorf("adoption token mapping is missing namespace")
	}
	switch ns {
	case "app", "wc", "slate":
	default:
		return fmt.Errorf("adoption token namespace %q is not governed", ns)
	}
	if len(mapping.Roles) == 0 && ns == "app" {
		// Narrow test seams can use the catalog namespace without a filesystem
		// reader. Production always supplies a scenario-owned mapping.
		mapping.Roles = make(map[string]TokenRoleMapping, len(roles))
		for _, role := range roles {
			mapping.Roles[role] = TokenRoleMapping{Target: role, CSSVariable: "--" + strings.ReplaceAll(role, "-", "-")}
		}
	}
	resolved := make(map[string]string, len(mapping.Roles))
	for role, entry := range mapping.Roles {
		if !tokenRoleRE.MatchString(role) || strings.TrimSpace(entry.Target) == "" || !cssVariableRE.MatchString(entry.CSSVariable) {
			return fmt.Errorf("adoption token mapping entry %q is incomplete or not CSS-variable-backed", role)
		}
		resolved[role] = entry.Target
	}
	if err := validateTokenMappingInjectiveString(ns, resolved, roles); err != nil {
		return err
	}
	floor := mapping.ContrastFloor
	if floor <= 0 {
		floor = 4.5
	}
	for _, pair := range mapping.ContrastPairs {
		if pair.Ratio < floor {
			return fmt.Errorf("adoption token contrast %s over %s is %.2f below floor %.2f", pair.Foreground, pair.Background, pair.Ratio, floor)
		}
	}
	return nil
}

func validateTokenMappingInjective(namespace string, mapping map[string]string, roles []string) error {
	return validateTokenMappingInjectiveString(namespace, mapping, roles)
}

func validateTokenMappingInjectiveString(namespace string, mapping map[string]string, roles []string) error {
	seen := map[string]string{}
	for _, role := range roles {
		if _, exists := mapping[role]; !exists {
			return fmt.Errorf("adoption token role %q is not governed for namespace %q", role, namespace)
		}
		if previous, exists := seen[mapping[role]]; exists && previous != role {
			return fmt.Errorf("adoption token collision in %q: %s and %s both map to %s", namespace, previous, role, mapping[role])
		}
		seen[mapping[role]] = role
	}
	return nil
}

// TranslateDesignTokens rewrites semantic app-* utilities into the target
// consumer vocabulary. Unknown target namespaces fail closed; silently
// emitting a class Tailwind cannot generate recreates the original defect.
func TranslateDesignTokens(body, targetNamespace string, supplied ...TokenMapping) (string, []TokenTranslation, error) {
	ns := strings.TrimSpace(targetNamespace)
	if ns == "" {
		ns = "app"
	}
	mapping := TokenMapping{Namespace: ns}
	if len(supplied) > 0 {
		mapping = supplied[0]
	}
	roles := map[string]struct{}{}
	classes := semanticAppClasses(body)
	for _, class := range classes {
		parts := strings.SplitN(class, "-app-", 2)
		if len(parts) == 2 {
			roles["app-"+strings.Split(parts[1], "/")[0]] = struct{}{}
		}
	}
	roleList := make([]string, 0, len(roles))
	for role := range roles {
		roleList = append(roleList, role)
	}
	sort.Strings(roleList)
	if err := validateTokenMapping(mapping, roleList); err != nil {
		return "", nil, err
	}
	resolved := make(map[string]string, len(mapping.Roles))
	for role, entry := range mapping.Roles {
		resolved[role] = entry.Target
	}
	if len(resolved) == 0 && ns == "app" {
		for _, role := range roleList {
			resolved[role] = role
		}
	}
	translations := map[string]TokenTranslation{}
	out := body
	for _, class := range classes {
		parts := strings.SplitN(class, "-app-", 2)
		if len(parts) != 2 {
			continue
		}
		base := "app-" + strings.Split(parts[1], "/")[0]
		mapped, exists := resolved[base]
		if !exists {
			continue
		}
		if ns == "wc" {
			prefix := parts[0]
			switch base {
			case "app-danger":
				switch prefix {
				case "border":
					mapped = "wc-error"
				case "text":
					mapped = "wc-error-text"
				}
			case "app-background":
				if prefix != "bg" {
					mapped = "wc-text-primary"
				}
			case "app-surface", "app-surface-muted":
				if prefix != "bg" {
					mapped = "wc-text-primary"
				}
			case "app-primary", "app-warning":
				// wc-accent-active is a background token and
				// wc-accent-border is a pre-composed border token. The
				// warning role remains on the readable accent vocabulary
				// for utility contexts that cannot consume a border token.
				if base == "app-warning" || prefix != "bg" {
					mapped = "wc-accent"
				}
			}
		}
		mappedClass := parts[0] + "-" + mapped
		if slash := strings.IndexByte(class, '/'); slash >= 0 {
			mappedClass += class[slash:]
		}
		translations[class] = TokenTranslation{From: class, To: mappedClass}
		out = strings.ReplaceAll(out, class, mappedClass)
	}
	result := make([]TokenTranslation, 0, len(translations))
	for _, translation := range translations {
		result = append(result, translation)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].From < result[j].From })
	return out, result, nil
}

func semanticAppClasses(source string) []string {
	seen := map[string]bool{}
	var classes []string
	for _, hit := range utilityclass.EmitsAny(source) {
		if !strings.Contains(hit.Class, "-app-") || seen[hit.Class] {
			continue
		}
		seen[hit.Class] = true
		classes = append(classes, hit.Class)
	}
	sort.Strings(classes)
	return classes
}

// CrossBoundaryImport reports counterfeit adoptions: a provenance-tagged
// file must contain its copied source and cannot re-export from the library's
// source tree across a scenario boundary.
func CrossBoundaryImport(body string) bool {
	return strings.Contains(body, "@vrooliComponentSource") &&
		(strings.Contains(body, "react-component-library/library") || strings.Contains(body, "react-component-library\\library"))
}
