package verify

import (
	"fmt"
	"sort"
	"strings"

	"flow-verifier/internal/flows/kind"
	"flow-verifier/internal/flows/kinds/navigation/compile"
	"flow-verifier/internal/flows/kinds/navigation/contract"
	"flow-verifier/internal/flows/kinds/navigation/predicate"
)

// CheckDeepLinkPolicy asserts that every route whose `requires`
// references a gated context has a matching `deep_link_policy` entry
// (either `for_routes` contains the route id, or `for_routes_where`
// evaluates true on the route).
//
// A context is "gated" when at least one deep_link_policy rule mentions
// it. This keeps the check declarative: if the user has chosen to
// declare a deep-link policy for `auth=logged_in`, every auth-required
// route must be covered.
func CheckDeepLinkPolicy(g compile.Graph) ([]kind.Finding, error) {
	if len(g.Contract.DeepLinkPolicy) == 0 {
		return nil, nil
	}

	// Derive the set of gated context keys from for_routes_where predicates.
	gated := map[string]bool{}
	type rulePred struct {
		rule contract.DeepLinkRule
		pred predicate.Predicate
		ids  map[string]bool
	}
	rules := make([]rulePred, 0, len(g.Contract.DeepLinkPolicy))
	for _, r := range g.Contract.DeepLinkPolicy {
		rp := rulePred{rule: r}
		rp.ids = map[string]bool{}
		for _, id := range r.ForRoutes {
			rp.ids[id] = true
		}
		if r.ForRoutesWhere != "" {
			p, err := predicate.Parse(r.ForRoutesWhere)
			if err != nil {
				return nil, fmt.Errorf("deep_link_policy %q for_routes_where: %w", r.ID, err)
			}
			rp.pred = p
			for k := range gatedKeysFrom(r.ForRoutesWhere) {
				gated[k] = true
			}
		}
		rules = append(rules, rp)
	}

	var findings []kind.Finding
	for _, route := range g.Contract.Routes {
		if route.Requires == "" {
			continue
		}
		needs := requiresMentionsGated(route.Requires, gated)
		if !needs {
			continue
		}
		covered := false
		for _, rp := range rules {
			if rp.ids[route.ID] {
				covered = true
				break
			}
			if rp.rule.ForRoutesWhere != "" {
				ok, err := rp.pred.Eval(func(name string) (string, bool) {
					if name == "requires" {
						return route.Requires, true
					}
					return "", false
				})
				if err != nil {
					return nil, fmt.Errorf("deep_link_policy %q on route %q: %w", rp.rule.ID, route.ID, err)
				}
				if ok {
					covered = true
					break
				}
			}
		}
		findings = append(findings, kind.Finding{
			ID:      "deep_link_route_" + route.ID,
			Passed:  covered,
			Message: deepLinkMessage(route, covered),
		})
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].ID < findings[j].ID })
	return findings, nil
}

func deepLinkMessage(r contract.Route, covered bool) string {
	if covered {
		return fmt.Sprintf("route %q deep-link policy covered", r.ID)
	}
	return fmt.Sprintf("route %q (requires %q) has no deep_link_policy entry", r.ID, r.Requires)
}

// gatedKeysFrom extracts the bareword identifiers from a CONTAINS rhs
// like `auth=logged_in` — i.e. the context names a policy is keyed on.
// Tokens before `=` are taken as the gated key.
func gatedKeysFrom(forRoutesWhere string) map[string]bool {
	out := map[string]bool{}
	parts := strings.Fields(forRoutesWhere)
	for i := 0; i < len(parts); i++ {
		if parts[i] == "CONTAINS" && i+1 < len(parts) {
			eq := strings.SplitN(parts[i+1], "=", 2)
			if len(eq) == 2 {
				out[eq[0]] = true
			}
		}
	}
	return out
}

// requiresMentionsGated returns true if the route's requires string
// mentions any gated context name.
func requiresMentionsGated(requires string, gated map[string]bool) bool {
	for k := range gated {
		if strings.Contains(requires, k+"=") || strings.Contains(requires, k+"!=") {
			return true
		}
	}
	return false
}
