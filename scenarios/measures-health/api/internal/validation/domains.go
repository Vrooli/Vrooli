package validation

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Mode is a scenario's measures-architecture maturity, decided purely by whether
// it has adopted screaming architecture (a v1/domain/ proto folder).
//
//   - ModeConformant: packages/proto/schemas/<s>/v1/domain/ exists. The folder is
//     the authoritative SSOT for stateful domains; declaring a NEW stateful domain
//     via the manifest measures.domains[] is forbidden (illegal-domain-declaration).
//   - ModeFallback: no v1/domain/ folder. The scenario must declare its stateful
//     domains via measures.domains[]; it carries a standing architecture-fallback
//     advisory nudging it toward adopting v1/domain/.
//
// Directory presence (not file count) is the switch, so an empty domain/ folder is
// still conformant-with-zero-domains.
type Mode string

const (
	ModeConformant Mode = "conformant"
	ModeFallback   Mode = "fallback"
)

// DomainSource derives a target scenario's stateful domains — the EXPECTED set —
// and reports its architecture Mode. Production scans the scenario's proto
// domain/ folder (ProtoDomainSource); tests inject a fake. Keeping it a seam lets
// Classify stay pure and lets the fleet rollup run against fixtures.
type DomainSource interface {
	StatefulDomains(scenario string) ([]DerivedDomain, error)
	// Mode reports whether the scenario has a v1/domain/ folder (conformant) or
	// not (fallback). See Mode.
	Mode(scenario string) (Mode, error)
}

// statelessDomains is the built-in substrate filter: domain names that, by
// strong convention, hold configuration/utility state with no countable
// historical rows — so a measure is NOT expected (they surface as
// NOT_EXPECTED / INFO, never a coverage gap). This is deliberately minimal:
// inference leads, and the manifest measures.domains[] override is the escape
// hatch for anything this misclassifies (see the plan's expectation model).
var statelessDomains = map[string]string{
	"settings":    "stateless configuration domain",
	"config":      "stateless configuration domain",
	"preferences": "stateless configuration domain",
}

// normalizeDomain canonicalizes a domain name for comparison across the proto
// folder (snake_case file names) and the manifest (which permits hyphens):
// lowercased, hyphens folded to underscores.
func normalizeDomain(s string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(s)), "-", "_")
}

// ProtoDomainSource derives stateful domains from
// packages/proto/schemas/<scenario>/v1/domain/*.proto under RepoRoot. Each
// domain proto file is, by construction, a persisted entity type — so each is a
// stateful domain unless the stateless filter marks it otherwise. A scenario
// with no domain/ folder yields no derived domains (nothing is "expected"; a
// pure INFO report, never an error).
type ProtoDomainSource struct {
	RepoRoot string
}

// StatefulDomains scans the proto domain/ folder and returns one DerivedDomain
// per *.proto file, with the stateless filter applied. The returned slice is
// sorted by name for deterministic reports.
func (p ProtoDomainSource) StatefulDomains(scenario string) ([]DerivedDomain, error) {
	dir := filepath.Join(p.RepoRoot, "packages", "proto", "schemas", scenario, "v1", "domain")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no domain folder -> nothing expected
		}
		return nil, err
	}
	var out []DerivedDomain
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".proto") {
			continue
		}
		name := normalizeDomain(strings.TrimSuffix(e.Name(), ".proto"))
		if name == "" {
			continue
		}
		dd := DerivedDomain{Name: name, Stateful: true}
		if note, stateless := statelessDomains[name]; stateless {
			dd.Stateful = false
			dd.Note = note
		}
		out = append(out, dd)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Mode reports ModeConformant when the scenario has a v1/domain/ folder (even an
// empty one) and ModeFallback otherwise. The directory's presence is the switch.
func (p ProtoDomainSource) Mode(scenario string) (Mode, error) {
	dir := filepath.Join(p.RepoRoot, "packages", "proto", "schemas", scenario, "v1", "domain")
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return ModeFallback, nil
		}
		return "", err
	}
	if !info.IsDir() {
		return ModeFallback, nil
	}
	return ModeConformant, nil
}
