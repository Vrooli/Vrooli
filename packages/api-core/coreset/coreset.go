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
	return Authority{Seed: normalizeSorted(seed), TrustedBase: normalizeSorted(trusted)}
}

// AttributionStep explains one link from a supervision-set member back to the
// operator-granted seed that caused it to be included. Chains are ordered from
// the member toward authority, so the final step always has Source=core.seed.
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
	return nil
}

type operatorState struct {
	Core Authority `json:"core"`
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
	state.Core.Seed = normalizeSorted(state.Core.Seed)
	state.Core.TrustedBase = normalizeSorted(state.Core.TrustedBase)
	if err := state.Core.Validate(); err != nil {
		return Authority{}, err
	}
	return state.Core, nil
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
