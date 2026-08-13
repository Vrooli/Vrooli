package strategy

import (
	"fmt"
	"sort"
	"strings"
)

// ResolveHostSupport is the first gate in strategy resolution. Adapters call
// it before probing prerequisites; unsupported is terminal and deliberately
// carries no next action because there is nothing the current host can
// install to make the strategy work.
func ResolveHostSupport(id, description string, supported []string) (Declaration, bool) {
	if supportsHost(supported, HostOS) {
		return Declaration{}, false
	}
	ordered := append([]string{}, supported...)
	sort.Strings(ordered)
	return Declaration{
		StrategyID:       id,
		Description:      description,
		SupportedHostOS:  append([]string{}, ordered...),
		Reason:           UnsupportedReason(supported),
		Status:           StatusUnsupported,
		Capabilities:     map[string]Capability{},
		Tiers:            []string{},
		NextActions:      []string{},
		Promotable:       false,
		EvidenceClass:    "release-grade",
		MinimumUsefulFPS: 5,
	}, true
}

func supportsHost(supported []string, host string) bool {
	for _, candidate := range supported {
		if strings.EqualFold(strings.TrimSpace(candidate), strings.TrimSpace(host)) {
			return true
		}
	}
	return false
}

// WithSupportedHostOS stamps the declaration so the CLI and UI can explain
// both what a strategy supports and why it is unavailable on this host.
func WithSupportedHostOS(d Declaration, supported ...string) Declaration {
	d.SupportedHostOS = append([]string{}, supported...)
	return d
}

func UnsupportedReason(supported []string) string {
	ordered := append([]string{}, supported...)
	sort.Strings(ordered)
	return fmt.Sprintf("host OS %q is unsupported; this strategy requires %s", HostOS, strings.Join(ordered, ", "))
}

// ProbeCapability converts a declared capability into a truthful disposition.
// Adapters call it after probing their transport; callers must never infer an
// available capability from a static manifest alone.
func ProbeCapability(name string, ok bool, prerequisite, nextAction, evidence string) Capability {
	c := Capability{Name: name, Prerequisite: strings.TrimSpace(prerequisite), NextAction: strings.TrimSpace(nextAction), ProbeEvidence: strings.TrimSpace(evidence)}
	if ok {
		c.Status = StatusAvailable
		c.Prerequisite = ""
		c.NextAction = ""
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
