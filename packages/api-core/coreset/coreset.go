// Package coreset reads the operator-granted core-set authority. The lists are
// manifest/operator data, not code-owned instance knowledge.
package coreset

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/api-core/storage"
)

type Authority struct {
	Seed        []string `json:"seed"`
	TrustedBase []string `json:"trusted_base"`
	// Important is a set of selected scenario/resource roots that autoheal
	// monitors without promoting them to the core auto-start tier.
	Important []ComponentID `json:"important,omitempty"`
	// CoreMembers adds typed core-tier roots while Seed remains the compatible
	// legacy scenario field.
	CoreMembers []ComponentID `json:"members,omitempty"`
}

const (
	MemberKindScenario = "scenario"
	MemberKindResource = "resource"

	IntentMustStart = "must_start"
	IntentTryStart  = "try_start"
)

// RequiredOperationalScenarios are the minimum recovery plane for a node that
// is expected to keep agents and remote access available. They are authority
// defaults, not scenario-owned knowledge: onboarding and the control plane
// both use this list when presenting or accepting the operator's core set.
var RequiredOperationalScenarios = []string{
	"agent-manager",
	"prompt-manager",
	"web-console",
	"vrooli-autoheal",
}

// NormalizeOperationalAuthority adds the required recovery plane to an
// operator declaration and marks it as trusted-base authority. Existing
// optional members and trusted-base members are preserved.
func NormalizeOperationalAuthority(authority Authority) Authority {
	seed := append([]string(nil), authority.Seed...)
	trusted := append([]string(nil), authority.TrustedBase...)
	seed = append(seed, RequiredOperationalScenarios...)
	trusted = append(trusted, RequiredOperationalScenarios...)
	coreMembers := normalizeImportant(authority.CoreMembers)
	coreSet := make(map[ComponentID]struct{}, len(coreMembers))
	for _, member := range coreMembers {
		coreSet[member] = struct{}{}
	}
	seedSet := make(map[string]struct{}, len(seed))
	for _, name := range seed {
		name = strings.ToLower(strings.TrimSpace(name))
		seedSet[name] = struct{}{}
		coreSet[ComponentID{Kind: MemberKindScenario, Name: name}] = struct{}{}
	}
	filteredCore := make([]ComponentID, 0, len(coreMembers))
	for _, member := range coreMembers {
		if member.Kind == MemberKindScenario {
			if _, exists := seedSet[member.Name]; exists {
				continue
			}
		}
		filteredCore = append(filteredCore, member)
	}
	coreMembers = filteredCore
	important := make([]ComponentID, 0, len(authority.Important))
	for _, member := range normalizeImportant(authority.Important) {
		if _, ok := coreSet[member]; !ok {
			important = append(important, member)
		}
	}
	return Authority{Seed: normalizeSorted(seed), TrustedBase: normalizeSorted(trusted), Important: important, CoreMembers: coreMembers}
}

// AttributionStep explains one link from a supervision-set member back to its
// operator-selected core or important root. Chains are ordered from the member
// toward authority, so the final step names the selected tier source.
type AttributionStep struct {
	Name              string `json:"name"`
	Kind              string `json:"kind"`
	DeclaredBy        string `json:"declared_by,omitempty"`
	SupervisionIntent string `json:"supervision_intent"`
	Source            string `json:"source"`
}

// Member is one scenario or resource in the computed supervision closure.
type Member struct {
	Name              string            `json:"name"`
	Kind              string            `json:"kind"`
	SupervisionIntent string            `json:"supervision_intent"`
	AttributionChain  []AttributionStep `json:"attribution_chain"`
}

