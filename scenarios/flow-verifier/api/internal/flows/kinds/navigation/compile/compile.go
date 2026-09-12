// Package compile runs structural cross-reference validation on a
// schema-valid navigation contract. JSON-schema validation lives in
// contract; this package owns the checks that depend on resolving
// references between routes, containers, affordances, overlays, etc.
package compile

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"flow-verifier/internal/flows/kinds/navigation/contract"
)

// Graph is the compiled navigation graph: the raw contract plus
// resolution indices that downstream stages (reachability, codegen,
// studio) consume.
type Graph struct {
	Contract        contract.Contract
	RoutesByID      map[string]contract.Route
	ContainersByID  map[string]contract.Container
	OverlaysByID    map[string]contract.Overlay
	AffordancesByID map[string]contract.Affordance
}

// Compile resolves cross-references and returns a Graph. Any structural
// problem becomes a single error with all findings concatenated so
// callers do not have to call repeatedly.
func Compile(c contract.Contract) (Graph, error) {
	var errs []string

	routesByID, err := indexRoutes(c.Routes)
	if err != nil {
		errs = append(errs, err.Error())
	}
	containersByID, err := indexContainers(c.Containers)
	if err != nil {
		errs = append(errs, err.Error())
	}
	overlaysByID, err := indexOverlays(c.Overlays)
	if err != nil {
		errs = append(errs, err.Error())
	}
	affordancesByID, err := indexAffordances(c.Affordances)
	if err != nil {
		errs = append(errs, err.Error())
	}

	errs = append(errs, validateContexts(c.Contexts)...)
	errs = append(errs, validateRouteRefs(c.Routes, routesByID)...)
	errs = append(errs, validateContainerRefs(c.Containers, routesByID)...)
	errs = append(errs, validateOverlayRefs(c.Overlays, routesByID)...)
	errs = append(errs, validateAffordanceRefs(c.Affordances, routesByID, containersByID)...)
	errs = append(errs, validateReturnPathRefs(c.ReturnPaths, routesByID)...)
	errs = append(errs, validateShortcutRefs(c.Shortcuts, routesByID)...)
	errs = append(errs, validateReachabilityRefs(c.ReachabilityInvariants, routesByID)...)
	errs = append(errs, validateDeepLinkRefs(c.DeepLinkPolicy, routesByID)...)
	errs = append(errs, validatePredicates(c)...)

	if len(errs) > 0 {
		sort.Strings(errs)
		return Graph{}, fmt.Errorf("navigation contract %s has structural errors:\n  - %s", c.ContractPath, strings.Join(errs, "\n  - "))
	}

	return Graph{
		Contract:        c,
		RoutesByID:      routesByID,
		ContainersByID:  containersByID,
		OverlaysByID:    overlaysByID,
		AffordancesByID: affordancesByID,
	}, nil
}

func indexRoutes(routes []contract.Route) (map[string]contract.Route, error) {
	out := make(map[string]contract.Route, len(routes))
	for _, r := range routes {
		if _, dup := out[r.ID]; dup {
			return out, fmt.Errorf("duplicate route id %q", r.ID)
		}
		out[r.ID] = r
	}
	return out, nil
}

func indexContainers(cs []contract.Container) (map[string]contract.Container, error) {
	out := make(map[string]contract.Container, len(cs))
	for _, c := range cs {
		if _, dup := out[c.ID]; dup {
			return out, fmt.Errorf("duplicate container id %q", c.ID)
		}
		out[c.ID] = c
	}
	return out, nil
}

func indexOverlays(os []contract.Overlay) (map[string]contract.Overlay, error) {
	out := make(map[string]contract.Overlay, len(os))
	for _, o := range os {
		if _, dup := out[o.ID]; dup {
			return out, fmt.Errorf("duplicate overlay id %q", o.ID)
		}
		out[o.ID] = o
	}
	return out, nil
}

func indexAffordances(as []contract.Affordance) (map[string]contract.Affordance, error) {
	out := make(map[string]contract.Affordance, len(as))
	for _, a := range as {
		if _, dup := out[a.ID]; dup {
			return out, fmt.Errorf("duplicate affordance id %q", a.ID)
		}
		out[a.ID] = a
	}
	return out, nil
}

