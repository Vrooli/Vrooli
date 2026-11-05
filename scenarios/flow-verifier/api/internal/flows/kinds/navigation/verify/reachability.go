package verify

import (
	"container/heap"
	"fmt"
	"strings"

	"flow-verifier/internal/flows/kinds/navigation/compile"
	"flow-verifier/internal/flows/kinds/navigation/contract"
	"flow-verifier/internal/flows/kinds/navigation/predicate"
)

// Step is one edge in a reachability path.
type Step struct {
	FromRoute   string
	Affordance  string
	Via         string // container or route id from presentation.in
	ToRoute     string
	ClicksAdded int
}

// String renders a step for trace output.
func (s Step) String() string {
	return fmt.Sprintf("%s --[%s via %s, +%d]--> %s", s.FromRoute, s.Affordance, s.Via, s.ClicksAdded, s.ToRoute)
}

// Path is a click-counted reachability path.
type Path struct {
	Clicks int
	Steps  []Step
}

// FormatTrace renders a Path for inclusion in a Finding message.
func (p Path) FormatTrace() string {
	if len(p.Steps) == 0 {
		return "(empty)"
	}
	parts := make([]string, len(p.Steps))
	for i, s := range p.Steps {
		parts[i] = s.String()
	}
	return strings.Join(parts, "\n    ")
}

// ShortestPaths runs a Dijkstra search from `from` under `world`, with
// side-effect-bearing affordances filtered by `given` (so we never
// traverse out of the bounded context envelope). Returns a map of
// route id → shortest Path.
func ShortestPaths(g compile.Graph, from string, world World, given predicate.Predicate) (map[string]Path, error) {
	if _, ok := g.RoutesByID[from]; !ok {
		return nil, fmt.Errorf("start route %q not declared", from)
	}

	// state/item are local to ShortestPaths but the heap needs named
	// types it can range over.

	// Precompile predicates we need repeatedly.
	affordanceWhen := make(map[string]predicate.Predicate, len(g.Contract.Affordances))
	for _, a := range g.Contract.Affordances {
		p, err := predicate.Parse(a.ShowWhen)
		if err != nil {
			return nil, fmt.Errorf("affordance %q show_when: %w", a.ID, err)
		}
		affordanceWhen[a.ID] = p
	}
	containerWhen := make(map[string]predicate.Predicate, len(g.Contract.Containers))
	for _, c := range g.Contract.Containers {
		p, err := predicate.Parse(c.ShowWhen)
		if err != nil {
			return nil, fmt.Errorf("container %q show_when: %w", c.ID, err)
		}
		containerWhen[c.ID] = p
	}
	routeRequires := make(map[string]predicate.Predicate, len(g.Contract.Routes))
	for _, r := range g.Contract.Routes {
		p, err := predicate.Parse(r.Requires)
		if err != nil {
			return nil, fmt.Errorf("route %q requires: %w", r.ID, err)
		}
		routeRequires[r.ID] = p
	}

	best := map[state]int{}
	results := map[string]Path{}

	pq := &priorityQueue{}
	heap.Init(pq)
	initial := item{
		st:     state{route: from, world: world.Key()},
		w:      world,
		clicks: 0,
	}
	best[initial.st] = 0
	results[from] = Path{Clicks: 0}
	heap.Push(pq, initial)

	for pq.Len() > 0 {
		cur, _ := heap.Pop(pq).(item)
		if v, ok := best[cur.st]; ok && v < cur.clicks {
			continue
		}
		for _, a := range g.Contract.Affordances {
			// Visibility under current world.
			vis, err := affordanceWhen[a.ID].Eval(cur.w.Lookup())
			if err != nil {
				return nil, err
			}
			if !vis {
				continue
			}
			// Side-effect admissibility: applying it must not violate `given`.
			newWorld := cur.w
			if a.SideEffect != "" {
				mut, err := parseSideEffect(a.SideEffect)
				if err != nil {
					return nil, err
				}
				nw := cur.w.Clone()
				for k, v := range mut {
					nw[k] = v
				}
				ok, err := given.Eval(nw.Lookup())
				if err != nil {
					return nil, err
				}
				if !ok {
					continue
				}
				newWorld = nw
			}
			// Destination route requires must hold in newWorld (else
			// it would redirect; we treat that as not landing at the
			// target).
			reqOK, err := routeRequires[a.To].Eval(newWorld.Lookup())
			if err != nil {
				return nil, err
			}
			if !reqOK {
				continue
			}
			// Find each presentation we can use from cur.route.
			for _, pres := range a.Presentations {
				cost, ok, err := presentationCost(g, pres, cur.st.route, cur.w, containerWhen)
				if err != nil {
					return nil, err
				}
				if !ok {
					continue
				}
				totalCost := cost + 1 // disclosure + click on the affordance
				next := item{
					st:     state{route: a.To, world: newWorld.Key()},
					w:      newWorld,
					clicks: cur.clicks + totalCost,
					steps: append(append([]Step{}, cur.steps...), Step{
						FromRoute:   cur.st.route,
						Affordance:  a.ID,
						Via:         pres.In,
						ToRoute:     a.To,
						ClicksAdded: totalCost,
					}),
				}
				if existing, seen := best[next.st]; seen && existing <= next.clicks {
					continue
				}
				best[next.st] = next.clicks
				if existing, seen := results[a.To]; !seen || next.clicks < existing.Clicks {
					results[a.To] = Path{Clicks: next.clicks, Steps: next.steps}
				}
				heap.Push(pq, next)
			}
		}
	}
	return results, nil
}

