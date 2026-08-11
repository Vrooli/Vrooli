package strategy

import "strings"

// ProbeCapability converts a declared capability into a truthful disposition.
// Adapters call it after probing their transport; callers must never infer an
// available capability from a static manifest alone.
func ProbeCapability(name string, ok bool, prerequisite, nextAction, evidence string) Capability {
	c := Capability{Name: name, Prerequisite: strings.TrimSpace(prerequisite), NextAction: strings.TrimSpace(nextAction), ProbeEvidence: strings.TrimSpace(evidence)}
	if ok {
		c.Status = StatusAvailable
	} else {
		c.Status = StatusUnavailable
	}
	return c
}

func UnavailableDeclaration(id, description string, caps []Capability, next ...string) Declaration {
	m := make(map[string]Capability, len(caps))
	for _, c := range caps {
		if c.Status == "" {
			c.Status = StatusUnavailable
		}
		m[c.Name] = c
	}
	return Declaration{StrategyID: id, Description: description, Status: StatusUnavailable, Capabilities: m, NextActions: next, Promotable: true, EvidenceClass: "release-grade", MinimumUsefulFPS: 5, Tiers: []string{}}
}