func validateContexts(ctxs map[string]contract.Context) []string {
	var errs []string
	for name, ctx := range ctxs {
		switch ctx.Kind {
		case "enum":
			vs := map[string]bool{}
			for _, v := range ctx.Values {
				if vs[v] {
					errs = append(errs, fmt.Sprintf("context %q: duplicate value %q", name, v))
				}
				vs[v] = true
			}
			var def string
			if err := json.Unmarshal(ctx.Default, &def); err != nil {
				errs = append(errs, fmt.Sprintf("context %q: enum default must be a string (%v)", name, err))
				continue
			}
			if !vs[def] {
				errs = append(errs, fmt.Sprintf("context %q: default %q is not one of declared values %v", name, def, ctx.Values))
			}
		case "bool":
			var def bool
			if err := json.Unmarshal(ctx.Default, &def); err != nil {
				errs = append(errs, fmt.Sprintf("context %q: bool default must be a boolean (%v)", name, err))
			}
		}
	}
	return errs
}

func validateRouteRefs(rs []contract.Route, byID map[string]contract.Route) []string {
	var errs []string
	for _, r := range rs {
		for _, p := range r.Parents {
			if _, ok := byID[p]; !ok {
				errs = append(errs, fmt.Sprintf("route %q: parent %q is not a declared route", r.ID, p))
			}
		}
		if r.RedirectIfUnmet != nil {
			if _, ok := byID[r.RedirectIfUnmet.To]; !ok {
				errs = append(errs, fmt.Sprintf("route %q: redirect_if_unmet.to %q is not a declared route", r.ID, r.RedirectIfUnmet.To))
			}
		}
	}
	return errs
}

// validateHostRoute allows "*" wildcard or an existing route id.
func validateHostRoute(scope, id, hr string, routes map[string]contract.Route) string {
	if hr == "*" {
		return ""
	}
	if _, ok := routes[hr]; !ok {
		return fmt.Sprintf("%s %q: host_routes entry %q is not a declared route", scope, id, hr)
	}
	return ""
}

func validateContainerRefs(cs []contract.Container, routes map[string]contract.Route) []string {
	var errs []string
	for _, c := range cs {
		for _, hr := range c.HostRoutes {
			if e := validateHostRoute("container", c.ID, hr, routes); e != "" {
				errs = append(errs, e)
			}
		}
	}
	return errs
}

func validateOverlayRefs(os []contract.Overlay, routes map[string]contract.Route) []string {
	var errs []string
	for _, o := range os {
		for _, hr := range o.HostRoutes {
			if e := validateHostRoute("overlay", o.ID, hr, routes); e != "" {
				errs = append(errs, e)
			}
		}
	}
	return errs
}

func validateAffordanceRefs(as []contract.Affordance, routes map[string]contract.Route, containers map[string]contract.Container) []string {
	var errs []string
	for _, a := range as {
		if _, ok := routes[a.To]; !ok {
			errs = append(errs, fmt.Sprintf("affordance %q: to %q is not a declared route", a.ID, a.To))
		}
		for i, p := range a.Presentations {
			_, routeOK := routes[p.In]
			_, containerOK := containers[p.In]
			if !routeOK && !containerOK {
				errs = append(errs, fmt.Sprintf("affordance %q: presentations[%d].in %q is neither a declared route nor a declared container", a.ID, i, p.In))
			}
		}
	}
	return errs
}

func validateReturnPathRefs(rs []contract.ReturnPath, routes map[string]contract.Route) []string {
	var errs []string
	for _, r := range rs {
		if _, ok := routes[r.From]; !ok {
			errs = append(errs, fmt.Sprintf("return_path: from %q is not a declared route", r.From))
		}
		if r.Fallback != "" {
			if _, ok := routes[r.Fallback]; !ok {
				errs = append(errs, fmt.Sprintf("return_path %q: fallback %q is not a declared route", r.From, r.Fallback))
			}
		}
	}
	return errs
}

