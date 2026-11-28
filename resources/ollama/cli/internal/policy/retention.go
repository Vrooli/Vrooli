package policy

import (
	"fmt"
	"sort"
	"strings"
)

// Retention separates the models disk eviction must keep from the ones it may
// evict when idle. A role's primary model and every pinned model are
// required. Fallbacks and models no role names are eviction-eligible; the
// fenced owner operation (`models evict`) and storage-manager's idle window
// decide whether one actually goes.
type Retention struct {
	// Pinned keeps models that are not a role primary (for example a fallback
	// an operator wants local regardless of use).
	Pinned    []string `json:"pinned"`
	Rationale string   `json:"rationale,omitempty"`
}

// RequiredModels maps every model disk eviction must keep to why.
func (p Policy) RequiredModels() map[string]string {
	out := map[string]string{}
	roles := make([]string, 0, len(p.Roles))
	for name := range p.Roles {
		roles = append(roles, name)
	}
	sort.Strings(roles)
	for _, name := range roles {
		model := strings.TrimSpace(p.Roles[name].Model)
		if model == "" {
			continue
		}
		if prior, ok := out[model]; ok {
			out[model] = prior + ", " + name
		} else {
			out[model] = "primary model of role " + name
		}
	}
	for _, model := range p.Retention.Pinned {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if prior, ok := out[model]; ok {
			out[model] = prior + "; pinned by retention.pinned"
		} else {
			out[model] = "pinned by retention.pinned"
		}
	}
	return out
}

func (r Retention) validate(p Policy) []error {
	var errs []error
	for _, model := range r.Pinned {
		if _, ok := p.Models[strings.TrimSpace(model)]; !ok {
			errs = append(errs, fmt.Errorf("retention.pinned model %q is not in models", model))
		}
	}
	return errs
}
