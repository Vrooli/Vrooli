// Package parity contains the static-analysis utilities and curated coverage
// map that enforce the principle:
//
//	Every prompt-manager API endpoint must have a corresponding CLI subcommand
//	(or be explicitly marked as intentionally-absent in the coverage map).
//
// The package is consumed by parity_test.go, which fails if a v1 route in the
// API source is not present in the coverage map, preventing future drift.
package parity

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// APIRoute represents a single v1.HandleFunc registration parsed from the
// API source. Methods is the set of HTTP verbs the route accepts.
type APIRoute struct {
	Methods []string // sorted, uppercase
	Path    string   // e.g. "/teams/{id}/decisions/{decisionId}"
	Handler string   // e.g. "heartbeatHandlers.UpdateDecisionHandler"
	Line    int      // source line for diagnostics
}

// Key returns a stable identifier suitable for use as a coverage-map key:
// "METHOD1,METHOD2 PATH" with verbs sorted alphabetically.
func (r APIRoute) Key() string {
	return strings.Join(r.Methods, ",") + " " + r.Path
}

// routeRe matches lines like:
//
//	v1.HandleFunc("/teams/{id}/decisions/{decisionId}", heartbeatHandlers.UpdateDecisionHandler).Methods("PATCH", "PUT")
//
// The pattern intentionally tolerates whitespace variations and one or many
// methods. Multi-line registrations would not match — none exist in the
// current source, and the test will flag any future regression.
var routeRe = regexp.MustCompile(
	`v1\.HandleFunc\(\s*"([^"]+)"\s*,\s*([^)]+)\)\.Methods\(\s*((?:"[A-Z]+"(?:\s*,\s*)?)+)\s*\)`,
)

// ExtractAPIRoutes parses the given Go source file and returns the
// v1.HandleFunc registrations declared in it.
func ExtractAPIRoutes(path string) ([]APIRoute, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var routes []APIRoute
	for lineNum, line := range strings.Split(string(raw), "\n") {
		matches := routeRe.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		methods := parseMethods(matches[3])
		routes = append(routes, APIRoute{
			Methods: methods,
			Path:    matches[1],
			Handler: strings.TrimSpace(matches[2]),
			Line:    lineNum + 1,
		})
	}
	return routes, nil
}

func parseMethods(raw string) []string {
	out := []string{}
	for _, m := range regexp.MustCompile(`"([A-Z]+)"`).FindAllStringSubmatch(raw, -1) {
		out = append(out, m[1])
	}
	sort.Strings(out)
	return out
}

// FindUnmapped returns the routes whose Key() is not present in the coverage
// map. It is the core drift-detection primitive used by the test.
func FindUnmapped(routes []APIRoute, coverage map[string]CoverageEntry) []APIRoute {
	var missing []APIRoute
	for _, r := range routes {
		if _, ok := coverage[r.Key()]; !ok {
			missing = append(missing, r)
		}
	}
	sort.Slice(missing, func(i, j int) bool { return missing[i].Key() < missing[j].Key() })
	return missing
}

// FindStale returns coverage-map keys that no longer correspond to any route
// in the API source. Useful to catch entries left behind after a route is
// deleted upstream.
func FindStale(routes []APIRoute, coverage map[string]CoverageEntry) []string {
	live := map[string]bool{}
	for _, r := range routes {
		live[r.Key()] = true
	}
	var stale []string
	for k := range coverage {
		if !live[k] {
			stale = append(stale, k)
		}
	}
	sort.Strings(stale)
	return stale
}

// FormatRoute renders an APIRoute for human-readable error output.
func FormatRoute(r APIRoute) string {
	return fmt.Sprintf("%-12s %s  (handler=%s, line=%d)",
		strings.Join(r.Methods, ","), r.Path, r.Handler, r.Line)
}