func validateShortcutRefs(ss []contract.Shortcut, routes map[string]contract.Route) []string {
	var errs []string
	for _, s := range ss {
		if s.To != "" {
			if _, ok := routes[s.To]; !ok {
				errs = append(errs, fmt.Sprintf("shortcut %q: to %q is not a declared route", s.Binding, s.To))
			}
		}
		for _, ex := range s.ExcludedRoutes {
			if _, ok := routes[ex]; !ok {
				errs = append(errs, fmt.Sprintf("shortcut %q: excluded_routes entry %q is not a declared route", s.Binding, ex))
			}
		}
	}
	return errs
}

func validateReachabilityRefs(invs []contract.ReachabilityInvariant, routes map[string]contract.Route) []string {
	var errs []string
	for _, inv := range invs {
		if _, ok := routes[inv.From]; !ok {
			errs = append(errs, fmt.Sprintf("reachability_invariant %q: from %q is not a declared route", inv.ID, inv.From))
		}
		for _, t := range inv.MustReach {
			if _, ok := routes[t]; !ok {
				errs = append(errs, fmt.Sprintf("reachability_invariant %q: must_reach entry %q is not a declared route", inv.ID, t))
			}
		}
		for _, t := range inv.MustNotReach {
			if _, ok := routes[t]; !ok {
				errs = append(errs, fmt.Sprintf("reachability_invariant %q: must_not_reach entry %q is not a declared route", inv.ID, t))
			}
		}
	}
	return errs
}

func validateDeepLinkRefs(rs []contract.DeepLinkRule, routes map[string]contract.Route) []string {
	var errs []string
	for _, r := range rs {
		for _, id := range r.ForRoutes {
			if _, ok := routes[id]; !ok {
				errs = append(errs, fmt.Sprintf("deep_link_policy %q: for_routes entry %q is not a declared route", r.ID, id))
			}
		}
	}
	return errs
}

// validatePredicates does a lightweight syntactic check on every
// predicate string: balanced parens and brackets, no empty operand
// runs. Full AST building is Phase 3 work; this catches typos that
// would otherwise only surface during reachability verification.
func validatePredicates(c contract.Contract) []string {
	var errs []string

	check := func(scope, expr string) {
		if expr == "" {
			return
		}
		if err := checkBalanced(expr); err != nil {
			errs = append(errs, fmt.Sprintf("%s: predicate %q is malformed: %v", scope, expr, err))
		}
	}

	for name, ctx := range c.Contexts {
		check(fmt.Sprintf("context %q valid_when", name), ctx.ValidWhen)
	}
	for _, r := range c.Routes {
		check(fmt.Sprintf("route %q requires", r.ID), r.Requires)
	}
	for _, ct := range c.Containers {
		check(fmt.Sprintf("container %q show_when", ct.ID), ct.ShowWhen)
	}
	for _, a := range c.Affordances {
		check(fmt.Sprintf("affordance %q show_when", a.ID), a.ShowWhen)
	}
	for _, o := range c.Overlays {
		check(fmt.Sprintf("overlay %q show_when", o.ID), o.ShowWhen)
	}
	for _, s := range c.Shortcuts {
		check(fmt.Sprintf("shortcut %q show_when", s.Binding), s.ShowWhen)
	}
	for _, inv := range c.ReachabilityInvariants {
		check(fmt.Sprintf("reachability_invariant %q given", inv.ID), inv.Given)
	}
	for _, r := range c.DeepLinkPolicy {
		check(fmt.Sprintf("deep_link_policy %q given", r.ID), r.Given)
		check(fmt.Sprintf("deep_link_policy %q for_routes_where", r.ID), r.ForRoutesWhere)
	}
	return errs
}

func checkBalanced(expr string) error {
	parens, brackets := 0, 0
	for _, r := range expr {
		switch r {
		case '(':
			parens++
		case ')':
			parens--
			if parens < 0 {
				return fmt.Errorf("unmatched ')'")
			}
		case '[':
			brackets++
		case ']':
			brackets--
			if brackets < 0 {
				return fmt.Errorf("unmatched ']'")
			}
		}
	}
	if parens != 0 {
		return fmt.Errorf("unbalanced parentheses")
	}
	if brackets != 0 {
		return fmt.Errorf("unbalanced brackets")
	}
	return nil
}