// Report is the database-free supervision closure computed from operator
// authority and canonical scenario manifests.
type Report struct {
	Source                string              `json:"source"`
	ClosureDigest         string              `json:"closure_digest"`
	CoreSet               []string            `json:"core_set"`
	Seed                  []string            `json:"seed"`
	AddedByClosure        []string            `json:"added_by_closure"`
	TrustedBase           []string            `json:"trusted_base"`
	Members               []Member            `json:"members"`
	MemberCounts          map[string]int      `json:"member_counts"`
	LoadErrors            map[string]string   `json:"load_errors,omitempty"`
	TrustedBaseViolations map[string][]string `json:"trusted_base_violations,omitempty"`
}

// Validate checks the structural invariants of operator-granted core
// authority. A trusted-base grant cannot name a scenario outside the seed
// authority: doing so would grant protection to an object that is not part of
// the operator's declared core.
func (a Authority) Validate() error {
	seed := normalizeSorted(a.Seed)
	trusted := normalizeSorted(a.TrustedBase)
	if len(seed) == 0 || len(trusted) == 0 {
		return os.ErrInvalid
	}
	seedSet := make(map[string]struct{}, len(seed))
	for _, name := range seed {
		seedSet[name] = struct{}{}
	}
	for _, name := range trusted {
		if _, ok := seedSet[name]; !ok {
			return fmt.Errorf("trusted-base member %q is not a core seed", name)
		}
	}
	seenImportant := map[ComponentID]struct{}{}
	seenCore := map[ComponentID]struct{}{}
	for _, member := range a.CoreMembers {
		member.Kind = strings.ToLower(strings.TrimSpace(member.Kind))
		member.Name = strings.ToLower(strings.TrimSpace(member.Name))
		if member.Kind != MemberKindScenario && member.Kind != MemberKindResource {
			return fmt.Errorf("core member %q has unsupported kind %q", member.Name, member.Kind)
		}
		if member.Name == "" {
			return fmt.Errorf("core member name is empty")
		}
		if _, duplicate := seenCore[member]; duplicate {
			return fmt.Errorf("duplicate core member %s:%s", member.Kind, member.Name)
		}
		seenCore[member] = struct{}{}
	}
	for _, member := range a.Important {
		member.Kind = strings.ToLower(strings.TrimSpace(member.Kind))
		member.Name = strings.ToLower(strings.TrimSpace(member.Name))
		if member.Kind != MemberKindScenario && member.Kind != MemberKindResource {
			return fmt.Errorf("important member %q has unsupported kind %q", member.Name, member.Kind)
		}
		if member.Name == "" {
			return fmt.Errorf("important member name is empty")
		}
		if _, duplicate := seenImportant[member]; duplicate {
			return fmt.Errorf("duplicate important member %s:%s", member.Kind, member.Name)
		}
		if _, core := seenCore[member]; core {
			return fmt.Errorf("member %s:%s cannot be both important and core", member.Kind, member.Name)
		}
		seenImportant[member] = struct{}{}
	}
	return nil
}

type operatorState struct {
	Core      *Authority                 `json:"core"`
	Scenarios map[string]selectionChoice `json:"scenarios"`
	Resources map[string]selectionChoice `json:"resources"`
}

type selectionChoice struct {
	Enabled *bool `json:"enabled"`
}

