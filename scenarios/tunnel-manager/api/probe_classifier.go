package main

// RouteClassification describes the combined internal+external probe status for a route.
type RouteClassification struct {
	RouteID    int    `json:"route_id"`
	Subdomain  string `json:"subdomain"`
	Status     string `json:"status"`     // "up", "tunnel-issue", "scenario-down", "unknown"
	Internal   string `json:"internal"`   // probe status
	External   string `json:"external"`   // probe status
	Assessment string `json:"assessment"` // human-readable description
}

// ClassifyProbeResults takes a set of probe results and classifies each route's status
// based on the combination of internal and external probe outcomes.
//
// Classification rules:
//   - up: both internal and external probes pass
//   - tunnel-issue: internal passes but external fails (tunnel not forwarding)
//   - scenario-down: internal fails (scenario itself is unreachable)
//   - unknown: both fail (could be anything)
func ClassifyProbeResults(results []ProbeResult) []RouteClassification {
	// Group results by route
	type pair struct {
		internal *ProbeResult
		external *ProbeResult
	}
	byRoute := make(map[int]*pair)
	subdomains := make(map[int]string)

	for i := range results {
		r := &results[i]
		if _, ok := byRoute[r.RouteID]; !ok {
			byRoute[r.RouteID] = &pair{}
		}
		subdomains[r.RouteID] = r.Subdomain
		switch r.ProbeType {
		case "internal":
			byRoute[r.RouteID].internal = r
		case "external":
			byRoute[r.RouteID].external = r
		}
	}

	var classifications []RouteClassification
	for routeID, p := range byRoute {
		c := RouteClassification{
			RouteID:   routeID,
			Subdomain: subdomains[routeID],
		}

		internalUp := p.internal != nil && p.internal.Status == "up"
		externalUp := p.external != nil && p.external.Status == "up"

		if p.internal != nil {
			c.Internal = p.internal.Status
		}
		if p.external != nil {
			c.External = p.external.Status
		}

		switch {
		case internalUp && externalUp:
			c.Status = "up"
			c.Assessment = "route is fully operational"
		case internalUp && !externalUp:
			c.Status = "tunnel-issue"
			c.Assessment = "scenario is running locally but not reachable via tunnel"
		case !internalUp && externalUp:
			// Unusual: external works but internal doesn't (possible DNS caching)
			c.Status = "scenario-down"
			c.Assessment = "internal probe failed but external still resolves (stale cache possible)"
		case !internalUp && !externalUp:
			c.Status = "unknown"
			c.Assessment = "both internal and external probes failed; investigate scenario and tunnel"
		}

		classifications = append(classifications, c)
	}
	return classifications
}
