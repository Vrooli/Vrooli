// Package readiness turns an Experience Manager-owned readiness profile into
// post-navigation waits that a workflow can execute.
//
// It exists as its own package because two callers need it and neither should
// depend on the other: the capture handler settles a page before snapshotting
// it, and the workflow executor settles a page before the next step runs. Both
// want the same answer to the same question — "has this route's required
// surfaces reached a terminal lifecycle state" — so the answer is computed once,
// here.
//
// Every path degrades to the caller's existing behavior. An unavailable
// Experience Manager, an undeclared scenario, an unmatched route, or a region
// with no runtime binding all produce zero waits and a stated reason, never an
// error that fails the run.
package readiness

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	actionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract"
	contractconnect "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract/contract_v1connect"
	"google.golang.org/protobuf/proto"
)

// Strategy names the rung of the settle ladder a caller ended up on. It is
// reported rather than inferred so a silent fallback cannot be mistaken for a
// fast pass.
const (
	// StrategyExplicitWait means the caller supplied its own wait, which always
	// wins over anything declared.
	StrategyExplicitWait = "explicit-wait"
	// StrategyDeclaredSurface means the route's declared required surfaces were
	// resolved and are being waited on by terminal lifecycle state.
	StrategyDeclaredSurface = "declared-surface"
	// StrategyGenericNavigation is the pre-existing behavior: whatever the
	// navigate action's own wait_until provides, and nothing more.
	StrategyGenericNavigation = "generic-navigation"
)

// settleTimeout bounds the injected readiness wait.
//
// The wait is an OPTIMIZATION, never a new failure mode: a route where the
// declared region does not render must cost a bounded settle and then proceed,
// not a fresh timeout. The injected node also carries continue_on_error so a
// lapse can never fail a case that would otherwise pass.
const settleTimeout = 8 * time.Second

// Resolver obtains the compiled profile for a known local scenario. It is
// optional everywhere it is used: an unavailable profile preserves generic
// behavior.
type Resolver interface {
	ResolveReadinessWaits(ctx context.Context, scenario, route string) (Resolution, error)
}

// Resolution carries the graph waits alongside the contract provenance, so a
// caller can report which profile produced them and which surfaces it waited on.
type Resolution struct {
	Waits              []*actionsv1.WaitParams
	ProfileVersion     string
	Route              string
	RequiredSurfaceIDs []string
	RouteMatched       bool
}

// FallbackReason explains, in one phrase, why a resolution produced no waits.
// It returns "" when the resolution did produce waits.
func (r Resolution) FallbackReason() string {
	if len(r.Waits) > 0 {
		return ""
	}
	switch {
	case r.ProfileVersion == "":
		return "declared readiness profile returned no version"
	case !r.RouteMatched:
		return "declared readiness profile does not include the requested route"
	default:
		return "declared readiness route has no bound required surfaces"
	}
}

type experienceResolver struct {
	resolver *discovery.Resolver
	client   *http.Client
}

// NewProfileResolver constructs the production resolver for the Experience
// Manager-owned profile RPC.
func NewProfileResolver() Resolver {
	return &experienceResolver{
		resolver: discovery.NewResolver(discovery.ResolverConfig{}),
		client:   &http.Client{Timeout: 5 * time.Second},
	}
}

func (r *experienceResolver) ResolveReadinessWaits(ctx context.Context, scenario, route string) (Resolution, error) {
	base, err := r.resolver.ResolveScenarioURLDefault(ctx, "experience-manager")
	if err != nil {
		return Resolution{}, err
	}
	resp, err := contractconnect.NewContractServiceClient(r.client, base).
		GetReadinessProfile(ctx, connect.NewRequest(&contractv1.GetReadinessProfileRequest{Scenario: scenario}))
	if err != nil {
		return Resolution{}, err
	}
	var profile profileDocument
	if err := json.Unmarshal([]byte(resp.Msg.GetProfileJson()), &profile); err != nil {
		return Resolution{}, fmt.Errorf("decode readiness profile: %w", err)
	}
	resolution := Resolution{ProfileVersion: resp.Msg.GetProfileVersion(), Route: route}
	for _, page := range profile.Pages {
		if !PageMatchesRoute(page.Routes, page.RuntimeRoutes, route) {
			continue
		}
		resolution.RouteMatched = true
		for _, region := range page.Regions {
			if !region.Required {
				continue
			}
			selector := strings.TrimSpace(region.Binding.Selector)
			if selector == "" && strings.TrimSpace(region.Binding.TestID) != "" {
				selector = `[data-testid="` + strings.TrimSpace(region.Binding.TestID) + `"]`
			}
			if selector == "" {
				continue
			}
			selector = TerminalSelector(selector, region.Lifecycle.Kind, region.Lifecycle.States)
			resolution.Waits = append(resolution.Waits, &actionsv1.WaitParams{
				WaitFor: &actionsv1.WaitParams_Selector{Selector: selector},
				// ATTACHED, not VISIBLE. This probes a lifecycle state, and a
				// surface reporting empty or error may legitimately render
				// nothing visible. Requiring visibility would time out on
				// exactly the terminal states this exists to detect fast.
				State: actionsv1.WaitState_WAIT_STATE_ATTACHED.Enum(),
				// Bounded well below the 30s driver default. A surface that is
				// going to settle settles quickly; anything longer is a route
				// where this region does not render, and paying 30s there would
				// cost more than the timeout this replaces.
				TimeoutMs: proto.Int32(int32(settleTimeout.Milliseconds())),
			})
			resolution.RequiredSurfaceIDs = append(resolution.RequiredSurfaceIDs, region.ID)
		}
		return resolution, nil
	}
	return resolution, nil
}

