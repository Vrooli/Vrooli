package statecontract

import (
	"fmt"
	"sort"
	"strings"
)

// Diff names every difference between two manifests that changes the
// contract: the owner, then the stored type and field, root or fact. An empty
// result means the contracts are equal.
func Diff(prior, candidate Manifest) []string {
	if prior.Version != candidate.Version {
		return []string{fmt.Sprintf("state-contract format changed from v%d to v%d; owners cannot be compared", prior.Version, candidate.Version)}
	}
	var changes []string
	if prior.Toolchain != candidate.Toolchain {
		changes = append(changes, fmt.Sprintf("Go toolchain release changed: %s -> %s (encoding/json behaviour is bound to the release)", prior.Toolchain, candidate.Toolchain))
	}
	priorOwners, candidateOwners := ownersByName(prior), ownersByName(candidate)
	for _, name := range unionKeys(priorOwners, candidateOwners) {
		p, inPrior := priorOwners[name]
		c, inCandidate := candidateOwners[name]
		switch {
		case !inPrior:
			changes = append(changes, fmt.Sprintf("owner %s: added", name))
		case !inCandidate:
			changes = append(changes, fmt.Sprintf("owner %s: removed", name))
		case p.Digest != c.Digest:
			changes = append(changes, ownerChanges(p, c)...)
		}
	}
	return changes
}

func ownerChanges(prior, candidate OwnerManifest) []string {
	owner := "owner " + prior.Owner + ": "
	var changes []string
	if strings.Join(prior.Roots, "\n") != strings.Join(candidate.Roots, "\n") {
		changes = append(changes, owner+"stored envelopes changed: "+listDelta(prior.Roots, candidate.Roots))
	}
	for _, key := range unionKeys(prior.Types, candidate.Types) {
		p, inPrior := prior.Types[key]
		c, inCandidate := candidate.Types[key]
		name := displayType(key)
		switch {
		case !inPrior:
			changes = append(changes, owner+"stored type "+name+" added")
		case !inCandidate:
			changes = append(changes, owner+"stored type "+name+" removed")
		default:
			for _, detail := range shapeChanges(p, c) {
				changes = append(changes, owner+"stored type "+name+" "+detail)
			}
		}
	}
	for _, name := range unionKeys(prior.Facts, candidate.Facts) {
		p, inPrior := prior.Facts[name]
		c, inCandidate := candidate.Facts[name]
		if constant, ok := strings.CutPrefix(name, storedConstantPrefix); ok {
			switch {
			case !inPrior:
				changes = append(changes, owner+"stored constant "+constant+" added (if it is never persisted, classify it in statecontract.NotStoredConstants)")
			case !inCandidate:
				changes = append(changes, owner+"stored constant "+constant+" removed")
			case p != c:
				changes = append(changes, owner+"stored constant "+constant+" changed")
			}
			continue
		}
		switch {
		case !inPrior:
			changes = append(changes, owner+"format fact "+name+" added")
		case !inCandidate:
			changes = append(changes, owner+"format fact "+name+" removed")
		case p != c && name == timeLayoutsFact:
			changes = append(changes, owner+"time layouts used to format or parse stored timestamps changed")
		case p != c && name == packageConstantsFact:
			changes = append(changes, owner+"package constants changed (its stored status, kind and reason values are untyped constants)")
		case p != c && name == wholeSourceFact:
			changes = append(changes, owner+"source changed (a whole-source owner: any edit to the rollback machinery or native identities changes its contract)")
		case p != c:
			changes = append(changes, owner+"format fact "+name+" changed")
		}
	}
	if len(changes) == 0 {
		changes = append(changes, owner+"contract digest changed")
	}
	return changes
}

func shapeChanges(prior, candidate TypeShape) []string {
	var changes []string
	if prior.Kind != candidate.Kind {
		changes = append(changes, fmt.Sprintf("kind changed: %s -> %s", prior.Kind, candidate.Kind))
	}
	if prior.Elem != candidate.Elem {
		changes = append(changes, fmt.Sprintf("element changed: %s -> %s", displayType(prior.Elem), displayType(candidate.Elem)))
	}
	priorFields, candidateFields := fieldsByName(prior.Fields), fieldsByName(candidate.Fields)
	for _, name := range unionKeys(priorFields, candidateFields) {
		p, inPrior := priorFields[name]
		c, inCandidate := candidateFields[name]
		switch {
		case !inPrior:
			changes = append(changes, "field "+name+" added ("+describeField(c)+")")
		case !inCandidate:
			changes = append(changes, "field "+name+" removed (was "+describeField(p)+")")
		case p != c:
			changes = append(changes, "field "+name+" changed: "+describeField(p)+" -> "+describeField(c))
		}
	}
	if prior.Codec != candidate.Codec {
		changes = append(changes, "enum constants, custom encoder or codec-consulted values changed")
	}
	return changes
}

func fieldsByName(fields []string) map[string]string {
	out := make(map[string]string, len(fields))
	for _, field := range fields {
		name, _, _ := strings.Cut(field, "|")
		out[name] = field
	}
	return out
}

func describeField(field string) string {
	parts := strings.SplitN(field, "|", 4)
	if len(parts) != 4 {
		return field
	}
	text := fmt.Sprintf("json %q, type %s", parts[1], displayType(parts[2]))
	if parts[3] == "true" {
		text += ", embedded"
	}
	return text
}

func listDelta(prior, candidate []string) string {
	in := func(list []string, value string) bool {
		for _, item := range list {
			if item == value {
				return true
			}
		}
		return false
	}
	var parts []string
	for _, value := range candidate {
		if !in(prior, value) {
			parts = append(parts, "+"+displayType(value))
		}
	}
	for _, value := range prior {
		if !in(candidate, value) {
			parts = append(parts, "-"+displayType(value))
		}
	}
	return strings.Join(parts, " ")
}

// displayType shortens repository import paths in human-facing text.
func displayType(key string) string {
	key = strings.ReplaceAll(key, "github.com/vrooli/vrooli/", "")
	return strings.ReplaceAll(key, "github.com/vrooli/", "")
}

func ownersByName(m Manifest) map[string]OwnerManifest {
	out := make(map[string]OwnerManifest, len(m.Owners))
	for _, owner := range m.Owners {
		out[owner.Owner] = owner
	}
	return out
}

func unionKeys[V any](a, b map[string]V) []string {
	keys := make([]string, 0, len(a)+len(b))
	for key := range a {
		keys = append(keys, key)
	}
	for key := range b {
		if _, ok := a[key]; !ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}
