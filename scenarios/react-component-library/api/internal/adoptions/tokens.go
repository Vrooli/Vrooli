package adoptions

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// TokenTranslation records a semantic design-token rewrite that happened at
// adoption time. It is intentionally stable text: provenance and drift tools
// can explain why an adopted copy differs from its catalog source without
// treating the translation as a local edit.
type TokenTranslation struct {
	From string
	To   string
}

var designTokenClassRE = regexp.MustCompile(`(?:bg|text|border|ring|outline|divide)-app-[a-z0-9-]+(?:/[0-9]+)?`)

var consumerTokenMaps = map[string]map[string]string{
	"app": {
		"app-background": "app-background", "app-border": "app-border", "app-danger": "app-danger", "app-foreground": "app-foreground",
		"app-info": "app-info", "app-muted-foreground": "app-muted-foreground", "app-primary": "app-primary", "app-primary-foreground": "app-primary-foreground",
		"app-surface": "app-surface", "app-surface-muted": "app-surface-muted", "app-warning": "app-warning", "app-success": "app-success",
	},
	"wc": {
		"app-background": "wc-surface-base", "app-border": "wc-default", "app-danger": "wc-error-surface", "app-foreground": "wc-text-primary",
		"app-info": "wc-accent", "app-muted-foreground": "wc-text-muted", "app-primary": "wc-accent", "app-primary-foreground": "wc-accent-fg",
		"app-surface": "wc-surface-raised", "app-surface-muted": "wc-surface-input", "app-warning": "wc-accent", "app-success": "wc-accent",
	},
	"slate": {
		"app-background": "slate-950", "app-border": "slate-700", "app-danger": "slate-700", "app-foreground": "slate-50",
		"app-info": "slate-400", "app-muted-foreground": "slate-400", "app-primary": "slate-400", "app-primary-foreground": "slate-950",
		"app-surface": "slate-900", "app-surface-muted": "slate-800", "app-warning": "slate-400", "app-success": "slate-400",
	},
}

// TranslateDesignTokens rewrites semantic app-* utilities into the target
// consumer vocabulary. Unknown target namespaces fail closed; silently
// emitting a class Tailwind cannot generate recreates the original defect.
func TranslateDesignTokens(body, targetNamespace string) (string, []TokenTranslation, error) {
	ns := strings.TrimSpace(targetNamespace)
	if ns == "" {
		ns = "app"
	}
	mapping, ok := consumerTokenMaps[ns]
	if !ok {
		return "", nil, fmt.Errorf("adoption token namespace %q is not governed", ns)
	}
	translations := map[string]TokenTranslation{}
	out := designTokenClassRE.ReplaceAllStringFunc(body, func(class string) string {
		parts := strings.SplitN(class, "-app-", 2)
		if len(parts) != 2 {
			return class
		}
		base := "app-" + strings.Split(parts[1], "/")[0]
		mapped, exists := mapping[base]
		if !exists {
			return class
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
			}
		}
		mappedClass := parts[0] + "-" + mapped
		if slash := strings.IndexByte(class, '/'); slash >= 0 {
			mappedClass += class[slash:]
		}
		translations[class] = TokenTranslation{From: class, To: mappedClass}
		return mappedClass
	})
	result := make([]TokenTranslation, 0, len(translations))
	for _, translation := range translations {
		result = append(result, translation)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].From < result[j].From })
	return out, result, nil
}

// CrossBoundaryImport reports counterfeit adoptions: a provenance-tagged
// file must contain its copied source and cannot re-export from the library's
// source tree across a scenario boundary.
func CrossBoundaryImport(body string) bool {
	return strings.Contains(body, "@vrooliComponentSource") &&
		(strings.Contains(body, "react-component-library/library") || strings.Contains(body, "react-component-library\\library"))
}