type profileDocument struct {
	Pages []struct {
		Routes        []string `json:"routes"`
		RuntimeRoutes []string `json:"runtimeRoutes"`
		Regions       []struct {
			ID       string `json:"id"`
			Required bool   `json:"required"`
			Binding  struct {
				TestID   string `json:"testid"`
				Selector string `json:"selector"`
			} `json:"binding"`
			Lifecycle struct {
				Kind   string   `json:"kind"`
				States []string `json:"states"`
			} `json:"lifecycle"`
		} `json:"regions"`
	} `json:"pages"`
}

// TerminalSelector turns an async surface binding into a selector matching any
// of its declared terminal states. Waiting on the bare binding would accept the
// surface's initial mount — usually `loading` — as functional readiness, which
// is the mistake this whole contract exists to prevent.
//
// Terminal deliberately includes `error`: a page that has failed has finished,
// and reporting that in milliseconds is the point. Only `loading` and `static`
// are excluded.
func TerminalSelector(binding, kind string, states []string) string {
	// Some scenarios expose a dedicated, already-terminal readiness selector
	// rather than the generic ExperienceSurface data-experience-state contract.
	// Treat it as authoritative instead of appending a second incompatible state
	// predicate (for example data-preview-state="ready").
	if strings.Contains(binding, "[data-preview-state=") || strings.Contains(binding, "[data-experience-state=") {
		return binding
	}
	if kind != "async" {
		return binding
	}
	var selectors []string
	for _, state := range states {
		state = strings.TrimSpace(state)
		if state == "" || state == "loading" || state == "static" {
			continue
		}
		selectors = append(selectors, binding+`[data-experience-state="`+state+`"]`)
	}
	if len(selectors) == 0 {
		return binding
	}
	return strings.Join(selectors, ", ")
}

// PageMatchesRoute decides whether a page's declared regions apply to the route
// a flow is navigating to.
//
// Path-only matching is not sufficient for a page whose asynchronous content is
// tab-scoped. BAS's dashboard is the worked example: its route is "/", but the
// projects grid only mounts under "?tab=projects", so every declared state pins
// that query and the compiled profile carries it in runtimeRoutes. Matching on
// the path alone made a navigation to bare "/" resolve a region that cannot be
// in the document yet, and the injected wait then burned its full timeout on
// every case that starts at the dashboard.
//
// So: when a page declares runtime routes and EVERY one of them carries a query,
// the page's regions are scoped to those queries and a request without one does
// not match. A page whose runtime routes are query-free keeps the original
// path-only behavior.
func PageMatchesRoute(routes, runtimeRoutes []string, route string) bool {
	if queryScoped(runtimeRoutes) {
		requested := strings.SplitN(route, "#", 2)[0]
		for _, candidate := range runtimeRoutes {
			if candidate == requested {
				return true
			}
		}
		return false
	}
	return ContainsRoute(append(append([]string(nil), routes...), runtimeRoutes...), route)
}

// queryScoped reports whether a page's runtime routes are all query-qualified,
// which is what marks its content as scoped to a query rather than to the path.
func queryScoped(runtimeRoutes []string) bool {
	if len(runtimeRoutes) == 0 {
		return false
	}
	for _, candidate := range runtimeRoutes {
		if !strings.Contains(candidate, "?") {
			return false
		}
	}
	return true
}

// ContainsRoute reports whether any declared route template matches the path
// portion of route.
func ContainsRoute(routes []string, route string) bool {
	path := strings.SplitN(strings.SplitN(route, "#", 2)[0], "?", 2)[0]
	for _, candidate := range routes {
		if RouteTemplateMatches(candidate, path) {
			return true
		}
	}
	return false
}

// RouteTemplateMatches recognizes the colon-parameter shape authored in
// experience page routes. It deliberately matches whole path segments only:
// /assets/:id accepts /assets/Button but not /assets/Button/history, keeping
// undeclared subroutes on the generic fallback path.
func RouteTemplateMatches(template, path string) bool {
	if template == path {
		return true
	}
	templateParts := strings.Split(strings.Trim(template, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(templateParts) != len(pathParts) {
		return false
	}
	for index, part := range templateParts {
		if strings.HasPrefix(part, ":") {
			if strings.TrimPrefix(part, ":") == "" || pathParts[index] == "" {
				return false
			}
			continue
		}
		if part != pathParts[index] {
			return false
		}
	}
	return true
}
