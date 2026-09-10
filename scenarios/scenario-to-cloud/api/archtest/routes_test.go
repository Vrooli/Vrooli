package archtest

import (
	"go/ast"
	"strconv"
	"strings"
	"testing"

	"scenario-to-cloud/authz"
)

// registeredRoute is one literal HandleFunc/Handle registration found in the
// source of the main package.
type registeredRoute struct {
	Method, Path, Where string
}

// routerPrefixes maps the receiver identifier of a registration call to the
// path prefix it mounts under. `api` is the /api/v1 subrouter; `s.router`
// is the root router.
var routerPrefixes = map[string]string{"api": "/api/v1", "router": ""}

// sourceRoutes collects every literal route registration in main.go and the
// handlers_*.go register*Routes functions. Connect mounts register through
// PathPrefix with a computed path and are covered by the Prefix rows of the
// authorization table and the live census test.
func sourceRoutes(t *testing.T, src []sourceFile) []registeredRoute {
	t.Helper()
	var out []registeredRoute
	for _, f := range src {
		if f.Pkg != "" || !(f.Rel == "main.go" || strings.HasPrefix(f.Rel, "handlers_")) {
			continue
		}
		ast.Inspect(f.File, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			var methods []string
			register := call
			if sel.Sel.Name == "Methods" {
				inner, ok := sel.X.(*ast.CallExpr)
				if !ok {
					return true
				}
				for _, arg := range call.Args {
					if lit, ok := arg.(*ast.BasicLit); ok {
						if value, err := strconv.Unquote(lit.Value); err == nil {
							methods = append(methods, value)
						}
					}
				}
				register = inner
				sel, ok = inner.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
			}
			if sel.Sel.Name != "HandleFunc" && sel.Sel.Name != "Handle" {
				return true
			}
			if len(register.Args) == 0 {
				return true
			}
			lit, ok := register.Args[0].(*ast.BasicLit)
			if !ok {
				return true
			}
			path, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}
			prefix, ok := routerPrefixes[receiverName(sel.X)]
			if !ok {
				return true
			}
			if len(methods) == 0 {
				methods = []string{"*"}
			}
			for _, method := range methods {
				out = append(out, registeredRoute{Method: method, Path: prefix + path, Where: f.position(register)})
			}
			// The Methods wrapper was handled here; do not descend into the
			// inner HandleFunc call again.
			return false
		})
	}
	return out
}

func receiverName(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return x.Sel.Name
	}
	return ""
}

// TestEveryRegisteredRouteHasAnAuthorizationRow [REQ:STC-P0-013] fails when
// a literal route registration in main.go or a handlers_*.go file has no row
// in authz.Table. The live census test proves the router; this test proves
// the source, so an unclassified route fails before the server is built.
func TestEveryRegisteredRouteHasAnAuthorizationRow(t *testing.T) {
	_, src := loadSources(t)
	routes := sourceRoutes(t, src)
	if len(routes) < 80 {
		t.Fatalf("found only %d literal route registrations; the source scan is broken", len(routes))
	}
	seen := map[string]struct{}{}
	for _, route := range routes {
		key := authz.Key(route.Method, route.Path)
		if _, dup := seen[key]; dup {
			t.Errorf("%s: %s is registered twice", route.Where, key)
		}
		seen[key] = struct{}{}
		if _, ok := authz.Lookup(route.Method, route.Path); !ok {
			t.Errorf("%s: route %s has no authorization row; add it to authz.Table with its effect, scope and target binding", route.Where, key)
		}
	}
}