// presentationCost returns the disclosure cost of using the
// presentation from `route` in `world`, plus an `ok` flag indicating
// whether the presentation is reachable at all from this route.
func presentationCost(g compile.Graph, p contract.Presentation, route string, world World, containerWhen map[string]predicate.Predicate) (int, bool, error) {
	// presentation.in may be a container id or a route id.
	if c, ok := g.ContainersByID[p.In]; ok {
		if !hostsRoute(c.HostRoutes, route) {
			return 0, false, nil
		}
		vis, err := containerWhen[c.ID].Eval(world.Lookup())
		if err != nil {
			return 0, false, err
		}
		if !vis {
			return 0, false, nil
		}
		return disclosureCost(c.Disclosure), true, nil
	}
	// Route-as-host: presentation lives on the route's page; user must
	// already be on that route to interact with it. No disclosure cost.
	if p.In == route {
		return 0, true, nil
	}
	return 0, false, nil
}

func hostsRoute(hosts []string, route string) bool {
	for _, h := range hosts {
		if h == "*" || h == route {
			return true
		}
	}
	return false
}

func disclosureCost(d string) int {
	switch d {
	case "always_visible":
		return 0
	case "click_to_open":
		return 1
	case "hover_to_open":
		// Hover is cheap on mouse devices but unreachable on touch; for
		// click-budget purposes count it as zero (Phase 3 scope —
		// touch-only viewport accounting is a Phase 6 refinement).
		return 0
	}
	return 0
}

// parseSideEffect understands the subset used in the schema:
// "set_context <name>=<value>" or comma-separated mutations.
func parseSideEffect(s string) (map[string]string, error) {
	const prefix = "set_context "
	if !strings.HasPrefix(s, prefix) {
		return nil, fmt.Errorf("side_effect %q: only `set_context <name>=<value>` is supported in Phase 3", s)
	}
	rest := strings.TrimSpace(s[len(prefix):])
	out := map[string]string{}
	for _, part := range strings.Split(rest, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 || kv[0] == "" || kv[1] == "" {
			return nil, fmt.Errorf("side_effect %q: malformed mutation %q", s, part)
		}
		out[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}
	return out, nil
}

type state struct {
	route string
	world string
}

type item struct {
	st     state
	w      World
	clicks int
	steps  []Step
}

type priorityQueue []item

func (pq priorityQueue) Len() int            { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool  { return pq[i].clicks < pq[j].clicks }
func (pq priorityQueue) Swap(i, j int)       { pq[i], pq[j] = pq[j], pq[i] }
func (pq *priorityQueue) Push(x interface{}) { *pq = append(*pq, x.(item)) }
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	x := old[n-1]
	*pq = old[:n-1]
	return x
}