// Load reads operator-state.json from the contract-routed runtime state
// namespace. The repoRoot parameter is retained for source compatibility; it
// no longer determines the mutable state location.
func Load(_ string) (Authority, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return Authority{}, fmt.Errorf("create operator-state storage resolver: %w", err)
	}
	paths, err := resolver.Resolve(storage.Options{ScenarioID: "vrooli-onboarding"})
	if err != nil {
		return Authority{}, fmt.Errorf("resolve operator-state storage: %w", err)
	}
	path := filepath.Join(paths.StateDir, "operator-state.json")
	raw, err := os.ReadFile(path) // #nosec G304 -- repoRoot is a control-plane workspace path.
	if err != nil {
		return Authority{}, err
	}
	var state operatorState
	if err := json.Unmarshal(raw, &state); err != nil {
		return Authority{}, err
	}
	var authority Authority
	if state.Core == nil {
		// Older operator-state documents predate the explicit core authority. Apply
		// the mandatory recovery-plane defaults before validating so those
		// documents remain bootable.
		authority = NormalizeOperationalAuthority(Authority{})
	} else {
		// Explicit authority remains operator-owned: normalize representation only
		// and preserve the existing trusted-base contract.
		authority = Authority{
			Seed:        normalizeSorted(state.Core.Seed),
			TrustedBase: normalizeSorted(state.Core.TrustedBase),
			Important:   normalizeImportant(state.Core.Important),
			CoreMembers: normalizeImportant(state.Core.CoreMembers),
		}
	}
	// Tier membership grants behavior only while its component remains
	// selected. Keep persisted preferences intact so a deliberate re-selection
	// can restore the operator's tier, but never supervise a disabled entry.
	selected := func(member ComponentID) bool {
		choices := state.Scenarios
		if member.Kind == MemberKindResource {
			choices = state.Resources
		}
		choice, ok := choices[member.Name]
		return ok && choice.Enabled != nil && *choice.Enabled
	}
	if state.Core != nil {
		// Old explicit core seeds predate independent selection flags. Keep that
		// persisted intent unless a newer explicit selection turns it off; the
		// mandatory recovery plane can never be removed this way.
		filteredSeed := make([]string, 0, len(authority.Seed))
		for _, name := range authority.Seed {
			choice, hasChoice := state.Scenarios[name]
			if hasChoice && choice.Enabled != nil && !*choice.Enabled && !isRequiredOperationalScenario(name) {
				continue
			}
			filteredSeed = append(filteredSeed, name)
		}
		authority.Seed = filteredSeed
		seed := make(map[string]struct{}, len(filteredSeed))
		for _, name := range filteredSeed {
			seed[name] = struct{}{}
		}
		filteredTrusted := authority.TrustedBase[:0]
		for _, name := range authority.TrustedBase {
			if _, ok := seed[name]; ok {
				filteredTrusted = append(filteredTrusted, name)
			}
		}
		authority.TrustedBase = filteredTrusted
	}
	authority.Important = filterSelected(authority.Important, selected)
	authority.CoreMembers = filterSelected(authority.CoreMembers, selected)
	if err := authority.Validate(); err != nil {
		return Authority{}, err
	}
	return authority, nil
}

func isRequiredOperationalScenario(name string) bool {
	for _, required := range RequiredOperationalScenarios {
		if name == required {
			return true
		}
	}
	return false
}

func filterSelected(values []ComponentID, selected func(ComponentID) bool) []ComponentID {
	out := make([]ComponentID, 0, len(values))
	for _, value := range values {
		if selected(value) {
			out = append(out, value)
		}
	}
	return out
}

func normalizeImportant(values []ComponentID) []ComponentID {
	set := make(map[ComponentID]struct{}, len(values))
	for _, value := range values {
		value.Kind = strings.ToLower(strings.TrimSpace(value.Kind))
		value.Name = strings.ToLower(strings.TrimSpace(value.Name))
		set[value] = struct{}{}
	}
	out := make([]ComponentID, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind == out[j].Kind {
			return out[i].Name < out[j].Name
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

func currentAuthority() Authority {
	if authority, err := Load(""); err == nil {
		return authority
	}
	return Authority{}
}

func CoreSeedScenarios() []string { return append([]string(nil), currentAuthority().Seed...) }

func TrustedBaseScenarios() []string { return append([]string(nil), currentAuthority().TrustedBase...) }

func DefaultFallbackCoreSet() []string { return CoreSeedScenarios() }

func IsCoreSeed(name string) bool { return contains(currentAuthority().Seed, name) }

func IsTrustedBase(name string) bool { return contains(currentAuthority().TrustedBase, name) }

func normalizeSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			seen[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func contains(values []string, name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, value := range values {
		if value == name {
			return true
		}
	}
	return false
}