// unboundEffectfulRoutes lists every effectful row that is not TargetBound
// and why it is legitimately not bound to a deployment id. Anything effectful
// and unlisted here must carry {id} and be TargetBound, so the authority
// checks the deployment's target binding before the handler runs.
var unboundEffectfulRoutes = map[string]string{
	"POST /api/v1/bundle/build":                                      "builds a bundle in the local store from a manifest; no deployment record and no target",
	"POST /api/v1/bundles/cleanup":                                   "prunes the local bundle store",
	"DELETE /api/v1/bundles/{sha256}":                                "deletes one local bundle by digest",
	"POST /api/v1/releases/build":                                    "builds a release set locally from a manifest and bundle; no target",
	"POST /api/v1/deployments":                                       "creates the deployment record; the id does not exist yet, so the target binding is created here",
	"POST /api/v1/vps/setup/apply":                                   "ad hoc manifest-scoped plan apply (P06): identity is derived from the manifest target because no record is stored",
	"POST /api/v1/vps/deploy/apply":                                  "ad hoc manifest-scoped plan apply (P06), as above",
	"POST /api/v1/instances":                                         "local QEMU qualification instance provider (P20); the host is this machine",
	"POST /api/v1/instances/{id}/{action}":                           "local QEMU qualification instance provider (P20); {id} is an instance, not a deployment",
	"POST /api/v1/preflight/fix/firewall":                            "pre-deployment host fix on an explicit connection before a record exists; runs edge.ufw.allow through the bounded adapter (DL-11 remainder)",
	"POST /api/v1/preflight/fix/stop-processes":                      "pre-deployment host fix on an explicit connection; scoped lifecycle stops through the bounded adapter (DL-11 remainder)",
	"POST /api/v1/preflight/disk/cleanup":                            "pre-deployment disk cleanup on an explicit connection (DL-03/DL-11 remainder; retire with the management tab)",
	"GET /api/v1/secrets/{scenario}":                                 "reads the scenario's declared secret expectations from the local secrets owner; no target",
	"POST /api/v1/operations/reconcile":                              "owner-wide reconciliation pass over every deployment's non-terminal operations",
	"POST /api/v1/operations/{id}/cancel":                            "{id} is an operation; the operation record names its deployment and the cancel is fenced by the owner",
	"* /vrooli.scenario_to_cloud.v1.credentials.CredentialsService/": "Connect mount; every RPC resolves the deployment and checks its target binding inside the service (P04 service tests)",
	"* /vrooli.scenario_to_cloud.v1.operations.OperationsService/":   "Connect mount; get/wait/list are reads, cancel and reconcile are operation-scoped",
	"* /vrooli.scenario_to_cloud.v1.plans.PlansService/":             "Connect mount; CompilePlan/ApplyPlan resolve the deployment inside the service",
	"* /vrooli.scenario_to_cloud.v1.releases.ReleasesService/":       "Connect mount; build/get/verify act on the local release store",
	"* /vrooli.scenario_to_cloud.v1.evidence.EvidenceService/":       "Connect mount; ApplyPublication resolves the deployment and re-checks approval inside the service (P17)",
	"* /vrooli.dev_routing.v1.routing.RoutingService/":               "development-only Test Genie routed-pool leases (Optional row)",
}

// TestEffectfulRoutesAreTargetBoundOrJustified [REQ:STC-P0-013] fails when
// an effectful authorization row is neither TargetBound nor allowlisted with a
// reason, and when an allowlist entry no longer matches a row.
func TestEffectfulRoutesAreTargetBoundOrJustified(t *testing.T) {
	seen := map[string]struct{}{}
	for _, row := range authz.Table {
		if !row.Effectful() {
			continue
		}
		key := row.Key()
		if row.TargetBound {
			if _, listed := unboundEffectfulRoutes[key]; listed {
				t.Errorf("%s is TargetBound and also allowlisted; drop the entry", key)
			}
			if !strings.Contains(row.Path, "{id}") {
				t.Errorf("%s is TargetBound but carries no {id}", key)
			}
			continue
		}
		seen[key] = struct{}{}
		if _, ok := unboundEffectfulRoutes[key]; !ok {
			t.Errorf("effectful route %s (%s) is not TargetBound and has no allowlist reason", key, row.Effect)
		}
	}
	for key := range unboundEffectfulRoutes {
		if _, ok := seen[key]; !ok {
			t.Errorf("allowlist entry %q matches no effectful unbound row; remove or correct it", key)
		}
	}
}
